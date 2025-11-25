package eval

import (
	"errors"

	"github.com/czcorpus/cqlizer/eval/nn"
	"github.com/czcorpus/cqlizer/eval/rf"
	"github.com/czcorpus/cqlizer/eval/xg"
	"github.com/czcorpus/cqlizer/eval/ym"
)

var ErrNoSuchModel = errors.New("no such model")

func GetMLModel(modelType, modelPath string) (MLModel, error) {

	var mlModel MLModel
	var err error

	switch modelType {
	case "rf":
		mlModel, err = rf.LoadFromFile(modelPath)
	case "nn":
		mlModel, err = nn.LoadFromFile(modelPath)
	case "xg":
		mlModel, err = xg.LoadFromFile(modelPath)
	case "ym":
		mlModel = &ym.Model{}
	default:
		err = ErrNoSuchModel
	}
	return mlModel, err
}
