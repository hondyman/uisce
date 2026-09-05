package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
)

// ForeignKeyColumn represents a single column pair in a FK relationship
type ForeignKeyColumn struct {
	SourceColumn string `json:"source_column"`
	TargetColumn string `json:"target_column"`
}

// ForeignKeyRelationship represents a single FK relationship between tables
type ForeignKeyRelationship struct {
	// FK identity
	EdgeID       string `json:"edge_id"`       // catalog_edge.id
	ConstraintID string `json:"constraint_id"` // FK constraint name

	// Source table
	SourceSchema string `json:"source_schema"`
	SourceTable  string `json:"source_table"`

	// Target table
	TargetSchema string `json:"target_schema"`
	TargetTable  string `json:"target_table"`

	// Column mapping
	Columns []ForeignKeyColumn `json:"columns"`

	// Cardinality and direction
	Direction    string `json:"direction"`     // "outbound" or "inbound"
	Cardinality  string `json:"cardinality"`   // "many-to-one", "one-to-many", "one-to-one"
	RelationType string `json:"relation_type"` // "reference", "composition", "association"

	// Lifecycle
	OnDelete  string    `json:"on_delete"`
	OnUpdate  string    `json:"on_update"`
	CreatedAt time.Time `json:"created_at"`
}

// EntityBackingTable represents the table(s) backing an entity
type EntityBackingTable struct {
	EntityID   string
	EntityName string
	TableName  string
	SchemaName string
	IsPrimary  bool // true if this is the main table (vs. joined table)
}

// EntityRelationshipFromFK represents a discovered relationship between two entities
type EntityRelationshipFromFK struct {
	// Source entity
	SourceEntityID   uuid.UUID `json:"source_entity_id"`
	SourceEntityName string    `json:"source_entity_name"`

	// Target entity
	TargetEntityID   uuid.UUID `json:"target_entity_id"`
	TargetEntityName string    `json:"target_entity_name"`

	// The FK that drove this discovery
	ForeignKey ForeignKeyRelationship `json:"foreign_key"`

	// Relationship properties
	Cardinality   string `json:"cardinality"`
	RelationType  string `json:"relation_type"`
	DiscoveryCode string `json:"discovery_code"` // "fk_outbound", "fk_inbound"

	// Confidence (FKs are definitive)
	Confidence float64 `json:"confidence"`

	// For edge creation
	EdgeProperties map[string]interface{} `json:"edge_properties,omitempty"`
}

// ForeignKeyDiscoveryEngine provides methods to discover entity relationships
// by analyzing foreign keys in the database catalog
type ForeignKeyDiscoveryEngine struct {
	db          *sql.DB
	cardinality *CardinalityResolver
}

// NewForeignKeyDiscoveryEngine creates a new FK discovery engine
func NewForeignKeyDiscoveryEngine(db *sql.DB) *ForeignKeyDiscoveryEngine {
	return &ForeignKeyDiscoveryEngine{db: db, cardinality: NewCardinalityResolver(db)}
}

