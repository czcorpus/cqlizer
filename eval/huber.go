package eval

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

const MaxPositions = 4

// -----------------------------------
// Huber Regression Implementation
// -----------------------------------

type HuberRegressor struct {
	Delta         float64
	LearningRate  float64
	MaxIterations int
	Tolerance     float64

	// Normalization parameters
	featureMeans []float64
	featureStds  []float64
	targetMean   float64
	targetStd    float64
}

func NewHuberRegressor() *HuberRegressor {
	return &HuberRegressor{
		Delta:         1.35,
		LearningRate:  0.01,
		MaxIterations: 10000,
		Tolerance:     1e-6,
	}
}

// huberLoss calculates the Huber loss for a single residual
func (hr *HuberRegressor) huberLoss(residual float64) float64 {
	absResidual := math.Abs(residual)
	if absResidual <= hr.Delta {
		return 0.5 * residual * residual
	}
	return hr.Delta * (absResidual - 0.5*hr.Delta)
}

// huberDerivative calculates the derivative of Huber loss
func (hr *HuberRegressor) huberDerivative(residual float64) float64 {
	if math.Abs(residual) <= hr.Delta {
		return residual
	}
	if residual > 0 {
		return hr.Delta
	}
	return -hr.Delta
}

// computeNormalizationParams calculates mean and std for features and targets
func (hr *HuberRegressor) computeNormalizationParams(features [][]float64, targets []float64) {
	n := len(features)
	if n == 0 {
		return
	}

	numFeatures := len(features[0])

	hr.featureMeans = make([]float64, numFeatures)
	hr.featureStds = make([]float64, numFeatures)

	// Compute feature means
	for i := 0; i < n; i++ {
		for j := 0; j < numFeatures; j++ {
			hr.featureMeans[j] += features[i][j]
		}
	}
	for j := 0; j < numFeatures; j++ {
		hr.featureMeans[j] /= float64(n)
	}

	// Compute feature standard deviations
	for i := 0; i < n; i++ {
		for j := 0; j < numFeatures; j++ {
			diff := features[i][j] - hr.featureMeans[j]
			hr.featureStds[j] += diff * diff
		}
	}
	for j := 0; j < numFeatures; j++ {
		hr.featureStds[j] = math.Sqrt(hr.featureStds[j] / float64(n))
		// Avoid division by zero
		if hr.featureStds[j] < 1e-10 {
			hr.featureStds[j] = 1.0
		}
	}

	// Compute target mean and std
	hr.targetMean = 0.0
	for _, t := range targets {
		hr.targetMean += t
	}
	hr.targetMean /= float64(n)

	hr.targetStd = 0.0
	for _, t := range targets {
		diff := t - hr.targetMean
		hr.targetStd += diff * diff
	}
	hr.targetStd = math.Sqrt(hr.targetStd / float64(n))
	if hr.targetStd < 1e-10 {
		hr.targetStd = 1.0
	}
}

// normalizeFeatures normalizes feature vectors using stored parameters
func (hr *HuberRegressor) normalizeFeatures(features [][]float64) [][]float64 {
	normalized := make([][]float64, len(features))
	for i := range features {
		normalized[i] = make([]float64, len(features[i]))
		for j := range features[i] {
			// Skip bias term (last feature)
			if j == len(features[i])-1 {
				normalized[i][j] = features[i][j]
			} else {
				normalized[i][j] = (features[i][j] - hr.featureMeans[j]) / hr.featureStds[j]
			}
		}
	}
	return normalized
}

// normalizeTargets normalizes target values
func (hr *HuberRegressor) normalizeTargets(targets []float64) []float64 {
	normalized := make([]float64, len(targets))
	for i := range targets {
		normalized[i] = (targets[i] - hr.targetMean) / hr.targetStd
	}
	return normalized
}

// denormalizeParams converts normalized parameters back to original scale
func (hr *HuberRegressor) denormalizeParams(normalizedParams ModelParams) ModelParams {
	normalizedSlice := normalizedParams.ToSlice()
	denormalizedSlice := make([]float64, len(normalizedSlice))

	// Convert normalized coefficients back to original scale
	for i := 0; i < len(normalizedSlice)-1; i++ { // -1 to skip bias
		denormalizedSlice[i] = normalizedSlice[i] * hr.targetStd / hr.featureStds[i]
	}

	// Adjust bias term
	biasAdjustment := hr.targetMean
	for i := 0; i < len(normalizedSlice)-1; i++ {
		biasAdjustment -= denormalizedSlice[i] * hr.featureMeans[i]
	}
	denormalizedSlice[len(normalizedSlice)-1] = normalizedSlice[len(normalizedSlice)-1]*hr.targetStd + biasAdjustment

	return SliceToModelParams(denormalizedSlice)
}

