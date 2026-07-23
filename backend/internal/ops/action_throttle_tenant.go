package ops

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ThrottleTenantAction throttles a tenant's request rate to contain blast radius
type ThrottleTenantAction struct{}

// ThrottleTenantParams defines parameters for throttling a tenant
type ThrottleTenantParams struct {
	TenantID        string        `json:"tenant_id"`
	RateLimitPerSec int64         `json:"rate_limit_per_sec"`
	Duration        time.Duration `json:"duration"`
	Reason          string        `json:"reason,omitempty"`
}

// ThrottleTenantResult is the result of a throttle action
type ThrottleTenantResult struct {
	TenantID          string        `json:"tenant_id"`
	RateLimitPerSec   int64         `json:"rate_limit_per_sec"`
	Duration          time.Duration `json:"duration"`
	ExpiresAt         time.Time     `json:"expires_at"`
	PreviousRateLimit *int64        `json:"previous_rate_limit,omitempty"`
}

// NewThrottleTenantAction creates a new throttle tenant action
func NewThrottleTenantAction() *ThrottleTenantAction {
	return &ThrottleTenantAction{}
}

// ID returns the action identifier
func (a *ThrottleTenantAction) ID() string {
	return "throttle_tenant"
}

// Name returns the human-readable action name
func (a *ThrottleTenantAction) Name() string {
	return "Throttle Tenant"
}

// Validate checks if preconditions are met
func (a *ThrottleTenantAction) Validate(ctx context.Context, params json.RawMessage) error {
	// In production, would check:
	// - Tenant exists
	// - Tenant is causing high error rate or latency
	// - Throttle not already applied
	// For now, basic validation
	return nil
}

// Execute performs the throttle tenant action
func (a *ThrottleTenantAction) Execute(ctx context.Context, params json.RawMessage) (map[string]interface{}, error) {
	// In production, this would:
	// 1. Insert rate limit policy in policy engine
	// 2. Update tenant's rate limiter configuration
	// 3. Set expiration timer for auto-unthrottle
	// 4. Send notification to tenant (optional)
	//
	// For now, simulate with 50ms delay
	time.Sleep(50 * time.Millisecond)

	expiresAt := time.Now().UTC().Add(10 * time.Minute)

	result := ThrottleTenantResult{
		TenantID:          "tenant-123",
		RateLimitPerSec:   100,
		Duration:          10 * time.Minute,
		ExpiresAt:         expiresAt,
		PreviousRateLimit: nil,
	}

	return map[string]interface{}{
		"tenant_id":          result.TenantID,
		"rate_limit_per_sec": result.RateLimitPerSec,
		"duration":           result.Duration.String(),
		"expires_at":         result.ExpiresAt,
	}, nil
}

// Rollback attempts to undo the throttle by removing the rate limit
func (a *ThrottleTenantAction) Rollback(ctx context.Context, store Store, historyID uuid.UUID) error {
	// In production, would:
	// 1. Fetch the original action history
	// 2. Remove rate limit policy
	// 3. Restore previous rate limit if any
	//
	// For now, return success
	return nil
}
