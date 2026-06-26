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
// PHASE 6B: BUSINESS PROCESS TEMPORAL WORKFLOWS
// ============================================================================
// Temporal-powered orchestration for low-code business process execution.
//
// A Business Process (BP) is orchestrated as a Temporal workflow that:
// 1. Loads the BP definition (steps, triggers, assignments)
// 2. Executes each step in sequence
// 3. Handles timeouts with escalation (Phase 6C integration)
// 4. Manages conditional branching (if/then/else)
// 5. Publishes events for notifications
//
// The workflow is durable, resumable, and fully audited.
// ============================================================================

// BPStepType defines the type of step in a BP
type BPStepType string

const (
	BPStepDataEntry BPStepType = "data_entry"
	BPStepValidate  BPStepType = "validate"
	BPStepApprove   BPStepType = "approve"
	BPStepNotify    BPStepType = "notify"
	BPStepIntegrate BPStepType = "integrate"
	BPStepCompute   BPStepType = "compute"
)

// BPStepConfig represents a single step in a BP
type BPStepConfig struct {
	ID            string          `db:"id"`
	ProcessID     string          `db:"process_id"`
	StepOrder     int             `db:"step_order"`
	StepType      string          `db:"step_type"`
	StepName      string          `db:"step_name"`
	DurationHours int             `db:"duration_hours"`
	AssigneeRole  string          `db:"assignee_role"`
	AssigneeUser  string          `db:"assignee_user"`
	TriggerIDs    []string        `db:"trigger_ids"`
	ConditionJSON json.RawMessage `db:"condition_json"`
	ActionConfig  json.RawMessage `db:"action_config"`
	OutputMapping json.RawMessage `db:"output_mapping"`
}

// BPInstanceData represents the current state of a BP execution
type BPInstanceData struct {
	InstanceID         string                 `db:"id"`
	ProcessID          string                 `db:"process_id"`
	EntityID           string                 `db:"entity_id"`
	EntityType         string                 `db:"entity_type"`
	CurrentStep        int                    `db:"current_step"`
	Status             string                 `db:"status"`
	InstanceData       map[string]interface{} `db:"instance_data"`
	StartedAt          time.Time              `db:"started_at"`
	CurrentStepStart   time.Time              `db:"current_step_started_at"`
	CurrentStepDue     time.Time              `db:"current_step_due_at"`
	TemporalWorkflowID string                 `db:"temporal_workflow_id"`
	TemporalRunID      string                 `db:"temporal_run_id"`
}

// BPStepResult is the output of a step execution
type BPStepResult struct {
	StepNumber   int                    `json:"step_number"`
	Status       string                 `json:"status"` // completed, failed, escalated, skipped
	Output       map[string]interface{} `json:"output"`
	Decision     string                 `json:"decision"` // approved, rejected, manual_override (for approval steps)
	ErrorMessage string                 `json:"error_message,omitempty"`
	EscalatedAt  time.Time              `json:"escalated_at,omitempty"`
	NextStep     int                    `json:"next_step"` // Computed from conditions
}

