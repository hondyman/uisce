package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/migrations"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"go.temporal.io/sdk/workflow"
	sdktemporal "go.temporal.io/sdk/temporal"
)

type TenantInstanceProvisioningWorkflow struct {
	Activities *activities.TenantProvisioningActivities
}

func NewTenantInstanceProvisioningWorkflow(acts *activities.TenantProvisioningActivities) *TenantInstanceProvisioningWorkflow {
	return &TenantInstanceProvisioningWorkflow{
		Activities: acts,
	}
}

func (w *TenantInstanceProvisioningWorkflow) Execute(ctx workflow.Context, input provisioning.ProvisioningWorkflowInput) (*provisioning.ProvisioningWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("TenantInstanceProvisioningWorkflow started",
		"tenant", input.TenantName,
		"tenantCode", input.TenantCode,
		"instance", input.InstanceName)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			MaximumAttempts:    3,
			BackoffCoefficient: 2.0,
			InitialInterval:    10 * time.Second,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := &provisioning.ProvisioningWorkflowResult{
		TenantID:     input.TenantID,
		InstanceID:   input.InstanceID,
		DatabaseName: input.DatabaseName,
		LakekeeperNS: input.LakekeeperNS,
		Status:       "provisioning",
	}

	var (
		step1Done, step2Done, step3Done, step4Done, step5Done bool
		registeredTenantID, registeredInstanceID string
	)

	defer func() {
		if ctx.Err() != nil {
			dc, _ := workflow.NewDisconnectedContext(ctx)
			if step5Done {
				workflow.ExecuteActivity(dc, w.Activities.EmitProvisioningEvent, provisioning.EmitEventInput{
					TenantID:    input.TenantID,
					InstanceID:  input.InstanceID,
					Status:      "failed",
					Error:       ctx.Err().Error(),
					CompletedAt: time.Now(),
				}).Get(dc, nil)
			}
			if step5Done {
				workflow.ExecuteActivity(dc, w.Activities.RollbackCloneGoldCopyProducts, provisioning.CloneProductsInput{
					GoldCopyTenantID:   input.GoldCopyTenantID,
					GoldCopyInstanceID: input.GoldCopyInstanceID,
					TargetTenantID:    input.TenantID,
					TargetInstanceID: input.InstanceID,
				}).Get(dc, nil)
			}
			if step4Done {
				workflow.ExecuteActivity(dc, w.Activities.RollbackCreateLakekeeperNamespace, input.LakekeeperNS).Get(dc, nil)
			}
			if step3Done {
				workflow.ExecuteActivity(dc, w.Activities.CloneSchemaFromGoldCopy, provisioning.CloneSchemaInput{
					SourceDatabase: input.GoldCopyDatabase,
					TargetDatabase: input.DatabaseName,
				}).Get(dc, nil)
			}
			if step2Done {
				workflow.ExecuteActivity(dc, w.Activities.RollbackRegisterInstance, registeredInstanceID).Get(dc, nil)
			}
			if step1Done {
				workflow.ExecuteActivity(dc, w.Activities.RollbackRegisterTenant, registeredTenantID).Get(dc, nil)
			}
			result.Status = "rolled_back"
		}
	}()

	step1Err := workflow.ExecuteActivity(ctx, w.Activities.RegisterTenant, provisioning.RegisterTenantInput{
		TenantID:   input.TenantID,
		TenantName: input.TenantName,
		TenantCode: input.TenantCode,
	}).Get(ctx, &registeredTenantID)
	if step1Err != nil {
		logger.Error("Step 1 failed: RegisterTenant", "error", step1Err)
		result.Error = step1Err.Error()
		return result, fmt.Errorf("saga failed at step 1 (RegisterTenant): %w", step1Err)
	}
	step1Done = true
	input.TenantID = registeredTenantID

	step2Err := workflow.ExecuteActivity(ctx, w.Activities.RegisterInstance, provisioning.RegisterInstanceInput{
		TenantID:     input.TenantID,
		InstanceID:   input.InstanceID,
		InstanceName: input.InstanceName,
	}).Get(ctx, &registeredInstanceID)
	if step2Err != nil {
		logger.Error("Step 2 failed: RegisterInstance", "error", step2Err)
		result.Error = step2Err.Error()
		return result, fmt.Errorf("saga failed at step 2 (RegisterInstance): %w", step2Err)
	}
	step2Done = true
	input.InstanceID = registeredInstanceID

	step3Err := workflow.ExecuteActivity(ctx, w.Activities.CreateTenantDatabase, input.DatabaseName).Get(ctx, nil)
	if step3Err != nil {
		logger.Error("Step 3 failed: CreateTenantDatabase", "error", step3Err)
		result.Error = step3Err.Error()
		return result, fmt.Errorf("saga failed at step 3 (CreateTenantDatabase): %w", step3Err)
	}
	step3Done = true

	step4Err := workflow.ExecuteActivity(ctx, w.Activities.CloneSchemaFromGoldCopy, provisioning.CloneSchemaInput{
		SourceDatabase: input.GoldCopyDatabase,
		TargetDatabase: input.DatabaseName,
	}).Get(ctx, nil)
	if step4Err != nil {
		logger.Error("Step 4 failed: CloneSchemaFromGoldCopy", "error", step4Err)
		result.Error = step4Err.Error()
		return result, fmt.Errorf("saga failed at step 4 (CloneSchemaFromGoldCopy): %w", step4Err)
	}
	step4Done = true

	step5Err := workflow.ExecuteActivity(ctx, w.Activities.CreateLakekeeperNamespace, input.LakekeeperNS).Get(ctx, nil)
	if step5Err != nil {
		logger.Error("Step 5 failed: CreateLakekeeperNamespace", "error", step5Err)
		result.Error = step5Err.Error()
		return result, fmt.Errorf("saga failed at step 5 (CreateLakekeeperNamespace): %w", step5Err)
	}
	step5Done = true

	step6Err := workflow.ExecuteActivity(ctx, w.Activities.CloneGoldCopyProducts, provisioning.CloneProductsInput{
		GoldCopyTenantID:   input.GoldCopyTenantID,
		GoldCopyInstanceID: input.GoldCopyInstanceID,
		TargetTenantID:    input.TenantID,
		TargetInstanceID: input.InstanceID,
	}).Get(ctx, nil)
	if step6Err != nil {
		logger.Error("Step 6 failed: CloneGoldCopyProducts", "error", step6Err)
		result.Error = step6Err.Error()
		return result, fmt.Errorf("saga failed at step 6 (CloneGoldCopyProducts): %w", step6Err)
	}
	step5Done = true

	_ = workflow.ExecuteActivity(ctx, w.Activities.UpdateTenantStatus, input.TenantID, "active").Get(ctx, nil)
	_ = workflow.ExecuteActivity(ctx, w.Activities.UpdateInstanceStatus, input.InstanceID, "active").Get(ctx, nil)

	_ = workflow.ExecuteActivity(ctx, w.Activities.EmitProvisioningEvent, provisioning.EmitEventInput{
		TenantID:     input.TenantID,
		InstanceID:   input.InstanceID,
		DatabaseName: input.DatabaseName,
		Status:       "completed",
		CompletedAt:  time.Now(),
	}).Get(ctx, nil)

	result.Status = "completed"
	result.CompletedAt = time.Now()
	logger.Info("TenantInstanceProvisioningWorkflow completed successfully",
		"tenantID", input.TenantID,
		"instanceID", input.InstanceID,
		"databaseName", input.DatabaseName)

	return result, nil
}

