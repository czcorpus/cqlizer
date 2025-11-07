// Copyright 2025 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2025 Department of Linguistics,
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

package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/czcorpus/cqlizer/eval/nn"
	"github.com/czcorpus/cqlizer/eval/rf"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// -----

type apiServer struct {
	conf       *cnf.Conf
	server     *http.Server
	rfEnsemble []ensembleModel
}

func (api *apiServer) Start(ctx context.Context) {
	if !api.conf.Logging.Level.IsDebugMode() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(logging.GinMiddleware())
	engine.Use(uniresp.AlwaysJSONContentType())
	engine.Use(corsMiddleware(api.conf))
	engine.NoMethod(uniresp.NoMethodHandler)
	engine.NoRoute(uniresp.NotFoundHandler)

	engine.GET("/test", api.handleTestPage)
	engine.GET("/cql/:corpusId", api.handleEvalCQL)
	engine.GET("/cql", api.handleEvalCQL)
	engine.GET("/simple/:corpusId", api.handleEvalSimple)
	engine.GET("/simple", api.handleEvalSimple)

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

func Run(
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
		var mlModel eval.MLModel
		var err error

		switch rfc.ModelType {
		case "rf":
			mlModel, err = rf.LoadFromFile(rfc.ModelPath)
		case "nn":
			mlModel, err = nn.LoadFromFile(rfc.ModelPath)
		default:
			err = fmt.Errorf("unkown model type '%s' for %s", rfc.ModelType, rfc.ModelPath)
		}
		if err != nil {
			log.Fatal().Err(err).Msg("Error loading RF model")
			return
		}
		mlModel.SetClassThreshold(rfc.VoteThreshold)

		log.Info().
			Float64("slowQueryTime", rfc.VoteThreshold).
			Msg("loaded RF model")
		server.rfEnsemble = append(
			server.rfEnsemble,
			ensembleModel{
				model:     mlModel,
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
