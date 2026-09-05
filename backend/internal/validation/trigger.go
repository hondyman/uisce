package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// TriggerDispatchPhase controls which class of trigger dispatch modes run.
type TriggerDispatchPhase int

const (
	// TriggerPhaseSync runs only "sync" and nil DispatchMode triggers.
	// These run before the BO write and can veto it.
	TriggerPhaseSync TriggerDispatchPhase = iota
	// TriggerPhaseAsync runs only "async" DispatchMode triggers.
	// The tx parameter (if non-nil) is passed through to the outbox publisher
	// so the outbox row rides the caller's transaction.
	TriggerPhaseAsync
)

// ValidationTrigger represents a trigger that ties actions to validation rules
type ValidationTrigger struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	TriggerType  string          `json:"trigger_type"` // "save", "create", "delete", "field_change", "workflow_step", etc.
	TargetEntity string          `json:"target_entity"`
	StepName     *string         `json:"step_name,omitempty"`
	RuleIDs      pq.StringArray  `json:"rule_ids"`
	Meta         json.RawMessage `json:"meta,omitempty"`

	// PipelineID optionally binds this trigger to a Data Pipeline Studio
	// pipeline (internal/datapipeline.PipelineDefinition) that runs after
	// the trigger's rule checks pass. Nil means no pipeline binding —
	// existing triggers with no pipeline configured are unaffected.
	PipelineID *uuid.UUID `json:"pipeline_id,omitempty"`
	// DispatchMode is "sync" (pipeline runs inline; a pipeline failure
	// blocks the write, same as a failed rule) or "async" (an outbox event
	// is written and the pipeline runs later via Temporal, never blocking
	// the write). Ignored when PipelineID is nil.
	DispatchMode string `json:"dispatch_mode,omitempty"`
}

// ValidationRule represents a single validation rule (from catalog_validation_rules)
type ValidationRule struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	RuleName       string          `json:"rule_name"`
	RuleType       string          `json:"rule_type"` // "field_format", "cardinality", etc.
	TargetEntities pq.StringArray  `json:"target_entities"`
	ConditionJSON  json.RawMessage `json:"condition_json"`
	ErrorMessage   string          `json:"error_message"`
	CoreRuleID     *string         `json:"core_rule_id,omitempty"`
	InheritMode    string          `json:"inherit_mode,omitempty"`
	CoreVersionPin *int            `json:"core_version_pin,omitempty"`
}

// TriggerValidationEngine extends ValidationEngine with trigger-aware validation
type TriggerValidationEngine struct {
	*ValidationEngine
	db     *sql.DB
	logger Logger
	// test-only in-memory overrides to avoid DB access during unit tests
	testTriggers []ValidationTrigger
	testRules    map[string]ValidationRule

	// pipelineExecutor and outboxPublisher back trigger->pipeline dispatch
	// (see trigger_dispatch.go). Both may be nil, in which case triggers
	// with PipelineID set are skipped with a warning rather than failing
	// the write — this keeps the feature fully opt-in/backward compatible.
	pipelineExecutor PipelineTriggerExecutor
	outboxPublisher  PipelineOutboxPublisher
}

// WithPipelineExecutor wires a synchronous pipeline executor used by
// triggers with DispatchMode "sync". Returns the engine for chaining.
func (tve *TriggerValidationEngine) WithPipelineExecutor(exec PipelineTriggerExecutor) *TriggerValidationEngine {
	tve.pipelineExecutor = exec
	return tve
}

// WithOutboxPublisher wires the outbox writer used by triggers with
// DispatchMode "async". Returns the engine for chaining.
func (tve *TriggerValidationEngine) WithOutboxPublisher(pub PipelineOutboxPublisher) *TriggerValidationEngine {
	tve.outboxPublisher = pub
	return tve
}