// Fit performs Huber regression with feature normalization
func (hr *HuberRegressor) Fit(evaluations []QueryEvaluation) ModelParams {

	n := len(evaluations)

	// Extract and normalize features
	features := make([][]float64, n)
	actualTimes := make([]float64, n)
	for i := range evaluations {
		features[i] = hr.extractFeatures(evaluations[i])
		actualTimes[i] = evaluations[i].ProcTime
	}

	hr.computeNormalizationParams(features, actualTimes)
	normalizedFeatures := hr.normalizeFeatures(features)
	normalizedTargets := hr.normalizeTargets(actualTimes)

	// Initialize parameters (all zeros for normalized data works well)
	params := ModelParams{}

	var prevLoss float64 = math.MaxFloat64

	// Gradient descent with Huber loss on normalized data
	for iter := 0; iter < hr.MaxIterations; iter++ {
		gradients := make([]float64, NumFeatures)
		totalLoss := 0.0

		// Compute predictions using normalized features
		for i := 0; i < n; i++ {
			predicted := hr.predictNormalized(normalizedFeatures[i], params)
			residual := predicted - normalizedTargets[i]

			totalLoss += hr.huberLoss(residual)
			derivative := hr.huberDerivative(residual)

			// Accumulate gradients for each parameter
			for j := 0; j < NumFeatures; j++ {
				gradients[j] += derivative * normalizedFeatures[i][j]
			}
		}

		// Average the gradients
		for j := range gradients {
			gradients[j] /= float64(n)
		}

		// Update parameters
		paramsSlice := params.ToSlice()
		for j := range paramsSlice {
			paramsSlice[j] -= hr.LearningRate * gradients[j]
		}
		params = SliceToModelParams(paramsSlice)

		// Check convergence
		avgLoss := totalLoss / float64(n)
		if math.Abs(prevLoss-avgLoss) < hr.Tolerance {
			break
		}
		prevLoss = avgLoss
	}

	// Denormalize parameters before returning
	return hr.denormalizeParams(params)
}

// predictNormalized makes prediction on normalized feature vector
func (hr *HuberRegressor) predictNormalized(features []float64, params ModelParams) float64 {
	paramsSlice := params.ToSlice()
	var sum float64
	for i, feat := range features {
		sum += paramsSlice[i] * feat
	}
	return sum
}

// extractFeatures converts a QueryEvaluation to feature vector
func (hr *HuberRegressor) extractFeatures(eval QueryEvaluation) []float64 {
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
	features[30] = float64(eval.AlignedPart)
	features[31] = 1.0 // Bias term

	return features
}

// CrossValidationResult holds the results of cross-validation
type CrossValidationResult struct {
	FoldResults []FoldResult
	MeanR2      float64
	StdR2       float64
	MeanMAE     float64
	StdMAE      float64
	MeanRMSE    float64
	StdRMSE     float64
}

type FoldResult struct {
	Fold int
	R2   float64
	MAE  float64
	RMSE float64
}

// CrossValidate performs k-fold cross-validation
func (hr *HuberRegressor) CrossValidate(evaluations []QueryEvaluation, k int) CrossValidationResult {
	n := len(evaluations)

	// Shuffle indices
	rand.Seed(time.Now().UnixNano())
	indices := rand.Perm(n)

	foldSize := n / k
	results := make([]FoldResult, k)

	for fold := 0; fold < k; fold++ {
		// Split data into train and test
		testStart := fold * foldSize
		testEnd := testStart + foldSize
		if fold == k-1 {
			testEnd = n // include remaining samples in last fold
		}

		var trainEvals []QueryEvaluation
		var testEvals []QueryEvaluation

		for i := 0; i < n; i++ {
			idx := indices[i]
			if i >= testStart && i < testEnd {
				testEvals = append(testEvals, evaluations[idx])

			} else {
				trainEvals = append(trainEvals, evaluations[idx])
			}
		}

		// Train on training set
		params := hr.Fit(trainEvals)

		// Evaluate on test set
		r2, mae, rmse := hr.EvaluateDetailed(testEvals, params)

		results[fold] = FoldResult{
			Fold: fold + 1,
			R2:   r2,
			MAE:  mae,
			RMSE: rmse,
		}
	}

	// Compute statistics
	var sumR2, sumMAE, sumRMSE float64
	for _, result := range results {
		sumR2 += result.R2
		sumMAE += result.MAE
		sumRMSE += result.RMSE
	}

	meanR2 := sumR2 / float64(k)
	meanMAE := sumMAE / float64(k)
	meanRMSE := sumRMSE / float64(k)

	var sumSqDiffR2, sumSqDiffMAE, sumSqDiffRMSE float64
	for _, result := range results {
		sumSqDiffR2 += (result.R2 - meanR2) * (result.R2 - meanR2)
		sumSqDiffMAE += (result.MAE - meanMAE) * (result.MAE - meanMAE)
		sumSqDiffRMSE += (result.RMSE - meanRMSE) * (result.RMSE - meanRMSE)
	}

	stdR2 := math.Sqrt(sumSqDiffR2 / float64(k))
	stdMAE := math.Sqrt(sumSqDiffMAE / float64(k))
	stdRMSE := math.Sqrt(sumSqDiffRMSE / float64(k))

	return CrossValidationResult{
		FoldResults: results,
		MeanR2:      meanR2,
		StdR2:       stdR2,
		MeanMAE:     meanMAE,
		StdMAE:      stdMAE,
		MeanRMSE:    meanRMSE,
		StdRMSE:     stdRMSE,
	}
}

