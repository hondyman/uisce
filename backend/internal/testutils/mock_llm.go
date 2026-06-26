package testutils

import "context"

// MockLLMProvider is a small shared mock implementation of an LLM provider used in tests.
// It exposes function hooks so tests can customize behavior.
type MockLLMProvider struct {
	GenerateResponseFunc func(ctx context.Context, prompt string) (string, error)
	EmbedFunc            func(ctx context.Context, text string) ([]float32, error)
}

func (m *MockLLMProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	if m == nil {
		return "", nil
	}
	if m.GenerateResponseFunc != nil {
		return m.GenerateResponseFunc(ctx, prompt)
	}
	return "Mock Response", nil
}

func (m *MockLLMProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if m == nil {
		return nil, nil
	}
	if m.EmbedFunc != nil {
		return m.EmbedFunc(ctx, text)
	}
	// Default to a small deterministic embedding for tests
	return []float32{0.1, 0.2, 0.3, 0.4}, nil
}
