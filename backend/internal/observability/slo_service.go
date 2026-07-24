package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SLOService manages SLOs and their evaluation
type SLOService struct {
	db               *sqlx.DB
	metricsCollector *MetricsCollector
	alertDispatcher  *AlertDispatcher
}

// NewSLOService creates a new SLO service
func NewSLOService(db *sqlx.DB, metricsCollector *MetricsCollector, alertDispatcher *AlertDispatcher) *SLOService {
	return &SLOService{
		db:               db,
		metricsCollector: metricsCollector,
		alertDispatcher:  alertDispatcher,
	}
}

// CreateSLO creates a new SLO
func (s *SLOService) CreateSLO(ctx context.Context, slo *SLO) error {
	query := `
		INSERT INTO obs_slos (tenant_id, name, description, target, time_window, metric_query)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	return s.db.QueryRowxContext(ctx, query,
		slo.TenantID, slo.Name, slo.Description, slo.Target, slo.Window, slo.MetricQuery,
	).Scan(&slo.ID, &slo.CreatedAt, &slo.UpdatedAt)
}

// UpdateSLO updates an existing SLO
func (s *SLOService) UpdateSLO(ctx context.Context, slo *SLO) error {
	query := `
		UPDATE obs_slos
		SET name = $2, description = $3, target = $4, time_window = $5, metric_query = $6, updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		slo.ID, slo.Name, slo.Description, slo.Target, slo.Window, slo.MetricQuery,
	)
	return err
}

// DeleteSLO deletes an SLO
func (s *SLOService) DeleteSLO(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM obs_slos WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// GetSLO retrieves an SLO by ID
func (s *SLOService) GetSLO(ctx context.Context, id uuid.UUID) (*SLO, error) {
	var slo SLO
	query := `SELECT * FROM obs_slos WHERE id = $1`
	err := s.db.GetContext(ctx, &slo, query, id)
	if err != nil {
		return nil, err
	}

	// Load alert rules
	slo.AlertRules, _ = s.GetAlertRulesForSLO(ctx, id)

	return &slo, nil
}

// ListSLOs lists all SLOs for a tenant
func (s *SLOService) ListSLOs(ctx context.Context, tenantID uuid.UUID) ([]SLO, error) {
	var slos []SLO
	query := `SELECT * FROM obs_slos WHERE tenant_id = $1 ORDER BY name`
	err := s.db.SelectContext(ctx, &slos, query, tenantID)
	return slos, err
}

// GetSLOStatus evaluates and returns the current status of an SLO
func (s *SLOService) GetSLOStatus(ctx context.Context, id uuid.UUID) (*SLOStatus, error) {
	slo, err := s.GetSLO(ctx, id)
	if err != nil {
		return nil, err
	}

	// Parse window duration
	windowDuration, err := parseWindow(slo.Window)
	if err != nil {
		windowDuration = 7 * 24 * time.Hour // Default 7 days
	}

	windowStart := time.Now().Add(-windowDuration)
	windowEnd := time.Now()

	// Calculate current value based on metric query
	currentValue, err := s.evaluateMetricQuery(ctx, slo.TenantID, slo.MetricQuery, windowDuration)
	if err != nil {
		// Return status with error indicator
		return &SLOStatus{
			SLOID:         slo.ID,
			SLOName:       slo.Name,
			Target:        slo.Target,
			Status:        "unknown",
			WindowStart:   windowStart,
			WindowEnd:     windowEnd,
			LastEvaluated: time.Now(),
		}, nil
	}

	// Calculate budget
	budgetTotal := 100.0 - slo.Target // e.g., if target is 99.9%, budget is 0.1%
	budgetConsumed := 100.0 - currentValue
	budgetRemaining := budgetTotal - budgetConsumed

	// Determine status
	var status string
	if budgetRemaining > budgetTotal*0.5 {
		status = string(SLOHealthy)
	} else if budgetRemaining > 0 {
		status = string(SLODegraded)
	} else {
		status = string(SLOBreached)
	}

	return &SLOStatus{
		SLOID:           slo.ID,
		SLOName:         slo.Name,
		CurrentValue:    currentValue,
		Target:          slo.Target,
		BudgetTotal:     budgetTotal,
		BudgetConsumed:  budgetConsumed,
		BudgetRemaining: budgetRemaining,
		Status:          status,
		WindowStart:     windowStart,
		WindowEnd:       windowEnd,
		LastEvaluated:   time.Now(),
	}, nil
}

// EvaluateAllSLOs evaluates all SLOs and triggers alerts if needed
func (s *SLOService) EvaluateAllSLOs(ctx context.Context, tenantID uuid.UUID) ([]SLOStatus, error) {
	slos, err := s.ListSLOs(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var statuses []SLOStatus

	for _, slo := range slos {
		status, err := s.GetSLOStatus(ctx, slo.ID)
		if err != nil {
			continue
		}
		statuses = append(statuses, *status)

		// Check alert rules
		for _, rule := range slo.AlertRules {
			if !rule.Enabled {
				continue
			}

			shouldAlert := s.evaluateAlertCondition(rule.Condition, status)
			if shouldAlert {
				alert := Alert{
					ID:        uuid.New(),
					TenantID:  tenantID,
					RuleID:    rule.ID,
					SLOID:     slo.ID,
					Severity:  rule.Severity,
					Message:   fmt.Sprintf("SLO '%s' alert: %s (current: %.2f%%, target: %.2f%%)", slo.Name, rule.Condition, status.CurrentValue, status.Target),
					Status:    string(AlertFiring),
					Value:     status.CurrentValue,
					Threshold: status.Target,
					FiredAt:   time.Now(),
				}

				// Persist and dispatch alert
				s.createAlert(ctx, &alert)
				if s.alertDispatcher != nil {
					s.alertDispatcher.Dispatch(ctx, &alert, rule.Channels)
				}
			}
		}
	}

	return statuses, nil
}

// CreateAlertRule creates a new alert rule for an SLO
func (s *SLOService) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	query := `
		INSERT INTO obs_alert_rules (tenant_id, slo_id, name, condition, severity, channels, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	channelsJSON := "[]"
	if len(rule.Channels) > 0 {
		channelsJSON = channelsToJSON(rule.Channels)
	}

	return s.db.QueryRowxContext(ctx, query,
		rule.TenantID, rule.SLOID, rule.Name, rule.Condition, rule.Severity, channelsJSON, rule.Enabled,
	).Scan(&rule.ID, &rule.CreatedAt)
}

// GetAlertRulesForSLO returns all alert rules for an SLO
func (s *SLOService) GetAlertRulesForSLO(ctx context.Context, sloID uuid.UUID) ([]AlertRule, error) {
	var rules []AlertRule
	query := `SELECT id, tenant_id, slo_id, name, condition, severity, enabled, created_at FROM obs_alert_rules WHERE slo_id = $1`
	err := s.db.SelectContext(ctx, &rules, query, sloID)
	return rules, err
}

// GetActiveAlerts returns all currently firing alerts
func (s *SLOService) GetActiveAlerts(ctx context.Context, tenantID uuid.UUID) ([]Alert, error) {
	var alerts []Alert
	query := `
		SELECT * FROM obs_alerts
		WHERE tenant_id = $1 AND status = 'firing'
		ORDER BY fired_at DESC
	`
	err := s.db.SelectContext(ctx, &alerts, query, tenantID)
	return alerts, err
}

// ResolveAlert marks an alert as resolved
func (s *SLOService) ResolveAlert(ctx context.Context, alertID uuid.UUID) error {
	query := `UPDATE obs_alerts SET status = 'resolved', resolved_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, alertID)
	return err
}

