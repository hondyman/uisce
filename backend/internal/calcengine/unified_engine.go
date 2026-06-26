package calcengine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql" // StarRocks MySQL protocol
)

// ============================================================================
// UNIFIED CALC ENGINE - Real-time App + Hot/Cold Analytics
// ============================================================================
// Architecture:
//   App Real-time:  Go CalcEngine → StarRocks Native Tables (sub-second)
//   Hot Analytics:  Cube.js → StarRocks Native (pre-aggregated, <90 days)
//   Cold Analytics: Cube.js → StarRocks External Tables on Parquet/Iceberg
//   Data Lifecycle: Hot → Cold push via scheduled jobs
// ============================================================================

// UnifiedCalcEngine provides both real-time and analytics query paths
type UnifiedCalcEngine struct {
	// StarRocks connection for both hot native and cold parquet
	starrocks *sql.DB
	config    *UnifiedCalcConfig

	// Calculation registry and cache
	calcRegistry map[string]*CalculationDef
	resultCache  *ResultCache
	mu           sync.RWMutex
}

// UnifiedCalcConfig configures the unified calc engine
type UnifiedCalcConfig struct {
	// StarRocks connection (handles both native and external tables)
	StarRocksHost     string `yaml:"starrocks_host"`
	StarRocksPort     int    `yaml:"starrocks_port"`
	StarRocksUser     string `yaml:"starrocks_user"`
	StarRocksPassword string `yaml:"starrocks_password"`
	StarRocksDB       string `yaml:"starrocks_db"`

	// Hot/Cold configuration
	HotDatabase  string `yaml:"hot_database"`  // e.g., "semantic_hot"
	ColdDatabase string `yaml:"cold_database"` // e.g., "semantic_cold" (external on Parquet)

	// Data retention
	HotRetentionDays  int `yaml:"hot_retention_days"`  // Default: 90
	ColdRetentionDays int `yaml:"cold_retention_days"` // Default: 2555 (7 years)

	// Real-time cache
	CacheTTL        time.Duration `yaml:"cache_ttl"`
	CacheMaxEntries int           `yaml:"cache_max_entries"`

	// Resource groups for QoS
	RealtimeResourceGroup  string `yaml:"realtime_resource_group"`  // High priority
	AnalyticsResourceGroup string `yaml:"analytics_resource_group"` // Normal priority
}

// CalculationDef defines a calculation that can be executed
type CalculationDef struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Formula     string            `json:"formula"`      // SQL template
	InputParams []ParamDef        `json:"input_params"` // Required inputs
	OutputType  string            `json:"output_type"`  // scalar, timeseries, breakdown
	Cacheable   bool              `json:"cacheable"`    // Can result be cached
	CacheTTL    time.Duration     `json:"cache_ttl"`
	DataSource  string            `json:"data_source"` // hot, cold, realtime
	Tags        map[string]string `json:"tags"`
}

// ParamDef defines a calculation parameter
type ParamDef struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // string, float64, time, []string
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
}

