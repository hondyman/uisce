package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ForecastExplainer struct {
	EmbeddingService *EmbeddingService
}

func NewForecastExplainer(embedder *EmbeddingService) *ForecastExplainer {
	return &ForecastExplainer{EmbeddingService: embedder}
}

type ForecastExplanation struct {
	SummaryNarrative string   `json:"summaryNarrative"`
	AuditableSteps   []string `json:"auditableSteps"`
}

// ExplainForecast sends attribution metrics to the LLM to generate an audit trail
func (e *ForecastExplainer) ExplainForecast(ctx context.Context, dimension string, measure string, rows []ProjectedRow) (*ForecastExplanation, error) {
	if e.EmbeddingService == nil || e.EmbeddingService.APIKey == "" {
		return &ForecastExplanation{
			SummaryNarrative: fmt.Sprintf("Forecast for %s by %s generated via linear trend extraction and seasonal decomposition.", measure, dimension),
			AuditableSteps: []string{
				"Step 1: Computed baseline linear regression slope and intercept across historical periods.",
				"Step 2: Calculated 12-period cyclical multiplicative seasonal adjustments.",
				"Step 3: Established 95% confidence intervals using standard error of historical residuals.",
			},
		}, nil
	}

	var forecastSamples []ProjectedRow
	for _, r := range rows {
		if r.IsForecast {
			forecastSamples = append(forecastSamples, r)
		}
	}

	sampleLimit := len(forecastSamples)
	if sampleLimit > 4 {
		sampleLimit = 4
	}

	payloadJSON, _ := json.MarshalIndent(forecastSamples[:sampleLimit], "", "  ")

	systemPrompt := `You are an institutional financial auditor and quantitative model validator.
Review the provided forecasting attribution telemetry (Trend component, Seasonal Multiplier, and Confidence Bounds) for the requested metric and dimension.
Explain step-by-step how the model arrived at these projections. Break down the baseline trend versus the seasonal impact.

Respond ONLY with a valid JSON object matching this schema:
{
  "summaryNarrative": "A concise 2-sentence executive summary of the forecast trajectory.",
  "auditableSteps": [
    "Step 1: Description of baseline trend extraction.",
    "Step 2: Description of seasonal adjustment applied.",
    "Step 3: Description of risk boundaries / confidence intervals."
  ]
}`

	userMessage := fmt.Sprintf("Metric: %s\nDimension: %s\n\nSample Projected Attribution Data:\n%s", measure, dimension, string(payloadJSON))

	reqBody := map[string]interface{}{
		"model":       e.EmbeddingService.Model,
		"temperature": 0.1,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"response_format": map[string]string{"type": "json_object"},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", e.EmbeddingService.BaseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.EmbeddingService.APIKey)

	resp, err := e.EmbeddingService.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM error status: %d", resp.StatusCode)
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil || len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("failed to decode LLM response")
	}

	var result ForecastExplanation
	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &result); err != nil {
		return nil, err
	}

	return &result, nil
}
