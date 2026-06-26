package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Audit Service - Workday-Inspired Audit & Explainability Layer
// ============================================================================

// AuditEventType defines types of audit events
type AuditEventType string

const (
	AuditStepStarted        AuditEventType = "STEP_STARTED"
	AuditStepCompleted      AuditEventType = "STEP_COMPLETED"
	AuditStepFailed         AuditEventType = "STEP_FAILED"
	AuditRoutingResolved    AuditEventType = "ROUTING_RESOLVED"
	AuditConditionEvaluated AuditEventType = "CONDITION_EVALUATED"
	AuditHumanDecision      AuditEventType = "HUMAN_DECISION"
	AuditLLMInvocation      AuditEventType = "LLM_INVOCATION"
	AuditPolicyCheck        AuditEventType = "POLICY_CHECK"
	AuditEscalation         AuditEventType = "ESCALATION"
	AuditCompensation       AuditEventType = "COMPENSATION"
	AuditWorkflowStarted    AuditEventType = "WORKFLOW_STARTED"
	AuditWorkflowCompleted  AuditEventType = "WORKFLOW_COMPLETED"
)

// AuditEvent represents a single audit trail entry
type AuditEvent struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	WorkflowID     string                 `json:"workflow_id"`
	RunID          string                 `json:"run_id"`
	InstanceID     string                 `json:"instance_id"`
	StepID         string                 `json:"step_id"`
	StepType       string                 `json:"step_type"`
	EventType      AuditEventType         `json:"event_type"`
	Timestamp      time.Time              `json:"timestamp"`
	ActorUserID    string                 `json:"actor_user_id,omitempty"`
	ActorType      string                 `json:"actor_type"` // "system", "user", "llm"
	Inputs         map[string]interface{} `json:"inputs,omitempty"`
	Outputs        map[string]interface{} `json:"outputs,omitempty"`
	LLMReasoning   *LLMReasoningSnapshot  `json:"llm_reasoning,omitempty"`
	RoutingTrace   *RoutingTrace          `json:"routing_trace,omitempty"`
	ConditionTrace *ConditionTrace        `json:"condition_trace,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Duration       time.Duration          `json:"duration,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// LLMReasoningSnapshot captures LLM interaction for audit
type LLMReasoningSnapshot struct {
	ProfileID        string                 `json:"profile_id"`
	PromptTemplate   string                 `json:"prompt_template"`
	PromptFilled     string                 `json:"prompt_filled"`
	InputSnapshot    map[string]interface{} `json:"input_snapshot"`
	OutputRaw        string                 `json:"output_raw"`
	OutputProcessed  interface{}            `json:"output_processed"`
	SafetyFlags      []string               `json:"safety_flags,omitempty"`
	PolicyViolations []string               `json:"policy_violations,omitempty"`
	ModelName        string                 `json:"model_name"`
	TokensUsed       int                    `json:"tokens_used,omitempty"`
}

// AuditService handles audit event recording
type AuditService struct {
	events   []AuditEvent
	buffer   chan AuditEvent
	tenantID string
}

// ============================================================================
// Audit Service Functions
// ============================================================================

// NewAuditService creates a new audit service
func NewAuditService(tenantID string) *AuditService {
	return &AuditService{
		events:   []AuditEvent{},
		buffer:   make(chan AuditEvent, 1000),
		tenantID: tenantID,
	}
}

// RecordEvent records an audit event within a workflow
func RecordEvent(
	ctx workflow.Context,
	event AuditEvent,
) error {
	logger := workflow.GetLogger(ctx)

	// Set workflow context
	info := workflow.GetInfo(ctx)
	event.WorkflowID = info.WorkflowExecution.ID
	event.RunID = info.WorkflowExecution.RunID
	event.Timestamp = workflow.Now(ctx)

	if event.ID == "" {
		event.ID = fmt.Sprintf("audit_%d", workflow.Now(ctx).UnixNano())
	}

	// Fire and forget via local activity
	activityOptions := workflow.LocalActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	}
	lctx := workflow.WithLocalActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteLocalActivity(lctx, RecordAuditEventActivity, event).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to record audit event", "error", err, "eventType", event.EventType)
		// Don't fail workflow for audit failures
	}

	return nil
}

