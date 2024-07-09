// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
// Faculty of Arts, Charles University
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate pigeon -o ./cql/grammar.go ./cql/grammar.peg

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/czcorpus/cnc-gokit/collections"
	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/czcorpus/cqlizer/benchmark"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/logproc"
	"github.com/czcorpus/cqlizer/prediction"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/gin-gonic/gin"
	randomforest "github.com/malaschitz/randomForest"
	"github.com/rs/zerolog/log"
)

var (
	version   string
	buildDate string
	gitCommit string
)

// VersionInfo provides a detailed information about the actual build
type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitCommit string `json:"gitCommit"`
}

func getEnv(name string) string {
	for _, p := range os.Environ() {
		items := strings.Split(p, "=")
		if len(items) == 2 && items[0] == name {
			return items[1]
		}
	}
	return ""
}

func getRequestOrigin(ctx *gin.Context) string {
	currOrigin, ok := ctx.Request.Header["Origin"]
	if ok {
		return currOrigin[0]
	}
	return ""
}

func additionalLogEvents() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logging.AddLogEvent(ctx, "userAgent", ctx.Request.UserAgent())
		logging.AddLogEvent(ctx, "corpusId", ctx.Param("corpusId"))
		ctx.Next()
	}
}

func CORSMiddleware(conf *cnf.Conf) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if strings.HasSuffix(ctx.Request.URL.Path, "/openapi") {
			ctx.Header("Access-Control-Allow-Origin", "*")
			ctx.Header("Access-Control-Allow-Methods", "GET")
			ctx.Header("Access-Control-Allow-Headers", "Content-Type")

		} else {
			var allowedOrigin string
			currOrigin := getRequestOrigin(ctx)
			for _, origin := range conf.CorsAllowedOrigins {
				if currOrigin == origin {
					allowedOrigin = origin
					break
				}
			}
			if allowedOrigin != "" {
				ctx.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				ctx.Writer.Header().Set(
					"Access-Control-Allow-Headers",
					"Content-Type, Content-Length, Accept-Encoding, Authorization, Accept, Origin, Cache-Control, X-Requested-With",
				)
				ctx.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
			}

			if ctx.Request.Method == "OPTIONS" {
				ctx.AbortWithStatus(204)
				return
			}
		}
		ctx.Next()
	}
}

func AuthRequired(conf *cnf.Conf) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(conf.AuthHeaderName) > 0 && !collections.SliceContains(conf.AuthTokens, ctx.GetHeader(conf.AuthHeaderName)) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		}
		ctx.Next()
	}
}

func loadModel(conf *cnf.Conf, statsDB *stats.Database, trainingID int) (randomforest.Forest, float64, error) {

	threshold, err := statsDB.GetTrainingThreshold(trainingID)
	if err != nil {
		return randomforest.Forest{},
			0.0,
			fmt.Errorf("failed to load model for training %d: %w", trainingID, err)
	}

	log.Info().Int("trainingId", trainingID).Msg("found required training")

	tdata, err := statsDB.GetTrainingData(trainingID)
	if err != nil {
		return randomforest.Forest{},
			threshold,
			fmt.Errorf("failed to load model for training %d: %w", trainingID, err)
	}

	vdata, err := statsDB.GetTrainingValidationData(trainingID)
	if err != nil {
		return randomforest.Forest{},
			threshold,
			fmt.Errorf("failed to load model for training %d: %w", trainingID, err)
	}

	eng := prediction.NewEngine(conf, statsDB)
	model, err := eng.TrainReplay(threshold, tdata, vdata)
	if err != nil {
		return randomforest.Forest{},
			threshold,
			fmt.Errorf("failed to load model for training %d: %w", trainingID, err)
	}
	return model, threshold, err
}

func runKontextImport(conf *cnf.Conf, path string, addToTrainingSet bool) {
	err := logproc.ImportLog(conf, path, addToTrainingSet)
	if err != nil {
		fmt.Println("FAILED: ", err)
	}
}

func runSizesImport(conf *cnf.Conf, path string) {
	db, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		fmt.Println("FAILED: ", err)
	}
	err = db.Init()
	if err != nil {
		fmt.Println("FAILED: ", err)
	}
	err = db.ImportCorpusSizesFromCSV(path)
	if err != nil {
		fmt.Println("FAILED: ", err)
	}
}

func runBenchmark(conf *cnf.Conf, overwriteBenchmarked bool) {
	db, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}
	err = db.Init()
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}

	exe := benchmark.NewExecutor(
		conf,
		db,
	)
	err = exe.RunFull(overwriteBenchmarked)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}
}