// sagaCompensationVersion is the workflow.GetVersion change id for the
// compensation fix below. Runs that started before it was deployed replay on the
// legacy path; everything started after takes the new one.
const sagaCompensationVersion = "saga-compensation-v2"

// sagaTenantDatabaseVersion gates the tenant-database steps (role and credential, datasource
// binding, tenant migrations, probe). They also require ProvisioningWorkflowInput.App, so a
// request that does not name an app takes exactly the path it took before.
const sagaTenantDatabaseVersion = "saga-tenant-database-v1"

// sagaTenantStructureVersion gates the compiled-structure path (ADR-050): the tenant's structure is compiled from what
// alpha holds after the gold copy's scan instead of being cloned from the gold copy's database. It also requires App and
// TemplateDatasourceID, so a request that names no template takes exactly the path it took before.
const sagaTenantStructureVersion = "saga-tenant-structure-v1"

// sagaProductPathVersion gates the product path ("create tenant X in region R with product P and label L"): the region's
// cluster is checked before anything is created, the database is created through the cluster administrator the worker holds,
// only the chosen products are registered, and the app's reference rows are seeded. It also requires
// ProvisioningWorkflowInput.Region, which only the product path sets, so every other run takes exactly the path it took before.
const sagaProductPathVersion = "saga-product-path-v1"

