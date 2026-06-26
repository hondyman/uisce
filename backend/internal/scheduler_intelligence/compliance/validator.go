package compliance

import (
	"errors"
	"fmt"
)

// Common errors for compliance validation
var (
	ErrResidencyViolation        = errors.New("residency violation: job region does not match residency requirement")
	ErrPIIRestricted             = errors.New("PII restricted: tenant is not approved for PII processing")
	ErrSensitivityViolation      = errors.New("sensitivity violation: high sensitivity job in non-secure tenant")
	ErrComplianceMetadataMissing = errors.New("compliance metadata missing")
)

// ComplianceMetadata represents the compliance configuration for a job
type ComplianceMetadata struct {
	PII         bool   `json:"pii"`
	Residency   string `json:"residency"`   // "EU", "US", "GLOBAL", etc.
	Sensitivity string `json:"sensitivity"` // "LOW", "MEDIUM", "HIGH"
}

// Job represents a scheduler job with compliance metadata
type Job struct {
	ID         string
	Name       string
	Compliance *ComplianceMetadata
}

// TenantProfile represents the compliance profile of a tenant
type TenantProfile struct {
	ID         string
	Region     string // "EU", "US"
	PIIAllowed bool
	SecureZone bool
}

// ResidencyValidator validates jobs against tenant compliance profiles
type ResidencyValidator struct{}

// NewResidencyValidator creates a new instance of ResidencyValidator
func NewResidencyValidator() *ResidencyValidator {
	return &ResidencyValidator{}
}

// Validate checks if the job complies with the tenant's profile
func (v *ResidencyValidator) Validate(job *Job, tenant TenantProfile) error {
	if job.Compliance == nil {
		return ErrComplianceMetadataMissing
	}

	// 1. Residency Check
	if job.Compliance.Residency != "GLOBAL" && job.Compliance.Residency != "" {
		if job.Compliance.Residency != tenant.Region {
			return fmt.Errorf("%w: job requires %s, tenant is %s", ErrResidencyViolation, job.Compliance.Residency, tenant.Region)
		}
	}

	// 2. PII Check
	if job.Compliance.PII && !tenant.PIIAllowed {
		return ErrPIIRestricted
	}

	// 3. Sensitivity Check
	if job.Compliance.Sensitivity == "HIGH" && !tenant.SecureZone {
		return ErrSensitivityViolation
	}

	return nil
}
