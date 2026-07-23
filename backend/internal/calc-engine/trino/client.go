package trino

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ClientConfig holds Trino connection configuration
type ClientConfig struct {
	Host     string        // e.g., "192.168.86.55"
	Port     int           // e.g., 8090
	Database string        // e.g., "iceberg"
	Schema   string        // e.g., "demo"
	User     string        // e.g., "admin"
	Password string        // optional; can be empty
	Timeout  time.Duration // query timeout
}

// Client wraps a database connection to Trino
type Client struct {
	db     *sql.DB
	config *ClientConfig
}

// NewClient creates a new Trino client
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}

	// Construct JDBC-style DSN for Trino
	// Format: trino://[user[:password]@]host[:port]/[catalog][/schema]
	dsn := fmt.Sprintf("trino://%s@%s:%d/%s/%s",
		cfg.User, cfg.Host, cfg.Port, cfg.Database, cfg.Schema)

	// If password provided, include it
	if cfg.Password != "" {
		dsn = fmt.Sprintf("trino://%s:%s@%s:%d/%s/%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.Schema)
	}

	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open Trino connection: %w", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping Trino: %w", err)
	}

	return &Client{
		db:     db,
		config: cfg,
	}, nil
}

// ExecuteQuery executes a read query and returns rows
func (c *Client) ExecuteQuery(ctx context.Context, query string) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return rows, nil
}

// ExecuteMerge executes a MERGE statement (for PoP and anomaly writes)
func (c *Client) ExecuteMerge(ctx context.Context, mergeSQL string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, mergeSQL)
	if err != nil {
		return nil, fmt.Errorf("merge execution failed: %w", err)
	}
	defer rows.Close()

	// Extract stats from result if available
	stats := make(map[string]interface{})

	// Most Trino MERGE statements return row count info
	// Parse if supported, otherwise just confirm no error
	for rows.Next() {
		// Could parse columns if MERGE returns a result set
		// For now, just ensure we consumed all rows
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading merge results: %w", err)
	}

	stats["success"] = true
	return stats, nil
}

// QueryScalar executes a query and returns a single scalar value
func (c *Client) QueryScalar(ctx context.Context, query string, dest interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	err := c.db.QueryRowContext(ctx, query).Scan(dest)
	if err != nil {
		return fmt.Errorf("scalar query failed: %w", err)
	}

	return nil
}

// TestConnection verifies the Trino connection is healthy
func (c *Client) TestConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var result string
	err := c.db.QueryRowContext(ctx, "SELECT 'OK'").Scan(&result)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	if result != "OK" {
		return fmt.Errorf("unexpected connection test result: %s", result)
	}

	return nil
}

// Close closes the Trino connection
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *ClientConfig {
	return c.config
}
