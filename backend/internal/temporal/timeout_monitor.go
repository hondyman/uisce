package temporal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// ============================================================================
// TIMEOUT MONITOR SERVICE
// ============================================================================
// Monitors workflow steps for timeout and executes escalation, notification, and logging actions
// Runs every hour to check for overdue steps

type TimeoutAction struct {
	Percent int    `json:"percent"`
	Type    string `json:"type"`   // "escalate", "notify", "log"
	Target  string `json:"target"` // "hr_director", "assignee", "audit"
	Message string `json:"message"`
}

type TimeoutTriggerRule struct {
	ID                 string          `db:"id"`
	TenantID           string          `db:"tenant_id"`
	WorkflowName       string          `db:"workflow_name"`
	StepName           string          `db:"step_name"`
	DueHours           int             `db:"due_hours"`
	TriggerPercentages []int           `db:"trigger_percentages"`
	ActionsJSON        json.RawMessage `db:"actions_json"`
	IsActive           bool            `db:"is_active"`
}

type WorkflowInstance struct {
	ID        string    `db:"id"`
	TenantID  string    `db:"tenant_id"`
	Workflow  string    `db:"workflow"`
	Step      string    `db:"step"`
	Assignee  string    `db:"assignee"`
	StepStart time.Time `db:"step_start"`
}

type TimeoutMonitor struct {
	db *sqlx.DB
}

// NewTimeoutMonitor creates a new timeout monitor service
func NewTimeoutMonitor(db *sqlx.DB) *TimeoutMonitor {
	return &TimeoutMonitor{db: db}
}

// Start begins the timeout monitoring service (runs every hour)
func (tm *TimeoutMonitor) Start(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Println("[TimeoutMonitor] Service started, checking every hour")

	for {
		select {
		case <-ctx.Done():
			log.Println("[TimeoutMonitor] Service stopped")
			return ctx.Err()
		case <-ticker.C:
			err := tm.CheckAndExecuteTimeouts(ctx)
			if err != nil {
				log.Printf("[TimeoutMonitor] Error checking timeouts: %v", err)
			}
		}
	}
}

// CheckAndExecuteTimeouts checks for overdue workflow steps and executes timeout actions
func (tm *TimeoutMonitor) CheckAndExecuteTimeouts(ctx context.Context) error {
	if tm.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Query all pending workflow instances
	// TODO: Replace with Hasura GraphQL query:
	//   query { workflow_instances(where: {_or: [{status: {_eq: "pending"}}, {status: {_eq: "in_progress"}}], step_start: {_is_null: false}}, limit: 1000) { id tenant_id workflow step assignee step_start } }
	var instances []WorkflowInstance
	query := `
		SELECT id, tenant_id, workflow, step, assignee, step_start
		FROM workflow_instances
		WHERE (status = 'pending' OR status = 'in_progress')
		AND step_start IS NOT NULL
		LIMIT 1000
	`

	err := tm.db.SelectContext(ctx, &instances, query)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query workflow instances: %w", err)
	}

	executedCount := 0

	// For each workflow instance, check timeout triggers
	for _, instance := range instances {
		// Fetch timeout triggers for this workflow/step
		// TODO: Replace with Hasura GraphQL query:
		//   query { workflow_timeout_triggers(where: {workflow_name: {_eq: $workflow}, step_name: {_eq: $step}, is_active: {_eq: true}}, limit: 100) { id tenant_id workflow_name step_name due_hours actions_json is_active } }
		var triggers []TimeoutTriggerRule
		triggerQuery := `
			SELECT id, tenant_id, workflow_name, step_name, due_hours, 
				   actions_json, is_active
			FROM workflow_timeout_triggers
			WHERE workflow_name = $1
			AND step_name = $2
			AND is_active = true
			LIMIT 100
		`

		err := tm.db.SelectContext(ctx, &triggers, triggerQuery, instance.Workflow, instance.Step)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("[TimeoutMonitor] Failed to fetch triggers: %v", err)
			continue
		}

		// Check each trigger
		for _, trigger := range triggers {
			// Calculate elapsed time
			elapsed := time.Since(instance.StepStart).Hours()
			dueHours := float64(trigger.DueHours)

			// Parse actions
			var actions []TimeoutAction
			if err := json.Unmarshal(trigger.ActionsJSON, &actions); err != nil {
				log.Printf("[TimeoutMonitor] Failed to parse actions: %v", err)
				continue
			}

			// Execute actions that match elapsed percentage
			for _, action := range actions {
				thresholdHours := dueHours * float64(action.Percent) / 100
				if elapsed >= thresholdHours {
					err := tm.executeTimeoutAction(ctx, instance, action)
					if err != nil {
						log.Printf("[TimeoutMonitor] Failed to execute action: %v", err)
					} else {
						executedCount++
					}
				}
			}
		}
	}

	if executedCount > 0 {
		log.Printf("[TimeoutMonitor] Checked timeouts, executed %d actions", executedCount)
	}

	return nil
}

