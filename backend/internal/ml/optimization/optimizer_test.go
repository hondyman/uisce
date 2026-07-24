package optimization

import (
	"context"
	"testing"
	"time"
)

func TestPerformanceOptimizer_CacheOperations(t *testing.T) {
	po := NewPerformanceOptimizer(1000, 1*time.Hour)

	err := po.Set(context.Background(), "key1", "value1", 1*time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, exists := po.Get(context.Background(), "key1")
	if !exists {
		t.Error("Value should be cached")
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
}

func TestPerformanceOptimizer_CachePrediction(t *testing.T) {
	po := NewPerformanceOptimizer(1000, 1*time.Hour)

	pred := &CachedPrediction{
		PredictionID:     "pred-1",
		InputHash:        "hash1",
		PredictionOutput: 0.75,
		SHAPValues:       map[string]float64{"f1": 0.1},
	}

	err := po.CachePrediction(context.Background(), "hash1", pred)
	if err != nil {
		t.Fatalf("CachePrediction failed: %v", err)
	}

	cached, exists := po.GetCachedPrediction(context.Background(), "hash1")
	if !exists {
		t.Error("Prediction should be cached")
	}

	if cached.PredictionOutput != 0.75 {
		t.Errorf("Expected 0.75, got %f", cached.PredictionOutput)
	}

	if cached.AccessCount != 1 {
		t.Errorf("Expected access count 1, got %d", cached.AccessCount)
	}
}

func TestPerformanceOptimizer_CacheEviction(t *testing.T) {
	po := NewPerformanceOptimizer(2, 1*time.Hour) // Small cache

	po.Set(context.Background(), "key1", "value1", 1*time.Hour)
	po.Set(context.Background(), "key2", "value2", 1*time.Hour)
	po.Set(context.Background(), "key3", "value3", 1*time.Hour) // Should evict key1

	_, exists := po.Get(context.Background(), "key1")
	if exists {
		t.Error("Key1 should be evicted")
	}

	_, exists = po.Get(context.Background(), "key3")
	if !exists {
		t.Error("Key3 should be in cache")
	}
}

func TestPerformanceOptimizer_GetMetrics(t *testing.T) {
	po := NewPerformanceOptimizer(1000, 1*time.Hour)

	metrics := po.GetMetrics(context.Background())
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}

	if metrics.CacheHitRate < 0 || metrics.CacheHitRate > 1 {
		t.Errorf("Invalid cache hit rate: %f", metrics.CacheHitRate)
	}

	if metrics.MemoryUsageMB < 0 {
		t.Errorf("Invalid memory usage: %f", metrics.MemoryUsageMB)
	}
}

func TestPerformanceOptimizer_ClearCache(t *testing.T) {
	po := NewPerformanceOptimizer(1000, 1*time.Hour)

	po.Set(context.Background(), "key1", "value1", 1*time.Hour)
	po.Set(context.Background(), "key2", "value2", 1*time.Hour)

	err := po.ClearCache(context.Background())
	if err != nil {
		t.Fatalf("ClearCache failed: %v", err)
	}

	stats := po.GetCacheStats(context.Background())
	entries := stats["cache_entries"].(int)
	if entries != 0 {
		t.Errorf("Expected 0 cache entries, got %d", entries)
	}
}

func TestPerformanceOptimizer_CacheExpiry(t *testing.T) {
	po := NewPerformanceOptimizer(1000, 100*time.Millisecond)

	po.Set(context.Background(), "key1", "value1", 100*time.Millisecond)
	time.Sleep(150 * time.Millisecond)

	value, exists := po.Get(context.Background(), "key1")
	if exists && value != nil {
		t.Error("Expired value should not be returned")
	}
}
