package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/pkg/llm"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Routing Engine - Workday-Inspired Dynamic Assignment Resolution
// ============================================================================

// RoutingRuleType defines the type of routing rule
type RoutingRuleType string

const (
	RoutingStatic     RoutingRuleType = "StaticGroup"
	RoutingDynamic    RoutingRuleType = "DynamicRole"
	RoutingExpression RoutingRuleType = "Expression"
	RoutingExternal   RoutingRuleType = "External"
	RoutingLLM        RoutingRuleType = "LLMAssisted"
)

// RoutingRule defines a routing specification
type RoutingRule struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Type RoutingRuleType `json:"type"`

	// StaticGroup routing
	StaticGroups []string `json:"static_groups,omitempty"`
	StaticUsers  []string `json:"static_users,omitempty"`

	// DynamicRole routing
	DynamicRole string `json:"dynamic_role,omitempty"` // e.g., "PrimaryAdvisor", "TeamLead"
	RolePath    string `json:"role_path,omitempty"`    // JSONPath to resolve: $.client.primary_advisor.id

	// Expression routing
	Expression     string `json:"expression,omitempty"`      // Expression to evaluate
	ExpressionType string `json:"expression_type,omitempty"` // jsonpath, graphql, sql

	// External routing
	ExternalSystem string `json:"external_system,omitempty"` // e.g. "Salesforce", "Jira"

	// LLM-assisted routing
	LLMProfile    string            `json:"llm_profile,omitempty"`
	LLMContext    map[string]string `json:"llm_context,omitempty"`    // Context for LLM
	CandidatePool []string          `json:"candidate_pool,omitempty"` // Users/groups for LLM to choose from

	// Escalation
	FallbackGroups []string      `json:"fallback_groups,omitempty"`
	EscalationRule *RoutingRule  `json:"escalation_rule,omitempty"`
	EscalationSLA  time.Duration `json:"escalation_sla,omitempty"`
}

// Assignee represents a resolved assignee
type Assignee struct {
	Type     string `json:"type"` // "user" or "group"
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	Priority int    `json:"priority"` // For ordering preferences
}

// RoutingTrace captures the full routing resolution for audit
type RoutingTrace struct {
	RuleID         string                 `json:"rule_id"`
	RuleName       string                 `json:"rule_name"`
	RuleType       string                 `json:"rule_type"`
	Timestamp      time.Time              `json:"timestamp"`
	Inputs         map[string]interface{} `json:"inputs"`
	ResolvedGroups []string               `json:"resolved_groups,omitempty"`
	ResolvedUsers  []Assignee             `json:"resolved_users"`
	LLMReasoning   string                 `json:"llm_reasoning,omitempty"`
	FallbackUsed   bool                   `json:"fallback_used"`
	Error          string                 `json:"error,omitempty"`
}

// RoutingResult holds the result of routing resolution
type RoutingResult struct {
	Assignees []Assignee    `json:"assignees"`
	Trace     *RoutingTrace `json:"trace"`
}

// ============================================================================
// Routing Resolution Functions
// ============================================================================

// ResolveRouting resolves assignees based on routing rule
func ResolveRouting(
	ctx workflow.Context,
	rule RoutingRule,
	currentState map[string]interface{},
) (*RoutingResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Resolving routing", "ruleID", rule.ID, "type", rule.Type)

	trace := &RoutingTrace{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		RuleType:  string(rule.Type),
		Timestamp: workflow.Now(ctx),
		Inputs:    make(map[string]interface{}),
	}

	var assignees []Assignee
	var err error

	switch rule.Type {
	case RoutingStatic:
		assignees, err = resolveStaticRouting(rule, trace)

	case RoutingDynamic:
		assignees, err = resolveDynamicRouting(ctx, rule, currentState, trace)

	case RoutingExpression:
		assignees, err = resolveExpressionRouting(ctx, rule, currentState, trace)

	case RoutingExternal:
		assignees, err = resolveExternalRouting(ctx, rule, currentState, trace)

	case RoutingLLM:
		assignees, err = resolveLLMRouting(ctx, rule, currentState, trace)

	default:
		err = fmt.Errorf("unknown routing type: %s", rule.Type)
	}

	if err != nil {
		logger.Warn("Routing resolution failed, trying fallback", "error", err)
		trace.Error = err.Error()

		// Try fallback
		if len(rule.FallbackGroups) > 0 {
			trace.FallbackUsed = true
			assignees = groupsToAssignees(rule.FallbackGroups)
			err = nil
		}
	}

	if len(assignees) == 0 && err == nil {
		err = fmt.Errorf("no assignees resolved")
	}

	trace.ResolvedUsers = assignees

	return &RoutingResult{
		Assignees: assignees,
		Trace:     trace,
	}, err
}

// ============================================================================
// Static Routing
// ============================================================================

