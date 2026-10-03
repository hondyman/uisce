package workflows_test

import (
	"sync"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// These tests pin the compensation behaviour of TenantInstanceProvisioningWorkflowFn.
// The failure they guard against is deleting something the run did not create:
// RegisterTenant/RegisterInstance upsert and can return an existing row, and the
// database and namespace steps treat "already exists" as success.

var (
	freshState = provisioning.ProvisioningState{TenantOwned: true, InstanceOwned: true}
	// An existing, active tenant: nothing about it may be rolled back.
	existingTenantState = provisioning.ProvisioningState{DatabaseExisted: true}
	// Rows are ours but the database was already there.
	existingDBState = provisioning.ProvisioningState{TenantOwned: true, InstanceOwned: true, DatabaseExisted: true}
)

type sagaRun struct {
	calls []string
	err   error
}

type scenario struct {
	state    provisioning.ProvisioningState
	failStep string // activity name that fails (non-retryable)
	failComp string // compensation activity name that fails
	cancelAt string // activity that blocks, then gets the workflow cancelled
}

func (s scenario) run(t *testing.T) sagaRun {
	t.Helper()
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	acts := &activities.TenantProvisioningActivities{}
	// Register methods one by one, as the worker does: the struct also has
	// methods (GetGoldCopyInfo) that are not valid activities.
	for _, fn := range []interface{}{
		acts.RegisterTenant, acts.RegisterInstance, acts.InspectProvisioningState,
		acts.CreateTenantDatabase, acts.CloneSchemaFromGoldCopy, acts.CreateLakekeeperNamespace, acts.CloneGoldCopyProducts,
		acts.UpdateTenantStatus, acts.UpdateInstanceStatus, acts.EmitProvisioningEvent,
		acts.RollbackRegisterTenant, acts.RollbackRegisterInstance, acts.RollbackCreateTenantDatabase,
		acts.RollbackCreateLakekeeperNamespace, acts.RollbackCloneGoldCopyProducts,
	} {
		env.RegisterActivity(fn)
	}

	var mu sync.Mutex
	var calls []string
	record := func(name string) func(mock.Arguments) {
		return func(mock.Arguments) {
			mu.Lock()
			defer mu.Unlock()
			calls = append(calls, name)
		}
	}
	fail := func(name string) error {
		if name == s.failStep || name == s.failComp {
			return sdktemporal.NewNonRetryableApplicationError("boom", "Boom", nil)
		}
		return nil
	}
	// strErr mocks an activity returning (string, error).
	strErr := func(name, id string, fn interface{}) {
		c := env.OnActivity(fn, mock.Anything, mock.Anything).Run(record(name))
		if err := fail(name); err != nil {
			c.Return("", err)
			return
		}
		c.Return(id, nil)
	}

	// errOnly mocks an activity returning just error, with any number of args.
	errOnly := func(name string, fn interface{}, nargs int) {
		args := make([]interface{}, 0, nargs+1)
		args = append(args, mock.Anything)
		for i := 0; i < nargs; i++ {
			args = append(args, mock.Anything)
		}
		c := env.OnActivity(fn, args...).Run(record(name))
		if name == s.cancelAt {
			c.After(10 * time.Minute)
		}
		c.Return(fail(name))
	}

	strErr("RegisterTenant", "tenant-1", acts.RegisterTenant)
	strErr("RegisterInstance", "instance-1", acts.RegisterInstance)

	inspect := env.OnActivity(acts.InspectProvisioningState, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(record("InspectProvisioningState"))
	if err := fail("InspectProvisioningState"); err != nil {
		inspect.Return(provisioning.ProvisioningState{}, err)
	} else {
		inspect.Return(s.state, nil)
	}

	errOnly("CreateTenantDatabase", acts.CreateTenantDatabase, 1)
	errOnly("CloneSchemaFromGoldCopy", acts.CloneSchemaFromGoldCopy, 1)
	errOnly("CreateLakekeeperNamespace", acts.CreateLakekeeperNamespace, 1)
	errOnly("CloneGoldCopyProducts", acts.CloneGoldCopyProducts, 1)
	errOnly("UpdateTenantStatus", acts.UpdateTenantStatus, 2)
	errOnly("UpdateInstanceStatus", acts.UpdateInstanceStatus, 2)
	errOnly("EmitProvisioningEvent", acts.EmitProvisioningEvent, 1)

	errOnly("RollbackRegisterTenant", acts.RollbackRegisterTenant, 1)
	errOnly("RollbackRegisterInstance", acts.RollbackRegisterInstance, 1)
	errOnly("RollbackCreateTenantDatabase", acts.RollbackCreateTenantDatabase, 1)
	errOnly("RollbackCreateLakekeeperNamespace", acts.RollbackCreateLakekeeperNamespace, 1)
	errOnly("RollbackCloneGoldCopyProducts", acts.RollbackCloneGoldCopyProducts, 1)

	if s.cancelAt != "" {
		env.RegisterDelayedCallback(env.CancelWorkflow, time.Minute)
	}

	env.ExecuteWorkflow(workflows.TenantInstanceProvisioningWorkflowFn, provisioning.ProvisioningWorkflowInput{
		TenantName:       "Acme",
		TenantCode:       "acme",
		InstanceName:     "prod",
		GoldCopyDatabase: "gold",
		DatabaseName:     "tenant_acme",
		LakekeeperNS:     "acme",
	})
	require.True(t, env.IsWorkflowCompleted())

	mu.Lock()
	defer mu.Unlock()
	return sagaRun{calls: append([]string(nil), calls...), err: env.GetWorkflowError()}
}

func TestSaga_Success_RunsAllStepsAndNoCompensation(t *testing.T) {
	r := scenario{state: freshState}.run(t)
	require.NoError(t, r.err)
	require.Equal(t, []string{
		"RegisterTenant", "RegisterInstance", "InspectProvisioningState",
		"CreateTenantDatabase", "CloneSchemaFromGoldCopy", "CreateLakekeeperNamespace", "CloneGoldCopyProducts",
		"UpdateTenantStatus", "UpdateInstanceStatus", "EmitProvisioningEvent",
	}, r.calls)
}

// A failed step must compensate (the old workflow compensated only on
// cancellation), in reverse order, and only for steps that completed.
func TestSaga_FailureCompensatesCompletedStepsInReverse(t *testing.T) {
	head := []string{"RegisterTenant", "RegisterInstance", "InspectProvisioningState"}
	cases := []struct {
		failStep string
		want     []string
	}{
		{"RegisterTenant", []string{"RegisterTenant", "EmitProvisioningEvent"}},
		{"RegisterInstance", []string{"RegisterTenant", "RegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent"}},
		{"InspectProvisioningState", append(append([]string{}, head...), "RollbackRegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent")},
		{"CreateTenantDatabase", append(append([]string{}, head...), "CreateTenantDatabase", "RollbackRegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent")},
		// Regression: the database used to be left behind, and the namespace
		// rollback ran instead (for a namespace that was never created).
		{"CloneSchemaFromGoldCopy", append(append([]string{}, head...),
			"CreateTenantDatabase", "CloneSchemaFromGoldCopy",
			"RollbackCreateTenantDatabase", "RollbackRegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent")},
		{"CreateLakekeeperNamespace", append(append([]string{}, head...),
			"CreateTenantDatabase", "CloneSchemaFromGoldCopy", "CreateLakekeeperNamespace",
			"RollbackCreateTenantDatabase", "RollbackRegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent")},
		{"CloneGoldCopyProducts", append(append([]string{}, head...),
			"CreateTenantDatabase", "CloneSchemaFromGoldCopy", "CreateLakekeeperNamespace", "CloneGoldCopyProducts",
			"RollbackCreateLakekeeperNamespace", "RollbackCreateTenantDatabase", "RollbackRegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent")},
	}
	for _, c := range cases {
		t.Run(c.failStep, func(t *testing.T) {
			r := scenario{state: freshState, failStep: c.failStep}.run(t)
			require.Error(t, r.err)
			require.Equal(t, c.want, r.calls)
		})
	}
}

// The destructive case: provisioning lands on a tenant that already exists and is
// active. A failure must not roll back its database, namespace, products, instance
// or tenant row.
func TestSaga_ExistingActiveTenant_IsNeverRolledBack(t *testing.T) {
	r := scenario{state: existingTenantState, failStep: "CloneGoldCopyProducts"}.run(t)
	require.Error(t, r.err)
	require.Equal(t, []string{
		"RegisterTenant", "RegisterInstance", "InspectProvisioningState",
		"CreateTenantDatabase", "CloneSchemaFromGoldCopy", "CreateLakekeeperNamespace", "CloneGoldCopyProducts",
		"EmitProvisioningEvent",
	}, r.calls)
	for _, c := range r.calls {
		require.NotContains(t, c, "Rollback", "a pre-existing tenant must not be compensated")
	}
}

// Rows are ours but the database pre-existed: it must not be dropped.
func TestSaga_PreExistingDatabase_IsNotDropped(t *testing.T) {
	r := scenario{state: existingDBState, failStep: "CreateLakekeeperNamespace"}.run(t)
	require.Error(t, r.err)
	require.NotContains(t, r.calls, "RollbackCreateTenantDatabase")
	require.Contains(t, r.calls, "RollbackRegisterInstance")
	require.Contains(t, r.calls, "RollbackRegisterTenant")
}

// One stuck compensation must not strand the others.
func TestSaga_FailedCompensationDoesNotStopTheRest(t *testing.T) {
	r := scenario{state: freshState, failStep: "CloneGoldCopyProducts", failComp: "RollbackCreateLakekeeperNamespace"}.run(t)
	require.Error(t, r.err)
	tail := r.calls[len(r.calls)-4:]
	require.Equal(t, []string{"RollbackCreateTenantDatabase", "RollbackRegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent"}, tail)
	require.Contains(t, r.calls, "RollbackCreateLakekeeperNamespace")
}

// Cancellation (the only case the old workflow handled) must still compensate,
// and in the right order.
func TestSaga_CancellationCompensates(t *testing.T) {
	r := scenario{state: freshState, cancelAt: "CreateLakekeeperNamespace"}.run(t)
	require.Error(t, r.err)
	require.Contains(t, r.calls, "RollbackCreateTenantDatabase")
	require.Contains(t, r.calls, "RollbackRegisterInstance")
	require.Contains(t, r.calls, "RollbackRegisterTenant")
	require.NotContains(t, r.calls, "RollbackCreateLakekeeperNamespace", "the cancelled step never completed")
}
