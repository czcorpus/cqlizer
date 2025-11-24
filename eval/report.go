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

package eval

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/czcorpus/cqlizer/eval/feats"
)

type misclassification struct {
	Evaluation feats.QueryEvaluation `json:"evaluation"`
	MLOutput   float64               `json:"mlOutput"`
	Threshold  float64               `json:"threhold"`
	NumRepeat  int                   `json:"numRepeat"`
	Type       string                `json:"type"`
}

func (m misclassification) AbsErrorSize() float64 {
	return math.Abs(m.MLOutput - m.Threshold)
}

// ------------------------

type Reporter struct {
	RFAccuracyScript       string
	misclassQueries        map[string]misclassification
	MisclassQueriesOutPath string
}

func (reporter *Reporter) AddMisclassifiedQuery(q feats.QueryEvaluation, mlOut, threshold, slowProcTime float64) {
	predictedSlow := mlOut >= threshold
	actuallySlow := q.ProcTime >= slowProcTime
	var tp string
	if actuallySlow && !predictedSlow {
		tp = "FN"

	} else if !actuallySlow && predictedSlow {
		tp = "FP"
	}
	if reporter.misclassQueries == nil {
		reporter.misclassQueries = make(map[string]misclassification)
	}
	curr, ok := reporter.misclassQueries[q.UniqKey()]
	if ok {
		curr.MLOutput += mlOut
		curr.NumRepeat += 1
		if tp != curr.Type {
			curr.Type = "*"
		}
		reporter.misclassQueries[q.UniqKey()] = curr

	} else {
		reporter.misclassQueries[q.UniqKey()] = misclassification{
			Evaluation: q,
			MLOutput:   mlOut,
			Threshold:  threshold,
			NumRepeat:  1,
			Type:       tp,
		}
	}
}

func (reporter *Reporter) sortedMisclassifiedQueries() []misclassification {
	ans := make([]misclassification, len(reporter.misclassQueries))
	i := 0
	for _, v := range reporter.misclassQueries {
		v.MLOutput /= float64(v.NumRepeat)
		ans[i] = v
		i++
	}
	slices.SortFunc(
		ans,
		func(v1, v2 misclassification) int {
			if v1.NumRepeat < v2.NumRepeat {
				return 1

			} else if v1.NumRepeat > v2.NumRepeat {
				return -1

			} else {
				if v1.AbsErrorSize() < v2.AbsErrorSize() {
					return 1
				}
				return -1
			}
		},
	)
	return ans
}

func (reporter *Reporter) ShowMisclassifiedQueries() {
	for i, v := range reporter.misclassQueries {
		fmt.Fprintf(os.Stderr, "%s\t%.2f\t%s\n", i, v.AbsErrorSize(), v.Evaluation.OrigQuery)
	}
}

func (reporter *Reporter) SaveMisclassifiedQueries() error {
	data := reporter.sortedMisclassifiedQueries()
	if reporter.MisclassQueriesOutPath == "" {
		return fmt.Errorf("misclassQueriesOutPath is not set")
	}

	f, err := os.Create(reporter.MisclassQueriesOutPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", reporter.MisclassQueriesOutPath, err)
	}
	defer f.Close()

	for _, item := range data {
		_, err := fmt.Fprintf(f, "%.0f\t%.2f\t%0.2f\t%s(%d)\t%s\n",
			math.Exp(item.Evaluation.CorpusSize), item.Evaluation.ProcTime, item.MLOutput, item.Type, item.NumRepeat, item.Evaluation.OrigQuery)
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	return nil
}

// PlotModelAccuracy creates a chart from CSV data using a Python plotting script.
// The output file name is derived from the provided modelPath
func (reporter *Reporter) PlotRFAccuracy(data, chartLabel, modelPath string) error {
	chartFilePath := fmt.Sprintf("%s.png", strings.TrimSuffix(modelPath, filepath.Ext(modelPath)))
	cmd := exec.Command("python3", "-c", reporter.RFAccuracyScript, "-o", chartFilePath, "-t", chartLabel)
	cmd.Stdin = bytes.NewBufferString(data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute plotting script: %w\nStderr: %s", err, stderr.String())
	}

	return nil
}
