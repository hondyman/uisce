package xgboost

import (
	"context"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/ml"
)

func TestXGBoostModel_Load(t *testing.T) {
	model := NewXGBoostModel("/tmp/test_model.bin", "1.0.0")
	if model.isReady {
		t.Error("Model should not be ready before loading")
	}

	if err := model.Load(context.Background()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !model.isReady {
		t.Error("Model should be ready after loading")
	}
}

func TestXGBoostModel_NormalizeFeatures(t *testing.T) {
	model := NewXGBoostModel("/tmp/test_model.bin", "1.0.0")
	_ = model.Load(context.Background())

	input := &ml.PredictionInput{HealthScore: 0.5}
	features := model.normalizeFeatures(input)
	if v, ok := features["health_score"]; !ok {
		t.Error("health_score missing from normalized features")
	} else if v < 0 || v > 1 {
		t.Errorf("normalized health_score out of [0,1]: %f", v)
	}
}

func TestXGBoostModel_Predict(t *testing.T) {
	model := NewXGBoostModel("/tmp/test_model.bin", "1.0.0")
	if err := model.Load(context.Background()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	input := &ml.PredictionInput{ChainID: "test-chain", Region: "us-east-1", HealthScore: 0.6}
	pred, err := model.Predict(context.Background(), input)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	if pred == nil {
		t.Fatal("Prediction is nil")
	}
	if pred.FailureProbability < 0 || pred.FailureProbability > 1 {
		t.Errorf("FailureProbability out of range: %f", pred.FailureProbability)
	}
	valid := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if !valid[pred.RiskLevel] {
		t.Errorf("Invalid risk level: %s", pred.RiskLevel)
	}
}

func TestXGBoostModel_GetModelMetrics(t *testing.T) {
	model := NewXGBoostModel("/tmp/test_model.bin", "1.0.0")
	if err := model.Load(context.Background()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	metrics, err := model.GetModelMetrics(context.Background())
	if err != nil {
		t.Fatalf("GetModelMetrics failed: %v", err)
	}
	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}
	if metrics.ModelVersion != model.version {
		t.Errorf("Expected version %s, got %s", model.version, metrics.ModelVersion)
	}
	if len(metrics.FeatureImportances) == 0 {
		t.Error("Expected non-empty FeatureImportances")
	}
}
