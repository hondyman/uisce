package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"calendar-service/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// TestDB provides a test database connection
type TestDB struct {
	Pool       *pgxpool.Pool
	Config     database.Config
	connStr    string
	ctx        context.Context
	logger     *logrus.Entry
	testDBName string
}

// NewTestDB creates a test database instance
func NewTestDB(t *testing.T) *TestDB {
	ctx := context.Background()
	testDBName := fmt.Sprintf("test_calendar_%s", uuid.New().String()[:8])

	// Connect to default postgres DB to create test DB
	cfg := database.Config{
		Host:     "localhost",
		Port:     5432,
		User:     os.Getenv("TEST_DB_USER"),
		Password: os.Getenv("TEST_DB_PASSWORD"),
		Database: "postgres",
		SSLMode:  "disable",
	}

	if cfg.User == "" {
		cfg.User = "calendar_user"
	}
	if cfg.Password == "" {
		cfg.Password = "calendar_password"
	}

	connStr := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("Cannot connect to postgres for test setup: %v", err)
	}
	defer pool.Close()

	// Create test database
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", testDBName)); err != nil {
		t.Skipf("Cannot create test database: %v", err)
	}

	// Connect to test database
	testCfg := database.Config{
		Host:            cfg.Host,
		Port:            cfg.Port,
		User:            cfg.User,
		Password:        cfg.Password,
		Database:        testDBName,
		SSLMode:         "disable",
		MaxConnections:  5,
		ConnMaxLifetime: 1 * time.Minute,
		ConnMaxIdleTime: 30 * time.Second,
	}

	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.ErrorLevel) // Suppress test logging

	testPool, err := pgxpool.New(ctx, fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		testCfg.User, testCfg.Password, testCfg.Host, testCfg.Port, testCfg.Database, testCfg.SSLMode,
	))
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	if err := runMigrations(ctx, testPool); err != nil {
		testPool.Close()
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return &TestDB{
		Pool:       testPool,
		Config:     testCfg,
		connStr:    connStr,
		ctx:        ctx,
		logger:     logger,
		testDBName: testDBName,
	}
}

// Close cleans up the test database
func (tdb *TestDB) Close(t *testing.T) {
	tdb.Pool.Close()

	// Connect to default postgres to drop test DB
	cfg := database.Config{
		Host:     tdb.Config.Host,
		Port:     tdb.Config.Port,
		User:     tdb.Config.User,
		Password: tdb.Config.Password,
		Database: "postgres",
		SSLMode:  "disable",
	}

	connStr := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	pool, err := pgxpool.New(tdb.ctx, connStr)
	if err != nil {
		t.Logf("Warning: cannot clean up test database: %v", err)
		return
	}
	defer pool.Close()

	// Drop test database
	if _, err := pool.Exec(tdb.ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s CASCADE", tdb.testDBName)); err != nil {
		t.Logf("Warning: failed to drop test database: %v", err)
	}
}

// Context returns the test database context
func (tdb *TestDB) Context() context.Context {
	return tdb.ctx
}

// runMigrations applies schema migrations to test database
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Create migrations table
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			version VARCHAR(255) UNIQUE NOT NULL,
			description VARCHAR(255),
			installed_on TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return err
	}

	// Read and execute migration (inline for testing)
	migrationSQL := `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE EXTENSION IF NOT EXISTS "pg_trgm";
		
		CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(255) NOT NULL UNIQUE,
			region VARCHAR(50) NOT NULL DEFAULT 'us-east-1',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP WITH TIME ZONE,
			metadata JSONB DEFAULT '{}'::jsonb,
			CONSTRAINT tenant_region_valid CHECK (region IN ('us-east-1', 'us-west-2', 'eu-west-1', 'ap-southeast-1', 'global'))
		);

		CREATE TABLE IF NOT EXISTS calendar_profiles (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
			region VARCHAR(50) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP WITH TIME ZONE,
			version VARCHAR(64) NOT NULL DEFAULT 'v1',
			CONSTRAINT tenant_profile_unique UNIQUE (tenant_id, name, deleted_at IS NULL),
			CONSTRAINT profile_region_valid CHECK (region IN ('us-east-1', 'us-west-2', 'eu-west-1', 'ap-southeast-1', 'global'))
		);

		CREATE TABLE IF NOT EXISTS holidays (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			profile_id UUID NOT NULL REFERENCES calendar_profiles(id) ON DELETE CASCADE,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			holiday_date DATE NOT NULL,
			name VARCHAR(255) NOT NULL,
			region VARCHAR(50),
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT holiday_unique UNIQUE (profile_id, holiday_date)
		);

		CREATE TABLE IF NOT EXISTS blackout_windows (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			profile_id UUID NOT NULL REFERENCES calendar_profiles(id) ON DELETE CASCADE,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			start_time TIMESTAMP WITH TIME ZONE NOT NULL,
			end_time TIMESTAMP WITH TIME ZONE NOT NULL,
			title VARCHAR(255) NOT NULL,
			reason TEXT,
			rrule TEXT,
			is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
			recurrence_start DATE,
			recurrence_end DATE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT blackout_time_order CHECK (start_time < end_time)
		);

		CREATE TABLE IF NOT EXISTS resolved_calendar_metadata (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			profile_id UUID NOT NULL REFERENCES calendar_profiles(id) ON DELETE CASCADE,
			region VARCHAR(50) NOT NULL,
			resolved_at TIMESTAMP WITH TIME ZONE,
			version VARCHAR(64),
			holidays_count INT DEFAULT 0,
			blackouts_count INT DEFAULT 0,
			content_hash VARCHAR(64),
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT resolved_metadata_unique UNIQUE (tenant_id, profile_id, region)
		);

		CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			entity_type VARCHAR(50) NOT NULL,
			entity_id UUID,
			action VARCHAR(20) NOT NULL,
			changes JSONB,
			performed_by VARCHAR(255),
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_calendar_profiles_tenant ON calendar_profiles(tenant_id, deleted_at);
		CREATE INDEX IF NOT EXISTS idx_holidays_profile ON holidays(profile_id, holiday_date);
		CREATE INDEX IF NOT EXISTS idx_blackouts_profile ON blackout_windows(profile_id);
		CREATE INDEX IF NOT EXISTS idx_resolved_metadata_tenant ON resolved_calendar_metadata(tenant_id, profile_id);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs(tenant_id, created_at DESC);
		
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER IF NOT EXISTS tenants_update_updated_at BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
		CREATE TRIGGER IF NOT EXISTS calendar_profiles_update_updated_at BEFORE UPDATE ON calendar_profiles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
		CREATE TRIGGER IF NOT EXISTS blackout_windows_update_updated_at BEFORE UPDATE ON blackout_windows FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
		CREATE TRIGGER IF NOT EXISTS resolved_metadata_update_updated_at BEFORE UPDATE ON resolved_calendar_metadata FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
	`

	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		return fmt.Errorf("failed to run schema migration: %w", err)
	}

	return nil
}
