package finops

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PrewarmCoordinator orchestrates off-peak cache seeding ahead of predicted compute spikes.
// It queries the DemandForecaster for tomorrow's forecast, consults the active
// SmoothingPolicy, then seeds the top-N hot Business Objects into calc_cache via
// the vectorised financial kernel pipeline.
type PrewarmCoordinator struct {
	db            *sqlx.DB
	forecaster    *DemandForecaster
	policyService *SmoothingPolicyService

	inFlightMu sync.Mutex
	// In-process lock map. NOTE: Replace with pg_try_advisory_lock(hashtext(tenant::text))
	// when scaling across >1 replica.
	inFlight   map[uuid.UUID]bool
	clock      Clock
}

// NewPrewarmCoordinator constructs a PrewarmCoordinator with the production RealClock.
func NewPrewarmCoordinator(db *sqlx.DB) *PrewarmCoordinator {
	return NewPrewarmCoordinatorWithClock(db, RealClock{})
}

// NewPrewarmCoordinatorWithClock constructs a PrewarmCoordinator using the supplied
// clock (passed through to its inner DemandForecaster). Used by tests and by
// callers that need to share a single clock across multiple components.
func NewPrewarmCoordinatorWithClock(db *sqlx.DB, clock Clock) *PrewarmCoordinator {
	if clock == nil {
		clock = RealClock{}
	}
	return &PrewarmCoordinator{
		db:            db,
		forecaster:    NewDemandForecasterWithClock(db, clock),
		policyService: NewSmoothingPolicyService(db),
		inFlight:      make(map[uuid.UUID]bool),
		clock:         clock,
	}
}

// RecoverStalePendingExecutions cleans up orphan PENDING rows left over after a server crash/restart.
func (c *PrewarmCoordinator) RecoverStalePendingExecutions(ctx context.Context) error {
	if c.db == nil {
		return nil
	}
	query := `
		UPDATE finops.prewarm_execution_ledger
		SET status = 'FAILED',
		    error_detail = 'abandoned: process restart or timeout while PENDING',
		    completed_at = NOW()
		WHERE status = 'PENDING'
		  AND executed_at < NOW() - INTERVAL '15 minutes';
	`
	_, err := c.db.ExecContext(ctx, query)
	return err
}

// hotTarget represents a frequently-accessed Business Object in the calc_cache.
type hotTarget struct {
	BOID        uuid.UUID `db:"bo_id"`
	FieldID     uuid.UUID `db:"field_id"`
	HitCount    int64     `db:"hit_count"`
	FormulaType string    // resolved after query
}

// PrewarmResult summarises the outcome of an off-peak seeding run.
type PrewarmResult struct {
	TenantID                uuid.UUID  `json:"tenantId"`
	JobID                   *uuid.UUID `json:"jobId,omitempty"`
	Triggered               bool       `json:"triggered"`
	PeakProbability         float64    `json:"peakProbability"`
	TargetsIdentified       int        `json:"targetsIdentified"`
	TargetsSeeded           int        `json:"targetsSeeded"`
	ComputeCostIncurredUSD  float64    `json:"computeCostIncurredUsd"`
	EstimatedPeakSavingsUSD float64    `json:"estimatedPeakSavingsUsd"`
	Status                  string     `json:"status"`
}

// TryLockTenant attempts to acquire the in-flight lock for a tenant.
// Returns false if an execution is already in flight.
func (c *PrewarmCoordinator) TryLockTenant(tenantID uuid.UUID) bool {
	c.inFlightMu.Lock()
	defer c.inFlightMu.Unlock()
	if c.inFlight[tenantID] {
		return false
	}
	c.inFlight[tenantID] = true
	return true
}

// UnlockTenant releases the in-flight lock for a tenant.
func (c *PrewarmCoordinator) UnlockTenant(tenantID uuid.UUID) {
	c.inFlightMu.Lock()
	defer c.inFlightMu.Unlock()
	delete(c.inFlight, tenantID)
}

// CreatePendingExecution creates a synchronous PENDING row in the execution ledger.
// Acts as a persistent job marker that survives crashes.
func (c *PrewarmCoordinator) CreatePendingExecution(
	ctx context.Context,
	tenantID uuid.UUID,
	jobID uuid.UUID,
	triggeredBy *uuid.UUID,
) error {
	if c.db == nil {
		return nil
	}

	query := `
		INSERT INTO finops.prewarm_execution_ledger (
			tenant_id, job_id, triggered_by, target_metric,
			status, executed_at
		) VALUES ($1, $2, $3, 'ALL', 'PENDING', NOW());
	`
	_, err := c.db.ExecContext(ctx, query, tenantID, jobID, triggeredBy)
	if err != nil {
		return fmt.Errorf("failed creating pending execution ledger entry: %w", err)
	}
	return nil
}

