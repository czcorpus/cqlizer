package heatmap

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
)

// Matrix represents a 2D slice of float64 values
type Matrix [][]float64

// ScalingMethod represents the method used for scaling values
type ScalingMethod int

const (
	Linear ScalingMethod = iota
	Percentile
	Logarithmic
)

// GenerateHeatmap creates a heatmap image from a matrix with customizable cell size and color scaling
func GenerateHeatmap(matrix Matrix, filename string, cellSize int, method ScalingMethod) error {
	height := len(matrix)
	width := len(matrix[0])

	// Create a new image with larger dimensions
	img := image.NewRGBA(image.Rect(0, 0, width*cellSize, height*cellSize))

	// Flatten the matrix and sort for percentile calculation
	flattenedMatrix := flattenMatrix(matrix)
	sort.Float64s(flattenedMatrix)

	// Find min and max values in the matrix
	minVal, maxVal := flattenedMatrix[0], flattenedMatrix[len(flattenedMatrix)-1]

	// Generate the heatmap
	for y, row := range matrix {
		for x, val := range row {
			// Scale the value based on the chosen method
			scaledVal := scaleValue(val, minVal, maxVal, flattenedMatrix, method)

			// Map the scaled value to a color
			r := uint8(scaledVal * 255)
			b := uint8((1 - scaledVal) * 255)
			cellColor := color.RGBA{r, 0, b, 255}

			// Fill the cell with the color
			for dy := 0; dy < cellSize; dy++ {
				for dx := 0; dx < cellSize; dx++ {
					img.Set(x*cellSize+dx, y*cellSize+dy, cellColor)
				}
			}
		}
	}

	// Save the image
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// flattenMatrix converts a 2D matrix to a 1D slice
func flattenMatrix(matrix Matrix) []float64 {
	flattened := make([]float64, 0, len(matrix)*len(matrix[0]))
	for _, row := range matrix {
		flattened = append(flattened, row...)
	}
	return flattened
}

// scaleValue scales a value based on the chosen method
func scaleValue(val, minVal, maxVal float64, flattenedMatrix []float64, method ScalingMethod) float64 {
	switch method {
	case Percentile:
		return percentileScale(val, flattenedMatrix)
	case Logarithmic:
		return logScale(val, minVal, maxVal)
	default: // Linear
		return (val - minVal) / (maxVal - minVal)
	}
}

// percentileScale calculates the percentile of a value in the dataset
func percentileScale(val float64, flattenedMatrix []float64) float64 {
	index := sort.SearchFloat64s(flattenedMatrix, val)
	return float64(index) / float64(len(flattenedMatrix)-1)
}

// logScale applies logarithmic scaling to a value
func logScale(val, minVal, maxVal float64) float64 {
	// Ensure all values are positive by shifting if necessary
	if minVal <= 0 {
		val -= minVal - 1
		maxVal -= minVal - 1
		minVal = 1
	}
	return (math.Log(val) - math.Log(minVal)) / (math.Log(maxVal) - math.Log(minVal))
}
