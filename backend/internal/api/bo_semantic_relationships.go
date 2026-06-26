package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

// BOSemanticRelationshipsService handles semantic term discovery and FK relationship mapping
// for business objects based on their driving tables
type BOSemanticRelationshipsService struct {
	db *sqlx.DB
}

// NewBOSemanticRelationshipsService creates a new service instance
func NewBOSemanticRelationshipsService(db *sqlx.DB) *BOSemanticRelationshipsService {
	return &BOSemanticRelationshipsService{db: db}
}

// ForeignKeyField represents a foreign key column mapping
type ForeignKeyField struct {
	SourceColumn string `json:"source_column"`
	TargetColumn string `json:"target_column"`
}

// RelatedTableFK represents a foreign key relationship to a related table
type RelatedTableFK struct {
	EdgeID           string                 `json:"edge_id"`
	RelatedTableID   string                 `json:"related_table_id"`
	RelatedTableName string                 `json:"related_table_name"`
	Cardinality      string                 `json:"cardinality"`
	Direction        string                 `json:"direction"` // "outbound" or "inbound"
	ForeignKeyFields []ForeignKeyField      `json:"foreign_key_fields"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
}

// RelatedTableSemanticTerm represents available semantic terms for a related table field
type RelatedTableSemanticTerm struct {
	SemanticTermID   string  `json:"semantic_term_id"`
	SemanticTermName string  `json:"semantic_term_name"`
	RelatedTableName string  `json:"related_table_name"`
	RelatedFieldName string  `json:"related_field_name"`
	RelatedFieldID   string  `json:"related_field_id"`
	SourceFkEdgeID   string  `json:"source_fk_edge_id"`
	JoinPath         string  `json:"join_path"` // e.g., "customer_id -> customers.id"
	Confidence       float64 `json:"confidence"`
	MatchReason      string  `json:"match_reason"` // e.g., "field_name_match", "type_match"
}

// BOSemanticLinkRequest represents request to link semantic terms to BO fields via FK relationships
type BOSemanticLinkRequest struct {
	BusinessObjectID string `json:"business_object_id"`
	SemanticTermID   string `json:"semantic_term_id"`
	RelatedTableID   string `json:"related_table_id"`
	ForeignKeyEdgeID string `json:"foreign_key_edge_id"`
	Role             string `json:"role,omitempty"` // e.g., "customer", "account", etc.
}

// DiscoverForeignKeyRelationshipsForBO discovers all FK relationships from a BO's driving table
func (s *BOSemanticRelationshipsService) DiscoverForeignKeyRelationshipsForBO(
	ctx context.Context,
	tenantID string,
	boID string,
) ([]RelatedTableFK, error) {
	// 1. Get the BO's driving table
	driverTableID, err := s.getDrivingTableID(ctx, tenantID, boID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driving table: %w", err)
	}

	if driverTableID == "" {
		return nil, fmt.Errorf("business object has no driving table")
	}

	// 2. Discover FK edges from/to the driving table
	fks, err := s.discoverForeignKeyEdges(ctx, tenantID, driverTableID)
	if err != nil {
		return nil, fmt.Errorf("failed to discover foreign keys: %w", err)
	}

	logging.GetLogger().Sugar().Infof(
		"Discovered %d foreign key relationships for BO %s (driving table %s)",
		len(fks), boID, driverTableID,
	)

	return fks, nil
}

// DiscoverSemanticTermsForRelatedTables discovers available semantic terms for tables
// related to a BO via foreign keys
func (s *BOSemanticRelationshipsService) DiscoverSemanticTermsForRelatedTables(
	ctx context.Context,
	tenantID string,
	boID string,
	limit int,
) ([]RelatedTableSemanticTerm, error) {
	if limit <= 0 {
		limit = 100
	}

	// 1. Discover FK relationships
	fks, err := s.DiscoverForeignKeyRelationshipsForBO(ctx, tenantID, boID)
	if err != nil {
		return nil, fmt.Errorf("failed to discover FK relationships: %w", err)
	}

	if len(fks) == 0 {
		logging.GetLogger().Sugar().Infof("No foreign key relationships found for BO %s", boID)
		return []RelatedTableSemanticTerm{}, nil
	}

	// 2. For each related table, discover semantic terms on related fields
	var allTerms []RelatedTableSemanticTerm
	for _, fk := range fks {
		terms, err := s.discoverSemanticTermsForRelatedTable(ctx, tenantID, &fk)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Error discovering terms for related table %s: %v", fk.RelatedTableName, err)
			continue
		}
		allTerms = append(allTerms, terms...)
	}

	// 3. Sort by confidence and limit
	if len(allTerms) > limit {
		allTerms = allTerms[:limit]
	}

	logging.GetLogger().Sugar().Infof(
		"Discovered %d available semantic terms from related tables for BO %s",
		len(allTerms), boID,
	)

	return allTerms, nil
}

// LinkSemanticTermToBusinessObject links a semantic term from a related table to a BO field
// This creates a join path so the BO knows how to fetch related table data
func (s *BOSemanticRelationshipsService) LinkSemanticTermToBusinessObject(
	ctx context.Context,
	tenantID string,
	req *BOSemanticLinkRequest,
) error {
	// 1. Validate FK edge exists
	fkEdge, err := s.getForeignKeyEdge(ctx, tenantID, req.ForeignKeyEdgeID)
	if err != nil {
		return fmt.Errorf("failed to validate foreign key edge: %w", err)
	}

	// 2. Create or update bo_field with the semantic term link
	// The field name will be based on the related table and role
	fieldKey := strings.ToLower(fmt.Sprintf("%s_%s", req.Role, fkEdge.RelatedTableName))
	fieldName := fmt.Sprintf("%s from %s", req.Role, fkEdge.RelatedTableName)

	// 3. Upsert bo_field with semantic_term_id and fk_edge_id reference
	err = s.createBOFieldWithSemanticTerm(
		ctx,
		tenantID,
		req.BusinessObjectID,
		fieldKey,
		fieldName,
		req.SemanticTermID,
		req.ForeignKeyEdgeID,
		fkEdge.Properties,
	)
	if err != nil {
		return fmt.Errorf("failed to link semantic term to BO: %w", err)
	}

	logging.GetLogger().Sugar().Infof(
		"Linked semantic term %s to BO %s via FK edge %s (related table %s)",
		req.SemanticTermID, req.BusinessObjectID, req.ForeignKeyEdgeID, fkEdge.RelatedTableName,
	)

	return nil
}

// GetBOSemanticJoinPaths returns the join paths needed to fetch semantic terms for a BO
// This is used when reconstructing queries to know which tables to join
func (s *BOSemanticRelationshipsService) GetBOSemanticJoinPaths(
	ctx context.Context,
	tenantID string,
	boID string,
) (map[string]interface{}, error) {
	query := `
		SELECT 
			bf.key,
			bf.semantic_term_id,
			bf.fk_edge_id,
			ce.properties,
			cn.node_name AS related_table_name
		FROM bo_fields bf
		LEFT JOIN catalog_edge ce ON bf.fk_edge_id = ce.id::text
		LEFT JOIN catalog_node cn ON ce.target_node_id = cn.id
		WHERE 
			bf.business_object_id = $1::uuid
			AND bf.semantic_term_id IS NOT NULL
			AND bf.fk_edge_id IS NOT NULL
			AND bf.tenant_id = $2::uuid
		ORDER BY bf.display_order
	`

	rows, err := s.db.QueryxContext(ctx, query, boID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query BO semantic join paths: %w", err)
	}
	defer rows.Close()

	joinPaths := make(map[string]interface{})
	for rows.Next() {
		var (
			fieldKey       string
			semanticTermID sql.NullString
			fkEdgeID       sql.NullString
			propsJSON      []byte
			relatedTable   sql.NullString
		)

		if err := rows.Scan(&fieldKey, &semanticTermID, &fkEdgeID, &propsJSON, &relatedTable); err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning join path: %v", err)
			continue
		}

		if semanticTermID.Valid && fkEdgeID.Valid {
			var props map[string]interface{}
			if len(propsJSON) > 0 {
				if err := json.Unmarshal(propsJSON, &props); err != nil {
					logging.GetLogger().Sugar().Warnf("Error unmarshaling FK edge properties: %v", err)
					props = make(map[string]interface{})
				}
			}

			joinPaths[fieldKey] = map[string]interface{}{
				"semantic_term_id": semanticTermID.String,
				"fk_edge_id":       fkEdgeID.String,
				"related_table":    relatedTable.String,
				"fk_properties":    props,
			}
		}
	}

	return joinPaths, rows.Err()
}

// ===== PRIVATE HELPER METHODS =====

// getDrivingTableID retrieves the driving table ID for a business object
func (s *BOSemanticRelationshipsService) getDrivingTableID(
	ctx context.Context,
	tenantID string,
	boID string,
) (string, error) {
	var driverTableID sql.NullString

	query := `
		SELECT driver_table_id
		FROM business_objects
		WHERE id = $1::uuid AND tenant_id = $2::uuid
		UNION ALL
		SELECT driver_table_id
		FROM business_object_def
		WHERE bo_def_id = $1::uuid AND tenant_id = $2::uuid
		LIMIT 1
	`

	err := s.db.GetContext(ctx, &driverTableID, query, boID, tenantID)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to query driving table: %w", err)
	}

	if driverTableID.Valid {
		return driverTableID.String, nil
	}

	return "", nil
}

// discoverForeignKeyEdges retrieves FK edges involving a table
// Returns both outbound (table A -> table B) and inbound (table B -> table A) relationships
func (s *BOSemanticRelationshipsService) discoverForeignKeyEdges(
	ctx context.Context,
	tenantID string,
	tableID string,
) ([]RelatedTableFK, error) {
	query := `
		WITH fk_edges AS (
			SELECT 
				ce.id,
				ce.source_node_id,
				ce.target_node_id,
				(ce.properties->>'edge_type_name')::text AS edge_type_name,
				ce.properties,
				CASE 
					WHEN ce.source_node_id = $2::uuid THEN 'outbound'
					WHEN ce.target_node_id = $2::uuid THEN 'inbound'
				END AS direction,
				CASE 
					WHEN ce.source_node_id = $2::uuid THEN ce.target_node_id
					WHEN ce.target_node_id = $2::uuid THEN ce.source_node_id
				END AS related_table_id,
				CASE 
					WHEN ce.source_node_id = $2::uuid THEN cn2.node_name
					WHEN ce.target_node_id = $2::uuid THEN cn1.node_name
				END AS related_table_name,
				(ce.properties->>'cardinality')::text AS cardinality
			FROM catalog_edge ce
			LEFT JOIN catalog_node cn1 ON ce.source_node_id = cn1.id
			LEFT JOIN catalog_node cn2 ON ce.target_node_id = cn2.id
			WHERE 
				(ce.source_node_id = $2::uuid OR ce.target_node_id = $2::uuid)
				AND (ce.properties->>'edge_type_name')::text = 'foreign_key'
				AND ce.tenant_id = $1::uuid
		)
		SELECT DISTINCT
			id,
			related_table_id,
			related_table_name,
			cardinality,
			direction,
			properties
		FROM fk_edges
		WHERE related_table_id IS NOT NULL AND related_table_name IS NOT NULL
		ORDER BY related_table_name
	`

	rows, err := s.db.QueryxContext(ctx, query, tenantID, tableID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query foreign key edges: %w", err)
	}
	defer rows.Close()

	var fks []RelatedTableFK
	for rows.Next() {
		var (
			edgeID           string
			relatedTableID   string
			relatedTableName string
			cardinality      sql.NullString
			direction        sql.NullString
			propsJSON        []byte
		)

		if err := rows.Scan(&edgeID, &relatedTableID, &relatedTableName, &cardinality, &direction, &propsJSON); err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning FK edge: %v", err)
			continue
		}

		var props map[string]interface{}
		if len(propsJSON) > 0 {
			if err := json.Unmarshal(propsJSON, &props); err != nil {
				logging.GetLogger().Sugar().Warnf("Error unmarshaling properties: %v", err)
				props = make(map[string]interface{})
			}
		}

		// Extract FK fields from properties
		fkFields := s.extractFKFields(props)

		fk := RelatedTableFK{
			EdgeID:           edgeID,
			RelatedTableID:   relatedTableID,
			RelatedTableName: relatedTableName,
			Cardinality:      cardinality.String,
			Direction:        direction.String,
			ForeignKeyFields: fkFields,
			Properties:       props,
		}

		fks = append(fks, fk)
	}

	return fks, rows.Err()
}

// extractFKFields extracts column mappings from FK edge properties
func (s *BOSemanticRelationshipsService) extractFKFields(props map[string]interface{}) []ForeignKeyField {
	var fields []ForeignKeyField

	// Try to extract from "columns" array in properties
	if columns, ok := props["columns"]; ok {
		if colArray, ok := columns.([]interface{}); ok {
			for _, col := range colArray {
				if colMap, ok := col.(map[string]interface{}); ok {
					field := ForeignKeyField{
						SourceColumn: s.stringValue(colMap["source_column"]),
						TargetColumn: s.stringValue(colMap["target_column"]),
					}
					if field.SourceColumn != "" && field.TargetColumn != "" {
						fields = append(fields, field)
					}
				}
			}
		}
	}

	return fields
}

// stringValue safely extracts string value from interface{}
func (s *BOSemanticRelationshipsService) stringValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case *string:
		if val != nil {
			return *val
		}
	}
	return ""
}

// discoverSemanticTermsForRelatedTable discovers semantic terms available on fields of a related table
func (s *BOSemanticRelationshipsService) discoverSemanticTermsForRelatedTable(
	ctx context.Context,
	tenantID string,
	fk *RelatedTableFK,
) ([]RelatedTableSemanticTerm, error) {
	// Query semantic terms linked to columns of the related table
	query := `
		WITH table_columns AS (
			SELECT 
				cn.id AS column_id,
				cn.node_name AS column_name
			FROM catalog_node cn
			WHERE cn.parent_id = $2::uuid
				AND cn.node_type_id IN (
					SELECT id FROM catalog_node_type WHERE name = 'database_column'
				)
				AND cn.tenant_id = $1::uuid
		),
		column_semantic_mappings AS (
			SELECT 
				tc.column_id,
				tc.column_name,
				ce.target_node_id AS semantic_term_id,
				cn2.node_name AS semantic_term_name
			FROM table_columns tc
			LEFT JOIN catalog_edge ce ON ce.source_node_id = tc.column_id
			LEFT JOIN catalog_node cn2 ON ce.target_node_id = cn2.id
			WHERE ce.properties->>'relationship_type' = 'has_semantic_term'
				AND cn2.node_type_id IN (
					SELECT id FROM catalog_node_type WHERE name IN ('semantic_term', 'business_term')
				)
		)
		SELECT DISTINCT
			semantic_term_id,
			semantic_term_name,
			column_id,
			column_name
		FROM column_semantic_mappings
		WHERE semantic_term_id IS NOT NULL
		ORDER BY semantic_term_name
	`

	rows, err := s.db.QueryxContext(ctx, query, tenantID, fk.RelatedTableID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query semantic terms for related table: %w", err)
	}
	defer rows.Close()

	var terms []RelatedTableSemanticTerm
	for rows.Next() {
		var (
			semanticTermID   sql.NullString
			semanticTermName sql.NullString
			fieldID          sql.NullString
			fieldName        sql.NullString
		)

		if err := rows.Scan(&semanticTermID, &semanticTermName, &fieldID, &fieldName); err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning semantic term: %v", err)
			continue
		}

		if semanticTermID.Valid && semanticTermName.Valid && fieldID.Valid && fieldName.Valid {
			// Build join path description
			joinPath := ""
			if len(fk.ForeignKeyFields) > 0 {
				fkField := fk.ForeignKeyFields[0]
				joinPath = fmt.Sprintf("%s -> %s.%s", fkField.SourceColumn, fk.RelatedTableName, fkField.TargetColumn)
			}

			term := RelatedTableSemanticTerm{
				SemanticTermID:   semanticTermID.String,
				SemanticTermName: semanticTermName.String,
				RelatedTableName: fk.RelatedTableName,
				RelatedFieldName: fieldName.String,
				RelatedFieldID:   fieldID.String,
				SourceFkEdgeID:   fk.EdgeID,
				JoinPath:         joinPath,
				Confidence:       0.95, // High confidence for direct semantic mappings
				MatchReason:      "semantic_term_mapped_in_catalog",
			}

			terms = append(terms, term)
		}
	}

	return terms, rows.Err()
}

// getForeignKeyEdge retrieves a single FK edge by ID
func (s *BOSemanticRelationshipsService) getForeignKeyEdge(
	ctx context.Context,
	tenantID string,
	edgeID string,
) (*RelatedTableFK, error) {
	query := `
		SELECT 
			ce.id,
			cn1.node_name AS source_table,
			cn2.node_name AS target_table,
			(ce.properties->>'cardinality')::text AS cardinality,
			CASE 
				WHEN EXISTS(
					SELECT 1 FROM bo_fields bf 
					WHERE bf.fk_edge_id = $2
					LIMIT 1
				) THEN 'outbound'
				ELSE 'inbound'
			END AS direction,
			cn2.id,
			cn2.node_name,
			ce.properties
		FROM catalog_edge ce
		LEFT JOIN catalog_node cn1 ON ce.source_node_id = cn1.id
		LEFT JOIN catalog_node cn2 ON ce.target_node_id = cn2.id
		WHERE ce.id = $2::uuid
			AND ce.tenant_id = $1::uuid
	`

	var (
		edgeID_          string
		sourceTable      sql.NullString
		targetTable      sql.NullString
		cardinality      sql.NullString
		direction        sql.NullString
		relatedTableID   string
		relatedTableName string
		propsJSON        []byte
	)

	err := s.db.QueryRowxContext(ctx, query, tenantID, edgeID).
		Scan(&edgeID_, &sourceTable, &targetTable, &cardinality, &direction, &relatedTableID, &relatedTableName, &propsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to query FK edge: %w", err)
	}

	var props map[string]interface{}
	if len(propsJSON) > 0 {
		if err := json.Unmarshal(propsJSON, &props); err != nil {
			logging.GetLogger().Sugar().Warnf("Error unmarshaling properties: %v", err)
			props = make(map[string]interface{})
		}
	}

	return &RelatedTableFK{
		EdgeID:           edgeID_,
		RelatedTableID:   relatedTableID,
		RelatedTableName: relatedTableName,
		Cardinality:      cardinality.String,
		Direction:        direction.String,
		ForeignKeyFields: s.extractFKFields(props),
		Properties:       props,
	}, nil
}

// createBOFieldWithSemanticTerm creates or updates a bo_field with a semantic term link via FK
func (s *BOSemanticRelationshipsService) createBOFieldWithSemanticTerm(
	ctx context.Context,
	tenantID string,
	boID string,
	fieldKey string,
	fieldName string,
	semanticTermID string,
	fkEdgeID string,
	fkProperties map[string]interface{},
) error {
	fieldID := uuid.New().String()

	query := `
		INSERT INTO bo_fields (
			id, tenant_id, business_object_id, key, name, display_label,
			field_type, is_core, display_order, semantic_term_id, fk_edge_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			semantic_term_id = $10,
			fk_edge_id = $11,
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(
		ctx,
		query,
		fieldID,          // id
		tenantID,         // tenant_id
		boID,             // business_object_id
		fieldKey,         // key
		fieldName,        // name
		fieldName,        // display_label
		"related_object", // field_type (enum for related objects)
		false,            // is_core
		999,              // display_order (last)
		semanticTermID,   // semantic_term_id
		fkEdgeID,         // fk_edge_id
		time.Now(),       // created_at
	)

	return err
}
