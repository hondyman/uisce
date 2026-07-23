package bridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// CalcCubeBridge bridges the calculations catalog with Cube.js for real-time analytics
type CalcCubeBridge struct {
	db         *sql.DB
	cache      *CalcCache
	cubeURL    string
	cubeSecret string
	calcEngine CalculationExecutor
	mu         sync.RWMutex
}

// CalculationExecutor interface for executing calculations
type CalculationExecutor interface {
	Execute(ctx context.Context, calcID string, params map[string]interface{}) (interface{}, error)
}

// NewCalcCubeBridge creates a new bridge between calculations and Cube
func NewCalcCubeBridge(db *sql.DB, cubeURL, cubeSecret string, calcEngine CalculationExecutor) *CalcCubeBridge {
	return &CalcCubeBridge{
		db:         db,
		cache:      NewCalcCache(15 * time.Minute),
		cubeURL:    cubeURL,
		cubeSecret: cubeSecret,
		calcEngine: calcEngine,
	}
}

// CalcCache provides caching for calculation results
type CalcCache struct {
	items map[string]*CacheItem
	ttl   time.Duration
	mu    sync.RWMutex
}

// CacheItem represents a cached calculation result
type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// NewCalcCache creates a new calculation cache
func NewCalcCache(ttl time.Duration) *CalcCache {
	cache := &CalcCache{
		items: make(map[string]*CacheItem),
		ttl:   ttl,
	}
	go cache.cleanup()
	return cache
}

func (c *CalcCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok || time.Now().After(item.ExpiresAt) {
		return nil, false
	}
	return item.Value, true
}

