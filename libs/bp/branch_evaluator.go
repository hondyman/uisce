// Package bp provides comprehensive business process workflow management
// with advanced branching capabilities including ML-powered routing,
// parallel execution, nested branches, and event-based gateways.
package bp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================
// Core Types
// ============================================

// BranchingConfig defines the branching strategy for a workflow step
type BranchingConfig struct {
	Type            string      `json:"type"`             // exclusive|inclusive|parallel|event|weighted|ml_powered
	GatewayPosition string      `json:"gateway_position"` // diverging|converging
	Description     string      `json:"description"`
	Branches        []Branch    `json:"branches"`
	DefaultBranchID string      `json:"default_branch_id"`
	MLConfig        *MLConfig   `json:"ml_config,omitempty"`
	JoinConfig      *JoinConfig `json:"join_config,omitempty"`
	MaxNestingDepth int         `json:"max_nesting_depth,omitempty"`
	InheritContext  bool        `json:"inherit_context,omitempty"`
	AllowLoopBack   bool        `json:"allow_loop_back,omitempty"`
	MaxIterations   int         `json:"max_iterations,omitempty"`
}

// Branch represents a single execution path in a branching gateway
type Branch struct {
	ID              string           `json:"id"`
	Label           string           `json:"label"`
	Priority        int              `json:"priority"`
	Condition       *Condition       `json:"condition"`
	Weight          float64          `json:"weight,omitempty"`
	Steps           []string         `json:"steps"`
	NestedBranching *BranchingConfig `json:"nested_branching,omitempty"`
	LoopBackConfig  *LoopBackConfig  `json:"loop_back_config,omitempty"`
	TimeoutConfig   *TimeoutConfig   `json:"timeout_config,omitempty"`
	Critical        bool             `json:"critical,omitempty"`
	SLAHours        int              `json:"sla_hours,omitempty"`
	Notification    *Notification    `json:"notification,omitempty"`
}

// Condition represents a condition to evaluate for branch selection
type Condition struct {
	Type         string      `json:"type"` // and|or|expression|ml_score
	Rules        []Rule      `json:"rules,omitempty"`
	Expression   string      `json:"expression,omitempty"`
	Children     []Condition `json:"children,omitempty"`
	Operator     string      `json:"operator,omitempty"` // For ML conditions: gte|lte|lt|gt|between
	Threshold    float64     `json:"threshold,omitempty"`
	ThresholdMin float64     `json:"threshold_min,omitempty"`
	ThresholdMax float64     `json:"threshold_max,omitempty"`
}

// Rule represents a single evaluation rule
type Rule struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq|ne|gt|gte|lt|lte|in|contains|matches
	Value    interface{} `json:"value"`
}

// JoinConfig defines how parallel/inclusive branches converge
type JoinConfig struct {
	Strategy               string      `json:"strategy"`       // wait_all|first_complete|m_of_n|majority_vote
	TimeoutAction          string      `json:"timeout_action"` // proceed|cancel|escalate
	TimeoutHours           int         `json:"timeout_hours"`
	PartialResultsHandling string      `json:"partial_results_handling"`
	MOfN                   *MOfNConfig `json:"join_config,omitempty"`
	CriticalOnly           bool        `json:"critical_only,omitempty"`
}

// MOfNConfig specifies M-of-N join requirement
type MOfNConfig struct {
	M int `json:"m"`
	N int `json:"n"`
}

// MLConfig configures ML-powered branch routing
type MLConfig struct {
	ModelID             string        `json:"model_id"`
	ModelEndpoint       string        `json:"model_endpoint"`
	InputFeatures       []string      `json:"input_features"`
	ConfidenceThreshold float64       `json:"confidence_threshold"`
	FallbackStrategy    string        `json:"fallback_strategy"` // conservative|optimistic
	ModelMonitoring     *MLMonitoring `json:"model_monitoring,omitempty"`
}

// MLMonitoring configures ML model monitoring
type MLMonitoring struct {
	TrackPredictions bool `json:"track_predictions"`
	AlertOnDrift     bool `json:"alert_on_drift"`
	FeedbackLoop     bool `json:"feedback_loop"`
}

