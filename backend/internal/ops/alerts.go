package ops

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AlertEvaluator evaluates alert conditions and fires events
type AlertEvaluator struct {
	store    Store
	timeline *TimelineService
}

// NewAlertEvaluator creates a new alert evaluator
func NewAlertEvaluator(store Store) *AlertEvaluator {
	return &AlertEvaluator{store: store, timeline: nil}
}

// NewAlertEvaluatorWithTimeline creates a new alert evaluator with timeline service
func NewAlertEvaluatorWithTimeline(store Store, timeline *TimelineService) *AlertEvaluator {
	return &AlertEvaluator{store: store, timeline: timeline}
}

// EvaluateAll evaluates all enabled alerts and records events if triggered
func (e *AlertEvaluator) EvaluateAll(ctx context.Context) error {
	alerts, err := e.store.ListAlerts(ctx, boolPtr(true))
	if err != nil {
		return fmt.Errorf("list alerts: %w", err)
	}

	for _, alert := range alerts {
		if err := e.Evaluate(ctx, alert); err != nil {
			// Log error but continue evaluating other alerts
			fmt.Printf("error evaluating alert %s: %v\n", alert.ID, err)
		}
	}

	return nil
}

// Evaluate evaluates a single alert and records an event if triggered
func (e *AlertEvaluator) Evaluate(ctx context.Context, a Alert) error {
	if !a.Enabled {
		return nil
	}

	window := time.Duration(a.WindowSecs) * time.Second

	// Fetch metric value based on scope
	value, scopeID, err := e.fetchMetric(ctx, a, window)
	if err != nil {
		return err
	}

	// Check if threshold is crossed
	triggered := e.compareTrigger(a.Comparison, value, a.Threshold)

	if !triggered {
		return nil
	}

	// Record event
	event := AlertEvent{
		ID:          uuid.New(),
		AlertID:     a.ID,
		ScopeID:     scopeID,
		Value:       value,
		TriggeredAt: time.Now().UTC(),
	}

	if err := e.store.InsertAlertEvent(ctx, event); err != nil {
		return err
	}

	// Emit timeline event if timeline service is available
	if e.timeline != nil {
		_ = e.timeline.RecordAlertEvent(ctx, a, value)
	}

	return nil
}

// fetchMetric fetches the metric value for an alert
func (e *AlertEvaluator) fetchMetric(ctx context.Context, a Alert, window time.Duration) (float64, *uuid.UUID, error) {
	since := time.Now().UTC().Add(-window)

	switch a.Scope {
	case "global":
		value, err := e.store.GetMetricValue(ctx, a.Metric, "global", since)
		return value, nil, err

	case "tenant":
		// For tenant scope, we would need to specify which tenant
		// This is typically called per-tenant in a loop
		return 0, nil, fmt.Errorf("tenant scope requires tenant_id")

	case "endpoint":
		// Similar to tenant scope
		return 0, nil, fmt.Errorf("endpoint scope requires endpoint")

	default:
		return 0, nil, fmt.Errorf("unknown alert scope: %s", a.Scope)
	}
}

// compareTrigger compares a value against a threshold
func (e *AlertEvaluator) compareTrigger(comparison string, value, threshold float64) bool {
	switch comparison {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	default:
		return false
	}
}

// AnomalyDetector detects anomalies by comparing current metrics to baselines
type AnomalyDetector struct {
	store Store
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(store Store) *AnomalyDetector {
	return &AnomalyDetector{store: store}
}

// DetectTenantAnomaly detects if a tenant's metrics deviate from baseline
func (a *AnomalyDetector) DetectTenantAnomaly(ctx context.Context, tenantID uuid.UUID, baselineWindow, currentWindow time.Duration) (bool, float64, error) {
	// Get baseline (e.g., same time yesterday)
	baselineTime := time.Now().UTC().Add(-24 * time.Hour)
	baselineSince := baselineTime.Add(-baselineWindow)

	// We'd query historical metrics and compare
	// This is a simplified example
	_ = baselineSince

	return false, 0, nil
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
