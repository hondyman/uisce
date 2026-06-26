package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/rules"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// VALIDATION RULE ENGINE - Workday-Like Low-Code Framework
// ============================================================================
// Provides expression evaluation, condition matching, and extensible rule registry
// Integrates with async validation service for Workday-like BP (Business Process) validation

// RuleCondition represents a single validation condition
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // =, !=, >, <, >=, <=, contains, startsWith, endsWith, in, regex
	Value    interface{} `json:"value"`
}

// ComplexCondition supports AND/OR logic for multi-field validation
type ComplexCondition struct {
	And []RuleCondition `json:"and,omitempty"`
	Or  []RuleCondition `json:"or,omitempty"`
	Not *RuleCondition  `json:"not,omitempty"`
}

// ValidationRuleDefinition represents a complete validation rule
type ValidationRuleDefinition struct {
	ID              string          `json:"id" db:"id"`
	TenantID        string          `json:"tenant_id" db:"tenant_id"`
	BPName          string          `json:"bp_name" db:"bp_name"`                     // Business Process name
	StepName        string          `json:"step_name" db:"step_name"`                 // Step in process
	ConditionJSON   json.RawMessage `json:"condition_json" db:"condition_json"`       // AND/OR/NOT logic
	ActionOnSuccess string          `json:"action_on_success" db:"action_on_success"` // route:queue_name or notify:email
	ActionOnFailure string          `json:"action_on_failure" db:"action_on_failure"` // route:queue_name
	ErrorMessage    string          `json:"error_message" db:"error_message"`
	Priority        int             `json:"priority" db:"priority"` // Execution order
	Enabled         bool            `json:"enabled" db:"enabled"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// RuleEvaluationResult holds the outcome of rule evaluation
type RuleEvaluationResult struct {
	RuleID         string
	Passed         bool
	ErrorMessage   string
	ActionToTake   string // Success or Failure action
	EvaluationTime time.Duration
	Details        map[string]interface{}
}

// ValidationRuleEngine evaluates complex business rules
type ValidationRuleEngine interface {
	// Evaluate a single rule condition against data
	EvaluateCondition(ctx context.Context, tenantID string, condition RuleCondition, data map[string]interface{}) (bool, error)

	// Evaluate complex AND/OR/NOT conditions
	EvaluateComplexCondition(ctx context.Context, tenantID string, condition ComplexCondition, data map[string]interface{}) (bool, error)

	// Evaluate a complete rule definition
	EvaluateRule(ctx context.Context, tenantID string, rule ValidationRuleDefinition, data map[string]interface{}) (*RuleEvaluationResult, error)

	// Evaluate all rules for a BP step
	EvaluateBPStep(ctx context.Context, tenantID string, bpName string, stepName string, data map[string]interface{}) ([]*RuleEvaluationResult, error)

	// Store rule definition
	StoreRule(ctx context.Context, rule *ValidationRuleDefinition) error

	// Get rules for BP step
	GetRulesForBPStep(ctx context.Context, tenantID string, bpName string, stepName string) ([]ValidationRuleDefinition, error)

	// Get all rules for tenant
	GetTenantRules(ctx context.Context, tenantID string) ([]ValidationRuleDefinition, error)

	// Delete rule
	DeleteRule(ctx context.Context, ruleID string) error

	// Get rule by ID
	GetRuleByID(ctx context.Context, ruleID string) (*ValidationRuleDefinition, error)
}

// ValidationRuleEngineImpl implements ValidationRuleEngine
type ValidationRuleEngineImpl struct {
	db       *sqlx.DB
	resolver *rules.PathResolver
}

// NewValidationRuleEngine creates a new rule engine
func NewValidationRuleEngine(db *sqlx.DB, instanceProvider rules.InstanceProvider) ValidationRuleEngine {
	var resolver *rules.PathResolver
	if instanceProvider != nil {
		resolver = rules.NewPathResolver(instanceProvider)
	}
	return &ValidationRuleEngineImpl{
		db:       db,
		resolver: resolver,
	}
}

// ============================================================================
// Core Rule Evaluation
// ============================================================================

// EvaluateCondition evaluates a single condition
func (vre *ValidationRuleEngineImpl) EvaluateCondition(ctx context.Context, tenantID string, condition RuleCondition, data map[string]interface{}) (bool, error) {
	// Check if field is a path (contains dot) and resolver is available
	if strings.Contains(condition.Field, ".") && vre.resolver != nil {
		val, err := vre.resolver.ResolvePath(ctx, tenantID, data, condition.Field)
		if err != nil {
			// Log error but treat as false/error?
			// For now return error
			return false, err
		}
		// Use resolved value
		return vre.evaluateOperator(condition.Operator, val, condition.Value), nil
	}

	value, exists := data[condition.Field]

	if !exists {
		// Field doesn't exist - check if operator handles this
		switch condition.Operator {
		case "=", "!=", "contains", "in":
			// These require the field to exist
			return false, fmt.Errorf("field '%s' not found in data", condition.Field)
		case "isEmpty":
			return true, nil
		default:
			return false, nil
		}
	}

	return vre.evaluateOperator(condition.Operator, value, condition.Value), nil
}

// EvaluateComplexCondition evaluates AND/OR/NOT combinations
func (vre *ValidationRuleEngineImpl) EvaluateComplexCondition(ctx context.Context, tenantID string, condition ComplexCondition, data map[string]interface{}) (bool, error) {
	// Evaluate AND conditions (all must be true)
	if len(condition.And) > 0 {
		for _, cond := range condition.And {
			result, err := vre.EvaluateCondition(ctx, tenantID, cond, data)
			if err != nil || !result {
				return false, err
			}
		}
		return true, nil
	}

	// Evaluate OR conditions (at least one must be true)
	if len(condition.Or) > 0 {
		for _, cond := range condition.Or {
			result, err := vre.EvaluateCondition(ctx, tenantID, cond, data)
			if err == nil && result {
				return true, nil
			}
		}
		return false, nil
	}

	// Evaluate NOT condition (must be false)
	if condition.Not != nil {
		result, err := vre.EvaluateCondition(ctx, tenantID, *condition.Not, data)
		if err != nil {
			return false, err
		}
		return !result, nil
	}

	return true, nil
}

// EvaluateRule evaluates a complete rule
func (vre *ValidationRuleEngineImpl) EvaluateRule(ctx context.Context, tenantID string, rule ValidationRuleDefinition, data map[string]interface{}) (*RuleEvaluationResult, error) {
	startTime := time.Now()
	// Parse condition into a generic structure and evaluate recursively
	var raw interface{}
	if len(rule.ConditionJSON) == 0 {
		raw = map[string]interface{}{}
	} else {
		if err := json.Unmarshal(rule.ConditionJSON, &raw); err != nil {
			// Try to parse as simple condition into RuleCondition
			var simpleCondition RuleCondition
			if err2 := json.Unmarshal(rule.ConditionJSON, &simpleCondition); err2 != nil {
				return nil, fmt.Errorf("invalid condition JSON: %w", err)
			}
			// Evaluate simple condition directly
			res, e := vre.EvaluateCondition(ctx, tenantID, simpleCondition, data)
			if e != nil {
				log.Printf("[RuleEngine] Error evaluating simple condition: %v", e)
				res = false
			}
			passed := res
			// Build result and return
			actionToTake := ""
			if passed && rule.ActionOnSuccess != "" {
				actionToTake = rule.ActionOnSuccess
			} else if !passed && rule.ActionOnFailure != "" {
				actionToTake = rule.ActionOnFailure
			}

			result := &RuleEvaluationResult{
				RuleID:         rule.ID,
				Passed:         passed,
				ErrorMessage:   rule.ErrorMessage,
				ActionToTake:   actionToTake,
				EvaluationTime: time.Since(startTime),
				Details: map[string]interface{}{
					"bp_name":   rule.BPName,
					"step_name": rule.StepName,
				},
			}

			return result, nil
		}
	}

	passed, err := vre.evaluateConditionTree(ctx, tenantID, raw, data)
	if err != nil {
		log.Printf("[RuleEngine] Error evaluating condition tree: %v", err)
		passed = false
	}

	// Determine action
	actionToTake := ""
	if passed && rule.ActionOnSuccess != "" {
		actionToTake = rule.ActionOnSuccess
	} else if !passed && rule.ActionOnFailure != "" {
		actionToTake = rule.ActionOnFailure
	}

	result := &RuleEvaluationResult{
		RuleID:         rule.ID,
		Passed:         passed,
		ErrorMessage:   rule.ErrorMessage,
		ActionToTake:   actionToTake,
		EvaluationTime: time.Since(startTime),
		Details: map[string]interface{}{
			"bp_name":   rule.BPName,
			"step_name": rule.StepName,
		},
	}

	return result, nil
}

// EvaluateBPStep evaluates all rules for a business process step
func (vre *ValidationRuleEngineImpl) EvaluateBPStep(ctx context.Context, tenantID string, bpName string, stepName string, data map[string]interface{}) ([]*RuleEvaluationResult, error) {
	log.Printf("[RuleEngine] Evaluating BP step: %s/%s", bpName, stepName)

	// Fetch rules for this step
	rules, err := vre.GetRulesForBPStep(ctx, tenantID, bpName, stepName)
	if err != nil {
		return nil, err
	}

	results := make([]*RuleEvaluationResult, 0)

	// Sort by priority
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		result, err := vre.EvaluateRule(ctx, tenantID, rule, data)
		if err != nil {
			log.Printf("[RuleEngine] Error evaluating rule %s: %v", rule.ID, err)
			continue
		}

		results = append(results, result)

		// If rule failed and has no continue-on-error flag, stop
		if !result.Passed {
			log.Printf("[RuleEngine] Rule %s failed with message: %s", rule.ID, result.ErrorMessage)
			break
		}
	}

	return results, nil
}

// ============================================================================
// Operator Implementations
// ============================================================================

// evaluateOperator applies operator-specific logic
func (vre *ValidationRuleEngineImpl) evaluateOperator(operator string, value interface{}, target interface{}) bool {
	switch operator {
	case "=", "==":
		return vre.equals(value, target)

	case "!=", "<>":
		return !vre.equals(value, target)

	case ">":
		return vre.greaterThan(value, target)

	case "<":
		return vre.lessThan(value, target)

	case ">=":
		return vre.greaterThanOrEqual(value, target)

	case "<=":
		return vre.lessThanOrEqual(value, target)

	case "contains":
		return strings.Contains(fmt.Sprint(value), fmt.Sprint(target))

	case "startsWith":
		return strings.HasPrefix(fmt.Sprint(value), fmt.Sprint(target))

	case "endsWith":
		return strings.HasSuffix(fmt.Sprint(value), fmt.Sprint(target))

	case "in":
		return vre.isIn(value, target)

	case "regex":
		return vre.matchesRegex(fmt.Sprint(value), fmt.Sprint(target))

	case "isEmpty":
		return value == nil || value == "" || value == 0 || value == false

	case "isNotEmpty":
		return value != nil && value != "" && value != 0 && value != false

	case "between":
		return vre.between(value, target)

	default:
		log.Printf("[RuleEngine] Unknown operator: %s", operator)
		return false
	}
}

// Comparison helpers
func (vre *ValidationRuleEngineImpl) equals(a, b interface{}) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func (vre *ValidationRuleEngineImpl) greaterThan(a, b interface{}) bool {
	return vre.toFloat64(a) > vre.toFloat64(b)
}

func (vre *ValidationRuleEngineImpl) lessThan(a, b interface{}) bool {
	return vre.toFloat64(a) < vre.toFloat64(b)
}

func (vre *ValidationRuleEngineImpl) greaterThanOrEqual(a, b interface{}) bool {
	return vre.toFloat64(a) >= vre.toFloat64(b)
}

func (vre *ValidationRuleEngineImpl) lessThanOrEqual(a, b interface{}) bool {
	return vre.toFloat64(a) <= vre.toFloat64(b)
}

func (vre *ValidationRuleEngineImpl) isIn(value interface{}, target interface{}) bool {
	// Target should be a slice or comma-separated string
	switch t := target.(type) {
	case []interface{}:
		for _, item := range t {
			if vre.equals(value, item) {
				return true
			}
		}
	case string:
		values := strings.Split(t, ",")
		for _, v := range values {
			if strings.TrimSpace(v) == fmt.Sprint(value) {
				return true
			}
		}
	}
	return false
}

func (vre *ValidationRuleEngineImpl) matchesRegex(value string, pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Printf("[RuleEngine] Invalid regex pattern: %v", err)
		return false
	}
	return re.MatchString(value)
}

func (vre *ValidationRuleEngineImpl) between(value interface{}, target interface{}) bool {
	// Target should be map with "min" and "max"
	if m, ok := target.(map[string]interface{}); ok {
		val := vre.toFloat64(value)
		min := vre.toFloat64(m["min"])
		max := vre.toFloat64(m["max"])
		return val >= min && val <= max
	}
	return false
}

func (vre *ValidationRuleEngineImpl) toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0
}

// evaluateConditionTree recursively evaluates a condition tree which may contain
// maps with keys "and", "or", "not" or a simple condition object with
// "field", "operator", "value". It supports nested groupings.
func (vre *ValidationRuleEngineImpl) evaluateConditionTree(ctx context.Context, tenantID string, node interface{}, data map[string]interface{}) (bool, error) {
	if node == nil {
		return true, nil
	}

	switch n := node.(type) {
	case map[string]interface{}:
		// Check for logical operators
		if andRaw, ok := n["and"]; ok {
			if arr, ok := andRaw.([]interface{}); ok {
				for _, v := range arr {
					res, err := vre.evaluateConditionTree(ctx, tenantID, v, data)
					if err != nil || !res {
						return false, err
					}
				}
				return true, nil
			}
		}

		if orRaw, ok := n["or"]; ok {
			if arr, ok := orRaw.([]interface{}); ok {
				for _, v := range arr {
					res, err := vre.evaluateConditionTree(ctx, tenantID, v, data)
					if err == nil && res {
						return true, nil
					}
				}
				return false, nil
			}
		}

		if notRaw, ok := n["not"]; ok {
			res, err := vre.evaluateConditionTree(ctx, tenantID, notRaw, data)
			if err != nil {
				return false, err
			}
			return !res, nil
		}

		// Otherwise, attempt to parse as a simple condition
		var rc RuleCondition
		// map -> marshal/unmarshal to reuse existing struct parsing
		b, err := json.Marshal(n)
		if err != nil {
			return false, fmt.Errorf("invalid condition node: %w", err)
		}
		if err := json.Unmarshal(b, &rc); err != nil {
			return false, fmt.Errorf("invalid simple condition: %w", err)
		}
		return vre.EvaluateCondition(ctx, tenantID, rc, data)

	case []interface{}:
		// Treat array as AND
		for _, v := range n {
			res, err := vre.evaluateConditionTree(ctx, tenantID, v, data)
			if err != nil || !res {
				return false, err
			}
		}
		return true, nil

	default:
		// try to decode into RuleCondition directly
		b, err := json.Marshal(n)
		if err != nil {
			return false, fmt.Errorf("invalid node: %w", err)
		}
		var rc RuleCondition
		if err := json.Unmarshal(b, &rc); err != nil {
			return false, fmt.Errorf("unrecognized condition node: %w", err)
		}
		return vre.EvaluateCondition(ctx, tenantID, rc, data)
	}
}

// ============================================================================
// Rule Storage & Retrieval
// ============================================================================

// StoreRule saves a rule definition to database
func (vre *ValidationRuleEngineImpl) StoreRule(ctx context.Context, rule *ValidationRuleDefinition) error {
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	}

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	query := `
		INSERT INTO bp_validations (
			id, tenant_id, bp_name, step_name, condition_json,
			action_on_success, action_on_failure, error_message,
			priority, enabled, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (id) DO UPDATE SET
			condition_json = $5,
			action_on_success = $6,
			action_on_failure = $7,
			error_message = $8,
			priority = $9,
			enabled = $10,
			updated_at = $12
	`

	_, err := vre.db.ExecContext(ctx, query,
		rule.ID, rule.TenantID, rule.BPName, rule.StepName,
		rule.ConditionJSON, rule.ActionOnSuccess, rule.ActionOnFailure,
		rule.ErrorMessage, rule.Priority, rule.Enabled,
		rule.CreatedAt, rule.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store rule: %w", err)
	}

	log.Printf("[RuleEngine] Rule stored: %s for %s/%s", rule.ID, rule.BPName, rule.StepName)
	return nil
}

// GetRulesForBPStep retrieves all rules for a business process step
func (vre *ValidationRuleEngineImpl) GetRulesForBPStep(ctx context.Context, tenantID string, bpName string, stepName string) ([]ValidationRuleDefinition, error) {
	query := `
		SELECT id, tenant_id, bp_name, step_name, condition_json,
		       action_on_success, action_on_failure, error_message,
		       priority, enabled, created_at, updated_at
		FROM bp_validations
		WHERE tenant_id = $1 AND bp_name = $2 AND step_name = $3 AND enabled = TRUE
		ORDER BY priority ASC
	`

	rows, err := vre.db.QueryxContext(ctx, query, tenantID, bpName, stepName)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []ValidationRuleDefinition
	for rows.Next() {
		var rule ValidationRuleDefinition
		err := rows.StructScan(&rule)
		if err != nil {
			log.Printf("[RuleEngine] Error scanning rule: %v", err)
			continue
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// GetTenantRules retrieves all rules for a tenant
func (vre *ValidationRuleEngineImpl) GetTenantRules(ctx context.Context, tenantID string) ([]ValidationRuleDefinition, error) {
	query := `
		SELECT id, tenant_id, bp_name, step_name, condition_json,
		       action_on_success, action_on_failure, error_message,
		       priority, enabled, created_at, updated_at
		FROM bp_validations
		WHERE tenant_id = $1
		ORDER BY bp_name, step_name, priority ASC
	`

	rows, err := vre.db.QueryxContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenant rules: %w", err)
	}
	defer rows.Close()

	var rules []ValidationRuleDefinition
	for rows.Next() {
		var rule ValidationRuleDefinition
		err := rows.StructScan(&rule)
		if err != nil {
			continue
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// GetRuleByID retrieves a specific rule
func (vre *ValidationRuleEngineImpl) GetRuleByID(ctx context.Context, ruleID string) (*ValidationRuleDefinition, error) {
	query := `
		SELECT id, tenant_id, bp_name, step_name, condition_json,
		       action_on_success, action_on_failure, error_message,
		       priority, enabled, created_at, updated_at
		FROM bp_validations
		WHERE id = $1
	`

	var rule ValidationRuleDefinition
	err := vre.db.GetContext(ctx, &rule, query, ruleID)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	return &rule, nil
}

// DeleteRule removes a rule
func (vre *ValidationRuleEngineImpl) DeleteRule(ctx context.Context, ruleID string) error {
	query := `DELETE FROM bp_validations WHERE id = $1`

	result, err := vre.db.ExecContext(ctx, query, ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	log.Printf("[RuleEngine] Rule deleted: %s", ruleID)
	return nil
}

// ============================================================================
// Rule Templates (Workday-style pre-built rules)
// ============================================================================

// RuleTemplate provides common validation scenarios
type RuleTemplate struct {
	Name      string
	Condition ComplexCondition
	Error     string
}

// GetCommonTemplates returns pre-built rule templates
func GetCommonTemplates() map[string]RuleTemplate {
	return map[string]RuleTemplate{
		"age_check": {
			Name: "Age >= 18",
			Condition: ComplexCondition{
				And: []RuleCondition{
					{Field: "age", Operator: ">=", Value: 18},
				},
			},
			Error: "Must be at least 18 years old",
		},
		"email_format": {
			Name: "Valid Email Format",
			Condition: ComplexCondition{
				And: []RuleCondition{
					{Field: "email", Operator: "regex", Value: `^[^\s@]+@[^\s@]+\.[^\s@]+$`},
				},
			},
			Error: "Invalid email format",
		},
		"marital_age": {
			Name: "Age >= 18 if Married",
			Condition: ComplexCondition{
				And: []RuleCondition{
					{Field: "marital_status", Operator: "=", Value: "married"},
					{Field: "age", Operator: ">=", Value: 18},
				},
			},
			Error: "Marital status requires age >= 18",
		},
		"salary_range": {
			Name: "Salary Within Range",
			Condition: ComplexCondition{
				And: []RuleCondition{
					{Field: "salary", Operator: "between", Value: map[string]interface{}{"min": 30000, "max": 500000}},
				},
			},
			Error: "Salary must be between $30,000 and $500,000",
		},
	}
}