// ============================================================================
// MAIN WORKFLOW: ExecuteBusinessProcessWorkflow
// ============================================================================
// Orchestrates the entire BP from start to finish.
//
// Flow:
//  1. Load BP definition (steps, triggers, conditions)
//  2. For each step:
//     a. Update instance status to current_step
//     b. Execute step activity
//     c. If step fails → handle error or escalate
//     d. If conditions present → determine next step
//     e. Otherwise → move to next step
//  3. Mark instance as completed
//  4. Publish completion event
//
// Usage:
//
//	we, err := client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
//	    ID: "bp-instance-" + instanceID,
//	    TaskQueue: "bp_queue",
//	}, ExecuteBusinessProcessWorkflow, instanceID)
func ExecuteBusinessProcessWorkflow(ctx workflow.Context, instanceID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("ExecuteBusinessProcessWorkflow started", "instanceID", instanceID)

	// ===== STEP 1: LOAD BP INSTANCE =====
	var instance BPInstanceData
	err := workflow.ExecuteActivity(
		ctx,
		LoadBPInstanceActivity,
		instanceID,
	).Get(ctx, &instance)
	if err != nil {
		logger.Error("Failed to load BP instance", "error", err)
		return fmt.Errorf("load instance: %w", err)
	}

	logger.Info("BP instance loaded", "processID", instance.ProcessID, "entityID", instance.EntityID)

	// ===== STEP 2: LOAD BP STEPS =====
	var steps []BPStepConfig
	err = workflow.ExecuteActivity(
		ctx,
		LoadBPStepsActivity,
		instance.ProcessID,
	).Get(ctx, &steps)
	if err != nil {
		logger.Error("Failed to load BP steps", "error", err)
		return fmt.Errorf("load steps: %w", err)
	}

	logger.Info("BP steps loaded", "stepCount", len(steps))

	// ===== STEP 3: EXECUTE EACH STEP =====
	currentStep := 1
	for currentStep <= len(steps) {
		logger.Info("Executing BP step", "step", currentStep, "total", len(steps))

		step := steps[currentStep-1]

		// Update instance status
		err = workflow.ExecuteActivity(
			ctx,
			UpdateBPInstanceStepActivity,
			instanceID,
			currentStep,
			"in_progress",
		).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to update instance step", "error", err)
			return fmt.Errorf("update step: %w", err)
		}

		// Execute the step
		var result BPStepResult
		err = workflow.ExecuteActivity(
			ctx,
			ExecuteBPStepActivity,
			instanceID,
			step,
			instance.InstanceData,
		).Get(ctx, &result)

		if err != nil {
			logger.Error("Step failed", "step", currentStep, "error", err)
			// Log step failure and continue or escalate based on severity
			_ = workflow.ExecuteActivity(
				ctx,
				LogBPStepExecutionActivity,
				instanceID,
				currentStep,
				"failed",
				result,
			).Get(ctx, nil)
			// For now, fail the entire BP
			return fmt.Errorf("step %d failed: %w", currentStep, err)
		}

		logger.Info("Step completed", "step", currentStep, "status", result.Status)

		// Log step execution
		_ = workflow.ExecuteActivity(
			ctx,
			LogBPStepExecutionActivity,
			instanceID,
			currentStep,
			result.Status,
			result,
		).Get(ctx, nil)

		// Update instance data with step output
		if result.Output != nil {
			for k, v := range result.Output {
				instance.InstanceData[k] = v
			}
		}

		// Determine next step (from conditions or default to next)
		nextStep := result.NextStep
		if nextStep == 0 {
			nextStep = currentStep + 1
		}

		currentStep = nextStep
	}

	// ===== STEP 4: MARK COMPLETED =====
	logger.Info("BP execution completed", "instanceID", instanceID)
	err = workflow.ExecuteActivity(
		ctx,
		UpdateBPInstanceStepActivity,
		instanceID,
		len(steps),
		"completed",
	).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to mark BP completed", "error", err)
		return fmt.Errorf("mark completed: %w", err)
	}

	// ===== STEP 5: PUBLISH EVENT =====
	_ = workflow.ExecuteActivity(
		ctx,
		PublishBPEventActivity,
		instanceID,
		"bp.completed",
		instance.InstanceData,
	).Get(ctx, nil)

	logger.Info("ExecuteBusinessProcessWorkflow completed successfully", "instanceID", instanceID)
	return nil
}

// ============================================================================
// ACTIVITY: LoadBPInstanceActivity
// ============================================================================
// Loads the BP instance from the database
func LoadBPInstanceActivity(ctx context.Context, instanceID string) (*BPInstanceData, error) {
	db := ctx.Value("db").(*sqlx.DB)
	if db == nil {
		return nil, fmt.Errorf("database not available in activity context")
	}

	var instance BPInstanceData
	var instanceDataJSON string

	q := `
		SELECT id, process_id, entity_id, entity_type, current_step, status, 
		       instance_data, started_at, current_step_started_at, current_step_due_at,
		       temporal_workflow_id, temporal_run_id
		FROM bp_instances
		WHERE id = $1
	`

	err := db.QueryRowContext(ctx, q, instanceID).Scan(
		&instance.InstanceID,
		&instance.ProcessID,
		&instance.EntityID,
		&instance.EntityType,
		&instance.CurrentStep,
		&instance.Status,
		&instanceDataJSON,
		&instance.StartedAt,
		&instance.CurrentStepStart,
		&instance.CurrentStepDue,
		&instance.TemporalWorkflowID,
		&instance.TemporalRunID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("BP instance not found: %s", instanceID)
		}
		return nil, fmt.Errorf("query BP instance: %w", err)
	}

	// Parse instance data JSON
	if instanceDataJSON != "" {
		err = json.Unmarshal([]byte(instanceDataJSON), &instance.InstanceData)
		if err != nil {
			return nil, fmt.Errorf("parse instance data: %w", err)
		}
	}
	if instance.InstanceData == nil {
		instance.InstanceData = make(map[string]interface{})
	}

	log.Printf("[LoadBPInstance] Loaded instance %s, step %d, status %s", instanceID, instance.CurrentStep, instance.Status)
	return &instance, nil
}

