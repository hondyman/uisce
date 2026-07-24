package risk

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ValidateRiskFactor performs basic validation on risk factors
func ValidateRiskFactor(r *RiskFactor) []error {
	var errs []error

	if r.FactorCode == "" {
		errs = append(errs, fmt.Errorf("factor_code is required"))
	}
	if r.FactorName == "" {
		errs = append(errs, fmt.Errorf("factor_name is required"))
	}

	validCategories := map[string]bool{"EQUITY": true, "FIXED_INCOME": true, "FX": true, "COMMODITY": true, "MACRO": true}
	if r.Category != nil && !validCategories[*r.Category] {
		errs = append(errs, fmt.Errorf("invalid category: %s", *r.Category))
	}

	validTypes := map[string]bool{"SYSTEMATIC": true, "IDIOSYNCRATIC": true}
	if r.FactorType != nil && !validTypes[*r.FactorType] {
		errs = append(errs, fmt.Errorf("invalid factor_type: %s", *r.FactorType))
	}

	return errs
}

// ValidateSecurityFactorExposure checks exposure records
func ValidateSecurityFactorExposure(e *SecurityFactorExposure) []error {
	var errs []error

	if e.SecurityID == uuid.Nil {
		errs = append(errs, fmt.Errorf("security_id is required"))
	}
	if e.FactorID == uuid.Nil {
		errs = append(errs, fmt.Errorf("factor_id is required"))
	}
	if e.AsOfDate.IsZero() {
		errs = append(errs, fmt.Errorf("as_of_date is required"))
	}
	if e.Exposure == nil {
		errs = append(errs, fmt.Errorf("exposure is required"))
	}

	return errs
}

// ValidateRiskScenario performs checks on risk stress scenarios
func ValidateRiskScenario(s *RiskScenario) []error {
	var errs []error

	if s.ScenarioCode == "" {
		errs = append(errs, fmt.Errorf("scenario_code is required"))
	}
	if s.ScenarioName == "" {
		errs = append(errs, fmt.Errorf("scenario_name is required"))
	}

	validTypes := map[string]bool{"HISTORICAL": true, "HYPOTHETICAL": true, "PARAMETRIC": true}
	if s.ScenarioType != nil && !validTypes[strings.ToUpper(*s.ScenarioType)] {
		errs = append(errs, fmt.Errorf("invalid scenario_type: %s", *s.ScenarioType))
	}

	validStatuses := map[string]bool{"ACTIVE": true, "INACTIVE": true, "DRAFT": true}
	if !validStatuses[strings.ToUpper(s.Status)] {
		errs = append(errs, fmt.Errorf("invalid status: %s", s.Status))
	}

	return errs
}
