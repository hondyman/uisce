package altinv

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// CONFIGURABLE BUSINESS RULES ENGINE
// ============================================================================
// Features:
// - Dynamic rule configuration stored in database
// - Support for complex conditions (AND, OR, NOT)
// - Multiple operator types (comparison, range, pattern, custom)
// - Weighted scoring system
// - Auto-gating and escalation
// - Rule versioning and A/B testing
// - Performance tracking per rule
// ============================================================================

// RuleEngine processes business rules against opportunities
type RuleEngine struct {
	rules       []RuleDefinition
	config      RuleEngineConfig
	ruleMetrics map[string]*RuleMetrics
}

// RuleEngineConfig contains engine configuration
type RuleEngineConfig struct {
	EnableCaching       bool          `json:"enable_caching"`
	CacheTTL            time.Duration `json:"cache_ttl"`
	MaxParallelRules    int           `json:"max_parallel_rules"`
	TimeoutPerRule      time.Duration `json:"timeout_per_rule"`
	EnableMetrics       bool          `json:"enable_metrics"`
	EnableABTesting     bool          `json:"enable_ab_testing"`
	ABTestPercentage    float64       `json:"ab_test_percentage"`
	DefaultPassingScore float64       `json:"default_passing_score"`
}

// RuleDefinition represents a configurable business rule
type RuleDefinition struct {
	RuleID      uuid.UUID `json:"rule_id"`
	RuleCode    string    `json:"rule_code"`
	RuleName    string    `json:"rule_name"`
	Description string    `json:"description"`
	Category    string    `json:"category"` // SCREENING, RISK, COMPLIANCE, ALLOCATION
	Version     int       `json:"version"`
	IsActive    bool      `json:"is_active"`
	Priority    int       `json:"priority"` // Lower = higher priority

	// Targeting
	OpportunityTypes []string `json:"opportunity_types,omitempty"` // Empty = all types
	ClientSegments   []string `json:"client_segments,omitempty"`

	// Condition
	Condition RuleCondition `json:"condition"`

	// Scoring
	Weight     float64 `json:"weight"`
	MaxScore   float64 `json:"max_score"`
	Required   bool    `json:"required"`    // Failure = auto-fail entire evaluation
	GatingRule bool    `json:"gating_rule"` // Must pass to proceed to next stage

	// Actions
	OnPassActions []RuleAction `json:"on_pass_actions,omitempty"`
	OnFailActions []RuleAction `json:"on_fail_actions,omitempty"`

	// Metadata
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveUntil *time.Time `json:"effective_until,omitempty"`
	CreatedBy      uuid.UUID  `json:"created_by"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// RuleCondition represents the condition to evaluate
type RuleCondition struct {
	Type       string          `json:"type"` // SIMPLE, AND, OR, NOT
	Field      string          `json:"field,omitempty"`
	Operator   string          `json:"operator,omitempty"`
	Value      interface{}     `json:"value,omitempty"`
	Children   []RuleCondition `json:"children,omitempty"`    // For compound conditions
	CustomFunc string          `json:"custom_func,omitempty"` // For custom logic
}

// Supported operators
const (
	OpEquals          = "=="
	OpNotEquals       = "!="
	OpGreaterThan     = ">"
	OpGreaterThanOrEq = ">="
	OpLessThan        = "<"
	OpLessThanOrEq    = "<="
	OpIn              = "IN"
	OpNotIn           = "NOT_IN"
	OpContains        = "CONTAINS"
	OpNotContains     = "NOT_CONTAINS"
	OpStartsWith      = "STARTS_WITH"
	OpEndsWith        = "ENDS_WITH"
	OpMatches         = "MATCHES" // Regex
	OpBetween         = "BETWEEN"
	OpIsNull          = "IS_NULL"
	OpIsNotNull       = "IS_NOT_NULL"
	OpIsEmpty         = "IS_EMPTY"
	OpIsNotEmpty      = "IS_NOT_EMPTY"
	OpOlderThanDays   = "OLDER_THAN_DAYS"
	OpNewerThanDays   = "NEWER_THAN_DAYS"
)

// RuleAction represents an action to take on rule evaluation
type RuleAction struct {
	ActionType string                 `json:"action_type"` // NOTIFY, TAG, ESCALATE, UPDATE_FIELD, CUSTOM
	Parameters map[string]interface{} `json:"parameters"`
}

// RuleMetrics tracks rule performance
type RuleMetrics struct {
	RuleID           uuid.UUID `json:"rule_id"`
	TotalEvaluations int64     `json:"total_evaluations"`
	PassCount        int64     `json:"pass_count"`
	FailCount        int64     `json:"fail_count"`
	ErrorCount       int64     `json:"error_count"`
	AvgEvalTimeMs    float64   `json:"avg_eval_time_ms"`
	LastEvaluatedAt  time.Time `json:"last_evaluated_at"`
}

// RuleEvaluationContext provides data for rule evaluation
type RuleEvaluationContext struct {
	Opportunity     map[string]interface{}   `json:"opportunity"`
	Client          map[string]interface{}   `json:"client,omitempty"`
	Portfolio       map[string]interface{}   `json:"portfolio,omitempty"`
	MarketData      map[string]interface{}   `json:"market_data,omitempty"`
	HistoricalDeals []map[string]interface{} `json:"historical_deals,omitempty"`
	Metadata        map[string]interface{}   `json:"metadata,omitempty"`
}

// RuleEvaluationResult contains the result of evaluating a single rule
type RuleEvaluationResult struct {
	RuleID      uuid.UUID          `json:"rule_id"`
	RuleCode    string             `json:"rule_code"`
	RuleName    string             `json:"rule_name"`
	Passed      bool               `json:"passed"`
	Score       float64            `json:"score"`
	MaxScore    float64            `json:"max_score"`
	Required    bool               `json:"required"`
	GatingRule  bool               `json:"gating_rule"`
	ActualValue interface{}        `json:"actual_value,omitempty"`
	Threshold   interface{}        `json:"threshold,omitempty"`
	Details     string             `json:"details"`
	EvalTimeMs  float64            `json:"eval_time_ms"`
	Actions     []RuleActionResult `json:"actions,omitempty"`
}

// RuleActionResult captures executed action results
type RuleActionResult struct {
	ActionType string `json:"action_type"`
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
}

// RuleSetEvaluationResult contains the aggregate result
type RuleSetEvaluationResult struct {
	RuleSetID     uuid.UUID `json:"rule_set_id"`
	RuleSetName   string    `json:"rule_set_name"`
	OpportunityID uuid.UUID `json:"opportunity_id"`

	// Aggregate results
	Passed           bool    `json:"passed"`
	TotalScore       float64 `json:"total_score"`
	MaxPossibleScore float64 `json:"max_possible_score"`
	ScorePercentage  float64 `json:"score_percentage"`

	// Individual results
	RuleResults []RuleEvaluationResult `json:"rule_results"`
	PassedRules int                    `json:"passed_rules"`
	FailedRules int                    `json:"failed_rules"`

	// Analysis
	GatingFailures   []string `json:"gating_failures,omitempty"`
	RequiredFailures []string `json:"required_failures,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
	Recommendations  []string `json:"recommendations,omitempty"`

	// Timing
	EvaluatedAt     time.Time `json:"evaluated_at"`
	TotalEvalTimeMs float64   `json:"total_eval_time_ms"`
}

