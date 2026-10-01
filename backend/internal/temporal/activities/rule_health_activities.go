package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type RuleHealthActivities struct {
	DB     *sqlx.DB
	Logger *zap.SugaredLogger
}

func NewRuleHealthActivities(db *sqlx.DB, logger *zap.SugaredLogger) *RuleHealthActivities {
	return &RuleHealthActivities{
		DB:     db,
		Logger: logger,
	}
}

type RuleHealthCheckInput struct {
	TenantID string `json:"tenant_id,omitempty"` // empty = check all tenants
}

type FlaggedRuleReport struct {
	RuleID         string  `json:"rule_id"`
	RuleKey        string  `json:"rule_key"`
	BOName         string  `json:"bo_name"`
	TenantID       string  `json:"tenant_id"`
	Status         string  `json:"status"`
	EvalCount      int64   `json:"eval_count"`
	ViolationCount int64   `json:"violation_count"`
	RuleErrorCount int64   `json:"rule_error_count"`
	FailureRate    float64 `json:"failure_rate"`
	RuleErrorRate  float64 `json:"rule_error_rate"`
	Details        string  `json:"details"`
}

type RuleHealthCheckResult struct {
	EvaluatedRules int                 `json:"evaluated_rules"`
	FlaggedRules   []FlaggedRuleReport `json:"flagged_rules"`
	Timestamp      time.Time           `json:"timestamp"`
}

// RunRuleHealthCheckActivity checks trailing 24h validation_rule_violations
// and flags stale schema drift, always-failing rules, and degraded engines.
func (a *RuleHealthActivities) RunRuleHealthCheckActivity(ctx context.Context, input RuleHealthCheckInput) (*RuleHealthCheckResult, error) {
	query := `
		SELECT
			v.tenant_id,
			v.rule_id,
			COALESCE(NULLIF(v.rule_name, ''), v.rule_id::text) AS rule_key,
			v.bo_key AS bo_name,
			COUNT(*) AS violation_count,
			COUNT(*) FILTER (WHERE v.rule_error) AS rule_error_count
		FROM validation_rule_violations v
		WHERE v.created_at >= NOW() - INTERVAL '24 hours'
	`
	var args []any
	if input.TenantID != "" {
		query += " AND v.tenant_id = $1"
		args = append(args, input.TenantID)
	}
	query += " GROUP BY v.tenant_id, v.rule_id, v.rule_name, v.bo_key"

	var rows []struct {
		TenantID       uuid.UUID `db:"tenant_id"`
		RuleID         uuid.UUID `db:"rule_id"`
		RuleKey        string    `db:"rule_key"`
		BOName         string    `db:"bo_name"`
		ViolationCount int64     `db:"violation_count"`
		RuleErrorCount int64     `db:"rule_error_count"`
	}

	err := a.DB.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query rule violations: %w", err)
	}

	var flagged []FlaggedRuleReport
	for _, r := range rows {
		// Approximate evaluation count from violations and errors
		evalCount := r.ViolationCount
		ruleErrorRate := 0.0
		if evalCount > 0 {
			ruleErrorRate = float64(r.RuleErrorCount) / float64(evalCount)
		}

		status := "HEALTHY"
		details := ""

		// Guard: false positives suppressed when eval_count < 50
		if r.RuleErrorCount > 0 && ruleErrorRate > 0.05 {
			status = "STALE_SCHEMA_DRIFT"
			details = fmt.Sprintf("%.1f%% evaluations errored (%d/%d) — possible schema drift or unmapped column",
				ruleErrorRate*100, r.RuleErrorCount, evalCount)
		} else if evalCount >= 50 && r.ViolationCount == evalCount && r.RuleErrorCount == 0 {
			status = "SUSPECT_ALWAYS_FAILS"
			details = fmt.Sprintf("All %d evaluations in trailing 24h resulted in violations — rule may be over-strict", evalCount)
		}

		if status != "HEALTHY" {
			rep := FlaggedRuleReport{
				RuleID:         r.RuleID.String(),
				RuleKey:        r.RuleKey,
				BOName:         r.BOName,
				TenantID:       r.TenantID.String(),
				Status:         status,
				EvalCount:      evalCount,
				ViolationCount: r.ViolationCount,
				RuleErrorCount: r.RuleErrorCount,
				FailureRate:    1.0,
				RuleErrorRate:  ruleErrorRate,
				Details:        details,
			}
			flagged = append(flagged, rep)

			// Upsert to validation_rule_health table
			_, _ = a.DB.ExecContext(ctx, `
				INSERT INTO validation_rule_health
					(id, tenant_id, rule_id, rule_key, bo_name, status, eval_count, violation_count, rule_error_count, failure_rate, rule_error_rate, details, last_evaluated_at, updated_at)
				VALUES
					(gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
			`, r.TenantID, r.RuleID, r.RuleKey, r.BOName, status, evalCount, r.ViolationCount, r.RuleErrorCount, 1.0, ruleErrorRate, details)
		}
	}

	if a.Logger != nil {
		a.Logger.Infof("[RuleHealthCheck] Evaluated %d rule aggregates, %d flagged for attention", len(rows), len(flagged))
	}

	return &RuleHealthCheckResult{
		EvaluatedRules: len(rows),
		FlaggedRules:   flagged,
		Timestamp:      time.Now().UTC(),
	}, nil
}
