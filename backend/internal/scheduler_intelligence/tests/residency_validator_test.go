package tests

import (
	"errors"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/scheduler_intelligence/compliance"
)

func TestResidencyValidator(t *testing.T) {
	validator := compliance.NewResidencyValidator()

	// Define base objects for reuse
	jobEU := &compliance.Job{
		ID: "job-eu",
		Compliance: &compliance.ComplianceMetadata{
			Residency:   "EU",
			PII:         false,
			Sensitivity: "LOW",
		},
	}
	jobPII := &compliance.Job{
		ID: "job-pii",
		Compliance: &compliance.ComplianceMetadata{
			Residency:   "US",
			PII:         true,
			Sensitivity: "LOW",
		},
	}
	jobHighSense := &compliance.Job{
		ID: "job-high-sense",
		Compliance: &compliance.ComplianceMetadata{
			Residency:   "US",
			PII:         false,
			Sensitivity: "HIGH",
		},
	}
	jobGlobal := &compliance.Job{
		ID: "job-global",
		Compliance: &compliance.ComplianceMetadata{
			Residency:   "GLOBAL",
			PII:         false,
			Sensitivity: "LOW",
		},
	}
	jobNoCompliance := &compliance.Job{
		ID:         "job-no-compliance",
		Compliance: nil,
	}
	jobNonPII := &compliance.Job{
		ID: "job-non-pii",
		Compliance: &compliance.ComplianceMetadata{
			Residency:   "US",
			PII:         false,
			Sensitivity: "LOW",
		},
	}

	tenantEU := compliance.TenantProfile{
		Region:     "EU",
		PIIAllowed: true,
		SecureZone: true,
	}
	tenantUS := compliance.TenantProfile{
		Region:     "US",
		PIIAllowed: true,
		SecureZone: true,
	}
	tenantNoPII := compliance.TenantProfile{
		Region:     "US",
		PIIAllowed: false,
		SecureZone: true,
	}
	tenantInsecure := compliance.TenantProfile{
		Region:     "US",
		PIIAllowed: true,
		SecureZone: false,
	}

	tests := []struct {
		name      string
		job       *compliance.Job
		tenant    compliance.TenantProfile
		expectErr error
	}{
		{
			name:      "Test 1: EU job, EU tenant -> allowed",
			job:       jobEU,
			tenant:    tenantEU,
			expectErr: nil,
		},
		{
			name:      "Test 2: EU job, US tenant -> blocked",
			job:       jobEU,
			tenant:    tenantUS,
			expectErr: compliance.ErrResidencyViolation,
		},
		{
			name:      "Test 3: PII job, tenant not PII-approved -> blocked",
			job:       jobPII,
			tenant:    tenantNoPII,
			expectErr: compliance.ErrPIIRestricted,
		},
		{
			name:      "Test 4: Non-PII job, tenant not PII-approved -> allowed",
			job:       jobNonPII,
			tenant:    tenantNoPII,
			expectErr: nil,
		},
		{
			name:      "Test 5: Global job, global tenant (any region) -> allowed",
			job:       jobGlobal,
			tenant:    tenantUS,
			expectErr: nil,
		},
		{
			name:      "Test 6: Sensitivity HIGH + non-secure tenant -> blocked",
			job:       jobHighSense,   // US residency
			tenant:    tenantInsecure, // US region, but not secure
			expectErr: compliance.ErrSensitivityViolation,
		},
		{
			name:      "Test 7: Compliance metadata missing -> fail closed",
			job:       jobNoCompliance,
			tenant:    tenantUS,
			expectErr: compliance.ErrComplianceMetadataMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.job, tt.tenant)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("expected error %v, got %v", tt.expectErr, err)
			}
		})
	}
}
