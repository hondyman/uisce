package cbo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// DBSLOProvider provides SLO constraints from the database
type DBSLOProvider struct {
	db *sqlx.DB
}

// NewDBSLOProvider creates a new database-backed SLO provider
func NewDBSLOProvider(db *sqlx.DB) *DBSLOProvider {
	return &DBSLOProvider{db: db}
}

// ForBO returns the SLO constraints for a business object
func (p *DBSLOProvider) ForBO(ctx context.Context, env string, tenantID *uuid.UUID, boName string) *QuerySLO {
	slo := &QuerySLO{}
	hasConstraints := false

	// Get latency SLO
	latency, err := p.getSLOTarget(ctx, env, tenantID, "bo", boName, "latency")
	if err == nil && latency != nil {
		slo.MaxP95LatencyMs = latency
		hasConstraints = true
	}

	// Get freshness SLO
	freshness, err := p.getSLOTarget(ctx, env, tenantID, "bo", boName, "freshness")
	if err == nil && freshness != nil {
		slo.MaxFreshnessLagSec = freshness
		hasConstraints = true
	}

	// Get error rate SLO
	errorRate, err := p.getSLOTarget(ctx, env, tenantID, "bo", boName, "error_rate")
	if err == nil && errorRate != nil {
		slo.MaxErrorRate = errorRate
		hasConstraints = true
	}

	if !hasConstraints {
		return nil
	}

	return slo
}

// ForPage returns the SLO constraints for a specific page
func (p *DBSLOProvider) ForPage(ctx context.Context, env string, tenantID *uuid.UUID, pageSlug string) *QuerySLO {
	slo := &QuerySLO{}
	hasConstraints := false

	// Get latency SLO
	latency, err := p.getSLOTarget(ctx, env, tenantID, "page", pageSlug, "latency")
	if err == nil && latency != nil {
		slo.MaxP95LatencyMs = latency
		hasConstraints = true
	}

	// Get error rate SLO
	errorRate, err := p.getSLOTarget(ctx, env, tenantID, "page", pageSlug, "error_rate")
	if err == nil && errorRate != nil {
		slo.MaxErrorRate = errorRate
		hasConstraints = true
	}

	if !hasConstraints {
		return nil
	}

	return slo
}

// getSLOTarget retrieves a specific SLO target from the database
func (p *DBSLOProvider) getSLOTarget(ctx context.Context, env string, tenantID *uuid.UUID, scopeType, scopeID, sloType string) (*float64, error) {
	query := `
		SELECT target
		FROM semantic_slos
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND scope_type = $3
		  AND scope_id = $4
		  AND slo_type = $5
		  AND enabled = true
		ORDER BY tenant_id NULLS LAST
		LIMIT 1
	`

	var target float64
	err := p.db.QueryRowxContext(ctx, query, env, tenantID, scopeType, scopeID, sloType).Scan(&target)
	if err != nil {
		return nil, err
	}

	return &target, nil
}

// ListSLOs returns all SLOs for a given scope
func (p *DBSLOProvider) ListSLOs(ctx context.Context, env string, tenantID *uuid.UUID, scopeType, scopeID string) ([]SLODefinition, error) {
	query := `
		SELECT id, env, tenant_id, scope_type, scope_id, slo_type, target, time_window, enabled, created_at, updated_at
		FROM semantic_slos
		WHERE env = $1
		  AND (tenant_id = $2 OR tenant_id IS NULL)
		  AND ($3 = '' OR scope_type = $3)
		  AND ($4 = '' OR scope_id = $4)
		ORDER BY scope_type, scope_id, slo_type
	`

	var slos []SLODefinition
	err := p.db.SelectContext(ctx, &slos, query, env, tenantID, scopeType, scopeID)
	if err != nil {
		return nil, err
	}

	return slos, nil
}

