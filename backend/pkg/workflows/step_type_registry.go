package workflows

import (
	"encoding/json"
	"fmt"
)

// ============================================================================
// Step Type Registry - Workday-Inspired Business Process Step Types
// ============================================================================

// StepTypeCategory categorizes step types
type StepTypeCategory string

const (
	CategoryHuman      StepTypeCategory = "Human"
	CategorySystem     StepTypeCategory = "System"
	CategoryLLM        StepTypeCategory = "LLM"
	CategoryStructural StepTypeCategory = "Structural"
)

// AuditFlags controls what is captured for audit/compliance
type AuditFlags struct {
	CaptureInputSnapshot  bool `json:"capture_input_snapshot"`
	CaptureOutputSnapshot bool `json:"capture_output_snapshot"`
	CaptureLLMReasoning   bool `json:"capture_llm_reasoning"`
	CaptureRoutingTrace   bool `json:"capture_routing_trace"`
}

// StepTypeDefinition defines metadata for a step type
type StepTypeDefinition struct {
	Type              string           `json:"type"`
	Category          StepTypeCategory `json:"category"`
	Description       string           `json:"description"`
	RequiresRouting   bool             `json:"requires_routing"`
	SupportsLLM       bool             `json:"supports_llm"`
	DefaultAuditFlags AuditFlags       `json:"default_audit_flags"`
}

// StepTypeRegistry holds all registered step types
var StepTypeRegistry = map[string]StepTypeDefinition{
	// ==================== HUMAN STEPS ====================
	"Approval": {
		Type:            "Approval",
		Category:        CategoryHuman,
		Description:     "Formal decision point with routing, escalation, and outcome branches",
		RequiresRouting: true,
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureLLMReasoning:   true,
			CaptureRoutingTrace:   true,
		},
	},
	"Review": {
		Type:            "Review",
		Category:        CategoryHuman,
		Description:     "Review data and optionally edit before proceeding",
		RequiresRouting: true,
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureLLMReasoning:   true,
			CaptureRoutingTrace:   true,
		},
	},
	"ToDo": {
		Type:            "ToDo",
		Category:        CategoryHuman,
		Description:     "Lightweight manual task requiring acknowledgment",
		RequiresRouting: true,
		SupportsLLM:     false,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureRoutingTrace:   true,
		},
	},
	"Acknowledgment": {
		Type:            "Acknowledgment",
		Category:        CategoryHuman,
		Description:     "Explicit acknowledgment of disclosures or documents",
		RequiresRouting: true,
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureRoutingTrace:   true,
		},
	},

	// ==================== SYSTEM STEPS ====================
	"ServiceCall": {
		Type:            "ServiceCall",
		Category:        CategorySystem,
		Description:     "Call external or internal service (REST, gRPC, GraphQL)",
		RequiresRouting: false,
		SupportsLLM:     false,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
		},
	},
	"Calculation": {
		Type:            "Calculation",
		Category:        CategorySystem,
		Description:     "Invoke calc engine for projections, risk, tax, etc.",
		RequiresRouting: false,
		SupportsLLM:     false,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
		},
	},
	"SemanticRollup": {
		Type:            "SemanticRollup",
		Category:        CategorySystem,
		Description:     "Refresh semantic aggregates and materialized views",
		RequiresRouting: false,
		SupportsLLM:     false,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
		},
	},
	"DataValidation": {
		Type:            "DataValidation",
		Category:        CategorySystem,
		Description:     "Run validation rules; may produce human tasks on failure",
		RequiresRouting: false,
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureLLMReasoning:   true,
		},
	},
	"Notification": {
		Type:            "Notification",
		Category:        CategorySystem,
		Description:     "Send notifications to users or channels",
		RequiresRouting: true, // May need to resolve recipients
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
		},
	},

	// ==================== LLM-ENHANCED STEPS ====================
	"Interpretation": {
		Type:            "Interpretation",
		Category:        CategoryLLM,
		Description:     "Interpret unstructured input into structured semantic objects",
		RequiresRouting: false,
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureLLMReasoning:   true,
		},
	},
	"Classification": {
		Type:            "Classification",
		Category:        CategoryLLM,
		Description:     "Classify context (risk, intent, category, sentiment)",
		RequiresRouting: false,
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureLLMReasoning:   true,
		},
	},
	"Drafting": {
		Type:            "Drafting",
		Category:        CategoryLLM,
		Description:     "Draft client-facing or internal text for human review",
		RequiresRouting: true, // Needs reviewer assignment
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureLLMReasoning:   true,
			CaptureRoutingTrace:   true,
		},
	},
	"Recommendation": {
		Type:            "Recommendation",
		Category:        CategoryLLM,
		Description:     "Generate recommendations constrained by policy and data",
		RequiresRouting: true, // Needs approval routing
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureLLMReasoning:   true,
			CaptureRoutingTrace:   true,
		},
	},
	"ExceptionExplanation": {
		Type:            "ExceptionExplanation",
		Category:        CategoryLLM,
		Description:     "Explain exceptions, rejections, or policy conflicts in plain language",
		RequiresRouting: false,
		SupportsLLM:     true,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
			CaptureLLMReasoning:   true,
		},
	},

	// ==================== STRUCTURAL STEPS ====================
	"Subprocess": {
		Type:            "Subprocess",
		Category:        CategoryStructural,
		Description:     "Invoke another BP definition inline",
		RequiresRouting: false,
		SupportsLLM:     false,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
		},
	},
	"ParallelBlock": {
		Type:            "ParallelBlock",
		Category:        CategoryStructural,
		Description:     "Run multiple child steps or subprocesses in parallel",
		RequiresRouting: false,
		SupportsLLM:     false,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
		},
	},
	"Completion": {
		Type:            "Completion",
		Category:        CategoryStructural,
		Description:     "Finalize process, persist state, and emit events",
		RequiresRouting: false,
		SupportsLLM:     false,
		DefaultAuditFlags: AuditFlags{
			CaptureInputSnapshot:  true,
			CaptureOutputSnapshot: true,
		},
	},
}

