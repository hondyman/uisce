package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type EmbeddingService struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

func NewEmbeddingService() *EmbeddingService {
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &EmbeddingService{
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Model:      "text-embedding-3-small", // 1536 dimensions
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type embeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// GenerateEmbedding converts a text string into a 1536-dimensional vector
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if s.APIKey == "" {
		return nil, fmt.Errorf("no OpenAI API key configured for embedding")
	}

	reqBody := embeddingRequest{
		Input: text,
		Model: s.Model,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.BaseURL+"/embeddings", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.APIKey)

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API error: status %d", resp.StatusCode)
	}

	var resData embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&resData); err != nil {
		return nil, err
	}

	if len(resData.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return resData.Data[0].Embedding, nil
}
