package aso

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ============================================================================
// Simulation Engine Types
// ============================================================================

// SimulationResult captures the detailed outcome of a what-if simulation
type SimulationResult struct {
	Window                     string  `json:"window"`
	WorkloadSampleSize         int     `json:"workload_sample_size"`
	ExpectedSpeedup            float64 `json:"expected_speedup"`
	ExpectedCostSavings        float64 `json:"expected_cost_savings"`
	ExpectedStorageDeltaBytes  int64   `json:"expected_storage_delta_bytes"`
	ExpectedRefreshCostDeltaMs float64 `json:"expected_refresh_cost_delta_ms"`
	QueriesImproved            int     `json:"queries_improved"`
	QueriesRegressed           int     `json:"queries_regressed"`
	CorrectnessMismatches      int     `json:"correctness_mismatches"`
	PreAggHitRateBefore        float64 `json:"preagg_hit_rate_before"`
	PreAggHitRateAfter         float64 `json:"preagg_hit_rate_after"`
	EntitlementCostBeforeMs    float64 `json:"entitlement_cost_before_ms"`
	EntitlementCostAfterMs     float64 `json:"entitlement_cost_after_ms"`
	Confidence                 float64 `json:"confidence"`
	Explanation                string  `json:"explanation,omitempty"`
}

// SimulationEngine provides what-if analysis by replaying workloads
type SimulationEngine interface {
	Simulate(ctx context.Context, opt ASOOptimization) (*SimulationResult, error)
}

// ============================================================================
// Internal Interfaces (Dependencies)
// ============================================================================

// WorkloadRepository provides access to historical queries
type WorkloadRepository interface {
	SampleQueries(ctx context.Context, env string, tenantID *uuid.UUID, targetID uuid.UUID, window string, limit int) ([]QueryEvent, error)
}

// BOSQLResolver resolves abstract BO queries to concrete SQL
type BOSQLResolver interface {
	ResolveWithModel(ctx context.Context, req BOSQLRequest, model ModelVersion) (string, map[string]interface{})
}

// SimulatedExecutor executes queries in a simulated environment (or cost model)
type SimulatedExecutor interface {
	Execute(ctx context.Context, sql string, meta map[string]interface{}) SimulatedExecutionResult
}

// QueryEvent represents a historical query
type QueryEvent struct {
	Env      string
	TenantID uuid.UUID
	UserID   string
	BOName   string
	SQL      string
	Region   string
}

// BOSQLRequest for resolution
type BOSQLRequest struct {
	Env           string
	TenantID      *uuid.UUID
	BOName        string
	CurrentUserID string
	Region        string
}

// ModelVersion represents the semantic model state
type ModelVersion struct {
	ChangeSetID  uuid.UUID
	Optimization *ASOOptimization
}

// CurrentModelVersion is a marker for the current state
var CurrentModelVersion = ModelVersion{}

// SimulatedExecutionResult captures simulated execution metrics
type SimulatedExecutionResult struct {
	LatencyMs         float64
	Cost              float64
	PreAggHit         bool
	EntitlementCostMs float64
}

// ============================================================================
// Implementation
// ============================================================================

type simulationEngine struct {
	workloadRepo      WorkloadRepository
	resolver          BOSQLResolver
	simulatedExecutor SimulatedExecutor
}

// NewSimulationEngine creates a new simulation engine
func NewSimulationEngine(w WorkloadRepository, r BOSQLResolver, e SimulatedExecutor) SimulationEngine {
	return &simulationEngine{workloadRepo: w, resolver: r, simulatedExecutor: e}
}

// Simulate runs a what-if simulation for an optimization
func (s *simulationEngine) Simulate(ctx context.Context, opt ASOOptimization) (*SimulationResult, error) {
	window := "7d"
	limit := 1000

	sample, err := s.workloadRepo.SampleQueries(ctx, opt.Env, opt.TenantID, opt.TargetID, window, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to sample workload: %w", err)
	}

	if len(sample) == 0 {
		return &SimulationResult{
			Window:             window,
			WorkloadSampleSize: 0,
			Confidence:         0.1,
			Explanation:        "Not enough workload data to simulate",
		}, nil
	}

	before := s.replay(ctx, sample, CurrentModelVersion)
	after := s.replay(ctx, sample, ModelVersion{Optimization: &opt})

	res := computeSimulationDeltas(before, after)
	res.Window = window
	res.WorkloadSampleSize = len(sample)
	res.Confidence = estimateConfidence(before, after)

	// Enrich with optimization details if available (e.g. storage/refresh estimates)
	// In a real implementation, this would come from the optimization proposal logic
	if opt.TargetType == TargetTypePreAgg {
		res.ExpectedStorageDeltaBytes = 1024 * 1024 * 50 // Mock: 50MB
		res.ExpectedRefreshCostDeltaMs = 5000            // Mock: 5s
	}

	return &res, nil
}

