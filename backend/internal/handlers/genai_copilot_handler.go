package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
)

// GenAICopilotHandler handles AI-powered co-pilot features
type GenAICopilotHandler struct {
	db             *sqlx.DB
	temporalClient client.Client
	llmConfig      *llm.LLMConfigService
}

// NewGenAICopilotHandler creates a new handler
func NewGenAICopilotHandler(db *sqlx.DB, temporalClient client.Client, llmCfg *llm.LLMConfigService) *GenAICopilotHandler {
	return &GenAICopilotHandler{
		db:             db,
		temporalClient: temporalClient,
		llmConfig:      llmCfg,
	}
}

// RegisterRoutes registers the co-pilot routes
func (h *GenAICopilotHandler) RegisterRoutes(r chi.Router) {
	r.Route("/copilot", func(r chi.Router) {
		r.Post("/generate-node", h.GenerateNodeFromText)
		r.Post("/generate-rego", h.GeneratePolicyFromText)
		r.Post("/generate-audit-summary", h.GenerateAuditSummary)
	})
}

// GenerateNodeRequest is the input for NL-to-Workflow generation
type GenerateNodeRequest struct {
	Text            string   `json:"text"`                      // Natural language description
	BusinessObjects []string `json:"businessObjects,omitempty"` // Relevant BO names for context
	ExistingNodeIDs []string `json:"existingNodeIds,omitempty"` // IDs of nodes already in workflow
}

// GenerateNodeResponse contains the generated workflow node(s)
type GenerateNodeResponse struct {
	NodeJSON    json.RawMessage `json:"nodeJson"`
	Explanation string          `json:"explanation"`
}

