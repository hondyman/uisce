package scanner

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/cardinality"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/logging"
	cardinalitymodels "github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/models"
)

// Placeholder UUIDs for node and edge types (must match catalog_node_type and catalog_edge_types)
var (
	NODE_TYPE_SCHEMA      = uuid.MustParse("68d6d495-0992-4d92-ad2f-7f66dc1e7d78")
	NODE_TYPE_TABLE       = uuid.MustParse("49a50271-ae58-4d3e-ae1c-2f5b89d89192")
	NODE_TYPE_COLUMN      = uuid.MustParse("a64c1011-16e8-4ddf-b447-363bf8e15c9a")
	EDGE_TYPE_FOREIGN_KEY = uuid.MustParse("f21b4a8f-05af-43b9-92cd-061265ed54e0")
)

// AnsiScanner extracts metadata from an ANSI SQL compliant database
type AnsiScanner struct {
	sourceDB           *sql.DB
	tenantId           uuid.UUID
	tenantDatasourceId uuid.UUID
	sourceSystem       string
	nodes              []*models.CatalogNode // Using a slice of pointers
	edges              []models.CatalogEdge
	columnMap          map[uuid.UUID]*models.CatalogNode
	goldCopyNodes      map[string]db.GoldCopyNodeInfo
	isGoldCopy         bool
	schemaWhitelist    []string
	cardinality        *cardinality.Resolver
}

// NewAnsiScanner creates a new scanner instance
func NewAnsiScanner(db *sql.DB, tenantId, tenantDatasourceId uuid.UUID, sourceSystem string, goldCopyNodes map[string]db.GoldCopyNodeInfo, isGoldCopy bool, schemaWhitelist []string) (*AnsiScanner, error) {
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping source database: %w", err)
	}
	return &AnsiScanner{
		sourceDB:           db,
		tenantId:           tenantId,
		tenantDatasourceId: tenantDatasourceId,
		sourceSystem:       sourceSystem,
		nodes:              []*models.CatalogNode{}, // Initialize slice
		columnMap:          make(map[uuid.UUID]*models.CatalogNode),
		goldCopyNodes:      goldCopyNodes,
		isGoldCopy:         isGoldCopy,
		edges:              []models.CatalogEdge{}, // Initialize edges slice
		schemaWhitelist:    schemaWhitelist,
		cardinality:        cardinality.NewResolver(db),
	}, nil
}

// getCoreNode retrieves the gold copy node info if a match is found.
func (s *AnsiScanner) getCoreNode(nodeTypeID uuid.UUID, qualifiedPath string) (db.GoldCopyNodeInfo, bool) {
	if s.goldCopyNodes != nil {
		key := fmt.Sprintf("%s:%s", nodeTypeID.String(), qualifiedPath)
		if info, ok := s.goldCopyNodes[key]; ok {
			logging.GetLogger().Sugar().Debugf("Resolved core_id for %s -> %s", key, info.ID.String())
			return info, true
		}
		// Debug log when map exists but no mapping found for this key
		logging.GetLogger().Sugar().Debugf("No core_id mapping found for %s", key)
	}
	return db.GoldCopyNodeInfo{}, false
}

// isSameProperty compares two JSON property blobs, ignoring "is_core" and "source_system"
func (s *AnsiScanner) isSameProperty(p1, p2 json.RawMessage) bool {
	var m1, m2 map[string]interface{}
	if err := json.Unmarshal(p1, &m1); err != nil {
		return false
	}
	if err := json.Unmarshal(p2, &m2); err != nil {
		return false
	}

	// Remove keys that should be ignored in comparison
	ignoreKeys := []string{"is_core", "source_system", "created_at", "updated_at"}
	for _, key := range ignoreKeys {
		delete(m1, key)
		delete(m2, key)
	}

	b1, _ := json.Marshal(m1)
	b2, _ := json.Marshal(m2)
	return string(b1) == string(b2)
}

// processPrimaryKeys marks columns that are part of primary keys
func (s *AnsiScanner) processPrimaryKeys() error {
	query := `
        SELECT kcu.table_schema, kcu.table_name, kcu.column_name
        FROM information_schema.table_constraints AS tc
        JOIN information_schema.key_column_usage AS kcu
            ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
        WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
    `

	var args []interface{}
	if len(s.schemaWhitelist) > 0 {
		placeholders := make([]string, len(s.schemaWhitelist))
		for i, v := range s.schemaWhitelist {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, v)
		}
		query += fmt.Sprintf(" AND tc.table_schema IN (%s)", strings.Join(placeholders, ", "))
	}

	rows, err := s.sourceDB.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query primary keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, column string
		if err := rows.Scan(&schema, &table, &column); err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning primary key details: %v", err)
			continue
		}

		colAssetPath := fmt.Sprintf("/%s/%s/%s", schema, table, column)
		colID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_COLUMN.String(), colAssetPath)

		if col, ok := s.columnMap[colID]; ok {
			var propsMap map[string]interface{}
			if err := json.Unmarshal(col.Properties, &propsMap); err != nil {
				logging.GetLogger().Sugar().Warnf("Error unmarshaling properties for column %s: %v", col.QualifiedPath, err)
				continue
			}
			propsMap["is_primary_key"] = true
			propsJSON, err := json.Marshal(propsMap)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("Error marshaling updated properties for column %s: %v", col.QualifiedPath, err)
				continue
			}
			col.Properties = propsJSON
		}
	}
	return nil
}

