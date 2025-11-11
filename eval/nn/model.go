package nn

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/predict"
	"github.com/patrikeh/go-deep"
	"github.com/patrikeh/go-deep/training"
	"github.com/rs/zerolog/log"
)

type FeatureStats struct {
	Min float64
	Max float64
}

type jsonizedModel struct {
	NeuralNet                *deep.Dump     `json:"neuralNet"`
	DataRanges               []FeatureStats `json:"dataRanges"`
	SlowQueriesThresholdTime float64        `json:"slowQueriesThresholdTime"`
	ClassThreshold           float64        `json:"classThreshold"`
}

type Model struct {
	NeuralNet                *deep.Neural
	DataRanges               []FeatureStats
	SlowQueriesThresholdTime float64
	ClassThreshold           float64
}

func (m *Model) SetClassThreshold(v float64) {
	m.ClassThreshold = v
}

// Train
// TODO: comment is not stored
func (m *Model) Train(data []feats.QueryEvaluation, slowQueriesTime float64, comment string) error {
	if len(data) == 0 {
		return fmt.Errorf("no training data provided")
	}
	if slowQueriesTime <= 0 {
		return fmt.Errorf("failed to train RF model - invalid value of SlowQueriesThresholdTime")
	}
	m.SlowQueriesThresholdTime = slowQueriesTime
	var featData = training.Examples{}
	//numTotal := len(dataModel.Evaluations)
	numProblematic := 0
	for _, eval := range data {
		features := feats.ExtractFeatures(eval)
		response := 0.0
		if eval.ProcTime >= m.SlowQueriesThresholdTime {
			numProblematic++
			response = 1.0
		}
		featData = append(
			featData,
			training.Example{
				Input:    features,
				Response: []float64{response},
			},
		)
	}
	log.Debug().
		Int("numPositive", numProblematic).
		Int("dataSize", len(data)).
		Msg("prepared training vectors")

	// TODO !!!!!! we use the same training and heldout data !!!
	trn, heldout := featData, featData //featData.Split(0.5)
	m.DataRanges = m.getDataStats(trn)
	fmt.Printf("STATS: >>> %#v\n", m.DataRanges)

	for _, item := range trn {
		m.normalizeNNFeats(item.Input)
	}

	/*
		for _, item := range heldout {
			m.normalizeNNFeats(item)
		}
	*/

	m.NeuralNet = deep.NewNeural(&deep.Config{
		Inputs:     49,
		Layout:     []int{15, 4, 1},
		Activation: deep.ActivationReLU,
		Mode:       deep.ModeBinary,
		Weight:     deep.NewUniform(1.0, 0.0),
		Bias:       true,
	})

	//optimizer := training.NewSGD(0.05, 0.4, 1e-5, true)
	optimizer := training.NewAdam(0.01, 0.9, 0.999, 1e-8)
	// params: optimizer, verbosity (print stats at every 50th iteration)
	trainer := training.NewTrainer(optimizer, 50)
	trainer.Train(m.NeuralNet, trn, heldout, 1000)
	return nil
}

func (m *Model) getDataStats(data training.Examples) []FeatureStats {
	stats := make([]FeatureStats, feats.NumFeatures)
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

func (m *Model) normalizeNNFeats(data []float64) {
	for i := 0; i < feats.NumFeatures; i++ {
		min := m.DataRanges[i].Min
		max := m.DataRanges[i].Max

		if max == min {
			data[i] = 0.0 // constant feature

		} else {
			data[i] = (data[i] - min) / (max - min)
		}
	}
}

func (m *Model) Predict(eval feats.QueryEvaluation) predict.Prediction {
	features := feats.ExtractFeatures(eval)
	m.normalizeNNFeats(features)
	out := m.NeuralNet.Predict(features)
	//fmt.Println("prediction of ", eval.OrigQuery, " = ", out)
	var predClass int
	if out[0] >= m.ClassThreshold {
		predClass = 1
	}
	return predict.Prediction{
		Votes:          []float64{1 - out[0], out[0]},
		PredictedClass: predClass,
	}
}

func (m *Model) SaveToFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to save RF model to a file: %w", err)
	}
	defer file.Close()
	dmp := m.NeuralNet.Dump()
	tmpModel := jsonizedModel{
		NeuralNet:                dmp,
		DataRanges:               m.DataRanges,
		SlowQueriesThresholdTime: m.SlowQueriesThresholdTime,
		ClassThreshold:           m.ClassThreshold,
	}
	bytes, err := json.Marshal(tmpModel)
	if err != nil {
		return fmt.Errorf("failed to save NN to file: %w", err)
	}
	_, err = file.Write(bytes)
	if err != nil {
		return fmt.Errorf("failed to save NN model to a file: %w", err)
	}
	return nil
}

func LoadFromFile(filePath string) (*Model, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var model jsonizedModel
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load Neural Network model from file %s: %w", filePath, err)
	}
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("failed to load Neural Network model from file %s: %w", filePath, err)
	}
	nn := deep.FromDump(model.NeuralNet)
	return &Model{
		NeuralNet:                nn,
		DataRanges:               model.DataRanges,
		SlowQueriesThresholdTime: model.SlowQueriesThresholdTime,
		ClassThreshold:           model.ClassThreshold,
	}, nil
}

func NewModel() *Model {
	return &Model{
		ClassThreshold: 0.5,
	}
}
