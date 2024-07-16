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

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/feats"
	"github.com/czcorpus/cqlizer/feats/heatmap"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/gin-gonic/gin"
	randomforest "github.com/malaschitz/randomForest"
)

type Actions struct {
	StatsDB *stats.Database
	rfModel randomforest.Forest
}

func (a *Actions) ParseQuery(ctx *gin.Context) {
	q := ctx.Query("q")
	parsed, err := cql.ParseCQL("#", q)
	if err != nil {
		uniresp.RespondWithErrorJSON(
			ctx,
			fmt.Errorf("failed to parse query \u25B6 %w", err),
			http.StatusUnprocessableEntity,
		)
		return
	}
	//feats.CalculateEffect(parsed)
	uniresp.WriteJSONResponse(ctx.Writer, parsed)
}

func (a *Actions) AnalyzeQuery(ctx *gin.Context) {
	q := ctx.Query("q")
	parsed, err := cql.ParseCQL("#", q)
	if err != nil {
		uniresp.RespondWithErrorJSON(
			ctx,
			fmt.Errorf("failed to parse query \u25B6 %w", err),
			http.StatusUnprocessableEntity,
		)
		return
	}
	features := feats.NewRecord()
	features.ImportFrom(parsed)
	features.ExportHeatmapToFile(fmt.Sprintf("./data/query-%s.png", stats.IdempotentID(time.Now(), q)))

	ans := a.rfModel.Vote(features.AsVector())

	uniresp.WriteJSONResponse(ctx.Writer, map[string]float64{"no": ans[0], "yes": ans[1]})
}

type storeQueryBody struct {
	Query      string  `json:"query"`
	CorpusName string  `json:"corpname"`
	ProcTime   float64 `json:"procTime"`
}

func (a *Actions) StoreQuery(ctx *gin.Context) {
	var data storeQueryBody
	if err := ctx.BindJSON(&data); err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusBadRequest)
		return
	}
	_, err := cql.ParseCQL("#", data.Query)
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusUnprocessableEntity)
		return
	}
	newID, err := a.StatsDB.AddRecord(stats.DBRecord{
		Query:           data.Query,
		Corpname:        data.CorpusName,
		Datetime:        time.Now().Unix(),
		ProcTime:        data.ProcTime,
		TrainingExclude: ctx.Query("add-to-training") != "1",
	})
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusInternalServerError)
		return
	}
	uniresp.WriteJSONResponse(ctx.Writer, map[string]any{"newID": newID})
}

func (a *Actions) Features(ctx *gin.Context) {
	q := ctx.Query("q")
	parsed, err := cql.ParseCQL("#", q)
	if err != nil {
		uniresp.RespondWithErrorJSON(
			ctx,
			fmt.Errorf("failed to parse query \u25B6 %w", err),
			http.StatusUnprocessableEntity,
		)
		return
	}
	features := feats.NewRecord()
	var result [][]float64

	if ctx.Query("full") == "1" {
		fullFeats := features.ImportFromGetFullFeats(parsed)
		rows, cols := fullFeats.Dims()
		rawData := fullFeats.RawMatrix().Data
		result = make([][]float64, rows)
		for i := 0; i < rows; i++ {
			result[i] = rawData[i*cols : (i+1)*cols]
		}

	} else {
		features.ImportFrom(parsed)
		rows, cols := features.Matrix().Dims()
		rawData := features.Matrix().RawMatrix().Data
		result = make([][]float64, rows)
		for i := 0; i < rows; i++ {
			result[i] = rawData[i*cols : (i+1)*cols]
		}
	}
	ctx.Header("content-type", "image/png")
	heatmap.GenerateHeatmapToWriter(result, ctx.Writer, 20, heatmap.Percentile)

}
