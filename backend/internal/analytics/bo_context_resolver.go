package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/cbo"
	"github.com/jmoiron/sqlx"
)

// BOContextResolver resolves calculations and terms using BO context
type BOContextResolver struct {
	db           *sqlx.DB
	graphService *SemanticGraphService
	planner      cbo.QueryPlanner
}

// NewBOContextResolver creates a new resolver
func NewBOContextResolver(
	db *sqlx.DB,
	graphService *SemanticGraphService,
) *BOContextResolver {
	return &BOContextResolver{
		db:           db,
		graphService: graphService,
	}
}

// SetPlanner sets the query planner
func (r *BOContextResolver) SetPlanner(planner cbo.QueryPlanner) {
	r.planner = planner
}

// ResolvedTerm represents a term resolved to physical columns
type ResolvedTerm struct {
	TermID         string `json:"term_id"`
	TermName       string `json:"term_name"`
	PhysicalTable  string `json:"physical_table"`
	PhysicalColumn string `json:"physical_column"`
	SQLFragment    string `json:"sql_fragment"`
}

// ResolvedCalculation represents a calculation resolved to SQL
type ResolvedCalculation struct {
	CalcID        string         `json:"calc_id"`
	CalcName      string         `json:"calc_name"`
	ExpressionDSL string         `json:"expression_dsl"`
	ResolvedSQL   string         `json:"resolved_sql"`
	Dependencies  []ResolvedTerm `json:"dependencies"`
	RequiredJoins []JoinClause   `json:"required_joins"`
}

// JoinClause represents a SQL join
type JoinClause struct {
	LeftTable   string `json:"left_table"`
	LeftColumn  string `json:"left_column"`
	RightTable  string `json:"right_table"`
	RightColumn string `json:"right_column"`
	JoinType    string `json:"join_type"`
}

// BOContext contains the context for resolution
type BOContext struct {
	BOID         uuid.UUID `json:"bo_id"`
	BOName       string    `json:"bo_name"`
	DrivingTable string    `json:"driving_table"`
	Dialect      string    `json:"dialect"` // postgres, snowflake, starrocks
	TenantID     uuid.UUID `json:"tenant_id"`
	DatasourceID uuid.UUID `json:"datasource_id"`
}

// ResolveTerm resolves a semantic term to physical column using BO context
func (r *BOContextResolver) ResolveTerm(termName string, ctx BOContext) (*ResolvedTerm, error) {
	// Get the term node
	termNode, err := r.graphService.GetNodeByName(NodeTypeSemanticTerm, termName, ctx.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get term: %w", err)
	}
	if termNode == nil {
		return nil, fmt.Errorf("term not found: %s", termName)
	}

	// Get physical mappings from config
	config := termNode.Config
	var mappings []interface{}

	if mappingsRaw, ok := config["physical_mappings"]; ok {
		if ms, ok := mappingsRaw.([]interface{}); ok {
			mappings = ms
		}
	}

	// Fallback to singular physical_mapping
	if len(mappings) == 0 {
		if mappingRaw, ok := config["physical_mapping"]; ok {
			if m, ok := mappingRaw.(map[string]interface{}); ok {
				mappings = []interface{}{m}
			}
		}
	}

	if len(mappings) == 0 {
		return nil, fmt.Errorf("term has no physical mappings: %s", termName)
	}

	// Find the mapping that matches the BO driving table
	var selectedMapping map[string]interface{}
	var defaultMapping map[string]interface{}

	for _, m := range mappings {
		mapping, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		table, _ := mapping["table"].(string)
		isDefault, _ := mapping["is_default"].(bool)

		if table == ctx.DrivingTable {
			selectedMapping = mapping
			break
		}

		if isDefault {
			defaultMapping = mapping
		}
	}

	// Use selected mapping, or fall back to default
	if selectedMapping == nil {
		selectedMapping = defaultMapping
	}

	if selectedMapping == nil && len(mappings) > 0 {
		// Use first mapping if no other match
		selectedMapping, _ = mappings[0].(map[string]interface{})
	}

	if selectedMapping == nil {
		return nil, fmt.Errorf("no suitable mapping found for term: %s", termName)
	}

	table, _ := selectedMapping["table"].(string)
	column, _ := selectedMapping["column"].(string)

	return &ResolvedTerm{
		TermID:         termNode.ID.String(),
		TermName:       termName,
		PhysicalTable:  table,
		PhysicalColumn: column,
		SQLFragment:    fmt.Sprintf("%s.%s", table, column),
	}, nil
}

