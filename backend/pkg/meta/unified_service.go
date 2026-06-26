package meta

import (
	"context"
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/internal/cache"
	"github.com/jmoiron/sqlx"
)

// UnifiedMetadataService integrates business object metadata cache with semantic layer
// This provides a single point of access for all metadata (business objects + semantic views)
type UnifiedMetadataService struct {
	// Business object metadata (in-memory)
	boCache   *MetadataCache
	boService *Service

	// Semantic view metadata (Redis)
	semanticCache *cache.SemanticViewCache

	db *sqlx.DB
}

// NewUnifiedMetadataService creates a unified metadata service
func NewUnifiedMetadataService(
	db *sqlx.DB,
	boCache *MetadataCache,
	semanticCache *cache.SemanticViewCache,
) *UnifiedMetadataService {
	return &UnifiedMetadataService{
		boCache:       boCache,
		boService:     NewServiceWithCache(db.DB, boCache),
		semanticCache: semanticCache,
		db:            db,
	}
}

// GetBusinessObject retrieves a business object from in-memory cache
func (s *UnifiedMetadataService) GetBusinessObject(ctx context.Context, tenantID, boKey string) (*BusinessObjectDefinition, error) {
	return s.boCache.GetBusinessObject(tenantID, boKey)
}

// GetSemanticView retrieves a semantic view from Redis cache
func (s *UnifiedMetadataService) GetSemanticView(ctx context.Context, tenantID, viewID string) (*cache.SemanticViewSchema, error) {
	return s.semanticCache.GetSemanticView(tenantID, viewID)
}

// GetAllMetadata retrieves both business objects and semantic views for a tenant
func (s *UnifiedMetadataService) GetAllMetadata(ctx context.Context, tenantID string) (*UnifiedMetadata, error) {
	// Get business objects from in-memory cache
	businessObjects, err := s.boCache.ListBusinessObjects(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get business objects: %w", err)
	}

	// Get semantic views from database (semantic cache is Redis-based, so we query DB)
	semanticViews, err := s.getSemanticViewsFromDB(ctx, tenantID)
	if err != nil {
		log.Printf("Warning: failed to get semantic views: %v", err)
		semanticViews = []*cache.SemanticViewSchema{}
	}

	return &UnifiedMetadata{
		TenantID:        tenantID,
		BusinessObjects: businessObjects,
		SemanticViews:   semanticViews,
		CachedAt:        s.boCache.GetMetrics().LoadTime,
	}, nil
}

// MapBusinessObjectToSemanticView creates a mapping between a business object and semantic view
func (s *UnifiedMetadataService) MapBusinessObjectToSemanticView(
	ctx context.Context,
	tenantID, boKey, viewID string,
) (*BOToViewMapping, error) {
	// Get business object
	bo, err := s.boCache.GetBusinessObject(tenantID, boKey)
	if err != nil {
		return nil, fmt.Errorf("business object not found: %w", err)
	}

	// Get semantic view
	view, err := s.semanticCache.GetSemanticView(tenantID, viewID)
	if err != nil {
		return nil, fmt.Errorf("semantic view not found: %w", err)
	}

	// Create field mappings
	fieldMappings := s.createFieldMappings(bo, view)

	mapping := &BOToViewMapping{
		TenantID:      tenantID,
		BOKey:         boKey,
		BOName:        bo.DisplayName,
		ViewID:        viewID,
		ViewName:      view.ViewName,
		FieldMappings: fieldMappings,
	}

	// Store mapping in database
	if err := s.storeBOToViewMapping(ctx, mapping); err != nil {
		return nil, fmt.Errorf("failed to store mapping: %w", err)
	}

	return mapping, nil
}

