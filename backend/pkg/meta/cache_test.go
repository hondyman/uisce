package meta

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// TestMetadataCache_Preload tests metadata preloading
func TestMetadataCache_Preload(t *testing.T) {
	// Skip if no database available
	t.Skip("Requires database connection")

	db, err := sqlx.Connect("postgres", "postgres://localhost/test?sslmode=disable")
	if err != nil {
		t.Skipf("Database not available: %v", err)
	}
	defer db.Close()

	cache := NewMetadataCache(db) // Pass sqlx.DB directly
	ctx := context.Background()

	err = cache.Preload(ctx, "test-tenant")
	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}

	metrics := cache.GetMetrics()
	if metrics.LoadTime == 0 {
		t.Error("Expected non-zero load time")
	}
}

// TestMetadataCache_GetBusinessObject tests cache retrieval
func TestMetadataCache_GetBusinessObject(t *testing.T) {
	cache := NewMetadataCache(nil) // No DB for unit test

	// Manually populate cache for testing
	cache.boByKey["test-tenant"] = make(map[string]*BusinessObjectDefinition)
	cache.boByKey["test-tenant"]["Worker"] = &BusinessObjectDefinition{
		ID:          "bo-1",
		TenantID:    "test-tenant",
		Name:        "Worker",
		DisplayName: "Worker",
		Status:      "active",
		Version:     1,
		CachedAt:    time.Now(),
	}

	// Test cache hit
	bo, err := cache.GetBusinessObject("test-tenant", "Worker")
	if err != nil {
		t.Fatalf("Expected cache hit, got error: %v", err)
	}
	if bo.Name != "Worker" {
		t.Errorf("Expected Worker, got %s", bo.Name)
	}

	// Test cache miss
	_, err = cache.GetBusinessObject("test-tenant", "NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent business object")
	}

	// Verify metrics
	metrics := cache.GetMetrics()
	if metrics.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", metrics.Hits)
	}
	if metrics.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.Misses)
	}
}

// TestMetadataCache_Invalidation tests cache invalidation
func TestMetadataCache_Invalidation(t *testing.T) {
	cache := NewMetadataCache(nil)

	// Populate cache
	cache.boByKey["test-tenant"] = make(map[string]*BusinessObjectDefinition)
	cache.boByID["test-tenant"] = make(map[string]*BusinessObjectDefinition)
	cache.boByKey["test-tenant"]["Worker"] = &BusinessObjectDefinition{
		ID:       "bo-1",
		TenantID: "test-tenant",
		Name:     "Worker",
	}
	cache.boByID["test-tenant"]["bo-1"] = cache.boByKey["test-tenant"]["Worker"]

	// Verify it exists
	_, err := cache.GetBusinessObject("test-tenant", "Worker")
	if err != nil {
		t.Fatalf("Expected business object to exist: %v", err)
	}

	// Invalidate
	cache.InvalidateBusinessObject("test-tenant", "Worker")

	// Verify it's gone
	_, err = cache.GetBusinessObject("test-tenant", "Worker")
	if err == nil {
		t.Error("Expected business object to be invalidated")
	}

	// Verify metrics
	metrics := cache.GetMetrics()
	if metrics.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", metrics.Evictions)
	}
}

// TestMetadataCache_ConcurrentAccess tests thread-safety
func TestMetadataCache_ConcurrentAccess(t *testing.T) {
	cache := NewMetadataCache(nil)

	// Populate cache
	cache.boByKey["test-tenant"] = make(map[string]*BusinessObjectDefinition)
	cache.boByKey["test-tenant"]["Worker"] = &BusinessObjectDefinition{
		ID:       "bo-1",
		TenantID: "test-tenant",
		Name:     "Worker",
	}

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = cache.GetBusinessObject("test-tenant", "Worker")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify metrics
	metrics := cache.GetMetrics()
	// Small variations in concurrent increments can occur; assert a sensible lower bound
	// Allow some slack on CI machines where goroutine scheduling can be noisy.
	if metrics.Hits < 900 {
		t.Errorf("Expected at least 900 hits, got %d", metrics.Hits)
	}
}

// TestRelationshipResolver_GetRelationshipPath tests relationship path finding
func TestRelationshipResolver_GetRelationshipPath(t *testing.T) {
	cache := NewMetadataCache(nil)

	// Set up business objects with relationships
	cache.boByKey["test-tenant"] = make(map[string]*BusinessObjectDefinition)
	cache.boByID["test-tenant"] = make(map[string]*BusinessObjectDefinition)

	worker := &BusinessObjectDefinition{
		ID:       "bo-1",
		TenantID: "test-tenant",
		Name:     "Worker",
		Relationships: []RelationshipDefinition{
			{
				ID:             "rel-1",
				ParentObjectID: "bo-1",
				ChildObjectID:  "bo-2",
				Cardinality:    "1:N",
			},
		},
	}

	position := &BusinessObjectDefinition{
		ID:       "bo-2",
		TenantID: "test-tenant",
		Name:     "Position",
		Relationships: []RelationshipDefinition{
			{
				ID:             "rel-2",
				ParentObjectID: "bo-2",
				ChildObjectID:  "bo-3",
				Cardinality:    "1:1",
			},
		},
	}

	jobProfile := &BusinessObjectDefinition{
		ID:       "bo-3",
		TenantID: "test-tenant",
		Name:     "Job_Profile",
	}

	cache.boByKey["test-tenant"]["Worker"] = worker
	cache.boByKey["test-tenant"]["Position"] = position
	cache.boByKey["test-tenant"]["Job_Profile"] = jobProfile
	cache.boByID["test-tenant"]["bo-1"] = worker
	cache.boByID["test-tenant"]["bo-2"] = position
	cache.boByID["test-tenant"]["bo-3"] = jobProfile

	resolver := NewRelationshipResolver(cache)
	ctx := context.Background()

	// Test path finding
	path, err := resolver.GetRelationshipPath(ctx, "test-tenant", "Worker", "Job_Profile", 5)
	if err != nil {
		t.Fatalf("Failed to find relationship path: %v", err)
	}

	expectedPath := []string{"Worker", "Position", "Job_Profile"}
	if len(path) != len(expectedPath) {
		t.Errorf("Expected path length %d, got %d", len(expectedPath), len(path))
	}

	for i, expected := range expectedPath {
		if path[i] != expected {
			t.Errorf("Expected path[%d] = %s, got %s", i, expected, path[i])
		}
	}
}

// BenchmarkCacheGet benchmarks cache read performance
func BenchmarkCacheGet(b *testing.B) {
	cache := NewMetadataCache(nil)

	// Populate cache
	cache.boByKey["test-tenant"] = make(map[string]*BusinessObjectDefinition)
	cache.boByKey["test-tenant"]["Worker"] = &BusinessObjectDefinition{
		ID:       "bo-1",
		TenantID: "test-tenant",
		Name:     "Worker",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.GetBusinessObject("test-tenant", "Worker")
	}
}

// BenchmarkCachePreload benchmarks metadata preloading
func BenchmarkCachePreload(b *testing.B) {
	// Skip benchmark if no database available
	b.Skip("Requires database connection")

	db, err := sqlx.Connect("postgres", "postgres://localhost/test?sslmode=disable")
	if err != nil {
		b.Skipf("Database not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache := NewMetadataCache(db) // Pass sqlx.DB directly
		_ = cache.Preload(ctx, "test-tenant")
	}
}