// Helper methods

func (s *SLOService) evaluateMetricQuery(ctx context.Context, tenantID uuid.UUID, query string, window time.Duration) (float64, error) {
	// Simple metric query evaluation
	// In production, this would parse a PromQL-like query

	// For now, evaluate based on common patterns
	// Example queries: "success_rate", "availability", "error_rate"

	switch query {
	case "availability", "success_rate":
		// Calculate success rate from metrics
		totalQueries, err := s.metricsCollector.GetAggregated(ctx, tenantID, "query_count", "sum", window)
		if err != nil || totalQueries == 0 {
			return 100.0, nil // No data = assume 100%
		}

		errorCount, err := s.metricsCollector.GetAggregated(ctx, tenantID, "query_error_count", "sum", window)
		if err != nil {
			errorCount = 0
		}

		return (1.0 - errorCount/totalQueries) * 100.0, nil

	case "error_rate":
		totalQueries, err := s.metricsCollector.GetAggregated(ctx, tenantID, "query_count", "sum", window)
		if err != nil || totalQueries == 0 {
			return 0, nil
		}

		errorCount, err := s.metricsCollector.GetAggregated(ctx, tenantID, "query_error_count", "sum", window)
		if err != nil {
			return 0, nil
		}

		return (errorCount / totalQueries) * 100.0, nil

	case "latency_p95":
		// Would need percentile calculation
		return s.metricsCollector.GetAggregated(ctx, tenantID, "query_latency_ms", "avg", window)

	default:
		// Try to get direct metric
		return s.metricsCollector.GetAggregated(ctx, tenantID, query, "avg", window)
	}
}

func (s *SLOService) evaluateAlertCondition(condition string, status *SLOStatus) bool {
	// Simple condition evaluation
	// In production, would parse conditions like "budget_remaining < 20%"

	switch condition {
	case "budget_exhausted":
		return status.BudgetRemaining <= 0
	case "budget_low":
		return status.BudgetRemaining < status.BudgetTotal*0.2
	case "slo_breached":
		return status.Status == string(SLOBreached)
	case "slo_degraded":
		return status.Status == string(SLODegraded) || status.Status == string(SLOBreached)
	default:
		return false
	}
}

func (s *SLOService) createAlert(ctx context.Context, alert *Alert) error {
	query := `
		INSERT INTO obs_alerts (tenant_id, rule_id, slo_id, severity, message, status, fired_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	return s.db.QueryRowxContext(ctx, query,
		alert.TenantID, alert.RuleID, alert.SLOID, alert.Severity, alert.Message, alert.Status, alert.FiredAt,
	).Scan(&alert.ID)
}

func parseWindow(window string) (time.Duration, error) {
	// Parse windows like "7d", "30d", "24h"
	if len(window) < 2 {
		return 0, fmt.Errorf("invalid window: %s", window)
	}

	unit := window[len(window)-1]
	value := window[:len(window)-1]

	var multiplier time.Duration
	switch unit {
	case 'd':
		multiplier = 24 * time.Hour
	case 'h':
		multiplier = time.Hour
	case 'm':
		multiplier = time.Minute
	default:
		return 0, fmt.Errorf("unknown unit: %c", unit)
	}

	var num int
	_, err := fmt.Sscanf(value, "%d", &num)
	if err != nil {
		return 0, err
	}

	return time.Duration(num) * multiplier, nil
}

func channelsToJSON(channels []string) string {
	result := "["
	for i, ch := range channels {
		if i > 0 {
			result += ","
		}
		result += `"` + ch + `"`
	}
	result += "]"
	return result
}
