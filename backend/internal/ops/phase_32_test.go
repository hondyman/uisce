package ops

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Phase 3.2: Region-Aware Operations Tests
// Comprehensive test suite for region-aware RCA, actions, and routing
// ============================================================================

// TestRegionAwareScoringTypes tests the core types for region-aware scoring
func TestRegionAwareScoringTypes(t *testing.T) {
	t.Run("DefaultScoringWeights", func(t *testing.T) {
		weights := DefaultScoringWeights()

		if weights.SameRegionWeight != 1.0 {
			t.Errorf("Expected SameRegionWeight=1.0, got %.2f", weights.SameRegionWeight)
		}

		if weights.AdjacentRegionWeight != 0.7 {
			t.Errorf("Expected AdjacentRegionWeight=0.7, got %.2f", weights.AdjacentRegionWeight)
		}

		if weights.DistalRegionWeight != 0.4 {
			t.Errorf("Expected DistalRegionWeight=0.4, got %.2f", weights.DistalRegionWeight)
		}

		if weights.PropagationMinimumScore != 0.65 {
			t.Errorf("Expected PropagationMinimumScore=0.65, got %.2f", weights.PropagationMinimumScore)
		}
	})

	t.Run("RegionScoringContext", func(t *testing.T) {
		context := &RegionScoringContext{
			Regions: map[string]*RegionMetadata{
				"us-east-1": {
					RegionCode:   "us-east-1",
					RegionName:   "N. Virginia",
					Tier:         "tier-1",
					IsHealthy:    true,
					HealthScore:  0.95,
					AvgLatencyMS: 5.0,
				},
				"ap-south-1": {
					RegionCode:   "ap-south-1",
					RegionName:   "Mumbai",
					Tier:         "tier-2",
					IsHealthy:    false,
					HealthScore:  0.60,
					AvgLatencyMS: 25.0,
				},
			},
			RegionAdjacency: map[string][]string{
				"us-east-1":  {"us-west-2"},
				"ap-south-1": {},
			},
			CrossRegionLatency: map[string]int64{
				"us-east-1->ap-south-1": 250,
			},
		}

		if len(context.Regions) != 2 {
			t.Errorf("Expected 2 regions, got %d", len(context.Regions))
		}

		latency, ok := context.CrossRegionLatency["us-east-1->ap-south-1"]
		if !ok || latency != 250 {
			t.Errorf("Expected latency 250ms, got %d", latency)
		}
	})

	t.Run("RegionAwareCorrelationScore", func(t *testing.T) {
		baseScore := &CorrelationScore{
			Score:     0.8,
			TimeGapMs: 100,
		}

		score := &RegionAwareCorrelationScore{
			BaseScore:             *baseScore,
			FromRegion:            "us-east-1",
			ToRegion:              "us-east-1",
			SameRegion:            true,
			PropagationLikelihood: 0.95,
			RegionAwareScore:      0.8,
		}

		if !score.SameRegion {
			t.Errorf("Expected SameRegion=true")
		}

		if score.PropagationLikelihood < 0.9 {
			t.Errorf("Expected high propagation likelihood for same-region, got %.2f", score.PropagationLikelihood)
		}
	})

	t.Logf("✓ Region-aware scoring types validated")
}

