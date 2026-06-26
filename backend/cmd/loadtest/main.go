package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/ops"
)

// LoadTestResult holds test aggregate statistics
type LoadTestResult struct {
	TotalIncidents       int64
	TotalEvents          int64
	TotalRCAComputations int64
	TotalPatternMatches  int64
	DurationMs           int64
	AvgRCALatencyMs      float64
	AvgPatternLatencyMs  float64
	MaxRCALatencyMs      int64
	MaxPatternLatencyMs  int64
	ErrorCount           int64
}

// Phase 3.4: Regional load test result
type RegionalLoadTestResult struct {
	SameRegionResult     LoadTestResult
	AdjacentRegionResult LoadTestResult
	CrossRegionResult    LoadTestResult
	RegionDiscoveryMs    int64 // Benchmark region lookup overhead
}

// MockStore implements Store interface for load testing
type MockStore struct{}

func (m *MockStore) ListAlerts(ctx context.Context, enabled *bool) ([]ops.Alert, error) {
	return []ops.Alert{}, nil
}
func (m *MockStore) GetAlert(ctx context.Context, id uuid.UUID) (*ops.Alert, error) {
	return nil, nil
}
func (m *MockStore) CreateAlert(ctx context.Context, alert ops.Alert) (*ops.Alert, error) {
	return &alert, nil
}
func (m *MockStore) UpdateAlert(ctx context.Context, id uuid.UUID, alert ops.Alert) error {
	return nil
}
func (m *MockStore) DeleteAlert(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *MockStore) InsertAlertEvent(ctx context.Context, event ops.AlertEvent) error {
	return nil
}
func (m *MockStore) GetAlertEvents(ctx context.Context, alertID uuid.UUID, limit int) ([]ops.AlertEvent, error) {
	return []ops.AlertEvent{}, nil
}
func (m *MockStore) GetOrCreateFingerprint(ctx context.Context, fingerprint, path string, statusCode int, sample string) (*ops.ErrorFingerprint, error) {
	return nil, nil
}
func (m *MockStore) UpdateFingerprintCount(ctx context.Context, fingerprintID uuid.UUID, newCount int64) error {
	return nil
}
func (m *MockStore) InsertErrorEvent(ctx context.Context, event ops.ErrorEvent) error {
	return nil
}
func (m *MockStore) ListFingerprints(ctx context.Context, limit int) ([]ops.ErrorFingerprint, error) {
	return []ops.ErrorFingerprint{}, nil
}
func (m *MockStore) GetFingerprintEvents(ctx context.Context, fingerprintID uuid.UUID, limit int) ([]ops.ErrorEvent, error) {
	return []ops.ErrorEvent{}, nil
}
func (m *MockStore) UpsertTenantHealth(ctx context.Context, health ops.TenantHealth) error {
	return nil
}
func (m *MockStore) GetTenantHealth(ctx context.Context, tenantID uuid.UUID) (*ops.TenantHealth, error) {
	return nil, nil
}
func (m *MockStore) GetTenantHealths(ctx context.Context, limit int) ([]ops.TenantHealth, error) {
	return []ops.TenantHealth{}, nil
}
func (m *MockStore) UpsertEndpointHealth(ctx context.Context, health ops.EndpointHealth) error {
	return nil
}
func (m *MockStore) GetEndpointHealth(ctx context.Context, endpoint string) (*ops.EndpointHealth, error) {
	return nil, nil
}
func (m *MockStore) GetEndpointHealths(ctx context.Context, limit int) ([]ops.EndpointHealth, error) {
	return []ops.EndpointHealth{}, nil
}
func (m *MockStore) InsertHeatmapBucket(ctx context.Context, bucketTime time.Time, dimensionType, dimensionValue string, p50, p95, p99 int, requestCount int) error {
	return nil
}
func (m *MockStore) GetHeatmapData(ctx context.Context, dimensionType, dimensionValue string, bucketSize time.Duration, window time.Duration) ([]ops.HeatmapSeriesPoint, error) {
	return []ops.HeatmapSeriesPoint{}, nil
}
func (m *MockStore) GetHeatmapSeries(ctx context.Context, dimensionType string, limit int, bucketSize time.Duration, window time.Duration) ([]ops.HeatmapSeries, error) {
	return []ops.HeatmapSeries{}, nil
}
func (m *MockStore) GetMetricValue(ctx context.Context, metric, scope string, since time.Time) (float64, error) {
	return 0.0, nil
}
func (m *MockStore) GetTenantMetrics(ctx context.Context, tenantID uuid.UUID, since time.Time) (*ops.TenantMetrics, error) {
	return nil, nil
}
func (m *MockStore) GetEndpointMetrics(ctx context.Context, endpoint string, since time.Time) (*ops.EndpointMetrics, error) {
	return nil, nil
}
func (m *MockStore) GetGlobalMetrics(ctx context.Context, since time.Time) (*ops.TenantMetrics, error) {
	return nil, nil
}
func (m *MockStore) InsertEvent(ctx context.Context, e ops.Event) error {
	return nil
}
func (m *MockStore) ListEvents(ctx context.Context, since time.Time, limit int) ([]ops.Event, error) {
	return []ops.Event{}, nil
}
func (m *MockStore) GetIncident(ctx context.Context, id uuid.UUID) (*ops.Incident, []ops.Event, error) {
	return &ops.Incident{ID: id}, []ops.Event{}, nil
}
func (m *MockStore) UpsertIncidentForEvent(ctx context.Context, e ops.Event) (*ops.Incident, error) {
	return nil, nil
}
func (m *MockStore) CloseIncident(ctx context.Context, id uuid.UUID, summary *string, rootCause *string) error {
	return nil
}
func (m *MockStore) InsertActionHistory(ctx context.Context, history ops.ActionHistory) error {
	return nil
}
func (m *MockStore) UpdateActionHistory(ctx context.Context, id uuid.UUID, status string, result []byte, errorMsg *string) error {
	return nil
}
func (m *MockStore) GetActionHistory(ctx context.Context, id uuid.UUID) (*ops.ActionHistory, error) {
	return nil, nil
}
func (m *MockStore) ListIncidentActions(ctx context.Context, incidentID uuid.UUID, limit int) ([]ops.ActionHistory, error) {
	return []ops.ActionHistory{}, nil
}
func (m *MockStore) InsertAuditLog(ctx context.Context, auditLog *ops.AuditLog) error {
	return nil
}
func (m *MockStore) GetAuditLog(ctx context.Context, id uuid.UUID) (*ops.AuditLog, error) {
	return nil, nil
}
func (m *MockStore) ListAuditLogs(ctx context.Context, filters ops.AuditLogFilters, limit int, offset int) ([]ops.AuditLog, error) {
	return []ops.AuditLog{}, nil
}
func (m *MockStore) ListIncidentAuditLogs(ctx context.Context, incidentID uuid.UUID, limit int) ([]ops.AuditLog, error) {
	return []ops.AuditLog{}, nil
}
func (m *MockStore) GetRegionConfig(ctx context.Context, regionCode string) (*ops.RegionConfig, error) {
	return nil, nil
}
func (m *MockStore) ListRegionConfigs(ctx context.Context, activeOnly bool) ([]ops.RegionConfig, error) {
	return []ops.RegionConfig{}, nil
}
func (m *MockStore) InsertRegionRouting(ctx context.Context, routing *ops.RegionRouting) error {
	return nil
}
func (m *MockStore) GetRegionRouting(ctx context.Context, tenantID uuid.UUID, region string) (*ops.RegionRouting, error) {
	return nil, nil
}
func (m *MockStore) ListRegionRoutings(ctx context.Context, tenantID uuid.UUID) ([]ops.RegionRouting, error) {
	return []ops.RegionRouting{}, nil
}