// ResolveCalculation resolves a calculation to SQL using BO context
func (r *BOContextResolver) ResolveCalculation(calcName string, ctx BOContext) (*ResolvedCalculation, error) {
	// Get the calculation node
	calcNode, err := r.graphService.GetNodeByName(NodeTypeCalculationTerm, calcName, ctx.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get calculation: %w", err)
	}
	if calcNode == nil {
		return nil, fmt.Errorf("calculation not found: %s", calcName)
	}

	// Get DSL expression from config
	config := calcNode.Config
	dsl, _ := config["expression_dsl"].(string)
	if dsl == "" {
		return nil, fmt.Errorf("calculation has no DSL expression: %s", calcName)
	}

	// Get dependencies from config
	depsRaw, _ := config["dependencies"].([]interface{})

	var resolvedTerms []ResolvedTerm
	var requiredJoins []JoinClause
	resolvedSQL := dsl

	// Resolve each dependency
	for _, depRaw := range depsRaw {
		dep, ok := depRaw.(map[string]interface{})
		if !ok {
			continue
		}

		depType, _ := dep["type"].(string)
		depRef, _ := dep["ref"].(string)

		switch depType {
		case "term":
			// Resolve term to physical column
			resolvedTerm, err := r.ResolveTerm(depRef, ctx)
			if err != nil {
				continue // Skip unresolvable terms
			}
			resolvedTerms = append(resolvedTerms, *resolvedTerm)

			// Replace term reference in DSL with SQL fragment
			resolvedSQL = strings.ReplaceAll(resolvedSQL, depRef, resolvedTerm.SQLFragment)

		case "calc":
			// Recursively resolve calculation
			nestedCalc, err := r.ResolveCalculation(depRef, ctx)
			if err != nil {
				continue // Skip unresolvable calcs
			}

			// Replace calc reference with resolved SQL (wrapped in parentheses)
			resolvedSQL = strings.ReplaceAll(resolvedSQL, depRef, "("+nestedCalc.ResolvedSQL+")")

			// Merge dependencies
			resolvedTerms = append(resolvedTerms, nestedCalc.Dependencies...)
			requiredJoins = append(requiredJoins, nestedCalc.RequiredJoins...)

		case "table":
			// Add join for table dependency
			join := r.buildJoin(ctx.DrivingTable, depRef, ctx.DatasourceID)
			if join != nil {
				requiredJoins = append(requiredJoins, *join)
			}
		}
	}

	// Convert DSL to SQL
	resolvedSQL = r.dslToSQL(resolvedSQL, ctx.Dialect)

	return &ResolvedCalculation{
		CalcID:        calcNode.ID.String(),
		CalcName:      calcName,
		ExpressionDSL: dsl,
		ResolvedSQL:   resolvedSQL,
		Dependencies:  resolvedTerms,
		RequiredJoins: requiredJoins,
	}, nil
}

// dslToSQL converts DSL expression to SQL
func (r *BOContextResolver) dslToSQL(dsl string, dialect string) string {
	sql := dsl

	// Convert LISP-like functions to SQL
	conversions := map[string]string{
		"(divide ":   "(",
		"(subtract ": "(",
		"(add ":      "(",
		"(multiply ": "(",
		"(sum ":      "SUM(",
		"(avg ":      "AVG(",
		"(min ":      "MIN(",
		"(max ":      "MAX(",
		"(count ":    "COUNT(",
		"(power ":    "POWER(",
		"(sqrt ":     "SQRT(",
		"(abs ":      "ABS(",
		"(round ":    "ROUND(",
	}

	for dslFunc, sqlFunc := range conversions {
		sql = strings.ReplaceAll(sql, dslFunc, sqlFunc)
	}

	// Convert operators
	// (divide a b) → (a / b)
	// This is a simplified conversion - a real parser would be more robust
	sql = r.convertOperators(sql)

	// Apply safe division wrapper if needed
	if dialect == "postgres" && strings.Contains(sql, "/") {
		sql = r.applySafeDivision(sql, dialect)
	}

	return sql
}

// convertOperators converts DSL operators to SQL
func (r *BOContextResolver) convertOperators(sql string) string {
	// Simple operator conversions
	// These would be better handled by a proper parser

	// Split by spaces and rebuild
	// For now, return as-is with basic cleanup
	sql = strings.ReplaceAll(sql, "divide", "/")
	sql = strings.ReplaceAll(sql, "subtract", "-")
	sql = strings.ReplaceAll(sql, "add", "+")
	sql = strings.ReplaceAll(sql, "multiply", "*")

	return sql
}

