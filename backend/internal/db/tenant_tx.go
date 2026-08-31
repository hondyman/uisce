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

func WithTenantTransaction(
	ctx context.Context,
	db *sql.DB,
	tenantID string,
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

	if _, err := tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("WithTenantTransaction: SET LOCAL uisce.current_tenant failed: %w", err)
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
