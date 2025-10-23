package eval

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	randomforest "github.com/malaschitz/randomForest"
)

// RFModel wraps a Random Forest classifier for regression via quantile binning
type RFModel struct {
	Forest             randomforest.Forest
	TrainPositiveRatio float64
	RealPositiveRatio  float64
}

// NewRFModel creates a new Random Forest model with time binning
func NewRFModel() *RFModel {
	return &RFModel{
		Forest:             randomforest.Forest{},
		TrainPositiveRatio: 0.5,
		RealPositiveRatio:  0.03,
	}
}

// Train trains the random forest on query evaluations and actual times
func (m *RFModel) Train(dataModel *BasicModel, numTrees int) error {
	if len(dataModel.Evaluations) == 0 {
		return fmt.Errorf("no training data provided")
	}

	// Prepare training data
	var xData [][]float64
	var yData []int
	numProblematic := 0
	for _, eval := range dataModel.Evaluations {
		features := extractFeatures(eval)
		isProblematic := 0
		if eval.ProcTime >= dataModel.BinMidpoint {
			numProblematic++
			isProblematic = 1
		}
		xData = append(xData, features)
		yData = append(yData, isProblematic)
	}
	fmt.Println("#### NUM PROBLEMATIC: ", numProblematic, "  OUT OF TRANING SAMPLE ", len(dataModel.Evaluations))

	// Train the forest
	m.Forest.Data = randomforest.ForestData{
		X:     xData,
		Class: yData,
	}
	m.Forest.Train(numTrees)

	return nil
}

// Predict estimates query execution time using the trained forest
func (m *RFModel) Predict(eval QueryEvaluation) float64 {
	features := extractFeatures(eval)
	votes := m.Forest.Vote(features)
	//fmt.Println("VOTES: ", votes, ", strength: ", votes[1])
	return votes[1]
}

// extractFeatures converts QueryEvaluation to feature vector (same as Huber)
func extractFeatures(eval QueryEvaluation) []float64 {
	features := make([]float64, NumFeatures)
	idx := 0

	// Extract features for up to 4 positions
	for i := 0; i < MaxPositions; i++ {
		if i < len(eval.Positions) {
			pos := eval.Positions[i]
			// Position-specific features (normalized by concrete chars)
			features[idx] = float64(pos.Regexp.StartsWithWildCard)
			features[idx+1] = float64(pos.Regexp.NumWildcards)
			features[idx+2] = float64(pos.Regexp.HasRange)
			features[idx+3] = float64(pos.HasSmallCardAttr)
			features[idx+4] = float64(pos.Regexp.NumConcreteChars)
			features[idx+5] = float64(pos.NumAlternatives)
		}
		// If position doesn't exist, features remain 0
		idx += 6
	}

	// Global features
	features[24] = float64(eval.NumGlobConditions)
	features[25] = float64(eval.ContainsMeet)
	features[26] = float64(eval.ContainsUnion)
	features[27] = float64(eval.ContainsWithin)
	features[28] = float64(eval.ContainsContaining)
	features[29] = math.Log(eval.CorpusSize)
	features[30] = 1.0 // Bias term

	return features
}

// RFModelMetadata stores metadata for saving/loading
type RFModelMetadata struct {
	BinMidpoint float64 `json:"bin_midpoint"`
	NumTrees    int     `json:"num_trees"`
	NumClasses  int     `json:"num_classes"`
}

// SaveToFile saves the RF model to a file
// Note: github.com/malaschitz/randomForest doesn't provide native serialization,
// so we save metadata only. For production, you'd need to retrain or use a
// package with proper model serialization.
func (m *RFModel) SaveToFile(filePath string) error {
	metadata := RFModelMetadata{
		NumTrees:   len(m.Forest.Trees),
		NumClasses: m.Forest.Classes,
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("failed to encode model metadata: %w", err)
	}

	return fmt.Errorf("WARNING: Full model serialization not supported by randomForest package. Only metadata saved. Model must be retrained on startup.")
}

// LoadFromFile loads model metadata from file
// Note: This is a placeholder - the actual forest cannot be serialized/deserialized
// with the current randomForest package
func LoadRFModelFromFile(filePath string) (*RFModel, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var metadata RFModelMetadata
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode model metadata: %w", err)
	}

	model := &RFModel{}

	return model, fmt.Errorf("WARNING: Model structure cannot be loaded. Forest must be retrained.")
}
