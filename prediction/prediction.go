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

		if rand.Float64() < 0.5 {

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
	forest.Train(1000)
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
		fmt.Println(
			"query: ", q, ", time: ", tst.Rec.BenchTime,
			", predict ", pred, ", actual: ", tst.Rec.BenchTime > threshold)
	}
	return nil
}

func NewEngine(conf *cnf.Conf, statsDB *stats.Database) *Engine {
	return &Engine{
		conf:    conf,
		statsDB: statsDB,
	}
}
