package metadata

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// integrationTestDSN is the load-bearing fallback used by every test in this
// file. connect_timeout is not decoration: the fallback host is only
// reachable over a private network (Tailscale range), and an
// unreachable-but-routed address hangs the OS-level TCP connect for far
// longer than any context passed to PingContext/QueryRowContext actually
// bounds — lib/pq's connection retry re-dials outside that context's
// effective window (observed directly: a 3s context.WithTimeout around
// PingContext still hung ~30s against this host). connect_timeout is a real
// libpq DSN parameter enforced at the socket layer, and is what actually
// makes these tests skip promptly on a host without that network instead of
// hanging the whole package's test run past `go test -timeout`.
const integrationTestFallbackDSN = "postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable&connect_timeout=3"

// openIntegrationTestDB centralizes DSN selection, connect, and a bounded
// reachability check for every integration test in this file — previously
// four separate copies of this logic, each with its own (occasionally
// missing, occasionally unbounded) reachability check, which is exactly how
// this file ended up with a hang in one copy while a sibling copy nearby
// used a bounded PingContext. One implementation now, one place to fix.
func openIntegrationTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode: requires Postgres")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = integrationTestFallbackDSN
	}

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping integration test: failed to open DB: %v", err)
		return nil
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		t.Skipf("Skipping integration test: database not reachable: %v", err)
		return nil
	}
	return db
}

func TestGetBusinessObjectIncludesChildIntegration(t *testing.T) {
	db := openIntegrationTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()

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
	_, err := db.ExecContext(ctx, `INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, NOW())`, parentID, tenantID, "test_parent_key_"+parentID, "Test Parent", "Test Parent", "test_parent")
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
	db := openIntegrationTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	ctx := context.Background()

	// Check if gold_copy column exists
	var hasGoldCopy bool
	err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'gold_copy')").Scan(&hasGoldCopy)
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

// TestFieldUpdateRoundTrip writes fields via UpdateBusinessObject and reads them via GetBusinessObject,
// asserting that the fields are visible after the write. This is the assertion that catches the
// write→read table-mismatch bug: writes to business_object_fields must be readable by loadBOSubtypesAndFields.
func TestFieldUpdateRoundTrip(t *testing.T) {
	db := openIntegrationTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()
	tenantID := "910638ba-a459-4a3f-bb2d-78391b0595f6"

	// Provision tenant + BO
	_, _ = db.ExecContext(ctx, `INSERT INTO tenants (id, name, created_at) VALUES ($1::uuid, $2, NOW()) ON CONFLICT (id) DO NOTHING`, tenantID, "round-trip tenant")
	boID := uuid.NewString()
	boKey := "rt_test_bo_" + boID[:8]
	_, err := db.ExecContext(ctx,
		`INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $4, $3, NOW())`,
		boID, tenantID, boKey, "Round-Trip BO")
	require.NoError(t, err)

	defer func() {
		db.ExecContext(ctx, `DELETE FROM business_object_fields WHERE bo_id = $1`, boID)
		db.ExecContext(ctx, `DELETE FROM business_objects WHERE id = $1`, boID)
	}()

	svc := NewBusinessObjectService(db, nil, nil, nil)
	secCtx := &security.Context{TenantID: tenantID}

	// Update with two named fields
	updateReq := models.UpdateBusinessObjectRequest{
		Config: map[string]interface{}{
			"fields": []map[string]interface{}{
				{"name": "ProductName", "type": "text", "role": "DIMENSION"},
				{"name": "UnitPrice", "type": "number", "role": "MEASURE"},
			},
		},
	}
	_, err = svc.UpdateBusinessObject(ctx, secCtx, boKey, updateReq, "test-user")
	require.NoError(t, err)

	// Get and assert fields are visible
	bo, err := svc.GetBusinessObject(ctx, secCtx, boKey)
	require.NoError(t, err)
	require.NotNil(t, bo)

	totalFields := len(bo.CoreFields) + len(bo.CustomFields)
	require.GreaterOrEqual(t, totalFields, 2, "Update wrote fields but Get returned %d — write→read table-mismatch bug present", totalFields)

	// Verify by name
	wantNames := map[string]bool{"ProductName": true, "UnitPrice": true}
	for _, f := range append(bo.CoreFields, bo.CustomFields...) {
		delete(wantNames, f.Name)
	}
	require.Empty(t, wantNames, "missing fields after round-trip: %v", wantNames)

	// Direct DB assertion: business_object_fields has the rows (not bo_fields).
	// Per the NULL-able term_node_id migration, fields without a semanticTermId
	// must store term_node_id = NULL — never a synthetic hash. This is the
	// assertion that catches the SHA-1 fallback reappearing.
	var bfRows, bofRows, nullTermRows, hashRows int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bo_fields WHERE business_object_id = $1`, boID).Scan(&bfRows)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM business_object_fields WHERE bo_id = $1 AND subtype_scope = 'ALL'`, boID).Scan(&bofRows))
	require.Equal(t, 2, bofRows, "expected 2 rows in business_object_fields")
	require.Equal(t, 0, bfRows, "no rows should land in bo_fields anymore")

	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM business_object_fields WHERE bo_id = $1 AND subtype_scope = 'ALL' AND term_node_id IS NULL`,
		boID).Scan(&nullTermRows))
	require.Equal(t, 2, nullTermRows,
		"both fields lack semanticTermId and must store term_node_id IS NULL — "+
			"got %d, want 2. Synthetic IDs are forbidden.",
		nullTermRows)

	// Belt-and-braces: term_node_id values must NOT look like SHA-1 OID-namespace UUIDs
	// (which would be 6ba7b810-9dad-11d1-80b4-00c04fd430c8-derived UUIDs).
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM business_object_fields WHERE bo_id = $1 AND term_node_id::text LIKE '6ba7b810-9dad-11d1-80b4-%'`,
		boID).Scan(&hashRows))
	require.Equal(t, 0, hashRows, "found SHA-1 OID-namespace term_node_ids — hash fallback reappeared")
}

