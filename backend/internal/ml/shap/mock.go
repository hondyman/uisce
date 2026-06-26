package shap

import (
	"context"
	"math"
	"math/rand"

	"github.com/hondyman/semlayer/backend/internal/ml"
)

// MockExecutor is a mock PythonExecutor for testing and demo
type MockExecutor struct {
	seed int64
}

// NewMockExecutor creates a new mock executor
func NewMockExecutor(seed int64) *MockExecutor {
	return &MockExecutor{seed: seed}
}

// ComputeSHAP simulates SHAP value computation
func (m *MockExecutor) ComputeSHAP(ctx context.Context, input *ml.PredictionInput, modelPath string) (*ShapComputeResult, error) {
	rng := rand.New(rand.NewSource(m.seed + int64(len(input.ChainID))))

	// Simulate SHAP values based on input features
	shapValues := make(map[string]float64)

	// Health score is negative contributor (lower health = higher risk)
	healthContrib := (1.0 - input.HealthScore) * 0.3
	shapValues["health_score"] = -healthContrib

	// Active conflicts are positive contributor (more conflicts = higher risk)
	conflictContrib := (float64(input.ActiveConflicts) / 10.0) * 0.25
	shapValues["active_conflicts"] = conflictContrib

	// P99 Latency
	latencyContrib := math.Min(input.P99Latency/1000.0, 1.0) * 0.2
	shapValues["p99_latency"] = latencyContrib

	// Error rate
	errorContrib := input.ErrorRate * 0.15
	shapValues["error_rate"] = errorContrib

	// Cross-region latency
	crossRegionContrib := math.Min(input.CrossRegionLatency/2000.0, 1.0) * 0.1
	shapValues["cross_region_latency"] = crossRegionContrib

	// SLA compliance (negative contributor)
	slaContrib := (1.0 - input.SLAComplianceScore) * 0.1
	shapValues["sla_compliance_score"] = -slaContrib

	// Add small noise
	for key := range shapValues {
		shapValues[key] += (rng.Float64() - 0.5) * 0.02
	}

	// Base value represents the average model prediction (~0.2 for failure probability)
	baseValue := 0.2

	// Sum of SHAP values + base value = model output
	modelOutput := baseValue
	for _, val := range shapValues {
		modelOutput += val
	}

	// Clamp to 0-1
	if modelOutput < 0 {
		modelOutput = 0
	} else if modelOutput > 1 {
		modelOutput = 1
	}

	return &ShapComputeResult{
		SHAPValues:  shapValues,
		BaseValue:   baseValue,
		ComputeTime: rng.Float64()*50 + 10, // 10-60ms
		ModelOutput: modelOutput,
	}, nil
}

// ComputeBatchSHAP simulates batch SHAP computation
func (m *MockExecutor) ComputeBatchSHAP(ctx context.Context, inputs []ml.PredictionInput, modelPath string) ([]ShapComputeResult, error) {
	results := make([]ShapComputeResult, len(inputs))

	for i, input := range inputs {
		result, _ := m.ComputeSHAP(ctx, &input, modelPath)
		results[i] = *result
	}

	return results, nil
}

// GetFeatureDistributions returns simulated feature distributions
func (m *MockExecutor) GetFeatureDistributions(ctx context.Context, modelPath string) (*FeatureDistributions, error) {
	return &FeatureDistributions{
		Features: map[string]*ml.FeatureRange{
			"health_score": {
				Min:    0.5,
				Max:    1.0,
				Mean:   0.85,
				StdDev: 0.12,
				Q1:     0.78,
				Median: 0.87,
				Q3:     0.92,
			},
			"active_conflicts": {
				Min:    0.0,
				Max:    50.0,
				Mean:   8.5,
				StdDev: 12.0,
				Q1:     2.0,
				Median: 5.0,
				Q3:     12.0,
			},
			"p99_latency": {
				Min:    50.0,
				Max:    2000.0,
				Mean:   400.0,
				StdDev: 500.0,
				Q1:     200.0,
				Median: 300.0,
				Q3:     600.0,
			},
			"error_rate": {
				Min:    0.0,
				Max:    0.5,
				Mean:   0.02,
				StdDev: 0.08,
				Q1:     0.001,
				Median: 0.005,
				Q3:     0.02,
			},
			"cross_region_latency": {
				Min:    100.0,
				Max:    5000.0,
				Mean:   800.0,
				StdDev: 1200.0,
				Q1:     400.0,
				Median: 700.0,
				Q3:     1500.0,
			},
			"sla_compliance_score": {
				Min:    0.7,
				Max:    1.0,
				Mean:   0.95,
				StdDev: 0.08,
				Q1:     0.9,
				Median: 0.98,
				Q3:     0.995,
			},
		},
	}, nil
}