func (c *CalcCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

func (c *CalcCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.ExpiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// CatalogCalculation represents a calculation from the catalog
type CatalogCalculation struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"display_name"`
	Category     string                 `json:"category"`
	Formula      string                 `json:"formula"`
	EngineType   string                 `json:"engine_type"` // "sql", "python", "cube", "hybrid"
	Arguments    map[string]interface{} `json:"arguments"`
	OutputType   string                 `json:"output_type"`
	Description  string                 `json:"description"`
	Dependencies []string               `json:"dependencies"`
	Tags         []string               `json:"tags"`

	// Cube.js integration
	CubeMeasure    string   `json:"cube_measure"`    // mapped Cube measure
	CubeDimensions []string `json:"cube_dimensions"` // required dimensions
	PreAggregation string   `json:"pre_aggregation"` // optional pre-agg
}

// CubeQuery represents a Cube.js query
type CubeQuery struct {
	Measures       []string            `json:"measures,omitempty"`
	Dimensions     []string            `json:"dimensions,omitempty"`
	Segments       []string            `json:"segments,omitempty"`
	TimeDimensions []CubeTimeDimension `json:"timeDimensions,omitempty"`
	Filters        []CubeFilter        `json:"filters,omitempty"`
	Order          []CubeOrder         `json:"order,omitempty"`
	Limit          int                 `json:"limit,omitempty"`
}

// CubeTimeDimension represents time dimension configuration
type CubeTimeDimension struct {
	Dimension   string   `json:"dimension"`
	DateRange   []string `json:"dateRange,omitempty"`
	Granularity string   `json:"granularity,omitempty"`
}

// CubeFilter represents a Cube filter
type CubeFilter struct {
	Member   string        `json:"member"`
	Operator string        `json:"operator"`
	Values   []interface{} `json:"values,omitempty"`
}

// CubeOrder represents ordering
type CubeOrder struct {
	ID   string `json:"id"`
	Desc bool   `json:"desc"`
}

// CubeResult represents Cube.js query result
type CubeResult struct {
	Data        []map[string]interface{} `json:"data"`
	Annotation  map[string]interface{}   `json:"annotation,omitempty"`
	LastRefresh time.Time                `json:"lastRefreshTime,omitempty"`
}

// BridgeResult contains the unified calculation result
type BridgeResult struct {
	CalculationID   string                   `json:"calculation_id"`
	CalculationName string                   `json:"calculation_name"`
	ExecutionType   string                   `json:"execution_type"` // "cube", "sql", "hybrid"
	Data            []map[string]interface{} `json:"data"`
	Metadata        *ResultMetadata          `json:"metadata"`
	CacheHit        bool                     `json:"cache_hit"`
	ExecutionTimeMs int64                    `json:"execution_time_ms"`
}

// ResultMetadata contains result metadata
type ResultMetadata struct {
	RowCount    int        `json:"row_count"`
	LastRefresh time.Time  `json:"last_refresh"`
	DataSource  string     `json:"data_source"`
	CubeQuery   *CubeQuery `json:"cube_query,omitempty"`
	SQLQuery    string     `json:"sql_query,omitempty"`
}

// ExecuteCalculation executes a catalog calculation with intelligent routing
func (b *CalcCubeBridge) ExecuteCalculation(ctx context.Context, calcID string, params map[string]interface{}) (*BridgeResult, error) {
	start := time.Now()

	// Check cache first
	cacheKey := b.buildCacheKey(calcID, params)
	if cached, ok := b.cache.Get(cacheKey); ok {
		result := cached.(*BridgeResult)
		result.CacheHit = true
		return result, nil
	}

	// Fetch calculation definition
	calc, err := b.getCalculation(ctx, calcID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calculation: %w", err)
	}

	var result *BridgeResult

	// Route based on engine type
	switch calc.EngineType {
	case "cube":
		result, err = b.executeCubeCalculation(ctx, calc, params)
	case "sql":
		result, err = b.executeSQLCalculation(ctx, calc, params)
	case "hybrid":
		result, err = b.executeHybridCalculation(ctx, calc, params)
	case "python":
		result, err = b.executePythonCalculation(ctx, calc, params)
	default:
		// Default to SQL
		result, err = b.executeSQLCalculation(ctx, calc, params)
	}

	if err != nil {
		return nil, err
	}

	result.ExecutionTimeMs = time.Since(start).Milliseconds()

	// Cache the result
	b.cache.Set(cacheKey, result)

	return result, nil
}

// executeCubeCalculation executes via Cube.js
func (b *CalcCubeBridge) executeCubeCalculation(ctx context.Context, calc *CatalogCalculation, params map[string]interface{}) (*BridgeResult, error) {
	// Build Cube query from calculation definition
	query := b.buildCubeQuery(calc, params)

	// Execute Cube query
	cubeResult, err := b.queryCube(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("cube query failed: %w", err)
	}

	return &BridgeResult{
		CalculationID:   calc.ID,
		CalculationName: calc.DisplayName,
		ExecutionType:   "cube",
		Data:            cubeResult.Data,
		Metadata: &ResultMetadata{
			RowCount:    len(cubeResult.Data),
			LastRefresh: cubeResult.LastRefresh,
			DataSource:  "cube",
			CubeQuery:   query,
		},
	}, nil
}

// executeSQLCalculation executes via direct SQL
func (b *CalcCubeBridge) executeSQLCalculation(ctx context.Context, calc *CatalogCalculation, params map[string]interface{}) (*BridgeResult, error) {
	// Parse and bind parameters to formula
	sqlQuery := b.bindParameters(calc.Formula, params)

	// Execute SQL
	rows, err := b.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("sql query failed: %w", err)
	}
	defer rows.Close()

	// Convert to map slice
	data, err := b.rowsToMaps(rows)
	if err != nil {
		return nil, err
	}

	return &BridgeResult{
		CalculationID:   calc.ID,
		CalculationName: calc.DisplayName,
		ExecutionType:   "sql",
		Data:            data,
		Metadata: &ResultMetadata{
			RowCount:    len(data),
			LastRefresh: time.Now(),
			DataSource:  "sql",
			SQLQuery:    sqlQuery,
		},
	}, nil
}

// executeHybridCalculation combines Cube pre-aggregation with SQL post-processing
func (b *CalcCubeBridge) executeHybridCalculation(ctx context.Context, calc *CatalogCalculation, params map[string]interface{}) (*BridgeResult, error) {
	// First, get aggregated data from Cube
	cubeQuery := b.buildCubeQuery(calc, params)
	cubeResult, err := b.queryCube(ctx, cubeQuery)
	if err != nil {
		return nil, fmt.Errorf("cube pre-aggregation failed: %w", err)
	}

	// Apply SQL post-processing formula
	if calc.Formula != "" {
		// Create temp table with Cube results
		tempTable := fmt.Sprintf("temp_calc_%s_%d", calc.ID, time.Now().UnixNano())

		// Insert Cube results into temp table
		err = b.createTempTableWithData(ctx, tempTable, cubeResult.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp table: %w", err)
		}
		defer b.dropTempTable(ctx, tempTable)

		// Execute formula against temp table
		formula := strings.ReplaceAll(calc.Formula, "{{source}}", tempTable)
		formula = b.bindParameters(formula, params)

		rows, err := b.db.QueryContext(ctx, formula)
		if err != nil {
			return nil, fmt.Errorf("hybrid sql failed: %w", err)
		}
		defer rows.Close()

		data, err := b.rowsToMaps(rows)
		if err != nil {
			return nil, err
		}

		return &BridgeResult{
			CalculationID:   calc.ID,
			CalculationName: calc.DisplayName,
			ExecutionType:   "hybrid",
			Data:            data,
			Metadata: &ResultMetadata{
				RowCount:    len(data),
				LastRefresh: cubeResult.LastRefresh,
				DataSource:  "hybrid",
				CubeQuery:   cubeQuery,
				SQLQuery:    formula,
			},
		}, nil
	}

	return &BridgeResult{
		CalculationID:   calc.ID,
		CalculationName: calc.DisplayName,
		ExecutionType:   "hybrid",
		Data:            cubeResult.Data,
		Metadata: &ResultMetadata{
			RowCount:    len(cubeResult.Data),
			LastRefresh: cubeResult.LastRefresh,
			DataSource:  "cube",
			CubeQuery:   cubeQuery,
		},
	}, nil
}

// executePythonCalculation delegates to external Python engine
func (b *CalcCubeBridge) executePythonCalculation(ctx context.Context, calc *CatalogCalculation, params map[string]interface{}) (*BridgeResult, error) {
	if b.calcEngine == nil {
		return nil, fmt.Errorf("python calculation engine not configured")
	}

	result, err := b.calcEngine.Execute(ctx, calc.ID, params)
	if err != nil {
		return nil, fmt.Errorf("python execution failed: %w", err)
	}

	// Convert result to standard format
	var data []map[string]interface{}
	switch v := result.(type) {
	case []map[string]interface{}:
		data = v
	case map[string]interface{}:
		data = []map[string]interface{}{v}
	default:
		data = []map[string]interface{}{{"result": result}}
	}

	return &BridgeResult{
		CalculationID:   calc.ID,
		CalculationName: calc.DisplayName,
		ExecutionType:   "python",
		Data:            data,
		Metadata: &ResultMetadata{
			RowCount:    len(data),
			LastRefresh: time.Now(),
			DataSource:  "python",
		},
	}, nil
}

// GenerateCubeMeasure generates a Cube.js measure definition from a calculation
func (b *CalcCubeBridge) GenerateCubeMeasure(calc *CatalogCalculation) (string, error) {
	// Map calculation types to Cube measure types
	measureType := "number"
	switch calc.OutputType {
	case "percentage", "ratio":
		measureType = "number"
	case "currency", "money":
		measureType = "number"
	case "count", "integer":
		measureType = "count"
	case "sum":
		measureType = "sum"
	case "average":
		measureType = "avg"
	}

	// Generate Cube.js measure definition
	// Note: Using ${BACKTICK} placeholder for JavaScript template literals
	measureDef := fmt.Sprintf(`
    %s: {
      type: '%s',
      title: '%s',
      description: '%s',
      sql: "${BACKTICK}%s${BACKTICK}",
      meta: {
        category: '%s',
        calculationId: '%s',
        tags: %s
      }
    }`,
		b.sanitizeMeasureName(calc.Name),
		measureType,
		calc.DisplayName,
		calc.Description,
		b.convertFormulaToCubeSQL(calc.Formula),
		calc.Category,
		calc.ID,
		b.tagsToJSON(calc.Tags),
	)

	// Replace placeholder with actual backticks for JS output
	measureDef = strings.ReplaceAll(measureDef, "${BACKTICK}", "`")

	return measureDef, nil
}

// GenerateCubeModel generates a complete Cube model from catalog calculations
func (b *CalcCubeBridge) GenerateCubeModel(ctx context.Context, category string, tableName string) (string, error) {
	// Fetch all calculations in category
	calcs, err := b.getCalculationsByCategory(ctx, category)
	if err != nil {
		return "", err
	}

	var measures []string
	for _, calc := range calcs {
		measure, err := b.GenerateCubeMeasure(calc)
		if err != nil {
			continue
		}
		measures = append(measures, measure)
	}

	// Generate complete Cube model
	// Using placeholder for JS template literals that will be replaced
	cubeName := b.sanitizeCubeName(category)
	modelTemplate := `
cube('%s', {
  sql: "${BACKTICK}SELECT * FROM %s${BACKTICK}",
  
  preAggregations: {
    main: {
      measures: [%s],
      dimensions: [CUBE.created_at],
      timeDimension: CUBE.created_at,
      granularity: 'day',
      refreshKey: {
        every: '1 hour'
      }
    }
  },
  
  measures: {
%s
  },
  
  dimensions: {
    id: {
      sql: "${BACKTICK}id${BACKTICK}",
      type: 'string',
      primaryKey: true
    },
    created_at: {
      sql: "${BACKTICK}created_at${BACKTICK}",
      type: 'time'
    }
  }
});
`
	model := fmt.Sprintf(modelTemplate, cubeName, tableName, b.getMeasureNames(calcs), strings.Join(measures, ",\n"))

	// Replace placeholder with actual backticks for JS output
	model = strings.ReplaceAll(model, "${BACKTICK}", "`")

	return model, nil
}

// BatchExecute executes multiple calculations in parallel
func (b *CalcCubeBridge) BatchExecute(ctx context.Context, calcIDs []string, params map[string]interface{}) (map[string]*BridgeResult, error) {
	results := make(map[string]*BridgeResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(calcIDs))

	for _, calcID := range calcIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			result, err := b.ExecuteCalculation(ctx, id, params)
			if err != nil {
				errChan <- fmt.Errorf("calculation %s failed: %w", id, err)
				return
			}

			mu.Lock()
			results[id] = result
			mu.Unlock()
		}(calcID)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("all calculations failed: %s", strings.Join(errors, "; "))
	}

	return results, nil
}