// TenantInstanceProvisioningWorkflowFn is the registered provisioning saga.
func TenantInstanceProvisioningWorkflowFn(ctx workflow.Context, input provisioning.ProvisioningWorkflowInput) (*provisioning.ProvisioningWorkflowResult, error) {
	if workflow.GetVersion(ctx, sagaCompensationVersion, workflow.DefaultVersion, 1) == workflow.DefaultVersion {
		return tenantInstanceProvisioningLegacy(ctx, input)
	}
	return tenantInstanceProvisioning(ctx, input)
}

// saga compensation: what a failed or cancelled run undoes, and why it is gated.
//
// The old workflow compensated only on context cancellation (never when a step
// failed), ran the wrong compensation for several steps, and never dropped the
// tenant database. Naively fixing that would be dangerous: RegisterTenant and
// RegisterInstance are upserts that can return an EXISTING row, and
// CreateTenantDatabase / CreateLakekeeperNamespace treat "already exists" as
// success, so a failed run can hold a reference to something it did not create.
// Compensating blindly would delete a live tenant.
//
// So compensation is limited to what this run provably owns:
//   - tenant and instance rows: their rollbacks only delete rows still in status
//     'provisioning' (enforced in SQL), so they are always safe to call;
//   - database, namespace and cloned products: undone only if
//     InspectProvisioningState, taken before anything was created, showed both
//     the tenant and the instance still 'provisioning' (and, for the database,
//     that it did not already exist).
type compensation struct {
	name string
	run  func(workflow.Context) error
}