// ExecuteOffPeakPrewarming evaluates tomorrow's forecast and seeds hot Business Objects.
// Updates the existing job record if jobID is provided, or inserts a new row.
func (c *PrewarmCoordinator) ExecuteOffPeakPrewarming(
	ctx context.Context,
	tenantID uuid.UUID,
	jobID *uuid.UUID,
) (*PrewarmResult, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	result := &PrewarmResult{
		TenantID: tenantID,
		JobID:    jobID,
	}

	// ── 1. Load active smoothing policy ────────────────────────────────────
	policy, err := c.policyService.GetActivePolicy(ctx, tenantID)
	if err != nil {
		result.Status = "FAILED"
		_ = c.updateOrPersistLedgerEntry(ctx, tenantID, jobID, nil, result, nil, err.Error())
		return nil, fmt.Errorf("failed loading smoothing policy: %w", err)
	}

	// ── 2. Forecast tomorrow ────────────────────────────────────────────────
	tomorrow := c.clock.Now().UTC().Add(24 * time.Hour)
	forecast, err := c.forecaster.GenerateTenantDemandForecast(ctx, tenantID, tomorrow)
	if err != nil {
		result.Status = "FAILED"
		_ = c.updateOrPersistLedgerEntry(ctx, tenantID, jobID, nil, result, policy, err.Error())
		return nil, fmt.Errorf("demand forecast failed: %w", err)
	}

	result.PeakProbability = forecast.PeakProbability

	// ── 3. Gate on policy threshold ────────────────────────────────────────
	// AND gate: Prewarming requires both peak probability threshold AND the projected multiplier
	// to exceed the configured policy thresholds.
	if forecast.PeakProbability < policy.MinPeakProbabilityToPrewarm ||
		forecast.ProjectedMultiplier < policy.PrewarmThresholdMultiplier {
		result.Triggered = false
		result.Status = "SKIPPED_BELOW_THRESHOLD"
		_ = c.updateOrPersistLedgerEntry(ctx, tenantID, jobID, nil, result, policy, "")
		return result, nil
	}

	// ── 4. Discover top hot Business Objects ────────────────────────────────
	targets, err := c.discoverHotTargets(ctx, tenantID)
	if err != nil {
		result.Status = "FAILED"
		_ = c.updateOrPersistLedgerEntry(ctx, tenantID, jobID, nil, result, policy, err.Error())
		return nil, fmt.Errorf("failed discovering hot targets: %w", err)
	}

	result.TargetsIdentified = len(targets)
	if len(targets) == 0 {
		result.Triggered = false
		result.Status = "SKIPPED_NO_TARGETS"
		// Record observable event in ledger: prewarm ran due to impending peak, but nothing hot needed seeding.
		_ = c.updateOrPersistLedgerEntry(ctx, tenantID, jobID, nil, result, policy, "")
		return result, nil
	}

	result.Triggered = true

	// ── 5. Seed each target via the vectorised kernel pipeline ──────────────
	var seedErrors []error
	for _, target := range targets {
		if err := c.seedMetricCache(ctx, tenantID, target.BOID, target.FieldID, target.FormulaType); err != nil {
			seedErrors = append(seedErrors, err)
		} else {
			result.TargetsSeeded++
		}
	}

	// Honest metrics: While seedMetricCache is a simulated stub, avoid recording fabricated costs.
	result.ComputeCostIncurredUSD = 0.0
	result.EstimatedPeakSavingsUSD = 0.0

	var errorDetail string
	if len(seedErrors) > 0 && result.TargetsSeeded == 0 {
		result.Status = "FAILED"
		errorDetail = fmt.Sprintf("all seed targets failed: %v", seedErrors[0])
	} else if len(seedErrors) > 0 {
		result.Status = "PARTIAL"
		errorDetail = fmt.Sprintf("%d targets failed during seeding", len(seedErrors))
	} else {
		// Mark as SIMULATED to preserve audit integrity until live Arrow kernel execution is wired.
		result.Status = "SIMULATED"
	}

	// ── 6. Persist/update the prewarm execution ledger ──────────────────────
	_ = c.updateOrPersistLedgerEntry(ctx, tenantID, jobID, targets, result, policy, errorDetail)

	return result, nil
}

