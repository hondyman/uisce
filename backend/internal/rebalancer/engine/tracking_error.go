package engine

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// TrackingErrorCalculator computes the ex-ante tracking error
type TrackingErrorCalculator struct{}

func NewTrackingErrorCalculator() *TrackingErrorCalculator {
	return &TrackingErrorCalculator{}
}

// CalculateTE computes sqrt(w_active' * Sigma * w_active)
func (c *TrackingErrorCalculator) CalculateTE(weights []float64, benchmarkWeights []float64, covariance *mat.Dense) float64 {
	n := len(weights)
	if n != len(benchmarkWeights) {
		return 0.0 // Error handling omitted for brevity
	}

	// 1. Calculate Active Weights (w_p - w_b)
	activeWeights := make([]float64, n)
	for i := 0; i < n; i++ {
		activeWeights[i] = weights[i] - benchmarkWeights[i]
	}
	wActive := mat.NewVecDense(n, activeWeights)

	// 2. Compute Variance: w' * Sigma * w
	// temp = Sigma * w
	temp := mat.NewVecDense(n, nil)
	temp.MulVec(covariance, wActive)

	// variance = w' * temp
	variance := mat.Dot(wActive, temp)

	// 3. Return Standard Deviation (TE)
	if variance < 0 {
		return 0.0 // Should not happen with PSD covariance matrix
	}
	return math.Sqrt(variance)
}