// GetCalculationDependencyGraph returns the dependency graph for a calculation
func (b *CalcCubeBridge) GetCalculationDependencyGraph(ctx context.Context, calcID string) (map[string][]string, error) {
	graph := make(map[string][]string)
	visited := make(map[string]bool)

	var buildGraph func(id string) error
	buildGraph = func(id string) error {
		if visited[id] {
			return nil
		}
		visited[id] = true

		calc, err := b.getCalculation(ctx, id)
		if err != nil {
			return err
		}

		graph[id] = calc.Dependencies

		for _, depID := range calc.Dependencies {
			if err := buildGraph(depID); err != nil {
				return err
			}
		}

		return nil
	}

	if err := buildGraph(calcID); err != nil {
		return nil, err
	}

	return graph, nil
}

// SyncCalculationsToCube syncs all calculations to Cube.js as measures
func (b *CalcCubeBridge) SyncCalculationsToCube(ctx context.Context) error {
	calcs, err := b.getAllCalculations(ctx)
	if err != nil {
		return err
	}

	// Group by category
	byCategory := make(map[string][]*CatalogCalculation)
	for _, calc := range calcs {
		byCategory[calc.Category] = append(byCategory[calc.Category], calc)
	}

	// Generate and save Cube models for each category
	for category, categoryCalcs := range byCategory {
		model, err := b.generateCategoryModel(categoryCalcs, category)
		if err != nil {
			continue
		}

		// Save to cube_models table
		err = b.saveCubeModel(ctx, category, model)
		if err != nil {
			return fmt.Errorf("failed to save model for %s: %w", category, err)
		}
	}

	return nil
}