// applySafeDivision wraps division in NULLIF to prevent divide-by-zero
func (r *BOContextResolver) applySafeDivision(sql string, dialect string) string {
	// For now, return as-is
	// A real implementation would parse and wrap divisors with NULLIF
	return sql
}

// buildJoin creates a join clause between two tables
func (r *BOContextResolver) buildJoin(leftTable string, rightTable string, datasourceID uuid.UUID) *JoinClause {
	// Look up FK relationship between tables
	var fk struct {
		LeftColumn  string `db:"left_column"`
		RightColumn string `db:"right_column"`
	}

	err := r.db.Get(&fk, `
		SELECT 
			e.properties->>'left_column' as left_column,
			e.properties->>'right_column' as right_column
		FROM catalog_edge e
		JOIN catalog_node left_tbl ON left_tbl.id = e.source_node_id
		JOIN catalog_node right_tbl ON right_tbl.id = e.target_node_id
		WHERE left_tbl.node_name = $1
		AND right_tbl.node_name = $2
		AND e.edge_type = 'FK_TO'
		LIMIT 1
	`, leftTable, rightTable)

	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return nil
	}

	return &JoinClause{
		LeftTable:   leftTable,
		LeftColumn:  fk.LeftColumn,
		RightTable:  rightTable,
		RightColumn: fk.RightColumn,
		JoinType:    "LEFT",
	}
}

// GetBOContext loads BO context from the graph
func (r *BOContextResolver) GetBOContext(boName string, tenantID uuid.UUID, datasourceID uuid.UUID, dialect string) (*BOContext, error) {
	boNode, err := r.graphService.GetNodeByName(NodeTypeBusinessObject, boName, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get BO: %w", err)
	}
	if boNode == nil {
		return nil, fmt.Errorf("BO not found: %s", boName)
	}

	// Get driving table from properties
	drivingTable, _ := boNode.Properties["driving_table"].(string)

	return &BOContext{
		BOID:         boNode.ID,
		BOName:       boName,
		DrivingTable: drivingTable,
		Dialect:      dialect,
		TenantID:     tenantID,
		DatasourceID: datasourceID,
	}, nil
}

// GenerateBOSQL generates full SQL for a BO with selected terms and calculations
func (r *BOContextResolver) GenerateBOSQL(ctx BOContext, termNames []string, calcNames []string) (string, error) {
	var selectClauses []string
	var joins []JoinClause

	// Resolve terms
	for _, termName := range termNames {
		resolved, err := r.ResolveTerm(termName, ctx)
		if err != nil {
			continue
		}
		selectClauses = append(selectClauses, fmt.Sprintf("%s AS %s", resolved.SQLFragment, termName))
	}

	// Resolve calculations
	for _, calcName := range calcNames {
		resolved, err := r.ResolveCalculation(calcName, ctx)
		if err != nil {
			continue
		}
		selectClauses = append(selectClauses, fmt.Sprintf("(%s) AS %s", resolved.ResolvedSQL, calcName))
		joins = append(joins, resolved.RequiredJoins...)
	}

	// Build SQL
	sql := fmt.Sprintf("SELECT\n  %s\nFROM %s",
		strings.Join(selectClauses, ",\n  "),
		ctx.DrivingTable,
	)

	// Add joins
	if len(joins) > 0 {
		joinClauses := []string{}
		for _, j := range joins {
			joinClauses = append(joinClauses, fmt.Sprintf(
				"%s JOIN %s ON %s.%s = %s.%s",
				j.JoinType, j.RightTable,
				j.LeftTable, j.LeftColumn,
				j.RightTable, j.RightColumn,
			))
		}
		sql += "\n" + strings.Join(joinClauses, "\n")
	}

	return sql, nil
}

// BOCalculation represents a calculation assigned to a BO
type BOCalculation struct {
	CalcID   string `json:"calc_id"`
	CalcName string `json:"calc_name"`
}

// GetBOCalculations retrieves calculations assigned to a BO
func (r *BOContextResolver) GetBOCalculations(boID uuid.UUID) ([]BOCalculation, error) {
	edges, err := r.graphService.GetEdgesByType(boID, EdgeTypeBOHasCalc)
	if err != nil {
		return nil, err
	}

	var calcs []BOCalculation
	for _, edge := range edges {
		var nodeName string
		err := r.db.Get(&nodeName, `SELECT node_name FROM catalog_node WHERE id = $1`, edge.TargetNodeID)
		if err != nil {
			continue
		}
		calcs = append(calcs, BOCalculation{
			CalcID:   edge.TargetNodeID.String(),
			CalcName: nodeName,
		})
	}

	return calcs, nil
}

