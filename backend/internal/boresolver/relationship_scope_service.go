package boresolver

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type RelatedObjectExpansion struct {
	RelationshipID   uuid.UUID                 `json:"relationshipId" db:"relationship_id"`
	SourceBOID       uuid.UUID                 `json:"sourceBoId" db:"source_bo_id"`
	RelatedBOID      uuid.UUID                 `json:"relatedBoId" db:"related_bo_id"`
	RelatedBOKey     string                    `json:"relatedBoKey" db:"related_bo_key"`
	RelatedBOName    string                    `json:"relatedBoName" db:"related_bo_name"`
	Cardinality      string                    `json:"cardinality" db:"cardinality"` // 1:1, 1:N, M:1, M:N
	JoinType         string                    `json:"joinType" db:"join_type"`       // INNER, LEFT, FULL
	JoinConditionSQL string                    `json:"joinConditionSql" db:"join_condition_sql"`
	AvailableFields  []EligibleFieldDescriptor `json:"availableFields"`
}

type EligibleFieldDescriptor struct {
	TermNodeID  uuid.UUID `json:"termNodeId" db:"term_node_id"`
	TermKey     string    `json:"termKey" db:"term_key"`
	TermName    string    `json:"termName" db:"term_name"`
	DataType    string    `json:"dataType" db:"data_type"`
	DefaultRole string    `json:"defaultRole" db:"default_role"`
}

type RelationshipScopeService struct {
	db *sqlx.DB
}

func NewRelationshipScopeService(db *sqlx.DB) *RelationshipScopeService {
	return &RelationshipScopeService{db: db}
}

// GetRelatedObjectsForBO retrieves all related business objects and their eligible fields with cardinality
func (s *RelationshipScopeService) GetRelatedObjectsForBO(ctx context.Context, tenantID, boID uuid.UUID) ([]RelatedObjectExpansion, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		return []RelatedObjectExpansion{}, nil
	}

	query := `
		SELECT 
			r.id AS relationship_id,
			r.source_bo_id,
			r.target_bo_id AS related_bo_id,
			COALESCE(t.bo_key, '') AS related_bo_key,
			COALESCE(t.name, '') AS related_bo_name,
			COALESCE(r.cardinality, 'M:1') AS cardinality,
			COALESCE(r.join_type, 'LEFT') AS join_type,
			COALESCE(r.join_condition_sql, '') AS join_condition_sql
		FROM business_object_relationship r
		JOIN business_object t ON r.target_bo_id = t.id
		WHERE r.source_bo_id = $1 AND r.tenant_id = $2
	`

	var results []RelatedObjectExpansion
	err := s.db.SelectContext(ctx, &results, query, boID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed fetching related objects: %w", err)
	}

	for i := range results {
		var fields []EligibleFieldDescriptor
		fieldsQuery := `
			SELECT 
				cn.id AS term_node_id,
				cn.node_key AS term_key,
				cn.node_name AS term_name,
				COALESCE(cn.properties->>'data_type', 'string') AS data_type,
				COALESCE(cn.properties->>'default_role', 'dimension') AS default_role
			FROM catalog_node cn
			JOIN catalog_edge ce ON ce.target_node_id = cn.id
			WHERE ce.source_node_id = $1 AND ce.edge_type = 'HAS_FIELD' AND cn.tenant_id = $2
		`
		if err := s.db.SelectContext(ctx, &fields, fieldsQuery, results[i].RelatedBOID, tenantID); err == nil {
			results[i].AvailableFields = fields
		}
	}

	return results, nil
}