// Helper methods
func (b *CalcCubeBridge) getCalculation(ctx context.Context, calcID string) (*CatalogCalculation, error) {
	query := `
		SELECT 
			id, name, display_name, category, formula, 
			COALESCE(engine_type, 'sql') as engine_type,
			COALESCE(arguments::text, '{}') as arguments,
			COALESCE(output_type, 'number') as output_type,
			COALESCE(description, '') as description,
			COALESCE(dependencies, '[]') as dependencies,
			COALESCE(tags, '[]') as tags,
			COALESCE(cube_measure, '') as cube_measure,
			COALESCE(cube_dimensions, '[]') as cube_dimensions,
			COALESCE(pre_aggregation, '') as pre_aggregation
		FROM calculations
		WHERE id = $1
	`

	var calc CatalogCalculation
	var argsJSON, depsJSON, tagsJSON, dimsJSON string

	err := b.db.QueryRowContext(ctx, query, calcID).Scan(
		&calc.ID, &calc.Name, &calc.DisplayName, &calc.Category,
		&calc.Formula, &calc.EngineType, &argsJSON, &calc.OutputType,
		&calc.Description, &depsJSON, &tagsJSON, &calc.CubeMeasure,
		&dimsJSON, &calc.PreAggregation,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	json.Unmarshal([]byte(argsJSON), &calc.Arguments)
	json.Unmarshal([]byte(depsJSON), &calc.Dependencies)
	json.Unmarshal([]byte(tagsJSON), &calc.Tags)
	json.Unmarshal([]byte(dimsJSON), &calc.CubeDimensions)

	return &calc, nil
}

func (b *CalcCubeBridge) getCalculationsByCategory(ctx context.Context, category string) ([]*CatalogCalculation, error) {
	query := `
		SELECT id FROM calculations WHERE category = $1
	`

	rows, err := b.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calcs []*CatalogCalculation
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		calc, err := b.getCalculation(ctx, id)
		if err != nil {
			continue
		}
		calcs = append(calcs, calc)
	}

	return calcs, nil
}