// Phase 3.5: Regional Metrics & SLA Tracking
func (m *MockStore) UpsertRegionalMetrics(ctx context.Context, metrics *ops.RegionalMetrics) error {
	return nil
}
func (m *MockStore) GetRegionalMetrics(ctx context.Context, region string) (*ops.RegionalMetrics, error) {
	return nil, nil
}
func (m *MockStore) ListRegionalMetrics(ctx context.Context, limit int) ([]ops.RegionalMetrics, error) {
	return []ops.RegionalMetrics{}, nil
}
func (m *MockStore) UpsertRegionalHealth(ctx context.Context, health *ops.RegionalHealth) error {
	return nil
}
func (m *MockStore) GetRegionalHealth(ctx context.Context, region string) (*ops.RegionalHealth, error) {
	return nil, nil
}
func (m *MockStore) ListRegionalHealth(ctx context.Context, limit int) ([]ops.RegionalHealth, error) {
	return []ops.RegionalHealth{}, nil
}
func (m *MockStore) UpsertRegionalSLA(ctx context.Context, sla *ops.RegionalSLA) error {
	return nil
}
func (m *MockStore) GetRegionalSLA(ctx context.Context, region string) (*ops.RegionalSLA, error) {
	return nil, nil
}
func (m *MockStore) ListRegionalSLAs(ctx context.Context, limit int) ([]ops.RegionalSLA, error) {
	return []ops.RegionalSLA{}, nil
}
func (m *MockStore) InsertRegionalSLAStatus(ctx context.Context, status *ops.RegionalSLAStatus) error {
	return nil
}
func (m *MockStore) GetRegionalSLAStatus(ctx context.Context, region string) (*ops.RegionalSLAStatus, error) {
	return nil, nil
}
func (m *MockStore) ListRegionalSLAStatuses(ctx context.Context, region string, limit int) ([]ops.RegionalSLAStatus, error) {
	return []ops.RegionalSLAStatus{}, nil
}

