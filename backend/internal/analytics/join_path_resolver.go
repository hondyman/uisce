package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// JoinPathResolver provides BFS-based join path resolution between tables
// using TABLE_RELATES_TO_TABLE edges from the catalog graph.
type JoinPathResolver struct {
	db *sqlx.DB
}

// NewJoinPathResolver creates a new join path resolver
func NewJoinPathResolver(db *sqlx.DB) *JoinPathResolver {
	return &JoinPathResolver{db: db}
}

// JoinPath represents a complete path of joins between tables
type JoinPath struct {
	Steps      []JoinPathStep `json:"steps"`
	TotalHops  int            `json:"total_hops"`
	Confidence float64        `json:"confidence"`
}

// JoinPathStep represents a single step in a join path
type JoinPathStep struct {
	LeftTable   string `json:"left_table"`
	LeftAlias   string `json:"left_alias"`
	LeftColumn  string `json:"left_column"`
	RightTable  string `json:"right_table"`
	RightAlias  string `json:"right_alias"`
	RightColumn string `json:"right_column"`
	JoinType    string `json:"join_type"`
	Cardinality string `json:"cardinality,omitempty"`
}

// TableEdge represents a relationship from the graph
type TableEdge struct {
	SourceTableID   uuid.UUID `db:"source_node_id"`
	TargetTableID   uuid.UUID `db:"target_node_id"`
	SourceTableName string    `db:"source_table_name"`
	TargetTableName string    `db:"target_table_name"`
	JoinCondition   string    `db:"join_condition"`
	JoinType        string    `db:"join_type"`
	Cardinality     string    `db:"cardinality"`
}

// ============================================================================
// Join Path Resolution (BFS)
// ============================================================================

// ResolveJoinPath finds the shortest path between two tables using BFS
// over TABLE_RELATES_TO_TABLE edges.
func (r *JoinPathResolver) ResolveJoinPath(
	ctx context.Context,
	datasourceID uuid.UUID,
	fromTableName, toTableName string,
	maxDepth int,
) (*JoinPath, error) {
	if maxDepth <= 0 {
		maxDepth = 3 // Default max depth
	}

	if fromTableName == toTableName {
		return &JoinPath{Steps: nil, TotalHops: 0, Confidence: 1.0}, nil
	}

	// Get all table relationship edges for this datasource
	edges, err := r.getTableEdges(ctx, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load table edges: %w", err)
	}

	// Build adjacency list
	adjacency := make(map[string][]TableEdge)
	tableNameToID := make(map[string]uuid.UUID)

	for _, edge := range edges {
		adjacency[edge.SourceTableName] = append(adjacency[edge.SourceTableName], edge)
		tableNameToID[edge.SourceTableName] = edge.SourceTableID
		tableNameToID[edge.TargetTableName] = edge.TargetTableID
	}

	// BFS
	type queueItem struct {
		tableName string
		path      []TableEdge
		depth     int
	}

	visited := make(map[string]bool)
	queue := []queueItem{{tableName: fromTableName, path: nil, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.tableName] {
			continue
		}
		visited[current.tableName] = true

		if current.depth > maxDepth {
			continue
		}

		// Check if we reached the destination
		if current.tableName == toTableName {
			return r.buildJoinPath(current.path), nil
		}

		// Explore neighbors
		for _, edge := range adjacency[current.tableName] {
			if !visited[edge.TargetTableName] {
				newPath := append([]TableEdge{}, current.path...)
				newPath = append(newPath, edge)
				queue = append(queue, queueItem{
					tableName: edge.TargetTableName,
					path:      newPath,
					depth:     current.depth + 1,
				})
			}
		}

		// Also check reverse edges (bidirectional search)
		for _, edge := range edges {
			if edge.TargetTableName == current.tableName && !visited[edge.SourceTableName] {
				reverseEdge := TableEdge{
					SourceTableID:   edge.TargetTableID,
					TargetTableID:   edge.SourceTableID,
					SourceTableName: edge.TargetTableName,
					TargetTableName: edge.SourceTableName,
					JoinType:        "LEFT", // Reverse might need different type
					Cardinality:     reverseCardinality(edge.Cardinality),
				}
				// Build reverse join condition
				if edge.JoinCondition != "" {
					parts := strings.Split(edge.JoinCondition, " = ")
					if len(parts) == 2 {
						reverseEdge.JoinCondition = parts[1] + " = " + parts[0]
					}
				}
				newPath := append([]TableEdge{}, current.path...)
				newPath = append(newPath, reverseEdge)
				queue = append(queue, queueItem{
					tableName: edge.SourceTableName,
					path:      newPath,
					depth:     current.depth + 1,
				})
			}
		}
	}

	return nil, fmt.Errorf("no join path found between %s and %s within %d hops", fromTableName, toTableName, maxDepth)
}

