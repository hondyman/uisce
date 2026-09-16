package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// FIXReconciliationInput drives FIXReconciliationWorkflow. Mirrors the
// shape of services/ai-trade-reconciliation/backend/temporal/workflows
// (per HANDOFF_FIX_OVER_PIPELINE.md §10).
type FIXReconciliationInput struct {
	TenantID    string    `json:"tenant_id"`
	BrokerID    string    `json:"broker_id"`
	LookbackStart time.Time `json:"lookback_start"`
	LookbackEnd   time.Time `json:"lookback_end"`
}

// FIXReconciliationReport is the workflow output.
type FIXReconciliationReport struct {
	TenantID           string    `json:"tenant_id"`
	BrokerID           string    `json:"broker_id"`
	ExecutionsScanned  int       `json:"executions_scanned"`
	Mismatches         []Mismatch `json:"mismatches"`
	GeneratedAt        time.Time `json:"generated_at"`
}

// Mismatch describes one execution that did not reconcile against its
// expected order.
type Mismatch struct {
	ClOrdID    string `json:"cl_ord_id"`
	ExecID     string `json:"exec_id,omitempty"`
	Reason     string `json:"reason"`
	RawDetails string `json:"raw_details,omitempty"`
}

// FIXReconciliationWorkflow reconciles FIX ExecutionReports against
// expected orders for a single (tenant, broker) over a lookback window.
// Per HANDOFF_FIX_OVER_PIPELINE.md §10, this mirrors
// services/ai-trade-reconciliation/backend/temporal/workflows/workflows.go.
//
// Triggered by the data-pipeline's fix_execution_writer tile on every
// persisted execution report (outbox pattern from internal/datapipeline/
// outbox.go), or on a periodic schedule per
// fix_tenant_config.reconciliation_interval_sec.
func FIXReconciliationWorkflow(ctx workflow.Context, input FIXReconciliationInput) (*FIXReconciliationReport, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("FIXReconciliationWorkflow started",
		"tenant_id", input.TenantID,
		"broker_id", input.BrokerID,
		"lookback_start", input.LookbackStart,
		"lookback_end", input.LookbackEnd,
	)

	// Activity options: idempotent retries (reconciliation is safe to
	// re-run; the underlying SQL queries are read-only).
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	actx := workflow.WithActivityOptions(ctx, ao)

	// Step 1: Load executions in the lookback window.
	var executionsScanned int
	if err := workflow.ExecuteActivity(actx, "LoadExecutionsActivity", input).Get(actx, &executionsScanned); err != nil {
		return nil, fmt.Errorf("load executions: %w", err)
	}

	// Step 2: Match each execution against the expected order in OMS.
	var mismatches []Mismatch
	if err := workflow.ExecuteActivity(actx, "MatchExecutionsActivity", input).Get(actx, &mismatches); err != nil {
		return nil, fmt.Errorf("match executions: %w", err)
	}

	// Step 3: Persist the report (and emit any outbox events for
	// downstream consumers).
	report := FIXReconciliationReport{
		TenantID:          input.TenantID,
		BrokerID:          input.BrokerID,
		ExecutionsScanned: executionsScanned,
		Mismatches:        mismatches,
		GeneratedAt:       workflow.Now(ctx),
	}
	if err := workflow.ExecuteActivity(actx, "PersistReconciliationReportActivity", report).Get(actx, nil); err != nil {
		return nil, fmt.Errorf("persist report: %w", err)
	}

	logger.Info("FIXReconciliationWorkflow completed",
		"executions_scanned", executionsScanned,
		"mismatches", len(mismatches),
	)
	return &report, nil
}

// --- Activities (stubs; full implementations read from oms.trade_order
//     and fix_session_log via the GSIFI OR-clause) ---

// LoadExecutionsActivity returns the count of executions in the window.
// Full impl: SELECT COUNT(*) FROM fix_session_log WHERE ... GSIFI clause ...
func LoadExecutionsActivity(ctx context.Context, input FIXReconciliationInput) (int, error) {
	// Stub: return 0 until the activity is wired to a real DB.
	_ = input
	return 0, nil
}

// MatchExecutionsActivity returns mismatches. Full impl: left join
// fix_session_log (ExecutionReport events) against oms.trade_order by
// ClOrdID, return rows where the join fails or values disagree.
func MatchExecutionsActivity(ctx context.Context, input FIXReconciliationInput) ([]Mismatch, error) {
	_ = input
	return nil, nil
}

// PersistReconciliationReportActivity writes the report to a
// reconciliation report table. Full impl: INSERT INTO
// fix_reconciliation_report (...).
func PersistReconciliationReportActivity(ctx context.Context, report FIXReconciliationReport) error {
	_ = ctx
	_ = report
	return nil
}
