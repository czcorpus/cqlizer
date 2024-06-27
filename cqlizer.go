// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
//                Faculty of Arts, Charles University
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
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/czcorpus/cnc-gokit/collections"
	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/logproc"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/gin-gonic/gin"
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

func runKontextImport(conf *cnf.Conf, path string) {
	err := logproc.ImportLog(conf, path)
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

func runRecalcPercentiles(conf *cnf.Conf) {
	db, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		fmt.Println("FAILED: ", err)
	}
	err = db.Init()
	if err != nil {
		fmt.Println("FAILED: ", err)
	}
	err = db.RecalculatePercentiles()
	if err != nil {
		fmt.Println("FAILED: ", err)
	}
}

func runApiServer(
	conf *cnf.Conf,
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
	}

	err = statsDB.Init()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize working database")
		syscallChan <- syscall.SIGTERM
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(additionalLogEvents())
	engine.Use(logging.GinMiddleware())
	engine.Use(uniresp.AlwaysJSONContentType())
	engine.Use(CORSMiddleware(conf))
	engine.NoMethod(uniresp.NoMethodHandler)
	engine.NoRoute(uniresp.NotFoundHandler)

	cqlActions := Actions{StatsDB: statsDB}

	engine.GET(
		"/analyze", cqlActions.AnalyzeQuery)

	engine.PUT(
		"/query", cqlActions.StoreQuery)

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

func main() {
	version := VersionInfo{
		Version:   cleanVersionInfo(version),
		BuildDate: cleanVersionInfo(buildDate),
		GitCommit: cleanVersionInfo(gitCommit),
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "CQLIZER - A CQL toolbox\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t%s [options] start [config.json]\n\t", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "%s [options] version\n", filepath.Base(os.Args[0]))
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
		runApiServer(conf, syscallChan, exitEvent)
	case "import":
		runKontextImport(conf, flag.Arg(2))
	case "corpsizes":
		runSizesImport(conf, flag.Arg(2))
	case "percentiles":
		runRecalcPercentiles(conf)
	default:
		log.Fatal().Msgf("Unknown action %s", action)
	}

}
