package temporal

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func LakehouseMaintenanceWorkflow(ctx workflow.Context, catalogName string) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 20 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var act *TenantActivities

	// Task 1: Expire old Iceberg snapshots
	err := workflow.ExecuteActivity(ctx, act.ExpireIcebergSnapshots, catalogName).Get(ctx, nil)
	if err != nil {
		return err
	}

	// Task 2: Remove orphan files from object storage
	err = workflow.ExecuteActivity(ctx, act.RemoveOrphanFiles, catalogName).Get(ctx, nil)
	if err != nil {
		return err
	}

	// Task 3: Compact small manifest files
	err = workflow.ExecuteActivity(ctx, act.CompactManifests, catalogName).Get(ctx, nil)
	return err
}
