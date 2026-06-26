package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hondyman/semlayer/backend/pkg/llm"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Condition Engine - Workday-Inspired Semantic Condition Evaluation
// ============================================================================

// ConditionType defines the type of condition
type ConditionType string

const (
	ConditionBoolean  ConditionType = "Boolean"
	ConditionSemantic ConditionType = "Semantic"
	ConditionLLM      ConditionType = "LLM"
	ConditionPolicy   ConditionType = "Policy"
)

// ConditionRule defines a condition specification
type ConditionRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Type        ConditionType `json:"type"`
	Description string        `json:"description"`

	// Boolean condition
	Expression string `json:"expression,omitempty"` // e.g., "$.amount > 10000 && $.risk_level == 'high'"

	// Semantic condition
	SemanticKey    string `json:"semantic_key,omitempty"`    // Semantic layer key to evaluate
	SemanticQuery  string `json:"semantic_query,omitempty"`  // GraphQL or semantic query
	ExpectedResult string `json:"expected_result,omitempty"` // Expected value for comparison

	// LLM condition
	LLMProfile  string            `json:"llm_profile,omitempty"`
	LLMContext  map[string]string `json:"llm_context,omitempty"`  // State paths to include
	LLMQuestion string            `json:"llm_question,omitempty"` // Question for LLM to answer yes/no

	// Policy condition
	PolicyRef string `json:"policy_ref,omitempty"` // Reference to policy engine rule
}

// ConditionTrace captures the full condition evaluation for audit
type ConditionTrace struct {
	RuleID            string                 `json:"rule_id"`
	RuleName          string                 `json:"rule_name"`
	RuleType          string                 `json:"rule_type"`
	Expression        string                 `json:"expression,omitempty"`
	DataSnapshot      map[string]interface{} `json:"data_snapshot"`
	LLMInterpretation string                 `json:"llm_interpretation,omitempty"`
	Result            bool                   `json:"result"`
	BranchTaken       string                 `json:"branch_taken"`
	Error             string                 `json:"error,omitempty"`
}

// ConditionResult holds the result of condition evaluation
type ConditionResult struct {
	Result      bool            `json:"result"`
	BranchTaken string          `json:"branch_taken"` // e.g., "true", "false", "approved", "rejected"
	Trace       *ConditionTrace `json:"trace"`
}

// ============================================================================
// Condition Evaluation Functions
// ============================================================================

// EvaluateCondition evaluates a condition and returns result with trace
func EvaluateCondition(
	ctx workflow.Context,
	rule ConditionRule,
	currentState map[string]interface{},
) (*ConditionResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Evaluating condition", "ruleID", rule.ID, "type", rule.Type)

	trace := &ConditionTrace{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		RuleType:     string(rule.Type),
		DataSnapshot: make(map[string]interface{}),
	}

	var result bool
	var err error

	switch rule.Type {
	case ConditionBoolean:
		result, err = evaluateBooleanCondition(rule, currentState, trace)

	case ConditionSemantic:
		result, err = evaluateSemanticCondition(ctx, rule, currentState, trace)

	case ConditionLLM:
		result, err = evaluateLLMCondition(ctx, rule, currentState, trace)

	case ConditionPolicy:
		result, err = evaluatePolicyCondition(ctx, rule, currentState, trace)

	default:
		err = fmt.Errorf("unknown condition type: %s", rule.Type)
	}

	if err != nil {
		trace.Error = err.Error()
		return nil, err
	}

	trace.Result = result
	trace.BranchTaken = boolToBranch(result)

	return &ConditionResult{
		Result:      result,
		BranchTaken: trace.BranchTaken,
		Trace:       trace,
	}, nil
}

// ============================================================================
// Boolean Condition Evaluation
// ============================================================================

func evaluateBooleanCondition(
	rule ConditionRule,
	state map[string]interface{},
	trace *ConditionTrace,
) (bool, error) {
	trace.Expression = rule.Expression

	// Parse and evaluate the expression
	expr := rule.Expression
	if expr == "" {
		return true, nil // No expression = always true
	}

	// Extract variable references and evaluate
	result, err := evaluateExpression(expr, state, trace)
	if err != nil {
		return false, fmt.Errorf("expression evaluation failed: %w", err)
	}

	return result, nil
}

