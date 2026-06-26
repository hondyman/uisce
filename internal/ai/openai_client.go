package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	OpenAIAPIBase = "https://api.openai.com/v1"
	DefaultModel  = "gpt-4o-mini"
	// Rate limiting & constraints
	DefaultMaxTokens   = 1000
	DefaultTemperature = 0.7
	RequestTimeoutSecs = 30
	MaxRetries         = 3
	InitialBackoffMs   = 500
	MaxBackoffMs       = 10000
)

// OpenAIClient wraps OpenAI API interactions for holiday intelligence
type OpenAIClient struct {
	apiKey      string
	model       string
	maxTokens   int
	httpClient  *http.Client
	logger      *slog.Logger
	cache       map[string]*CachedResponse
	cacheTTL    time.Duration
	callMetrics *CallMetrics
}

// CachedResponse stores API response with timestamp
type CachedResponse struct {
	Data      interface{}
	ExpiresAt time.Time
}

// CallMetrics tracks API usage
type CallMetrics struct {
	TotalCalls      int64
	SuccessfulCalls int64
	FailedCalls     int64
	TotalTokens     int64
	EstimatedCost   float64 // USD
}

// HolidayGenerationRequest defines input for holiday generation
type HolidayGenerationRequest struct {
	Region          string
	Country         string
	Industry        string
	Language        string
	Year            int
	ExcludeHistoric bool
	IncludeRegional bool
	MaxSuggestions  int
}

// GeneratedHoliday represents AI-suggested holiday
type GeneratedHoliday struct {
	Name             string  `json:"name"`
	DateStart        string  `json:"date_start"` // YYYY-MM-DD
	DateEnd          string  `json:"date_end"`
	HolidayType      string  `json:"holiday_type"` // national, regional, cultural
	Confidence       float64 `json:"confidence"`   // 0.0-1.0
	Reason           string  `json:"reason"`
	IsRecurring      bool    `json:"is_recurring"`
	RecurringPattern string  `json:"recurring_pattern"` // annual, monthly, etc.
}

// HolidayGenerationResponse wraps AI response
type HolidayGenerationResponse struct {
	Holidays   []GeneratedHoliday `json:"holidays"`
	Confidence float64            `json:"overall_confidence"`
	Notes      string             `json:"notes"`
}

// ConflictAnalysisRequest defines input for conflict detection
type ConflictAnalysisRequest struct {
	Holidays     []GeneratedHoliday
	ExistingJobs []JobDetail
	Profiles     []ProfileAvailability
}

// ConflictResult represents detected conflict
type ConflictResult struct {
	HolidayName      string
	ConflictType     string // overlap, capacity, resource, provider_conflict
	Severity         string // low, medium, high, critical
	Description      string
	AffectedProfiles []string
	Recommendation   string
	Confidence       float64
}

// JobDetail minimal job data for conflict analysis
type JobDetail struct {
	ID               string
	ProfileID        string
	StartTime        time.Time
	EndTime          time.Time
	Status           string
	RequiredCapacity int
}

// ProfileAvailability capacity info
type ProfileAvailability struct {
	ID             string
	Name           string
	MaxCapacity    int
	CurrentLoad    int
	Region         string
	MarketCalendar string
}

// OpenAI API request/response types
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Usage   openAIUsage    `json:"usage"`
	Choices []openAIChoice `json:"choices"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewOpenAIClient creates configured OpenAI client
func NewOpenAIClient(apiKey string, logger *slog.Logger) *OpenAIClient {
	if apiKey == "" {
		panic("OpenAI API key required")
	}

	return &OpenAIClient{
		apiKey:    apiKey,
		model:     DefaultModel,
		maxTokens: DefaultMaxTokens,
		httpClient: &http.Client{
			Timeout: RequestTimeoutSecs * time.Second,
		},
		logger:   logger,
		cache:    make(map[string]*CachedResponse),
		cacheTTL: 24 * time.Hour, // 24-hour cache for holiday suggestions
		callMetrics: &CallMetrics{
			TotalCalls:      0,
			SuccessfulCalls: 0,
			FailedCalls:     0,
			TotalTokens:     0,
			EstimatedCost:   0,
		},
	}
}

// GenerateHolidaysForRegion generates holiday suggestions via AI
func (c *OpenAIClient) GenerateHolidaysForRegion(ctx context.Context, req HolidayGenerationRequest) (*HolidayGenerationResponse, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("holidays:%s:%s:%d", req.Region, req.Country, req.Year)
	if cached := c.getCached(cacheKey); cached != nil {
		c.logger.Info("holiday generation cache hit", "region", req.Region, "year", req.Year)
		return cached.(*HolidayGenerationResponse), nil
	}

	// Build prompt
	systemPrompt := `You are an expert in holiday and calendar management. Generate a JSON list of official holidays for the given region and year.
Requirements:
- Return valid JSON array of objects
- Each object must include: name, date_start, date_end (YYYY-MM-DD format), holiday_type, confidence (0.0-1.0), reason
- Include both national and regional holidays if requested
- Confidence should reflect certainty of the holiday's official status
- Focus on government-recognized and widely-observed holidays`

	userPrompt := fmt.Sprintf(`Generate holidays for %s (%s) in %d.
Industry context: %s
Language: %s
Include regional holidays: %v
Exclude historic/obsolete holidays: %v
Maximum suggestions: %d

