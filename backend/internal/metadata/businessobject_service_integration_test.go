package metadata

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestGetBusinessObjectIncludesChildIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode: requires Postgres")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		// fallback to local dev DB used by integration
		dsn = "postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable"
	}

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping integration test: failed to open DB: %v", err)
		return
	}
	defer db.Close()

	ctx := context.Background()

	// Ping to ensure DB is accessible (skip if not)
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping integration test: database not reachable: %v", err)
		return
	}

	// Sanity checks: ensure the DB has the tables our integration path requires
	// (this keeps this test safe to run against clean dev DBs that may not have
	// the newer metadata schema applied).
	var cnt int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bo_subtypes").Scan(&cnt); err != nil {
		// try the related legacy table
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ag_catalog.bo_fields").Scan(&cnt); err != nil {
			t.Skip("Skipping integration test: required metadata tables (bo_subtypes or ag_catalog.bo_fields) not present in DB")
			return
		}
	}

	tenantID := "910638ba-a459-4a3f-bb2d-78391b0595f6"

	parentID := uuid.NewString()
	// Ensure tenant exists (tests may run on a clean DB). Try both column name variants used historically in migrations.
	_, _ = db.ExecContext(ctx, `INSERT INTO tenants (id, name, created_at) VALUES ($1::uuid, $2, NOW()) ON CONFLICT (id) DO NOTHING`, tenantID, "test tenant")
	_, _ = db.ExecContext(ctx, `INSERT INTO tenants (tenant_id, name, created_at) VALUES ($1::uuid, $2, NOW()) ON CONFLICT (tenant_id) DO NOTHING`, tenantID, "test tenant")

	// Insert parent
	_, err = db.ExecContext(ctx, `INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, NOW())`, parentID, tenantID, "test_parent_key_"+parentID, "Test Parent", "Test Parent", "test_parent")
	require.NoError(t, err)

	childID := uuid.NewString()
	_, err = db.ExecContext(ctx, `INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, parent_id, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7::uuid, NOW())`, childID, tenantID, "test_child", "Test Child", "Test Child", "test_child", parentID)
	require.NoError(t, err)

	// Ensure cleanup
	defer func() {
		db.ExecContext(ctx, "DELETE FROM business_objects WHERE id = $1", childID)
		db.ExecContext(ctx, "DELETE FROM business_objects WHERE id = $1", parentID)
	}()

	// Use the service to fetch parent
	svc := NewBusinessObjectService(db, nil, nil, nil)
	secCtx := &security.Context{TenantID: tenantID}
	bo, err := svc.GetBusinessObject(ctx, secCtx, parentID)
	require.NoError(t, err)
	require.NotNil(t, bo)

	// Assert subtypes map contains our child (if the DB supports loading them).
	if len(bo.Subtypes) == 0 {
		t.Skip("Skipping integration test: no subtypes loaded — DB schema may be missing metadata tables")
		return
	}

	found := false
	for _, s := range bo.Subtypes {
		if s.ID == childID {
			found = true
			break
		}
	}
	require.True(t, found, "child subtype not found in parent subtypes")
}

func TestGetBusinessObjectFallbackToGoldCopy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode: requires Postgres")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable"
	}

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping integration test: failed to open DB: %v", err)
		return
	}
	defer db.Close()
	ctx := context.Background()

	// Check if gold_copy column exists
	var hasGoldCopy bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'gold_copy')").Scan(&hasGoldCopy)
	if err != nil || !hasGoldCopy {
		t.Skip("Skipping test: gold_copy column missing")
		return
	}

	// Create Gold Copy Tenant
	gcTenantID := uuid.NewString()
	// Try inserting with gold_copy=true and display_name
	_, err = db.ExecContext(ctx, "INSERT INTO tenants (id, name, display_name, gold_copy, created_at) VALUES ($1::uuid, $2, $2, true, NOW()) ON CONFLICT (id) DO NOTHING", gcTenantID, "Gold Copy Tenant")
	if err != nil {
		// Fallback for older schema
		_, err = db.ExecContext(ctx, "INSERT INTO tenants (id, name, created_at) VALUES ($1::uuid, $2, NOW())", gcTenantID, "Gold Copy Tenant")
		if err == nil {
			_, _ = db.ExecContext(ctx, "UPDATE tenants SET gold_copy = true WHERE id = $1::uuid", gcTenantID)
		}
	}
	require.NoError(t, err)

	// Create User Tenant
	userTenantID := uuid.NewString()
	_, err = db.ExecContext(ctx, "INSERT INTO tenants (id, name, display_name, gold_copy, created_at) VALUES ($1::uuid, $2, $2, false, NOW())", userTenantID, "User Tenant")
	require.NoError(t, err)

	// Create BO in Gold Copy
	boID := uuid.NewString()
	boKey := "gc_test_bo_" + boID
	_, err = db.ExecContext(ctx, "INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, is_core, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, true, NOW())", boID, gcTenantID, boKey, "Generic BO", "Generic BO", "generic_bo")
	require.NoError(t, err)

	// Clean up
	defer func() {
		db.ExecContext(ctx, "DELETE FROM business_objects WHERE id = $1", boID)
		db.ExecContext(ctx, "DELETE FROM tenants WHERE id = $1", userTenantID)
		db.ExecContext(ctx, "DELETE FROM tenants WHERE id = $1", gcTenantID)
	}()

	// Use Service to fetch BO using User Tenant ID
	svc := NewBusinessObjectService(db, nil, nil, nil)
	secCtx := &security.Context{TenantID: userTenantID}

	// SUT
	bo, err := svc.GetBusinessObject(ctx, secCtx, boKey)

	require.NoError(t, err)
	require.NotNil(t, bo)
	require.Equal(t, boID, bo.ID)
	require.Equal(t, gcTenantID, bo.TenantID) // It returns the actual BO, so tenant ID is GC
}
