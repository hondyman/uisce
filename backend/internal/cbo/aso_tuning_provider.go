package cbo

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ASOTuningHints provides tuning hints for ASO based on SLO status
type ASOTuningHints struct {
	PriorityBoost     float64 `json:"priority_boost"`     // >1.0 for hot BOs
	MaxAggressiveness float64 `json:"max_aggressiveness"` // 0-1 scale
	AutoApplyEnabled  bool    `json:"auto_apply_enabled"`
	Reason            string  `json:"reason,omitempty"`
	Source            string  `json:"source,omitempty"` // 'slo_violation' | 'manual' | 'aso_recommendation'
}

// ASOTuningProvider provides ASO tuning hints based on SLO status
type ASOTuningProvider interface {
	HintsForBO(ctx context.Context, env string, tenantID *uuid.UUID, boName string) ASOTuningHints
}

// DBASOTuningProvider provides tuning hints from the database
type DBASOTuningProvider struct {
	db *sqlx.DB
}

// NewDBASOTuningProvider creates a new database-backed ASO tuning provider
func NewDBASOTuningProvider(db *sqlx.DB) *DBASOTuningProvider {
	return &DBASOTuningProvider{db: db}
}

// HintsForBO returns tuning hints for a business object
func (p *DBASOTuningProvider) HintsForBO(ctx context.Context, env string, tenantID *uuid.UUID, boName string) ASOTuningHints {
	// Default hints
	hints := ASOTuningHints{
		PriorityBoost:     1.0,
		MaxAggressiveness: 1.0,
		AutoApplyEnabled:  true,
	}

	query := `
		SELECT priority_boost, max_aggressiveness, auto_apply_enabled, reason, source
		FROM aso_tuning_hints
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND bo_name = $3
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY tenant_id NULLS LAST
		LIMIT 1
	`

	err := p.db.QueryRowxContext(ctx, query, env, tenantID, boName).Scan(
		&hints.PriorityBoost, &hints.MaxAggressiveness, &hints.AutoApplyEnabled,
		&hints.Reason, &hints.Source,
	)
	if err != nil {
		// Return defaults if no hints found
		return hints
	}

	return hints
}

// SetHints sets tuning hints for a BO (upsert)
func (p *DBASOTuningProvider) SetHints(ctx context.Context, env string, tenantID *uuid.UUID, boName string, hints ASOTuningHints, expiresIn *time.Duration) error {
	var expiresAt *time.Time
	if expiresIn != nil {
		t := time.Now().Add(*expiresIn)
		expiresAt = &t
	}

	query := `
		INSERT INTO aso_tuning_hints (env, tenant_id, bo_name, priority_boost, max_aggressiveness, auto_apply_enabled, reason, source, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (env, tenant_id, bo_name) DO UPDATE SET
			priority_boost = EXCLUDED.priority_boost,
			max_aggressiveness = EXCLUDED.max_aggressiveness,
			auto_apply_enabled = EXCLUDED.auto_apply_enabled,
			reason = EXCLUDED.reason,
			source = EXCLUDED.source,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
	`

	_, err := p.db.ExecContext(ctx, query, env, tenantID, boName,
		hints.PriorityBoost, hints.MaxAggressiveness, hints.AutoApplyEnabled,
		hints.Reason, hints.Source, expiresAt)
	return err
}

// ClearHints removes tuning hints for a BO
func (p *DBASOTuningProvider) ClearHints(ctx context.Context, env string, tenantID *uuid.UUID, boName string) error {
	query := `
		DELETE FROM aso_tuning_hints
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND bo_name = $3
	`
	_, err := p.db.ExecContext(ctx, query, env, tenantID, boName)
	return err
}

