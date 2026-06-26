package api

import (
	"math"
)

// CalculateIRR finds the internal rate of return for a series of cash flows.
func CalculateIRR(cashFlows []float64, guess float64) float64 {
	const maxIter = 100
	const tol = 1e-7

	rate := guess
	if rate == 0 {
		rate = 0.1
	}

	for i := 0; i < maxIter; i++ {
		npv := 0.0
		dnpv := 0.0
		for t, cf := range cashFlows {
			npv += cf / math.Pow(1+rate, float64(t))
			if t > 0 {
				dnpv -= float64(t) * cf / math.Pow(1+rate, float64(t+1))
			}
		}

		if math.Abs(dnpv) < tol {
			break // Avoid division by zero
		}

		newRate := rate - npv/dnpv

		if math.Abs(newRate-rate) < tol {
			return newRate
		}
		rate = newRate
	}

	return math.NaN() // Failed to converge
}
