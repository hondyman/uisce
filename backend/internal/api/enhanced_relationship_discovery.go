package api

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/lib/pq"
)

// ============================================================================
// Enhanced Relationship Discovery Structures
// ============================================================================

// EnhancedRelatedEntity adds semantic context and multi-hop support to RelatedEntity
type EnhancedRelatedEntity struct {
	// Basic relationship info
	EntityID         string `json:"entity_id"`
	EntityName       string `json:"entity_name"`
	EntityKey        string `json:"entity_key"`
	SemanticTermID   string `json:"semantic_term_id"`
	SemanticTermName string `json:"semantic_term_name"`
	TableName        string `json:"table_name"`

	// Relationship details
	LinkType       string `json:"link_type"`       // "DIRECT_FK", "SEMANTIC", "MULTI_HOP"
	Cardinality    string `json:"cardinality"`     // "1:1", "1:N", "N:1", "N:M"
	HierarchyDepth int    `json:"hierarchy_depth"` // 1=direct, 2+=multi-hop

	// FK details
	SourceColumn   string `json:"source_column"`
	SourceTable    string `json:"source_table"`
	TargetColumn   string `json:"target_column"`
	TargetTable    string `json:"target_table"`
	ForeignKeyPath string `json:"foreign_key_path"` // "source_table.col -> target_table.col"

	// Quality metrics
	Confidence       float64 `json:"confidence"` // 0.0-1.0
	ConfidenceReason string  `json:"confidence_reason"`
	LinkReason       string  `json:"link_reason"` // Human-readable explanation

	// Multi-hop support
	RelationshipPath []PathHop `json:"relationship_path,omitempty"` // For multi-hop

	// Metadata
	DiscoveryMethod string    `json:"discovery_method"` // "FK_SCAN", "SEMANTIC_MATCH", "PATTERN"
	DiscoveredAt    time.Time `json:"discovered_at"`
}

// PathHop represents one step in a multi-hop relationship path
type PathHop struct {
	Order            int    `json:"order"`
	EntityID         string `json:"entity_id"`
	EntityName       string `json:"entity_name"`
	SemanticTermName string `json:"semantic_term_name"`
	LinkType         string `json:"link_type"`
	SourceColumn     string `json:"source_column"`
	TargetColumn     string `json:"target_column"`
	ForeignKeyPath   string `json:"foreign_key_path"`
	Cardinality      string `json:"cardinality"`
}

// RelationshipPath represents a complete multi-hop path between entities
type RelationshipPath struct {
	PathID           string                 `json:"path_id"`
	SourceEntityID   string                 `json:"source_entity_id"`
	TargetEntityID   string                 `json:"target_entity_id"`
	HierarchyDepth   int                    `json:"hierarchy_depth"`
	Hops             []PathHop              `json:"hops"`
	TotalConfidence  float64                `json:"total_confidence"`
	TotalCardinality string                 `json:"total_cardinality"`
	Entities         []RelationshipPathNode `json:"entities"`
}

// RelationshipPathNode is a node in the path
type RelationshipPathNode struct {
	Order            int    `json:"order"`
	EntityID         string `json:"entity_id"`
	EntityName       string `json:"entity_name"`
	SemanticTermName string `json:"semantic_term_name"`
	IsPrimaryKey     bool   `json:"is_primary_key"`
	ColumnName       string `json:"column_name"`
}

// EnhancedRelationshipDiscoveryService extends the basic discovery with semantic context
type EnhancedRelationshipDiscoveryService struct {
	db *sql.DB
}

// NewEnhancedRelationshipDiscoveryService creates a new enhanced discovery service
func NewEnhancedRelationshipDiscoveryService(db *sql.DB) *EnhancedRelationshipDiscoveryService {
	return &EnhancedRelationshipDiscoveryService{
		db: db,
	}
}

// ============================================================================
// Discovery Methods
// ============================================================================

