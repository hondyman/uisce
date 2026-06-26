package load

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	ops "github.com/hondyman/semlayer/backend/internal/ops"
)

// ============================================================================
// Phase 3.3: Benchmark Tests for Performance Profiling
// Measure operation latencies and throughput
// ============================================================================

// BenchmarkRoutingDecision measures single routing decision latency
func BenchmarkRoutingDecision(b *testing.B) {
	ctx := context.Background()

	routingEngine := ops.NewMultiRegionRoutingEngine(&mockLoadRouter{})
	tenantID := "bench-tenant-1"

	pref := &ops.TenantRegionPreference{
		TenantID:        tenantID,
		PreferredRegion: "us-east-1",
		AllowedRegions:  []string{"us-east-1", "us-west-2", "eu-west-1"},
	}

	require.NoError(b, routingEngine.SetTenantRegionPreference(ctx, tenantID, pref))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		routingCtx := &ops.RoutingContext{
			PerformanceRequirements: &ops.PerformanceRequirements{
				MaxLatencyMs: 200,
			},
		}
		_ = simulateRoutingDecision(ctx, routingEngine, tenantID, routingCtx)
	}
}

// BenchmarkRCAScoring measures RCA scoring latency
func BenchmarkRCAScoring(b *testing.B) {
	scorer := &ops.RegionAwareRCAScorer{}
	regions := setupTestRegions(3)
	regionContext := buildRegionContext(regions)

	baseRCA := &ops.RCAResult{
		ConfidenceScore: 0.85,
		CausalityChain:  make([]ops.ScoredEvent, 0),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = scorer.ScoreRCAWithRegionContext(baseRCA, regionContext, ops.DefaultScoringWeights())
	}
}

// BenchmarkActionExecution measures action execution latency
func BenchmarkActionExecution(b *testing.B) {
	ctx := context.Background()

	executor := ops.NewRegionAwareActionExecutor(
		&mockLoadRouter{},
		nil,
		nil,
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		plan := &ops.RegionExecutionPlan{
			PlanID:           fmt.Sprintf("plan-bench-%d", i),
			TargetIncidentID: fmt.Sprintf("incident-%d", i),
			TenantID:         fmt.Sprintf("tenant-%d", i),
			CurrentPhase:     1,
			Status:           "pending",
			PhaseOne: &ops.ExecutionPhase{
				PhaseNumber:     1,
				TargetRegions:   []string{"us-east-1"},
				TimeoutMs:       30000,
				RequiredSuccess: 0.9,
				Actions:         generateMockActions(5, "us-east-1"),
			},
			ExecutionResults: make(map[string]*ops.RegionExecutionResult),
		}

		_ = executor.ExecuteWithRegionAwareness(ctx, plan)
	}
}

// BenchmarkFailover measures failover latency
func BenchmarkFailover(b *testing.B) {
	ctx := context.Background()

	routingEngine := ops.NewMultiRegionRoutingEngine(&mockLoadRouter{})

	pref := &ops.TenantRegionPreference{
		TenantID:        "bench-failover",
		PreferredRegion: "us-east-1",
		AllowedRegions:  []string{"us-east-1", "us-west-2", "eu-west-1"},
	}
	require.NoError(b, routingEngine.SetTenantRegionPreference(ctx, "bench-failover", pref))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = routingEngine.ForceRegionFailover(ctx, "bench-failover", "us-east-1", "us-west-2")
	}
}

// Note: Stress tests  (TestStressLongRunningMultiRegionOps, TestStressHighConcurrencyFailover, etc.)
// are defined in phase_33_load_test.go for better code organization.
// The benchmark tests above (BenchmarkRoutingDecision, BenchmarkRCAScoring, etc.)
// measure operation latencies using Go's testing/b utilities.
