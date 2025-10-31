package eval

import (
	"context"
	"errors"
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

	// slowQueryPercentile specifies which percentile of queries (by time)
	// is considered as "slow times".
	// This value is the one user enters.
	slowQueryPercentile float64

	// midpointIdx is derived from SlowQueryPercentile and represents a sorted data index
	// from which SlowQueryPercentile starts.
	midpointIdx int

	// binMidpoint is the threshold time where SlowQueryPercentile starts. The value
	// is derived from SlowQueryPercentile
	binMidpoint float64
}

func (model *BasicModel) BalanceSample() []QueryEvaluation {
	slices.SortFunc(model.Evaluations, func(v1, v2 QueryEvaluation) int {
		if v1.ProcTime < v2.ProcTime {
			return -1
		}
		return 1
	})
	//model.MidpointIdx, model.BinMidpoint = model.computeThreshold()
	log.Info().Msg("creating a balanced sample for learning")
	model.binMidpoint, model.midpointIdx = findKneeDistance(model.Evaluations)
	model.slowQueryPercentile = float64(model.midpointIdx) / float64(len(model.Evaluations))
	log.Info().
		Float64("thresholdTime", model.binMidpoint).
		Int("thresholdIdx", model.midpointIdx).
		Float64("slowQueryPercentile", model.slowQueryPercentile).
		Int("totalQueries", len(model.Evaluations)).
		Int("positiveExamples", len(model.Evaluations)-model.midpointIdx).
		Msg("calculated threshold for slow queries")

	numPositive := len(model.Evaluations) - model.midpointIdx
	balEval := make([]QueryEvaluation, numPositive*3)
	for i := 0; i < numPositive*2; i++ {
		balEval[i] = model.Evaluations[rand.IntN(model.midpointIdx)]
	}
	for i := 0; i < numPositive; i++ {
		balEval[i+numPositive*2] = model.Evaluations[model.midpointIdx+i]
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
		log.Warn().
			Err(errors.New(errMsg)).
			Str("query", entry.GetCQL()).
			Msg("Warning: Failed to parse query")
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
		trulySlow := model.Evaluations[i].ProcTime >= model.binMidpoint
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

	fmt.Printf("%.2f;%.2f;%.2f;%.2f\n", threshold, precision, recall, fbeta)

}

func (model *BasicModel) showSampleEvaluations(rfModel *RFModel, maxSamples int, votingThreshold float64) {

	if len(model.Evaluations) < maxSamples {
		maxSamples = len(model.Evaluations)
	}

	// Test predictions on training data (for diagnostic purposes)
	fmt.Printf("\nSample predictions on training data (voting threshold: %.2f):\n", votingThreshold)

	fmt.Println("negative examples test: ")
	for i := 0; i < maxSamples; i++ {
		randomIdx := rand.IntN(model.midpointIdx)
		predicted := rfModel.Predict(model.Evaluations[randomIdx])
		actual := model.Evaluations[randomIdx].ProcTime < model.binMidpoint
		fmt.Printf(
			"       %d, match: %t, vote NO: %.2f (time: %.2f)\n",
			randomIdx, actual == (predicted < votingThreshold), 1-predicted, model.Evaluations[randomIdx].ProcTime,
		)
	}

	fmt.Println("POSITIVE examples test: ")
	for i := 0; i < maxSamples; i++ {
		randomIdx := rand.IntN(len(model.Evaluations)-model.midpointIdx) + model.midpointIdx
		predicted := rfModel.Predict(model.Evaluations[randomIdx])
		actual := model.Evaluations[randomIdx].ProcTime >= model.binMidpoint
		fmt.Printf(
			"       %d, match: %t, vote YES: %.2f (time: %.2f)\n",
			randomIdx, actual == (predicted >= votingThreshold), predicted, model.Evaluations[randomIdx].ProcTime,
		)
	}
}

// EvaluateWithRF trains a Random Forest model instead of Huber regression
func (model *BasicModel) EvaluateWithRF(
	ctx context.Context,
	numTrees int,
	votingThreshold float64,
	testData []QueryEvaluation,
	outputPath string,
) error {
	if len(model.Evaluations) == 0 {
		return fmt.Errorf("no training data available")
	}

	log.Info().
		Int("numTrees", numTrees).
		Int("trainingDataSize", len(model.Evaluations)).
		Msg("Training Random Forest")

	rfModel := NewRFModel(model.slowQueryPercentile)
	if err := rfModel.Train(model, numTrees); err != nil {
		return fmt.Errorf("RF training failed: %w", err)
	}

	timestamp := time.Now().Format("20060102T150405")
	modelPath := fmt.Sprintf("cqlizer_rfmodel_%s.json", timestamp)
	if err := rfModel.SaveToFile(modelPath); err != nil {
		return fmt.Errorf("error saving model: %w", err)

	} else {
		log.Debug().Str("path", modelPath).Msg("saved model file")
	}

	// ----- testing

	model.Evaluations = testData

	log.Info().
		Int("evalDataSize", len(model.Evaluations)).
		Msg("calculating precision and recall using full data")

	fmt.Println("vote;precision;recall;f-beta")
	for th := 0.7; th <= 0.991; th += 0.01 {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		model.PrecisionAndRecall(rfModel, th)
	}

	return nil
}