// Logger interface for dependency injection
type Logger interface {
	Warn(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
}

// SimpleLogger is a basic logger for when none is provided
type SimpleLogger struct{}

func (s *SimpleLogger) Warn(msg string, keyvals ...interface{}) {
	log.Printf("[WARN] %s %v", msg, keyvals)
}
func (s *SimpleLogger) Error(msg string, keyvals ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, keyvals)
}
func (s *SimpleLogger) Info(msg string, keyvals ...interface{}) {
	log.Printf("[INFO] %s %v", msg, keyvals)
}

// NewTriggerValidationEngine creates a new trigger-aware validation engine
func NewTriggerValidationEngine(db *sql.DB, logger Logger) *TriggerValidationEngine {
	if logger == nil {
		logger = &SimpleLogger{}
	}
	return &TriggerValidationEngine{
		ValidationEngine: NewValidationEngine(),
		db:               db,
		logger:           logger,
	}
}

// WithTestTriggers sets in-memory triggers for testing and returns the engine for chaining.
// When provided, fetchTriggers will return data from this in-memory slice instead of querying
// the database. This is intended for unit tests only.
func (tve *TriggerValidationEngine) WithTestTriggers(triggers []ValidationTrigger) *TriggerValidationEngine {
	tve.testTriggers = triggers
	return tve
}

// WithTestRules sets in-memory rules for testing and returns the engine for chaining.
// When provided, fetchRuleByID will return rules from this map instead of querying the DB.
// This is intended for unit tests only.
func (tve *TriggerValidationEngine) WithTestRules(rules map[string]ValidationRule) *TriggerValidationEngine {
	tve.testRules = rules
	return tve
}

// TriggerValidate enforces triggers for a given action/entity payload.
// Returns nil when all validation rules pass; returns an error describing the first failure otherwise.
//
// TriggerValidate is the legacy entry point that runs sync then async triggers
// back-to-back in a single call. For transactional atomicity (async triggers
// riding the BO write's transaction), use DispatchWithPhase directly.
func (tve *TriggerValidationEngine) TriggerValidate(ctx context.Context, tenantID uuid.UUID, triggerType, entity, stepName string, data map[string]interface{}) error {
	if tve.db == nil && tve.testRules == nil && len(tve.testTriggers) == 0 {
		return fmt.Errorf("trigger validation: db not configured")
	}

	triggers, err := tve.fetchTriggers(ctx, tenantID.String(), triggerType, entity, stepName)
	if err != nil {
		tve.logger.Error("fetchTriggers failed", "error", err.Error())
		return fmt.Errorf("fetch triggers: %w", err)
	}

	// Walk triggers twice: sync phase (pre-write, can veto) then async phase.
	for _, t := range triggers {
		if err := tve.evaluateTriggerRulesAndDispatchPhase(ctx, tenantID, t, data, TriggerPhaseSync, nil); err != nil {
			return err
		}
	}
	for _, t := range triggers {
		if err := tve.evaluateTriggerRulesAndDispatchPhase(ctx, tenantID, t, data, TriggerPhaseAsync, nil); err != nil {
			return err
		}
	}
	return nil
}

// DispatchWithPhase evaluates and dispatches triggers for a specific phase.
// Rules are always evaluated (both phases). The pipeline dispatch is gated by phase:
//   - TriggerPhaseSync: dispatches only "sync"/"" DispatchMode triggers;
//     errors are returned and can veto the BO write.
//   - TriggerPhaseAsync: dispatches only "async" DispatchMode triggers;
//     if tx is non-nil the outbox row rides that transaction (atomic with the
//     BO write). Async errors are logged, never returned.
func (tve *TriggerValidationEngine) DispatchWithPhase(
	ctx context.Context,
	phase TriggerDispatchPhase,
	tx *sqlx.Tx,
	tenantID uuid.UUID,
	triggerType string,
	entity string,
	stepName string,
	data map[string]interface{},
) error {
	if tve.db == nil && tve.testRules == nil && len(tve.testTriggers) == 0 {
		return fmt.Errorf("trigger validation: db not configured")
	}

	triggers, err := tve.fetchTriggers(ctx, tenantID.String(), triggerType, entity, stepName)
	if err != nil {
		tve.logger.Error("fetchTriggers failed", "error", err.Error())
		return fmt.Errorf("fetch triggers: %w", err)
	}

	for _, t := range triggers {
		if err := tve.evaluateTriggerRulesAndDispatchPhase(ctx, tenantID, t, data, phase, tx); err != nil {
			return err
		}
	}
	return nil
}

