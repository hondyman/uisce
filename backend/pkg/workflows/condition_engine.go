package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/pkg/llm"

	"go.temporal.io/sdk/activity"
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

// ============================================================================
// Boolean Condition Evaluation
// ============================================================================

// ============================================================================
// Semantic Condition Evaluation
// ============================================================================

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

// ============================================================================
// Helpers
// ============================================================================
