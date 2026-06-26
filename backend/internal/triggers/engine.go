package triggers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.temporal.io/sdk/client"
	temporalpkg "go.temporal.io/sdk/temporal"

	"github.com/hondyman/semlayer/backend/internal/workflows"
)

// EntityEvent represents a domain event that triggers business processes
type EntityEvent struct {
	TenantID  string                 `json:"tenant_id"`
	Entity    string                 `json:"entity"`
	Action    string                 `json:"action"`
	EntityID  string                 `json:"entity_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Trigger represents a trigger configuration from bp_triggers table
type Trigger struct {
	ID              uuid.UUID              `json:"id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	TriggerName     string                 `json:"trigger_name"`
	TriggerType     string                 `json:"trigger_type"`
	Enabled         bool                   `json:"enabled"`
	EventConfig     map[string]interface{} `json:"event_config"`
	ConditionConfig map[string]interface{} `json:"condition_config"`
	TargetProcessID uuid.UUID              `json:"target_process_id"`
	Priority        int                    `json:"priority"`
	NotifyConfig    map[string]interface{} `json:"notification_config"`
}

// TriggerEngine coordinates reading triggers from DB, listening for events,
// and starting Temporal workflows.
type TriggerEngine struct {
	temporal client.Client
	db       *sql.DB
	listener *pq.Listener
}

// NewTriggerEngine constructs a TriggerEngine. Temporal client and DB must be provided.
func NewTriggerEngine(temporal client.Client, db *sql.DB) *TriggerEngine {
	return &TriggerEngine{temporal: temporal, db: db}
}

// Start initializes the trigger engine by starting event listener and escalation monitor.
func (e *TriggerEngine) Start(ctx context.Context, pgURL string) error {
	log.Println("🚀 Starting TriggerEngine...")

	// Start PostgreSQL event listener in background
	go func() {
		if err := e.StartEventListener(ctx, pgURL); err != nil {
			log.Printf("❌ Event listener error: %v", err)
		}
	}()

	// Start escalation monitor in background
	go func() {
		if err := e.StartEscalationMonitor(ctx); err != nil {
			log.Printf("❌ Escalation monitor error: %v", err)
		}
	}()

	log.Println("✅ TriggerEngine started successfully")
	return nil
}

// StartEventListener listens for PostgreSQL NOTIFY events on 'entity_events' and dispatches them.
func (e *TriggerEngine) StartEventListener(ctx context.Context, pgURL string) error {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("PostgreSQL listener error: %v", err)
		}
	}

	minReconnectInterval := 10 * time.Second
	maxReconnectInterval := time.Minute
	listener := pq.NewListener(pgURL, minReconnectInterval, maxReconnectInterval, reportProblem)
	e.listener = listener

	if err := listener.Listen("entity_events"); err != nil {
		return fmt.Errorf("failed to listen on channel: %w", err)
	}

	log.Println("📡 Listening for entity_events on PostgreSQL...")

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Stopping event listener")
			return listener.Close()

		case notification := <-listener.Notify:
			if notification != nil {
				var event EntityEvent
				if err := json.Unmarshal([]byte(notification.Extra), &event); err != nil {
					log.Printf("❌ Failed to parse event: %v", err)
					continue
				}

				log.Printf("📨 Received event: %s.%s (ID: %s)", event.Entity, event.Action, event.EntityID)

				// Process event triggers asynchronously
				go func(evt EntityEvent) {
					if err := e.ProcessEventTriggers(context.Background(), evt); err != nil {
						log.Printf("❌ Error processing triggers: %v", err)
					}
				}(event)
			}

		case <-time.After(90 * time.Second):
			// Ping the connection to keep it alive
			go listener.Ping()
		}
	}
}

