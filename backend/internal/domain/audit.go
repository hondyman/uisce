package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// AuditEvent represents an auditable event in the governance system
type AuditEvent struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	EventType  string                 `json:"event_type"`
	UserID     string                 `json:"user_id"`
	TenantID   string                 `json:"tenant_id"`
	ResourceID string                 `json:"resource_id"`
	Action     string                 `json:"action"`
	Result     string                 `json:"result"` // "allow", "deny", "error"
	Reason     string                 `json:"reason"`
	Duration   time.Duration          `json:"duration_ms"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Policies   []string               `json:"policies,omitempty"`
	Claims     []string               `json:"claims,omitempty"`
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogEvent(ctx context.Context, event *AuditEvent) error
	QueryEvents(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error)
}

// AuditFilter for querying audit events
type AuditFilter struct {
	UserID    string    `json:"user_id,omitempty"`
	TenantID  string    `json:"tenant_id,omitempty"`
	EventType string    `json:"event_type,omitempty"`
	Result    string    `json:"result,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

// SlogAuditLogger implements AuditLogger using structured logging
type SlogAuditLogger struct {
	Logger *slog.Logger
}

func (l *SlogAuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	// Add context as JSON string
	contextJSON := ""
	if len(event.Context) > 0 {
		if jsonBytes, err := json.Marshal(event.Context); err == nil {
			contextJSON = string(jsonBytes)
		}
	}

	l.Logger.InfoContext(ctx, "audit event",
		"id", event.ID,
		"timestamp", event.Timestamp,
		"event_type", event.EventType,
		"user_id", event.UserID,
		"tenant_id", event.TenantID,
		"resource_id", event.ResourceID,
		"action", event.Action,
		"result", event.Result,
		"reason", event.Reason,
		"duration", event.Duration,
		"request_id", event.RequestID,
		"ip_address", event.IPAddress,
		"context", contextJSON,
	)
	return nil
}

func (l *SlogAuditLogger) QueryEvents(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error) {
	// In a real implementation, this would query a database
	// For now, return empty slice
	return []*AuditEvent{}, nil
}

// AuditedEvaluator wraps an evaluator with audit logging
type AuditedEvaluator struct {
	Evaluator Evaluator
	Auditor   AuditLogger
}

func (ae *AuditedEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (bool, string, []EffectiveClaim, error) {
	start := time.Now()

	allow, reason, claims, err := ae.Evaluator.Evaluate(ctx, req)

	duration := time.Since(start)

	// Create audit event
	event := &AuditEvent{
		ID:         generateEventID(),
		Timestamp:  start,
		EventType:  "evaluation",
		UserID:     req.UserID,
		TenantID:   req.TenantID,
		ResourceID: req.AssetID,
		Action:     string(req.Action),
		Duration:   duration,
		Context:    req.Context,
	}

	if err != nil {
		event.Result = "error"
		event.Reason = err.Error()
	} else {
		if allow {
			event.Result = "allow"
		} else {
			event.Result = "deny"
		}
		event.Reason = reason
	}

	// Extract claim IDs for audit
	for _, claim := range claims {
		event.Claims = append(event.Claims, claim.Source)
	}

	// Extract request ID from context if available
	if reqID, ok := req.Context["request_id"].(string); ok {
		event.RequestID = reqID
	}

	// Log the event (don't fail the evaluation if audit logging fails)
	if auditErr := ae.Auditor.LogEvent(ctx, event); auditErr != nil {
		// In production, you might want to use a more sophisticated error handling
		// For now, we'll just log the audit error
		slog.WarnContext(ctx, "failed to log audit event", "error", auditErr)
	}

	return allow, reason, claims, err
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("audit_%d", time.Now().UnixNano())
}

// ComplianceReport generates compliance reports from audit data
type ComplianceReport struct {
	TimeRange    string           `json:"time_range"`
	TotalEvents  int64            `json:"total_events"`
	AllowRate    float64          `json:"allow_rate"`
	DenyRate     float64          `json:"deny_rate"`
	TopUsers     map[string]int64 `json:"top_users"`
	TopResources map[string]int64 `json:"top_resources"`
	PolicyUsage  map[string]int64 `json:"policy_usage"`
	Violations   []*AuditEvent    `json:"violations"`
}

// ComplianceReporter generates compliance reports
type ComplianceReporter struct {
	Auditor AuditLogger
}

func (cr *ComplianceReporter) GenerateReport(ctx context.Context, start, end time.Time) (*ComplianceReport, error) {
	events, err := cr.Auditor.QueryEvents(ctx, AuditFilter{
		StartTime: start,
		EndTime:   end,
	})

	if err != nil {
		return nil, err
	}

	report := &ComplianceReport{
		TimeRange:    fmt.Sprintf("%s to %s", start.Format(time.RFC3339), end.Format(time.RFC3339)),
		TotalEvents:  int64(len(events)),
		TopUsers:     make(map[string]int64),
		TopResources: make(map[string]int64),
		PolicyUsage:  make(map[string]int64),
		Violations:   make([]*AuditEvent, 0),
	}

	var allowCount, denyCount int64

	for _, event := range events {
		// Count results
		switch event.Result {
		case "allow":
			allowCount++
		case "deny":
			denyCount++
			report.Violations = append(report.Violations, event)
		}

		// Track top users
		report.TopUsers[event.UserID]++

		// Track top resources
		report.TopResources[event.ResourceID]++

		// Track policy usage
		for _, policy := range event.Policies {
			report.PolicyUsage[policy]++
		}
	}

	if report.TotalEvents > 0 {
		report.AllowRate = float64(allowCount) / float64(report.TotalEvents)
		report.DenyRate = float64(denyCount) / float64(report.TotalEvents)
	}

	return report, nil
}
