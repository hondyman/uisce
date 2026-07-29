package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/ai/vocabulary"
)

// LLMIntentParser enhances intent parsing with LLM capabilities
type LLMIntentParser struct {
	baseParser    *IntentParser
	llmEndpoint   string
	llmAPIKey     string
	schemaContext string
	userContext   map[string]string
	enabled       bool
}

// LLMParseRequest represents the request to the LLM
type LLMParseRequest struct {
	Query            string            `json:"query"`
	SchemaContext    string            `json:"schema_context"`
	UserContext      map[string]string `json:"user_context"`
	ResolvedTerms    []string         `json:"resolved_terms,omitempty"`
	MemoryRecall     []string         `json:"memory_recall,omitempty"`
	SimilarPrompts   []string         `json:"similar_prior_prompts,omitempty"`
	Examples         []string         `json:"examples"`
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
		userContext: make(map[string]string),
	}
}

// ParseIntentWithLLM parses intent using LLM enhancement
func (llm *LLMIntentParser) ParseIntentWithLLM(ctx context.Context, text string, userContext map[string]string) (*ParsedIntent, error) {
	mergedCtx := llm.mergeUserContext(userContext)

	baseIntent, err := llm.baseParser.ParseIntent(text)
	if err != nil {
		return nil, err
	}

	if !llm.enabled || baseIntent.Confidence >= 0.8 {
		return baseIntent, nil
	}

	llmIntent, err := llm.callLLM(ctx, text, mergedCtx)
	if err != nil {
		fmt.Printf("LLM parsing failed, using base parser: %v\n", err)
		return baseIntent, nil
	}

	mergedIntent := llm.mergeIntents(baseIntent, llmIntent)
	return mergedIntent, nil
}

func (llm *LLMIntentParser) mergeUserContext(incoming map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range llm.userContext {
		merged[k] = v
	}
	for k, v := range incoming {
		merged[k] = v
	}
	return merged
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

	if terms, ok := userContext["resolved_vocabulary"]; ok && terms != "" {
		request.ResolvedTerms = strings.Split(terms, ", ")
	}
	if recall, ok := userContext["memory_recall"]; ok && recall != "" {
		request.SimilarPrompts = strings.Split(recall, "|")
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

// EnrichWithVocabulary updates the user context with resolved graph vocabulary terms
func (llm *LLMIntentParser) EnrichWithVocabulary(ctx context.Context, tenantID string, vocabResolver *VocabularyResolver, text string) error {
	if vocabResolver == nil || vocabResolver.Resolver == nil {
		return nil
	}
	tokens := extractVocabularyTokens(text)
	var terms []string
	for _, token := range tokens {
		canonicalTerms, err := vocabResolver.Resolver.ResolveTerm(ctx, tenantID, token)
		if err == nil && len(canonicalTerms) > 0 {
			for _, ct := range canonicalTerms {
				terms = append(terms, ct.TermName)
			}
		}
	}
	if len(terms) > 0 {
		if llm.userContext == nil {
			llm.userContext = make(map[string]string)
		}
		unique := removeDuplicatesStrings(terms)
		llm.userContext["resolved_vocabulary"] = strings.Join(unique, ", ")
	}
	return nil
}

// VocabularyResolver wraps the vocabulary resolver for use in LLM intent parser
type VocabularyResolver struct {
	*vocabulary.Resolver
}

func extractVocabularyTokens(text string) []string {
	raw := strings.Fields(strings.ToLower(text))
	var tokens []string
	for _, t := range raw {
		cleaned := strings.Trim(t, ".,?!'\"():;")
		if len(cleaned) > 1 {
			tokens = append(tokens, cleaned)
		}
	}
	return tokens
}

func removeDuplicatesStrings(slice []string) []string {
	seen := make(map[string]struct{})
	out := []string{}
	for _, s := range slice {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
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
