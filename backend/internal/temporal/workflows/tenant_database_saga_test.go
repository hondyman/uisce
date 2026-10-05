package workflows_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"github.com/hondyman/uisce/backend/internal/migrations"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
)

// These tests pin the tenant-database steps of the provisioning saga (ADR-030): they run only
// when the request names an app, they run before the tenant is activated, they thread the
// datasource id through, and a failure compensates in the right order and only for a run that
// owns the tenant.

type tdScenario struct {
	app          string
	template     string // TemplateDatasourceID; non-empty with an app selects the compiled-structure path (ADR-050)
	fromMarker   bool   // StructureFromGoldCopy: the saga resolves the template from the gold copy's marker
	resolved     string // what ResolveStructureTemplate returns (default goldTemplate)
	planHash     string // returned by PlanTenantStructure (default "plan-hash")
	state        provisioning.ProvisioningState
	failStep     string
	report       *migrations.Report // returned by ApplyTenantMigrations; nil means Done
	bindResult   string
	stepInputs   map[string]provisioning.TenantDatabaseInput
	stepInputsMu sync.Mutex
}

func (s *tdScenario) run(t *testing.T) sagaRun {
	t.Helper()
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	acts := &activities.TenantProvisioningActivities{}
	for _, fn := range []interface{}{
		acts.RegisterTenant, acts.RegisterInstance, acts.InspectProvisioningState,
		acts.CreateTenantDatabase, acts.CloneSchemaFromGoldCopy, acts.CreateLakekeeperNamespace, acts.CloneGoldCopyProducts,
		acts.BindTenantDatabase, acts.ProvisionTenantDatabaseAccess, acts.ApplyTenantMigrations,
		acts.ResolveStructureTemplate, acts.PlanTenantStructure, acts.ApplyTenantStructure,
		acts.ProbeTenantDatabase, acts.ActivateTenantDatabase,
		acts.UpdateTenantStatus, acts.UpdateInstanceStatus, acts.EmitProvisioningEvent,
		acts.RollbackRegisterTenant, acts.RollbackRegisterInstance, acts.RollbackCreateTenantDatabase,
		acts.RollbackCreateLakekeeperNamespace, acts.RollbackCloneGoldCopyProducts, acts.RollbackTenantDatabase,
	} {
		env.RegisterActivity(fn)
	}
	s.stepInputs = map[string]provisioning.TenantDatabaseInput{}
	var mu sync.Mutex
	var calls []string
	record := func(name string) func(mock.Arguments) {
		return func(args mock.Arguments) {
			mu.Lock()
			calls = append(calls, name)
			mu.Unlock()
			if len(args) > 1 {
				if in, ok := args.Get(1).(provisioning.TenantDatabaseInput); ok {
					s.stepInputsMu.Lock()
					s.stepInputs[name] = in
					s.stepInputsMu.Unlock()
				}
			}
		}
	}
	fail := func(name string) error {
		if name == s.failStep {
			return sdktemporal.NewNonRetryableApplicationError("boom", "Boom", nil)
		}
		return nil
	}
	errOnly := func(name string, fn interface{}, nargs int) {
		args := []interface{}{mock.Anything}
		for i := 0; i < nargs; i++ {
			args = append(args, mock.Anything)
		}
		env.OnActivity(fn, args...).Run(record(name)).Return(fail(name))
	}

	env.OnActivity(acts.RegisterTenant, mock.Anything, mock.Anything).Run(record("RegisterTenant")).Return("tenant-1", nil)
	env.OnActivity(acts.RegisterInstance, mock.Anything, mock.Anything).Run(record("RegisterInstance")).Return("instance-1", nil)
	env.OnActivity(acts.InspectProvisioningState, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(record("InspectProvisioningState")).Return(s.state, nil)
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

	bound := s.bindResult
	if bound == "" {
		bound = "ds-orm"
	}
	if err := fail("BindTenantDatabase"); err != nil {
		env.OnActivity(acts.BindTenantDatabase, mock.Anything, mock.Anything).Run(record("BindTenantDatabase")).
			Return(provisioning.TenantDatabaseBinding{}, err)
	} else {
		env.OnActivity(acts.BindTenantDatabase, mock.Anything, mock.Anything).Run(record("BindTenantDatabase")).
			Return(provisioning.TenantDatabaseBinding{DatasourceID: bound, Role: "r", SecretPath: "/p"}, nil)
	}
	errOnly("ProvisionTenantDatabaseAccess", acts.ProvisionTenantDatabaseAccess, 1)
	rep := migrations.Report{Target: "tenant:t:orm", Done: true}
	if s.report != nil {
		rep = *s.report
	}
	if err := fail("ApplyTenantMigrations"); err != nil {
		env.OnActivity(acts.ApplyTenantMigrations, mock.Anything, mock.Anything).Run(record("ApplyTenantMigrations")).
			Return(migrations.Report{}, err)
	} else {
		env.OnActivity(acts.ApplyTenantMigrations, mock.Anything, mock.Anything).Run(record("ApplyTenantMigrations")).Return(rep, nil)
	}
	hash := s.planHash
	if hash == "" {
		hash = "plan-hash"
	}
	resolved := s.resolved
	if resolved == "" {
		resolved = goldTemplate
	}
	if err := fail("ResolveStructureTemplate"); err != nil {
		env.OnActivity(acts.ResolveStructureTemplate, mock.Anything, mock.Anything).Run(record("ResolveStructureTemplate")).Return("", err)
	} else {
		env.OnActivity(acts.ResolveStructureTemplate, mock.Anything, mock.Anything).Run(record("ResolveStructureTemplate")).Return(resolved, nil)
	}
	if err := fail("PlanTenantStructure"); err != nil {
		env.OnActivity(acts.PlanTenantStructure, mock.Anything, mock.Anything).Run(record("PlanTenantStructure")).
			Return(provisioning.StructurePlan{}, err)
	} else {
		env.OnActivity(acts.PlanTenantStructure, mock.Anything, mock.Anything).Run(record("PlanTenantStructure")).
			Return(provisioning.StructurePlan{Hash: hash, Tables: 522, Statements: 3520, Schemas: []string{"orm", "mdm"}}, nil)
	}
	if err := fail("ApplyTenantStructure"); err != nil {
		env.OnActivity(acts.ApplyTenantStructure, mock.Anything, mock.Anything).Run(record("ApplyTenantStructure")).
			Return(migrations.Report{}, err)
	} else {
		env.OnActivity(acts.ApplyTenantStructure, mock.Anything, mock.Anything).Run(record("ApplyTenantStructure")).Return(rep, nil)
	}
	errOnly("ProbeTenantDatabase", acts.ProbeTenantDatabase, 1)
	errOnly("ActivateTenantDatabase", acts.ActivateTenantDatabase, 1)
	errOnly("RollbackTenantDatabase", acts.RollbackTenantDatabase, 1)

	env.ExecuteWorkflow(workflows.TenantInstanceProvisioningWorkflowFn, provisioning.ProvisioningWorkflowInput{
		TenantName: "Acme", TenantCode: "acme", InstanceName: "prod", GoldCopyDatabase: "gold",
		DatabaseName: "orm_acme", LakekeeperNS: "acme", App: s.app, BaselineThrough: "0001_x.up.sql",
		TemplateDatasourceID: s.template, StructureFromGoldCopy: s.fromMarker,
	})
	require.True(t, env.IsWorkflowCompleted())
	mu.Lock()
	defer mu.Unlock()
	return sagaRun{calls: append([]string(nil), calls...), err: env.GetWorkflowError()}
}

func index(calls []string, name string) int {
	for i, c := range calls {
		if c == name {
			return i
		}
	}
	return -1
}

var tdSteps = []string{"BindTenantDatabase", "ProvisionTenantDatabaseAccess", "ApplyTenantMigrations", "ProbeTenantDatabase", "ActivateTenantDatabase"}

func TestTenantDatabaseSaga_WithoutAnAppTheSagaIsUnchanged(t *testing.T) {
	r := (&tdScenario{state: freshState}).run(t)
	require.NoError(t, r.err)
	for _, step := range append(tdSteps, "RollbackTenantDatabase") {
		require.Equal(t, -1, index(r.calls, step), "%s must not run without an app", step)
	}
	require.Contains(t, r.calls, "UpdateTenantStatus")
}

func TestTenantDatabaseSaga_RunsTheStepsInOrderBeforeActivation(t *testing.T) {
	s := &tdScenario{app: "orm", state: freshState, bindResult: "ds-42"}
	r := s.run(t)
	require.NoError(t, r.err)

	last := index(r.calls, "CloneGoldCopyProducts")
	for _, step := range tdSteps {
		i := index(r.calls, step)
		require.Greater(t, i, last, "%s must run after the products are cloned, in order", step)
		last = i
	}
	require.Less(t, last, index(r.calls, "UpdateTenantStatus"), "the tenant must not be marked active before its database is proven")
	require.Equal(t, -1, index(r.calls, "RollbackTenantDatabase"))

	// Every later step receives the datasource the bind chose, and the request's inputs.
	for _, step := range tdSteps[1:] {
		in := s.stepInputs[step]
		require.Equal(t, "ds-42", in.DatasourceID, step)
		require.Equal(t, "orm", in.App, step)
		require.Equal(t, "orm_acme", in.DatabaseName, step)
		require.Equal(t, "0001_x.up.sql", in.BaselineThrough, step)
		require.Equal(t, "tenant-1", in.TenantID, step)
		require.Equal(t, "instance-1", in.InstanceID, step)
	}
	require.Equal(t, "gold", s.stepInputs["BindTenantDatabase"].GoldCopyDatabase)
}

func TestTenantDatabaseSaga_AFailureCompensatesAndNeverActivatesTheTenant(t *testing.T) {
	for _, failStep := range tdSteps {
		t.Run(failStep, func(t *testing.T) {
			r := (&tdScenario{app: "orm", state: freshState, failStep: failStep}).run(t)
			require.Error(t, r.err)
			require.Equal(t, -1, index(r.calls, "UpdateTenantStatus"), "a tenant whose database failed must not be activated")
			require.Equal(t, -1, index(r.calls, "UpdateInstanceStatus"))
			for _, later := range tdSteps[index(tdSteps, failStep)+1:] {
				require.Equal(t, -1, index(r.calls, later), "%s must not run after %s failed", later, failStep)
			}

			rb, products := index(r.calls, "RollbackTenantDatabase"), index(r.calls, "RollbackCloneGoldCopyProducts")
			require.NotEqual(t, -1, products)
			if failStep == "BindTenantDatabase" {
				require.Equal(t, -1, rb, "nothing was bound, so there is nothing to undo")
				return
			}
			require.NotEqual(t, -1, rb, "a bound database must be undone")
			require.Less(t, rb, products, "the role and credential go before the products (the binding row cascades with them)")
			require.Less(t, products, index(r.calls, "RollbackCreateTenantDatabase"), "and the database itself goes last of the three")
			require.Less(t, rb, index(r.calls, "RollbackRegisterTenant"))
		})
	}
}

func TestTenantDatabaseSaga_AnExistingTenantIsNeverRolledBack(t *testing.T) {
	r := (&tdScenario{app: "orm", state: existingTenantState, failStep: "ProbeTenantDatabase"}).run(t)
	require.Error(t, r.err)
	for _, c := range r.calls {
		require.NotContains(t, c, "Rollback", "a pre-existing tenant must not be compensated")
	}
}

func TestTenantDatabaseSaga_AnIncompleteMigrationReportFailsTheRun(t *testing.T) {
	r := (&tdScenario{app: "orm", state: freshState, report: &migrations.Report{Target: "tenant:t:orm", Done: false, Pending: []string{"0002.up.sql"}}}).run(t)
	require.Error(t, r.err)
	require.Equal(t, -1, index(r.calls, "ProbeTenantDatabase"))
	require.Equal(t, -1, index(r.calls, "ActivateTenantDatabase"))
	require.NotEqual(t, -1, index(r.calls, "RollbackTenantDatabase"))
}

// ---- the compiled-structure path (ADR-050) ----------------------------------------------------

const goldTemplate = "11111111-1111-4111-8111-111111111111"

var structureSteps = []string{"BindTenantDatabase", "ProvisionTenantDatabaseAccess", "ApplyTenantStructure", "ProbeTenantDatabase", "ActivateTenantDatabase"}

func TestStructureSaga_PlansBeforeAnythingIsCreatedAndNeverClonesTheGoldCopy(t *testing.T) {
	s := &tdScenario{app: "orm", template: goldTemplate, state: freshState, bindResult: "ds-42", planHash: "abc123"}
	r := s.run(t)
	require.NoError(t, r.err)

	require.NotEqual(t, -1, index(r.calls, "PlanTenantStructure"))
	require.Greater(t, index(r.calls, "PlanTenantStructure"), index(r.calls, "InspectProvisioningState"))
	require.Less(t, index(r.calls, "PlanTenantStructure"), index(r.calls, "CreateTenantDatabase"),
		"a template that cannot be deployed must refuse the run before a database exists")
	require.Equal(t, -1, index(r.calls, "CloneSchemaFromGoldCopy"), "the structure comes from alpha's scan; the gold copy's database is never read")
	require.Equal(t, -1, index(r.calls, "ApplyTenantMigrations"), "the compiled structure replaces the migration directory step")

	last := index(r.calls, "CloneGoldCopyProducts")
	for _, step := range structureSteps {
		i := index(r.calls, step)
		require.Greater(t, i, last, "%s must run after the products are cloned, in order", step)
		last = i
	}
	require.Less(t, last, index(r.calls, "UpdateTenantStatus"))

	// What was planned is what is applied, for the datasource the bind chose.
	apply := s.stepInputs["ApplyTenantStructure"]
	require.Equal(t, "abc123", apply.StructureHash)
	require.Equal(t, goldTemplate, apply.TemplateDatasourceID)
	require.Equal(t, "ds-42", apply.DatasourceID)
	require.Equal(t, goldTemplate, s.stepInputs["PlanTenantStructure"].TemplateDatasourceID)
	require.Equal(t, "orm_acme", s.stepInputs["PlanTenantStructure"].DatabaseName)
}

func TestStructureSaga_TheMarkerIsResolvedBeforeThePlanAndTheResolvedIdIsUsedThroughout(t *testing.T) {
	s := &tdScenario{app: "orm", fromMarker: true, state: freshState, bindResult: "ds-42", resolved: "22222222-2222-4222-8222-222222222222"}
	r := s.run(t)
	require.NoError(t, r.err)
	require.NotEqual(t, -1, index(r.calls, "ResolveStructureTemplate"))
	require.Less(t, index(r.calls, "ResolveStructureTemplate"), index(r.calls, "PlanTenantStructure"))
	require.Less(t, index(r.calls, "PlanTenantStructure"), index(r.calls, "CreateTenantDatabase"))
	for _, step := range []string{"PlanTenantStructure", "BindTenantDatabase", "ApplyTenantStructure"} {
		require.Equal(t, "22222222-2222-4222-8222-222222222222", s.stepInputs[step].TemplateDatasourceID, step)
	}
}

func TestStructureSaga_AnExplicitIdNeedsNoResolution(t *testing.T) {
	r := (&tdScenario{app: "orm", template: goldTemplate, state: freshState}).run(t)
	require.NoError(t, r.err)
	require.Equal(t, -1, index(r.calls, "ResolveStructureTemplate"))
}

func TestStructureSaga_ZeroOrSeveralMarkedTemplatesCreateNothing(t *testing.T) {
	r := (&tdScenario{app: "orm", fromMarker: true, state: freshState, failStep: "ResolveStructureTemplate"}).run(t)
	require.Error(t, r.err)
	for _, never := range []string{"PlanTenantStructure", "CreateTenantDatabase", "CloneGoldCopyProducts", "BindTenantDatabase", "ApplyTenantStructure"} {
		require.Equal(t, -1, index(r.calls, never), never)
	}
	require.NotEqual(t, -1, index(r.calls, "RollbackRegisterTenant"))
}

func TestStructureSaga_ARefusedPlanCreatesNothingAndUndoesOnlyTheRows(t *testing.T) {
	r := (&tdScenario{app: "orm", template: goldTemplate, state: freshState, failStep: "PlanTenantStructure"}).run(t)
	require.Error(t, r.err)
	for _, never := range []string{"CreateTenantDatabase", "CloneSchemaFromGoldCopy", "CreateLakekeeperNamespace", "CloneGoldCopyProducts",
		"BindTenantDatabase", "ProvisionTenantDatabaseAccess", "ApplyTenantStructure", "UpdateTenantStatus"} {
		require.Equal(t, -1, index(r.calls, never), "%s must not run after the plan was refused", never)
	}
	require.NotEqual(t, -1, index(r.calls, "RollbackRegisterInstance"))
	require.NotEqual(t, -1, index(r.calls, "RollbackRegisterTenant"))
	require.Equal(t, -1, index(r.calls, "RollbackCreateTenantDatabase"), "there is no database to drop")
}

func TestStructureSaga_AFailedApplyCompensatesLikeAFailedMigration(t *testing.T) {
	r := (&tdScenario{app: "orm", template: goldTemplate, state: freshState, failStep: "ApplyTenantStructure"}).run(t)
	require.Error(t, r.err)
	require.Equal(t, -1, index(r.calls, "ProbeTenantDatabase"))
	require.Equal(t, -1, index(r.calls, "UpdateTenantStatus"), "a tenant whose structure failed must not be activated")
	require.NotEqual(t, -1, index(r.calls, "RollbackTenantDatabase"))
	require.Less(t, index(r.calls, "RollbackTenantDatabase"), index(r.calls, "RollbackCreateTenantDatabase"))

	r = (&tdScenario{app: "orm", template: goldTemplate, state: freshState, report: &migrations.Report{Target: "tenant:t:structure", Done: false, Pending: []string{"0001_structure.up.sql"}}}).run(t)
	require.Error(t, r.err, "an incomplete report is not a built structure")
	require.Equal(t, -1, index(r.calls, "ActivateTenantDatabase"))
}

func TestStructureSaga_ATemplateWithoutAnAppOrAnAppWithoutATemplateTakesTheOldPath(t *testing.T) {
	r := (&tdScenario{template: goldTemplate, state: freshState}).run(t)
	require.NoError(t, r.err)
	require.Equal(t, -1, index(r.calls, "PlanTenantStructure"), "no app: the tenant-database steps do not run at all")
	require.NotEqual(t, -1, index(r.calls, "CloneSchemaFromGoldCopy"))

	r = (&tdScenario{app: "orm", state: freshState}).run(t)
	require.NoError(t, r.err)
	require.Equal(t, -1, index(r.calls, "PlanTenantStructure"))
	require.Equal(t, -1, index(r.calls, "ApplyTenantStructure"))
	require.NotEqual(t, -1, index(r.calls, "CloneSchemaFromGoldCopy"), "no template: the gold copy is cloned as before")
	require.NotEqual(t, -1, index(r.calls, "ApplyTenantMigrations"))
}

func TestStructureSaga_AnExistingTenantIsNeverRolledBack(t *testing.T) {
	r := (&tdScenario{app: "orm", template: goldTemplate, state: existingTenantState, failStep: "ApplyTenantStructure"}).run(t)
	require.Error(t, r.err)
	for _, c := range r.calls {
		require.NotContains(t, c, "Rollback", "a pre-existing tenant must not be compensated")
	}
}