// RecordAuditEventActivity persists the audit event
func RecordAuditEventActivity(ctx context.Context, event AuditEvent) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Recording audit event",
		"eventType", event.EventType,
		"workflowID", event.WorkflowID,
		"stepID", event.StepID,
	)

	// TODO: Persist to database (PostgreSQL, StarRocks, etc.)
	// For now, log it
	eventJSON, _ := json.Marshal(event)
	logger.Debug("Audit event", "event", string(eventJSON))

	return nil
}

// ============================================================================
// Convenience Functions for Recording Specific Events
// ============================================================================

// RecordStepStarted records a step starting
func RecordStepStarted(ctx workflow.Context, stepID, stepType string, inputs map[string]interface{}) error {
	return RecordEvent(ctx, AuditEvent{
		StepID:    stepID,
		StepType:  stepType,
		EventType: AuditStepStarted,
		ActorType: "system",
		Inputs:    inputs,
	})
}

// RecordStepCompleted records a step completing
func RecordStepCompleted(ctx workflow.Context, stepID, stepType string, outputs map[string]interface{}, duration time.Duration) error {
	return RecordEvent(ctx, AuditEvent{
		StepID:    stepID,
		StepType:  stepType,
		EventType: AuditStepCompleted,
		ActorType: "system",
		Outputs:   outputs,
		Duration:  duration,
	})
}

// RecordStepFailed records a step failure
func RecordStepFailed(ctx workflow.Context, stepID, stepType string, err error) error {
	return RecordEvent(ctx, AuditEvent{
		StepID:    stepID,
		StepType:  stepType,
		EventType: AuditStepFailed,
		ActorType: "system",
		Error:     err.Error(),
	})
}

// RecordRoutingResolved records routing resolution
func RecordRoutingResolved(ctx workflow.Context, stepID string, trace *RoutingTrace) error {
	return RecordEvent(ctx, AuditEvent{
		StepID:       stepID,
		EventType:    AuditRoutingResolved,
		ActorType:    "system",
		RoutingTrace: trace,
	})
}

// RecordConditionEvaluated records condition evaluation
func RecordConditionEvaluated(ctx workflow.Context, stepID string, trace *ConditionTrace) error {
	return RecordEvent(ctx, AuditEvent{
		StepID:         stepID,
		EventType:      AuditConditionEvaluated,
		ActorType:      "system",
		ConditionTrace: trace,
	})
}

// RecordHumanDecision records a human decision
func RecordHumanDecision(ctx workflow.Context, stepID string, userID string, action string, formData map[string]interface{}) error {
	return RecordEvent(ctx, AuditEvent{
		StepID:      stepID,
		EventType:   AuditHumanDecision,
		ActorUserID: userID,
		ActorType:   "user",
		Outputs: map[string]interface{}{
			"action":    action,
			"form_data": formData,
		},
	})
}

// RecordLLMInvocation records an LLM invocation
func RecordLLMInvocation(ctx workflow.Context, stepID string, reasoning *LLMReasoningSnapshot) error {
	return RecordEvent(ctx, AuditEvent{
		StepID:       stepID,
		EventType:    AuditLLMInvocation,
		ActorType:    "llm",
		LLMReasoning: reasoning,
	})
}

// RecordPolicyCheck records a policy check
func RecordPolicyCheck(ctx workflow.Context, stepID string, policyRef string, passed bool, details map[string]interface{}) error {
	return RecordEvent(ctx, AuditEvent{
		StepID:    stepID,
		EventType: AuditPolicyCheck,
		ActorType: "system",
		Outputs: map[string]interface{}{
			"policy_ref": policyRef,
			"passed":     passed,
			"details":    details,
		},
	})
}

