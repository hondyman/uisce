package accessibility

import (
	"context"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type Violation struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"` // critical, major, minor
	Description string `json:"description"`
	ComponentID string `json:"component_id,omitempty"`
}

type CleanupSuggestion struct {
	Description string `json:"description"`
	Action      string `json:"action"`
}

type Report struct {
	Score       int                 `json:"score"` // 0-100
	Violations  []Violation         `json:"violations"`
	Suggestions []CleanupSuggestion `json:"suggestions"`
}

type Linter struct{}

func NewLinter() *Linter {
	return &Linter{}
}

func (l *Linter) LintPage(ctx context.Context, page *pagestudio.CorePage) (*Report, error) {
	report := &Report{
		Score:      100,
		Violations: make([]Violation, 0),
	}

	compStr := string(page.Components)

	// 1. Check for basic Aria Labels on buttons if missing text
	if strings.Contains(compStr, "IconButton") && !strings.Contains(compStr, "aria-label") {
		report.Violations = append(report.Violations, Violation{
			RuleID:      "wcag:aria-label",
			Severity:    "critical",
			Description: "Icon Buttons must have aria-label defined.",
		})
		report.Score -= 20
	}

	// 2. Check for Contrast (Mock)
	// Would parse theme colors in real implementation

	return report, nil
}
