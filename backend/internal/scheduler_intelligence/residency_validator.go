package scheduler_intelligence

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// ResidencyValidator validates if a job complies with residency and privacy rules
// for the target tenant.
type ResidencyValidator struct {
	// In a real system, we'd inject a TenantService or similar to look up tenant metadata
}

// NewResidencyValidator creates a new validator
func NewResidencyValidator() *ResidencyValidator {
	return &ResidencyValidator{}
}

// ValidationResult contains the outcome of a validation check
type ValidationResult struct {
	Allowed        bool
	BlockReason    string
	SuggestionType string // e.g. RESIDENCY_VIOLATION, PII_RISK
}

// Validate checks if the job can run in the context of the tenant
func (v *ResidencyValidator) Validate(ctx context.Context, job Job, tenantID string) ValidationResult {
	// 1. Check Residency
	// Assumption: We might infer tenant region from ID or look it up.
	// For now, we simulate tenant region check.
	// If Job.Compliance.Residency is EU, and Tenant is NOT EU, block.

	// Mock logic: If tenantID starts with "us-" but job is "EU", block.
	tenantRegion := "GLOBAL"
	if strings.HasPrefix(tenantID, "us-") {
		tenantRegion = "US"
	} else if strings.HasPrefix(tenantID, "eu-") {
		tenantRegion = "EU"
	}

	jobRegion := job.Compliance.Residency
	if jobRegion == "" {
		jobRegion = "GLOBAL"
	}

	if jobRegion != "GLOBAL" && jobRegion != tenantRegion {
		logging.GetLogger().Sugar().Warnf("Blocked job %s due to residency mismatch: Job=%s, Tenant=%s", job.ID, jobRegion, tenantRegion)
		return ValidationResult{
			Allowed:        false,
			BlockReason:    fmt.Sprintf("Job residency constraint (%s) does not match tenant region (%s).", jobRegion, tenantRegion),
			SuggestionType: "RESIDENCY_VIOLATION",
		}
	}

	// 2. Check PII
	// If Job involves PII (Compliance.PII = true), and Tenant has restricted PII handling (mocked config), warn or block.
	// For this exercise, we just log a warning if PII is true but allow it,
	// UNLESS StrictPII is enforced (simulated).
	if job.Compliance.PII {
		logging.GetLogger().Info("Running PII-sensitive job",
			zap.String("job_id", job.ID.String()),
			zap.String("tenant_id", tenantID),
		)
		// We could block if tenant region is strict about PII export, but residency check above covers location.
	}

	return ValidationResult{Allowed: true}
}