// evaluateTriggerRulesAndDispatchPhase evaluates rules for a single trigger and
// dispatches its pipeline for the given phase. Rules are always evaluated;
// the pipeline dispatch is gated by phase. Errors from failed rules are returned
// (can veto the BO write); async dispatch errors are logged only.
func (tve *TriggerValidationEngine) evaluateTriggerRulesAndDispatchPhase(
	ctx context.Context,
	tenantID uuid.UUID,
	t ValidationTrigger,
	data map[string]interface{},
	phase TriggerDispatchPhase,
	tx *sqlx.Tx,
) error {
	for _, rid := range t.RuleIDs {
		rule, err := tve.fetchRuleByID(ctx, rid)
		if err != nil {
			tve.logger.Warn("TriggerValidate: missing rule", "rule_id", rid, "err", err.Error())
			continue
		}

		condition, err := UnwrapConditionPayload(rule.ConditionJSON)
		if err != nil {
			tve.logger.Error("TriggerValidate: unmarshal condition failed", "rule_id", rid, "err", err.Error())
			continue
		}

		result := tve.Execute(ExecutionContext{
			RuleID:       rid,
			RuleType:     rule.RuleType,
			TargetEntity: t.TargetEntity,
			Condition:    condition,
			Data:         data,
		})

		if !result.Passed {
			msg := rule.ErrorMessage
			if msg == "" {
				msg = result.Message
			}
			return fmt.Errorf("%s: %s", rule.RuleName, msg)
		}
	}

	if t.PipelineID != nil {
		return tve.dispatchPipelineForPhase(ctx, phase, tx, tenantID, t, data)
	}
	return nil
}

// dispatchPipelineForPhase dispatches a trigger's pipeline filtered by phase.
// This is the single method that handles both sync and async dispatch, distinguished
// by phase. Kept separate from evaluateTriggerRulesAndDispatchPhase so rule
// evaluation and dispatch are independently testable.
func (tve *TriggerValidationEngine) dispatchPipelineForPhase(
	ctx context.Context,
	phase TriggerDispatchPhase,
	tx *sqlx.Tx,
	tenantID uuid.UUID,
	t ValidationTrigger,
	data map[string]interface{},
) error {
	switch phase {
	case TriggerPhaseSync:
		if t.DispatchMode != "sync" && t.DispatchMode != "" {
			return nil
		}
		if tve.pipelineExecutor == nil {
			tve.logger.Warn("dispatchPipelineForPhase: sync dispatch requested but no pipeline executor configured", "trigger_id", t.ID, "pipeline_id", t.PipelineID.String())
			return nil
		}
		if err := tve.pipelineExecutor.RunPipelineSync(ctx, tenantID, *t.PipelineID, data); err != nil {
			return fmt.Errorf("pipeline '%s' failed: %w", t.PipelineID.String(), err)
		}
		return nil

	case TriggerPhaseAsync:
		if t.DispatchMode != "async" {
			return nil
		}
		if tve.outboxPublisher == nil {
			tve.logger.Warn("dispatchPipelineForPhase: async dispatch requested but no outbox publisher configured", "trigger_id", t.ID, "pipeline_id", t.PipelineID.String())
			return nil
		}
		if tx != nil {
			if pubTx, ok := tve.outboxPublisher.(interface {
				PublishPipelineTriggerTx(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, pipelineID uuid.UUID, triggerID uuid.UUID, record map[string]interface{}) error
			}); ok {
				triggerID, _ := uuid.Parse(t.ID)
				if err := pubTx.PublishPipelineTriggerTx(ctx, tx, tenantID, *t.PipelineID, triggerID, data); err != nil {
					tve.logger.Error("dispatchPipelineForPhase: failed to enqueue async pipeline trigger", "trigger_id", t.ID, "pipeline_id", t.PipelineID.String(), "err", err.Error())
				}
			}
		} else {
			triggerID, _ := uuid.Parse(t.ID)
			if err := tve.outboxPublisher.PublishPipelineTrigger(ctx, tenantID, *t.PipelineID, triggerID, data); err != nil {
				tve.logger.Error("dispatchPipelineForPhase: failed to enqueue async pipeline trigger (legacy)", "trigger_id", t.ID, "pipeline_id", t.PipelineID.String(), "err", err.Error())
			}
		}
		return nil

	default:
		tve.logger.Warn("dispatchPipelineForPhase: unknown phase, skipping", "trigger_id", t.ID, "phase", phase)
		return nil
	}
}

