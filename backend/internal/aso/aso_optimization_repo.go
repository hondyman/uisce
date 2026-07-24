package aso

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ASOOptimizationRepository manages optimization records
type ASOOptimizationRepository interface {
	// Create persists a new optimization
	Create(ctx context.Context, opt *ASOOptimization) error

	// GetByID retrieves an optimization by ID
	GetByID(ctx context.Context, id uuid.UUID) (*ASOOptimization, error)

	// List returns optimizations with filters
	List(ctx context.Context, filter OptimizationFilter) ([]ASOOptimization, error)

	// UpdateStatus updates the status of an optimization
	UpdateStatus(ctx context.Context, id uuid.UUID, status OptimizationStatus, actor string, reason string) error

	// MarkApplied marks an optimization as applied
	MarkApplied(ctx context.Context, id uuid.UUID, actor string, afterConfig json.RawMessage) error

	// MarkRejected marks an optimization as rejected
	MarkRejected(ctx context.Context, id uuid.UUID, actor, reason string) error

	// GetPendingForTarget returns pending optimizations for a target
	GetPendingForTarget(ctx context.Context, targetType TargetType, targetID uuid.UUID) ([]ASOOptimization, error)

	// SupersedeOptimization marks older optimizations as superseded
	SupersedeOptimization(ctx context.Context, targetType TargetType, targetID uuid.UUID, exceptID uuid.UUID) error

	// GetDailyStats returns daily stats for rate limiting
	GetDailyStats(ctx context.Context, env string, tenantID *uuid.UUID, date time.Time) (*ASODailyStats, error)

	// IncrementDailyStats increments daily stats counters
	IncrementDailyStats(ctx context.Context, env string, tenantID *uuid.UUID, field string) error
}

// OptimizationFilter for listing optimizations
type OptimizationFilter struct {
	Env        *string
	TenantID   *uuid.UUID
	Status     *OptimizationStatus
	Type       *OptimizationType
	TargetType *TargetType
	TargetID   *uuid.UUID
	Scope      *ASOScope
	Limit      int
	Offset     int
}

// asoOptimizationRepo implements ASOOptimizationRepository
type asoOptimizationRepo struct {
	db *sqlx.DB
}

// NewASOOptimizationRepository creates a new optimization repository
func NewASOOptimizationRepository(db *sqlx.DB) ASOOptimizationRepository {
	return &asoOptimizationRepo{db: db}
}

// Create persists a new optimization
func (r *asoOptimizationRepo) Create(ctx context.Context, opt *ASOOptimization) error {
	if opt.ID == uuid.Nil {
		opt.ID = uuid.New()
	}

	query := `
		INSERT INTO semantic.aso_optimization (
			id, env, tenant_id, scope,
			optimization_type, target_type, target_id, target_name,
			status, mode, score, reason, details,
			workload_window_days, queries_per_day, avg_latency_ms,
			p95_latency_ms, avg_rows_scanned, policy_id,
			created_by, before_config
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19,
			$20, $21
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		opt.ID, opt.Env, opt.TenantID, opt.Scope,
		opt.OptimizationType, opt.TargetType, opt.TargetID, opt.TargetName,
		opt.Status, opt.Mode, opt.Score, opt.Reason, opt.Details,
		opt.WorkloadWindowDays, opt.QueriesPerDay, opt.AvgLatencyMs,
		opt.P95LatencyMs, opt.AvgRowsScanned, opt.PolicyID,
		opt.CreatedBy, opt.BeforeConfig,
	)

	if err != nil {
		return fmt.Errorf("failed to create optimization: %w", err)
	}

	// Record audit entry
	r.recordAudit(ctx, opt.ID, "proposed", opt.CreatedBy, nil)

	return nil
}

// GetByID retrieves an optimization by ID
func (r *asoOptimizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*ASOOptimization, error) {
	var opt ASOOptimization
	err := r.db.GetContext(ctx, &opt, `
		SELECT * FROM semantic.aso_optimization WHERE id = $1
	`, id)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get optimization: %w", err)
	}

	return &opt, nil
}

// List returns optimizations with filters
func (r *asoOptimizationRepo) List(ctx context.Context, filter OptimizationFilter) ([]ASOOptimization, error) {
	query := `SELECT * FROM semantic.aso_optimization WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if filter.Env != nil {
		query += fmt.Sprintf(" AND env = $%d", argNum)
		args = append(args, *filter.Env)
		argNum++
	}

	if filter.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argNum)
		args = append(args, *filter.TenantID)
		argNum++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.Type != nil {
		query += fmt.Sprintf(" AND optimization_type = $%d", argNum)
		args = append(args, *filter.Type)
		argNum++
	}

	if filter.TargetType != nil {
		query += fmt.Sprintf(" AND target_type = $%d", argNum)
		args = append(args, *filter.TargetType)
		argNum++
	}

	if filter.TargetID != nil {
		query += fmt.Sprintf(" AND target_id = $%d", argNum)
		args = append(args, *filter.TargetID)
		argNum++
	}

	if filter.Scope != nil {
		query += fmt.Sprintf(" AND scope = $%d", argNum)
		args = append(args, *filter.Scope)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	var opts []ASOOptimization
	err := r.db.SelectContext(ctx, &opts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list optimizations: %w", err)
	}

	return opts, nil
}

