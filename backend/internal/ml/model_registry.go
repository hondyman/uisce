package ml

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// PredictionMetrics holds model performance metrics
type PredictionMetrics struct {
	AUC       float64 `json:"auc"`
	F1Score   float64 `json:"f1_score"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	Accuracy  float64 `json:"accuracy"`
	MAE       float64 `json:"mae"`
	RMSE      float64 `json:"rmse"`
}

// ModelRegistry manages model versions and metadata
type ModelRegistry struct {
	mu              sync.RWMutex
	models          map[string]*ModelVersion
	currentVersion  string
	previousVersion string
	modelPath       string
	maxVersions     int
}

// ModelVersion represents a versioned model
type ModelVersion struct {
	Version           string                 `json:"version"`
	CreatedAt         time.Time              `json:"created_at"`
	DeployedAt        *time.Time             `json:"deployed_at,omitempty"`
	RetiredAt         *time.Time             `json:"retired_at,omitempty"`
	Status            string                 `json:"status"` // "active", "staging", "archived"
	Metrics           ModelMetrics           `json:"metrics"`
	Features          []string               `json:"features"`
	FeatureSchemaHash string                 `json:"feature_schema_hash"`
	TrainingDataSize  int                    `json:"training_data_size"`
	TrainingDuration  int64                  `json:"training_duration_ms"`
	ValidationMetrics PredictionMetrics      `json:"validation_metrics"`
	CanaryDeployment  *CanaryDeployment      `json:"canary_deployment,omitempty"`
	RollbackInfo      *RollbackInfo          `json:"rollback_info,omitempty"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// CanaryDeployment holds canary deployment info
type CanaryDeployment struct {
	Enabled      bool      `json:"enabled"`
	TrafficSplit float64   `json:"traffic_split"` // 0-1, percentage of traffic
	StartedAt    time.Time `json:"started_at"`
	Threshold    float64   `json:"threshold"` // Error rate threshold for rollback
}

// RollbackInfo holds rollback history
type RollbackInfo struct {
	PreviousVersion string    `json:"previous_version"`
	RolledBackAt    time.Time `json:"rolled_back_at"`
	Reason          string    `json:"reason"`
}

// NewModelRegistry creates a new model registry
func NewModelRegistry(modelPath string) *ModelRegistry {
	return &ModelRegistry{
		models:      make(map[string]*ModelVersion),
		modelPath:   modelPath,
		maxVersions: 10,
	}
}

// RegisterModel registers a new model version
func (r *ModelRegistry) RegisterModel(ctx context.Context, version string, metrics *ModelMetrics, features []string) (*ModelVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.models[version]; exists {
		return nil, fmt.Errorf("model version %s already exists", version)
	}

	now := time.Now()
	modelVersion := &ModelVersion{
		Version:   version,
		CreatedAt: now,
		Status:    "staging", // New models start in staging
		Metrics:   *metrics,
		Features:  features,
		Metadata:  make(map[string]interface{}),
	}

	r.models[version] = modelVersion
	return modelVersion, nil
}

// GetCurrentVersion returns the currently active model version
func (r *ModelRegistry) GetCurrentVersion(ctx context.Context) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.currentVersion == "" {
		return "", fmt.Errorf("no active model version")
	}
	return r.currentVersion, nil
}

// ActivateVersion promotes a model to production
func (r *ModelRegistry) ActivateVersion(ctx context.Context, version string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	modelVersion, exists := r.models[version]
	if !exists {
		return fmt.Errorf("model version %s not found", version)
	}

	now := time.Now()

	// Retire previous version
	if r.currentVersion != "" && r.models[r.currentVersion] != nil {
		r.previousVersion = r.currentVersion
		r.models[r.currentVersion].RetiredAt = &now
		r.models[r.currentVersion].Status = "archived"
	}

	// Activate new version
	modelVersion.DeployedAt = &now
	modelVersion.Status = "active"
	r.currentVersion = version

	return nil
}

