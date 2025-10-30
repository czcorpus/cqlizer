package eval

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
)

func findKneeDistance(items []QueryEvaluation) (threshold float64, kneeIdx int) {

	n := len(items)
	if n < 2 {
		return items[n-1].ProcTime, 100.0
	}

	// Line from first to last point
	x1, y1 := 0.0, items[0].ProcTime
	x2, y2 := float64(n-1), items[n-1].ProcTime

	// Line equation coefficients: ax + by + c = 0
	a := y2 - y1
	b := x1 - x2
	c := x2*y1 - x1*y2

	normFactor := math.Sqrt(a*a + b*b)

	maxDist := 0.0
	kneeIdx = 0

	for i := 0; i < n; i++ {
		// Perpendicular distance from point to line
		dist := math.Abs(a*float64(i)+b*items[i].ProcTime+c) / normFactor
		if dist > maxDist {
			maxDist = dist
			kneeIdx = i
		}
	}

	threshold = items[kneeIdx].ProcTime
	return threshold, kneeIdx
}

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

	// SlowQueryPercentile specifies which percentile of queries (by time)
	// is considered as "slow times".
	// This value is the one user enters.
	SlowQueryPercentile float64

	// MidpointIdx is derived from SlowQueryPercentile and represents a sorted data index
	// from which SlowQueryPercentile starts.
	MidpointIdx int

	// BinMidpoint is the threshold time where SlowQueryPercentile starts. The value
	// is derived from SlowQueryPercentile
	BinMidpoint float64
}

func (model *BasicModel) computeThreshold() (int, float64) {
	if len(model.Evaluations) == 0 {
		return 0, 0
	}
	slowPercIdx := int(math.Ceil(float64(len(model.Evaluations)) * model.SlowQueryPercentile))
	return slowPercIdx, model.Evaluations[slowPercIdx].ProcTime
}

func (model *BasicModel) BalanceSample() []QueryEvaluation {
	slices.SortFunc(model.Evaluations, func(v1, v2 QueryEvaluation) int {
		if v1.ProcTime < v2.ProcTime {
			return -1
		}
		return 1
	})
	//model.MidpointIdx, model.BinMidpoint = model.computeThreshold()
	fmt.Println("-=============== BALANCED MODEL ====")
	model.BinMidpoint, model.MidpointIdx = findKneeDistance(model.Evaluations)
	fmt.Println("AUTOMATIC>>> threshold idx ", model.MidpointIdx, ", threshold time: ", model.BinMidpoint)
	fmt.Printf("SHOULD BALANCE STUFF, num valid: %d, num total: %d\n", len(model.Evaluations)-model.MidpointIdx, len(model.Evaluations))
	numPositive := len(model.Evaluations) - model.MidpointIdx
	balEval := make([]QueryEvaluation, numPositive*3)
	for i := 0; i < numPositive*2; i++ {
		balEval[i] = model.Evaluations[rand.IntN(model.MidpointIdx)]
	}
	for i := 0; i < numPositive; i++ {
		balEval[i+numPositive*2] = model.Evaluations[model.MidpointIdx+i]
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

	for i := 0; i < len(model.Evaluations); i++ {
		trulySlow := model.Evaluations[i].ProcTime >= model.BinMidpoint
		predictedSlow := rfModel.Predict(model.Evaluations[i]) >= threshold
		if trulySlow {
			numRelevant++
		}
		if predictedSlow {
			numRetrieved++

			if trulySlow {
				numTruePositives++
			}

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
func (model *BasicModel) EvaluateWithRF(
	numTrees int,
	slowQueriesPerc float64,
	votingThreshold float64,
	testData []QueryEvaluation,
	outputPath string,
) error {
	if len(model.Evaluations) == 0 {
		return fmt.Errorf("no training data available")
	}

	fmt.Printf("Training Random Forest with %d trees and slow queries percentile %.2f\n", numTrees, slowQueriesPerc)
	fmt.Printf("Training data: %d samples\n", len(model.Evaluations))

	rfModel := NewRFModel(slowQueriesPerc)
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
	fmt.Printf("\nSample predictions on training data (voting threshold: %.2f):\n", votingThreshold)

	fmt.Println("negative examples test: ")
	for i := 0; i < maxSamples; i++ {
		randomIdx := rand.IntN(model.MidpointIdx)
		predicted := rfModel.Predict(model.Evaluations[randomIdx])
		actual := model.Evaluations[randomIdx].ProcTime < model.BinMidpoint
		fmt.Printf(
			"       %d, match: %t, vote NO: %.2f (time: %.2f)\n",
			randomIdx, actual == (predicted < votingThreshold), 1-predicted, model.Evaluations[randomIdx].ProcTime,
		)
	}

	fmt.Println("POSITIVE examples test: ")
	for i := 0; i < maxSamples; i++ {
		randomIdx := rand.IntN(len(model.Evaluations)-model.MidpointIdx) + model.MidpointIdx
		predicted := rfModel.Predict(model.Evaluations[randomIdx])
		actual := model.Evaluations[randomIdx].ProcTime >= model.BinMidpoint
		fmt.Printf(
			"       %d, match: %t, vote YES: %.2f (time: %.2f)\n",
			randomIdx, actual == (predicted >= votingThreshold), predicted, model.Evaluations[randomIdx].ProcTime,
		)
	}

	fmt.Println("calculating precision and recall using data of size ", len(model.Evaluations))

	model.PrecisionAndRecall(rfModel, votingThreshold)

	timestamp := time.Now().Format("20060102T150405")
	modelPath := fmt.Sprintf("cqlizer_rfmodel_%s.json", timestamp)
	fmt.Printf("\n=== Saving RF Model ===\n")
	if err := rfModel.SaveToFile(modelPath); err != nil {
		fmt.Printf("Error saving model: %v\n", err)

	} else {
		fmt.Printf("Model saved to %s\n", modelPath)
	}

	return nil
}