// RecordEscalation records an escalation event
func RecordEscalation(ctx workflow.Context, stepID string, reason string, newAssignees []Assignee) error {
	return RecordEvent(ctx, AuditEvent{
		StepID:    stepID,
		EventType: AuditEscalation,
		ActorType: "system",
		Metadata: map[string]interface{}{
			"reason":        reason,
			"new_assignees": newAssignees,
		},
	})
}

// RecordWorkflowStarted records workflow start
func RecordWorkflowStarted(ctx workflow.Context, instanceID string, inputs map[string]interface{}) error {
	return RecordEvent(ctx, AuditEvent{
		InstanceID: instanceID,
		EventType:  AuditWorkflowStarted,
		ActorType:  "system",
		Inputs:     inputs,
	})
}

// RecordWorkflowCompleted records workflow completion
func RecordWorkflowCompleted(ctx workflow.Context, instanceID string, status string, outputs map[string]interface{}) error {
	return RecordEvent(ctx, AuditEvent{
		InstanceID: instanceID,
		EventType:  AuditWorkflowCompleted,
		ActorType:  "system",
		Outputs: map[string]interface{}{
			"status":  status,
			"outputs": outputs,
		},
	})
}

// ============================================================================
// Audit Querying (for Support Console)
// ============================================================================

// AuditQuery defines parameters for querying audit events
type AuditQuery struct {
	TenantID   string           `json:"tenant_id"`
	WorkflowID string           `json:"workflow_id,omitempty"`
	InstanceID string           `json:"instance_id,omitempty"`
	StepID     string           `json:"step_id,omitempty"`
	EventTypes []AuditEventType `json:"event_types,omitempty"`
	StartTime  time.Time        `json:"start_time,omitempty"`
	EndTime    time.Time        `json:"end_time,omitempty"`
	ActorID    string           `json:"actor_id,omitempty"`
	Limit      int              `json:"limit,omitempty"`
	Offset     int              `json:"offset,omitempty"`
}

// AuditQueryResult holds query results
type AuditQueryResult struct {
	Events     []AuditEvent `json:"events"`
	TotalCount int          `json:"total_count"`
	HasMore    bool         `json:"has_more"`
}

// GetAuditEvents retrieves audit events based on query
func GetAuditEvents(ctx context.Context, query AuditQuery) (*AuditQueryResult, error) {
	// TODO: Implement database query
	return &AuditQueryResult{
		Events:     []AuditEvent{},
		TotalCount: 0,
		HasMore:    false,
	}, nil
}

// GetStepAuditTrail retrieves complete audit trail for a step
func GetStepAuditTrail(ctx context.Context, workflowID, stepID string) ([]AuditEvent, error) {
	query := AuditQuery{
		WorkflowID: workflowID,
		StepID:     stepID,
	}
	result, err := GetAuditEvents(ctx, query)
	if err != nil {
		return nil, err
	}
	return result.Events, nil
}

// GetLLMReasoningHistory retrieves all LLM invocations for a workflow
func GetLLMReasoningHistory(ctx context.Context, workflowID string) ([]AuditEvent, error) {
	query := AuditQuery{
		WorkflowID: workflowID,
		EventTypes: []AuditEventType{AuditLLMInvocation},
	}
	result, err := GetAuditEvents(ctx, query)
	if err != nil {
		return nil, err
	}
	return result.Events, nil
}

// GetHumanDecisionHistory retrieves all human decisions for a workflow
func GetHumanDecisionHistory(ctx context.Context, workflowID string) ([]AuditEvent, error) {
	query := AuditQuery{
		WorkflowID: workflowID,
		EventTypes: []AuditEventType{AuditHumanDecision},
	}
	result, err := GetAuditEvents(ctx, query)
	if err != nil {
		return nil, err
	}
	return result.Events, nil
}