// CalcRequest represents a calculation request
type CalcRequest struct {
	// Required: tenant isolation
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`

	// Calculation to run
	CalculationID string `json:"calculation_id,omitempty"`
	MetricName    string `json:"metric_name,omitempty"`

	// Parameters
	Params map[string]interface{} `json:"params"`

	// Query mode
	Mode QueryMode `json:"mode"` // realtime, hot, cold, auto

	// Options
	BypassCache bool          `json:"bypass_cache"`
	Timeout     time.Duration `json:"timeout"`
}

// QueryMode determines how to route the query
type QueryMode string

const (
	ModeRealtime QueryMode = "realtime" // Direct StarRocks native, highest priority
	ModeHot      QueryMode = "hot"      // StarRocks native tables (<90 days)
	ModeCold     QueryMode = "cold"     // StarRocks external on Parquet
	ModeAuto     QueryMode = "auto"     // Auto-detect based on date range
)

// CalcResponse represents a calculation result
type CalcResponse struct {
	// Request info
	CalculationID string    `json:"calculation_id"`
	TenantID      string    `json:"tenant_id"`
	DatasourceID  string    `json:"datasource_id"`
	Mode          QueryMode `json:"mode"`

	// Result
	Value     interface{}              `json:"value"`               // Primary result
	Breakdown []map[string]interface{} `json:"breakdown,omitempty"` // Detailed breakdown
	Metadata  map[string]interface{}   `json:"metadata,omitempty"`  // Additional info

	// Performance
	ExecutionTime time.Duration `json:"execution_time"`
	FromCache     bool          `json:"from_cache"`
	DataSource    string        `json:"data_source"` // which table/database was queried
}

// NewUnifiedCalcEngine creates a new unified calc engine
func NewUnifiedCalcEngine(cfg *UnifiedCalcConfig) (*UnifiedCalcEngine, error) {
	// Set defaults
	if cfg.StarRocksPort == 0 {
		cfg.StarRocksPort = 9030
	}
	if cfg.HotRetentionDays == 0 {
		cfg.HotRetentionDays = 90
	}
	if cfg.ColdRetentionDays == 0 {
		cfg.ColdRetentionDays = 2555 // 7 years
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 5 * time.Minute
	}
	if cfg.CacheMaxEntries == 0 {
		cfg.CacheMaxEntries = 10000
	}
	if cfg.HotDatabase == "" {
		cfg.HotDatabase = "semantic_hot"
	}
	if cfg.ColdDatabase == "" {
		cfg.ColdDatabase = "semantic_cold"
	}
	if cfg.RealtimeResourceGroup == "" {
		cfg.RealtimeResourceGroup = "realtime_high"
	}
	if cfg.AnalyticsResourceGroup == "" {
		cfg.AnalyticsResourceGroup = "analytics_normal"
	}

	// Connect to StarRocks
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=30s&parseTime=true",
		cfg.StarRocksUser, cfg.StarRocksPassword,
		cfg.StarRocksHost, cfg.StarRocksPort, cfg.StarRocksDB)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to StarRocks: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("StarRocks ping failed: %w", err)
	}

	engine := &UnifiedCalcEngine{
		starrocks:    db,
		config:       cfg,
		calcRegistry: make(map[string]*CalculationDef),
		resultCache:  NewResultCache(cfg.CacheMaxEntries, cfg.CacheTTL),
	}

	// Load calculation definitions
	if err := engine.loadCalculationRegistry(); err != nil {
		// Log warning but don't fail - calculations can be registered dynamically
		fmt.Printf("Warning: failed to load calculation registry: %v\n", err)
	}

	return engine, nil
}

// ============================================================================
// REAL-TIME CALCULATION API (for your app)
// ============================================================================

// Calculate executes a calculation in real-time
func (e *UnifiedCalcEngine) Calculate(ctx context.Context, req *CalcRequest) (*CalcResponse, error) {
	start := time.Now()

	// Validate tenant isolation
	if req.TenantID == "" || req.DatasourceID == "" {
		return nil, fmt.Errorf("tenant_id and datasource_id are required")
	}

	// Check cache first (unless bypassed)
	if !req.BypassCache {
		if cached := e.resultCache.Get(req); cached != nil {
			cached.FromCache = true
			cached.ExecutionTime = time.Since(start)
			return cached, nil
		}
	}

	// Determine query mode
	mode := req.Mode
	if mode == "" || mode == ModeAuto {
		mode = e.determineQueryMode(req)
	}

	// Set resource group based on mode
	resourceGroup := e.config.AnalyticsResourceGroup
	if mode == ModeRealtime {
		resourceGroup = e.config.RealtimeResourceGroup
	}

	// Execute with resource group
	conn, err := e.starrocks.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Close()

	// Set resource group for QoS
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET resource_group = '%s'", resourceGroup)); err != nil {
		// Log but continue - resource groups may not be configured
	}

	// Execute calculation
	var result *CalcResponse
	switch {
	case req.CalculationID != "":
		result, err = e.executeRegisteredCalc(ctx, conn, req, mode)
	case req.MetricName != "":
		result, err = e.executeMetricCalc(ctx, conn, req, mode)
	default:
		return nil, fmt.Errorf("either calculation_id or metric_name required")
	}

	if err != nil {
		return nil, err
	}

	result.ExecutionTime = time.Since(start)
	result.Mode = mode

	// Cache result if cacheable
	if !req.BypassCache {
		e.resultCache.Set(req, result)
	}

	return result, nil
}

// CalculateNAV calculates Net Asset Value in real-time
func (e *UnifiedCalcEngine) CalculateNAV(ctx context.Context, tenantID, datasourceID, portfolioID string, asOfDate time.Time) (*CalcResponse, error) {
	return e.Calculate(ctx, &CalcRequest{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		MetricName:   "NAV",
		Mode:         ModeRealtime,
		Params: map[string]interface{}{
			"portfolio_id": portfolioID,
			"as_of_date":   asOfDate,
		},
	})
}

// CalculateReturns calculates portfolio returns in real-time
func (e *UnifiedCalcEngine) CalculateReturns(ctx context.Context, tenantID, datasourceID, portfolioID string,
	startDate, endDate time.Time, returnType string) (*CalcResponse, error) {
	return e.Calculate(ctx, &CalcRequest{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		MetricName:   "Returns",
		Mode:         ModeRealtime,
		Params: map[string]interface{}{
			"portfolio_id": portfolioID,
			"start_date":   startDate,
			"end_date":     endDate,
			"return_type":  returnType, // "daily", "monthly", "ytd", "inception"
		},
	})
}

// CalculateRiskMetrics calculates risk metrics in real-time
func (e *UnifiedCalcEngine) CalculateRiskMetrics(ctx context.Context, tenantID, datasourceID, portfolioID string,
	metrics []string, params map[string]interface{}) (*CalcResponse, error) {

	calcParams := map[string]interface{}{
		"portfolio_id": portfolioID,
		"metrics":      metrics, // ["var", "cvar", "volatility", "sharpe", "beta"]
	}
	for k, v := range params {
		calcParams[k] = v
	}

	return e.Calculate(ctx, &CalcRequest{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		MetricName:   "RiskMetrics",
		Mode:         ModeRealtime,
		Params:       calcParams,
	})
}

// ============================================================================
// QUERY EXECUTION
// ============================================================================

func (e *UnifiedCalcEngine) executeRegisteredCalc(ctx context.Context, conn *sql.Conn,
	req *CalcRequest, mode QueryMode) (*CalcResponse, error) {

	e.mu.RLock()
	calcDef, ok := e.calcRegistry[req.CalculationID]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("calculation not found: %s", req.CalculationID)
	}

	// Build SQL from formula template
	sql, err := e.buildSQL(calcDef.Formula, req, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL: %w", err)
	}

	// Execute query
	return e.executeSQL(ctx, conn, sql, req, calcDef.OutputType)
}

func (e *UnifiedCalcEngine) executeMetricCalc(ctx context.Context, conn *sql.Conn,
	req *CalcRequest, mode QueryMode) (*CalcResponse, error) {

	// Get database based on mode
	database := e.getDatabase(mode)

	var sql string
	var outputType string

	switch req.MetricName {
	case "NAV":
		sql = e.buildNAVQuery(database, req)
		outputType = "breakdown"
	case "Returns":
		sql = e.buildReturnsQuery(database, req)
		outputType = "timeseries"
	case "RiskMetrics":
		sql = e.buildRiskQuery(database, req)
		outputType = "breakdown"
	case "Holdings":
		sql = e.buildHoldingsQuery(database, req)
		outputType = "breakdown"
	case "Transactions":
		sql = e.buildTransactionsQuery(database, req)
		outputType = "breakdown"
	case "Performance":
		sql = e.buildPerformanceQuery(database, req)
		outputType = "breakdown"
	default:
		return nil, fmt.Errorf("unknown metric: %s", req.MetricName)
	}

	return e.executeSQL(ctx, conn, sql, req, outputType)
}

func (e *UnifiedCalcEngine) executeSQL(ctx context.Context, conn *sql.Conn,
	sqlQuery string, req *CalcRequest, outputType string) (*CalcResponse, error) {

	rows, err := conn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get columns
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Scan results
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

	// Build response based on output type
	response := &CalcResponse{
		CalculationID: req.CalculationID,
		TenantID:      req.TenantID,
		DatasourceID:  req.DatasourceID,
		DataSource:    sqlQuery[:50] + "...", // First 50 chars for debugging
	}

	switch outputType {
	case "scalar":
		if len(results) > 0 {
			for _, v := range results[0] {
				response.Value = v
				break
			}
		}
	case "timeseries", "breakdown":
		response.Breakdown = results
		if len(results) > 0 {
			// Use first numeric value as primary value
			for _, v := range results[0] {
				if f, ok := toFloat64(v); ok {
					response.Value = f
					break
				}
			}
		}
	}

	return response, nil
}

// ============================================================================
// SQL BUILDERS (StarRocks-optimized)
// ============================================================================

func (e *UnifiedCalcEngine) buildNAVQuery(database string, req *CalcRequest) string {
	portfolioID := getStringParam(req.Params, "portfolio_id")
	asOfDate := getTimeParam(req.Params, "as_of_date", time.Now())

	return fmt.Sprintf(`
		SELECT 
			h.ticker,
			h.quantity,
			p.price,
			h.currency,
			COALESCE(fx.rate, 1.0) as fx_rate,
			(h.quantity * p.price * COALESCE(fx.rate, 1.0)) as position_value
		FROM %s.holdings h
		LEFT JOIN %s.prices p ON h.ticker = p.ticker AND p.price_date = '%s'
		LEFT JOIN %s.fx_rates fx ON h.currency = fx.from_currency AND fx.to_currency = 'USD' AND fx.rate_date = '%s'
		WHERE h.tenant_id = '%s'
		  AND h.datasource_id = '%s'
		  AND h.portfolio_id = '%s'
		  AND h.as_of_date <= '%s'
		ORDER BY position_value DESC
	`, database, database, asOfDate.Format("2006-01-02"),
		database, asOfDate.Format("2006-01-02"),
		req.TenantID, req.DatasourceID, portfolioID,
		asOfDate.Format("2006-01-02"))
}

func (e *UnifiedCalcEngine) buildReturnsQuery(database string, req *CalcRequest) string {
	portfolioID := getStringParam(req.Params, "portfolio_id")
	startDate := getTimeParam(req.Params, "start_date", time.Now().AddDate(0, -1, 0))
	endDate := getTimeParam(req.Params, "end_date", time.Now())
	returnType := getStringParam(req.Params, "return_type")

	granularity := "DAY"
	if returnType == "monthly" {
		granularity = "MONTH"
	}

	return fmt.Sprintf(`
		SELECT 
			time_slice(as_of_date, INTERVAL 1 %s) as period,
			SUM(nav_value) as nav,
			(SUM(nav_value) - LAG(SUM(nav_value)) OVER (ORDER BY time_slice(as_of_date, INTERVAL 1 %s))) / 
				NULLIF(LAG(SUM(nav_value)) OVER (ORDER BY time_slice(as_of_date, INTERVAL 1 %s)), 0) as period_return
		FROM %s.portfolio_nav
		WHERE tenant_id = '%s'
		  AND datasource_id = '%s'
		  AND portfolio_id = '%s'
		  AND as_of_date BETWEEN '%s' AND '%s'
		GROUP BY time_slice(as_of_date, INTERVAL 1 %s)
		ORDER BY period
	`, granularity, granularity, granularity,
		database, req.TenantID, req.DatasourceID, portfolioID,
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"),
		granularity)
}

func (e *UnifiedCalcEngine) buildRiskQuery(database string, req *CalcRequest) string {
	portfolioID := getStringParam(req.Params, "portfolio_id")
	lookbackDays := getIntParam(req.Params, "lookback_days", 252)

	return fmt.Sprintf(`
		WITH daily_returns AS (
			SELECT 
				as_of_date,
				(nav_value - LAG(nav_value) OVER (ORDER BY as_of_date)) / 
					NULLIF(LAG(nav_value) OVER (ORDER BY as_of_date), 0) as daily_return
			FROM %s.portfolio_nav
			WHERE tenant_id = '%s'
			  AND datasource_id = '%s'
			  AND portfolio_id = '%s'
			  AND as_of_date >= DATE_SUB(CURDATE(), INTERVAL %d DAY)
		)
		SELECT 
			AVG(daily_return) * 252 as annualized_return,
			STDDEV(daily_return) * SQRT(252) as volatility,
			AVG(daily_return) / NULLIF(STDDEV(daily_return), 0) * SQRT(252) as sharpe_ratio,
			PERCENTILE_APPROX(daily_return, 0.05) as var_95,
			MIN(daily_return) as max_daily_loss
		FROM daily_returns
		WHERE daily_return IS NOT NULL
	`, database, req.TenantID, req.DatasourceID, portfolioID, lookbackDays)
}

func (e *UnifiedCalcEngine) buildHoldingsQuery(database string, req *CalcRequest) string {
	portfolioID := getStringParam(req.Params, "portfolio_id")
	asOfDate := getTimeParam(req.Params, "as_of_date", time.Now())

	return fmt.Sprintf(`
		SELECT 
			h.holding_id,
			h.ticker,
			h.security_name,
			h.quantity,
			h.cost_basis,
			h.currency,
			h.sector,
			h.asset_class,
			p.price as current_price,
			(h.quantity * p.price) as market_value,
			((h.quantity * p.price) - h.cost_basis) as unrealized_pnl,
			((h.quantity * p.price) - h.cost_basis) / NULLIF(h.cost_basis, 0) * 100 as return_pct
		FROM %s.holdings h
		LEFT JOIN %s.prices p ON h.ticker = p.ticker AND p.price_date = '%s'
		WHERE h.tenant_id = '%s'
		  AND h.datasource_id = '%s'
		  AND h.portfolio_id = '%s'
		  AND h.as_of_date <= '%s'
		ORDER BY market_value DESC
	`, database, database, asOfDate.Format("2006-01-02"),
		req.TenantID, req.DatasourceID, portfolioID, asOfDate.Format("2006-01-02"))
}

func (e *UnifiedCalcEngine) buildTransactionsQuery(database string, req *CalcRequest) string {
	portfolioID := getStringParam(req.Params, "portfolio_id")
	startDate := getTimeParam(req.Params, "start_date", time.Now().AddDate(0, -3, 0))
	endDate := getTimeParam(req.Params, "end_date", time.Now())

	return fmt.Sprintf(`
		SELECT 
			transaction_id,
			transaction_date,
			ticker,
			transaction_type,
			quantity,
			price,
			(quantity * price) as amount,
			currency,
			fees,
			notes
		FROM %s.transactions
		WHERE tenant_id = '%s'
		  AND datasource_id = '%s'
		  AND portfolio_id = '%s'
		  AND transaction_date BETWEEN '%s' AND '%s'
		ORDER BY transaction_date DESC
	`, database, req.TenantID, req.DatasourceID, portfolioID,
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
}

func (e *UnifiedCalcEngine) buildPerformanceQuery(database string, req *CalcRequest) string {
	portfolioID := getStringParam(req.Params, "portfolio_id")

	return fmt.Sprintf(`
		SELECT 
			'1D' as period, return_1d as return_pct FROM %s.portfolio_performance WHERE portfolio_id = '%s' AND tenant_id = '%s' AND datasource_id = '%s'
		UNION ALL SELECT '1W', return_1w FROM %s.portfolio_performance WHERE portfolio_id = '%s' AND tenant_id = '%s' AND datasource_id = '%s'
		UNION ALL SELECT '1M', return_1m FROM %s.portfolio_performance WHERE portfolio_id = '%s' AND tenant_id = '%s' AND datasource_id = '%s'
		UNION ALL SELECT '3M', return_3m FROM %s.portfolio_performance WHERE portfolio_id = '%s' AND tenant_id = '%s' AND datasource_id = '%s'
		UNION ALL SELECT 'YTD', return_ytd FROM %s.portfolio_performance WHERE portfolio_id = '%s' AND tenant_id = '%s' AND datasource_id = '%s'
		UNION ALL SELECT '1Y', return_1y FROM %s.portfolio_performance WHERE portfolio_id = '%s' AND tenant_id = '%s' AND datasource_id = '%s'
		UNION ALL SELECT 'ITD', return_itd FROM %s.portfolio_performance WHERE portfolio_id = '%s' AND tenant_id = '%s' AND datasource_id = '%s'
	`, database, portfolioID, req.TenantID, req.DatasourceID,
		database, portfolioID, req.TenantID, req.DatasourceID,
		database, portfolioID, req.TenantID, req.DatasourceID,
		database, portfolioID, req.TenantID, req.DatasourceID,
		database, portfolioID, req.TenantID, req.DatasourceID,
		database, portfolioID, req.TenantID, req.DatasourceID,
		database, portfolioID, req.TenantID, req.DatasourceID)
}

// ============================================================================
// HOT → COLD DATA LIFECYCLE
// ============================================================================

// DataLifecycleConfig configures the hot-to-cold data migration
type DataLifecycleConfig struct {
	Tables          []string      `json:"tables"`           // Tables to migrate
	HotRetention    time.Duration `json:"hot_retention"`    // How long to keep in hot
	BatchSize       int           `json:"batch_size"`       // Rows per batch
	ParquetPath     string        `json:"parquet_path"`     // S3/HDFS path for Parquet
	PartitionColumn string        `json:"partition_column"` // Usually as_of_date
}

// MigrateHotToCold migrates data from hot native tables to cold Parquet storage
func (e *UnifiedCalcEngine) MigrateHotToCold(ctx context.Context, cfg *DataLifecycleConfig) (*MigrationResult, error) {
	result := &MigrationResult{
		StartedAt: time.Now(),
		Tables:    make(map[string]TableMigrationResult),
	}

	cutoffDate := time.Now().Add(-cfg.HotRetention)

	for _, table := range cfg.Tables {
		tableResult, err := e.migrateTable(ctx, table, cutoffDate, cfg)
		if err != nil {
			tableResult.Error = err.Error()
		}
		result.Tables[table] = tableResult
		result.TotalRowsMigrated += tableResult.RowsMigrated
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	return result, nil
}

// MigrationResult contains migration statistics
type MigrationResult struct {
	StartedAt         time.Time                       `json:"started_at"`
	CompletedAt       time.Time                       `json:"completed_at"`
	Duration          time.Duration                   `json:"duration"`
	Tables            map[string]TableMigrationResult `json:"tables"`
	TotalRowsMigrated int64                           `json:"total_rows_migrated"`
}

// TableMigrationResult contains per-table migration stats
type TableMigrationResult struct {
	RowsMigrated int64  `json:"rows_migrated"`
	RowsDeleted  int64  `json:"rows_deleted"`
	Error        string `json:"error,omitempty"`
}

func (e *UnifiedCalcEngine) migrateTable(ctx context.Context, table string, cutoffDate time.Time, cfg *DataLifecycleConfig) (TableMigrationResult, error) {
	result := TableMigrationResult{}

	// 1. Export hot data to Parquet via StarRocks EXPORT
	exportSQL := fmt.Sprintf(`
		EXPORT TABLE %s.%s
		TO '%s/%s/'
		PARTITION (dt < '%s')
		WITH BROKER 'hdfs_broker'
		PROPERTIES (
			"format" = "parquet",
			"max_file_size" = "256MB"
		)
	`, e.config.HotDatabase, table, cfg.ParquetPath, table, cutoffDate.Format("2006-01-02"))

	if _, err := e.starrocks.ExecContext(ctx, exportSQL); err != nil {
		return result, fmt.Errorf("export failed: %w", err)
	}

	// 2. Count migrated rows
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s.%s 
		WHERE %s < '%s'
	`, e.config.HotDatabase, table, cfg.PartitionColumn, cutoffDate.Format("2006-01-02"))

	if err := e.starrocks.QueryRowContext(ctx, countSQL).Scan(&result.RowsMigrated); err != nil {
		return result, fmt.Errorf("count failed: %w", err)
	}

	// 3. Delete from hot table
	deleteSQL := fmt.Sprintf(`
		DELETE FROM %s.%s 
		WHERE %s < '%s'
	`, e.config.HotDatabase, table, cfg.PartitionColumn, cutoffDate.Format("2006-01-02"))

	res, err := e.starrocks.ExecContext(ctx, deleteSQL)
	if err != nil {
		return result, fmt.Errorf("delete failed: %w", err)
	}
	result.RowsDeleted, _ = res.RowsAffected()

	// 4. Refresh external table catalog (for querying cold data)
	refreshSQL := fmt.Sprintf(`
		REFRESH EXTERNAL TABLE %s.%s
	`, e.config.ColdDatabase, table)
	_, _ = e.starrocks.ExecContext(ctx, refreshSQL)

	return result, nil
}

