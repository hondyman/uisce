package planner

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// TestDB provides a real PostgreSQL connection for tests
// Connection: postgres://postgres@100.84.126.19/alpha
type TestDB struct {
	conn *sql.DB
	t    *testing.T
}

// NewTestDB creates a connection to the alpha database
func NewTestDB(t *testing.T) (*TestDB, error) {
	// Build connection string from environment or defaults
	host := os.Getenv("PGHOST")
	if host == "" {
		host = "100.84.126.19"
	}

	user := os.Getenv("PGUSER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("PGPASSWORD")
	database := os.Getenv("PGDATABASE")
	if database == "" {
		database = "alpha"
	}

	// Construct DSN
	var dsn string
	if password != "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable&connect_timeout=10",
			user, password, host, database)
	} else {
		dsn = fmt.Sprintf("postgres://%s@%s/%s?sslmode=disable&connect_timeout=10",
			user, host, database)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Logf("Failed to open database connection to %s:%s - %v", host, database, err)
		return nil, err
	}

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Logf("Failed to ping database at %s:%s - set PGPASSWORD environment variable if password is required", host, database)
		return nil, err
	}

	return &TestDB{conn: db, t: t}, nil
}

// Close closes the database connection
func (tdb *TestDB) Close() error {
	return tdb.conn.Close()
}

// Setup initializes the planner schema
func (tdb *TestDB) Setup(ctx context.Context) error {
	// Create schema if not exists
	schema := `
	CREATE SCHEMA IF NOT EXISTS planner;
	
	CREATE TABLE IF NOT EXISTS planner.planner_decisions (
		id BIGSERIAL PRIMARY KEY,
		plan_id VARCHAR(255) UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		tenant_id VARCHAR(255) NOT NULL,
		query_type VARCHAR(50) NOT NULL,
		semantic_target VARCHAR(255) NOT NULL,
		selected_regions TEXT[] NOT NULL DEFAULT '{}',
		plan_type VARCHAR(50) NOT NULL,
		estimated_cost FLOAT8 NOT NULL DEFAULT 0.0,
		estimated_latency_ms FLOAT8 NOT NULL DEFAULT 0.0,
		degradation_strategy JSONB,
		explain TEXT,
		raw_request JSONB,
		raw_plan JSONB,
		region_health_snapshot JSONB,
		executed_at TIMESTAMP,
		actual_latency_ms FLOAT8,
		actual_cost FLOAT8,
		execution_status VARCHAR(50),
		execution_error TEXT
	);
	
	CREATE TABLE IF NOT EXISTS planner.region_performance (
		id BIGSERIAL PRIMARY KEY,
		region VARCHAR(50) UNIQUE NOT NULL,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		is_healthy BOOLEAN DEFAULT TRUE,
		latency_ms_p50 FLOAT8,
		latency_ms_p95 FLOAT8,
		latency_ms_p99 FLOAT8,
		error_rate FLOAT8,
		active_features INT DEFAULT 0,
		materialization_freshness_pct FLOAT8,
		cache_hit_rate FLOAT8
	);
	`

	_, err := tdb.conn.ExecContext(ctx, schema)
	return err
}

// Cleanup removes test data
func (tdb *TestDB) Cleanup(ctx context.Context) error {
	queries := []string{
		"TRUNCATE planner.planner_decisions CASCADE",
		"TRUNCATE planner.region_performance CASCADE",
	}

	for _, query := range queries {
		if _, err := tdb.conn.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
	}
	return nil
}

// InsertPlannedDecision inserts a test decision
func (tdb *TestDB) InsertPlannedDecision(ctx context.Context, decision *PlannerDecision) error {
	query := `
	INSERT INTO planner.planner_decisions (
		plan_id, tenant_id, query_type, semantic_target, selected_regions,
		plan_type, estimated_cost, estimated_latency_ms, degradation_strategy,
		explain, execution_status
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING id
	`

	var id int64
	return tdb.conn.QueryRowContext(ctx, query,
		decision.PlanID,
		decision.TenantID,
		decision.QueryType,
		decision.SemanticTarget,
		pq.Array(decision.SelectedRegions),
		decision.PlanType,
		decision.EstimatedCost,
		decision.EstimatedLatencyMS,
		decision.DegradationStrategy,
		decision.Explain,
		"pending",
	).Scan(&id)
}

