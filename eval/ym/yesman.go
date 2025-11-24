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

package ym

import (
	"context"
	"fmt"

	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/predict"
)

// Model is a constant classifier model which evaluates any query as slow (ym = yes-man). It is for debugging
// purposes (for debugging and developing cqlizer's clients).
type Model struct {
	SlowQueriesThresholdTime float64
	ClassThreshold           float64
}

func (ym *Model) IsInferenceOnly() bool {
	return true
}

func (ym *Model) CreateModelFileName(featsFile string) string {
	return "ym-model"
}

func (ym *Model) Train(ctx context.Context, data []feats.QueryEvaluation, slowQueriesTime float64, comment string) error {
	return nil
}

func (ym *Model) Predict(feats feats.QueryEvaluation) predict.Prediction {
	return predict.Prediction{
		Votes:          []float64{0, 1},
		PredictedClass: 1,
	}
}

func (ym *Model) SetClassThreshold(v float64) {
	ym.ClassThreshold = v
}

func (ym *Model) GetClassThreshold() float64 {
	return ym.ClassThreshold
}

func (ym *Model) GetSlowQueriesThresholdTime() float64 {
	return ym.SlowQueriesThresholdTime
}

func (ym *Model) SaveToFile(string) error {
	return fmt.Errorf("cannot save ym model")
}

func (ym *Model) GetInfo() string {
	return "Constant classifier model (always 1)"
}
