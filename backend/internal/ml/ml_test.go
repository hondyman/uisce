package ml_test

import (
	"context"
	"testing"
	"time"

	"github.com/hondyman/semlayer/backend/internal/ml"
	"github.com/hondyman/semlayer/backend/internal/ml/shap"
)

func TestPredictionInput_Validation(t *testing.T) {
	input := &ml.PredictionInput{
		ChainID:            "test-chain",
		Region:             "us-east-1",
		TenantID:           "test-tenant",
		HealthScore:        0.85,
		ActiveConflicts:    3,
		P99Latency:         450,
		SLAComplianceScore: 0.95,
		ErrorRate:          0.01,
	}

	if input.ChainID == "" {
		t.Errorf("Expected ChainID to be set")
	}
	if input.HealthScore < 0 || input.HealthScore > 1.0 {
		t.Errorf("HealthScore should be between 0 and 1")
	}
}

func TestExplainability_SHAP(t *testing.T) {
	ctx := context.Background()
	executor := shap.NewMockExecutor(42)
	engine := shap.NewEngine(executor, &shap.EngineConfig{
		MaxInteractions: 5,
		SHAPType:        "kernel",
	})

	inputs := []ml.PredictionInput{
		{ChainID: "chain-1", TenantID: "tenant-1", Region: "us-east", HealthScore: 0.9, ActiveConflicts: 1, P99Latency: 300, ErrorRate: 0.01},
		{ChainID: "chain-2", TenantID: "tenant-1", Region: "eu-west", HealthScore: 0.8, ActiveConflicts: 2, P99Latency: 800, ErrorRate: 0.02},
	}

	explanations, err := engine.ExplainBatch(ctx, inputs, "model.pkl")
	if err != nil {
		t.Fatalf("ExplainBatch failed: %v", err)
	}

	if len(explanations) != len(inputs) {
		t.Fatalf("Expected %d explanations, got %d", len(inputs), len(explanations))
	}

	for _, in := range inputs {
		exp, ok := explanations[in.ChainID]
		if !ok || exp == nil {
			t.Fatalf("Missing or nil explanation for %s", in.ChainID)
		}
	}

	// clear cache should not panic
	engine.ClearCache()

	// ensure Explain works
	// Create a dummy prediction for the Explain call
	prediction := &ml.Prediction{
		ChainID:            "chain-1",
		Region:             "us-east",
		TenantID:           "tenant-1",
		FailureProbability: 0.1,
		Confidence:         0.9,
		PredictedAt:        time.Now(),
	}

	exp, err := engine.Explain(ctx, &inputs[0], "model.pkl", prediction)
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if exp == nil {
		t.Fatalf("Explain returned nil")
	}
}
