package workflows

import (
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	TenantLakehouseAuditVerifyWorkflowName = "TenantLakehouseAuditVerifyWorkflow"

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
			return rep, nil
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
			return rep, nil
		}
		if p.Done {
			rep.Verified = true
			return rep, nil
		}
	}
	return nil, workflow.NewContinueAsNewError(ctx, TenantLakehouseAuditVerifyWorkflow, AuditVerifyInput{Lakehouse: in.Lakehouse, Cursor: cur})
}