// NewRuleEngine creates a new rule engine instance
func NewRuleEngine(config RuleEngineConfig) *RuleEngine {
	if config.DefaultPassingScore == 0 {
		config.DefaultPassingScore = 70.0
	}
	if config.MaxParallelRules == 0 {
		config.MaxParallelRules = 10
	}
	if config.TimeoutPerRule == 0 {
		config.TimeoutPerRule = 5 * time.Second
	}

	return &RuleEngine{
		rules:       make([]RuleDefinition, 0),
		config:      config,
		ruleMetrics: make(map[string]*RuleMetrics),
	}
}

// LoadRules loads rules from definitions
func (e *RuleEngine) LoadRules(rules []RuleDefinition) {
	e.rules = rules
}

// EvaluateRuleSet evaluates all applicable rules against the context
func (e *RuleEngine) EvaluateRuleSet(ctx context.Context, ruleSetID uuid.UUID, ruleSetName string, evalCtx *RuleEvaluationContext) (*RuleSetEvaluationResult, error) {
	startTime := time.Now()

	result := &RuleSetEvaluationResult{
		RuleSetID:     ruleSetID,
		RuleSetName:   ruleSetName,
		OpportunityID: getUUIDFromContext(evalCtx, "opportunity.opportunity_id"),
		RuleResults:   make([]RuleEvaluationResult, 0),
		EvaluatedAt:   startTime,
	}

	// Filter applicable rules
	applicableRules := e.filterApplicableRules(evalCtx)

	// Sort by priority
	sortRulesByPriority(applicableRules)

	var totalScore, maxScore float64

	// Evaluate each rule
	for _, rule := range applicableRules {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		ruleResult := e.evaluateRule(ctx, rule, evalCtx)
		result.RuleResults = append(result.RuleResults, ruleResult)

		// Track scores
		maxScore += rule.MaxScore
		if ruleResult.Passed {
			totalScore += ruleResult.Score
			result.PassedRules++
		} else {
			result.FailedRules++

			// Check for blocking failures
			if rule.GatingRule {
				result.GatingFailures = append(result.GatingFailures, rule.RuleName)
			}
			if rule.Required {
				result.RequiredFailures = append(result.RequiredFailures, rule.RuleName)
			}
		}

		// Update metrics
		if e.config.EnableMetrics {
			e.updateMetrics(rule.RuleID.String(), ruleResult)
		}
	}

	// Calculate aggregate scores
	result.TotalScore = totalScore
	result.MaxPossibleScore = maxScore
	if maxScore > 0 {
		result.ScorePercentage = (totalScore / maxScore) * 100
	}

	// Determine overall pass/fail
	result.Passed = len(result.RequiredFailures) == 0 &&
		len(result.GatingFailures) == 0 &&
		result.ScorePercentage >= e.config.DefaultPassingScore

	// Generate recommendations
	result.Recommendations = e.generateRecommendations(result)

	result.TotalEvalTimeMs = float64(time.Since(startTime).Milliseconds())

	return result, nil
}