// DiscoverLinkableEntitiesWithSemanticContext discovers related entities with full semantic context
// This is the enhanced version that shows what relationships mean
func (s *EnhancedRelationshipDiscoveryService) DiscoverLinkableEntitiesWithSemanticContext(
	ctx context.Context,
	tenantID, datasourceID, sourceEntityID string,
) ([]EnhancedRelatedEntity, error) {

	if sourceEntityID == "" {
		return nil, fmt.Errorf("source entity ID is required")
	}

	// SQL query that discovers relationships with semantic context
	query := `
WITH source_entity_data AS (
	-- Get the source entity and its semantic context
	SELECT 
		ea.id,
		ea.entity_key,
		ea.name as entity_name,
		cn.id as semantic_term_id,
		cn.node_name as semantic_term_name
	FROM public.entity_attribute ea
	LEFT JOIN public.catalog_node cn ON ea.catalog_node_id = cn.id
	WHERE ea.id = $1::uuid
		AND ea.tenant_id = $2::uuid
		AND ea.tenant_datasource_id = $3::uuid
),

source_entity_attributes AS (
	-- Get all columns mapped to the source entity
	SELECT DISTINCT
		eacm.table_name,
		eacm.column_name,
		eacm.is_primary_key,
		eacm.semantic_term_id,
		cn.node_name as semantic_name,
		eacm.confidence
	FROM public.entity_attribute_column_mapping eacm
	LEFT JOIN public.catalog_node cn ON eacm.semantic_term_id = cn.id
	WHERE eacm.entity_attribute_id = $1::uuid
		AND eacm.tenant_datasource_id = $3::uuid
),

semantic_to_columns AS (
	-- Link semantic terms to physical columns for matching
	SELECT DISTINCT
		eacm.semantic_term_id,
		eacm.table_name,
		eacm.column_name,
		eacm.is_foreign_key,
		eacm.confidence
	FROM public.entity_attribute_column_mapping eacm
	WHERE eacm.tenant_datasource_id = $3::uuid
		AND eacm.semantic_term_id IS NOT NULL
),

foreign_key_relationships AS (
	-- Find FK relationships using both catalog_edge and information_schema
	SELECT DISTINCT
		source_table.table_name as source_table_name,
		source_table.column_name as source_column,
		constraint_table.constraint_name,
		tc.table_name as target_table,
		kcu2.column_name as target_column,
		CASE 
			WHEN (SELECT COUNT(*) FROM information_schema.constraint_column_usage ccu 
				  WHERE ccu.constraint_name = tc.constraint_name AND ccu.table_schema = 'public') > 1 
			THEN 'MANY_TO_MANY'
			ELSE 'ONE_TO_MANY'
		END as cardinality
	FROM public.entity_attribute_column_mapping source_table
	JOIN information_schema.table_constraints tc 
		ON source_table.table_name = tc.table_name 
		AND tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = 'public'
	JOIN information_schema.key_column_usage kcu 
		ON tc.constraint_name = kcu.constraint_name 
		AND tc.table_schema = kcu.table_schema
		AND source_table.column_name = kcu.column_name
	JOIN information_schema.constraint_column_usage ccu2 
		ON tc.constraint_name = ccu2.constraint_name 
		AND tc.table_schema = ccu2.table_schema
	WHERE source_table.tenant_datasource_id = $3::uuid
),

target_entities_found AS (
	-- Match FK target tables to entities
	SELECT DISTINCT
		fk.target_table,
		fk.target_column,
		fk.source_table_name,
		fk.source_column,
		fk.constraint_name,
		fk.cardinality,
		ea.id as entity_id,
		ea.entity_key,
		ea.name as entity_name,
		cn.id as semantic_term_id,
		cn.node_name as semantic_term_name,
		eacm.confidence,
		'DIRECT_FK' as link_type,
		1 as hierarchy_depth,
		CONCAT(fk.source_table_name, '.', fk.source_column, ' -> ', fk.target_table, '.', fk.target_column) as fk_path
	FROM foreign_key_relationships fk
	JOIN public.entity_attribute_column_mapping eacm 
		ON fk.target_table = eacm.table_name
		AND fk.target_column = eacm.column_name
		AND eacm.tenant_datasource_id = $3::uuid
	JOIN public.entity_attribute ea ON eacm.entity_attribute_id = ea.id
	LEFT JOIN public.catalog_node cn ON ea.catalog_node_id = cn.id
	WHERE eacm.entity_attribute_id != $1::uuid  -- Exclude self
		AND ea.tenant_id = $2::uuid
),

column_hierarchy AS (
	-- For hierarchical columns (parent_id), find relationships
	SELECT DISTINCT
		source_col.table_name as source_table,
		source_col.column_name as source_column,
		target_col.table_name as target_table,
		target_col.column_name as target_column,
		source_col.semantic_term_id,
		target_col.semantic_term_id as target_semantic_id
	FROM public.entity_attribute_column_mapping source_col
	JOIN public.entity_attribute_column_mapping target_col 
		ON source_col.table_name = target_col.table_name
		AND source_col.column_name LIKE '%id'
		AND target_col.column_name = 'id'
	WHERE source_col.tenant_datasource_id = $3::uuid
		AND source_col.is_foreign_key = true
),

confidence_scores AS (
	-- Calculate confidence for each relationship
	SELECT 
		te.entity_id,
		te.link_type,
		-- Base score: FK exists = 0.95
		0.95::numeric as base_score,
		-- Boost if semantic terms match
		CASE 
			WHEN te.semantic_term_id IS NOT NULL THEN 0.90
			ELSE 0.80
		END as semantic_boost,
		-- Boost if column names match patterns (e.g., *_id, customer_id)
		CASE 
			WHEN te.source_column ILIKE '%id' AND te.target_column ILIKE '%id' THEN 0.85
			ELSE 0.70
		END as naming_score,
		LEAST(1.0, (0.95 + CASE WHEN te.semantic_term_id IS NOT NULL THEN 0.05 ELSE 0.0 END))::numeric as final_confidence
	FROM target_entities_found te
)

SELECT 
	te.entity_id,
	te.entity_name,
	te.entity_key,
	te.semantic_term_id,
	te.semantic_term_name,
	te.target_table,
	te.link_type,
	te.cardinality,
	te.hierarchy_depth,
	te.source_column,
	te.source_table_name,
	te.target_column,
	te.fk_path,
	cs.final_confidence as confidence,
	'FK constraint detected' as confidence_reason,
	CONCAT(te.entity_name, ' is related via ', te.constraint_name) as link_reason,
	'FK_SCAN'::text as discovery_method,
	now()::timestamp as discovered_at
FROM target_entities_found te
LEFT JOIN confidence_scores cs ON te.entity_id = cs.entity_id AND te.link_type = cs.link_type
WHERE te.source_table_name IN (SELECT DISTINCT table_name FROM source_entity_attributes)
ORDER BY cs.final_confidence DESC NULLS LAST
LIMIT 100;
`

	rows, err := s.db.QueryContext(ctx, query, sourceEntityID, tenantID, datasourceID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to discover relationships: %v", err)
		return nil, fmt.Errorf("failed to discover relationships: %w", err)
	}
	defer rows.Close()

	var entities []EnhancedRelatedEntity
	for rows.Next() {
		var entity EnhancedRelatedEntity
		var semanticTermID, semanticTermName sql.NullString
		var confidence sql.NullFloat64

		err := rows.Scan(
			&entity.EntityID,
			&entity.EntityName,
			&entity.EntityKey,
			&semanticTermID,
			&semanticTermName,
			&entity.TargetTable,
			&entity.LinkType,
			&entity.Cardinality,
			&entity.HierarchyDepth,
			&entity.SourceColumn,
			&entity.SourceTable,
			&entity.TargetColumn,
			&entity.ForeignKeyPath,
			&confidence,
			&entity.ConfidenceReason,
			&entity.LinkReason,
			&entity.DiscoveryMethod,
			&entity.DiscoveredAt,
		)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("failed to scan relationship row: %v", err)
			continue
		}

		entity.SemanticTermID = semanticTermID.String
		entity.SemanticTermName = semanticTermName.String
		if confidence.Valid {
			entity.Confidence = confidence.Float64
		}

		// Use source_table for consistency
		entity.TableName = entity.SourceTable

		entities = append(entities, entity)
	}

	if err = rows.Err(); err != nil {
		logging.GetLogger().Sugar().Errorf("error iterating relationship rows: %v", err)
		return nil, fmt.Errorf("error iterating relationship rows: %w", err)
	}

	return entities, nil
}

