package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ============================================================================
// DATA SOURCE ROUTER
// Routes queries to Ignite (real-time) or StarRocks (analytics)
// ============================================================================

// DataSource represents available data sources
type DataSource string

const (
	DataSourceIgnite    DataSource = "ignite"
	DataSourceStarRocks DataSource = "starrocks"
	DataSourcePostgres  DataSource = "postgres" // fallback
)

// QueryType classifies query intent
type QueryType string

const (
	QueryTypeRealTime   QueryType = "realtime"
	QueryTypeAnalytics  QueryType = "analytics"
	QueryTypeHistorical QueryType = "historical"
)

// DataSourceRouter routes queries to optimal data source
type DataSourceRouter struct {
	igniteDB    *sql.DB
	starrocksDB *sql.DB
	postgresDB  *sql.DB
	logger      *zap.Logger
}

// NewDataSourceRouter creates a new router
func NewDataSourceRouter(igniteDB, starrocksDB, postgresDB *sql.DB) *DataSourceRouter {
	logger, _ := zap.NewProduction()
	return &DataSourceRouter{
		igniteDB:    igniteDB,
		starrocksDB: starrocksDB,
		postgresDB:  postgresDB,
		logger:      logger,
	}
}

// QueryRequest represents a query request
type QueryRequest struct {
	SQL          string                 `json:"sql"`
	Cube         string                 `json:"cube,omitempty"`
	Measures     []string               `json:"measures,omitempty"`
	Dimensions   []string               `json:"dimensions,omitempty"`
	Filters      []QueryFilter          `json:"filters,omitempty"`
	TimeRange    *TimeRange             `json:"time_range,omitempty"`
	ForceSource  DataSource             `json:"force_source,omitempty"`
	RealTimeOnly bool                   `json:"realtime_only,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

type QueryFilter struct {
	Dimension string        `json:"dimension"`
	Operator  string        `json:"operator"`
	Values    []interface{} `json:"values"`
}

type TimeRange struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Granularity string    `json:"granularity"`
}

// QueryResult represents query results
type QueryResult struct {
	Data      []map[string]interface{} `json:"data"`
	Columns   []ColumnInfo             `json:"columns"`
	RowCount  int                      `json:"row_count"`
	Source    DataSource               `json:"source"`
	QueryTime int64                    `json:"query_time_ms"`
	CacheHit  bool                     `json:"cache_hit"`
}

type ColumnInfo struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Label string `json:"label,omitempty"`
}

// RouteAndExecute analyzes query and routes to optimal source
func (r *DataSourceRouter) RouteAndExecute(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	startTime := time.Now()

	// Determine best data source
	source := r.determineSource(req)
	r.logger.Info("Query routed", zap.String("source", string(source)))

	// Execute on selected source
	var result *QueryResult
	var err error

	switch source {
	case DataSourceIgnite:
		result, err = r.executeOnIgnite(ctx, req)
	case DataSourceStarRocks:
		result, err = r.executeOnStarRocks(ctx, req)
	default:
		result, err = r.executeOnPostgres(ctx, req)
	}

	if err != nil {
		return nil, err
	}

	result.Source = source
	result.QueryTime = time.Since(startTime).Milliseconds()

	return result, nil
}

// determineSource analyzes query to pick optimal source
func (r *DataSourceRouter) determineSource(req QueryRequest) DataSource {
	// User can force a specific source
	if req.ForceSource != "" {
		return req.ForceSource
	}

	// Real-time queries always go to Ignite
	if req.RealTimeOnly {
		return DataSourceIgnite
	}

	// Check time range
	if req.TimeRange != nil {
		daysDiff := req.TimeRange.End.Sub(req.TimeRange.Start).Hours() / 24
		// Historical data > 90 days goes to StarRocks
		if daysDiff > 90 {
			return DataSourceStarRocks
		}
	}

	// Check for aggregation patterns (analytics)
	if r.hasAggregations(req) && len(req.Measures) > 0 {
		// Large aggregations better on StarRocks
		return DataSourceStarRocks
	}

	// Default: Ignite for real-time hot data
	return DataSourceIgnite
}

// hasAggregations checks if query contains aggregations
func (r *DataSourceRouter) hasAggregations(req QueryRequest) bool {
	sql := strings.ToUpper(req.SQL)
	aggregations := []string{"SUM(", "AVG(", "COUNT(", "MAX(", "MIN(", "GROUP BY"}
	for _, agg := range aggregations {
		if strings.Contains(sql, agg) {
			return true
		}
	}
	return len(req.Measures) > 0 && len(req.Dimensions) > 0
}

// executeOnIgnite runs query on Apache Ignite
func (r *DataSourceRouter) executeOnIgnite(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	if r.igniteDB == nil {
		return r.executeOnPostgres(ctx, req) // fallback
	}

	rows, err := r.igniteDB.QueryContext(ctx, req.SQL)
	if err != nil {
		return nil, fmt.Errorf("ignite query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// executeOnStarRocks runs query on StarRocks
func (r *DataSourceRouter) executeOnStarRocks(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	if r.starrocksDB == nil {
		return r.executeOnIgnite(ctx, req) // fallback to Ignite
	}

	rows, err := r.starrocksDB.QueryContext(ctx, req.SQL)
	if err != nil {
		return nil, fmt.Errorf("starrocks query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// executeOnPostgres runs query on PostgreSQL (fallback)
func (r *DataSourceRouter) executeOnPostgres(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	if r.postgresDB == nil {
		return nil, fmt.Errorf("no database connection available")
	}

	rows, err := r.postgresDB.QueryContext(ctx, req.SQL)
	if err != nil {
		return nil, fmt.Errorf("postgres query failed: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// scanRows converts sql.Rows to QueryResult
func (r *DataSourceRouter) scanRows(rows *sql.Rows) (*QueryResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, _ := rows.ColumnTypes()

	columnInfo := make([]ColumnInfo, len(columns))
	for i, col := range columns {
		colType := "string"
		if i < len(columnTypes) {
			colType = columnTypes[i].DatabaseTypeName()
		}
		columnInfo[i] = ColumnInfo{
			Name: col,
			Type: colType,
		}
	}

	var data []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		data = append(data, row)
	}

	return &QueryResult{
		Data:     data,
		Columns:  columnInfo,
		RowCount: len(data),
	}, nil
}

// ============================================================================
// SELF-SERVICE REPORT BUILDER
// ============================================================================

// ReportDefinitionBuilder helps build reports
type ReportDefinitionBuilder struct {
	router *DataSourceRouter
	logger *zap.Logger
}

// NewReportDefinitionBuilder creates a new builder
func NewReportDefinitionBuilder(router *DataSourceRouter) *ReportDefinitionBuilder {
	logger, _ := zap.NewProduction()
	return &ReportDefinitionBuilder{
		router: router,
		logger: logger,
	}
}

// SelfServiceReport represents a user-created report
type SelfServiceReport struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Cube        string                 `json:"cube"`
	Measures    []string               `json:"measures"`
	Dimensions  []string               `json:"dimensions"`
	Filters     []QueryFilter          `json:"filters,omitempty"`
	SortBy      []SortSpec             `json:"sort_by,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	ChartType   string                 `json:"chart_type,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	IsPublic    bool                   `json:"is_public"`
}

