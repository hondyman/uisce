package bp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hondyman/uisce/libs/db/queries"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ============================================================================
// TRIGGER ENGINE: PostgreSQL LISTEN/NOTIFY → Temporal Workflows
// ============================================================================

// TriggerEngine listens for business process triggers and fires Temporal workflows
type TriggerEngine struct {
	db                *sqlx.DB
	workflowInitiator WorkflowInitiator
	tenantID          string
	stopChan          chan bool
	wg                sync.WaitGroup
	logger            *log.Logger
}

// WorkflowInitiator interface for starting Temporal workflows
type WorkflowInitiator interface {
	StartBPWorkflow(ctx context.Context, bpID string, data map[string]interface{}) (string, error)
}

// TriggerEvent represents a business process trigger fired from PostgreSQL
type TriggerEvent struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	TriggerID      string                 `json:"trigger_id"`
	TriggerType    string                 `json:"trigger_type"` // event|schedule|manual
	SourceTable    string                 `json:"source_table"`
	SourceRecordID string                 `json:"source_record_id"`
	SourceData     map[string]interface{} `json:"source_data"`
	ProcessID      string                 `json:"process_id"`
	TriggeredAt    time.Time              `json:"triggered_at"`
	Status         string                 `json:"status"` // pending|firing|completed|failed
	ExecutionID    string                 `json:"execution_id"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// NewTriggerEngine creates a new trigger engine
func NewTriggerEngine(db *sqlx.DB, initiator WorkflowInitiator, tenantID string, logger *log.Logger) *TriggerEngine {
	return &TriggerEngine{
		db:                db,
		workflowInitiator: initiator,
		tenantID:          tenantID,
		stopChan:          make(chan bool),
		logger:            logger,
	}
}

// Start begins listening for PostgreSQL notifications
func (te *TriggerEngine) Start(ctx context.Context) error {
	te.logger.Println("🚀 TriggerEngine starting, listening for bp_trigger_events...")

	te.wg.Add(1)
	go te.listenForNotifications(ctx)

	return nil
}

// Stop stops the trigger engine
func (te *TriggerEngine) Stop() {
	te.logger.Println("⏹️  TriggerEngine stopping...")
	close(te.stopChan)
	te.wg.Wait()
	te.logger.Println("✅ TriggerEngine stopped")
}

// listenForNotifications listens for PostgreSQL NOTIFY events
func (te *TriggerEngine) listenForNotifications(ctx context.Context) {
	defer te.wg.Done()

	// Create listener
	listener := pq.NewListener(
		"postgres://app_user:password@localhost:5432/alpha?sslmode=disable",
		10*time.Second,
		time.Minute,
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				te.logger.Printf("❌ Listener error: %v", err)
			}
		},
	)
	defer listener.Close()

	// Listen for bp_trigger_events channel
	err := listener.Listen("bp_trigger_events")
	if err != nil {
		te.logger.Printf("❌ Failed to listen: %v", err)
		return
	}

	te.logger.Println("👂 Listening on bp_trigger_events channel...")

	for {
		select {
		case <-te.stopChan:
			te.logger.Println("📍 Stop signal received, exiting listener")
			return

		case notification := <-listener.Notify:
			if notification == nil {
				continue
			}

			te.logger.Printf("📬 Received notification: %s", notification.Extra)

			// Parse trigger event from NOTIFY payload
			event := &TriggerEvent{}
			if err := json.Unmarshal([]byte(notification.Extra), event); err != nil {
				te.logger.Printf("❌ Failed to parse trigger event: %v", err)
				continue
			}

			// Fire the workflow
			go te.fireTrigger(ctx, event)

		case <-ctx.Done():
			te.logger.Println("📍 Context cancelled, exiting listener")
			return
		}
	}
}

// fireTrigger evaluates the trigger condition and fires the workflow if matched
func (te *TriggerEngine) fireTrigger(ctx context.Context, event *TriggerEvent) {
	te.logger.Printf("🔥 Processing trigger: %s (process: %s)", event.TriggerID, event.ProcessID)

	// Step 1: Load trigger configuration
	trigger, err := te.loadTrigger(ctx, event.TriggerID)
	if err != nil {
		te.logger.Printf("❌ Failed to load trigger: %v", err)
		te.recordTriggerFailure(ctx, event.ID, fmt.Sprintf("Load error: %v", err))
		return
	}

	// Step 2: Evaluate trigger condition
	shouldFire, err := te.evaluateTriggerCondition(ctx, trigger, event)
	if err != nil {
		te.logger.Printf("❌ Failed to evaluate trigger condition: %v", err)
		te.recordTriggerFailure(ctx, event.ID, fmt.Sprintf("Eval error: %v", err))
		return
	}

	if !shouldFire {
		te.logger.Printf("⏭️  Trigger condition not met, skipping")
		return
	}

	// Step 3: Load business process configuration
	bp, err := te.loadBP(ctx, event.ProcessID)
	if err != nil {
		te.logger.Printf("❌ Failed to load BP: %v", err)
		te.recordTriggerFailure(ctx, event.ID, fmt.Sprintf("BP load error: %v", err))
		return
	}

	te.logger.Printf("📋 Loaded BP: %s (%d steps)", bp.ProcessName, len(bp.Steps))

	// Step 4: Fire Temporal workflow with BP context
	workflowID, err := te.workflowInitiator.StartBPWorkflow(ctx, event.ProcessID, map[string]interface{}{
		"trigger_id":   event.TriggerID,
		"source_data":  event.SourceData,
		"tenant_id":    event.TenantID,
		"process_id":   event.ProcessID,
		"bp_steps":     bp.Steps,
		"triggered_at": event.TriggeredAt,
	})

	if err != nil {
		te.logger.Printf("❌ Failed to start workflow: %v", err)
		te.recordTriggerFailure(ctx, event.ID, fmt.Sprintf("Workflow start error: %v", err))
		return
	}

	te.logger.Printf("✅ Workflow started: %s", workflowID)

	// Step 5: Record successful trigger fire
	te.recordTriggerSuccess(ctx, event.ID, workflowID)
}

// evaluateTriggerCondition evaluates whether trigger should fire based on condition
func (te *TriggerEngine) evaluateTriggerCondition(ctx context.Context, trigger *BPAdaptiveTrigger, event *TriggerEvent) (bool, error) {
	// Parse condition from trigger.TriggerCondition
	// Example: "source_table == 'employees' AND salary > 100000"
	//
	// In production, use expression evaluator (expr.Eval or rego-style)

	// For now, simple examples:
	switch trigger.TriggerType {
	case "event":
		// Event-based: check if source_table matches
		return te.evaluateEventCondition(trigger.TriggerCondition, event)

	case "schedule":
		// Schedule-based: always fire (Temporal handles scheduling)
		return true, nil

	case "manual":
		// Manual trigger from UI
		return true, nil

	default:
		return false, fmt.Errorf("unknown trigger type: %s", trigger.TriggerType)
	}
}

// evaluateEventCondition evaluates event-based trigger conditions
func (te *TriggerEngine) evaluateEventCondition(condition string, event *TriggerEvent) (bool, error) {
	// Simple condition parser - in production use proper expression engine
	// Example conditions:
	// - "source_table == 'employees'"
	// - "source_table == 'employees' AND action == 'INSERT'"
	// - "source_data.salary > 100000"

	// Placeholder: always fire for now
	te.logger.Printf("📋 Evaluating condition: %s", condition)
	return true, nil
}

// loadTrigger loads trigger configuration from database
func (te *TriggerEngine) loadTrigger(ctx context.Context, triggerID string) (*BPAdaptiveTrigger, error) {
	trigger := &BPAdaptiveTrigger{}

	query := `
		SELECT id, tenant_id, step_id, trigger_name, trigger_condition, trigger_type,
		       action_type, action_config, context_variables, is_active
		FROM bp_adaptive_triggers
		WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE
	`

	err := te.db.QueryRowContext(ctx, query, triggerID, te.tenantID).Scan(
		&trigger.ID, &trigger.TenantID, &trigger.StepID, &trigger.TriggerName,
		&trigger.TriggerCondition, &trigger.TriggerType, &trigger.ActionType,
		&trigger.ActionConfig, &trigger.ContextVariables, &trigger.IsActive,
	)

	return trigger, err
}

// loadBP loads business process definition and steps
func (te *TriggerEngine) loadBP(ctx context.Context, processID string) (*BusinessProcess, error) {
	bp := &BusinessProcess{}

	// Load BP header
	err := te.db.QueryRowContext(ctx, queries.GetBusinessProcessHeader, processID, te.tenantID).Scan(
		&bp.ID, &bp.TenantID, &bp.ProcessName, &bp.Description, &bp.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load BP: %w", err)
	}

	// Load BP steps
	rows, err := te.db.QueryContext(ctx, queries.ListBPStepsForTrigger, processID, te.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load BP steps: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		step := BPStep{}

		err := rows.Scan(
			&step.ID, &step.ProcessID, &step.StepOrder, &step.StepType, &step.StepName, &step.Description,
			&step.DurationHours, &step.Status, &step.Config, &step.CreatedAt, &step.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan BP step: %w", err)
		}

		bp.Steps = append(bp.Steps, step)
	}

	return bp, rows.Err()
}

// recordTriggerSuccess records successful trigger execution.
func (te *TriggerEngine) recordTriggerSuccess(ctx context.Context, triggerEventID string, workflowID string) {
	if _, err := te.db.ExecContext(ctx, queries.UpdateTriggerEventCompleted, workflowID, triggerEventID); err != nil {
		te.logger.Printf("❌ Failed to record trigger success: %v", err)
	} else {
		te.logger.Printf("✅ Recorded trigger success: %s", workflowID)
	}
}

// recordTriggerFailure records trigger failure
func (te *TriggerEngine) recordTriggerFailure(ctx context.Context, triggerEventID string, errorMsg string) {
	if _, err := te.db.ExecContext(ctx, queries.UpdateTriggerEventFailed, errorMsg, triggerEventID); err != nil {
		te.logger.Printf("❌ Failed to record trigger failure: %v", err)
	} else {
		te.logger.Printf("❌ Recorded trigger failure: %s", errorMsg)
	}
}

// ============================================================================
// TRIGGER TABLE STRUCTURES (For reference)
// ============================================================================

// BPAdaptiveTrigger represents a trigger definition
type BPAdaptiveTrigger struct {
	ID               string          `db:"id"`
	TenantID         string          `db:"tenant_id"`
	StepID           string          `db:"step_id"`
	TriggerName      string          `db:"trigger_name"`
	TriggerCondition string          `db:"trigger_condition"`
	TriggerType      string          `db:"trigger_type"` // event|schedule|manual
	ActionType       string          `db:"action_type"`
	ActionConfig     json.RawMessage `db:"action_config"`
	ContextVariables pq.StringArray  `db:"context_variables"`
	IsActive         bool            `db:"is_active"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
}

