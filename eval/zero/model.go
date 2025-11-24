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

package zero

import (
	"context"
	"fmt"

	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/predict"
)

type ZeroModel struct {
	SlowQueriesThresholdTime float64
	ClassThreshold           float64
}

func (zm *ZeroModel) IsInferenceOnly() bool {
	return true
}

func (zm *ZeroModel) CreateModelFileName(featsFile string) string {
	return "zero-model"
}

func (zm *ZeroModel) Train(ctx context.Context, data []feats.QueryEvaluation, slowQueriesTime float64, comment string) error {
	return fmt.Errorf("cannot train zero model")
}

func (zm *ZeroModel) Predict(feats feats.QueryEvaluation) predict.Prediction {
	return predict.Prediction{}
}

func (zm *ZeroModel) SetClassThreshold(v float64) {
	zm.ClassThreshold = v
}

func (zm *ZeroModel) GetClassThreshold() float64 {
	return zm.ClassThreshold
}

func (zm *ZeroModel) GetSlowQueriesThresholdTime() float64 {
	return zm.SlowQueriesThresholdTime
}

func (zm *ZeroModel) SaveToFile(string) error {
	return fmt.Errorf("cannot save zero model")
}

func (zm *ZeroModel) GetInfo() string {
	return "ZeroModel"
}
