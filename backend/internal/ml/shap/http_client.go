package shap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hondyman/semlayer/backend/internal/ml"
)

// HTTPClient represents an HTTP SHAP service client
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	retries    int
}

// ExplainRequestDTO is the request DTO for SHAP service
type ExplainRequestDTO struct {
	ChainID      string             `json:"chain_id"`
	Region       string             `json:"region"`
	Features     map[string]float64 `json:"features"`
	ModelVersion string             `json:"model_version"`
}

// SHAPCoefficientDTO represents a SHAP value
type SHAPCoefficientDTO struct {
	Feature     string  `json:"feature"`
	Index       int     `json:"index"`
	Coefficient float64 `json:"coefficient"`
	Baseline    float64 `json:"baseline"`
}

// ExplainResponseDTO is the response DTO from SHAP service
type ExplainResponseDTO struct {
	ChainID           string               `json:"chain_id"`
	BaseValue         float64              `json:"base_value"`
	SHAPValues        []SHAPCoefficientDTO `json:"shap_values"`
	FeatureImportance map[string]float64   `json:"feature_importance"`
	ComputeTimeMs     float64              `json:"computation_time_ms"`
	Timestamp         string               `json:"timestamp"`
}

// ExplainBatchRequestDTO for batch requests
type ExplainBatchRequestDTO struct {
	Requests        []ExplainRequestDTO `json:"requests"`
	Parallelization int                 `json:"parallelization"`
}

// ExplainBatchResponseDTO for batch responses
type ExplainBatchResponseDTO struct {
	TotalRequests          int                  `json:"total_requests"`
	SuccessfulExplanations int                  `json:"successful_explanations"`
	FailedExplanations     int                  `json:"failed_explanations"`
	Explanations           []ExplainResponseDTO `json:"explanations"`
	TotalComputeTimeMs     float64              `json:"total_compute_time_ms"`
	Timestamp              string               `json:"timestamp"`
}