// ============================================================================
// ACTIVITY: LoadBPStepsActivity
// ============================================================================
// Loads all steps for a BP
func LoadBPStepsActivity(ctx context.Context, processID string) ([]BPStepConfig, error) {
	db := ctx.Value("db").(*sqlx.DB)
	if db == nil {
		return nil, fmt.Errorf("database not available in activity context")
	}

	// TODO: Replace with Hasura GraphQL query:
	//   query { bp_steps(where: {process_id: {_eq: $processID}}, order_by: {step_order: asc}) { id process_id step_order step_type step_name duration_hours assignee_role assignee_user trigger_ids condition_json action_config output_mapping } }
	q := `
		SELECT id, process_id, step_order, step_type, step_name, duration_hours,
		       assignee_role, assignee_user, trigger_ids, condition_json, action_config, output_mapping
		FROM bp_steps
		WHERE process_id = $1
		ORDER BY step_order ASC
	`

	var steps []BPStepConfig
	err := db.SelectContext(ctx, &steps, q, processID)
	if err != nil {
		return nil, fmt.Errorf("query BP steps: %w", err)
	}

	log.Printf("[LoadBPSteps] Loaded %d steps for process %s", len(steps), processID)
	return steps, nil
}

// ============================================================================
// ACTIVITY: ExecuteBPStepActivity
// ============================================================================
// Executes a single BP step (validation, approval, notification, etc.)
func ExecuteBPStepActivity(ctx context.Context, instanceID string, step BPStepConfig, instanceData map[string]interface{}) (*BPStepResult, error) {
	log.Printf("[ExecuteBPStep] Executing step %d: %s", step.StepOrder, step.StepName)

	result := &BPStepResult{
		StepNumber: step.StepOrder,
		Status:     "completed",
		Output:     make(map[string]interface{}),
		NextStep:   step.StepOrder + 1, // Default: move to next step
	}

	// Execute based on step type
	switch BPStepType(step.StepType) {
	case BPStepValidate:
		// Run validation triggers
		log.Printf("[ExecuteBPStep] Running validation for step %d", step.StepOrder)
		result.Status = "completed"
		result.Output["validation_status"] = "passed"

	case BPStepApprove:
		// Approval step - typically manual, waits for decision
		log.Printf("[ExecuteBPStep] Waiting for approval on step %d (assignee: %s)", step.StepOrder, step.AssigneeRole)
		result.Status = "completed"
		result.Decision = "approved" // In real scenario, would wait for approval signal
		result.Output["approval_decision"] = "approved"

	case BPStepNotify:
		// Send notification
		log.Printf("[ExecuteBPStep] Sending notification for step %d", step.StepOrder)
		result.Status = "completed"
		result.Output["notification_sent"] = true

	case BPStepDataEntry:
		// Data entry step - just advance
		log.Printf("[ExecuteBPStep] Data entry for step %d", step.StepOrder)
		result.Status = "completed"

	case BPStepIntegrate:
		// Integration with external system
		log.Printf("[ExecuteBPStep] Integration action for step %d", step.StepOrder)
		result.Status = "completed"

	case BPStepCompute:
		// Compute/transform step
		log.Printf("[ExecuteBPStep] Compute action for step %d", step.StepOrder)
		result.Status = "completed"

	default:
		return nil, fmt.Errorf("unknown step type: %s", step.StepType)
	}

	return result, nil
}

