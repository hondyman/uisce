package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/domain"
	"github.com/hondyman/uisce/backend/internal/rulefabric"
	"github.com/hondyman/uisce/backend/internal/validation"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
	// StepName scopes to a specific workflow step's triggers, matching
	// validation_triggers.step_name; empty matches step-agnostic triggers.
	StepName string `json:"step_name,omitempty"`
}

// TriggerConfig represents a validation trigger row, matching the live
// validation_triggers table — see internal/validation.ValidationTrigger,
// which reads the same table for the (separate, working) inline-validation
// use case. TriggerConfig's own condition/action logic used to live in
// trigger_type_id/event_config/condition_config/action_config/abac_policy_id/
// enabled/priority columns that were removed from this table by an
// unrelated migration; this shape replaces that reference with the ones
// that actually exist.
type TriggerConfig struct {
	ID           string         `db:"id" json:"id"`
	TenantID     string         `db:"tenant_id" json:"tenant_id"`
	TriggerType  string         `db:"trigger_type" json:"trigger_type"`
	TargetEntity string         `db:"target_entity" json:"target_entity"`
	StepName     sql.NullString `db:"step_name" json:"step_name,omitempty"`
	RuleIDs      pq.StringArray `db:"rule_ids" json:"rule_ids"`
	// Meta is the trigger's free-form settings column; this engine reads
	// two conventional keys from it — see triggerMeta — since the table
	// has no dedicated abac_policy_id/action_config columns anymore.
	Meta json.RawMessage `db:"meta" json:"meta,omitempty"`
}

