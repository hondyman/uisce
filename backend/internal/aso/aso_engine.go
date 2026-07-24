package aso

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ASOEngine is the brain of Autonomous Semantic Optimization
type ASOEngine interface {
	// EvaluateTenant evaluates optimizations for a single tenant
	EvaluateTenant(ctx context.Context, env, tenantID string) ([]ASOOptimization, error)

	// EvaluateAllTenants evaluates all tenants in an environment
	EvaluateAllTenants(ctx context.Context, env string) ([]ASOOptimization, error)

	// EvaluateCore evaluates core-level (gold copy) optimizations
	EvaluateCore(ctx context.Context, env string) ([]ASOOptimization, error)

	// ApplyOptimization applies a specific optimization
	ApplyOptimization(ctx context.Context, optID uuid.UUID, actor string) error

	// ValidateChangeSet validates a changeset for performance regressions
	ValidateChangeSet(ctx context.Context, changeSetID uuid.UUID) (*ASOValidationResult, error)

	// GetSummary returns dashboard summary for an environment
	GetSummary(ctx context.Context, env string) (*ASOSummary, error)
}

// asoEngine implements ASOEngine
type asoEngine struct {
	db               *sqlx.DB
	policyStore      ASOPolicyStore
	optimizationRepo ASOOptimizationRepository
}

// NewASOEngine creates a new ASO engine
func NewASOEngine(
	db *sqlx.DB,
	policyStore ASOPolicyStore,
	optimizationRepo ASOOptimizationRepository,
) ASOEngine {
	return &asoEngine{
		db:               db,
		policyStore:      policyStore,
		optimizationRepo: optimizationRepo,
	}
}

// EvaluateTenant evaluates optimizations for a single tenant
func (e *asoEngine) EvaluateTenant(ctx context.Context, env, tenantID string) ([]ASOOptimization, error) {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	// Get effective policy for this tenant
	policy, err := e.policyStore.GetPolicy(ctx, env, &tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if !policy.Enabled {
		return nil, nil // ASO disabled for this tenant
	}

	var allOpts []ASOOptimization

	// 1. Analyze workload and find hot paths
	profiles, err := e.analyzeWorkload(ctx, env, tenantID, policy.LookbackWindow())
	if err != nil {
		return nil, fmt.Errorf("failed to analyze workload: %w", err)
	}

	for _, profile := range profiles {
		// 2. Check for hot paths without pre-agg coverage
		if profile.P95DurationMs > float64(policy.HotPathThresholdMs) && profile.PreAggMissRate > 0.5 {
			opt := e.buildCreatePreAggOptimization(env, &tenantUUID, policy, profile)
			if opt != nil && opt.Score >= policy.MinScoreForNewPreAgg {
				allOpts = append(allOpts, *opt)
			}
		}
	}

	// 3. Evaluate existing pre-aggs for tuning
	tuningOpts, err := e.evaluatePreAggTuning(ctx, env, &tenantUUID, policy)
	if err == nil {
		allOpts = append(allOpts, tuningOpts...)
	}

	// 4. Check for retirement candidates
	retirementOpts, err := e.evaluateRetirementCandidates(ctx, env, &tenantUUID, policy)
	if err == nil {
		allOpts = append(allOpts, retirementOpts...)
	}

	// 5. Evaluate pre-warm opportunities
	if policy.PrewarmEnabled {
		prewarmOpts, err := e.evaluatePrewarmOpportunities(ctx, env, &tenantUUID, policy, profiles)
		if err == nil {
			allOpts = append(allOpts, prewarmOpts...)
		}
	}

	// 6. Persist optimizations
	for i := range allOpts {
		allOpts[i].PolicyID = &policy.ID
		if err := e.optimizationRepo.Create(ctx, &allOpts[i]); err != nil {
			// Log but continue
			continue
		}
	}

	// 7. Auto-apply if policy allows
	if policy.CanAutoTune() || policy.CanAutoApply() {
		e.autoApplyOptimizations(ctx, allOpts, policy, "aso_engine")
	}

	return allOpts, nil
}

// EvaluateAllTenants evaluates all tenants in an environment
func (e *asoEngine) EvaluateAllTenants(ctx context.Context, env string) ([]ASOOptimization, error) {
	// Get all tenant IDs in this environment
	tenantIDs, err := e.getTenantIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant IDs: %w", err)
	}

	var allOpts []ASOOptimization
	for _, tenantID := range tenantIDs {
		opts, err := e.EvaluateTenant(ctx, env, tenantID)
		if err != nil {
			// Log and continue
			continue
		}
		allOpts = append(allOpts, opts...)
	}

	return allOpts, nil
}

