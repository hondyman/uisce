package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// -----------------------------------------------------------------------------
// Domain Types & Contracts for AI Explorer
// -----------------------------------------------------------------------------

type AIExplorerSemanticCategory string

const (
	AIExplorerCategoryDimension AIExplorerSemanticCategory = "dimension"
	AIExplorerCategoryMeasure   AIExplorerSemanticCategory = "measure"
	AIExplorerCategoryTime      AIExplorerSemanticCategory = "time"
)

type AIExplorerSemanticField struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Label       string                     `json:"label"`
	Category    AIExplorerSemanticCategory `json:"category"`
	Type        string                     `json:"type"` // string, number, date, boolean
	Aggregation string                     `json:"aggregation,omitempty"`
}

type AIExplorerFilter struct {
	ID          string      `json:"id"`
	FieldID     string      `json:"fieldId"`
	Operator    string      `json:"operator"` // =, !=, IN, NOT IN, >, <, >=, <=, LIKE, BETWEEN
	Value       interface{} `json:"value"`
	IsParameter bool        `json:"isParameter,omitempty"`
}

type AIExplorerMeasureSelection struct {
	FieldID string `json:"fieldId"`
	Agg     string `json:"agg"` // SUM, AVG, COUNT, MIN, MAX
}

type AIExplorerTimeDimensionSelection struct {
	FieldID     string `json:"fieldId"`
	Granularity string `json:"granularity"` // raw, day, week, month, quarter, year
}

type AIExplorerQueryDefinition struct {
	Title          string                             `json:"title"`
	Dimensions     []string                           `json:"dimensions"`
	Measures       []AIExplorerMeasureSelection       `json:"measures"`
	TimeDimensions []AIExplorerTimeDimensionSelection `json:"timeDimensions"`
	Filters        []AIExplorerFilter                 `json:"filters"`
	Limit          int                                `json:"limit"`
	SuggestedChart string                             `json:"suggestedChart"` // table, bar, line, area, pie, kpi
}

type AIExplorerChatTurn struct {
	Role    string `json:"role"` // "user", "assistant"
	Content string `json:"content"`
}

type AIExplorerCompletionRequest struct {
	Messages      []AIExplorerChatTurn      `json:"messages"`
	CurrentQuery  AIExplorerQueryDefinition `json:"currentQuery"`
	Catalog       []AIExplorerSemanticField `json:"catalog"`
	UserAccountID string                    `json:"userAccountId,omitempty"`
	UserTenantID  string                    `json:"userTenantId,omitempty"`
}

type AIExplorerCompletionResponse struct {
	AssistantMessage   string                    `json:"assistantMessage"`
	GeneratedQuery     AIExplorerQueryDefinition `json:"generatedQuery"`
	SuggestedFollowUps []string                  `json:"suggestedFollowUps"`
	AmbiguityQuestions []string                  `json:"ambiguityQuestions,omitempty"`
	InsightSummary     string                    `json:"insightSummary,omitempty"`
	TopDriver          string                    `json:"topDriver,omitempty"`
	Anomalies          []string                  `json:"anomalies,omitempty"`
	IsCacheHit         bool                      `json:"isCacheHit,omitempty"`
	MutationIntent     string                    `json:"mutation_intent,omitempty"`
}

