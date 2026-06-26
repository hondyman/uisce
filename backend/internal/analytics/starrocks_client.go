package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // StarRocks uses MySQL protocol
)

// Trade represents a trade record from Iceberg
type Trade struct {
	EventTime   time.Time
	TradeID     string
	TenantID    string
	PortfolioID string
	DeskID      string
	Symbol      string
	Side        string
	Quantity    float64
	Price       float64
	Notional    float64
	Currency    string
	BasisID     string
}

// DailyPnL represents aggregated daily P&L from materialized view
type DailyPnL struct {
	TradeDate     time.Time
	TenantID      string
	PortfolioID   string
	DeskID        string
	Currency      string
	TotalTrades   int64
	TotalVolume   float64
	TotalNotional float64
}

// ComplianceEvent represents a compliance event record
type ComplianceEvent struct {
	EventTime   time.Time
	RuleID      string
	TenantID    string
	PortfolioID string
	Status      string
	Details     string
}

// StarRocksClient handles analytics queries via StarRocks over Iceberg
type StarRocksClient struct {
	db           *sql.DB
	catalogName  string
	databaseName string
}

// StarRocksConfig holds StarRocks connection configuration
type StarRocksConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	CatalogName  string // External catalog name for Iceberg
	DatabaseName string // Database within the catalog
}

// NewStarRocksClient creates a new StarRocks connection
func NewStarRocksClient(cfg StarRocksConfig) (*StarRocksClient, error) {
	// StarRocks uses MySQL wire protocol
	// DSN format: user:password@tcp(host:port)/
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", cfg.User, cfg.Password, cfg.Host, cfg.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to StarRocks: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping StarRocks: %w", err)
	}

	catalogName := cfg.CatalogName
	if catalogName == "" {
		catalogName = "iceberg_catalog"
	}

	databaseName := cfg.DatabaseName
	if databaseName == "" {
		databaseName = "wealth"
	}

	return &StarRocksClient{
		db:           db,
		catalogName:  catalogName,
		databaseName: databaseName,
	}, nil
}

// NewStarRocksClientFromDSN creates a client from a DSN string
func NewStarRocksClientFromDSN(dsn string) (*StarRocksClient, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to StarRocks: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping StarRocks: %w", err)
	}

	return &StarRocksClient{
		db:           db,
		catalogName:  "iceberg_catalog",
		databaseName: "wealth",
	}, nil
}

// SetCatalog configures the Iceberg catalog and database to use
func (c *StarRocksClient) SetCatalog(catalogName, databaseName string) {
	c.catalogName = catalogName
	c.databaseName = databaseName
}

// tableName returns the fully qualified table name
func (c *StarRocksClient) tableName(table string) string {
	return fmt.Sprintf("%s.%s.%s", c.catalogName, c.databaseName, table)
}