// ============================================================================
// CALCULATION REGISTRY
// ============================================================================

// RegisterCalculation adds a calculation to the registry
func (e *UnifiedCalcEngine) RegisterCalculation(calc *CalculationDef) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calcRegistry[calc.ID] = calc
}

func (e *UnifiedCalcEngine) loadCalculationRegistry() error {
	// Load from database
	query := `
		SELECT id, name, description, formula, input_params, output_type, 
		       cacheable, cache_ttl_seconds, data_source, tags
		FROM calculation_definitions
		WHERE active = true
	`

	rows, err := e.starrocks.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	e.mu.Lock()
	defer e.mu.Unlock()

	for rows.Next() {
		var calc CalculationDef
		var inputParamsJSON, tagsJSON string
		var cacheTTLSeconds int

		if err := rows.Scan(&calc.ID, &calc.Name, &calc.Description, &calc.Formula,
			&inputParamsJSON, &calc.OutputType, &calc.Cacheable, &cacheTTLSeconds,
			&calc.DataSource, &tagsJSON); err != nil {
			continue
		}

		json.Unmarshal([]byte(inputParamsJSON), &calc.InputParams)
		json.Unmarshal([]byte(tagsJSON), &calc.Tags)
		calc.CacheTTL = time.Duration(cacheTTLSeconds) * time.Second

		e.calcRegistry[calc.ID] = &calc
	}

	return nil
}