// ServiceHealthDTO for health check
type ServiceHealthDTO struct {
	Status        string  `json:"status"`
	SHAPAvailable bool    `json:"shap_available"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	Version       string  `json:"version"`
	Timestamp     string  `json:"timestamp"`
}

// NewHTTPClient creates a new SHAP service HTTP client
func NewHTTPClient(baseURL string, timeout time.Duration, retries int) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
		retries: retries,
	}
}

// ComputeSHAP sends a single request to the SHAP service
func (c *HTTPClient) ComputeSHAP(ctx context.Context, chainID string, region string, features map[string]float64, modelVersion string) (*ml.Explainability, error) {
	request := ExplainRequestDTO{
		ChainID:      chainID,
		Region:       region,
		Features:     features,
		ModelVersion: modelVersion,
	}

	response, err := c.doRequestSingle(ctx, "POST", "/explain", request, c.retries)
	if err != nil {
		return nil, fmt.Errorf("SHAP computation failed: %w", err)
	}

	// Convert response to ml.Explainability
	explainability := &ml.Explainability{
		BaseValue:          response.BaseValue,
		FeatureImportance:  response.FeatureImportance,
		LocalContributions: []ml.LocalContribution{},
		ComputationTime:    response.ComputeTimeMs,
	}

	// Build local contributions from SHAP values
	for _, coeff := range response.SHAPValues {
		impact := "neutral"
		if coeff.Coefficient > 0.05 {
			impact = "positive"
		} else if coeff.Coefficient < -0.05 {
			impact = "negative"
		}

		explainability.LocalContributions = append(explainability.LocalContributions, ml.LocalContribution{
			Feature:   coeff.Feature,
			SHAPValue: coeff.Coefficient,
			Impact:    impact,
		})
	}

	// Sort by absolute contribution
	sortContributionsByImportance(explainability.LocalContributions)

	return explainability, nil
}

// ComputeBatchSHAP sends a batch request to the SHAP service
func (c *HTTPClient) ComputeBatchSHAP(ctx context.Context, requests []struct {
	ChainID, Region string
	Features        map[string]float64
	ModelVersion    string
}, parallelization int) (map[string]*ml.Explainability, error) {
	if len(requests) == 0 {
		return make(map[string]*ml.Explainability), nil
	}

	// Convert to DTOs
	dtoRequests := make([]ExplainRequestDTO, len(requests))
	for i, req := range requests {
		dtoRequests[i] = ExplainRequestDTO{
			ChainID:      req.ChainID,
			Region:       req.Region,
			Features:     req.Features,
			ModelVersion: req.ModelVersion,
		}
	}

	batchRequest := ExplainBatchRequestDTO{
		Requests:        dtoRequests,
		Parallelization: parallelization,
	}

	response, err := c.doRequestBatch(ctx, "POST", "/explain/batch", batchRequest, c.retries)
	if err != nil {
		return nil, fmt.Errorf("batch SHAP computation failed: %w", err)
	}

	// Convert batch response
	result := make(map[string]*ml.Explainability)
	for _, expl := range response.Explanations {
		convertedExpl := &ml.Explainability{
			BaseValue:          expl.BaseValue,
			FeatureImportance:  expl.FeatureImportance,
			LocalContributions: []ml.LocalContribution{},
			ComputationTime:    expl.ComputeTimeMs,
		}

		for _, coeff := range expl.SHAPValues {
			impact := "neutral"
			if coeff.Coefficient > 0.05 {
				impact = "positive"
			} else if coeff.Coefficient < -0.05 {
				impact = "negative"
			}

			convertedExpl.LocalContributions = append(convertedExpl.LocalContributions, ml.LocalContribution{
				Feature:   coeff.Feature,
				SHAPValue: coeff.Coefficient,
				Impact:    impact,
			})
		}

		sortContributionsByImportance(convertedExpl.LocalContributions)
		result[expl.ChainID] = convertedExpl
	}

	return result, nil
}

// HealthCheck checks if the SHAP service is healthy
func (c *HTTPClient) HealthCheck(ctx context.Context) (bool, error) {
	url := c.baseURL + "/health"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("service returned status %d", resp.StatusCode)
	}

	var health ServiceHealthDTO
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return false, err
	}

	return health.Status == "healthy", nil
}

// GetMetrics retrieves service metrics
func (c *HTTPClient) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	url := c.baseURL + "/metrics"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service returned status %d", resp.StatusCode)
	}

	var metrics map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

// doRequestSingle performs an HTTP request and decodes a single ExplainResponseDTO
func (c *HTTPClient) doRequestSingle(ctx context.Context, method string, endpoint string, body interface{}, retries int) (*ExplainResponseDTO, error) {
	url := c.baseURL + endpoint

	// Encode body
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "semlayer/3.18")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("service returned status %d: %s", resp.StatusCode, string(respBody))
			if resp.StatusCode >= 500 && attempt < retries {
				continue
			}
			return nil, lastErr
		}

		var result ExplainResponseDTO
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		return &result, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", retries, lastErr)
}

// doRequestBatch performs an HTTP request and decodes an ExplainBatchResponseDTO
func (c *HTTPClient) doRequestBatch(ctx context.Context, method string, endpoint string, body interface{}, retries int) (*ExplainBatchResponseDTO, error) {
	url := c.baseURL + endpoint

	// Encode body
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "semlayer/3.18")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("service returned status %d: %s", resp.StatusCode, string(respBody))
			if resp.StatusCode >= 500 && attempt < retries {
				continue
			}
			return nil, lastErr
		}

		var result ExplainBatchResponseDTO
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		return &result, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", retries, lastErr)
}

// sortContributionsByImportance sorts contributions by absolute SHAP value
func sortContributionsByImportance(contributions []ml.LocalContribution) {
	for i := 0; i < len(contributions)-1; i++ {
		for j := i + 1; j < len(contributions); j++ {
			if abs(contributions[j].SHAPValue) > abs(contributions[i].SHAPValue) {
				contributions[i], contributions[j] = contributions[j], contributions[i]
			}
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
