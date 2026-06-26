package semantic

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// QueryEngine executes semantic queries and generates SQL
type QueryEngine struct {
	service *Service
}

// NewQueryEngine creates a new query engine
func NewQueryEngine(service *Service) *QueryEngine {
	return &QueryEngine{service: service}
}

// ExecuteQuery executes a semantic query
func (qe *QueryEngine) ExecuteQuery(ctx context.Context, tenantID string, query *Query) (*QueryResult, error) {
	startTime := time.Now()

	// Check cache first
	queryHash := qe.hashQuery(query)
	cachedResult, err := qe.getCachedResult(ctx, tenantID, queryHash)
	if err == nil && cachedResult != nil {
		// Update access stats
		qe.updateCacheAccess(ctx, cachedResult.ID)

		return &QueryResult{
			Data:          cachedResult.Result,
			ExecutionTime: int64(cachedResult.ExecutionTimeMs),
			CacheHit:      true,
		}, nil
	}

	// Generate SQL
	sql, annotation, err := qe.GenerateSQL(ctx, tenantID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// TODO: Refactor to Hasura GraphQL
	// Consider using Hasura for query execution with semantic views
	// Direct SQL execution may remain for complex cube queries
	// Execute SQL
	rows, err := qe.service.db.QueryContext(ctx, sql)
	if err != nil {
		// Record error in history
		qe.recordQueryError(ctx, tenantID, query, sql, err)
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Parse results
	data, err := qe.parseResults(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	executionTime := time.Since(startTime).Milliseconds()

	// Cache result
	qe.cacheResult(ctx, tenantID, queryHash, query, data, int(executionTime))

	// Record in history
	qe.recordQuerySuccess(ctx, tenantID, query, sql, data, int(executionTime))

	return &QueryResult{
		Data:          data,
		Annotation:    *annotation,
		ExecutionTime: executionTime,
		CacheHit:      false,
	}, nil
}

// GenerateSQL generates SQL from a semantic query
func (qe *QueryEngine) GenerateSQL(ctx context.Context, tenantID string, query *Query) (string, *QueryAnnotation, error) {
	// Parse cube names from measures and dimensions
	cubeNames := qe.extractCubeNames(query)
	if len(cubeNames) == 0 {
		return "", nil, fmt.Errorf("no cubes found in query")
	}

	// Load cube metadata
	cubes := make(map[string]*Cube)
	for _, cubeName := range cubeNames {
		cube, err := qe.service.GetCube(ctx, tenantID, cubeName)
		if err != nil {
			return "", nil, fmt.Errorf("failed to load cube %s: %w", cubeName, err)
		}
		cubes[cubeName] = cube
	}

	// Build SQL
	sql, annotation := qe.buildSQL(cubes, query)

	return sql, annotation, nil
}

// buildSQL constructs the SQL query
func (qe *QueryEngine) buildSQL(cubes map[string]*Cube, query *Query) (string, *QueryAnnotation) {
	var selectClauses []string
	var fromClauses []string
	var whereClauses []string
	var groupByClauses []string
	var orderByClauses []string

	annotation := &QueryAnnotation{
		Measures:       make(map[string]MemberAnnotation),
		Dimensions:     make(map[string]MemberAnnotation),
		TimeDimensions: make(map[string]MemberAnnotation),
	}

	// Process measures
	for _, measureRef := range query.Measures {
		parts := strings.Split(measureRef, ".")
		if len(parts) != 2 {
			continue
		}
		cubeName, measureName := parts[0], parts[1]

		cube, ok := cubes[cubeName]
		if !ok {
			continue
		}

		for _, measure := range cube.Measures {
			if measure.Name == measureName {
				selectClauses = append(selectClauses, fmt.Sprintf("%s AS %s", measure.SQL, measureRef))
				annotation.Measures[measureRef] = MemberAnnotation{
					Title:      measure.DisplayName,
					ShortTitle: measure.Name,
					Type:       measure.Type,
					Format:     measure.Format,
				}
				break
			}
		}
	}

	// Process dimensions
	for _, dimRef := range query.Dimensions {
		parts := strings.Split(dimRef, ".")
		if len(parts) != 2 {
			continue
		}
		cubeName, dimName := parts[0], parts[1]

		cube, ok := cubes[cubeName]
		if !ok {
			continue
		}

		for _, dim := range cube.Dimensions {
			if dim.Name == dimName {
				selectClauses = append(selectClauses, fmt.Sprintf("%s AS %s", dim.SQL, dimRef))
				groupByClauses = append(groupByClauses, dim.SQL)
				annotation.Dimensions[dimRef] = MemberAnnotation{
					Title:      dim.DisplayName,
					ShortTitle: dim.Name,
					Type:       dim.Type,
					Format:     dim.Format,
				}
				break
			}
		}
	}

	// Process time dimensions
	for _, timeDim := range query.TimeDimensions {
		parts := strings.Split(timeDim.Dimension, ".")
		if len(parts) != 2 {
			continue
		}
		cubeName, dimName := parts[0], parts[1]

		cube, ok := cubes[cubeName]
		if !ok {
			continue
		}

		for _, dim := range cube.Dimensions {
			if dim.Name == dimName && dim.Type == "time" {
				// Apply granularity
				sqlExpr := qe.applyTimeGranularity(dim.SQL, timeDim.Granularity)
				selectClauses = append(selectClauses, fmt.Sprintf("%s AS %s", sqlExpr, timeDim.Dimension))
				groupByClauses = append(groupByClauses, sqlExpr)

				// Apply date range filter
				if len(timeDim.DateRange) == 2 {
					whereClauses = append(whereClauses, fmt.Sprintf(
						"%s BETWEEN '%s' AND '%s'",
						dim.SQL, timeDim.DateRange[0], timeDim.DateRange[1],
					))
				}

				annotation.TimeDimensions[timeDim.Dimension] = MemberAnnotation{
					Title:      dim.DisplayName,
					ShortTitle: dim.Name,
					Type:       "time",
					Format:     dim.Format,
				}
				break
			}
		}
	}

	// Process filters
	for _, filter := range query.Filters {
		parts := strings.Split(filter.Member, ".")
		if len(parts) != 2 {
			continue
		}
		cubeName, memberName := parts[0], parts[1]

		cube, ok := cubes[cubeName]
		if !ok {
			continue
		}

		// Find dimension or measure
		var sqlExpr string
		for _, dim := range cube.Dimensions {
			if dim.Name == memberName {
				sqlExpr = dim.SQL
				break
			}
		}

		if sqlExpr != "" {
			whereClause := qe.buildFilterClause(sqlExpr, filter)
			if whereClause != "" {
				whereClauses = append(whereClauses, whereClause)
			}
		}
	}

	// Build FROM clause
	for cubeName, cube := range cubes {
		fromClauses = append(fromClauses, fmt.Sprintf("(%s) AS %s", cube.SQL, cubeName))
	}

	// Build ORDER BY clause
	for member, direction := range query.Order {
		orderByClauses = append(orderByClauses, fmt.Sprintf("%s %s", member, strings.ToUpper(direction)))
	}

	// Construct final SQL
	sql := "SELECT " + strings.Join(selectClauses, ", ")
	sql += "\nFROM " + strings.Join(fromClauses, ", ")

	if len(whereClauses) > 0 {
		sql += "\nWHERE " + strings.Join(whereClauses, " AND ")
	}

	if len(groupByClauses) > 0 {
		sql += "\nGROUP BY " + strings.Join(groupByClauses, ", ")
	}

	if len(orderByClauses) > 0 {
		sql += "\nORDER BY " + strings.Join(orderByClauses, ", ")
	}

	if query.Limit > 0 {
		sql += fmt.Sprintf("\nLIMIT %d", query.Limit)
	}

	if query.Offset > 0 {
		sql += fmt.Sprintf("\nOFFSET %d", query.Offset)
	}

	annotation.GeneratedSQL = sql

	return sql, annotation
}

// Helper functions

func (qe *QueryEngine) extractCubeNames(query *Query) []string {
	cubeMap := make(map[string]bool)

	for _, measure := range query.Measures {
		parts := strings.Split(measure, ".")
		if len(parts) == 2 {
			cubeMap[parts[0]] = true
		}
	}

	for _, dim := range query.Dimensions {
		parts := strings.Split(dim, ".")
		if len(parts) == 2 {
			cubeMap[parts[0]] = true
		}
	}

	for _, timeDim := range query.TimeDimensions {
		parts := strings.Split(timeDim.Dimension, ".")
		if len(parts) == 2 {
			cubeMap[parts[0]] = true
		}
	}

	cubes := make([]string, 0, len(cubeMap))
	for cube := range cubeMap {
		cubes = append(cubes, cube)
	}

	return cubes
}

func (qe *QueryEngine) applyTimeGranularity(sqlExpr, granularity string) string {
	switch granularity {
	case "hour":
		return fmt.Sprintf("DATE_TRUNC('hour', %s)", sqlExpr)
	case "day":
		return fmt.Sprintf("DATE_TRUNC('day', %s)", sqlExpr)
	case "week":
		return fmt.Sprintf("DATE_TRUNC('week', %s)", sqlExpr)
	case "month":
		return fmt.Sprintf("DATE_TRUNC('month', %s)", sqlExpr)
	case "quarter":
		return fmt.Sprintf("DATE_TRUNC('quarter', %s)", sqlExpr)
	case "year":
		return fmt.Sprintf("DATE_TRUNC('year', %s)", sqlExpr)
	default:
		return sqlExpr
	}
}

func (qe *QueryEngine) buildFilterClause(sqlExpr string, filter Filter) string {
	switch filter.Operator {
	case "equals":
		if len(filter.Values) > 0 {
			return fmt.Sprintf("%s = '%s'", sqlExpr, filter.Values[0])
		}
	case "notEquals":
		if len(filter.Values) > 0 {
			return fmt.Sprintf("%s != '%s'", sqlExpr, filter.Values[0])
		}
	case "contains":
		if len(filter.Values) > 0 {
			return fmt.Sprintf("%s LIKE '%%%s%%'", sqlExpr, filter.Values[0])
		}
	case "gt":
		if len(filter.Values) > 0 {
			return fmt.Sprintf("%s > '%s'", sqlExpr, filter.Values[0])
		}
	case "gte":
		if len(filter.Values) > 0 {
			return fmt.Sprintf("%s >= '%s'", sqlExpr, filter.Values[0])
		}
	case "lt":
		if len(filter.Values) > 0 {
			return fmt.Sprintf("%s < '%s'", sqlExpr, filter.Values[0])
		}
	case "lte":
		if len(filter.Values) > 0 {
			return fmt.Sprintf("%s <= '%s'", sqlExpr, filter.Values[0])
		}
	case "in":
		if len(filter.Values) > 0 {
			values := make([]string, len(filter.Values))
			for i, v := range filter.Values {
				values[i] = fmt.Sprintf("'%s'", v)
			}
			return fmt.Sprintf("%s IN (%s)", sqlExpr, strings.Join(values, ", "))
		}
	}
	return ""
}

func (qe *QueryEngine) hashQuery(query *Query) string {
	queryJSON, _ := json.Marshal(query)
	hash := sha256.Sum256(queryJSON)
	return fmt.Sprintf("%x", hash)
}

func (qe *QueryEngine) parseResults(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

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
			row[col] = values[i]
		}

		results = append(results, row)
	}

	return results, nil
}

func (qe *QueryEngine) getCachedResult(ctx context.Context, tenantID, queryHash string) (*QueryCache, error) {
	query := `
		SELECT id, tenant_id, query_hash, query, result, result_rows, execution_time_ms,
		       cache_key, created_at, expires_at, last_accessed_at, access_count
		FROM semantic_query_cache_v2
		WHERE tenant_id = $1 AND query_hash = $2 AND expires_at > now()
	`

	cached := &QueryCache{}
	var queryJSON, resultJSON []byte

	err := qe.service.db.QueryRowContext(ctx, query, tenantID, queryHash).Scan(
		&cached.ID, &cached.TenantID, &cached.QueryHash, &queryJSON, &resultJSON, &cached.ResultRows, &cached.ExecutionTimeMs,
		&cached.CacheKey, &cached.CreatedAt, &cached.ExpiresAt, &cached.LastAccessedAt, &cached.AccessCount,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(queryJSON, &cached.Query)
	json.Unmarshal(resultJSON, &cached.Result)

	return cached, nil
}

func (qe *QueryEngine) cacheResult(ctx context.Context, tenantID, queryHash string, query *Query, result []map[string]interface{}, executionTimeMs int) {
	queryJSON, _ := json.Marshal(query)
	resultJSON, _ := json.Marshal(result)

	expiresAt := time.Now().Add(1 * time.Hour) // 1 hour TTL

	sql := `
		INSERT INTO semantic_query_cache_v2 (
			tenant_id, query_hash, query, result, result_rows, execution_time_ms, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, query_hash) DO UPDATE
		SET result = EXCLUDED.result,
		    result_rows = EXCLUDED.result_rows,
		    execution_time_ms = EXCLUDED.execution_time_ms,
		    expires_at = EXCLUDED.expires_at,
		    last_accessed_at = now(),
		    access_count = semantic_query_cache_v2.access_count + 1
	`

	qe.service.db.ExecContext(ctx, sql, tenantID, queryHash, queryJSON, resultJSON, len(result), executionTimeMs, expiresAt)
}

func (qe *QueryEngine) updateCacheAccess(ctx context.Context, cacheID string) {
	sql := `
		UPDATE semantic_query_cache_v2
		SET last_accessed_at = now(), access_count = access_count + 1
		WHERE id = $1
	`
	qe.service.db.ExecContext(ctx, sql, cacheID)
}

func (qe *QueryEngine) recordQuerySuccess(ctx context.Context, tenantID string, query *Query, sql string, result []map[string]interface{}, executionTimeMs int) {
	history := &QueryHistory{
		TenantID:        tenantID,
		Query:           *query,
		GeneratedSQL:    sql,
		ExecutionTimeMs: executionTimeMs,
		ResultRows:      len(result),
		CacheHit:        false,
	}
	qe.service.RecordQueryHistory(ctx, history)
}

func (qe *QueryEngine) recordQueryError(ctx context.Context, tenantID string, query *Query, sql string, err error) {
	history := &QueryHistory{
		TenantID:     tenantID,
		Query:        *query,
		GeneratedSQL: sql,
		Error:        err.Error(),
	}
	qe.service.RecordQueryHistory(ctx, history)
}
