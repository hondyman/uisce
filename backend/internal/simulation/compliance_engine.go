package simulation

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

// ComplianceEngine defines the interface for regulatory and mandate checks
type ComplianceEngine interface {
	CheckCompliance(ctx context.Context, req ComplianceRequest) (*ComplianceResult, error)
}

// MockComplianceEngine implements basic placeholder logic
type MockComplianceEngine struct{}

func NewMockComplianceEngine() *MockComplianceEngine {
	return &MockComplianceEngine{}
}

func (m *MockComplianceEngine) CheckCompliance(ctx context.Context, req ComplianceRequest) (*ComplianceResult, error) {
	result := &ComplianceResult{
		Status:         "PASSED",
		NewIssues:      []ComplianceIssue{},
		ResolvedIssues: []ComplianceIssue{},
		ChangedIssues:  []ComplianceIssue{},
		Metrics:        []SimulationMetric{},
	}

	// Calculate Total Value
	totalValue := 0.0
	assetValues := make(map[string]float64)
	for asset, qty := range req.Positions {
		price := 100.0 // Mock price
		val := qty * price
		assetValues[asset] = val
		totalValue += val
	}

	if totalValue == 0 {
		return result, nil
	}

	// 1. Check Concentration (Max Position Weight > 20%)
	for asset, val := range assetValues {
		weight := val / totalValue
		if weight > 0.20 {
			result.Status = "PASSED_WITH_WARNINGS"
			result.NewIssues = append(result.NewIssues, ComplianceIssue{
				RuleID:      "RULE_CONCENTRATION_01",
				Severity:    "WARN",
				Description: fmt.Sprintf("Position %s exceeds 20%% concentration limit (%.1f%%)", asset, weight*100),
				EntityID:    asset,
				Expression:  fmt.Sprintf("weight('%s') <= 0.20", asset),
			})
		}

		// Add Metric
		result.Metrics = append(result.Metrics, SimulationMetric{
			ID:             uuid.NewString(), // Generates a new UUID
			MetricName:     fmt.Sprintf("Weight_%s", asset),
			SimulatedValue: weight,
			Unit:           "Ratio",
		})
	}

	// 2. Check Liquidity (Cash < 5%)
	// Assuming "USD" or "EUR" are cash
	cashVal := assetValues["USD"] + assetValues["EUR"]
	cashWeight := cashVal / totalValue
	if cashWeight < 0.05 {
		result.Status = "FAILED"
		result.NewIssues = append(result.NewIssues, ComplianceIssue{
			RuleID:      "RULE_LIQUIDITY_01",
			Severity:    "CRITICAL",
			Description: fmt.Sprintf("Cash buffer below 5%% minimum (%.1f%%)", cashWeight*100),
			EntityID:    "portfolio",
			Expression:  "liquidityRatio >= 0.05",
		})
	}
	result.Metrics = append(result.Metrics, SimulationMetric{
		ID:             uuid.NewString(),
		MetricName:     "Liquidity_CashRatio",
		SimulatedValue: cashWeight,
		Unit:           "Ratio",
	})

	// 3. ESG Score (Mock)
	esgScore := 75.0 - (rand.Float64() * 10.0) // Random avg
	result.Metrics = append(result.Metrics, SimulationMetric{
		ID:             uuid.NewString(),
		MetricName:     "ESG_Score",
		SimulatedValue: esgScore,
		Unit:           "Score",
	})

	return result, nil
}