// discoverHotTargets finds the top-5 most frequently accessed (bo_id, field_id) pairs
// from calc_cache for the given tenant.
func (c *PrewarmCoordinator) discoverHotTargets(
	ctx context.Context,
	tenantID uuid.UUID,
) ([]hotTarget, error) {
	if c.db == nil {
		return nil, nil
	}

	query := `
		SELECT
			c.bo_id,
			c.field_id,
			COUNT(1) AS hit_count
		FROM audit.analytical_query_execution_logs l
		JOIN calc_cache c
		  ON c.tenant_id = l.tenant_id
		 AND c.bo_id = l.bo_id
		WHERE l.tenant_id = $1
		  AND l.created_at >= NOW() - INTERVAL '7 days'
		GROUP BY c.bo_id, c.field_id
		ORDER BY hit_count DESC
		LIMIT 5;
	`
	var targets []hotTarget
	if err := c.db.SelectContext(ctx, &targets, query, tenantID); err != nil {
		return nil, fmt.Errorf("hot target query failed: %w", err)
	}

	for i := range targets {
		targets[i].FormulaType = "XIRR"
	}

	return targets, nil
}

// seedMetricCache executes an off-peak warm calculation.
func (c *PrewarmCoordinator) seedMetricCache(
	ctx context.Context,
	tenantID, boID, fieldID uuid.UUID,
	formulaType string,
) error {
	_ = ctx
	_ = tenantID
	_ = boID
	_ = fieldID
	_ = formulaType
	return nil
}

// updateOrPersistLedgerEntry updates an existing PENDING row (by job_id) or inserts a new row.
// When targets is empty, bo_id is set to NULL (clean audit representation without fake tenant IDs).
func (c *PrewarmCoordinator) updateOrPersistLedgerEntry(
	ctx context.Context,
	tenantID uuid.UUID,
	jobID *uuid.UUID,
	targets []hotTarget,
	result *PrewarmResult,
	policy *WorkloadSmoothingPolicy,
	errorDetail string,
) error {
	if c.db == nil {
		return nil
	}

	// Single source of truth for the (bo_id, target_metric) pair: both derive from
	// the same `targets` presence check so the two columns can never describe
	// different row states on the same write. Run-level = (NULL, 'ALL');
	// per-target = (first target's BOID, first target's FormulaType).
	var primaryBOID *uuid.UUID = nil
	targetMetric := "ALL"
	if len(targets) > 0 {
		b := targets[0].BOID
		primaryBOID = &b
		targetMetric = targets[0].FormulaType
	}

	var policyID any = nil
	if policy != nil && policy.PolicyID != uuid.Nil {
		policyID = policy.PolicyID
	}

	var errDetail any = nil
	if errorDetail != "" {
		errDetail = errorDetail
	}

	// Update existing PENDING row if jobID was provided
	if jobID != nil && *jobID != uuid.Nil {
		updateQuery := `
			UPDATE finops.prewarm_execution_ledger
			SET bo_id = $1,
			    target_metric = $2,
			    entities_prewarmed_count = $3,
			    compute_cost_incurred_usd = $4,
			    estimated_peak_savings_usd = $5,
			    peak_probability_at_trigger = $6,
			    policy_id = $7,
			    status = $8,
			    error_detail = $9,
			    completed_at = NOW()
			WHERE tenant_id = $10 AND job_id = $11;
		`
		res, err := c.db.ExecContext(ctx, updateQuery,
			primaryBOID,
			targetMetric,
			result.TargetsSeeded,
			result.ComputeCostIncurredUSD,
			result.EstimatedPeakSavingsUSD,
			result.PeakProbability,
			policyID,
			result.Status,
			errDetail,
			tenantID,
			*jobID,
		)
		if err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				return nil
			}
		}
	}

	// Otherwise insert a new terminal row
	insertQuery := `
		INSERT INTO finops.prewarm_execution_ledger (
			tenant_id, job_id, bo_id, target_metric,
			entities_prewarmed_count, compute_cost_incurred_usd, estimated_peak_savings_usd,
			peak_probability_at_trigger, policy_id, status, error_detail, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW());
	`
	_, err := c.db.ExecContext(ctx, insertQuery,
		tenantID,
		jobID,
		primaryBOID,
		targetMetric,
		result.TargetsSeeded,
		result.ComputeCostIncurredUSD,
		result.EstimatedPeakSavingsUSD,
		result.PeakProbability,
		policyID,
		result.Status,
		errDetail,
	)
	if err != nil {
		return fmt.Errorf("failed persisting prewarm ledger entry: %w", err)
	}
	return nil
}

