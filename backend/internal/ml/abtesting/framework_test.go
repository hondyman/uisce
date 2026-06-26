package abtesting

import (
	"context"
	"math"
	"testing"
)

func TestExperimentFramework_CreateExperiment(t *testing.T) {
	ef := NewExperimentFramework()

	exp := &Experiment{
		Name:         "model_v2_test",
		Type:         "model",
		TrafficSplit: 0.5,
		Control:      VariantConfig{Name: "v1.0", ModelVersion: "1.0.0"},
		Treatment:    VariantConfig{Name: "v1.1", ModelVersion: "1.1.0"},
		PrimaryMetric: PrimaryMetric{
			Name:                "auc",
			Direction:           "higher",
			MinDetectableEffect: 0.01,
			BaselineValue:       0.96,
		},
	}

	id, err := ef.CreateExperiment(context.Background(), exp)
	if err != nil {
		t.Fatalf("CreateExperiment failed: %v", err)
	}

	if id == "" {
		t.Error("Experiment ID should not be empty")
	}
}

func TestExperimentFramework_AssignVariant(t *testing.T) {
	ef := NewExperimentFramework()

	exp := &Experiment{
		Name:         "test",
		Type:         "model",
		TrafficSplit: 0.5,
		Control:      VariantConfig{Name: "control"},
		Treatment:    VariantConfig{Name: "treatment"},
		Status:       "running",
	}

	expID, _ := ef.CreateExperiment(context.Background(), exp)
	ef.experiments[expID].Status = "running"

	assignment, err := ef.AssignVariant(context.Background(), expID, "entity-1", map[string]interface{}{})
	if err != nil {
		t.Fatalf("AssignVariant failed: %v", err)
	}

	if assignment.Variant == "" {
		t.Error("Variant should be assigned")
	}
}

func TestExperimentFramework_DeterministicAssignment(t *testing.T) {
	ef := NewExperimentFramework()

	exp := &Experiment{
		ID:           "exp-1",
		TrafficSplit: 0.5,
		Status:       "running",
	}
	ef.experiments["exp-1"] = exp

	// Same entity should get same variant
	a1, _ := ef.AssignVariant(context.Background(), "exp-1", "entity-1", map[string]interface{}{})
	a2, _ := ef.AssignVariant(context.Background(), "exp-1", "entity-1", map[string]interface{}{})

	if a1.Variant != a2.Variant {
		t.Error("Same entity should get same variant")
	}
}

func TestExperimentFramework_RecordEvent(t *testing.T) {
	ef := NewExperimentFramework()

	ef.experiments["exp-1"] = &Experiment{Status: "running"}
	ef.metrics["exp-1"] = &ExperimentMetrics{
		ExperimentID:     "exp-1",
		ControlMetrics:   make(map[string]float64),
		TreatmentMetrics: make(map[string]float64),
	}

	log := &EventLog{
		ExperimentID: "exp-1",
		Variant:      "control",
		MetricValues: map[string]float64{"auc": 0.96},
	}

	err := ef.RecordEvent(context.Background(), log)
	if err != nil {
		t.Fatalf("RecordEvent failed: %v", err)
	}
}

func TestExperimentFramework_GetResults(t *testing.T) {
	ef := NewExperimentFramework()

	ef.experiments["exp-1"] = &Experiment{
		PrimaryMetric: PrimaryMetric{Name: "auc"},
	}
	ef.metrics["exp-1"] = &ExperimentMetrics{
		ExperimentID:     "exp-1",
		ControlMetrics:   map[string]float64{"auc": 0.960},
		TreatmentMetrics: map[string]float64{"auc": 0.965},
	}

	results, err := ef.GetExperimentResults(context.Background(), "exp-1")
	if err != nil {
		t.Fatalf("GetExperimentResults failed: %v", err)
	}

	delta := results.TreatmentMetrics["auc"] - results.ControlMetrics["auc"]
	if math.Abs(delta-0.005) > 1e-9 {
		t.Errorf("Expected delta 0.005, got %f", delta)
	}
}
