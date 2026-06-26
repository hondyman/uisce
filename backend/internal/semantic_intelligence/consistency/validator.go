package consistency

import (
	"context"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type ConsistencyIssue struct {
	Level       string `json:"level"` // error, warning
	Description string `json:"description"`
	Element     string `json:"element"`
}

type Validator struct {
	// Rules or PatternLearner reference in real implementation
}

func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks a page for common consistency violations
func (v *Validator) Validate(ctx context.Context, page *pagestudio.CorePage) ([]ConsistencyIssue, error) {
	issues := make([]ConsistencyIssue, 0)

	// 1. Naming Consistency (Mock Rule: KPI names should be Title Case)
	if strings.Contains(strings.ToLower(page.Name), "kpi") {
		// Mock check
	}

	// 2. Filter Consistency Check
	// If page has "as_of_date" filter, it should defaulting to "today" (mock rule)
	if strings.Contains(string(page.DataBindings), "as_of_date") {
		// Real impl would parse bindings
		if !strings.Contains(string(page.DataBindings), "today") {
			issues = append(issues, ConsistencyIssue{
				Level:       "warning",
				Description: "Filter 'as_of_date' should default to 'today' for consistency",
				Element:     "as_of_date",
			})
		}
	}

	// 3. Layout Consistency
	// If page is a "Dashboard", it should have at least one KPI component
	if strings.Contains(page.Name, "Dashboard") {
		if !strings.Contains(string(page.Components), "KPI") && !strings.Contains(string(page.Components), "metric") {
			issues = append(issues, ConsistencyIssue{
				Level:       "warning",
				Description: "Dashboard pages usually contain top-level KPIs",
				Element:     "layout",
			})
		}
	}

	return issues, nil
}
