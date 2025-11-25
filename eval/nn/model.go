// Copyright 2025 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2025 Department of Linguistics,
// Faculty of Arts, Charles University
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nn

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/modutils"
	"github.com/czcorpus/cqlizer/eval/predict"
	"github.com/patrikeh/go-deep"
	"github.com/patrikeh/go-deep/training"
	"github.com/rs/zerolog/log"
)

type FeatureStats struct {
	Min float64
	Max float64
}

var (
	//networkLayout = []int{20, 14, 7, 1}
	networkLayout = []int{50, 15, 1}
	//networkLayout = []int{30, 10, 1}
	numEpochs = 800
	//learningRate  = 0.001
	learningRate = 0.0005
)

type jsonizedModel struct {
	NeuralNet                *deep.Dump     `json:"neuralNet"`
	DataRanges               []FeatureStats `json:"dataRanges"`
	SlowQueriesThresholdTime float64        `json:"slowQueriesThresholdTime"`
	ClassThreshold           float64        `json:"classThreshold"`
}

// Model is a neural-network based model for evaluating CQL queries.
// It is rather experimental and does not perform as well as other
// models here so it is not recommended for production use.
type Model struct {
	NeuralNet                *deep.Neural
	DataRanges               []FeatureStats
	SlowQueriesThresholdTime float64
	ClassThreshold           float64
}

func (m *Model) IsInferenceOnly() bool {
	return false
}

func (m *Model) CreateModelFileName(featsFile string) string {
	return modutils.ExtractModelNameBaseFromFeatFile(featsFile) + ".model.nn.json"
}

func (m *Model) GetClassThreshold() float64 {
	return m.ClassThreshold
}

func (m *Model) SetClassThreshold(v float64) {
	m.ClassThreshold = v
}

func (m *Model) GetSlowQueriesThresholdTime() float64 {
	return m.SlowQueriesThresholdTime
}

func (m *Model) GetInfo() string {
	return fmt.Sprintf("NN model, layout: #%v, epochs: %d, slow q. threshold time: %.2fs", networkLayout, numEpochs, m.SlowQueriesThresholdTime)
}

// Train
// TODO: comment is not stored
func (m *Model) Train(ctx context.Context, data []feats.QueryEvaluation, slowQueriesTime float64, comment string) error {
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
	data2 := make([]feats.QueryEvaluation, len(data), len(data)*4)
	copy(data2, data)
	data2 = append(data2, data...)
	data2 = append(data2, data...)
	data2 = append(data2, data...)
	data = data2
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

	m.DataRanges = m.getDataStats(featData)
	for _, item := range featData {
		m.normalizeNNFeats(item.Input)
	}

	// TODO !!!!!! we use the same training and heldout data !!!
	// trn, heldout := featData, featData
	trn, heldout := featData.Split(0.5)

	fmt.Printf("STATS: >>> %#v\n", m.DataRanges)

	//for _, item := range heldout {
	//m.normalizeNNFeats(item)
	//}

	m.NeuralNet = deep.NewNeural(&deep.Config{
		Inputs:     50,
		Layout:     networkLayout,
		Activation: deep.ActivationReLU,
		Mode:       deep.ModeBinary,
		Weight:     deep.NewUniform(1.0, 0.0),
		Bias:       true,
	})

	//optimizer := training.NewSGD(0.05, 0.4, 1e-5, true)
	optimizer := training.NewAdam(learningRate, 0.9, 0.999, 1e-8)
	// params: optimizer, verbosity (print stats at every 50th iteration)
	trainer := training.NewTrainer(optimizer, 50)
	trainer.TrainContext(ctx, m.NeuralNet, trn, heldout, numEpochs)
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

	var reader io.Reader = file
	if strings.HasSuffix(filePath, ".gz") || strings.HasSuffix(filePath, ".gzip") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	var model jsonizedModel
	data, err := io.ReadAll(reader)
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
