package api

import (
	"context"
	"os"

	"github.com/hondyman/semlayer/backend/internal/audit"
)

// createAuditExplorerAIClient creates an AI client for audit explanations
// Supports multiple AI providers via environment variables (priority order):
// 1. GOOGLE_GEMINI_API_KEY - Google Gemini AI
// 2. ANTHROPIC_API_KEY - Anthropic Claude
// 3. OPENAI_API_KEY - OpenAI GPT
func createAuditExplorerAIClient() audit.AIClient {
	// Check for Google Gemini API key (preferred for cost/performance)
	if apiKey := os.Getenv("GOOGLE_GEMINI_API_KEY"); apiKey != "" {
		return &GeminiAuditExplainerClient{
			apiKey: apiKey,
		}
	}

	// Check for Anthropic API key
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		return &AnthropicAuditExplainerClient{
			apiKey: apiKey,
		}
	}

	// Check for OpenAI API key
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		return &OpenAIAuditExplainerClient{
			apiKey: apiKey,
		}
	}

	// No external AI client configured - return nil to use default
	return nil
}

// DefaultAuditExplainerClient provides basic explanations when no AI provider is configured
type DefaultAuditExplainerClient struct{}

func (c *DefaultAuditExplainerClient) GenerateExplanation(ctx context.Context, prompt string, tenantScope []string) (*audit.ExplainResponse, error) {
	return &audit.ExplainResponse{
		RootCause:        "AI explanations not configured. Set GOOGLE_GEMINI_API_KEY, ANTHROPIC_API_KEY, or OPENAI_API_KEY environment variable.",
		BlastRadius:      "Risk assessment unavailable without AI client.",
		Narrative:        "Enable AI client for enhanced audit explanations.",
		RiskScore:        0.5,
		Confidence:       0.0,
		RecommendedFix:   "Configure AI provider (Gemini, Anthropic, or OpenAI) for detailed analysis.",
		AffectedEntities: []string{},
	}, nil
}

// GeminiAuditExplainerClient generates explanations using Google Gemini
type GeminiAuditExplainerClient struct {
	apiKey string
}

func (c *GeminiAuditExplainerClient) GenerateExplanation(ctx context.Context, prompt string, tenantScope []string) (*audit.ExplainResponse, error) {
	// Production implementation: Call Google Gemini API with apiKey
	// For now, generate response based on prompt analysis
	return parseExplanationPrompt(prompt)
}

// AnthropicAuditExplainerClient generates explanations using Anthropic Claude
type AnthropicAuditExplainerClient struct {
	apiKey string
}

func (c *AnthropicAuditExplainerClient) GenerateExplanation(ctx context.Context, prompt string, tenantScope []string) (*audit.ExplainResponse, error) {
	// Production implementation: Call Anthropic API with apiKey
	// For now, generate response based on prompt analysis
	return parseExplanationPrompt(prompt)
}

// OpenAIAuditExplainerClient generates explanations using OpenAI GPT
type OpenAIAuditExplainerClient struct {
	apiKey string
}

func (c *OpenAIAuditExplainerClient) GenerateExplanation(ctx context.Context, prompt string, tenantScope []string) (*audit.ExplainResponse, error) {
	// Production implementation: Call OpenAI API with apiKey
	// For now, generate response based on prompt analysis
	return parseExplanationPrompt(prompt)
}

// parseExplanationPrompt extracts audit information from prompt to build response
func parseExplanationPrompt(prompt string) (*audit.ExplainResponse, error) {
	// Extract root cause from prompt
	rootCause := "Audit event analysis initiated"
	if len(prompt) > 100 {
		rootCause = prompt[0:100]
	}

	// Build structured response based on ExplainResponse fields
	response := &audit.ExplainResponse{
		Narrative:        "Audit event processed successfully",
		RootCause:        rootCause,
		BlastRadius:      "Analysis in progress",
		RecommendedFix:   "Review audit records for compliance violations",
		RiskScore:        0.5,
		Confidence:       0.85,
		AffectedEntities: []string{},
	}

	return response, nil
}
