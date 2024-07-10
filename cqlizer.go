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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/prediction"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	randomforest "github.com/malaschitz/randomForest"
	"github.com/rs/zerolog/log"
)

type action string

func (a action) String() string {
	return string(a)
}

func (a action) validate() error {
	if a != actionServer && a != actionImport && a != actionCorpsizes && a != actionBenchmark &&
		a != actionReplay && a != actionEvaluate && a != actionLearn && a != actionVersion {
		return fmt.Errorf("unknown action: %s", a)
	}
	return nil
}

const (
	actionServer    action = "server"
	actionImport    action = "import"
	actionCorpsizes action = "corpsizes"
	actionBenchmark action = "benchmark"
	actionReplay    action = "replay"
	actionEvaluate  action = "evaluate"
	actionLearn     action = "learn"
	actionVersion   action = "version"
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

func loadModel(conf *cnf.Conf, statsDB *stats.Database, trainingID int) (randomforest.Forest, float64, error) {

	threshold, err := statsDB.GetTrainingThreshold(trainingID)
	if err != nil {
		return randomforest.Forest{},
			0.0,
			fmt.Errorf("failed to load model for training %d \u25B6 %w", trainingID, err)
	}

	log.Info().Int("trainingId", trainingID).Msg("found required training")

	tdata, err := statsDB.GetTrainingData(trainingID)
	if err != nil {
		return randomforest.Forest{},
			threshold,
			fmt.Errorf("failed to load model for training %d \u25B6 %w", trainingID, err)
	}

	vdata, err := statsDB.GetTrainingValidationData(trainingID)
	if err != nil {
		return randomforest.Forest{},
			threshold,
			fmt.Errorf("failed to load model for training %d \u25B6 %w", trainingID, err)
	}

	eng := prediction.NewEngine(conf, statsDB)
	model, err := eng.TrainReplay(threshold, tdata, vdata)
	if err != nil {
		return randomforest.Forest{},
			threshold,
			fmt.Errorf("failed to load model for training %d \u25B6 %w", trainingID, err)
	}
	return model, threshold, err
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
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return int(trainingID)
}

func topLevelUsage() {
	fmt.Fprintf(os.Stderr, "CQLIZER - a CQL analysis tool\n")
	fmt.Fprintf(os.Stderr, "-----------------------------\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "\t%s\t\tshow version info\n", actionVersion.String())
	fmt.Fprintf(os.Stderr, "\t%s\t\tstart HTTP API server\n", actionServer.String())
	fmt.Fprintf(os.Stderr, "\t%s\t\timport queries from KonText logs\n", actionImport.String())
	fmt.Fprintf(os.Stderr, "\t%s\timport corpora sizes (currently unused)\n", actionCorpsizes.String())
	fmt.Fprintf(os.Stderr, "\t%s\trun benchmarks on stored queries\n", actionBenchmark.String())
	fmt.Fprintf(os.Stderr, "\t%s\t\treplay stored training\n", actionReplay.String())
	fmt.Fprintf(os.Stderr, "\t%s\tevaluate queries with `trainingExclude` flag\n", actionEvaluate.String())
	fmt.Fprintf(os.Stderr, "\t%s\t\tlearn using benchmarked queries (ones without `trainingExclude` flag)\n", actionLearn.String())
	fmt.Fprintf(os.Stderr, "\nUse `cqlizer command -h` for information about a specific action\n\n")
}

func setupConfAndLogging(cmd *flag.FlagSet, idx int) *cnf.Conf {
	conf := cnf.LoadConfig(cmd.Arg(idx))
	logging.SetupLogging(conf.LogFile, conf.LogLevel)
	cnf.ValidateAndDefaults(conf)
	return conf
}

func main() {
	version := VersionInfo{
		Version:   cleanVersionInfo(version),
		BuildDate: cleanVersionInfo(buildDate),
		GitCommit: cleanVersionInfo(gitCommit),
	}

	cmdServer := flag.NewFlagSet(actionServer.String(), flag.ExitOnError)
	cmdServer.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\t%s %s [options] config.json [trainingID]\n\t",
			filepath.Base(os.Args[0]), actionServer.String())
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdServer.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nStart HTTP API Server for CQL analysis\n")
	}

	cmdBenchmark := flag.NewFlagSet(actionBenchmark.String(), flag.ExitOnError)
	overwriteAll := cmdBenchmark.Bool("overwrite-all", false, "If set, then all the queries will be benchmarked even if they already have a result attached")
	cmdBenchmark.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\t%s %s [options] config.json\n\t",
			filepath.Base(os.Args[0]), actionBenchmark.String())
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdBenchmark.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nBenchmark available queries.\n")
	}

	cmdCorpsizes := flag.NewFlagSet(actionCorpsizes.String(), flag.ExitOnError)
	addToTrainingSet := cmdCorpsizes.Bool("add-to-training", false, "If set, than all the imported records will become part of the training&validation set")

	cmdEvaluate := flag.NewFlagSet(actionEvaluate.String(), flag.ExitOnError)
	numSamples := cmdEvaluate.Int("num-samples", 10, "Number of samples for the validation action")
	sampleSize := cmdEvaluate.Int("sample-size", 500, "Sample size for the validation action")
	cmdEvaluate.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\t%s %s [options] config.json [trainingID]\n\t",
			filepath.Base(os.Args[0]), actionEvaluate.String())
		fmt.Fprintf(os.Stderr, "\nArguments:\n")
		fmt.Fprintf(os.Stderr, "\tconfig.json\ta path to a config file\n")
		fmt.Fprintf(os.Stderr, "\ttrainingID\tAn ID of a training used as a base for running service. If omitted, the latest training ID will be used\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdEvaluate.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEvaluate stored queries (the ones with the `trainingExclude` flag set). This is intended for the \"real word\" model testing.\n")
	}

	cmdImport := flag.NewFlagSet(actionImport.String(), flag.ExitOnError)
	cmdImport.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\t%s %s [options] config.json tsource_file\n\t",
			filepath.Base(os.Args[0]), actionImport.String())
		fmt.Fprintf(os.Stderr, "\nArguments:\n")
		fmt.Fprintf(os.Stderr, "\tconfig.json\ta path to a config file\n")
		fmt.Fprintf(os.Stderr, "\tsource_file\tA KonText log file to import training/validation user queries from")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdImport.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nImport queries from a KonText application log file\n")
	}

	cmdLearn := flag.NewFlagSet(actionLearn.String(), flag.ExitOnError)
	cmdLearn.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\t%s %s [options] config.json threshold\n\t",
			filepath.Base(os.Args[0]), actionLearn.String())
		fmt.Fprintf(os.Stderr, "\nArguments:\n")
		fmt.Fprintf(os.Stderr, "\tconfig.json\ta path to a config file\n")
		fmt.Fprintf(os.Stderr, "\threshold\tA threshold value (in seconds) for what is considered a problematic query in a benchmark environment\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdLearn.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nLearn a new model based on queries stored in database (the ones without the `trainingExclude` flag)\n")
	}

	cmdReplay := flag.NewFlagSet(actionReplay.String(), flag.ExitOnError)
	cmdReplay.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\t%s %s [options] config.json [training_ID]\n\t",
			filepath.Base(os.Args[0]), actionReplay.String())
		fmt.Fprintf(os.Stderr, "\nArguments:\n")
		fmt.Fprintf(os.Stderr, "\tconfig.json\ta path to a config file\n")
		fmt.Fprintf(os.Stderr, "\ttraining_ID\tAn ID of a training used as a base for running service. If omitted, the latest training ID will be used\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		cmdLearn.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nReplay learn and validation for the specified training set.\n")
	}

	if len(os.Args) < 2 {
		topLevelUsage()
		os.Exit(10)
	}

	action := action(os.Args[1])
	if err := action.validate(); err != nil {
		fmt.Println(err)
		os.Exit(11)
	}

	switch action {

	case actionVersion:
		fmt.Printf("CQLizer %s\nbuild date: %s\nlast commit: %s\n", version.Version, version.BuildDate, version.GitCommit)
		return

	case actionServer:
		var trainingID int64
		var err error
		cmdServer.Parse(os.Args[2:])
		if cmdServer.Arg(1) != "" {
			trainingID, err = strconv.ParseInt(cmdServer.Arg(1), 10, 64)
			if err != nil {
				color.New(errColor).Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		conf := setupConfAndLogging(cmdServer, 0)
		runApiServer(conf, int(trainingID))

	case actionImport:
		cmdImport.Parse(os.Args[2:])
		conf := setupConfAndLogging(cmdImport, 0)
		runKontextImport(conf, cmdImport.Arg(1), *addToTrainingSet)

	case actionCorpsizes:
		cmdCorpsizes.Parse(os.Args[2:])
		conf := setupConfAndLogging(cmdCorpsizes, 0)
		runSizesImport(conf, cmdCorpsizes.Arg(1))

	case actionBenchmark:
		cmdBenchmark.Parse(os.Args[2:])
		conf := setupConfAndLogging(cmdBenchmark, 0)
		runBenchmark(conf, *overwriteAll)

	case actionReplay:
		cmdReplay.Parse(os.Args[2:])
		conf := setupConfAndLogging(cmdReplay, 0)
		trainingID := parseTrainingIdOrExit(cmdReplay.Arg(1))
		runTrainingReplay(conf, trainingID)

	case actionEvaluate:
		cmdEvaluate.Parse(os.Args[2:])
		conf := setupConfAndLogging(cmdEvaluate, 0)
		trainingID := parseTrainingIdOrExit(cmdEvaluate.Arg(1))
		runEvaluation(conf, trainingID, *numSamples, *sampleSize)

	case actionLearn:
		cmdLearn.Parse(os.Args[2:])
		conf := setupConfAndLogging(cmdLearn, 0)
		thr, err := strconv.ParseFloat(cmdLearn.Arg(1), 64)
		if err != nil {
			color.New(errColor).Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		runLearning(conf, thr)

	default:
		log.Fatal().Msgf("Unknown action %s", action)
	}

}
