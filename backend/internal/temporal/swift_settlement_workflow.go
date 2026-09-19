package temporal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ErrPipelineNotImplemented is returned by RunSWIFTPipelineDAGActivity until
// the pipeline engine wiring lands (nifty-greider-015b86 worktree merge).
// A NonRetryableApplicationError makes this a deterministic workflow failure at
// VALIDATING — no fake-green SETTLED state is reachable while this stub exists.
type workerDBKey struct{}

var ErrPipelineNotImplemented = temporal.NewNonRetryableApplicationError(
	"SWIFT pipeline engine not yet wired: RunSWIFTPipelineDAGActivity is a stub. "+
		"Wire to pipeline engine after nifty-greider-015b86 worktree merge.",
	"ErrPipelineNotImplemented",
	nil,
)

type SWIFTSettlementInput struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	CustodianID        uuid.UUID `json:"custodian_id"`
	TransactionRef     string    `json:"transaction_ref"`
	UETR               string    `json:"uetr,omitempty"`
	AdminURL           string    `json:"admin_url"`
	AdminToken         string    `json:"admin_token"`
	PipelineDAGID      string    `json:"pipeline_dag_id"`
	SettlementBudgetMs int64     `json:"settlement_budget_ms"`
	RawMessage         []byte    `json:"raw_message"`
	MsgType            string    `json:"msg_type"`
}

type SWIFTSettlementResult struct {
	TransactionRef string    `json:"transaction_ref"`
	UETR           string    `json:"uetr,omitempty"`
	FinalStatus    string    `json:"final_status"` // SETTLED | FAILED | CANCELLED
	SettledAt      time.Time `json:"settled_at,omitempty"`
	FailureReason  string    `json:"failure_reason,omitempty"`
}