// GetPlannedDecision retrieves a decision from the database
func (tdb *TestDB) GetPlannedDecision(ctx context.Context, planID string) (*PlannerDecision, error) {
	query := `
	SELECT
		plan_id, created_at, tenant_id, query_type, semantic_target,
		selected_regions, plan_type, estimated_cost, estimated_latency_ms,
		degradation_strategy, explain, execution_status, execution_error,
		actual_latency_ms, actual_cost, executed_at
	FROM planner.planner_decisions
	WHERE plan_id = $1
	`

	var decision PlannerDecision
	err := tdb.conn.QueryRowContext(ctx, query, planID).Scan(
		&decision.PlanID,
		&decision.CreatedAt,
		&decision.TenantID,
		&decision.QueryType,
		&decision.SemanticTarget,
		pq.Array(&decision.SelectedRegions),
		&decision.PlanType,
		&decision.EstimatedCost,
		&decision.EstimatedLatencyMS,
		&decision.DegradationStrategy,
		&decision.Explain,
		&decision.ExecutionStatus,
		&decision.ExecutionError,
		&decision.ActualLatencyMS,
		&decision.ActualCost,
		&decision.ExecutedAt,
	)

	return &decision, err
}

// UpdateDecisionExecution updates decision with actual results
func (tdb *TestDB) UpdateDecisionExecution(ctx context.Context, planID string, actualLatency float64, actualCost float64, status string) error {
	query := `
	UPDATE planner.planner_decisions
	SET executed_at = CURRENT_TIMESTAMP,
		actual_latency_ms = $1,
		actual_cost = $2,
		execution_status = $3
	WHERE plan_id = $4
	`

	_, err := tdb.conn.ExecContext(ctx, query, actualLatency, actualCost, status, planID)
	return err
}

// GetRegionPerformance retrieves region health
func (tdb *TestDB) GetRegionPerformance(ctx context.Context, region string) (*RegionPerformance, error) {
	query := `
	SELECT
		region, last_updated, is_healthy,
		latency_ms_p50, latency_ms_p95, latency_ms_p99,
		error_rate, active_features, materialization_freshness_pct, cache_hit_rate
	FROM planner.region_performance
	WHERE region = $1
	`

	var perf RegionPerformance
	err := tdb.conn.QueryRowContext(ctx, query, region).Scan(
		&perf.Region,
		&perf.LastUpdated,
		&perf.IsHealthy,
		&perf.LatencyP50MS,
		&perf.LatencyP95MS,
		&perf.LatencyP99MS,
		&perf.ErrorRate,
		&perf.ActiveFeatures,
		&perf.MaterializationFreshnessPercent,
		&perf.CacheHitRate,
	)

	return &perf, err
}

// InsertRegionPerformance inserts or updates region health
func (tdb *TestDB) InsertRegionPerformance(ctx context.Context, perf *RegionPerformance) error {
	query := `
	INSERT INTO planner.region_performance (
		region, is_healthy, latency_ms_p50, latency_ms_p95, latency_ms_p99,
		error_rate, active_features, materialization_freshness_pct, cache_hit_rate
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	ON CONFLICT (region) DO UPDATE SET
		is_healthy = EXCLUDED.is_healthy,
		latency_ms_p50 = EXCLUDED.latency_ms_p50,
		latency_ms_p95 = EXCLUDED.latency_ms_p95,
		latency_ms_p99 = EXCLUDED.latency_ms_p99,
		error_rate = EXCLUDED.error_rate,
		active_features = EXCLUDED.active_features,
		materialization_freshness_pct = EXCLUDED.materialization_freshness_pct,
		cache_hit_rate = EXCLUDED.cache_hit_rate,
		last_updated = CURRENT_TIMESTAMP
	`

	_, err := tdb.conn.ExecContext(ctx, query,
		perf.Region,
		perf.IsHealthy,
		perf.LatencyP50MS,
		perf.LatencyP95MS,
		perf.LatencyP99MS,
		perf.ErrorRate,
		perf.ActiveFeatures,
		perf.MaterializationFreshnessPercent,
		perf.CacheHitRate,
	)

	return err
}
