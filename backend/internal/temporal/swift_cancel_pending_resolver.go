// This file is a scaffold — add to swift_reconciliation.go or as a new file.
// The CANCEL_PENDING resolver is the first priority in swift/stub-work.

package temporal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// SWIFTCancelPendingRow is a settlement row parked in CANCEL_PENDING state.
type SWIFTCancelPendingRow struct {
	ID             string
	TenantID       string
	CustodianID    string
	TransactionRef string
	UpdatedAt      time.Time
}

// ResolveCancelPendingActivity queries CANCEL_PENDING rows older than the SLA
// window and resolves each one.
//
// Resolution logic (implement in next stub-work session):
//   - Query custodian channel / swift_session_log for a confirmation or rejection
//     message for each transaction_ref (look for MT192 response or camt.056 reply).
//   - If confirmed → UPDATE settlement_status = 'CANCELLED'
//   - If rejected or SLA exceeded → UPDATE settlement_status = 'FAILED',
//     failure_reason = 'recall_unresolved'
//   - If still ambiguous → leave in CANCEL_PENDING (recon runs again next interval)
//
// SLA: rows older than cancelPendingSLAHours (24h) with no resolution → auto-FAILED.
// Override via swift_tenant_config if needed per custodian.
const cancelPendingSLAHours = 24

// ErrCancelPendingResolverNotImplemented is returned until the resolver is wired.
var ErrCancelPendingResolverNotImplemented = fmt.Errorf(
	"ResolveCancelPendingActivity: not yet implemented. " +
		"Implement in swift/stub-work branch before enabling recall in production. " +
		"Until then, CANCEL_PENDING rows age out with no SLA and no escalation.",
)

// ResolveCancelPendingActivity is registered on bp_queue.
// Returns ErrCancelPendingResolverNotImplemented until wired.
func ResolveCancelPendingActivity(ctx context.Context, tenantID string, db *sql.DB) error {
	log.Printf("[SWIFT] ResolveCancelPendingActivity: STUB — tenantID=%s", tenantID)
	// TODO(swift/stub-work): implement resolver
	// 1. Query CANCEL_PENDING rows for this tenant
	// 2. For each row older than cancelPendingSLAHours: set FAILED(recall_unresolved)
	// 3. For each row with custodian confirmation: set CANCELLED
	return ErrCancelPendingResolverNotImplemented
}
