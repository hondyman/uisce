package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/pkg/llm"
)

type SemanticAssistant struct {
	llmProvider llm.LLMProvider
}

func NewSemanticAssistant(llmProvider llm.LLMProvider) *SemanticAssistant {
	return &SemanticAssistant{llmProvider: llmProvider}
}

// SuggestTerms analyzes a table schema or description and returns suggested semantic terms
// schemaContext: could be DDL or a list of column names/types
func (s *SemanticAssistant) SuggestTerms(ctx context.Context, schemaContext string) ([]models.SemanticTerm, error) {
	prompt := fmt.Sprintf(`
You are an expert Data Engineer. 
Analyze the following database table schema and suggest 3-5 useful "Calculated Fields" or "Semantic Terms" 
that would be valuable for business analytics (e.g., ratios, time differences, categorizations).

SCHEMA:
%s

Output MUST be a valid JSON array of objects with the following keys:
- "node_name": (string) e.g. "churn_risk_score"
- "description": (string) what it calculates
- "type": (string) "calculated" 
- "expression": (string) the formula using column names (e.g. "revenue - cost")
- "data_type": (string) "number", "string", "boolean"

Do not include markdown formatting. Just the JSON.
`, schemaContext)

	resp, err := s.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Clean up markdown if present
	cleaned := strings.TrimPrefix(resp, "```json")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var suggestions []struct {
		NodeName    string `json:"node_name"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Expression  string `json:"expression"`
		DataType    string `json:"data_type"`
	}

	if err := json.Unmarshal([]byte(cleaned), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse LLM suggestion: %w. Response: %s", err, resp)
	}

	// Map to model
	var result []models.SemanticTerm
	for _, raw := range suggestions {
		term := models.SemanticTerm{
			NodeName:    raw.NodeName,
			Type:        models.SemanticTermType(raw.Type),
			Description: raw.Description,
			DataType:    models.SemanticDataType(raw.DataType),
			Expression:  raw.Expression,
		}
		result = append(result, term)
	}

	return result, nil
}
