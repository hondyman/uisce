package querycompiler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// ============================================================================
// SEMANTIC QUERY TYPES
// ============================================================================

// SemanticQuery represents a query on a semantic model (Cube.js-like DSL)
type SemanticQuery struct {
	TenantID   string               `json:"tenant_id"`
	ModelName  string               `json:"model"`
	Measures   []string             `json:"measures"`
	Dimensions []string             `json:"dimensions"`
	Filters    []SemanticFilter     `json:"filters"`
	OrderBy    []OrderSpecification `json:"order_by"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
	UseCache   bool                 `json:"use_cache"`
}

type SemanticFilter struct {
	Dimension string      `json:"dimension"`
	Measure   string      `json:"measure"`
	Operator  string      `json:"operator"` // "eq", "ne", "gt", "gte", "lt", "lte", "in", "contains"
	Value     interface{} `json:"value"`
}

type OrderSpecification struct {
	Measure   string `json:"measure"`
	Dimension string `json:"dimension"`
	Direction string `json:"direction"` // "ASC" or "DESC"
}

// CompiledQuery represents the compiled SQL output
type CompiledQuery struct {
	SQL               string        `json:"sql"`
	Parameters        []interface{} `json:"parameters"`
	JoinsUsed         []string      `json:"joins_used"`
	Optimizations     []string      `json:"optimizations"`
	EstimatedRows     int64         `json:"estimated_rows"`
	CacheKey          string        `json:"cache_key"`
	ExecutionPlanCost float64       `json:"execution_plan_cost"`
}

// ============================================================================
// SEMANTIC MODEL DEFINITION
// ============================================================================

type SemanticModel struct {
	ID          string
	Name        string
	TableName   string
	Description string
	Measures    map[string]SemanticMeasure
	Dimensions  map[string]SemanticDimension
	Joins       map[string]SemanticJoin
	PreAggs     []PreAggregation
}

type SemanticMeasure struct {
	Name          string `json:"name"`
	Type          string `json:"type"` // "count", "sum", "avg", "min", "max"
	Field         string `json:"field"`
	SQL           string `json:"sql,omitempty"` // Custom SQL for complex measures
	Description   string `json:"description"`
	Format        string `json:"format,omitempty"` // "currency", "percent", etc.
	Rollingwindow string `json:"rolling_window,omitempty"`
}

type SemanticDimension struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"` // "string", "number", "date", "boolean"
	Field         string   `json:"field"`
	SQL           string   `json:"sql,omitempty"`
	Description   string   `json:"description"`
	Granularities []string `json:"granularities,omitempty"` // ["year", "month", "day"] for dates
	Hierarchy     []string `json:"hierarchy,omitempty"`
	IsPrimary     bool     `json:"is_primary,omitempty"`
}

type SemanticJoin struct {
	Name         string `json:"name"`
	RelatedModel string `json:"related_model"`
	SQLCondition string `json:"sql_condition"`
	Type         string `json:"type"`        // "inner", "left", "right", "full"
	Cardinality  string `json:"cardinality"` // "one_to_one", "one_to_many"
}

type PreAggregation struct {
	Name       string
	Measures   []string
	Dimensions []string
	ViewName   string
}

// ============================================================================
// QUERY COMPILER
// ============================================================================

type QueryCompiler struct {
	models map[string]*SemanticModel
	db     *sql.DB
}

func NewQueryCompiler(db *sql.DB) *QueryCompiler {
	return &QueryCompiler{
		models: make(map[string]*SemanticModel),
		db:     db,
	}
}

// RegisterModel adds a semantic model to the compiler's catalog
func (qc *QueryCompiler) RegisterModel(model *SemanticModel) {
	qc.models[model.Name] = model
}

