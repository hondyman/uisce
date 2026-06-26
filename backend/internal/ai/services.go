package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/indexing"
	"github.com/hondyman/semlayer/backend/internal/values"
)

// SignalIngestionService analyzes text to extract ESG signals
type SignalIngestionService interface {
	AnalyzeText(ctx context.Context, text string) ([]values.ValueSignal, error)
}

// ConstraintGenerationService converts natural language to constraints
type ConstraintGenerationService interface {
	GenerateConstraints(ctx context.Context, prompt string) ([]values.Constraint, error)
}

// ExplanationService generates human-readable explanations for portfolios
type ExplanationService interface {
	ExplainPortfolio(ctx context.Context, portfolio indexing.Portfolio, constraints []values.Constraint) (string, error)
}

// AnalyzeText extracts signals from text using Gemini
func (s *AIService) AnalyzeText(ctx context.Context, text string) ([]values.ValueSignal, error) {
	prompt := fmt.Sprintf(`
You are an expert ESG analyst. Analyze the following text and extract any ESG signals.
Return a JSON array of objects with the following fields:
- issuer_id: The ticker symbol of the company involved (e.g., "AAPL", "TSLA").
- signal_source_id: Use "GEMINI_ANALYSIS".
- signal_type: One of "CONTROVERSY", "RATING_CHANGE", "NEWS_EVENT".
- score: A number from -100 (very negative) to 100 (very positive).
- summary: A brief summary of the signal.
- evidence_refs: An array of objects with "url" (can be empty), "date" (YYYY-MM-DD), "type" ("TEXT_ANALYSIS"), "summary".

Text:
%s

Output JSON only.
`, text)

	response, err := s.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Clean response (remove markdown code blocks if present)
	cleaned := cleanJSON(response)

	var signals []values.ValueSignal
	if err := json.Unmarshal([]byte(cleaned), &signals); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return signals, nil
}

// GenerateConstraints converts user prompt to constraints
func (s *AIService) GenerateConstraints(ctx context.Context, userPrompt string) ([]values.Constraint, error) {
	prompt := fmt.Sprintf(`
You are a wealth management assistant. Convert the user's natural language request into a list of formal constraints.
Return a JSON array of objects with the following fields:
- name: A short name for the constraint.
- description: A description of what it does.
- operator: "EXCLUDE", "INCLUDE", "OVERWEIGHT", "UNDERWEIGHT".
- scope: An object with "sector" (optional), "region" (optional), "issuer" (optional ticker).
- severity: "HIGH", "MEDIUM", "LOW".

User Request: "%s"

Output JSON only.
`, userPrompt)

	response, err := s.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	cleaned := cleanJSON(response)

	var constraints []values.Constraint
	if err := json.Unmarshal([]byte(cleaned), &constraints); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return constraints, nil
}

// ExplainPortfolio generates a narrative explanation
func (s *AIService) ExplainPortfolio(ctx context.Context, portfolio indexing.Portfolio, constraints []values.Constraint) (string, error) {
	// Serialize inputs for the prompt
	constraintsJSON, _ := json.Marshal(constraints)

	// Summarize holdings for brevity
	holdingsSummary := ""
	for i, h := range portfolio.Holdings {
		if i > 10 {
			break // Limit to top 10 for prompt context window
		}
		holdingsSummary += fmt.Sprintf("- %s: %.2f%%\n", h.Ticker, h.Weight*100)
	}

	prompt := fmt.Sprintf(`
You are a portfolio manager. Explain the client's portfolio composition, highlighting how their values constraints influenced the holdings.
Be professional, clear, and concise.

Constraints:
%s

Top Holdings:
%s

Explanation:
`, string(constraintsJSON), holdingsSummary)

	response, err := s.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}