// LoopBackConfig defines loop-back branching for corrections
type LoopBackConfig struct {
	TargetStepID            string                 `json:"target_step_id"`
	MaxIterations           int                    `json:"max_iterations"`
	IterationCounterField   string                 `json:"iteration_counter_field"`
	OnMaxIterationsExceeded map[string]interface{} `json:"on_max_iterations_exceeded"`
}

// TimeoutConfig defines timeout-based branch selection
type TimeoutConfig struct {
	TimeoutHours     int    `json:"timeout_hours"`
	FallbackBranchID string `json:"fallback_branch_id"`
}

// Notification defines notification configuration for a branch
type Notification struct {
	Channels   []string `json:"channels"`
	Recipients []string `json:"recipients"`
	Template   string   `json:"template"`
	Urgent     bool     `json:"urgent,omitempty"`
}

// BranchExecutionResult captures the result of branch execution
type BranchExecutionResult struct {
	ExecutionID       uuid.UUID
	BranchID          string
	BranchLabel       string
	SelectedBy        string
	Status            string
	StartedAt         time.Time
	CompletedAt       *time.Time
	DurationMs        int
	MLModelScore      *float64
	MLModelConfidence *float64
	ConditionEval     map[string]interface{}
	ResultData        map[string]interface{}
	NextStepID        *uuid.UUID
	JoinStrategy      string
	IsLastInJoin      bool
	NestingLevel      int
	LoopIteration     int
}

// BranchEvaluator orchestrates branch evaluation and selection
type BranchEvaluator struct {
	db *sqlx.DB
	// Optional Hasura GraphQL client for Hasura-first execution paths
	hasura HasuraClient
}

// NewBranchEvaluator creates a new branch evaluator
func NewBranchEvaluator(db *sqlx.DB) *BranchEvaluator {
	return &BranchEvaluator{db: db}
}

// NewBranchEvaluatorWithHasura creates evaluator with optional Hasura client
func NewBranchEvaluatorWithHasura(db *sqlx.DB, hasura HasuraClient) *BranchEvaluator {
	return &BranchEvaluator{db: db, hasura: hasura}
}

// ============================================
// Main Evaluation Logic
// ============================================

// EvaluateBranches evaluates the branching config and returns selected branches
func (e *BranchEvaluator) EvaluateBranches(
	ctx context.Context,
	config *BranchingConfig,
	data map[string]interface{},
) ([]Branch, error) {

	if config == nil {
		return nil, fmt.Errorf("branching config is nil")
	}

	switch config.Type {
	case "exclusive":
		return e.evaluateExclusive(config, data)
	case "inclusive":
		return e.evaluateInclusive(config, data)
	case "parallel":
		return e.evaluateParallel(config, data)
	case "weighted":
		return e.evaluateWeighted(config, data)
	case "ml_powered":
		return e.evaluateMLPowered(ctx, config, data)
	case "event":
		return e.evaluateEvent(ctx, config, data)
	default:
		return nil, fmt.Errorf("unsupported branching type: %s", config.Type)
	}
}

// ============================================
// Exclusive Gateway (XOR)
// ============================================

// evaluateExclusive evaluates XOR - single path selection
func (e *BranchEvaluator) evaluateExclusive(config *BranchingConfig, data map[string]interface{}) ([]Branch, error) {
	branches := make([]Branch, len(config.Branches))
	copy(branches, config.Branches)
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Priority < branches[j].Priority
	})

	for _, branch := range branches {
		if branch.Condition == nil {
			continue
		}

		if e.evaluateCondition(branch.Condition, data) {
			return []Branch{branch}, nil
		}
	}

	if config.DefaultBranchID != "" {
		for _, branch := range branches {
			if branch.ID == config.DefaultBranchID {
				return []Branch{branch}, nil
			}
		}
	}

	return nil, fmt.Errorf("no matching branch found in exclusive gateway")
}

// ============================================
// Inclusive Gateway (OR)
// ============================================

