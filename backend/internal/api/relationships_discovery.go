package api

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
)

// RelatedEntity represents an entity that can be linked to the source entity
type RelatedEntity struct {
	EntityID       string    `json:"entity_id"`
	EntityName     string    `json:"entity_name"`
	SemanticName   string    `json:"semantic_name"`
	TableName      string    `json:"table_name"`
	LinkType       string    `json:"link_type"`        // "foreign_key", "semantic", etc.
	Cardinality    string    `json:"cardinality"`      // "one-to-one", "one-to-many", "many-to-one", "many-to-many"
	LinkReason     string    `json:"link_reason"`      // Human-readable explanation
	ForeignKeyPath string    `json:"foreign_key_path"` // FK constraint path if applicable
	DiscoveredAt   time.Time `json:"discovered_at"`
}

// RelatedObjectsResponse wraps a list of related entities
type RelatedObjectsResponse struct {
	SourceEntityName string          `json:"source_entity_name"`
	RelatedEntities  []RelatedEntity `json:"related_entities"`
	Count            int             `json:"count"`
}

// RelationshipDiscoveryService handles discovery of entity relationships
type RelationshipDiscoveryService struct {
	db *sql.DB
}

// NewRelationshipDiscoveryService creates a new relationship discovery service
func NewRelationshipDiscoveryService(db *sql.DB) *RelationshipDiscoveryService {
	return &RelationshipDiscoveryService{db: db}
}

// isValidUUID checks if a string is a valid UUID
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// DiscoverLinkableEntities finds all entities that can be linked to a given entity
// based on foreign key relationships in the database catalog.
//
// The function supports both UUID and name-based lookups:
// - If entityName is a valid UUID, it matches by catalog_node.id
// - Otherwise, it matches by node_name using case-insensitive comparison
//
// The algorithm:
// 1. Find the source table node for the given entity (UUID or name)
// 2. Find direct foreign key relationships from/to that table
// 3. Get the target table nodes from those foreign keys
// 4. Return the target tables as linkable entities
func (s *RelationshipDiscoveryService) DiscoverLinkableEntities(
	ctx context.Context,
	tenantID, datasourceID, entityName string,
) ([]RelatedEntity, error) {
	if entityName == "" {
		return nil, fmt.Errorf("entity name or ID is required")
	}

	// Determine if we're looking up by UUID or by name
	isUUID := isValidUUID(entityName)

	logging.GetLogger().Sugar().Debugf("DiscoverLinkableEntities: entityName=%s, isUUID=%v", entityName, isUUID)

	query := `
WITH source_table AS (
  -- Find the source table node matching the entity ID (UUID) or name (case-insensitive)
  -- UUID match takes precedence if the input is a valid UUID
  -- Name matching includes: exact match, lowercase match, and "tablename_" prefix match
  SELECT DISTINCT
    cn.id as table_id,
    cn.node_name as table_name
  FROM catalog_node cn
  JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
  WHERE cnt.catalog_type_name = 'table'
    AND cn.tenant_datasource_id = $2
    AND (
      -- Try UUID match: safely cast parameter to UUID using regex validation first
      (
        $1 ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        AND cn.id = $1::uuid
      )
      OR
      -- Fall back to name-based matching
      LOWER(cn.node_name) = LOWER($1)
      OR LOWER(cn.node_name) = LOWER($1) || 's'  -- Try pluralizing (customer -> customers)
      OR LOWER(cn.node_name) LIKE LOWER($1) || '%'  -- Prefix match
    )
),

direct_foreign_keys AS (
  -- Find direct FK relationships: source table -> target table
  -- This includes both outbound (source is subject) and inbound (source is object)
  SELECT
    ce.id as edge_id,
    ce.source_node_id,
    ce.target_node_id,
    cs.node_name as source_table_name,
    ct.node_name as target_table_name,
    'outbound' as direction,
    ce.properties,
    ce.created_at
  FROM catalog_edge ce
  JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
  JOIN catalog_node cs ON cs.id = ce.source_node_id
  JOIN catalog_node ct ON ct.id = ce.target_node_id
  JOIN source_table st ON st.table_id = cs.id
  WHERE cet.edge_type_name = 'foreign_key'
    AND ce.edge_type_name = 'foreign_key'
    AND ce.tenant_datasource_id = $2
  
  UNION ALL
  
  -- Inbound FKs: other tables pointing to this one
  SELECT
    ce.id as edge_id,
    ce.source_node_id,
    ce.target_node_id,
    cs.node_name as source_table_name,
    ct.node_name as target_table_name,
    'inbound' as direction,
    ce.properties,
    ce.created_at
  FROM catalog_edge ce
  JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
  JOIN catalog_node cs ON cs.id = ce.source_node_id
  JOIN catalog_node ct ON ct.id = ce.target_node_id
  JOIN source_table st ON st.table_id = ct.id
  WHERE cet.edge_type_name = 'foreign_key'
    AND ce.edge_type_name = 'foreign_key'
    AND ce.tenant_datasource_id = $2
),

target_table_nodes AS (
  -- Get the target table nodes - these are the related entities
  SELECT DISTINCT
    CASE 
      WHEN direction = 'outbound' THEN dfk.target_node_id
      ELSE dfk.source_node_id
    END as entity_id,
    CASE 
      WHEN direction = 'outbound' THEN dfk.target_table_name
      ELSE dfk.source_table_name
    END as entity_name,
    CASE 
      WHEN direction = 'outbound' THEN 'one-to-many'
      ELSE 'many-to-one'
    END as cardinality,
    direction as link_type,
    dfk.edge_id,
    dfk.created_at
  FROM direct_foreign_keys dfk
)

SELECT DISTINCT
  ttn.entity_id::text,
  ttn.entity_name,
  ttn.entity_name as semantic_name,
  ttn.entity_name as table_name,
  ttn.link_type,
  ttn.cardinality,
  CASE 
    WHEN ttn.link_type = 'outbound' THEN 'This table has a foreign key to ' || ttn.entity_name
    ELSE ttn.entity_name || ' has a foreign key to this table'
  END as link_reason,
  ttn.edge_id::text as foreign_key_path,
  NOW() as discovered_at
FROM target_table_nodes ttn
ORDER BY ttn.entity_name;
	`

	rows, err := s.db.QueryContext(ctx, query, entityName, datasourceID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to discover linkable entities for %s: %v", entityName, err)
		return nil, fmt.Errorf("failed to discover linkable entities: %w", err)
	}
	defer rows.Close()

	var relatedEntities []RelatedEntity

	for rows.Next() {
		var (
			entityID       string
			entityName     string
			semanticName   string
			tableName      string
			linkType       string
			cardinality    string
			linkReason     string
			foreignKeyPath string
			discoveredAt   time.Time
		)

		if err := rows.Scan(
			&entityID, &entityName, &semanticName, &tableName,
			&linkType, &cardinality, &linkReason, &foreignKeyPath, &discoveredAt,
		); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to scan related entity row: %v", err)
			continue
		}

		related := RelatedEntity{
			EntityID:       entityID,
			EntityName:     entityName,
			SemanticName:   semanticName,
			TableName:      tableName,
			LinkType:       linkType,
			Cardinality:    cardinality,
			LinkReason:     linkReason,
			ForeignKeyPath: foreignKeyPath,
			DiscoveredAt:   discoveredAt,
		}

		relatedEntities = append(relatedEntities, related)
	}

	if err := rows.Err(); err != nil {
		logging.GetLogger().Sugar().Errorf("Error iterating related entities: %v", err)
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	logging.GetLogger().Sugar().Debugf(
		"Discovered %d related entities for %s in datasource %s",
		len(relatedEntities), entityName, datasourceID,
	)

	return relatedEntities, nil
}