// Phase 3.9: Region-Aware API Layer - MockStore implementations
func (m *MockStore) ListIncidents(ctx context.Context, limit int) ([]ops.Incident, error) {
	return []ops.Incident{}, nil
}

func (m *MockStore) ListIncidentsByRegion(ctx context.Context, region string, limit int) ([]ops.Incident, error) {
	return []ops.Incident{}, nil
}

func (m *MockStore) ListLatestRegionalSLAStatuses(ctx context.Context) ([]ops.RegionalSLAStatus, error) {
	return []ops.RegionalSLAStatus{}, nil
}

func (m *MockStore) ListRegionalIncidentCounts(ctx context.Context, since, until time.Time) ([]ops.RegionalIncidentCount, error) {
	return []ops.RegionalIncidentCount{}, nil
}

func (m *MockStore) ListOpsEventsByRegion(ctx context.Context, region string, limit int) ([]ops.Event, error) {
	return []ops.Event{}, nil
}

func (m *MockStore) ListAuditLogsByRegion(ctx context.Context, region string, limit int) ([]ops.AuditLog, error) {
	return []ops.AuditLog{}, nil
}

// Phase 3.10: Failover Policies & Automated Regional Failover - MockStore implementations
func (m *MockStore) InsertFailoverPolicy(ctx context.Context, policy *ops.FailoverPolicy) error {
	return nil
}

func (m *MockStore) GetFailoverPolicy(ctx context.Context, id uuid.UUID) (*ops.FailoverPolicy, error) {
	return nil, nil
}

func (m *MockStore) ListFailoverPolicies(ctx context.Context, tenantID uuid.UUID) ([]ops.FailoverPolicy, error) {
	return []ops.FailoverPolicy{}, nil
}

func (m *MockStore) UpdateFailoverPolicy(ctx context.Context, id uuid.UUID, policy *ops.FailoverPolicy) error {
	return nil
}

func (m *MockStore) DeleteFailoverPolicy(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockStore) InsertFailoverEvent(ctx context.Context, event *ops.FailoverEvent) error {
	return nil
}

func (m *MockStore) UpdateFailoverEvent(ctx context.Context, id uuid.UUID, status string, errorMsg *string, completedAt *time.Time) error {
	return nil
}

func (m *MockStore) ListFailoverEvents(ctx context.Context, policyID uuid.UUID, limit int) ([]ops.FailoverEvent, error) {
	return []ops.FailoverEvent{}, nil
}

func (m *MockStore) ListIncidentFailoverEvents(ctx context.Context, incidentID uuid.UUID) ([]ops.FailoverEvent, error) {
	return []ops.FailoverEvent{}, nil
}

func (m *MockStore) UpsertFailoverMetrics(ctx context.Context, metrics *ops.FailoverMetrics) error {
	return nil
}

func (m *MockStore) GetFailoverMetrics(ctx context.Context, policyID uuid.UUID) (*ops.FailoverMetrics, error) {
	return nil, nil
}

// Phase 3.11: Failover Chain Orchestration - MockStore implementations
func (m *MockStore) InsertFailoverChain(ctx context.Context, chain *ops.FailoverChain) error {
	return nil
}

func (m *MockStore) GetFailoverChain(ctx context.Context, id uuid.UUID) (*ops.FailoverChain, error) {
	return nil, nil
}

func (m *MockStore) ListFailoverChains(ctx context.Context, tenantID uuid.UUID) ([]ops.FailoverChain, error) {
	return []ops.FailoverChain{}, nil
}

func (m *MockStore) UpdateFailoverChain(ctx context.Context, id uuid.UUID, chain *ops.FailoverChain) error {
	return nil
}

func (m *MockStore) DeleteFailoverChain(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockStore) InsertFailoverChainExecution(ctx context.Context, execution *ops.FailoverChainExecution) error {
	return nil
}

func (m *MockStore) UpdateFailoverChainExecution(ctx context.Context, id uuid.UUID, status string, stepsExecuted []string, failureReasons []string, completedAt *time.Time) error {
	return nil
}

func (m *MockStore) ListFailoverChainExecutions(ctx context.Context, chainID uuid.UUID, limit int) ([]ops.FailoverChainExecution, error) {
	return []ops.FailoverChainExecution{}, nil
}

func (m *MockStore) ListIncidentChainExecutions(ctx context.Context, incidentID uuid.UUID) ([]ops.FailoverChainExecution, error) {
	return []ops.FailoverChainExecution{}, nil
}