// evaluateInclusive evaluates OR - multiple independent paths
func (e *BranchEvaluator) evaluateInclusive(config *BranchingConfig, data map[string]interface{}) ([]Branch, error) {
	var selectedBranches []Branch

	for _, branch := range config.Branches {
		if branch.Condition == nil {
			selectedBranches = append(selectedBranches, branch)
			continue
		}

		if e.evaluateCondition(branch.Condition, data) {
			selectedBranches = append(selectedBranches, branch)
		}
	}

	if len(selectedBranches) == 0 {
		return nil, fmt.Errorf("no branches matched in inclusive gateway")
	}

	return selectedBranches, nil
}

// ============================================
// Parallel Gateway (AND)
// ============================================

// evaluateParallel evaluates AND - all paths execute
func (e *BranchEvaluator) evaluateParallel(config *BranchingConfig, data map[string]interface{}) ([]Branch, error) {
	var selectedBranches []Branch

	for _, branch := range config.Branches {
		if branch.Condition != nil {
			if !e.evaluateCondition(branch.Condition, data) {
				continue
			}
		}
		selectedBranches = append(selectedBranches, branch)
	}

	if len(selectedBranches) == 0 {
		return config.Branches, nil
	}

	return selectedBranches, nil
}

// ============================================
// Weighted Gateway (Probabilistic)
// ============================================

// evaluateWeighted evaluates probabilistic routing for A/B testing
func (e *BranchEvaluator) evaluateWeighted(config *BranchingConfig, data map[string]interface{}) ([]Branch, error) {
	totalWeight := 0.0
	for _, branch := range config.Branches {
		totalWeight += branch.Weight
	}

	if totalWeight <= 0 {
		return nil, fmt.Errorf("total weight must be positive")
	}

	randomValue := rand.Float64() * totalWeight
	cumulativeWeight := 0.0
	for _, branch := range config.Branches {
		cumulativeWeight += branch.Weight
		if randomValue <= cumulativeWeight {
			return []Branch{branch}, nil
		}
	}

	return nil, fmt.Errorf("weighted selection failed")
}

// ============================================
// ML-Powered Gateway
// ============================================

// evaluateMLPowered evaluates ML-based routing
func (e *BranchEvaluator) evaluateMLPowered(ctx context.Context, config *BranchingConfig, data map[string]interface{}) ([]Branch, error) {
	if config.MLConfig == nil {
		return nil, fmt.Errorf("ML config missing for ml_powered branching")
	}

	// Placeholder for ML prediction - actual implementation would call real ML service
	score := 0.7

	sort.Slice(config.Branches, func(i, j int) bool {
		return config.Branches[i].Priority < config.Branches[j].Priority
	})

	for _, branch := range config.Branches {
		if branch.Condition != nil && branch.Condition.Type == "ml_score" {
			if e.evaluateMLCondition(branch.Condition, score) {
				return []Branch{branch}, nil
			}
		}
	}

	return e.handleMLFallback(config, data)
}

// evaluateMLCondition evaluates ML score condition
func (e *BranchEvaluator) evaluateMLCondition(condition *Condition, score float64) bool {
	switch condition.Operator {
	case "gte":
		return score >= condition.Threshold
	case "lte":
		return score <= condition.Threshold
	case "gt":
		return score > condition.Threshold
	case "lt":
		return score < condition.Threshold
	case "between":
		return score >= condition.ThresholdMin && score <= condition.ThresholdMax
	default:
		return false
	}
}

// handleMLFallback handles ML model fallback
func (e *BranchEvaluator) handleMLFallback(config *BranchingConfig, data map[string]interface{}) ([]Branch, error) {
	strategy := "conservative"
	if config.MLConfig != nil && config.MLConfig.FallbackStrategy != "" {
		strategy = config.MLConfig.FallbackStrategy
	}

	switch strategy {
	case "conservative":
		if config.DefaultBranchID != "" {
			for _, branch := range config.Branches {
				if branch.ID == config.DefaultBranchID {
					return []Branch{branch}, nil
				}
			}
		}
	case "optimistic":
		if len(config.Branches) > 0 {
			return []Branch{config.Branches[0]}, nil
		}
	case "random":
		if len(config.Branches) > 0 {
			idx := rand.Intn(len(config.Branches))
			return []Branch{config.Branches[idx]}, nil
		}
	}

	return nil, fmt.Errorf("ML fallback failed")
}

