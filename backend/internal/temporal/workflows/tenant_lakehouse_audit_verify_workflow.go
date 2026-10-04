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

const (
	TenantLakehouseAuditVerifyWorkflowName    = "TenantLakehouseAuditVerifyWorkflow"
	TenantLakehouseAuditVerifyAllWorkflowName = "TenantLakehouseAuditVerifyAllWorkflow"

	// AuditVerifyAllWorkflowID is the fixed id of the scheduled all-tenants run, so starting it at every
	// worker boot is a no-op while it is already scheduled.
	AuditVerifyAllWorkflowID = "lakehouse-audit-verify-all"

	// AuditVerifyCron is how often every provisioned tenant is verified: nightly, off the copy's
	// quarter-hour cadence. It only reads, so it adds no objects under Object Lock.
	AuditVerifyCron = "17 3 * * *"

	verifyAllConcurrency = 5

	// verifyPagesPerRun bounds one run's history. A longer chain continues as a new run from the cursor.
	verifyPagesPerRun = 50
)

// AuditVerifyWorkflowID is the one id a tenant's verification runs under. Verification only reads, so
// this is not a single-writer rule; it stops two operators racing the same long scan.
func AuditVerifyWorkflowID(tenantID string) string { return "lakehouse-audit-verify-" + tenantID }

// AuditVerifyInput starts a verification, or continues one from its cursor.
type AuditVerifyInput struct {
	Lakehouse activities.LakehouseProvisionInput
	Cursor    activities.AuditVerifyCursor
}

// TenantLakehouseAuditVerifyWorkflow proves a tenant's Iceberg audit copy matches alpha (ADR-035). It is
// read-only: it never copies, repairs or marks anything, so a finding stays visible until a person acts on
// it. A finding is a RESULT, returned with a nil error, so it is not retried; only an outage is.
func TenantLakehouseAuditVerifyWorkflow(ctx workflow.Context, in AuditVerifyInput) (*activities.AuditVerifyReport, error) {
	acts := &activities.TenantLakehouseActivities{} // method references only
	rep := &activities.AuditVerifyReport{TenantID: in.Lakehouse.TenantID}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
		},
	})

	// record stores the run's final outcome where the tiering job reads it. A run that hands over to a
	// continuation records nothing: only the last run knows the outcome.
	record := func() error {
		return workflow.ExecuteActivity(ctx, acts.RecordAuditVerification, in.Lakehouse, *rep).Get(ctx, nil)
	}

	cur := in.Cursor
	// alpha's own chain is checked once, at the start. A comparison against a chain that does not
	// recompute would show the copy faithfully reproducing a broken record.
	if cur.AfterID == 0 && cur.Rows == 0 {
		var f *activities.AuditFinding
		if err := workflow.ExecuteActivity(ctx, acts.VerifyAlphaAuditChain, in.Lakehouse).Get(ctx, &f); err != nil {
			return rep, err
		}
		if f != nil {
			rep.Finding = f
			return rep, record()
		}
	}

	for pages := 0; pages < verifyPagesPerRun; pages++ {
		var p activities.AuditVerifyPage
		if err := workflow.ExecuteActivity(ctx, acts.VerifyAuditPage, in.Lakehouse, cur, activities.AuditVerifyPageSize).Get(ctx, &p); err != nil {
			return rep, err
		}
		cur = p.Cursor
		rep.Rows, rep.ThroughID = cur.Rows, cur.AfterID
		if p.Finding != nil {
			rep.Finding = p.Finding
			return rep, record()
		}
		if p.Done {
			rep.Verified = true
			return rep, record()
		}
	}
	return nil, workflow.NewContinueAsNewError(ctx, TenantLakehouseAuditVerifyWorkflow, AuditVerifyInput{Lakehouse: in.Lakehouse, Cursor: cur})
}

// AuditVerifyAllResult summarizes a scheduled verification of every provisioned tenant.
type AuditVerifyAllResult struct {
	Tenants  int
	Verified int
	Findings map[string]string // tenant id -> finding kind
	Failed   map[string]string // tenant id -> reason the run itself could not finish
	Skipped  []string          // a verification for this tenant was already running
}

// TenantLakehouseAuditVerifyAllWorkflow verifies every provisioned tenant, a few at a time. A finding or a
// failing tenant is reported in the result and never fails the run, so one tenant cannot stop the rest
// and the schedule keeps running.
func TenantLakehouseAuditVerifyAllWorkflow(ctx workflow.Context) (*AuditVerifyAllResult, error) {
	logger := workflow.GetLogger(ctx)
	acts := &activities.TenantLakehouseActivities{}
	res := &AuditVerifyAllResult{Findings: map[string]string{}, Failed: map[string]string{}}

	actx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3, InitialInterval: 5 * time.Second, BackoffCoefficient: 2.0},
	})
	var tenants []string
	if err := workflow.ExecuteActivity(actx, acts.ListProvisionedTenants).Get(actx, &tenants); err != nil {
		return res, err
	}
	res.Tenants = len(tenants)

	for start := 0; start < len(tenants); start += verifyAllConcurrency {
		end := start + verifyAllConcurrency
		if end > len(tenants) {
			end = len(tenants)
		}
		chunk := tenants[start:end]
		futures := make([]workflow.ChildWorkflowFuture, len(chunk))
		for i, id := range chunk {
			cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID:            AuditVerifyWorkflowID(id),
				WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				ParentClosePolicy:     enumspb.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
			})
			futures[i] = workflow.ExecuteChildWorkflow(cctx, TenantLakehouseAuditVerifyWorkflow, AuditVerifyInput{
				Lakehouse: activities.LakehouseProvisionInput{TenantID: id, ActorID: "system:audit-verify", ActorRole: "system"}})
		}
		for i, f := range futures {
			id := chunk[i]
			var r activities.AuditVerifyReport
			err := f.Get(ctx, &r)
			var already *temporal.ChildWorkflowExecutionAlreadyStartedError
			switch {
			case errors.As(err, &already):
				res.Skipped = append(res.Skipped, id)
			case err != nil:
				res.Failed[id] = fmt.Sprintf("%v", err)
				logger.Error("lakehouse audit verification could not finish", "tenant", id, "error", err)
			case r.Finding != nil:
				res.Findings[id] = r.Finding.Kind
				logger.Error("lakehouse audit copy does not match alpha", "tenant", id, "kind", r.Finding.Kind, "entry", r.Finding.ID)
			default:
				res.Verified++
			}
		}
	}
	return res, nil
}
