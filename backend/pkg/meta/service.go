package meta

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// Service provides CRUD operations for metadata definitions with in-memory caching
type Service struct {
	db     *sql.DB
	hasura HasuraClient
	cache  *MetadataCache // In-memory cache for fast metadata access
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// NewServiceWithHasura creates a new meta service with Hasura support
func NewServiceWithHasura(db *sql.DB, hasura HasuraClient) *Service {
	return &Service{db: db, hasura: hasura}
}

// NewServiceWithCache creates a new meta service with caching enabled
func NewServiceWithCache(db *sql.DB, cache *MetadataCache) *Service {
	return &Service{db: db, cache: cache}
}

// NewServiceWithAll creates a new meta service with both Hasura and caching
func NewServiceWithAll(db *sql.DB, hasura HasuraClient, cache *MetadataCache) *Service {
	return &Service{db: db, hasura: hasura, cache: cache}
}

// CreateBusinessObject creates a new business object definition
func (s *Service) CreateBusinessObject(ctx context.Context, bo *BusinessObjectDefinition) error {
	fieldsJSON, err := json.Marshal(bo.Fields)
	if err != nil {
		return fmt.Errorf("failed to marshal fields: %w", err)
	}

	metadataJSON, err := json.Marshal(bo.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if s.hasura != nil {
		err := s.createBusinessObjectWithHasura(ctx, bo, fieldsJSON, metadataJSON)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	query := `
		INSERT INTO core_bo (id, tenant_id, name, storage, version, status, fields, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = s.db.ExecContext(ctx, query,
		bo.ID, bo.TenantID, bo.Name, bo.Storage,
		bo.Version, bo.Status, fieldsJSON, metadataJSON,
	)

	return err
}

// GetBusinessObject retrieves a business object by ID
// If cache is enabled, tries cache first before falling back to database
func (s *Service) GetBusinessObject(ctx context.Context, id string) (*BusinessObjectDefinition, error) {
	// Try cache first if available
	if s.cache != nil {
		// Note: cache uses name as key, so we need to query by ID differently
		// For now, fall through to database query
	}

	if s.hasura != nil {
		bo, err := s.getBusinessObjectWithHasura(ctx, id)
		if err == nil {
			return bo, nil
		}
		fmt.Printf("Hasura query failed, falling back to SQL: %v\n", err)
	}

	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetBusinessObject($id: uuid!) {
	//   core_bo_by_pk(id: $id) {
	//     id tenant_id name storage version status fields metadata
	//   }
	// }
	//
	// SQL fallback:
	query := `
		SELECT id, tenant_id, name, storage, version, status, fields, metadata
		FROM core_bo
		WHERE id = $1
	`

	var bo BusinessObjectDefinition
	var fieldsJSON, metadataJSON []byte

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&bo.ID, &bo.TenantID, &bo.Name, &bo.Storage,
		&bo.Version, &bo.Status, &fieldsJSON, &metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(fieldsJSON, &bo.Fields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &bo.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &bo, nil
}

// GetBusinessObjectByName retrieves a business object by name (uses cache if available)
func (s *Service) GetBusinessObjectByName(ctx context.Context, tenantID, name string) (*BusinessObjectDefinition, error) {
	// Try cache first if available
	if s.cache != nil {
		bo, err := s.cache.GetBusinessObject(tenantID, name)
		if err == nil {
			return bo, nil
		}
		// Cache miss, fall through to database
	}

	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetBusinessObjectByName($tenantId: String!, $name: String!) {
	//   core_bo(where: {tenant_id: {_eq: $tenantId}, name: {_eq: $name}, status: {_eq: "active"}}) {
	//     id tenant_id name storage version status fields metadata
	//   }
	// }
	//
	// SQL fallback - Query database
	query := `
		SELECT id, tenant_id, name, storage, version, status, fields, metadata
		FROM core_bo
		WHERE tenant_id = $1 AND name = $2 AND status = 'active'
	`

	var bo BusinessObjectDefinition
	var fieldsJSON, metadataJSON []byte

	err := s.db.QueryRowContext(ctx, query, tenantID, name).Scan(
		&bo.ID, &bo.TenantID, &bo.Name, &bo.Storage,
		&bo.Version, &bo.Status, &fieldsJSON, &metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(fieldsJSON, &bo.Fields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &bo.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &bo, nil
}

// ListBusinessObjects returns all business objects for a tenant
// Uses cache if available for fast access
func (s *Service) ListBusinessObjects(ctx context.Context, tenantID string) ([]*BusinessObjectDefinition, error) {
	// Try cache first if available
	if s.cache != nil {
		objects, err := s.cache.ListBusinessObjects(tenantID)
		if err == nil {
			return objects, nil
		}
		// Cache miss, fall through to database
	}

	if s.hasura != nil {
		objects, err := s.listBusinessObjectsWithHasura(ctx, tenantID)
		if err == nil {
			return objects, nil
		}
		fmt.Printf("Hasura query failed, falling back to SQL: %v\n", err)
	}

	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query ListBusinessObjects($tenantId: String!) {
	//   core_bo(where: {tenant_id: {_eq: $tenantId}, status: {_eq: "active"}}, order_by: {name: asc}) {
	//     id tenant_id name storage version status fields metadata
	//   }
	// }
	//
	// SQL fallback:
	query := `
		SELECT id, tenant_id, name, storage, version, status, fields, metadata
		FROM core_bo
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []*BusinessObjectDefinition

	for rows.Next() {
		var bo BusinessObjectDefinition
		var fieldsJSON, metadataJSON []byte

		err := rows.Scan(
			&bo.ID, &bo.TenantID, &bo.Name, &bo.Storage,
			&bo.Version, &bo.Status, &fieldsJSON, &metadataJSON,
		)

		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(fieldsJSON, &bo.Fields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal fields: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &bo.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		objects = append(objects, &bo)
	}

	return objects, rows.Err()
}

// WarmCache preloads all metadata for a tenant into cache
func (s *Service) WarmCache(ctx context.Context, tenantID string) error {
	if s.cache == nil {
		return fmt.Errorf("cache not enabled")
	}
	return s.cache.WarmCache(ctx, tenantID)
}

// GetCacheMetrics returns cache performance metrics
func (s *Service) GetCacheMetrics() (CacheMetrics, error) {
	if s.cache == nil {
		return CacheMetrics{}, fmt.Errorf("cache not enabled")
	}
	return s.cache.GetMetrics(), nil
}

// InvalidateCache invalidates the cache for a tenant
func (s *Service) InvalidateCache(tenantID string) {
	if s.cache != nil {
		s.cache.InvalidateTenant(tenantID)
	}
}

// UpdateBusinessObject updates an existing business object
func (s *Service) UpdateBusinessObject(ctx context.Context, bo *BusinessObjectDefinition) error {
	fieldsJSON, err := json.Marshal(bo.Fields)
	if err != nil {
		return fmt.Errorf("failed to marshal fields: %w", err)
	}

	metadataJSON, err := json.Marshal(bo.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if s.hasura != nil {
		err := s.updateBusinessObjectWithHasura(ctx, bo, fieldsJSON, metadataJSON)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// TODO(hasura-migration): Replace SQL UPDATE with Hasura GraphQL mutation
	// Example GraphQL mutation:
	// mutation UpdateBusinessObject($id: uuid!, $tenantId: String!, $set: core_bo_set_input!) {
	//   update_core_bo(
	//     where: {id: {_eq: $id}, tenant_id: {_eq: $tenantId}}
	//     _set: $set
	//   ) { affected_rows }
	// }
	//
	// SQL fallback:
	query := `
		UPDATE core_bo
		SET name = $1, storage = $2, version = $3, status = $4, fields = $5, metadata = $6
		WHERE id = $7 AND tenant_id = $8
	`

	_, err = s.db.ExecContext(ctx, query,
		bo.Name, bo.Storage, bo.Version, bo.Status,
		fieldsJSON, metadataJSON, bo.ID, bo.TenantID,
	)

	return err
}

// DeleteBusinessObject soft-deletes a business object
func (s *Service) DeleteBusinessObject(ctx context.Context, id string) error {
	if s.hasura != nil {
		err := s.deleteBusinessObjectWithHasura(ctx, id)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	query := `UPDATE core_bo SET status = 'deprecated' WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// createBusinessObjectWithHasura creates a business object using Hasura GraphQL
func (s *Service) createBusinessObjectWithHasura(ctx context.Context, bo *BusinessObjectDefinition, fieldsJSON, metadataJSON []byte) error {
	mutation := `
		mutation CreateBusinessObject($object: core_bo_insert_input!) {
			insert_core_bo_one(object: $object) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"object": map[string]interface{}{
			"id":        bo.ID,
			"tenant_id": bo.TenantID,
			"name":      bo.Name,
			"storage":   bo.Storage,
			"version":   bo.Version,
			"status":    bo.Status,
			"fields":    json.RawMessage(fieldsJSON),
			"metadata":  json.RawMessage(metadataJSON),
		},
	}

	_, err := s.hasura.Mutate(mutation, variables)
	return err
}

// getBusinessObjectWithHasura retrieves a business object using Hasura GraphQL
func (s *Service) getBusinessObjectWithHasura(ctx context.Context, id string) (*BusinessObjectDefinition, error) {
	query := `
		query GetBusinessObject($id: String!) {
			core_bo_by_pk(id: $id) {
				id
				tenant_id
				name
				storage
				version
				status
				fields
				metadata
			}
		}
	`

	variables := map[string]interface{}{
		"id": id,
	}

	result, err := s.hasura.Query(query, variables)
	if err != nil {
		return nil, err
	}

	boData, ok := result["core_bo_by_pk"].(map[string]interface{})
	if !ok || boData == nil {
		return nil, sql.ErrNoRows
	}

	bo := &BusinessObjectDefinition{
		ID:       getString(boData, "id"),
		TenantID: getString(boData, "tenant_id"),
		Name:     getString(boData, "name"),
		Storage:  getString(boData, "storage"),
		Version:  getInt(boData, "version"),
		Status:   getString(boData, "status"),
	}

	// Parse JSONB fields
	if fieldsData, ok := boData["fields"]; ok && fieldsData != nil {
		fieldsJSON, err := json.Marshal(fieldsData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal fields data: %w", err)
		}
		if err := json.Unmarshal(fieldsJSON, &bo.Fields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal fields: %w", err)
		}
	}

	if metadataData, ok := boData["metadata"]; ok && metadataData != nil {
		metadataJSON, err := json.Marshal(metadataData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata data: %w", err)
		}
		if err := json.Unmarshal(metadataJSON, &bo.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return bo, nil
}

// listBusinessObjectsWithHasura retrieves business objects using Hasura GraphQL
func (s *Service) listBusinessObjectsWithHasura(ctx context.Context, tenantID string) ([]*BusinessObjectDefinition, error) {
	query := `
		query ListBusinessObjects($tenant_id: String!) {
			core_bo(
where: {
tenant_id: {_eq: $tenant_id},
status: {_eq: "active"}
},
order_by: {name: asc}
) {
				id
				tenant_id
				name
				storage
				version
				status
				fields
				metadata
			}
		}
	`

	variables := map[string]interface{}{
		"tenant_id": tenantID,
	}

	result, err := s.hasura.Query(query, variables)
	if err != nil {
		return nil, err
	}

	boList, ok := result["core_bo"].([]interface{})
	if !ok {
		return []*BusinessObjectDefinition{}, nil
	}

	objects := make([]*BusinessObjectDefinition, 0, len(boList))
	for _, item := range boList {
		boData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		bo := &BusinessObjectDefinition{
			ID:       getString(boData, "id"),
			TenantID: getString(boData, "tenant_id"),
			Name:     getString(boData, "name"),
			Storage:  getString(boData, "storage"),
			Version:  getInt(boData, "version"),
			Status:   getString(boData, "status"),
		}

		// Parse JSONB fields
		if fieldsData, ok := boData["fields"]; ok && fieldsData != nil {
			fieldsJSON, err := json.Marshal(fieldsData)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal fields data: %w", err)
			}
			if err := json.Unmarshal(fieldsJSON, &bo.Fields); err != nil {
				return nil, fmt.Errorf("failed to unmarshal fields: %w", err)
			}
		}

		if metadataData, ok := boData["metadata"]; ok && metadataData != nil {
			metadataJSON, err := json.Marshal(metadataData)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata data: %w", err)
			}
			if err := json.Unmarshal(metadataJSON, &bo.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		objects = append(objects, bo)
	}

	return objects, nil
}

// updateBusinessObjectWithHasura updates a business object using Hasura GraphQL
func (s *Service) updateBusinessObjectWithHasura(ctx context.Context, bo *BusinessObjectDefinition, fieldsJSON, metadataJSON []byte) error {
	mutation := `
		mutation UpdateBusinessObject($id: String!, $tenant_id: String!, $updates: core_bo_set_input!) {
			update_core_bo(
where: {
id: {_eq: $id},
tenant_id: {_eq: $tenant_id}
},
_set: $updates
) {
				affected_rows
			}
		}
	`

	variables := map[string]interface{}{
		"id":        bo.ID,
		"tenant_id": bo.TenantID,
		"updates": map[string]interface{}{
			"name":     bo.Name,
			"storage":  bo.Storage,
			"version":  bo.Version,
			"status":   bo.Status,
			"fields":   json.RawMessage(fieldsJSON),
			"metadata": json.RawMessage(metadataJSON),
		},
	}

	_, err := s.hasura.Mutate(mutation, variables)
	return err
}

// deleteBusinessObjectWithHasura soft-deletes a business object using Hasura GraphQL
func (s *Service) deleteBusinessObjectWithHasura(ctx context.Context, id string) error {
	mutation := `
		mutation DeleteBusinessObject($id: String!) {
			update_core_bo(
where: {id: {_eq: $id}},
_set: {status: "deprecated"}
) {
				affected_rows
			}
		}
	`

	variables := map[string]interface{}{
		"id": id,
	}

	_, err := s.hasura.Mutate(mutation, variables)
	return err
}

// Helper functions for type extraction
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	if val, ok := data[key]; ok && val != nil {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case int64:
			return int(v)
		}
	}
	return 0
}
