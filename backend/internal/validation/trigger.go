package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// ValidationTrigger represents a trigger that ties actions to validation rules
type ValidationTrigger struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	TriggerType  string          `json:"trigger_type"` // "save", "create", "delete", "field_change", "workflow_step", etc.
	TargetEntity string          `json:"target_entity"`
	StepName     *string         `json:"step_name,omitempty"`
	RuleIDs      pq.StringArray  `json:"rule_ids"`
	Meta         json.RawMessage `json:"meta,omitempty"`
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
	hasura HasuraClient
	logger Logger
	// test-only in-memory overrides to avoid DB access during unit tests
	testTriggers []ValidationTrigger
	testRules    map[string]ValidationRule
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

// NewTriggerValidationEngineWithHasura creates a new validation engine with Hasura support
func NewTriggerValidationEngineWithHasura(db *sql.DB, hasura HasuraClient, logger Logger) *TriggerValidationEngine {
	if logger == nil {
		logger = &SimpleLogger{}
	}
	return &TriggerValidationEngine{
		ValidationEngine: NewValidationEngine(),
		db:               db,
		hasura:           hasura,
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
func (tve *TriggerValidationEngine) TriggerValidate(ctx context.Context, tenantID uuid.UUID, triggerType, entity, stepName string, data map[string]interface{}) error {
	// Allow test-only in-memory overrides to be used without a DB connection.
	if tve.db == nil && tve.testRules == nil && len(tve.testTriggers) == 0 {
		return fmt.Errorf("trigger validation: db not configured")
	}

	// 1. fetch triggers for tenant/triggerType/entity (stepName may be empty)
	triggers, err := tve.fetchTriggers(ctx, tenantID.String(), triggerType, entity, stepName)
	if err != nil {
		tve.logger.Error("fetchTriggers failed", "error", err.Error())
		return fmt.Errorf("fetch triggers: %w", err)
	}

	// 2. for each trigger, evaluate each rule
	for _, t := range triggers {
		for _, rid := range t.RuleIDs {
			rule, err := tve.fetchRuleByID(ctx, rid)
			if err != nil {
				// log and continue - missing rule should not panic
				tve.logger.Warn("TriggerValidate: missing rule", "rule_id", rid, "err", err.Error())
				continue
			}

			// Convert condition JSON to map
			var condition map[string]interface{}
			if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
				tve.logger.Error("TriggerValidate: unmarshal condition failed", "rule_id", rid, "err", err.Error())
				continue
			}

			// Evaluate rule via ExecutionContext
			result := tve.Execute(ExecutionContext{
				RuleID:       rid,
				RuleType:     rule.RuleType,
				TargetEntity: entity,
				Condition:    condition,
				Data:         data,
			})

			if !result.Passed {
				// rule failed -> bubble error message (prefer rule's ErrorMessage over result.Message)
				msg := rule.ErrorMessage
				if msg == "" {
					msg = result.Message
				}
				return fmt.Errorf("%s: %s", rule.RuleName, msg)
			}
		}
	}

	return nil
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

	if tve.hasura != nil {
		triggers, err := tve.fetchTriggersWithHasura(ctx, tenantID, triggerType, targetEntity, stepName)
		if err == nil {
			return triggers, nil
		}
		tve.logger.Warn("Hasura query failed, falling back to SQL", "err", err.Error())
	}

	// SQL fallback
	q := `
	SELECT id, tenant_id, trigger_type, target_entity, step_name, rule_ids, COALESCE(meta, '{}'::jsonb)::text
	FROM validation_triggers
	WHERE tenant_id = $1 
	  AND trigger_type = $2 
	  AND target_entity = $3
	  AND (step_name IS NULL OR step_name = $4 OR step_name = '')
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

		if err := rows.Scan(&t.ID, &t.TenantID, &t.TriggerType, &t.TargetEntity, &stepNameNull, &ruleIDsArray, &metaStr); err != nil {
			tve.logger.Warn("fetchTriggers: scan error", "err", err.Error())
			continue
		}

		if stepNameNull.Valid {
			t.StepName = &stepNameNull.String
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

	if tve.hasura != nil {
		rule, err := tve.fetchRuleByIDWithHasura(ctx, ruleID)
		if err == nil {
			return rule, nil
		}
		tve.logger.Warn("Hasura query failed, falling back to SQL", "err", err.Error())
	}

	// SQL fallback
	q := `
	SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, error_message,
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

			// unmarshal condition and check the 'field' matches
			var condition map[string]interface{}
			if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
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

	// Fetch rules that target this entity and field, type field_format
	q := `
	SELECT id, tenant_id, rule_name, rule_type, target_entities, condition_json, error_message
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

		var condition map[string]interface{}
		if err := json.Unmarshal(rule.ConditionJSON, &condition); err != nil {
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

// Hasura helper functions

func (tve *TriggerValidationEngine) fetchTriggersWithHasura(ctx context.Context, tenantID, triggerType, targetEntity, stepName string) ([]ValidationTrigger, error) {
	query := `
		query FetchTriggers($tenantId: String!, $triggerType: String!, $targetEntity: String!, $stepName: String) {
			validation_triggers(
where: {
tenant_id: {_eq: $tenantId},
trigger_type: {_eq: $triggerType},
target_entity: {_eq: $targetEntity},
_or: [
{step_name: {_is_null: true}},
{step_name: {_eq: ""}},
{step_name: {_eq: $stepName}}
]
},
order_by: {created_at: desc}
) {
				id
				tenant_id
				trigger_type
				target_entity
				step_name
				rule_ids
				meta
			}
		}
	`

	variables := map[string]interface{}{
		"tenantId":     tenantID,
		"triggerType":  triggerType,
		"targetEntity": targetEntity,
	}
	if stepName != "" {
		variables["stepName"] = stepName
	}

	resp, err := tve.hasura.Query(query, variables)
	if err != nil {
		return nil, err
	}

	triggersData, ok := resp["validation_triggers"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	var out []ValidationTrigger
	for _, item := range triggersData {
		trigMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		t := ValidationTrigger{}
		if id, ok := trigMap["id"].(string); ok {
			t.ID = id
		}
		if tenantID, ok := trigMap["tenant_id"].(string); ok {
			t.TenantID = tenantID
		}
		if triggerType, ok := trigMap["trigger_type"].(string); ok {
			t.TriggerType = triggerType
		}
		if targetEntity, ok := trigMap["target_entity"].(string); ok {
			t.TargetEntity = targetEntity
		}
		if stepName, ok := trigMap["step_name"].(string); ok && stepName != "" {
			t.StepName = &stepName
		}
		if ruleIDs, ok := trigMap["rule_ids"].([]interface{}); ok {
			var stringArray pq.StringArray
			for _, rid := range ruleIDs {
				if ridStr, ok := rid.(string); ok {
					stringArray = append(stringArray, ridStr)
				}
			}
			t.RuleIDs = stringArray
		}
		if meta, ok := trigMap["meta"]; ok {
			if metaJSON, err := json.Marshal(meta); err == nil {
				t.Meta = metaJSON
			}
		}

		out = append(out, t)
	}

	return out, nil
}

func (tve *TriggerValidationEngine) fetchRuleByIDWithHasura(ctx context.Context, ruleID string) (*ValidationRule, error) {
	query := `
		query FetchRuleByID($ruleId: String!) {
			catalog_validation_rules_by_pk(id: $ruleId) {
				id
				tenant_id
				rule_name
				rule_type
				target_entities
				condition_json
				error_message
			}
		}
	`

	variables := map[string]interface{}{
		"ruleId": ruleID,
	}

	resp, err := tve.hasura.Query(query, variables)
	if err != nil {
		return nil, err
	}

	ruleData, ok := resp["catalog_validation_rules_by_pk"].(map[string]interface{})
	if !ok || ruleData == nil {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}

	rule := &ValidationRule{}
	if id, ok := ruleData["id"].(string); ok {
		rule.ID = id
	}
	if tenantID, ok := ruleData["tenant_id"].(string); ok {
		rule.TenantID = tenantID
	}
	if ruleName, ok := ruleData["rule_name"].(string); ok {
		rule.RuleName = ruleName
	}
	if ruleType, ok := ruleData["rule_type"].(string); ok {
		rule.RuleType = ruleType
	}
	if targetEntities, ok := ruleData["target_entities"].([]interface{}); ok {
		var stringArray pq.StringArray
		for _, te := range targetEntities {
			if teStr, ok := te.(string); ok {
				stringArray = append(stringArray, teStr)
			}
		}
		rule.TargetEntities = stringArray
	}
	if conditionJSON, ok := ruleData["condition_json"]; ok {
		if condJSON, err := json.Marshal(conditionJSON); err == nil {
			rule.ConditionJSON = condJSON
		}
	}
	if errorMessage, ok := ruleData["error_message"].(string); ok {
		rule.ErrorMessage = errorMessage
	}

	return rule, nil
}
