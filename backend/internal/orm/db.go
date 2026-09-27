package orm

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// OpenAlpha opens a connection to the alpha database, where the orm schema
// lives. Reuses DATABASE_URL / POSTGRES_DSN, matching internal/trading.
func OpenAlpha(ctx context.Context) (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_DSN")
	}
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL and POSTGRES_DSN are unset")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open alpha: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping alpha: %w", err)
	}
	return db, nil
}

// WithTenant runs fn inside a transaction whose connection has
// app.current_tenant set for the duration. Every write must go through
// this helper, otherwise RLS on the new tables will reject the INSERT.
func WithTenant(ctx context.Context, db *sql.DB, tenantID string, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		tx.Rollback()
		return fmt.Errorf("set current_tenant: %w", err)
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
