package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hondyman/semlayer/backend/pkg/llm"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// LLM Activities - Workday-Inspired LLM-Enhanced Steps
// ============================================================================

// LLMStepConfig defines common configuration for LLM steps
type LLMStepConfig struct {
	ProfileID      string            `json:"profile_id"`      // LLM profile reference
	PromptTemplate string            `json:"prompt_template"` // Override profile template
	InputContext   map[string]string `json:"input_context"`   // State → template variable mapping
	OutputVariable string            `json:"output_variable"` // Where to store result
	SafetyChecks   []string          `json:"safety_checks"`   // Policy constraints
	OutputFormat   string            `json:"output_format"`   // json, text, structured
	MaxRetries     int               `json:"max_retries"`     // Retry count on failure
}

// LLMStepResult holds the result of an LLM step
type LLMStepResult struct {
	Success          bool               `json:"success"`
	Output           interface{}        `json:"output"`
	RawOutput        string             `json:"raw_output"`
	ReasoningTrace   *LLMReasoningTrace `json:"reasoning_trace"`
	SafetyViolations []string           `json:"safety_violations"`
	Error            string             `json:"error,omitempty"`
}

// LLMReasoningTrace captures the full LLM interaction for audit
type LLMReasoningTrace struct {
	ProfileID       string                 `json:"profile_id"`
	PromptTemplate  string                 `json:"prompt_template"`
	FilledPrompt    string                 `json:"filled_prompt"`
	InputSnapshot   map[string]interface{} `json:"input_snapshot"`
	RawOutput       string                 `json:"raw_output"`
	ProcessedOutput interface{}            `json:"processed_output"`
	SafetyChecks    []SafetyCheckResult    `json:"safety_checks"`
	ModelName       string                 `json:"model_name"`
	Timestamp       int64                  `json:"timestamp"`
}

// SafetyCheckResult holds result of a safety check
type SafetyCheckResult struct {
	CheckName string `json:"check_name"`
	Passed    bool   `json:"passed"`
	Message   string `json:"message,omitempty"`
}

// ============================================================================
// LLM Step Execution Functions
// ============================================================================

// ExecuteInterpretationStep interprets unstructured input into structured data
func ExecuteInterpretationStep(
	ctx workflow.Context,
	config LLMStepConfig,
	currentState map[string]interface{},
) (*LLMStepResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing Interpretation step", "profile", config.ProfileID)

	// Execute as activity (LLM calls must be in activities)
	var result LLMStepResult
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 5,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, LLMInterpretationActivity,
		config, currentState).Get(ctx, &result)

	if err != nil {
		logger.Error("Interpretation step failed", "error", err)
		return &LLMStepResult{Success: false, Error: err.Error()}, nil
	}

	return &result, nil
}

// ExecuteClassificationStep classifies input into categories
func ExecuteClassificationStep(
	ctx workflow.Context,
	config LLMStepConfig,
	currentState map[string]interface{},
) (*LLMStepResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing Classification step", "profile", config.ProfileID)

	var result LLMStepResult
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 5,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, LLMClassificationActivity,
		config, currentState).Get(ctx, &result)

	if err != nil {
		return &LLMStepResult{Success: false, Error: err.Error()}, nil
	}

	return &result, nil
}

// ExecuteDraftingStep drafts text for human review
func ExecuteDraftingStep(
	ctx workflow.Context,
	config LLMStepConfig,
	currentState map[string]interface{},
) (*LLMStepResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing Drafting step", "profile", config.ProfileID)

	var result LLMStepResult
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 5,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, LLMDraftingActivity,
		config, currentState).Get(ctx, &result)

	if err != nil {
		return &LLMStepResult{Success: false, Error: err.Error()}, nil
	}

	return &result, nil
}

// ExecuteRecommendationStep generates constrained recommendations
func ExecuteRecommendationStep(
	ctx workflow.Context,
	config LLMStepConfig,
	currentState map[string]interface{},
) (*LLMStepResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing Recommendation step", "profile", config.ProfileID)

	var result LLMStepResult
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 5,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, LLMRecommendationActivity,
		config, currentState).Get(ctx, &result)

	if err != nil {
		return &LLMStepResult{Success: false, Error: err.Error()}, nil
	}

	return &result, nil
}