// evaluateRule evaluates a single rule
func (e *RuleEngine) evaluateRule(ctx context.Context, rule RuleDefinition, evalCtx *RuleEvaluationContext) RuleEvaluationResult {
	startTime := time.Now()

	result := RuleEvaluationResult{
		RuleID:     rule.RuleID,
		RuleCode:   rule.RuleCode,
		RuleName:   rule.RuleName,
		MaxScore:   rule.MaxScore,
		Required:   rule.Required,
		GatingRule: rule.GatingRule,
	}

	// Evaluate condition with timeout
	evalCtxWithTimeout, cancel := context.WithTimeout(ctx, e.config.TimeoutPerRule)
	defer cancel()

	passed, actualValue, err := e.evaluateCondition(evalCtxWithTimeout, rule.Condition, evalCtx)

	result.EvalTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
	result.ActualValue = actualValue
	result.Threshold = rule.Condition.Value

	if err != nil {
		result.Passed = false
		result.Details = fmt.Sprintf("Evaluation error: %v", err)
		return result
	}

	result.Passed = passed
	if passed {
		result.Score = rule.MaxScore * rule.Weight
		result.Details = "Rule passed"

		// Execute pass actions
		for _, action := range rule.OnPassActions {
			actionResult := e.executeAction(action, evalCtx)
			result.Actions = append(result.Actions, actionResult)
		}
	} else {
		result.Score = 0
		result.Details = fmt.Sprintf("Rule failed: %v %s %v", actualValue, rule.Condition.Operator, rule.Condition.Value)

		// Execute fail actions
		for _, action := range rule.OnFailActions {
			actionResult := e.executeAction(action, evalCtx)
			result.Actions = append(result.Actions, actionResult)
		}
	}

	return result
}

