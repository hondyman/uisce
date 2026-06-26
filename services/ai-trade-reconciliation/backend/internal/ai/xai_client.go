package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// xAIClient handles communication with xAI LLM
type xAIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewxAIClient initializes the xAI client
func NewxAIClient() *xAIClient {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GROK_API_KEY") // fallback
	}

	return &xAIClient{
		baseURL:    "https://api.x.ai/v1",
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// CompleteRequest represents a completion request
type CompleteRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    float64         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat specifies JSON mode
type ResponseFormat struct {
	Type string `json:"type"` // "json_object"
}

// CompleteResponse represents the response from xAI
type CompleteResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Message Message `json:"message"`
	Index   int     `json:"index"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Complete calls the xAI API with JSON response mode
func (c *xAIClient) Complete(ctx context.Context, prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("XAI_API_KEY not configured")
	}

	req := CompleteRequest{
		Model: "grok-beta", // or "grok" for production
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3, // Lower temp for deterministic output
		MaxTokens:   4000,
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call xAI API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("xAI API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var respData CompleteResponse
	if err := json.Unmarshal(respBody, &respData); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(respData.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return respData.Choices[0].Message.Content, nil
}
