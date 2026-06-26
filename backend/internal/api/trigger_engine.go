package api

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
// TRIGGER ENGINE - Generic, Zero Hard-Coded Logic
// ============================================================================

type TriggerEngine struct {
	db              *sqlx.DB
	abacEngine      *ABACEngine
	eventBus        EventBus
	notificationSvc NotificationService
}

// TriggerContext wraps evaluation data
type TriggerContext struct {
	TenantID     string                 `json:"tenant_id"`
	UserID       string                 `json:"user_id"`
	TriggerKey   string                 `json:"trigger_key"`
	TargetEntity string                 `json:"target_entity"`
	EntityID     string                 `json:"entity_id"`
	EventData    map[string]interface{} `json:"event_data"`
	ClientIP     string                 `json:"client_ip"`
	UserAgent    string                 `json:"user_agent"`
	RequestedAt  time.Time              `json:"requested_at"`
}

// TriggerConfig represents a validation trigger from DB
type TriggerConfig struct {
	ID              string          `db:"id" json:"id"`
	TriggerTypeID   string          `db:"trigger_type_id" json:"trigger_type_id"`
	TargetEntity    string          `db:"target_entity" json:"target_entity"`
	EventConfig     json.RawMessage `db:"event_config" json:"event_config"`
	ConditionConfig json.RawMessage `db:"condition_config" json:"condition_config"`
	ActionConfig    json.RawMessage `db:"action_config" json:"action_config"`
	ABACPolicyID    *string         `db:"abac_policy_id" json:"abac_policy_id"`
	Enabled         bool            `db:"enabled" json:"enabled"`
	Priority        int             `db:"priority" json:"priority"`
}

// RuleCondition represents a single rule in condition_config
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// ActionConfig represents post-commit actions
type ActionConfig struct {
	Type           string                 `json:"type"` // temporal, rabbitmq, notification, webhook
	WorkflowID     string                 `json:"workflow_id,omitempty"`
	NotificationID string                 `json:"notification_id,omitempty"`
	WebhookURL     string                 `json:"webhook_url,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionResult tracks trigger execution
type ExecutionResult struct {
	TriggerID        string                 `json:"trigger_id"`
	Status           string                 `json:"status"` // success, blocked, error
	ConditionsMet    bool                   `json:"conditions_met"`
	ABACAllowed      bool                   `json:"abac_allowed"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	EvaluationResult map[string]interface{} `json:"evaluation_result"`
	ActionResult     map[string]interface{} `json:"action_result"`
	DurationMs       int64                  `json:"duration_ms"`
}

// NewTriggerEngine creates a new trigger engine
func NewTriggerEngine(db *sqlx.DB, abac *ABACEngine, eventBus EventBus, notifSvc NotificationService) *TriggerEngine {
	return &TriggerEngine{
		db:              db,
		abacEngine:      abac,
		eventBus:        eventBus,
		notificationSvc: notifSvc,
	}
}

// ============================================================================
// CORE EVALUATION LOGIC
// ============================================================================