// -----------------------------------------------------------------------------
// OpenAI Client Types
// -----------------------------------------------------------------------------

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools"`
	ToolChoice  interface{}     `json:"tool_choice"`
	Temperature float64         `json:"temperature"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// -----------------------------------------------------------------------------
// AI Explorer Service
// -----------------------------------------------------------------------------

type AIExplorerService struct {
	OpenAIKey      string
	OpenAIBaseURL  string
	GeminiKey      string
	GeminiModel    string
	ModelName      string
	HTTPClient     *http.Client
	ContextBuilder *DynamicContextBuilder
	Cache          *SemanticCache
}

func NewAIExplorerService(db *sql.DB) *AIExplorerService {
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		geminiKey = os.Getenv("GOOGLE_GEMINI_API_KEY")
	}
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-1.5-flash"
	}

	embedder := NewEmbeddingService()
	var ctxBuilder *DynamicContextBuilder
	var cache *SemanticCache
	if db != nil {
		ctxBuilder = NewDynamicContextBuilder(db, embedder)
		cache = NewSemanticCache(db, embedder)
	}

	return &AIExplorerService{
		OpenAIKey:      apiKey,
		OpenAIBaseURL:  baseURL,
		GeminiKey:      geminiKey,
		GeminiModel:    geminiModel,
		ModelName:      model,
		HTTPClient:     &http.Client{Timeout: 35 * time.Second},
		ContextBuilder: ctxBuilder,
		Cache:          cache,
	}
}

func (s *AIExplorerService) BuildToolDefinition() openAITool {
	return openAITool{
		Type: "function",
		Function: openAIToolFunction{
			Name:        "generate_semantic_query",
			Description: "Translates natural language questions into an analytical dataset query using the defined semantic catalog.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"intentDescription": map[string]interface{}{
						"type":        "string",
						"description": "Executive summary of what this query shows.",
					},
					"mutation_intent": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"new_query", "drill_down", "drill_across", "add_filter", "add_measure", "remove_element"},
						"description": "Categorize how the user's prompt modifies the previous active query state.",
					},
					"dimensions": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "IDs of selected categorical dimension fields.",
					},
					"measures": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"fieldId": map[string]interface{}{"type": "string"},
								"agg":     map[string]interface{}{"type": "string", "enum": []string{"SUM", "AVG", "COUNT", "MIN", "MAX"}},
							},
							"required": []string{"fieldId", "agg"},
						},
					},
					"timeDimensions": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"fieldId":     map[string]interface{}{"type": "string"},
								"granularity": map[string]interface{}{"type": "string", "enum": []string{"raw", "day", "week", "month", "quarter", "year"}},
							},
							"required": []string{"fieldId", "granularity"},
						},
					},
					"filters": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"fieldId":  map[string]interface{}{"type": "string"},
								"operator": map[string]interface{}{"type": "string", "enum": []string{"=", "!=", "IN", "NOT IN", ">", "<", ">=", "<=", "LIKE", "BETWEEN"}},
								"value":    map[string]interface{}{"type": "string"},
							},
							"required": []string{"fieldId", "operator", "value"},
						},
					},
					"suggestedChart": map[string]interface{}{
						"type": "string",
						"enum": []string{"table", "bar", "line", "area", "pie", "kpi"},
					},
					"suggestedFollowUps": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "string"},
					},
					"ambiguityQuestions": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "string"},
					},
					"insightSummary": map[string]interface{}{
						"type":        "string",
						"description": "Short automated statistical takeaway based on the query structure.",
					},
					"topDriver": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"intentDescription", "dimensions", "measures", "suggestedChart", "suggestedFollowUps"},
			},
		},
	}
}

func (s *AIExplorerService) BuildSystemPrompt(catalog []AIExplorerSemanticField, currentQuery AIExplorerQueryDefinition, accountID, tenantID string) string {
	var catalogBuf bytes.Buffer
	for _, f := range catalog {
		catalogBuf.WriteString(fmt.Sprintf("- ID: `%s` | Label: \"%s\" | Category: %s | Type: %s | Default Agg: %s\n",
			f.ID, f.Label, f.Category, f.Type, f.Aggregation))
	}

	currQueryJSON, _ := json.MarshalIndent(currentQuery, "", "  ")

	return fmt.Sprintf(`You are the Uisce Data Explorer AI Analytical Engine.
Translate natural language requests into structured semantic queries.

SEMANTIC CATALOG:
%s

ACTIVE USER CONTEXT:
- Account ID: %s
- Tenant ID: %s

CURRENT QUERY STATE:
%s

