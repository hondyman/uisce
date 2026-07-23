package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
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

	resp, err := s.agent.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, fmt.Errorf("empty response from LLM")
	}

	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			responseText += string(txt)
		}
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