// FINAL FIX: processForeignKeys - deduplicates by relationship, not constraint name
func (s *AnsiScanner) processForeignKeys() error {
	query := `
        SELECT DISTINCT
            rc.constraint_name,
            rc.constraint_schema,
            kcu.table_schema AS source_schema,
            kcu.table_name AS source_table,
            kcu.column_name AS source_column,
            pku.table_schema AS target_schema,
            pku.table_name AS target_table,
            pku.column_name AS target_column,
            rc.update_rule AS on_update,
            rc.delete_rule AS on_delete,
            tc.is_deferrable,
            tc.initially_deferred,
            kcu.ordinal_position
        FROM
            information_schema.referential_constraints AS rc
        JOIN
            information_schema.table_constraints AS tc
                ON rc.constraint_name = tc.constraint_name
                AND rc.constraint_schema = tc.constraint_schema
        JOIN
            information_schema.key_column_usage AS kcu
                ON rc.constraint_name = kcu.constraint_name
                AND rc.constraint_schema = kcu.constraint_schema
        JOIN
            information_schema.key_column_usage AS pku
                ON rc.unique_constraint_name = pku.constraint_name
                AND rc.unique_constraint_schema = pku.constraint_schema
                AND kcu.ordinal_position = pku.ordinal_position
        WHERE
            kcu.table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
    `

	var args []interface{}
	if len(s.schemaWhitelist) > 0 {
		placeholders := make([]string, len(s.schemaWhitelist))
		for i, v := range s.schemaWhitelist {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, v)
		}
		// Apply filter to both source (kcu) and target (pku) schemas to be safe,
		// though typically we only care about edges where at least one side is in our whitelist.
		// For now, let's restrict edges where the SOURCE table is in our whitelist.
		query += fmt.Sprintf(" AND kcu.table_schema IN (%s)", strings.Join(placeholders, ", "))
	}

	query += `
        ORDER BY
            rc.constraint_schema, rc.constraint_name, kcu.ordinal_position;
    `

	logging.GetLogger().Sugar().Infof("Querying foreign keys...")
	rows, err := s.sourceDB.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer rows.Close()

	// Key insight: We need to deduplicate by RELATIONSHIP, not constraint name
	// Multiple constraints can define the same relationship (like your example)
	// So we group by (source_table, target_table, columns) instead of constraint name

	type relationshipKey struct {
		sourceTableID uuid.UUID
		targetTableID uuid.UUID
		columnsHash   string // Hash of sorted column pairs
	}

	type relationshipData struct {
		sourceSchema      string
		sourceTable       string
		targetSchema      string
		targetTable       string
		sourceTableID     uuid.UUID
		targetTableID     uuid.UUID
		constraints       []string // List of constraint names for this relationship
		onDelete          string
		onUpdate          string
		isDeferrable      string
		initiallyDeferred string
		columns           []map[string]interface{}
	}

	// First, collect all constraint data by constraint name
	constraintData := make(map[string][]struct {
		sourceSchema      string
		sourceTable       string
		sourceColumn      string
		targetSchema      string
		targetTable       string
		targetColumn      string
		onDelete          string
		onUpdate          string
		isDeferrable      string
		initiallyDeferred string
		ordinalPosition   int
	})

	rowCount := 0
	for rows.Next() {
		var constraintName, constraintSchema, sourceSchema, sourceTable, sourceColumn string
		var targetSchema, targetTable, targetColumn, onDelete, onUpdate, isDeferrable, initiallyDeferred string
		var ordinalPosition int

		if err := rows.Scan(&constraintName, &constraintSchema, &sourceSchema, &sourceTable, &sourceColumn,
			&targetSchema, &targetTable, &targetColumn, &onUpdate, &onDelete,
			&isDeferrable, &initiallyDeferred, &ordinalPosition); err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning foreign key details: %v", err)
			continue
		}

		rowCount++
		fullConstraintName := fmt.Sprintf("%s.%s", constraintSchema, constraintName)

		constraintData[fullConstraintName] = append(constraintData[fullConstraintName], struct {
			sourceSchema      string
			sourceTable       string
			sourceColumn      string
			targetSchema      string
			targetTable       string
			targetColumn      string
			onDelete          string
			onUpdate          string
			isDeferrable      string
			initiallyDeferred string
			ordinalPosition   int
		}{
			sourceSchema, sourceTable, sourceColumn, targetSchema, targetTable, targetColumn,
			onDelete, onUpdate, isDeferrable, initiallyDeferred, ordinalPosition,
		})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error processing foreign key rows: %w", err)
	}

	logging.GetLogger().Sugar().Infof("Processed %d FK rows into %d constraints", rowCount, len(constraintData))

	// Now group constraints by relationship (same table pair + same columns)
	relationships := make(map[relationshipKey]*relationshipData)

	for constraintName, rows := range constraintData {
		if len(rows) == 0 {
			continue
		}

		firstRow := rows[0]

		// Generate table IDs
		sourceAssetPath := fmt.Sprintf("/%s/%s", firstRow.sourceSchema, firstRow.sourceTable)
		sourceTableID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_TABLE.String(), sourceAssetPath)

		targetAssetPath := fmt.Sprintf("/%s/%s", firstRow.targetSchema, firstRow.targetTable)
		targetTableID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_TABLE.String(), targetAssetPath)

		// Create a hash of the column mappings to identify identical relationships
		var columnPairs []string
		var columns []map[string]interface{}

		for _, row := range rows {
			// Mark source columns as foreign keys
			sourceColAssetPath := fmt.Sprintf("/%s/%s/%s", row.sourceSchema, row.sourceTable, row.sourceColumn)
			sourceColID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_COLUMN.String(), sourceColAssetPath)

			if col, ok := s.columnMap[sourceColID]; ok {
				var propsMap map[string]interface{}
				if err := json.Unmarshal(col.Properties, &propsMap); err == nil {
					propsMap["is_foreign_key"] = true
					propsMap["foreign_key_constraints"] = []string{constraintName} // Note: might be overwritten
					propsMap["foreign_key_target_table"] = fmt.Sprintf("%s.%s", row.targetSchema, row.targetTable)
					propsMap["foreign_key_target_column"] = row.targetColumn
					propsJSON, _ := json.Marshal(propsMap)
					col.Properties = propsJSON
				}
			}

			targetColAssetPath := fmt.Sprintf("/%s/%s/%s", row.targetSchema, row.targetTable, row.targetColumn)
			targetColID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_COLUMN.String(), targetColAssetPath)

			columnPair := fmt.Sprintf("%s:%s->%s:%s",
				row.sourceColumn, sourceColID.String(),
				row.targetColumn, targetColID.String())
			columnPairs = append(columnPairs, columnPair)

			columnInfo := map[string]interface{}{
				"source_column":    row.sourceColumn,
				"source_column_id": sourceColID.String(),
				"target_column":    row.targetColumn,
				"target_column_id": targetColID.String(),
				"ordinal_position": row.ordinalPosition,
			}
			columns = append(columns, columnInfo)
		}

		// Sort column pairs to create consistent hash
		sort.Strings(columnPairs)
		columnsHash := generateID(strings.Join(columnPairs, "|")).String()

		// Create relationship key
		relKey := relationshipKey{
			sourceTableID: sourceTableID,
			targetTableID: targetTableID,
			columnsHash:   columnsHash,
		}

		// Check if we've seen this relationship before
		if existing, exists := relationships[relKey]; exists {
			// Same relationship exists - add this constraint name to the list
			existing.constraints = append(existing.constraints, constraintName)
			logging.GetLogger().Sugar().Infof("Found duplicate relationship: %s (constraint %s) - merging with existing",
				constraintName, constraintName)
		} else {
			// New relationship
			relationships[relKey] = &relationshipData{
				sourceSchema:      firstRow.sourceSchema,
				sourceTable:       firstRow.sourceTable,
				targetSchema:      firstRow.targetSchema,
				targetTable:       firstRow.targetTable,
				sourceTableID:     sourceTableID,
				targetTableID:     targetTableID,
				constraints:       []string{constraintName},
				onDelete:          firstRow.onDelete,
				onUpdate:          firstRow.onUpdate,
				isDeferrable:      firstRow.isDeferrable,
				initiallyDeferred: firstRow.initiallyDeferred,
				columns:           columns,
			}
			logging.GetLogger().Sugar().Infof("New relationship: %s.%s -> %s.%s (%d columns, constraint %s)",
				firstRow.sourceSchema, firstRow.sourceTable,
				firstRow.targetSchema, firstRow.targetTable,
				len(columns), constraintName)
		}
	}

	logging.GetLogger().Sugar().Infof("Grouped %d constraints into %d unique relationships", len(constraintData), len(relationships))

	// Create exactly one edge per unique relationship
	edgesBefore := len(s.edges)

	for relKey, rel := range relationships {
		// Use the first constraint name for the edge, but list all in properties
		primaryConstraint := rel.constraints[0]

		// Resolve real cardinality from PK/unique constraints (composite-key
		// aware), rather than guessing from column count/direction alone —
		// see inferFKCardinality's doc comment for why that heuristic is
		// wrong for 1:1 and can't detect N:M at all.
		relCardinality := s.resolveFKCardinality(rel.sourceSchema, rel.sourceTable, rel.targetSchema, rel.targetTable, rel.columns)

		// Build properties with all constraint names
		props := map[string]interface{}{
			"primary_constraint_name": primaryConstraint,
			"all_constraint_names":    rel.constraints,
			"constraint_count":        len(rel.constraints),
			"columns":                 rel.columns,
			"column_count":            len(rel.columns),
			"on_delete":               rel.onDelete,
			"on_update":               rel.onUpdate,
			"is_deferrable":           rel.isDeferrable,
			"initially_deferred":      rel.initiallyDeferred,
			// Enhanced for semantic discovery
			"edge_type_name": "foreign_key",
			"cardinality":    relCardinality,
			"source_table":   rel.sourceTable,
			"target_table":   rel.targetTable,
			"source_schema":  rel.sourceSchema,
			"target_schema":  rel.targetSchema,
		}

		propsJSON, err := json.Marshal(props)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Error marshaling properties for relationship: %v", err)
			continue
		}

		// Generate edge ID using the relationship key for true uniqueness
		edgeID := generateID(
			s.tenantDatasourceId.String(),
			"fk_relationship",
			relKey.sourceTableID.String(),
			relKey.targetTableID.String(),
			relKey.columnsHash,
		)

		edge := models.CatalogEdge{
			ID:                 edgeID,
			CoreID:             uuid.NullUUID{Valid: false},
			TenantID:           s.tenantId,
			TenantDatasourceId: s.tenantDatasourceId,
			SourceNodeID:       rel.sourceTableID,
			TargetNodeID:       rel.targetTableID,
			Properties:         propsJSON,
			CreatedAt:          time.Now(),
			EdgeTypeID:         EDGE_TYPE_FOREIGN_KEY,
			EdgeTypeName:       "foreign_key",
		}

		// Skip self-referencing edges (same source and target table)
		if rel.sourceTableID == rel.targetTableID {
			logging.GetLogger().Sugar().Warnf("Skipping self-referencing edge: %s.%s -> %s.%s",
				rel.sourceSchema, rel.sourceTable, rel.targetSchema, rel.targetTable)
			continue
		}

		s.edges = append(s.edges, edge)

		constraintNames := strings.Join(rel.constraints, ", ")
		logging.GetLogger().Sugar().Infof("Created edge: %s.%s -> %s.%s (%d cols, %d constraints: %s)",
			rel.sourceSchema, rel.sourceTable,
			rel.targetSchema, rel.targetTable,
			len(rel.columns), len(rel.constraints), constraintNames)
	}

	edgesCreated := len(s.edges) - edgesBefore
	logging.GetLogger().Sugar().Infof("Created %d unique foreign key relationship edges", edgesCreated)

	var sourceTables []scannedTableRef
	seenSourceTables := map[scannedTableRef]bool{}
	for _, rel := range relationships {
		ref := scannedTableRef{schema: rel.sourceSchema, table: rel.sourceTable}
		if !seenSourceTables[ref] {
			seenSourceTables[ref] = true
			sourceTables = append(sourceTables, ref)
		}
	}
	s.detectAndAppendJunctionEdges(sourceTables)

	return nil
}

