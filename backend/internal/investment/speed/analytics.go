package speed

import (
	"math"
)

// AnalyticsEngine handles performance calculations
type AnalyticsEngine struct {
	// In a real implementation, this would hold a StarRocks client to query Iceberg materialized views
}

func NewAnalyticsEngine() *AnalyticsEngine {
	return &AnalyticsEngine{}
}

// CalculateTWR computes the Time-Weighted Return from a sum of log returns.
// Formula: TWR = exp(sum(log(1 + r))) - 1
func (e *AnalyticsEngine) CalculateTWR(logSumReturn float64) float64 {
	return math.Exp(logSumReturn) - 1
}

// CalculateAnnualizedReturn computes the annualized return given a TWR and number of years.
// Formula: (1 + TWR)^(1/n) - 1
func (e *AnalyticsEngine) CalculateAnnualizedReturn(twr float64, years float64) float64 {
	if years <= 0 {
		return 0
	}
	return math.Pow(1+twr, 1/years) - 1
}

// MockQueryTWR simulates fetching the pre-aggregated log sum from StarRocks Iceberg
func (e *AnalyticsEngine) MockQueryTWR(portfolioID uint64, months int) float64 {
	// Simulate a 10% annual return (approx 0.8% monthly)
	// log(1.008) approx 0.00796
	monthlyLogReturn := 0.00796
	return monthlyLogReturn * float64(months)
}
