package services

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// AdverseCondition represents a type of adverse condition to simulate
type AdverseCondition int

const (
	ConditionNormal AdverseCondition = iota
	ConditionDelayedPolicyService
	ConditionCacheMisses
	ConditionInvalidationStorm
	ConditionDownstreamThrottling
	ConditionHighContention
)

// contextKey is a private type to avoid context key collisions.
type contextKey string

const (
	policyDelayKey       = contextKey("policy_delay")
	throttleRateKey      = contextKey("throttle_rate")
	forceCacheMissKey    = contextKey("force_cache_miss")
	invalidationStormKey = contextKey("invalidation_storm")
)

// AdverseConditionSimulator simulates adverse conditions for testing
type AdverseConditionSimulator struct {
	condition         AdverseCondition
	mu                sync.RWMutex
	activeConditions  map[AdverseCondition]bool
	delayDuration     time.Duration
	throttleRate      float64 // 0.0 to 1.0
	invalidationCount int64
}

// NewAdverseConditionSimulator creates a new adverse condition simulator
func NewAdverseConditionSimulator() *AdverseConditionSimulator {
	return &AdverseConditionSimulator{
		condition:        ConditionNormal,
		activeConditions: make(map[AdverseCondition]bool),
		delayDuration:    500 * time.Millisecond,
		throttleRate:     0.1, // 10% throttling
	}
}

// SetCondition sets the current adverse condition
func (acs *AdverseConditionSimulator) SetCondition(condition AdverseCondition) {
	acs.mu.Lock()
	defer acs.mu.Unlock()

	acs.condition = condition
	logging.GetLogger().Sugar().Infof("Adverse condition set to: %v", acs.getConditionName(condition))
}

// AddCondition adds an additional adverse condition
func (acs *AdverseConditionSimulator) AddCondition(condition AdverseCondition) {
	acs.mu.Lock()
	defer acs.mu.Unlock()

	acs.activeConditions[condition] = true
	logging.GetLogger().Sugar().Infof("Added adverse condition: %v", acs.getConditionName(condition))
}

// RemoveCondition removes an adverse condition
func (acs *AdverseConditionSimulator) RemoveCondition(condition AdverseCondition) {
	acs.mu.Lock()
	defer acs.mu.Unlock()

	delete(acs.activeConditions, condition)
	logging.GetLogger().Sugar().Infof("Removed adverse condition: %v", acs.getConditionName(condition))
}

// ClearAllConditions clears all adverse conditions
func (acs *AdverseConditionSimulator) ClearAllConditions() {
	acs.mu.Lock()
	defer acs.mu.Unlock()

	acs.condition = ConditionNormal
	acs.activeConditions = make(map[AdverseCondition]bool)
	logging.GetLogger().Sugar().Info("All adverse conditions cleared")
}

// IsConditionActive checks if a specific condition is active
func (acs *AdverseConditionSimulator) IsConditionActive(condition AdverseCondition) bool {
	acs.mu.RLock()
	defer acs.mu.RUnlock()

	return acs.condition == condition || acs.activeConditions[condition]
}

// ApplyCondition applies the current adverse conditions to a context
func (acs *AdverseConditionSimulator) ApplyCondition(ctx context.Context) context.Context {
	acs.mu.RLock()
	defer acs.mu.RUnlock()

	// Apply delay if delayed policy service is active
	if acs.IsConditionActive(ConditionDelayedPolicyService) {
		ctx = context.WithValue(ctx, policyDelayKey, acs.delayDuration)
	}

	// Apply throttling if downstream throttling is active
	if acs.IsConditionActive(ConditionDownstreamThrottling) {
		ctx = context.WithValue(ctx, throttleRateKey, acs.throttleRate)
	}

	// Apply cache miss simulation
	if acs.IsConditionActive(ConditionCacheMisses) {
		ctx = context.WithValue(ctx, forceCacheMissKey, true)
	}

	// Apply invalidation storm
	if acs.IsConditionActive(ConditionInvalidationStorm) {
		atomic.AddInt64(&acs.invalidationCount, 1)
		ctx = context.WithValue(ctx, invalidationStormKey, true)
	}

	return ctx
}