// scannedTableRef identifies a table by schema+name within a single scan.
type scannedTableRef struct{ schema, table string }

// detectAndAppendJunctionEdges checks every distinct source table that held
// at least one FK in this scan for the associative/junction-table pattern
// (e.g. order_items between orders and products: two FKs whose combined
// columns are covered by the table's own PK/unique constraint), and — when
// found — appends one additional synthesized MANY_TO_MANY edge directly
// between the two parent tables. This is additive: the two raw FK edges
// (junction -> parentA, junction -> parentB) created above are unaffected.
//
// Without this, a plain FK-direction scan can never produce a true N:M
// relationship: two MANY_TO_ONE edges are structurally correct on their
// own, but nothing else stands for "these two tables are related many rows
// to many rows through this junction" the way page/report/query designers
// need to decide between a reference control and an embedded grid.
func (s *AnsiScanner) detectAndAppendJunctionEdges(sourceTables []scannedTableRef) {
	if s.cardinality == nil {
		return
	}

	ctx := context.Background()
	for _, t := range sourceTables {
		junction, err := s.cardinality.DetectJunctionTable(ctx, t.schema, t.table)
		if err != nil {
			logging.GetLogger().Sugar().Debugf("junction detection failed for %s.%s: %v", t.schema, t.table, err)
			continue
		}
		if junction == nil {
			continue
		}

		parentAPath := fmt.Sprintf("/%s/%s", t.schema, junction.ParentA)
		parentATableID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_TABLE.String(), parentAPath)
		parentBPath := fmt.Sprintf("/%s/%s", t.schema, junction.ParentB)
		parentBTableID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_TABLE.String(), parentBPath)

		if parentATableID == parentBTableID {
			continue
		}

		props := map[string]interface{}{
			"edge_type_name": "many_to_many_junction",
			"cardinality":    string(cardinalitymodels.CardinalityManyToMany),
			"junction_table": junction.TableName,
			"source_table":   junction.ParentA,
			"target_table":   junction.ParentB,
			"source_schema":  t.schema,
			"target_schema":  t.schema,
		}
		propsJSON, err := json.Marshal(props)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Error marshaling junction edge properties for %s: %v", junction.TableName, err)
			continue
		}

		edgeID := generateID(
			s.tenantDatasourceId.String(),
			"many_to_many_junction",
			parentATableID.String(),
			parentBTableID.String(),
			junction.TableName,
		)

		s.edges = append(s.edges, models.CatalogEdge{
			ID:                 edgeID,
			CoreID:             uuid.NullUUID{Valid: false},
			TenantID:           s.tenantId,
			TenantDatasourceId: s.tenantDatasourceId,
			SourceNodeID:       parentATableID,
			TargetNodeID:       parentBTableID,
			Properties:         propsJSON,
			CreatedAt:          time.Now(),
			EdgeTypeID:         EDGE_TYPE_FOREIGN_KEY,
			EdgeTypeName:       "many_to_many_junction",
		})

		logging.GetLogger().Sugar().Infof(
			"Detected junction table %s.%s: synthesized MANY_TO_MANY edge %s <-> %s",
			t.schema, junction.TableName, junction.ParentA, junction.ParentB,
		)

		// Tag the two raw FK edges (junction -> parentA, junction -> parentB)
		// the main relationship loop already created, so consumers can tell
		// a raw junction-side edge apart from an ordinary FK edge, and know
		// which synthesized M:M edge it rolls up into — without this, the
		// two forms are indistinguishable in catalog_edge.
		junctionPath := fmt.Sprintf("/%s/%s", t.schema, t.table)
		junctionTableID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_TABLE.String(), junctionPath)
		s.tagRawJunctionEdges(junctionTableID, junction.TableName)
	}
}