// evaluateExpression handles simple boolean expressions
func evaluateExpression(expr string, state map[string]interface{}, trace *ConditionTrace) (bool, error) {
	// Handle compound expressions with && and ||
	if strings.Contains(expr, "&&") {
		parts := strings.Split(expr, "&&")
		for _, part := range parts {
			result, err := evaluateExpression(strings.TrimSpace(part), state, trace)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil // Short circuit
			}
		}
		return true, nil
	}

	if strings.Contains(expr, "||") {
		parts := strings.Split(expr, "||")
		for _, part := range parts {
			result, err := evaluateExpression(strings.TrimSpace(part), state, trace)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil // Short circuit
			}
		}
		return false, nil
	}

	// Handle comparison operators
	operators := []string{">=", "<=", "!=", "==", ">", "<"}
	for _, op := range operators {
		if strings.Contains(expr, op) {
			parts := strings.SplitN(expr, op, 2)
			if len(parts) == 2 {
				leftStr := strings.TrimSpace(parts[0])
				rightStr := strings.TrimSpace(parts[1])

				left, err := resolveValue(leftStr, state, trace)
				if err != nil {
					return false, err
				}
				right, err := resolveValue(rightStr, state, trace)
				if err != nil {
					return false, err
				}

				return compareValues(left, op, right), nil
			}
		}
	}

	// Handle simple truthy check
	value, err := resolveValue(strings.TrimSpace(expr), state, trace)
	if err != nil {
		return false, err
	}

	return isTruthy(value), nil
}

// resolveValue resolves a value from state or literal
func resolveValue(valueStr string, state map[string]interface{}, trace *ConditionTrace) (interface{}, error) {
	valueStr = strings.TrimSpace(valueStr)

	// JSONPath reference
	if strings.HasPrefix(valueStr, "$.") {
		value, err := resolveDataPath(valueStr, state)
		if err != nil {
			return nil, err
		}
		trace.DataSnapshot[valueStr] = value
		return value, nil
	}

	// Quoted string literal
	if (strings.HasPrefix(valueStr, "'") && strings.HasSuffix(valueStr, "'")) ||
		(strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"")) {
		return valueStr[1 : len(valueStr)-1], nil
	}

	// Numeric literal
	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return num, nil
	}

	// Boolean literal
	if valueStr == "true" {
		return true, nil
	}
	if valueStr == "false" {
		return false, nil
	}

	// Null
	if valueStr == "null" || valueStr == "nil" {
		return nil, nil
	}

	// Try as state path without $. prefix
	if value, err := resolveDataPath("$."+valueStr, state); err == nil {
		trace.DataSnapshot[valueStr] = value
		return value, nil
	}

	return valueStr, nil
}

// compareValues compares two values with an operator
func compareValues(left interface{}, op string, right interface{}) bool {
	// Convert to comparable types
	leftNum, leftIsNum := toNumber(left)
	rightNum, rightIsNum := toNumber(right)

	switch op {
	case "==":
		if leftIsNum && rightIsNum {
			return leftNum == rightNum
		}
		return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)

	case "!=":
		if leftIsNum && rightIsNum {
			return leftNum != rightNum
		}
		return fmt.Sprintf("%v", left) != fmt.Sprintf("%v", right)

	case ">":
		if leftIsNum && rightIsNum {
			return leftNum > rightNum
		}
		return fmt.Sprintf("%v", left) > fmt.Sprintf("%v", right)

	case "<":
		if leftIsNum && rightIsNum {
			return leftNum < rightNum
		}
		return fmt.Sprintf("%v", left) < fmt.Sprintf("%v", right)

	case ">=":
		if leftIsNum && rightIsNum {
			return leftNum >= rightNum
		}
		return fmt.Sprintf("%v", left) >= fmt.Sprintf("%v", right)

	case "<=":
		if leftIsNum && rightIsNum {
			return leftNum <= rightNum
		}
		return fmt.Sprintf("%v", left) <= fmt.Sprintf("%v", right)
	}

	return false
}

// toNumber attempts to convert interface{} to float64
func toNumber(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case string:
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// isTruthy checks if a value is truthy
func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "false" && val != "0"
	case int, int64, float64:
		return val != 0
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	}
	return true
}

func boolToBranch(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ============================================================================
// Semantic Condition Evaluation
// ============================================================================

func evaluateSemanticCondition(
	ctx workflow.Context,
	rule ConditionRule,
	state map[string]interface{},
	trace *ConditionTrace,
) (bool, error) {
	trace.Expression = rule.SemanticQuery

	// Execute semantic query via activity
	var result interface{}
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 10,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, SemanticConditionActivity,
		rule.SemanticKey, rule.SemanticQuery, state).Get(ctx, &result)

	if err != nil {
		return false, fmt.Errorf("semantic query failed: %w", err)
	}

	trace.DataSnapshot["semantic_result"] = result

	// Compare with expected
	if rule.ExpectedResult != "" {
		return fmt.Sprintf("%v", result) == rule.ExpectedResult, nil
	}

	return isTruthy(result), nil
}

// SemanticConditionActivity evaluates semantic layer queries
func SemanticConditionActivity(ctx context.Context, semanticKey string, query string, state map[string]interface{}) (interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Evaluating semantic condition", "key", semanticKey)

	// For now, try to resolve from state if semantic key is provided
	if semanticKey != "" {
		value, err := resolveDataPath(semanticKey, state)
		if err == nil {
			return value, nil
		}
	}

	// TODO: Execute actual semantic layer query (GraphQL/Hasura)
	// For now, return placeholder
	return nil, fmt.Errorf("semantic query evaluation not yet implemented")
}

