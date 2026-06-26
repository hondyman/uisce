package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TimelineService handles event recording and incident correlation
type TimelineService struct {
	store Store
}

// NewTimelineService creates a new timeline service
func NewTimelineService(store Store) *TimelineService {
	return &TimelineService{store: store}
}

// RecordAlertEvent records an alert trigger event and correlates to incident
func (s *TimelineService) RecordAlertEvent(ctx context.Context, alert Alert, value float64) error {
	title := fmt.Sprintf("Alert triggered: %s", alert.Name)

	details, _ := json.Marshal(map[string]any{
		"metric":     alert.Metric,
		"threshold":  alert.Threshold,
		"comparison": alert.Comparison,
		"value":      value,
		"scope":      alert.Scope,
	})

	severity := SeverityWarning
	if value > 100 {
		severity = SeverityError
	}

	e := Event{
		ID:         uuid.New(),
		EventType:  EventAlert,
		Scope:      alert.Scope,
		AlertID:    &alert.ID,
		Severity:   severity,
		Title:      title,
		Details:    details,
		OccurredAt: time.Now().UTC(),
	}

	// Correlate to existing incident
	inc, err := s.store.UpsertIncidentForEvent(ctx, e)
	if err != nil {
		return fmt.Errorf("upsert incident: %w", err)
	}
	e.IncidentID = &inc.ID

	return s.store.InsertEvent(ctx, e)
}

// RecordTenantHealthChange records a health score change event
func (s *TimelineService) RecordTenantHealthChange(ctx context.Context, tenantID uuid.UUID, oldScore, newScore int) error {
	if oldScore == newScore {
		return nil
	}

	severity := SeverityInfo
	if newScore < 70 {
		severity = SeverityWarning
	}
	if newScore < 50 {
		severity = SeverityError
	}
	if newScore < 30 {
		severity = SeverityCritical
	}

	title := fmt.Sprintf("Tenant health changed: %d → %d", oldScore, newScore)

	details, _ := json.Marshal(map[string]any{
		"old_score": oldScore,
		"new_score": newScore,
	})

	e := Event{
		ID:         uuid.New(),
		EventType:  EventTenantHealth,
		Scope:      "tenant",
		TenantID:   &tenantID,
		Severity:   severity,
		Title:      title,
		Details:    details,
		OccurredAt: time.Now().UTC(),
	}

	inc, err := s.store.UpsertIncidentForEvent(ctx, e)
	if err != nil {
		return fmt.Errorf("upsert incident: %w", err)
	}
	e.IncidentID = &inc.ID

	return s.store.InsertEvent(ctx, e)
}

// RecordEndpointHealthChange records an endpoint health score change event
func (s *TimelineService) RecordEndpointHealthChange(ctx context.Context, endpoint string, oldScore, newScore int) error {
	if oldScore == newScore {
		return nil
	}

	severity := SeverityInfo
	if newScore < 70 {
		severity = SeverityWarning
	}
	if newScore < 50 {
		severity = SeverityError
	}
	if newScore < 30 {
		severity = SeverityCritical
	}

	title := fmt.Sprintf("Endpoint health changed: %d → %d (%s)", oldScore, newScore, endpoint)

	details, _ := json.Marshal(map[string]any{
		"old_score": oldScore,
		"new_score": newScore,
		"endpoint":  endpoint,
	})

	e := Event{
		ID:           uuid.New(),
		EventType:    EventEndpointHealth,
		Scope:        "endpoint",
		EndpointPath: &endpoint,
		Severity:     severity,
		Title:        title,
		Details:      details,
		OccurredAt:   time.Now().UTC(),
	}

	inc, err := s.store.UpsertIncidentForEvent(ctx, e)
	if err != nil {
		return fmt.Errorf("upsert incident: %w", err)
	}
	e.IncidentID = &inc.ID

	return s.store.InsertEvent(ctx, e)
}

// RecordErrorFingerprint records a new error fingerprint discovery event
func (s *TimelineService) RecordErrorFingerprint(ctx context.Context, fp ErrorFingerprint) error {
	title := fmt.Sprintf("New error fingerprint: %s %d", fp.Path, fp.StatusCode)

	details, _ := json.Marshal(map[string]any{
		"path":        fp.Path,
		"status_code": fp.StatusCode,
		"message":     fp.SampleMessage,
		"count":       fp.Count,
		"first_seen":  fp.FirstSeen,
	})

	e := Event{
		ID:            uuid.New(),
		EventType:     EventFingerprint,
		Scope:         "endpoint",
		EndpointPath:  &fp.Path,
		FingerprintID: &fp.ID,
		Severity:      SeverityWarning,
		Title:         title,
		Details:       details,
		OccurredAt:    time.Now().UTC(),
	}

	inc, err := s.store.UpsertIncidentForEvent(ctx, e)
	if err != nil {
		return fmt.Errorf("upsert incident: %w", err)
	}
	e.IncidentID = &inc.ID

	return s.store.InsertEvent(ctx, e)
}

// RecordLatencyAnomaly records a latency anomaly detection event
func (s *TimelineService) RecordLatencyAnomaly(ctx context.Context, endpoint string, p95, baseline int) error {
	if p95 <= baseline {
		return nil
	}

	severity := SeverityWarning
	if p95 > baseline*150/100 {
		severity = SeverityError
	}

	title := fmt.Sprintf("Latency anomaly: p95 %dms (baseline %dms) – %s", p95, baseline, endpoint)

	details, _ := json.Marshal(map[string]any{
		"endpoint":         endpoint,
		"p95_ms":           p95,
		"baseline_ms":      baseline,
		"increase_percent": (float64(p95) / float64(baseline) * 100) - 100,
	})

	e := Event{
		ID:           uuid.New(),
		EventType:    EventLatencyAnomaly,
		Scope:        "endpoint",
		EndpointPath: &endpoint,
		Severity:     severity,
		Title:        title,
		Details:      details,
		OccurredAt:   time.Now().UTC(),
	}

	inc, err := s.store.UpsertIncidentForEvent(ctx, e)
	if err != nil {
		return fmt.Errorf("upsert incident: %w", err)
	}
	e.IncidentID = &inc.ID

	return s.store.InsertEvent(ctx, e)
}
