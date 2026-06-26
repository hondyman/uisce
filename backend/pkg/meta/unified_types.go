package meta

import (
	"time"

	"github.com/hondyman/semlayer/backend/internal/cache"
)

// UnifiedMetadata represents all metadata for a tenant (business objects + semantic views)
type UnifiedMetadata struct {
	TenantID        string                      `json:"tenant_id"`
	BusinessObjects []*BusinessObjectDefinition `json:"business_objects"`
	SemanticViews   []*cache.SemanticViewSchema `json:"semantic_views"`
	CachedAt        time.Duration               `json:"cached_at"`
}

// BOToViewMapping represents a mapping between a business object and semantic view
type BOToViewMapping struct {
	TenantID      string         `json:"tenant_id"`
	BOKey         string         `json:"bo_key"`
	BOName        string         `json:"bo_name"`
	ViewID        string         `json:"view_id"`
	ViewName      string         `json:"view_name"`
	FieldMappings []FieldMapping `json:"field_mappings"`
}

// FieldMapping represents a mapping between a BO field and semantic view field
type FieldMapping struct {
	BOFieldName   string `json:"bo_field_name"`
	BOFieldType   string `json:"bo_field_type"`
	ViewFieldName string `json:"view_field_name"`
	ViewFieldType string `json:"view_field_type"`
	MappingType   string `json:"mapping_type"` // direct, computed, derived
}

// CombinedMetrics represents metrics from both caches
type CombinedMetrics struct {
	BusinessObjects CacheMetrics           `json:"business_objects"`
	SemanticViews   map[string]interface{} `json:"semantic_views"`
}
