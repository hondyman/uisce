package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hondyman/semlayer/backend/pkg/cache"
	"github.com/jmoiron/sqlx"
)

// MetadataCache provides in-memory caching for business object metadata
// following Workday's pattern of storing metadata in memory for fast access
type MetadataCache struct {
	cache *cache.InMemoryCache
	db    *sqlx.DB
	mu    sync.RWMutex

	// In-memory indexes for fast lookup
	boByKey       map[string]map[string]*BusinessObjectDefinition // tenantID -> key -> BO
	boByID        map[string]map[string]*BusinessObjectDefinition // tenantID -> id -> BO
	fieldsByBO    map[string]map[string][]FieldDefinition         // tenantID -> boID -> fields
	relationships map[string]map[string][]RelationshipDefinition  // tenantID -> boID -> relationships
	enumsByTenant map[string]map[string]*EnumDefinition           // tenantID -> enumID -> enum

	// Metrics
	hits      int64
	misses    int64
	evictions int64
	loadTime  time.Duration
}

// NewMetadataCache creates a new metadata cache instance
func NewMetadataCache(db *sqlx.DB) *MetadataCache {
	return &MetadataCache{
		cache:         cache.New(cache.WithMaxItems(100000)),
		db:            db,
		boByKey:       make(map[string]map[string]*BusinessObjectDefinition),
		boByID:        make(map[string]map[string]*BusinessObjectDefinition),
		fieldsByBO:    make(map[string]map[string][]FieldDefinition),
		relationships: make(map[string]map[string][]RelationshipDefinition),
		enumsByTenant: make(map[string]map[string]*EnumDefinition),
	}
}

// Preload loads all metadata into memory for a tenant on startup
// This follows Workday's pattern of loading metadata at startup for performance
func (mc *MetadataCache) Preload(ctx context.Context, tenantID string) error {
	startTime := time.Now()
	mc.mu.Lock()
	defer mc.mu.Unlock()

	log.Printf("[MetadataCache] Preloading metadata for tenant %s...", tenantID)

	// Initialize maps for this tenant
	mc.boByKey[tenantID] = make(map[string]*BusinessObjectDefinition)
	mc.boByID[tenantID] = make(map[string]*BusinessObjectDefinition)
	mc.fieldsByBO[tenantID] = make(map[string][]FieldDefinition)
	mc.relationships[tenantID] = make(map[string][]RelationshipDefinition)
	mc.enumsByTenant[tenantID] = make(map[string]*EnumDefinition)

	// Load business objects
	if err := mc.loadBusinessObjects(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to load business objects: %w", err)
	}

	// Load fields
	if err := mc.loadFields(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to load fields: %w", err)
	}

	// Load relationships
	if err := mc.loadRelationships(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to load relationships: %w", err)
	}

	// Load enums
	if err := mc.loadEnums(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to load enums: %w", err)
	}

	mc.loadTime = time.Since(startTime)
	log.Printf("[MetadataCache] Preloaded %d business objects for tenant %s in %v",
		len(mc.boByKey[tenantID]), tenantID, mc.loadTime)

	return nil
}

// GetBusinessObject retrieves a business object from memory cache
func (mc *MetadataCache) GetBusinessObject(tenantID, key string) (*BusinessObjectDefinition, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	tenantBOs, ok := mc.boByKey[tenantID]
	if !ok {
		mc.misses++
		return nil, fmt.Errorf("tenant %s not found in cache", tenantID)
	}

	bo, ok := tenantBOs[key]
	if !ok {
		mc.misses++
		return nil, fmt.Errorf("business object %s not found for tenant %s", key, tenantID)
	}

	mc.hits++

	// Return a copy to prevent external modifications
	boCopy := *bo
	boCopy.Fields = append([]FieldDefinition{}, bo.Fields...)
	boCopy.Relationships = append([]RelationshipDefinition{}, bo.Relationships...)

	return &boCopy, nil
}

