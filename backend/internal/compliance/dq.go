package compliance

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ValidateComplianceRule performs basic data quality checks on a rule
func ValidateComplianceRule(r *ComplianceRule) []error {
	var errs []error

	if r.RuleCode == "" {
		errs = append(errs, fmt.Errorf("rule_code is required"))
	}
	if r.RuleName == "" {
		errs = append(errs, fmt.Errorf("rule_name is required"))
	}
	if r.Expression == "" {
		errs = append(errs, fmt.Errorf("expression is required"))
	}

	validScopes := map[string]bool{"PORTFOLIO": true, "STRATEGY": true, "GLOBAL": true}
	if r.ScopeType != nil && !validScopes[*r.ScopeType] {
		errs = append(errs, fmt.Errorf("invalid scope_type: %s", *r.ScopeType))
	}

	validSeverities := map[string]bool{"HARD": true, "SOFT": true, "WARNING": true, "ALERT": true}
	if r.Severity != nil && !validSeverities[*r.Severity] {
		errs = append(errs, fmt.Errorf("invalid severity: %s", *r.Severity))
	}

	validStatuses := map[string]bool{"ACTIVE": true, "INACTIVE": true, "DRAFT": true, "ARCHIVED": true}
	if !validStatuses[r.Status] {
		errs = append(errs, fmt.Errorf("invalid status: %s", r.Status))
	}

	if r.EffectiveFrom.IsZero() {
		errs = append(errs, fmt.Errorf("effective_from date is required"))
	}

	return errs
}

// ValidateComplianceEvaluation checks eval results
func ValidateComplianceEvaluation(e *ComplianceEvaluation) []error {
	var errs []error

	if e.RuleID == uuid.Nil {
		errs = append(errs, fmt.Errorf("rule_id is required"))
	}
	if e.PortfolioID == uuid.Nil {
		errs = append(errs, fmt.Errorf("portfolio_id is required"))
	}
	if e.ValuationDate.IsZero() {
		errs = append(errs, fmt.Errorf("valuation_date is required"))
	}

	if e.Result != nil {
		validResults := map[string]bool{"PASS": true, "FAIL": true, "WARNING": true}
		if !validResults[*e.Result] {
			errs = append(errs, fmt.Errorf("invalid result: %s", *e.Result))
		}
	}

	return errs
}

// ValidateComplianceBreach checks breach objects
func ValidateComplianceBreach(b *ComplianceBreach) []error {
	var errs []error

	if b.EvaluationID == uuid.Nil {
		errs = append(errs, fmt.Errorf("evaluation_id is required"))
	}
	if b.RuleID == uuid.Nil {
		errs = append(errs, fmt.Errorf("rule_id is required"))
	}
	if b.PortfolioID == uuid.Nil {
		errs = append(errs, fmt.Errorf("portfolio_id is required"))
	}
	if b.ValuationDate.IsZero() {
		errs = append(errs, fmt.Errorf("valuation_date is required"))
	}
	if b.Severity == "" {
		errs = append(errs, fmt.Errorf("severity is required"))
	}

	validStatuses := map[string]bool{"OPEN": true, "ACKNOWLEDGED": true, "RESOLVED": true, "WAIVED": true}
	if !validStatuses[strings.ToUpper(b.Status)] {
		errs = append(errs, fmt.Errorf("invalid status: %s", b.Status))
	}

	return errs
}