// tagRawJunctionEdges finds the raw FK edges whose source is junctionTableID
// (i.e. edges from the junction table to one of its parents, created by the
// main relationship loop in processForeignKeys) and adds a "junction_table"
// property to each, so a consumer reading catalog_edge can distinguish a
// junction-side raw edge from an ordinary FK edge and associate it with the
// synthesized MANY_TO_MANY edge detectAndAppendJunctionEdges appends.
func (s *AnsiScanner) tagRawJunctionEdges(junctionTableID uuid.UUID, junctionTableName string) {
	for i := range s.edges {
		if s.edges[i].SourceNodeID != junctionTableID {
			continue
		}
		var props map[string]interface{}
		if err := json.Unmarshal(s.edges[i].Properties, &props); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to unmarshal properties while tagging junction edge: %v", err)
			continue
		}
		props["junction_table"] = junctionTableName
		propsJSON, err := json.Marshal(props)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to re-marshal properties while tagging junction edge: %v", err)
			continue
		}
		s.edges[i].Properties = propsJSON
	}
}

// resolveFKCardinality determines the real cardinality of a foreign-key
// relationship by inspecting whether the source/target columns are covered
// by a PRIMARY KEY/UNIQUE constraint (via s.cardinality, backed by the same
// live connection this scan is reading from), returning the DB-canonical
// wire value (e.g. "MANY_TO_ONE") that downstream readers normalize via
// models.ParseCardinality. Falls back to the direction-only heuristic
// (inferFKCardinality) when columns are missing or resolution fails, so a
// scan never aborts over a single relationship's cardinality lookup.
func (s *AnsiScanner) resolveFKCardinality(sourceSchema, sourceTable, targetSchema, targetTable string, columns []map[string]interface{}) string {
	if s.cardinality == nil || len(columns) == 0 {
		return s.inferFKCardinality(len(columns), sourceSchema, sourceTable)
	}

	var sourceCols, targetCols []string
	for _, c := range columns {
		src, _ := c["source_column"].(string)
		tgt, _ := c["target_column"].(string)
		if src == "" || tgt == "" {
			continue
		}
		sourceCols = append(sourceCols, src)
		targetCols = append(targetCols, tgt)
	}
	if len(sourceCols) == 0 {
		return s.inferFKCardinality(len(columns), sourceSchema, sourceTable)
	}

	resolved, err := s.cardinality.ResolveEdgeCardinality(
		context.Background(), sourceSchema, sourceTable, sourceCols, targetSchema, targetTable, targetCols,
	)
	if err != nil || resolved == cardinalitymodels.CardinalityUnknown {
		logging.GetLogger().Sugar().Debugf(
			"falling back to heuristic cardinality for %s.%s -> %s.%s: %v",
			sourceSchema, sourceTable, targetSchema, targetTable, err,
		)
		return s.inferFKCardinality(len(columns), sourceSchema, sourceTable)
	}

	return string(resolved)
}

