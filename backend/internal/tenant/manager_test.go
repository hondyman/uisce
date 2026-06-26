package tenant

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hondyman/semlayer/backend/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantManager_Integration(t *testing.T) {
	// Skip integration tests if short mode is enabled
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load configuration
	// Assuming the test is run from backend/internal/tenant, we need to go up to find config.yaml
	// Or we can just use the environment variable if set, or a default local DSN
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Try to load from config file relative to this test
		// We are in backend/internal/tenant, so config.yaml is in backend/
		cfg, err := config.LoadConfig("../../config.yaml")
		if err == nil {
			dsn = cfg.DSN
		} else {
			// Fallback to default local DSN
			dsn = "postgres://postgres:postgres@localhost:5432/semlayer?sslmode=disable"
		}
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
	}

	tm := NewTenantManager(db, nil)
	ctx := context.Background()

	// Skip if the environment doesn't have the 'vector' extension available.
	var extName string
	if err := db.QueryRowContext(ctx, "SELECT name FROM pg_available_extensions WHERE name = 'vector' LIMIT 1").Scan(&extName); err != nil || extName != "vector" {
		t.Skip("Skipping integration test: 'vector' extension not available in this DB")
	}

	// Create a unique tenant code
	tenantCode := fmt.Sprintf("test_tenant_%d", time.Now().Unix())
	tenantName := "Test Tenant"

	// 1. Test CreateTenant
	tenant, err := tm.CreateTenant(ctx, tenantCode, tenantName)
	require.NoError(t, err)
	assert.NotEmpty(t, tenant.TenantID)
	assert.Equal(t, tenantCode, tenant.TenantCode)
	assert.Equal(t, fmt.Sprintf("tenant_%s", tenantCode), tenant.SchemaName)

	// Verify schema exists
	var schemaExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)", tenant.SchemaName).Scan(&schemaExists)
	require.NoError(t, err)
	assert.True(t, schemaExists, "Schema should exist")

	// Verify tables exist in the schema
	tables := []string{"documents", "document_chunks", "query_logs"}
	for _, table := range tables {
		var tableExists bool
		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)", tenant.SchemaName, table).Scan(&tableExists)
		require.NoError(t, err)
		assert.True(t, tableExists, fmt.Sprintf("Table %s should exist in schema %s", table, tenant.SchemaName))
	}

	// 2. Test GetTenantConnection (Isolation)
	conn, err := tm.GetTenantConnection(ctx, tenant.TenantID)
	require.NoError(t, err)
	defer conn.Close()

	// Verify search_path
	var searchPath string
	err = conn.QueryRowContext(ctx, "SHOW search_path").Scan(&searchPath)
	require.NoError(t, err)
	assert.Equal(t, tenant.SchemaName, searchPath)

	// Clean up (optional, but good for local dev)
	// _, _ = db.ExecContext(ctx, fmt.Sprintf("DROP SCHEMA %s CASCADE", tenant.SchemaName))
	// _, _ = db.ExecContext(ctx, "DELETE FROM tenants WHERE tenant_id = $1", tenant.TenantID)
}
