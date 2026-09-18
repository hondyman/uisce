package db

import (
	"context"
	"database/sql"
	"fmt"
)

type ctxKey string

const tenantGUCKey ctxKey = "uisce.current_tenant"

type TenantCtx struct {
	TenantID string
}

// WithTenantTransaction opens a tx and SET LOCALs uisce.current_tenant
// (is_local=true). Prefer WithTenantGoldTransaction when policies also read
// uisce.gold_tenant (post gold-copy-widen pages/BO policies).
func WithTenantTransaction(
	ctx context.Context,
	db *sql.DB,
	tenantID string,
	fn func(tx *sql.Tx) error,
) error {
	return WithTenantGoldTransaction(ctx, db, tenantID, "", fn)
}

// WithTenantGoldTransaction is the RLS choke point after gold-copy widen:
// SET LOCAL uisce.current_tenant and, when goldTenantID is non-empty,
// uisce.gold_tenant. Both are transaction-scoped (pooling-safe).
//
// Pages policy shape (is_core is a page_definitions column, not a caller flag):
//
//	tenant_id = uisce_get_current_tenant()
//	OR (is_core AND tenant_id = uisce_get_gold_tenant())
//
// BO policy shape: tenant_id = current OR tenant_id = gold.
func WithTenantGoldTransaction(
	ctx context.Context,
	db *sql.DB,
	tenantID string,
	goldTenantID string,
	fn func(tx *sql.Tx) error,
) error {
	if tenantID == "" {
		return fmt.Errorf("WithTenantTransaction: tenantID cannot be empty")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WithTenantTransaction: BeginTx failed: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := ApplyTenantGUCs(ctx, tx, tenantID, goldTenantID); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("WithTenantTransaction: Commit failed: %w", err)
	}

	return nil
}

// ApplyTenantGUCs SET LOCALs tenant (and optional gold) GUCs on an open tx.
// Use when Begin already happened (e.g. sqlx.BeginTxx). is_local=true always.
func ApplyTenantGUCs(ctx context.Context, tx *sql.Tx, tenantID, goldTenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("ApplyTenantGUCs: tenantID cannot be empty")
	}
	if _, err := tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantID); err != nil {
		return fmt.Errorf("ApplyTenantGUCs: SET LOCAL uisce.current_tenant failed: %w", err)
	}
	// Legacy policies (swift / older migrations) still read app.tenant_id.
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("ApplyTenantGUCs: SET LOCAL app.tenant_id failed: %w", err)
	}
	if goldTenantID != "" {
		if _, err := tx.ExecContext(ctx, "SELECT set_config('uisce.gold_tenant', $1, true)", goldTenantID); err != nil {
			return fmt.Errorf("ApplyTenantGUCs: SET LOCAL uisce.gold_tenant failed: %w", err)
		}
	}
	return nil
}

func RequireTenantID(ctx context.Context, tenantID *string) error {
	if tenantID == nil || *tenantID == "" {
		return fmt.Errorf("security boundary violation: request has no verified tenant context")
	}
	return nil
}

type contextKey string

const (
	tenantContextKey contextKey = "tenant_context"
)

func WithTenantContextToCtx(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKey, &TenantCtx{TenantID: tenantID})
}

func GetTenantIDFromCtx(ctx context.Context) (string, error) {
	v := ctx.Value(tenantContextKey)
	if v == nil {
		return "", fmt.Errorf("security boundary violation: request has no verified tenant context")
	}
	tc, ok := v.(*TenantCtx)
	if !ok || tc.TenantID == "" {
		return "", fmt.Errorf("security boundary violation: request has no verified tenant context")
	}
	return tc.TenantID, nil
}

func RequireVerifiedTenantFromCtx(ctx context.Context) error {
	_, err := GetTenantIDFromCtx(ctx)
	return err
}
