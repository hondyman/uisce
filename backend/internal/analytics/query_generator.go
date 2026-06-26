package analytics

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/pkg/llm"
)

// QueryGenerator handles the generation of structured queries (SQL, Metrics) from natural language.
type QueryGenerator struct {
	llmProvider llm.LLMProvider
}

// NewQueryGenerator creates a new QueryGenerator.
func NewQueryGenerator(llmProvider llm.LLMProvider) *QueryGenerator {
	return &QueryGenerator{
		llmProvider: llmProvider,
	}
}

// GenerateSQL converts a natural language question into a SQL query based on the provided schema context.
func (g *QueryGenerator) GenerateSQL(ctx context.Context, question string, schemaContext string) (string, error) {
	prompt := fmt.Sprintf(`
SYSTEM: You are an expert SQL generator. Convert the user's question into a valid PostgreSQL query.
- Use ONLY the tables and columns defined in the context.
- Do not hallucinate tables or columns.
- Return ONLY the SQL query. Do not include markdown formatting or explanations.

CONTEXT:
%s

USER: %s
`, schemaContext, question)

	response, err := g.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Clean up response (remove markdown code blocks if present)
	sql := strings.TrimSpace(response)
	sql = strings.TrimPrefix(sql, "```sql")
	sql = strings.TrimPrefix(sql, "```")
	sql = strings.TrimSuffix(sql, "```")
	return strings.TrimSpace(sql), nil
}

// GenerateMetric converts a natural language question into a semantic metric query.
// This is a placeholder for future semantic layer integration.
func (g *QueryGenerator) GenerateMetric(ctx context.Context, question string, metricContext string) (string, error) {
	prompt := fmt.Sprintf(`
SYSTEM: You are an expert semantic layer query generator. Convert the user's question into a metric query.
CONTEXT:
%s

USER: %s
`, metricContext, question)

	return g.llmProvider.GenerateResponse(ctx, prompt)
}
