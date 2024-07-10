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

	"github.com/czcorpus/cnc-gokit/maths"
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
	var listFilter stats.ListFilter
	rows, err := eng.statsDB.GetAllRecords(
		listFilter.
			SetBenchmarked(true).
			SetTrainingExcluded(false),
	)
	if err != nil {
		return fmt.Errorf("failed to run prediction test \u25B6 %w", err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("no data suitable for learning and validation found")
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
		return feats.Record{}, fmt.Errorf("failed to get feats of the query \u25B6 %w", err)

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

	if len(training) == 0 {
		panic("missing training data")
	}
	if len(eval) == 0 {
		panic("missing evaluation data")
	}

	var trainingID int
	var err error
	if storeToDB {
		trainingID, err = eng.statsDB.CreateNewTraining(threshold)
	}
	if err != nil {
		return randomforest.Forest{}, fmt.Errorf("failed to run prediction test \u25B6 %w", err)
	}

	for _, row := range training {
		if storeToDB {
			if err := eng.statsDB.SetTrainingQuery(trainingID, row.ID); err != nil {
				return randomforest.Forest{}, fmt.Errorf("failed to prepare training \u25B6 %w", err)
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

	result, err := eng.Evaluate(
		forest,
		eval,
		threshold,
		func(itemID string, prediction bool) error {
			if storeToDB {
				if err := eng.statsDB.SetValidationQuery(trainingID, itemID, prediction); err != nil {
					if err := eng.statsDB.SetTrainingQuery(trainingID, itemID); err != nil {
						return fmt.Errorf("failed to store validation data \u25B6 %w", err)
					}
				}
			}
			return nil
		},
	)
	if err != nil {
		return forest, fmt.Errorf("failed to perform Eval \u25B6 %w", err)
	}
	fmt.Println("============================================================")
	fmt.Print("\n\n")
	fmt.Printf("total validated items: %d\n", len(eval))
	fmt.Printf("precision: %01.2f\n", result.Precision())
	fmt.Printf("recall: %01.2f\n", result.Recall())
	return forest, nil
}

func (eng *Engine) Evaluate(
	forest randomforest.Forest,
	eval []stats.DBRecord,
	threshold float64,
	onEvalItem func(itemID string, prediction bool) error,
) (EvalResult, error) {
	var ans EvalResult

	for _, row := range eval {
		fts, err := eng.getFeats(row.Query)
		if err != nil {
			log.Error().Err(err).Str("queryId", row.ID).Msg("failed to use query for validation - skipping")
			continue
		}

		votes := forest.Vote(fts.AsVector())
		q := row.Query
		qr := []rune(q)
		if len(qr) > 100 {
			q = string([]rune(q)[:100])
		}
		pred := votes[1] > votes[0]
		actual := row.BenchTime > threshold
		if pred && !actual {
			log.Debug().
				Float64("votesFor", maths.RoundToN(votes[1], 2)).
				Float64("votesAgainst", maths.RoundToN(votes[0], 2)).
				Str("query", q).
				Float64("benchTime", maths.RoundToN(row.BenchTime, 2)).
				Msg("false positive evaluation")
			ans.FalsePositives++

		} else if !pred && actual {
			log.Debug().
				Float64("votesFor", maths.RoundToN(votes[1], 2)).
				Float64("votesAgainst", maths.RoundToN(votes[0], 2)).
				Str("query", q).
				Float64("benchTime", maths.RoundToN(row.BenchTime, 2)).
				Msg("false negative evaluation")
			ans.FalseNegatives++

		} else if pred && actual {
			ans.TruePositives++
		}
		ans.TotalTests++
		if err := onEvalItem(row.ID, pred); err != nil {
			return ans, err
		}
	}
	return ans, nil
}

func NewEngine(conf *cnf.Conf, statsDB *stats.Database) *Engine {
	return &Engine{
		conf:    conf,
		statsDB: statsDB,
	}
}