func runLearning(conf *cnf.Conf, threshold float64) {
	db, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}
	err = db.Init()
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}
	eng := prediction.NewEngine(conf, db)
	err = eng.Train(threshold)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}
}

func runTrainingReplay(conf *cnf.Conf, trainingId int) {
	db, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}
	err = db.Init()
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}
	threshold, err := db.GetTrainingThreshold(trainingId)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}

	tdata, err := db.GetTrainingData(trainingId)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}

	vdata, err := db.GetTrainingValidationData(trainingId)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}

	eng := prediction.NewEngine(conf, db)
	_, err = eng.TrainReplay(threshold, tdata, vdata)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}
}

func runEvaluation(conf *cnf.Conf, trainingID, numSamples, sampleSize int) {
	statsDB, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to run evaluation")
		os.Exit(1)
		return
	}

	err = statsDB.Init()
	if err != nil {
		log.Error().Err(err).Msg("failed to run evaluation")
		os.Exit(1)
		return
	}

	if trainingID == 0 {
		log.Warn().Msg("no training ID provided, going to use the latest one")
		trainingID, err = statsDB.GetLatestTrainingID()
		if err != nil {
			log.Error().Err(err).Msg("failed to run evaluation")
			os.Exit(1)
			return
		}
	}

	model, threshold, err := loadModel(conf, statsDB, trainingID)
	if err != nil {
		os.Exit(1)
		return
	}

	recs, err := statsDB.GetAllRecords(
		stats.ListFilter{}.SetBenchmarked(true).SetTrainingExcluded(true))
	if err != nil {
		log.Error().Err(err).Msg("failed to run validation, exiting")
		os.Exit(1)
	}

	var avgPrecision, avgRecall float64
	for i := 0; i < numSamples; i++ {
		smpl := collections.SliceSample(recs, sampleSize)
		eng := prediction.NewEngine(conf, statsDB)
		result, err := eng.Evaluate(model, smpl, threshold, func(itemID string, prediction bool) error {
			log.Debug().Str("queryId", itemID).Bool("prediction", prediction).Send()
			return nil
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to run validation, exiting")
			os.Exit(1)
		}
		log.Info().
			Int("sampleNum", i).
			Int("truePositives", result.TruePositives).
			Int("falsePositives", result.FalsePositives).
			Int("falseNegatives", result.FalseNegatives).
			Int("total", result.TotalTests).
			Float64("precision", result.Precision()).
			Float64("recall", result.Recall()).
			Send()
		avgPrecision += result.Precision()
		avgRecall += result.Recall()
	}
	fmt.Println("----------------------------------------------------")
	fmt.Println("sample size: ", sampleSize)
	fmt.Println("number of samples (test runs): ", numSamples)
	fmt.Printf("AVG PRECISION: %01.2f\n", avgPrecision/float64(numSamples))
	fmt.Printf("AVG RECALL: %01.2f\n", avgRecall/float64(numSamples))
	fmt.Println("----------------------------------------------------")
}

func runApiServer(
	conf *cnf.Conf,
	trainingID int,
	syscallChan chan os.Signal,
	exitEvent chan os.Signal,
) {
	if !conf.LogLevel.IsDebugMode() {
		gin.SetMode(gin.ReleaseMode)
	}

	statsDB, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to open working database")
		syscallChan <- syscall.SIGTERM
		return
	}

	err = statsDB.Init()
	if err != nil {
		log.Error().Err(err).Msg("failed to start service")
		syscallChan <- syscall.SIGTERM
		return
	}

	if trainingID == 0 {
		log.Warn().Msg("no training ID provided, going to use the latest one")
		trainingID, err = statsDB.GetLatestTrainingID()
		if err != nil {
			log.Error().Err(err).Msg("failed to start service")
			syscallChan <- syscall.SIGTERM
			return
		}
	}
	model, _, err := loadModel(conf, statsDB, trainingID)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize model")
		syscallChan <- syscall.SIGTERM
		return
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(additionalLogEvents())
	engine.Use(logging.GinMiddleware())
	engine.Use(uniresp.AlwaysJSONContentType())
	engine.Use(CORSMiddleware(conf))
	engine.NoMethod(uniresp.NoMethodHandler)
	engine.NoRoute(uniresp.NotFoundHandler)

	cqlActions := Actions{
		StatsDB: statsDB,
		rfModel: model,
	}

	engine.GET("/analyze", cqlActions.AnalyzeQuery)

	engine.GET("/parse", cqlActions.ParseQuery)

	engine.PUT("/query", cqlActions.StoreQuery)

	log.Info().Msgf("starting to listen at %s:%d", conf.ListenAddress, conf.ListenPort)
	srv := &http.Server{
		Handler:      engine,
		Addr:         fmt.Sprintf("%s:%d", conf.ListenAddress, conf.ListenPort),
		WriteTimeout: time.Duration(conf.ServerWriteTimeoutSecs) * time.Second,
		ReadTimeout:  time.Duration(conf.ServerReadTimeoutSecs) * time.Second,
	}
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		syscallChan <- syscall.SIGTERM
	}()

	<-exitEvent
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = srv.Shutdown(ctx)
	if err != nil {
		log.Info().Err(err).Msg("Shutdown request error")
	}
}

