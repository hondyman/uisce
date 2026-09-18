package temporal

import (
	"context"
	"database/sql"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// CancelPendingSLAHours is how long a CANCEL_PENDING row may remain unresolved
// before the resolver auto-fails it with reason 'recall_unresolved'.
// TODO(stub-work): make per-tenant configurable via swift_tenant_config.
const CancelPendingSLAHours = 24

// CancelResponseMsgTypes lists inbound message types that may carry a
// custodian's RESPONSE to a cancellation request.
//
// v1: camt.029 ONLY.
//
// DO NOT add "MT548" without status parsing. MT548 (Settlement Status and
// Confirmation) is sent for EVERY settlement instruction — matched, affirmed,
// pending, rejected, cancelled all arrive as MT548. Matching any MT548
// resolves a CANCEL_PENDING row to terminal CANCELLED even when the custodian
// never confirmed the cancellation and the trade is still live. That is the
// false-CANCELLED bug CANCEL_PENDING exists to prevent, reintroduced through
// the response matcher.
//
// MT548 becomes safe to include only when hasCancelResponse parses the
// message body and requires a cancel-confirmation event (e.g. a CANC status
// code / :24B:CANC// qualifier). TestCancelResponseMsgTypes_V1 fails the
// build if this list grows without that work landing first.
var CancelResponseMsgTypes = []string{"camt.029"}

// ResolveCancelPendingResult reports what one resolver pass did.
type ResolveCancelPendingResult struct {
	Scanned      int `json:"scanned"`
	Resolved     int `json:"resolved"`      // → CANCELLED (custodian confirmed)
	AutoFailed   int `json:"auto_failed"`   // → FAILED (SLA exceeded)
	StillPending int `json:"still_pending"` // within SLA, no response yet
}

// ResolveCancelPendingActivity resolves settlement rows parked in CANCEL_PENDING
// by failed recalls. Called from SWIFTReconciliationWorkflow on each scheduled run.
//
// Activity-argument contract: only the tenantID is passed. The DB handle is
// retrieved from the activity context (workerDBKey, injected by StartWorker) —
// *sql.DB is NOT serializable and must never appear in an activity signature.
//
// Resolution rules:
//   - Row within SLA + custodian response found  → CANCELLED
//   - Row within SLA + no response               → stays CANCEL_PENDING (recon re-checks)
//   - Row older than SLA                          → FAILED, reason 'recall_unresolved'
//
// All writes are tenant-scoped (GSIFI Write Rule: no gold-copy OR on writes).
func ResolveCancelPendingActivity(ctx context.Context, tenantID string) (ResolveCancelPendingResult, error) {
	logger := activity.GetLogger(ctx)

	db := workerDBFromContext(ctx)
	if db == nil {
		return ResolveCancelPendingResult{}, temporal.NewNonRetryableApplicationError(
			"ResolveCancelPendingActivity: no DB in activity context — "+
				"ensure StartWorker passes db into registerActivities via workerDBKey",
			"ErrNoDB", nil)
	}

	// Tenant-scoped READ of data rows (not config — gold-copy OR does not apply).
	rows, err := db.QueryContext(ctx, `
		SELECT id, transaction_ref, updated_at
		FROM   cash_flow.settlement
		WHERE  settlement_status = 'CANCEL_PENDING'
		  AND  tenant_id = $1
		  AND  valid_to IS NULL
		ORDER BY updated_at ASC
	`, tenantID)
	if err != nil {
		return ResolveCancelPendingResult{}, err
	}
	defer rows.Close()

	type pendingRow struct {
		ID        string
		TxRef     string
		UpdatedAt time.Time
	}
	var pending []pendingRow
	for rows.Next() {
		var p pendingRow
		var updated sql.NullTime
		if err := rows.Scan(&p.ID, &p.TxRef, &updated); err != nil {
			return ResolveCancelPendingResult{}, err
		}
		if updated.Valid {
			p.UpdatedAt = updated.Time
		}
		pending = append(pending, p)
	}
	if err := rows.Err(); err != nil {
		return ResolveCancelPendingResult{}, err
	}

	result := ResolveCancelPendingResult{Scanned: len(pending)}
	sla := time.Duration(CancelPendingSLAHours) * time.Hour

	for _, p := range pending {
		age := time.Since(p.UpdatedAt)

		if age < sla {
			confirmed, err := hasCancelResponse(ctx, db, tenantID, p.TxRef)
			if err != nil {
				logger.Error("ResolveCancelPending: response lookup failed; leaving pending",
					"transaction_ref", p.TxRef, "error", err)
				result.StillPending++
				continue
			}
			if confirmed {
				if err := setSettlementStatus(ctx, db, tenantID, p.TxRef, "CANCELLED"); err != nil {
					logger.Error("ResolveCancelPending: CANCELLED write failed",
						"transaction_ref", p.TxRef, "error", err)
					result.StillPending++
					continue
				}
				result.Resolved++
				continue
			}
			result.StillPending++ // recon will look again next run
			continue
		}

		// SLA exceeded — auto-fail so the row can never park forever.
		if err := setSettlementStatus(ctx, db, tenantID, p.TxRef, "FAILED"); err != nil {
			logger.Error("ResolveCancelPending: auto-FAILED write failed",
				"transaction_ref", p.TxRef, "error", err)
			result.StillPending++
			continue
		}
		// NOTE: 'recall_unresolved' reason — if cash_flow.settlement gains a
		// status_reason column, persist it here (see Migration note in the patch).
		logger.Info("ResolveCancelPending: SLA exceeded, auto-FAILED",
			"transaction_ref", p.TxRef, "age_hours", age.Hours(), "reason", "recall_unresolved")
		result.AutoFailed++
	}

	logger.Info("ResolveCancelPending: pass complete",
		"tenant", tenantID, "scanned", result.Scanned,
		"resolved", result.Resolved, "auto_failed", result.AutoFailed,
		"still_pending", result.StillPending)
	return result, nil
}

// hasCancelResponse checks the session log for an inbound cancellation-response
// message referencing this transaction. v1: camt.029 only (see
// CancelResponseMsgTypes for why MT548 is excluded).
//
// TODO(recall-traffic): v1 cannot distinguish acceptance from rejection.
// Parse the camt.029 CxlSts element: ACCEPTED → CANCELLED; REJECTED → the
// trade is still live and must return to PENDING_SETTLEMENT (or FAILED with
// the custodian's reason), NOT CANCELLED. Required before recall traffic.
func hasCancelResponse(ctx context.Context, db *sql.DB, tenantID, txRef string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT count(*)
		FROM   vend.swift_session_log
		WHERE  tenant_id = $1
		  AND  transaction_ref = $2
		  AND  msg_type = ANY($3)
		  AND  event_type = 'INBOUND'
	`, tenantID, txRef, CancelResponseMsgTypes).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// setSettlementStatus performs the tenant-scoped terminal-status write.
func setSettlementStatus(ctx context.Context, db *sql.DB, tenantID, txRef, status string) error {
	res, err := db.ExecContext(ctx, `
		UPDATE cash_flow.settlement
		SET    settlement_status = $3,
		       updated_at = NOW()
		WHERE  transaction_ref = $2
		  AND  tenant_id = $1
		  AND  valid_to IS NULL
	`, tenantID, txRef, status)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Row gone (superseded/rolled back) — not an error for the resolver.
		return nil
	}
	return nil
}
