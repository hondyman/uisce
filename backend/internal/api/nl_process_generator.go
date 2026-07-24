package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

// NLProcessRequest represents the natural language input
type NLProcessRequest struct {
	Description  string `json:"description"`
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`
}

// NLProcessResponse wraps the generated process and insights
type NLProcessResponse struct {
	Process  BusinessProcess `json:"process"`
	Insights []string        `json:"insights"`
}

// AIProvider interface for different AI backends
type AIProvider interface {
	GenerateProcess(description string) (*NLProcessResponse, error)
}

// OpenAIProvider implements AIProvider for OpenAI
type OpenAIProvider struct {
	APIKey string
	Model  string
}

// ClaudeProvider implements AIProvider for Anthropic Claude
type ClaudeProvider struct {
	APIKey string
	Model  string
}

// GenerateProcessFromNaturalLanguage generates a business process from natural language description
func (h *BPBuilderHandlers) GenerateProcessFromNaturalLanguage(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, "tenant_id is required"))
		return
	}

	var req NLProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, "Invalid request body"))
		return
	}

	if req.Description == "" {
		respondJSON(w, http.StatusBadRequest, newBPAPIResponse(false, nil, "Description is required"))
		return
	}

	// Get AI provider (OpenAI, Claude, or custom)
	provider := h.getAIProvider()
	if provider == nil {
		// Fallback to rule-based generation if no AI provider configured
		response := h.generateProcessRuleBased(req.Description, tenantID, req.DatasourceID)
		respondJSON(w, http.StatusOK, newBPAPIResponse(true, response, ""))
		return
	}

	// Generate process using AI
	response, err := provider.GenerateProcess(req.Description)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, newBPAPIResponse(false, nil, fmt.Sprintf("AI generation failed: %v", err)))
		return
	}

	// Set tenant and datasource IDs
	response.Process.ID = uuid.New().String()
	response.Process.TenantID = tenantID
	response.Process.DatasourceID = req.DatasourceID
	response.Process.CreatedBy = "nl_builder_ai"
	response.Process.Version = 1

	respondJSON(w, http.StatusOK, newBPAPIResponse(true, response, ""))
}

// getAIProvider returns the configured AI provider
func (h *BPBuilderHandlers) getAIProvider() AIProvider {
	// Check for OpenAI API key
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		return &OpenAIProvider{
			APIKey: apiKey,
			Model:  getEnvOrDefault("OPENAI_MODEL", "gpt-4"),
		}
	}

	// Check for Claude API key
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		return &ClaudeProvider{
			APIKey: apiKey,
			Model:  getEnvOrDefault("CLAUDE_MODEL", "claude-3-5-sonnet-20241022"),
		}
	}

	return nil
}

// GenerateProcess implements AIProvider for OpenAI
func (p *OpenAIProvider) GenerateProcess(description string) (*NLProcessResponse, error) {
	systemPrompt := getSystemPrompt()
	userPrompt := fmt.Sprintf("Generate a business process from this description:\n\n%s", description)

	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.7,
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error: %d", resp.StatusCode)
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, err
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	return parseAIResponse(openaiResp.Choices[0].Message.Content)
}

// GenerateProcess implements AIProvider for Claude
func (p *ClaudeProvider) GenerateProcess(description string) (*NLProcessResponse, error) {
	systemPrompt := getSystemPrompt()
	userPrompt := fmt.Sprintf("Generate a business process from this description:\n\n%s", description)

	requestBody := map[string]interface{}{
		"model":      p.Model,
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Claude API error: %d", resp.StatusCode)
	}

	var claudeResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, err
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("no response from Claude")
	}

	return parseAIResponse(claudeResp.Content[0].Text)
}

// getSystemPrompt returns the AI system prompt for process generation
func getSystemPrompt() string {
	return `You are an expert business process designer. Generate a complete business process definition from natural language descriptions.

Your response MUST be valid JSON matching this exact structure:
{
  "process": {
    "processName": "string",
    "entity": "string (Employee|Order|Invoice|Project|Contract|Asset)",
    "description": "string",
    "steps": [
      {
        "id": "step-{number}",
        "stepOrder": number,
        "stepType": "data_entry|validate|approve|notify|integrate|condition",
        "stepName": "string",
        "durationHours": number,
        "assigneeRole": "string (optional)",
        "validationRules": ["string"] (optional),
        "executionMode": "sequential|parallel",
        "parallelGroup": "string (optional, for parallel steps)",
        "waitForAll": boolean (optional),
        "conditionLogic": {
          "operator": "AND|OR|NOT",
          "conditions": [
            {
              "field": "string",
              "operator": "==|!=|>|<|>=|<=|in|contains",
              "value": "any"
            }
          ],
          "trueBranch": ["step-id"],
          "falseBranch": ["step-id"]
        } (optional),
        "approvalChain": {
          "type": "role|multi_role|org_hierarchy",
          "roles": ["string"],
          "approvalMode": "all|any|majority",
          "escalationPath": ["string"] (optional)
        } (optional),
        "description": "string (optional)",
        "escalationThresholdHours": number (optional)
      }
    ],
    "isActive": false,
    "tags": ["string"]
  },
  "insights": ["string"]
}

Key instructions:
1. Extract the process name, entity type, and description from the input
2. Break down the workflow into logical steps
3. Identify approval requirements and create appropriate approval steps
4. Detect conditional logic (if/then) and create condition steps with proper conditionLogic
5. Identify parallel operations and group them with parallelGroup
6. Add realistic duration estimates
7. Include validation rules for data entry steps
8. Provide 3-5 helpful insights about the process design
9. Ensure stepOrder is sequential starting from 1
10. For amounts/thresholds mentioned, create proper condition operators

Example insights:
- "Process includes 2 approval levels for amounts over $1000"
- "Parallel execution of IT and HR tasks will reduce total time by 40%"
- "Escalation path ensures no approvals are blocked for more than 24 hours"

Return ONLY valid JSON, no markdown, no explanations outside the JSON structure.`
}

// parseAIResponse parses the AI response into structured data
func parseAIResponse(content string) (*NLProcessResponse, error) {
	// Remove markdown code blocks if present
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var response NLProcessResponse
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	// Set default values for missing fields
	for i := range response.Process.Steps {
		if response.Process.Steps[i].ExecutionMode == "" {
			response.Process.Steps[i].ExecutionMode = "sequential"
		}
	}

	return &response, nil
}

// generateProcessRuleBased generates a process using rule-based logic (fallback)
func (h *BPBuilderHandlers) generateProcessRuleBased(description string, tenantID, datasourceID string) *NLProcessResponse {
	lower := strings.ToLower(description)

	// Extract process name
	processName := "Custom Process"
	if strings.Contains(lower, "expense") {
		processName = "Expense Approval Process"
	} else if strings.Contains(lower, "employee") || strings.Contains(lower, "onboarding") {
		processName = "Employee Onboarding Process"
	} else if strings.Contains(lower, "purchase") || strings.Contains(lower, "order") {
		processName = "Purchase Order Process"
	}

	// Determine entity
	entity := "Order"
	if strings.Contains(lower, "employee") {
		entity = "Employee"
	} else if strings.Contains(lower, "invoice") {
		entity = "Invoice"
	}

	steps := []BPStep{
		{
			ID:            "step-1",
			StepOrder:     1,
			StepType:      "data_entry",
			StepName:      "Submit Request",
			DurationHours: 0.5,
			Description:   stringPtr("Initial data entry step"),
			ExecutionMode: "sequential",
		},
		{
			ID:              "step-2",
			StepOrder:       2,
			StepType:        "validate",
			StepName:        "Validate Information",
			DurationHours:   1,
			ValidationRules: []string{"Required fields present", "Data format valid"},
			ExecutionMode:   "sequential",
		},
	}

	// Add approval if mentioned
	if strings.Contains(lower, "approv") {
		steps = append(steps, BPStep{
			ID:            "step-3",
			StepOrder:     3,
			StepType:      "approve",
			StepName:      "Manager Approval",
			DurationHours: 4,
			AssigneeRole:  stringPtr("Manager"),
			ExecutionMode: "sequential",
		})
	}

	// Add notification if mentioned
	if strings.Contains(lower, "notif") || strings.Contains(lower, "email") {
		steps = append(steps, BPStep{
			ID:            fmt.Sprintf("step-%d", len(steps)+1),
			StepOrder:     len(steps) + 1,
			StepType:      "notify",
			StepName:      "Send Notification",
			DurationHours: 0.25,
			ExecutionMode: "sequential",
		})
	}

	return &NLProcessResponse{
		Process: BusinessProcess{
			ProcessName: processName,
			Entity:      entity,
			Description: description,
			Steps:       steps,
			IsActive:    false,
			Tags:        []string{"auto-generated"},
		},
		Insights: []string{
			"Process created using rule-based generation",
			"Consider adding more specific approval conditions",
			"Review step durations based on your organization",
		},
	}
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func stringPtr(s string) *string {
	return &s
}