// ============================================================================
// LLM Condition Evaluation
// ============================================================================

func evaluateLLMCondition(
	ctx workflow.Context,
	rule ConditionRule,
	state map[string]interface{},
	trace *ConditionTrace,
) (bool, error) {
	trace.Expression = rule.LLMQuestion

	// Execute LLM evaluation via activity
	var result LLMConditionResult
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 5,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, LLMConditionActivity,
		rule, state).Get(ctx, &result)

	if err != nil {
		return false, fmt.Errorf("LLM condition evaluation failed: %w", err)
	}

	trace.LLMInterpretation = result.Reasoning
	trace.DataSnapshot["llm_answer"] = result.Answer

	return result.Result, nil
}

// LLMConditionResult holds LLM condition output
type LLMConditionResult struct {
	Result    bool   `json:"result"`
	Answer    string `json:"answer"`
	Reasoning string `json:"reasoning"`
}

// LLMConditionActivity evaluates condition using LLM
func LLMConditionActivity(ctx context.Context, rule ConditionRule, state map[string]interface{}) (*LLMConditionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("LLM condition evaluation", "question", rule.LLMQuestion)

	// Build context from state
	contextParts := []string{}
	for key, path := range rule.LLMContext {
		value, err := resolveDataPath(path, state)
		if err == nil {
			valueJSON, _ := json.Marshal(value)
			contextParts = append(contextParts, fmt.Sprintf("%s: %s", key, string(valueJSON)))
		}
	}

	prompt := fmt.Sprintf(`You are a decision-making assistant. Answer the following yes/no question based on the provided context.

Context:
%s

Question: %s

Respond in JSON format:
{
  "answer": "yes" or "no",
  "reasoning": "Brief explanation of your answer"
}`,
		strings.Join(contextParts, "\n"),
		rule.LLMQuestion,
	)

	// Call LLM
	provider := llm.NewGeminiProvider("", "")
	response, err := provider.GenerateResponse(ctx, prompt)
	if err != nil {
		logger.Error("LLM condition call failed", "error", err)
		return &LLMConditionResult{
			Result:    false,
			Answer:    "error",
			Reasoning: err.Error(),
		}, nil
	}

	// Parse response
	var llmResult struct {
		Answer    string `json:"answer"`
		Reasoning string `json:"reasoning"`
	}

	cleanedResponse := extractJSONFromMarkdown(response)
	if err := json.Unmarshal([]byte(cleanedResponse), &llmResult); err != nil {
		// Try to extract yes/no from raw response
		lowerResp := strings.ToLower(response)
		if strings.Contains(lowerResp, "yes") {
			return &LLMConditionResult{Result: true, Answer: "yes", Reasoning: response}, nil
		}
		return &LLMConditionResult{Result: false, Answer: "no", Reasoning: response}, nil
	}

	result := strings.ToLower(llmResult.Answer) == "yes"

	return &LLMConditionResult{
		Result:    result,
		Answer:    llmResult.Answer,
		Reasoning: llmResult.Reasoning,
	}, nil
}

// ============================================================================
// Policy Condition Evaluation
// ============================================================================

func evaluatePolicyCondition(
	ctx workflow.Context,
	rule ConditionRule,
	state map[string]interface{},
	trace *ConditionTrace,
) (bool, error) {
	trace.Expression = rule.PolicyRef

	// Execute policy check via activity
	var result bool
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 10,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, PolicyConditionActivity,
		rule.PolicyRef, state).Get(ctx, &result)

	if err != nil {
		return false, fmt.Errorf("policy evaluation failed: %w", err)
	}

	trace.DataSnapshot["policy_result"] = result

	return result, nil
}

// PolicyConditionActivity evaluates policy rules
func PolicyConditionActivity(ctx context.Context, policyRef string, state map[string]interface{}) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Evaluating policy condition", "policy", policyRef)

	// TODO: Integrate with OPA or policy engine
	// For now, return true (pass)
	return true, nil
}

// ============================================================================
// Parser
// ============================================================================

// ParseConditionRule extracts condition rule from node config
func ParseConditionRule(config map[string]interface{}) (*ConditionRule, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var rule ConditionRule
	if err := json.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("failed to parse condition rule: %w", err)
	}

	if rule.Type == "" {
		rule.Type = ConditionBoolean // Default
	}

	return &rule, nil
}

// ============================================================================
// Helpers
// ============================================================================

// ContainsVariable checks if expression contains a variable reference
func ContainsVariable(expr string) bool {
	re := regexp.MustCompile(`\$\.[a-zA-Z_][a-zA-Z0-9_\.]*`)
	return re.MatchString(expr)
}

// ExtractVariables extracts all variable references from expression
func ExtractVariables(expr string) []string {
	re := regexp.MustCompile(`\$\.[a-zA-Z_][a-zA-Z0-9_\.]*`)
	return re.FindAllString(expr, -1)
}