// SimulateDelay simulates network/service delays
func (acs *AdverseConditionSimulator) SimulateDelay(ctx context.Context) {
	if delay, ok := ctx.Value(policyDelayKey).(time.Duration); ok && delay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

// ShouldThrottle determines if a request should be throttled
func (acs *AdverseConditionSimulator) ShouldThrottle(ctx context.Context) bool {
	if rate, ok := ctx.Value(throttleRateKey).(float64); ok {
		return rand.Float64() < rate
	}
	return false
}

// ShouldForceCacheMiss determines if cache should be forced to miss
func (acs *AdverseConditionSimulator) ShouldForceCacheMiss(ctx context.Context) bool {
	_, ok := ctx.Value(forceCacheMissKey).(bool)
	return ok
}

// IsInvalidationStormActive checks if invalidation storm is active
func (acs *AdverseConditionSimulator) IsInvalidationStormActive(ctx context.Context) bool {
	_, ok := ctx.Value(invalidationStormKey).(bool)
	return ok
}

// GetStats returns statistics about adverse conditions
func (acs *AdverseConditionSimulator) GetStats() map[string]interface{} {
	acs.mu.RLock()
	defer acs.mu.RUnlock()

	activeConditions := make([]string, 0)
	for condition := range acs.activeConditions {
		activeConditions = append(activeConditions, acs.getConditionName(condition))
	}
	if acs.condition != ConditionNormal {
		activeConditions = append(activeConditions, acs.getConditionName(acs.condition))
	}

	return map[string]interface{}{
		"current_condition":  acs.getConditionName(acs.condition),
		"active_conditions":  activeConditions,
		"delay_duration":     acs.delayDuration.String(),
		"throttle_rate":      acs.throttleRate,
		"invalidation_count": atomic.LoadInt64(&acs.invalidationCount),
	}
}

// getConditionName returns the string name of a condition
func (acs *AdverseConditionSimulator) getConditionName(condition AdverseCondition) string {
	switch condition {
	case ConditionNormal:
		return "normal"
	case ConditionDelayedPolicyService:
		return "delayed_policy_service"
	case ConditionCacheMisses:
		return "cache_misses"
	case ConditionInvalidationStorm:
		return "invalidation_storm"
	case ConditionDownstreamThrottling:
		return "downstream_throttling"
	case ConditionHighContention:
		return "high_contention"
	default:
		return "unknown"
	}
}

// SimulateNetworkDelay simulates network delays for testing
func (acs *AdverseConditionSimulator) SimulateNetworkDelay(ctx context.Context, baseDelay time.Duration) {
	if acs.IsConditionActive(ConditionDelayedPolicyService) {
		totalDelay := baseDelay + acs.delayDuration
		select {
		case <-ctx.Done():
			return
		case <-time.After(totalDelay):
		}
	} else {
		select {
		case <-ctx.Done():
			return
		case <-time.After(baseDelay):
		}
	}
}

// SimulateDatabaseContention simulates database contention
func (acs *AdverseConditionSimulator) SimulateDatabaseContention(ctx context.Context) {
	if acs.IsConditionActive(ConditionHighContention) {
		// Simulate lock contention with random delay
		delay := time.Duration(rand.Intn(100)+50) * time.Millisecond
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

// SimulateInvalidationStorm simulates cache invalidation storms
func (acs *AdverseConditionSimulator) SimulateInvalidationStorm(ctx context.Context, invalidationFunc func()) {
	if acs.IsInvalidationStormActive(ctx) {
		// Trigger multiple invalidations
		for i := 0; i < 5; i++ {
			go func() {
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
				invalidationFunc()
			}()
		}
	}
}

// SetDelayDuration sets the delay duration for delayed conditions
func (acs *AdverseConditionSimulator) SetDelayDuration(duration time.Duration) {
	acs.mu.Lock()
	defer acs.mu.Unlock()
	acs.delayDuration = duration
}

// SetThrottleRate sets the throttle rate for throttling conditions
func (acs *AdverseConditionSimulator) SetThrottleRate(rate float64) {
	acs.mu.Lock()
	defer acs.mu.Unlock()
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	acs.throttleRate = rate
}

// CreateAdverseConditionTestScenario creates a test scenario with specific adverse conditions
func CreateAdverseConditionTestScenario(name string, conditions []AdverseCondition, duration time.Duration) *AdverseTestScenario {
	return &AdverseTestScenario{
		Name:       name,
		Conditions: conditions,
		Duration:   duration,
	}
}

// AdverseTestScenario represents a test scenario with adverse conditions
type AdverseTestScenario struct {
	Name       string
	Conditions []AdverseCondition
	Duration   time.Duration
}

// RunAdverseTestScenario runs a test scenario with adverse conditions
func (acs *AdverseConditionSimulator) RunAdverseTestScenario(scenario *AdverseTestScenario, testFunc func(context.Context) error) error {
	logging.GetLogger().Sugar().Infof("Starting adverse test scenario: %s", scenario.Name)

	// Set up conditions
	for _, condition := range scenario.Conditions {
		acs.AddCondition(condition)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), scenario.Duration)
	defer cancel()

	// Apply conditions to context
	ctx = acs.ApplyCondition(ctx)

	// Run the test function
	err := testFunc(ctx)

	// Clean up conditions
	acs.ClearAllConditions()

	logging.GetLogger().Sugar().Infof("Completed adverse test scenario: %s", scenario.Name)
	return err
}

// GetRecommendedAdverseScenarios returns recommended adverse test scenarios
func GetRecommendedAdverseScenarios() []*AdverseTestScenario {
	return []*AdverseTestScenario{
		CreateAdverseConditionTestScenario(
			"delayed_policy_service",
			[]AdverseCondition{ConditionDelayedPolicyService},
			5*time.Minute,
		),
		CreateAdverseConditionTestScenario(
			"cache_miss_storm",
			[]AdverseCondition{ConditionCacheMisses, ConditionInvalidationStorm},
			10*time.Minute,
		),
		CreateAdverseConditionTestScenario(
			"downstream_throttling",
			[]AdverseCondition{ConditionDownstreamThrottling},
			5*time.Minute,
		),
		CreateAdverseConditionTestScenario(
			"high_contention",
			[]AdverseCondition{ConditionHighContention},
			3*time.Minute,
		),
		CreateAdverseConditionTestScenario(
			"combined_stress",
			[]AdverseCondition{
				ConditionDelayedPolicyService,
				ConditionCacheMisses,
				ConditionDownstreamThrottling,
				ConditionHighContention,
			},
			15*time.Minute,
		),
	}
}
