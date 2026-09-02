package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// FinancialEntity represents an entity extracted from text
type FinancialEntity struct {
	Name string `json:"name"`
	Type string `json:"type"` // e.g., "TICKER", "PERSON", "COMPANY"
}

// NERService extracts financial entities from text
type NERService struct {
	agent *IPSComplianceAgent // Reusing the agent wrapper for convenience
}

func NewNERService(agent *IPSComplianceAgent) *NERService {
	return &NERService{agent: agent}
}

// ExtractEntities identifies key financial entities in the text
func (s *NERService) ExtractEntities(ctx context.Context, text string) ([]FinancialEntity, error) {
	prompt := fmt.Sprintf(`
Extract all financial entities (Tickers, Company Names, Key Personnel) from the following text.
Return a JSON array of objects with "name" and "type" fields.
Types should be one of: "TICKER", "COMPANY", "PERSON".

Text:
%s
`, text)

	responseText, err := s.agent.provider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}
	if responseText == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	var entities []FinancialEntity
	if err := json.Unmarshal([]byte(responseText), &entities); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return entities, nil
}