// EvaluateTriggers is the main entry point - evaluates all triggers for a given type
func (e *TriggerEngine) EvaluateTriggers(ctx context.Context, tc *TriggerContext) ([]ExecutionResult, error) {
	start := time.Now()
	results := []ExecutionResult{}

	// 1. Fetch all enabled triggers for this type + entity
	query := `
		SELECT vt.id, vt.trigger_type_id, vt.target_entity, 
		       vt.event_config, vt.condition_config, vt.action_config,
		       vt.abac_policy_id, vt.enabled, vt.priority
		FROM validation_triggers vt
		JOIN trigger_types tt ON vt.trigger_type_id = tt.id
		WHERE vt.tenant_id = $1 
		  AND tt.key = $2 
		  AND vt.target_entity = $3 
		  AND vt.enabled = true
		ORDER BY vt.priority ASC`

	triggers := []TriggerConfig{}
	err := e.db.SelectContext(ctx, &triggers, query, tc.TenantID, tc.TriggerKey, tc.TargetEntity)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ERROR] Failed to fetch triggers: %v", err)
		return results, err
	}

	// 2. Evaluate each trigger in priority order
	var blockingError error
	for _, trigger := range triggers {
		execResult := ExecutionResult{
			TriggerID: trigger.ID,
		}

		// 2a. Evaluate conditions
		conditionsMet, conditionResult, err := e.evaluateConditions(trigger.ConditionConfig, tc.EventData)
		if err != nil {
			execResult.Status = "error"
			execResult.ErrorMessage = fmt.Sprintf("Condition evaluation failed: %v", err)
			execResult.EvaluationResult = conditionResult
			results = append(results, execResult)
			continue
		}

		execResult.ConditionsMet = conditionsMet
		execResult.EvaluationResult = conditionResult

		// 2b. If conditions not met, skip to next trigger
		if !conditionsMet {
			execResult.Status = "blocked"
			execResult.ErrorMessage = "Conditions not met"
			results = append(results, execResult)
			continue
		}

		// 2c. ABAC evaluation
		abacAllowed := true
		if trigger.ABACPolicyID != nil {
			abacAllowed = e.abacEngine.Evaluate(ctx, &ABACContext{
				TenantID:  tc.TenantID,
				SubjectID: tc.UserID,
				Action:    fmt.Sprintf("execute_trigger:%s", tc.TriggerKey),
				Resource:  tc.TargetEntity,
				PolicyID:  *trigger.ABACPolicyID,
				ClientIP:  tc.ClientIP,
				Time:      tc.RequestedAt,
			})
		}

		execResult.ABACAllowed = abacAllowed
		if !abacAllowed {
			execResult.Status = "blocked"
			execResult.ErrorMessage = "ABAC policy denied"
			results = append(results, execResult)
			continue
		}

		// 2d. Execute post-commit actions
		actionResult, err := e.executeActions(ctx, trigger.ActionConfig, tc, trigger.ID)
		if err != nil {
			execResult.Status = "error"
			execResult.ErrorMessage = fmt.Sprintf("Action execution failed: %v", err)
			execResult.ActionResult = actionResult
			results = append(results, execResult)
			blockingError = err
			continue
		}

		execResult.Status = "success"
		execResult.ActionResult = actionResult
		results = append(results, execResult)
	}

	// 3. Log to audit_log
	execDuration := time.Since(start).Milliseconds()
	e.auditTriggerExecution(ctx, tc, results, execDuration)

	return results, blockingError
}

// ============================================================================
// CONDITION EVALUATION (Rule Engine)
// ============================================================================

func (e *TriggerEngine) evaluateConditions(conditionConfig json.RawMessage, eventData map[string]interface{}) (bool, map[string]interface{}, error) {
	result := map[string]interface{}{
		"rules": []map[string]interface{}{},
		"met":   true,
	}

	if len(conditionConfig) == 0 {
		return true, result, nil // No conditions = always pass
	}

	var conditions []RuleCondition
	if err := json.Unmarshal(conditionConfig, &conditions); err != nil {
		return false, result, fmt.Errorf("invalid condition config: %w", err)
	}

	ruleResults := []map[string]interface{}{}
	for _, cond := range conditions {
		ruleResult := e.evaluateRule(cond, eventData)
		ruleResults = append(ruleResults, ruleResult)

		// All rules must pass (AND logic)
		if !ruleResult["passed"].(bool) {
			result["met"] = false
		}
	}

	result["rules"] = ruleResults
	return result["met"].(bool), result, nil
}

// evaluateRule checks a single rule
func (e *TriggerEngine) evaluateRule(rule RuleCondition, data map[string]interface{}) map[string]interface{} {
	fieldValue, exists := data[rule.Field]
	result := map[string]interface{}{
		"field":    rule.Field,
		"operator": rule.Operator,
		"value":    rule.Value,
		"passed":   false,
		"reason":   "",
	}

	if !exists {
		result["reason"] = "field not in event data"
		return result
	}

	// Operator evaluation
	switch rule.Operator {
	case "equals":
		result["passed"] = fieldValue == rule.Value
		if !result["passed"].(bool) {
			result["reason"] = fmt.Sprintf("%v != %v", fieldValue, rule.Value)
		}

	case "notEquals":
		result["passed"] = fieldValue != rule.Value

	case "greaterThan":
		fv, ok := toFloat64(fieldValue)
		rv, ok2 := toFloat64(rule.Value)
		if ok && ok2 {
			result["passed"] = fv > rv
		} else {
			result["reason"] = "non-numeric comparison"
		}

	case "lessThan":
		fv, ok := toFloat64(fieldValue)
		rv, ok2 := toFloat64(rule.Value)
		if ok && ok2 {
			result["passed"] = fv < rv
		} else {
			result["reason"] = "non-numeric comparison"
		}

	case "greaterThanOrEqual":
		fv, ok := toFloat64(fieldValue)
		rv, ok2 := toFloat64(rule.Value)
		if ok && ok2 {
			result["passed"] = fv >= rv
		}

	case "lessThanOrEqual":
		fv, ok := toFloat64(fieldValue)
		rv, ok2 := toFloat64(rule.Value)
		if ok && ok2 {
			result["passed"] = fv <= rv
		}

	case "contains":
		fvStr := fmt.Sprint(fieldValue)
		rvStr := fmt.Sprint(rule.Value)
		result["passed"] = len(rvStr) > 0 && contains(fvStr, rvStr)

	case "inList":
		// rule.Value should be []interface{}
		if list, ok := rule.Value.([]interface{}); ok {
			result["passed"] = contains(fmt.Sprint(fieldValue), fmt.Sprint(list))
		}

	case "isEmpty":
		result["passed"] = fieldValue == nil || fieldValue == "" || fieldValue == 0

	case "isNotEmpty":
		result["passed"] = fieldValue != nil && fieldValue != "" && fieldValue != 0

	case "isTrue":
		if b, ok := fieldValue.(bool); ok {
			result["passed"] = b
		}

	case "isFalse":
		if b, ok := fieldValue.(bool); ok {
			result["passed"] = !b
		}

	default:
		result["reason"] = fmt.Sprintf("unknown operator: %s", rule.Operator)
	}

	return result
}

