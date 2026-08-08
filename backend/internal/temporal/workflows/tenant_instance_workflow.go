package workflows

import (
	"fmt"
	"time"

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

func TenantInstanceProvisioningWorkflowFn(ctx workflow.Context, input provisioning.ProvisioningWorkflowInput) (*provisioning.ProvisioningWorkflowResult, error) {
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