// QueryTrades queries trades from Iceberg via StarRocks
func (c *StarRocksClient) QueryTrades(ctx context.Context, tenantID string, lookbackMinutes int) ([]Trade, error) {
	query := fmt.Sprintf(`
		SELECT 
			event_time, trade_id, tenant_id, portfolio_id, desk_id,
			symbol, side, quantity, price, notional, currency, basis_id
		FROM %s
		WHERE tenant_id = ?
		  AND event_time >= DATE_SUB(NOW(), INTERVAL ? MINUTE)
		ORDER BY event_time DESC
		LIMIT 10000
	`, c.tableName("trades"))

	rows, err := c.db.QueryContext(ctx, query, tenantID, lookbackMinutes)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var trades []Trade
	for rows.Next() {
		var t Trade
		if err := rows.Scan(
			&t.EventTime, &t.TradeID, &t.TenantID, &t.PortfolioID, &t.DeskID,
			&t.Symbol, &t.Side, &t.Quantity, &t.Price, &t.Notional, &t.Currency, &t.BasisID,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		trades = append(trades, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return trades, nil
}

// QueryTradesByPortfolio queries trades for a specific portfolio
func (c *StarRocksClient) QueryTradesByPortfolio(ctx context.Context, tenantID, portfolioID string, startTime, endTime time.Time) ([]Trade, error) {
	query := fmt.Sprintf(`
		SELECT 
			event_time, trade_id, tenant_id, portfolio_id, desk_id,
			symbol, side, quantity, price, notional, currency, basis_id
		FROM %s
		WHERE tenant_id = ?
		  AND portfolio_id = ?
		  AND event_time >= ?
		  AND event_time < ?
		ORDER BY event_time DESC
	`, c.tableName("trades"))

	rows, err := c.db.QueryContext(ctx, query, tenantID, portfolioID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var trades []Trade
	for rows.Next() {
		var t Trade
		if err := rows.Scan(
			&t.EventTime, &t.TradeID, &t.TenantID, &t.PortfolioID, &t.DeskID,
			&t.Symbol, &t.Side, &t.Quantity, &t.Price, &t.Notional, &t.Currency, &t.BasisID,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		trades = append(trades, t)
	}

	return trades, nil
}

// QueryDailyPnL uses the materialized view for fast aggregates
func (c *StarRocksClient) QueryDailyPnL(ctx context.Context, tenantID, portfolioID string, days int) ([]DailyPnL, error) {
	query := `
		SELECT 
			trade_date, tenant_id, portfolio_id, desk_id, currency,
			total_trades, total_volume, total_notional
		FROM daily_pnl_mv
		WHERE tenant_id = ?
		  AND portfolio_id = ?
		  AND trade_date >= DATE_SUB(CURRENT_DATE(), INTERVAL ? DAY)
		ORDER BY trade_date DESC
	`

	rows, err := c.db.QueryContext(ctx, query, tenantID, portfolioID, days)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []DailyPnL
	for rows.Next() {
		var p DailyPnL
		if err := rows.Scan(
			&p.TradeDate, &p.TenantID, &p.PortfolioID, &p.DeskID, &p.Currency,
			&p.TotalTrades, &p.TotalVolume, &p.TotalNotional,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, p)
	}

	return results, nil
}

// QueryComplianceEvents queries compliance events for a tenant
func (c *StarRocksClient) QueryComplianceEvents(ctx context.Context, tenantID string, lookbackMinutes int) ([]ComplianceEvent, error) {
	query := fmt.Sprintf(`
		SELECT 
			event_time, rule_id, tenant_id, portfolio_id, status, details
		FROM %s
		WHERE tenant_id = ?
		  AND event_time >= DATE_SUB(NOW(), INTERVAL ? MINUTE)
		ORDER BY event_time DESC
		LIMIT 1000
	`, c.tableName("compliance_events"))

	rows, err := c.db.QueryContext(ctx, query, tenantID, lookbackMinutes)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var events []ComplianceEvent
	for rows.Next() {
		var e ComplianceEvent
		if err := rows.Scan(
			&e.EventTime, &e.RuleID, &e.TenantID, &e.PortfolioID, &e.Status, &e.Details,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

// QueryComplianceStats queries aggregated compliance statistics
func (c *StarRocksClient) QueryComplianceStats(ctx context.Context, tenantID string, days int) (map[string]int64, error) {
	query := `
		SELECT 
			status, COUNT(*) as count
		FROM compliance_stats_mv
		WHERE tenant_id = ?
		  AND date >= DATE_SUB(CURRENT_DATE(), INTERVAL ? DAY)
		GROUP BY status
	`

	rows, err := c.db.QueryContext(ctx, query, tenantID, days)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		stats[status] = count
	}

	return stats, nil
}

// ExecuteQuery executes a raw SQL query (for ad-hoc analytics)
func (c *StarRocksClient) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

// GetIcebergSnapshot returns the current snapshot ID for a table (for audit/replay)
func (c *StarRocksClient) GetIcebergSnapshot(ctx context.Context, tableName string) (int64, error) {
	query := fmt.Sprintf(`
		SELECT current_snapshot_id 
		FROM %s.%s."%s$snapshots"
		ORDER BY committed_at DESC
		LIMIT 1
	`, c.catalogName, c.databaseName, tableName)

	var snapshotID int64
	err := c.db.QueryRowContext(ctx, query).Scan(&snapshotID)
	if err != nil {
		return 0, fmt.Errorf("failed to get snapshot: %w", err)
	}

	return snapshotID, nil
}

// QueryAtSnapshot queries data at a specific Iceberg snapshot (for replay)
func (c *StarRocksClient) QueryAtSnapshot(ctx context.Context, tableName string, snapshotID int64, query string, args ...interface{}) (*sql.Rows, error) {
	// StarRocks supports time travel via snapshot_id
	// Set the snapshot context before query
	setSnapshotQuery := fmt.Sprintf("SET @snapshot_id = %d", snapshotID)
	if _, err := c.db.ExecContext(ctx, setSnapshotQuery); err != nil {
		return nil, fmt.Errorf("failed to set snapshot context: %w", err)
	}

	return c.db.QueryContext(ctx, query, args...)
}

// HealthCheck verifies StarRocks connectivity
func (c *StarRocksClient) HealthCheck(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Close closes the database connection
func (c *StarRocksClient) Close() error {
	return c.db.Close()
}

// --- Resource Group Management for Multi-Tenancy ---

// CreateResourceGroup creates a resource group for tenant workload isolation
func (c *StarRocksClient) CreateResourceGroup(ctx context.Context, name string, cpuWeight int, memLimit string, concurrencyLimit int) error {
	query := fmt.Sprintf(`
		CREATE RESOURCE GROUP IF NOT EXISTS %s
		WITH (
			cpu_weight = %d,
			mem_limit = '%s',
			concurrency_limit = %d,
			type = 'normal'
		)
	`, name, cpuWeight, memLimit, concurrencyLimit)

	_, err := c.db.ExecContext(ctx, query)
	return err
}

// SetUserResourceGroup assigns a user to a resource group
func (c *StarRocksClient) SetUserResourceGroup(ctx context.Context, username, resourceGroup string) error {
	query := fmt.Sprintf("SET PROPERTY FOR '%s' 'resource_group' = '%s'", username, resourceGroup)
	_, err := c.db.ExecContext(ctx, query)
	return err
}
