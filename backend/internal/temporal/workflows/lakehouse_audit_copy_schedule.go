package workflows

import (
	"context"
	"errors"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

// StartTenantLakehouseAuditCopyCron starts the scheduled all-tenants audit copy (every AuditCopyCron). It
// uses a fixed workflow id, so calling it at every worker boot is a no-op while the schedule is already
// running; "already started" is success, not an error. taskQueue must be a queue the deployed worker polls.
func StartTenantLakehouseAuditCopyCron(ctx context.Context, c client.Client, taskQueue string) error {
	_, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    AuditCopyAllWorkflowID,
		TaskQueue:             taskQueue,
		CronSchedule:          AuditCopyCron,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		// One tick must finish well before the next, or runs would pile up.
		WorkflowRunTimeout: 14 * time.Minute,
	}, TenantLakehouseAuditCopyAllWorkflowName)
	var started *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &started) {
		return nil
	}
	return err
}