// ExecuteExplanationStep explains exceptions/decisions in plain language
func ExecuteExplanationStep(
	ctx workflow.Context,
	config LLMStepConfig,
	currentState map[string]interface{},
) (*LLMStepResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing Explanation step", "profile", config.ProfileID)

	var result LLMStepResult
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 5,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, LLMExplanationActivity,
		config, currentState).Get(ctx, &result)

	if err != nil {
		return &LLMStepResult{Success: false, Error: err.Error()}, nil
	}

	return &result, nil
}

// ============================================================================
// Activity Implementations (These run outside the workflow determinism)
// ============================================================================

// LLMInterpretationActivity is the activity for interpretation
func LLMInterpretationActivity(ctx context.Context, config LLMStepConfig, state map[string]interface{}) (*LLMStepResult, error) {
	return executeLLMActivity(ctx, "interpretation", config, state)
}

// LLMClassificationActivity is the activity for classification
func LLMClassificationActivity(ctx context.Context, config LLMStepConfig, state map[string]interface{}) (*LLMStepResult, error) {
	return executeLLMActivity(ctx, "classification", config, state)
}

// LLMDraftingActivity is the activity for drafting
func LLMDraftingActivity(ctx context.Context, config LLMStepConfig, state map[string]interface{}) (*LLMStepResult, error) {
	return executeLLMActivity(ctx, "drafting", config, state)
}

// LLMRecommendationActivity is the activity for recommendations
func LLMRecommendationActivity(ctx context.Context, config LLMStepConfig, state map[string]interface{}) (*LLMStepResult, error) {
	return executeLLMActivity(ctx, "recommendation", config, state)
}

// LLMExplanationActivity is the activity for explanations
func LLMExplanationActivity(ctx context.Context, config LLMStepConfig, state map[string]interface{}) (*LLMStepResult, error) {
	return executeLLMActivity(ctx, "explanation", config, state)
}

