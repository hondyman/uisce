package workflows_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"github.com/hondyman/uisce/backend/internal/migrations"
	"github.com/hondyman/uisce/backend/internal/migrations/ormmove"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
)

// The product path ("create tenant X in region R with product P and label L"): the region's
// cluster is checked before anything is created, the database is made on that cluster, only the
// chosen products are registered, and the app is seeded. These pin the order, what each failure
// undoes, and that the older database steps are not used.

type productRun struct {
	calls     []string
	err       error
	register  provisioning.RegisterTenantInput
	clone     provisioning.CloneProductsInput
	regionDB  provisioning.RegionDatabaseInput
	rollbackR provisioning.RegionDatabaseInput
}

func runProductPath(t *testing.T, failStep string) productRun {
	t.Helper()
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	acts := &activities.TenantProvisioningActivities{}
	for _, fn := range []interface{}{
		acts.RegisterTenant, acts.RegisterInstance, acts.InspectProvisioningState,
		acts.ResolveStructureTemplate, acts.PlanTenantStructure,
		acts.AssertRegionCluster, acts.CreateTenantDatabaseInRegion, acts.RollbackCreateTenantDatabaseInRegion,
		acts.CreateTenantDatabase, acts.CloneSchemaFromGoldCopy, acts.RollbackCreateTenantDatabase,
		acts.CreateLakekeeperNamespace, acts.RollbackCreateLakekeeperNamespace,
		acts.CloneGoldCopyProducts, acts.RollbackCloneGoldCopyProducts,
		acts.BindTenantDatabase, acts.ProvisionTenantDatabaseAccess, acts.ApplyTenantStructure,
		acts.SeedTenantDatabase, acts.ProbeTenantDatabase, acts.ActivateTenantDatabase, acts.RollbackTenantDatabase,
		acts.UpdateTenantStatus, acts.UpdateInstanceStatus, acts.EmitProvisioningEvent,
		acts.RollbackRegisterTenant, acts.RollbackRegisterInstance,
	} {
		env.RegisterActivity(fn)
	}

	var (
		mu  sync.Mutex
		out productRun
	)
	record := func(name string) func(mock.Arguments) {
		return func(args mock.Arguments) {
			mu.Lock()
			defer mu.Unlock()
			out.calls = append(out.calls, name)
			switch name {
			case "RegisterTenant":
				out.register = args.Get(1).(provisioning.RegisterTenantInput)
			case "CloneGoldCopyProducts":
				out.clone = args.Get(1).(provisioning.CloneProductsInput)
			case "CreateTenantDatabaseInRegion":
				out.regionDB = args.Get(1).(provisioning.RegionDatabaseInput)
			case "RollbackCreateTenantDatabaseInRegion":
				out.rollbackR = args.Get(1).(provisioning.RegionDatabaseInput)
			}
		}
	}
	boom := func(name string) error {
		if name == failStep {
			return sdktemporal.NewNonRetryableApplicationError("boom", "Boom", nil)
		}
		return nil
	}
	// ret mocks an activity of one argument that returns only an error.
	ret := func(name string, fn interface{}) {
		env.OnActivity(fn, mock.Anything, mock.Anything).Run(record(name)).Return(boom(name))
	}
	ret2 := func(name string, fn interface{}) { // two arguments after ctx
		env.OnActivity(fn, mock.Anything, mock.Anything, mock.Anything).Run(record(name)).Return(boom(name))
	}

	str := func(name, id string, fn interface{}) {
		c := env.OnActivity(fn, mock.Anything, mock.Anything).Run(record(name))
		if err := boom(name); err != nil {
			c.Return("", err)
			return
		}
		c.Return(id, nil)
	}
	str("RegisterTenant", "tenant-1", acts.RegisterTenant)
	str("RegisterInstance", "instance-1", acts.RegisterInstance)
	str("ResolveStructureTemplate", "template-1", acts.ResolveStructureTemplate)

	insp := env.OnActivity(acts.InspectProvisioningState, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(record("InspectProvisioningState"))
	insp.Return(provisioning.ProvisioningState{TenantOwned: true, InstanceOwned: true}, nil)

	plan := env.OnActivity(acts.PlanTenantStructure, mock.Anything, mock.Anything).Run(record("PlanTenantStructure"))
	plan.Return(provisioning.StructurePlan{Hash: "h", Tables: 522}, nil)

	ret("AssertRegionCluster", acts.AssertRegionCluster)
	ret("CreateTenantDatabaseInRegion", acts.CreateTenantDatabaseInRegion)
	ret("RollbackCreateTenantDatabaseInRegion", acts.RollbackCreateTenantDatabaseInRegion)
	ret("CreateLakekeeperNamespace", acts.CreateLakekeeperNamespace)
	ret("RollbackCreateLakekeeperNamespace", acts.RollbackCreateLakekeeperNamespace)
	ret("CloneGoldCopyProducts", acts.CloneGoldCopyProducts)
	ret("RollbackCloneGoldCopyProducts", acts.RollbackCloneGoldCopyProducts)

	bind := env.OnActivity(acts.BindTenantDatabase, mock.Anything, mock.Anything).Run(record("BindTenantDatabase"))
	if err := boom("BindTenantDatabase"); err != nil {
		bind.Return(provisioning.TenantDatabaseBinding{}, err)
	} else {
		bind.Return(provisioning.TenantDatabaseBinding{DatasourceID: "ds-1", Role: "abc_orm_app"}, nil)
	}
	ret("ProvisionTenantDatabaseAccess", acts.ProvisionTenantDatabaseAccess)
	apply := env.OnActivity(acts.ApplyTenantStructure, mock.Anything, mock.Anything).Run(record("ApplyTenantStructure"))
	if err := boom("ApplyTenantStructure"); err != nil {
		apply.Return(migrations.Report{}, err)
	} else {
		apply.Return(migrations.Report{Done: true}, nil)
	}
	seed := env.OnActivity(acts.SeedTenantDatabase, mock.Anything, mock.Anything).Run(record("SeedTenantDatabase"))
	if err := boom("SeedTenantDatabase"); err != nil {
		seed.Return(ormmove.Report{}, err)
	} else {
		seed.Return(ormmove.Report{Done: true}, nil)
	}
	ret("ProbeTenantDatabase", acts.ProbeTenantDatabase)
	ret("ActivateTenantDatabase", acts.ActivateTenantDatabase)
	ret("RollbackTenantDatabase", acts.RollbackTenantDatabase)
	ret2("UpdateTenantStatus", acts.UpdateTenantStatus)
	ret2("UpdateInstanceStatus", acts.UpdateInstanceStatus)
	ret("EmitProvisioningEvent", acts.EmitProvisioningEvent)
	ret("RollbackRegisterTenant", acts.RollbackRegisterTenant)
	ret("RollbackRegisterInstance", acts.RollbackRegisterInstance)

	env.ExecuteWorkflow(workflows.TenantInstanceProvisioningWorkflowFn, provisioning.ProvisioningWorkflowInput{
		TenantName: "XYZ Investments", TenantCode: "xyz_i", InstanceName: "primary",
		GoldCopyDatabase: "crims", DatabaseName: "abc_orm", LakekeeperNS: "xyz_i",
		App: "orm", StructureFromGoldCopy: true,
		Region: "us-east-1", ClusterHost: "100.84.50.65", ClusterPort: 5432,
		ProductCodes: []string{"orm"}, Seed: true,
	})
	require.True(t, env.IsWorkflowCompleted())

	mu.Lock()
	defer mu.Unlock()
	out.err = env.GetWorkflowError()
	return out
}

func TestProductPath_RunsTheStepsInOrder(t *testing.T) {
	r := runProductPath(t, "")
	require.NoError(t, r.err)
	require.Equal(t, []string{
		"RegisterTenant", "RegisterInstance", "InspectProvisioningState",
		"ResolveStructureTemplate", "PlanTenantStructure",
		"AssertRegionCluster", "CreateTenantDatabaseInRegion",
		"CreateLakekeeperNamespace", "CloneGoldCopyProducts",
		"BindTenantDatabase", "ProvisionTenantDatabaseAccess", "ApplyTenantStructure", "SeedTenantDatabase",
		"ProbeTenantDatabase", "ActivateTenantDatabase",
		"UpdateTenantStatus", "UpdateInstanceStatus", "EmitProvisioningEvent",
	}, r.calls)
	require.NotContains(t, r.calls, "CreateTenantDatabase", "the older create reads DATABASE_URL and defaults; the product path must not use it")
	require.NotContains(t, r.calls, "CloneSchemaFromGoldCopy")

	require.Equal(t, "us-east-1", r.register.Region, "the tenant row records the region it was asked for")
	require.Equal(t, []string{"orm"}, r.clone.ProductCodes, "only the chosen product is registered")
	require.Equal(t, provisioning.RegionDatabaseInput{Region: "us-east-1", Host: "100.84.50.65", Port: 5432, DatabaseName: "abc_orm"}, r.regionDB)
}

// A region that is not this worker's cluster, or a name already in use, is refused before a database,
// namespace or product exists: only the two registration rows are undone.
func TestProductPath_ARefusedClusterCreatesNothing(t *testing.T) {
	r := runProductPath(t, "AssertRegionCluster")
	require.Error(t, r.err)
	require.Equal(t, []string{
		"RegisterTenant", "RegisterInstance", "InspectProvisioningState",
		"ResolveStructureTemplate", "PlanTenantStructure", "AssertRegionCluster",
		"RollbackRegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent",
	}, r.calls)
}

func TestProductPath_AFailureAfterTheDatabaseDropsItOnTheRegionsCluster(t *testing.T) {
	cases := map[string][]string{
		"SeedTenantDatabase": {
			"RollbackTenantDatabase", "RollbackCloneGoldCopyProducts", "RollbackCreateLakekeeperNamespace",
			"RollbackCreateTenantDatabaseInRegion", "RollbackRegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent",
		},
		"CloneGoldCopyProducts": {
			"RollbackCreateLakekeeperNamespace", "RollbackCreateTenantDatabaseInRegion",
			"RollbackRegisterInstance", "RollbackRegisterTenant", "EmitProvisioningEvent",
		},
	}
	for step, wantTail := range cases {
		t.Run(step, func(t *testing.T) {
			r := runProductPath(t, step)
			require.Error(t, r.err)
			require.Equal(t, wantTail, r.calls[len(r.calls)-len(wantTail):])
			require.Equal(t, "abc_orm", r.rollbackR.DatabaseName)
			require.Equal(t, "100.84.50.65", r.rollbackR.Host, "the drop goes to the cluster the database was made on")
			require.NotContains(t, r.calls, "RollbackCreateTenantDatabase", "the older rollback is not used on this path")
		})
	}
}

// A failed seed must stop the tenant from going active: the reference rows are part of a tenant that works.
func TestProductPath_ASeedFailureLeavesTheTenantInactive(t *testing.T) {
	r := runProductPath(t, "SeedTenantDatabase")
	require.Error(t, r.err)
	for _, c := range []string{"ProbeTenantDatabase", "ActivateTenantDatabase", "UpdateTenantStatus", "UpdateInstanceStatus"} {
		require.NotContains(t, r.calls, c)
	}
}
