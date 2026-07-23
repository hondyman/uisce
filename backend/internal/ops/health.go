package ops

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// HealthCalculator computes health scores for tenants and endpoints
type HealthCalculator struct {
	store    Store
	timeline *TimelineService
}

// NewHealthCalculator creates a new health calculator
func NewHealthCalculator(store Store) *HealthCalculator {
	return &HealthCalculator{store: store, timeline: nil}
}

// NewHealthCalculatorWithTimeline creates a new health calculator with timeline service
func NewHealthCalculatorWithTimeline(store Store, timeline *TimelineService) *HealthCalculator {
	return &HealthCalculator{store: store, timeline: timeline}
}

// ComputeTenantHealth computes a tenant's composite health score
func (h *HealthCalculator) ComputeTenantHealth(ctx context.Context, tenantID uuid.UUID, window time.Duration) (*TenantHealth, error) {
	since := time.Now().UTC().Add(-window)

	metrics, err := h.store.GetTenantMetrics(ctx, tenantID, since)
	if err != nil {
		return nil, fmt.Errorf("get tenant metrics: %w", err)
	}

	if metrics == nil {
		// No metrics, assume healthy
		return &TenantHealth{
			TenantID:   tenantID,
			Score:      100,
			Components: map[string]float64{},
			ComputedAt: time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}, nil
	}

	// Normalize metrics to 0-100 scales
	availabilityScore := normalize(metrics.AvailabilityPct, 99)    // target 99%
	latencyScore := normalize(float64(metrics.P95), 200)           // target <200ms
	errorRateScore := normalize(metrics.ErrorRate, 0.01)           // target <1%
	rateLimitScore := normalize(float64(metrics.RateLimited), 100) // target <100 per hour

	// Weighted composite (must sum to 1.0)
	composite := (0.40 * availabilityScore) +
		(0.30 * latencyScore) +
		(0.20 * errorRateScore) +
		(0.10 * rateLimitScore)

	score := int(math.Round(composite))

	health := &TenantHealth{
		TenantID: tenantID,
		Score:    score,
		Components: map[string]float64{
			"availability": availabilityScore,
			"latency":      latencyScore,
			"error_rate":   errorRateScore,
			"rate_limits":  rateLimitScore,
		},
		ComputedAt: time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	// Cache it
	if err := h.store.UpsertTenantHealth(ctx, *health); err != nil {
		fmt.Printf("warn: failed to cache tenant health: %v\n", err)
	}

	// Emit timeline event if health changed significantly or timeline service available
	if h.timeline != nil {
		old := int(metrics.AvailabilityPct) // Rough approximation for old score
		if old != score {                   // Only emit if score changed
			_ = h.timeline.RecordTenantHealthChange(ctx, tenantID, old, score)
		}
	}

	return health, nil
}

// ComputeEndpointHealth computes an endpoint's health score
func (h *HealthCalculator) ComputeEndpointHealth(ctx context.Context, endpoint string, window time.Duration) (*EndpointHealth, error) {
	since := time.Now().UTC().Add(-window)

	metrics, err := h.store.GetEndpointMetrics(ctx, endpoint, since)
	if err != nil {
		return nil, fmt.Errorf("get endpoint metrics: %w", err)
	}

	if metrics == nil {
		return &EndpointHealth{
			Endpoint:   endpoint,
			Score:      100,
			ErrorRate:  0,
			P95MS:      0,
			Requests1H: 0,
			ComputedAt: time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}, nil
	}

	// Normalize metrics
	errorRateScore := normalize(metrics.ErrorRate, 0.01) // target <1%
	latencyScore := normalize(float64(metrics.P95), 200) // target <200ms

	// Weight by traffic (importance)
	trafficFactor := math.Min(1.0, float64(metrics.Requests)/10000)
	composite := (0.5*errorRateScore + 0.5*latencyScore) * trafficFactor

	score := int(math.Round(composite))

	health := &EndpointHealth{
		Endpoint:   endpoint,
		Score:      score,
		ErrorRate:  metrics.ErrorRate,
		P95MS:      metrics.P95,
		Requests1H: metrics.Requests,
		Components: map[string]float64{
			"error_rate": errorRateScore,
			"latency":    latencyScore,
		},
		ComputedAt: time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	// Cache it
	if err := h.store.UpsertEndpointHealth(ctx, *health); err != nil {
		fmt.Printf("warn: failed to cache endpoint health: %v\n", err)
	}

	// Emit timeline event if health changed and timeline service available
	if h.timeline != nil {
		old := int(metrics.ErrorRate * 100) // Rough approximation for old score
		if old != score {                   // Only emit if score changed
			_ = h.timeline.RecordEndpointHealthChange(ctx, endpoint, old, score)
		}
	}

	return health, nil
}

// normalize converts a measured value to a 0-100 score based on a target
// Higher measured values = lower scores (bad)
// Example: if target is 200ms and measured is 400ms, score is ~0
func normalize(measured, target float64) float64 {
	if target == 0 {
		if measured == 0 {
			return 100
		}
		return 0
	}

	// Ratio of measured to target
	ratio := measured / target

	// Convert to score: ratio 1.0 = 100, ratio 2.0 = 0, ratio 0.5 = 100+
	score := 100 - (ratio * 100)

	// Clamp to 0-100
	return math.Max(0, math.Min(100, score))
}

// GetHealthStatus returns a categorical status based on score
func GetHealthStatus(score int) HealthStatus {
	if score >= 80 {
		return HealthStatusHealthy
	}
	if score >= 50 {
		return HealthStatusDegraded
	}
	return HealthStatusCritical
}