// evaluateCondition evaluates a condition against the context
func (e *RuleEngine) evaluateCondition(ctx context.Context, condition RuleCondition, evalCtx *RuleEvaluationContext) (bool, interface{}, error) {
	switch condition.Type {
	case "AND":
		for _, child := range condition.Children {
			passed, val, err := e.evaluateCondition(ctx, child, evalCtx)
			if err != nil {
				return false, val, err
			}
			if !passed {
				return false, val, nil
			}
		}
		return true, nil, nil

	case "OR":
		for _, child := range condition.Children {
			passed, val, err := e.evaluateCondition(ctx, child, evalCtx)
			if err != nil {
				continue // Try next condition on error
			}
			if passed {
				return true, val, nil
			}
		}
		return false, nil, nil

	case "NOT":
		if len(condition.Children) > 0 {
			passed, val, err := e.evaluateCondition(ctx, condition.Children[0], evalCtx)
			return !passed, val, err
		}
		return false, nil, fmt.Errorf("NOT condition requires a child")

	case "SIMPLE", "":
		return e.evaluateSimpleCondition(condition, evalCtx)

	default:
		return false, nil, fmt.Errorf("unknown condition type: %s", condition.Type)
	}
}

// evaluateSimpleCondition evaluates a simple field comparison
func (e *RuleEngine) evaluateSimpleCondition(condition RuleCondition, evalCtx *RuleEvaluationContext) (bool, interface{}, error) {
	// Get actual value from context
	actualValue, err := e.getFieldValue(condition.Field, evalCtx)
	if err != nil {
		// Handle special operators that work with missing values
		if condition.Operator == OpIsNull {
			return true, nil, nil
		}
		if condition.Operator == OpIsNotNull {
			return false, nil, nil
		}
		return false, nil, err
	}

	// Evaluate based on operator
	switch condition.Operator {
	case OpEquals:
		return compareEquals(actualValue, condition.Value), actualValue, nil
	case OpNotEquals:
		return !compareEquals(actualValue, condition.Value), actualValue, nil
	case OpGreaterThan:
		return compareNumeric(actualValue, condition.Value, ">"), actualValue, nil
	case OpGreaterThanOrEq:
		return compareNumeric(actualValue, condition.Value, ">="), actualValue, nil
	case OpLessThan:
		return compareNumeric(actualValue, condition.Value, "<"), actualValue, nil
	case OpLessThanOrEq:
		return compareNumeric(actualValue, condition.Value, "<="), actualValue, nil
	case OpIn:
		return valueInList(actualValue, condition.Value), actualValue, nil
	case OpNotIn:
		return !valueInList(actualValue, condition.Value), actualValue, nil
	case OpContains:
		return stringContains(actualValue, condition.Value), actualValue, nil
	case OpNotContains:
		return !stringContains(actualValue, condition.Value), actualValue, nil
	case OpStartsWith:
		return stringStartsWith(actualValue, condition.Value), actualValue, nil
	case OpEndsWith:
		return stringEndsWith(actualValue, condition.Value), actualValue, nil
	case OpMatches:
		return regexMatches(actualValue, condition.Value), actualValue, nil
	case OpBetween:
		return valueBetween(actualValue, condition.Value), actualValue, nil
	case OpIsNull:
		return actualValue == nil, actualValue, nil
	case OpIsNotNull:
		return actualValue != nil, actualValue, nil
	case OpIsEmpty:
		return isEmpty(actualValue), actualValue, nil
	case OpIsNotEmpty:
		return !isEmpty(actualValue), actualValue, nil
	case OpOlderThanDays:
		return olderThanDays(actualValue, condition.Value), actualValue, nil
	case OpNewerThanDays:
		return newerThanDays(actualValue, condition.Value), actualValue, nil
	default:
		return false, actualValue, fmt.Errorf("unknown operator: %s", condition.Operator)
	}
}