// GenerateNodeFromText handles POST /api/v1/copilot/generate-node
// Translates natural language into a Titan workflow node configuration
func (h *GenAICopilotHandler) GenerateNodeFromText(w http.ResponseWriter, r *http.Request) {
	var req GenerateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	// 1. Fetch schema context for business objects
	dataContextSchema := h.fetchBusinessObjectSchemas(r.Context(), req.BusinessObjects)

	// 2. Build the RAG-enhanced prompt
	prompt := buildNodeGenerationPrompt(req.Text, dataContextSchema, req.ExistingNodeIDs)

	// 3. Call LLM directly (or via Temporal for durability)
	cfg, err := h.llmConfig.Get()
	if err != nil {
		http.Error(w, "LLM configuration unavailable", http.StatusServiceUnavailable)
		return
	}

	provider := llm.NewGeminiProvider(cfg.APIKey, cfg.Model)
	response, err := provider.GenerateResponse(r.Context(), prompt)
	if err != nil {
		http.Error(w, "AI generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Parse and return
	nodeJSON, explanation := parseNodeGenerationResponse(response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GenerateNodeResponse{
		NodeJSON:    nodeJSON,
		Explanation: explanation,
	})
}

// GeneratePolicyRequest is the input for NL-to-Rego generation
type GeneratePolicyRequest struct {
	Text            string   `json:"text"`                      // Natural language business rule
	PolicyName      string   `json:"policyName,omitempty"`      // Name for the policy package
	BusinessObjects []string `json:"businessObjects,omitempty"` // Relevant BO names
}

// GeneratePolicyResponse contains the generated Rego policy
type GeneratePolicyResponse struct {
	RegoCode    string `json:"regoCode"`
	PolicyName  string `json:"policyName"`
	Explanation string `json:"explanation"`
}

// GeneratePolicyFromText handles POST /api/v1/copilot/generate-rego
// Translates natural language business rules into OPA Rego policies
func (h *GenAICopilotHandler) GeneratePolicyFromText(w http.ResponseWriter, r *http.Request) {
	var req GeneratePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	if req.PolicyName == "" {
		req.PolicyName = fmt.Sprintf("policy_%s", uuid.New().String()[:8])
	}

	// 1. Fetch schema context
	dataContextSchema := h.fetchBusinessObjectSchemas(r.Context(), req.BusinessObjects)

	// 2. Build the Rego generation prompt
	prompt := buildRegoGenerationPrompt(req.Text, req.PolicyName, dataContextSchema)

	// 3. Call LLM
	cfg, err := h.llmConfig.Get()
	if err != nil {
		http.Error(w, "LLM configuration unavailable", http.StatusServiceUnavailable)
		return
	}

	provider := llm.NewGeminiProvider(cfg.APIKey, cfg.Model)
	response, err := provider.GenerateResponse(r.Context(), prompt)
	if err != nil {
		http.Error(w, "AI generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Parse and return
	regoCode, explanation := parseRegoGenerationResponse(response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GeneratePolicyResponse{
		RegoCode:    regoCode,
		PolicyName:  req.PolicyName,
		Explanation: explanation,
	})
}

// GenerateAuditSummaryRequest is the input for generating audit documentation
type GenerateAuditSummaryRequest struct {
	WorkflowState map[string]interface{} `json:"workflowState"` // Final state of the workflow
	WorkflowName  string                 `json:"workflowName"`
	StartTime     time.Time              `json:"startTime"`
	EndTime       time.Time              `json:"endTime"`
}

// GenerateAuditSummaryResponse contains the AI-generated audit summary
type GenerateAuditSummaryResponse struct {
	Summary    string   `json:"summary"`
	Highlights []string `json:"highlights,omitempty"`
}

// GenerateAuditSummary handles POST /api/v1/copilot/generate-audit-summary
// Creates human-readable audit documentation from workflow state
func (h *GenAICopilotHandler) GenerateAuditSummary(w http.ResponseWriter, r *http.Request) {
	var req GenerateAuditSummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Build the audit prompt
	stateJSON, _ := json.MarshalIndent(req.WorkflowState, "", "  ")
	duration := req.EndTime.Sub(req.StartTime)

	prompt := fmt.Sprintf(`You are an auditor. Based on the following workflow execution, write a concise, professional summary suitable for compliance reporting.

### Workflow: %s
### Duration: %s

### Final State:
%s

### Instructions:
1. Write a one-paragraph summary of what occurred
2. Highlight any exceptions or notable events
3. Note the final outcome and any approvals/rejections

### Response Format (JSON):
{
  "summary": "...",
  "highlights": ["..."]
}`, req.WorkflowName, duration.String(), string(stateJSON))

	// Call LLM
	cfg, err := h.llmConfig.Get()
	if err != nil {
		http.Error(w, "LLM configuration unavailable", http.StatusServiceUnavailable)
		return
	}

	provider := llm.NewGeminiProvider(cfg.APIKey, cfg.Model)
	response, err := provider.GenerateResponse(r.Context(), prompt)
	if err != nil {
		http.Error(w, "AI generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse response
	var result GenerateAuditSummaryResponse
	if err := json.Unmarshal([]byte(cleanJSONResponse(response)), &result); err != nil {
		// Fallback: use raw response as summary
		result.Summary = response
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Helper functions

func (h *GenAICopilotHandler) fetchBusinessObjectSchemas(ctx context.Context, boNames []string) string {
	if len(boNames) == 0 {
		return "No specific business objects specified. Use generic property paths like $.context.<property>"
	}

	// Query business object definitions from DB
	schemas := make(map[string]interface{})
	for _, name := range boNames {
		var config json.RawMessage
		err := h.db.GetContext(ctx, &config,
			"SELECT config FROM business_objects WHERE key = $1 OR name = $1 LIMIT 1", name)
		if err == nil {
			schemas[name] = json.RawMessage(config)
		}
	}

	if len(schemas) == 0 {
		return "Business objects not found. Use generic paths."
	}

	schemaJSON, _ := json.MarshalIndent(schemas, "", "  ")
	return string(schemaJSON)
}

func buildNodeGenerationPrompt(userText, contextSchema string, existingNodes []string) string {
	return fmt.Sprintf(`You are a Titan Platform workflow configuration expert.
Given a user's request, generate a valid JSON configuration for a workflow node.

### User Request
%s

### Data Context Schema
The workflow has access to the following data objects:
%s

### Existing Nodes in Workflow
%v

### Available Node Types

1. **BranchNode** - Conditional routing
{ "id": "unique_id", "type": "BRANCH", "config": { "conditionField": "$.context.<field>", "operator": "gt|lt|eq|ne|contains", "value": <any>, "trueNext": "node_id", "falseNext": "node_id" } }

2. **ActivityNode** - Execute an action
{ "id": "unique_id", "type": "ACTIVITY", "config": { "activityName": "SendEmail|CallWebhook|ActivityUserInteraction|ActivityCheckCompliance" }, "next": "node_id" }

3. **UserInteractionNode** - Human approval/input
{ "id": "unique_id", "type": "ACTIVITY", "config": { "activityName": "ActivityUserInteraction", "viewDefinitionName": "form_name", "title": "Approval Title" }, "next": "node_id" }

### Examples
- User: "If order total is > 1000" -> Condition: "$.context.order.total" operator: "gt" value: 1000
- User: "If customer is from the USA" -> Condition: "$.context.customer.country" operator: "eq" value: "USA"
- User: "Send for manager approval" -> ActivityUserInteraction with appropriate form

### Response Format (JSON only):
{
  "node": { ... the generated node configuration ... },
  "explanation": "Brief explanation of what this node does"
}

### GENERATE:`, userText, contextSchema, existingNodes)
}

func buildRegoGenerationPrompt(userText, policyName, contextSchema string) string {
	return fmt.Sprintf(`You are an Open Policy Agent (OPA) expert.
Given a user's business rule, write a valid Rego policy.
The policy should return 'allow = false' if the rule is violated.

### Business Rule
%s

### Policy Name
%s

### Data Input Schema (passed to OPA)
%s

### Rego Structure Template
package titan.compliance.%s

import rego.v1

default allow := true

# Rule: <description>
allow := false if {
    # condition that causes violation
}

# Additional helper rules as needed

### Example
- Rule: "Block trades over $1M"
  Rego:
    package titan.compliance.trade_limits
    import rego.v1
    default allow := true
    allow := false if { input.trade.value > 1000000 }

### Response Format (JSON):
{
  "rego": "package titan.compliance...",
  "explanation": "Brief explanation of the policy"
}

### GENERATE:`, userText, policyName, contextSchema, policyName)
}

func parseNodeGenerationResponse(response string) (json.RawMessage, string) {
	cleaned := cleanJSONResponse(response)

	var parsed struct {
		Node        json.RawMessage `json:"node"`
		Explanation string          `json:"explanation"`
	}

	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		// Return raw response
		return json.RawMessage(`{"error": "parsing_failed"}`), response
	}

	return parsed.Node, parsed.Explanation
}

func parseRegoGenerationResponse(response string) (string, string) {
	cleaned := cleanJSONResponse(response)

	var parsed struct {
		Rego        string `json:"rego"`
		Explanation string `json:"explanation"`
	}

	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		// Extract rego from response if JSON parsing fails
		return response, ""
	}

	return parsed.Rego, parsed.Explanation
}

func cleanJSONResponse(response string) string {
	// Remove markdown code fences
	cleaned := response
	for _, prefix := range []string{"```json", "```"} {
		if len(cleaned) > len(prefix) && cleaned[:len(prefix)] == prefix {
			cleaned = cleaned[len(prefix):]
			break
		}
	}
	for _, suffix := range []string{"```"} {
		if len(cleaned) > len(suffix) && cleaned[len(cleaned)-len(suffix):] == suffix {
			cleaned = cleaned[:len(cleaned)-len(suffix)]
			break
		}
	}

	// Trim whitespace
	for len(cleaned) > 0 && (cleaned[0] == ' ' || cleaned[0] == '\n' || cleaned[0] == '\t') {
		cleaned = cleaned[1:]
	}
	for len(cleaned) > 0 && (cleaned[len(cleaned)-1] == ' ' || cleaned[len(cleaned)-1] == '\n' || cleaned[len(cleaned)-1] == '\t') {
		cleaned = cleaned[:len(cleaned)-1]
	}

	return cleaned
}
