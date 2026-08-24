package math

import (
	"fmt"
	"math"
)

// RoundFloat rounds a float64 to a specified precision
func RoundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

type DataPoint struct {
	X float64
	Y float64
}

func CalculateLinearRegressionSlope(dataPoints []DataPoint) (float64, error) {
	if len(dataPoints) < 2 {
		return 0.0, fmt.Errorf("not enough data points to calculate trend")
	}

	// Perform linear regression to find the slope
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(dataPoints))

	for _, point := range dataPoints {
		x := point.X
		y := point.Y
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	numerator := n*sumXY - sumX*sumY
	denominator := n*sumX2 - sumX*sumX

	if denominator == 0 {
		return 0.0, fmt.Errorf("denominator is zero, cannot calculate slope")
	}

	slope := numerator / denominator
	return slope, nil
}
