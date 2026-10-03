package workflows

import (
	"errors"
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Registered names; the API and the schedule start these by name.
const (
	TenantLakehouseAuditCopyWorkflowName    = "TenantLakehouseAuditCopyWorkflow"
	TenantLakehouseAuditCopyAllWorkflowName = "TenantLakehouseAuditCopyAllWorkflow"

	// AuditCopyAllWorkflowID is the fixed id of the scheduled all-tenants run, so starting it again
	// at every worker boot is a no-op while it is already scheduled.
	AuditCopyAllWorkflowID = "lakehouse-audit-copy-all"

	// AuditCopyCron is how often every provisioned tenant's audit is copied. Under compliance-mode
	// Object Lock each commit's files are kept for the whole retention period, so this is
	// deliberately not per-event, and a run with nothing to ship commits nothing (ADR-036).
	AuditCopyCron = "*/15 * * * *"

	// maxAuditCopyBatches bounds one run. A tenant with more backlog than this is finished by
	// the next run.
	maxAuditCopyBatches = 40

	// allCopyConcurrency is how many tenants are copied at once.
	allCopyConcurrency = 5
)

// AuditCopyRunResult is what one tenant's copy run did.
type AuditCopyRunResult struct {
	TenantID  string
	Copied    int
	ThroughID int64
	Batches   int
	// Truncated is true when the run stopped at the batch limit with more still to ship.
	Truncated bool
}

// TenantLakehouseAuditCopyWorkflow copies a tenant's lakehouse audit chain into the Iceberg table in its
// own warehouse (ADR-036). alpha stays the system of record; this adds an immutable copy.
//
// It is safe to retry, re-run, or run after a crash: every batch starts by asking the destination for its
// highest id, so rows are never duplicated or skipped. There is no compensation, since nothing is undone.
// Its workflow id is per tenant, because an Iceberg append is not idempotent and there must be one writer.
func TenantLakehouseAuditCopyWorkflow(ctx workflow.Context, in activities.LakehouseProvisionInput) (*AuditCopyRunResult, error) {
	acts := &activities.TenantLakehouseActivities{} // method references only
	res := &AuditCopyRunResult{TenantID: in.TenantID}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
		},
	})

	if err := workflow.ExecuteActivity(ctx, acts.PrepareAuditCopy, in).Get(ctx, nil); err != nil {
		return res, err
	}

	for res.Batches < maxAuditCopyBatches {
		var b activities.AuditCopyResult
		if err := workflow.ExecuteActivity(ctx, acts.CopyAuditBatch, in, activities.AuditCopyBatchSize).Get(ctx, &b); err != nil {
			return res, err
		}
		res.Batches++
		res.Copied += b.Copied
		res.ThroughID = b.ThroughID
		if b.Done {
			return res, nil
		}
	}
	res.Truncated = true
	return res, nil
}

// AuditCopyAllResult summarizes a scheduled run over every provisioned tenant.
type AuditCopyAllResult struct {
	Tenants int
	Copied  int
	Skipped []string          // a copy for this tenant was already running
	Failed  map[string]string // tenant id -> reason; one tenant failing never stops the others
}

// TenantLakehouseAuditCopyAllWorkflow copies the audit of every provisioned tenant, a few at a time. A
// failing tenant is reported in the result and does not fail the run, so one broken tenant cannot stop
// the rest, and the scheduled workflow keeps running.
func TenantLakehouseAuditCopyAllWorkflow(ctx workflow.Context) (*AuditCopyAllResult, error) {
	logger := workflow.GetLogger(ctx)
	acts := &activities.TenantLakehouseActivities{}
	res := &AuditCopyAllResult{Failed: map[string]string{}}

	actx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3, InitialInterval: 5 * time.Second, BackoffCoefficient: 2.0},
	})
	var tenants []string
	if err := workflow.ExecuteActivity(actx, acts.ListProvisionedTenants).Get(actx, &tenants); err != nil {
		return res, err
	}
	res.Tenants = len(tenants)

	for start := 0; start < len(tenants); start += allCopyConcurrency {
		end := start + allCopyConcurrency
		if end > len(tenants) {
			end = len(tenants)
		}
		chunk := tenants[start:end]

		futures := make([]workflow.ChildWorkflowFuture, len(chunk))
		for i, id := range chunk {
			cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID:            "lakehouse-audit-copy-" + id,
				WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				ParentClosePolicy:     enumspb.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
			})
			futures[i] = workflow.ExecuteChildWorkflow(cctx, TenantLakehouseAuditCopyWorkflow,
				activities.LakehouseProvisionInput{TenantID: id, ActorID: "system:audit-copy", ActorRole: "system"})
		}
		for i, f := range futures {
			id := chunk[i]
			var r AuditCopyRunResult
			err := f.Get(ctx, &r)
			var already *temporal.ChildWorkflowExecutionAlreadyStartedError
			switch {
			case errors.As(err, &already):
				res.Skipped = append(res.Skipped, id) // a manual run is already copying this tenant
			case err != nil:
				res.Failed[id] = fmt.Sprintf("%v", err)
				logger.Error("lakehouse audit copy failed", "tenant", id, "error", err)
			default:
				res.Copied += r.Copied
			}
		}
	}
	return res, nil
}