func (m *MockStore) UpsertFailoverChainMetrics(ctx context.Context, metrics *ops.FailoverChainMetrics) error {
	return nil
}

func (m *MockStore) GetFailoverChainMetrics(ctx context.Context, chainID uuid.UUID) (*ops.FailoverChainMetrics, error) {
	return nil, nil
}

// Phase 3.12: Multi-Tenant & Priority Failover (MockStore stubs)
func (m *MockStore) InsertFailoverChainState(ctx context.Context, state *ops.FailoverChainState) error {
	return nil
}

func (m *MockStore) UpdateFailoverChainState(ctx context.Context, id uuid.UUID, state *ops.FailoverChainState) error {
	return nil
}

func (m *MockStore) GetFailoverChainState(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID) (*ops.FailoverChainState, error) {
	return nil, nil
}

func (m *MockStore) ListFailoverChainStates(ctx context.Context, tenantID uuid.UUID) ([]ops.FailoverChainState, error) {
	return nil, nil
}

func (m *MockStore) LockChainForExecution(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID, lockDurationMs int) error {
	return nil
}

func (m *MockStore) UnlockChainForExecution(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID) error {
	return nil
}

func (m *MockStore) InsertFailoverChainConflict(ctx context.Context, conflict *ops.FailoverChainConflict) error {
	return nil
}

func (m *MockStore) ListFailoverChainConflicts(ctx context.Context, tenantID uuid.UUID, chainID uuid.UUID) ([]ops.FailoverChainConflict, error) {
	return nil, nil
}

func (m *MockStore) UpdateConflictResolution(ctx context.Context, conflictID uuid.UUID, resolved bool, rule string) error {
	return nil
}

func (m *MockStore) GetConflictingChains(ctx context.Context, tenantID uuid.UUID, chainID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *MockStore) UpsertChainExecutionMetricsAdvanced(ctx context.Context, metrics *ops.ChainExecutionMetricsAdvanced) error {
	return nil
}

func (m *MockStore) GetChainExecutionMetricsAdvanced(ctx context.Context, chainID uuid.UUID) (*ops.ChainExecutionMetricsAdvanced, error) {
	return nil, nil
}

func (m *MockStore) ListChainsSortedBySLACompliance(ctx context.Context, tenantID uuid.UUID) ([]ops.ChainExecutionMetricsAdvanced, error) {
	return nil, nil
}

func (m *MockStore) InsertChainPriorityExecution(ctx context.Context, execution *ops.ChainPriorityExecution) error {
	return nil
}

func (m *MockStore) UpdateChainPriorityExecution(ctx context.Context, id uuid.UUID, currentIdx int, status string, completedChains []string, failedChains []string) error {
	return nil
}

func (m *MockStore) GetChainPriorityExecution(ctx context.Context, id uuid.UUID) (*ops.ChainPriorityExecution, error) {
	return nil, nil
}

func (m *MockStore) ListPendingChainQueues(ctx context.Context, tenantID uuid.UUID) ([]ops.ChainPriorityExecution, error) {
	return nil, nil
}

// Phase 3.14: Analytics & Trends (MockStore stubs)

func (m *MockStore) UpsertSLAComplianceTrend(ctx context.Context, trend *ops.SLAComplianceTrend) error {
	return nil
}

func (m *MockStore) ListSLAComplianceTrends(ctx context.Context, tenantID uuid.UUID, limit int) ([]ops.SLAComplianceTrend, error) {
	return nil, nil
}

func (m *MockStore) UpsertConflictResolutionTrend(ctx context.Context, trend *ops.ConflictResolutionTrend) error {
	return nil
}

func (m *MockStore) GetConflictResolutionTrend(ctx context.Context, tenantID uuid.UUID, periodStart time.Time) (*ops.ConflictResolutionTrend, error) {
	return nil, nil
}

func (m *MockStore) UpsertChainExecutionStats(ctx context.Context, stats *ops.ChainExecutionStats) error {
	return nil
}

func (m *MockStore) GetChainExecutionStats(ctx context.Context, chainID uuid.UUID) (*ops.ChainExecutionStats, error) {
	return nil, nil
}

func (m *MockStore) UpsertChainHealthReport(ctx context.Context, report *ops.ChainHealthReport) error {
	return nil
}

func (m *MockStore) GetChainHealthReport(ctx context.Context, chainID uuid.UUID) (*ops.ChainHealthReport, error) {
	return nil, nil
}

func (m *MockStore) ListChainsByFilter(ctx context.Context, criteria *ops.ChainFilterCriteria) ([]ops.FailoverChain, error) {
	return nil, nil
}

func (m *MockStore) SearchChains(ctx context.Context, tenantID uuid.UUID, searchTerm string, limit int) ([]ops.FailoverChain, error) {
	return nil, nil
}

