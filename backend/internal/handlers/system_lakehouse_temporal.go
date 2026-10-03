package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

const (
	// LakehouseWorkflowName is the registered name of the provisioning workflow. It must
	// equal workflows.TenantLakehouseProvisioningWorkflowName (a test enforces it).
	LakehouseWorkflowName = "TenantLakehouseProvisioningWorkflow"

	// LakehouseTaskQueue is the queue the DEPLOYED worker (cmd/worker) polls. It is not
	// pkg/workflows.BPTaskQueue ("bp-framework-queue"), which that worker does not poll, and
	// it is not "tenant-provisioning", which no worker polls (the older provisioning saga was
	// started there and never ran). A test reads cmd/worker/main.go to keep this honest.
	LakehouseTaskQueue = "bp_queue"
)

// workflowStarter is the part of client.Client the provisioner uses.
type workflowStarter interface {
	ExecuteWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error)
}

// TemporalLakehouseProvisioner starts the tenant lakehouse provisioning workflow.
type TemporalLakehouseProvisioner struct {
	c     workflowStarter
	queue string
}

func NewTemporalLakehouseProvisioner(c client.Client) *TemporalLakehouseProvisioner {
	return &TemporalLakehouseProvisioner{c: c, queue: LakehouseTaskQueue}
}

// StartProvision starts the workflow with an id derived from the tenant, so at most one
// run per tenant is ever active. A new run is allowed only after the previous one failed
// or was cancelled (the activities are idempotent, so that resumes the work); while one is
// running, or after one completed, the start is refused as ErrProvisionInProgress.
func (p *TemporalLakehouseProvisioner) StartProvision(ctx context.Context, tenantID uuid.UUID, actor registry.Actor) (string, error) {
	run, err := p.c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       "lakehouse-provision-" + tenantID.String(),
		TaskQueue:                p.queue,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		WorkflowExecutionTimeout: 30 * time.Minute,
	}, LakehouseWorkflowName, activities.LakehouseProvisionInput{
		TenantID:  tenantID.String(),
		ActorID:   actor.ID,
		ActorRole: actor.Role,
	})
	var started *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &started) {
		return "", ErrProvisionInProgress
	}
	if err != nil {
		return "", err
	}
	return run.GetID(), nil
}
