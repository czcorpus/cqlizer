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

package xg

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/czcorpus/cnc-gokit/fs"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/modutils"
	"github.com/czcorpus/cqlizer/eval/predict"
	"github.com/dmitryikh/leaves"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
)

type metadata struct {
	Objective       string    `json:"objective"`
	Metric          [2]string `json:"metric"`
	ScalePosWeight  float64   `json:"scale_pos_weight"`
	MaxDepth        int       `json:"max_depth"`
	LearningRate    float64   `json:"learning_rate"`
	NumLeaves       int       `json:"num_leaves"`
	MinChildSamples int       `json:"min_child_samples"`
	Subsample       float64   `json:"subsample"`
	ColsampleBytree float64   `json:"colsample_bytree"`
	RandomState     int       `json:"random_state"`
	Verbose         int       `json:"verbose"`
}

type Model struct {
	ClassThreshold           float64
	SlowQueriesThresholdTime float64
	trainXData               [][]float64
	trainYData               []int
	xgboost                  *leaves.Ensemble
	metadata                 metadata
}

func (m *Model) IsInferenceOnly() bool {
	return true
}

func (m *Model) CreateModelFileName(featsFile string) string {
	return modutils.ExtractModelNameBaseFromFeatFile(featsFile) + ".feats.xg.msgpack"
}

func (m *Model) Train(ctx context.Context, data []feats.QueryEvaluation, slowQueriesTime float64, comment string) error {
	if len(data) == 0 {
		return fmt.Errorf("no training data provided")
	}
	if slowQueriesTime <= 0 {
		return fmt.Errorf("failed to train RF model - invalid value of SlowQueriesThresholdTime")
	}
	m.SlowQueriesThresholdTime = slowQueriesTime

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
	m.trainXData = xData
	m.trainYData = yData
	return nil
}

func (m *Model) Predict(eval feats.QueryEvaluation) predict.Prediction {
	features := feats.ExtractFeatures(eval)
	pred := m.xgboost.PredictSingle(features, 0)
	var ans int
	if pred > m.ClassThreshold {
		ans = 1
	}
	return predict.Prediction{
		Votes:          []float64{1 - pred, pred},
		PredictedClass: ans,
	}
}

func (m *Model) SetClassThreshold(v float64) {
	m.ClassThreshold = v
}

func (m *Model) GetClassThreshold() float64 {
	return m.ClassThreshold
}

func (m *Model) GetSlowQueriesThresholdTime() float64 {
	return m.SlowQueriesThresholdTime
}

func (m *Model) SaveToFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to save RF model to a file: %w", err)
	}
	defer file.Close()
	out := make(map[string]any)
	out["features"] = m.trainXData
	out["label"] = m.trainYData

	outData, err := msgpack.Marshal(out)
	if err != nil {
		return fmt.Errorf("failed to create XGBoost training data: %w", err)
	}
	_, err = file.Write(outData)
	if err != nil {
		return fmt.Errorf("failed to create XGBoost training data: %w", err)
	}
	return nil
}

func (m *Model) GetInfo() string {
	return fmt.Sprintf(
		"XGBoost model, metric: %s / %s, NL: %d, SPV: %.2f, LR: %.2f",
		m.metadata.Metric[0],
		m.metadata.Metric[1],
		m.metadata.NumLeaves,
		m.metadata.ScalePosWeight,
		m.metadata.LearningRate,
	)
}

func loadMetadata(modelPath string) (metadata, error) {
	var mt metadata
	var metadataFilePath string
	ext := filepath.Ext(modelPath)
	if ext == ".gz" || ext == ".gzip" {
		modelPath = modelPath[:len(modelPath)-len(ext)]
		ext = filepath.Ext(modelPath)
	}
	metadataFilePath = modelPath[:len(modelPath)-len(ext)] + ".metadata.json"
	isFile, err := fs.IsFile(metadataFilePath)
	if err != nil {
		return mt, fmt.Errorf("failed to load XG model metadata: %w", err)
	}
	if !isFile {
		log.Warn().Msg("Cannot load XG model metadata - no file found. For inference, this doesn't matter.")
		return mt, nil
	}
	data, err := os.ReadFile(metadataFilePath)
	if err != nil {
		return mt, fmt.Errorf("failed to load XG model metadata: %w", err)
	}
	if err := json.Unmarshal(data, &mt); err != nil {
		return mt, fmt.Errorf("failed to load XG model metadata: %w", err)
	}
	return mt, nil
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

	model, err := leaves.LGEnsembleFromReader(bufio.NewReader(reader), true)
	if err != nil {
		return nil, fmt.Errorf("failed to load XG model: %w", err)
	}
	metadata, err := loadMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load XG model: %w", err)
	}
	return &Model{xgboost: model, metadata: metadata}, nil
}

func NewModel() *Model {
	return &Model{
		ClassThreshold: 0.5,
	}
}
