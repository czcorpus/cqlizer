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

package cql

import (
	"fmt"
	"net/http"

	"github.com/czcorpus/cnc-gokit/uniresp"
	"github.com/gin-gonic/gin"
)

type Actions struct {
}

func (a *Actions) AnalyzeQuery(ctx *gin.Context) {
	q := ctx.Query("q")
	parsed, err := ParseCQL("#", q)
	if err != nil {
		uniresp.RespondWithErrorJSON(
			ctx,
			fmt.Errorf("failed to parse query: %w", err),
			http.StatusUnprocessableEntity,
		)
		return
	}
	uniresp.WriteJSONResponse(ctx.Writer, parsed)
}
