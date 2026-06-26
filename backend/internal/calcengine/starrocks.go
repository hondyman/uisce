package calcengine

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // StarRocks uses MySQL protocol
)

// StarRocksConfig holds StarRocks connection parameters
type StarRocksConfig struct {
	Host     string        `yaml:"host" json:"host"`
	Port     int           `yaml:"port" json:"port"` // Default: 9030
	User     string        `yaml:"user" json:"user"`
	Password string        `yaml:"password" json:"password"`
	Database string        `yaml:"database" json:"database"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
	MaxConns int           `yaml:"max_conns" json:"max_conns"`
}

// StarRocksClient provides high-performance OLAP queries via StarRocks
type StarRocksClient struct {
	db     *sql.DB
	config *StarRocksConfig
}

// NewStarRocksClient creates a new StarRocks client using MySQL protocol
func NewStarRocksClient(cfg *StarRocksConfig) (*StarRocksClient, error) {
	if cfg.Port == 0 {
		cfg.Port = 9030 // Default StarRocks FE query port
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 20
	}

	// MySQL DSN for StarRocks: user:password@tcp(host:port)/database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%s&parseTime=true&multiStatements=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
		cfg.Timeout.String())

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open StarRocks connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping StarRocks: %w", err)
	}

	return &StarRocksClient{
		db:     db,
		config: cfg,
	}, nil
}

// Query executes a read query and returns results
func (c *StarRocksClient) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("StarRocks query failed: %w", err)
	}

	return rows, nil
}

// QueryRow executes a query returning a single row
func (c *StarRocksClient) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	return c.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a non-query statement (INSERT, UPDATE, DELETE)
func (c *StarRocksClient) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("StarRocks exec failed: %w", err)
	}

	return result, nil
}

// QueryAggregation executes an aggregation query optimized for StarRocks
func (c *StarRocksClient) QueryAggregation(ctx context.Context, tenantID, datasourceID string,
	table string, measures []string, dimensions []string, filters map[string]interface{}) ([]map[string]interface{}, error) {

	// Build optimized aggregation query
	query := buildAggregationQuery(table, measures, dimensions, filters, tenantID, datasourceID)

	rows, err := c.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToMaps(rows)
}

// QueryMetric executes a metric query with caching hint
func (c *StarRocksClient) QueryMetric(ctx context.Context, tenantID, datasourceID, metricID string,
	asOfDate time.Time) (*MetricResult, error) {

	query := `
		SELECT 
			metric_id,
			metric_value,
			as_of_date,
			computation_status,
			last_computed_at
		FROM metric_results
		WHERE tenant_id = ?
		  AND datasource_id = ?
		  AND metric_id = ?
		  AND as_of_date = ?
		LIMIT 1
	`

	var result MetricResult
	err := c.db.QueryRowContext(ctx, query, tenantID, datasourceID, metricID, asOfDate).Scan(
		&result.MetricID,
		&result.Value,
		&result.AsOfDate,
		&result.Status,
		&result.LastComputedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query metric: %w", err)
	}

	return &result, nil
}

// MetricResult represents a computed metric value
type MetricResult struct {
	MetricID       string    `json:"metric_id"`
	Value          float64   `json:"value"`
	AsOfDate       time.Time `json:"as_of_date"`
	Status         string    `json:"status"`
	LastComputedAt time.Time `json:"last_computed_at"`
}

// QueryTimeSeries executes a time series query optimized for StarRocks
func (c *StarRocksClient) QueryTimeSeries(ctx context.Context, tenantID, datasourceID string,
	table string, valueColumn string, timeColumn string,
	startDate, endDate time.Time, granularity string) ([]TimeSeriesPoint, error) {

	// StarRocks-optimized time series query using time_slice function
	query := fmt.Sprintf(`
		SELECT 
			time_slice(%s, INTERVAL 1 %s) AS time_bucket,
			SUM(%s) AS value,
			COUNT(*) AS count
		FROM %s
		WHERE tenant_id = ?
		  AND datasource_id = ?
		  AND %s >= ?
		  AND %s <= ?
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`, timeColumn, granularity, valueColumn, table, timeColumn, timeColumn)

	rows, err := c.Query(ctx, query, tenantID, datasourceID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TimeSeriesPoint
	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Timestamp, &point.Value, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan time series point: %w", err)
		}
		results = append(results, point)
	}

	return results, rows.Err()
}

// TimeSeriesPoint represents a single point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Count     int64     `json:"count"`
}

// ExecutePreAggregation refreshes a pre-aggregation table
func (c *StarRocksClient) ExecutePreAggregation(ctx context.Context,
	tenantID, datasourceID, preAggName, preAggSQL string) error {

	// StarRocks materialized view refresh
	refreshSQL := fmt.Sprintf(`
		INSERT OVERWRITE %s
		%s
	`, preAggName, preAggSQL)

	_, err := c.Exec(ctx, refreshSQL)
	if err != nil {
		return fmt.Errorf("failed to refresh pre-aggregation %s: %w", preAggName, err)
	}

	return nil
}

// SetResourceGroup sets the resource group for QoS management
func (c *StarRocksClient) SetResourceGroup(ctx context.Context, resourceGroup string) error {
	_, err := c.Exec(ctx, fmt.Sprintf("SET resource_group = '%s'", resourceGroup))
	return err
}

// TestConnection verifies StarRocks connectivity
func (c *StarRocksClient) TestConnection(ctx context.Context) error {
	var result string
	err := c.db.QueryRowContext(ctx, "SELECT 'OK'").Scan(&result)
	if err != nil {
		return fmt.Errorf("StarRocks connection test failed: %w", err)
	}
	if result != "OK" {
		return fmt.Errorf("unexpected StarRocks test result: %s", result)
	}
	return nil
}

// GetDB returns the underlying database connection
func (c *StarRocksClient) GetDB() *sql.DB {
	return c.db
}

// Close closes the StarRocks connection
func (c *StarRocksClient) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Helper functions

func buildAggregationQuery(table string, measures []string, dimensions []string,
	filters map[string]interface{}, tenantID, datasourceID string) string {

	// Build SELECT clause
	selectParts := make([]string, 0, len(measures)+len(dimensions))
	for _, dim := range dimensions {
		selectParts = append(selectParts, dim)
	}
	for _, measure := range measures {
		selectParts = append(selectParts, measure)
	}

	// Build WHERE clause with mandatory tenant isolation
	whereParts := []string{
		fmt.Sprintf("tenant_id = '%s'", tenantID),
		fmt.Sprintf("datasource_id = '%s'", datasourceID),
	}
	for col, val := range filters {
		whereParts = append(whereParts, fmt.Sprintf("%s = '%v'", col, val))
	}

	// Build GROUP BY clause
	groupBy := ""
	if len(dimensions) > 0 {
		groupBy = fmt.Sprintf("GROUP BY %s", joinStrings(dimensions, ", "))
	}

	return fmt.Sprintf("SELECT %s FROM %s WHERE %s %s",
		joinStrings(selectParts, ", "),
		table,
		joinStrings(whereParts, " AND "),
		groupBy)
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func scanToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
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

	return results, rows.Err()
}
