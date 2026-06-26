package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// Config holds database configuration
type Config struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	User            string        `json:"user"`
	Password        string        `json:"password"`
	Database        string        `json:"database"`
	SSLMode         string        `json:"ssl_mode"`
	MaxConnections  int32         `json:"max_connections"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

// Client wraps pgxpool with logging and health checks
type Client struct {
	pool   *pgxpool.Pool
	logger *logrus.Entry
	config Config
}

// NewClient creates a new database client with connection pooling
func NewClient(ctx context.Context, cfg Config, logger *logrus.Entry) (*Client, error) {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}

	logger = logger.WithField("component", "database")

	// Build connection string
	connStr := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)

	// Create pool config
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		logger.WithError(err).Error("Failed to parse database config")
		return nil, err
	}

	// Set pool parameters
	if cfg.MaxConnections > 0 {
		poolConfig.MaxConns = cfg.MaxConnections
	} else {
		poolConfig.MaxConns = 20 // Default
	}

	if cfg.ConnMaxLifetime > 0 {
		poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	} else {
		poolConfig.MaxConnLifetime = 15 * time.Minute
	}

	if cfg.ConnMaxIdleTime > 0 {
		poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime
	} else {
		poolConfig.MaxConnIdleTime = 5 * time.Minute
	}

	poolConfig.MinConns = 5 // Maintain minimum connections

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.WithError(err).Error("Failed to create database pool")
		return nil, err
	}

	client := &Client{
		pool:   pool,
		logger: logger,
		config: cfg,
	}

	// Test connection
	if err := client.Health(ctx); err != nil {
		pool.Close()
		logger.WithError(err).Error("Database health check failed")
		return nil, err
	}

	logger.WithFields(map[string]interface{}{
		"host":            cfg.Host,
		"port":            cfg.Port,
		"database":        cfg.Database,
		"max_connections": poolConfig.MaxConns,
	}).Info("Connected to database")

	return client, nil
}

// Health performs a database health check
func (c *Client) Health(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// Pool returns the underlying pgxpool.Pool for direct access if needed
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

// Close closes the database connection pool
func (c *Client) Close() {
	if c.pool != nil {
		c.pool.Close()
		c.logger.Info("Database connection pool closed")
	}
}

// Exec executes a statement and returns number of rows affected
func (c *Client) Exec(ctx context.Context, sql string, args ...interface{}) (int64, error) {
	result, err := c.pool.Exec(ctx, sql, args...)
	if err != nil {
		c.logger.WithError(err).WithField("sql", sql).Error("Exec failed")
		return 0, err
	}
	return result.RowsAffected(), nil
}

// QueryRow queries a single row
func (c *Client) QueryRow(ctx context.Context, sql string, args ...interface{}) interface{} {
	return c.pool.QueryRow(ctx, sql, args...)
}

// Query queries multiple rows
func (c *Client) Query(ctx context.Context, sql string, args ...interface{}) interface{} {
	rows, err := c.pool.Query(ctx, sql, args...)
	if err != nil {
		c.logger.WithError(err).WithField("sql", sql).Error("Query failed")
		return nil
	}
	return rows
}

// Transaction executes a function within a database transaction
func (c *Client) Transaction(ctx context.Context, fn func(ctx context.Context, tx interface{}) error) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		c.logger.WithError(err).Error("Failed to begin transaction")
		return err
	}

	err = fn(ctx, tx)
	if err != nil {
		_ = tx.Rollback(ctx)
		c.logger.WithError(err).Error("Transaction rolled back")
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		c.logger.WithError(err).Error("Failed to commit transaction")
		return err
	}

	return nil
}

// GetPoolStats returns pool statistics for monitoring
func (c *Client) GetPoolStats() map[string]interface{} {
	stat := c.pool.Stat()
	return map[string]interface{}{
		"total_conns":      stat.TotalConns(),
		"acquire_count":    stat.AcquireCount(),
		"acquire_duration": stat.AcquireDuration(),
		"idle_conns":       stat.IdleConns(),
		"max_conns":        c.config.MaxConnections,
	}
}