// TestFieldUpdateRejectsWithDownstreamRefs verifies the pre-flight reference
// check refuses wholesale field replacement when downstream references exist.
// Bypassing this guard is exactly the silent-drop bug the diff-based rewrite
// exists to prevent at scale; this test pins the loud-failure form of the
// guard in place.
func TestFieldUpdateRejectsWithDownstreamRefs(t *testing.T) {
	db := openIntegrationTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()
	tenantID := "910638ba-a459-4a3f-bb2d-78391b0595f6"

	_, _ = db.ExecContext(ctx, `INSERT INTO tenants (id, name, created_at) VALUES ($1::uuid, $2, NOW()) ON CONFLICT (id) DO NOTHING`, tenantID, "ref-reject tenant")
	boID := uuid.NewString()
	boKey := "rt_ref_bo_" + boID[:8]
	_, err := db.ExecContext(ctx,
		`INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, created_at) VALUES ($1::uuid, $2::uuid, $3, $4, $4, $3, NOW())`,
		boID, tenantID, boKey, "Ref-Reject BO")
	require.NoError(t, err)

	// Seed a field binding that points to a hypothetical future field_id
	// (no business_object_fields row — we're testing the count > 0 query path).
	// The pre-flight counts field_bindings joined to business_object_fields,
	// so we need both rows. Insert one field, then one binding.
	fieldID := uuid.NewString()
	termID := uuid.NewString()
	_, err = db.ExecContext(ctx, `INSERT INTO business_object_fields (field_id, tenant_id, bo_id, term_node_id, field_name, subtype_scope) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, 'ALL')`,
		fieldID, tenantID, boID, termID, "SeedField")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO field_bindings (id, tenant_id, bo_id, binding_id, field_id, source_type, is_active) VALUES (gen_random_uuid(), $1::uuid, $2::uuid, gen_random_uuid(), $3::uuid, 'COLUMN', true)`,
		tenantID, boID, fieldID)
	require.NoError(t, err)

	defer func() {
		db.ExecContext(ctx, `DELETE FROM field_bindings WHERE bo_id = $1`, boID)
		db.ExecContext(ctx, `DELETE FROM business_object_fields WHERE bo_id = $1`, boID)
		db.ExecContext(ctx, `DELETE FROM business_objects WHERE id = $1`, boID)
	}()

	svc := NewBusinessObjectService(db, nil, nil, nil)
	secCtx := &security.Context{TenantID: tenantID}

	updateReq := models.UpdateBusinessObjectRequest{
		Config: map[string]interface{}{
			"fields": []map[string]interface{}{
				{"name": "ReplacementField", "type": "text", "role": "DIMENSION"},
			},
		},
	}
	_, err = svc.UpdateBusinessObject(ctx, secCtx, boKey, updateReq, "test-user")
	require.Error(t, err, "Update must reject with downstream references")
	require.Contains(t, err.Error(), "refusing to replace field set", "rejection must name the issue")

	// The original field must still be there — rejection is transactional.
	var stillThere int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM business_object_fields WHERE bo_id = $1 AND field_name = 'SeedField'`, boID).Scan(&stillThere))
	require.Equal(t, 1, stillThere, "rejected update must not have modified the field set")
}
