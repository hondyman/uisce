package query

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// LLMClient interface for dependency injection
type LLMClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// QueryParser handles natural language decomposition
type QueryParser struct {
	llmClient LLMClient
}

func NewQueryParser(llmClient LLMClient) *QueryParser {
	return &QueryParser{
		llmClient: llmClient,
	}
}

func (qp *QueryParser) Parse(ctx context.Context, query string, tenantID uuid.UUID) (*QueryUnderstanding, error) {
	// 1. Use LLM to decompose query
	decomposed, err := qp.llmDecompose(ctx, query)
	if err != nil {
		return nil, err
	}

	// 2. Entity Linking (Stubbed)
	// entities := qp.entityLinker.ExtractAndLink(...)
	// decomposed.Entities = entities

	return decomposed, nil
}

func (qp *QueryParser) llmDecompose(ctx context.Context, query string) (*QueryUnderstanding, error) {
	prompt := fmt.Sprintf(`Analyze this wealth management query and extract structured information:
Query: "%s"
Respond ONLY with JSON matching the QueryUnderstanding schema.`, query)

	response, err := qp.llmClient.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var understanding QueryUnderstanding
	// In a real implementation, we'd be more robust about parsing JSON from LLM output
	// For now, assuming the LLM returns clean JSON or we mock it
	if response == "" {
		// Return a mock for testing/stubbing purposes if LLM is not connected
		return &QueryUnderstanding{
			OriginalQuery: query,
			Intent: QueryIntent{PrimaryIntent: "search"},
		}, nil
	}

	err = json.Unmarshal([]byte(response), &understanding)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	understanding.OriginalQuery = query
	return &understanding, nil
}