// ProcessEventTriggers loads matching triggers and enqueues them for execution.
func (e *TriggerEngine) ProcessEventTriggers(ctx context.Context, event EntityEvent) error {
	log.Printf("🔍 Processing triggers for: %s.%s", event.Entity, event.Action)

	query := `
		SELECT id, tenant_id, trigger_name, trigger_type, event_config, condition_config, target_process_id, priority, notification_config
		FROM bp_triggers
		WHERE tenant_id = $1
		  AND trigger_type = 'event'
		  AND enabled = true
		ORDER BY priority ASC
	`

	rows, err := e.db.QueryContext(ctx, query, event.TenantID)
	if err != nil {
		return fmt.Errorf("query triggers failed: %w", err)
	}
	defer rows.Close()

	triggerCount := 0
	for rows.Next() {
		var trigger Trigger
		var eventConfigJSON, conditionConfigJSON, notifyConfigJSON []byte

		if err := rows.Scan(
			&trigger.ID,
			&trigger.TenantID,
			&trigger.TriggerName,
			&trigger.TriggerType,
			&eventConfigJSON,
			&conditionConfigJSON,
			&trigger.TargetProcessID,
			&trigger.Priority,
			&notifyConfigJSON,
		); err != nil {
			log.Printf("❌ Failed to scan trigger: %v", err)
			continue
		}

		if len(eventConfigJSON) > 0 {
			_ = json.Unmarshal(eventConfigJSON, &trigger.EventConfig)
		}
		if len(conditionConfigJSON) > 0 {
			_ = json.Unmarshal(conditionConfigJSON, &trigger.ConditionConfig)
		}
		if len(notifyConfigJSON) > 0 {
			_ = json.Unmarshal(notifyConfigJSON, &trigger.NotifyConfig)
		}

		// Match event configuration
		if !e.matchesEventConfig(trigger.EventConfig, event) {
			continue
		}

		// Evaluate conditions
		if trigger.ConditionConfig != nil && !e.evaluateConditions(trigger.ConditionConfig, event.Data) {
			log.Printf("⏭️  Skipping trigger '%s': conditions not met", trigger.TriggerName)
			continue
		}

		// Execute trigger
		log.Printf("✅ Trigger matched: '%s' (priority: %d)", trigger.TriggerName, trigger.Priority)
		triggerCount++

		if err := e.executeTrigger(ctx, trigger, event); err != nil {
			log.Printf("❌ Failed to execute trigger '%s': %v", trigger.TriggerName, err)
		}
	}

	if triggerCount == 0 {
		log.Printf("ℹ️  No matching triggers found for %s.%s", event.Entity, event.Action)
	} else {
		log.Printf("✅ Executed %d trigger(s)", triggerCount)
	}

	return nil
}

// executeTrigger starts a Temporal workflow for a matched trigger.
func (e *TriggerEngine) executeTrigger(ctx context.Context, trigger Trigger, event EntityEvent) error {
	executionID := uuid.New()
	startTime := time.Now()

	log.Printf("🚀 Executing trigger '%s' -> Starting Temporal workflow...", trigger.TriggerName)

	// Prepare workflow input
	workflowInput := map[string]interface{}{
		"process_id":   trigger.TargetProcessID.String(),
		"tenant_id":    trigger.TenantID.String(),
		"trigger_name": trigger.TriggerName,
		"event_data":   event.Data,
		"entity":       event.Entity,
		"entity_id":    event.EntityID,
	}

	// Log execution start
	workflowInputJSON, _ := json.Marshal(workflowInput)
	_, err := e.db.ExecContext(ctx, `
		INSERT INTO bp_trigger_executions (id, trigger_id, tenant_id, execution_status, trigger_payload, executed_at)
		VALUES ($1, $2, $3, 'running', $4, $5)
	`, executionID, trigger.ID, trigger.TenantID, workflowInputJSON, startTime)
	if err != nil {
		log.Printf("⚠️  Failed to log execution: %v", err)
	}

	// Check if Temporal client is available
	if e.temporal == nil {
		log.Printf("⚠️  Temporal client not available: simulating workflow execution for process %s", trigger.TargetProcessID)
		// Log as if workflow was started (for testing without Temporal server)
		_, _ = e.db.ExecContext(ctx, `
			UPDATE bp_trigger_executions
			SET execution_status = 'simulated', completed_at = NOW()
			WHERE id = $1
		`, executionID)
		return nil
	}

	// Start Temporal workflow
	workflowID := fmt.Sprintf("bp-trigger-%s-%s", trigger.ID.String(), executionID.String())
	workflowOptions := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                "bp_queue",
		WorkflowExecutionTimeout: 24 * time.Hour,
		RetryPolicy: &temporalpkg.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}

	we, err := e.temporal.ExecuteWorkflow(ctx, workflowOptions, workflows.DynamicBPWorkflow, workflowInput)
	if err != nil {
		// Update execution as failed
		_, _ = e.db.ExecContext(ctx, `
			UPDATE bp_trigger_executions
			SET execution_status = 'failed', error_message = $1, completed_at = NOW()
			WHERE id = $2
		`, err.Error(), executionID)
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	log.Printf("✅ Workflow started: %s (RunID: %s)", we.GetID(), we.GetRunID())

	// Update execution log with workflow ID
	_, err = e.db.ExecContext(ctx, `
		UPDATE bp_trigger_executions
		SET workflow_id = $1, execution_status = 'completed', 
		    execution_time_ms = $2, completed_at = NOW()
		WHERE id = $3
	`, we.GetID(), time.Since(startTime).Milliseconds(), executionID)

	return err
}