// inferFKCardinality is the pre-existing direction-only heuristic, kept as
// a fallback for relationships whose FK columns aren't known (so real
// constraint inspection isn't possible). It cannot distinguish 1:1 or
// detect N:M — see resolveFKCardinality, which is the primary path.
func (s *AnsiScanner) inferFKCardinality(columnCount int, sourceSchema, sourceTable string) string {
	return "N:1" // Foreign key: many-to-one
}

// generateID creates a deterministic UUID from a set of strings
func generateID(parts ...string) uuid.UUID {
	var combined string
	for _, p := range parts {
		combined += p
	}
	namespace := uuid.MustParse("1b671a64-40d5-491e-99b0-da01ff1f3341")
	return uuid.NewSHA1(namespace, []byte(combined))
}

// processSchema handles a single schema
func (s *AnsiScanner) processSchema(schemaName string) error {
	qualifiedPath := fmt.Sprintf("/%s", schemaName)
	schemaID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_SCHEMA.String(), qualifiedPath)
	coreNode, hasCore := s.getCoreNode(NODE_TYPE_SCHEMA, qualifiedPath)

	isCore := s.isGoldCopy
	coreID := uuid.NullUUID{Valid: false}
	if hasCore {
		coreID = uuid.NullUUID{UUID: coreNode.ID, Valid: true}
	}

	props := map[string]interface{}{
		"is_core":       isCore,
		"source_system": s.sourceSystem,
	}
	propsJSON, _ := json.Marshal(props)

	// DELTA LOGIC: If not gold copy, and we have a core node that matches perfectly, we skip storing it
	if !s.isGoldCopy && hasCore {
		if s.isSameProperty(propsJSON, coreNode.Properties) {
			logging.GetLogger().Sugar().Debugf("Schema %s matches gold copy perfectly; skipping local storage", schemaName)
			return s.processTables(schemaName, schemaID) // Still process children
		}
	}

	schemaNode := &models.CatalogNode{
		ID:                 schemaID,
		CoreID:             coreID,
		TenantID:           s.tenantId,
		TenantDatasourceId: s.tenantDatasourceId,
		NodeTypeID:         NODE_TYPE_SCHEMA,
		NodeName:           schemaName,
		QualifiedPath:      qualifiedPath,
		Properties:         propsJSON,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	s.nodes = append(s.nodes, schemaNode)
	return s.processTables(schemaName, schemaID)
}

// processTables handles tables within a schema
func (s *AnsiScanner) processTables(schemaName string, schemaID uuid.UUID) error {
	query := `SELECT table_name FROM information_schema.tables WHERE table_schema = $1 AND table_type = 'BASE TABLE'`
	rows, err := s.sourceDB.Query(query, schemaName)
	if err != nil {
		return fmt.Errorf("failed to query tables for schema %s: %w", schemaName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning table name: %v", err)
			continue
		}

		assetPath := fmt.Sprintf("/%s/%s", schemaName, tableName)
		tableID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_TABLE.String(), assetPath)
		coreNode, hasCore := s.getCoreNode(NODE_TYPE_TABLE, assetPath)

		isCore := s.isGoldCopy
		coreID := uuid.NullUUID{Valid: false}
		if hasCore {
			coreID = uuid.NullUUID{UUID: coreNode.ID, Valid: true}
		}

		tableProps := map[string]interface{}{
			"schema":        schemaName,
			"is_core":       isCore,
			"source_system": s.sourceSystem,
		}
		propsJSON, _ := json.Marshal(tableProps)

		// DELTA LOGIC
		skipTable := false
		if !s.isGoldCopy && hasCore {
			if s.isSameProperty(propsJSON, coreNode.Properties) {
				logging.GetLogger().Sugar().Debugf("Table %s matches gold copy perfectly; skipping local node storage", assetPath)
				skipTable = true
			}
		}

		if !skipTable {
			tableNode := &models.CatalogNode{
				ID:                 tableID,
				CoreID:             coreID,
				TenantID:           s.tenantId,
				TenantDatasourceId: s.tenantDatasourceId,
				NodeTypeID:         NODE_TYPE_TABLE,
				NodeName:           tableName,
				QualifiedPath:      assetPath,
				ParentID:           uuid.NullUUID{UUID: schemaID, Valid: true},
				Properties:         propsJSON,
				CreatedAt:          time.Now(),
				IsAlpha:            false,
				UpdatedAt:          time.Now(),
			}
			s.nodes = append(s.nodes, tableNode)
		}

		if err := s.processColumns(schemaName, tableName, tableID); err != nil {
			logging.GetLogger().Sugar().Warnf("Error processing columns for table %s.%s: %v", schemaName, tableName, err)
		}
	}
	return nil
}