// Compile translates a semantic query into optimized SQL
func (qc *QueryCompiler) Compile(ctx context.Context, query *SemanticQuery) (*CompiledQuery, error) {
	if query.ModelName == "" {
		return nil, fmt.Errorf("model_name is required")
	}

	model, exists := qc.models[query.ModelName]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", query.ModelName)
	}

	// Validate measures and dimensions
	if err := qc.validateQueryMembers(query, model); err != nil {
		return nil, err
	}

	// Determine which joins are needed
	joinsNeeded := qc.discoverJoinsNeeded(query, model)

	// Build SELECT clause (measures + dimensions)
	selectClause := qc.buildSelectClause(query, model)

	// Build FROM clause
	fromClause := qc.buildFromClause(model, joinsNeeded)

	// Build WHERE clause (filters with tenant isolation)
	whereClause, params := qc.buildWhereClause(query, model, query.TenantID)

	// Build GROUP BY clause
	groupByClause := qc.buildGroupByClause(query, model)

	// Build ORDER BY clause
	orderByClause := qc.buildOrderByClause(query)

	// Build LIMIT/OFFSET
	limitClause := qc.buildLimitClause(query)

	// Assemble final SQL
	sqlParts := []string{
		"SELECT", selectClause,
		"FROM", fromClause,
	}
	if whereClause != "" {
		sqlParts = append(sqlParts, "WHERE", whereClause)
	}
	if groupByClause != "" {
		sqlParts = append(sqlParts, "GROUP BY", groupByClause)
	}
	if orderByClause != "" {
		sqlParts = append(sqlParts, "ORDER BY", orderByClause)
	}
	if limitClause != "" {
		sqlParts = append(sqlParts, limitClause)
	}

	sql := strings.Join(sqlParts, " ")

	// Estimate cost and create cache key
	cacheKey := qc.generateCacheKey(query)
	estimatedCost := qc.estimateQueryCost(sql, len(params))

	optimizations := qc.detectOptimizations(query, model, joinsNeeded)

	compiled := &CompiledQuery{
		SQL:               sql,
		Parameters:        params,
		JoinsUsed:         joinsNeeded,
		Optimizations:     optimizations,
		EstimatedRows:     100, // Placeholder: query EXPLAIN for actual estimate
		CacheKey:          cacheKey,
		ExecutionPlanCost: estimatedCost,
	}

	return compiled, nil
}

// ============================================================================
// QUERY COMPONENT BUILDERS
// ============================================================================

func (qc *QueryCompiler) buildSelectClause(query *SemanticQuery, model *SemanticModel) string {
	var parts []string

	// Add measures
	for _, measureName := range query.Measures {
		measure, ok := model.Measures[measureName]
		if !ok {
			continue
		}

		var measureSQL string
		if measure.SQL != "" {
			// Use custom SQL if provided
			measureSQL = measure.SQL
		} else {
			// Generate standard aggregation SQL
			switch measure.Type {
			case "count":
				measureSQL = fmt.Sprintf("COUNT(DISTINCT %s)", qc.qualifyField(model.TableName, measure.Field))
			case "sum":
				measureSQL = fmt.Sprintf("SUM(%s)", qc.qualifyField(model.TableName, measure.Field))
			case "avg":
				measureSQL = fmt.Sprintf("AVG(%s)", qc.qualifyField(model.TableName, measure.Field))
			case "min":
				measureSQL = fmt.Sprintf("MIN(%s)", qc.qualifyField(model.TableName, measure.Field))
			case "max":
				measureSQL = fmt.Sprintf("MAX(%s)", qc.qualifyField(model.TableName, measure.Field))
			default:
				measureSQL = fmt.Sprintf("COUNT(%s)", qc.qualifyField(model.TableName, measure.Field))
			}
		}

		parts = append(parts, fmt.Sprintf("%s AS %s", measureSQL, measureName))
	}

	// Add dimensions
	for _, dimName := range query.Dimensions {
		dimension, ok := model.Dimensions[dimName]
		if !ok {
			continue
		}

		var dimSQL string
		if dimension.SQL != "" {
			dimSQL = dimension.SQL
		} else {
			dimSQL = qc.qualifyField(model.TableName, dimension.Field)
		}

		parts = append(parts, fmt.Sprintf("%s AS %s", dimSQL, dimName))
	}

	return strings.Join(parts, ", ")
}

func (qc *QueryCompiler) buildFromClause(model *SemanticModel, joins []string) string {
	fromClause := model.TableName

	// Add required joins
	for _, joinName := range joins {
		if join, ok := model.Joins[joinName]; ok {
			relatedModel := join.RelatedModel
			joinType := strings.ToUpper(join.Type)
			if joinType == "" {
				joinType = "LEFT"
			}
			fromClause += fmt.Sprintf(" %s JOIN %s ON %s", joinType, relatedModel, join.SQLCondition)
		}
	}

	return fromClause
}