// getFieldValue extracts a field value from the evaluation context
func (e *RuleEngine) getFieldValue(fieldPath string, evalCtx *RuleEvaluationContext) (interface{}, error) {
	parts := strings.Split(fieldPath, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty field path")
	}

	// Determine which context map to use
	var data map[string]interface{}
	switch parts[0] {
	case "opportunity":
		data = evalCtx.Opportunity
		parts = parts[1:]
	case "client":
		data = evalCtx.Client
		parts = parts[1:]
	case "portfolio":
		data = evalCtx.Portfolio
		parts = parts[1:]
	case "market":
		data = evalCtx.MarketData
		parts = parts[1:]
	case "metadata":
		data = evalCtx.Metadata
		parts = parts[1:]
	default:
		// Default to opportunity
		data = evalCtx.Opportunity
	}

	// Navigate the path
	var current interface{} = data
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("field not found: %s", part)
			}
		default:
			return nil, fmt.Errorf("cannot navigate into %T", current)
		}
	}

	return current, nil
}

// filterApplicableRules returns rules that apply to the given context
func (e *RuleEngine) filterApplicableRules(evalCtx *RuleEvaluationContext) []RuleDefinition {
	now := time.Now()
	oppType := getStringFromContext(evalCtx, "opportunity.opportunity_type")
	clientSegment := getStringFromContext(evalCtx, "client.segment")

	applicable := make([]RuleDefinition, 0)
	for _, rule := range e.rules {
		// Check if active
		if !rule.IsActive {
			continue
		}

		// Check effective dates
		if now.Before(rule.EffectiveFrom) {
			continue
		}
		if rule.EffectiveUntil != nil && now.After(*rule.EffectiveUntil) {
			continue
		}

		// Check opportunity type targeting
		if len(rule.OpportunityTypes) > 0 && !stringInSlice(oppType, rule.OpportunityTypes) {
			continue
		}

		// Check client segment targeting
		if len(rule.ClientSegments) > 0 && !stringInSlice(clientSegment, rule.ClientSegments) {
			continue
		}

		applicable = append(applicable, rule)
	}

	return applicable
}

// executeAction executes a rule action
func (e *RuleEngine) executeAction(action RuleAction, evalCtx *RuleEvaluationContext) RuleActionResult {
	result := RuleActionResult{
		ActionType: action.ActionType,
		Success:    true,
	}

	switch action.ActionType {
	case "NOTIFY":
		// In production, this would trigger notification service
		result.Message = fmt.Sprintf("Notification queued: %v", action.Parameters)
	case "TAG":
		// Add tag to opportunity
		result.Message = fmt.Sprintf("Tag added: %v", action.Parameters["tag"])
	case "ESCALATE":
		// Trigger escalation
		result.Message = fmt.Sprintf("Escalated to: %v", action.Parameters["escalate_to"])
	case "UPDATE_FIELD":
		// Update field value
		result.Message = fmt.Sprintf("Field updated: %s = %v", action.Parameters["field"], action.Parameters["value"])
	default:
		result.Success = false
		result.Message = fmt.Sprintf("Unknown action type: %s", action.ActionType)
	}

	return result
}

// updateMetrics updates rule performance metrics
func (e *RuleEngine) updateMetrics(ruleID string, result RuleEvaluationResult) {
	metrics, exists := e.ruleMetrics[ruleID]
	if !exists {
		metrics = &RuleMetrics{RuleID: result.RuleID}
		e.ruleMetrics[ruleID] = metrics
	}

	metrics.TotalEvaluations++
	if result.Passed {
		metrics.PassCount++
	} else {
		metrics.FailCount++
	}
	metrics.LastEvaluatedAt = time.Now()

	// Update rolling average
	metrics.AvgEvalTimeMs = (metrics.AvgEvalTimeMs*float64(metrics.TotalEvaluations-1) + result.EvalTimeMs) / float64(metrics.TotalEvaluations)
}

// generateRecommendations generates recommendations based on results
func (e *RuleEngine) generateRecommendations(result *RuleSetEvaluationResult) []string {
	recommendations := make([]string, 0)

	if len(result.RequiredFailures) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Address required rule failures: %s", strings.Join(result.RequiredFailures, ", ")))
	}

	if len(result.GatingFailures) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Resolve gating issues before proceeding: %s", strings.Join(result.GatingFailures, ", ")))
	}

	if result.ScorePercentage < 50 {
		recommendations = append(recommendations,
			"Score below 50% - recommend declining or significant restructuring")
	} else if result.ScorePercentage < 70 {
		recommendations = append(recommendations,
			"Score between 50-70% - recommend conditional approval with monitoring")
	}

	return recommendations
}