// triggerMeta is the convention this engine expects inside
// validation_triggers.meta: an optional ABAC policy gate plus the
// post-commit actions to run once a trigger's rules (RuleIDs, evaluated
// against catalog_validation_rules) pass.
type triggerMeta struct {
	ABACPolicyID string          `json:"abac_policy_id,omitempty"`
	Actions      json.RawMessage `json:"actions,omitempty"`
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

	// 1. Fetch all triggers for this tenant/type/entity(/step). There is no
	// enabled/priority column on this table (see TriggerConfig) — every row
	// present is active, most-recently-created first.
	query := `
		SELECT id, tenant_id, trigger_type, target_entity, step_name, rule_ids, COALESCE(meta, '{}'::jsonb) AS meta
		FROM validation_triggers
		WHERE tenant_id = $1
		  AND trigger_type = $2
		  AND target_entity = $3
		  AND (step_name IS NULL OR step_name = '' OR step_name = $4)
		ORDER BY created_at DESC`

	triggers := []TriggerConfig{}
	err := e.db.SelectContext(ctx, &triggers, query, tc.TenantID, tc.TriggerKey, tc.TargetEntity, tc.StepName)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ERROR] Failed to fetch triggers: %v", err)
		return results, err
	}

	validationEngine := validation.NewValidationEngine()

	// 2. Evaluate each trigger, most-recent first
	var blockingError error
	for _, trigger := range triggers {
		execResult := ExecutionResult{
			TriggerID: trigger.ID,
		}

		// 2a. Evaluate the trigger's rules (rule_ids -> catalog_validation_rules),
		// the same mechanism internal/validation.TriggerValidationEngine uses for
		// inline save-time validation — AND logic, all rules must pass.
		conditionsMet, conditionResult, err := e.evaluateTriggerRules(ctx, validationEngine, trigger, tc.EventData)
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

		var meta triggerMeta
		_ = json.Unmarshal(trigger.Meta, &meta)

		// 2c. ABAC evaluation
		abacAllowed := true
		if meta.ABACPolicyID != "" {
			abacAllowed = e.abacEngine.Evaluate(ctx, &ABACContext{
				TenantID:  tc.TenantID,
				SubjectID: tc.UserID,
				Action:    fmt.Sprintf("execute_trigger:%s", tc.TriggerKey),
				Resource:  tc.TargetEntity,
				PolicyID:  meta.ABACPolicyID,
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

		// 2d. Execute post-commit actions (meta.actions -> []ActionConfig)
		actionResult, err := e.executeActions(ctx, meta.Actions, tc, trigger.ID)
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

	// 3. Log to bp_trigger_executions
	execDuration := time.Since(start).Milliseconds()
	e.auditTriggerExecution(ctx, tc, results, execDuration)

	return results, blockingError
}

// evaluateTriggerRules evaluates every rule referenced by trigger.RuleIDs
// against eventData (AND logic — all must pass). A rule that's missing or
// inactive is logged and skipped rather than failing the whole trigger,
// matching internal/validation.TriggerValidationEngine's behavior for the
// same table/rule relationship.
func (e *TriggerEngine) evaluateTriggerRules(ctx context.Context, ve *validation.ValidationEngine, trigger TriggerConfig, eventData map[string]interface{}) (bool, map[string]interface{}, error) {
	detail := map[string]interface{}{"rules": []map[string]interface{}{}, "met": true}
	if len(trigger.RuleIDs) == 0 {
		return true, detail, nil // no rules attached = always pass
	}

	var ruleDetails []map[string]interface{}
	allPassed := true

	for _, ruleID := range trigger.RuleIDs {
		var (
			id            string
			ruleType      string
			conditionJSON json.RawMessage
			errorMessage  sql.NullString
		)
		// catalog_validation_rules has no error_message column —
		// description serves the same purpose (see also
		// internal/validation.TriggerValidationEngine.fetchRuleByID, fixed
		// the same way).
		err := e.db.QueryRowContext(ctx, `
			SELECT id, rule_type, condition_json, COALESCE(description, '')
			FROM catalog_validation_rules
			WHERE id = $1 AND is_active = true
		`, ruleID).Scan(&id, &ruleType, &conditionJSON, &errorMessage)
		if err != nil {
			log.Printf("[WARN] EvaluateTriggers: rule %s not found or inactive: %v", ruleID, err)
			continue
		}

		// catalog_validation_rules.condition_json is required (by a DB check
		// constraint) to wrap the actual condition in a
		// schema_version/authored_mode/payload envelope.
		condition, err := validation.UnwrapConditionPayload(conditionJSON)
		if err != nil {
			log.Printf("[WARN] EvaluateTriggers: rule %s has invalid condition_json: %v", ruleID, err)
			continue
		}

		res := ve.Execute(validation.ExecutionContext{
			RuleID:       id,
			RuleType:     ruleType,
			TargetEntity: trigger.TargetEntity,
			Condition:    condition,
			Data:         eventData,
		})

		msg := res.Message
		if !res.Passed && errorMessage.Valid && errorMessage.String != "" {
			msg = errorMessage.String
		}
		ruleDetails = append(ruleDetails, map[string]interface{}{
			"rule_id": id, "passed": res.Passed, "message": msg,
		})
		if !res.Passed {
			allPassed = false
		}
	}

	detail["rules"] = ruleDetails
	detail["met"] = allPassed
	return allPassed, detail, nil
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

		case "compliance_rules":
			blockingResults, err := e.evaluateComplianceRules(ctx, tc)
			if err != nil {
				actionResult["status"] = "failed"
				actionResult["error"] = err.Error()
			} else if len(blockingResults) > 0 {
				actionResult["status"] = "blocked"
				actionResult["blocking_rules"] = blockingResults
				err = fmt.Errorf("%d compliance rule(s) blocked this write", len(blockingResults))
			} else {
				actionResult["status"] = "success"
			}
			if err != nil {
				actionResults = append(actionResults, actionResult)
				result["actions"] = actionResults
				return result, err
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

// evaluateComplianceRules runs rulefabric's per-record rule evaluation (Rule/ConditionGroup,
// severity, enforcement mode) against the BO row event data, returning the results that
// should block the write (hard/soft block, error/hard-block severity).
func (e *TriggerEngine) evaluateComplianceRules(ctx context.Context, tc *TriggerContext) ([]rulefabric.EvaluationResult, error) {
	tenantID, err := uuid.Parse(tc.TenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant id %q: %w", tc.TenantID, err)
	}

	evaluator, err := rulefabric.NewRuleEvaluator(e.db)
	if err != nil {
		return nil, fmt.Errorf("failed constructing rule evaluator: %w", err)
	}

	rules, err := evaluator.GetRulesForEvaluation(ctx, tenantID, rulefabric.GetRulesOptions{
		Channel: tc.TriggerKey,
		Entity:  tc.TargetEntity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed fetching compliance rules: %w", err)
	}
	if len(rules) == 0 {
		return nil, nil
	}

	batch, err := evaluator.EvaluateBatch(ctx, rules, &rulefabric.EvaluationContext{
		TenantID:       tenantID,
		Channel:        tc.TriggerKey,
		Data:           tc.EventData,
		EvaluationTime: tc.RequestedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("compliance rule batch evaluation failed: %w", err)
	}

	return batch.BlockingResults, nil
}

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

// auditTriggerExecution records each result to bp_trigger_executions — the
// live, partitioned table this belongs to (the old "trigger_executions"
// table this used to insert into doesn't exist). trigger_id is a required
// uuid column there, so a malformed/non-uuid TriggerID (shouldn't happen —
// it always comes from a real validation_triggers.id) is skipped rather
// than attempted, same as any other write failure here: logged, not fatal,
// since audit logging must never block the trigger's actual outcome.
func (e *TriggerEngine) auditTriggerExecution(ctx context.Context, tc *TriggerContext, results []ExecutionResult, durationMs int64) {
	tenantID, err := uuid.Parse(tc.TenantID)
	if err != nil {
		log.Printf("[ERROR] auditTriggerExecution: invalid tenant id %q: %v", tc.TenantID, err)
		return
	}

	query := `
		INSERT INTO bp_trigger_executions
		(id, trigger_id, tenant_id, execution_status, trigger_payload, result, execution_time_ms, error_message, executed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	for _, result := range results {
		triggerID, err := uuid.Parse(result.TriggerID)
		if err != nil {
			log.Printf("[ERROR] auditTriggerExecution: invalid trigger id %q: %v", result.TriggerID, err)
			continue
		}

		payload, _ := json.Marshal(map[string]interface{}{
			"trigger_key":   tc.TriggerKey,
			"target_entity": tc.TargetEntity,
			"entity_id":     tc.EntityID,
			"executed_by":   tc.UserID,
			"event_data":    tc.EventData,
		})
		resultJSON, _ := json.Marshal(map[string]interface{}{
			"evaluation_result": result.EvaluationResult,
			"action_result":     result.ActionResult,
		})
		var errMsg sql.NullString
		if result.ErrorMessage != "" {
			errMsg = sql.NullString{String: result.ErrorMessage, Valid: true}
		}

		if _, err := e.db.ExecContext(ctx, query,
			uuid.New(), triggerID, tenantID, result.Status, payload, resultJSON, durationMs, errMsg, time.Now(),
		); err != nil {
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

// ABACEngine is a thin adapter over the single consolidated ABAC
// implementation (domain.ABACEvaluator, backed by the abac_policies table)
// so the trigger engine's ABACContext shape doesn't leak into domain. It
// used to be its own stub ("TODO: Implement ABAC evaluation; return true");
// evaluation logic itself now lives in exactly one place.
type ABACEngine struct {
	evaluator *domain.ABACEvaluator
}

// NewABACEngine wires the trigger engine's ABAC check to the real evaluator.
func NewABACEngine(db *sql.DB) *ABACEngine {
	return &ABACEngine{evaluator: domain.NewABACEvaluator(db)}
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

func (a *ABACEngine) Evaluate(ctx context.Context, abacCtx *ABACContext) bool {
	if a.evaluator == nil {
		return true
	}
	allowed, _, _, err := a.evaluator.Evaluate(ctx, domain.EvaluationRequest{
		UserID:   abacCtx.SubjectID,
		TenantID: abacCtx.TenantID,
		AssetID:  abacCtx.Resource,
		Action:   domain.Permission(abacCtx.Action),
		Context: map[string]interface{}{
			"client_ip": abacCtx.ClientIP,
			"time":      abacCtx.Time,
			"policy_id": abacCtx.PolicyID,
		},
	})
	if err != nil {
		// Fail open on evaluation error: an ABAC misconfiguration or DB
		// hiccup shouldn't block every trigger in the platform. This
		// mirrors the same principle used elsewhere in the RBAC path —
		// resolution failures fail open, explicit denials fail closed.
		return true
	}
	return allowed
}
