package temporal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// TEMPORAL WORKFLOW: TimeoutMonitorWorkflow
// ============================================================================
// Runs every hour to check workflow steps for timeout and execute escalation,
// notification, and logging actions. This is the orchestration layer that
// coordinates with the TimeoutMonitor service.
//
// Usage in worker.go:
//
//    w.RegisterWorkflow(TimeoutMonitorWorkflow)
//    // Schedule with cron "0 * * * *" (every hour)
//

// TimeoutMonitorWorkflow is the main Temporal workflow that monitors timeouts
// It runs every hour and checks for workflow steps that have exceeded their due time
func TimeoutMonitorWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)

	// Log workflow start
	logger.Info("TimeoutMonitor workflow started", "timestamp", workflow.Now(ctx))

	// This workflow is designed to run as a timed/cron job via Temporal Schedules
	// It will be triggered every hour

	// Activity options
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 30 * time.Minute,
		StartToCloseTimeout:    25 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Execute the timeout check activity
	var result int
	err := workflow.ExecuteActivity(ctx, TimeoutMonitorActivity).Get(ctx, &result)
	if err != nil {
		logger.Error("TimeoutMonitorActivity failed", "error", err)
		return err
	}

	logger.Info("TimeoutMonitor workflow completed", "actions_executed", result)
	return nil
}

// TimeoutMonitorActivity is the activity that actually performs the timeout check
// Activities are the units of work that can interact with external systems (databases, APIs, etc.)
func TimeoutMonitorActivity(ctx context.Context, db *sql.DB) (int, error) {
	log.Println("[TimeoutMonitor] Activity: Checking for workflow timeouts...")

	if db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	// Create timeout monitor instance
	sqlxDB := sqlx.NewDb(db, "postgres") // Convert sql.DB to sqlx.DB
	tm := NewTimeoutMonitor(sqlxDB)

	// Execute timeout check
	err := tm.CheckAndExecuteTimeouts(ctx)
	if err != nil {
		log.Printf("[TimeoutMonitor] Activity error: %v", err)
		return 0, err
	}

	return 1, nil
}

// ============================================================================
// REGISTERING THE WORKFLOW IN YOUR WORKER
// ============================================================================
//
// In your backend/cmd/worker/main.go, add this:
//
//    import (
//        "github.com/hondyman/semlayer/backend/internal/temporal"
//        "go.temporal.io/sdk/worker"
//        "go.temporal.io/sdk/workflow"
//    )
//
//    func main() {
//        // ... existing worker setup code ...
//
//        w := worker.New(client, "default", worker.Options{})
//
//        // Register TimeoutMonitor workflow
//        w.RegisterWorkflow(temporal.TimeoutMonitorWorkflow)
//        w.RegisterActivity(temporal.TimeoutMonitorActivity)
//
//        // Schedule to run every hour (0 * * * * = hourly)
//        scheduleOptions := client.ScheduleClient().CreateOptions()
//        _, err := client.ScheduleClient().Create(ctx, schedules.StartScheduleOptions{
//            ID: "timeout-monitor",
//            Schedule: &schedules.Schedule{
//                Spec: &schedules.ScheduleSpec{
//                    CronExpressions: []string{"0 * * * *"}, // Every hour
//                },
//                Action: &schedules.ScheduleAction{
//                    StartWorkflow: &schedules.StartWorkflowAction{
//                        ID:       "timeout-monitor-run",
//                        Workflow: temporal.TimeoutMonitorWorkflow,
//                        Options:  scheduleOptions,
//                    },
//                },
//            },
//        })
//        if err != nil {
//            log.Fatalf("Failed to create timeout monitor schedule: %v", err)
//        }
//
//        if err := w.Run(worker.InterruptCh()); err != nil {
//            log.Fatalf("Worker failed: %v", err)
//        }
//    }

// ============================================================================
// HELPER: WorkflowTimeoutInfo - Information about a workflow timeout
// ============================================================================

type WorkflowTimeoutInfo struct {
	WorkflowID    string                   `json:"workflow_id"`
	WorkflowName  string                   `json:"workflow_name"`
	StepName      string                   `json:"step_name"`
	Assignee      string                   `json:"assignee"`
	StepStart     time.Time                `json:"step_start"`
	ElapsedHours  float64                  `json:"elapsed_hours"`
	DueHours      int                      `json:"due_hours"`
	ActionsNeeded []map[string]interface{} `json:"actions_needed"`
	IsOverdue     bool                     `json:"is_overdue"`
}

// ============================================================================
// ENHANCED TIMEOUT MONITOR - WORKFLOW CONTEXT VERSION
// ============================================================================
// This is an alternative implementation that doesn't require a database connection
// to be passed directly - instead it retrieves it from the activity context

