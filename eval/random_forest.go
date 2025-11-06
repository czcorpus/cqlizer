package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"

	randomforest "github.com/malaschitz/randomForest"
	"github.com/patrikeh/go-deep"
	"github.com/patrikeh/go-deep/training"
	"github.com/rs/zerolog/log"
)

type jsonizedRFModel struct {
	Forest                json.RawMessage `json:"forest"`
	Comment               string          `json:"comment"`
	SlowQueriesPercentile float64         `json:"slowQueriesPercentile"`
}

// RFModel wraps a Random Forest classifier for regression via quantile binning
type RFModel struct {
	Forest                *randomforest.Forest `json:"forest"`
	NeuralNet             *deep.Neural
	Comment               string  `json:"comment"`
	SlowQueriesPercentile float64 `json:"slowQueriesPercentile"`
}

type Prediction struct {
	Votes          []float64
	PredictedClass float64
}

// NewRFModel creates a new Random Forest model with time binning
func NewRFModel(slowQueriesPerc float64) *RFModel {
	return &RFModel{
		Forest:                &randomforest.Forest{},
		SlowQueriesPercentile: slowQueriesPerc,
	}
}

type FeatureStats struct {
	Min float64
	Max float64
}

func (m *RFModel) getDataStats(data training.Examples) []FeatureStats {
	stats := make([]FeatureStats, NumFeatures)
	for _, item := range data {
		for i := 0; i < len(item.Input); i++ {
			if item.Input[i] > stats[i].Max {
				stats[i].Max = item.Input[i]
			}
			if item.Input[i] < stats[i].Min {
				stats[i].Min = item.Input[i]
			}
		}
	}
	return stats
}

func (m *RFModel) Train(dataModel *BasicModel, numTrees int) error {
	if len(dataModel.Evaluations) == 0 {
		return fmt.Errorf("no training data provided")
	}
	var data = training.Examples{}
	//numTotal := len(dataModel.Evaluations)
	numProblematic := 0
	for _, eval := range dataModel.Evaluations {
		features := extractFeatures(eval)
		//isPositive := 0
		if eval.ProcTime >= dataModel.binMidpoint {
			numProblematic++
			//isPositive = 1
		}
		response := 0.0
		if eval.ProcTime >= dataModel.binMidpoint {
			response = 1.0
		}
		data = append(
			data,
			training.Example{
				Input:    features,
				Response: []float64{response},
			},
		)
	}
	log.Debug().
		Int("numPositive", numProblematic).
		Int("dataSize", len(dataModel.Evaluations)).
		Msg("prepared training vectors")

	trn, heldout := data, data //data.Split(0.5)
	stats := m.getDataStats(trn)
	fmt.Printf("STATS: >>> %#v\n", stats)

	for _, item := range trn {
		for i := 0; i < NumFeatures; i++ {
			min := stats[i].Min
			max := stats[i].Max

			if max == min {
				item.Input[i] = 0.0 // constant feature

			} else {
				item.Input[i] = (item.Input[i] - min) / (max - min)
			}
		}
		//fmt.Printf(">> %#v\n", item)
	}

	for _, item := range heldout {
		for i := 0; i < NumFeatures; i++ {
			min := stats[i].Min
			max := stats[i].Max

			if max == min {
				item.Input[i] = 0.0 // constant feature

			} else {
				item.Input[i] = (item.Input[i] - min) / (max - min)
			}
		}
	}

	m.NeuralNet = deep.NewNeural(&deep.Config{
		/* Input dimensionality */
		Inputs: 49,
		/* Two hidden layers consisting of two neurons each, and a single output */
		Layout: []int{49, 25, 1},
		/* Activation functions: Sigmoid, Tanh, ReLU, Linear */
		Activation: deep.ActivationReLU,
		/* Determines output layer activation & loss function:
		ModeRegression: linear outputs with MSE loss
		ModeMultiClass: softmax output with Cross Entropy loss
		ModeMultiLabel: sigmoid output with Cross Entropy loss
		ModeBinary: sigmoid output with binary CE loss */
		Mode: deep.ModeBinary,
		/* Weight initializers: {deep.NewNormal(μ, σ), deep.NewUniform(μ, σ)} */
		Weight: deep.NewUniform(1.0, 0.0),
		/* Apply bias */
		Bias: true,
	})

	optimizer := training.NewSGD(0.025, 0.1, 1e-5, true)
	//optimizer := training.NewAdam(0.05, 0.1, 0, 1e-6)
	// params: optimizer, verbosity (print stats at every 50th iteration)
	trainer := training.NewTrainer(optimizer, 50)

	trainer.Train(m.NeuralNet, trn, heldout, 1000)
	return nil
}