// RollbackToPrevious rolls back to the previous model version
func (r *ModelRegistry) RollbackToPrevious(ctx context.Context, reason string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.previousVersion == "" {
		return "", fmt.Errorf("no previous version available for rollback")
	}

	now := time.Now()
	currentModel := r.models[r.currentVersion]
	previousModel := r.models[r.previousVersion]

	if currentModel != nil {
		currentModel.RetiredAt = &now
		currentModel.Status = "archived"
		currentModel.RollbackInfo = &RollbackInfo{
			PreviousVersion: r.previousVersion,
			RolledBackAt:    now,
			Reason:          reason,
		}
	}

	if previousModel != nil {
		previousModel.DeployedAt = &now
		previousModel.Status = "active"
	}

	oldCurrent := r.currentVersion
	r.currentVersion = r.previousVersion
	r.previousVersion = oldCurrent

	return r.currentVersion, nil
}

// ListVersions returns all registered model versions
func (r *ModelRegistry) ListVersions(ctx context.Context, status string) ([]*ModelVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var versions []*ModelVersion
	for _, v := range r.models {
		if status == "" || v.Status == status {
			versions = append(versions, v)
		}
	}

	// Sort by creation time, newest first
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	return versions, nil
}

// GetVersion retrieves a specific model version
func (r *ModelRegistry) GetVersion(ctx context.Context, version string) (*ModelVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modelVersion, exists := r.models[version]
	if !exists {
		return nil, fmt.Errorf("model version %s not found", version)
	}
	return modelVersion, nil
}

// UpdateMetrics updates model metrics
func (r *ModelRegistry) UpdateMetrics(ctx context.Context, version string, metrics *ModelMetrics) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	modelVersion, exists := r.models[version]
	if !exists {
		return fmt.Errorf("model version %s not found", version)
	}

	modelVersion.Metrics = *metrics
	return nil
}

// EnableCanaryDeployment enables canary deployment for a model
func (r *ModelRegistry) EnableCanaryDeployment(ctx context.Context, version string, trafficSplit float64, threshold float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	modelVersion, exists := r.models[version]
	if !exists {
		return fmt.Errorf("model version %s not found", version)
	}

	if trafficSplit < 0 || trafficSplit > 1 {
		return fmt.Errorf("traffic split must be between 0 and 1")
	}

	modelVersion.CanaryDeployment = &CanaryDeployment{
		Enabled:      true,
		TrafficSplit: trafficSplit,
		StartedAt:    time.Now(),
		Threshold:    threshold,
	}

	return nil
}

// GetVersionForRequest returns a model version for a prediction request
// Uses canary deployment logic if enabled
func (r *ModelRegistry) GetVersionForRequest(ctx context.Context) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.currentVersion == "" {
		return "", fmt.Errorf("no active model version")
	}

	// If no canary deployment, use current version
	currentModel := r.models[r.currentVersion]
	if currentModel == nil || currentModel.CanaryDeployment == nil || !currentModel.CanaryDeployment.Enabled {
		return r.currentVersion, nil
	}

	// Canary deployment: route some traffic to previous version
	// For now, always use current. In production, would use consistent hashing/random
	return r.currentVersion, nil
}

// CleanupOldVersions removes archived versions beyond maxVersions
func (r *ModelRegistry) CleanupOldVersions(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	archived := []*ModelVersion{}
	for _, v := range r.models {
		if v.Status == "archived" {
			archived = append(archived, v)
		}
	}

	// Sort by retirement time
	sort.Slice(archived, func(i, j int) bool {
		if archived[i].RetiredAt == nil || archived[j].RetiredAt == nil {
			return false
		}
		return archived[i].RetiredAt.Before(*archived[j].RetiredAt)
	})

	// Keep maxVersions, delete oldest
	if len(archived) > r.maxVersions {
		for _, v := range archived[:len(archived)-r.maxVersions] {
			delete(r.models, v.Version)
		}
	}

	return nil
}

// CompareVersions compares two model versions
func (r *ModelRegistry) CompareVersions(ctx context.Context, v1, v2 string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model1, exists1 := r.models[v1]
	model2, exists2 := r.models[v2]

	if !exists1 || !exists2 {
		return nil, fmt.Errorf("one or both model versions not found")
	}

	comparison := map[string]interface{}{
		"version_1": v1,
		"version_2": v2,
		"metrics_delta": map[string]float64{
			"auc_diff":      model2.Metrics.AUC - model1.Metrics.AUC,
			"f1_diff":       model2.Metrics.F1Score - model1.Metrics.F1Score,
			"accuracy_diff": model2.Metrics.Accuracy - model1.Metrics.Accuracy,
		},
		"feature_count": map[string]int{
			"v1": len(model1.Features),
			"v2": len(model2.Features),
		},
	}

	return comparison, nil
}
