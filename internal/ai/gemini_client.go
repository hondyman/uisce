package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

// GeminiClient wraps the Gemini Pro model for generating structured proposals.
type GeminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// NewGeminiClient creates a client using the GOOGLE_GENERATIVE_AI_API_KEY env var.
func NewGeminiClient(ctx context.Context) (*GeminiClient, error) {
	apiKey := os.Getenv("GOOGLE_GENERATIVE_AI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_GENERATIVE_AI_API_KEY not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Use the "gemini-pro" model
	model := client.GenerativeModel("gemini-pro")
	
	// Configure for JSON output
	model.GenerationConfig.ResponseMIMEType = "application/json"

	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

// GenerateProposal calls the LLM with a structured prompt and returns the raw JSON string.
func (c *GeminiClient) GenerateProposal(ctx context.Context, driftJSON string) (string, error) {
	// Build a system prompt that tells the model to output a TradeProposal JSON.
	systemPrompt := `You are an autonomous wealth‑management AI. Given a drift report JSON, output a TradeProposal JSON with the following schema:
{
  "id": "<uuid>",
  "trades": [{"side": "BUY|SELL", "symbol": "<ticker>", "qty": <int> }],
  "explanation": "<human readable rationale>",
  "confidence": <0.0‑1.0>,
  "grounding": [{"source": "<source name>", "snippet": "<text>", "snapshot_id": "<hash>"}]
}
Only output the JSON object, no surrounding text.`

	// Combine system prompt with the drift payload.
	resp, err := c.model.GenerateContent(ctx, genai.Text(systemPrompt), genai.Text(driftJSON))
	if err != nil {
		return "", fmt.Errorf("gemini generate error: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	// The model returns a single part with the JSON string.
	jsonStr := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	// Ensure the proposal has a UUID – if the model omitted it, inject one.
	if !containsID(jsonStr) {
		// naive injection – in production use proper JSON parsing.
		jsonStr = fmt.Sprintf(`{"id":"%s",%s`, uuid.New().String(), jsonStr[1:])
	}

	return jsonStr, nil
}

// Close releases the Gemini client resources.
func (c *GeminiClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func containsID(s string) bool {
	// Very simple check for "\"id\"" key.
	return strings.Contains(s, `"id"`)
}