func tenantInstanceProvisioning(ctx workflow.Context, input provisioning.ProvisioningWorkflowInput) (res *provisioning.ProvisioningWorkflowResult, err error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("TenantInstanceProvisioningWorkflowFn started",
		"tenant", input.TenantName, "tenantCode", input.TenantCode, "instance", input.InstanceName)

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			MaximumAttempts:    3,
			BackoffCoefficient: 2.0,
			InitialInterval:    10 * time.Second,
		},
	})

	result := &provisioning.ProvisioningWorkflowResult{
		TenantID:     input.TenantID,
		InstanceID:   input.InstanceID,
		DatabaseName: input.DatabaseName,
		LakekeeperNS: input.LakekeeperNS,
		Status:       "provisioning",
	}
	res = result

	// acts is only a source of method references; Temporal dispatches by name to
	// the instance registered on the worker.
	acts := &activities.TenantProvisioningActivities{}

	var comps []compensation
	defer func() {
		if err == nil && ctx.Err() == nil {
			return
		}
		// A disconnected context so compensation still runs when the workflow
		// was cancelled.
		dc, _ := workflow.NewDisconnectedContext(ctx)
		dc = workflow.WithActivityOptions(dc, workflow.ActivityOptions{
			StartToCloseTimeout: 5 * time.Minute,
			RetryPolicy:         &sdktemporal.RetryPolicy{MaximumAttempts: 5, InitialInterval: 5 * time.Second, BackoffCoefficient: 2.0},
		})

		incomplete := false
		for i := len(comps) - 1; i >= 0; i-- {
			if cerr := comps[i].run(dc); cerr != nil {
				// Keep going: one stuck compensation must not strand the rest.
				incomplete = true
				logger.Error("Compensation failed", "step", comps[i].name, "error", cerr)
			}
		}

		failure := ""
		if err != nil {
			failure = err.Error()
		} else if cerr := ctx.Err(); cerr != nil {
			failure = cerr.Error()
		}
		_ = workflow.ExecuteActivity(dc, acts.EmitProvisioningEvent, provisioning.EmitEventInput{
			TenantID:     input.TenantID,
			InstanceID:   input.InstanceID,
			DatabaseName: input.DatabaseName,
			Status:       "failed",
			Error:        failure,
			CompletedAt:  workflow.Now(ctx),
		}).Get(dc, nil)

		result.Status = "rolled_back"
		if incomplete {
			result.Status = "rollback_incomplete"
		}
		if result.Error == "" {
			result.Error = failure
		}
	}()

	fail := func(step string, cause error) (*provisioning.ProvisioningWorkflowResult, error) {
		logger.Error("Saga step failed", "step", step, "error", cause)
		result.Error = cause.Error()
		return result, fmt.Errorf("saga failed at %s: %w", step, cause)
	}

	// 1. Tenant row.
	var tenantID string
	if e := workflow.ExecuteActivity(ctx, acts.RegisterTenant, provisioning.RegisterTenantInput{
		TenantID:   input.TenantID,
		TenantName: input.TenantName,
		TenantCode: input.TenantCode,
		Region:     input.Region,
	}).Get(ctx, &tenantID); e != nil {
		return fail("RegisterTenant", e)
	}
	input.TenantID = tenantID
	result.TenantID = tenantID
	// inspected/state are read when a compensation runs. Before inspection the
	// row's status is unknown here, and the rollback's own SQL guard covers it.
	var (
		state     provisioning.ProvisioningState
		inspected bool
	)
	comps = append(comps, compensation{"RollbackRegisterTenant", func(c workflow.Context) error {
		if inspected && !state.TenantOwned {
			return nil
		}
		return workflow.ExecuteActivity(c, acts.RollbackRegisterTenant, tenantID).Get(c, nil)
	}})

	// 2. Instance row.
	var instanceID string
	if e := workflow.ExecuteActivity(ctx, acts.RegisterInstance, provisioning.RegisterInstanceInput{
		TenantID:     input.TenantID,
		InstanceID:   input.InstanceID,
		InstanceName: input.InstanceName,
	}).Get(ctx, &instanceID); e != nil {
		return fail("RegisterInstance", e)
	}
	input.InstanceID = instanceID
	result.InstanceID = instanceID
	comps = append(comps, compensation{"RollbackRegisterInstance", func(c workflow.Context) error {
		if inspected && !state.InstanceOwned {
			return nil
		}
		return workflow.ExecuteActivity(c, acts.RollbackRegisterInstance, instanceID).Get(c, nil)
	}})

	// 3. Record what this run owns, before creating anything.
	if e := workflow.ExecuteActivity(ctx, acts.InspectProvisioningState, input.TenantID, input.InstanceID, input.DatabaseName).Get(ctx, &state); e != nil {
		return fail("InspectProvisioningState", e)
	}
	inspected = true
	if !state.Owned() {
		logger.Warn("Provisioning an existing tenant or instance; its database, namespace and products will not be rolled back",
			"tenantOwned", state.TenantOwned, "instanceOwned", state.InstanceOwned)
	}

	// 3b. Plan the structure (ADR-050), before anything is created: a template that cannot be deployed (an incomplete or
	// unsupported scan, or a datasource that is not the gold copy's) refuses the run here, with nothing but the two rows
	// above to undo.
	structureMode := input.App != "" && (input.TemplateDatasourceID != "" || input.StructureFromGoldCopy) &&
		workflow.GetVersion(ctx, sagaTenantStructureVersion, workflow.DefaultVersion, 1) == 1
	var structure provisioning.StructurePlan
	templateID := input.TemplateDatasourceID
	if structureMode {
		planIn := provisioning.TenantDatabaseInput{TenantID: input.TenantID, InstanceID: input.InstanceID, App: input.App,
			DatabaseName: input.DatabaseName, TemplateDatasourceID: templateID}
		if templateID == "" {
			// The template is whatever the gold copy marks for this app; zero or several refuse the run here.
			if e := workflow.ExecuteActivity(ctx, acts.ResolveStructureTemplate, planIn).Get(ctx, &templateID); e != nil {
				return fail("ResolveStructureTemplate", e)
			}
			planIn.TemplateDatasourceID = templateID
		}
		if e := workflow.ExecuteActivity(ctx, acts.PlanTenantStructure, planIn).Get(ctx, &structure); e != nil {
			return fail("PlanTenantStructure", e)
		}
		logger.Info("Tenant structure planned", "template", structure.TemplateDatasourceID, "hash", structure.Hash, "tables", structure.Tables, "statements", structure.Statements)
	}

	// 3c. The product path: the region's cluster must be the one this worker administers, and the database name must be
	// free there. Both are refused here, before anything is created.
	productPath := input.Region != "" && input.App != "" &&
		workflow.GetVersion(ctx, sagaProductPathVersion, workflow.DefaultVersion, 1) == 1
	regionDB := provisioning.RegionDatabaseInput{
		Region: input.Region, Host: input.ClusterHost, Port: input.ClusterPort, DatabaseName: input.DatabaseName,
	}
	if productPath {
		if e := workflow.ExecuteActivity(ctx, acts.AssertRegionCluster, regionDB).Get(ctx, nil); e != nil {
			return fail("AssertRegionCluster", e)
		}
	}

	// 4. Tenant database.
	if productPath {
		if e := workflow.ExecuteActivity(ctx, acts.CreateTenantDatabaseInRegion, regionDB).Get(ctx, nil); e != nil {
			return fail("CreateTenantDatabaseInRegion", e)
		}
		// AssertRegionCluster proved the name was free, so this run made the database.
		comps = append(comps, compensation{"RollbackCreateTenantDatabaseInRegion", func(c workflow.Context) error {
			return workflow.ExecuteActivity(c, acts.RollbackCreateTenantDatabaseInRegion, regionDB).Get(c, nil)
		}})
	} else {
		if e := workflow.ExecuteActivity(ctx, acts.CreateTenantDatabase, input.DatabaseName).Get(ctx, nil); e != nil {
			return fail("CreateTenantDatabase", e)
		}
		if state.Owned() && !state.DatabaseExisted {
			comps = append(comps, compensation{"RollbackCreateTenantDatabase", func(c workflow.Context) error {
				return workflow.ExecuteActivity(c, acts.RollbackCreateTenantDatabase, input.DatabaseName).Get(c, nil)
			}})
		}
	}

	// 5. Schema. Undone by dropping the database above, so no compensation of its own. In structure mode there is no
	// clone: the structure is applied in step 8, from alpha's scan, and the gold copy's database is never read.
	if !structureMode {
		if e := workflow.ExecuteActivity(ctx, acts.CloneSchemaFromGoldCopy, provisioning.CloneSchemaInput{
			SourceDatabase: input.GoldCopyDatabase,
			TargetDatabase: input.DatabaseName,
		}).Get(ctx, nil); e != nil {
			return fail("CloneSchemaFromGoldCopy", e)
		}
	}

	// 6. Lakekeeper namespace.
	if e := workflow.ExecuteActivity(ctx, acts.CreateLakekeeperNamespace, input.LakekeeperNS).Get(ctx, nil); e != nil {
		return fail("CreateLakekeeperNamespace", e)
	}
	if state.Owned() {
		comps = append(comps, compensation{"RollbackCreateLakekeeperNamespace", func(c workflow.Context) error {
			return workflow.ExecuteActivity(c, acts.RollbackCreateLakekeeperNamespace, input.LakekeeperNS).Get(c, nil)
		}})
	}

	// 7. Gold-copy products.
	cloneIn := provisioning.CloneProductsInput{
		GoldCopyTenantID:   input.GoldCopyTenantID,
		GoldCopyInstanceID: input.GoldCopyInstanceID,
		TargetTenantID:     input.TenantID,
		TargetInstanceID:   input.InstanceID,
		ProductCodes:       input.ProductCodes,
	}
	if e := workflow.ExecuteActivity(ctx, acts.CloneGoldCopyProducts, cloneIn).Get(ctx, nil); e != nil {
		return fail("CloneGoldCopyProducts", e)
	}
	if state.Owned() {
		comps = append(comps, compensation{"RollbackCloneGoldCopyProducts", func(c workflow.Context) error {
			return workflow.ExecuteActivity(c, acts.RollbackCloneGoldCopyProducts, cloneIn).Get(c, nil)
		}})
	}

	// 8. Tenant database (ADR-030): its own role and credential, the app's datasource repointed
	// at it, the app's migrations applied, and a probe through tenantdb, all before the binding
	// goes active. A failure here is fatal and compensates: unlike the status writes below, an
	// unreachable tenant database is not a tenant that is "otherwise fully provisioned".
	if input.App != "" && workflow.GetVersion(ctx, sagaTenantDatabaseVersion, workflow.DefaultVersion, 1) == 1 {
		tdIn := provisioning.TenantDatabaseInput{
			TenantID:         input.TenantID,
			InstanceID:       input.InstanceID,
			App:              input.App,
			DatabaseName:     input.DatabaseName,
			GoldCopyDatabase: input.GoldCopyDatabase,
			BaselineThrough:  input.BaselineThrough,
			TemplateDatasourceID: templateID,
			StructureHash:        structure.Hash,
		}
		var bound provisioning.TenantDatabaseBinding
		if e := workflow.ExecuteActivity(ctx, acts.BindTenantDatabase, tdIn).Get(ctx, &bound); e != nil {
			return fail("BindTenantDatabase", e)
		}
		tdIn.DatasourceID = bound.DatasourceID
		if state.Owned() {
			rollbackIn := tdIn
			comps = append(comps, compensation{"RollbackTenantDatabase", func(c workflow.Context) error {
				return workflow.ExecuteActivity(c, acts.RollbackTenantDatabase, rollbackIn).Get(c, nil)
			}})
		}
		if e := workflow.ExecuteActivity(ctx, acts.ProvisionTenantDatabaseAccess, tdIn).Get(ctx, nil); e != nil {
			return fail("ProvisionTenantDatabaseAccess", e)
		}
		var report migrations.Report
		if structureMode {
			if e := workflow.ExecuteActivity(ctx, acts.ApplyTenantStructure, tdIn).Get(ctx, &report); e != nil {
				return fail("ApplyTenantStructure", e)
			}
			if !report.Done {
				return fail("ApplyTenantStructure", fmt.Errorf("structure report for %s is not done", report.Target))
			}
			// The app's reference rows, after its structure and before the probe: a tenant is not active without them.
			if productPath && input.Seed {
				if e := workflow.ExecuteActivity(ctx, acts.SeedTenantDatabase, tdIn).Get(ctx, nil); e != nil {
					return fail("SeedTenantDatabase", e)
				}
			}
		} else {
			if e := workflow.ExecuteActivity(ctx, acts.ApplyTenantMigrations, tdIn).Get(ctx, &report); e != nil {
				return fail("ApplyTenantMigrations", e)
			}
			if !report.Done {
				return fail("ApplyTenantMigrations", fmt.Errorf("migration report for %s is not done", report.Target))
			}
		}
		if e := workflow.ExecuteActivity(ctx, acts.ProbeTenantDatabase, tdIn).Get(ctx, nil); e != nil {
			return fail("ProbeTenantDatabase", e)
		}
		if e := workflow.ExecuteActivity(ctx, acts.ActivateTenantDatabase, tdIn).Get(ctx, nil); e != nil {
			return fail("ActivateTenantDatabase", e)
		}
	}

	// Activation. Unchanged behaviour: a failure here is logged, not fatal.
	// Failing it would compensate (drop the database) a tenant that is otherwise
	// fully provisioned over a transient status write.
	if e := workflow.ExecuteActivity(ctx, acts.UpdateTenantStatus, input.TenantID, "active").Get(ctx, nil); e != nil {
		logger.Error("Could not mark tenant active; it remains in 'provisioning'", "error", e)
	}
	if e := workflow.ExecuteActivity(ctx, acts.UpdateInstanceStatus, input.InstanceID, "active").Get(ctx, nil); e != nil {
		logger.Error("Could not mark instance active; it remains in 'provisioning'", "error", e)
	}

	_ = workflow.ExecuteActivity(ctx, acts.EmitProvisioningEvent, provisioning.EmitEventInput{
		TenantID:     input.TenantID,
		InstanceID:   input.InstanceID,
		DatabaseName: input.DatabaseName,
		Status:       "completed",
		CompletedAt:  workflow.Now(ctx),
	}).Get(ctx, nil)

	result.Status = "completed"
	result.CompletedAt = workflow.Now(ctx)
	logger.Info("TenantInstanceProvisioningWorkflowFn completed", "tenantID", input.TenantID, "instanceID", input.InstanceID)
	return result, nil
}