// EvaluateCore evaluates core-level (gold copy) optimizations
func (e *asoEngine) EvaluateCore(ctx context.Context, env string) ([]ASOOptimization, error) {
	policy, err := e.policyStore.GetCorePolicy(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("failed to get core policy: %w", err)
	}

	if !policy.Enabled {
		return nil, nil
	}

	var allOpts []ASOOptimization

	// Core-level analysis: aggregate across tenants for core BOs
	coreBOs, err := e.getCoreBOs(ctx, env)
	if err != nil {
		return nil, err
	}

	for _, bo := range coreBOs {
		// Aggregate workload across all tenants for this core BO
		aggregatedProfile := e.aggregateWorkloadForCoreBO(ctx, env, bo, policy.LookbackWindow())
		if aggregatedProfile == nil {
			continue
		}

		// If high usage across tenants, propose core pre-agg
		if aggregatedProfile.QueriesPerDay > 100 && aggregatedProfile.P95DurationMs > float64(policy.HotPathThresholdMs) {
			opt := e.buildCorePreAggOptimization(env, policy, bo, aggregatedProfile)
			if opt != nil {
				allOpts = append(allOpts, *opt)
			}
		}
	}

	// Persist
	for i := range allOpts {
		allOpts[i].PolicyID = &policy.ID
		if err := e.optimizationRepo.Create(ctx, &allOpts[i]); err != nil {
			continue
		}
	}

	return allOpts, nil
}

// ApplyOptimization applies a specific optimization
func (e *asoEngine) ApplyOptimization(ctx context.Context, optID uuid.UUID, actor string) error {
	opt, err := e.optimizationRepo.GetByID(ctx, optID)
	if err != nil {
		return fmt.Errorf("failed to get optimization: %w", err)
	}
	if opt == nil {
		return fmt.Errorf("optimization not found")
	}

	// Check status
	if opt.Status != OptStatusProposed && opt.Status != OptStatusApproved {
		return fmt.Errorf("optimization cannot be applied in status: %s", opt.Status)
	}

	// Get policy for safety check
	policy, err := e.policyStore.GetPolicy(ctx, opt.Env, opt.TenantID)
	if err != nil {
		return fmt.Errorf("failed to get policy: %w", err)
	}

	// Check rate limits
	stats, err := e.optimizationRepo.GetDailyStats(ctx, opt.Env, opt.TenantID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to get daily stats: %w", err)
	}

	if opt.OptimizationType == OptTypeCreatePreAgg && stats.PreAggsCreated >= policy.MaxNewPreAggsPerDay {
		return fmt.Errorf("daily limit reached for new pre-aggs (%d/%d)", stats.PreAggsCreated, policy.MaxNewPreAggsPerDay)
	}
	if stats.ChangesApplied >= policy.MaxChangesPerDay {
		return fmt.Errorf("daily limit reached for changes (%d/%d)", stats.ChangesApplied, policy.MaxChangesPerDay)
	}

	// Apply based on type
	var afterConfig json.RawMessage
	switch opt.OptimizationType {
	case OptTypeTuneRefresh:
		afterConfig, err = e.applyTuneRefresh(ctx, opt)
	case OptTypeCreatePreAgg:
		afterConfig, err = e.applyCreatePreAgg(ctx, opt, policy)
	case OptTypeRetireAsset:
		afterConfig, err = e.applyRetireAsset(ctx, opt)
	case OptTypePrewarm:
		afterConfig, err = e.applyPrewarm(ctx, opt)
	default:
		return fmt.Errorf("unsupported optimization type: %s", opt.OptimizationType)
	}

	if err != nil {
		e.optimizationRepo.UpdateStatus(ctx, optID, OptStatusFailed, actor, err.Error())
		return fmt.Errorf("failed to apply optimization: %w", err)
	}

	// Mark as applied
	if err := e.optimizationRepo.MarkApplied(ctx, optID, actor, afterConfig); err != nil {
		return fmt.Errorf("failed to mark applied: %w", err)
	}

	// Increment stats
	e.optimizationRepo.IncrementDailyStats(ctx, opt.Env, opt.TenantID, "changes_applied")
	if opt.OptimizationType == OptTypeCreatePreAgg {
		e.optimizationRepo.IncrementDailyStats(ctx, opt.Env, opt.TenantID, "preaggs_created")
	}

	// Supersede older optimizations for same target
	e.optimizationRepo.SupersedeOptimization(ctx, opt.TargetType, opt.TargetID, optID)

	return nil
}

// ValidateChangeSet validates a changeset for performance regressions
func (e *asoEngine) ValidateChangeSet(ctx context.Context, changeSetID uuid.UUID) (*ASOValidationResult, error) {
	result := &ASOValidationResult{Valid: true}

	// Get changeset items (BOs, calcs, pre-aggs being modified)
	// For each item, check for performance impacts

	// Example checks:
	// 1. Is a heavily-used pre-agg being deleted?
	// 2. Is a BO change invalidating a pre-agg?
	// 3. Is a calculation becoming more expensive?

	// This is a placeholder - integrate with your changeset service
	return result, nil
}