// Train trains the random forest on query evaluations and actual times
func (m *RFModel) TrainRF(dataModel *BasicModel, numTrees int) error {
	if len(dataModel.Evaluations) == 0 {
		return fmt.Errorf("no training data provided")
	}

	var xData [][]float64
	var yData []int
	numTotal := len(dataModel.Evaluations)
	numProblematic := 0
	for i, eval := range dataModel.Evaluations {
		features := extractFeatures(eval)
		//isPositive := 0
		if eval.ProcTime >= dataModel.binMidpoint {
			numProblematic++
			//isPositive = 1
		}
		xData = append(xData, features)
		yData = append(yData, m.bucketize(float64(i)/float64(numTotal)))
	}
	log.Debug().
		Int("numPositive", numProblematic).
		Int("dataSize", len(dataModel.Evaluations)).
		Msg("prepared training vectors")

	m.Forest.Data = randomforest.ForestData{
		X:     xData,
		Class: yData,
	}
	m.Forest.Train(numTrees)
	return nil
}

func (m *RFModel) bucketize(percentile float64) int {
	thresholds := []float64{0.5, 0.75, 0.875, 0.9375, 0.96875, 0.984375, 1.0}
	for i, v := range thresholds {
		if percentile < v {
			return i
		}
	}
	return len(thresholds) - 1
}

// Predict estimates query execution time using the trained forest
func (m *RFModel) PredictRF(eval QueryEvaluation) Prediction {
	features := extractFeatures(eval)
	best := 0.0
	bestIdx := 0
	votes := m.Forest.Vote(features)
	for i, v := range votes {
		if v > best {
			best = v
			bestIdx = i
		}
	}
	return Prediction{
		Votes:          votes,
		PredictedClass: float64(bestIdx),
	}
}

func (m *RFModel) Predict(eval QueryEvaluation) Prediction {
	features := extractFeatures(eval)
	out := m.NeuralNet.Predict(features)
	fmt.Println("prediction of ", eval.OrigQuery, " = ", out)
	return Prediction{
		Votes:          []float64{},
		PredictedClass: out[0],
	}
}

// extractFeatures converts QueryEvaluation to feature vector (same as Huber)
func extractFeatures(eval QueryEvaluation) []float64 {
	features := make([]float64, NumFeatures)
	idx := 0

	// Extract features for up to 4 positions
	for i := 0; i < MaxPositions; i++ {
		if i < len(eval.Positions) {
			pos := eval.Positions[i]
			// Position-specific features (normalized by concrete chars)
			features[idx] = float64(pos.Regexp.StartsWithWildCard)
			features[idx+1] = pos.Regexp.WildcardScore
			features[idx+2] = float64(pos.Regexp.HasRange)
			features[idx+3] = float64(pos.HasSmallCardAttr)
			features[idx+4] = float64(pos.Regexp.NumConcreteChars)
			features[idx+5] = pos.Regexp.AvgCharProb
			features[idx+6] = float64(pos.NumAlternatives)
			features[idx+7] = pos.PosRepetition
			features[idx+8] = pos.Regexp.CharClasses
			features[idx+9] = float64(pos.HasNegation)
		}
		// If position doesn't exist, features remain 0
		idx += 10
	}

	// Global features
	features[40] = float64(eval.NumGlobConditions)
	features[41] = float64(eval.ContainsMeet)
	features[42] = float64(eval.ContainsUnion)
	features[43] = float64(eval.ContainsWithin)
	features[44] = eval.AdhocSubcorpus
	features[45] = float64(eval.ContainsContaining)
	features[46] = math.Log(eval.CorpusSize)
	features[47] = float64(eval.AlignedPart)
	features[48] = 1.0 // Bias term

	return features
}

// SaveToFile saves the RF model to a file
func (m *RFModel) SaveToFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to save RF model to a file: %w", err)
	}
	defer file.Close()

	tmpModel := jsonizedRFModel{
		Comment:               m.Comment,
		SlowQueriesPercentile: m.SlowQueriesPercentile,
	}

	bytes, err := json.Marshal(&m.Forest)
	if err != nil {
		return fmt.Errorf("failed to save RF model to a file: %w", err)
	}

	tmpModel.Forest = bytes

	bytes, err = json.Marshal(tmpModel)
	if err != nil {
		return fmt.Errorf("failed to save RF model to a file: %w", err)
	}
	_, err = file.Write(bytes)
	if err != nil {
		return fmt.Errorf("failed to save RF model to a file: %w", err)
	}
	return nil
}

// LoadFromFile loads model metadata from file
// Note: This is a placeholder - the actual forest cannot be serialized/deserialized
// with the current randomForest package
func LoadRFModelFromFile(filePath string) (*RFModel, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var tmpModel jsonizedRFModel
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load Random Forest model from file: %w", err)
	}
	if err := json.Unmarshal(data, &tmpModel); err != nil {
		return nil, fmt.Errorf("failed to load Random Forest model from file: %w", err)
	}

	model := &RFModel{
		Comment:               tmpModel.Comment,
		SlowQueriesPercentile: tmpModel.SlowQueriesPercentile,
	}

	var forest randomforest.Forest
	if err := json.Unmarshal(tmpModel.Forest, &forest); err != nil {
		return nil, fmt.Errorf("failed to load Random Forest model from file: %w", err)
	}
	model.Forest = &forest
	return model, nil
}