// ============================================
// Event-Based Gateway
// ============================================

// evaluateEvent evaluates event-based routing
func (e *BranchEvaluator) evaluateEvent(ctx context.Context, config *BranchingConfig, data map[string]interface{}) ([]Branch, error) {
	return config.Branches, nil
}

// ============================================
// Condition Evaluation
// ============================================

// evaluateCondition evaluates a condition against data
func (e *BranchEvaluator) evaluateCondition(condition *Condition, data map[string]interface{}) bool {
	if condition == nil {
		return true
	}

	switch condition.Type {
	case "and":
		return e.evaluateAndCondition(condition, data)
	case "or":
		return e.evaluateOrCondition(condition, data)
	case "expression":
		return e.evaluateExpression(condition.Expression, data)
	default:
		return false
	}
}

// evaluateAndCondition evaluates AND logic
func (e *BranchEvaluator) evaluateAndCondition(condition *Condition, data map[string]interface{}) bool {
	for _, rule := range condition.Rules {
		if !e.evaluateRule(rule, data) {
			return false
		}
	}
	for _, child := range condition.Children {
		if !e.evaluateCondition(&child, data) {
			return false
		}
	}
	return true
}

// evaluateOrCondition evaluates OR logic
func (e *BranchEvaluator) evaluateOrCondition(condition *Condition, data map[string]interface{}) bool {
	for _, rule := range condition.Rules {
		if e.evaluateRule(rule, data) {
			return true
		}
	}
	for _, child := range condition.Children {
		if e.evaluateCondition(&child, data) {
			return true
		}
	}
	return false
}

// evaluateRule evaluates a single rule
func (e *BranchEvaluator) evaluateRule(rule Rule, data map[string]interface{}) bool {
	value := e.extractNestedValue(data, rule.Field)
	if value == nil {
		return false
	}

	switch rule.Operator {
	case "eq":
		return value == rule.Value
	case "ne":
		return value != rule.Value
	case "gt":
		return e.compareValues(value, rule.Value) > 0
	case "gte":
		return e.compareValues(value, rule.Value) >= 0
	case "lt":
		return e.compareValues(value, rule.Value) < 0
	case "lte":
		return e.compareValues(value, rule.Value) <= 0
	case "in":
		return e.valueInList(value, rule.Value)
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", value), fmt.Sprintf("%v", rule.Value))
	default:
		return false
	}
}

// evaluateExpression evaluates expression (simplified)
func (e *BranchEvaluator) evaluateExpression(expr string, data map[string]interface{}) bool {
	return true
}

// compareValues compares two values numerically
func (e *BranchEvaluator) compareValues(a, b interface{}) int {
	aF := e.toFloat(a)
	bF := e.toFloat(b)
	if aF < bF {
		return -1
	} else if aF > bF {
		return 1
	}
	return 0
}

// toFloat converts value to float64
func (e *BranchEvaluator) toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

// valueInList checks if value is in list
func (e *BranchEvaluator) valueInList(value, listVal interface{}) bool {
	valueStr := fmt.Sprintf("%v", value)
	if lst, ok := listVal.([]interface{}); ok {
		for _, item := range lst {
			if fmt.Sprintf("%v", item) == valueStr {
				return true
			}
		}
	}
	return false
}

// extractNestedValue extracts value from nested map using dot notation
func (e *BranchEvaluator) extractNestedValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return nil
		}
	}

	return current
}

// ============================================
// Execution Logging
// ============================================