// GetStepType returns the step type definition for a given type
func GetStepType(stepType string) (*StepTypeDefinition, error) {
	def, ok := StepTypeRegistry[stepType]
	if !ok {
		return nil, fmt.Errorf("unknown step type: %s", stepType)
	}
	return &def, nil
}

// GetStepsByCategory returns all step types for a category
func GetStepsByCategory(category StepTypeCategory) []StepTypeDefinition {
	var steps []StepTypeDefinition
	for _, def := range StepTypeRegistry {
		if def.Category == category {
			steps = append(steps, def)
		}
	}
	return steps
}

// IsHumanStep returns true if the step requires human action
func IsHumanStep(stepType string) bool {
	def, err := GetStepType(stepType)
	if err != nil {
		return false
	}
	return def.Category == CategoryHuman
}

// IsLLMStep returns true if the step is LLM-enhanced
func IsLLMStep(stepType string) bool {
	def, err := GetStepType(stepType)
	if err != nil {
		return false
	}
	return def.Category == CategoryLLM
}

// GetStepTypeRegistryJSON returns the entire registry as JSON (for API)
func GetStepTypeRegistryJSON() ([]byte, error) {
	return json.Marshal(StepTypeRegistry)
}

// ============================================================================
// LLM Profile Configuration
// ============================================================================

// LLMProfile defines configuration for LLM-enhanced steps
type LLMProfile struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	ModelName      string            `json:"model_name"` // e.g., "gemini-2.0-flash-exp"
	Temperature    float64           `json:"temperature"`
	MaxTokens      int               `json:"max_tokens"`
	SystemPrompt   string            `json:"system_prompt"`   // Base system context
	PromptTemplate string            `json:"prompt_template"` // Template with {{variables}}
	SafetyChecks   []string          `json:"safety_checks"`   // Policy constraints
	OutputFormat   string            `json:"output_format"`   // json, text, structured
	Constraints    map[string]string `json:"constraints"`     // Additional constraints
}

// DefaultLLMProfiles provides common profiles
var DefaultLLMProfiles = map[string]LLMProfile{
	"interpretation_default": {
		ID:          "interpretation_default",
		Name:        "Default Interpretation",
		Description: "Parse unstructured input into structured data",
		ModelName:   "gemini-2.0-flash-exp",
		Temperature: 0.1,
		MaxTokens:   4096,
		SystemPrompt: `You are a precise data extraction assistant. 
Extract structured information from the input and return valid JSON only.
Do not include any explanation or commentary.`,
		PromptTemplate: `Extract the following fields from the input:
{{fields_to_extract}}

Input:
{{input_text}}

Return as JSON:`,
		OutputFormat: "json",
	},
	"classification_default": {
		ID:          "classification_default",
		Name:        "Default Classification",
		Description: "Classify input into categories",
		ModelName:   "gemini-2.0-flash-exp",
		Temperature: 0.0,
		MaxTokens:   1024,
		SystemPrompt: `You are a classification assistant.
Classify the input into one of the provided categories.
Return only the category name, nothing else.`,
		PromptTemplate: `Categories:
{{categories}}

Input to classify:
{{input_text}}

Classification:`,
		OutputFormat: "text",
	},
	"drafting_default": {
		ID:          "drafting_default",
		Name:        "Default Drafting",
		Description: "Draft professional communications",
		ModelName:   "gemini-2.0-flash-exp",
		Temperature: 0.3,
		MaxTokens:   4096,
		SystemPrompt: `You are a professional communication assistant.
Draft clear, professional communications based on the provided context.
Match the tone and style appropriate for the target audience.`,
		PromptTemplate: `Audience: {{audience}}
Purpose: {{purpose}}
Context: {{context}}

Draft:`,
		OutputFormat: "text",
	},
	"recommendation_default": {
		ID:          "recommendation_default",
		Name:        "Default Recommendation",
		Description: "Generate constrained recommendations",
		ModelName:   "gemini-2.0-flash-exp",
		Temperature: 0.2,
		MaxTokens:   4096,
		SystemPrompt: `You are an advisory assistant that generates recommendations.
All recommendations must comply with the provided constraints and policies.
Be specific, actionable, and cite your reasoning.`,
		PromptTemplate: `Context:
{{context}}

Constraints:
{{constraints}}

Policies to comply with:
{{policies}}

Generate recommendation:`,
		OutputFormat: "json",
		SafetyChecks: []string{"policy_compliance", "risk_assessment"},
	},
	"explanation_default": {
		ID:          "explanation_default",
		Name:        "Default Explanation",
		Description: "Explain exceptions and decisions",
		ModelName:   "gemini-2.0-flash-exp",
		Temperature: 0.2,
		MaxTokens:   2048,
		SystemPrompt: `You are an explanation assistant.
Explain technical decisions, exceptions, or rejections in plain language.
Be clear, concise, and user-friendly.`,
		PromptTemplate: `Decision/Exception:
{{decision}}

Technical Details:
{{details}}

Explain this to the user in plain language:`,
		OutputFormat: "text",
	},
}

// GetLLMProfile returns an LLM profile by ID
func GetLLMProfile(profileID string) (*LLMProfile, error) {
	profile, ok := DefaultLLMProfiles[profileID]
	if !ok {
		return nil, fmt.Errorf("unknown LLM profile: %s", profileID)
	}
	return &profile, nil
}
