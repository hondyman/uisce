package analytics

import (
	"context"
	"strings"
)

// BusinessTermSuggestion represents a suggestion from a matcher.
type BusinessTermSuggestion struct {
	TermName   string
	Confidence float64
	Source     string // e.g., "FIBO", "InternalGlossary"
}

// BusinessTermMatcher defines the interface for a service that can suggest
// business terms for a given semantic term.
type BusinessTermMatcher interface {
	Suggest(ctx context.Context, semanticTermName string) ([]BusinessTermSuggestion, error)
}

// SimpleFIBOMatcher is a mock implementation of the BusinessTermMatcher.
// In a real-world scenario, this would make an API call to an external service
// or query a local knowledge graph like FIBO.
type SimpleFIBOMatcher struct{}

// NewSimpleFIBOMatcher creates a new mock matcher.
func NewSimpleFIBOMatcher() *SimpleFIBOMatcher {
	return &SimpleFIBOMatcher{}
}

// Suggest provides mock suggestions based on simple string matching.
func (m *SimpleFIBOMatcher) Suggest(ctx context.Context, semanticTermName string) ([]BusinessTermSuggestion, error) {
	var suggestions []BusinessTermSuggestion

	// Mock logic: If the term contains "CUSTOMER", suggest related business terms.
	if strings.Contains(strings.ToUpper(semanticTermName), "CUSTOMER") {
		suggestions = append(suggestions, BusinessTermSuggestion{
			TermName:   "CUSTOMER_IDENTIFIER",
			Confidence: 0.9,
			Source:     "FIBO_MATCHER",
		})
	}

	return suggestions, nil
}
