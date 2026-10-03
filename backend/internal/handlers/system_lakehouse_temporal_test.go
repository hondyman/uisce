package handlers

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

type fakeRun struct{ client.WorkflowRun }

func (fakeRun) GetID() string { return "run-id" }

type fakeStarter struct {
	opts client.StartWorkflowOptions
	wf   interface{}
	args []interface{}
	err  error
}

func (f *fakeStarter) ExecuteWorkflow(_ context.Context, o client.StartWorkflowOptions, wf interface{}, args ...interface{}) (client.WorkflowRun, error) {
	f.opts, f.wf, f.args = o, wf, args
	if f.err != nil {
		return nil, f.err
	}
	return fakeRun{}, nil
}

func TestTemporalLakehouseProvisioner_StartsOnePerTenantOnTheDeployedQueue(t *testing.T) {
	f := &fakeStarter{}
	p := &TemporalLakehouseProvisioner{c: f, queue: LakehouseTaskQueue}
	id := uuid.New()

	wfID, err := p.StartProvision(context.Background(), id, registry.Actor{ID: "alice", Role: "global_admin"})
	require.NoError(t, err)
	require.Equal(t, "run-id", wfID)

	require.Equal(t, "lakehouse-provision-"+id.String(), f.opts.ID, "one workflow id per tenant means at most one active run")
	require.Equal(t, "bp_queue", f.opts.TaskQueue)
	require.Equal(t, enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY, f.opts.WorkflowIDReusePolicy,
		"a failed run may be retried; a running or completed one may not be duplicated")
	require.Equal(t, 30*time.Minute, f.opts.WorkflowExecutionTimeout)
	require.Equal(t, LakehouseWorkflowName, f.wf)
	require.Equal(t, []interface{}{activities.LakehouseProvisionInput{TenantID: id.String(), ActorID: "alice", ActorRole: "global_admin"}}, f.args)
}

func TestTemporalLakehouseProvisioner_Errors(t *testing.T) {
	id := uuid.New()
	f := &fakeStarter{err: serviceerror.NewWorkflowExecutionAlreadyStarted("running", "", "")}
	_, err := (&TemporalLakehouseProvisioner{c: f, queue: LakehouseTaskQueue}).StartProvision(context.Background(), id, registry.Actor{})
	require.ErrorIs(t, err, ErrProvisionInProgress)

	f = &fakeStarter{err: errors.New("temporal unavailable")}
	_, err = (&TemporalLakehouseProvisioner{c: f, queue: LakehouseTaskQueue}).StartProvision(context.Background(), id, registry.Actor{})
	require.ErrorContains(t, err, "temporal unavailable")
	require.NotErrorIs(t, err, ErrProvisionInProgress, "an outage is not 'already running'")
}

// The starter and the worker must agree on the workflow name and queue. The older
// provisioning saga was started on a queue no worker polled and never ran; these keep that
// from happening again, silently.
func TestLakehouseWorkflowIsWiredIntoTheDeployedWorker(t *testing.T) {
	require.Equal(t, workflows.TenantLakehouseProvisioningWorkflowName, LakehouseWorkflowName)
	require.Equal(t, workflows.TenantLakehouseRetentionWorkflowName, LakehouseRetentionWorkflowName)
	require.Equal(t, workflows.TenantLakehouseAuditCopyWorkflowName, LakehouseAuditCopyWorkflowName)

	src, err := os.ReadFile("../../cmd/worker/main.go")
	require.NoError(t, err)
	main := string(src)
	// Not require.Contains: on failure it would print all of main.go.
	has := func(fragment, why string) {
		t.Helper()
		if !strings.Contains(main, fragment) {
			t.Errorf("cmd/worker/main.go does not contain %q: %s", fragment, why)
		}
	}
	has(`worker.New(temporalClient, "`+LakehouseTaskQueue+`"`, "cmd/worker must poll the queue the starter uses")
	has("TenantLakehouseProvisioningWorkflow", "the workflow must be registered on that worker")
	has("TenantLakehouseRetentionWorkflow", "and so must the retention reconcile")
	has("TenantLakehouseActivities", "and so must its activities")
	has("TenantLakehouseAuditCopyWorkflow,", "the per-tenant audit copy must be registered")
	has("TenantLakehouseAuditCopyAllWorkflow,", "and so must the all-tenants run the schedule starts")
	has("StartTenantLakehouseAuditCopyCron(", "the worker must start the audit copy schedule")
	has("AuditDestinationFromEnv()", "the activities need their StarRocks destination")
	for _, line := range strings.Split(main, "\n") {
		if strings.Contains(line, "RegisterSafeActivity") && strings.Contains(line, "Lakehouse") {
			t.Errorf("lakehouse activities must never be BP-Designer client-safe: %s", strings.TrimSpace(line))
		}
	}
}

func TestTemporalLakehouseProvisioner_RetentionSync(t *testing.T) {
	id := uuid.New()
	f := &fakeStarter{}
	p := &TemporalLakehouseProvisioner{c: f, queue: LakehouseTaskQueue}

	wfID, err := p.StartRetentionSync(context.Background(), id, registry.Actor{ID: "alice", Role: "global_admin"})
	require.NoError(t, err)
	require.Equal(t, "run-id", wfID)
	require.Equal(t, "lakehouse-retention-"+id.String(), f.opts.ID, "at most one sync runs per tenant")
	require.Equal(t, "bp_queue", f.opts.TaskQueue)
	require.Equal(t, enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE, f.opts.WorkflowIDReusePolicy,
		"every extension needs a new run, so a finished one must not block it")
	require.Equal(t, LakehouseRetentionWorkflowName, f.wf)
	require.Equal(t, []interface{}{activities.LakehouseProvisionInput{TenantID: id.String(), ActorID: "alice", ActorRole: "global_admin"}}, f.args)

	f.err = serviceerror.NewWorkflowExecutionAlreadyStarted("running", "", "")
	_, err = p.StartRetentionSync(context.Background(), id, registry.Actor{})
	require.ErrorIs(t, err, ErrRetentionSyncInProgress)

	f.err = errors.New("temporal unavailable")
	_, err = p.StartRetentionSync(context.Background(), id, registry.Actor{})
	require.ErrorContains(t, err, "temporal unavailable")
	require.NotErrorIs(t, err, ErrRetentionSyncInProgress)
}

func TestTemporalLakehouseProvisioner_AuditCopy(t *testing.T) {
	id := uuid.New()
	f := &fakeStarter{}
	p := &TemporalLakehouseProvisioner{c: f, queue: LakehouseTaskQueue}

	wfID, err := p.StartAuditCopy(context.Background(), id, registry.Actor{ID: "alice", Role: "global_admin"})
	require.NoError(t, err)
	require.Equal(t, "run-id", wfID)
	require.Equal(t, workflows.AuditCopyWorkflowID(id.String()), f.opts.ID,
		"the same id the schedule's children use, so there is one writer per tenant")
	require.Equal(t, "bp_queue", f.opts.TaskQueue)
	require.Equal(t, LakehouseAuditCopyWorkflowName, f.wf)

	f.err = serviceerror.NewWorkflowExecutionAlreadyStarted("running", "", "")
	_, err = p.StartAuditCopy(context.Background(), id, registry.Actor{})
	require.ErrorIs(t, err, ErrAuditCopyInProgress)

	f.err = errors.New("temporal unavailable")
	_, err = p.StartAuditCopy(context.Background(), id, registry.Actor{})
	require.NotErrorIs(t, err, ErrAuditCopyInProgress)
}
