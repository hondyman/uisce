package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// LLMProvider defines the interface for interacting with a large language model.
type LLMProvider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}

// GeminiProvider is an implementation of LLMProvider for Google's Gemini models.
type GeminiProvider struct {
	APIKey         string
	ModelName      string
	EmbeddingModel string
	Client         *http.Client
}

// NewGeminiProvider creates a new Gemini provider.
func NewGeminiProvider(apiKey, modelName string) *GeminiProvider {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if modelName == "" {
		modelName = "gemini-2.0-flash-exp" // Default model
	}
	return &GeminiProvider{
		APIKey:         apiKey,
		ModelName:      modelName,
		EmbeddingModel: "text-embedding-004",
		Client:         &http.Client{},
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
		"generationConfig": map[string]interface{}{
			"temperature":     0.2,
			"topP":            0.95,
			"topK":            40,
			"maxOutputTokens": 8192,
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
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return "", fmt.Errorf("Gemini API returned non-200 status: %s, body: %s", resp.Status, errBody.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Gemini API response: %w", err)
	}

	// Extract the text from the complex response structure
	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	content, ok := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid content structure in response")
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", fmt.Errorf("no parts in response content")
	}

	text, ok := parts[0].(map[string]interface{})["text"].(string)
	if !ok {
		return "", fmt.Errorf("no text in response part")
	}

	return text, nil
}

// Embed generates an embedding vector for the given text using Gemini's embedding model.
func (g *GeminiProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if g.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s", g.EmbeddingModel, g.APIKey)

	requestBody, err := json.Marshal(map[string]interface{}{
		"content": map[string]interface{}{
			"parts": []map[string]string{
				{"text": text},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send embedding request to Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("Gemini embedding API returned non-200 status: %s, body: %s", resp.Status, errBody.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	// Extract embedding values
	embedding, ok := result["embedding"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no embedding in response")
	}

	values, ok := embedding["values"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no values in embedding")
	}

	floats := make([]float32, len(values))
	for i, v := range values {
		if f, ok := v.(float64); ok {
			floats[i] = float32(f)
		} else {
			return nil, fmt.Errorf("invalid value type in embedding at index %d", i)
		}
	}

	return floats, nil
}