// ============================================================================
// HELPERS
// ============================================================================

func (e *UnifiedCalcEngine) determineQueryMode(req *CalcRequest) QueryMode {
	// Check for explicit date range
	if startDate := getTimeParam(req.Params, "start_date", time.Time{}); !startDate.IsZero() {
		boundary := time.Now().AddDate(0, 0, -e.config.HotRetentionDays)
		if startDate.Before(boundary) {
			return ModeCold
		}
	}

	if asOfDate := getTimeParam(req.Params, "as_of_date", time.Time{}); !asOfDate.IsZero() {
		boundary := time.Now().AddDate(0, 0, -e.config.HotRetentionDays)
		if asOfDate.Before(boundary) {
			return ModeCold
		}
	}

	return ModeHot
}

func (e *UnifiedCalcEngine) getDatabase(mode QueryMode) string {
	switch mode {
	case ModeCold:
		return e.config.ColdDatabase
	default:
		return e.config.HotDatabase
	}
}

func (e *UnifiedCalcEngine) buildSQL(formula string, req *CalcRequest, mode QueryMode) (string, error) {
	// Simple template replacement - could use text/template for more complex cases
	sql := formula
	database := e.getDatabase(mode)

	// Replace placeholders
	sql = replaceAll(sql, "{{database}}", database)
	sql = replaceAll(sql, "{{tenant_id}}", req.TenantID)
	sql = replaceAll(sql, "{{datasource_id}}", req.DatasourceID)

	for key, val := range req.Params {
		placeholder := fmt.Sprintf("{{%s}}", key)
		sql = replaceAll(sql, placeholder, fmt.Sprintf("%v", val))
	}

	return sql, nil
}

// Close closes the engine connections
func (e *UnifiedCalcEngine) Close() error {
	if e.starrocks != nil {
		return e.starrocks.Close()
	}
	return nil
}

// Helper functions
func getStringParam(params map[string]interface{}, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func getIntParam(params map[string]interface{}, key string, def int) int {
	if v, ok := params[key].(int); ok {
		return v
	}
	if v, ok := params[key].(float64); ok {
		return int(v)
	}
	return def
}

func getTimeParam(params map[string]interface{}, key string, def time.Time) time.Time {
	if v, ok := params[key].(time.Time); ok {
		return v
	}
	if v, ok := params[key].(string); ok {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			return t
		}
	}
	return def
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func replaceAll(s, old, new string) string {
	for {
		if idx := indexOf(s, old); idx >= 0 {
			s = s[:idx] + new + s[idx+len(old):]
		} else {
			break
		}
	}
	return s
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
