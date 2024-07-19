package combo

import (
	"fmt"

	"github.com/agnivade/levenshtein"
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/models"
	"github.com/czcorpus/cqlizer/models/rf"
	"github.com/czcorpus/cqlizer/stats"
	randomforest "github.com/malaschitz/randomForest"
)

func Prediction(rfPredict [2]float64, qsPredict *stats.BestMatches, threshold float64) bool {
	if rfPredict[1] > 0.5 {
		return true
	}
	if rfPredict[1] > 0.35 && qsPredict.SmartBenchTime() > threshold && qsPredict.At(0).Distance == 0 {
		return true
	}
	return false
}

func EvaluateByMultimodel(
	records []stats.DBRecord,
	rfModel randomforest.Forest,
	threshold float64,
	statsDB *stats.Database,
	anyCorpus bool,
) (models.EvalResult, error) {
	var result models.EvalResult
	for i, item := range records {
		parsed, err := cql.ParseCQL("#", item.Query)
		if err != nil {
			return result, fmt.Errorf("failed to run evaluation: %w", err)
		}

		norm := parsed.Normalize()
		recs, err := statsDB.GetAllRecords(
			stats.ListFilter{}.
				SetBenchmarked(true).
				SetTrainingExcluded(false).
				SetWithNormalizedQuery(true).
				SetAnyCorpus(anyCorpus),
		)
		if err != nil {
			return result, fmt.Errorf("failed to run evaluation: %w", err)
		}
		matches := stats.NewBestMatches(5)
		for _, rec := range recs {
			dist := levenshtein.ComputeDistance(rec.QueryNormalized, norm)
			item := rec
			matches.TryAdd(&item, dist)
		}

		features := rf.NewFeats()
		features.ImportFrom(parsed)
		rfPredict := rfModel.Vote(features.AsVector())
		votes := [2]float64{rfPredict[0], rfPredict[1]}
		predict := Prediction(votes, matches, threshold)
		actual := item.BenchTime >= threshold

		if predict && !actual {
			fmt.Println("false positive ---------------------------")
			fmt.Println("Q: ", item.Query)
			fmt.Println("prediction: ", matches.SmartBenchTime(), ", actual: ", item.BenchTime)
			matches.Print()
			result.FalsePositives++

		} else if predict && actual {
			result.TruePositives++

		} else if !predict && actual {
			fmt.Println("false negative ---------------------------")
			fmt.Println("Q: ", item.Query)
			fmt.Println("prediction: ", matches.SmartBenchTime(), ", actual: ", item.BenchTime)
			matches.Print()
			result.FalseNegatives++
		}
		result.TotalTests++
		if i%100 == 0 {
			fmt.Printf("processed %d records\n", i)
		}
	}
	return result, nil
}
