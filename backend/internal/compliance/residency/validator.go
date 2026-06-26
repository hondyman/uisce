package residency

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type Violation struct {
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(ctx context.Context, page *pagestudio.CorePage, tenantRegion string) ([]Violation, error) {
	violations := make([]Violation, 0)

	// Mock Rule: "US-Only" data cannot be accessed by EU tenants
	// In a real system, we'd check the page's bound APIs/Objects against a registry of regional restrictions.

	// Simulated check: if page name implies US data and tenant is EU
	if (page.Name == "US Market Data" || page.Name == "SEC Filings") && tenantRegion == "EU" {
		violations = append(violations, Violation{
			RuleID:      "region:us_data_in_eu",
			Description: fmt.Sprintf("Page '%s' contains US-restricted data but tenant is in %s region.", page.Name, tenantRegion),
			Severity:    "critical",
		})
	}

	return violations, nil
}