func resolveStaticRouting(rule RoutingRule, trace *RoutingTrace) ([]Assignee, error) {
	var assignees []Assignee

	// Add static users
	for i, userID := range rule.StaticUsers {
		assignees = append(assignees, Assignee{
			Type:     "user",
			ID:       userID,
			Name:     userID, // Would be resolved from user service
			Priority: i,
		})
	}

	// Add static groups
	for _, groupID := range rule.StaticGroups {
		assignees = append(assignees, Assignee{
			Type:     "group",
			ID:       groupID,
			Name:     groupID,
			Priority: len(assignees),
		})
	}

	trace.ResolvedGroups = rule.StaticGroups

	return assignees, nil
}

// ============================================================================
// Dynamic Role Routing
// ============================================================================

func resolveDynamicRouting(
	ctx workflow.Context,
	rule RoutingRule,
	currentState map[string]interface{},
	trace *RoutingTrace,
) ([]Assignee, error) {
	trace.Inputs["dynamic_role"] = rule.DynamicRole
	trace.Inputs["role_path"] = rule.RolePath

	// Resolve the role path from state
	// e.g., $.client.primary_advisor.id resolves to user ID
	rolePath := rule.RolePath
	if rolePath == "" {
		// Build default path based on role name
		rolePath = fmt.Sprintf("$.%s", strings.ToLower(rule.DynamicRole))
	}

	value, err := resolveDataPath(rolePath, currentState)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dynamic role path %s: %w", rolePath, err)
	}

	// Value could be a single ID or array
	switch v := value.(type) {
	case string:
		return []Assignee{{
			Type:     "user",
			ID:       v,
			Name:     v,
			Priority: 0,
		}}, nil

	case []interface{}:
		var assignees []Assignee
		for i, item := range v {
			if id, ok := item.(string); ok {
				assignees = append(assignees, Assignee{
					Type:     "user",
					ID:       id,
					Name:     id,
					Priority: i,
				})
			}
		}
		return assignees, nil

	case map[string]interface{}:
		// Try to extract ID from object
		if id, ok := v["id"].(string); ok {
			return []Assignee{{
				Type:     "user",
				ID:       id,
				Name:     getStringVal(v, "name", id),
				Email:    getStringVal(v, "email", ""),
				Priority: 0,
			}}, nil
		}
	}

	return nil, fmt.Errorf("could not extract assignee from resolved value: %T", value)
}

// ============================================================================
// Expression-Based Routing
// ============================================================================

func resolveExpressionRouting(
	ctx workflow.Context,
	rule RoutingRule,
	currentState map[string]interface{},
	trace *RoutingTrace,
) ([]Assignee, error) {
	trace.Inputs["expression"] = rule.Expression
	trace.Inputs["expression_type"] = rule.ExpressionType

	// Execute expression via activity (non-deterministic)
	var result []Assignee
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 10,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, RoutingExpressionActivity,
		rule.Expression, rule.ExpressionType, currentState).Get(ctx, &result)

	if err != nil {
		return nil, fmt.Errorf("expression routing failed: %w", err)
	}

	return result, nil
}

// RoutingExpressionActivity evaluates routing expression
func RoutingExpressionActivity(ctx context.Context, expression string, exprType string, state map[string]interface{}) ([]Assignee, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Evaluating routing expression", "type", exprType)

	switch exprType {
	case "jsonpath", "":
		// Simple JSONPath evaluation
		value, err := resolveDataPath(expression, state)
		if err != nil {
			return nil, err
		}
		return valueToAssignees(value)

	case "graphql":
		// Would call GraphQL endpoint
		// For now, return empty
		return nil, fmt.Errorf("graphql routing not yet implemented")

	case "sql":
		// Would execute SQL query
		return nil, fmt.Errorf("sql routing not yet implemented")
	}

	return nil, fmt.Errorf("unknown expression type: %s", exprType)
}

// ============================================================================
// LLM-Assisted Routing
// ============================================================================

func resolveLLMRouting(
	ctx workflow.Context,
	rule RoutingRule,
	currentState map[string]interface{},
	trace *RoutingTrace,
) ([]Assignee, error) {
	trace.Inputs["llm_profile"] = rule.LLMProfile
	trace.Inputs["candidate_pool"] = rule.CandidatePool

	// Build candidates from pool
	candidates := rule.CandidatePool
	if len(candidates) == 0 {
		// Try to get candidates from fallback groups
		candidates = rule.FallbackGroups
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidate pool for LLM routing")
	}

	// Execute LLM routing via activity
	var result LLMRoutingResult
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 5,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, LLMRoutingActivity,
		rule, currentState, candidates).Get(ctx, &result)

	if err != nil {
		return nil, fmt.Errorf("LLM routing failed: %w", err)
	}

	trace.LLMReasoning = result.Reasoning

	return result.Assignees, nil
}

// LLMRoutingResult holds LLM routing output
type LLMRoutingResult struct {
	Assignees []Assignee `json:"assignees"`
	Reasoning string     `json:"reasoning"`
}

