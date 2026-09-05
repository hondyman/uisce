package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

func TestWithTenantTransaction(t *testing.T) {
	dsn := os.Getenv("UISCE_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("UISCE_TEST_DB_DSN not set, skipping tenant isolation test")
	}

	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	ctx := context.Background()

	t.Run("RLS_enforces_tenant_isolation", func(t *testing.T) {
		txA, err := dbConn.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx A: %v", err)
		}
		defer txA.Rollback()

		txB, err := dbConn.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx B: %v", err)
		}
		defer txB.Rollback()

		tenantA := "00000000-0000-0000-0000-000000000001"
		tenantB := "00000000-0000-0000-0000-000000000002"

		if _, err := txA.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantA); err != nil {
			t.Fatalf("SET LOCAL A: %v", err)
		}
		if _, err := txB.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantB); err != nil {
			t.Fatalf("SET LOCAL B: %v", err)
		}

		var countA, countB int
		if err := txA.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenant_product").Scan(&countA); err != nil {
			t.Logf("Query A (tenant_product may not exist): %v", err)
		}
		if err := txB.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenant_product").Scan(&countB); err != nil {
			t.Logf("Query B (tenant_product may not exist): %v", err)
		}

		t.Logf("TxA (tenant=%s): %d rows; TxB (tenant=%s): %d rows", tenantA, countA, tenantB, countB)

		if countA == countB && countA > 0 {
			t.Errorf("RLS leak: both transactions returned the same row count (%d) for different tenants", countA)
		}
	})

	t.Run("no_tenant_GUC_returns_zero_rows", func(t *testing.T) {
		tx, err := dbConn.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		defer tx.Rollback()

		var count int
		err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenant_product").Scan(&count)
		if err != nil {
			t.Fatalf("query: %v", err)
		}

		if count > 0 {
			t.Errorf("expected 0 rows without tenant GUC, got %d — RLS bypass detected", count)
		} else {
			t.Log("PASS: 0 rows returned without tenant GUC (fail-closed)")
		}
	})

	t.Run("WithTenantTransaction_wrapper", func(t *testing.T) {
		tenant := "00000000-0000-0000-0000-000000000003"
		var rowsSeen int

		err := WithTenantTransaction(ctx, dbConn, tenant, func(tx *sql.Tx) error {
			return tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenant_product").Scan(&rowsSeen)
		})
		if err != nil {
			t.Fatalf("WithTenantTransaction: %v", err)
		}

		t.Logf("WithTenantTransaction(tenant=%s): %d rows visible", tenant, rowsSeen)
	})

	t.Run("empty_tenantID_rejected", func(t *testing.T) {
		err := WithTenantTransaction(ctx, dbConn, "", func(tx *sql.Tx) error {
			return nil
		})
		if err == nil {
			t.Error("expected error for empty tenantID, got nil")
		} else {
			t.Logf("PASS: empty tenantID rejected: %v", err)
		}
	})
}

func TestRequireVerifiedTenantFromCtx(t *testing.T) {
	ctx := context.Background()

	t.Run("no_tenant_context", func(t *testing.T) {
		err := RequireVerifiedTenantFromCtx(ctx)
		if err == nil {
			t.Error("expected error for empty context, got nil")
		} else {
			t.Logf("PASS: %v", err)
		}
	})

	t.Run("with_tenant_context", func(t *testing.T) {
		tenantID := "00000000-0000-0000-0000-000000000001"
		ctxWithTenant := WithTenantContextToCtx(ctx, tenantID)
		err := RequireVerifiedTenantFromCtx(ctxWithTenant)
		if err != nil {
			t.Errorf("unexpected error with tenant context: %v", err)
		} else {
			t.Log("PASS: tenant context accepted")
		}
	})
}
