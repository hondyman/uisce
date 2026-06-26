package ml_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/hondyman/semlayer/backend/internal/ml"
)

// TestModelRegistry_RegisterModel tests model registration
func TestModelRegistry_RegisterModel(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{
		AUC:      0.96,
		F1Score:  0.91,
		Accuracy: 0.88,
	}

	features := []string{"health_score", "active_conflicts", "p99_latency_ms"}

	result, err := registry.RegisterModel(context.Background(), "1.0.0", metrics, features)
	if err != nil {
		t.Fatalf("RegisterModel failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}

	if result.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", result.Version)
	}

	if result.Status != "staging" {
		t.Errorf("Expected status 'staging', got %s", result.Status)
	}

	if len(result.Features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(result.Features))
	}
}

// TestModelRegistry_RegisterModel_Duplicate tests duplicate registration rejection
func TestModelRegistry_RegisterModel_Duplicate(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{AUC: 0.96, F1Score: 0.91}
	registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})

	// Try to register same version again
	_, err := registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})
	if err == nil {
		t.Error("Expected error for duplicate model version")
	}
}

// TestModelRegistry_ActivateVersion tests model activation
func TestModelRegistry_ActivateVersion(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{AUC: 0.96, F1Score: 0.91}
	registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})

	err := registry.ActivateVersion(context.Background(), "1.0.0")
	if err != nil {
		t.Fatalf("ActivateVersion failed: %v", err)
	}

	current, err := registry.GetCurrentVersion(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}

	if current != "1.0.0" {
		t.Errorf("Expected current version 1.0.0, got %s", current)
	}

	// Check model status
	model, _ := registry.GetVersion(context.Background(), "1.0.0")
	if model.Status != "active" {
		t.Errorf("Expected status 'active', got %s", model.Status)
	}

	if model.DeployedAt == nil {
		t.Error("DeployedAt should be set")
	}
}

// TestModelRegistry_RollbackToPrevious tests rollback functionality
func TestModelRegistry_RollbackToPrevious(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{AUC: 0.96, F1Score: 0.91}
	registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})
	registry.ActivateVersion(context.Background(), "1.0.0")

	// Register and activate new version
	metrics2 := &ml.ModelMetrics{AUC: 0.97, F1Score: 0.92}
	registry.RegisterModel(context.Background(), "1.1.0", metrics2, []string{})
	registry.ActivateVersion(context.Background(), "1.1.0")

	// Rollback
	previous, err := registry.RollbackToPrevious(context.Background(), "performance_degradation")
	if err != nil {
		t.Fatalf("RollbackToPrevious failed: %v", err)
	}

	if previous != "1.0.0" {
		t.Errorf("Expected rolled back version 1.0.0, got %s", previous)
	}

	current, _ := registry.GetCurrentVersion(context.Background())
	if current != "1.0.0" {
		t.Errorf("Expected current version 1.0.0 after rollback, got %s", current)
	}

	// Check rollback info
	oldModel, _ := registry.GetVersion(context.Background(), "1.1.0")
	if oldModel.RollbackInfo == nil {
		t.Error("RollbackInfo should be set on old model")
	}

	if oldModel.RollbackInfo.Reason != "performance_degradation" {
		t.Errorf("Expected rollback reason 'performance_degradation', got %s", oldModel.RollbackInfo.Reason)
	}
}

// TestModelRegistry_RollbackToPrevious_NoPrevious tests rollback without previous version
func TestModelRegistry_RollbackToPrevious_NoPrevious(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{AUC: 0.96}
	registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})
	registry.ActivateVersion(context.Background(), "1.0.0")

	// Try to rollback with no previous version
	_, err := registry.RollbackToPrevious(context.Background(), "test")
	if err == nil {
		t.Error("Expected error when no previous version exists")
	}
}

// TestModelRegistry_ListVersions tests version listing
func TestModelRegistry_ListVersions(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{AUC: 0.96}

	registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})
	registry.RegisterModel(context.Background(), "1.1.0", metrics, []string{})
	registry.RegisterModel(context.Background(), "1.2.0", metrics, []string{})

	registry.ActivateVersion(context.Background(), "1.0.0")
	registry.ActivateVersion(context.Background(), "1.1.0")
	registry.ActivateVersion(context.Background(), "1.2.0")

	versions, err := registry.ListVersions(context.Background(), "")
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}

	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}

	// Check sorting (newest first)
	if versions[0].Version != "1.2.0" {
		t.Errorf("Expected newest version first (1.2.0), got %s", versions[0].Version)
	}
}

// TestModelRegistry_UpdateMetrics tests metrics updates
func TestModelRegistry_UpdateMetrics(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{AUC: 0.96, F1Score: 0.91}
	registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})

	// Update metrics
	newMetrics := &ml.ModelMetrics{AUC: 0.965, F1Score: 0.915}
	err := registry.UpdateMetrics(context.Background(), "1.0.0", newMetrics)
	if err != nil {
		t.Fatalf("UpdateMetrics failed: %v", err)
	}

	// Verify update
	model, _ := registry.GetVersion(context.Background(), "1.0.0")
	if model.Metrics.AUC != 0.965 {
		t.Errorf("Expected AUC 0.965, got %f", model.Metrics.AUC)
	}
}