// Add this debug logging to your processColumns function to verify parent_id relationships
func (s *AnsiScanner) processColumns(schemaName, tableName string, tableID uuid.UUID) error {
	// Add debug logging for table ID
	logging.GetLogger().Sugar().Debugf("Processing columns for table %s.%s with tableID: %s", schemaName, tableName, tableID.String())

	query := `
        SELECT 
			c.column_name, c.data_type, c.is_nullable = 'YES' AS is_nullable, c.column_default, 
            c.character_maximum_length, c.numeric_precision, c.numeric_scale, c.ordinal_position,
			COALESCE(pgd.description, '') AS column_comment
        FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_statio_all_tables st 
			ON c.table_schema = st.schemaname AND c.table_name = st.relname
		LEFT JOIN pg_catalog.pg_description pgd 
			ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
        WHERE c.table_schema = $1 AND c.table_name = $2
        ORDER BY c.ordinal_position
    `
	rows, err := s.sourceDB.Query(query, schemaName, tableName)
	if err != nil {
		return fmt.Errorf("failed to query columns for table %s.%s: %w", schemaName, tableName, err)
	}
	defer rows.Close()

	columnCount := 0
	for rows.Next() {
		var colName, dataType string
		var isNullable bool
		var columnDefault, columnComment sql.NullString
		var maxLength, precision, scale, ordinalPosition sql.NullInt32

		if err := rows.Scan(&colName, &dataType, &isNullable, &columnDefault, &maxLength, &precision, &scale, &ordinalPosition, &columnComment); err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning column details: %v", err)
			continue
		}

		assetPath := fmt.Sprintf("/%s/%s/%s", schemaName, tableName, colName)
		colID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_COLUMN.String(), assetPath)
		coreNode, hasCore := s.getCoreNode(NODE_TYPE_COLUMN, assetPath)

		isCore := s.isGoldCopy
		coreID := uuid.NullUUID{Valid: false}
		if hasCore {
			coreID = uuid.NullUUID{UUID: coreNode.ID, Valid: true}
		}

		props := map[string]interface{}{"data_type": dataType, "is_nullable": isNullable}
		if columnDefault.Valid {
			props["default_value"] = columnDefault.String
		}
		if columnComment.Valid && columnComment.String != "" {
			props["column_comment"] = columnComment.String
		}
		if maxLength.Valid {
			props["max_length"] = maxLength.Int32
		}
		if precision.Valid {
			props["precision"] = precision.Int32
		}
		if scale.Valid {
			props["scale"] = scale.Int32
		}
		if ordinalPosition.Valid {
			props["ordinal_position"] = ordinalPosition.Int32
		}
		props["is_core"] = isCore
		props["is_physical_column"] = true
		props["parent_id"] = tableID.String()

		// GENERATE TITLE AND TITLE_SHORT
		// title_short: Title case of column name (abbreviations intact)
		// title: Title case of expanded column name (abbreviations expanded)
		props["title_short"] = toTitleCase(colName)
		props["title"] = toTitleCase(colName) // Will be updated by abbreviation expansion in wizard

		// AUTO-GENERATE VALIDATION RULES
		// If column is NOT NULL, suggest a "isNotEmpty" validation rule
		if !isNullable {
			rule := map[string]interface{}{
				"type":     "condition",
				"field":    colName,
				"operator": "isNotEmpty",
				"severity": "error",
			}
			// Store as a single object or array? The requirement implies specific JSON structure.
			// "Query information_schema... to create validationrules entries. Store as conditionjson..."
			// We store it in properties for now so the wizard can pick it up.
			props["suggested_validation_rules"] = rule
		}

		propsJSON, _ := json.Marshal(props)

		// DELTA LOGIC
		if !s.isGoldCopy && hasCore {
			if s.isSameProperty(propsJSON, coreNode.Properties) {
				logging.GetLogger().Sugar().Debugf("Column %s matches gold copy perfectly; skipping local storage", assetPath)
				continue
			}
		}

		colNode := &models.CatalogNode{
			ID:                 colID,
			CoreID:             coreID,
			TenantID:           s.tenantId,
			TenantDatasourceId: s.tenantDatasourceId,
			NodeTypeID:         NODE_TYPE_COLUMN,
			NodeName:           colName,
			QualifiedPath:      assetPath,
			ParentID:           uuid.NullUUID{UUID: tableID, Valid: true},
			Properties:         propsJSON,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
			IsAlpha:            false,
		}

		s.nodes = append(s.nodes, colNode)
		s.columnMap[colID] = colNode
		columnCount++
	}

	logging.GetLogger().Sugar().Debugf("Created %d columns for table %s.%s (tableID: %s)",
		columnCount, schemaName, tableName, tableID.String())

	return nil
}

// Add these functions to your existing scanner file (don't duplicate ExtractMetadata)

