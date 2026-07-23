package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// SimpleTextProvider defines the interface for basic text generation from an LLM.
type SimpleTextProvider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
}

// GeminiProvider is an implementation of SimpleTextProvider for Google's Gemini models.
type GeminiProvider struct {
	APIKey    string
	ModelName string
	Client    *http.Client
}

// NewGeminiProvider creates a new Gemini provider.
func NewGeminiProvider(apiKey, modelName string) *GeminiProvider {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if modelName == "" {
		modelName = "gemini-pro" // Default model
	}
	return &GeminiProvider{
		APIKey:    apiKey,
		ModelName: modelName,
		Client:    &http.Client{},
	}
}

// GenerateResponse sends a prompt to the Gemini API and returns the response.
func (g *GeminiProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	if g.APIKey == "" {
		return "", fmt.Errorf("Gemini API key is not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.ModelName, g.APIKey)

	requestBody, err := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned non-200 status: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Gemini API response: %w", err)
	}

	// Extract the text from the complex response structure
	candidates := result["candidates"].([]interface{})
	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	text := parts[0].(map[string]interface{})["text"].(string)

	return text, nil
}
