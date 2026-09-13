package activities

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/events"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// TestSyncConnectionToTenant_WorksUnderStrictRLS proves syncConnectionToTenant
// (which previously issued its INSERT/UPDATE/DELETE directly on a.DB with no
// transaction at all) actually writes into the target tenant under the
// strict RLS policy, rather than silently affecting zero rows.
func TestSyncConnectionToTenant_WorksUnderStrictRLS(t *testing.T) {
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

	ctx := context.Background()
	targetTenantID := uuid.New().String()
	goldConnectionID := uuid.New().String()

	if _, err := sqlDB.Exec("INSERT INTO public.tenants (id, name, gold_copy) VALUES ($1, $2, false)", targetTenantID, "activity-test-"+targetTenantID[:8]); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = sqlDB.Exec("DELETE FROM public.connections WHERE tenant_id = $1", targetTenantID)
		_, _ = sqlDB.Exec("DELETE FROM public.tenants WHERE id = $1", targetTenantID)
	})

	logger := zap.NewNop().Sugar()
	a := NewGoldCopyActivities(sqlDB, logger, nil)

	event := events.GoldCopyConnectionEvent{
		EventID:      uuid.New().String(),
		ConnectionID: goldConnectionID,
		Action:       "INSERT",
		ConnectionData: map[string]interface{}{
			"name": "test-conn",
			"type": "postgres",
		},
	}

	if err := a.syncConnectionToTenant(ctx, targetTenantID, event); err != nil {
		t.Fatalf("syncConnectionToTenant: %v", err)
	}

	// Verify scoped to the target tenant's own context, matching how a
	// real request would see it — not via a bare unscoped read, which
	// would (correctly) see nothing under strict RLS regardless of
	// whether the write actually happened.
	db := sqlx.NewDb(sqlDB, "pgx")
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("begin verify tx: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", targetTenantID); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM public.connections WHERE tenant_id = $1 AND core_id = $2", targetTenantID, goldConnectionID).Scan(&count); err != nil {
		t.Fatalf("verify query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 connection propagated to tenant %s, got %d", targetTenantID, count)
	}
}