// GetPrewarmExecutionByJobID retrieves the prewarm execution status for a specific tenant and job_id.
func (c *PrewarmCoordinator) GetPrewarmExecutionByJobID(
	ctx context.Context,
	tenantID uuid.UUID,
	jobID uuid.UUID,
) (*PrewarmResult, error) {
	if c.db == nil {
		return nil, nil
	}

	var row struct {
		TenantID                uuid.UUID  `db:"tenant_id"`
		JobID                   *uuid.UUID `db:"job_id"`
		EntitiesPrewarmedCount  int        `db:"entities_prewarmed_count"`
		ComputeCostIncurredUSD  float64    `db:"compute_cost_incurred_usd"`
		EstimatedPeakSavingsUSD float64    `db:"estimated_peak_savings_usd"`
		PeakProbability         float64    `db:"peak_probability_at_trigger"`
		Status                  string     `db:"status"`
	}

	query := `
		SELECT tenant_id, job_id, entities_prewarmed_count, compute_cost_incurred_usd,
		       estimated_peak_savings_usd, COALESCE(peak_probability_at_trigger, 0.0) AS peak_probability_at_trigger,
		       status
		FROM finops.prewarm_execution_ledger
		WHERE tenant_id = $1 AND job_id = $2
		LIMIT 1;
	`
	if err := c.db.GetContext(ctx, &row, query, tenantID, jobID); err != nil {
		return nil, err
	}

	return &PrewarmResult{
		TenantID:                row.TenantID,
		JobID:                   row.JobID,
		Triggered:               row.Status != "SKIPPED_BELOW_THRESHOLD" && row.Status != "SKIPPED_NO_TARGETS",
		PeakProbability:         row.PeakProbability,
		TargetsSeeded:           row.EntitiesPrewarmedCount,
		ComputeCostIncurredUSD:  row.ComputeCostIncurredUSD,
		EstimatedPeakSavingsUSD: row.EstimatedPeakSavingsUSD,
		Status:                  row.Status,
	}, nil
}

// GetLatestPrewarmExecution retrieves the most recent prewarm execution ledger entry for a tenant.
// Guarantees strict Rule 7 tenant isolation by filtering WHERE tenant_id = $1.
func (c *PrewarmCoordinator) GetLatestPrewarmExecution(ctx context.Context, tenantID uuid.UUID) (*PrewarmResult, error) {
	if c.db == nil {
		return nil, nil
	}

	var row struct {
		TenantID                uuid.UUID  `db:"tenant_id"`
		JobID                   *uuid.UUID `db:"job_id"`
		EntitiesPrewarmedCount  int        `db:"entities_prewarmed_count"`
		ComputeCostIncurredUSD  float64    `db:"compute_cost_incurred_usd"`
		EstimatedPeakSavingsUSD float64    `db:"estimated_peak_savings_usd"`
		PeakProbability         float64    `db:"peak_probability_at_trigger"`
		Status                  string     `db:"status"`
	}

	query := `
		SELECT tenant_id, job_id, entities_prewarmed_count, compute_cost_incurred_usd,
		       estimated_peak_savings_usd, COALESCE(peak_probability_at_trigger, 0.0) AS peak_probability_at_trigger,
		       status
		FROM finops.prewarm_execution_ledger
		WHERE tenant_id = $1
		ORDER BY executed_at DESC
		LIMIT 1;
	`
	if err := c.db.GetContext(ctx, &row, query, tenantID); err != nil {
		return nil, err
	}

	return &PrewarmResult{
		TenantID:                row.TenantID,
		JobID:                   row.JobID,
		Triggered:               row.Status != "SKIPPED_BELOW_THRESHOLD" && row.Status != "SKIPPED_NO_TARGETS",
		PeakProbability:         row.PeakProbability,
		TargetsSeeded:           row.EntitiesPrewarmedCount,
		ComputeCostIncurredUSD:  row.ComputeCostIncurredUSD,
		EstimatedPeakSavingsUSD: row.EstimatedPeakSavingsUSD,
		Status:                  row.Status,
	}, nil
}