func (b *CalcCubeBridge) getAllCalculations(ctx context.Context) ([]*CatalogCalculation, error) {
	query := `SELECT id FROM calculations WHERE enabled = true`

	rows, err := b.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calcs []*CatalogCalculation
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		calc, err := b.getCalculation(ctx, id)
		if err != nil {
			continue
		}
		calcs = append(calcs, calc)
	}

	return calcs, nil
}

func (b *CalcCubeBridge) buildCubeQuery(calc *CatalogCalculation, params map[string]interface{}) *CubeQuery {
	query := &CubeQuery{
		Measures:   []string{},
		Dimensions: calc.CubeDimensions,
	}

	// Add the calculation's Cube measure
	if calc.CubeMeasure != "" {
		query.Measures = append(query.Measures, calc.CubeMeasure)
	}

	// Apply parameters as filters
	if portfolioID, ok := params["portfolio_id"].(string); ok {
		query.Filters = append(query.Filters, CubeFilter{
			Member:   "Portfolio.id",
			Operator: "equals",
			Values:   []interface{}{portfolioID},
		})
	}

	// Handle date range
	if startDate, ok := params["start_date"].(string); ok {
		if endDate, ok := params["end_date"].(string); ok {
			query.TimeDimensions = append(query.TimeDimensions, CubeTimeDimension{
				Dimension: "Transactions.date",
				DateRange: []string{startDate, endDate},
			})
		}
	}

	// Handle granularity
	if granularity, ok := params["granularity"].(string); ok && len(query.TimeDimensions) > 0 {
		query.TimeDimensions[0].Granularity = granularity
	}

	// Handle limit
	if limit, ok := params["limit"].(int); ok {
		query.Limit = limit
	}

	return query
}

func (b *CalcCubeBridge) queryCube(ctx context.Context, query *CubeQuery) (*CubeResult, error) {
	// In production, this would make an HTTP call to Cube.js
	// For now, we'll simulate by converting to SQL and executing directly

	sqlQuery := b.cubeQueryToSQL(query)

	rows, err := b.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data, err := b.rowsToMaps(rows)
	if err != nil {
		return nil, err
	}

	return &CubeResult{
		Data:        data,
		LastRefresh: time.Now(),
	}, nil
}

func (b *CalcCubeBridge) cubeQueryToSQL(query *CubeQuery) string {
	// Simplified Cube to SQL conversion
	// In production, Cube.js handles this
	selectParts := make([]string, 0)

	for _, m := range query.Measures {
		parts := strings.Split(m, ".")
		if len(parts) == 2 {
			selectParts = append(selectParts, fmt.Sprintf("%s as %s", parts[1], strings.ReplaceAll(m, ".", "_")))
		}
	}

	for _, d := range query.Dimensions {
		parts := strings.Split(d, ".")
		if len(parts) == 2 {
			selectParts = append(selectParts, parts[1])
		}
	}

	if len(selectParts) == 0 {
		selectParts = append(selectParts, "*")
	}

	sql := fmt.Sprintf("SELECT %s FROM analytics_data", strings.Join(selectParts, ", "))

	// Add WHERE clauses from filters
	var whereClauses []string
	for _, f := range query.Filters {
		parts := strings.Split(f.Member, ".")
		if len(parts) == 2 {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = '%v'", parts[1], f.Values[0]))
		}
	}

	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	if query.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	return sql
}