// ReplayStats aggregates replay results
type ReplayStats struct {
	TotalQueries          int
	TotalLatencyBeforeMs  float64 // Not used in 'after' struct but kept for symmetry/potential use
	TotalLatencyAfterMs   float64 // Used to accumulate latency for the run
	TotalCostBefore       float64
	TotalCostAfter        float64
	HitRateBefore         float64
	HitRateAfter          float64
	EntitlementCostBefore float64
	EntitlementCostAfter  float64
	QueriesImproved       int
	QueriesRegressed      int
	CorrectnessMismatches int
}

func (s *simulationEngine) replay(ctx context.Context, sample []QueryEvent, model ModelVersion) ReplayStats {
	stats := ReplayStats{}
	for _, q := range sample {
		// Mock resolution if resolver not fully implemented
		var sql string
		var meta map[string]interface{}

		if s.resolver != nil {
			sql, meta = s.resolver.ResolveWithModel(ctx, BOSQLRequest{
				Env:           q.Env,
				TenantID:      &q.TenantID,
				BOName:        q.BOName,
				CurrentUserID: q.UserID,
				Region:        q.Region,
			}, model)
		} else {
			sql = q.SQL // Fallback
		}

		exec := s.simulatedExecutor.Execute(ctx, sql, meta)

		stats.TotalQueries++
		stats.TotalLatencyAfterMs += exec.LatencyMs
		stats.TotalCostAfter += exec.Cost
		if exec.PreAggHit {
			stats.HitRateAfter += 1.0
		}
		stats.EntitlementCostAfter += exec.EntitlementCostMs
		// correctness comparison would go here
	}

	if stats.TotalQueries > 0 {
		stats.HitRateAfter /= float64(stats.TotalQueries)
	}
	return stats
}

func computeSimulationDeltas(before, after ReplayStats) SimulationResult {
	avgBefore := before.TotalLatencyAfterMs / float64(max(1, before.TotalQueries))
	avgAfter := after.TotalLatencyAfterMs / float64(max(1, after.TotalQueries))

	var speedup float64
	if avgAfter > 0 {
		speedup = avgBefore / avgAfter
	} else if avgBefore > 0 {
		speedup = 100.0 // Infinite speedup (0 latency)
	} else {
		speedup = 1.0 // No change
	}

	costSavings := 0.0
	if before.TotalCostAfter > 0 {
		costSavings = (before.TotalCostAfter - after.TotalCostAfter) / before.TotalCostAfter
	}

	return SimulationResult{
		ExpectedSpeedup:            speedup,
		ExpectedCostSavings:        costSavings,
		ExpectedStorageDeltaBytes:  0,                     // fill from optimization details
		ExpectedRefreshCostDeltaMs: 0,                     // fill from pre-agg metadata
		QueriesImproved:            after.QueriesImproved, // Note: This requires per-query comparison logic in replay which is mocked here
		QueriesRegressed:           after.QueriesRegressed,
		CorrectnessMismatches:      after.CorrectnessMismatches,
		PreAggHitRateBefore:        before.HitRateAfter,
		PreAggHitRateAfter:         after.HitRateAfter,
		EntitlementCostBeforeMs:    before.EntitlementCostAfter,
		EntitlementCostAfterMs:     after.EntitlementCostAfter,
	}
}

func estimateConfidence(before, after ReplayStats) float64 {
	// Simple confidence heuristic based on sample size and variance (mocked)
	base := 0.5
	if before.TotalQueries > 100 {
		base += 0.2
	}
	if before.TotalQueries > 1000 {
		base += 0.2
	}
	// Penalize if too many mismatches
	if after.CorrectnessMismatches > 0 {
		base -= 0.3
	}
	if base > 1.0 {
		base = 1.0
	}
	if base < 0.0 {
		base = 0.0
	}
	return base
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