// EvaluateDetailed returns R², MAE, and RMSE
func (hr *HuberRegressor) EvaluateDetailed(evaluations []QueryEvaluation, params ModelParams) (r2 float64, mae float64, rmse float64) {
	n := len(evaluations)
	var sumSquaredResiduals, sumAbsResiduals, sumActual float64

	for i := 0; i < n; i++ {
		sumActual += evaluations[i].ProcTime
	}
	meanActual := sumActual / float64(n)

	var sumSquaredTotal float64
	for i := 0; i < n; i++ {
		predicted := evaluations[i].Cost(params)
		residual := predicted - evaluations[i].ProcTime

		sumSquaredResiduals += residual * residual
		sumAbsResiduals += math.Abs(residual)
		sumSquaredTotal += (evaluations[i].ProcTime - meanActual) * (evaluations[i].ProcTime - meanActual)
	}

	r2 = 1.0 - (sumSquaredResiduals / sumSquaredTotal)
	mae = sumAbsResiduals / float64(n)
	rmse = math.Sqrt(sumSquaredResiduals / float64(n))

	return r2, mae, rmse
}

// Evaluate returns R² score and mean absolute error (backward compatibility)
func (hr *HuberRegressor) Evaluate(evaluations []QueryEvaluation, params ModelParams) (r2 float64, mae float64) {
	r2, mae, _ = hr.EvaluateDetailed(evaluations, params)
	return r2, mae
}