func (m *MockStore) InsertBatchConflictResolution(ctx context.Context, batch *ops.BatchConflictResolution) error {
	return nil
}

func (m *MockStore) UpdateBatchConflictResolution(ctx context.Context, id uuid.UUID, resolvedCount int, failedCount int, status string) error {
	return nil
}

func (m *MockStore) GetBatchConflictResolution(ctx context.Context, id uuid.UUID) (*ops.BatchConflictResolution, error) {
	return nil, nil
}

// generateRandomEvents creates a random sequence of incident events
func generateRandomEvents(count int) []ops.Event {
	events := make([]ops.Event, count)
	eventTypes := []ops.EventType{
		ops.EventLatencyAnomaly,
		ops.EventFingerprint,
		ops.EventEndpointHealth,
		ops.EventAlert,
	}
	severities := []ops.Severity{
		ops.SeverityInfo,
		ops.SeverityWarning,
		ops.SeverityError,
		ops.SeverityCritical,
	}

	now := time.Now()
	for i := 0; i < count; i++ {
		eventType := eventTypes[rand.Intn(len(eventTypes))]
		severity := severities[rand.Intn(len(severities))]

		events[i] = ops.Event{
			ID:         uuid.New(),
			EventType:  eventType,
			Severity:   severity,
			OccurredAt: now.Add(time.Duration(i*100) * time.Millisecond),
		}
	}

	return events
}

// Phase 3.4: generateRegionAwareEvents creates events with region information
func generateRegionAwareEvents(count int, region string) []ops.Event {
	events := make([]ops.Event, count)
	eventTypes := []ops.EventType{
		ops.EventLatencyAnomaly,
		ops.EventFingerprint,
		ops.EventEndpointHealth,
		ops.EventAlert,
	}
	severities := []ops.Severity{
		ops.SeverityInfo,
		ops.SeverityWarning,
		ops.SeverityError,
		ops.SeverityCritical,
	}

	now := time.Now()
	for i := 0; i < count; i++ {
		eventType := eventTypes[rand.Intn(len(eventTypes))]
		severity := severities[rand.Intn(len(severities))]
		regionCopy := region // Create copy for pointer

		events[i] = ops.Event{
			ID:         uuid.New(),
			EventType:  eventType,
			Severity:   severity,
			Region:     &regionCopy, // Phase 3.4: Add region to event
			OccurredAt: now.Add(time.Duration(i*100) * time.Millisecond),
		}
	}

	return events
}

// Phase 3.4: generateCrossRegionEvents creates events spanning multiple regions to test propagation
func generateCrossRegionEvents(count int, region1, region2 string) []ops.Event {
	events := make([]ops.Event, count)
	eventTypes := []ops.EventType{
		ops.EventLatencyAnomaly,
		ops.EventFingerprint,
		ops.EventEndpointHealth,
		ops.EventAlert,
	}
	severities := []ops.Severity{
		ops.SeverityInfo,
		ops.SeverityWarning,
		ops.SeverityError,
		ops.SeverityCritical,
	}

	now := time.Now()
	for i := 0; i < count; i++ {
		eventType := eventTypes[rand.Intn(len(eventTypes))]
		severity := severities[rand.Intn(len(severities))]

		// Alternate between regions
		var region string
		if i%2 == 0 {
			region = region1
		} else {
			region = region2
		}
		regionCopy := region

		events[i] = ops.Event{
			ID:         uuid.New(),
			EventType:  eventType,
			Severity:   severity,
			Region:     &regionCopy, // Phase 3.4: Add region for cross-region propagation
			OccurredAt: now.Add(time.Duration(i*100) * time.Millisecond),
		}
	}

	return events
}