// GetBusinessObjectByID retrieves a business object by ID from memory cache
func (mc *MetadataCache) GetBusinessObjectByID(tenantID, id string) (*BusinessObjectDefinition, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	tenantBOs, ok := mc.boByID[tenantID]
	if !ok {
		mc.misses++
		return nil, fmt.Errorf("tenant %s not found in cache", tenantID)
	}

	bo, ok := tenantBOs[id]
	if !ok {
		mc.misses++
		return nil, fmt.Errorf("business object with ID %s not found for tenant %s", id, tenantID)
	}

	mc.hits++

	// Return a copy
	boCopy := *bo
	boCopy.Fields = append([]FieldDefinition{}, bo.Fields...)
	boCopy.Relationships = append([]RelationshipDefinition{}, bo.Relationships...)

	return &boCopy, nil
}

// ListBusinessObjects returns all business objects for a tenant
func (mc *MetadataCache) ListBusinessObjects(tenantID string) ([]*BusinessObjectDefinition, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	tenantBOs, ok := mc.boByKey[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant %s not found in cache", tenantID)
	}

	result := make([]*BusinessObjectDefinition, 0, len(tenantBOs))
	for _, bo := range tenantBOs {
		boCopy := *bo
		boCopy.Fields = append([]FieldDefinition{}, bo.Fields...)
		boCopy.Relationships = append([]RelationshipDefinition{}, bo.Relationships...)
		result = append(result, &boCopy)
	}

	mc.hits++
	return result, nil
}

// GetEnum retrieves an enum definition from cache
func (mc *MetadataCache) GetEnum(tenantID, enumID string) (*EnumDefinition, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	tenantEnums, ok := mc.enumsByTenant[tenantID]
	if !ok {
		mc.misses++
		return nil, fmt.Errorf("tenant %s not found in cache", tenantID)
	}

	enum, ok := tenantEnums[enumID]
	if !ok {
		mc.misses++
		return nil, fmt.Errorf("enum %s not found for tenant %s", enumID, tenantID)
	}

	mc.hits++
	return enum, nil
}

// InvalidateBusinessObject clears cache for a specific business object
func (mc *MetadataCache) InvalidateBusinessObject(tenantID, key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if tenantBOs, ok := mc.boByKey[tenantID]; ok {
		if bo, exists := tenantBOs[key]; exists {
			delete(mc.boByID[tenantID], bo.ID)
			delete(tenantBOs, key)
			delete(mc.fieldsByBO[tenantID], bo.ID)
			delete(mc.relationships[tenantID], bo.ID)
			mc.evictions++
		}
	}
}

// InvalidateTenant clears all cache for a tenant
func (mc *MetadataCache) InvalidateTenant(tenantID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.boByKey, tenantID)
	delete(mc.boByID, tenantID)
	delete(mc.fieldsByBO, tenantID)
	delete(mc.relationships, tenantID)
	delete(mc.enumsByTenant, tenantID)
	mc.evictions++
}

// WarmCache refreshes all metadata from database for a tenant
func (mc *MetadataCache) WarmCache(ctx context.Context, tenantID string) error {
	mc.InvalidateTenant(tenantID)
	return mc.Preload(ctx, tenantID)
}

// GetMetrics returns cache performance metrics
func (mc *MetadataCache) GetMetrics() CacheMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	totalItems := 0
	for _, tenantBOs := range mc.boByKey {
		totalItems += len(tenantBOs)
	}

	var hitRate float64
	total := mc.hits + mc.misses
	if total > 0 {
		hitRate = float64(mc.hits) / float64(total)
	}

	return CacheMetrics{
		Hits:        mc.hits,
		Misses:      mc.misses,
		Evictions:   mc.evictions,
		HitRate:     hitRate,
		LoadTime:    mc.loadTime,
		ItemCount:   totalItems,
		MemoryBytes: mc.estimateMemoryUsage(),
	}
}

// Private helper methods