// ============================================================================
// HELPER: Create trigger notification (call from INSERT trigger)
// ============================================================================

// CreateTriggerNotificationSQL returns SQL to fire a trigger event via NOTIFY
// Call this in PostgreSQL trigger after INSERT on source table
const CreateTriggerNotificationSQL = `
CREATE OR REPLACE FUNCTION notify_bp_trigger()
RETURNS TRIGGER AS $$
DECLARE
    v_trigger_id UUID;
    v_process_id UUID;
    v_trigger_event bp_trigger_events%ROWTYPE;
BEGIN
    -- Find matching trigger for this table/operation
    SELECT id, process_id INTO v_trigger_id, v_process_id
    FROM bp_adaptive_triggers
    WHERE source_table = TG_TABLE_NAME
      AND trigger_type = 'event'
      AND is_active = TRUE
      AND tenant_id = COALESCE(NEW.tenant_id, OLD.tenant_id)
    LIMIT 1;

    -- If trigger found, create event and notify
    IF v_trigger_id IS NOT NULL THEN
        INSERT INTO bp_trigger_events (
            id, tenant_id, trigger_id, trigger_type, source_table, 
            source_record_id, source_data, process_id, status
        ) VALUES (
            gen_random_uuid(),
            NEW.tenant_id,
            v_trigger_id,
            'event',
            TG_TABLE_NAME,
            NEW.id::TEXT,
            to_jsonb(NEW),
            v_process_id,
            'pending'
        ) RETURNING * INTO v_trigger_event;

        -- Notify listeners
        PERFORM pg_notify(
            'bp_trigger_events',
            jsonb_build_object(
                'id', v_trigger_event.id,
                'tenant_id', v_trigger_event.tenant_id,
                'trigger_id', v_trigger_event.trigger_id,
                'trigger_type', v_trigger_event.trigger_type,
                'source_table', v_trigger_event.source_table,
                'source_record_id', v_trigger_event.source_record_id,
                'source_data', v_trigger_event.source_data,
                'process_id', v_trigger_event.process_id,
                'triggered_at', v_trigger_event.created_at
            )::TEXT
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`
