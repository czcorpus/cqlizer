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

	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/predict"
	"github.com/gin-gonic/gin"
)

// VersionInfo provides a detailed information about the actual build
type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitCommit string `json:"gitCommit"`
}

// ---------------------

type service interface {
	Start(ctx context.Context)
	Stop(ctx context.Context) error
}

// ------

type evaluation struct {
	CorpusSize  int    `json:"corpusSize"`
	Votes       []vote `json:"votes"`
	IsSlowQuery bool   `json:"isSlowQuery"`
	AltCorpus   string `json:"altCorpus,omitempty"`
}

type vote struct {
	Votes  []float64 `json:"votes"`
	Result int       `json:"result"`
}

// ------

type ensembleModel struct {
	model     eval.MLModel
	threshold float64
}

func (md ensembleModel) Predict(queryEval feats.QueryEvaluation) predict.Prediction {
	return md.model.Predict(queryEval)
}

// -----

func corsMiddleware(conf *cnf.Conf) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		var allowedOrigin string
		currOrigin := ctx.Request.Header.Get("Origin")
		for _, origin := range conf.CorsAllowedOrigins {
			if currOrigin == origin || origin == "*" {
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