// runLoadTest executes a load test with the given parameters
func runLoadTest(incidents, eventsPerIncident, workers int) LoadTestResult {
	startTime := time.Now()
	store := &MockStore{}
	engine := ops.NewCorrelationEngine(store)
	matcher := ops.NewPatternMatcher(store)

	var wg sync.WaitGroup
	var result LoadTestResult

	incidentsPerWorker := incidents / workers
	if incidents%workers != 0 {
		incidentsPerWorker++
	}

	fmt.Printf("[TEST] Starting load test: %d incidents, %d events/incident, %d workers\n",
		incidents, eventsPerIncident, workers)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			localRCALatencies := []int64{}
			localPatternLatencies := []int64{}

			for i := 0; i < incidentsPerWorker; i++ {
				// Generate random events for incident
				events := generateRandomEvents(eventsPerIncident)

				// Compute RCA
				rcaStart := time.Now()
				rca := engine.ComputeRCA(events)
				rcaLatency := time.Since(rcaStart).Milliseconds()
				localRCALatencies = append(localRCALatencies, rcaLatency)

				if rca != nil && i%100 == 0 {
					fmt.Printf("[W%d] RCA: confidence=%.3f, chain=%d\n",
						workerID, rca.ConfidenceScore, len(rca.CausalityChain))
				}

				// Pattern matching
				patternStart := time.Now()
				pattern := matcher.CreateIncidentPattern(events)
				patternLatency := time.Since(patternStart).Milliseconds()
				localPatternLatencies = append(localPatternLatencies, patternLatency)

				atomic.AddInt64(&result.TotalIncidents, 1)
				atomic.AddInt64(&result.TotalEvents, int64(len(events)))
				atomic.AddInt64(&result.TotalRCAComputations, 1)
				atomic.AddInt64(&result.TotalPatternMatches, 1)

				if pattern == nil {
					atomic.AddInt64(&result.ErrorCount, 1)
				}
			}

			// Calculate local max latencies
			for _, lat := range localRCALatencies {
				for {
					current := atomic.LoadInt64(&result.MaxRCALatencyMs)
					if lat <= current || atomic.CompareAndSwapInt64(&result.MaxRCALatencyMs, current, lat) {
						break
					}
				}
			}
			for _, lat := range localPatternLatencies {
				for {
					current := atomic.LoadInt64(&result.MaxPatternLatencyMs)
					if lat <= current || atomic.CompareAndSwapInt64(&result.MaxPatternLatencyMs, current, lat) {
						break
					}
				}
			}
		}(w)
	}

	wg.Wait()
	result.DurationMs = time.Since(startTime).Milliseconds()

	return result
}

// Phase 3.4: runRegionalLoadTest executes regional multi-region load testing
func runRegionalLoadTest(incidents, eventsPerIncident, workers int) RegionalLoadTestResult {
	fmt.Println("\n[PHASE 3.4] Starting regional load test...")
	var result RegionalLoadTestResult

	// Test 1: Same region events (baseline - should have highest correlation scores)
	fmt.Println("[REGION TEST 1/3] Same-region events (us-east-1)...")
	startSame := time.Now()
	store := &MockStore{}
	engine := ops.NewCorrelationEngine(store)

	var wg sync.WaitGroup
	incidentsPerWorker := incidents / workers
	if incidents%workers != 0 {
		incidentsPerWorker++
	}

	// Same region test
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < incidentsPerWorker; i++ {
				events := generateRegionAwareEvents(eventsPerIncident, "us-east-1")
				_ = engine.ComputeRCA(events)
				atomic.AddInt64(&result.SameRegionResult.TotalIncidents, 1)
				atomic.AddInt64(&result.SameRegionResult.TotalEvents, int64(len(events)))
				atomic.AddInt64(&result.SameRegionResult.TotalRCAComputations, 1)
			}
		}(w)
	}
	wg.Wait()
	result.SameRegionResult.DurationMs = time.Since(startSame).Milliseconds()

	// Test 2: Adjacent region events (us-east-1 and us-east-2)
	fmt.Println("[REGION TEST 2/3] Adjacent-region events (us-east-1 ↔ us-east-2)...")
	startAdj := time.Now()
	engine = ops.NewCorrelationEngine(store)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < incidentsPerWorker; i++ {
				events := generateCrossRegionEvents(eventsPerIncident, "us-east-1", "us-east-2")
				_ = engine.ComputeRCA(events)
				atomic.AddInt64(&result.AdjacentRegionResult.TotalIncidents, 1)
				atomic.AddInt64(&result.AdjacentRegionResult.TotalEvents, int64(len(events)))
				atomic.AddInt64(&result.AdjacentRegionResult.TotalRCAComputations, 1)
			}
		}(w)
	}
	wg.Wait()
	result.AdjacentRegionResult.DurationMs = time.Since(startAdj).Milliseconds()

	// Test 3: Cross-region events (us-east-1 and eu-west-1)
	fmt.Println("[REGION TEST 3/3] Cross-region events (us-east-1 ↔ eu-west-1)...")
	startCross := time.Now()
	engine = ops.NewCorrelationEngine(store)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < incidentsPerWorker; i++ {
				events := generateCrossRegionEvents(eventsPerIncident, "us-east-1", "eu-west-1")
				_ = engine.ComputeRCA(events)
				atomic.AddInt64(&result.CrossRegionResult.TotalIncidents, 1)
				atomic.AddInt64(&result.CrossRegionResult.TotalEvents, int64(len(events)))
				atomic.AddInt64(&result.CrossRegionResult.TotalRCAComputations, 1)
			}
		}(w)
	}
	wg.Wait()
	result.CrossRegionResult.DurationMs = time.Since(startCross).Milliseconds()

	// Benchmark region discovery overhead
	fmt.Println("[REGION TEST] Benchmarking region discovery overhead...")
	discoveryStart := time.Now()
	for i := 0; i < 10000; i++ {
		regions := []string{"us-east-1", "us-east-2", "eu-west-1", "eu-west-2", "ap-southeast-1"}
		region1 := regions[rand.Intn(len(regions))]
		region2 := regions[rand.Intn(len(regions))]
		// Simulate region affinity lookup
		_ = ops.GetRegionAffinityScore(&region1, &region2)
	}
	result.RegionDiscoveryMs = time.Since(discoveryStart).Milliseconds()

	return result
}