RULES:
1. Retain existing filters and dimensions unless the user explicitly asks to clear or replace them.
2. Only select field IDs that exist in the SEMANTIC CATALOG.
3. If no dimensions exist and 1 measure is selected, set suggestedChart to 'kpi'.
4. If a timeDimension is selected with a continuous metric, choose 'line' or 'area'.
5. If categorical dimensions are selected, choose 'bar' or 'table'.
6. Always invoke the generate_semantic_query tool.`,
		catalogBuf.String(), accountID, tenantID, string(currQueryJSON))
}

func (s *AIExplorerService) ValidateQuery(q *AIExplorerQueryDefinition, catalog []AIExplorerSemanticField) error {
	validIDs := make(map[string]AIExplorerSemanticField)
	for _, f := range catalog {
		validIDs[f.ID] = f
	}

	for _, dim := range q.Dimensions {
		if f, exists := validIDs[dim]; !exists || (f.Category != AIExplorerCategoryDimension && f.Category != AIExplorerCategoryTime) {
			return fmt.Errorf("invalid dimension ID: '%s'", dim)
		}
	}

	for i, m := range q.Measures {
		if f, exists := validIDs[m.FieldID]; !exists || f.Category != AIExplorerCategoryMeasure {
			return fmt.Errorf("invalid measure ID: '%s'", m.FieldID)
		}
		if q.Measures[i].Agg == "" {
			q.Measures[i].Agg = "SUM"
		}
	}

	for _, flt := range q.Filters {
		if _, exists := validIDs[flt.FieldID]; !exists {
			return fmt.Errorf("invalid filter field ID: '%s'", flt.FieldID)
		}
	}

	if q.Limit <= 0 || q.Limit > 5000 {
		q.Limit = 500
	}
	return nil
}

// FallbackRuleEngine provides deterministic synthesis when LLM is unavailable or unconfigured
func (s *AIExplorerService) FallbackRuleEngine(req AIExplorerCompletionRequest) *AIExplorerCompletionResponse {
	lastPrompt := ""
	if len(req.Messages) > 0 {
		lastPrompt = strings.ToLower(req.Messages[len(req.Messages)-1].Content)
	}

	dims := append([]string{}, req.CurrentQuery.Dimensions...)
	meas := append([]AIExplorerMeasureSelection{}, req.CurrentQuery.Measures...)
	timeDims := append([]AIExplorerTimeDimensionSelection{}, req.CurrentQuery.TimeDimensions...)
	filters := append([]AIExplorerFilter{}, req.CurrentQuery.Filters...)

	dimMap := make(map[string]bool)
	for _, d := range dims {
		dimMap[d] = true
	}
	measMap := make(map[string]bool)
	for _, m := range meas {
		measMap[m.FieldID] = true
	}

	for _, f := range req.Catalog {
		lowerName := strings.ToLower(f.Name)
		lowerLabel := strings.ToLower(f.Label)
		matched := strings.Contains(lastPrompt, lowerName) || strings.Contains(lastPrompt, lowerLabel)

		if matched {
			if f.Category == AIExplorerCategoryDimension && !dimMap[f.ID] {
				dims = append(dims, f.ID)
				dimMap[f.ID] = true
			} else if f.Category == AIExplorerCategoryMeasure && !measMap[f.ID] {
				agg := f.Aggregation
				if agg == "" {
					agg = "SUM"
				}
				if strings.Contains(lastPrompt, "avg") || strings.Contains(lastPrompt, "average") {
					agg = "AVG"
				} else if strings.Contains(lastPrompt, "count") {
					agg = "COUNT"
				}
				meas = append(meas, AIExplorerMeasureSelection{FieldID: f.ID, Agg: agg})
				measMap[f.ID] = true
			} else if f.Category == AIExplorerCategoryTime && len(timeDims) == 0 {
				timeDims = append(timeDims, AIExplorerTimeDimensionSelection{FieldID: f.ID, Granularity: "month"})
			}
		}
	}

	// Extract Year Filter
	reYear := regexp.MustCompile(`\b(20\d{2})\b`)
	if match := reYear.FindString(lastPrompt); match != "" {
		filters = append(filters, AIExplorerFilter{
			ID:          fmt.Sprintf("f-%d", time.Now().UnixNano()),
			FieldID:     "year",
			Operator:    "=",
			Value:       match,
			IsParameter: true,
		})
	}

	// Extract Status Filter
	if strings.Contains(lastPrompt, "active") {
		filters = append(filters, AIExplorerFilter{
			ID:       fmt.Sprintf("f-%d", time.Now().UnixNano()+1),
			FieldID:  "status",
			Operator: "=",
			Value:    "Active",
		})
	}

	suggestedChart := "table"
	if len(timeDims) > 0 && len(meas) > 0 {
		suggestedChart = "line"
	} else if len(dims) == 0 && len(meas) == 1 {
		suggestedChart = "kpi"
	} else if len(dims) == 1 && len(meas) >= 1 {
		suggestedChart = "bar"
	}

	desc := "Synthesized query based on natural language terms."
	if lastPrompt != "" {
		desc = fmt.Sprintf("Query matching: \"%s\"", lastPrompt)
	}

	return &AIExplorerCompletionResponse{
		AssistantMessage: desc,
		GeneratedQuery: AIExplorerQueryDefinition{
			Title:          desc,
			Dimensions:     dims,
			Measures:       meas,
			TimeDimensions: timeDims,
			Filters:        filters,
			Limit:          500,
			SuggestedChart: suggestedChart,
		},
		SuggestedFollowUps: []string{
			"Break down by Region",
			"Filter for Institutional clients only",
			"Compare performance across quarters",
		},
		InsightSummary: "Deterministic fallback applied. Dimensions and measures aligned with catalog.",
	}
}

func (s *AIExplorerService) ProcessChatTurn(ctx context.Context, req AIExplorerCompletionRequest) (*AIExplorerCompletionResponse, error) {
	lastPrompt := ""
	if len(req.Messages) > 0 {
		lastPrompt = req.Messages[len(req.Messages)-1].Content
	}

	// 1. Intercept: Check Semantic Cache for initial turns (< 50ms)
	if len(req.Messages) <= 1 && s.Cache != nil && lastPrompt != "" {
		tenantID := req.UserTenantID
		if tenantID == "" {
			tenantID = "default"
		}
		cachedQuery, err := s.Cache.CheckCache(ctx, tenantID, lastPrompt)
		if err == nil && cachedQuery != nil {
			return AICompletionResponseToExplorer(cachedQuery, lastPrompt), nil
		}
	}

	var systemPrompt string
	if s.ContextBuilder != nil {
		systemPrompt = s.ContextBuilder.BuildAugmentedSystemPrompt(
			ctx,
			req.UserTenantID,
			req.UserAccountID,
			req.Catalog,
			req.CurrentQuery,
			lastPrompt,
		)
	} else {
		systemPrompt = s.BuildSystemPrompt(req.Catalog, req.CurrentQuery, req.UserAccountID, req.UserTenantID)
	}

	// 2. Try Gemini API if configured
	if s.GeminiKey != "" {
		if geminiRes, err := s.ProcessWithGemini(ctx, req, systemPrompt); err == nil && geminiRes != nil {
			if s.Cache != nil && len(req.Messages) <= 1 && lastPrompt != "" {
				tenantID := req.UserTenantID
				if tenantID == "" {
					tenantID = "default"
				}
				go s.Cache.SetCache(context.Background(), tenantID, lastPrompt, geminiRes.GeneratedQuery)
			}
			return geminiRes, nil
		}
	}

	// 3. Try OpenAI API if configured
	if s.OpenAIKey != "" {
		if openAIRes, err := s.ProcessWithOpenAI(ctx, req, systemPrompt); err == nil && openAIRes != nil {
			if s.Cache != nil && len(req.Messages) <= 1 && lastPrompt != "" {
				tenantID := req.UserTenantID
				if tenantID == "" {
					tenantID = "default"
				}
				go s.Cache.SetCache(context.Background(), tenantID, lastPrompt, openAIRes.GeneratedQuery)
			}
			return openAIRes, nil
		}
	}

	// 4. Deterministic fallback
	return s.FallbackRuleEngine(req), nil
}

func (s *AIExplorerService) ProcessWithGemini(ctx context.Context, req AIExplorerCompletionRequest, systemPrompt string) (*AIExplorerCompletionResponse, error) {
	if s.GeminiKey == "" {
		return nil, fmt.Errorf("gemini key not configured")
	}

	var chatHistory strings.Builder
	for _, m := range req.Messages {
		chatHistory.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(m.Role), m.Content))
	}

	prompt := fmt.Sprintf(`%s