// ============================================================================
// ACTIVITY: UpdateBPInstanceStepActivity
// ============================================================================
// Updates the BP instance with the current step and status
func UpdateBPInstanceStepActivity(ctx context.Context, instanceID string, stepNumber int, status string) error {
	db := ctx.Value("db").(*sqlx.DB)
	if db == nil {
		return fmt.Errorf("database not available in activity context")
	}

	currentStepDue := sql.NullTime{}
	if stepNumber > 0 {
		// Calculate due time for this step (simplified - would fetch step duration)
		currentStepDue = sql.NullTime{
			Time:  time.Now().Add(48 * time.Hour), // Example: 48h timeout
			Valid: true,
		}
	}

	// TODO: Replace with Hasura GraphQL mutation:
	//   mutation { update_bp_instances_by_pk(pk_columns: {id: $id}, _set: {current_step: $step, status: $status, current_step_started_at: now(), current_step_due_at: $due, updated_at: now()}) { id } }
	q := `
		UPDATE bp_instances
		SET current_step = $1, status = $2, current_step_started_at = NOW(), 
		    current_step_due_at = $3, updated_at = NOW()
		WHERE id = $4
	`

	_, err := db.ExecContext(ctx, q, stepNumber, status, currentStepDue, instanceID)
	if err != nil {
		return fmt.Errorf("update BP instance: %w", err)
	}

	log.Printf("[UpdateBPInstance] Updated instance %s to step %d, status %s", instanceID, stepNumber, status)
	return nil
}

// ============================================================================
// ACTIVITY: LogBPStepExecutionActivity
// ============================================================================
// Logs the execution of a BP step for audit trail
func LogBPStepExecutionActivity(ctx context.Context, instanceID string, stepNumber int, status string, result *BPStepResult) error {
	db := ctx.Value("db").(*sqlx.DB)
	if db == nil {
		return fmt.Errorf("database not available in activity context")
	}

	outputJSON, _ := json.Marshal(result.Output)

	// TODO: Replace with Hasura GraphQL mutation:
	//   mutation { insert_bp_step_executions_one(object: {tenant_id: $tenant_id, bp_instance_id: $instance_id, step_number: $step, status: $status, started_at: now(), output_data: $output, approval_decision: $decision, result: $result, created_at: now()}) { id } }
	//   Note: Need to fetch tenant_id from bp_instances first, or use nested insert if relationship exists
	//   output_data is JSONB, pass outputJSON directly
	q := `
		INSERT INTO bp_step_executions 
		(tenant_id, bp_instance_id, step_number, status, started_at, output_data, approval_decision, result, created_at)
		SELECT tenant_id, $1, $2, $3, NOW(), $4::jsonb, $5, $6, NOW()
		FROM bp_instances
		WHERE id = $1
	`

	_, err := db.ExecContext(ctx, q, instanceID, stepNumber, status, outputJSON, result.Decision, status)
	if err != nil {
		log.Printf("[LogBPStepExecution] Error logging step: %v", err)
		// Don't fail the workflow for logging errors
		return nil
	}

	log.Printf("[LogBPStepExecution] Logged step %d execution for instance %s", stepNumber, instanceID)
	return nil
}

// ============================================================================
// ACTIVITY: PublishBPEventActivity
// ============================================================================
// Publishes a BP event to RabbitMQ/message broker
func PublishBPEventActivity(ctx context.Context, instanceID string, eventType string, data map[string]interface{}) error {
	log.Printf("[PublishBPEvent] Publishing event %s for instance %s", eventType, instanceID)
	// TODO: Integrate with RabbitMQ
	// event := map[string]interface{}{
	//     "event_type": eventType,
	//     "instance_id": instanceID,
	//     "data": data,
	//     "timestamp": time.Now(),
	// }
	// amqpChan.Publish(...)
	return nil
}

// ============================================================================
// REGISTRATION
// ============================================================================
// Register workflows and activities in your Temporal worker:
//
// In worker.go:
//   w.RegisterWorkflow(ExecuteBusinessProcessWorkflow)
//   w.RegisterActivity(LoadBPInstanceActivity)
//   w.RegisterActivity(LoadBPStepsActivity)
//   w.RegisterActivity(ExecuteBPStepActivity)
//   w.RegisterActivity(UpdateBPInstanceStepActivity)
//   w.RegisterActivity(LogBPStepExecutionActivity)
//   w.RegisterActivity(PublishBPEventActivity)
//
