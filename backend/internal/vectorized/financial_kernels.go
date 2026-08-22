package vectorized

import (
	"errors"
	"math"
)

// VectorizedFinancialKernels provides high-performance SIMD/WASM-ready numerical algorithms
type VectorizedFinancialKernels struct{}

func NewVectorizedFinancialKernels() *VectorizedFinancialKernels {
	return &VectorizedFinancialKernels{}
}

// CalculateXIRR solves Extended Internal Rate of Return via Newton-Raphson iteration
// Formula: sum(CF_i / (1 + r)^((d_i - d_0)/365.0)) = 0
func (k *VectorizedFinancialKernels) CalculateXIRR(dates []int64, amounts []float64, maxIter int, tol float64) (float64, error) {
	n := len(dates)
	if n < 2 || len(amounts) != n {
		return 0.0, errors.New("insufficient cashflows for XIRR calculation")
	}

	hasPositive, hasNegative := false, false
	for _, a := range amounts {
		if a > 0 {
			hasPositive = true
		}
		if a < 0 {
			hasNegative = true
		}
	}
	if !hasPositive || !hasNegative {
		return 0.0, errors.New("cashflows must contain both positive and negative values")
	}

	d0 := dates[0]
	yearFractions := make([]float64, n)
	for i := 0; i < n; i++ {
		yearFractions[i] = float64(dates[i]-d0) / (365.25 * 86400.0)
	}

	rate := 0.1 // 10% initial guess
	for iter := 0; iter < maxIter; iter++ {
		fVal := 0.0
		fDeriv := 0.0

		for i := 0; i < n; i++ {
			yf := yearFractions[i]
			denom := math.Pow(1.0+rate, yf)
			if math.IsNaN(denom) || math.IsInf(denom, 0) || denom == 0.0 {
				denom = 1e-12
			}

			fVal += amounts[i] / denom
			fDeriv -= (yf * amounts[i]) / (denom * (1.0 + rate))
		}

		if math.Abs(fVal) < tol {
			return rate, nil
		}
		if fDeriv == 0.0 {
			rate += 0.001
			continue
		}

		newRate := rate - (fVal / fDeriv)
		if math.IsNaN(newRate) || math.IsInf(newRate, 0) {
			return rate, nil
		}
		if math.Abs(newRate-rate) < tol {
			return newRate, nil
		}
		rate = newRate
	}

	return rate, nil
}

// CalculateModifiedDuration computes Macaulay / Modified Duration across vectorized bond cashflows
func (k *VectorizedFinancialKernels) CalculateModifiedDuration(cashflowDates []int64, cashflowAmounts []float64, ytm float64) float64 {
	n := len(cashflowDates)
	if n == 0 || ytm <= -1.0 {
		return 0.0
	}
	d0 := cashflowDates[0]
	pvSum := 0.0
	weightedTimeSum := 0.0

	for i := 0; i < n; i++ {
		t := float64(cashflowDates[i]-d0) / (365.25 * 86400.0)
		pv := cashflowAmounts[i] / math.Pow(1.0+ytm, t)
		pvSum += pv
		weightedTimeSum += t * pv
	}

	if pvSum == 0.0 {
		return 0.0
	}
	macDuration := weightedTimeSum / pvSum
	return macDuration / (1.0 + ytm)
}

// CalculateSharpeRatio computes annualized Sharpe Ratio across vectorized returns
func (k *VectorizedFinancialKernels) CalculateSharpeRatio(returns []float64, riskFreeRate float64, periodsPerYear float64) float64 {
	n := len(returns)
	if n < 2 {
		return 0.0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(n)

	variance := 0.0
	for _, r := range returns {
		variance += math.Pow(r-mean, 2)
	}
	stdDev := math.Sqrt(variance / float64(n-1))

	if stdDev == 0.0 {
		return 0.0
	}

	excessReturn := (mean * periodsPerYear) - riskFreeRate
	annualizedStdDev := stdDev * math.Sqrt(periodsPerYear)
	return excessReturn / annualizedStdDev
}