func (r *JoinPathResolver) buildJoinPath(edges []TableEdge) *JoinPath {
	steps := make([]JoinPathStep, len(edges))
	totalConfidence := 1.0

	for i, edge := range edges {
		// Parse join condition to extract columns
		leftCol, rightCol := parseJoinCondition(edge.JoinCondition, edge.SourceTableName, edge.TargetTableName)

		steps[i] = JoinPathStep{
			LeftTable:   edge.SourceTableName,
			LeftAlias:   fmt.Sprintf("t%d", i),
			LeftColumn:  leftCol,
			RightTable:  edge.TargetTableName,
			RightAlias:  fmt.Sprintf("t%d", i+1),
			RightColumn: rightCol,
			JoinType:    edge.JoinType,
			Cardinality: edge.Cardinality,
		}
	}

	return &JoinPath{
		Steps:      steps,
		TotalHops:  len(edges),
		Confidence: totalConfidence,
	}
}

func (r *JoinPathResolver) getTableEdges(ctx context.Context, datasourceID uuid.UUID) ([]TableEdge, error) {
	query := `
		SELECT 
			ce.source_node_id,
			ce.target_node_id,
			src.node_name AS source_table_name,
			tgt.node_name AS target_table_name,
			COALESCE(ce.properties->>'join_condition', '') AS join_condition,
			COALESCE(ce.properties->>'join_type', 'left') AS join_type,
			COALESCE(ce.properties->>'cardinality', 'unknown') AS cardinality
		FROM catalog_edge ce
		JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
		JOIN catalog_node src ON ce.source_node_id = src.id
		JOIN catalog_node tgt ON ce.target_node_id = tgt.id
		WHERE ce.tenant_datasource_id = $1
		AND cet.edge_type_name = 'TABLE_RELATES_TO_TABLE'
	`

	var edges []TableEdge
	err := r.db.SelectContext(ctx, &edges, query, datasourceID)
	if err != nil {
		return nil, err
	}

	return edges, nil
}

// ============================================================================
// SQL Generation
// ============================================================================

// GenerateJoinSQL generates SQL FROM/JOIN clauses from a JoinPath
func (r *JoinPathResolver) GenerateJoinSQL(path *JoinPath, baseTable string) string {
	if path == nil || len(path.Steps) == 0 {
		return fmt.Sprintf("FROM %s AS t0", baseTable)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("FROM %s AS t0", baseTable))

	for i, step := range path.Steps {
		joinType := strings.ToUpper(step.JoinType)
		if joinType == "" {
			joinType = "LEFT"
		}

		builder.WriteString(fmt.Sprintf("\n%s JOIN %s AS t%d ON t%d.%s = t%d.%s",
			joinType,
			step.RightTable,
			i+1,
			i,
			step.LeftColumn,
			i+1,
			step.RightColumn,
		))
	}

	return builder.String()
}