func (b *CalcCubeBridge) bindParameters(formula string, params map[string]interface{}) string {
	result := formula

	// Replace {{param}} placeholders
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		paramName := match[2 : len(match)-2]
		if val, ok := params[paramName]; ok {
			switch v := val.(type) {
			case string:
				return fmt.Sprintf("'%s'", v)
			default:
				return fmt.Sprintf("%v", v)
			}
		}
		return match
	})

	// Replace :param placeholders
	for key, val := range params {
		placeholder := ":" + key
		var replacement string
		switch v := val.(type) {
		case string:
			replacement = fmt.Sprintf("'%s'", v)
		default:
			replacement = fmt.Sprintf("%v", v)
		}
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	return result
}

func (b *CalcCubeBridge) rowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}

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
		result = append(result, row)
	}

	return result, nil
}

func (b *CalcCubeBridge) createTempTableWithData(ctx context.Context, tableName string, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// Get columns from first row
	var columns []string
	for col := range data[0] {
		columns = append(columns, col)
	}

	// Create table
	createSQL := fmt.Sprintf("CREATE TEMP TABLE %s (%s)", tableName,
		strings.Join(columns, " TEXT, ")+" TEXT")

	_, err := b.db.ExecContext(ctx, createSQL)
	if err != nil {
		return err
	}

	// Insert data
	for _, row := range data {
		values := make([]string, len(columns))
		for i, col := range columns {
			values[i] = fmt.Sprintf("'%v'", row[col])
		}

		insertSQL := fmt.Sprintf("INSERT INTO %s VALUES (%s)", tableName, strings.Join(values, ", "))
		_, err := b.db.ExecContext(ctx, insertSQL)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *CalcCubeBridge) dropTempTable(ctx context.Context, tableName string) {
	b.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
}

func (b *CalcCubeBridge) buildCacheKey(calcID string, params map[string]interface{}) string {
	paramsJSON, _ := json.Marshal(params)
	return fmt.Sprintf("calc:%s:%s", calcID, string(paramsJSON))
}

func (b *CalcCubeBridge) sanitizeMeasureName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return re.ReplaceAllString(name, "_")
}

func (b *CalcCubeBridge) sanitizeCubeName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]`)
	result := re.ReplaceAllString(name, "")
	if len(result) > 0 {
		return strings.ToUpper(result[:1]) + result[1:]
	}
	return "Analytics"
}

func (b *CalcCubeBridge) convertFormulaToCubeSQL(formula string) string {
	// Convert standard SQL to Cube.js SQL template
	result := formula

	// Replace table references with ${TABLE}
	result = regexp.MustCompile(`FROM\s+(\w+)`).ReplaceAllString(result, "FROM ${TABLE}")

	// Replace column references
	result = regexp.MustCompile(`SELECT\s+(\w+)`).ReplaceAllString(result, "SELECT ${CUBE}.$1")

	return result
}

func (b *CalcCubeBridge) tagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(tags)
	return string(data)
}

func (b *CalcCubeBridge) getMeasureNames(calcs []*CatalogCalculation) string {
	names := make([]string, len(calcs))
	for i, calc := range calcs {
		names[i] = fmt.Sprintf("CUBE.%s", b.sanitizeMeasureName(calc.Name))
	}
	return strings.Join(names, ", ")
}

func (b *CalcCubeBridge) generateCategoryModel(calcs []*CatalogCalculation, category string) (string, error) {
	var measures []string
	for _, calc := range calcs {
		measure, err := b.GenerateCubeMeasure(calc)
		if err != nil {
			continue
		}
		measures = append(measures, measure)
	}

	cubeName := b.sanitizeCubeName(category)
	// Using placeholder for JS template literals
	modelTemplate := `
cube('%s', {
  sql: "${BACKTICK}SELECT * FROM analytics_data WHERE category = '%s'${BACKTICK}",
  
  measures: {
%s
  },
  
  dimensions: {
    id: {
      sql: "${BACKTICK}id${BACKTICK}",
      type: 'string',
      primaryKey: true
    }
  }
});
`
	model := fmt.Sprintf(modelTemplate, cubeName, category, strings.Join(measures, ",\n"))
	// Replace placeholder with actual backticks
	model = strings.ReplaceAll(model, "${BACKTICK}", "`")
	return model, nil
}

func (b *CalcCubeBridge) saveCubeModel(ctx context.Context, category string, model string) error {
	query := `
		INSERT INTO cube_models (name, category, model_yaml, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET
			model_yaml = EXCLUDED.model_yaml,
			updated_at = NOW()
	`

	_, err := b.db.ExecContext(ctx, query, category, category, model)
	return err
}
