// services/ai-builder/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"

	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
)

type WorkflowSuggestion struct {
	Description string                 `json:"description"`
	Elements    []WorkflowElement      `json:"elements"`
	YAML        string                 `json:"yaml"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type WorkflowElement struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position map[string]float64     `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "worker" {
		runWorker()
		return
	}

	// Default: run the API server
	runServer()
}

func runServer() {
	r := gin.Default()
	// Use centralized temporal helper to get a client (with retries configured via env)
	tc, err := temporalclient.NewClientWithRetry()
	if err != nil {
		log.Printf("Warning: could not connect to Temporal; starting server without Temporal client: %v", err)
		tc = nil
	} else {
		defer tc.Close()
	}

	// Note: ABAC engine would be initialized here in production
	// abacEngine := abac.NewEngine()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "ai-builder"})
	})

	// AI workflow suggestion endpoint
	r.POST("/workflows/suggest", func(c *gin.Context) {
		// TODO: Add ABAC permission check when ABAC engine is available
		// if !abacEngine.Evaluate(c.Request.Context(), "suggest", "workflow") {
		//     c.JSON(403, gin.H{"error": "ABAC denied: insufficient permissions to generate workflow suggestions"})
		//     return
		// }

		var req struct {
			Description string                 `json:"description" binding:"required"`
			Context     map[string]interface{} `json:"context,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request: description is required"})
			return
		}

		// Call xAI API for workflow suggestion
		suggestion, err := generateWorkflowSuggestion(req.Description, req.Context)
		if err != nil {
			log.Printf("Error generating workflow suggestion: %v", err)
			c.JSON(500, gin.H{"error": "Failed to generate workflow suggestion"})
			return
		}

		// Start Temporal workflow to process the suggestion
		workflowOptions := client.StartWorkflowOptions{
			ID:        fmt.Sprintf("ai-workflow-suggestion-%d", time.Now().Unix()),
			TaskQueue: "ai-builder",
		}

		if tc == nil {
			log.Printf("Temporal client unavailable; cannot start workflow for suggestion")
			c.JSON(503, gin.H{"error": "Temporal unavailable"})
			return
		}

		we, err := tc.ExecuteWorkflow(context.Background(), workflowOptions, "ProcessWorkflowSuggestion", *suggestion)
		if err != nil {
			log.Printf("Error starting Temporal workflow: %v", err)
			c.JSON(500, gin.H{"error": "Failed to process workflow suggestion"})
			return
		}

		// TODO: Log ABAC audit event when ABAC engine is available
		// abacEngine.LogAudit(c.Request.Context(), map[string]interface{}{
		//     "action":      "suggest_workflow",
		//     "resource":    "workflow",
		//     "workflow_id": we.GetID(),
		//     "description": req.Description,
		//     "timestamp":   time.Now(),
		// })

		c.JSON(200, gin.H{
			"suggestion":  suggestion,
			"workflow_id": we.GetID(),
		})
	})

	// Get workflow suggestion status
	r.GET("/workflows/:id/status", func(c *gin.Context) {
		workflowID := c.Param("id")

		// TODO: Add ABAC permission check when ABAC engine is available
		// if !abacEngine.Evaluate(c.Request.Context(), "read", fmt.Sprintf("workflow-%s", workflowID)) {
		//     c.JSON(403, gin.H{"error": "ABAC denied: insufficient permissions to view workflow status"})
		//     return
		// }

		resp, err := tc.DescribeWorkflowExecution(context.Background(), workflowID, "")
		if err != nil {
			c.JSON(404, gin.H{"error": "Workflow not found"})
			return
		}

		c.JSON(200, gin.H{
			"workflow_id": resp.WorkflowExecutionInfo.Execution.WorkflowId,
			"status":      resp.WorkflowExecutionInfo.Status.String(),
			"start_time":  resp.WorkflowExecutionInfo.StartTime,
			"close_time":  resp.WorkflowExecutionInfo.CloseTime,
		})
	})

	// AI chat endpoint for semantic processing
	r.POST("/chat", func(c *gin.Context) {
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Temperature float64 `json:"temperature,omitempty"`
			MaxTokens   int     `json:"max_tokens,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		// Call xAI API for chat completion
		response, err := callXAICHAT(req)
		if err != nil {
			log.Printf("Error calling xAI API: %v", err)
			c.JSON(500, gin.H{"error": "Failed to process chat request"})
			return
		}

		c.JSON(200, response)
	})

	log.Println("AI Workflow Builder service starting on :8082")
	if err := r.Run(":8082"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// generateWorkflowSuggestion calls xAI API to generate workflow suggestions
func generateWorkflowSuggestion(description string, context map[string]interface{}) (*WorkflowSuggestion, error) {
	// Prepare xAI API request
	prompt := fmt.Sprintf(`Generate a Temporal workflow YAML and ReactFlow elements for wealth management: %s

Include:
- Start/End nodes
- Decision points for compliance checks
- Activity nodes for data processing
- Error handling
- ABAC policy integration points

Return JSON with: { "yaml": "...", "elements": [...], "metadata": {...} }`, description)

	if context != nil {
		if industry, ok := context["industry"].(string); ok {
			prompt += fmt.Sprintf("\nIndustry context: %s", industry)
		}
		if compliance, ok := context["compliance"].(string); ok {
			prompt += fmt.Sprintf("\nCompliance requirements: %s", compliance)
		}
	}

	reqBody := map[string]interface{}{
		"model": "grok-beta",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
		"max_tokens":  2000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.x.ai/v1/chat/completions", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getXAIAPIKey())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call xAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("xAI API error: %s - %s", resp.Status, string(body))
	}

	var xaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&xaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode xAI response: %w", err)
	}

	if len(xaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from xAI API")
	}

	// Parse the JSON response from xAI
	var suggestion WorkflowSuggestion
	content := xaiResp.Choices[0].Message.Content

	// Extract JSON from markdown code blocks if present
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end != -1 {
			content = content[start : start+end]
		}
	}

	if err := json.Unmarshal([]byte(content), &suggestion); err != nil {
		// If direct parsing fails, try to extract JSON
		log.Printf("Failed to parse xAI response as JSON, content: %s", content)
		return nil, fmt.Errorf("invalid response format from xAI API")
	}

	// Set description and add metadata
	suggestion.Description = description
	if suggestion.Metadata == nil {
		suggestion.Metadata = make(map[string]interface{})
	}
	suggestion.Metadata["generated_at"] = time.Now()
	suggestion.Metadata["source"] = "xai-api"

	return &suggestion, nil
}

// callXAICHAT calls xAI API for chat completions (mocked for testing)
func callXAICHAT(req struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}) (map[string]interface{}, error) {
	// For testing purposes, return a mock response instead of calling xAI API
	// TODO: Replace with actual xAI API call when API key is available

	// Extract the user message for processing
	var userMessage string
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userMessage = msg.Content
			break
		}
	}

	// Generate a mock semantic response
	mockResponse := fmt.Sprintf(`{
		"id": "mock-chat-completion",
		"object": "chat.completion",
		"created": %d,
		"model": "%s",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Mock AI response for: %s\\n\\nThis is a simulated response for testing microservices integration. The semantic processing would analyze the input and provide relevant insights about the query."
				},
				"finish_reason": "stop"
			}
		],
		"usage": {
			"prompt_tokens": 50,
			"completion_tokens": 100,
			"total_tokens": 150
		}
	}`, time.Now().Unix(), req.Model, userMessage)

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(mockResponse), &response); err != nil {
		return nil, fmt.Errorf("failed to create mock response: %w", err)
	}

	return response, nil
}

// getXAIAPIKey retrieves the xAI API key from environment
func getXAIAPIKey() string {
	// In production, this should come from a secure secret store
	return "your-xai-api-key-here"
}