// executeLLMActivity is the core LLM execution logic
func executeLLMActivity(ctx context.Context, stepType string, config LLMStepConfig, state map[string]interface{}) (*LLMStepResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing LLM activity", "stepType", stepType, "profile", config.ProfileID)

	// Get profile
	profileID := config.ProfileID
	if profileID == "" {
		profileID = stepType + "_default"
	}
	profile, err := GetLLMProfile(profileID)
	if err != nil {
		// Use defaults
		profile = &LLMProfile{
			ModelName:   "gemini-2.0-flash-exp",
			Temperature: 0.2,
			MaxTokens:   4096,
		}
	}

	// Fill prompt template
	promptTemplate := config.PromptTemplate
	if promptTemplate == "" {
		promptTemplate = profile.PromptTemplate
	}

	filledPrompt := fillPromptTemplate(promptTemplate, config.InputContext, state)

	// Add system prompt
	fullPrompt := filledPrompt
	if profile.SystemPrompt != "" {
		fullPrompt = profile.SystemPrompt + "\n\n" + filledPrompt
	}

	// Create input snapshot for audit
	inputSnapshot := make(map[string]interface{})
	for key, path := range config.InputContext {
		value, _ := resolveDataPath(path, state)
		inputSnapshot[key] = value
	}

	// Initialize LLM provider
	provider := llm.NewGeminiProvider("", profile.ModelName)

	// Call LLM
	rawOutput, err := provider.GenerateResponse(ctx, fullPrompt)
	if err != nil {
		logger.Error("LLM call failed", "error", err)
		return &LLMStepResult{
			Success: false,
			Error:   err.Error(),
			ReasoningTrace: &LLMReasoningTrace{
				ProfileID:      profileID,
				PromptTemplate: promptTemplate,
				FilledPrompt:   filledPrompt,
				InputSnapshot:  inputSnapshot,
				ModelName:      profile.ModelName,
			},
		}, nil
	}

	// Process output based on format
	var processedOutput interface{}
	if config.OutputFormat == "json" || profile.OutputFormat == "json" {
		// Try to parse as JSON
		var jsonOutput map[string]interface{}
		if err := json.Unmarshal([]byte(rawOutput), &jsonOutput); err != nil {
			// Try to extract JSON from markdown code blocks
			cleanedOutput := extractJSONFromMarkdown(rawOutput)
			if err := json.Unmarshal([]byte(cleanedOutput), &jsonOutput); err != nil {
				processedOutput = rawOutput // Fall back to raw
			} else {
				processedOutput = jsonOutput
			}
		} else {
			processedOutput = jsonOutput
		}
	} else {
		processedOutput = rawOutput
	}

	// Run safety checks
	safetyResults := runSafetyChecks(config.SafetyChecks, rawOutput, state)
	safetyViolations := []string{}
	for _, check := range safetyResults {
		if !check.Passed {
			safetyViolations = append(safetyViolations, check.Message)
		}
	}

	// Build reasoning trace
	trace := &LLMReasoningTrace{
		ProfileID:       profileID,
		PromptTemplate:  promptTemplate,
		FilledPrompt:    filledPrompt,
		InputSnapshot:   inputSnapshot,
		RawOutput:       rawOutput,
		ProcessedOutput: processedOutput,
		SafetyChecks:    safetyResults,
		ModelName:       profile.ModelName,
	}

	return &LLMStepResult{
		Success:          len(safetyViolations) == 0,
		Output:           processedOutput,
		RawOutput:        rawOutput,
		ReasoningTrace:   trace,
		SafetyViolations: safetyViolations,
	}, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// fillPromptTemplate replaces {{variables}} with actual values
func fillPromptTemplate(template string, inputContext map[string]string, state map[string]interface{}) string {
	result := template

	// Replace from inputContext mapping
	for varName, statePath := range inputContext {
		value, err := resolveDataPath(statePath, state)
		if err == nil {
			// Convert value to string
			var strValue string
			switch v := value.(type) {
			case string:
				strValue = v
			default:
				jsonBytes, _ := json.Marshal(v)
				strValue = string(jsonBytes)
			}
			result = strings.ReplaceAll(result, "{{"+varName+"}}", strValue)
		}
	}

	// Replace any remaining {{variable}} with empty
	re := regexp.MustCompile(`\{\{[^}]+\}\}`)
	result = re.ReplaceAllString(result, "")

	return result
}

// extractJSONFromMarkdown extracts JSON from markdown code blocks
func extractJSONFromMarkdown(text string) string {
	// Try to find JSON in ```json ... ``` blocks
	re := regexp.MustCompile("(?s)```(?:json)?\\s*\\n(.+?)\\n```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return text
}

// runSafetyChecks runs configured safety checks on LLM output
func runSafetyChecks(checks []string, output string, state map[string]interface{}) []SafetyCheckResult {
	results := []SafetyCheckResult{}

	for _, check := range checks {
		result := SafetyCheckResult{
			CheckName: check,
			Passed:    true,
		}

		switch check {
		case "no_pii":
			// Basic PII detection (simplified)
			if containsPII(output) {
				result.Passed = false
				result.Message = "Output may contain PII"
			}

		case "no_financial_advice":
			// Check for financial advice indicators
			if containsFinancialAdvice(output) {
				result.Passed = false
				result.Message = "Output may contain unauthorized financial advice"
			}

		case "policy_compliance":
			// Placeholder for policy compliance check
			result.Passed = true

		case "risk_assessment":
			// Placeholder for risk assessment
			result.Passed = true
		}

		results = append(results, result)
	}

	return results
}

// containsPII checks for potential PII in output
func containsPII(text string) bool {
	lower := strings.ToLower(text)
	// Very basic checks - should be much more sophisticated in production
	piiPatterns := []string{
		"ssn:", "social security",
		"credit card", "card number",
		"password:", "secret:",
	}
	for _, pattern := range piiPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// containsFinancialAdvice checks for financial advice indicators
func containsFinancialAdvice(text string) bool {
	lower := strings.ToLower(text)
	advicePatterns := []string{
		"you should invest",
		"i recommend buying",
		"guaranteed return",
		"risk-free investment",
	}
	for _, pattern := range advicePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// ParseLLMStepConfig extracts config from node
func ParseLLMStepConfig(config map[string]interface{}) (*LLMStepConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg LLMStepConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse LLM step config: %w", err)
	}

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return &cfg, nil
}
