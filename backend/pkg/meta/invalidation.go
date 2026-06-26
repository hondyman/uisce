package meta

import (
	"context"
	"fmt"
	"log"
)

// InvalidationService handles cache invalidation across the system
// This ensures cache consistency when metadata changes
type InvalidationService struct {
	cache *MetadataCache
}

// NewInvalidationService creates a new invalidation service
func NewInvalidationService(cache *MetadataCache) *InvalidationService {
	return &InvalidationService{
		cache: cache,
	}
}

// InvalidateOnChange invalidates cache when a business object changes
func (s *InvalidationService) InvalidateOnChange(
	ctx context.Context,
	tenantID, boKey string,
) error {
	log.Printf("[InvalidationService] Invalidating cache for tenant=%s, bo=%s", tenantID, boKey)
	s.cache.InvalidateBusinessObject(tenantID, boKey)
	return nil
}

// InvalidateOnCreate invalidates cache when a new business object is created
func (s *InvalidationService) InvalidateOnCreate(
	ctx context.Context,
	tenantID string,
) error {
	log.Printf("[InvalidationService] Invalidating entire cache for tenant=%s (new BO created)", tenantID)
	s.cache.InvalidateTenant(tenantID)
	return nil
}

// InvalidateOnDelete invalidates cache when a business object is deleted
func (s *InvalidationService) InvalidateOnDelete(
	ctx context.Context,
	tenantID, boKey string,
) error {
	log.Printf("[InvalidationService] Invalidating cache for tenant=%s, bo=%s (deleted)", tenantID, boKey)
	s.cache.InvalidateBusinessObject(tenantID, boKey)
	return nil
}

// InvalidateOnFieldChange invalidates cache when fields are modified
func (s *InvalidationService) InvalidateOnFieldChange(
	ctx context.Context,
	tenantID, boKey string,
) error {
	log.Printf("[InvalidationService] Invalidating cache for tenant=%s, bo=%s (fields changed)", tenantID, boKey)
	s.cache.InvalidateBusinessObject(tenantID, boKey)
	return nil
}

// InvalidateOnRelationshipChange invalidates cache when relationships change
func (s *InvalidationService) InvalidateOnRelationshipChange(
	ctx context.Context,
	tenantID, boKey string,
) error {
	log.Printf("[InvalidationService] Invalidating cache for tenant=%s, bo=%s (relationships changed)", tenantID, boKey)
	s.cache.InvalidateBusinessObject(tenantID, boKey)
	return nil
}

// WarmCacheAfterInvalidation reloads metadata after invalidation
func (s *InvalidationService) WarmCacheAfterInvalidation(
	ctx context.Context,
	tenantID string,
) error {
	log.Printf("[InvalidationService] Warming cache for tenant=%s", tenantID)
	return s.cache.WarmCache(ctx, tenantID)
}

// InvalidateAll invalidates all cached metadata (use sparingly!)
func (s *InvalidationService) InvalidateAll(ctx context.Context, tenantIDs []string) error {
	log.Printf("[InvalidationService] Invalidating cache for %d tenants", len(tenantIDs))
	for _, tenantID := range tenantIDs {
		s.cache.InvalidateTenant(tenantID)
	}
	return nil
}

// GetInvalidationStats returns statistics about cache invalidations
func (s *InvalidationService) GetInvalidationStats() map[string]interface{} {
	metrics := s.cache.GetMetrics()
	return map[string]interface{}{
		"total_evictions": metrics.Evictions,
		"current_items":   metrics.ItemCount,
		"hit_rate":        fmt.Sprintf("%.2f%%", metrics.HitRate*100),
	}
}