// GetSummary returns dashboard summary for an environment
func (e *asoEngine) GetSummary(ctx context.Context, env string) (*ASOSummary, error) {
	policy, err := e.policyStore.GetCorePolicy(ctx, env)
	if err != nil {
		return nil, err
	}

	summary := &ASOSummary{
		Env:           env,
		PolicyEnabled: policy.Enabled,
		PolicyMode:    policy.Mode,
	}

	// Count optimizations
	proposed := OptStatusProposed
	filter := OptimizationFilter{Env: &env, Status: &proposed}
	pending, _ := e.optimizationRepo.List(ctx, filter)
	summary.OptimizationsPending = len(pending)

	// Count today's optimizations
	var todayCount int
	e.db.GetContext(ctx, &todayCount, `
		SELECT COUNT(*) FROM semantic.aso_optimization
		WHERE env = $1 AND DATE(created_at) = CURRENT_DATE
	`, env)
	summary.OptimizationsToday = todayCount

	// Count applied in last 7 days
	var appliedCount int
	e.db.GetContext(ctx, &appliedCount, `
		SELECT COUNT(*) FROM semantic.aso_optimization
		WHERE env = $1 AND status = 'applied' 
		AND applied_at >= NOW() - INTERVAL '7 days'
	`, env)
	summary.OptimizationsApplied7d = appliedCount

	return summary, nil
}

// ============================================================================
// Internal Methods
// ============================================================================

func (e *asoEngine) analyzeWorkload(ctx context.Context, env, tenantID string, lookback time.Duration) ([]WorkloadProfile, error) {
	// Query telemetry to build workload profiles per BO
	// This assumes a query_telemetry table exists

	var profiles []WorkloadProfile
	query := `
		WITH bo_stats AS (
			SELECT 
				bo_id,
				bo_name,
				COUNT(*) as total_queries,
				AVG(duration_ms) as avg_duration,
				PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration,
				AVG(rows_scanned) as avg_rows
			FROM query_telemetry
			WHERE tenant_id = $1
			AND executed_at >= NOW() - $2::interval
			GROUP BY bo_id, bo_name
		)
		SELECT * FROM bo_stats
		WHERE total_queries > 10
		ORDER BY total_queries DESC
	`

	// Execute query and scan into profiles
	// This is a simplified version - actual implementation would need proper telemetry table
	_ = query

	return profiles, nil
}

func (e *asoEngine) buildCreatePreAggOptimization(env string, tenantID *uuid.UUID, policy *ASOPolicy, profile WorkloadProfile) *ASOOptimization {
	// Calculate score based on workload metrics
	score := e.calculateOptimizationScore(profile)

	if score < policy.MinScoreForNewPreAgg {
		return nil
	}

	details := CreatePreAggDetails{
		BOName:   profile.BOName,
		Grain:    profile.HotGrains,
		Measures: profile.HotMeasures,
	}
	details.CostEstimate.EstimatedQueriesPerDay = profile.QueriesPerDay
	details.CostEstimate.AvgDurationMs = profile.AvgDurationMs
	details.CostEstimate.P95DurationMs = profile.P95DurationMs
	details.CostEstimate.AvgRowsScanned = profile.AvgRowsScanned
	details.CostEstimate.EstimatedSpeedupFactor = profile.P95DurationMs / 100 // Rough estimate

	detailsJSON, _ := json.Marshal(details)

	qpd := profile.QueriesPerDay
	avgLat := profile.AvgDurationMs
	p95Lat := profile.P95DurationMs

	return &ASOOptimization{
		ID:                 uuid.New(),
		Env:                env,
		TenantID:           tenantID,
		Scope:              ASOScopeTenant,
		OptimizationType:   OptTypeCreatePreAgg,
		TargetType:         TargetTypeBO,
		TargetID:           profile.BOID,
		TargetName:         profile.BOName,
		Status:             OptStatusProposed,
		Mode:               string(policy.Mode),
		Score:              score,
		Reason:             fmt.Sprintf("High latency (p95: %.0fms) with high query volume (%.0f/day) and low pre-agg coverage", profile.P95DurationMs, profile.QueriesPerDay),
		Details:            detailsJSON,
		WorkloadWindowDays: profile.WindowDays,
		QueriesPerDay:      &qpd,
		AvgLatencyMs:       &avgLat,
		P95LatencyMs:       &p95Lat,
		CreatedBy:          "aso_engine",
	}
}