// validateIDConsistency validates that ID generation is deterministic
func (s *AnsiScanner) validateIDConsistency() {
	logging.GetLogger().Sugar().Info("=== ID GENERATION VALIDATION ===")

	// Test table ID generation
	testSchema := "public"
	testTable := "customers"
	testColumn := "id"

	tableAssetPath := fmt.Sprintf("/%s/%s", testSchema, testTable)
	columnAssetPath := fmt.Sprintf("/%s/%s/%s", testSchema, testTable, testColumn)

	tableID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_TABLE.String(), tableAssetPath)
	columnID := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_COLUMN.String(), columnAssetPath)

	logging.GetLogger().Sugar().Infof("TEST: tenant_datasource_id: %s", s.tenantDatasourceId.String())
	logging.GetLogger().Sugar().Infof("TEST: sourceSystem: %s", s.sourceSystem)
	logging.GetLogger().Sugar().Infof("TEST: table asset path: %s", tableAssetPath)
	logging.GetLogger().Sugar().Infof("TEST: column asset path: %s", columnAssetPath)
	logging.GetLogger().Sugar().Infof("TEST: generated table ID: %s", tableID.String())
	logging.GetLogger().Sugar().Infof("TEST: generated column ID: %s", columnID.String())

	// Verify the same ID generation produces same results
	tableID2 := generateID(s.tenantDatasourceId.String(), s.sourceSystem, NODE_TYPE_TABLE.String(), tableAssetPath)
	if tableID != tableID2 {
		logging.GetLogger().Sugar().Error("ID generation is not deterministic!")
	} else {
		logging.GetLogger().Sugar().Info("SUCCESS: ID generation is deterministic")
	}
}

// Call this validation function at the start of ExtractMetadata
func (s *AnsiScanner) ExtractMetadata() ([]*models.CatalogNode, []models.CatalogEdge, error) {
	defer s.sourceDB.Close()
	s.validateIDConsistency()

	// Add ID validation
	s.validateIDConsistency()

	query := `
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
    `

	var args []interface{}
	if len(s.schemaWhitelist) > 0 {
		placeholders := make([]string, len(s.schemaWhitelist))
		for i, v := range s.schemaWhitelist {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, v)
		}
		query += fmt.Sprintf(" AND schema_name IN (%s)", strings.Join(placeholders, ", "))
	}

	rows, err := s.sourceDB.Query(query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning schema name: %v", err)
			continue
		}
		logging.GetLogger().Sugar().Infof("Processing schema: %s", schemaName)
		if err := s.processSchema(schemaName); err != nil {
			logging.GetLogger().Sugar().Warnf("Error processing schema %s: %v", schemaName, err)
		}
	}

	if err := s.processPrimaryKeys(); err != nil {
		logging.GetLogger().Sugar().Warnf("Error processing primary keys: %v", err)
	}
	if err := s.processForeignKeys(); err != nil {
		logging.GetLogger().Sugar().Warnf("Error processing foreign keys: %v", err)
	}

	// Process data profiling (unique counts, sample values)
	if err := s.processDataProfile(); err != nil {
		logging.GetLogger().Sugar().Warnf("Error processing data profile: %v", err)
	}

	// Add final validation
	s.validateParentChildRelationships()

	logging.GetLogger().Sugar().Infof("Extracted %d nodes and %d edges", len(s.nodes), len(s.edges))
	s.validateParentChildRelationships()

	return s.nodes, s.edges, nil
}

// processDataProfile collects unique value counts and sample values for each column
func (s *AnsiScanner) processDataProfile() error {
	logger := logging.GetLogger().Sugar()
	logger.Info("Starting data profiling for columns...")

	profiledCount := 0
	for colID, colNode := range s.columnMap {
		// Parse existing properties
		var propsMap map[string]interface{}
		if err := json.Unmarshal(colNode.Properties, &propsMap); err != nil {
			logger.Warnf("Error unmarshaling properties for column %s: %v", colNode.QualifiedPath, err)
			continue
		}

		// Extract schema.table.column from qualified path
		parts := strings.Split(strings.TrimPrefix(colNode.QualifiedPath, "/"), "/")
		if len(parts) < 3 {
			continue
		}
		schemaName := parts[0]
		tableName := parts[1]
		columnName := parts[2]

		// Query unique count and sample values from source database
		// Using a safe approach with quoting
		query := fmt.Sprintf(`
			SELECT 
				COUNT(DISTINCT %q) as unique_count,
				COUNT(*) as total_count
			FROM %q.%q
			LIMIT 1
		`, columnName, schemaName, tableName)

		var uniqueCount, totalCount int64
		err := s.sourceDB.QueryRow(query).Scan(&uniqueCount, &totalCount)
		if err != nil {
			logger.Debugf("Could not profile column %s.%s.%s: %v", schemaName, tableName, columnName, err)
			continue
		}

		propsMap["unique_count"] = uniqueCount
		propsMap["total_count"] = totalCount

		// Calculate cardinality ratio
		if totalCount > 0 {
			cardinalityRatio := float64(uniqueCount) / float64(totalCount)
			propsMap["cardinality_ratio"] = cardinalityRatio

			// Mark as low cardinality if ratio < 0.1 (potential lookup/enum column)
			if cardinalityRatio < 0.1 && uniqueCount < 100 {
				propsMap["is_low_cardinality"] = true
			}
		}

		// Get sample values for ALL columns (limit to 20)
		// For low cardinality/small tables, use DISTINCT to get representative set
		// For high cardinality, just get any 20 non-null values to verify format/content
		var sampleQuery string
		if uniqueCount <= 50 || totalCount < 1000 {
			sampleQuery = fmt.Sprintf(`
				SELECT DISTINCT %q::text 
				FROM %q.%q 
				WHERE %q IS NOT NULL 
				LIMIT 20
			`, columnName, schemaName, tableName, columnName)
		} else {
			// Fast sampling for large tables
			sampleQuery = fmt.Sprintf(`
				SELECT %q::text 
				FROM %q.%q 
				WHERE %q IS NOT NULL 
				LIMIT 20
			`, columnName, schemaName, tableName, columnName)
		}

		rows, err := s.sourceDB.Query(sampleQuery)
		if err == nil {
			var sampleValues []string
			for rows.Next() {
				var val string
				if err := rows.Scan(&val); err == nil {
					sampleValues = append(sampleValues, val)
				}
			}
			rows.Close()

			// If simpler query failed/returned no rows, keep sampleValues empty
			if len(sampleValues) > 0 {
				propsMap["sample_values"] = sampleValues

				// Detect value format patterns from samples
				detectedFormat := detectValueFormat(sampleValues)
				if detectedFormat != "" {
					propsMap["detected_format"] = detectedFormat
				}
			}
		} else {
			logger.Debugf("Failed to fetch samples for %s.%s.%s: %v", schemaName, tableName, columnName, err)
		}

		// Update properties
		propsJSON, err := json.Marshal(propsMap)
		if err != nil {
			continue
		}
		colNode.Properties = propsJSON
		s.columnMap[colID] = colNode
		profiledCount++
	}

	logger.Infof("Data profiling complete: profiled %d columns", profiledCount)
	return nil
}

