package apiserver

import (
	"context"

	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/gin-gonic/gin"
)

type service interface {
	Start(ctx context.Context)
	Stop(ctx context.Context) error
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

func corsMiddleware(conf *cnf.Conf) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		var allowedOrigin string
		currOrigin := ctx.Request.Header.Get("Origin")
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