func (e *asoEngine) evaluatePreAggTuning(ctx context.Context, env string, tenantID *uuid.UUID, policy *ASOPolicy) ([]ASOOptimization, error) {
	// Query pre-aggs and their usage stats
	// Find opportunities to tune refresh intervals
	var opts []ASOOptimization

	// This would query:
	// - Pre-aggs with high usage but infrequent refresh → tighten
	// - Pre-aggs with low usage but frequent refresh → loosen
	// - Pre-aggs with high refresh failures → flag for investigation

	return opts, nil
}

func (e *asoEngine) evaluateRetirementCandidates(ctx context.Context, env string, tenantID *uuid.UUID, policy *ASOPolicy) ([]ASOOptimization, error) {
	// Find unused assets (pre-aggs, BOs, calcs) for retirement
	var opts []ASOOptimization

	// Query for assets with no usage in policy.MinUsageForRetirement period

	return opts, nil
}

func (e *asoEngine) evaluatePrewarmOpportunities(ctx context.Context, env string, tenantID *uuid.UUID, policy *ASOPolicy, profiles []WorkloadProfile) ([]ASOOptimization, error) {
	var opts []ASOOptimization

	for _, profile := range profiles {
		if len(profile.PeakHours) == 0 {
			continue
		}

		// Suggest pre-warm schedule based on peak patterns
		// e.g., if peak at 9am, suggest refresh at 8:45am
	}

	return opts, nil
}

func (e *asoEngine) calculateOptimizationScore(profile WorkloadProfile) float64 {
	// Score = (benefit) / (cost)
	// Benefit: latency reduction * query volume
	// Cost: storage + refresh overhead

	benefit := profile.P95DurationMs * profile.QueriesPerDay
	cost := float64(profile.AvgRowsScanned) / 1000000 // Rough storage estimate

	if cost == 0 {
		cost = 1
	}

	return benefit / cost / 1000 // Normalize
}

func (e *asoEngine) autoApplyOptimizations(ctx context.Context, opts []ASOOptimization, policy *ASOPolicy, actor string) {
	for _, opt := range opts {
		// Only auto-apply safe operations based on policy mode
		canApply := false

		switch policy.Mode {
		case ASOModeAutoApply:
			canApply = true
		case ASOModeAutoTune:
			canApply = opt.OptimizationType == OptTypeTuneRefresh || opt.OptimizationType == OptTypePrewarm
		}

		if canApply {
			e.ApplyOptimization(ctx, opt.ID, actor)
		}
	}
}

func (e *asoEngine) applyTuneRefresh(ctx context.Context, opt *ASOOptimization) (json.RawMessage, error) {
	// Update pre-agg refresh interval
	// This would call your pre-agg service
	return json.RawMessage(`{"applied": true}`), nil
}

func (e *asoEngine) applyCreatePreAgg(ctx context.Context, opt *ASOOptimization, policy *ASOPolicy) (json.RawMessage, error) {
	// Create new pre-agg in draft state
	// In prod, require approval to activate
	// This would call your pre-agg service
	return json.RawMessage(`{"preagg_id": "new-preagg-id", "status": "draft"}`), nil
}

func (e *asoEngine) applyRetireAsset(ctx context.Context, opt *ASOOptimization) (json.RawMessage, error) {
	// Mark asset as deprecated
	// Stop scheduling refreshes
	// This would call your asset service
	return json.RawMessage(`{"retired": true}`), nil
}

func (e *asoEngine) applyPrewarm(ctx context.Context, opt *ASOOptimization) (json.RawMessage, error) {
	// Update pre-agg with pre-warm schedule
	// This would call your scheduler service
	return json.RawMessage(`{"prewarm_scheduled": true}`), nil
}

func (e *asoEngine) getTenantIDs(ctx context.Context) ([]string, error) {
	var ids []string
	err := e.db.SelectContext(ctx, &ids, `SELECT id::text FROM public.tenants`)
	return ids, err
}

func (e *asoEngine) getCoreBOs(ctx context.Context, env string) ([]struct {
	ID   uuid.UUID
	Name string
}, error) {
	// Return core BOs (not tenant-specific)
	var bos []struct {
		ID   uuid.UUID
		Name string
	}
	return bos, nil
}

func (e *asoEngine) buildCorePreAggOptimization(env string, policy *ASOPolicy, bo struct {
	ID   uuid.UUID
	Name string
}, profile *WorkloadProfile) *ASOOptimization {
	// Build optimization for core pre-agg
	return nil
}

func (e *asoEngine) aggregateWorkloadForCoreBO(ctx context.Context, env string, bo struct {
	ID   uuid.UUID
	Name string
}, lookback time.Duration) *WorkloadProfile {
	// Aggregate workload across all tenants for a core BO
	return nil
}