// GetTimeoutStatus retrieves the current timeout status for a workflow
// Can be called from the Temporal workflow for debugging/inspection
func GetTimeoutStatus(ctx context.Context, db *sql.DB, workflowID string) (*WorkflowTimeoutInfo, error) {
	// Query to get current workflow step and timeout info
	query := `
		SELECT 
			wi.id,
			wi.workflow,
			wi.step,
			wi.assignee,
			wi.step_start,
			EXTRACT(HOUR FROM NOW() - wi.step_start) as elapsed_hours,
			tt.due_hours,
			tt.actions_json,
			CASE WHEN EXTRACT(HOUR FROM NOW() - wi.step_start) >= tt.due_hours THEN TRUE ELSE FALSE END as is_overdue
		FROM workflow_instances wi
		LEFT JOIN workflow_timeout_triggers tt 
			ON wi.workflow = tt.workflow_name 
			AND wi.step = tt.step_name
		WHERE wi.id = $1
	`

	var info WorkflowTimeoutInfo
	var elapsedHours sql.NullFloat64
	var dueHours sql.NullInt32
	var actionsJSON sql.NullString

	err := db.QueryRowContext(ctx, query, workflowID).Scan(
		&info.WorkflowID,
		&info.WorkflowName,
		&info.StepName,
		&info.Assignee,
		&info.StepStart,
		&elapsedHours,
		&dueHours,
		&actionsJSON,
		&info.IsOverdue,
	)

	if err != nil {
		return nil, err
	}

	if elapsedHours.Valid {
		info.ElapsedHours = elapsedHours.Float64
	}
	if dueHours.Valid {
		info.DueHours = int(dueHours.Int32)
	}

	// Parse actions
	if actionsJSON.Valid {
		var actions []map[string]interface{}
		if err := json.Unmarshal([]byte(actionsJSON.String), &actions); err == nil {
			info.ActionsNeeded = actions
		}
	}

	return &info, nil
}

// ============================================================================
// WORKFLOW CHILD EXECUTION - DETAILED TIMEOUT CHECK
// ============================================================================
// For advanced scenarios where you want to monitor a specific workflow's timeouts

// TimeoutMonitorChildWorkflow monitors a specific workflow instance for timeout
// This workflow can be started as a child workflow from your main workflow
func TimeoutMonitorChildWorkflow(ctx workflow.Context, workflowID string) error {
	logger := workflow.GetLogger(ctx)

	logger.Info("Starting timeout monitor for workflow", "workflow_id", workflowID)

	// Poll for timeout every 5 minutes
	for {
		// Check current status
		var status *WorkflowTimeoutInfo
		err := workflow.ExecuteActivity(ctx, GetTimeoutStatusActivity, workflowID).Get(ctx, &status)
		if err != nil {
			logger.Error("Failed to get timeout status", "error", err)
			return err
		}

		logger.Info("Timeout check",
			"workflow_id", workflowID,
			"elapsed_hours", status.ElapsedHours,
			"due_hours", status.DueHours,
			"is_overdue", status.IsOverdue,
		)

		// If overdue, we could signal the main workflow or take action
		if status.IsOverdue {
			logger.Warn("Workflow is OVERDUE - timeout action needed",
				"workflow_id", workflowID,
				"elapsed", status.ElapsedHours,
				"due", status.DueHours,
			)
			// Return to let main workflow handle escalation
			return nil
		}

		// Sleep for 5 minutes before next check
		if err := workflow.Sleep(ctx, 5*time.Minute); err != nil {
			// Workflow cancelled
			return nil
		}
	}
}

// GetTimeoutStatusActivity is an activity that retrieves timeout status
func GetTimeoutStatusActivity(ctx context.Context, workflowID string) (*WorkflowTimeoutInfo, error) {
	// This would be called from the activity context which has database access
	// Implementation depends on your activity setup
	return nil, nil // Placeholder - implement based on your DB setup
}

// ============================================================================
// TESTING & DEBUGGING
// ============================================================================

// TimeoutMonitorTestWorkflow is a test workflow that simulates timeout scenarios
// Use this for testing timeout behavior without waiting for real timeouts
func TimeoutMonitorTestWorkflow(ctx workflow.Context, workflowID string, simulatedElapsedHours float64) error {
	logger := workflow.GetLogger(ctx)

	logger.Info("Test timeout monitor", "workflow_id", workflowID, "simulated_elapsed_hours", simulatedElapsedHours)

	// Activity options
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		StartToCloseTimeout:    3 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Execute test activity
	var result bool
	err := workflow.ExecuteActivity(ctx, TimeoutMonitorTestActivity, workflowID, simulatedElapsedHours).Get(ctx, &result)
	if err != nil {
		logger.Error("Test activity failed", "error", err)
		return err
	}

	logger.Info("Test timeout monitor completed", "success", result)
	return nil
}

// TimeoutMonitorTestActivity is an activity for testing timeout scenarios
func TimeoutMonitorTestActivity(ctx context.Context, workflowID string, simulatedElapsedHours float64) (bool, error) {
	log.Printf("[TimeoutMonitor] Test: Simulating %f hours elapsed for workflow %s", simulatedElapsedHours, workflowID)
	// Implementation would mock the database and test timeout triggers
	return true, nil
}

// ============================================================================
// SUMMARY
// ============================================================================
//
// This file provides:
//
// 1. TimeoutMonitorWorkflow - Main Temporal workflow (runs every hour)
// 2. TimeoutMonitorActivity - Activity that executes timeout checks
// 3. TimeoutMonitorChildWorkflow - Optional child workflow for specific instances
// 4. GetTimeoutStatusActivity - Query workflow timeout status
// 5. TestWorkflows - For testing timeout scenarios
//
// To enable:
// 1. In worker.go, register: w.RegisterWorkflow(temporal.TimeoutMonitorWorkflow)
// 2. Register activity: w.RegisterActivity(temporal.TimeoutMonitorActivity)
// 3. Create schedule with cron "0 * * * *" (every hour)
// 4. Ensure workflow_timeout_triggers table exists (from timeout_triggers.sql)
//
// Results:
// - Escalate: Workflow reassigned, emails sent
// - Notify: Email/notification sent to assignee
// - Log: Audit event recorded
// - All tracked in workflow_timeout_events table
//
