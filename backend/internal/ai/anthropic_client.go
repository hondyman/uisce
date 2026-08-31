package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AnthropicClient is a concrete implementation of the
// audit.AuditNarrativeAIClient seam (GenerateNarrative(ctx, prompt, context)
// (string, error)) backed by the real Anthropic Messages API. It is
// implemented here (not in package audit) to avoid an import cycle, since
// it needs ProviderDispatcher for BYOK credential resolution; callers wire
// it in via the interface, same as any other implementation would be.
type AnthropicClient struct {
	dispatcher *ProviderDispatcher
	httpClient *http.Client
	// defaultModel is used when a tenant's BYOK config doesn't specify one.
	defaultModel string
	// platformAPIKey is used when ResolveProvider falls back to the
	// platform default rather than a tenant's own Anthropic credentials.
	platformAPIKey string
}

// NewAnthropicClient builds a client that resolves credentials per-tenant
// via dispatcher.ResolveProvider (BYOK-aware) and falls back to
// platformAPIKey (e.g. from ANTHROPIC_API_KEY) for tenants without their
// own provider configured.
func NewAnthropicClient(dispatcher *ProviderDispatcher, platformAPIKey string) *AnthropicClient {
	return &AnthropicClient{
		dispatcher:     dispatcher,
		httpClient:     &http.Client{Timeout: 60 * time.Second},
		defaultModel:   "claude-sonnet-4-5",
		platformAPIKey: platformAPIKey,
	}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Error   *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateNarrative implements audit.AuditNarrativeAIClient. contextData is
// flattened into the system prompt as JSON so callers (audit narratives,
// exception root-cause/fix/closure prompts) don't need bespoke wire types.
func (c *AnthropicClient) GenerateNarrative(ctx context.Context, prompt string, contextData map[string]interface{}) (string, error) {
	return c.generate(ctx, "", prompt, contextData)
}

// GenerateNarrativeForTenant is like GenerateNarrative but resolves
// BYOK credentials for the given tenant instead of the platform default.
func (c *AnthropicClient) GenerateNarrativeForTenant(ctx context.Context, tenantID string, prompt string, contextData map[string]interface{}) (string, error) {
	return c.generate(ctx, tenantID, prompt, contextData)
}

func (c *AnthropicClient) generate(ctx context.Context, tenantID, prompt string, contextData map[string]interface{}) (string, error) {
	apiKey := c.platformAPIKey
	model := c.defaultModel
	endpoint := "https://api.anthropic.com/v1/messages"

	if c.dispatcher != nil && tenantID != "" {
		cfg, err := c.dispatcher.ResolveProvider(ctx, tenantID)
		if err == nil && cfg != nil && cfg.ProviderType == ProviderAnthropic && cfg.DecryptedAPIKey != "" && cfg.DecryptedAPIKey != "platform_default_key" {
			apiKey = cfg.DecryptedAPIKey
			if cfg.ModelDeploymentName != "" {
				model = cfg.ModelDeploymentName
			}
			if cfg.APIEndpoint != "" {
				endpoint = cfg.APIEndpoint
			}
		}
	}
	if apiKey == "" {
		return "", fmt.Errorf("anthropic: no API key configured (set ANTHROPIC_API_KEY or a tenant BYOK provider)")
	}

	systemPrompt := "You are an assistant embedded in a platform intelligence console. Be concise and factual."
	if len(contextData) > 0 {
		ctxJSON, err := json.Marshal(contextData)
		if err == nil {
			systemPrompt += "\n\nContext:\n" + string(ctxJSON)
		}
	}

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: read response: %w", err)
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("anthropic: parse response: %w (status %d)", err, resp.StatusCode)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("anthropic: %s: %s", parsed.Error.Type, parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic: unexpected status %d: %s", resp.StatusCode, string(body))
	}
	if len(parsed.Content) == 0 {
		return "", fmt.Errorf("anthropic: empty response content")
	}

	var text string
	for _, block := range parsed.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return text, nil
}
