package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RuleStats represents execution statistics for a validation rule
type RuleStats struct {
	RuleID          uuid.UUID `json:"rule_id"`
	TotalExecutions int       `json:"total_executions"`
	SuccessCount    int       `json:"success_count"`
	FailureCount    int       `json:"failure_count"`
	ViolationCount  int       `json:"violation_count"`
	AvgExecutionMS  float64   `json:"avg_execution_ms"`
	LastExecutedAt  time.Time `json:"last_executed_at"`
}

// ViolationTrend represents violation trends over time
type ViolationTrend struct {
	Date           time.Time `json:"date"`
	ViolationCount int       `json:"violation_count"`
	RuleType       string    `json:"rule_type"`
}

// RuleViolation represents a top violated rule
type RuleViolation struct {
	RuleID         uuid.UUID `json:"rule_id"`
	RuleName       string    `json:"rule_name"`
	RuleType       string    `json:"rule_type"`
	ViolationCount int       `json:"violation_count"`
	Severity       string    `json:"severity"`
}

// ValidationAnalytics provides analytics for validation rules
type ValidationAnalytics struct {
	db *sql.DB
}

// NewValidationAnalytics creates a new validation analytics service
func NewValidationAnalytics(db *sql.DB) *ValidationAnalytics {
	return &ValidationAnalytics{db: db}
}

// GetRuleExecutionStats retrieves execution statistics for a rule
func (va *ValidationAnalytics) GetRuleExecutionStats(ctx context.Context, ruleID uuid.UUID, period time.Duration) (*RuleStats, error) {
	query := `
		SELECT
			rule_id,
			COUNT(*) as total_executions,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status = 'failure' THEN 1 ELSE 0 END) as failure_count,
			SUM(CASE WHEN status = 'violation' THEN 1 ELSE 0 END) as violation_count,
			AVG(execution_time_ms) as avg_execution_ms,
			MAX(executed_at) as last_executed_at
		FROM validation_rule_executions
		WHERE rule_id = $1
		  AND executed_at > NOW() - $2::interval
		GROUP BY rule_id
	`

	var stats RuleStats
	err := va.db.QueryRowContext(ctx, query, ruleID, period).Scan(
		&stats.RuleID,
		&stats.TotalExecutions,
		&stats.SuccessCount,
		&stats.FailureCount,
		&stats.ViolationCount,
		&stats.AvgExecutionMS,
		&stats.LastExecutedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &RuleStats{RuleID: ruleID}, nil
		}
		return nil, fmt.Errorf("failed to get rule stats: %w", err)
	}

	return &stats, nil
}

// GetViolationTrends retrieves violation trends for a tenant
func (va *ValidationAnalytics) GetViolationTrends(ctx context.Context, tenantID uuid.UUID, days int) ([]ViolationTrend, error) {
	query := `
		SELECT
			DATE(executed_at) as date,
			COUNT(*) as violation_count,
			r.rule_type
		FROM validation_rule_executions e
		JOIN validation_rules r ON e.rule_id = r.id
		WHERE e.tenant_id = $1
		  AND e.status = 'violation'
		  AND e.executed_at > NOW() - $2::interval
		GROUP BY DATE(executed_at), r.rule_type
		ORDER BY date DESC, violation_count DESC
	`

	rows, err := va.db.QueryContext(ctx, query, tenantID, fmt.Sprintf("%d days", days))
	if err != nil {
		return nil, fmt.Errorf("failed to get violation trends: %w", err)
	}
	defer rows.Close()

	var trends []ViolationTrend
	for rows.Next() {
		var t ViolationTrend
		if err := rows.Scan(&t.Date, &t.ViolationCount, &t.RuleType); err != nil {
			return nil, fmt.Errorf("failed to scan trend: %w", err)
		}
		trends = append(trends, t)
	}

	return trends, nil
}

// GetTopViolatedRules retrieves the most violated rules
func (va *ValidationAnalytics) GetTopViolatedRules(ctx context.Context, tenantID uuid.UUID, limit int) ([]RuleViolation, error) {
	query := `
		SELECT
			r.id,
			r.name,
			r.rule_type,
			COUNT(*) as violation_count,
			r.severity
		FROM validation_rule_executions e
		JOIN validation_rules r ON e.rule_id = r.id
		WHERE e.tenant_id = $1
		  AND e.status = 'violation'
		  AND e.executed_at > NOW() - '30 days'::interval
		GROUP BY r.id, r.name, r.rule_type, r.severity
		ORDER BY violation_count DESC
		LIMIT $2
	`

	rows, err := va.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top violated rules: %w", err)
	}
	defer rows.Close()

	var violations []RuleViolation
	for rows.Next() {
		var v RuleViolation
		if err := rows.Scan(&v.RuleID, &v.RuleName, &v.RuleType, &v.ViolationCount, &v.Severity); err != nil {
			return nil, fmt.Errorf("failed to scan violation: %w", err)
		}
		violations = append(violations, v)
	}

	return violations, nil
}

// GetDashboardMetrics retrieves comprehensive dashboard metrics
func (va *ValidationAnalytics) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	// Get overall stats
	totalRules, activeRules, err := va.getRuleCounts(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Get recent execution stats
	execStats, err := va.getRecentExecutionStats(ctx, tenantID, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	// Get top violations
	topViolations, err := va.GetTopViolatedRules(ctx, tenantID, 5)
	if err != nil {
		return nil, err
	}

	// Get violation trends
	trends, err := va.GetViolationTrends(ctx, tenantID, 7)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_rules":      totalRules,
		"active_rules":     activeRules,
		"executions_24h":   execStats["total"],
		"violations_24h":   execStats["violations"],
		"success_rate":     execStats["success_rate"],
		"top_violations":   topViolations,
		"violation_trends": trends,
	}, nil
}

func (va *ValidationAnalytics) getRuleCounts(ctx context.Context, tenantID uuid.UUID) (int, int, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN is_active = true THEN 1 ELSE 0 END) as active
		FROM validation_rules
		WHERE tenant_id = $1
	`

	var total, active int
	err := va.db.QueryRowContext(ctx, query, tenantID).Scan(&total, &active)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get rule counts: %w", err)
	}

	return total, active, nil
}

func (va *ValidationAnalytics) getRecentExecutionStats(ctx context.Context, tenantID uuid.UUID, period time.Duration) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'violation' THEN 1 ELSE 0 END) as violations,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successes
		FROM validation_rule_executions
		WHERE tenant_id = $1
		  AND executed_at > NOW() - $2::interval
	`

	var total, violations, successes int
	err := va.db.QueryRowContext(ctx, query, tenantID, period).Scan(&total, &violations, &successes)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution stats: %w", err)
	}

	successRate := 0.0
	if total > 0 {
		successRate = float64(successes) / float64(total) * 100
	}

	return map[string]interface{}{
		"total":        total,
		"violations":   violations,
		"successes":    successes,
		"success_rate": successRate,
	}, nil
}
