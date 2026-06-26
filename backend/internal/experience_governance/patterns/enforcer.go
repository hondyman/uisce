package patterns

import (
	"context"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type Enforcer struct{}

func NewEnforcer() *Enforcer {
	return &Enforcer{}
}

type ComplianceReport struct {
	PatternName string   `json:"pattern_name"`
	IsCompliant bool     `json:"is_compliant"`
	Violations  []string `json:"violations"`
}

func (e *Enforcer) CheckCompliance(ctx context.Context, page *pagestudio.CorePage) (*ComplianceReport, error) {
	report := &ComplianceReport{
		PatternName: "Unknown",
		IsCompliant: true,
		Violations:  make([]string, 0),
	}

	// Mock identification
	if strings.Contains(page.Name, "Dashboard") {
		report.PatternName = "Enterprise Dashboard"

		// Rule: Dashboard must have date filter
		if !strings.Contains(string(page.Components), "DateFilter") {
			report.IsCompliant = false
			report.Violations = append(report.Violations, "Missing required component: DateFilter")
		}
	}

	return report, nil
}