func (mc *MetadataCache) loadBusinessObjects(ctx context.Context, tenantID string) error {
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query LoadBusinessObjects($tenantId: String!) {
	//   business_objects(
	//     where: {tenant_id: {_eq: $tenantId}},
	//     order_by: {name: asc}
	//   ) {
	//     id
	//     tenant_id
	//     name
	//     display_name
	//     description
	//     icon
	//     metadata
	//     created_at
	//     updated_at
	//   }
	// }
	//
	// SQL fallback:
	query := `
		SELECT id, tenant_id, name, display_name, description, icon, 
		       metadata, created_at, updated_at
		FROM business_objects
		WHERE tenant_id = $1
		ORDER BY name
	`

	rows, err := mc.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var bo BusinessObjectDefinition
		var metadataJSON []byte

		err := rows.Scan(
			&bo.ID, &bo.TenantID, &bo.Name, &bo.DisplayName,
			&bo.Description, &bo.Icon, &metadataJSON,
			&bo.CreatedAt, &bo.UpdatedAt,
		)
		if err != nil {
			return err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &bo.Metadata); err != nil {
				log.Printf("Warning: failed to unmarshal metadata for BO %s: %v", bo.Name, err)
			}
		}

		// Set defaults
		bo.Status = "active"
		bo.Version = 1
		bo.Storage = "row"
		bo.CachedAt = time.Now()

		// Store in both indexes
		mc.boByKey[tenantID][bo.Name] = &bo
		mc.boByID[tenantID][bo.ID] = &bo
	}

	return rows.Err()
}

func (mc *MetadataCache) loadFields(ctx context.Context, tenantID string) error {
	// TODO(hasura-migration): Replace SQL JOIN query with Hasura GraphQL query with relationship
	// Example GraphQL query:
	// query LoadFields($tenantId: String!) {
	//   bo_fields(
	//     where: {business_object: {tenant_id: {_eq: $tenantId}}},
	//     order_by: [{business_object_id: asc}, {sequence: asc}]
	//   ) {
	//     id
	//     business_object_id
	//     name
	//     label
	//     type
	//     is_required
	//     is_unique
	//     enum_id
	//     ref_object_id
	//     default_value
	//     validation_json
	//     visibility_json
	//   }
	// }
	//
	// SQL fallback:
	query := `
		SELECT f.id, f.business_object_id, f.name, f.label, f.type,
		       f.is_required, f.is_unique, f.enum_id, f.ref_object_id,
		       f.default_value, f.validation_json, f.visibility_json
		FROM bo_fields f
		JOIN business_objects bo ON f.business_object_id = bo.id
		WHERE bo.tenant_id = $1
		ORDER BY f.business_object_id, f.sequence
	`

	rows, err := mc.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var field FieldDefinition
		var validationJSON, visibilityJSON []byte

		err := rows.Scan(
			&field.ID, &field.BusinessObjectID, &field.Name, &field.Label,
			&field.Type, &field.IsRequired, &field.IsUnique,
			&field.EnumID, &field.RefObjectID, &field.DefaultValue,
			&validationJSON, &visibilityJSON,
		)
		if err != nil {
			return err
		}

		field.TenantID = tenantID
		field.ValidationJSON = validationJSON
		field.VisibilityJSON = visibilityJSON

		// Add to fields index
		mc.fieldsByBO[tenantID][field.BusinessObjectID] = append(
			mc.fieldsByBO[tenantID][field.BusinessObjectID],
			field,
		)

		// Add to business object
		if bo, ok := mc.boByID[tenantID][field.BusinessObjectID]; ok {
			bo.Fields = append(bo.Fields, field)
		}
	}

	return rows.Err()
}

func (mc *MetadataCache) loadRelationships(ctx context.Context, tenantID string) error {
	// Relationships table may not exist yet, so we'll skip for now
	// This will be implemented when the relationships table is created
	return nil
}

func (mc *MetadataCache) loadEnums(ctx context.Context, tenantID string) error {
	// Enums table may not exist yet, so we'll skip for now
	// This will be implemented when the enums table is created
	return nil
}

func (mc *MetadataCache) estimateMemoryUsage() int64 {
	// Rough estimation: 1KB per business object + 100 bytes per field
	var total int64
	for _, tenantBOs := range mc.boByKey {
		total += int64(len(tenantBOs)) * 1024
	}
	for _, tenantFields := range mc.fieldsByBO {
		for _, fields := range tenantFields {
			total += int64(len(fields)) * 100
		}
	}
	return total
}

// CacheMetrics holds cache performance statistics
type CacheMetrics struct {
	Hits        int64         `json:"hits"`
	Misses      int64         `json:"misses"`
	Evictions   int64         `json:"evictions"`
	HitRate     float64       `json:"hit_rate"`
	LoadTime    time.Duration `json:"load_time"`
	ItemCount   int           `json:"item_count"`
	MemoryBytes int64         `json:"memory_bytes"`
}