// HandleSLOViolation handles an SLO violation by adjusting tuning hints
func (p *DBASOTuningProvider) HandleSLOViolation(ctx context.Context, violation *SLOViolation) error {
	log.Printf("[aso_tuning] Handling SLO violation: scope=%s/%s, type=%s, target=%.2f, actual=%.2f",
		violation.ScopeType, violation.ScopeID, violation.SLOType, violation.TargetValue, violation.ActualValue)

	// Handle page-scoped violations by boosting all associated BOs
	if violation.ScopeType == "page" {
		return p.HandlePageSLOViolation(ctx, violation)
	}

	// Handle api-scoped violations
	if violation.ScopeType == "api" {
		return p.HandleApiSLOViolation(ctx, violation)
	}

	// Only handle BO-scoped violations otherwise
	if violation.ScopeType != "bo" {
		return nil
	}

	// Calculate new hints based on violation severity
	severity := violation.ActualValue / violation.TargetValue // >1 means violated

	hints := ASOTuningHints{
		Source: "slo_violation",
	}

	switch violation.SLOType {
	case "latency":
		// Increase priority and aggressiveness for slow BOs
		hints.PriorityBoost = 1.0 + (severity-1.0)*0.5 // Up to 1.5x boost
		if hints.PriorityBoost > 2.0 {
			hints.PriorityBoost = 2.0
		}
		hints.MaxAggressiveness = 1.0 // Allow full aggressiveness
		hints.AutoApplyEnabled = true
		hints.Reason = "Latency SLO violated - boosting pre-agg priority"

	case "error_rate":
		// Be more conservative for high error rates
		hints.PriorityBoost = 1.0
		hints.MaxAggressiveness = 0.5  // Reduce aggressiveness
		hints.AutoApplyEnabled = false // Disable auto-apply
		hints.Reason = "Error rate SLO violated - reducing aggressiveness"

	case "freshness":
		// Increase refresh frequency priority
		hints.PriorityBoost = 1.2
		hints.MaxAggressiveness = 0.8
		hints.AutoApplyEnabled = true
		hints.Reason = "Freshness SLO violated - adjusting refresh priority"

	case "preagg_hit_rate":
		// Increase priority to create more pre-aggs
		hints.PriorityBoost = 1.5
		hints.MaxAggressiveness = 1.0
		hints.AutoApplyEnabled = true
		hints.Reason = "Pre-agg hit rate SLO violated - suggesting new pre-aggs"

	default:
		return nil
	}

	// Set hints with 24-hour expiration
	expiry := 24 * time.Hour
	return p.SetHints(ctx, violation.Env, violation.TenantID, violation.ScopeID, hints, &expiry)
}

// HandlePageSLOViolation handles a page-scoped SLO violation
func (p *DBASOTuningProvider) HandlePageSLOViolation(ctx context.Context, v *SLOViolation) error {
	boNames, err := p.identifyBOsForPage(ctx, v.Env, v.TenantID, v.ScopeID)
	if err != nil {
		return err
	}

	for _, bo := range boNames {
		hints := ASOTuningHints{
			PriorityBoost:     1.5,
			MaxAggressiveness: 1.0,
			AutoApplyEnabled:  true,
			Source:            "page_slo_violation",
			Reason:            "Page SLO violation - boosting all constituent BOs",
		}
		expiry := 12 * time.Hour
		if err := p.SetHints(ctx, v.Env, v.TenantID, bo, hints, &expiry); err != nil {
			log.Printf("[aso_tuning] Error boosting BO %s for page %s: %v", bo, v.ScopeID, err)
		}
	}
	return nil
}

// HandleApiSLOViolation handles an API-scoped SLO violation
func (p *DBASOTuningProvider) HandleApiSLOViolation(ctx context.Context, v *SLOViolation) error {
	boName, err := p.identifyBOForApi(ctx, v.ScopeID)
	if err != nil {
		return err
	}

	hints := ASOTuningHints{
		PriorityBoost:     2.0, // Higher boost for API endpoints
		MaxAggressiveness: 1.0,
		AutoApplyEnabled:  true,
		Source:            "api_slo_violation",
		Reason:            "API SLO violation - boosting underlying BO",
	}
	expiry := 12 * time.Hour
	return p.SetHints(ctx, v.Env, v.TenantID, boName, hints, &expiry)
}

func (p *DBASOTuningProvider) identifyBOForApi(ctx context.Context, apiID string) (string, error) {
	var boName string
	err := p.db.GetContext(ctx, &boName, "SELECT bo_name FROM semantic.api_endpoints WHERE id=$1", apiID)
	return boName, err
}

// identifyBOsForPage identifies all BOs associated with a page via telemetry
func (p *DBASOTuningProvider) identifyBOsForPage(ctx context.Context, env string, tenantID *uuid.UUID, pageSlug string) ([]string, error) {
	query := `
		SELECT DISTINCT bo_name
		FROM planner_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND page_slug = $3
		  AND created_at > NOW() - INTERVAL '24 hours'
	`
	var boNames []string
	err := p.db.SelectContext(ctx, &boNames, query, env, tenantID, pageSlug)
	return boNames, err
}