// DiscoverMultiHopPaths discovers relationships across multiple hops
func (s *EnhancedRelationshipDiscoveryService) DiscoverMultiHopPaths(
	ctx context.Context,
	tenantID, datasourceID, sourceEntityID string,
	maxDepth int,
) ([]RelationshipPath, error) {

	if maxDepth < 1 {
		maxDepth = 3 // Default to 3 hops
	}
	if maxDepth > 5 {
		maxDepth = 5 // Cap at 5 hops to prevent runaway queries
	}

	// Get direct relationships first
	directRelationships, err := s.DiscoverLinkableEntitiesWithSemanticContext(
		ctx, tenantID, datasourceID, sourceEntityID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to discover direct relationships: %w", err)
	}

	var paths []RelationshipPath

	// Convert direct relationships to paths
	for _, rel := range directRelationships {
		if rel.HierarchyDepth == 1 {
			path := RelationshipPath{
				PathID:           fmt.Sprintf("path_%s_to_%s_direct", sourceEntityID, rel.EntityID),
				SourceEntityID:   sourceEntityID,
				TargetEntityID:   rel.EntityID,
				HierarchyDepth:   1,
				TotalConfidence:  rel.Confidence,
				TotalCardinality: rel.Cardinality,
				Hops: []PathHop{
					{
						Order:            1,
						EntityID:         rel.EntityID,
						EntityName:       rel.EntityName,
						SemanticTermName: rel.SemanticTermName,
						LinkType:         rel.LinkType,
						SourceColumn:     rel.SourceColumn,
						TargetColumn:     rel.TargetColumn,
						ForeignKeyPath:   rel.ForeignKeyPath,
						Cardinality:      rel.Cardinality,
					},
				},
			}
			paths = append(paths, path)
		}
	}

	// For multi-hop discovery (depth > 1), recursively discover from each target
	if maxDepth > 1 {
		for _, rel := range directRelationships {
			multiHopPaths, err := s.discoverMultiHopPathsRecursive(
				ctx, tenantID, datasourceID, rel.EntityID, sourceEntityID, 2, maxDepth,
			)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("failed to discover multi-hop paths from %s: %v", rel.EntityID, err)
				continue
			}
			paths = append(paths, multiHopPaths...)
		}
	}

	return paths, nil
}

