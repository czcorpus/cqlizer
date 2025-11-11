package eval

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/predict"
	"github.com/rs/zerolog/log"
)

type PrecAndRecall struct {
	Precision float64
	Recall    float64
	FBeta     float64
}

func (pr PrecAndRecall) CSV(x float64) string {
	return fmt.Sprintf("%.2f;%.2f;%.2f;%.2f", x, pr.Precision, pr.Recall, pr.FBeta)
}

func findKneeDistance(items []feats.QueryEvaluation) (threshold float64, kneeIdx int) {

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

// ------------------------------------

type QueryStatsRecord struct {
	Corpus        string  `json:"corpus"`
	CorpusSize    int64   `json:"corpusSize"`
	SubcorpusSize int64   `json:"subcorpusSize"`
	TimeProc      float64 `json:"timeProc"`
	Query         string  `json:"query"`
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

type LearningDataStats struct {
	NumProcessed       int     `msgpack:"numProcessed"`
	NumFailed          int     `msgpack:"numFailed"`
	DeduplicationRatio float64 `msgpack:"deduplicationRatio"`
}

func (stats LearningDataStats) AsComment() string {
	return fmt.Sprintf("source data - total items: %d, failed imports: %d, deduplicated ratio: %.2f", stats.NumProcessed, stats.NumFailed, stats.DeduplicationRatio)
}

// ----------------------------

// MLModel is a generalization of a Machine Learning model used to extract knowledge
// about CQL queries.
type MLModel interface {
	Train(data []feats.QueryEvaluation, slowQueriesTime float64, comment string) error
	Predict(feats.QueryEvaluation) predict.Prediction
	SetClassThreshold(v float64)
	SaveToFile(string) error
}

// ----------------------------

type Predictor struct {
	mlModel MLModel

	Evaluations []feats.QueryEvaluation

	LearningDataStats LearningDataStats

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

	corpora map[string]feats.CorpusProps
}

func NewPredictor(mlModel MLModel, corporaProps map[string]feats.CorpusProps) *Predictor {
	return &Predictor{
		corpora: corporaProps,
		mlModel: mlModel,
	}
}

func (model *Predictor) BalanceSample() []feats.QueryEvaluation {
	slices.SortFunc(model.Evaluations, func(v1, v2 feats.QueryEvaluation) int {
		if v1.ProcTime < v2.ProcTime {
			return -1

		} else if v1.ProcTime > v2.ProcTime {
			return 1
		}
		return 0
	})
	for i := 0; i < len(model.Evaluations); i++ {
		if model.Evaluations[i].ProcTime > 450 {
			model.Evaluations[i].ProcTime = 450
			fmt.Println("HUGE WQUERY ------------ ", model.Evaluations[i].Positions)
		}
	}
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
		Float64("maxProcTime", model.Evaluations[len(model.Evaluations)-1].ProcTime).
		Float64("minProcTime", model.Evaluations[0].ProcTime).
		Msg("calculated threshold for slow queries")

	numPositive := len(model.Evaluations) - model.midpointIdx
	balEval := make([]feats.QueryEvaluation, numPositive*3)
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

func (model *Predictor) ProcessEntry(entry QueryStatsRecord) error {
	if entry.CorpusSize == 0 {
		return fmt.Errorf("zero corpus size")
	}
	if entry.TimeProc <= 0 {
		return fmt.Errorf("invalid processing time %.2f", entry.TimeProc)
	}

	// Parse the CQL query and create evaluation with corpus size
	corpInfo := model.corpora[entry.Corpus]
	eval, err := feats.NewQueryEvaluation(
		entry.GetCQL(),
		float64(entry.CorpusSize),
		float64(entry.SubcorpusSize),
		entry.TimeProc,
		feats.GetCharProbabilityProvider(corpInfo.Lang),
	)
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

func (model *Predictor) SetStats(numProcessed, numFailed int) {
	model.LearningDataStats.NumProcessed = numProcessed
	model.LearningDataStats.NumFailed = numFailed
}

func (model *Predictor) PrecisionAndRecall() PrecAndRecall {

	numTruePositives := 0
	numRelevant := 0
	numRetrieved := 0

	for i := 0; i < len(model.Evaluations); i++ {
		trulySlow := model.Evaluations[i].ProcTime >= model.binMidpoint
		prediction := model.mlModel.Predict(model.Evaluations[i]).PredictedClass
		if trulySlow {
			numRelevant++
		}
		if prediction == 1 {
			numRetrieved++
			if trulySlow {
				numTruePositives++

			} else {
				/*
					fmt.Printf(
						"WE SAY %s IS SLOW (%0.2f) BUT IT IS NOT (time %.2f, corpsize: %0.2f)\n",
						model.Evaluations[i].OrigQuery, prediction, model.Evaluations[i].ProcTime, math.Exp(model.Evaluations[i].CorpusSize),
					)
				*/
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
	return PrecAndRecall{Precision: precision, Recall: recall, FBeta: fbeta}

}

func (model *Predictor) showSampleEvaluations(rfModel MLModel, maxSamples int, votingThreshold float64) {

	if len(model.Evaluations) < maxSamples {
		maxSamples = len(model.Evaluations)
	}

	// Test predictions on training data (for diagnostic purposes)
	fmt.Printf("\nSample predictions on training data (voting threshold: %.2f):\n", votingThreshold)

	fmt.Println("negative examples test: ")
	for i := 0; i < maxSamples; i++ {
		randomIdx := rand.IntN(model.midpointIdx)
		predicted := float64(rfModel.Predict(model.Evaluations[randomIdx]).PredictedClass) / 100.0
		actual := model.Evaluations[randomIdx].ProcTime < model.binMidpoint
		fmt.Printf(
			"       %d, match: %t, vote NO: %.2f (time: %.2f)\n",
			randomIdx, actual == (predicted < votingThreshold), 1-predicted, model.Evaluations[randomIdx].ProcTime,
		)
	}

	fmt.Println("POSITIVE examples test: ")
	for i := 0; i < maxSamples; i++ {
		randomIdx := rand.IntN(len(model.Evaluations)-model.midpointIdx) + model.midpointIdx
		predicted := float64(rfModel.Predict(model.Evaluations[randomIdx]).PredictedClass) / 100.0
		actual := model.Evaluations[randomIdx].ProcTime >= model.binMidpoint
		fmt.Printf(
			"       %d, match: %t, vote YES: %.2f (time: %.2f)\n",
			randomIdx, actual == (predicted >= votingThreshold), predicted, model.Evaluations[randomIdx].ProcTime,
		)
	}
}

func (model *Predictor) Deduplicate() {
	uniq := make(map[string][]feats.QueryEvaluation)
	for _, v := range model.Evaluations {
		_, ok := uniq[v.UniqKey()]
		if !ok {
			uniq[v.UniqKey()] = make([]feats.QueryEvaluation, 0, 10)
		}
		uniq[v.UniqKey()] = append(uniq[v.UniqKey()], v)
	}
	for _, evals := range uniq {
		slices.SortFunc(evals, func(v1, v2 feats.QueryEvaluation) int {
			if v1.ProcTime < v2.ProcTime {
				return -1
			}
			return 1
		})
		sum := 0.0
		sum2 := 0.0
		n := 0.0
		for _, v := range evals {
			sum += v.ProcTime
			sum2 += v.ProcTime * v.ProcTime
			n += 1
		}
		mean := sum / n
		//variance := (sum2 / n) - (mean * mean)
		//stdDev := math.Sqrt(variance)
		var median float64
		if len(evals) <= 2 {
			median = mean

		} else {
			middle := int(math.Ceil(float64(len(evals)) / 2.0))
			median = evals[middle].ProcTime
		}
		evals[0].ProcTime = median
	}
	model.Evaluations = make([]feats.QueryEvaluation, len(uniq))
	i := 0
	for _, u := range uniq {
		model.Evaluations[i] = u[0]
		i++
	}
	model.LearningDataStats.DeduplicationRatio = float64(len(uniq)) / float64(model.LearningDataStats.NumProcessed)
	log.Info().Int("newSize", len(model.Evaluations)).Msg("deduplicated queries")
}

// CreateAndTestModel trains a ML model
func (model *Predictor) CreateAndTestModel(
	ctx context.Context,
	testData []feats.QueryEvaluation,
	outputPath string,
) error {
	if len(model.Evaluations) == 0 {
		return fmt.Errorf("no training data available")
	}

	log.Info().
		Int("trainingDataSize", len(model.Evaluations)).
		Msg("Training Random Forest")

	if err := model.mlModel.Train(model.Evaluations, model.binMidpoint, model.LearningDataStats.AsComment()); err != nil {
		return fmt.Errorf("RF training failed: %w", err)
	}

	if err := model.mlModel.SaveToFile(outputPath); err != nil {
		return fmt.Errorf("error saving model: %w", err)

	} else {
		log.Debug().Str("path", outputPath).Msg("saved model file")
	}

	// ----- testing
	slices.SortFunc(
		testData,
		func(v1, v2 feats.QueryEvaluation) int {
			if v1.ProcTime < v2.ProcTime {
				return -1
			}
			if v1.ProcTime > v2.ProcTime {
				return 1
			}
			return 0
		})
	model.Evaluations = testData

	log.Info().
		Int("evalDataSize", len(model.Evaluations)).
		Msg("calculating precision and recall using full data")

	//for th := 0.7; th <= 0.991; th += 0.01 {
	var wg sync.WaitGroup
	chunks := []float64{0.6, 0.7, 0.8, 0.9}
	wg.Add(len(chunks))
	for v := 0.5; v < 1; v += 0.01 {
		model.mlModel.SetClassThreshold(v)
		precall := model.PrecisionAndRecall()
		fmt.Println(precall.CSV(v))
	}

	return nil
}