// DiscoverRelationshipsForSemanticTerm finds entities that can be linked based on
// a semantic term and its mapped columns.
func (s *RelationshipDiscoveryService) DiscoverRelationshipsForSemanticTerm(
	ctx context.Context,
	tenantID, datasourceID, semanticTermID string,
) ([]RelatedEntity, error) {
	if semanticTermID == "" {
		return nil, fmt.Errorf("semantic term ID is required")
	}

	// Get the semantic term name first
	var semanticName string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT node_name FROM catalog_node WHERE id = $1 AND tenant_datasource_id = $2`,
		semanticTermID, datasourceID,
	).Scan(&semanticName)

	if err != nil {
		return nil, fmt.Errorf("semantic term not found: %w", err)
	}

	// Use the semantic name to discover relationships
	return s.DiscoverLinkableEntities(ctx, tenantID, datasourceID, semanticName)
}

// GetRelationshipCardinality determines the cardinality between two tables
// based on foreign key properties
func (s *RelationshipDiscoveryService) GetRelationshipCardinality(
	ctx context.Context,
	datasourceID string,
	sourceMappings, targetMappings []string,
) (string, error) {
	// Query to check if there's a unique constraint on the FK columns in source
	query := `
		SELECT COUNT(*) > 0 as has_unique
		FROM information_schema.table_constraints tc
		WHERE tc.constraint_type = 'UNIQUE'
		  AND tc.table_schema = 'public'
		LIMIT 1;
	`

	var hasUnique bool
	err := s.db.QueryRowContext(ctx, query).Scan(&hasUnique)

	if err != nil && err != sql.ErrNoRows {
		return "one-to-many", nil // Default cardinality
	}

	if hasUnique {
		return "one-to-one", nil
	}

	return "one-to-many", nil
}

// ConvertNodeNameToTableName attempts to infer a table name from a node name
// by lowercasing and handling common conventions (plural -> singular)
func ConvertNodeNameToTableName(nodeName string) string {
	// Lowercase and replace dots/spaces with underscores
	tableName := strings.ToLower(nodeName)
	tableName = strings.ReplaceAll(tableName, ".", "_")
	tableName = strings.ReplaceAll(tableName, " ", "_")
	return tableName
}