CONVERSATION HISTORY:
%s

Respond with a JSON object strictly matching this schema:
{
  "intentDescription": "Executive explanation of what this query shows and direct answer to user question",
  "mutation_intent": "new_query",
  "dimensions": ["dimension_field_id"],
  "measures": [{"fieldId": "measure_field_id", "agg": "SUM"}],
  "timeDimensions": [{"fieldId": "time_field_id", "granularity": "month"}],
  "filters": [{"id": "f-1", "fieldId": "field_id", "operator": "=", "value": "val"}],
  "suggestedChart": "table",
  "suggestedFollowUps": ["Follow up 1", "Follow up 2", "Follow up 3"],
  "ambiguityQuestions": [],
  "insightSummary": "Executive takeaway answering user question directly",
  "topDriver": "driver name or dimension",
  "anomalies": []
}`, systemPrompt, chatHistory.String())

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", s.GeminiModel, s.GeminiKey)

	reqPayload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.1,
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini returned status %d", resp.StatusCode)
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty gemini response")
	}

	rawText := geminiResp.Candidates[0].Content.Parts[0].Text
	jsonStr := extractJSON(rawText)
	if jsonStr == "" {
		jsonStr = rawText
	}

	var parsedArgs struct {
		IntentDescription  string                             `json:"intentDescription"`
		MutationIntent     string                             `json:"mutation_intent"`
		Dimensions         []string                           `json:"dimensions"`
		Measures           []AIExplorerMeasureSelection       `json:"measures"`
		TimeDimensions     []AIExplorerTimeDimensionSelection `json:"timeDimensions"`
		Filters            []AIExplorerFilter                 `json:"filters"`
		SuggestedChart     string                             `json:"suggestedChart"`
		SuggestedFollowUps []string                           `json:"suggestedFollowUps"`
		AmbiguityQuestions []string                           `json:"ambiguityQuestions"`
		InsightSummary     string                             `json:"insightSummary"`
		TopDriver          string                             `json:"topDriver"`
		Anomalies          []string                           `json:"anomalies"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsedArgs); err != nil {
		return nil, fmt.Errorf("failed to parse gemini json: %w", err)
	}

	chart := parsedArgs.SuggestedChart
	if chart == "" {
		chart = "table"
	}

	generatedQuery := AIExplorerQueryDefinition{
		Title:          parsedArgs.IntentDescription,
		Dimensions:     parsedArgs.Dimensions,
		Measures:       parsedArgs.Measures,
		TimeDimensions: parsedArgs.TimeDimensions,
		Filters:        parsedArgs.Filters,
		Limit:          500,
		SuggestedChart: chart,
	}

	if generatedQuery.Title == "" {
		generatedQuery.Title = "Query generated by Gemini"
	}

	_ = s.ValidateQuery(&generatedQuery, req.Catalog)

	mutationIntent := parsedArgs.MutationIntent
	if mutationIntent == "" {
		mutationIntent = "new_query"
	}

	return &AIExplorerCompletionResponse{
		AssistantMessage:   parsedArgs.IntentDescription,
		GeneratedQuery:     generatedQuery,
		SuggestedFollowUps: parsedArgs.SuggestedFollowUps,
		AmbiguityQuestions: parsedArgs.AmbiguityQuestions,
		InsightSummary:     parsedArgs.InsightSummary,
		TopDriver:          parsedArgs.TopDriver,
		Anomalies:          parsedArgs.Anomalies,
		IsCacheHit:         false,
		MutationIntent:     mutationIntent,
	}, nil
}

