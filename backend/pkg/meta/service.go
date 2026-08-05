package meta

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type Service struct {
	db    *sql.DB
	cache *MetadataCache
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func NewServiceWithCache(db *sql.DB, cache *MetadataCache) *Service {
	return &Service{db: db, cache: cache}
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
func (s *Service) GetBusinessObject(ctx context.Context, id string) (*BusinessObjectDefinition, error) {
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
func (s *Service) ListBusinessObjects(ctx context.Context, tenantID string) ([]*BusinessObjectDefinition, error) {
	if s.cache != nil {
		objects, err := s.cache.ListBusinessObjects(tenantID)
		if err == nil {
			return objects, nil
		}
	}

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
	query := `UPDATE core_bo SET status = 'deprecated' WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
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