// DiscoverForeignKeysForTable returns all foreign keys (inbound and outbound)
// for a given table
func (e *ForeignKeyDiscoveryEngine) DiscoverForeignKeysForTable(
	ctx context.Context,
	tenantID, datasourceID, tableName string,
) ([]ForeignKeyRelationship, error) {
	query := `
		WITH table_fks AS (
			-- Outbound FKs (this table is source)
			SELECT
				ce.id as edge_id,
				source_table.node_name as source_table,
				target_table.node_name as target_table,
				'outbound' as direction,
				ce.properties,
				ce.created_at
			FROM public.catalog_edge ce
			JOIN public.catalog_node source_table ON ce.source_node_id = source_table.id
			JOIN public.catalog_node target_table ON ce.target_node_id = target_table.id
			WHERE source_table.node_name = $1
			  AND ce.relationship_type = 'foreign_key'
			  AND ce.tenant_datasource_id = $2
			  AND source_table.node_type_id = (
				  SELECT id FROM public.node_type WHERE name = 'table'
			  )
			  AND target_table.node_type_id = (
				  SELECT id FROM public.node_type WHERE name = 'table'
			  )

			UNION ALL

			-- Inbound FKs (this table is target)
			SELECT
				ce.id as edge_id,
				source_table.node_name as source_table,
				target_table.node_name as target_table,
				'inbound' as direction,
				ce.properties,
				ce.created_at
			FROM public.catalog_edge ce
			JOIN public.catalog_node source_table ON ce.source_node_id = source_table.id
			JOIN public.catalog_node target_table ON ce.target_node_id = target_table.id
			WHERE target_table.node_name = $1
			  AND ce.relationship_type = 'foreign_key'
			  AND ce.tenant_datasource_id = $2
			  AND source_table.node_type_id = (
				  SELECT id FROM public.node_type WHERE name = 'table'
			  )
			  AND target_table.node_type_id = (
				  SELECT id FROM public.node_type WHERE name = 'table'
			  )
		)
		SELECT * FROM table_fks ORDER BY direction, source_table;
	`

	rows, err := e.db.QueryContext(ctx, query, tableName, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer rows.Close()

	var fks []ForeignKeyRelationship

	for rows.Next() {
		var (
			edgeID      string
			sourceTable string
			targetTable string
			direction   string
			propsJSON   []byte
			createdAt   time.Time
		)

		if err := rows.Scan(&edgeID, &sourceTable, &targetTable, &direction, &propsJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan FK row: %w", err)
		}

		// Parse properties
		var props map[string]interface{}
		if err := json.Unmarshal(propsJSON, &props); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to unmarshal FK properties: %v", err)
			props = make(map[string]interface{})
		}

		// Extract column mappings
		columns := e.extractColumnMappings(props)

		// Get constraint name
		constraintID := ""
		if constraints, ok := props["foreign_key_constraints"].([]interface{}); ok && len(constraints) > 0 {
			constraintID = fmt.Sprintf("%v", constraints[0])
		}

		// Resolve cardinality from real PK/unique constraints where the FK's
		// columns are known; fall back to the direction-based heuristic
		// otherwise (e.g. legacy edges whose properties never captured
		// column mappings).
		cardinality := e.resolveCardinality(ctx, sourceTable, targetTable, direction, columns)

		// Infer relationship type
		relType := e.inferRelationType(direction, string(cardinality))

		fk := ForeignKeyRelationship{
			EdgeID:       edgeID,
			ConstraintID: constraintID,
			SourceTable:  sourceTable,
			TargetTable:  targetTable,
			Columns:      columns,
			Direction:    direction,
			Cardinality:  string(cardinality),
			RelationType: relType,
			CreatedAt:    createdAt,
		}

		fks = append(fks, fk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating FK rows: %w", err)
	}

	return fks, nil
}

// DiscoverEntityRelationshipsFromFK discovers entity-to-entity relationships
// by analyzing foreign keys from an entity's backing table(s)
func (e *ForeignKeyDiscoveryEngine) DiscoverEntityRelationshipsFromFK(
	ctx context.Context,
	tenantID, datasourceID string,
	sourceEntity *Entity,
) ([]EntityRelationshipFromFK, error) {
	// Get the backing table(s) for the source entity
	backingTables, err := e.getEntityBackingTables(ctx, tenantID, datasourceID, sourceEntity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backing tables: %w", err)
	}

	if len(backingTables) == 0 {
		return nil, fmt.Errorf("entity %s has no backing tables", sourceEntity.ID)
	}

	var allRelationships []EntityRelationshipFromFK

	// Process each backing table (usually just one, but support multiple for flexibility)
	for _, backingTable := range backingTables {
		// Skip non-primary tables for now (could extend later)
		if !backingTable.IsPrimary {
			continue
		}

		// Discover FKs for this table
		fks, err := e.DiscoverForeignKeysForTable(ctx, tenantID, datasourceID, backingTable.TableName)
		if err != nil {
			logging.GetLogger().Sugar().Warnf(
				"Failed to discover FKs for table %s: %v", backingTable.TableName, err,
			)
			continue
		}

		// If this entity's backing table is itself an associative/junction
		// table (e.g. order_items between orders and products), also
		// synthesize a single MANY_TO_MANY relationship directly between
		// its two parent entities. The two raw FK relationships discovered
		// above (junction -> parentA, junction -> parentB) are kept as-is;
		// this is additive, not a replacement.
		if junctionRel, err := e.discoverJunctionRelationship(ctx, tenantID, datasourceID, backingTable.TableName); err != nil {
			logging.GetLogger().Sugar().Debugf(
				"junction detection skipped for %s: %v", backingTable.TableName, err,
			)
		} else if junctionRel != nil {
			allRelationships = append(allRelationships, *junctionRel)
		}

		// For each FK, find the target entity
		for _, fk := range fks {
			var targetTableName string
			var discoveryCode string

			if fk.Direction == "outbound" {
				// This entity references another entity
				targetTableName = fk.TargetTable
				discoveryCode = "fk_outbound"
			} else {
				// Another entity references this entity
				targetTableName = fk.SourceTable
				discoveryCode = "fk_inbound"
			}

			// Find the entity backed by targetTableName
			targetEntity, err := e.findEntityByBackingTable(ctx, tenantID, datasourceID, targetTableName)
			if err != nil {
				logging.GetLogger().Sugar().Debugf(
					"No entity found for table %s: %v", targetTableName, err,
				)
				continue
			}

			sourceEntityID, err := uuid.Parse(sourceEntity.ID)
			if err != nil {
				logging.GetLogger().Sugar().Warnf(
					"Failed to parse source entity ID %s: %v", sourceEntity.ID, err,
				)
				continue
			}
			targetEntityID, err := uuid.Parse(targetEntity.ID)
			if err != nil {
				logging.GetLogger().Sugar().Warnf(
					"Failed to parse target entity ID %s: %v", targetEntity.ID, err,
				)
				continue
			}
			// Create relationship pair
			rel := EntityRelationshipFromFK{
				SourceEntityID:   sourceEntityID,
				SourceEntityName: sourceEntity.Name,
				TargetEntityID:   targetEntityID,
				TargetEntityName: targetEntity.Name,
				ForeignKey:       fk,
				Cardinality:      fk.Cardinality,
				RelationType:     fk.RelationType,
				DiscoveryCode:    discoveryCode,
				Confidence:       1.0, // FKs are definitive
			}

			// Build edge properties
			rel.EdgeProperties = map[string]interface{}{
				"discovery_method": "foreign_key_analysis",
				"source_table":     fk.SourceTable,
				"target_table":     fk.TargetTable,
				"fk_constraint":    fk.ConstraintID,
				"fk_edge_id":       fk.EdgeID,
				"cardinality":      fk.Cardinality,
				"relation_type":    fk.RelationType,
				"discovery_code":   discoveryCode,
				"discovered_at":    time.Now().Format(time.RFC3339),
				"columns":          fk.Columns,
				"on_delete":        fk.OnDelete,
				"on_update":        fk.OnUpdate,
			}

			allRelationships = append(allRelationships, rel)
		}
	}

	return allRelationships, nil
}

// discoverJunctionRelationship checks whether tableName is an associative
// table (via CardinalityResolver.DetectJunctionTable) and, if both of its
// parent tables are backed by known entities, returns a single synthesized
// MANY_TO_MANY relationship between those two entities. Returns (nil, nil)
// when tableName isn't a junction table, or when one/both parent entities
// aren't discoverable (e.g. haven't been scanned/registered as entities).
func (e *ForeignKeyDiscoveryEngine) discoverJunctionRelationship(
	ctx context.Context,
	tenantID, datasourceID, tableName string,
) (*EntityRelationshipFromFK, error) {
	if e.cardinality == nil {
		return nil, nil
	}

	junction, err := e.cardinality.DetectJunctionTable(ctx, "public", tableName)
	if err != nil {
		return nil, fmt.Errorf("detecting junction table: %w", err)
	}
	if junction == nil {
		return nil, nil
	}

	parentAEntity, err := e.findEntityByBackingTable(ctx, tenantID, datasourceID, junction.ParentA)
	if err != nil {
		return nil, fmt.Errorf("no entity for junction parent %s: %w", junction.ParentA, err)
	}
	parentBEntity, err := e.findEntityByBackingTable(ctx, tenantID, datasourceID, junction.ParentB)
	if err != nil {
		return nil, fmt.Errorf("no entity for junction parent %s: %w", junction.ParentB, err)
	}

	sourceEntityID, err := uuid.Parse(parentAEntity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parent entity ID %s: %w", parentAEntity.ID, err)
	}
	targetEntityID, err := uuid.Parse(parentBEntity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parent entity ID %s: %w", parentBEntity.ID, err)
	}

	rel := &EntityRelationshipFromFK{
		SourceEntityID:   sourceEntityID,
		SourceEntityName: parentAEntity.Name,
		TargetEntityID:   targetEntityID,
		TargetEntityName: parentBEntity.Name,
		ForeignKey: ForeignKeyRelationship{
			SourceTable:  junction.ParentA,
			TargetTable:  junction.ParentB,
			Cardinality:  string(models.CardinalityManyToMany),
			RelationType: "association",
		},
		Cardinality:   string(models.CardinalityManyToMany),
		RelationType:  "association",
		DiscoveryCode: "fk_junction",
		Confidence:    1.0,
		EdgeProperties: map[string]interface{}{
			"discovery_method": "junction_table_analysis",
			"junction_table":   junction.TableName,
			"source_table":     junction.ParentA,
			"target_table":     junction.ParentB,
			"cardinality":      string(models.CardinalityManyToMany),
			"relation_type":    "association",
			"discovery_code":   "fk_junction",
			"discovered_at":    time.Now().Format(time.RFC3339),
		},
	}

	return rel, nil
}

// extractColumnMappings extracts column pairs from FK properties
func (e *ForeignKeyDiscoveryEngine) extractColumnMappings(props map[string]interface{}) []ForeignKeyColumn {
	var columns []ForeignKeyColumn

	// Try to get columns from 'columns' array
	if colsRaw, ok := props["columns"].([]interface{}); ok {
		for _, colRaw := range colsRaw {
			if colMap, ok := colRaw.(map[string]interface{}); ok {
				col := ForeignKeyColumn{
					SourceColumn: fmt.Sprintf("%v", colMap["source_column"]),
					TargetColumn: fmt.Sprintf("%v", colMap["target_column"]),
				}
				if col.SourceColumn != "" && col.TargetColumn != "" {
					columns = append(columns, col)
				}
			}
		}
	}

	// Fallback: try 'source_column' and 'target_column' direct properties
	if len(columns) == 0 {
		srcCol := fmt.Sprintf("%v", props["source_column"])
		tgtCol := fmt.Sprintf("%v", props["target_column"])
		if srcCol != "" && tgtCol != "" {
			columns = append(columns, ForeignKeyColumn{
				SourceColumn: srcCol,
				TargetColumn: tgtCol,
			})
		}
	}

	// Fallback: try 'foreign_key_target_column' pattern
	if len(columns) == 0 {
		tgtCol := fmt.Sprintf("%v", props["foreign_key_target_column"])
		if tgtCol != "" && tgtCol != "<nil>" {
			// Try to infer source column from common patterns
			srcCol := ""
			if targetTable, ok := props["foreign_key_target_table"].(string); ok {
				// Common pattern: foreign key is named like target_id
				srcCol = strings.ToLower(strings.TrimSuffix(targetTable, "s")) + "_id"
			}
			if srcCol != "" {
				columns = append(columns, ForeignKeyColumn{
					SourceColumn: srcCol,
					TargetColumn: tgtCol,
				})
			}
		}
	}

	return columns
}

// resolveCardinality computes the real cardinality of a "this table
// references source/target via columns" edge, taking the FK's direction
// into account so the result is always expressed from tableName's (the
// table DiscoverForeignKeysForTable was called for) perspective.
//
// direction "outbound" means tableName is the FK-holding (source) side, so
// CardinalityResolver's source/target map directly. direction "inbound"
// means tableName is the referenced (target) side, so the resolver's result
// must be inverted before returning it.
func (e *ForeignKeyDiscoveryEngine) resolveCardinality(
	ctx context.Context,
	sourceTable, targetTable, direction string,
	columns []ForeignKeyColumn,
) models.Cardinality {
	if e.cardinality == nil || len(columns) == 0 {
		return legacyInferCardinality(direction)
	}

	sourceCols := make([]string, len(columns))
	targetCols := make([]string, len(columns))
	for i, c := range columns {
		sourceCols[i] = c.SourceColumn
		targetCols[i] = c.TargetColumn
	}

	resolved, err := e.cardinality.ResolveEdgeCardinality(
		ctx, "public", sourceTable, sourceCols, "public", targetTable, targetCols,
	)
	if err != nil || resolved == models.CardinalityUnknown {
		logging.GetLogger().Sugar().Debugf(
			"falling back to direction-based cardinality for %s -> %s: %v", sourceTable, targetTable, err,
		)
		return legacyInferCardinality(direction)
	}

	// ResolveEdgeCardinality is expressed from the FK-holder's (outbound)
	// perspective. When this table is the inbound/referenced side, flip it.
	if direction == "inbound" {
		return resolved.Inverse()
	}
	return resolved
}

// legacyInferCardinality is the pre-existing direction-only heuristic,
// kept as a fallback for edges whose FK columns aren't known (so real
// constraint inspection isn't possible).
func legacyInferCardinality(direction string) models.Cardinality {
	if direction == "outbound" {
		return models.CardinalityManyToOne
	}
	if direction == "inbound" {
		return models.CardinalityOneToMany
	}
	return models.CardinalityUnknown
}

// inferRelationType infers the relationship type semantics
func (e *ForeignKeyDiscoveryEngine) inferRelationType(_ string, cardinality string) string {
	switch models.Cardinality(cardinality) {
	// Many-to-One = Reference (Child references Parent)
	case models.CardinalityManyToOne:
		return "reference"
	// One-to-Many = Composition (Parent owns/contains Children)
	case models.CardinalityOneToMany:
		return "composition"
	// One-to-One = Association
	case models.CardinalityOneToOne:
		return "association"
	// Many-to-Many = Association
	case models.CardinalityManyToMany:
		return "association"
	default:
		return "association"
	}
}

// getEntityBackingTables retrieves the table(s) that back an entity
func (e *ForeignKeyDiscoveryEngine) getEntityBackingTables(
	ctx context.Context, tenantID, _, entityID string,
) ([]EntityBackingTable, error) {
	// This query assumes your entities table has schema/table_name columns
	// Adjust as needed based on your actual schema
	query := `
		SELECT 
			e.id,
			e.name,
			COALESCE(e.table_name, '') as table_name,
			COALESCE(e.schema_name, 'public') as schema_name,
			true as is_primary
		FROM public.entities e
		WHERE e.id = $1
		  AND e.tenant_id = $2
		ORDER BY is_primary DESC;
	`

	rows, err := e.db.QueryContext(ctx, query, entityID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity backing tables: %w", err)
	}
	defer rows.Close()

	var tables []EntityBackingTable

	for rows.Next() {
		var (
			id         string
			name       string
			tableName  string
			schemaName string
			isPrimary  bool
		)

		if err := rows.Scan(&id, &name, &tableName, &schemaName, &isPrimary); err != nil {
			return nil, fmt.Errorf("failed to scan entity row: %w", err)
		}

		if tableName == "" {
			continue // Skip entities without table backing
		}

		tables = append(tables, EntityBackingTable{
			EntityID:   id,
			EntityName: name,
			TableName:  tableName,
			SchemaName: schemaName,
			IsPrimary:  isPrimary,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity rows: %w", err)
	}

	return tables, nil
}

// findEntityByBackingTable finds an entity that is backed by a specific table
func (e *ForeignKeyDiscoveryEngine) findEntityByBackingTable(
	ctx context.Context,
	tenantID, datasourceID, tableName string,
) (*Entity, error) {
	query := `
		SELECT 
			e.id,
			e.name,
			e.description,
			e.created_at
		FROM public.entities e
		WHERE (
			e.table_name = $1 OR
			LOWER(e.table_name) = LOWER($1)
		)
		  AND e.tenant_id = $2
		  AND e.tenant_datasource_id = $3
		LIMIT 1;
	`

	var (
		id          string
		name        string
		description sql.NullString
		createdAt   time.Time
	)

	err := e.db.QueryRowContext(ctx, query, tableName, tenantID, datasourceID).
		Scan(&id, &name, &description, &createdAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no entity found for table %s", tableName)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query entity: %w", err)
	}

	entity := &Entity{
		ID:        id,
		Name:      name,
		CreatedAt: createdAt,
	}

	if description.Valid {
		entity.Description = description.String
	}

	return entity, nil
}

// CreateEntityRelationshipEdgeFromFK creates an entity-to-entity relationship edge
// in catalog_edge based on FK discovery
func (e *ForeignKeyDiscoveryEngine) CreateEntityRelationshipEdgeFromFK(
	ctx context.Context,
	tenantID, datasourceID string,
	rel EntityRelationshipFromFK,
) (string, error) {
	edgeID := uuid.New().String()

	// Get the edge type ID for entity-to-entity relationships
	edgeTypeID, err := e.getEdgeTypeID(ctx, "entity_to_entity")
	if err != nil {
		return "", fmt.Errorf("failed to get edge type: %w", err)
	}

	// Marshal edge properties
	propsJSON, err := json.Marshal(rel.EdgeProperties)
	if err != nil {
		return "", fmt.Errorf("failed to marshal edge properties: %w", err)
	}

	query := `
		INSERT INTO public.catalog_edge (
			id, tenant_id, tenant_datasource_id, source_node_id, target_node_id,
			edge_type_id, relationship_type, properties, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
		DO UPDATE SET
			relationship_type = EXCLUDED.relationship_type,
			properties = EXCLUDED.properties,
			updated_at = EXCLUDED.updated_at
		RETURNING id;
	`

	var returnedID string
	err = e.db.QueryRowContext(
		ctx, query,
		edgeID,
		tenantID,
		datasourceID,
		rel.SourceEntityID,
		rel.TargetEntityID,
		edgeTypeID,
		"entity_relationship_fk",
		propsJSON,
		time.Now(),
		time.Now(),
	).Scan(&returnedID)

	if err != nil {
		return "", fmt.Errorf("failed to create relationship edge: %w", err)
	}

	logging.GetLogger().Sugar().Infof(
		"Created entity relationship edge: %s → %s (via FK: %s.%s)",
		rel.SourceEntityName, rel.TargetEntityName,
		rel.ForeignKey.SourceTable, rel.ForeignKey.TargetTable,
	)

	return returnedID, nil
}

// getEdgeTypeID retrieves the edge type ID for a given type name
func (e *ForeignKeyDiscoveryEngine) getEdgeTypeID(ctx context.Context, typeName string) (string, error) {
	query := `SELECT id FROM public.edge_type WHERE name = $1 LIMIT 1;`

	var id string
	err := e.db.QueryRowContext(ctx, query, typeName).Scan(&id)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("edge type '%s' not found", typeName)
	}

	if err != nil {
		return "", fmt.Errorf("failed to query edge type: %w", err)
	}

	return id, nil
}

// Entity is a minimal entity struct (adjust based on your actual Entity type)
type Entity struct {
	ID          string
	Name        string
	Description string
	TableName   string
	SchemaName  string
	CreatedAt   time.Time
}
