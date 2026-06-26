
package services

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"calendar-service/internal/availability"
)

// AvailabilityAdapter adapts internal/availability.Checker to the
// services.AvailabilityServiceTenantAwareInterface expected by API handlers.
//
// Maps (tenantID, calendarID) to an active schedule_profile via Hasura,
// with graceful fallback to default profile when mapping is unavailable.
// Non-breaking: adapter always returns a result (falls back to default).
type AvailabilityAdapter struct {
	checker              *availability.Checker
	logger               *logrus.Entry
	region               string
	profile              string
	lastResolvedProfile  string
	lastResolutionSource string // "hasura", "fallback", "error"
	hasuraConfigured     bool
	cacheEnabled         bool
}

// NewAvailabilityAdapter creates an adapter using the provided checker.
// regionDefault and profileDefault are used when the public API doesn't supply them.
func NewAvailabilityAdapter(checker *availability.Checker, regionDefault, profileDefault string, logger *logrus.Entry) *AvailabilityAdapter {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}
	if regionDefault == "" {
		regionDefault = os.Getenv("DEFAULT_REGION")
		if regionDefault == "" {
			regionDefault = "us-east-1"
		}
	}
	if profileDefault == "" {
		profileDefault = "default"
	}

	hasuraConfigured := os.Getenv("HASURA_ENDPOINT") != ""
	cacheEnabled := os.Getenv("CACHE_ENABLED") == "true" || os.Getenv("CACHE_ENABLED") == "1"

	return &AvailabilityAdapter{
		checker:              checker,
		logger:               logger.WithField("service", "availability_adapter"),
		region:               regionDefault,
		profile:              profileDefault,
		hasuraConfigured:     hasuraConfigured,
		cacheEnabled:         cacheEnabled,
		lastResolutionSource: "not_initialized",
	}
}

// CheckAvailability implements services.AvailabilityServiceTenantAwareInterface
// It maps the public (tenantID, calendarID) call to a cached profile resolution.
func (a *AvailabilityAdapter) CheckAvailability(ctx context.Context, tenantID, calendarID string) (bool, error) {
	start := time.Now().UTC()

	// Try to resolve profile via Hasura-backed mapping (profile_calendars).
	// Fall back to adapter default profile when no mapping exists or Hasura is not configured.
	profileToUse := a.profile
	resolutionSource := "fallback"

	if a.checker != nil && a.hasuraConfigured {
		if resolved, err := a.checker.ResolveProfileNameForCalendar(ctx, tenantID, calendarID); err != nil {
			a.logger.WithError(err).WithFields(logrus.Fields{
				"tenant_id":   tenantID,
				"calendar_id": calendarID,
			}).Warn("failed to resolve profile from Hasura; using default")
			resolutionSource = "error"
		} else if resolved != "" {
			profileToUse = resolved
			resolutionSource = "hasura"
			a.logger.WithFields(logrus.Fields{
				"tenant_id":        tenantID,
				"calendar_id":      calendarID,
				"resolved_profile": resolved,
			}).Debug("resolved profile via Hasura")
		} else {
			// No mapping found in Hasura, use default
			resolutionSource = "fallback"
		}
	}

	// Store for GetMetrics
	a.lastResolvedProfile = profileToUse
	a.lastResolutionSource = resolutionSource

	result, err := a.checker.CheckAvailability(ctx, tenantID, a.region, profileToUse, time.Now().UTC(), time.Now().UTC().Add(1*time.Hour))
	if err != nil {
		a.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":            tenantID,
			"calendar_id":          calendarID,
			"profile":              profileToUse,
			"resolution_source":    resolutionSource,
			"elapsed_milliseconds": time.Since(start).Milliseconds(),
		}).Warn("availability check failed via adapter")
		return false, err
	}

	a.logger.WithFields(logrus.Fields{
		"tenant_id":            tenantID,
		"calendar_id":          calendarID,
		"profile":              profileToUse,
		"resolution_source":    resolutionSource,
		"available":            result.Available,
		"elapsed_milliseconds": time.Since(start).Milliseconds(),
	}).Debug("availability check completed")

	return result.Available, nil
}

// GetMetrics returns availability metrics including cache and resolution stats
func (a *AvailabilityAdapter) GetMetrics(ctx context.Context, tenantID, calendarID string) (map[string]interface{}, error) {
	metrics := map[string]interface{}{
		"tenant_id":              tenantID,
		"calendar_id":            calendarID,
		"cache_enabled":          a.cacheEnabled,
		"hasura_configured":      a.hasuraConfigured,
		"default_profile":        a.profile,
		"default_region":         a.region,
		"last_resolved_profile":  a.lastResolvedProfile,
		"last_resolution_source": a.lastResolutionSource,
		"checker_initialized":    a.checker != nil,
	}

	return metrics, nil
}
