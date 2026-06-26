package workflows

import (
	"context"
	"fmt"
	"time"

	"calendar-service/internal/temporal/activities"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// CalendarChangedSignal is sent when calendar state changes
type CalendarChangedSignal struct {
	EntityType string // "calendar", "schedule_profile", "blackout"
	EntityID   string
	TenantID   string
	Action     string // "create", "update", "delete"
	Timestamp  time.Time
}

// RescheduleRequest is sent to the reschedule activity
type RescheduleRequest struct {
	JobID     string
	TenantID  string
	ProfileID string
	NewTime   time.Time
}

// CalendarChangedWorkflow processes calendar changes and reschedules affected jobs
func CalendarChangedWorkflow(ctx workflow.Context, signal CalendarChangedSignal) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("CalendarChangedWorkflow started",
		"entityType", signal.EntityType,
		"entityID", signal.EntityID,
		"action", signal.Action,
	)

	// Set up options
	opts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Minute,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, opts)

	// Step 1: Fetch affected jobs
	var affectedJobs []struct {
		JobID     string
		ProfileID string
		NextRun   time.Time
	}
	err := workflow.ExecuteActivity(ctx, "FetchAffectedJobsActivity", activities.FetchAffectedJobsRequest{
		EntityType: signal.EntityType,
		EntityID:   signal.EntityID,
		TenantID:   signal.TenantID,
	}).Get(ctx, &affectedJobs)
	if err != nil {
		logger.Error("Failed to fetch affected jobs", "error", err)
		return err
	}

	logger.Info("Fetched affected jobs", "count", len(affectedJobs))

	// Step 2: For each affected job, resolve calendar and check availability
	for _, job := range affectedJobs {
		logger.Info("Processing job", "jobID", job.JobID, "profileID", job.ProfileID)

		// Check if job's next_run is still available
		var available bool
		err := workflow.ExecuteActivity(ctx, "CheckAvailabilityActivity", activities.CheckAvailabilityRequest{
			TenantID:  signal.TenantID,
			ProfileID: job.ProfileID,
			Start:     job.NextRun,
			End:       job.NextRun.Add(1 * time.Hour),
		}).Get(ctx, &available)
		if err != nil {
			logger.Warn("Failed to check availability", "error", err, "jobID", job.JobID)
			continue
		}

		if !available {
			logger.Info("Job slot no longer available, finding next slot", "jobID", job.JobID)

			// Find next available slot
			var nextSlot time.Time
			err := workflow.ExecuteActivity(ctx, "FindNextSlotActivity", activities.FindNextSlotRequest{
				TenantID:  signal.TenantID,
				ProfileID: job.ProfileID,
				After:     job.NextRun,
				Duration:  1 * time.Hour,
			}).Get(ctx, &nextSlot)
			if err != nil {
				logger.Error("Failed to find next available slot", "error", err, "jobID", job.JobID)
				continue
			}

			// Reschedule the job
			err = workflow.ExecuteActivity(ctx, "RescheduleJobActivity", RescheduleRequest{
				JobID:     job.JobID,
				TenantID:  signal.TenantID,
				ProfileID: job.ProfileID,
				NewTime:   nextSlot,
			}).Get(ctx, nil)

			if err != nil {
				logger.Error("Failed to reschedule job", "error", err, "jobID", job.JobID, "newTime", nextSlot)
				continue
			}

			logger.Info("Successfully rescheduled job", "jobID", job.JobID, "newTime", nextSlot)
		}
	}

	logger.Info("CalendarChangedWorkflow completed")
	return nil
}

// ListenForCalendarChanges is a long-running workflow that listens for calendar change signals
func ListenForCalendarChanges(ctx workflow.Context, tenantID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("ListenForCalendarChanges started", "tenantID", tenantID)

	// Channel to receive signals
	signalChan := workflow.GetSignalChannel(ctx, "calendar-changed")

	// Long-running loop
	for {
		var signal CalendarChangedSignal
		signalChan.Receive(ctx, &signal)

		logger.Info("Received calendar change signal",
			"entityType", signal.EntityType,
			"entityID", signal.EntityID,
			"action", signal.Action,
		)

		// Execute child workflow to handle the change
		childWorkflowOptions := workflow.ChildWorkflowOptions{
			WorkflowExecutionTimeout: 30 * time.Minute,
		}
		childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

		err := workflow.ExecuteChildWorkflow(childCtx, CalendarChangedWorkflow, signal).Get(childCtx, nil)
		if err != nil {
			logger.Error("Child workflow failed", "error", err)
			// Continue listening despite errors
		}
	}
}

// SignalCalendarChange sends a calendar change signal to the listening workflow
func SignalCalendarChange(ctx context.Context, workflowClient interface{}, tenantID string, signal CalendarChangedSignal) error {
	// This would be called from your CDC consumer or mutation handlers
	// Implementation depends on your Temporal client setup
	return fmt.Errorf("signal implementation depends on temporal client type")
}
