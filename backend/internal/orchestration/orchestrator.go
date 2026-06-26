package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/calcengine"
	"github.com/hondyman/semlayer/backend/internal/financial"
	"github.com/hondyman/semlayer/backend/internal/pricing"
)

// Intent represents a detected user intent
type Intent struct {
	Type       IntentType             `json:"type"`
	Entities   map[string]string      `json:"entities"` // e.g., {"ticker": "MSFT", "metric": "NAV"}
	Confidence float64                `json:"confidence"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// IntentType defines types of user intents
type IntentType string

const (
	IntentCatalogQuery   IntentType = "catalog_query"   // Standard catalog lookup
	IntentPricingQuery   IntentType = "pricing_query"   // Real-time price lookup
	IntentCalculation    IntentType = "calculation"     // Calculation with external data
	IntentFactorExposure IntentType = "factor_exposure" // Factor analysis
	IntentRiskAnalysis   IntentType = "risk_analysis"   // VaR, stress tests
	IntentPerformance    IntentType = "performance"     // Performance attribution
)

// IntentDetector analyzes questions to determine intent
type IntentDetector struct{}

// NewIntentDetector creates a new intent detector
func NewIntentDetector() *IntentDetector {
	return &IntentDetector{}
}

// Detect analyzes a question and returns detected intents
func (d *IntentDetector) Detect(question string) []Intent {
	questionLower := strings.ToLower(question)
	var intents []Intent

	// Pricing query patterns
	pricingPatterns := []string{
		"current price", "latest price", "price of", "what is .* trading at",
		"quote for", "market price", "spot price", "fx rate",
	}
	for _, pattern := range pricingPatterns {
		if matched, _ := regexp.MatchString(pattern, questionLower); matched {
			intent := Intent{
				Type:       IntentPricingQuery,
				Confidence: 0.85,
				Entities:   d.extractTickers(question),
			}
			intents = append(intents, intent)
			break
		}
	}

	// NAV calculation patterns
	navPatterns := []string{
		"nav", "net asset value", "portfolio value", "portfolio worth",
		"total value", "current value of portfolio",
	}
	for _, pattern := range navPatterns {
		if strings.Contains(questionLower, pattern) {
			intent := Intent{
				Type:       IntentCalculation,
				Confidence: 0.9,
				Entities:   map[string]string{"metric": "NAV"},
			}
			intents = append(intents, intent)
			break
		}
	}

	// VaR patterns
	varPatterns := []string{
		"var", "value at risk", "risk exposure", "downside risk",
	}
	for _, pattern := range varPatterns {
		if strings.Contains(questionLower, pattern) {
			intent := Intent{
				Type:       IntentRiskAnalysis,
				Confidence: 0.85,
				Entities:   map[string]string{"metric": "VaR"},
			}
			intents = append(intents, intent)
			break
		}
	}

	// Factor exposure patterns
	factorPatterns := []string{
		"factor exposure", "factor loadings", "beta", "market exposure",
		"style factors", "fama french",
	}
	for _, pattern := range factorPatterns {
		if strings.Contains(questionLower, pattern) {
			intent := Intent{
				Type:       IntentFactorExposure,
				Confidence: 0.9,
				Entities:   map[string]string{},
			}
			intents = append(intents, intent)
			break
		}
	}

	// Default to catalog query if no specific intent detected
	if len(intents) == 0 {
		intents = append(intents, Intent{
			Type:       IntentCatalogQuery,
			Confidence: 0.5,
			Entities:   map[string]string{},
		})
	}

	return intents
}

// extractTickers extracts ticker symbols from question
func (d *IntentDetector) extractTickers(question string) map[string]string {
	entities := make(map[string]string)

	// Simple pattern: uppercase 1-5 letter words (common ticker format)
	tickerPattern := regexp.MustCompile(`\b([A-Z]{1,5})\b`)
	matches := tickerPattern.FindAllString(question, -1)

	if len(matches) > 0 {
		entities["ticker"] = matches[0]
	}

	return entities
}

// Orchestrator coordinates between intent detection, tool calling, and response composition
type Orchestrator struct {
	intentDetector  *IntentDetector
	pricingProvider pricing.PricingProvider
	calcEngine      calcengine.CalcEngine
	financialTools  *financial.ToolRegistry
}

// NewOrchestrator creates a new orchestration service
func NewOrchestrator(
	pricingProvider pricing.PricingProvider,
	calcEngine calcengine.CalcEngine,
	financialTools *financial.ToolRegistry,
) *Orchestrator {
	return &Orchestrator{
		intentDetector:  NewIntentDetector(),
		pricingProvider: pricingProvider,
		calcEngine:      calcEngine,
		financialTools:  financialTools,
	}
}

// OrchestratedResponse contains the full response with all execution details
type OrchestratedResponse struct {
	Intents     []Intent                 `json:"intents"`
	ToolCalls   []ToolCall               `json:"tool_calls"`
	CalcResults []*calcengine.CalcResult `json:"calc_results,omitempty"`
	PriceQuotes []pricing.PriceQuote     `json:"price_quotes,omitempty"`
	Answer      string                   `json:"answer"`
	Sources     []string                 `json:"sources"`
	Confidence  string                   `json:"confidence"`
}

// ToolCall represents an external tool invocation
type ToolCall struct {
	Tool       string                 `json:"tool"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"result"`
	Error      string                 `json:"error,omitempty"`
}

// Process orchestrates the full flow from question to answer
func (o *Orchestrator) Process(ctx context.Context, question string, inputs map[string]interface{}) (*OrchestratedResponse, error) {
	response := &OrchestratedResponse{
		ToolCalls: []ToolCall{},
		Sources:   []string{},
	}

	// 1. Detect intents
	intents := o.intentDetector.Detect(question)
	response.Intents = intents

	// 2. Route to appropriate handlers based on intent
	for _, intent := range intents {
		switch intent.Type {
		case IntentPricingQuery:
			if err := o.handlePricingQuery(ctx, intent, inputs, response); err != nil {
				return nil, err
			}

		case IntentCalculation:
			if err := o.handleCalculation(ctx, intent, inputs, response); err != nil {
				return nil, err
			}

		case IntentFactorExposure, IntentRiskAnalysis:
			if err := o.handleFinancialTool(ctx, intent, inputs, response); err != nil {
				return nil, err
			}
		}
	}

	// 3. Compose natural language answer
	o.composeAnswer(response)

	// 4. Set confidence based on data quality
	response.Confidence = o.calculateConfidence(response)

	return response, nil
}

// handlePricingQuery fetches prices from external provider
func (o *Orchestrator) handlePricingQuery(ctx context.Context, intent Intent, inputs map[string]interface{}, response *OrchestratedResponse) error {
	ticker, ok := intent.Entities["ticker"]
	if !ok {
		return fmt.Errorf("no ticker found in pricing query")
	}

	price, err := o.pricingProvider.GetPrice(ctx, ticker)
	if err != nil {
		response.ToolCalls = append(response.ToolCalls, ToolCall{
			Tool:       "pricing_provider",
			Parameters: map[string]interface{}{"ticker": ticker},
			Error:      err.Error(),
		})
		return err
	}

	quote := pricing.PriceQuote{
		Ticker: ticker,
		Price:  price,
		Source: o.pricingProvider.Name(),
	}

	response.PriceQuotes = append(response.PriceQuotes, quote)
	response.ToolCalls = append(response.ToolCalls, ToolCall{
		Tool:       "pricing_provider",
		Parameters: map[string]interface{}{"ticker": ticker},
		Result:     quote,
	})
	response.Sources = append(response.Sources, ticker)

	return nil
}

// handleCalculation runs calculation engine
func (o *Orchestrator) handleCalculation(ctx context.Context, intent Intent, inputs map[string]interface{}, response *OrchestratedResponse) error {
	metric := intent.Entities["metric"]

	result, err := o.calcEngine.Run(ctx, metric, inputs)
	if err != nil {
		response.ToolCalls = append(response.ToolCalls, ToolCall{
			Tool:       "calc_engine",
			Parameters: map[string]interface{}{"metric": metric},
			Error:      err.Error(),
		})
		return err
	}

	response.CalcResults = append(response.CalcResults, result)
	response.ToolCalls = append(response.ToolCalls, ToolCall{
		Tool:       "calc_engine",
		Parameters: map[string]interface{}{"metric": metric},
		Result:     result,
	})
	response.Sources = append(response.Sources, result.Sources...)

	return nil
}

// handleFinancialTool calls financial calculation tools
func (o *Orchestrator) handleFinancialTool(ctx context.Context, intent Intent, inputs map[string]interface{}, response *OrchestratedResponse) error {
	var toolName string

	switch intent.Type {
	case IntentFactorExposure:
		toolName = "calculate_factor_exposure"
	case IntentRiskAnalysis:
		toolName = "calculate_var"
	default:
		return fmt.Errorf("unsupported financial tool intent: %s", intent.Type)
	}

	tool, ok := o.financialTools.Get(ctx, toolName)
	if !ok {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	paramsJSON, _ := json.Marshal(inputs)
	result, err := tool.Execute(ctx, paramsJSON)
	if err != nil {
		response.ToolCalls = append(response.ToolCalls, ToolCall{
			Tool:       toolName,
			Parameters: inputs,
			Error:      err.Error(),
		})
		return err
	}

	response.ToolCalls = append(response.ToolCalls, ToolCall{
		Tool:       toolName,
		Parameters: inputs,
		Result:     result,
	})

	return nil
}

// composeAnswer generates natural language answer from results
func (o *Orchestrator) composeAnswer(response *OrchestratedResponse) {
	var parts []string

	// Add price quotes
	for _, quote := range response.PriceQuotes {
		parts = append(parts, fmt.Sprintf("%s is trading at $%.2f (source: %s)",
			quote.Ticker, quote.Price, quote.Source))
	}

	// Add calculation results
	for _, result := range response.CalcResults {
		parts = append(parts, fmt.Sprintf("%s: $%.2f", result.Metric, result.Value))

		if len(result.Breakdown) > 0 {
			parts = append(parts, "Breakdown:")
			for _, item := range result.Breakdown {
				if holding, ok := item["holding"].(string); ok {
					if value, ok := item["value_usd"].(float64); ok {
						parts = append(parts, fmt.Sprintf("  - %s: $%.2f", holding, value))
					}
				}
			}
		}
	}

	// Add tool call results
	for _, call := range response.ToolCalls {
		if call.Tool == "calculate_var" || call.Tool == "calculate_factor_exposure" {
			if _, ok := call.Result.(map[string]interface{}); ok {
				parts = append(parts, fmt.Sprintf("Analysis completed using %s", call.Tool))
			}
		}
	}

	response.Answer = strings.Join(parts, "\n")
}

// calculateConfidence determines overall confidence based on data quality
func (o *Orchestrator) calculateConfidence(response *OrchestratedResponse) string {
	// Check for errors in tool calls
	for _, call := range response.ToolCalls {
		if call.Error != "" {
			return "low"
		}
	}

	// Check if we have sufficient sources
	if len(response.Sources) >= 2 {
		return "high"
	}

	return "medium"
}