type SettlementSignal struct {
	Status     string    `json:"status"` // "matched", "settled", "failed"
	UETR       string    `json:"uetr,omitempty"`
	Details    string    `json:"details,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

type PipelineResult struct {
	Success bool   `json:"success"`
	Details string `json:"details,omitempty"`
}

func SWIFTSettlementWorkflow(ctx workflow.Context, input SWIFTSettlementInput) (*SWIFTSettlementResult, error) {
	logger := workflow.GetLogger(ctx)

	state := "RECEIVED"
	logger.Info("SWIFTSettlementWorkflow: state transition", "to", state)

	ackCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // outbound sends are not safely re-runnable
		},
	})

	// Send ACK. On timeout/error the message may have already been delivered,
	// so we log and continue — SWIFTReconciliationWorkflow resolves the
	// ambiguity on its next run rather than driving the workflow to FAILED here.
	// See HANDOFF_SWIFT_SETTLEMENT.md §Ack-timeout semantics.
	if err := workflow.ExecuteActivity(ackCtx, SWIFTAckActivity, input).Get(ctx, nil); err != nil {
		logger.Warn("SWIFTSettlementWorkflow: ACK failed or timed out; continuing to pipeline (recon will resolve)",
			"error", err)
	}

	state = "VALIDATING"
	logger.Info("SWIFTSettlementWorkflow: state transition", "to", state)

	budget := input.SettlementBudgetMs
	if budget <= 0 {
		budget = 60000
	}

	pipelineCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(budget) * time.Millisecond,
		// Default retry policy: NonRetryable errors (e.g. ErrPipelineNotImplemented)
		// surface immediately without retries.
	})

	// dbCtx hoisted before the pipeline call so it is in scope for
	// PersistSettlementStatusActivity on the pipeline-error path.
	dbCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	})

	var pResult PipelineResult
	if err := workflow.ExecuteActivity(pipelineCtx, RunSWIFTPipelineDAGActivity, input).Get(ctx, &pResult); err != nil {
		// Pipeline not yet wired, OR real pipeline failure. Drive to FAILED.
		// Persist so recon and GET /instructions/{id} see a terminal status row.
		// 0-rows-affected log is expected while the pipeline stub is active (no row
		// exists yet to update), but the call is correct-by-construction once real rows appear.
		state = "FAILED"
		pipelineFailReason := err.Error()
		logger.Error("SWIFTSettlementWorkflow: pipeline failed; state transition to FAILED", "error", err)
		if persistErr := workflow.ExecuteActivity(dbCtx, PersistSettlementStatusActivity,
			input.TransactionRef, "FAILED",
			input.TenantID.String(), input.CustodianID.String(),
		).Get(ctx, nil); persistErr != nil {
			logger.Error("SWIFTSettlementWorkflow: failed to persist FAILED status on pipeline error",
				"error", persistErr)
		}
		return &SWIFTSettlementResult{
			TransactionRef: input.TransactionRef,
			UETR:           input.UETR,
			FinalStatus:    state,
			FailureReason:  pipelineFailReason,
		}, nil
	}

	state = "MATCHED"
	logger.Info("SWIFTSettlementWorkflow: state transition", "to", state)

	settlementUpdateSig := workflow.GetSignalChannel(ctx, "SettlementUpdate")
	cancelSig := workflow.GetSignalChannel(ctx, "Cancel")
	deadlineTimer := workflow.NewTimer(ctx, 72*time.Hour)

	recallCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // outbound; not safe to retry
		},
	})



	var result SWIFTSettlementResult
	result.TransactionRef = input.TransactionRef
	result.UETR = input.UETR

	for {
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(settlementUpdateSig, func(c workflow.ReceiveChannel, more bool) {
			var sig SettlementSignal
			c.Receive(ctx, &sig)
			logger.Info("SWIFTSettlementWorkflow: SettlementUpdate signal received", "status", sig.Status)

			switch sig.Status {
			case "matched":
				state = "PENDING_SETTLEMENT"
				logger.Info("SWIFTSettlementWorkflow: state transition", "to", state)

			case "settled":
				if err := workflow.ExecuteActivity(dbCtx, PersistSettlementStatusActivity,
					input.TransactionRef, "SETTLED",
					input.TenantID.String(), input.CustodianID.String(),
				).Get(ctx, nil); err != nil {
					logger.Error("SWIFTSettlementWorkflow: failed to persist SETTLED status", "error", err)
				}
				state = "SETTLED"
				logger.Info("SWIFTSettlementWorkflow: state transition", "to", state)
				result.FinalStatus = state
				result.SettledAt = sig.OccurredAt

			case "failed":
				if err := workflow.ExecuteActivity(dbCtx, PersistSettlementStatusActivity,
					input.TransactionRef, "FAILED",
					input.TenantID.String(), input.CustodianID.String(),
				).Get(ctx, nil); err != nil {
					logger.Error("SWIFTSettlementWorkflow: failed to persist FAILED status", "error", err)
				}
				state = "FAILED"
				logger.Info("SWIFTSettlementWorkflow: state transition", "to", state)
				result.FinalStatus = state
				result.FailureReason = sig.Details
			}
		})

		selector.AddReceive(cancelSig, func(c workflow.ReceiveChannel, more bool) {
			var reason struct{ Reason string }
			c.Receive(ctx, &reason)
			logger.Info("SWIFTSettlementWorkflow: Cancel signal received", "reason", reason.Reason)

			recallErr := workflow.ExecuteActivity(recallCtx, SWIFTRecallActivity, input).Get(ctx, nil)

			// If recall failed, the instruction may still be live at the custodian.
			// Persisting CANCELLED when the recall has not been confirmed writes a lie
			// that SWIFTReconciliationWorkflow would have to unwind.
			// Park in CANCEL_PENDING instead; recon resolves it to CANCELLED or FAILED
			// once the custodian's status is known. Same ambiguity-deferral contract as
			// the ack-timeout path.
			persistStatus := "CANCELLED"
			finalState := "CANCELLED"
			if recallErr != nil {
				logger.Error("SWIFTSettlementWorkflow: recall failed; parking in CANCEL_PENDING (recon will resolve)",
					"error", recallErr)
				persistStatus = "CANCEL_PENDING"
				finalState = "CANCEL_PENDING"
			}

			if err := workflow.ExecuteActivity(dbCtx, PersistSettlementStatusActivity,
				input.TransactionRef, persistStatus,
				input.TenantID.String(), input.CustodianID.String(),
			).Get(ctx, nil); err != nil {
				logger.Error("SWIFTSettlementWorkflow: failed to persist status",
					"status", persistStatus, "error", err)
			}
			state = finalState
			logger.Info("SWIFTSettlementWorkflow: state transition", "to", state)
			result.FinalStatus = finalState
			result.FailureReason = "cancelled:" + reason.Reason
		})

		selector.AddFuture(deadlineTimer, func(f workflow.Future) {
			if err := workflow.ExecuteActivity(dbCtx, PersistSettlementStatusActivity,
				input.TransactionRef, "FAILED",
				input.TenantID.String(), input.CustodianID.String(),
			).Get(ctx, nil); err != nil {
				logger.Error("SWIFTSettlementWorkflow: failed to persist deadline-exceeded FAILED status", "error", err)
			}
			state = "FAILED"
			logger.Info("SWIFTSettlementWorkflow: deadline exceeded; state transition", "to", state)
			result.FinalStatus = state
			result.FailureReason = "settlement_deadline_exceeded"
		})

		selector.Select(ctx)

		if state == "SETTLED" || state == "FAILED" || state == "CANCELLED" || state == "CANCEL_PENDING" {
			return &result, nil
		}
	}
}

// SWIFTAckActivity sends an MT0xx / pacs.002 acknowledgement via the admin
// server. MaximumAttempts=1 — outbound sends are not safely re-runnable.
// On error the workflow logs a warning and continues (not FAILED) because
// the message may have been delivered; SWIFTReconciliationWorkflow resolves
// the ambiguity on its next scheduled run.
//
// TODO(ack-body): build the real MT011/MT012 or pacs.002 ACK message body.
// Until then, returns ErrAckNotImplemented so the workflow logs the gap but
// continues (consistent with ack-timeout semantics — ambiguous, not failed).
var ErrAckNotImplemented = fmt.Errorf(
	"SWIFTAckActivity: ACK message body not yet implemented. " +
		"Build MT011/MT012 or pacs.002 body before sending to custodian. " +
		"Workflow continues; SWIFTReconciliationWorkflow will resolve ambiguity.")

func SWIFTAckActivity(ctx context.Context, input SWIFTSettlementInput) error {
	activity.GetLogger(ctx).Warn("SWIFTAckActivity: NOT IMPLEMENTED — ACK body is a stub",
		"custodian_id", input.CustodianID,
		"transaction_ref", input.TransactionRef,
		"msg_type", input.MsgType,
		"action_required", "build MT011/MT012 or pacs.002 ACK body before production use")
	// Return the not-implemented error. The workflow caller treats ACK errors
	// as warnings (MaximumAttempts=1, continues on failure) so the settlement
	// workflow is not blocked — but the caller sees a real error in Temporal UI.
	return ErrAckNotImplemented
}

// RunSWIFTPipelineDAGActivity executes the SWIFT inbound pipeline DAG.
//
// TODO(worktree-merge): replace stub body with:
//
//	engine.ExecuteDAG(ctx, input.PipelineDAGID, record)
//
// Until then, returns ErrPipelineNotImplemented (NonRetryable) which drives
// SWIFTSettlementWorkflow to FAILED at VALIDATING. No path to SETTLED exists
// while this stub is active.
func RunSWIFTPipelineDAGActivity(ctx context.Context, input SWIFTSettlementInput) (PipelineResult, error) {
	activity.GetLogger(ctx).Error("RunSWIFTPipelineDAGActivity: NOT IMPLEMENTED — workflow will fail at VALIDATING",
		"dag_id", input.PipelineDAGID,
		"tenant_id", input.TenantID,
		"transaction_ref", input.TransactionRef,
		"action_required", "wire pipeline engine after nifty-greider-015b86 merge")
	return PipelineResult{}, ErrPipelineNotImplemented
}

// PersistSettlementStatusActivity writes the terminal settlement status to
// cash_flow.settlement using the GSIFI isolation clause. Called for SETTLED,
// FAILED (signal or deadline), and CANCELLED states.
//
// DB is retrieved from context via workerDBKey injected by StartWorker.
// Returns NonRetryable if no DB is available so the workflow surfaces the
// misconfiguration immediately rather than retrying 3x.
func PersistSettlementStatusActivity(ctx context.Context, transactionRef, status, tenantIDStr, custodianIDStr string) error {
	logger := activity.GetLogger(ctx)

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("PersistSettlementStatusActivity: invalid tenant_id %q", tenantIDStr),
			"ErrInvalidInput", err,
		)
	}

	db := workerDBFromContext(ctx)
	if db == nil {
		return temporal.NewNonRetryableApplicationError(
			"PersistSettlementStatusActivity: no database in activity context — "+
				"ensure StartWorker passes db into registerActivities via workerDBKey",
			"ErrNoDB", nil,
		)
	}

	// Write: per-tenant row wins; the gold-copy OR-clause is included so that
	// during testing/dev where a settlement was written by the gold-copy tenant,
	// the status update still lands. In production the tenant_id = $3 branch
	// is always the matching arm.
	// GSIFI write rule: scope strictly to tenant_id = $3.
	// The gold-copy OR-clause applies to reads (SELECT/inheritance) only.
	// Using it on a write would allow tenant A's workflow to mutate a gold-copy
	// row if they share a transaction_ref — a cross-tenant isolation violation.
	res, err := db.ExecContext(ctx, `
		UPDATE cash_flow.settlement
		SET    settlement_status = $2,
		       updated_at        = NOW()
		WHERE  transaction_ref = $1
		  AND  tenant_id = $3
		  AND  valid_to IS NULL
	`, transactionRef, status, tenantID)
	if err != nil {
		return fmt.Errorf("PersistSettlementStatusActivity: UPDATE failed: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		// Row absent because pipeline stub never wrote it. Log warning only —
		// do not error so the workflow can still reach its terminal state.
		logger.Warn("PersistSettlementStatusActivity: no rows updated",
			"transaction_ref", transactionRef, "status", status, "tenant_id", tenantID,
			"note", "cash_flow.settlement row absent — pipeline stub hasn't written it yet")
	} else {
		logger.Info("PersistSettlementStatusActivity: updated",
			"transaction_ref", transactionRef, "status", status, "rows_affected", n)
	}
	return nil
}

// workerDBFromContext retrieves the *sql.DB stored by StartWorker via workerDBKey.
func workerDBFromContext(ctx context.Context) *sql.DB {
	db, _ := ctx.Value(workerDBKey{}).(*sql.DB)
	return db
}

// SWIFTRecallActivity dispatches an MT192 (MT) or camt.056 (MX) recall via
// the admin server. MaximumAttempts=1 — outbound sends are not safely re-runnable.
//
// TODO(recall-body): build the real MT192 or camt.056 recall message body.
// Until then, returns ErrRecallNotImplemented (NonRetryable) so the cancel
// path fails loudly in Temporal UI rather than sending an empty-body POST
// to the custodian and pretending the recall succeeded.
var ErrRecallNotImplemented = temporal.NewNonRetryableApplicationError(
	"SWIFTRecallActivity: MT192/camt.056 recall message body not yet implemented. "+
		"Build the full recall message body before enabling the cancel path in production.",
	"ErrRecallNotImplemented",
	nil,
)

func SWIFTRecallActivity(ctx context.Context, input SWIFTSettlementInput) error {
	activity.GetLogger(ctx).Error("SWIFTRecallActivity: NOT IMPLEMENTED — no recall message sent",
		"custodian_id", input.CustodianID,
		"transaction_ref", input.TransactionRef,
		"msg_type", input.MsgType,
		"action_required", "build MT192 or camt.056 recall body before production use")
	return ErrRecallNotImplemented
}