// Add this validation function to check parent-child relationships before saving
func (s *AnsiScanner) validateParentChildRelationships() {
	logging.GetLogger().Sugar().Info("=== PARENT-CHILD RELATIONSHIP VALIDATION ===")

	tableCount := 0
	columnCount := 0
	orphanedColumns := 0

	// Create a map of all table IDs
	tableIDs := make(map[uuid.UUID]string)

	for _, node := range s.nodes {
		if node.NodeTypeID == NODE_TYPE_TABLE {
			tableIDs[node.ID] = node.QualifiedPath
			tableCount++
		}
	}

	// Check all columns have valid parent IDs
	for _, node := range s.nodes {
		if node.NodeTypeID == NODE_TYPE_COLUMN {
			columnCount++
			if !node.ParentID.Valid {
				logging.GetLogger().Sugar().Errorf("Column %s has no parent_id", node.QualifiedPath)
				orphanedColumns++
			} else {
				if _, exists := tableIDs[node.ParentID.UUID]; !exists {
					logging.GetLogger().Sugar().Errorf("Column %s has invalid parent_id: %s",
						node.QualifiedPath, node.ParentID.UUID.String())
					orphanedColumns++
				} else {
					logging.GetLogger().Sugar().Debugf("Column %s correctly linked to table %s",
						node.QualifiedPath, tableIDs[node.ParentID.UUID])
				}
			}
		}
	}

	logging.GetLogger().Sugar().Info("VALIDATION SUMMARY:")
	logging.GetLogger().Sugar().Infof("  Tables: %d", tableCount)
	logging.GetLogger().Sugar().Infof("  Columns: %d", columnCount)
	logging.GetLogger().Sugar().Infof("  Orphaned columns: %d", orphanedColumns)

	if orphanedColumns > 0 {
		logging.GetLogger().Sugar().Warnf("Found %d orphaned columns that will not be properly linked!", orphanedColumns)
	} else {
		logging.GetLogger().Sugar().Info("SUCCESS: All columns have valid parent relationships")
	}
}

// toTitleCase converts SNAKE_CASE column names to Title Case
// e.g., "customer_id" -> "Customer Id", "ACCOUNT_BALANCE" -> "Account Balance"
func toTitleCase(s string) string {
	// Replace underscores with spaces
	s = strings.ReplaceAll(s, "_", " ")

	// Title case each word
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// detectValueFormat analyzes sample values to detect common data format patterns
func detectValueFormat(sampleValues []string) string {
	if len(sampleValues) == 0 {
		return ""
	}

	// Count matches for each pattern
	emailCount := 0
	phoneCount := 0
	uuidCount := 0
	currencyCount := 0
	dateCount := 0
	urlCount := 0

	for _, val := range sampleValues {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}

		// Email pattern
		if strings.Contains(val, "@") && strings.Contains(val, ".") {
			emailCount++
		}

		// Phone pattern (various formats)
		if isPhoneNumber(val) {
			phoneCount++
		}

		// UUID pattern
		if isUUID(val) {
			uuidCount++
		}

		// Currency pattern
		if strings.HasPrefix(val, "$") || strings.HasPrefix(val, "€") || strings.HasPrefix(val, "£") {
			currencyCount++
		}

		// Date-like pattern
		if isDateLike(val) {
			dateCount++
		}

		// URL pattern
		if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
			urlCount++
		}
	}

	threshold := len(sampleValues) / 2 // At least 50% must match

	if emailCount > threshold {
		return "email"
	}
	if phoneCount > threshold {
		return "phone"
	}
	if uuidCount > threshold {
		return "uuid"
	}
	if currencyCount > threshold {
		return "currency"
	}
	if dateCount > threshold {
		return "date"
	}
	if urlCount > threshold {
		return "url"
	}

	return ""
}

// isPhoneNumber checks if a string looks like a phone number
func isPhoneNumber(s string) bool {
	// Remove common phone separators and check if mostly digits
	cleaned := strings.ReplaceAll(s, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")

	if len(cleaned) < 7 || len(cleaned) > 15 {
		return false
	}

	for _, c := range cleaned {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isUUID checks if a string is a valid UUID format
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	// Check for dash positions in UUID format: 8-4-4-4-12
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// isDateLike checks if a string looks like a date
func isDateLike(s string) bool {
	// Common date patterns
	if len(s) < 8 || len(s) > 25 {
		return false
	}
	// Contains date separators
	if strings.Contains(s, "/") || strings.Contains(s, "-") {
		// Has at least 2 separators and some digits
		digitCount := 0
		for _, c := range s {
			if c >= '0' && c <= '9' {
				digitCount++
			}
		}
		return digitCount >= 4
	}
	return false
}