// printResults prints the load test results
func printResults(result LoadTestResult) {
	fmt.Println("\n========== LOAD TEST RESULTS ==========")
	fmt.Printf("Duration:                  %dms\n", result.DurationMs)
	fmt.Printf("Total Incidents:           %d\n", result.TotalIncidents)
	fmt.Printf("Total Events:              %d\n", result.TotalEvents)
	fmt.Printf("RCA Computations:          %d\n", result.TotalRCAComputations)
	fmt.Printf("Pattern Matches:           %d\n", result.TotalPatternMatches)
	fmt.Printf("Max RCA Latency:           %dms\n", result.MaxRCALatencyMs)
	fmt.Printf("Max Pattern Latency:       %dms\n", result.MaxPatternLatencyMs)

	if result.TotalIncidents > 0 {
		throughputIncidents := float64(result.TotalIncidents) * 1000.0 / float64(result.DurationMs)
		throughputEvents := float64(result.TotalEvents) * 1000.0 / float64(result.DurationMs)
		throughputRCA := float64(result.TotalRCAComputations) * 1000.0 / float64(result.DurationMs)

		fmt.Printf("\nThroughput:\n")
		fmt.Printf("  Incidents/sec:           %.1f\n", throughputIncidents)
		fmt.Printf("  Events/sec:              %.1f\n", throughputEvents)
		fmt.Printf("  RCA computations/sec:    %.1f\n", throughputRCA)
	}

	fmt.Println("======================================")

	// Performance assessment
	fmt.Println("[PERFORMANCE ASSESSMENT]")
	if result.MaxRCALatencyMs <= 500 {
		fmt.Println("✓ RCA latency: EXCELLENT (≤500ms)")
	} else if result.MaxRCALatencyMs <= 1000 {
		fmt.Println("⚠ RCA latency: ACCEPTABLE (≤1000ms)")
	} else {
		fmt.Println("✗ RCA latency: POOR (>1000ms)")
	}

	if result.MaxPatternLatencyMs <= 300 {
		fmt.Println("✓ Pattern latency: EXCELLENT (≤300ms)")
	} else if result.MaxPatternLatencyMs <= 1000 {
		fmt.Println("⚠ Pattern latency: ACCEPTABLE (≤1000ms)")
	} else {
		fmt.Println("✗ Pattern latency: POOR (>1000ms)")
	}

	if result.ErrorCount == 0 {
		fmt.Println("✓ Error rate: 0%")
	} else {
		rate := float64(result.ErrorCount) / float64(result.TotalIncidents) * 100.0
		fmt.Printf("✗ Error rate: %.2f%%\n", rate)
	}
}

