package temporal

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type SWIFTReconciliationInput struct {
	TenantID      string    `json:"tenant_id"`
	CustodianID   string    `json:"custodian_id"`
	LookbackStart time.Time `json:"lookback_start"`
	LookbackEnd   time.Time `json:"lookback_end"`
}

type SWIFTReconciliationReport struct {
	TenantID            string          `json:"tenant_id"`
	CustodianID         string          `json:"custodian_id"`
	InstructionsScanned int             `json:"instructions_scanned"`
	Mismatches          []SWIFTMismatch `json:"mismatches"`
	GeneratedAt         time.Time       `json:"generated_at"`
}

type SWIFTMismatch struct {
	TransactionRef string `json:"transaction_ref"`
	UETR           string `json:"uetr,omitempty"`
	Reason         string `json:"reason"`
	RawDetails     string `json:"raw_details,omitempty"`
}

func SWIFTReconciliationWorkflow(ctx workflow.Context, input SWIFTReconciliationInput) (*SWIFTReconciliationReport, error) {
	logger := workflow.GetLogger(ctx)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	actx := workflow.WithActivityOptions(ctx, ao)

	var instructionsScanned int
	err := workflow.ExecuteActivity(actx, LoadSWIFTExpectedSettlementsActivity, input).Get(ctx, &instructionsScanned)
	if err != nil {
		logger.Error("SWIFTReconciliationWorkflow: failed to load expected settlements", "error", err)
		return nil, err
	}

	var mismatches []SWIFTMismatch
	err = workflow.ExecuteActivity(actx, MatchSWIFTSettlementsActivity, input).Get(ctx, &mismatches)
	if err != nil {
		logger.Error("SWIFTReconciliationWorkflow: failed to match settlements", "error", err)
		return nil, err
	}

	report := &SWIFTReconciliationReport{
		TenantID:            input.TenantID,
		CustodianID:         input.CustodianID,
		InstructionsScanned: instructionsScanned,
		Mismatches:          mismatches,
		GeneratedAt:         workflow.Now(ctx),
	}

	err = workflow.ExecuteActivity(actx, PersistSWIFTReconciliationReportActivity, report).Get(ctx, nil)
	if err != nil {
		logger.Error("SWIFTReconciliationWorkflow: failed to persist report", "error", err)
		return nil, err
	}

	if len(mismatches) > 0 {
		err = workflow.ExecuteActivity(actx, EscalateUnmatchedActivity, mismatches).Get(ctx, nil)
		if err != nil {
			logger.Error("SWIFTReconciliationWorkflow: failed to escalate unmatched items", "error", err)
			return nil, err
		}
	}

	// Resolve CANCEL_PENDING settlement rows that have aged out or received responses.
	resolveCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})
	var resolveResult ResolveCancelPendingResult
	if err := workflow.ExecuteActivity(resolveCtx, ResolveCancelPendingActivity, input.TenantID).Get(ctx, &resolveResult); err != nil {
		logger.Error("SWIFTReconciliationWorkflow: CANCEL_PENDING resolver failed", "error", err)
		// Non-fatal for the recon run — mismatch detection still completes.
	} else {
		logger.Info("SWIFTReconciliationWorkflow: CANCEL_PENDING resolver finished",
			"scanned", resolveResult.Scanned,
			"resolved", resolveResult.Resolved,
			"auto_failed", resolveResult.AutoFailed,
			"still_pending", resolveResult.StillPending)
	}

	return report, nil
}

func LoadSWIFTExpectedSettlementsActivity(ctx context.Context, input SWIFTReconciliationInput) (int, error) {
	activity.GetLogger(ctx).Info("LoadSWIFTExpectedSettlementsActivity", "tenant", input.TenantID)
	// SELECT COUNT(*) FROM cash_flow.settlement WHERE settlement_status = 'PENDING' AND (tenant_id = $1 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)) AND created_at BETWEEN $lookback_start AND $lookback_end
	return 100, nil
}

func MatchSWIFTSettlementsActivity(ctx context.Context, input SWIFTReconciliationInput) ([]SWIFTMismatch, error) {
	activity.GetLogger(ctx).Info("MatchSWIFTSettlementsActivity", "tenant", input.TenantID)
	// LEFT JOIN cash_flow.settlement against swift_session_log WHERE event_type = 'settled' AND (tenant_id = $1 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
	return []SWIFTMismatch{}, nil
}

func PersistSWIFTReconciliationReportActivity(ctx context.Context, report *SWIFTReconciliationReport) error {
	activity.GetLogger(ctx).Info("PersistSWIFTReconciliationReportActivity", "tenant", report.TenantID)
	// INSERT INTO swift_reconciliation_report with GSIFI context for the tenant (tenant_id = $1 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
	return nil
}

func EscalateUnmatchedActivity(ctx context.Context, mismatches []SWIFTMismatch) error {
	activity.GetLogger(ctx).Info("EscalateUnmatchedActivity", "count", len(mismatches))
	// create human tasks
	return nil
}