func RunModel(evaluations []QueryEvaluation) {
	if len(evaluations) == 0 {
		fmt.Println("Error: No data provided for TestHuber")
		return
	}

	// Perform k-fold cross-validation
	fmt.Println("=== Cross-Validation (5-fold) ===")
	regressor := NewHuberRegressor()
	cvResult := regressor.CrossValidate(evaluations, 5)

	// Print results for each fold
	fmt.Println("\nFold Results:")
	for _, fold := range cvResult.FoldResults {
		fmt.Printf("Fold %d: R²=%.4f, MAE=%.4f, RMSE=%.4f\n",
			fold.Fold, fold.R2, fold.MAE, fold.RMSE)
	}

	// Print overall statistics
	fmt.Printf("\nOverall Performance:\n")
	fmt.Printf("Mean R²: %.4f (±%.4f)\n", cvResult.MeanR2, cvResult.StdR2)
	fmt.Printf("Mean MAE: %.4f (±%.4f) seconds\n", cvResult.MeanMAE, cvResult.StdMAE)
	fmt.Printf("Mean RMSE: %.4f (±%.4f) seconds\n", cvResult.MeanRMSE, cvResult.StdRMSE)

	// Train final model on all data
	fmt.Println("\n=== Training Final Model ===")
	finalParams := regressor.Fit(evaluations)

	fmt.Printf("\nOptimized Parameters:\n")
	fmt.Printf("Position 0:\n")
	fmt.Printf("  WildcardPrefix: %.6f\n", finalParams.WildcardPrefix0)
	fmt.Printf("  Wildcards:      %.6f\n", finalParams.Wildcards0)
	fmt.Printf("  RangeOp:        %.6f\n", finalParams.RangeOp0)
	fmt.Printf("  SmallCardAttr:  %.6f\n", finalParams.SmallCardAttr0)
	fmt.Printf("  ConcreteChars:  %.6f\n", finalParams.ConcreteChars0)
	fmt.Printf("  NumPosAlts:     %.6f\n", finalParams.NumPosAlts0)
	fmt.Printf("Position 1:\n")
	fmt.Printf("  WildcardPrefix: %.6f\n", finalParams.WildcardPrefix1)
	fmt.Printf("  Wildcards:      %.6f\n", finalParams.Wildcards1)
	fmt.Printf("  RangeOp:        %.6f\n", finalParams.RangeOp1)
	fmt.Printf("  SmallCardAttr:  %.6f\n", finalParams.SmallCardAttr1)
	fmt.Printf("  ConcreteChars:  %.6f\n", finalParams.ConcreteChars1)
	fmt.Printf("  NumPosAlts:     %.6f\n", finalParams.NumPosAlts1)
	fmt.Printf("Position 2:\n")
	fmt.Printf("  WildcardPrefix: %.6f\n", finalParams.WildcardPrefix2)
	fmt.Printf("  Wildcards:      %.6f\n", finalParams.Wildcards2)
	fmt.Printf("  RangeOp:        %.6f\n", finalParams.RangeOp2)
	fmt.Printf("  SmallCardAttr:  %.6f\n", finalParams.SmallCardAttr2)
	fmt.Printf("  ConcreteChars:  %.6f\n", finalParams.ConcreteChars2)
	fmt.Printf("  NumPosAlts:     %.6f\n", finalParams.NumPosAlts2)
	fmt.Printf("Position 3:\n")
	fmt.Printf("  WildcardPrefix: %.6f\n", finalParams.WildcardPrefix3)
	fmt.Printf("  Wildcards:      %.6f\n", finalParams.Wildcards3)
	fmt.Printf("  RangeOp:        %.6f\n", finalParams.RangeOp3)
	fmt.Printf("  SmallCardAttr:  %.6f\n", finalParams.SmallCardAttr3)
	fmt.Printf("  ConcreteChars:  %.6f\n", finalParams.ConcreteChars3)
	fmt.Printf("  NumPosAlts:     %.6f\n", finalParams.NumPosAlts3)
	fmt.Printf("Global:\n")
	fmt.Printf("  GlobCond:       %.6f\n", finalParams.GlobCond)
	fmt.Printf("  Meet:           %.6f\n", finalParams.Meet)
	fmt.Printf("  Union:          %.6f\n", finalParams.Union)
	fmt.Printf("  Within:         %.6f\n", finalParams.Within)
	fmt.Printf("  Containing:     %.6f\n", finalParams.Containing)
	fmt.Printf("  CorpusSize:     %.6f\n", finalParams.CorpusSize)
	fmt.Printf("  AlignedPart:    %.6f\n", finalParams.AlignedPart)
	fmt.Printf("  Bias:           %.6f\n", finalParams.Bias)

	// Final evaluation on all data
	r2, mae, rmse := regressor.EvaluateDetailed(evaluations, finalParams)
	fmt.Printf("\nFinal Model Performance (on all data):\n")
	fmt.Printf("R² Score: %.4f\n", r2)
	fmt.Printf("MAE: %.4f seconds\n", mae)
	fmt.Printf("RMSE: %.4f seconds\n", rmse)

	// Example: Analyze how costs decrease with position
	fmt.Println("\n=== Cost Analysis by Position ===")
	fmt.Printf("Wildcard cost decreases: Pos0=%.4f, Pos1=%.4f, Pos2=%.4f, Pos3=%.4f\n",
		finalParams.Wildcards0, finalParams.Wildcards1,
		finalParams.Wildcards2, finalParams.Wildcards3)
	var numDec int
	if finalParams.Wildcards1 < finalParams.Wildcards0 {
		numDec++
	}
	if finalParams.Wildcards2 < finalParams.Wildcards1 {
		numDec++
	}
	if finalParams.Wildcards3 < finalParams.Wildcards2 {
		numDec++
	}

	if numDec >= 2 {
		fmt.Println("✓ Model (mostly) learned that later positions are cheaper (as expected)")
	}

	// Save the model to file
	timestamp := time.Now().Format("20060102T150405")
	modelPath := fmt.Sprintf("cqlizer_model_%s.json", timestamp)
	fmt.Printf("\n=== Saving Model ===\n")
	if err := finalParams.SaveToFile(modelPath); err != nil {
		fmt.Printf("Error saving model: %v\n", err)
	} else {
		fmt.Printf("Model saved to %s\n", modelPath)
	}
}