// LogBranchExecution logs branch execution for analytics
// TODO: Refactor to Hasura GraphQL
//
//	mutation {
//	  insert_bp_branch_executions_one(object: {
//	    tenant_id: "uuid", datasource_id: "uuid", workflow_instance_id: "uuid", step_id: "uuid"
//	    branch_id: "branch-1", branch_label: "High Risk", selected_by: "ml_model"
//	    condition_evaluation: {field: "value"}, ml_model_score: 0.85
//	    started_at: "2024-01-15T10:00:00Z", completed_at: "2024-01-15T10:05:00Z"
//	    duration_ms: 300000, status: "completed", result_data: {}
//	    next_step_id: "uuid", join_strategy: "wait_all", is_last_in_join: false
//	    nesting_level: 0, loop_iteration: 1
//	  }) { id }
//	}
func (e *BranchEvaluator) LogBranchExecution(
	ctx context.Context,
	tenantID, datasourceID, workflowInstanceID, stepID uuid.UUID,
	result *BranchExecutionResult,
) error {

	mlScore := sql.NullFloat64{}
	if result.MLModelScore != nil {
		mlScore = sql.NullFloat64{Float64: *result.MLModelScore, Valid: true}
	}

	conditionEval, _ := json.Marshal(result.ConditionEval)
	resultData, _ := json.Marshal(result.ResultData)

	_, err := e.db.ExecContext(ctx, `
		INSERT INTO bp_branch_executions (
			tenant_id, datasource_id, workflow_instance_id, step_id,
			branch_id, branch_label, selected_by, condition_evaluation,
			ml_model_score, started_at, completed_at, duration_ms,
			status, result_data, next_step_id, join_strategy,
			is_last_in_join, nesting_level, loop_iteration
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`,
		tenantID, datasourceID, workflowInstanceID, stepID,
		result.BranchID, result.BranchLabel, result.SelectedBy, conditionEval,
		mlScore, result.StartedAt, result.CompletedAt, result.DurationMs,
		result.Status, resultData, result.NextStepID, result.JoinStrategy,
		result.IsLastInJoin, result.NestingLevel, result.LoopIteration,
	)

	return err
}

// ============================================
// Join Management
// ============================================

// CreateJoinPoint creates a join convergence point for parallel/inclusive branches
// TODO: Refactor to Hasura GraphQL
//
//	mutation {
//	  insert_bp_join_convergences_one(object: {
//	    tenant_id: "uuid", workflow_instance_id: "uuid", step_id: "uuid"
//	    join_id: "join-1", join_strategy: "wait_all", required_branches: 3, status: "waiting"
//	  }) { id }
//	}
func (e *BranchEvaluator) CreateJoinPoint(
	ctx context.Context,
	tenantID, datasourceID, workflowInstanceID, stepID uuid.UUID,
	joinID string,
	strategy string,
	requiredBranches int,
) (string, error) {

	var id string
	err := e.db.GetContext(ctx, &id, `
		INSERT INTO bp_join_convergences (
			tenant_id, workflow_instance_id, step_id, join_id,
			join_strategy, required_branches, status
		) VALUES ($1, $2, $3, $4, $5, $6, 'waiting')
		RETURNING id
	`, tenantID, workflowInstanceID, stepID, joinID, strategy, requiredBranches)

	return id, err
}

// CheckJoinCompletion checks if join is complete based on strategy
func (e *BranchEvaluator) CheckJoinCompletion(
	ctx context.Context,
	joinID uuid.UUID,
	strategy string,
	mValue int,
) (bool, error) {

	var result struct {
		Completed int `db:"completed_branches"`
		Required  int `db:"required_branches"`
	}

	err := e.db.GetContext(ctx, &result, `
		SELECT completed_branches, required_branches FROM bp_join_convergences WHERE id = $1
	`, joinID)

	if err != nil {
		return false, err
	}

	switch strategy {
	case "wait_all":
		return result.Completed >= result.Required, nil
	case "first_complete":
		return result.Completed >= 1, nil
	case "m_of_n":
		return result.Completed >= mValue, nil
	case "majority_vote":
		return result.Completed > result.Required/2, nil
	default:
		return false, fmt.Errorf("unknown join strategy: %s", strategy)
	}
}

// ============================================
// Loop-Back Management
// ============================================

// CheckLoopBackEligibility checks if branch can loop back
func (e *BranchEvaluator) CheckLoopBackEligibility(
	loopConfig *LoopBackConfig,
	currentIteration int,
) bool {
	return loopConfig != nil && currentIteration < loopConfig.MaxIterations
}

// GetLoopBackTarget returns the target step for loop-back
func (e *BranchEvaluator) GetLoopBackTarget(loopConfig *LoopBackConfig) string {
	if loopConfig != nil {
		return loopConfig.TargetStepID
	}
	return ""
}
