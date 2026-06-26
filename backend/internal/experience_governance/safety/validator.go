package safety

import (
	"context"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type SafetyViolation struct {
	Rule        string `json:"rule"`
	Description string `json:"description"`
	RiskLevel   string `json:"risk_level"` // high, medium
}

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(ctx context.Context, page *pagestudio.CorePage) ([]SafetyViolation, error) {
	violations := make([]SafetyViolation, 0)

	compStr := string(page.Components)

	// Rule: Destructive actions need confirmation
	if strings.Contains(compStr, "DeleteButton") {
		if !strings.Contains(compStr, "ConfirmationModal") && !strings.Contains(compStr, "confirm: true") {
			violations = append(violations, SafetyViolation{
				Rule:        "destructive_confirmation",
				Description: "Destructive 'Delete' action detected without confirmation configuration.",
				RiskLevel:   "high",
			})
		}
	}

	// Rule: Mutation requires feedback
	// Mock check

	return violations, nil
}
