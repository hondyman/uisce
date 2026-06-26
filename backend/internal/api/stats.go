package api

import (
	"fmt"
	"math"
)

func variance(data []float64, population bool) float64 {
	if len(data) == 0 {
		return 0.0
	}
	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(len(data))

	sumSqDiff := 0.0
	for _, v := range data {
		sumSqDiff += math.Pow(v-mean, 2)
	}

	if population {
		return sumSqDiff / float64(len(data))
	}
	if len(data) < 2 {
		return 0.0
	}
	return sumSqDiff / float64(len(data)-1)
}

func covariance(data1, data2 []float64) (float64, error) {
	if len(data1) != len(data2) || len(data1) == 0 {
		return 0, fmt.Errorf("data series must have the same, non-zero length")
	}
	mean1, mean2 := 0.0, 0.0
	for i := range data1 {
		mean1 += data1[i]
		mean2 += data2[i]
	}
	mean1 /= float64(len(data1))
	mean2 /= float64(len(data2))

	cov := 0.0
	for i := range data1 {
		cov += (data1[i] - mean1) * (data2[i] - mean2)
	}
	return cov / float64(len(data1)-1), nil // Sample covariance
}

func CalculateStdev(returns []float64, population bool) float64 {
	return math.Sqrt(variance(returns, population))
}

func CalculateBeta(assetReturns, benchReturns []float64) (float64, error) {
	cov, err := covariance(assetReturns, benchReturns)
	if err != nil {
		return 0, err
	}
	benchVar := variance(benchReturns, false) // Use sample variance
	if benchVar == 0 {
		return 0, fmt.Errorf("benchmark returns have zero variance")
	}
	return cov / benchVar, nil
}