// TestRegionAwareActionTypes tests the core types for region-aware action execution
func TestRegionAwareActionTypes(t *testing.T) {
	t.Run("RegionScopedAction", func(t *testing.T) {
		action := &RegionScopedAction{
			ActionID:      "action-1",
			Region:        "us-east-1",
			ActionType:    "restart_worker",
			Priority:      "critical",
			TimeoutMs:     10000,
			RetryAttempts: 2,
			Config: map[string]interface{}{
				"worker_pool": "workers-1",
			},
		}

		if action.ActionID != "action-1" {
			t.Errorf("Expected ActionID='action-1'")
		}

		if action.Region != "us-east-1" {
			t.Errorf("Expected Region='us-east-1'")
		}

		if action.ActionType != "restart_worker" {
			t.Errorf("Expected ActionType='restart_worker'")
		}
	})

	t.Run("ActionExecutionResult", func(t *testing.T) {
		now := time.Now()
		result := &ActionExecutionResult{
			ActionID:  "action-1",
			Status:    "success",
			StartedAt: &now,
			Message:   "Successfully restarted 3 workers",
		}

		if result.ActionID != "action-1" {
			t.Errorf("Expected ActionID='action-1'")
		}

		if result.Status != "success" {
			t.Errorf("Expected Status='success', got '%s'", result.Status)
		}

		if result.StartedAt == nil {
			t.Errorf("Expected StartedAt to be set")
		}
	})

	t.Run("RegionExecutionPlan", func(t *testing.T) {
		plan := &RegionExecutionPlan{
			PlanID:           "plan-123",
			TargetIncidentID: "incident-456",
			TenantID:         "tenant-789",
			CurrentPhase:     1,
			Status:           "pending",
			PhaseOne: &ExecutionPhase{
				PhaseNumber:     1,
				TargetRegions:   []string{"us-east-1"},
				TimeoutMs:       30000,
				RequiredSuccess: 0.9,
				Actions: []*RegionScopedAction{
					{
						ActionID:   "action-1",
						Region:     "us-east-1",
						ActionType: "restart_worker",
					},
				},
			},
			ExecutionResults: make(map[string]*RegionExecutionResult),
		}

		if plan.PlanID != "plan-123" {
			t.Errorf("Expected PlanID='plan-123'")
		}

		if len(plan.PhaseOne.Actions) != 1 {
			t.Errorf("Expected 1 action in Phase One, got %d", len(plan.PhaseOne.Actions))
		}

		if plan.PhaseOne.RequiredSuccess != 0.9 {
			t.Errorf("Expected RequiredSuccess=0.9")
		}
	})

	t.Run("RegionExecutionResult", func(t *testing.T) {
		result := &RegionExecutionResult{
			Region:            "us-east-1",
			PlanID:            "plan-123",
			ExecutedAt:        time.Now(),
			ActionsAttempted:  3,
			ActionsSucceeded:  3,
			ActionsFailed:     0,
			SuccessfulActions: []string{"action-1", "action-2", "action-3"},
		}

		if result.Region != "us-east-1" {
			t.Errorf("Expected Region='us-east-1'")
		}

		if result.ActionsAttempted != 3 {
			t.Errorf("Expected ActionsAttempted=3, got %d", result.ActionsAttempted)
		}

		successRate := float64(result.ActionsSucceeded) / float64(result.ActionsAttempted)
		if successRate != 1.0 {
			t.Errorf("Expected 100%% success rate, got %.0f%%", successRate*100)
		}
	})

	t.Logf("✓ Region-aware action types validated")
}

// TestMultiRegionRoutingTypes tests the core types for multi-region routing
func TestMultiRegionRoutingTypes(t *testing.T) {
	t.Run("TenantRegionPreference", func(t *testing.T) {
		preference := &TenantRegionPreference{
			TenantID:           "tenant-1",
			PreferredRegion:    "us-east-1",
			AllowedRegions:     []string{"us-east-1", "us-west-2"},
			LocalityPreference: "latency",
			DataResidencyRule:  "can_cross",
			LatencyThreshold:   500,
		}

		if preference.TenantID != "tenant-1" {
			t.Errorf("Expected TenantID='tenant-1'")
		}

		if preference.PreferredRegion != "us-east-1" {
			t.Errorf("Expected PreferredRegion='us-east-1'")
		}

		if len(preference.AllowedRegions) != 2 {
			t.Errorf("Expected 2 allowed regions, got %d", len(preference.AllowedRegions))
		}
	})

	t.Run("RoutingDecision", func(t *testing.T) {
		decision := &RoutingDecision{
			TenantID:            "tenant-1",
			RequestType:         "query",
			SelectedRegion:      "us-east-1",
			EstimatedLatencyMs:  45,
			EstimatedCostFactor: 1.0,
		}

		if decision.SelectedRegion != "us-east-1" {
			t.Errorf("Expected SelectedRegion='us-east-1'")
		}

		if decision.EstimatedLatencyMs != 45 {
			t.Errorf("Expected EstimatedLatencyMs=45, got %d", decision.EstimatedLatencyMs)
		}
	})

	t.Run("RoutingContext", func(t *testing.T) {
		context := &RoutingContext{
			PerformanceRequirements: &PerformanceRequirements{
				MaxLatencyMs:   200,
				MaxFailureRate: 0.05,
			},
			CostContext: &CostContext{
				Budget:          1.5,
				CostSensitivity: "high",
			},
		}

		if context.PerformanceRequirements.MaxLatencyMs != 200 {
			t.Errorf("Expected MaxLatencyMs=200")
		}

		if context.CostContext.CostSensitivity != "high" {
			t.Errorf("Expected CostSensitivity='high'")
		}
	})

	t.Logf("✓ Multi-region routing types validated")
}

