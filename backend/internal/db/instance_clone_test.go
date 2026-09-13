package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TestCloneGoldCopyInstance_WorksUnderStrictRLS proves the actual wired
// function, not just the underlying role mechanism, against real
// tenant_instance/tenant_product/connections tables with the strict RLS
// policy live. Before wiring, this would either return zero cloned rows
// (RLS silently filtering the writes) or a permission/RLS error,
// depending on the connecting role.
func TestCloneGoldCopyInstance_WorksUnderStrictRLS(t *testing.T) {
	dsn := os.Getenv("UISCE_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("UISCE_TEST_DB_DSN not set, skipping")
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	var haveRole bool
	if err := sqlDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = 'uisce_gold_copy_sync')").Scan(&haveRole); err != nil {
		t.Fatalf("checking for uisce_gold_copy_sync role: %v", err)
	}
	if !haveRole {
		t.Skip("uisce_gold_copy_sync role does not exist on test database; skipping")
	}

	db := sqlx.NewDb(sqlDB, "pgx")
	ctx := context.Background()

	goldTenantID := uuid.New()
	goldInstanceID := uuid.New()
	goldProductAlphaID := uuid.New()
	targetTenantID := uuid.New()
	targetInstanceID := uuid.New()

	// Seed a gold-copy tenant with one product. public.tenants has no RLS,
	// so that insert runs directly; tenant_instance/tenant_product are
	// FORCE RLS and this seeds rows across two different tenants in one
	// go, so it needs the same elevated role the function under test
	// uses — this is a fixture-setup detail, not something the test is
	// meant to exercise.
	mustExec(t, sqlDB, "INSERT INTO public.tenants (id, name, gold_copy) VALUES ($1, $2, true)", goldTenantID, "gold-copy-test-"+goldTenantID.String()[:8])
	mustExec(t, sqlDB, "INSERT INTO public.tenants (id, name, gold_copy) VALUES ($1, $2, false)", targetTenantID, "target-test-"+targetTenantID.String()[:8])
	if err := WithGoldCopySync(ctx, sqlDB, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO public.tenant_instance (id, tenant_id) VALUES ($1, $2)", goldInstanceID, goldTenantID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO public.tenant_instance (id, tenant_id) VALUES ($1, $2)", targetInstanceID, targetTenantID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO public.tenant_product (id, datasource_id, alpha_product_id, tenant_id, version, is_active)
			VALUES ($1, $2, $3, $4, 1, true)
		`, uuid.New(), goldInstanceID, goldProductAlphaID, goldTenantID)
		return err
	}); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	t.Cleanup(func() {
		_, _ = sqlDB.Exec("DELETE FROM public.tenant_product WHERE tenant_id IN ($1, $2)", goldTenantID, targetTenantID)
		_, _ = sqlDB.Exec("DELETE FROM public.tenant_instance WHERE tenant_id IN ($1, $2)", goldTenantID, targetTenantID)
		_, _ = sqlDB.Exec("DELETE FROM public.tenants WHERE id IN ($1, $2)", goldTenantID, targetTenantID)
	})

	result, err := CloneGoldCopyInstance(ctx, db, targetTenantID, targetInstanceID)
	if err != nil {
		t.Fatalf("CloneGoldCopyInstance: %v", err)
	}
	if result.ProductsCloned != 1 {
		t.Fatalf("expected 1 product cloned, got %d — this is the exact symptom of the role assumption silently failing", result.ProductsCloned)
	}

	// Confirm the clone actually landed under the TARGET tenant, reading
	// it back the way a real request would: scoped to that tenant's own
	// context, not via the elevated role and not via a bare unscoped
	// query (which — correctly — sees nothing at all under strict RLS;
	// verified that directly too, since it's the same fail-closed
	// property this whole fix depends on).
	var bareReadCount int
	if err := sqlDB.QueryRowContext(ctx, "SELECT count(*) FROM public.tenant_product WHERE id = $1", result.ClonedProductIDs[0]).Scan(&bareReadCount); err != nil {
		t.Fatalf("bare read: %v", err)
	}
	if bareReadCount != 0 {
		t.Fatalf("expected a bare, unscoped read to see 0 rows under strict RLS, saw %d", bareReadCount)
	}

	var clonedTenantID uuid.UUID
	scopedTx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin scoped tx: %v", err)
	}
	defer scopedTx.Rollback()
	if _, err := scopedTx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", targetTenantID.String()); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}
	if err := scopedTx.QueryRowContext(ctx,
		"SELECT tenant_id FROM public.tenant_product WHERE id = $1", result.ClonedProductIDs[0],
	).Scan(&clonedTenantID); err != nil {
		t.Fatalf("querying cloned product scoped to target tenant: %v", err)
	}
	if clonedTenantID != targetTenantID {
		t.Fatalf("cloned product landed under tenant %s, expected target tenant %s", clonedTenantID, targetTenantID)
	}
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...interface{}) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("seed query failed (%s): %v", query, err)
	}
}
