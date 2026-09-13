package api

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"

	uisce_db "github.com/hondyman/uisce/backend/internal/db"
)

// TestSyncConnectionsTx_WorksUnderStrictRLS proves syncConnectionsTx (the
// extracted body of SyncConnectionsFromGoldCopy) actually reads gold-copy's
// connections and writes them into the target tenant under strict RLS,
// rather than every step silently returning zero rows.
func TestSyncConnectionsTx_WorksUnderStrictRLS(t *testing.T) {
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
	h := &ConnectionSyncHandler{DB: db}

	goldTenantID := uuid.New()
	goldInstanceID := uuid.New()
	targetTenantID := uuid.New()
	targetInstanceID := uuid.New()
	alphaProductID := uuid.New()
	alphaDatasourceID := uuid.New()
	goldConnID := uuid.New()
	goldTenantProductID := uuid.New()
	targetTenantProductID := uuid.New()

	if _, err := db.Exec("INSERT INTO public.tenants (id, name, gold_copy) VALUES ($1, $2, true)", goldTenantID, "gold-handler-test-"+goldTenantID.String()[:8]); err != nil {
		t.Fatalf("seed gold tenant: %v", err)
	}
	if _, err := db.Exec("INSERT INTO public.tenants (id, name, gold_copy) VALUES ($1, $2, false)", targetTenantID, "target-handler-test-"+targetTenantID.String()[:8]); err != nil {
		t.Fatalf("seed target tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM public.tenant_product_datasource WHERE tenant_product_id IN ($1, $2)", goldTenantProductID, targetTenantProductID)
		_, _ = db.Exec("DELETE FROM public.connections WHERE tenant_id IN ($1, $2)", goldTenantID, targetTenantID)
		_, _ = db.Exec("DELETE FROM public.tenant_product WHERE id IN ($1, $2)", goldTenantProductID, targetTenantProductID)
		_, _ = db.Exec("DELETE FROM public.tenant_instance WHERE tenant_id IN ($1, $2)", goldTenantID, targetTenantID)
		_, _ = db.Exec("DELETE FROM public.tenants WHERE id IN ($1, $2)", goldTenantID, targetTenantID)
	})

	// Seed gold-copy's own product/datasource/connection and the target
	// tenant's instance+product-registration, all cross-tenant, so through
	// the elevated role.
	if err := uisce_db.WithGoldCopySync(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO public.tenant_instance (id, tenant_id) VALUES ($1, $2)", goldInstanceID, goldTenantID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO public.tenant_instance (id, tenant_id) VALUES ($1, $2)", targetInstanceID, targetTenantID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO public.tenant_product (id, datasource_id, alpha_product_id, tenant_id, version, is_active)
			VALUES ($1, $2, $3, $4, 1, true)
		`, goldTenantProductID, goldInstanceID, alphaProductID, goldTenantID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO public.tenant_product (id, datasource_id, alpha_product_id, tenant_id, version, is_active)
			VALUES ($1, $2, $3, $4, 1, true)
		`, targetTenantProductID, targetInstanceID, alphaProductID, targetTenantID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO public.connections (id, tenant_id, name, type, is_active, core_id)
			VALUES ($1, $2, 'gold-conn', 'postgres', true, NULL)
		`, goldConnID, goldTenantID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO public.tenant_product_datasource (id, tenant_product_id, alpha_datasource_id, connection_id, source_name, is_active, config)
			VALUES ($1, $2, $3, $4, 'gold-ds', true, '{}')
		`, uuid.New(), goldTenantProductID, alphaDatasourceID, goldConnID)
		return err
	}); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	resp, err := h.syncConnectionsTx(ctx, targetTenantID.String(), goldTenantID.String())
	if err != nil {
		t.Fatalf("syncConnectionsTx: %v", err)
	}
	if resp.ConnectionsAdded != 1 {
		t.Fatalf("expected 1 connection added, got %d (message: %s)", resp.ConnectionsAdded, resp.Message)
	}

	// Verify scoped to the target tenant's own context — not a bare
	// unscoped read, which would see nothing under strict RLS regardless.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin verify tx: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", targetTenantID.String()); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM public.connections WHERE tenant_id = $1 AND core_id = $2", targetTenantID, goldConnID).Scan(&count); err != nil {
		t.Fatalf("verify query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 connection cloned into target tenant, got %d", count)
	}
}