// discoverMultiHopPathsRecursive recursively discovers multi-hop paths
func (s *EnhancedRelationshipDiscoveryService) discoverMultiHopPathsRecursive(
	ctx context.Context,
	tenantID, datasourceID, currentEntityID, sourceEntityID string,
	currentDepth, maxDepth int,
) ([]RelationshipPath, error) {

	if currentDepth > maxDepth {
		return nil, nil
	}

	// Discover from current entity
	entities, err := s.DiscoverLinkableEntitiesWithSemanticContext(
		ctx, tenantID, datasourceID, currentEntityID,
	)
	if err != nil {
		return nil, err
	}

	var paths []RelationshipPath
	for _, entity := range entities {
		// Avoid cycles by checking if target is the source
		if entity.EntityID == sourceEntityID {
			continue
		}

		// Build multi-hop path
		path := RelationshipPath{
			PathID:           fmt.Sprintf("path_%s_to_%s_hop%d", sourceEntityID, entity.EntityID, currentDepth),
			SourceEntityID:   sourceEntityID,
			TargetEntityID:   entity.EntityID,
			HierarchyDepth:   currentDepth,
			TotalConfidence:  entity.Confidence * 0.85, // Reduce confidence for multi-hop
			TotalCardinality: entity.Cardinality,
			Hops: []PathHop{
				{
					Order:            currentDepth,
					EntityID:         entity.EntityID,
					EntityName:       entity.EntityName,
					SemanticTermName: entity.SemanticTermName,
					LinkType:         entity.LinkType,
					SourceColumn:     entity.SourceColumn,
					TargetColumn:     entity.TargetColumn,
					ForeignKeyPath:   entity.ForeignKeyPath,
					Cardinality:      entity.Cardinality,
				},
			},
		}
		paths = append(paths, path)
	}

	return paths, nil
}

