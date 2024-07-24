package ndw

import (
	"fmt"
	"math"

	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/models"
	"github.com/czcorpus/cqlizer/models/ndw/optimizer"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/rs/zerolog/log"
)

type NDWResult struct {
	models.EvalResult
	prediction float64
	totalError float64
}

func (r NDWResult) TotalError() float64 {
	return r.totalError
}

func (r NDWResult) Prediction() float64 {
	return r.prediction
}

func (r NDWResult) PrintOverview() {
	fmt.Printf("TotalError: %01.2f\n", r.totalError)
	fmt.Printf("Precision: %01.2f\n", r.Precision())
	fmt.Printf("Recall: %01.2f\n", r.Recall())
	fmt.Printf("FN: %d\n", r.EvalResult.FalseNegatives)
	fmt.Printf("FP: %d\n", r.EvalResult.FalsePositives)
	fmt.Printf("TP: %d\n", r.EvalResult.TruePositives)
	fmt.Printf("TN: %d\n", r.TotalTests-r.EvalResult.FalseNegatives-r.EvalResult.FalsePositives-r.EvalResult.TruePositives)
}

func Test(statsDB *stats.Database, threshold, ratioOfTrues float64, synCompat bool) error {
	rows, err := statsDB.MixBiasedTrainingList(threshold, ratioOfTrues, synCompat)
	if err != nil {
		return fmt.Errorf("failed to run prediction test: %w", err)
	}

	astMap := make(map[string]*cql.Query)
	queryDBG := make(map[string]string)
	// prepare AST for all queries:
	for _, row := range rows {
		parsed, err := cql.ParseCQL("#", row.Query)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse query, skipping")
			continue
		}
		astMap[row.ID] = parsed
		queryDBG[row.ID] = row.Query
	}

	fn := func(vec optimizer.Chromosome) optimizer.Result {

		var ans NDWResult
		for _, row := range rows {
			var params Params
			params.FromVec(vec)
			ast := astMap[row.ID]
			sm := Evaluate(ast, params)
			err = sm.Run()
			if err != nil {
				log.Error().Err(err).Str("query", row.Query).Msg("Failed to evaluate query, skipping")
				continue
			}
			result, err := sm.Peek()
			ans.prediction += result.AsFloat64()
			if err != nil {
				log.Error().Err(err).Str("query", row.Query).Msg("Failed to evaluate query, skipping")
				continue
			}

			//fmt.Println(row.Query, "\ttime: ", row.BenchTime, "\t: eval: ", result)
			pred := result.AsFloat64() > threshold
			actual := row.BenchTime > threshold
			if pred != actual {
				//	fmt.Println("pred: ", result.AsFloat64(), ", actual: ", row.BenchTime, ", thresh: ", threshold)
			}
			ans.totalError += math.Abs(result.AsFloat64() - row.BenchTime)
			if pred && !actual {
				ans.FalsePositives++

			} else if !pred && actual {
				ans.FalseNegatives++

			} else if pred && actual {
				ans.TruePositives++
			}
			ans.TotalTests++
		}
		ans.totalError /= float64(ans.TotalTests)
		return ans
	}

	//feats.Optimize(500, 50, fn)
	best := optimizer.Optimize(
		1000, paramsVectorSize, 30, 29, 0.15, fn)
	fmt.Println("ROWS USED: ", len(rows))
	fmt.Println("normalized score: ", best.Result.TotalError()/float64(len(rows)))
	return nil

}
