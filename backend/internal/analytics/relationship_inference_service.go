package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RelationshipInferenceService discovers and scores relationships between tables
// using profile data (cardinality, uniqueness, value distributions).
type RelationshipInferenceService struct {
	db *sqlx.DB
}

// NewRelationshipInferenceService creates a new relationship inference service
func NewRelationshipInferenceService(db *sqlx.DB) *RelationshipInferenceService {
	return &RelationshipInferenceService{db: db}
}

// ============================================================================
// Types
// ============================================================================

// RelationshipCandidate represents a potential relationship between two tables
type RelationshipCandidate struct {
	LeftTableID     uuid.UUID            `json:"left_table_id" db:"left_table_id"`
	LeftTableName   string               `json:"left_table_name" db:"left_table_name"`
	LeftColumn      string               `json:"left_column" db:"left_column"`
	RightTableID    uuid.UUID            `json:"right_table_id" db:"right_table_id"`
	RightTableName  string               `json:"right_table_name" db:"right_table_name"`
	RightColumn     string               `json:"right_column" db:"right_column"`
	Cardinality     string               `json:"cardinality"`
	Confidence      float64              `json:"confidence"`
	JoinCondition   string               `json:"join_condition"`
	JoinType        string               `json:"join_type"`
	Origin          string               `json:"origin"`
	LookupCandidate bool                 `json:"lookup_candidate"`
	Profile         *RelationshipProfile `json:"profile,omitempty"`
	MatchReasons    []string             `json:"match_reasons,omitempty"`
}

// RelationshipProfile contains profiling metadata for relationship scoring
type RelationshipProfile struct {
	LeftDistinct    int64   `json:"left_distinct" db:"left_distinct"`
	RightDistinct   int64   `json:"right_distinct" db:"right_distinct"`
	LeftRowCount    int64   `json:"left_row_count" db:"left_row_count"`
	RightRowCount   int64   `json:"right_row_count" db:"right_row_count"`
	JoinSelectivity float64 `json:"join_selectivity"`
	LeftUnique      bool    `json:"left_unique"`
	RightUnique     bool    `json:"right_unique"`
}

// TableRelationshipEdge represents a persisted TABLE_RELATES_TO_TABLE edge
type TableRelationshipEdge struct {
	ID              uuid.UUID            `json:"id" db:"id"`
	SourceNodeID    uuid.UUID            `json:"source_node_id" db:"source_node_id"`
	TargetNodeID    uuid.UUID            `json:"target_node_id" db:"target_node_id"`
	JoinCondition   string               `json:"join_condition"`
	JoinType        string               `json:"join_type"`
	Cardinality     string               `json:"cardinality"`
	Confidence      float64              `json:"confidence"`
	Origin          string               `json:"origin"`
	LookupCandidate bool                 `json:"lookup_candidate"`
	Profile         *RelationshipProfile `json:"profile,omitempty"`
	Notes           string               `json:"notes,omitempty"`
}

// BORelationshipEdge represents a persisted BO_RELATES_TO_BO edge
type BORelationshipEdge struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	SourceBOID       uuid.UUID  `json:"source_bo_id" db:"source_node_id"`
	TargetBOID       uuid.UUID  `json:"target_bo_id" db:"target_node_id"`
	RelationshipType string     `json:"relationship_type"`
	JoinPath         []JoinStep `json:"join_path"`
	ViaTables        []string   `json:"via_tables"`
	Lookup           bool       `json:"lookup"`
	UIRole           string     `json:"ui_role"`
	Description      string     `json:"description"`
	Origin           string     `json:"origin"`
	Confidence       float64    `json:"confidence"`
}

// JoinStep represents a single step in a join path
type JoinStep struct {
	LeftTable   string `json:"left_table,omitempty"`
	LeftAlias   string `json:"left_alias,omitempty"`
	LeftColumn  string `json:"left_column,omitempty"`
	RightTable  string `json:"right_table,omitempty"`
	RightAlias  string `json:"right_alias,omitempty"`
	RightColumn string `json:"right_column,omitempty"`
	JoinType    string `json:"join_type,omitempty"`
	// Simplified form
	Table  string `json:"table,omitempty"`
	Alias  string `json:"alias,omitempty"`
	Column string `json:"column,omitempty"`
}