// GetMetrics returns current rule metrics
func (e *RuleEngine) GetMetrics() map[string]*RuleMetrics {
	return e.ruleMetrics
}

// ============================================================================
// COMPARISON HELPERS
// ============================================================================

func compareEquals(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

func compareNumeric(a, b interface{}, op string) bool {
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)

	switch op {
	case ">":
		return aFloat > bFloat
	case ">=":
		return aFloat >= bFloat
	case "<":
		return aFloat < bFloat
	case "<=":
		return aFloat <= bFloat
	default:
		return false
	}
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func valueInList(value interface{}, list interface{}) bool {
	switch l := list.(type) {
	case []interface{}:
		for _, item := range l {
			if compareEquals(value, item) {
				return true
			}
		}
	case []string:
		str := fmt.Sprintf("%v", value)
		for _, item := range l {
			if str == item {
				return true
			}
		}
	}
	return false
}

func stringContains(value, substr interface{}) bool {
	return strings.Contains(fmt.Sprintf("%v", value), fmt.Sprintf("%v", substr))
}

func stringStartsWith(value, prefix interface{}) bool {
	return strings.HasPrefix(fmt.Sprintf("%v", value), fmt.Sprintf("%v", prefix))
}

func stringEndsWith(value, suffix interface{}) bool {
	return strings.HasSuffix(fmt.Sprintf("%v", value), fmt.Sprintf("%v", suffix))
}

func regexMatches(value, pattern interface{}) bool {
	re, err := regexp.Compile(fmt.Sprintf("%v", pattern))
	if err != nil {
		return false
	}
	return re.MatchString(fmt.Sprintf("%v", value))
}

func valueBetween(value, bounds interface{}) bool {
	switch b := bounds.(type) {
	case []interface{}:
		if len(b) >= 2 {
			v := toFloat64(value)
			return v >= toFloat64(b[0]) && v <= toFloat64(b[1])
		}
	case map[string]interface{}:
		v := toFloat64(value)
		min := toFloat64(b["min"])
		max := toFloat64(b["max"])
		return v >= min && v <= max
	}
	return false
}

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() == 0
	}
	return false
}

func olderThanDays(value, days interface{}) bool {
	t, ok := value.(time.Time)
	if !ok {
		return false
	}
	d := int(toFloat64(days))
	return time.Since(t).Hours()/24 > float64(d)
}

func newerThanDays(value, days interface{}) bool {
	t, ok := value.(time.Time)
	if !ok {
		return false
	}
	d := int(toFloat64(days))
	return time.Since(t).Hours()/24 <= float64(d)
}

func stringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func sortRulesByPriority(rules []RuleDefinition) {
	// Simple bubble sort for clarity (use sort.Slice in production)
	for i := 0; i < len(rules)-1; i++ {
		for j := 0; j < len(rules)-i-1; j++ {
			if rules[j].Priority > rules[j+1].Priority {
				rules[j], rules[j+1] = rules[j+1], rules[j]
			}
		}
	}
}

