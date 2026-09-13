package sync

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"

	uisce_db "github.com/hondyman/uisce/backend/internal/db"
)

// TestDeleteTenantResources_ActuallyDeletesUnderStrictRLS proves the wired
// function against real tables with the strict RLS policy live. Before
// wiring, this would silently delete zero rows under RLS (connCount,
// prodCount, instCount all 0) while still logging and returning success —
// the exact silent-no-op class this fix exists to close.
func TestDeleteTenantResources_ActuallyDeletesUnderStrictRLS(t *testing.T) {
	dsn := os.Getenv("UISCE_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("UISCE_TEST_DB_DSN not set, skipping")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	var haveRole bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = 'uisce_gold_copy_sync')").Scan(&haveRole); err != nil {
		t.Fatalf("checking for uisce_gold_copy_sync role: %v", err)
	}
	if !haveRole {
		t.Skip("uisce_gold_copy_sync role does not exist on test database; skipping")
	}

	ctx := context.Background()
	tenantID := uuid.New()
	instanceID := uuid.New()
	connID := uuid.New()

	if _, err := db.Exec("INSERT INTO public.tenants (id, name, gold_copy) VALUES ($1, $2, false)", tenantID, "worker-test-"+tenantID.String()[:8]); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	// tenant_instance/connections are FORCE RLS — seed scoped to this
	// tenant's own context (not the elevated bypass role: this is a
	// single-tenant scenario, so the ordinary tenant-scoped path is the
	// realistic one).
	seedTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin seed tx: %v", err)
	}
	if _, err := seedTx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantID.String()); err != nil {
		t.Fatalf("set tenant context for seeding: %v", err)
	}
	if _, err := seedTx.ExecContext(ctx, "INSERT INTO public.tenant_instance (id, tenant_id) VALUES ($1, $2)", instanceID, tenantID); err != nil {
		t.Fatalf("seed tenant_instance: %v", err)
	}
	if _, err := seedTx.ExecContext(ctx, "INSERT INTO public.connections (id, tenant_id, name, type) VALUES ($1, $2, 'test', 'postgres')", connID, tenantID); err != nil {
		t.Fatalf("seed connections: %v", err)
	}
	if err := seedTx.Commit(); err != nil {
		t.Fatalf("commit seed tx: %v", err)
	}
	t.Cleanup(func() {
		// Bare deletes would no-op under strict RLS (no tenant context) —
		// use the elevated role so cleanup actually works even if the
		// test fails before DeleteTenantResources runs.
		_ = uisce_db.WithGoldCopySync(ctx, db, func(tx *sql.Tx) error {
			_, _ = tx.ExecContext(ctx, "DELETE FROM public.connections WHERE tenant_id = $1", tenantID)
			_, _ = tx.ExecContext(ctx, "DELETE FROM public.tenant_instance WHERE tenant_id = $1", tenantID)
			return nil
		})
		_, _ = db.Exec("DELETE FROM public.tenants WHERE id = $1", tenantID)
	})

	worker := NewTenantWorker(db)
	if err := worker.DeleteTenantResources(ctx, tenantID.String()); err != nil {
		t.Fatalf("DeleteTenantResources: %v", err)
	}

	// A bare, unscoped read would see 0 rows under strict RLS regardless
	// of whether the delete actually ran — making that assertion
	// meaningless (it could never fail, even on a silent no-op). Verify
	// via the same elevated role the delete itself uses, so this
	// genuinely reflects the row's real state, not RLS hiding it either way.
	var remaining int
	err = uisce_db.WithGoldCopySync(ctx, db, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, "SELECT count(*) FROM public.connections WHERE tenant_id = $1", tenantID).Scan(&remaining)
	})
	if err != nil {
		t.Fatalf("checking remaining connections: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected the seeded connection to be deleted, but %d remain — this is the exact silent-no-op symptom this fix closes", remaining)
	}
}