// SaveDiscoveredRelationship saves a discovered relationship to the database
func (s *EnhancedRelationshipDiscoveryService) SaveDiscoveredRelationship(
	ctx context.Context,
	tenantID, datasourceID, sourceEntityID, targetEntityID string,
	rel *EnhancedRelatedEntity,
	isUserApplied bool,
) (string, error) {

	if sourceEntityID == "" || targetEntityID == "" {
		return "", fmt.Errorf("source and target entity IDs are required")
	}

	query := `
INSERT INTO public.entity_relationship (
	tenant_id, tenant_datasource_id,
	source_entity_id, target_entity_id,
	relationship_type, cardinality, hierarchy_depth,
	fk_constraint, source_column, source_table, target_column, target_table,
	confidence, confidence_reason,
	is_user_applied, user_applied_at,
	source_discovery_method, is_active,
	description
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
ON CONFLICT (tenant_datasource_id, source_entity_id, target_entity_id, relationship_type)
DO UPDATE SET
	is_user_applied = COALESCE($15, entity_relationship.is_user_applied),
	confidence = COALESCE($13, entity_relationship.confidence),
	updated_at = now()
RETURNING id;
`

	var relationshipID string
	err := s.db.QueryRowContext(
		ctx, query,
		tenantID, datasourceID,
		sourceEntityID, targetEntityID,
		rel.LinkType, rel.Cardinality, rel.HierarchyDepth,
		rel.ForeignKeyPath, rel.SourceColumn, rel.SourceTable, rel.TargetColumn, rel.TargetTable,
		rel.Confidence, rel.ConfidenceReason,
		isUserApplied, time.Now(),
		rel.DiscoveryMethod, true,
		rel.LinkReason,
	).Scan(&relationshipID)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to save relationship: %v", err)
		return "", fmt.Errorf("failed to save relationship: %w", err)
	}

	logging.GetLogger().Sugar().Infof("saved relationship %s -> %s as %s", sourceEntityID, targetEntityID, relationshipID)
	return relationshipID, nil
}

// ============================================================================
// JSON Serialization Support
// ============================================================================

// Value implements the sql.driver.Valuer interface for PathHop
func (ph PathHop) Value() (driver.Value, error) {
	return json.Marshal(ph)
}

// Scan implements the sql.Scanner interface for PathHop
func (ph *PathHop) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &ph)
}

// RelationshipPathSlice for database operations
type RelationshipPathSlice []PathHop

// Value implements the sql.driver.Valuer interface
func (rps RelationshipPathSlice) Value() (driver.Value, error) {
	return json.Marshal(rps)
}

// Scan implements the sql.Scanner interface
func (rps *RelationshipPathSlice) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &rps)
}

// ============================================================================
// Helper Functions
// ============================================================================

// CalculateConfidenceScore calculates the confidence score for a relationship
func CalculateConfidenceScore(
	fkExists, semanticLinked, namingMatch, columnTypeMatch bool,
) float64 {
	confidence := 0.0

	// Base scoring
	if fkExists {
		confidence = 0.95 // Strong indicator
	} else if semanticLinked {
		confidence = 0.85 // Good indicator
	} else if namingMatch {
		confidence = 0.70 // Moderate indicator
	} else {
		confidence = 0.50 // Weak base
	}

	// Boost if multiple signals align
	if semanticLinked && namingMatch {
		confidence = (confidence + 0.05)
		if confidence > 1.0 {
			confidence = 1.0
		}
	}

	if columnTypeMatch && fkExists {
		confidence = (confidence + 0.05)
		if confidence > 1.0 {
			confidence = 1.0
		}
	}

	return confidence
}

// StringArrayContains checks if a string slice contains a value
func StringArrayContains(arr pq.StringArray, value string) bool {
	for _, v := range arr {
		if strings.EqualFold(v, value) {
			return true
		}
	}
	return false
}
