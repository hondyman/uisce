package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"go.temporal.io/sdk/activity"
)

// UIActivities holds dependencies for UI-related activities
type UIActivities struct {
	DB *sql.DB
}

func NewUIActivities(db *sql.DB) *UIActivities {
	return &UIActivities{DB: db}
}

// ActivityUserInteraction pauses the workflow and creates a task record in the DB
// requiring human intervention via the MDUI.
func (a *UIActivities) ActivityUserInteraction(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ActivityUserInteraction started", "config", config)

	// 1. Extract Config
	viewDefName, _ := config["viewDefinitionName"].(string)
	title, _ := config["title"].(string)
	if title == "" {
		title = "Action Required"
	}
	// "inputKeys" config can specify which state variables to pass to the UI
	// For simplicity, we pass the entire state or a filtered subset
	inputContext := state

	// 2. Get Temporal Info to resume later
	activityInfo := activity.GetInfo(ctx)
	taskToken := activityInfo.TaskToken

	// 3. Resolve View ID from Name if needed (optional optimization, UI can lookup by name)
	// For now, we store the name or ID in view_definition_id if it's a UUID,
	// but our schema uses UUID. Let's lookup the UUID if a name is provided.
	var viewID string
	if viewDefName != "" {
		err := a.DB.QueryRowContext(ctx, "SELECT id FROM view_definitions WHERE name = $1", viewDefName).Scan(&viewID)
		if err != nil {
			logger.Error("Failed to find view definition by name", "name", viewDefName, "error", err)
			// Proceeding with null viewID might be valid for generic tasks, or fail.
			// Let's assume fail for now if View is required.
			return nil, fmt.Errorf("view definition not found: %s", viewDefName)
		}
	}

	// 4. Persist Task Record
	inputJSON, _ := json.Marshal(inputContext)
	query := `
		INSERT INTO human_tasks (workflow_id, run_id, task_token, view_definition_id, title, input_context, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'PENDING')
		RETURNING id
	`
	var taskID string
	err := a.DB.QueryRowContext(ctx, query,
		activityInfo.WorkflowExecution.ID,
		activityInfo.WorkflowExecution.RunID,
		taskToken,
		viewID,
		title,
		inputJSON,
	).Scan(&taskID)

	if err != nil {
		logger.Error("Failed to create human task record", "error", err)
		return nil, err
	}

	logger.Info("Human Task Created", "taskID", taskID, "view", viewDefName)

	// 5. Return ErrResultPending to pause workflow execution
	// The workflow will resume when CompletionHandler calls Client.CompleteActivity(taskToken, result)
	return nil, activity.ErrResultPending
}
