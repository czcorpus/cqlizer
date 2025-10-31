package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cnc-gokit/unireq"
	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type service interface {
	Start(ctx context.Context)
	Stop(ctx context.Context) error
}

// ----------------------

func getRequestOrigin(ctx *gin.Context) string {
	currOrigin, ok := ctx.Request.Header["Origin"]
	if ok {
		return currOrigin[0]
	}
	return ""
}

func CORSMiddleware(conf *cnf.Conf) gin.HandlerFunc {
	return func(ctx *gin.Context) {

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
		ctx.Next()
	}
}

// ------

type evaluation struct {
	CorpusSize  int    `json:"corpusSize"`
	Votes       []vote `json:"votes"`
	IsSlowQuery bool   `json:"isSlowQuery"`
}

type vote struct {
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
}

// ------

type ensembleModel struct {
	model     *eval.RFModel
	threshold float64
}

func (md ensembleModel) Predict(queryEval eval.QueryEvaluation) float64 {
	return md.model.Predict(queryEval)
}

// -----

type apiServer struct {
	conf       *cnf.Conf
	server     *http.Server
	rfEnsemble []ensembleModel
}

func (api *apiServer) handleEval(ctx *gin.Context) {
	q := ctx.Query("q")
	corpname := ctx.Param("corpusId")
	var corpusSize int
	var ok bool
	if corpname != "" {
		corpusSize, ok = api.conf.CorporaSizes[corpname]

		if !ok {
			uniresp.RespondWithErrorJSON(
				ctx, fmt.Errorf("corpus not found"), http.StatusNotFound,
			)
			return
		}

		if ctx.Query("corpusSize") != "" {
			uniresp.RespondWithErrorJSON(
				ctx, fmt.Errorf("cannot specify corpusSize for a concrete corpus"), http.StatusBadRequest,
			)
			return
		}

	} else {
		corpusSize, ok = unireq.GetURLIntArgOrFail(ctx, "corpusSize", 1000000000)
		if !ok {
			return
		}
	}
	queryEval, err := eval.NewQueryEvaluation(q, float64(corpusSize), 3)
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusInternalServerError)
		return
	}
	predictions := make([]vote, 0, len(api.rfEnsemble))
	for _, md := range api.rfEnsemble {
		predictions = append(predictions, vote{Value: md.Predict(queryEval), Threshold: md.threshold})
	}

	var votesFor int
	for i, pred := range predictions {
		if pred.Value >= api.rfEnsemble[i].threshold {
			votesFor++
		}
	}
	resp := evaluation{
		CorpusSize:  corpusSize,
		Votes:       predictions,
		IsSlowQuery: votesFor > int(math.Floor(float64(len(api.rfEnsemble))/2)),
	}

	uniresp.WriteJSONResponse(ctx.Writer, resp)
}

func (api *apiServer) Start(ctx context.Context) {
	if !api.conf.Logging.Level.IsDebugMode() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(logging.GinMiddleware())
	engine.Use(uniresp.AlwaysJSONContentType())
	engine.Use(CORSMiddleware(api.conf))
	engine.NoMethod(uniresp.NoMethodHandler)
	engine.NoRoute(uniresp.NotFoundHandler)

	engine.GET("/evaluate/:corpusId", api.handleEval)
	engine.GET("/evaluate", api.handleEval)

	log.Info().Msgf("starting to listen at %s:%d", api.conf.ListenAddress, api.conf.ListenPort)
	api.server = &http.Server{
		Handler:      engine,
		Addr:         fmt.Sprintf("%s:%d", api.conf.ListenAddress, api.conf.ListenPort),
		WriteTimeout: time.Duration(api.conf.ServerWriteTimeoutSecs) * time.Second,
		ReadTimeout:  time.Duration(api.conf.ServerReadTimeoutSecs) * time.Second,
	}
	go func() {
		if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()
}

func (api *apiServer) Stop(ctx context.Context) error {
	log.Warn().Msg("shutting down CQLizer HTTP API server")
	return api.server.Shutdown(ctx)
}

// -------------------------

func runApiServer(
	ctx context.Context,
	conf *cnf.Conf,
) {

	server := &apiServer{
		conf:       conf,
		rfEnsemble: make([]ensembleModel, 0, len(conf.RFEnsemble)),
	}

	for _, rfc := range conf.RFEnsemble {
		if rfc.Disable {
			continue
		}
		rfModel, err := eval.LoadRFModelFromFile(rfc.ModelPath)
		if err != nil {
			log.Fatal().Err(err).Msg("Error loading RF model")
			return
		}
		log.Info().
			Float64("slowQueryPercentile", rfModel.SlowQueriesPercentile).
			Msg("loaded RF model")
		server.rfEnsemble = append(
			server.rfEnsemble,
			ensembleModel{
				model:     rfModel,
				threshold: rfc.VoteThreshold,
			},
		)
	}

	services := []service{server}
	for _, m := range services {
		m.Start(ctx)
	}
	<-ctx.Done()
	log.Warn().Msg("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, s := range services {
		wg.Add(1)
		go func(srv service) {
			defer wg.Done()
			if err := srv.Stop(shutdownCtx); err != nil {
				log.Error().Err(err).Type("service", srv).Msg("Error shutting down service")
			}
		}(s)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("Graceful shutdown completed")
	case <-shutdownCtx.Done():
		log.Warn().Msg("Shutdown timed out")
	}
}
