package api

import (
	"fmt"
	"math"
	"sort"
	"time"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat/distuv"
	"math/rand/v2"
)

// normCDF calculates the cumulative distribution function for the standard normal distribution.
func normCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

// CalculateBlackScholes computes the price of a European option.
func CalculateBlackScholes(optionType string, S, K, T, r, sigma float64) (float64, error) {
	if T <= 0 || sigma <= 0 {
		return 0, fmt.Errorf("time to maturity and volatility must be positive")
	}

	d1 := (math.Log(S/K) + (r+sigma*sigma/2)*T) / (sigma * math.Sqrt(T))
	d2 := d1 - sigma*math.Sqrt(T)

	if optionType == "call" {
		return S*normCDF(d1) - K*math.Exp(-r*T)*normCDF(d2), nil
	}
	if optionType == "put" {
		return K*math.Exp(-r*T)*normCDF(-d2) - S*normCDF(-d1), nil
	}

	return 0, fmt.Errorf("invalid option type: %s", optionType)
}

// CalculateParametricVaR calculates Value at Risk using the variance-covariance method.
func CalculateParametricVaR(confidence, mean, stdDev, holdingPeriod float64) (float64, error) {
	if confidence <= 0 || confidence >= 1 {
		return 0, fmt.Errorf("confidence level must be between 0 and 1")
	}
	zScore := distuv.UnitNormal.Quantile(confidence)
	return (mean * holdingPeriod) - (stdDev * zScore * math.Sqrt(holdingPeriod)), nil
}

// CalculateCVaR calculates Conditional Value at Risk (Expected Shortfall).
func CalculateCVaR(confidence float64, returns []float64) (float64, error) {
	if len(returns) == 0 {
		return 0, fmt.Errorf("returns slice cannot be empty")
	}
	if confidence <= 0 || confidence >= 1 {
		return 0, fmt.Errorf("confidence level must be between 0 and 1")
	}

	sort.Float64s(returns)
	k := int(float64(len(returns)) * (1 - confidence))
	if k >= len(returns) {
		k = len(returns) - 1
	}

	var tailSum float64
	for i := 0; i <= k; i++ {
		tailSum += returns[i]
	}

	return tailSum / float64(k+1), nil
}

// CalculateSCR calculates the Solvency Capital Requirement using a correlation matrix.
func CalculateSCR(riskModules map[string]float64, correlationMatrix [][]float64) (float64, error) {
	// This is a simplified example. A real SCR calculation is far more complex.
	// We assume the order of risk_modules matches the correlation matrix.
	var riskValues []float64
	// Ensure a consistent order for matrix multiplication
	keys := make([]string, 0, len(riskModules))
	for k := range riskModules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		riskValues = append(riskValues, riskModules[k])
	}

	n := len(riskValues)
	if n == 0 {
		return 0, nil
	}
	if len(correlationMatrix) != n || len(correlationMatrix[0]) != n {
		return 0, fmt.Errorf("correlation matrix dimensions do not match number of risk modules")
	}

	V := mat.NewVecDense(n, riskValues)
	C := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		C.SetRow(i, correlationMatrix[i])
	}

	var temp mat.VecDense
	temp.MulVec(C, V)

	scrSquared := mat.Dot(&temp, V)
	return math.Sqrt(scrSquared), nil
}

// CalculateCreditVaR calculates Credit Value at Risk using a simplified simulation.
func CalculateCreditVaR(confidence float64, exposures, pds, lgds []float64) (float64, error) {
	if len(exposures) != len(pds) || len(exposures) != len(lgds) {
		return 0, fmt.Errorf("exposures, pds, and lgds must have the same length")
	}

	numSimulations := 10000
	losses := make([]float64, numSimulations)
	r := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))

	for i := 0; i < numSimulations; i++ {
		totalLoss := 0.0
		for j := 0; j < len(exposures); j++ {
			if r.Float64() < pds[j] {
				totalLoss += exposures[j] * lgds[j]
			}
		}
		losses[i] = totalLoss
	}

	sort.Float64s(losses)
	index := int(float64(numSimulations) * confidence)
	if index >= numSimulations {
		index = numSimulations - 1
	}

	return losses[index], nil
}
