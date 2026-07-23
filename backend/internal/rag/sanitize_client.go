package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SanitizationClient handles communication with the PII sanitization proxy
type SanitizationClient struct {
	BaseURL string
	Client  *http.Client
}

// SanitizeRequest represents a request to sanitize text
type SanitizeRequest struct {
	TenantID  string `json:"tenant_id"`
	ClientID  string `json:"client_id"`
	Text      string `json:"text"`
	RequestID string `json:"request_id"`
}

// SanitizeResponse represents the response from the sanitization proxy
type SanitizeResponse struct {
	SanitizedText string `json:"sanitized_text"`
	PIIMapID      string `json:"pii_map_id"`
}

// SanitizeText sends text to the proxy for PII masking
func (s *SanitizationClient) SanitizeText(req SanitizeRequest) (SanitizeResponse, error) {
	if s.Client == nil {
		s.Client = &http.Client{Timeout: 5 * time.Second}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return SanitizeResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.BaseURL+"/sanitize", bytes.NewReader(body))
	if err != nil {
		return SanitizeResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(httpReq)
	if err != nil {
		return SanitizeResponse{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SanitizeResponse{}, fmt.Errorf("sanitization failed with status: %d", resp.StatusCode)
	}

	var out SanitizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return SanitizeResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return out, nil
}