// matchesEventConfig checks if a DB trigger's event_config matches the incoming event
func (e *TriggerEngine) matchesEventConfig(config map[string]interface{}, event EntityEvent) bool {
	if len(config) == 0 {
		return true
	}

	// Check entity match
	if entity, ok := config["entity"].(string); ok && entity != event.Entity {
		return false
	}

	// Check action match
	if action, ok := config["action"].(string); ok && action != event.Action {
		return false
	}

	// Check filters
	if filters, ok := config["filters"].(map[string]interface{}); ok {
		for field, expectedValue := range filters {
			if actualValue, exists := event.Data[field]; !exists || actualValue != expectedValue {
				return false
			}
		}
	}

	return true
}

// evaluateConditions runs simple condition checks; placeholder for richer evaluator
func (e *TriggerEngine) evaluateConditions(cfg map[string]interface{}, event map[string]interface{}) bool {
	// For now, accept always; implement expression evaluation later
	_ = cfg
	_ = event
	return true
}

// StartEscalationMonitor runs periodic checks to process escalations based on step durations.
// It checks for running workflows that have exceeded their expected duration and escalates them.
func (e *TriggerEngine) StartEscalationMonitor(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("⏰ Escalation monitor started")

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Stopping escalation monitor")
			return ctx.Err()

		case <-ticker.C:
			log.Println("🔍 Checking for escalations...")

			// Query for running executions that have exceeded their duration
			query := `
				SELECT id, trigger_id, tenant_id, workflow_id
				FROM bp_trigger_executions
				WHERE execution_status = 'running'
				  AND executed_at < NOW() - INTERVAL '24 hours'
				LIMIT 10
			`

			rows, err := e.db.QueryContext(ctx, query)
			if err != nil {
				log.Printf("❌ Error querying escalations: %v", err)
				continue
			}

			escalationCount := 0
			for rows.Next() {
				var execID, triggerID, tenantID, workflowID uuid.UUID

				if err := rows.Scan(&execID, &triggerID, &tenantID, &workflowID); err != nil {
					log.Printf("❌ Failed to scan escalation row: %v", err)
					continue
				}

				log.Printf("⚠️  Escalating workflow: %s (Execution: %s)", workflowID, execID)

				// Mark execution as escalated
				_, _ = e.db.ExecContext(ctx, `
					UPDATE bp_trigger_executions
					SET execution_status = 'escalated', escalation_time = NOW()
					WHERE id = $1
				`, execID)

				// TODO: Send notification to manager/escalation handler
				escalationCount++
			}
			rows.Close()

			if escalationCount > 0 {
				log.Printf("✅ Escalated %d workflow(s)", escalationCount)
			}
		}
	}
}
