package featurestore

import (
	"context"
	"testing"
	"time"
)

func TestFeatureStore_RegisterFeature(t *testing.T) {
	fs := NewFeatureStore(1*time.Hour, 1000)

	feature := &FeatureDefinition{
		Name:        "health_score",
		Category:    "numerical",
		Description: "Chain health score",
		DataType:    "float64",
		IsActive:    true,
	}

	err := fs.RegisterFeature(context.Background(), feature)
	if err != nil {
		t.Fatalf("RegisterFeature failed: %v", err)
	}

	retrieved, _ := fs.GetFeatureDefinition(context.Background(), "health_score")
	if retrieved == nil {
		t.Error("Feature not retrieved")
	}

	if retrieved.Name != "health_score" {
		t.Errorf("Expected name health_score, got %s", retrieved.Name)
	}
}

func TestFeatureStore_ComputeFeatures(t *testing.T) {
	fs := NewFeatureStore(1*time.Hour, 1000)

	feature := &FeatureDefinition{Name: "health_score", IsActive: true}
	fs.RegisterFeature(context.Background(), feature)

	request := &FeatureRequest{
		EntityID:     "chain-1",
		EntityType:   "chain",
		FeatureNames: []string{"health_score"},
	}

	batch, err := fs.ComputeFeatures(context.Background(), request)
	if err != nil {
		t.Fatalf("ComputeFeatures failed: %v", err)
	}

	if len(batch.Features) == 0 {
		t.Error("No features computed")
	}

	if batch.EntityID != "chain-1" {
		t.Errorf("Expected entity_id chain-1, got %s", batch.EntityID)
	}
}

func TestFeatureStore_GetFeatureSnapshot(t *testing.T) {
	fs := NewFeatureStore(1*time.Hour, 1000)

	timestamp := time.Now().Add(-24 * time.Hour)
	snapshot, err := fs.GetFeatureSnapshot(context.Background(), "chain-1", timestamp)
	if err != nil {
		t.Fatalf("GetFeatureSnapshot failed: %v", err)
	}

	if snapshot.EntityID != "chain-1" {
		t.Errorf("Expected entity_id chain-1, got %s", snapshot.EntityID)
	}

	if len(snapshot.Features) == 0 {
		t.Error("Snapshot should have features")
	}
}

func TestFeatureStore_CacheInvalidation(t *testing.T) {
	fs := NewFeatureStore(1*time.Hour, 1000)

	err := fs.InvalidateCache(context.Background(), "chain-1")
	if err != nil {
		t.Fatalf("InvalidateCache failed: %v", err)
	}

	err = fs.ClearCache(context.Background())
	if err != nil {
		t.Fatalf("ClearCache failed: %v", err)
	}
}