// GenerateMultiTableBOSQL generates SQL for a BO that spans multiple tables
func (r *JoinPathResolver) GenerateMultiTableBOSQL(
	ctx context.Context,
	datasourceID uuid.UUID,
	drivingTable string,
	relatedTables []string,
	selectColumns map[string][]string, // table -> columns
) (string, error) {
	var allJoins []JoinPathStep

	for _, table := range relatedTables {
		if table == drivingTable {
			continue
		}

		path, err := r.ResolveJoinPath(ctx, datasourceID, drivingTable, table, 3)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path to %s: %w", table, err)
		}

		// Append steps (avoiding duplicates)
		for _, step := range path.Steps {
			exists := false
			for _, existing := range allJoins {
				if existing.RightTable == step.RightTable {
					exists = true
					break
				}
			}
			if !exists {
				allJoins = append(allJoins, step)
			}
		}
	}

	// Build SELECT clause
	var selectClauses []string

	// Get columns from driving table
	if cols, ok := selectColumns[drivingTable]; ok {
		for _, col := range cols {
			selectClauses = append(selectClauses, fmt.Sprintf("t0.%s", col))
		}
	}

	// Get columns from related tables
	for i, step := range allJoins {
		if cols, ok := selectColumns[step.RightTable]; ok {
			for _, col := range cols {
				selectClauses = append(selectClauses, fmt.Sprintf("t%d.%s", i+1, col))
			}
		}
	}

	// Build full SQL
	path := &JoinPath{Steps: allJoins, TotalHops: len(allJoins)}
	joinSQL := r.GenerateJoinSQL(path, drivingTable)

	if len(selectClauses) == 0 {
		selectClauses = append(selectClauses, "*")
	}

	return fmt.Sprintf("SELECT\n  %s\n%s",
		strings.Join(selectClauses, ",\n  "),
		joinSQL,
	), nil
}

// ============================================================================
// Helpers
// ============================================================================

func parseJoinCondition(condition, leftTable, rightTable string) (leftCol, rightCol string) {
	if condition == "" {
		return "id", "id" // Default
	}

	// Parse "left_table.col = right_table.col" format
	parts := strings.Split(condition, " = ")
	if len(parts) != 2 {
		return "id", "id"
	}

	// Extract column names
	leftParts := strings.Split(strings.TrimSpace(parts[0]), ".")
	rightParts := strings.Split(strings.TrimSpace(parts[1]), ".")

	if len(leftParts) >= 2 {
		leftCol = leftParts[len(leftParts)-1]
	} else {
		leftCol = leftParts[0]
	}

	if len(rightParts) >= 2 {
		rightCol = rightParts[len(rightParts)-1]
	} else {
		rightCol = rightParts[0]
	}

	return leftCol, rightCol
}

func reverseCardinality(cardinality string) string {
	switch cardinality {
	case "1:M":
		return "M:1"
	case "M:1":
		return "1:M"
	default:
		return cardinality
	}
}

// ValidateJoinPath checks if a join path is valid
func (r *JoinPathResolver) ValidateJoinPath(ctx context.Context, datasourceID uuid.UUID, path *JoinPath) error {
	if path == nil || len(path.Steps) == 0 {
		return nil
	}

	// Check that all tables in the path exist
	for _, step := range path.Steps {
		var exists bool
		err := r.db.GetContext(ctx, &exists, `
			SELECT EXISTS(
				SELECT 1 FROM catalog_node cn
				JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
				WHERE cn.tenant_datasource_id = $1
				AND cn.node_name = $2
				AND cnt.catalog_type_name IN ('physical_table', 'table')
			)
		`, datasourceID, step.RightTable)

		if err != nil {
			return fmt.Errorf("failed to validate table %s: %w", step.RightTable, err)
		}
		if !exists {
			return fmt.Errorf("table %s does not exist", step.RightTable)
		}
	}

	// Check for cycles
	visited := make(map[string]bool)
	for _, step := range path.Steps {
		if visited[step.RightTable] {
			return fmt.Errorf("cycle detected in join path at table %s", step.RightTable)
		}
		visited[step.RightTable] = true
	}

	return nil
}

// Ensure the types needed for serialization
func (p *JoinPath) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

func JoinPathFromJSON(data []byte) (*JoinPath, error) {
	var path JoinPath
	if err := json.Unmarshal(data, &path); err != nil {
		return nil, err
	}
	return &path, nil
}
