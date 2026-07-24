package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMIntentParser enhances intent parsing with LLM capabilities
type LLMIntentParser struct {
	baseParser    *IntentParser
	llmEndpoint   string
	llmAPIKey     string
	schemaContext string
	enabled       bool
}

// LLMParseRequest represents the request to the LLM
type LLMParseRequest struct {
	Query         string            `json:"query"`
	SchemaContext string            `json:"schema_context"`
	UserContext   map[string]string `json:"user_context"`
	Examples      []string          `json:"examples"`
}

// LLMParseResponse represents the response from the LLM
type LLMParseResponse struct {
	Metrics     []string          `json:"metrics"`
	Dimensions  []string          `json:"dimensions"`
	Filters     []LLMFilter       `json:"filters"`
	TimeRange   *LLMTimeRange     `json:"time_range,omitempty"`
	Aggregation string            `json:"aggregation,omitempty"`
	Confidence  float64           `json:"confidence"`
	Explanation string            `json:"explanation"`
	RawEntities map[string]string `json:"raw_entities"`
}

// LLMFilter represents a filter from LLM parsing
type LLMFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// LLMTimeRange represents a time range from LLM parsing
type LLMTimeRange struct {
	Start    string `json:"start,omitempty"`
	End      string `json:"end,omitempty"`
	Label    string `json:"label"`
	Relative string `json:"relative,omitempty"`
}

// NewLLMIntentParser creates a new LLM-enhanced intent parser
func NewLLMIntentParser(baseParser *IntentParser, llmEndpoint, apiKey string) *LLMIntentParser {
	return &LLMIntentParser{
		baseParser:  baseParser,
		llmEndpoint: llmEndpoint,
		llmAPIKey:   apiKey,
		enabled:     llmEndpoint != "" && apiKey != "",
	}
}

// ParseIntentWithLLM parses intent using LLM enhancement
func (llm *LLMIntentParser) ParseIntentWithLLM(ctx context.Context, text string, userContext map[string]string) (*ParsedIntent, error) {
	// First try the base parser for fast results
	baseIntent, err := llm.baseParser.ParseIntent(text)
	if err != nil {
		return nil, err
	}

	// If LLM is not enabled or base parser has high confidence, return base result
	if !llm.enabled || baseIntent.Confidence >= 0.8 {
		return baseIntent, nil
	}

	// Try LLM enhancement
	llmIntent, err := llm.callLLM(ctx, text, userContext)
	if err != nil {
		// Log error but return base result
		fmt.Printf("LLM parsing failed, using base parser: %v\n", err)
		return baseIntent, nil
	}

	// Merge LLM results with base parser results
	mergedIntent := llm.mergeIntents(baseIntent, llmIntent)

	return mergedIntent, nil
}

// callLLM makes the actual LLM API call
func (llm *LLMIntentParser) callLLM(ctx context.Context, query string, userContext map[string]string) (*LLMParseResponse, error) {
	if !llm.enabled {
		return nil, fmt.Errorf("LLM not enabled")
	}

	request := LLMParseRequest{
		Query:         query,
		SchemaContext: llm.schemaContext,
		UserContext:   userContext,
		Examples: []string{
			"Show me average order value by region last quarter",
			"What is the total revenue for EMEA in Q3 2024",
			"Show me customer count by product category this month",
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", llm.llmEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+llm.llmAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(body))
	}

	var llmResponse LLMParseResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return nil, fmt.Errorf("failed to decode LLM response: %w", err)
	}

	return &llmResponse, nil
}

// mergeIntents combines base parser and LLM results
func (llm *LLMIntentParser) mergeIntents(baseIntent *ParsedIntent, llmResponse *LLMParseResponse) *ParsedIntent {
	merged := &ParsedIntent{
		Metrics:     baseIntent.Metrics,
		Dimensions:  baseIntent.Dimensions,
		Filters:     baseIntent.Filters,
		TimeRange:   baseIntent.TimeRange,
		Aggregation: baseIntent.Aggregation,
		Confidence:  baseIntent.Confidence,
		RawEntities: make(map[string]string),
	}

	// Merge raw entities
	for k, v := range baseIntent.RawEntities {
		merged.RawEntities[k] = v
	}
	for k, v := range llmResponse.RawEntities {
		merged.RawEntities[k] = v
	}

	// Use LLM results if confidence is higher
	if llmResponse.Confidence > baseIntent.Confidence {
		merged.Metrics = llmResponse.Metrics
		merged.Dimensions = llmResponse.Dimensions
		merged.Aggregation = llmResponse.Aggregation
		merged.Confidence = llmResponse.Confidence

		// Convert LLM filters to intent filters
		merged.Filters = make([]IntentFilter, len(llmResponse.Filters))
		for i, f := range llmResponse.Filters {
			merged.Filters[i] = IntentFilter{
				Field:    f.Field,
				Operator: f.Operator,
				Values:   []string{f.Value},
			}
		}

		// Convert LLM time range
		if llmResponse.TimeRange != nil {
			merged.TimeRange = &TimeRange{
				Start: llmResponse.TimeRange.Start,
				End:   llmResponse.TimeRange.End,
				Label: llmResponse.TimeRange.Label,
			}
		}
	}

	return merged
}

// UpdateSchemaContext updates the schema context for LLM prompts
func (llm *LLMIntentParser) UpdateSchemaContext(schemaContext string) {
	llm.schemaContext = schemaContext
}

// GetLLMStatus returns the status of LLM integration
func (llm *LLMIntentParser) GetLLMStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":               llm.enabled,
		"endpoint":              llm.llmEndpoint,
		"has_api_key":           llm.llmAPIKey != "",
		"schema_context_length": len(llm.schemaContext),
	}
}
