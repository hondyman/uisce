package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hondyman/semlayer/backend/pkg/llm"
)

// GenerateDAGRequest is the input for the AI Logic-to-Config endpoint
type GenerateDAGRequest struct {
	Prompt string `json:"prompt"` // Natural language description
	// Optional: existing DAG to modify
	ExistingDefinition json.RawMessage `json:"existingDefinition,omitempty"`
}

// GenerateDAGResponse is the output containing the generated DAG
type GenerateDAGResponse struct {
	DAGDefinition json.RawMessage `json:"dagDefinition"`
	Explanation   string          `json:"explanation"`
}

// GenerateDAGFromNL handles POST /ai/generate-dag
// It uses GenAI to translate natural language into a Titan DAG JSON configuration.
func (s *Server) GenerateDAGFromNL(w http.ResponseWriter, r *http.Request) {
	var req GenerateDAGRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Prompt) == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// 1. Load LLM Config
	cfg, err := s.LLMConfigSvc.Get()
	if err != nil {
		http.Error(w, "LLM configuration not available", http.StatusServiceUnavailable)
		return
	}

	// 2. Build System Prompt with Schema Definition
	systemPrompt := buildDAGGenerationSystemPrompt()

	// 3. Construct User Prompt
	userPrompt := fmt.Sprintf(`User Request: %s

Generate a valid Titan DAG JSON configuration for this workflow.
If modifying an existing definition, here it is: %s`, req.Prompt, string(req.ExistingDefinition))

	fullPrompt := fmt.Sprintf("System:\n%s\n\nUser:\n%s", systemPrompt, userPrompt)

	// 4. Call LLM
	provider := llm.NewGeminiProvider(cfg.APIKey, cfg.Model)
	response, err := provider.GenerateResponse(r.Context(), fullPrompt)
	if err != nil {
		http.Error(w, "AI generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Parse and Validate Response
	dagJSON, explanation, err := parseDAGResponse(response)
	if err != nil {
		// Return raw response for debugging if parsing fails
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":       "Failed to parse AI response",
			"rawResponse": response,
		})
		return
	}

	// 6. Return Result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GenerateDAGResponse{
		DAGDefinition: dagJSON,
		Explanation:   explanation,
	})
}

// buildDAGGenerationSystemPrompt returns the schema and instructions for the LLM
func buildDAGGenerationSystemPrompt() string {
	return `You are an expert workflow configuration assistant for the Titan platform.
Your task is to generate valid JSON workflow definitions based on user descriptions.

## Titan DAG Schema

A workflow definition has this structure:
{
  "nodes": {
    "node_id": {
      "id": "node_id",
      "type": "ACTIVITY" | "BRANCH" | "USER_INTERACTION",
      "config": { ... activity-specific config ... },
      "next": "next_node_id" | null
    }
  },
  "startNodeId": "first_node_id"
}

## Available Activity Types

1. **SendEmail** - Send email notification
   Config: { "to": "{{state.recipient}}", "subject": "...", "body": "..." }

2. **CallWebhook** - Make HTTP request
   Config: { "url": "...", "method": "POST", "body": "..." }

3. **ActivityCheckCompliance** - Pre-trade compliance check
   Config: { "tradeType": "equity", "jurisdiction": "US" }

4. **ActivityGenerateContent** - GenAI content generation
   Config: { "promptTemplate": "...", "systemInstruction": "...", "outputKey": "ai_result" }

5. **ActivityPredictSettlementRisk** - Risk scoring
   Config: {}

6. **ActivityUserInteraction** - Human approval/input
   Config: { "viewDefinitionName": "high_value_order_approval_form", "title": "Approval Required" }

7. **DurableLedgerWrite** - Immutable audit log
   Config: { "eventType": "TRADE_EXECUTED" }

## BRANCH Node

For conditional logic:
{
  "id": "decision_node",
  "type": "BRANCH",
  "config": {
    "conditionField": "state.amount",
    "operator": "gt",
    "value": 100000,
    "trueNext": "approval_node",
    "falseNext": "auto_execute_node"
  }
}

## Response Format

Return ONLY valid JSON in this format:
{
  "dag": { ... the workflow definition ... },
  "explanation": "Brief explanation of the generated workflow"
}

Do not include markdown code fences or any other text outside the JSON object.`
}

// parseDAGResponse extracts the DAG JSON and explanation from the LLM response
func parseDAGResponse(response string) (json.RawMessage, string, error) {
	// Try to parse the entire response as JSON first
	var parsed struct {
		DAG         json.RawMessage `json:"dag"`
		Explanation string          `json:"explanation"`
	}

	// Clean up common LLM quirks (markdown code blocks)
	cleaned := strings.TrimSpace(response)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, "", fmt.Errorf("failed to parse response: %w", err)
	}

	return parsed.DAG, parsed.Explanation, nil
}
