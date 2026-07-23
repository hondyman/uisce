package trino

import (
	"context"
	"database/sql"
	"fmt"
	// Trino driver temporarily disabled due to Go version compatibility issues
	// _ "github.com/trinodb/trino-go-client/trino"
)

// Client wraps a Trino database connection for Iceberg operations
type Client struct {
	db *sql.DB
}

// NewClient creates a new Trino client
func NewClient(dsn string) (*Client, error) {
	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open trino connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping trino: %w", err)
	}

	return &Client{db: db}, nil
}

// Execute runs a query that doesn't return rows (INSERT, UPDATE, DELETE)
func (c *Client) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...)
}

// Query runs a query that returns rows
func (c *Client) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

// QueryRow runs a query that returns a single row
func (c *Client) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

// Close closes the Trino connection
func (c *Client) Close() error {
	return c.db.Close()
}

// BeginTx starts a transaction (note: Trino has limited transaction support)
func (c *Client) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return c.db.BeginTx(ctx, opts)
}