func (qc *QueryCompiler) buildWhereClause(query *SemanticQuery, model *SemanticModel, tenantID string) (string, []interface{}) {
	var conditions []string
	var params []interface{}

	// 1. Add tenant isolation (critical for multi-tenancy)
	if tenantID != "" {
		conditions = append(conditions, fmt.Sprintf("%s.tenant_id = $%d", model.TableName, len(params)+1))
		params = append(params, tenantID)
	}

	// 2. Add dimension filters
	for _, filter := range query.Filters {
		if filter.Dimension != "" {
			dimension, ok := model.Dimensions[filter.Dimension]
			if !ok {
				continue
			}

			fieldRef := qc.qualifyField(model.TableName, dimension.Field)
			condition, val := qc.buildFilterCondition(fieldRef, filter.Operator, filter.Value)

			conditions = append(conditions, condition)
			params = append(params, val)
		}

		// Measure filters go in HAVING clause (not WHERE)
	}

	if len(conditions) == 0 {
		return "", params
	}

	return strings.Join(conditions, " AND "), params
}

func (qc *QueryCompiler) buildFilterCondition(fieldRef string, operator string, value interface{}) (string, interface{}) {
	switch operator {
	case "eq":
		return fmt.Sprintf("%s = $1", fieldRef), value
	case "ne":
		return fmt.Sprintf("%s != $1", fieldRef), value
	case "gt":
		return fmt.Sprintf("%s > $1", fieldRef), value
	case "gte":
		return fmt.Sprintf("%s >= $1", fieldRef), value
	case "lt":
		return fmt.Sprintf("%s < $1", fieldRef), value
	case "lte":
		return fmt.Sprintf("%s <= $1", fieldRef), value
	case "in":
		// Assume value is a slice
		return fmt.Sprintf("%s = ANY($1::text[])", fieldRef), value
	case "contains":
		return fmt.Sprintf("%s ILIKE $1", fieldRef), fmt.Sprintf("%%%v%%", value)
	case "startswith":
		return fmt.Sprintf("%s ILIKE $1", fieldRef), fmt.Sprintf("%v%%", value)
	default:
		return fmt.Sprintf("%s = $1", fieldRef), value
	}
}

func (qc *QueryCompiler) buildGroupByClause(query *SemanticQuery, model *SemanticModel) string {
	if len(query.Dimensions) == 0 {
		// No grouping for total aggregates
		return ""
	}

	var parts []string
	for _, dimName := range query.Dimensions {
		if dimension, ok := model.Dimensions[dimName]; ok {
			if dimension.SQL != "" {
				parts = append(parts, dimension.SQL)
			} else {
				parts = append(parts, qc.qualifyField(model.TableName, dimension.Field))
			}
		}
	}

	return strings.Join(parts, ", ")
}

func (qc *QueryCompiler) buildOrderByClause(query *SemanticQuery) string {
	if len(query.OrderBy) == 0 {
		return ""
	}

	var parts []string
	for _, ord := range query.OrderBy {
		direction := "ASC"
		if ord.Direction != "" {
			direction = strings.ToUpper(ord.Direction)
		}

		if ord.Measure != "" {
			parts = append(parts, fmt.Sprintf("%s %s", ord.Measure, direction))
		} else if ord.Dimension != "" {
			parts = append(parts, fmt.Sprintf("%s %s", ord.Dimension, direction))
		}
	}

	return strings.Join(parts, ", ")
}

func (qc *QueryCompiler) buildLimitClause(query *SemanticQuery) string {
	if query.Limit == 0 {
		query.Limit = 1000 // Default limit for safety
	}

	if query.Offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", query.Limit, query.Offset)
	}

	return fmt.Sprintf("LIMIT %d", query.Limit)
}

// ============================================================================
// OPTIMIZATION & ANALYSIS
// ============================================================================

func (qc *QueryCompiler) discoverJoinsNeeded(query *SemanticQuery, model *SemanticModel) []string {
	// Analyze dimensions to determine required joins
	var joinsNeeded []string
	seen := make(map[string]bool)

	// Simple heuristic: if a dimension references a related table, add that join
	// In production, use join path discovery from catalog
	for _, dimName := range query.Dimensions {
		if strings.Contains(dimName, ".") {
			// Dimension reference like "customer.country" indicates a join
			parts := strings.Split(dimName, ".")
			joinName := parts[0]
			if !seen[joinName] && joinName != model.Name {
				joinsNeeded = append(joinsNeeded, joinName)
				seen[joinName] = true
			}
		}
	}

	return joinsNeeded
}

