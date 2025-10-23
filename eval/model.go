package eval

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
)

type QueryStatsRecord struct {
	Corpus     string  `json:"corpus"`
	CorpusSize int64   `json:"corpusSize"`
	TimeProc   float64 `json:"timeProc"`
	Query      string  `json:"query"`
}

func (rec QueryStatsRecord) GetCQL() string {
	if strings.HasPrefix(rec.Query, "q") {
		return rec.Query[1:]
	}
	tmp := strings.SplitN(rec.Query, ",", 2)
	if len(tmp) > 1 {
		return tmp[1]
	}
	return rec.Query
}

// ----------------------------

type BasicModel struct {
	Evaluations []QueryEvaluation
	BinMidpoint float64 // Midpoint of each bin for prediction
	MidpointIdx int
}

func (model *BasicModel) computeThreshold() (int, float64) {
	if len(model.Evaluations) == 0 {
		return 0, 0
	}
	perc90Idx := int(math.Ceil(float64(len(model.Evaluations)) * 0.97))
	return perc90Idx, model.Evaluations[perc90Idx].ProcTime
}

func (model *BasicModel) BalanceSample() []QueryEvaluation {
	slices.SortFunc(model.Evaluations, func(v1, v2 QueryEvaluation) int {
		if v1.ProcTime < v2.ProcTime {
			return -1
		}
		return 1
	})
	model.MidpointIdx, model.BinMidpoint = model.computeThreshold()
	fmt.Println("-=============== BALANCED MODEL ====")
	fmt.Printf("SHOULD BALANCE STUFF, num valid: %d, num total: %d\n", len(model.Evaluations)-model.MidpointIdx, len(model.Evaluations))
	numPositive := len(model.Evaluations) - model.MidpointIdx
	balEval := make([]QueryEvaluation, numPositive*2)
	for i := 0; i < numPositive; i++ {
		balEval[i] = model.Evaluations[rand.IntN(model.MidpointIdx)]
	}
	for i := 0; i < numPositive; i++ {
		balEval[i+numPositive] = model.Evaluations[model.MidpointIdx+i]
	}
	oldEvals := model.Evaluations
	model.Evaluations = balEval
	return oldEvals
}

func (model *BasicModel) ProcessEntry(entry QueryStatsRecord) error {
	if entry.CorpusSize == 0 {
		return fmt.Errorf("zero processing time or corpus size")
	}
	if entry.TimeProc == 0 {
		entry.TimeProc = 0.01
	}

	// Parse the CQL query and create evaluation with corpus size
	eval, err := NewQueryEvaluation(entry.GetCQL(), float64(entry.CorpusSize), entry.TimeProc)
	if err != nil {
		errMsg := err.Error()
		if utf8.RuneCountInString(errMsg) > 80 {
			errMsg = string([]rune(errMsg)[:80])
		}
		log.Warn().Err(fmt.Errorf(errMsg)).Str("query", entry.GetCQL()).Msg("Warning: Failed to parse query")
		return nil // Skip unparseable queries
	}

	model.Evaluations = append(model.Evaluations, eval)

	return nil
}

func (model *BasicModel) Evaluate() {
	RunModel(model.Evaluations)
}

func (model *BasicModel) PrecisionAndRecall(rfModel *RFModel, threshold float64) {
	numTruePositives := 0
	numRelevant := 0
	numRetrieved := 0

	// In the group below, "actual" is always FALSE
	for i := 0; i < model.MidpointIdx; i++ {
		//actual := model.Evaluations[i].ProcTime < rfModel.BinMidpoint
		predicted := rfModel.Predict(model.Evaluations[i]) >= threshold
		if predicted {
			// = false positive
			numRetrieved++
		}
	}

	// In the group below, "actual" is always TRUE
	for i := model.MidpointIdx; i < len(model.Evaluations); i++ {
		actual := model.Evaluations[i].ProcTime >= model.BinMidpoint
		predicted := rfModel.Predict(model.Evaluations[i]) >= threshold
		numRelevant++
		if predicted {
			numRetrieved++
		}
		if actual == predicted {
			numTruePositives++
		}
	}

	precision := float64(numTruePositives) / float64(numRetrieved)
	recall := float64(numTruePositives) / float64(numRelevant)
	beta := 1.0
	fbeta := 0.0
	if precision+recall > 0 {
		betaSquared := beta * beta
		fbeta = (1 + betaSquared) * (precision * recall) / (betaSquared*precision + recall)
	}

	fmt.Printf("precision: %.2f, recall: %.2f, f-beta: %.2f\n", precision, recall, fbeta)

}

// EvaluateWithRF trains a Random Forest model instead of Huber regression
func (model *BasicModel) EvaluateWithRF(numTrees int, threshold float64, testData []QueryEvaluation, outputPath string) error {
	if len(model.Evaluations) == 0 {
		return fmt.Errorf("no training data available")
	}

	fmt.Printf("Training Random Forest with %d trees and voting threshold %.2f\n", numTrees, threshold)
	fmt.Printf("Training data: %d samples\n", len(model.Evaluations))

	rfModel := NewRFModel()
	if err := rfModel.Train(model, numTrees); err != nil {
		return fmt.Errorf("RF training failed: %w", err)
	}

	fmt.Println("\nRandom Forest trained successfully!")
	fmt.Printf("Bin midpoint: %v\n", model.BinMidpoint)

	model.Evaluations = testData

	maxSamples := 30
	if len(model.Evaluations) < maxSamples {
		maxSamples = len(model.Evaluations)
	}

	// Test predictions on training data (for diagnostic purposes)
	fmt.Println("\nSample predictions on training data:")

	fmt.Println("negative examples test: ")
	for i := 0; i < maxSamples; i++ {
		randomIdx := rand.IntN(model.MidpointIdx)
		predicted := rfModel.Predict(model.Evaluations[randomIdx])
		actual := model.Evaluations[randomIdx].ProcTime < model.BinMidpoint
		fmt.Printf(
			"       %d, match: %t, vote NO: %.2f (time: %.2f)\n",
			randomIdx, actual == (predicted < threshold), 1-predicted, model.Evaluations[randomIdx].ProcTime,
		)
	}

	fmt.Println("POSITIVE examples test: ")
	for i := 0; i < maxSamples; i++ {
		randomIdx := rand.IntN(len(model.Evaluations)-model.MidpointIdx) + model.MidpointIdx
		predicted := rfModel.Predict(model.Evaluations[randomIdx])
		actual := model.Evaluations[randomIdx].ProcTime >= model.BinMidpoint
		fmt.Printf(
			"       %d, match: %t, vote YES: %.2f (time: %.2f)\n",
			randomIdx, actual == (predicted >= threshold), predicted, model.Evaluations[randomIdx].ProcTime,
		)
	}

	model.PrecisionAndRecall(rfModel, threshold)

	// Note: Saving is not fully supported by the randomForest package
	// This is just metadata - the actual model needs to be retrained
	if outputPath != "" {
		fmt.Printf("\nNote: github.com/malaschitz/randomForest does not support full model serialization.\n")
		fmt.Printf("You would need to retrain the model or switch to a different package for production use.\n")
	}

	return nil
}