// LLMRoutingActivity uses LLM to select assignees
func LLMRoutingActivity(ctx context.Context, rule RoutingRule, state map[string]interface{}, candidates []string) (*LLMRoutingResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("LLM routing", "candidates", len(candidates))

	// Build context for LLM
	contextParts := []string{}
	for key, path := range rule.LLMContext {
		value, err := resolveDataPath(path, state)
		if err == nil {
			valueJSON, _ := json.Marshal(value)
			contextParts = append(contextParts, fmt.Sprintf("%s: %s", key, string(valueJSON)))
		}
	}

	prompt := fmt.Sprintf(`You are an intelligent task routing assistant.

Given the following context:
%s

And these available assignees:
%s

Select the most appropriate assignee(s) for this task.
Consider workload, expertise, and relevance.

Return your answer as JSON:
{
  "selected": ["assignee_id_1"],
  "reasoning": "Brief explanation"
}`,
		strings.Join(contextParts, "\n"),
		strings.Join(candidates, ", "),
	)

	// Call LLM
	provider := llm.NewGeminiProvider("", "")
	response, err := provider.GenerateResponse(ctx, prompt)
	if err != nil {
		logger.Error("LLM routing call failed", "error", err)
		// Fall back to first candidate
		return &LLMRoutingResult{
			Assignees: []Assignee{{Type: "user", ID: candidates[0], Name: candidates[0]}},
			Reasoning: "LLM call failed, using first candidate",
		}, nil
	}

	// Parse response
	var llmResult struct {
		Selected  []string `json:"selected"`
		Reasoning string   `json:"reasoning"`
	}

	cleanedResponse := extractJSONFromMarkdown(response)
	if err := json.Unmarshal([]byte(cleanedResponse), &llmResult); err != nil {
		logger.Warn("Failed to parse LLM routing response", "error", err)
		return &LLMRoutingResult{
			Assignees: []Assignee{{Type: "user", ID: candidates[0], Name: candidates[0]}},
			Reasoning: "Failed to parse LLM response, using first candidate",
		}, nil
	}

	// Convert to assignees
	var assignees []Assignee
	for i, id := range llmResult.Selected {
		assignees = append(assignees, Assignee{
			Type:     "user",
			ID:       id,
			Name:     id,
			Priority: i,
		})
	}

	return &LLMRoutingResult{
		Assignees: assignees,
		Reasoning: llmResult.Reasoning,
	}, nil
}

// ============================================================================
// External System Routing
// ============================================================================

func resolveExternalRouting(
	ctx workflow.Context,
	rule RoutingRule,
	currentState map[string]interface{},
	trace *RoutingTrace,
) ([]Assignee, error) {
	trace.Inputs["external_system"] = rule.ExternalSystem
	trace.Inputs["expression"] = rule.Expression

	// In a real implementation, we might call an activity to check system availability
	// or resolve a specific queue ID from the expression.
	// For now, we return a virtual assignee representing the system queue.

	queueID := rule.Expression // e.g. "salesforce.queue = 'OpsCases'" -> "OpsCases"
	if queueID == "" {
		queueID = "Default"
	}

	assignee := Assignee{
		Type:     "external_system",
		ID:       fmt.Sprintf("%s:%s", rule.ExternalSystem, queueID),
		Name:     fmt.Sprintf("%s Queue", rule.ExternalSystem),
		Priority: 0,
	}

	return []Assignee{assignee}, nil
}

// ============================================================================
// Escalation
// ============================================================================

// CheckEscalation determines if escalation is needed
func CheckEscalation(
	ctx workflow.Context,
	rule RoutingRule,
	stepStartTime time.Time,
) (bool, *RoutingRule) {
	if rule.EscalationRule == nil {
		return false, nil
	}

	elapsed := workflow.Now(ctx).Sub(stepStartTime)
	if elapsed > rule.EscalationSLA {
		return true, rule.EscalationRule
	}

	return false, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func groupsToAssignees(groups []string) []Assignee {
	var assignees []Assignee
	for i, g := range groups {
		assignees = append(assignees, Assignee{
			Type:     "group",
			ID:       g,
			Name:     g,
			Priority: i,
		})
	}
	return assignees
}

func valueToAssignees(value interface{}) ([]Assignee, error) {
	switch v := value.(type) {
	case string:
		return []Assignee{{Type: "user", ID: v, Name: v}}, nil
	case []interface{}:
		var result []Assignee
		for _, item := range v {
			if id, ok := item.(string); ok {
				result = append(result, Assignee{Type: "user", ID: id, Name: id})
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("cannot convert %T to assignees", value)
}

func getStringVal(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}

// ParseRoutingRule extracts routing rule from node config
func ParseRoutingRule(config map[string]interface{}) (*RoutingRule, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var rule RoutingRule
	if err := json.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("failed to parse routing rule: %w", err)
	}

	if rule.Type == "" {
		rule.Type = RoutingStatic // Default
	}

	return &rule, nil
}
