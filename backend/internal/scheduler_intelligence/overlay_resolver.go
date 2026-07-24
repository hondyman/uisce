package scheduler_intelligence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ResolveJobForTenant resolves a global job with tenant-specific overrides
func (s *Service) ResolveJobForTenant(ctx context.Context, jobID uuid.UUID, tenantID uuid.UUID) (*Job, error) {
	// First, try to get the base job
	base, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base job: %w", err)
	}

	// If it's already a tenant-specific job, return it
	if base.Scope == ScopeTenant {
		return base, nil
	}

	// It's a global job. Look for a tenant-specific override.
	// We need a repository method for this: GetTenantJobOverride
	override, err := s.repo.GetTenantJobOverride(ctx, jobID, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No override, return base global job
			return base, nil
		}
		return nil, fmt.Errorf("failed to check for job override: %w", err)
	}

	if override != nil {
		return override, nil
	}

	return base, nil
}

// Note: GetTenantJobOverride needs to be implemented in repository.go
// It should look for a job where parent_job_id = jobID AND tenant_id = tenantID
// This implies we need a parent_job_id field if we want true overlays,
// OR we just use the name as a link, but ID is safer.
// For now, let's assume overlays are handled by name or a specific field.
