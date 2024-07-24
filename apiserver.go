package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/czcorpus/cnc-gokit/collections"
	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

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

func runApiServer(
	conf *cnf.Conf,
	trainingID int,
) {
	if !conf.LogLevel.IsDebugMode() {
		gin.SetMode(gin.ReleaseMode)
	}

	syscallChan := make(chan os.Signal, 1)
	signal.Notify(syscallChan, os.Interrupt)
	signal.Notify(syscallChan, syscall.SIGTERM)
	exitEvent := make(chan os.Signal)
	go func() {
		evt := <-syscallChan
		exitEvent <- evt
		close(exitEvent)
	}()

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

	threshold, err := statsDB.GetTrainingThreshold(trainingID)
	if err != nil {
		log.Error().Err(err).Msg("failed to start service")
		syscallChan <- syscall.SIGTERM
		return
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
		StatsDB:   statsDB,
		rfModel:   model,
		threshold: threshold,
	}

	engine.GET("/analyze", cqlActions.AnalyzeQuery)

	engine.GET("/analyze-verbose", cqlActions.AnalyzeQuery2)

	engine.GET("/parse", cqlActions.ParseQuery)

	engine.GET("/parse-ndw", cqlActions.ParseQueryNDW)

	engine.PUT("/query", cqlActions.StoreQuery)

	engine.GET("/normalize", cqlActions.Normalize)

	log.Info().Msg("Starting CQLizer API server")
	log.Info().
		Str("address", conf.ListenAddress).
		Int("port", conf.ListenPort).
		Msgf("starting to listen")
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
