package xg

import (
	"context"
	"fmt"
	"os"

	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/czcorpus/cqlizer/eval/predict"
	"github.com/dmitryikh/leaves"
	"github.com/vmihailenco/msgpack/v5"
)

type Model struct {
	ClassThreshold           float64
	SlowQueriesThresholdTime float64
	trainXData               [][]float64
	trainYData               []int
	xgboost                  *leaves.Ensemble
}

func (m *Model) IsInferenceOnly() bool {
	return true
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
	return "XGBOOST MODEL"
}

func LoadFromFile(filePath string) (*Model, error) {
	model, err := leaves.LGEnsembleFromFile(filePath, true)
	if err != nil {
		return nil, err
	}
	return &Model{xgboost: model}, nil
}

func NewModel() *Model {
	return &Model{
		ClassThreshold: 0.5,
	}
}
