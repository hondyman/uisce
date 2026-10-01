package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// JoinPathResolver provides BFS-based join path resolution between tables
// using TABLE_RELATES_TO_TABLE edges from the catalog graph.
type JoinPathResolver struct {
	db *sqlx.DB

	edgeCacheMu sync.RWMutex
	edgeCache   map[uuid.UUID][]TableEdge // datasourceID -> all TABLE_RELATES_TO_TABLE edges
}

// NewJoinPathResolver creates a new join path resolver
func NewJoinPathResolver(db *sqlx.DB) *JoinPathResolver {
	return &JoinPathResolver{db: db, edgeCache: make(map[uuid.UUID][]TableEdge)}
}

// InvalidateDatasource drops the cached edge list for a datasource. Must be
// called whenever a TABLE_RELATES_TO_TABLE edge is created/updated, otherwise
// ResolveJoinPath keeps serving a stale graph.
func (r *JoinPathResolver) InvalidateDatasource(datasourceID uuid.UUID) {
	r.edgeCacheMu.Lock()
	defer r.edgeCacheMu.Unlock()
	delete(r.edgeCache, datasourceID)
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

// PathAnalysis is the single fold over a JoinPath's steps that both
// TraversalCardinality and RootOwnership used to compute separately, with
// two different ad-hoc unknown-hop policies between them (a bug: see
// TraversalCardinality's own doc comment). There is exactly one fold over
// JoinPathStep.Cardinality data in this package now; both facts come out
// of it together, sharing one "what does an unrecognized hop mean" rule,
// so the next fact anyone needs from a JoinPath (and there will be one)
// has one place to be added rather than a third ad-hoc loop with a third
// policy.
//
//   - Cardinality is TraversalCardinality's own value: "one", "many", or
//     "unresolved" - does the path fan out in the traversal (FROM-to-
//     target, i.e. "down") direction.
//   - Ownership is RootOwnership's own value: "unique", "shared", or
//     "unresolved" - is the target row reachable from more than one FROM
//     row, reading the same steps' UP-cardinality.
//
// These are independent axes, not two spellings of the same fact - see
// RootOwnership's doc comment for why "M:1" is "one"/"shared" at once,
// and why a consumer fix for one axis does not fix the other.
type PathAnalysis struct {
	Cardinality string
	Ownership   string
}

// Analyze runs the single fold. A definite "many" (down) or "shared" (up)
// verdict from a recognized hop always wins over "unresolved" from an
// unrecognized one elsewhere in the same path, on both axes
// independently - a known-bad hop is already at least as bad as anything
// an unknown hop could turn out to be, on whichever axis it was bad on.
func (p *JoinPath) Analyze() PathAnalysis {
	if p == nil {
		return PathAnalysis{Cardinality: "one", Ownership: "unique"}
	}
	cardinality := "one"
	ownership := "unique"
	unresolvedCardinality := false
	unresolvedOwnership := false
	for _, step := range p.Steps {
		switch step.Cardinality {
		case "1:1":
			// No fan-out either direction; nothing to update.
		case "1:M":
			cardinality = "many"
		case "M:1":
			ownership = "shared"
		case "M:M":
			cardinality = "many"
			ownership = "shared"
		default:
			unresolvedCardinality = true
			unresolvedOwnership = true
		}
	}
	if unresolvedCardinality && cardinality != "many" {
		cardinality = "unresolved"
	}
	if unresolvedOwnership && ownership != "shared" {
		ownership = "unresolved"
	}
	return PathAnalysis{Cardinality: cardinality, Ownership: ownership}
}

// TraversalCardinality classifies a join path relative to its FROM table:
// "one" means every hop is 1:1 or M:1, so following the path from a single
// FROM row lands on at most one row — a lookup, safe to flatten into the
// same result row. "many" means at least one hop is 1:M or M:M, so a single
// FROM row can fan out into multiple joined rows — a detail/child collection
// (PeopleSoft calls this a "scroll level") that callers must not silently
// flatten without either aggregating or nesting the result. "unresolved"
// means at least one hop's Cardinality couldn't be classified and no OTHER
// hop already proved "many" on its own - treated as NOT safely "one",
// since an unrecognized hop might be a fan-out hop the resolver simply
// couldn't name.
//
// Existing callers that only ever check `== "many"` are unaffected by
// "unresolved" existing as a third value: they were already treating an
// unrecognized hop the same as "one" (silently) before this value existed
// to say otherwise. A caller that wants the older, stricter true/false
// question should compare `!= "one"` instead of relying on "not many".
func (p *JoinPath) TraversalCardinality() string {
	return p.Analyze().Cardinality
}

// RootOwnership classifies a join path by whether the TARGET row (the far
// end of the path) has exactly one owning FROM row, or is reachable from
// more than one. It is TraversalCardinality's mirror over the SAME fold
// (see Analyze): that reads each step's DOWN-cardinality (does one
// FROM row expand to many target rows); this reads the same steps'
// UP-cardinality (does one target row trace back to many FROM rows).
//
// "unique" means every hop's up-cardinality is at most one - no "M:1" or
// "M:M" step anywhere, and every step's cardinality is one of the known
// values - so the target row has exactly one owning root and nothing can
// inflate it. "shared" means at least one hop has up-cardinality greater
// than one (an "M:1" step: many FROM-side rows map to one target row,
// e.g. many orders to one customer; or "M:M"), so the SAME target row is
// reachable from more than one root row. "unresolved" means at least one
// step's Cardinality is empty or not one of "1:1"/"1:M"/"M:1"/"M:M" - the
// resolver couldn't classify that hop - and no OTHER step already proved
// "shared" on its own.
//
// "unresolved" exists because the alternative is worse: falling through
// an unrecognized cardinality string to "unique" would tell a caller a
// column is safe to sum across rows when the truth is simply unknown.
// Absence of information must never read as evidence of safety - the
// cost of a false "unresolved" (one unnecessary pin, or a consumer that
// declines to roll up a column that was actually fine) is far smaller
// than the cost of a false "unique" (a silently inflated total). A
// definite "shared" step still wins over an unresolved one elsewhere in
// the same path, since "shared" is already at least as bad as anything
// "unresolved" could turn out to be.
//
// This is independent of TraversalCardinality/down-cardinality and of
// whether the path involves any related-BO join planning at all - a path
// with a single "M:1" hop and nothing else (customer -> region, say) is
// "one" under TraversalCardinality (no fan-out, never triggers aggregation
// logic) and "shared" under RootOwnership at the same time. TWO DIFFERENT
// ROOTS attached to the same "M:1" target both correctly show that target's
// value, so per-row output is fine; the hazard is downstream, in a caller
// that sums or averages the returned column ACROSS rows, since the shared
// target's value would then be counted once per owning root rather than
// once.
//
// A caveat for whatever eventually builds the consumer-side rule this
// field feeds: "M:1" and "M:M" are NOT interchangeable there. "M:1"
// leaves the join correct at root grain - only a roll-up PAST root grain
// is unsafe. "M:M" duplicates the ROOT's own attributes within the flat
// join itself, corrupting the join at root grain before any rollup is
// even considered. Both currently return "shared" from this function -
// that's correct, since both mean "reachable from more than one root" -
// but a consumer must not apply the same fix to both.
//
// CRITICAL for any consumer deciding whether to roll a column up ACROSS
// RETURNED ROWS: RootOwnership (and TraversalCardinality) are properties
// of ONE column's path. Whether summing a query's results across rows
// reconstructs a real total is a property of the QUERY - if ANY other
// selected column's path fans out (TraversalCardinality "many" or
// "unresolved"), every row this query returns is duplicated relative to
// root by THAT join, regardless of how clean this column's own path is.
// A per-column "unique"+"one" verdict is necessary, not sufficient, for
// roll-up safety; the sufficient condition folds every column's path in
// the query, not just the one being summed. See
// savedQueryApi.ts's isRowGrainIntact for where that fold has to live.
func (p *JoinPath) RootOwnership() string {
	return p.Analyze().Ownership
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
	r.edgeCacheMu.RLock()
	if cached, ok := r.edgeCache[datasourceID]; ok {
		r.edgeCacheMu.RUnlock()
		return cached, nil
	}
	r.edgeCacheMu.RUnlock()

	edges, err := r.loadTableEdges(ctx, datasourceID)
	if err != nil {
		return nil, err
	}

	r.edgeCacheMu.Lock()
	r.edgeCache[datasourceID] = edges
	r.edgeCacheMu.Unlock()

	return edges, nil
}

func (r *JoinPathResolver) loadTableEdges(ctx context.Context, datasourceID uuid.UUID) ([]TableEdge, error) {
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

// ResolveJoinPathBetweenBOs resolves a join path between two Business Objects
// by their catalog_node IDs, looking up each BO's driving table name before
// delegating to ResolveJoinPath. This is what Report Builder, Query Builder,
// and Page Studio should call once a user picks a related BO — they deal in
// BO IDs, not physical table names.
func (r *JoinPathResolver) ResolveJoinPathBetweenBOs(
	ctx context.Context,
	datasourceID, fromBONodeID, toBONodeID uuid.UUID,
	maxDepth int,
) (*JoinPath, error) {
	fromTable, err := r.boDrivingTableName(ctx, datasourceID, fromBONodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve driving table for BO %s: %w", fromBONodeID, err)
	}
	toTable, err := r.boDrivingTableName(ctx, datasourceID, toBONodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve driving table for BO %s: %w", toBONodeID, err)
	}

	return r.ResolveJoinPath(ctx, datasourceID, fromTable, toTable, maxDepth)
}

func (r *JoinPathResolver) boDrivingTableName(ctx context.Context, datasourceID, boNodeID uuid.UUID) (string, error) {
	var driverTableID uuid.UUID
	err := r.db.GetContext(ctx, &driverTableID, `
		SELECT (properties->>'driver_table_id')::uuid FROM catalog_node
		WHERE tenant_datasource_id = $1 AND id = $2
	`, datasourceID, boNodeID)
	if err != nil {
		return "", err
	}

	var tableName string
	err = r.db.GetContext(ctx, &tableName, `SELECT node_name FROM catalog_node WHERE id = $1`, driverTableID)
	if err != nil {
		return "", err
	}
	return tableName, nil
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
