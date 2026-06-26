package scheduler_intelligence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/jmoiron/sqlx"
)

// SemanticAdapter wraps the analytics semantic service to provide
// scheduler specific semantic operations
type SemanticAdapter struct {
	db          *sqlx.DB
	semanticSvc *analytics.SemanticService
}

// NewSemanticAdapter creates a new adapter
func NewSemanticAdapter(db *sqlx.DB, svc *analytics.SemanticService) *SemanticAdapter {
	return &SemanticAdapter{
		db:          db,
		semanticSvc: svc,
	}
}

// ResolveBindings resolves string references to IDs
func (s *SemanticAdapter) ResolveBindings(ctx context.Context, refIDs []string) (SemanticBinding, error) {
	// Simple implementation: check if the string is a valid UUID, if so use it.
	// Otherwise, try to look up by name in catalog_node.
	// This is a basic implementation for Phase 13.

	binding := SemanticBinding{}

	// TODO: Bulk lookup optimization
	for _, ref := range refIDs {
		// Identify type based on prefix or metadata if available
		// For now, we'll try to find it in catalog_node

		var id string
		var nodeType string

		query := `
			SELECT cn.id, cnt.catalog_type_name
			FROM catalog_node cn
			JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
			WHERE cn.node_name = $1 OR cn.qualified_path = $1 OR cn.id::text = $1
			LIMIT 1
		`

		row := s.db.QueryRowContext(ctx, query, ref)
		if err := row.Scan(&id, &nodeType); err != nil {
			if err != sql.ErrNoRows {
				// Log error but continue
				continue
			}
			// Not found
			continue
		}

		// Map based on type (simplified mapping)
		switch {
		case strings.Contains(nodeType, "business_object") || strings.Contains(nodeType, "table"):
			binding.BOIDs = append(binding.BOIDs, id)
		case strings.Contains(nodeType, "api"):
			binding.APIIDs = append(binding.APIIDs, id)
		case strings.Contains(nodeType, "page"):
			binding.PageIDs = append(binding.PageIDs, id)
		case strings.Contains(nodeType, "workflow"):
			binding.WorkflowIDs = append(binding.WorkflowIDs, id)
		case strings.Contains(nodeType, "metric") || strings.Contains(nodeType, "cube"):
			binding.PreAggIDs = append(binding.PreAggIDs, id)
		default:
			// Fallback to BO
			binding.BOIDs = append(binding.BOIDs, id)
		}
	}

	return binding, nil
}

// GetImpactedObjects returns downstream impacted objects
func (s *SemanticAdapter) GetImpactedObjects(ctx context.Context, semanticIDs []string) ([]ImpactedObject, error) {
	// Stub for Phase 13
	return []ImpactedObject{}, nil
}

// GetDriftStatus returns drift status
func (s *SemanticAdapter) GetDriftStatus(ctx context.Context, semanticIDs []string) ([]DriftStatus, error) {
	// Stub for Phase 13
	return []DriftStatus{}, nil
}

// GetBOsByPhysicalTable returns Business Object IDs for a given table
func (s *SemanticAdapter) GetBOsByPhysicalTable(ctx context.Context, tableName string) ([]string, error) {
	// Lookup business objects where driver_table_name matches
	// We handle fully qualified names "schema.table" vs just "table" loosely

	var ids []string

	// Try exact match first
	query := `SELECT id FROM business_objects WHERE driver_table_name = $1 AND is_active = true`
	err := s.db.SelectContext(ctx, &ids, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup BOs by table: %w", err)
	}

	if len(ids) > 0 {
		return ids, nil
	}

	// Try suffix match if tableName provided is fully qualified
	if strings.Contains(tableName, ".") {
		parts := strings.Split(tableName, ".")
		if len(parts) == 2 {
			simpleName := parts[1]
			query = `SELECT id FROM business_objects WHERE driver_table_name = $1 AND is_active = true`
			_ = s.db.SelectContext(ctx, &ids, query, simpleName)
		}
	}

	return ids, nil
}