// dispatchTriggerPipeline runs (sync) or enqueues (async) the pipeline
// bound to trigger t. Sync failures block the write, matching the blocking
// semantics already used for failed validation rules. Async dispatch never
// blocks the write: enqueue failures are logged, not returned.
func (tve *TriggerValidationEngine) dispatchTriggerPipeline(ctx context.Context, tenantID uuid.UUID, t ValidationTrigger, data map[string]interface{}) error {
	if t.PipelineID == nil {
		return nil
	}

	switch t.DispatchMode {
	case "async":
		if tve.outboxPublisher == nil {
			tve.logger.Warn("dispatchTriggerPipeline: async dispatch requested but no outbox publisher configured", "trigger_id", t.ID, "pipeline_id", t.PipelineID.String())
			return nil
		}
		triggerID, _ := uuid.Parse(t.ID)
		if err := tve.outboxPublisher.PublishPipelineTrigger(ctx, tenantID, *t.PipelineID, triggerID, data); err != nil {
			// Async dispatch must never block the write it validated.
			tve.logger.Error("dispatchTriggerPipeline: failed to enqueue async pipeline trigger", "trigger_id", t.ID, "pipeline_id", t.PipelineID.String(), "err", err.Error())
		}
		return nil

	case "sync", "":
		if tve.pipelineExecutor == nil {
			tve.logger.Warn("dispatchTriggerPipeline: sync dispatch requested but no pipeline executor configured", "trigger_id", t.ID, "pipeline_id", t.PipelineID.String())
			return nil
		}
		if err := tve.pipelineExecutor.RunPipelineSync(ctx, tenantID, *t.PipelineID, data); err != nil {
			return fmt.Errorf("pipeline '%s' failed: %w", t.PipelineID.String(), err)
		}
		return nil

	default:
		tve.logger.Warn("dispatchTriggerPipeline: unknown dispatch_mode, skipping", "trigger_id", t.ID, "dispatch_mode", t.DispatchMode)
		return nil
	}
}