// tenantInstanceProvisioningLegacy is the pre-fix saga, kept verbatim ONLY so
// that runs already in flight when sagaCompensationVersion shipped still replay
// deterministically. Its compensation is wrong (see the note above
// tenantInstanceProvisioning); do not use it for new work. It can be deleted once
// no run older than the workflow execution timeout (15 minutes) remains.
func tenantInstanceProvisioningLegacy(ctx workflow.Context, input provisioning.ProvisioningWorkflowInput) (*provisioning.ProvisioningWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("TenantInstanceProvisioningWorkflowFn started",
		"tenant", input.TenantName,
		"tenantCode", input.TenantCode,
		"instance", input.InstanceName)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			MaximumAttempts:    3,
			BackoffCoefficient: 2.0,
			InitialInterval:    10 * time.Second,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := &provisioning.ProvisioningWorkflowResult{
		TenantID:     input.TenantID,
		InstanceID:   input.InstanceID,
		DatabaseName: input.DatabaseName,
		LakekeeperNS: input.LakekeeperNS,
		Status:       "provisioning",
	}

	step1Done, step2Done, step3Done, step4Done := false, false, false, false
	var tenantID, instanceID string

	activities := &activities.TenantProvisioningActivities{}

	defer func() {
		if ctx.Err() != nil {
			dc, _ := workflow.NewDisconnectedContext(ctx)
			if step4Done {
				workflow.ExecuteActivity(dc, activities.RollbackCloneGoldCopyProducts, provisioning.CloneProductsInput{
					GoldCopyTenantID:   input.GoldCopyTenantID,
					GoldCopyInstanceID: input.GoldCopyInstanceID,
					TargetTenantID:    input.TenantID,
					TargetInstanceID: input.InstanceID,
				}).Get(dc, nil)
			}
			if step3Done {
				workflow.ExecuteActivity(dc, activities.RollbackCreateLakekeeperNamespace, input.LakekeeperNS).Get(dc, nil)
			}
			if step2Done {
				workflow.ExecuteActivity(dc, activities.RollbackRegisterInstance, instanceID).Get(dc, nil)
			}
			if step1Done {
				workflow.ExecuteActivity(dc, activities.RollbackRegisterTenant, tenantID).Get(dc, nil)
			}
			result.Status = "rolled_back"
		}
	}()

	if err := workflow.ExecuteActivity(ctx, activities.RegisterTenant, provisioning.RegisterTenantInput{
		TenantID:   input.TenantID,
		TenantName: input.TenantName,
		TenantCode: input.TenantCode,
	}).Get(ctx, &tenantID); err != nil {
		logger.Error("Step 1 failed", "error", err)
		result.Error = err.Error()
		return result, fmt.Errorf("RegisterTenant failed: %w", err)
	}
	step1Done = true
	input.TenantID = tenantID

	if err := workflow.ExecuteActivity(ctx, activities.RegisterInstance, provisioning.RegisterInstanceInput{
		TenantID:     input.TenantID,
		InstanceID:   input.InstanceID,
		InstanceName: input.InstanceName,
	}).Get(ctx, &instanceID); err != nil {
		logger.Error("Step 2 failed", "error", err)
		result.Error = err.Error()
		return result, fmt.Errorf("RegisterInstance failed: %w", err)
	}
	step2Done = true
	input.InstanceID = instanceID

	if err := workflow.ExecuteActivity(ctx, activities.CreateTenantDatabase, input.DatabaseName).Get(ctx, nil); err != nil {
		logger.Error("Step 3 failed", "error", err)
		result.Error = err.Error()
		return result, fmt.Errorf("CreateTenantDatabase failed: %w", err)
	}
	step3Done = true

	if err := workflow.ExecuteActivity(ctx, activities.CloneSchemaFromGoldCopy, provisioning.CloneSchemaInput{
		SourceDatabase: input.GoldCopyDatabase,
		TargetDatabase: input.DatabaseName,
	}).Get(ctx, nil); err != nil {
		logger.Error("Step 4 failed", "error", err)
		result.Error = err.Error()
		return result, fmt.Errorf("CloneSchemaFromGoldCopy failed: %w", err)
	}
	step4Done = true

	if err := workflow.ExecuteActivity(ctx, activities.CreateLakekeeperNamespace, input.LakekeeperNS).Get(ctx, nil); err != nil {
		logger.Error("Step 5 failed", "error", err)
		result.Error = err.Error()
		return result, fmt.Errorf("CreateLakekeeperNamespace failed: %w", err)
	}

	if err := workflow.ExecuteActivity(ctx, activities.CloneGoldCopyProducts, provisioning.CloneProductsInput{
		GoldCopyTenantID:   input.GoldCopyTenantID,
		GoldCopyInstanceID: input.GoldCopyInstanceID,
		TargetTenantID:    input.TenantID,
		TargetInstanceID: input.InstanceID,
	}).Get(ctx, nil); err != nil {
		logger.Error("Step 6 failed", "error", err)
		result.Error = err.Error()
		return result, fmt.Errorf("CloneGoldCopyProducts failed: %w", err)
	}

	_ = workflow.ExecuteActivity(ctx, activities.UpdateTenantStatus, input.TenantID, "active").Get(ctx, nil)
	_ = workflow.ExecuteActivity(ctx, activities.UpdateInstanceStatus, input.InstanceID, "active").Get(ctx, nil)

	_ = workflow.ExecuteActivity(ctx, activities.EmitProvisioningEvent, provisioning.EmitEventInput{
		TenantID:     input.TenantID,
		InstanceID:   input.InstanceID,
		DatabaseName: input.DatabaseName,
		Status:       "completed",
		CompletedAt:  time.Now(),
	}).Get(ctx, nil)

	result.Status = "completed"
	result.CompletedAt = time.Now()
	logger.Info("TenantInstanceProvisioningWorkflow completed",
		"tenantID", input.TenantID,
		"instanceID", input.InstanceID)

	return result, nil
}
