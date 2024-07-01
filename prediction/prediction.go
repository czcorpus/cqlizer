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

type TestingItem struct {
	Rec   stats.DBRecord
	Feats feats.Record
	Vote  int
}

type Engine struct {
	conf    *cnf.Conf
	statsDB *stats.Database
}

func (eng *Engine) Test2(threshold float64) error {

	rows, err := eng.statsDB.GetAllRecords(false)
	if err != nil {
		return fmt.Errorf("failed to run prediction test: %w", err)
	}
	scoreThresh := 0.7
	var numFalsePositives, numTruePositives, numFalseNegatives int

	for _, row := range rows {
		parsed, err := cql.ParseCQL("#", row.Query)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse query, skipping")
			continue
		}
		sm := feats.Evaluate(parsed)
		err = sm.Run()
		if err != nil {
			log.Error().Err(err).Str("query", row.Query).Msg("Failed to evaluate query, skipping")
			continue
		}
		result, err := sm.Peek()
		if err != nil {
			log.Error().Err(err).Str("query", row.Query).Msg("Failed to evaluate query, skipping")
			continue
		}

		//fmt.Println(row.Query, "\ttime: ", row.BenchTime, "\t: eval: ", result)
		pred := result.Value >= scoreThresh
		actual := row.BenchTime > threshold
		if pred && !actual {
			fmt.Println(
				"FALSE POSITIVE, query: ", row.Query, ", time: ", row.BenchTime, ", score: ", result.Value, ", predict: ", pred, ", actual: ", row.BenchTime > threshold)
			numFalsePositives++

		} else if !pred && actual {
			fmt.Println(
				"FALSE NEGATIVE, query: ", row.Query, ", time: ", row.BenchTime, ", score: ", result.Value, ", predict: ", pred, ", actual: ", row.BenchTime > threshold)
			numFalseNegatives++

		} else if pred && actual {
			numTruePositives++
		}
	}
	fmt.Println("============================================================\n\n")
	fmt.Printf("total tested items: %d\n", len(rows))
	fmt.Printf("precision: %01.2f\n", float64(numTruePositives)/float64(numTruePositives+numFalsePositives))
	fmt.Printf("recall: %01.2f\n", float64(numTruePositives)/float64(numTruePositives+numFalseNegatives))
	return nil
}

func (eng *Engine) Test(threshold float64) error {

	xData := [][]float64{}
	yData := []int{}

	rows, err := eng.statsDB.GetAllRecords(false)
	if err != nil {
		return fmt.Errorf("failed to run prediction test: %w", err)
	}
	testingData := make([]TestingItem, 0, 1000)
	for _, row := range rows {
		p, err := cql.ParseCQL("#", row.Query)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse query, skipping")
			continue
		}
		var fts feats.Record
		fts.ImportFrom(p, 200000000) // TODO

		if rand.Float64() < 0.75 {

			xData = append(xData, fts.AsVector())
			res := 0
			if row.BenchTime > threshold {
				res = 1
			}
			yData = append(yData, res)

		} else {
			testingData = append(testingData, TestingItem{
				Rec:   row,
				Feats: fts,
				Vote:  -1,
			})
		}
	}

	forest := randomforest.Forest{}
	forest.Data = randomforest.ForestData{X: xData, Class: yData}
	forest.Train(1200)
	var numFalsePositives, numTruePositives, numFalseNegatives int
	for _, tst := range testingData {
		ans := forest.Vote(tst.Feats.AsVector())
		q := tst.Rec.Query
		if len(q) > 100 {
			q = string([]rune(q)[:100])
		}
		pred := false
		if ans[1] > ans[0] {
			pred = true
		}
		actual := tst.Rec.BenchTime > threshold
		if pred && !actual {
			fmt.Println(
				"FALSE POSITIVE, query: ", q, ", time: ", tst.Rec.BenchTime, ", predict ", pred, ", actual: ", tst.Rec.BenchTime > threshold)
			numFalsePositives++

		} else if !pred && actual {
			fmt.Println(
				"FALSE NEGATIVE, query: ", q, ", time: ", tst.Rec.BenchTime, ", predict ", pred, ", actual: ", tst.Rec.BenchTime > threshold)
			numFalseNegatives++

		} else if pred && actual {
			numTruePositives++
		}

	}
	fmt.Println("============================================================\n\n")
	fmt.Printf("total tested items: %d\n", len(testingData))
	fmt.Printf("precision: %01.2f\n", float64(numTruePositives)/float64(numTruePositives+numFalsePositives))
	fmt.Printf("recall: %01.2f\n", float64(numTruePositives)/float64(numTruePositives+numFalseNegatives))
	return nil
}

func NewEngine(conf *cnf.Conf, statsDB *stats.Database) *Engine {
	return &Engine{
		conf:    conf,
		statsDB: statsDB,
	}
}