// fetchTriggers retrieves validation triggers for a given action from the DB
func (tve *TriggerValidationEngine) fetchTriggers(ctx context.Context, tenantID, triggerType, targetEntity, stepName string) ([]ValidationTrigger, error) {
	// If testTriggers override is present (including empty slice), return matching in-memory triggers
	if tve.testTriggers != nil {
		var out []ValidationTrigger
		for _, t := range tve.testTriggers {
			if t.TenantID != tenantID {
				continue
			}
			if t.TriggerType != triggerType {
				continue
			}
			if t.TargetEntity != targetEntity {
				continue
			}
			// stepName may be empty
			if stepName != "" {
				if t.StepName == nil || *t.StepName != stepName {
					continue
				}
			}
			out = append(out, t)
		}
		return out, nil
	}

	q := `
	SELECT id, tenant_id, trigger_type, target_entity, step_name, rule_ids, COALESCE(meta, '{}'::jsonb)::text,
	       pipeline_id, COALESCE(dispatch_mode, '')
	FROM validation_triggers
	WHERE tenant_id = $1
	  AND trigger_type = $2
	  AND target_entity = $3
	  AND (step_name IS NULL OR step_name = $4 OR step_name = '')
	  AND is_active = true
	ORDER BY created_at DESC
  `

	var stepNameParam interface{}
	if stepName != "" {
		stepNameParam = stepName
	}

	rows, err := tve.db.QueryContext(ctx, q, tenantID, triggerType, targetEntity, stepNameParam)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ValidationTrigger
	for rows.Next() {
		var t ValidationTrigger
		var stepNameNull sql.NullString
		var ruleIDsArray pq.StringArray
		var metaStr string
		var pipelineIDNull sql.NullString

		if err := rows.Scan(&t.ID, &t.TenantID, &t.TriggerType, &t.TargetEntity, &stepNameNull, &ruleIDsArray, &metaStr, &pipelineIDNull, &t.DispatchMode); err != nil {
			tve.logger.Warn("fetchTriggers: scan error", "err", err.Error())
			continue
		}

		if stepNameNull.Valid {
			t.StepName = &stepNameNull.String
		}
		if pipelineIDNull.Valid && pipelineIDNull.String != "" {
			if pid, err := uuid.Parse(pipelineIDNull.String); err == nil {
				t.PipelineID = &pid
			}
		}
		t.RuleIDs = ruleIDsArray
		t.Meta = json.RawMessage(metaStr)
		out = append(out, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// fetchRuleByID retrieves a single validation rule by ID (from catalog_validation_rules or similar)
func (tve *TriggerValidationEngine) fetchRuleByID(ctx context.Context, ruleID string) (*ValidationRule, error) {
	// If testRules override is present use it
	if tve.testRules != nil {
		if r, ok := tve.testRules[ruleID]; ok {
			return &r, nil
		}
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}

	// catalog_validation_rules has no error_message column — description
	// serves the same "human-readable explanation" purpose.
	q := `
	SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, COALESCE(description, ''),
	       core_rule_id, inherit_mode, core_version_pin
	FROM catalog_validation_rules
	WHERE id = $1
  `

	var rule ValidationRule
	var targetEntitiesArray pq.StringArray
	var coreRuleID sql.NullString
	var inheritMode sql.NullString
	var coreVersionPin sql.NullInt32

	err := tve.db.QueryRowContext(ctx, q, ruleID).Scan(
		&rule.ID,
		&rule.TenantID,
		&rule.RuleName,
		&rule.RuleType,
		&targetEntitiesArray,
		&rule.ConditionJSON,
		&rule.ErrorMessage,
		&coreRuleID,
		&inheritMode,
		&coreVersionPin,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rule not found: %s", ruleID)
		}
		return nil, err
	}

	rule.TargetEntities = targetEntitiesArray
	if coreRuleID.Valid {
		rule.CoreRuleID = &coreRuleID.String
	}
	if inheritMode.Valid {
		rule.InheritMode = inheritMode.String
	}
	if coreVersionPin.Valid {
		v := int(coreVersionPin.Int32)
		rule.CoreVersionPin = &v
	}

	// If this tenant rule is an inheriting instance, resolve its effective definition from the core template.
	if rule.CoreRuleID != nil && strings.TrimSpace(rule.InheritMode) == "inherit" {
		coreCond, err := tve.resolveCoreConditionJSON(ctx, *rule.CoreRuleID, rule.CoreVersionPin)
		if err == nil && len(coreCond) > 0 {
			rule.ConditionJSON = coreCond
		}
	}
	return &rule, nil
}

func (tve *TriggerValidationEngine) resolveCoreConditionJSON(ctx context.Context, coreRuleID string, coreVersionPin *int) (json.RawMessage, error) {
	// Step 1: find rule_key from the referenced core row.
	var ruleKey string
	err := tve.db.QueryRowContext(ctx, `
		SELECT rule_key
		FROM public.catalog_validation_rule_cores
		WHERE id = $1
	`, coreRuleID).Scan(&ruleKey)
	if err != nil {
		return nil, err
	}

	// Step 2: choose the effective version.
	if coreVersionPin != nil {
		var cond json.RawMessage
		err := tve.db.QueryRowContext(ctx, `
			SELECT condition_json
			FROM public.catalog_validation_rule_cores
			WHERE rule_key = $1 AND version = $2
		`, ruleKey, *coreVersionPin).Scan(&cond)
		return cond, err
	}

	var cond json.RawMessage
	err = tve.db.QueryRowContext(ctx, `
		SELECT condition_json
		FROM public.catalog_validation_rule_cores
		WHERE rule_key = $1 AND status = 'active'
		ORDER BY version DESC
		LIMIT 1
	`, ruleKey).Scan(&cond)
	return cond, err
}

// ValidateField performs quick field validation (used for onChange events)
// This is a lightweight check that only evaluates field_format rules for the given field
func (tve *TriggerValidationEngine) ValidateField(ctx context.Context, tenantID uuid.UUID, entity, fieldName string, fieldValue interface{}) error {
	// Allow test-only in-memory rules to be used without a DB connection.
	if tve.db == nil && tve.testRules == nil {
		return fmt.Errorf("field validation: db not configured")
	}
	// If in-memory testRules map is provided, use it instead of querying DB
	if tve.testRules != nil {
		for _, rule := range tve.testRules {
			if rule.RuleType != "field_format" {
				continue
			}

			// check if rule targets this entity
			found := false
			for _, te := range rule.TargetEntities {
				if te == entity {
					found = true
					break
				}
			}
			if !found {
				continue
			}

			// unmarshal condition (unwrapping the payload envelope) and
			// check the 'field' matches
			condition, err := UnwrapConditionPayload(rule.ConditionJSON)
			if err != nil {
				continue
			}
			if f, ok := condition["field"].(string); !ok || f != fieldName {
				continue
			}

			result := tve.Execute(ExecutionContext{
				RuleID:       rule.ID,
				RuleType:     rule.RuleType,
				TargetEntity: entity,
				Condition:    condition,
				Data:         map[string]interface{}{fieldName: fieldValue},
			})

			if !result.Passed {
				msg := rule.ErrorMessage
				if msg == "" {
					msg = result.Message
				}
				return fmt.Errorf("%s: %s", rule.RuleName, msg)
			}
		}

		return nil
	}

	// Fetch rules that target this entity and field, type field_format.
	// catalog_validation_rules has no error_message column — description
	// serves the same purpose (see fetchRuleByID above).
	q := `
	SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, COALESCE(description, '')
	FROM catalog_validation_rules
	WHERE tenant_id = $1
	  AND rule_type = 'field_format'
	  AND target_entities @> ARRAY[$2]::text[]
	  AND condition_json @> jsonb_build_object('field', $3)
  `

	rows, err := tve.db.QueryContext(ctx, q, tenantID.String(), entity, fieldName)
	if err != nil {
		return fmt.Errorf("query rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rule ValidationRule
		var targetEntitiesArray pq.StringArray

		if err := rows.Scan(
			&rule.ID,
			&rule.TenantID,
			&rule.RuleName,
			&rule.RuleType,
			&targetEntitiesArray,
			&rule.ConditionJSON,
			&rule.ErrorMessage,
		); err != nil {
			tve.logger.Warn("ValidateField: scan error", "err", err.Error())
			continue
		}

		condition, err := UnwrapConditionPayload(rule.ConditionJSON)
		if err != nil {
			continue
		}

		result := tve.Execute(ExecutionContext{
			RuleID:       rule.ID,
			RuleType:     rule.RuleType,
			TargetEntity: entity,
			Condition:    condition,
			Data:         map[string]interface{}{fieldName: fieldValue},
		})

		if !result.Passed {
			msg := rule.ErrorMessage
			if msg == "" {
				msg = result.Message
			}
			return fmt.Errorf("%s: %s", rule.RuleName, msg)
		}
	}

	return nil
}
