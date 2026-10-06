package querybuilder

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// CubeReconcileWorkflowName is the registered Temporal workflow type (CUBE-1.4).
const CubeReconcileWorkflowName = "CubeReconcileWorkflow"

// ActCubeReconcile is the activity that runs one reconcile pass.
const ActCubeReconcile = "CubeReconcile"

// CubeReconcileWorkflowID is the single-flight ID for a reconcile pass.
const CubeReconcileWorkflowID = "cube-reconcile"

// CubeReconcileStartOptions starts CubeReconcileWorkflow with REJECT-style single flight
// via a fixed workflow ID on uisce-cubes.
func CubeReconcileStartOptions() client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		ID:        CubeReconcileWorkflowID,
		TaskQueue: CubeTaskQueue,
	}
}

// CubeReconcileWorkflow runs one Active-vs-physical reconcile pass and returns the receipt.
func CubeReconcileWorkflow(ctx workflow.Context) (*CubeReconcileReceipt, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("CubeReconcileWorkflow started")

	ao := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    2,
		},
	})

	var receipt CubeReconcileReceipt
	if err := workflow.ExecuteActivity(ao, ActCubeReconcile).Get(ctx, &receipt); err != nil {
		return nil, err
	}
	logger.Info("CubeReconcileWorkflow finished",
		"examined", receipt.Examined,
		"actions", len(receipt.Actions),
		"errors", len(receipt.Errors),
	)
	return &receipt, nil
}

// CubeReconcileActivities wraps CubeReconciler for Temporal.
type CubeReconcileActivities struct {
	Reconciler *CubeReconciler
}

// NewCubeReconcileActivities builds reconcile activities.
func NewCubeReconcileActivities(r *CubeReconciler) *CubeReconcileActivities {
	return &CubeReconcileActivities{Reconciler: r}
}

// CubeReconcile runs Reconcile and returns the receipt.
func (a *CubeReconcileActivities) CubeReconcile(ctx context.Context) (*CubeReconcileReceipt, error) {
	if a == nil || a.Reconciler == nil {
		return nil, fmt.Errorf("cube reconcile activities: not configured")
	}
	return a.Reconciler.Reconcile(ctx)
}