// BOTerm represents a term assigned to a BO
type BOTerm struct {
	TermID   string `json:"term_id"`
	TermName string `json:"term_name"`
}

// GetBOTerms retrieves terms assigned to a BO
func (r *BOContextResolver) GetBOTerms(boID uuid.UUID) ([]BOTerm, error) {
	edges, err := r.graphService.GetEdgesByType(boID, EdgeTypeBOHasTerm)
	if err != nil {
		return nil, err
	}

	var terms []BOTerm
	for _, edge := range edges {
		var nodeName string
		err := r.db.Get(&nodeName, `SELECT node_name FROM catalog_node WHERE id = $1`, edge.TargetNodeID)
		if err != nil {
			continue
		}
		terms = append(terms, BOTerm{
			TermID:   edge.TargetNodeID.String(),
			TermName: nodeName,
		})
	}

	return terms, nil
}

// AssignCalculationToBO creates a BO_HAS_CALC edge
func (r *BOContextResolver) AssignCalculationToBO(boID uuid.UUID, calcID uuid.UUID, tenantID uuid.UUID, datasourceID uuid.UUID) error {
	_, err := r.graphService.CreateEdge(boID, calcID, EdgeTypeBOHasCalc, tenantID, datasourceID, nil)
	return err
}

// AssignTermToBO creates a BO_HAS_TERM edge
func (r *BOContextResolver) AssignTermToBO(boID uuid.UUID, termID uuid.UUID, tenantID uuid.UUID, datasourceID uuid.UUID) error {
	_, err := r.graphService.CreateEdge(boID, termID, EdgeTypeBOHasTerm, tenantID, datasourceID, nil)
	return err
}

// CreateTermMapping creates a TERM_MAPS_TO_COLUMN edge
func (r *BOContextResolver) CreateTermMapping(termID uuid.UUID, columnID uuid.UUID, tenantID uuid.UUID, datasourceID uuid.UUID, properties map[string]interface{}) error {
	_, err := r.graphService.CreateEdge(termID, columnID, EdgeTypeTermMapsToColumn, tenantID, datasourceID, properties)
	return err
}

// BOSQLRequest represents a request to resolve BO SQL
type BOSQLRequest struct {
	Env           string                 `json:"env"`
	TenantID      *uuid.UUID             `json:"tenant_id,omitempty"`
	DatasourceID  *uuid.UUID             `json:"datasource_id,omitempty"`
	BOName        string                 `json:"bo_name"`
	EndpointID    *uuid.UUID             `json:"endpoint_id,omitempty"` // For API SLOs/ASO
	Filters       map[string]interface{} `json:"filters,omitempty"`
	GroupBy       []string               `json:"group_by,omitempty"`
	Measures      []string               `json:"measures,omitempty"`
	CurrentUserID string                 `json:"current_user_id,omitempty"`
	// Region is required for all semantic runtime operations
	Region string `json:"region,omitempty"`
	// Optional snapshot the caller may provide
	Snapshot *audit.SemanticSnapshot `json:"snapshot,omitempty"`
}

// ResolveQuery resolves a BO query using the CBO planner
func (r *BOContextResolver) ResolveQuery(ctx context.Context, req BOSQLRequest) (string, cbo.QueryPlanMetadata, error) {
	if r.planner == nil {
		// Fallback to legacy resolution if planner not configured
		// But for now, let's assume planner is there, or return error
		return "", cbo.QueryPlanMetadata{}, fmt.Errorf("planner not configured")
	}

	// Build PlanContext
	pc := cbo.PlanContext{
		Env:           req.Env,
		TenantID:      req.TenantID,
		DatasourceID:  req.DatasourceID,
		BOName:        req.BOName,
		Filters:       req.Filters,
		GroupBy:       req.GroupBy,
		Measures:      req.Measures,
		CurrentUserID: req.CurrentUserID,
		RequestedAt:   time.Now(),
		Region:        req.Region,
		Snapshot:      req.Snapshot,
	}

	// Plan the query
	plan, err := r.planner.Plan(ctx, pc)
	if err != nil {
		return "", cbo.QueryPlanMetadata{}, err
	}

	// Create metadata
	meta := cbo.QueryPlanMetadata{
		PlanType:            plan.PlanType,
		PreAggName:          plan.PreAggName,
		EntitlementStrategy: plan.EntitlementStrategy,
		Cost:                plan.Cost,
		CandidatesEvaluated: plan.CandidatesEvaluated,
		PlanningTimeMs:      plan.PlanningTimeMs,
	}

	return plan.SQL, meta, nil
}