// UpdateStatus updates the status of an optimization
func (r *asoOptimizationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status OptimizationStatus, actor string, reason string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE semantic.aso_optimization
		SET status = $2
		WHERE id = $1
	`, id, status)

	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	details := map[string]string{}
	if reason != "" {
		details["reason"] = reason
	}
	r.recordAudit(ctx, id, string(status), actor, details)

	return nil
}

// MarkApplied marks an optimization as applied
func (r *asoOptimizationRepo) MarkApplied(ctx context.Context, id uuid.UUID, actor string, afterConfig json.RawMessage) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE semantic.aso_optimization
		SET status = 'applied', applied_at = $2, applied_by = $3, after_config = $4
		WHERE id = $1
	`, id, now, actor, afterConfig)

	if err != nil {
		return fmt.Errorf("failed to mark applied: %w", err)
	}

	r.recordAudit(ctx, id, "applied", actor, nil)

	return nil
}

// MarkRejected marks an optimization as rejected
func (r *asoOptimizationRepo) MarkRejected(ctx context.Context, id uuid.UUID, actor, reason string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE semantic.aso_optimization
		SET status = 'rejected', rejected_at = $2, rejected_by = $3, rejection_reason = $4
		WHERE id = $1
	`, id, now, actor, reason)

	if err != nil {
		return fmt.Errorf("failed to mark rejected: %w", err)
	}

	r.recordAudit(ctx, id, "rejected", actor, map[string]string{"reason": reason})

	return nil
}

// GetPendingForTarget returns pending optimizations for a target
func (r *asoOptimizationRepo) GetPendingForTarget(ctx context.Context, targetType TargetType, targetID uuid.UUID) ([]ASOOptimization, error) {
	var opts []ASOOptimization
	err := r.db.SelectContext(ctx, &opts, `
		SELECT * FROM semantic.aso_optimization
		WHERE target_type = $1 AND target_id = $2
		AND status IN ('proposed', 'approved')
		ORDER BY created_at DESC
	`, targetType, targetID)

	if err != nil {
		return nil, fmt.Errorf("failed to get pending optimizations: %w", err)
	}

	return opts, nil
}

// SupersedeOptimization marks older optimizations as superseded
func (r *asoOptimizationRepo) SupersedeOptimization(ctx context.Context, targetType TargetType, targetID uuid.UUID, exceptID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE semantic.aso_optimization
		SET status = 'superseded'
		WHERE target_type = $1 AND target_id = $2
		AND id != $3
		AND status IN ('proposed', 'approved')
	`, targetType, targetID, exceptID)

	if err != nil {
		return fmt.Errorf("failed to supersede optimizations: %w", err)
	}

	return nil
}

// GetDailyStats returns daily stats for rate limiting
func (r *asoOptimizationRepo) GetDailyStats(ctx context.Context, env string, tenantID *uuid.UUID, date time.Time) (*ASODailyStats, error) {
	var stats ASODailyStats

	var err error
	if tenantID != nil {
		err = r.db.GetContext(ctx, &stats, `
			SELECT * FROM semantic.aso_daily_stats
			WHERE env = $1 AND tenant_id = $2 AND stat_date = $3
		`, env, *tenantID, date.Format("2006-01-02"))
	} else {
		err = r.db.GetContext(ctx, &stats, `
			SELECT * FROM semantic.aso_daily_stats
			WHERE env = $1 AND tenant_id IS NULL AND stat_date = $2
		`, env, date.Format("2006-01-02"))
	}

	if err == sql.ErrNoRows {
		return &ASODailyStats{Env: env, TenantID: tenantID, StatDate: date}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	return &stats, nil
}

// IncrementDailyStats increments daily stats counters
func (r *asoOptimizationRepo) IncrementDailyStats(ctx context.Context, env string, tenantID *uuid.UUID, field string) error {
	query := fmt.Sprintf(`
		INSERT INTO semantic.aso_daily_stats (env, tenant_id, stat_date, %s)
		VALUES ($1, $2, CURRENT_DATE, 1)
		ON CONFLICT (env, tenant_id, stat_date) DO UPDATE
		SET %s = semantic.aso_daily_stats.%s + 1
	`, field, field, field)

	_, err := r.db.ExecContext(ctx, query, env, tenantID)
	if err != nil {
		return fmt.Errorf("failed to increment daily stats: %w", err)
	}

	return nil
}

// recordAudit records an audit entry
func (r *asoOptimizationRepo) recordAudit(ctx context.Context, optID uuid.UUID, action, actor string, details map[string]string) {
	detailsJSON, _ := json.Marshal(details)
	_, _ = r.db.ExecContext(ctx, `
		INSERT INTO semantic.aso_optimization_audit (optimization_id, action, actor, details)
		VALUES ($1, $2, $3, $4)
	`, optID, action, actor, detailsJSON)
}