type SortSpec struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // ASC or DESC
}

// BuildSQL generates SQL from report definition
func (b *ReportDefinitionBuilder) BuildSQL(report SelfServiceReport) (string, error) {
	// Build SELECT clause
	var selectParts []string
	for _, dim := range report.Dimensions {
		selectParts = append(selectParts, dim)
	}
	for _, measure := range report.Measures {
		selectParts = append(selectParts, measure)
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectParts, ", "), report.Cube)

	// Add WHERE clause
	if len(report.Filters) > 0 {
		var conditions []string
		for _, f := range report.Filters {
			condition := b.buildFilterCondition(f)
			if condition != "" {
				conditions = append(conditions, condition)
			}
		}
		if len(conditions) > 0 {
			sql += " WHERE " + strings.Join(conditions, " AND ")
		}
	}

	// Add GROUP BY for aggregations
	if len(report.Dimensions) > 0 && len(report.Measures) > 0 {
		sql += " GROUP BY " + strings.Join(report.Dimensions, ", ")
	}

	// Add ORDER BY
	if len(report.SortBy) > 0 {
		var sortParts []string
		for _, s := range report.SortBy {
			sortParts = append(sortParts, fmt.Sprintf("%s %s", s.Field, s.Direction))
		}
		sql += " ORDER BY " + strings.Join(sortParts, ", ")
	}

	// Add LIMIT
	if report.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", report.Limit)
	}

	return sql, nil
}

func (b *ReportDefinitionBuilder) buildFilterCondition(f QueryFilter) string {
	if len(f.Values) == 0 {
		return ""
	}

	switch strings.ToLower(f.Operator) {
	case "equals", "=":
		return fmt.Sprintf("%s = '%v'", f.Dimension, f.Values[0])
	case "not_equals", "!=":
		return fmt.Sprintf("%s != '%v'", f.Dimension, f.Values[0])
	case "in":
		vals := make([]string, len(f.Values))
		for i, v := range f.Values {
			vals[i] = fmt.Sprintf("'%v'", v)
		}
		return fmt.Sprintf("%s IN (%s)", f.Dimension, strings.Join(vals, ", "))
	case "gt", ">":
		return fmt.Sprintf("%s > %v", f.Dimension, f.Values[0])
	case "lt", "<":
		return fmt.Sprintf("%s < %v", f.Dimension, f.Values[0])
	case "between":
		if len(f.Values) >= 2 {
			return fmt.Sprintf("%s BETWEEN %v AND %v", f.Dimension, f.Values[0], f.Values[1])
		}
	case "contains", "like":
		return fmt.Sprintf("%s LIKE '%%%v%%'", f.Dimension, f.Values[0])
	}
	return ""
}

// ExecuteReport builds and executes a self-service report
func (b *ReportDefinitionBuilder) ExecuteReport(ctx context.Context, report SelfServiceReport) (*QueryResult, error) {
	sql, err := b.BuildSQL(report)
	if err != nil {
		return nil, err
	}

	b.logger.Info("Executing self-service report",
		zap.String("report", report.Name),
		zap.String("sql", sql))

	return b.router.RouteAndExecute(ctx, QueryRequest{
		SQL:        sql,
		Cube:       report.Cube,
		Measures:   report.Measures,
		Dimensions: report.Dimensions,
		Filters:    report.Filters,
	})
}
