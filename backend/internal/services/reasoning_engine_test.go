package services

import (
	"context"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestReasoningEnginePlan(t *testing.T) {
	llmProvider := &testutils.MockLLMProvider{}
	engine := NewReasoningEngine(llmProvider)
	ctx := context.Background()

	t.Run("Generate Plan", func(t *testing.T) {
		question := "How do I calculate revenue?"

		llmProvider.GenerateResponseFunc = func(ctx context.Context, prompt string) (string, error) {
			return "1. Find revenue term\n2. Check dependencies\n3. Explain calculation", nil
		}

		steps, err := engine.Plan(ctx, question)
		require.NoError(t, err)
		require.Len(t, steps, 3)
		require.Equal(t, "1. Find revenue term", steps[0])
		require.Equal(t, "2. Check dependencies", steps[1])
		require.Equal(t, "3. Explain calculation", steps[2])
	})
}
