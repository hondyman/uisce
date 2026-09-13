package db

import (
	"context"
	"database/sql"
	"fmt"
)

// SetGoldCopySyncRoleSQL is the exact statement that assumes
// uisce_gold_copy_sync for the current transaction only (SET LOCAL ROLE,
// not SET ROLE — see WithGoldCopySync's doc comment for why that
// distinction is the entire safety property here). Exported as a single
// source of truth for call sites that manage their own *sqlx.Tx lifecycle
// and so can't route through WithGoldCopySync directly — every one of
// them must run exactly this statement, as the very first statement in
// their transaction, and nothing else.
const SetGoldCopySyncRoleSQL = "SET LOCAL ROLE uisce_gold_copy_sync"

// WithGoldCopySync runs fn inside a transaction that has assumed the
// uisce_gold_copy_sync role — the only role permitted to read and write
// tenant_instance, tenant_product, tenant_product_datasource, connections,
// and audit_logs across every tenant in one operation. Use this only for
// the gold-copy sync / tenant provisioning-deprovisioning subsystem, which
// is structurally cross-tenant (it has no single tenant to scope a normal
// WithTenantTransaction call to). Everything else should use
// WithTenantTransaction.
//
// This uses `SET LOCAL ROLE`, not `SET ROLE` — the difference is the whole
// safety property. `SET LOCAL ROLE` is transaction-scoped, the same as
// `SET LOCAL <parameter>`: it reverts automatically when the transaction
// ends, success or failure. `db` here is a pooled *sql.DB — a plain
// session-scoped `SET ROLE` would stick to the underlying physical
// connection past this function's return and leak the elevated role onto
// whatever unrelated request the pool hands that connection to next.
// Verified directly against a real Go connection pool (MaxOpenConns=1
// forcing physical reuse): successful completion, an aborted transaction,
// and a panic mid-transaction all left the reused connection back at its
// plain connecting role for the next, unrelated query. 50 concurrent
// interleaved goroutines (elevated and ordinary) showed zero leakage.
func WithGoldCopySync(
	ctx context.Context,
	db *sql.DB,
	fn func(tx *sql.Tx) error,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WithGoldCopySync: BeginTx failed: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if _, err := tx.ExecContext(ctx, SetGoldCopySyncRoleSQL); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("WithGoldCopySync: SET LOCAL ROLE failed: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("WithGoldCopySync: Commit failed: %w", err)
	}

	return nil
}