func (s *AIExplorerService) ProcessWithOpenAI(ctx context.Context, req AIExplorerCompletionRequest, systemPrompt string) (*AIExplorerCompletionResponse, error) {
	tool := s.BuildToolDefinition()

	openAIMsgs := []openAIMessage{
		{Role: "system", Content: systemPrompt},
	}

	for _, m := range req.Messages {
		openAIMsgs = append(openAIMsgs, openAIMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	payload := openAIChatRequest{
		Model:       s.ModelName,
		Messages:    openAIMsgs,
		Tools:       []openAITool{tool},
		ToolChoice:  map[string]interface{}{"type": "function", "function": map[string]string{"name": "generate_semantic_query"}},
		Temperature: 0.1,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.OpenAIBaseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.OpenAIKey)

	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned status %d", resp.StatusCode)
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 || len(chatResp.Choices[0].Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("empty openai tool call response")
	}

	toolCall := chatResp.Choices[0].Message.ToolCalls[0]
	if toolCall.Function.Name != "generate_semantic_query" {
		return nil, fmt.Errorf("unexpected tool call name: %s", toolCall.Function.Name)
	}

	var parsedArgs struct {
		IntentDescription  string                             `json:"intentDescription"`
		MutationIntent     string                             `json:"mutation_intent"`
		Dimensions         []string                           `json:"dimensions"`
		Measures           []AIExplorerMeasureSelection       `json:"measures"`
		TimeDimensions     []AIExplorerTimeDimensionSelection `json:"timeDimensions"`
		Filters            []AIExplorerFilter                 `json:"filters"`
		SuggestedChart     string                             `json:"suggestedChart"`
		SuggestedFollowUps []string                           `json:"suggestedFollowUps"`
		AmbiguityQuestions []string                           `json:"ambiguityQuestions"`
		InsightSummary     string                             `json:"insightSummary"`
		TopDriver          string                             `json:"topDriver"`
		Anomalies          []string                           `json:"anomalies"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &parsedArgs); err != nil {
		return nil, err
	}

	generatedQuery := AIExplorerQueryDefinition{
		Title:          parsedArgs.IntentDescription,
		Dimensions:     parsedArgs.Dimensions,
		Measures:       parsedArgs.Measures,
		TimeDimensions: parsedArgs.TimeDimensions,
		Filters:        parsedArgs.Filters,
		Limit:          500,
		SuggestedChart: parsedArgs.SuggestedChart,
	}

	if err := s.ValidateQuery(&generatedQuery, req.Catalog); err != nil {
		return nil, err
	}

	mutationIntent := parsedArgs.MutationIntent
	if mutationIntent == "" {
		mutationIntent = "new_query"
	}

	return &AIExplorerCompletionResponse{
		AssistantMessage:   parsedArgs.IntentDescription,
		GeneratedQuery:     generatedQuery,
		SuggestedFollowUps: parsedArgs.SuggestedFollowUps,
		AmbiguityQuestions: parsedArgs.AmbiguityQuestions,
		InsightSummary:     parsedArgs.InsightSummary,
		TopDriver:          parsedArgs.TopDriver,
		Anomalies:          parsedArgs.Anomalies,
		IsCacheHit:         false,
		MutationIntent:     mutationIntent,
	}, nil
}

func AICompletionResponseToExplorer(cachedQuery *AIExplorerQueryDefinition, prompt string) *AIExplorerCompletionResponse {
	return &AIExplorerCompletionResponse{
		AssistantMessage:   fmt.Sprintf("Recognized query from semantic cache: \"%s\"", prompt),
		GeneratedQuery:     *cachedQuery,
		SuggestedFollowUps: []string{"Break down further", "Filter for specific status", "Compare against previous period"},
		InsightSummary:     "Instant response served from Vector Semantic Cache (< 50ms).",
		IsCacheHit:         true,
	}
}

func HandleAIQueryCompletion(service *AIExplorerService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req AIExplorerCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		result, err := service.ProcessChatTurn(r.Context(), req)
		if err != nil {
			http.Error(w, "AI synthesis failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