// executeTimeoutAction executes the appropriate action based on type
func (tm *TimeoutMonitor) executeTimeoutAction(ctx context.Context, instance WorkflowInstance, action TimeoutAction) error {
	switch action.Type {
	case "escalate":
		return tm.escalateWorkflow(ctx, instance, action)
	case "notify":
		return tm.notifyAssignee(ctx, instance, action)
	case "log":
		return tm.logTimeoutEvent(ctx, instance, action)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// escalateWorkflow reassigns the workflow to a higher-level user
func (tm *TimeoutMonitor) escalateWorkflow(ctx context.Context, instance WorkflowInstance, action TimeoutAction) error {
	log.Printf("[Timeout] Escalating %s.%s to %s - Reason: %s",
		instance.Workflow, instance.Step, action.Target, action.Message)

	// Update workflow instance with new assignee
	// TODO: Replace with Hasura GraphQL mutation:
	//   mutation { update_workflow_instances_by_pk(pk_columns: {id: $id}, _set: {assignee: $target, escalated_at: now(), escalation_reason: $message}) { id } }
	updateQuery := `
		UPDATE workflow_instances
		SET assignee = $1, escalated_at = NOW(), escalation_reason = $2
		WHERE id = $3
	`

	_, err := tm.db.ExecContext(ctx, updateQuery, action.Target, action.Message, instance.ID)
	if err != nil {
		return fmt.Errorf("failed to escalate workflow: %w", err)
	}

	// Publish escalation event
	tm.publishTimeoutEvent("timeout.escalated", map[string]interface{}{
		"workflow_id":  instance.ID,
		"workflow":     instance.Workflow,
		"step":         instance.Step,
		"escalated_to": action.Target,
		"message":      action.Message,
		"timestamp":    time.Now(),
	})

	return nil
}

// notifyAssignee sends a notification to the current assignee
func (tm *TimeoutMonitor) notifyAssignee(ctx context.Context, instance WorkflowInstance, action TimeoutAction) error {
	log.Printf("[Timeout] Notifying %s - Message: %s", instance.Assignee, action.Message)

	// Insert notification record
	// TODO: Replace with Hasura GraphQL mutation:
	//   mutation { insert_workflow_notifications_one(object: {workflow_id: $id, recipient: $assignee, message: $message, type: "timeout_warning", created_at: now()}, on_conflict: {constraint: workflow_notifications_pkey, update_columns: []}) { id } }
	//   Note: ON CONFLICT DO NOTHING with empty update_columns array
	notifyQuery := `
		INSERT INTO workflow_notifications (workflow_id, recipient, message, type, created_at)
		VALUES ($1, $2, $3, 'timeout_warning', NOW())
		ON CONFLICT DO NOTHING
	`

	_, err := tm.db.ExecContext(ctx, notifyQuery, instance.ID, instance.Assignee, action.Message)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Publish notification event
	tm.publishTimeoutEvent("timeout.notified", map[string]interface{}{
		"workflow_id": instance.ID,
		"recipient":   instance.Assignee,
		"message":     action.Message,
		"timestamp":   time.Now(),
	})

	return nil
}

// logTimeoutEvent records the timeout event in the audit log
func (tm *TimeoutMonitor) logTimeoutEvent(ctx context.Context, instance WorkflowInstance, action TimeoutAction) error {
	log.Printf("[Timeout] Logging event for %s.%s - Message: %s",
		instance.Workflow, instance.Step, action.Message)

	details := map[string]interface{}{
		"message":    action.Message,
		"target":     action.Target,
		"step_start": instance.StepStart,
		"timeout_at": time.Now(),
	}

	detailsJSON, _ := json.Marshal(details)

	// TODO: Replace with Hasura GraphQL mutation:
	//   mutation { insert_workflow_audit_log_one(object: {workflow_id: $id, workflow_name: $workflow, step_name: $step, action: "timeout", details: $details, created_at: now()}) { id } }
	//   Note: details is JSONB, pass detailsJSON directly
	auditQuery := `
		INSERT INTO workflow_audit_log (workflow_id, workflow_name, step_name, action, details, created_at)
		VALUES ($1, $2, $3, 'timeout', $4, NOW())
	`

	_, err := tm.db.ExecContext(ctx, auditQuery, instance.ID, instance.Workflow, instance.Step, string(detailsJSON))
	if err != nil {
		return fmt.Errorf("failed to log timeout event: %w", err)
	}

	// Publish audit event
	tm.publishTimeoutEvent("timeout.logged", map[string]interface{}{
		"workflow_id": instance.ID,
		"workflow":    instance.Workflow,
		"step":        instance.Step,
		"details":     details,
		"timestamp":   time.Now(),
	})

	return nil
}

// publishTimeoutEvent publishes an event to your message queue
func (tm *TimeoutMonitor) publishTimeoutEvent(eventType string, data map[string]interface{}) {
	// TODO: Integrate with your event bus / RabbitMQ publisher
	logData, _ := json.Marshal(data)
	log.Printf("[TimeoutEvent] %s: %s", eventType, string(logData))
}