func cleanVersionInfo(v string) string {
	return strings.TrimLeft(strings.Trim(v, "'"), "v")
}

func parseTrainingIdOrExit(v string) int {
	if v == "" {
		return 0
	}
	trainingID, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		fmt.Println("FAILED: ", err)
		os.Exit(1)
	}
	return int(trainingID)
}

func main() {
	version := VersionInfo{
		Version:   cleanVersionInfo(version),
		BuildDate: cleanVersionInfo(buildDate),
		GitCommit: cleanVersionInfo(gitCommit),
	}
	overwriteAll := flag.Bool("overwrite-all", false, "If set, then all the queries will be benchmarked even if they already have a result attached")
	addToTrainingSet := flag.Bool("add-to-training", false, "If set, than all the imported records will become part of the training&validation set")
	numSamples := flag.Int("num-samples", 10, "Number of samples for the validation action")
	sampleSize := flag.Int("sample-size", 500, "Sample size for the validation action")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "CQLIZER - A CQL toolbox\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t%s [options] start config.json [trainingID]\n\t", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\n\t%s [options] import config.json source_file\n\t", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\n\t%s [options] benchmark config.json\n\t", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\n\t%s [options] learn config.json\n\t", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "%s [options] version\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\nArguments:\n\n")
		fmt.Fprintf(os.Stderr, "\tconfig.json\ta path to a config file\n")
		fmt.Fprintf(os.Stderr, "\ttrainingID\tAn ID of a training used as a base for running service. If omitted, the latest training ID will be used\n")
		fmt.Fprintf(os.Stderr, "\tsource_file\tA KonText log file to import training/validation user queries from")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	action := flag.Arg(0)
	if action == "version" {
		fmt.Printf("CQLizer %s\nbuild date: %s\nlast commit: %s\n", version.Version, version.BuildDate, version.GitCommit)
		return
	}
	conf := cnf.LoadConfig(flag.Arg(1))
	logging.SetupLogging(conf.LogFile, conf.LogLevel)

	log.Info().Msg("Starting CQLizer")
	cnf.ValidateAndDefaults(conf)
	syscallChan := make(chan os.Signal, 1)
	signal.Notify(syscallChan, os.Interrupt)
	signal.Notify(syscallChan, syscall.SIGTERM)
	exitEvent := make(chan os.Signal)
	go func() {
		evt := <-syscallChan
		exitEvent <- evt
		close(exitEvent)
	}()

	switch action {
	case "start":
		var trainingID int64
		var err error
		if flag.Arg(2) != "" {
			trainingID, err = strconv.ParseInt(flag.Arg(2), 10, 64)
			if err != nil {
				fmt.Println("FAILED: ", err)
				os.Exit(1)
			}
		}
		runApiServer(conf, int(trainingID), syscallChan, exitEvent)
	case "import":
		runKontextImport(conf, flag.Arg(2), *addToTrainingSet)
	case "corpsizes":
		runSizesImport(conf, flag.Arg(2))
	case "benchmark":
		runBenchmark(conf, *overwriteAll)
	case "replay":
		trainingID := parseTrainingIdOrExit(flag.Arg(2))
		runTrainingReplay(conf, trainingID)
	case "evaluate":
		trainingID := parseTrainingIdOrExit(flag.Arg(2))
		runEvaluation(conf, trainingID, *numSamples, *sampleSize)
	case "learn":
		thr, err := strconv.ParseFloat(flag.Arg(2), 64)
		if err != nil {
			fmt.Println("FAILED: ", err)
			os.Exit(1)
		}
		runLearning(conf, thr)
	default:
		log.Fatal().Msgf("Unknown action %s", action)
	}

}
