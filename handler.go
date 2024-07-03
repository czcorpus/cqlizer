// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
//                Faculty of Arts, Charles University
//   This file is part of CQLIZER.
//
//  CQLIZER is free software: you can redistribute it and/or modify
//  it under the terms of the GNU General Public License as published by
//  the Free Software Foundation, either version 3 of the License, or
//  (at your option) any later version.
//
//  CQLIZER is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with CQLIZER.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/feats"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/gin-gonic/gin"
)

type Actions struct {
	StatsDB *stats.Database
}

func (a *Actions) AnalyzeQuery(ctx *gin.Context) {
	q := ctx.Query("q")
	parsed, err := cql.ParseCQL("#", q)
	if err != nil {
		uniresp.RespondWithErrorJSON(
			ctx,
			fmt.Errorf("failed to parse query: %w", err),
			http.StatusUnprocessableEntity,
		)
		return
	}
	/*
		var features feats.Record
		features.ImportFrom(parsed, a.StatsDB.GetCorpusSize(ctx.Query("corpname")))
		spew.Dump(features)
	*/
	params := feats.NewDefaultParams()
	(&params).FromVec(feats.CurrWinner)
	sm := feats.Evaluate(parsed, params)
	sm.PrintProgram()

	err = sm.Run()
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusBadRequest)
		return
	}
	final, err := sm.Peek()
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusBadRequest)
		return
	}
	fmt.Println("BENCH. TIME ESTIMATE: ", final.Value*60)

	row, err := a.StatsDB.GetQueryAvgBenchTime(q)
	if err != nil {
		fmt.Println("failed to load stored query: ", err)

	} else {
		fmt.Printf("TRUE AVG BENCH TIME: %01.3f", row)
	}

	uniresp.WriteJSONResponse(ctx.Writer, parsed)
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
	parsed, err := cql.ParseCQL("#", data.Query)
	fmt.Println("data: ", data)
	fmt.Println("parsed: ", parsed)
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusUnprocessableEntity)
		return
	}
	var features feats.Record
	features.ImportFrom(parsed, a.StatsDB.GetCorpusSize(data.CorpusName))
	newID, err := a.StatsDB.AddRecord(data.Query, data.CorpusName, features, time.Now(), data.ProcTime)
	if err != nil {
		uniresp.RespondWithErrorJSON(ctx, err, http.StatusInternalServerError)
		return
	}

	uniresp.WriteJSONResponse(ctx.Writer, map[string]any{"newID": newID})
}
