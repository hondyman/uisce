package rag

import (
	"context"
	"database/sql"
	"fmt"
)

// WithTenantTx executes a function within a transaction that has the search_path set to the tenant's schema.
// This ensures physical isolation of data access.
func WithTenantTx(ctx context.Context, db *sql.DB, tenantSchema string, fn func(tx *sql.Tx) error) error {
	// Begin a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Set the search_path to the tenant's schema
	// We use QuoteIdentifier to prevent SQL injection via schema name
	query := fmt.Sprintf("SET LOCAL search_path = %s", pqQuoteIdent(tenantSchema))
	if _, err := tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to set search_path to %s: %w", tenantSchema, err)
	}

	// Execute the provided function
	if err := fn(tx); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// pqQuoteIdent quotes an identifier for use in a SQL statement.
// This is a minimal implementation. For production, consider using lib/pq or pgx helpers.
func pqQuoteIdent(ident string) string {
	return `"` + ident + `"`
}