// CreateSLO creates a new SLO definition
func (p *DBSLOProvider) CreateSLO(ctx context.Context, slo *SLODefinition) error {
	query := `
		INSERT INTO semantic_slos (env, tenant_id, scope_type, scope_id, slo_type, target, time_window, enabled, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	return p.db.QueryRowxContext(ctx, query,
		slo.Env, slo.TenantID, slo.ScopeType, slo.ScopeID, slo.SLOType,
		slo.Target, slo.TimeWindow, slo.Enabled, slo.CreatedBy,
	).Scan(&slo.ID, &slo.CreatedAt, &slo.UpdatedAt)
}

// UpdateSLO updates an existing SLO definition
func (p *DBSLOProvider) UpdateSLO(ctx context.Context, slo *SLODefinition) error {
	query := `
		UPDATE semantic_slos
		SET target = $2, time_window = $3, enabled = $4, updated_at = NOW()
		WHERE id = $1
	`

	_, err := p.db.ExecContext(ctx, query, slo.ID, slo.Target, slo.TimeWindow, slo.Enabled)
	return err
}

// DeleteSLO deletes an SLO definition
func (p *DBSLOProvider) DeleteSLO(ctx context.Context, id uuid.UUID) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM semantic_slos WHERE id = $1", id)
	return err
}

// SLODefinition represents an SLO definition from the database
type SLODefinition struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	Env        string     `json:"env" db:"env"`
	TenantID   *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	ScopeType  string     `json:"scope_type" db:"scope_type"`   // 'bo' | 'preagg' | 'entitlement' | 'planner'
	ScopeID    string     `json:"scope_id" db:"scope_id"`       // e.g. 'Positions'
	SLOType    string     `json:"slo_type" db:"slo_type"`       // 'latency' | 'freshness' | 'error_rate' | 'preagg_hit_rate'
	Target     float64    `json:"target" db:"target"`           // Target value
	TimeWindow string     `json:"time_window" db:"time_window"` // '7d', '30d'
	Enabled    bool       `json:"enabled" db:"enabled"`
	CreatedBy  *string    `json:"created_by,omitempty" db:"created_by"`
	CreatedAt  string     `json:"created_at" db:"created_at"`
	UpdatedAt  string     `json:"updated_at" db:"updated_at"`
}

// SLOEvaluation represents an SLO evaluation result
type SLOEvaluation struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	SLOID         uuid.UUID  `json:"slo_id" db:"slo_id"`
	Env           string     `json:"env" db:"env"`
	TenantID      *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	ScopeType     string     `json:"scope_type" db:"scope_type"`
	ScopeID       string     `json:"scope_id" db:"scope_id"`
	WindowStart   string     `json:"window_start" db:"window_start"`
	WindowEnd     string     `json:"window_end" db:"window_end"`
	MeasuredValue float64    `json:"measured_value" db:"measured_value"`
	TargetValue   float64    `json:"target_value" db:"target_value"`
	Status        string     `json:"status" db:"status"` // 'met' | 'violated'
	DeltaPercent  *float64   `json:"delta_percent,omitempty" db:"delta_percent"`
	CreatedAt     string     `json:"created_at" db:"created_at"`
}

// SLOViolation represents an SLO violation
type SLOViolation struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	SLOID          uuid.UUID  `json:"slo_id" db:"slo_id"`
	EvaluationID   *uuid.UUID `json:"evaluation_id,omitempty" db:"evaluation_id"`
	Env            string     `json:"env" db:"env"`
	TenantID       *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	ScopeType      string     `json:"scope_type" db:"scope_type"`
	ScopeID        string     `json:"scope_id" db:"scope_id"`
	SLOType        string     `json:"slo_type" db:"slo_type"`
	TargetValue    float64    `json:"target_value" db:"target_value"`
	ActualValue    float64    `json:"actual_value" db:"actual_value"`
	Severity       string     `json:"severity" db:"severity"`
	Acknowledged   bool       `json:"acknowledged" db:"acknowledged"`
	AcknowledgedBy *string    `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	AcknowledgedAt *string    `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	Resolved       bool       `json:"resolved" db:"resolved"`
	ResolvedAt     *string    `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt      string     `json:"created_at" db:"created_at"`
}
