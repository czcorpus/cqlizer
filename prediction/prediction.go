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

package prediction

import (
	"fmt"
	"math/rand"

	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/feats"
	"github.com/czcorpus/cqlizer/stats"
	randomforest "github.com/malaschitz/randomForest"
	"github.com/rs/zerolog/log"
)

type Engine struct {
	conf    *cnf.Conf
	statsDB *stats.Database
}

func (eng *Engine) Train(threshold float64) error {
	rows, err := eng.statsDB.GetAllRecords(false)
	if err != nil {
		return fmt.Errorf("failed to run prediction test: %w", err)
	}

	training := make([]stats.DBRecord, 0, 1000)
	eval := make([]stats.DBRecord, 0, 500)
	for _, row := range rows {
		if rand.Float64() < 0.75 {
			training = append(training, row)

		} else {
			eval = append(eval, row)
		}
	}
	_, err = eng.train(threshold, training, eval, true)
	return err
}

func (eng *Engine) getFeats(q string) (feats.Record, error) {
	p, err := cql.ParseCQL("#", q)
	if err != nil {
		return feats.Record{}, fmt.Errorf("failed to get feats of the query: %w", err)

	}
	fts := feats.NewRecord()
	fts.ImportFrom(p)
	return fts, nil
}

func (eng *Engine) TrainReplay(
	threshold float64,
	training []stats.DBRecord,
	eval []stats.TrainingResult,
) (randomforest.Forest, error) {
	eval2 := make([]stats.DBRecord, len(eval))
	for i, v := range eval {
		eval2[i] = stats.DBRecord{
			ID:        v.QueryID,
			Query:     v.Query,
			BenchTime: v.BenchTime,
		}
	}
	return eng.train(threshold, training, eval2, false)
}

func (eng *Engine) train(
	threshold float64,
	training []stats.DBRecord,
	eval []stats.DBRecord,
	storeToDB bool,
) (randomforest.Forest, error) {

	xData := [][]float64{}
	yData := []int{}

	var trainingID int
	var err error
	if storeToDB {
		trainingID, err = eng.statsDB.CreateNewTraining(threshold)
	}
	if err != nil {
		return randomforest.Forest{}, fmt.Errorf("failed to run prediction test: %w", err)
	}

	for _, row := range training {
		if storeToDB {
			if err := eng.statsDB.SetTrainingQuery(trainingID, row.ID); err != nil {
				return randomforest.Forest{}, fmt.Errorf("failed to prepare training: %w", err)
			}
		}

		fts, err := eng.getFeats(row.Query)
		if err != nil {
			log.Error().Err(err).Str("queryId", row.ID).Msg("failed to use query for training - skipping")
			continue
		}
		xData = append(xData, fts.AsVector())

		res := 0
		if row.BenchTime >= threshold {
			res = 1
		}
		yData = append(yData, res)
	}
	forest := randomforest.Forest{}
	forest.Data = randomforest.ForestData{X: xData, Class: yData}
	forest.Train(1200)
	fmt.Println("training done")
	fmt.Printf("total training items: %d\n", len(training))

	var numFalsePositives, numTruePositives, numFalseNegatives int

	for _, row := range eval {
		fts, err := eng.getFeats(row.Query)
		if err != nil {
			log.Error().Err(err).Str("queryId", row.ID).Msg("failed to use query for validation - skipping")
			continue
		}

		ans := forest.Vote(fts.AsVector())
		q := row.Query
		if len(q) > 100 {
			q = string([]rune(q)[:100])
		}
		pred := false
		if ans[1] > ans[0] {
			pred = true
		}
		actual := row.BenchTime > threshold
		if pred && !actual {
			fmt.Println("FALSE POSITIVE, query: ", q, ", time: ", row.BenchTime)
			fmt.Printf("  prediction - yes: %1.2f, no: %1.2f\n", ans[1], ans[0])
			numFalsePositives++

		} else if !pred && actual {
			fmt.Println("FALSE NEGATIVE, query: ", q, ", time: ", row.BenchTime)
			fmt.Printf("  prediction - yes: %1.2f, no: %1.2f\n", ans[1], ans[0])
			numFalseNegatives++

		} else if pred && actual {
			numTruePositives++
		}
		if storeToDB {
			if err := eng.statsDB.SetValidationQuery(trainingID, row.ID, pred); err != nil {
				if err := eng.statsDB.SetTrainingQuery(trainingID, row.ID); err != nil {
					return randomforest.Forest{}, fmt.Errorf("failed to store validation data: %w", err)
				}
			}
		}
	}
	fmt.Println("============================================================\n\n")
	fmt.Printf("total validated items: %d\n", len(eval))
	fmt.Printf("precision: %01.2f\n", float64(numTruePositives)/float64(numTruePositives+numFalsePositives))
	fmt.Printf("recall: %01.2f\n", float64(numTruePositives)/float64(numTruePositives+numFalseNegatives))
	fmt.Println("training ID: ", trainingID)
	return forest, nil
}

func NewEngine(conf *cnf.Conf, statsDB *stats.Database) *Engine {
	return &Engine{
		conf:    conf,
		statsDB: statsDB,
	}
}