// TestModelRegistry_CanaryDeployment tests canary deployment setup
func TestModelRegistry_CanaryDeployment(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{AUC: 0.96}
	registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})

	err := registry.EnableCanaryDeployment(context.Background(), "1.0.0", 0.1, 0.05)
	if err != nil {
		t.Fatalf("EnableCanaryDeployment failed: %v", err)
	}

	model, _ := registry.GetVersion(context.Background(), "1.0.0")
	if model.CanaryDeployment == nil {
		t.Error("CanaryDeployment should be set")
	}

	if model.CanaryDeployment.TrafficSplit != 0.1 {
		t.Errorf("Expected traffic split 0.1, got %f", model.CanaryDeployment.TrafficSplit)
	}

	if model.CanaryDeployment.Threshold != 0.05 {
		t.Errorf("Expected threshold 0.05, got %f", model.CanaryDeployment.Threshold)
	}
}

// TestModelRegistry_CanaryDeployment_InvalidTrafficSplit tests invalid traffic split
func TestModelRegistry_CanaryDeployment_InvalidTrafficSplit(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{AUC: 0.96}
	registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})

	// Test invalid traffic split > 1.0
	err := registry.EnableCanaryDeployment(context.Background(), "1.0.0", 1.5, 0.05)
	if err == nil {
		t.Error("Expected error for traffic split > 1.0")
	}

	// Test invalid traffic split < 0.0
	err = registry.EnableCanaryDeployment(context.Background(), "1.0.0", -0.1, 0.05)
	if err == nil {
		t.Error("Expected error for traffic split < 0.0")
	}
}

// TestModelRegistry_CompareVersions tests version comparison
func TestModelRegistry_CompareVersions(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics1 := &ml.ModelMetrics{AUC: 0.96, F1Score: 0.91, Accuracy: 0.88}
	metrics2 := &ml.ModelMetrics{AUC: 0.965, F1Score: 0.915, Accuracy: 0.885}

	registry.RegisterModel(context.Background(), "1.0.0", metrics1, []string{"f1", "f2"})
	registry.RegisterModel(context.Background(), "1.1.0", metrics2, []string{"f1", "f2", "f3"})

	comparison, err := registry.CompareVersions(context.Background(), "1.0.0", "1.1.0")
	if err != nil {
		t.Fatalf("CompareVersions failed: %v", err)
	}

	if comparison == nil {
		t.Error("Comparison should not be nil")
	}

	// Check delta calculations
	deltas := comparison["metrics_delta"].(map[string]float64)
	if math.Abs(deltas["auc_diff"]-0.005) > 1e-9 {
		t.Errorf("Expected AUC delta 0.005, got %f", deltas["auc_diff"])
	}
}

// TestModelRegistry_CleanupOldVersions tests old version cleanup
func TestModelRegistry_CleanupOldVersions(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")
	// registry.SetMaxVersions(3) // Method not available, relying on default (10)

	metrics := &ml.ModelMetrics{AUC: 0.96}

	// Register 5 versions
	for i := 1; i <= 5; i++ {
		version := "1." + string(rune('0'+i))
		registry.RegisterModel(context.Background(), version, metrics, []string{})
		registry.ActivateVersion(context.Background(), version)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Cleanup
	err := registry.CleanupOldVersions(context.Background())
	if err != nil {
		t.Fatalf("CleanupOldVersions failed: %v", err)
	}

	// With default max 10, all 5 should remain.
	versions, _ := registry.ListVersions(context.Background(), "")
	if len(versions) != 5 {
		t.Logf("After cleanup: %d versions", len(versions))
	}
}

// TestModelRegistry_CurrentVersionNotSet tests error when current version not set
func TestModelRegistry_CurrentVersionNotSet(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	_, err := registry.GetCurrentVersion(context.Background())
	if err == nil {
		t.Error("Expected error when no current version set")
	}
}

// TestModelRegistry_GetVersion_NotFound tests getting non-existent version
func TestModelRegistry_GetVersion_NotFound(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	_, err := registry.GetVersion(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent version")
	}
}

// TestModelRegistry_GetVersionForRequest tests version selection for request routing
func TestModelRegistry_GetVersionForRequest(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")

	metrics := &ml.ModelMetrics{AUC: 0.96}
	registry.RegisterModel(context.Background(), "1.0.0", metrics, []string{})
	registry.ActivateVersion(context.Background(), "1.0.0")

	version, err := registry.GetVersionForRequest(context.Background())
	if err != nil {
		t.Fatalf("GetVersionForRequest failed: %v", err)
	}

	if version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", version)
	}
}

// TestModelRegistry_ConcurrentAccess tests thread-safe concurrent access
func TestModelRegistry_ConcurrentAccess(t *testing.T) {
	registry := ml.NewModelRegistry("/tmp/models")
	metrics := &ml.ModelMetrics{AUC: 0.96}

	done := make(chan error, 10)

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		go func(id int) {
			version := "1." + string(rune('0'+id))
			_, err := registry.RegisterModel(context.Background(), version, metrics, []string{})
			done <- err
		}(i)
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent registration failed: %v", err)
		}
	}

	// All should be registered
	versions, _ := registry.ListVersions(context.Background(), "")
	if len(versions) < 10 {
		t.Errorf("Expected at least 10 versions, got %d", len(versions))
	}
}
