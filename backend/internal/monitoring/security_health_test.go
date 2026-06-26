package monitoring_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
	"github.com/hondyman/semlayer/backend/internal/monitoring"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create schemas required for tests
	_, err = db.Exec(`
		-- SQLite mock schema for tests
		CREATE TABLE IF NOT EXISTS security_master (
			id TEXT,
			tenant_id TEXT,
			confidence_score REAL,
			valid_to DATETIME
		);
		CREATE TABLE IF NOT EXISTS data_quality_results (
			security_id TEXT,
			tenant_id TEXT,
			severity TEXT,
			resolved BOOLEAN
		);
		CREATE TABLE IF NOT EXISTS catalog_node_type (
			id TEXT,
			type_name TEXT
		);
		CREATE TABLE IF NOT EXISTS catalog_edge_type (
			id TEXT,
			type_name TEXT
		);
		CREATE TABLE IF NOT EXISTS catalog_node (
			id TEXT,
			tenant_id TEXT,
			type_id TEXT
		);
		CREATE TABLE IF NOT EXISTS catalog_edge (
			id TEXT,
			source_node_id TEXT,
			edge_type_id TEXT
		);
	`)
	require.NoError(t, err)

	return db
}

func TestSecurityHealthMonitor_CalculateTenantHealth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// 1. Setup Data
	tenantID := uuid.New()

	// Securities
	db.Exec(`INSERT INTO security_master (id, tenant_id, confidence_score, valid_to) VALUES (?, ?, 95.0, NULL)`, uuid.New().String(), tenantID.String())
	db.Exec(`INSERT INTO security_master (id, tenant_id, confidence_score, valid_to) VALUES (?, ?, 85.0, NULL)`, uuid.New().String(), tenantID.String())
	db.Exec(`INSERT INTO security_master (id, tenant_id, confidence_score, valid_to) VALUES (?, ?, 90.0, NULL)`, uuid.New().String(), tenantID.String())

	// Note: We won't fully mock the complex graph join for drift here due to SQLite syntax vs Postgres differences
	// But we can test the primary logic handling 0-counts gracefully or basic penalites

	// Setup monitor (passing nil for goldRepo as it's not strictly used in current calculate logic)
	repo := goldcopy.NewRepository(db.DB, nil)
	monitor := monitoring.NewSecurityHealthMonitor(db.DB, repo)

	// 2. Execute
	// We expect the query for average to work, but the other queries might fail or return 0 on SQLite if table names differ from "edm.*"
	// To make the test robust, let's adjust the query in the implementation or accept the fallback to 0.

	// Actually, the implementation uses "edm.security_master" and "catalog_node".
	// Since we are using an in-memory SQLite, "edm." prefix will cause a "no such table" error unless we attach a database or create a view.
	// For simplicity in this test, we verify instantiation and structure.

	// Create view to alias edm schema
	db.Exec(`CREATE VIEW "edm.security_master" AS SELECT * FROM security_master`)
	db.Exec(`CREATE VIEW "edm.data_quality_results" AS SELECT * FROM data_quality_results`)

	status, err := monitor.CalculateTenantHealth(context.Background(), tenantID)

	// 3. Assertions
	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, tenantID, status.TenantID)
	assert.Equal(t, 3, status.TotalSecurities)
	assert.Equal(t, 90.0, status.AverageConfidence) // (95+85+90)/3 = 90

	// Since penalites are 0 (no DQ inserts), score should be base * confidence
	// Score = 100 * (90/100) = 90
	assert.Equal(t, 90.0, status.HealthScore)
	assert.Equal(t, "Healthy", status.Status)
	assert.Empty(t, status.RecommendedActions) // No drift or DQ
}
