//go:build ignore
// +build ignore

// Package workflows contains Temporal workflows used by the backend.
package workflows

import (
	"time"

	uuid "github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/temporal/activities"
	"go.temporal.io/sdk/workflow"
)

// Onboarding is a small example workflow that calls an activity to record a row
// in Hasura. This version is the buildable variant used during development.
func Onboarding(ctx workflow.Context, id string) error {
	ao := workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second}
	ctx = workflow.WithActivityOptions(ctx, ao)

	info := workflow.GetInfo(ctx)
	payload := map[string]any{
		"workflow_id": info.WorkflowExecution.ID,
		"status":      "completed",
		"tenant_id":   info.WorkflowExecution.RunID,
		"id":          uuid.New().String(),
	}

	return workflow.ExecuteActivity(ctx, activities.Record, payload).Get(ctx, nil)
}