// ============================================================================
// Relationship Discovery
// ============================================================================

// DiscoverTableRelationships finds candidate relationships for the given tables
// using column name matching, type matching, and profile data analysis.
func (s *RelationshipInferenceService) DiscoverTableRelationships(
	ctx context.Context,
	datasourceID uuid.UUID,
	tableIDs []uuid.UUID,
) ([]RelationshipCandidate, error) {
	if len(tableIDs) == 0 {
		return nil, nil
	}

	// Query to find columns with matching names across tables
	// Uses column_profiles for cardinality/uniqueness data
	query := `
		WITH table_columns AS (
			SELECT 
				cn_table.id AS table_id,
				cn_table.node_name AS table_name,
				cn_col.id AS column_id,
				cn_col.node_name AS column_name,
				COALESCE(cn_col.properties->>'data_type', 'unknown') AS data_type,
				cp.cardinality AS distinct_count,
				cp.row_count,
				CASE 
					WHEN cp.cardinality IS NOT NULL AND cp.row_count IS NOT NULL 
					AND cp.row_count > 0 AND cp.cardinality::float / cp.row_count::float > 0.95
					THEN true ELSE false
				END AS is_unique
			FROM catalog_node cn_table
			JOIN catalog_node cn_col ON cn_col.parent_id = cn_table.id
			LEFT JOIN sml.column_profiles cp ON cp.column_name = cn_col.node_name 
				AND cp.table_name = cn_table.node_name
			WHERE cn_table.tenant_datasource_id = $1
			AND cn_table.id = ANY($2)
		)
		SELECT 
			l.table_id AS left_table_id,
			l.table_name AS left_table_name,
			l.column_name AS left_column,
			l.distinct_count AS left_distinct,
			l.row_count AS left_row_count,
			l.is_unique AS left_unique,
			r.table_id AS right_table_id,
			r.table_name AS right_table_name,
			r.column_name AS right_column,
			r.distinct_count AS right_distinct,
			r.row_count AS right_row_count,
			r.is_unique AS right_unique,
			l.data_type AS left_data_type,
			r.data_type AS right_data_type
		FROM table_columns l
		JOIN table_columns r ON l.column_name = r.column_name
			AND l.table_id < r.table_id  -- Avoid duplicates and self-joins
			AND l.data_type = r.data_type
		ORDER BY l.table_name, r.table_name, l.column_name
	`

	rows, err := s.db.QueryxContext(ctx, query, datasourceID, tableIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query relationship candidates: %w", err)
	}
	defer rows.Close()

	var candidates []RelationshipCandidate
	for rows.Next() {
		var row struct {
			LeftTableID    uuid.UUID `db:"left_table_id"`
			LeftTableName  string    `db:"left_table_name"`
			LeftColumn     string    `db:"left_column"`
			LeftDistinct   *int64    `db:"left_distinct"`
			LeftRowCount   *int64    `db:"left_row_count"`
			LeftUnique     bool      `db:"left_unique"`
			RightTableID   uuid.UUID `db:"right_table_id"`
			RightTableName string    `db:"right_table_name"`
			RightColumn    string    `db:"right_column"`
			RightDistinct  *int64    `db:"right_distinct"`
			RightRowCount  *int64    `db:"right_row_count"`
			RightUnique    bool      `db:"right_unique"`
			LeftDataType   string    `db:"left_data_type"`
			RightDataType  string    `db:"right_data_type"`
		}

		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Build profile
		profile := &RelationshipProfile{
			LeftUnique:  row.LeftUnique,
			RightUnique: row.RightUnique,
		}
		if row.LeftDistinct != nil {
			profile.LeftDistinct = *row.LeftDistinct
		}
		if row.RightDistinct != nil {
			profile.RightDistinct = *row.RightDistinct
		}
		if row.LeftRowCount != nil {
			profile.LeftRowCount = *row.LeftRowCount
		}
		if row.RightRowCount != nil {
			profile.RightRowCount = *row.RightRowCount
		}

		// Infer cardinality
		cardinality := s.InferCardinality(profile)

		// Calculate confidence score
		confidence, reasons := s.ScoreRelationship(row.LeftColumn, row.RightColumn, row.LeftDataType, row.RightDataType, profile)

		// Detect lookup candidate
		isLookup := s.IsLookupCandidate(row.RightTableName, profile.RightRowCount, row.RightUnique)

		// Build join condition
		joinCondition := fmt.Sprintf("%s.%s = %s.%s",
			row.LeftTableName, row.LeftColumn,
			row.RightTableName, row.RightColumn)

		candidate := RelationshipCandidate{
			LeftTableID:     row.LeftTableID,
			LeftTableName:   row.LeftTableName,
			LeftColumn:      row.LeftColumn,
			RightTableID:    row.RightTableID,
			RightTableName:  row.RightTableName,
			RightColumn:     row.RightColumn,
			Cardinality:     cardinality,
			Confidence:      confidence,
			JoinCondition:   joinCondition,
			JoinType:        "left",
			Origin:          "inferred",
			LookupCandidate: isLookup,
			Profile:         profile,
			MatchReasons:    reasons,
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// ============================================================================
// Cardinality Detection
// ============================================================================

// InferCardinality determines the relationship cardinality from profile data
func (s *RelationshipInferenceService) InferCardinality(profile *RelationshipProfile) string {
	if profile == nil {
		return "unknown"
	}

	// Calculate selectivity if we have row counts
	if profile.LeftRowCount > 0 && profile.RightRowCount > 0 {
		// High selectivity threshold
		minRows := profile.LeftRowCount
		if profile.RightRowCount < minRows {
			minRows = profile.RightRowCount
		}

		// Estimate selectivity based on distinct counts
		if profile.LeftDistinct > 0 && profile.RightDistinct > 0 {
			overlapRatio := float64(minInt64(profile.LeftDistinct, profile.RightDistinct)) / float64(maxInt64(profile.LeftDistinct, profile.RightDistinct))
			profile.JoinSelectivity = overlapRatio
		}
	}

	highSelectivity := profile.JoinSelectivity > 0.9

	// Cardinality inference rules
	switch {
	case profile.LeftUnique && profile.RightUnique && highSelectivity:
		return "1:1"
	case !profile.LeftUnique && profile.RightUnique:
		return "M:1"
	case profile.LeftUnique && !profile.RightUnique:
		return "1:M"
	case !profile.LeftUnique && !profile.RightUnique:
		return "M:M"
	default:
		return "unknown"
	}
}

// ============================================================================
// Confidence Scoring
// ============================================================================

// ScoreRelationship calculates a confidence score for a relationship candidate
func (s *RelationshipInferenceService) ScoreRelationship(
	leftCol, rightCol, leftType, rightType string,
	profile *RelationshipProfile,
) (float64, []string) {
	score := 0.3 // Base score
	var reasons []string

	// Name match bonus (exact match already guaranteed by query)
	score += 0.15
	reasons = append(reasons, "Exact column name match")

	// ID pattern bonus
	if strings.HasSuffix(strings.ToLower(leftCol), "_id") ||
		strings.HasSuffix(strings.ToLower(leftCol), "id") {
		score += 0.1
		reasons = append(reasons, "Column follows *_id naming pattern")
	}

	// Type match bonus (already guaranteed by query)
	score += 0.1
	reasons = append(reasons, "Data types match")

	// Profile-based scoring
	if profile != nil {
		// High selectivity bonus
		if profile.JoinSelectivity > 0.9 {
			score += 0.2
			reasons = append(reasons, "High join selectivity (>90%)")
		} else if profile.JoinSelectivity > 0.7 {
			score += 0.1
			reasons = append(reasons, "Good join selectivity (>70%)")
		}

		// Uniqueness pattern bonus
		if (profile.LeftUnique && !profile.RightUnique) || (!profile.LeftUnique && profile.RightUnique) {
			score += 0.15
			reasons = append(reasons, "Uniqueness pattern matches typical FK relationship")
		}
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score, reasons
}

// ============================================================================
// Lookup Detection
// ============================================================================

// LookupThreshold is the maximum row count for a table to be considered a lookup
const LookupThreshold = 100000

// IsLookupCandidate determines if a table is likely a lookup/reference table
func (s *RelationshipInferenceService) IsLookupCandidate(tableName string, rowCount int64, hasUniqueKey bool) bool {
	// Rule 1: Small row count
	if rowCount > 0 && rowCount < LookupThreshold {
		// Rule 2: Has a unique/primary key
		if hasUniqueKey {
			return true
		}
	}

	// Rule 3: Naming patterns that suggest dimension/lookup
	lowerName := strings.ToLower(tableName)
	lookupPrefixes := []string{"dim_", "lkp_", "ref_", "lookup_"}
	for _, prefix := range lookupPrefixes {
		if strings.HasPrefix(lowerName, prefix) {
			return true
		}
	}

	return false
}

// ============================================================================
// Persistence
// ============================================================================

// CreateTableRelationship persists a TABLE_RELATES_TO_TABLE edge
func (s *RelationshipInferenceService) CreateTableRelationship(
	ctx context.Context,
	tenantID, datasourceID uuid.UUID,
	edge TableRelationshipEdge,
) (uuid.UUID, error) {
	// Serialize profile to JSON
	profileJSON, err := json.Marshal(edge.Profile)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal profile: %w", err)
	}

	properties := map[string]interface{}{
		"join_condition":   edge.JoinCondition,
		"join_type":        edge.JoinType,
		"cardinality":      edge.Cardinality,
		"confidence":       edge.Confidence,
		"origin":           edge.Origin,
		"lookup_candidate": edge.LookupCandidate,
		"profile":          json.RawMessage(profileJSON),
		"notes":            edge.Notes,
	}

	propsJSON, err := json.Marshal(properties)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	// Get edge type ID
	var edgeTypeID uuid.UUID
	err = s.db.GetContext(ctx, &edgeTypeID, `
		SELECT id FROM catalog_edge_type 
		WHERE tenant_id = $1 AND edge_type_name = 'TABLE_RELATES_TO_TABLE'
		LIMIT 1
	`, tenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to find TABLE_RELATES_TO_TABLE edge type: %w", err)
	}

	// Insert edge
	var edgeID uuid.UUID
	err = s.db.GetContext(ctx, &edgeID, `
		INSERT INTO catalog_edge (
			tenant_datasource_id, source_node_id, target_node_id, 
			relationship_type, edge_type_id, edge_type_name, properties, tenant_id,
			created_at, updated_at
		) VALUES ($1, $2, $3, 'TABLE_RELATES_TO_TABLE', $4, 'TABLE_RELATES_TO_TABLE', $5, $6, now(), now())
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_name, target_node_id)
		DO UPDATE SET properties = EXCLUDED.properties, updated_at = now()
		RETURNING id
	`, datasourceID, edge.SourceNodeID, edge.TargetNodeID, edgeTypeID, propsJSON, tenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create table relationship edge: %w", err)
	}

	return edgeID, nil
}

// GetTableRelationships retrieves all TABLE_RELATES_TO_TABLE edges for a table
func (s *RelationshipInferenceService) GetTableRelationships(
	ctx context.Context,
	datasourceID, tableNodeID uuid.UUID,
) ([]TableRelationshipEdge, error) {
	query := `
		SELECT 
			ce.id,
			ce.source_node_id,
			ce.target_node_id,
			ce.properties
		FROM catalog_edge ce
		WHERE ce.tenant_datasource_id = $1
		AND (ce.source_node_id = $2 OR ce.target_node_id = $2)
		AND ce.edge_type_name = 'TABLE_RELATES_TO_TABLE'
	`

	rows, err := s.db.QueryxContext(ctx, query, datasourceID, tableNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query table relationships: %w", err)
	}
	defer rows.Close()

	var edges []TableRelationshipEdge
	for rows.Next() {
		var row struct {
			ID           uuid.UUID       `db:"id"`
			SourceNodeID uuid.UUID       `db:"source_node_id"`
			TargetNodeID uuid.UUID       `db:"target_node_id"`
			Properties   json.RawMessage `db:"properties"`
		}
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		edge := TableRelationshipEdge{
			ID:           row.ID,
			SourceNodeID: row.SourceNodeID,
			TargetNodeID: row.TargetNodeID,
		}

		// Parse properties
		if len(row.Properties) > 0 {
			var props map[string]interface{}
			if err := json.Unmarshal(row.Properties, &props); err == nil {
				if v, ok := props["join_condition"].(string); ok {
					edge.JoinCondition = v
				}
				if v, ok := props["join_type"].(string); ok {
					edge.JoinType = v
				}
				if v, ok := props["cardinality"].(string); ok {
					edge.Cardinality = v
				}
				if v, ok := props["confidence"].(float64); ok {
					edge.Confidence = v
				}
				if v, ok := props["origin"].(string); ok {
					edge.Origin = v
				}
				if v, ok := props["lookup_candidate"].(bool); ok {
					edge.LookupCandidate = v
				}
				if v, ok := props["notes"].(string); ok {
					edge.Notes = v
				}
			}
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// ============================================================================
// BO Relationship Inheritance
// ============================================================================

// InheritBORelationshipsFromTable creates BO_RELATES_TO_BO edges based on
// physical TABLE_RELATES_TO_TABLE edges from a BO's driving table.
func (s *RelationshipInferenceService) InheritBORelationshipsFromTable(
	ctx context.Context,
	tenantID, datasourceID, boNodeID, drivingTableNodeID uuid.UUID,
) ([]BORelationshipEdge, error) {
	// Get physical relationships from driving table
	tableRels, err := s.GetTableRelationships(ctx, datasourceID, drivingTableNodeID)
	if err != nil {
		return nil, err
	}

	// Get the driving table name for building join paths
	var drivingTableName string
	_ = s.db.GetContext(ctx, &drivingTableName, `
		SELECT node_name FROM catalog_node WHERE id = $1
	`, drivingTableNodeID)

	var boRels []BORelationshipEdge

	for _, tableRel := range tableRels {
		// Get target table name
		var targetTableName string
		_ = s.db.GetContext(ctx, &targetTableName, `
			SELECT node_name FROM catalog_node WHERE id = $1
		`, tableRel.TargetNodeID)

		// Find if the related table is a driving table for another BO
		var relatedBONodeID uuid.UUID
		err := s.db.GetContext(ctx, &relatedBONodeID, `
			SELECT cn.id 
			FROM catalog_node cn
			WHERE cn.tenant_datasource_id = $1
			AND cn.properties->>'driver_table_id' = $2::text
			LIMIT 1
		`, datasourceID, tableRel.TargetNodeID.String())

		if err != nil {
			// No BO for this table, skip
			continue
		}

		// Determine UI role based on cardinality and lookup status
		uiRole := "detail"
		if tableRel.LookupCandidate {
			uiRole = "lookup"
		} else if tableRel.Cardinality == "1:M" || tableRel.Cardinality == "M:M" {
			uiRole = "child_collection"
		}

		// Parse join condition to extract columns
		leftCol, rightCol := parseJoinConditionColumns(tableRel.JoinCondition)

		boRel := BORelationshipEdge{
			SourceBOID:       boNodeID,
			TargetBOID:       relatedBONodeID,
			RelationshipType: tableRel.Cardinality,
			JoinPath: []JoinStep{
				{
					LeftTable:   drivingTableName,
					LeftColumn:  leftCol,
					RightTable:  targetTableName,
					RightColumn: rightCol,
					JoinType:    tableRel.JoinType,
				},
			},
			ViaTables:  []string{},
			Lookup:     tableRel.LookupCandidate,
			UIRole:     uiRole,
			Origin:     "inferred",
			Confidence: tableRel.Confidence,
		}

		boRels = append(boRels, boRel)
	}

	return boRels, nil
}

// GetBORelationships retrieves all BO_RELATES_TO_BO edges for a business object
func (s *RelationshipInferenceService) GetBORelationships(
	ctx context.Context,
	datasourceID, boNodeID uuid.UUID,
) ([]BORelationshipEdge, error) {
	query := `
		SELECT 
			ce.id,
			ce.source_node_id,
			ce.target_node_id,
			ce.properties,
			src_bo.node_name AS source_bo_name,
			tgt_bo.node_name AS target_bo_name
		FROM catalog_edge ce
		JOIN catalog_node src_bo ON ce.source_node_id = src_bo.id
		JOIN catalog_node tgt_bo ON ce.target_node_id = tgt_bo.id
		WHERE ce.tenant_datasource_id = $1
		AND (ce.source_node_id = $2 OR ce.target_node_id = $2)
		AND ce.edge_type_name = 'BO_RELATES_TO_BO'
	`

	rows, err := s.db.QueryxContext(ctx, query, datasourceID, boNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query BO relationships: %w", err)
	}
	defer rows.Close()

	var edges []BORelationshipEdge
	for rows.Next() {
		var row struct {
			ID           uuid.UUID       `db:"id"`
			SourceNodeID uuid.UUID       `db:"source_node_id"`
			TargetNodeID uuid.UUID       `db:"target_node_id"`
			Properties   json.RawMessage `db:"properties"`
			SourceBOName string          `db:"source_bo_name"`
			TargetBOName string          `db:"target_bo_name"`
		}
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		edge := BORelationshipEdge{
			ID:         row.ID,
			SourceBOID: row.SourceNodeID,
			TargetBOID: row.TargetNodeID,
		}

		// Parse properties
		if len(row.Properties) > 0 {
			var props map[string]interface{}
			if err := json.Unmarshal(row.Properties, &props); err == nil {
				if v, ok := props["relationship_type"].(string); ok {
					edge.RelationshipType = v
				}
				if v, ok := props["ui_role"].(string); ok {
					edge.UIRole = v
				}
				if v, ok := props["lookup"].(bool); ok {
					edge.Lookup = v
				}
				if v, ok := props["description"].(string); ok {
					edge.Description = v
				}
				if v, ok := props["origin"].(string); ok {
					edge.Origin = v
				}
				if v, ok := props["confidence"].(float64); ok {
					edge.Confidence = v
				}
				// Parse join_path array
				if jpRaw, ok := props["join_path"].([]interface{}); ok {
					for _, stepRaw := range jpRaw {
						if stepMap, ok := stepRaw.(map[string]interface{}); ok {
							step := JoinStep{}
							if v, ok := stepMap["left_table"].(string); ok {
								step.LeftTable = v
							}
							if v, ok := stepMap["left_column"].(string); ok {
								step.LeftColumn = v
							}
							if v, ok := stepMap["right_table"].(string); ok {
								step.RightTable = v
							}
							if v, ok := stepMap["right_column"].(string); ok {
								step.RightColumn = v
							}
							if v, ok := stepMap["join_type"].(string); ok {
								step.JoinType = v
							}
							edge.JoinPath = append(edge.JoinPath, step)
						}
					}
				}
				// Parse via_tables array
				if vtRaw, ok := props["via_tables"].([]interface{}); ok {
					for _, t := range vtRaw {
						if tStr, ok := t.(string); ok {
							edge.ViaTables = append(edge.ViaTables, tStr)
						}
					}
				}
			}
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// parseJoinConditionColumns extracts column names from a join condition like "table.col = other.col"
func parseJoinConditionColumns(condition string) (leftCol, rightCol string) {
	if condition == "" {
		return "id", "id"
	}

	parts := strings.Split(condition, " = ")
	if len(parts) != 2 {
		return "id", "id"
	}

	// Extract column from "table.column" format
	leftParts := strings.Split(strings.TrimSpace(parts[0]), ".")
	rightParts := strings.Split(strings.TrimSpace(parts[1]), ".")

	if len(leftParts) >= 2 {
		leftCol = leftParts[len(leftParts)-1]
	} else if len(leftParts) == 1 {
		leftCol = leftParts[0]
	} else {
		leftCol = "id"
	}

	if len(rightParts) >= 2 {
		rightCol = rightParts[len(rightParts)-1]
	} else if len(rightParts) == 1 {
		rightCol = rightParts[0]
	} else {
		rightCol = "id"
	}

	return leftCol, rightCol
}

// Helper functions
func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
