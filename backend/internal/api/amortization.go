package api

import (
	"fmt"
	"math"
)

// CalculateAmortizationPayment calculates the fixed periodic payment for a loan (PMT).
func CalculateAmortizationPayment(rate float64, periods int, principal float64) (float64, error) {
	if periods <= 0 {
		return 0, fmt.Errorf("number of periods must be positive")
	}
	if rate == 0 {
		return -principal / float64(periods), nil
	}
	ratePow := math.Pow(1+rate, float64(periods))
	payment := -principal * (rate * ratePow) / (ratePow - 1)
	return payment, nil
}
