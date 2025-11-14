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
