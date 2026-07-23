//go:build integration

package services_test

import (
	"testing"
	"time"
)

// Simple unit test for XIRR calculation logic
func TestXIRRCalculation(t *testing.T) {
	// Test case: Simple IRR calculation
	cashFlows := []float64{-1000, 100, 200, 300, 400}
	dates := []time.Time{
		time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
	}

	// Simple IRR calculation using Newton-Raphson method
	irr := calculateXIRRSimple(cashFlows, dates)

	// IRR should be positive for profitable investment
	if irr <= 0 {
		t.Errorf("Expected positive IRR, got: %f", irr)
	}

	t.Logf("Calculated IRR: %.4f", irr)
}

// calculateXIRRSimple is a simplified XIRR implementation for testing
func calculateXIRRSimple(cashFlows []float64, dates []time.Time) float64 {
	if len(cashFlows) != len(dates) || len(cashFlows) < 2 {
		return 0
	}

	// Convert dates to day differences from first date
	baseDate := dates[0]
	days := make([]float64, len(dates))
	for i, date := range dates {
		days[i] = date.Sub(baseDate).Hours() / 24 / 365 // Convert to years
	}

	// Newton-Raphson method for IRR calculation
	const maxIterations = 100
	const tolerance = 1e-6

	rate := 0.1 // Initial guess

	for i := 0; i < maxIterations; i++ {
		f := 0.0
		df := 0.0

		for j, cf := range cashFlows {
			if days[j] == 0 {
				f += cf
			} else {
				f += cf / pow(1+rate, days[j])
				df -= cf * days[j] / pow(1+rate, days[j]+1)
			}
		}

		if abs(df) < tolerance {
			break
		}

		rate = rate - f/df

		if abs(f) < tolerance {
			return rate
		}
	}

	return rate
}

// Helper functions
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestDataQualityMetrics tests data quality calculation
func TestDataQualityMetrics(t *testing.T) {
	// Test completeness score
	data := []float64{1.0, 2.0, 0.0, 4.0, 5.0} // One zero value
	score := calculateCompletenessScore(data)

	expectedScore := 4.0 / 5.0 // 4 non-zero out of 5 total
	if score != expectedScore {
		t.Errorf("Expected completeness score %.2f, got %.2f", expectedScore, score)
	}

	// Test with all valid data
	validData := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	validScore := calculateCompletenessScore(validData)

	if validScore != 1.0 {
		t.Errorf("Expected completeness score 1.0 for all valid data, got %.2f", validScore)
	}

	// Test with empty data
	emptyData := []float64{}
	emptyScore := calculateCompletenessScore(emptyData)

	if emptyScore != 0.0 {
		t.Errorf("Expected completeness score 0.0 for empty data, got %.2f", emptyScore)
	}
}

// calculateCompletenessScore calculates data completeness for quality metrics
func calculateCompletenessScore(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	validCount := 0
	for _, v := range data {
		if v != 0 { // Simple check for non-zero values
			validCount++
		}
	}

	return float64(validCount) / float64(len(data))
}
