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

package rf

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/czcorpus/cqlizer/eval"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/predict"
	randomforest "github.com/malaschitz/randomForest"
	"github.com/rs/zerolog/log"
)

type jsonizedRFModel struct {
	Forest                   json.RawMessage `json:"forest"`
	Comment                  string          `json:"comment"`
	slowQueriesThresholdTime float64         `json:"slowQueriesThresholdTime"`
}

// Model wraps a Random Forest classifier for regression via quantile binning
type Model struct {
	Forest                   *randomforest.Forest `json:"forest"`
	NumTrees                 int                  `json:"numTrees"`
	VotingThreshold          float64              `json:"votingThreshold"`
	SlowQueriesThresholdTime float64              `json:"slowQueriesThresholdTime"`
	Comment                  string               `json:"comment"`
}

// NewModel creates a new Random Forest model with time binning
func NewModel(numTrees int, votingThreshold float64) *Model {
	return &Model{
		Forest:          &randomforest.Forest{},
		NumTrees:        numTrees,
		VotingThreshold: votingThreshold,
	}
}

func (m *Model) IsInferenceOnly() bool {
	return false
}

func (m *Model) CreateModelFileName(featsFile string) string {
	return eval.ExtractModelNameBaseFromFeatFile(featsFile) + ".model.rf.json"
}

func (m *Model) GetClassThreshold() float64 {
	return m.VotingThreshold
}

func (m *Model) SetClassThreshold(v float64) {
	m.VotingThreshold = v
}

func (m *Model) GetSlowQueriesThresholdTime() float64 {
	return m.SlowQueriesThresholdTime
}

func (m *Model) GetInfo() string {
	return fmt.Sprintf("RF model, num. trees: %d, slow q. threshold time: %.2fs", m.NumTrees, m.SlowQueriesThresholdTime)
}

// Train trains the random forest on query evaluations and actual times
// note: the `comment` argument will be stored with the model for easier model review
func (m *Model) Train(ctx context.Context, data []feats.QueryEvaluation, slowQueriesThresholdTime float64, comment string) error {
	if len(data) == 0 {
		return fmt.Errorf("no training data provided")
	}
	if slowQueriesThresholdTime <= 0 {
		return fmt.Errorf("failed to train RF model - invalid value of SlowQueriesThresholdTime")
	}
	m.SlowQueriesThresholdTime = slowQueriesThresholdTime
	if m.NumTrees <= 0 {
		return fmt.Errorf("failed to train RF model - invalid value of NumTrees")
	}
	var xData [][]float64
	var yData []int
	numProblematic := 0
	for i, eval := range data {
		if i%100 == 0 && ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		features := feats.ExtractFeatures(eval)
		isPositive := 0
		if eval.ProcTime >= m.SlowQueriesThresholdTime {
			numProblematic++
			isPositive = 1
		}
		xData = append(xData, features)
		yData = append(yData, isPositive)
	}
	log.Debug().
		Int("numPositive", numProblematic).
		Int("dataSize", len(data)).
		Msg("prepared training vectors")

	m.Forest.Data = randomforest.ForestData{
		X:     xData,
		Class: yData,
	}
	m.Forest.Train(m.NumTrees)
	m.Comment = comment
	return nil
}

// Predict estimates query execution time using the trained forest
func (m *Model) Predict(eval feats.QueryEvaluation) predict.Prediction {
	features := feats.ExtractFeatures(eval)
	votes := m.Forest.Vote(features)
	var ans int
	if votes[1] > m.VotingThreshold {
		ans = 1
	}
	return predict.Prediction{
		Votes:          votes,
		PredictedClass: ans,
	}
}

// SaveToFile saves the RF model to a file
func (m *Model) SaveToFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to save RF model to a file: %w", err)
	}
	defer file.Close()

	tmpModel := jsonizedRFModel{
		Comment:                  m.Comment,
		slowQueriesThresholdTime: m.SlowQueriesThresholdTime,
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

	var tmpModel jsonizedRFModel
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to load Random Forest model from file: %w", err)
	}
	if err := json.Unmarshal(data, &tmpModel); err != nil {
		return nil, fmt.Errorf("failed to load Random Forest model from file: %w", err)
	}

	model := &Model{
		Comment:                  tmpModel.Comment,
		SlowQueriesThresholdTime: tmpModel.slowQueriesThresholdTime,
	}

	var forest randomforest.Forest
	if err := json.Unmarshal(tmpModel.Forest, &forest); err != nil {
		return nil, fmt.Errorf("failed to load Random Forest model from file: %w", err)
	}
	model.Forest = &forest
	model.NumTrees = forest.NTrees
	return model, nil
}
