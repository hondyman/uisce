package analytics

import (
	"context"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/testutils"
	"github.com/stretchr/testify/require"
)

// Using centralized testutils.MockLLMProvider — no local mock needed.

func TestGenerateSQL(t *testing.T) {
	llmProvider := &testutils.MockLLMProvider{}
	generator := NewQueryGenerator(llmProvider)
	ctx := context.Background()

	t.Run("Generate SQL", func(t *testing.T) {
		schema := "Table: users (id, name, email)"
		question := "Count users"

		llmProvider.GenerateResponseFunc = func(ctx context.Context, prompt string) (string, error) {
			return "SELECT COUNT(*) FROM users;", nil
		}

		sql, err := generator.GenerateSQL(ctx, question, schema)
		require.NoError(t, err)
		require.Equal(t, "SELECT COUNT(*) FROM users;", sql)
	})

	t.Run("Generate SQL with Markdown", func(t *testing.T) {
		schema := "Table: users"
		question := "Count users"

		llmProvider.GenerateResponseFunc = func(ctx context.Context, prompt string) (string, error) {
			return "```sql\nSELECT COUNT(*) FROM users;\n```", nil
		}

		sql, err := generator.GenerateSQL(ctx, question, schema)
		require.NoError(t, err)
		require.Equal(t, "SELECT COUNT(*) FROM users;", sql)
	})
}