// ============================================================================
// ACTION EXECUTION (Post-Commit)
// ============================================================================

func (e *TriggerEngine) executeActions(ctx context.Context, actionConfig json.RawMessage, tc *TriggerContext, _ string) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"actions": []map[string]interface{}{},
	}

	if len(actionConfig) == 0 {
		return result, nil // No actions = success
	}

	var actions []ActionConfig
	if err := json.Unmarshal(actionConfig, &actions); err != nil {
		return result, fmt.Errorf("invalid action config: %w", err)
	}

	actionResults := []map[string]interface{}{}
	for _, action := range actions {
		actionResult := map[string]interface{}{
			"type": action.Type,
		}

		switch action.Type {
		case "notification":
			err := e.sendNotification(ctx, action.NotificationID, tc, action.Metadata)
			if err != nil {
				actionResult["status"] = "failed"
				actionResult["error"] = err.Error()
			} else {
				actionResult["status"] = "success"
			}

		case "temporal":
			workflowID := e.startTemporalWorkflow(ctx, action.WorkflowID, tc, action.Metadata)
			actionResult["status"] = "success"
			actionResult["workflow_id"] = workflowID

		case "rabbitmq":
			err := e.emitRabbitMQEvent(ctx, tc, action.Metadata)
			if err != nil {
				actionResult["status"] = "failed"
				actionResult["error"] = err.Error()
			} else {
				actionResult["status"] = "success"
			}

		case "webhook":
			err := e.callWebhook(ctx, action.WebhookURL, tc)
			if err != nil {
				actionResult["status"] = "failed"
				actionResult["error"] = err.Error()
			} else {
				actionResult["status"] = "success"
			}

		default:
			actionResult["status"] = "unknown"
			actionResult["error"] = fmt.Sprintf("unknown action type: %s", action.Type)
		}

		actionResults = append(actionResults, actionResult)
	}

	result["actions"] = actionResults
	return result, nil
}

// ============================================================================
// ACTION HANDLERS
// ============================================================================

func (e *TriggerEngine) sendNotification(ctx context.Context, notificationID string, tc *TriggerContext, _ map[string]interface{}) error {
	// Fetch template from DB
	var template struct {
		Channel  string `db:"channel"`
		Subject  string `db:"subject"`
		Template string `db:"body_template"`
	}

	query := `SELECT channel, subject, body_template FROM notification_templates WHERE id = $1`
	if err := e.db.GetContext(ctx, &template, query, notificationID); err != nil {
		return err
	}

	// Render template
	body := renderTemplate(template.Template, tc)

	// Send via appropriate channel
	return e.notificationSvc.Send(ctx, template.Channel, &NotificationPayload{ //nolint:errcheck
		Recipients: nil, // metadata["recipients"].([]string),
		Subject:    template.Subject,
		Body:       body,
	})
}

func (e *TriggerEngine) startTemporalWorkflow(_ context.Context, workflowID string, tc *TriggerContext, _ map[string]interface{}) string {
	// TODO: Integrate with Temporal SDK
	log.Printf("[TEMPORAL] Start workflow %s with context %+v", workflowID, tc)
	return fmt.Sprintf("workflow_%d", time.Now().Unix())
}

func (e *TriggerEngine) emitRabbitMQEvent(_ context.Context, _ *TriggerContext, _ map[string]interface{}) error {
	// TODO: Emit to RabbitMQ event bus
	return nil
}

func (e *TriggerEngine) callWebhook(_ context.Context, webhookURL string, _ *TriggerContext) error {
	// TODO: HTTP POST to webhook
	log.Printf("[WEBHOOK] Call %s", webhookURL)
	return nil
}

// ============================================================================
// AUDIT & LOGGING
// ============================================================================

