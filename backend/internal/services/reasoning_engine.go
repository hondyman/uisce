package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/pkg/llm"
)

// ReasoningEngine orchestrates the Chain-of-Thought process for complex questions.
type ReasoningEngine struct {
	llmProvider llm.LLMProvider
}

// NewReasoningEngine creates a new ReasoningEngine.
func NewReasoningEngine(llmProvider llm.LLMProvider) *ReasoningEngine {
	return &ReasoningEngine{
		llmProvider: llmProvider,
	}
}

// Plan generates a step-by-step plan for answering the question.
func (e *ReasoningEngine) Plan(ctx context.Context, question string) ([]string, error) {
	prompt := fmt.Sprintf(`
SYSTEM: You are an expert data analyst planner.
USER: Create a step-by-step plan to answer the following question using a data catalog and SQL database.
Question: "%s"
Output format: A numbered list of steps. Do not include any other text.
`, question)

	response, err := e.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// Parse the response into a slice of strings
	lines := strings.Split(response, "\n")
	var steps []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			steps = append(steps, trimmed)
		}
	}
	return steps, nil
}

// Synthesize combines information to answer the original question.
func (e *ReasoningEngine) Synthesize(ctx context.Context, question string, contextInfo string) (string, error) {
	prompt := fmt.Sprintf(`
SYSTEM: You are an expert data analyst. Answer the user's question based on the provided context.
CONTEXT:
%s

USER: %s
`, contextInfo, question)

	return e.llmProvider.GenerateResponse(ctx, prompt)
}
