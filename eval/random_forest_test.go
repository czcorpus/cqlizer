package eval

import (
	"testing"
)

func TestRFModelTraining(t *testing.T) {
	// Create sample training data
	evaluations := []QueryEvaluation{
		{
			Positions: []Position{
				{Index: 0, Regexp: Regexp{NumConcreteChars: 5, NumWildcards: 0}, HasSmallCardAttr: 0},
			},
			CorpusSize: 1000000.0,
		},
		{
			Positions: []Position{
				{Index: 0, Regexp: Regexp{NumConcreteChars: 3, NumWildcards: 2}, HasSmallCardAttr: 0},
			},
			CorpusSize: 1000000.0,
		},
		{
			Positions: []Position{
				{Index: 0, Regexp: Regexp{NumConcreteChars: 2, NumWildcards: 0}, HasSmallCardAttr: 1},
			},
			CorpusSize: 1000000.0,
		},
		{
			Positions: []Position{
				{Index: 0, Regexp: Regexp{NumConcreteChars: 1, NumWildcards: 5}, HasSmallCardAttr: 0},
			},
			CorpusSize: 1000000.0,
		},
	}

	actualTimes := []float64{0.1, 0.5, 2.0, 8.0}

	// Train model
	model := NewRFModel()
	err := model.Train(evaluations, actualTimes, 100, 4) // 100 trees, 4 bins
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Test predictions
	for i, eval := range evaluations {
		predicted := model.Predict(eval)
		t.Logf("Query %d: Actual=%.2f, Predicted=%.2f", i, actualTimes[i], predicted)
	}

	// Verify bins were created
	if len(model.TimeBins) != 3 {
		t.Errorf("Expected 3 bin boundaries, got %d", len(model.TimeBins))
	}
	if len(model.BinMidpoint) != 4 {
		t.Errorf("Expected 4 bin midpoints, got %d", len(model.BinMidpoint))
	}
}

func TestRFModelQuantileBinning(t *testing.T) {
	model := NewRFModel()
	times := []float64{0.1, 0.2, 0.5, 1.0, 2.0, 5.0, 10.0, 20.0}
	model.computeBins(times, 4)

	t.Logf("Bin boundaries: %v", model.TimeBins)
	t.Logf("Bin midpoints: %v", model.BinMidpoint)

	// Test bin assignment
	testCases := []struct {
		time        float64
		expectedBin int
	}{
		{0.15, 0}, // Should be in first bin
		{0.6, 1},  // Should be in second bin
		{3.0, 2},  // Should be in third bin
		{25.0, 3}, // Should be in last bin
	}

	for _, tc := range testCases {
		bin := model.timeToBin(tc.time)
		if bin != tc.expectedBin {
			t.Errorf("timeToBin(%.2f) = %d, expected %d", tc.time, bin, tc.expectedBin)
		}
	}
}