Return JSON with structure:
{
  "holidays": [{
    "name": "Holiday Name",
    "date_start": "2025-01-01",
    "date_end": "2025-01-01",
    "holiday_type": "national",
    "confidence": 0.95,
    "reason": "Official national holiday",
    "is_recurring": true,
    "recurring_pattern": "annual"
  }],
  "overall_confidence": 0.92,
  "notes": "Summary of holidays generated"
}`,
		req.Country, req.Region, req.Year,
		req.Industry, req.Language,
		req.IncludeRegional, req.ExcludeHistoric, req.MaxSuggestions)

	response, err := c.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		c.logger.Error("holiday generation failed", "region", req.Region, "error", err)
		c.callMetrics.FailedCalls++
		return nil, err
	}

	// Parse response
	var result HolidayGenerationResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		c.logger.Error("failed to parse holiday response", "error", err)
		return nil, fmt.Errorf("invalid response format: %w", err)
	}

	// Cache successful response
	c.setCached(cacheKey, &result)
	c.logger.Info("generated holidays", "region", req.Region, "count", len(result.Holidays))
	c.callMetrics.SuccessfulCalls++

	return &result, nil
}

// DetectHolidayConflicts analyzes conflicts between holidays and existing schedule
func (c *OpenAIClient) DetectHolidayConflicts(ctx context.Context, req ConflictAnalysisRequest) ([]ConflictResult, error) {
	if len(req.Holidays) == 0 || len(req.ExistingJobs) == 0 {
		return []ConflictResult{}, nil // No conflicts if no data
	}

	systemPrompt := `You are an expert in capacity planning and scheduling. Analyze holidays against existing jobs/schedules for conflicts.
Conflict types: overlap, capacity, resource, provider_conflict, external
Severity levels: low, medium, high, critical`

	// Build data for analysis
	holidaysJSON, _ := json.Marshal(req.Holidays)
	jobsJSON, _ := json.Marshal(req.ExistingJobs)
	profilesJSON, _ := json.Marshal(req.Profiles)

	userPrompt := fmt.Sprintf(`Analyze conflicts between these holidays and existing jobs:

Holidays:
%s

Existing Jobs:
%s

Profiles (Capacity):
%s

Return JSON with structure:
{
  "conflicts": [{
    "holiday_name": "Holiday Name",
    "conflict_type": "overlap",
    "severity": "high",
    "description": "Details of conflict",
    "affected_profiles": ["profile1", "profile2"],
    "recommendation": "Suggested resolution",
    "confidence": 0.85
  }],
  "summary": "Overall conflict analysis"
}`,
		string(holidaysJSON), string(jobsJSON), string(profilesJSON))

	response, err := c.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		c.logger.Error("conflict detection failed", "error", err)
		c.callMetrics.FailedCalls++
		return nil, err
	}

	// Parse response
	var result struct {
		Conflicts []ConflictResult `json:"conflicts"`
		Summary   string           `json:"summary"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		c.logger.Error("failed to parse conflict response", "error", err)
		return nil, err
	}

	c.logger.Info("detected conflicts", "count", len(result.Conflicts), "summary", result.Summary)
	c.callMetrics.SuccessfulCalls++
	return result.Conflicts, nil
}

// callOpenAI makes actual HTTP request to OpenAI API with retry logic
func (c *OpenAIClient) callOpenAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	var lastErr error
	backoff := time.Duration(InitialBackoffMs) * time.Millisecond

	for attempt := 0; attempt < MaxRetries; attempt++ {
		response, err := c.sendRequest(ctx, systemPrompt, userPrompt)
		if err == nil {
			return response, nil
		}

		lastErr = err
		c.logger.Warn("OpenAI request failed, retrying", "attempt", attempt+1, "error", err)

		// Exponential backoff
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return "", ctx.Err()
		}

		backoff = time.Duration(float64(backoff) * 1.5)
		if backoff > time.Duration(MaxBackoffMs)*time.Millisecond {
			backoff = time.Duration(MaxBackoffMs) * time.Millisecond
		}
	}

	return "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

// sendRequest sends single HTTP request to OpenAI
func (c *OpenAIClient) sendRequest(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	c.callMetrics.TotalCalls++

	req := openAIRequest{
		Model:       c.model,
		MaxTokens:   c.maxTokens,
		Temperature: DefaultTemperature,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", OpenAIAPIBase+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var openaiResp openAIResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if openaiResp.Error != nil {
		return "", fmt.Errorf("api error [%s]: %s", openaiResp.Error.Code, openaiResp.Error.Message)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// Update metrics
	c.callMetrics.TotalTokens += int64(openaiResp.Usage.TotalTokens)
	c.callMetrics.EstimatedCost = float64(c.callMetrics.TotalTokens) * 0.00000015 // gpt-4o-mini: ~$0.15/1M tokens

	return openaiResp.Choices[0].Message.Content, nil
}

// Cache management
func (c *OpenAIClient) getCached(key string) interface{} {
	if cached, exists := c.cache[key]; exists && time.Now().Before(cached.ExpiresAt) {
		return cached.Data
	}
	delete(c.cache, key)
	return nil
}

func (c *OpenAIClient) setCached(key string, data interface{}) {
	c.cache[key] = &CachedResponse{
		Data:      data,
		ExpiresAt: time.Now().Add(c.cacheTTL),
	}
}

// ClearCache removes expired entries
func (c *OpenAIClient) ClearCache() {
	now := time.Now()
	for key, cached := range c.cache {
		if now.After(cached.ExpiresAt) {
			delete(c.cache, key)
		}
	}
}

// SetModel updates the AI model to use
func (c *OpenAIClient) SetModel(model string) {
	c.model = model
	c.logger.Info("updated model", "model", model)
}

// SetMaxTokens updates token limit
func (c *OpenAIClient) SetMaxTokens(tokens int) {
	c.maxTokens = tokens
}

// GetMetrics returns current usage metrics
func (c *OpenAIClient) GetMetrics() CallMetrics {
	return *c.callMetrics
}

// ResetMetrics resets usage counters
func (c *OpenAIClient) ResetMetrics() {
	c.callMetrics = &CallMetrics{}
}
