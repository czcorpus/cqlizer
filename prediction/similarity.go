package prediction

import (
	"fmt"

	"github.com/agnivade/levenshtein"
	"github.com/czcorpus/cqlizer/cql"
	"github.com/czcorpus/cqlizer/feats"
	"github.com/czcorpus/cqlizer/stats"
	randomforest "github.com/malaschitz/randomForest"
)

type QueryEstimation struct {
	Problematic         bool `json:"problematic"`
	ConfidentEstimation bool `json:"confidentEstimation"`
}

func CombinedEstimation(rfPredict [2]float64, qsPredict, threshold float64) bool {
	if rfPredict[1] > 0.6 || qsPredict > threshold*1.2 {
		return true
	}
	if rfPredict[1] > 0.4 && qsPredict > threshold/20.0 {
		return true
	}
	return false
}

func EvaluateBySimilarity(
	records []stats.DBRecord,
	threshold float64,
	statsDB *stats.Database,
	anyCorpus bool,
) (EvalResult, error) {
	var result EvalResult
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
		predict := matches.SmartBenchTime() >= threshold
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

func EvaluateByMultimodel(
	records []stats.DBRecord,
	rfModel randomforest.Forest,
	threshold float64,
	statsDB *stats.Database,
	anyCorpus bool,
) (EvalResult, error) {
	var result EvalResult
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
		qsPredict := matches.SmartBenchTime()
		actual := item.BenchTime >= threshold

		features := feats.NewRecord()
		features.ImportFrom(parsed)
		rfPredict := rfModel.Vote(features.AsVector())
		votes := [2]float64{rfPredict[0], rfPredict[1]}
		predict := CombinedEstimation(votes, qsPredict, threshold)

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