// Phase 3.4: printRegionalResults prints the regional load test results
func printRegionalResults(result RegionalLoadTestResult) {
	fmt.Println("\n========== PHASE 3.4: REGIONAL LOAD TEST RESULTS ==========")

	// Same region results
	fmt.Println("\n[SAME REGION TEST] us-east-1 (Baseline)")
	fmt.Printf("Duration:                  %dms\n", result.SameRegionResult.DurationMs)
	fmt.Printf("Total Incidents:           %d\n", result.SameRegionResult.TotalIncidents)
	fmt.Printf("Total Events:              %d\n", result.SameRegionResult.TotalEvents)
	fmt.Printf("RCA Computations:          %d\n", result.SameRegionResult.TotalRCAComputations)
	if result.SameRegionResult.TotalIncidents > 0 {
		throughput := float64(result.SameRegionResult.TotalIncidents) * 1000.0 / float64(result.SameRegionResult.DurationMs)
		fmt.Printf("Throughput:                %.1f incidents/sec\n", throughput)
	}

	// Adjacent region results
	fmt.Println("\n[ADJACENT REGION TEST] us-east-1 ↔ us-east-2 (Regional Affinity: 0.7)")
	fmt.Printf("Duration:                  %dms\n", result.AdjacentRegionResult.DurationMs)
	fmt.Printf("Total Incidents:           %d\n", result.AdjacentRegionResult.TotalIncidents)
	fmt.Printf("Total Events:              %d\n", result.AdjacentRegionResult.TotalEvents)
	fmt.Printf("RCA Computations:          %d\n", result.AdjacentRegionResult.TotalRCAComputations)
	if result.AdjacentRegionResult.TotalIncidents > 0 {
		throughput := float64(result.AdjacentRegionResult.TotalIncidents) * 1000.0 / float64(result.AdjacentRegionResult.DurationMs)
		fmt.Printf("Throughput:                %.1f incidents/sec\n", throughput)
	}

	// Cross region results
	fmt.Println("\n[CROSS REGION TEST] us-east-1 ↔ eu-west-1 (Regional Affinity: 0.3)")
	fmt.Printf("Duration:                  %dms\n", result.CrossRegionResult.DurationMs)
	fmt.Printf("Total Incidents:           %d\n", result.CrossRegionResult.TotalIncidents)
	fmt.Printf("Total Events:              %d\n", result.CrossRegionResult.TotalEvents)
	fmt.Printf("RCA Computations:          %d\n", result.CrossRegionResult.TotalRCAComputations)
	if result.CrossRegionResult.TotalIncidents > 0 {
		throughput := float64(result.CrossRegionResult.TotalIncidents) * 1000.0 / float64(result.CrossRegionResult.DurationMs)
		fmt.Printf("Throughput:                %.1f incidents/sec\n", throughput)
	}

	// Region discovery benchmark
	fmt.Println("\n[REGION DISCOVERY BENCHMARK]")
	fmt.Printf("10,000 region affinity lookups: %dms\n", result.RegionDiscoveryMs)
	fmt.Printf("Per-lookup overhead:           %.2fµs\n", float64(result.RegionDiscoveryMs)*1000.0/10000.0)

	// Analysis and comparison
	fmt.Println("\n========== REGIONAL ANALYSIS ==========")
	fmt.Println("\n✓ Region-Aware RCA Scoring Impact:")
	fmt.Printf("  Same-region RCA:      %.0f incidents/sec\n", float64(result.SameRegionResult.TotalIncidents)*1000.0/float64(result.SameRegionResult.DurationMs))
	fmt.Printf("  Adjacent-region RCA:  %.0f incidents/sec (%.1f%% overhead)\n",
		float64(result.AdjacentRegionResult.TotalIncidents)*1000.0/float64(result.AdjacentRegionResult.DurationMs),
		(float64(result.AdjacentRegionResult.DurationMs)/float64(result.SameRegionResult.DurationMs)-1.0)*100.0)
	fmt.Printf("  Cross-region RCA:     %.0f incidents/sec (%.1f%% overhead)\n",
		float64(result.CrossRegionResult.TotalIncidents)*1000.0/float64(result.CrossRegionResult.DurationMs),
		(float64(result.CrossRegionResult.DurationMs)/float64(result.SameRegionResult.DurationMs)-1.0)*100.0)

	// Performance assessment
	fmt.Println("\n[PHASE 3.4 ASSESSMENT]")
	if result.RegionDiscoveryMs < 100 {
		fmt.Println("✓ Region discovery overhead: EXCELLENT (<100ms for 10k lookups)")
	} else if result.RegionDiscoveryMs < 500 {
		fmt.Println("⚠ Region discovery overhead: ACCEPTABLE (<500ms for 10k lookups)")
	} else {
		fmt.Println("✗ Region discovery overhead: POOR (>500ms for 10k lookups)")
	}

	if result.SameRegionResult.DurationMs > 0 && result.CrossRegionResult.DurationMs > 0 {
		overhead := (float64(result.CrossRegionResult.DurationMs)/float64(result.SameRegionResult.DurationMs) - 1.0) * 100.0
		if overhead < 5 {
			fmt.Println("✓ Cross-region RCA penalty: ACCEPTABLE (<5% overhead)")
		} else if overhead < 15 {
			fmt.Println("⚠ Cross-region RCA penalty: MODERATE (5-15% overhead)")
		} else {
			fmt.Printf("✗ Cross-region RCA penalty: HIGH (%.1f%% overhead)\n", overhead)
		}
	}

	fmt.Println("\n========== Phase 3.4 Complete ==========")
}

func main() {
	incidents := flag.Int("incidents", 1000, "Number of incidents to simulate")
	eventsPerIncident := flag.Int("events", 5, "Number of events per incident")
	workers := flag.Int("workers", 4, "Number of concurrent workers")
	testName := flag.String("test", "throughput", "Test name (throughput, latency, stability, regional)")

	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	fmt.Println("\n🔥 SemLayer Load Test Suite")
	fmt.Printf("Test Type: %s\n", *testName)

	// Phase 3.4: Support regional load testing
	if *testName == "regional" {
		result := runRegionalLoadTest(*incidents, *eventsPerIncident, *workers)
		printRegionalResults(result)
	} else {
		result := runLoadTest(*incidents, *eventsPerIncident, *workers)
		printResults(result)
	}
}
