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
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/czcorpus/cnc-gokit/unireq"
	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/gin-gonic/gin"
)

func (api *apiServer) handleEvalSimple(ctx *gin.Context) {
	q := ctx.Query("q")
	defaultAttr := ctx.QueryArray("defaultAttr")
	cqlChunks := make([]string, len(defaultAttr))
	for i, da := range defaultAttr {
		cqlChunks[i] = fmt.Sprintf("%s=\"%s\"", da, q)
	}
	q = strings.Join(cqlChunks, " | ")
	api.evaluateRawQuery(ctx, q)

}

func (api *apiServer) handleEvalCQL(ctx *gin.Context) {
	q := ctx.Query("q")
	api.evaluateRawQuery(ctx, q)
}

func (api *apiServer) evaluateRawQuery(ctx *gin.Context, q string) {
	corpname := ctx.Param("corpusId")
	//aligned := ctx.QueryArray("aligned")
	var corpusInfo eval.CorpusProps
	var ok bool
	if corpname != "" {
		corpusInfo, ok = api.conf.CorporaProps[corpname]

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
		corpusInfo.Size, ok = unireq.GetURLIntArgOrFail(ctx, "corpusSize", 1000000000)
		if !ok {
			return
		}
		corpusInfo.Lang = ctx.Query("lang")
	}
	charProb := eval.GetCharProbabilityProvider(corpusInfo.Lang)
	queryEval, err := eval.NewQueryEvaluation(q, float64(corpusInfo.Size), 3, charProb)
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusInternalServerError)
		return
	}
	predictions := make([]vote, 0, len(api.rfEnsemble))
	for _, md := range api.rfEnsemble {
		predictions = append(predictions, vote{Value: float64(md.Predict(queryEval).PredictedClass) / 100, Threshold: md.threshold})
	}

	var votesFor int
	for i, pred := range predictions {
		if pred.Value >= api.rfEnsemble[i].threshold {
			votesFor++
		}
	}
	resp := evaluation{
		CorpusSize:  corpusInfo.Size,
		Votes:       predictions,
		IsSlowQuery: votesFor > int(math.Floor(float64(len(api.rfEnsemble))/2)),
	}

	uniresp.WriteJSONResponse(ctx.Writer, resp)
}
