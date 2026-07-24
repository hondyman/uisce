package fairness

import (
	"context"
	"testing"
	"time"
)

func TestFairnessAnalyzer_RegisterProtectedAttribute(t *testing.T) {
	fa := NewFairnessAnalyzer()

	attr := &ProtectedAttribute{
		Name:   "region",
		Values: []string{"us-east-1", "us-west-2", "eu-central-1"},
	}

	err := fa.RegisterProtectedAttribute(context.Background(), attr)
	if err != nil {
		t.Fatalf("RegisterProtectedAttribute failed: %v", err)
	}
}

func TestFairnessAnalyzer_AnalyzeFairness(t *testing.T) {
	fa := NewFairnessAnalyzer()

	attr := &ProtectedAttribute{
		Name:             "region",
		Values:           []string{"us-east-1", "us-west-2"},
		AllowedDisparity: 0.10,
	}
	fa.RegisterProtectedAttribute(context.Background(), attr)

	predictions := []*PredictionAudit{
		{
			ChainID:             "chain-1",
			PredictionOutput:    0.8,
			ProtectedAttributes: map[string]string{"region": "us-east-1"},
		},
		{
			ChainID:             "chain-2",
			PredictionOutput:    0.6,
			ProtectedAttributes: map[string]string{"region": "us-west-2"},
		},
	}

	report, err := fa.AnalyzeFairness(context.Background(), predictions)
	if err != nil {
		t.Fatalf("AnalyzeFairness failed: %v", err)
	}

	if report == nil {
		t.Error("Report should not be nil")
	}

	if report.SampleSize != 2 {
		t.Errorf("Expected sample size 2, got %d", report.SampleSize)
	}
}

func TestFairnessAnalyzer_CreateAuditLog(t *testing.T) {
	fa := NewFairnessAnalyzer()

	audit := &PredictionAudit{
		ChainID:             "chain-1",
		ModelVersion:        "1.0.0",
		PredictionOutput:    0.75,
		RiskLevel:           "medium",
		Timestamp:           time.Now(),
		ProtectedAttributes: map[string]string{"region": "us-east-1"},
	}

	err := fa.CreatePredictionAudit(context.Background(), audit)
	if err != nil {
		t.Fatalf("CreatePredictionAudit failed: %v", err)
	}

	if audit.PredictionID == "" {
		t.Error("PredictionID should be set")
	}

	if audit.Hash == "" {
		t.Error("Hash should be set")
	}
}

func TestFairnessAnalyzer_VerifyIntegrity(t *testing.T) {
	fa := NewFairnessAnalyzer()

	audit := &PredictionAudit{
		ChainID:          "chain-1",
		ModelVersion:     "1.0.0",
		PredictionOutput: 0.75,
	}

	fa.CreatePredictionAudit(context.Background(), audit)

	isValid := fa.VerifyAuditIntegrity(context.Background(), audit)
	if !isValid {
		t.Error("Audit should be valid")
	}
}

func TestFairnessAnalyzer_CompareFairnessAcrossVersions(t *testing.T) {
	fa := NewFairnessAnalyzer()

	v1Preds := []*PredictionAudit{
		{ChainID: "c1", PredictionOutput: 0.8, ProtectedAttributes: map[string]string{"region": "us-east-1"}},
		{ChainID: "c2", PredictionOutput: 0.5, ProtectedAttributes: map[string]string{"region": "us-west-2"}},
	}

	v2Preds := []*PredictionAudit{
		{ChainID: "c1", PredictionOutput: 0.75, ProtectedAttributes: map[string]string{"region": "us-east-1"}},
		{ChainID: "c2", PredictionOutput: 0.72, ProtectedAttributes: map[string]string{"region": "us-west-2"}},
	}

	comparison, err := fa.CompareFairnessAcrossVersions(context.Background(), v1Preds, v2Preds)
	if err != nil {
		t.Fatalf("CompareFairnessAcrossVersions failed: %v", err)
	}

	if comparison == nil {
		t.Error("Comparison should not be nil")
	}
}