func getStringFromContext(ctx *RuleEvaluationContext, path string) string {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return ""
	}

	var data map[string]interface{}
	switch parts[0] {
	case "opportunity":
		data = ctx.Opportunity
	case "client":
		data = ctx.Client
	default:
		return ""
	}

	if val, ok := data[parts[1]]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func getUUIDFromContext(ctx *RuleEvaluationContext, path string) uuid.UUID {
	str := getStringFromContext(ctx, path)
	if str == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(str)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// ============================================================================
// PREDEFINED RULE TEMPLATES
// ============================================================================

// GetDefaultScreeningRules returns standard screening rules
func GetDefaultScreeningRules() []RuleDefinition {
	return []RuleDefinition{
		{
			RuleID:   uuid.New(),
			RuleCode: "SCREEN_MIN_FUND_SIZE",
			RuleName: "Minimum Fund Size",
			Category: "SCREENING",
			Version:  1,
			IsActive: true,
			Priority: 10,
			Condition: RuleCondition{
				Type:     "SIMPLE",
				Field:    "opportunity.fund_size",
				Operator: OpGreaterThanOrEq,
				Value:    50000000, // $50M minimum
			},
			Weight:        0.1,
			MaxScore:      10,
			Required:      false,
			GatingRule:    false,
			EffectiveFrom: time.Now().AddDate(-1, 0, 0),
		},
		{
			RuleID:   uuid.New(),
			RuleCode: "SCREEN_TRACK_RECORD",
			RuleName: "Manager Track Record",
			Category: "SCREENING",
			Version:  1,
			IsActive: true,
			Priority: 5,
			Condition: RuleCondition{
				Type:     "SIMPLE",
				Field:    "opportunity.track_record_years_min",
				Operator: OpGreaterThanOrEq,
				Value:    3,
			},
			Weight:        0.2,
			MaxScore:      20,
			Required:      true,
			GatingRule:    true,
			EffectiveFrom: time.Now().AddDate(-1, 0, 0),
			OnFailActions: []RuleAction{
				{ActionType: "TAG", Parameters: map[string]interface{}{"tag": "insufficient_track_record"}},
				{ActionType: "NOTIFY", Parameters: map[string]interface{}{"template": "track_record_warning"}},
			},
		},
		{
			RuleID:   uuid.New(),
			RuleCode: "SCREEN_TARGET_IRR",
			RuleName: "Target IRR Threshold",
			Category: "SCREENING",
			Version:  1,
			IsActive: true,
			Priority: 15,
			Condition: RuleCondition{
				Type:     "SIMPLE",
				Field:    "opportunity.target_irr_min",
				Operator: OpGreaterThanOrEq,
				Value:    12,
			},
			Weight:        0.15,
			MaxScore:      15,
			Required:      false,
			GatingRule:    false,
			EffectiveFrom: time.Now().AddDate(-1, 0, 0),
		},
		{
			RuleID:   uuid.New(),
			RuleCode: "SCREEN_MGMT_FEE",
			RuleName: "Management Fee Cap",
			Category: "SCREENING",
			Version:  1,
			IsActive: true,
			Priority: 20,
			Condition: RuleCondition{
				Type:     "SIMPLE",
				Field:    "opportunity.management_fee_rate",
				Operator: OpLessThanOrEq,
				Value:    0.025, // 2.5%
			},
			Weight:        0.1,
			MaxScore:      10,
			Required:      false,
			GatingRule:    false,
			EffectiveFrom: time.Now().AddDate(-1, 0, 0),
		},
		{
			RuleID:   uuid.New(),
			RuleCode: "SCREEN_LEVERAGE",
			RuleName: "Maximum Leverage Ratio",
			Category: "SCREENING",
			Version:  1,
			IsActive: true,
			Priority: 8,
			Condition: RuleCondition{
				Type:     "SIMPLE",
				Field:    "opportunity.max_leverage_ratio",
				Operator: OpLessThanOrEq,
				Value:    4.0,
			},
			Weight:        0.15,
			MaxScore:      15,
			Required:      true,
			GatingRule:    true,
			EffectiveFrom: time.Now().AddDate(-1, 0, 0),
			OnFailActions: []RuleAction{
				{ActionType: "ESCALATE", Parameters: map[string]interface{}{"reason": "High leverage"}},
			},
		},
		{
			RuleID:   uuid.New(),
			RuleCode: "SCREEN_LIQUIDITY",
			RuleName: "Client Liquidity Check",
			Category: "SCREENING",
			Version:  1,
			IsActive: true,
			Priority: 3,
			Condition: RuleCondition{
				Type: "AND",
				Children: []RuleCondition{
					{
						Type:     "SIMPLE",
						Field:    "client.liquid_assets",
						Operator: OpGreaterThanOrEq,
						Value:    500000,
					},
					{
						Type:     "SIMPLE",
						Field:    "portfolio.liquidity_ratio",
						Operator: OpGreaterThanOrEq,
						Value:    0.2,
					},
				},
			},
			Weight:        0.2,
			MaxScore:      20,
			Required:      true,
			GatingRule:    true,
			EffectiveFrom: time.Now().AddDate(-1, 0, 0),
		},
	}
}
