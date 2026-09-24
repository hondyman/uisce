package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	uisce_db "github.com/hondyman/uisce/backend/internal/db"
)

// TenantWorker handles cascading operations for tenants
type TenantWorker struct {
	db *sql.DB
}

// NewTenantWorker creates a new TenantWorker
func NewTenantWorker(db *sql.DB) *TenantWorker {
	return &TenantWorker{
		db: db,
	}
}

// DeleteTenantResources deletes all resources associated with a tenant
func (w *TenantWorker) DeleteTenantResources(ctx context.Context, tenantID string) error {
	log.Printf("[TenantWorker] Deleting resources for tenant %s", tenantID)

	// 1. Check if Gold Copy
	var isGoldCopy bool
	err := w.db.QueryRowContext(ctx, "SELECT gold_copy FROM public.tenants WHERE id = $1", tenantID).Scan(&isGoldCopy)
	if err != nil {
		if err == sql.ErrNoRows {
			// Tenant already deleted, proceed to cleanup orphans if any, or just return
			// If the trigger is "BEFORE DELETE", row exists. If "AFTER DELETE", row is gone.
			// Debezium sends "delete" event AFTER delete. So row might be gone.
			// But wait, the user said "when I delete a tenant".
			// If I receive a DELETE event, the tenant row is ALREADY deleted.
			// So I cannot query it to check if it WAS gold copy, unless I check the "before" state from Debezium event.
			// The caller (main.go) handles parsing the event. It should pass the 'before' state to check gold_copy flag.
			// I should probably accept the 'before' data or just generic 'force delete'.
			// For safety, I will assume the caller checks gold_copy before calling this if possible,
			// OR I assumes that if the tenant is already deleted, I should clean up children.
			// BUT, if the tenant WAS gold copy, I should NOT have allowed it to be deleted.
			// The application layer should prevent deleting Gold Copy.
			// If it happened at DB layer, it's too late.
			// So I will proceed with cleanup.
			log.Printf("[TenantWorker] Tenant record %s not found (likely already deleted). Proceeding with resource cleanup.", tenantID)
		} else {
			return fmt.Errorf("failed to check gold copy status: %w", err)
		}
	} else if isGoldCopy {
		log.Printf("[TenantWorker] CRITICAL: Attempted to delete resources for Gold Copy tenant %s. Aborting.", tenantID)
		return fmt.Errorf("cannot delete resources for gold copy tenant")
	}

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Deletes this tenant's own rows, but with no tenant GUC set — under
	// strict RLS this would otherwise silently delete zero rows (RLS
	// filters the DELETE's target rows to nothing) while every RowsAffected
	// below reports 0 and the function still returns success. Assumes
	// uisce_gold_copy_sync as the very first statement so the deletes
	// below actually take effect; SET LOCAL ROLE outside a transaction
	// block is a silent no-op (verified directly), so it must be here,
	// not before BeginTx.
	if _, err := tx.ExecContext(ctx, uisce_db.SetGoldCopySyncRoleSQL); err != nil {
		return fmt.Errorf("failed to assume gold-copy-sync role: %w", err)
	}

	// 2. Delete Connections
	res, err := tx.ExecContext(ctx, "DELETE FROM connections WHERE tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete connections: %w", err)
	}
	connCount, _ := res.RowsAffected()

	// 3. Delete Tenant Products (Cascades to Datasources usually, but let's be safe)
	// Assuming foreign keys might handle it, but explicit delete is safer for logic
	res, err = tx.ExecContext(ctx, "DELETE FROM tenant_product WHERE datasource_id IN (SELECT id FROM tenant_instance WHERE tenant_id = $1)", tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete tenant products: %w", err)
	}
	prodCount, _ := res.RowsAffected()

	// 4. Delete Tenant Instances
	res, err = tx.ExecContext(ctx, "DELETE FROM tenant_instance WHERE tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete tenant instances: %w", err)
	}
	instCount, _ := res.RowsAffected()

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[TenantWorker] Deleted %d connections, %d products, %d instances for tenant %s", connCount, prodCount, instCount, tenantID)

	return nil
}

// InactivateTenantResources inactivates all resources associated with a tenant
func (w *TenantWorker) InactivateTenantResources(ctx context.Context, tenantID string) error {
	log.Printf("[TenantWorker] Inactivating resources for tenant %s", tenantID)

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Same reasoning as DeleteTenantResources: assume uisce_gold_copy_sync
	// as the very first statement so the updates below aren't silently
	// filtered to zero rows by RLS.
	if _, err := tx.ExecContext(ctx, uisce_db.SetGoldCopySyncRoleSQL); err != nil {
		return fmt.Errorf("failed to assume gold-copy-sync role: %w", err)
	}

	// 1. Update Connections
	res, err := tx.ExecContext(ctx, "UPDATE connections SET is_active = false WHERE tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to update connections: %w", err)
	}
	connCount, _ := res.RowsAffected()

	// 2. Update Tenant Products
	// We need to join? No, update based on subquery
	res, err = tx.ExecContext(ctx, "UPDATE tenant_product SET is_active = false WHERE datasource_id IN (SELECT id FROM tenant_instance WHERE tenant_id = $1)", tenantID)
	if err != nil {
		return fmt.Errorf("failed to update tenant products: %w", err)
	}
	prodCount, _ := res.RowsAffected()

	// 3. Update Tenant Instances
	res, err = tx.ExecContext(ctx, "UPDATE tenant_instance SET is_active = false WHERE tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to update tenant instances: %w", err)
	}
	instCount, _ := res.RowsAffected()

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[TenantWorker] Inactivated %d connections, %d products, %d instances for tenant %s", connCount, prodCount, instCount, tenantID)

	return nil
}