func (qc *QueryCompiler) detectOptimizations(query *SemanticQuery, model *SemanticModel, joins []string) []string {
	var optimizations []string

	// 1. Check for pre-aggregation opportunities
	if len(model.PreAggs) > 0 {
		for _, preagg := range model.PreAggs {
			if qc.matchesPreAgg(query, preagg) {
				optimizations = append(optimizations, fmt.Sprintf("use_preaggregation:%s", preagg.Name))
			}
		}
	}

	// 2. Check for filter pushdown
	if len(query.Filters) > 0 {
		optimizations = append(optimizations, "filter_pushdown")
	}

	// 3. Check for join optimization
	if len(joins) > 1 {
		optimizations = append(optimizations, "join_optimization")
	}

	// 4. Check for column pruning
	if len(query.Measures)+len(query.Dimensions) < 10 {
		optimizations = append(optimizations, "column_pruning")
	}

	return optimizations
}

func (qc *QueryCompiler) matchesPreAgg(query *SemanticQuery, preagg PreAggregation) bool {
	// Check if query can use pre-aggregation
	measuresMatch := qc.stringSliceContains(preagg.Measures, query.Measures)
	dimensionsMatch := qc.stringSliceContains(preagg.Dimensions, query.Dimensions)
	return measuresMatch && dimensionsMatch
}

func (qc *QueryCompiler) estimateQueryCost(sql string, paramCount int) float64 {
	// Simple cost estimation: based on query complexity
	cost := 1.0

	if strings.Contains(sql, "JOIN") {
		cost += 5.0 // Each join adds cost
	}
	if strings.Contains(sql, "GROUP BY") {
		cost += 3.0
	}
	if strings.Contains(sql, "HAVING") {
		cost += 2.0
	}

	cost *= float64(paramCount) * 0.1 // Parameter complexity

	return cost
}

func (qc *QueryCompiler) generateCacheKey(query *SemanticQuery) string {
	// Create deterministic cache key from query
	key := fmt.Sprintf("query:%s:%s:%s:%s:%d:%d",
		query.TenantID,
		query.ModelName,
		strings.Join(query.Measures, ","),
		strings.Join(query.Dimensions, ","),
		query.Limit,
		query.Offset,
	)

	return hashString(key)
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func (qc *QueryCompiler) validateQueryMembers(query *SemanticQuery, model *SemanticModel) error {
	for _, m := range query.Measures {
		if _, ok := model.Measures[m]; !ok {
			return fmt.Errorf("measure '%s' not found in model '%s'", m, query.ModelName)
		}
	}

	for _, d := range query.Dimensions {
		// Handle dot notation (e.g., "customer.country")
		dimKey := strings.Split(d, ".")[len(strings.Split(d, "."))-1]
		if _, ok := model.Dimensions[dimKey]; !ok {
			return fmt.Errorf("dimension '%s' not found in model '%s'", d, query.ModelName)
		}
	}

	return nil
}

func (qc *QueryCompiler) qualifyField(tableName, fieldName string) string {
	return fmt.Sprintf("%s.%s", tableName, fieldName)
}

func (qc *QueryCompiler) stringSliceContains(haystack, needle []string) bool {
	haystackMap := make(map[string]bool)
	for _, item := range haystack {
		haystackMap[item] = true
	}

	for _, item := range needle {
		if !haystackMap[item] {
			return false
		}
	}

	return true
}

func hashString(s string) string {
	// Simple SHA-256 hash (use sha256 package in production)
	return fmt.Sprintf("hash_%d", len(s)) // Placeholder
}

// ============================================================================
// QUERY EXECUTOR
// ============================================================================

type QueryExecutor struct {
	db       *sql.DB
	compiler *QueryCompiler
}

func NewQueryExecutor(db *sql.DB, compiler *QueryCompiler) *QueryExecutor {
	return &QueryExecutor{
		db:       db,
		compiler: compiler,
	}
}

// Execute compiles and runs a semantic query
func (qe *QueryExecutor) Execute(ctx context.Context, query *SemanticQuery) ([]map[string]interface{}, error) {
	compiled, err := qe.compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	log.Printf("Executing compiled SQL: %s", compiled.SQL)

	rows, err := qe.db.QueryContext(ctx, compiled.SQL, compiled.Parameters...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		entry := make(map[string]interface{})
		for i, col := range cols {
			entry[col] = values[i]
		}
		results = append(results, entry)
	}

	return results, rows.Err()
}