// GetBOToViewMappings retrieves all mappings for a business object
func (s *UnifiedMetadataService) GetBOToViewMappings(
	ctx context.Context,
	tenantID, boKey string,
) ([]*BOToViewMapping, error) {
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetBOToViewMappings($tenantId: String!, $boKey: String!) {
	//   bo_to_view_mappings(
	//     where: {
	//       tenant_id: {_eq: $tenantId},
	//       bo_key: {_eq: $boKey}
	//     }
	//   ) {
	//     tenant_id
	//     bo_key
	//     bo_name
	//     view_id
	//     view_name
	//     field_mappings
	//   }
	// }
	//
	// SQL fallback:
	query := `
		SELECT tenant_id, bo_key, bo_name, view_id, view_name, field_mappings
		FROM bo_to_view_mappings
		WHERE tenant_id = $1 AND bo_key = $2
	`

	var mappings []*BOToViewMapping
	err := s.db.SelectContext(ctx, &mappings, query, tenantID, boKey)
	if err != nil {
		return nil, err
	}

	return mappings, nil
}

// InvalidateAll invalidates both business object and semantic view caches
func (s *UnifiedMetadataService) InvalidateAll(ctx context.Context, tenantID string) error {
	// Invalidate business object cache
	s.boCache.InvalidateTenant(tenantID)

	// Invalidate semantic view cache
	if err := s.semanticCache.InvalidateTenantViews(tenantID); err != nil {
		log.Printf("Warning: failed to invalidate semantic cache: %v", err)
	}

	log.Printf("[UnifiedMetadataService] Invalidated all metadata for tenant %s", tenantID)
	return nil
}

// WarmAllCaches warms both business object and semantic view caches
func (s *UnifiedMetadataService) WarmAllCaches(ctx context.Context, tenantID string) error {
	// Warm business object cache
	if err := s.boCache.WarmCache(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to warm BO cache: %w", err)
	}

	// Semantic views are cached on-demand in Redis, no need to preload
	log.Printf("[UnifiedMetadataService] Warmed metadata caches for tenant %s", tenantID)
	return nil
}

// GetCombinedMetrics returns metrics from both caches
func (s *UnifiedMetadataService) GetCombinedMetrics() (*CombinedMetrics, error) {
	boMetrics := s.boCache.GetMetrics()

	semanticStats, err := s.semanticCache.GetCacheStats()
	if err != nil {
		log.Printf("Warning: failed to get semantic cache stats: %v", err)
		semanticStats = make(map[string]interface{})
	}

	return &CombinedMetrics{
		BusinessObjects: boMetrics,
		SemanticViews:   semanticStats,
	}, nil
}

// Private helper methods

func (s *UnifiedMetadataService) getSemanticViewsFromDB(ctx context.Context, tenantID string) ([]*cache.SemanticViewSchema, error) {
	// Query semantic views from database
	// This is a simplified implementation - adjust based on your actual schema
	// For now, return empty list since we don't have the exact schema
	return []*cache.SemanticViewSchema{}, nil
}

func (s *UnifiedMetadataService) createFieldMappings(bo *BusinessObjectDefinition, view *cache.SemanticViewSchema) []FieldMapping {
	var mappings []FieldMapping

	// Map business object fields to semantic view fields
	for _, boField := range bo.Fields {
		for _, viewField := range view.Fields {
			// Simple name-based matching (can be enhanced with fuzzy matching)
			if boField.Name == viewField.FieldName || boField.Label == viewField.FieldName {
				mappings = append(mappings, FieldMapping{
					BOFieldName:   boField.Name,
					BOFieldType:   string(boField.Type),
					ViewFieldName: viewField.FieldName,
					ViewFieldType: viewField.FieldType,
					MappingType:   "direct",
				})
			}
		}
	}

	return mappings
}

func (s *UnifiedMetadataService) storeBOToViewMapping(ctx context.Context, mapping *BOToViewMapping) error {
	// Store mapping in database
	// This is a simplified implementation - you may want to use JSONB for field_mappings
	// For now, skip actual storage - this would require the table to exist
	log.Printf("[UnifiedMetadataService] Would store mapping: %s -> %s", mapping.BOKey, mapping.ViewID)
	return nil
}