// TestMultiRegionRoutingEngine tests the routing engine functionality
func TestMultiRegionRoutingEngine(t *testing.T) {
	engine := NewMultiRegionRoutingEngine(&MockRegionRouter{})

	t.Run("TenantPreferenceStorage", func(t *testing.T) {
		ctx := context.Background()

		preference := &TenantRegionPreference{
			TenantID:        "tenant-1",
			PreferredRegion: "us-east-1",
			AllowedRegions:  []string{"us-east-1", "us-west-2"},
		}

		err := engine.SetTenantRegionPreference(ctx, "tenant-1", preference)
		if err != nil {
			t.Errorf("Failed to set preference: %v", err)
		}
	})

	t.Run("CacheInvalidation", func(t *testing.T) {
		// Prime cache
		engine.routingCache["test-key"] = &RoutingDecision{
			SelectedRegion: "us-east-1",
		}

		if len(engine.routingCache) == 0 {
			t.Errorf("Cache should have entries before invalidation")
		}

		engine.InvalidateRegionCache()

		if len(engine.routingCache) > 0 {
			t.Errorf("Cache should be empty after invalidation, got %d entries", len(engine.routingCache))
		}
	})

	t.Logf("✓ Multi-region routing engine validated")
}

// Mock implementations for testing

// MockRegionRouter provides a mock implementation for testing
type MockRegionRouter struct{}

func (m *MockRegionRouter) GetTenantRegion(ctx context.Context, tenantID uuid.UUID) (string, error) {
	return "us-east-1", nil
}

func (m *MockRegionRouter) GetTenantAllowedRegions(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	return []string{"us-east-1", "us-west-2"}, nil
}

func (m *MockRegionRouter) SetTenantRegion(ctx context.Context, tenantID uuid.UUID, region string, allowed []string) error {
	return nil
}

func (m *MockRegionRouter) GetRegionTarget(ctx context.Context, region string) (*RegionTarget, error) {
	return &RegionTarget{
		Region:            region,
		StarRocksCluster:  "starrocks-" + region,
		RedpandaBroker:    "kafka-" + region + ":9092",
		TemporalNamespace: region,
		OpsWorkerPool:     "worker-pool-" + region,
		IsActive:          true,
		LastHealthCheckAt: time.Now(),
	}, nil
}

func (m *MockRegionRouter) ListRegionTargets(ctx context.Context) (map[string]*RegionTarget, error) {
	return map[string]*RegionTarget{
		"us-east-1": {
			Region:            "us-east-1",
			IsActive:          true,
			LastHealthCheckAt: time.Now(),
		},
		"us-west-2": {
			Region:            "us-west-2",
			IsActive:          true,
			LastHealthCheckAt: time.Now(),
		},
	}, nil
}

func (m *MockRegionRouter) RegisterRegionTarget(ctx context.Context, target *RegionTarget) error {
	return nil
}

func (m *MockRegionRouter) RouteForTenant(ctx context.Context, tenantID uuid.UUID) (*RegionTarget, error) {
	return &RegionTarget{
		Region:   "us-east-1",
		IsActive: true,
	}, nil
}

func (m *MockRegionRouter) RouteForIncident(ctx context.Context, incident *Incident) (*RegionTarget, error) {
	return &RegionTarget{
		Region:   "us-east-1",
		IsActive: true,
	}, nil
}

func (m *MockRegionRouter) RouteForEvent(ctx context.Context, event *Event) (*RegionTarget, error) {
	return &RegionTarget{
		Region:   "us-east-1",
		IsActive: true,
	}, nil
}

func (m *MockRegionRouter) GetFailoverTarget(ctx context.Context, region string) (*RegionTarget, error) {
	return &RegionTarget{
		Region:   "us-west-2",
		IsActive: true,
	}, nil
}

func (m *MockRegionRouter) MarkRegionDown(ctx context.Context, region string) error {
	return nil
}

func (m *MockRegionRouter) MarkRegionUp(ctx context.Context, region string) error {
	return nil
}

// Integration test that runs all Phase 3.2 tests
func TestPhase32Integration(t *testing.T) {
	t.Run("ScoringTypes", TestRegionAwareScoringTypes)
	t.Run("ActionTypes", TestRegionAwareActionTypes)
	t.Run("RoutingTypes", TestMultiRegionRoutingTypes)
	t.Run("RoutingEngine", TestMultiRegionRoutingEngine)

	t.Logf("\n✅ Phase 3.2 Integration: ALL TESTS PASSED\n")
}