func (e *TriggerEngine) auditTriggerExecution(ctx context.Context, tc *TriggerContext, results []ExecutionResult, durationMs int64) {
	query := `
		INSERT INTO trigger_executions 
		(tenant_id, trigger_id, trigger_key, target_entity, entity_id, event_data, 
		 evaluation_result, action_result, status, executed_by, duration_ms, executed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	for _, result := range results {
		evaluationJSON, _ := json.Marshal(result.EvaluationResult)
		actionJSON, _ := json.Marshal(result.ActionResult)
		eventJSON, _ := json.Marshal(tc.EventData)

		_, err := e.db.ExecContext(ctx, query,
			tc.TenantID, result.TriggerID, tc.TriggerKey, tc.TargetEntity, tc.EntityID,
			eventJSON, evaluationJSON, actionJSON, result.Status, tc.UserID, durationMs, time.Now())

		if err != nil {
			log.Printf("[ERROR] Failed to audit trigger execution: %v", err)
		}
	}
}

// ============================================================================
// TIMEOUT HANDLING
// ============================================================================

// ProcessTimeoutTriggers runs as a background job (Temporal worker)
func (e *TriggerEngine) ProcessTimeoutTriggers(ctx context.Context, tenantID string) error {
	query := `
		SELECT id, tenant_id, bp_execution_id, step_name, timeout_at, 
		       escalation_action, escalate_to_user, timeout_trigger_id
		FROM step_timeouts
		WHERE tenant_id = $1 AND status = 'pending' AND timeout_at <= NOW()`

	timeouts := []struct {
		ID               string    `db:"id"`
		TenantID         string    `db:"tenant_id"`
		BPExecutionID    string    `db:"bp_execution_id"`
		StepName         string    `db:"step_name"`
		TimeoutAt        time.Time `db:"timeout_at"`
		EscalationAction string    `db:"escalation_action"`
		EscalateToUser   *string   `db:"escalate_to_user"`
		TimeoutTriggerID string    `db:"timeout_trigger_id"`
	}{}

	err := e.db.SelectContext(ctx, &timeouts, query, tenantID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	for _, timeout := range timeouts {
		// Execute escalation action
		switch timeout.EscalationAction {
		case "notify":
			e.notifyManager(ctx, timeout.EscalateToUser, timeout)

		case "escalate":
			e.escalateToHierarchy(ctx, timeout.EscalateToUser)

		case "auto_approve":
			e.autoApproveStep(ctx, timeout.BPExecutionID, timeout.StepName)

		case "auto_reject":
			e.autoRejectStep(ctx, timeout.BPExecutionID, timeout.StepName)
		}

		// Mark as escalated
		_, _ = e.db.ExecContext(ctx, `
			UPDATE step_timeouts SET status = 'escalated', escalated_at = NOW() WHERE id = $1`,
			timeout.ID)
	}

	return nil
}

func (e *TriggerEngine) notifyManager(_ context.Context, userID *string, _ interface{}) {
	log.Printf("[TIMEOUT] Notify manager %s", *userID)
	// TODO: Send notification
}

func (e *TriggerEngine) escalateToHierarchy(_ context.Context, userID *string) {
	log.Printf("[TIMEOUT] Escalate to %s", *userID)
	// TODO: Escalate workflow
}

func (e *TriggerEngine) autoApproveStep(_ context.Context, bpExecutionID, stepName string) {
	log.Printf("[TIMEOUT] Auto-approve %s:%s", bpExecutionID, stepName)
	// TODO: Auto-approve
}

func (e *TriggerEngine) autoRejectStep(_ context.Context, bpExecutionID, stepName string) {
	log.Printf("[TIMEOUT] Auto-reject %s:%s", bpExecutionID, stepName)
	// TODO: Auto-reject
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func contains(str, substr string) bool {
	return len(substr) > 0 && (str == substr || len(str) >= len(substr))
}

func renderTemplate(template string, tc *TriggerContext) string {
	// Simple template rendering (can use text/template or mustache for production)
	result := template
	result += fmt.Sprintf("\nEntity: %s\nID: %s\nUser: %s", tc.TargetEntity, tc.EntityID, tc.UserID)
	return result
}

// ============================================================================
// INTERFACES (Implementations provided separately)
// ============================================================================

type EventBus interface {
	Emit(ctx context.Context, event string, data interface{}) error
}

type NotificationService interface {
	Send(ctx context.Context, channel string, payload *NotificationPayload) error
}

type NotificationPayload struct {
	Recipients []string
	Subject    string
	Body       string
}

type ABACEngine struct {
	db *sqlx.DB
}

type ABACContext struct {
	TenantID  string
	SubjectID string
	Action    string
	Resource  string
	PolicyID  string
	ClientIP  string
	Time      time.Time
}

func (a *ABACEngine) Evaluate(_ context.Context, _ *ABACContext) bool {
	// TODO: Implement ABAC evaluation
	return true
}
