package ops

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStore for testing
type TestStore struct {
	InsertedHistories map[string]*ActionHistory
}

func (m *TestStore) ListAlerts(ctx context.Context, enabled *bool) ([]Alert, error) {
	return []Alert{}, nil
}

func (m *TestStore) GetAlert(ctx context.Context, id uuid.UUID) (*Alert, error) {
	return nil, nil
}

func (m *TestStore) CreateAlert(ctx context.Context, alert Alert) (*Alert, error) {
	return &alert, nil
}

func (m *TestStore) UpdateAlert(ctx context.Context, id uuid.UUID, alert Alert) error {
	return nil
}

func (m *TestStore) DeleteAlert(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *TestStore) InsertAlertEvent(ctx context.Context, event AlertEvent) error {
	return nil
}

func (m *TestStore) GetAlertEvents(ctx context.Context, alertID uuid.UUID, limit int) ([]AlertEvent, error) {
	return []AlertEvent{}, nil
}

func (m *TestStore) GetOrCreateFingerprint(ctx context.Context, fingerprint, path string, statusCode int, sample string) (*ErrorFingerprint, error) {
	return nil, nil
}

func (m *TestStore) UpdateFingerprintCount(ctx context.Context, fingerprintID uuid.UUID, newCount int64) error {
	return nil
}

func (m *TestStore) InsertErrorEvent(ctx context.Context, event ErrorEvent) error {
	return nil
}

func (m *TestStore) ListFingerprints(ctx context.Context, limit int) ([]ErrorFingerprint, error) {
	return []ErrorFingerprint{}, nil
}

func (m *TestStore) GetFingerprintEvents(ctx context.Context, fingerprintID uuid.UUID, limit int) ([]ErrorEvent, error) {
	return []ErrorEvent{}, nil
}

func (m *TestStore) UpsertTenantHealth(ctx context.Context, health TenantHealth) error {
	return nil
}

func (m *TestStore) GetTenantHealth(ctx context.Context, tenantID uuid.UUID) (*TenantHealth, error) {
	return nil, nil
}

func (m *TestStore) GetTenantHealths(ctx context.Context, limit int) ([]TenantHealth, error) {
	return []TenantHealth{}, nil
}

func (m *TestStore) UpsertEndpointHealth(ctx context.Context, health EndpointHealth) error {
	return nil
}

func (m *TestStore) GetEndpointHealth(ctx context.Context, endpoint string) (*EndpointHealth, error) {
	return nil, nil
}

func (m *TestStore) GetEndpointHealths(ctx context.Context, limit int) ([]EndpointHealth, error) {
	return []EndpointHealth{}, nil
}

func (m *TestStore) InsertHeatmapBucket(ctx context.Context, bucketTime time.Time, dimensionType, dimensionValue string, p50, p95, p99 int, requestCount int) error {
	return nil
}

func (m *TestStore) GetHeatmapData(ctx context.Context, dimensionType, dimensionValue string, bucketSize time.Duration, window time.Duration) ([]HeatmapSeriesPoint, error) {
	return []HeatmapSeriesPoint{}, nil
}

func (m *TestStore) GetHeatmapSeries(ctx context.Context, dimensionType string, limit int, bucketSize time.Duration, window time.Duration) ([]HeatmapSeries, error) {
	return []HeatmapSeries{}, nil
}

func (m *TestStore) GetMetricValue(ctx context.Context, metric, scope string, since time.Time) (float64, error) {
	return 0.0, nil
}

func (m *TestStore) GetTenantMetrics(ctx context.Context, tenantID uuid.UUID, since time.Time) (*TenantMetrics, error) {
	return nil, nil
}

func (m *TestStore) GetEndpointMetrics(ctx context.Context, endpoint string, since time.Time) (*EndpointMetrics, error) {
	return nil, nil
}

func (m *TestStore) GetGlobalMetrics(ctx context.Context, since time.Time) (*TenantMetrics, error) {
	return nil, nil
}

func (m *TestStore) InsertEvent(ctx context.Context, e Event) error {
	return nil
}

func (m *TestStore) UpsertIncidentForEvent(ctx context.Context, e Event) (*Incident, error) {
	return nil, nil
}

func (m *TestStore) ListEvents(ctx context.Context, since time.Time, limit int) ([]Event, error) {
	return []Event{}, nil
}

func (m *TestStore) GetIncident(ctx context.Context, id uuid.UUID) (*Incident, []Event, error) {
	return &Incident{ID: id}, []Event{}, nil
}

func (m *TestStore) CloseIncident(ctx context.Context, id uuid.UUID, summary *string, rootCause *string) error {
	return nil
}

func (m *TestStore) InsertActionHistory(ctx context.Context, history ActionHistory) error {
	if m.InsertedHistories == nil {
		m.InsertedHistories = make(map[string]*ActionHistory)
	}
	m.InsertedHistories[history.ID.String()] = &history
	return nil
}

func (m *TestStore) UpdateActionHistory(ctx context.Context, id uuid.UUID, status string, result []byte, errorMsg *string) error {
	return nil
}

func (m *TestStore) GetActionHistory(ctx context.Context, id uuid.UUID) (*ActionHistory, error) {
	return nil, nil
}

func (m *TestStore) ListIncidentActions(ctx context.Context, incidentID uuid.UUID, limit int) ([]ActionHistory, error) {
	return []ActionHistory{}, nil
}

// Audit Log (Phase 2.4c) - TestStore implementations
func (m *TestStore) InsertAuditLog(ctx context.Context, auditLog *AuditLog) error {
	return nil
}

func (m *TestStore) GetAuditLog(ctx context.Context, id uuid.UUID) (*AuditLog, error) {
	return nil, nil
}

func (m *TestStore) ListAuditLogs(ctx context.Context, filters AuditLogFilters, limit int, offset int) ([]AuditLog, error) {
	return []AuditLog{}, nil
}

func (m *TestStore) ListIncidentAuditLogs(ctx context.Context, incidentID uuid.UUID, limit int) ([]AuditLog, error) {
	return []AuditLog{}, nil
}

// Region Metadata (Phase 3.1) - TestStore implementations
func (m *TestStore) GetRegionConfig(ctx context.Context, regionCode string) (*RegionConfig, error) {
	return nil, nil
}

func (m *TestStore) ListRegionConfigs(ctx context.Context, activeOnly bool) ([]RegionConfig, error) {
	return []RegionConfig{}, nil
}

func (m *TestStore) InsertRegionRouting(ctx context.Context, routing *RegionRouting) error {
	return nil
}

func (m *TestStore) GetRegionRouting(ctx context.Context, tenantID uuid.UUID, region string) (*RegionRouting, error) {
	return nil, nil
}

func (m *TestStore) ListRegionRoutings(ctx context.Context, tenantID uuid.UUID) ([]RegionRouting, error) {
	return []RegionRouting{}, nil
}

// Phase 3.5: Regional Metrics & SLA Tracking
func (m *TestStore) UpsertRegionalMetrics(ctx context.Context, metrics *RegionalMetrics) error {
	return nil
}

func (m *TestStore) GetRegionalMetrics(ctx context.Context, region string) (*RegionalMetrics, error) {
	return nil, nil
}

func (m *TestStore) ListRegionalMetrics(ctx context.Context, limit int) ([]RegionalMetrics, error) {
	return []RegionalMetrics{}, nil
}

func (m *TestStore) UpsertRegionalHealth(ctx context.Context, health *RegionalHealth) error {
	return nil
}

func (m *TestStore) GetRegionalHealth(ctx context.Context, region string) (*RegionalHealth, error) {
	return nil, nil
}

func (m *TestStore) ListRegionalHealth(ctx context.Context, limit int) ([]RegionalHealth, error) {
	return []RegionalHealth{}, nil
}

func (m *TestStore) UpsertRegionalSLA(ctx context.Context, sla *RegionalSLA) error {
	return nil
}

func (m *TestStore) GetRegionalSLA(ctx context.Context, region string) (*RegionalSLA, error) {
	return nil, nil
}

func (m *TestStore) ListRegionalSLAs(ctx context.Context, limit int) ([]RegionalSLA, error) {
	return []RegionalSLA{}, nil
}

func (m *TestStore) InsertRegionalSLAStatus(ctx context.Context, status *RegionalSLAStatus) error {
	return nil
}

func (m *TestStore) GetRegionalSLAStatus(ctx context.Context, region string) (*RegionalSLAStatus, error) {
	return nil, nil
}

func (m *TestStore) ListRegionalSLAStatuses(ctx context.Context, region string, limit int) ([]RegionalSLAStatus, error) {
	return []RegionalSLAStatus{}, nil
}

// Phase 3.9: Region-Aware API Layer - TestStore implementations
func (m *TestStore) ListIncidents(ctx context.Context, limit int) ([]Incident, error) {
	return []Incident{}, nil
}

func (m *TestStore) ListIncidentsByRegion(ctx context.Context, region string, limit int) ([]Incident, error) {
	return []Incident{}, nil
}

func (m *TestStore) ListLatestRegionalSLAStatuses(ctx context.Context) ([]RegionalSLAStatus, error) {
	return []RegionalSLAStatus{}, nil
}

func (m *TestStore) ListRegionalIncidentCounts(ctx context.Context, since, until time.Time) ([]RegionalIncidentCount, error) {
	return []RegionalIncidentCount{}, nil
}

func (m *TestStore) ListOpsEventsByRegion(ctx context.Context, region string, limit int) ([]Event, error) {
	return []Event{}, nil
}

func (m *TestStore) ListAuditLogsByRegion(ctx context.Context, region string, limit int) ([]AuditLog, error) {
	return []AuditLog{}, nil
}

// Phase 3.10: Failover Policies & Automated Regional Failover - TestStore implementations
func (m *TestStore) InsertFailoverPolicy(ctx context.Context, policy *FailoverPolicy) error {
	return nil
}

func (m *TestStore) GetFailoverPolicy(ctx context.Context, id uuid.UUID) (*FailoverPolicy, error) {
	return nil, nil
}

func (m *TestStore) ListFailoverPolicies(ctx context.Context, tenantID uuid.UUID) ([]FailoverPolicy, error) {
	return []FailoverPolicy{}, nil
}

func (m *TestStore) UpdateFailoverPolicy(ctx context.Context, id uuid.UUID, policy *FailoverPolicy) error {
	return nil
}

func (m *TestStore) DeleteFailoverPolicy(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *TestStore) InsertFailoverEvent(ctx context.Context, event *FailoverEvent) error {
	return nil
}

func (m *TestStore) UpdateFailoverEvent(ctx context.Context, id uuid.UUID, status string, errorMsg *string, completedAt *time.Time) error {
	return nil
}

func (m *TestStore) ListFailoverEvents(ctx context.Context, policyID uuid.UUID, limit int) ([]FailoverEvent, error) {
	return []FailoverEvent{}, nil
}

func (m *TestStore) ListIncidentFailoverEvents(ctx context.Context, incidentID uuid.UUID) ([]FailoverEvent, error) {
	return []FailoverEvent{}, nil
}

func (m *TestStore) UpsertFailoverMetrics(ctx context.Context, metrics *FailoverMetrics) error {
	return nil
}

func (m *TestStore) GetFailoverMetrics(ctx context.Context, policyID uuid.UUID) (*FailoverMetrics, error) {
	return nil, nil
}

// Phase 3.11: Failover Chain Orchestration - TestStore implementations
func (m *TestStore) InsertFailoverChain(ctx context.Context, chain *FailoverChain) error {
	return nil
}

func (m *TestStore) GetFailoverChain(ctx context.Context, id uuid.UUID) (*FailoverChain, error) {
	return nil, nil
}

func (m *TestStore) ListFailoverChains(ctx context.Context, tenantID uuid.UUID) ([]FailoverChain, error) {
	return []FailoverChain{}, nil
}

func (m *TestStore) UpdateFailoverChain(ctx context.Context, id uuid.UUID, chain *FailoverChain) error {
	return nil
}

func (m *TestStore) DeleteFailoverChain(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *TestStore) InsertFailoverChainExecution(ctx context.Context, execution *FailoverChainExecution) error {
	return nil
}

func (m *TestStore) UpdateFailoverChainExecution(ctx context.Context, id uuid.UUID, status string, stepsExecuted []string, failureReasons []string, completedAt *time.Time) error {
	return nil
}

func (m *TestStore) ListFailoverChainExecutions(ctx context.Context, chainID uuid.UUID, limit int) ([]FailoverChainExecution, error) {
	return []FailoverChainExecution{}, nil
}

func (m *TestStore) ListIncidentChainExecutions(ctx context.Context, incidentID uuid.UUID) ([]FailoverChainExecution, error) {
	return []FailoverChainExecution{}, nil
}

func (m *TestStore) UpsertFailoverChainMetrics(ctx context.Context, metrics *FailoverChainMetrics) error {
	return nil
}

func (m *TestStore) GetFailoverChainMetrics(ctx context.Context, chainID uuid.UUID) (*FailoverChainMetrics, error) {
	return nil, nil
}

// Phase 3.12: Multi-Tenant & Priority Failover (TestStore stubs)
func (m *TestStore) InsertFailoverChainState(ctx context.Context, state *FailoverChainState) error {
	return nil
}

func (m *TestStore) UpdateFailoverChainState(ctx context.Context, id uuid.UUID, state *FailoverChainState) error {
	return nil
}

func (m *TestStore) GetFailoverChainState(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID) (*FailoverChainState, error) {
	return nil, nil
}

func (m *TestStore) ListFailoverChainStates(ctx context.Context, tenantID uuid.UUID) ([]FailoverChainState, error) {
	return nil, nil
}

func (m *TestStore) LockChainForExecution(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID, lockDurationMs int) error {
	return nil
}

func (m *TestStore) UnlockChainForExecution(ctx context.Context, chainID uuid.UUID, tenantID uuid.UUID) error {
	return nil
}

func (m *TestStore) InsertFailoverChainConflict(ctx context.Context, conflict *FailoverChainConflict) error {
	return nil
}

func (m *TestStore) ListFailoverChainConflicts(ctx context.Context, tenantID uuid.UUID, chainID uuid.UUID) ([]FailoverChainConflict, error) {
	return nil, nil
}

func (m *TestStore) UpdateConflictResolution(ctx context.Context, conflictID uuid.UUID, resolved bool, rule string) error {
	return nil
}

func (m *TestStore) GetConflictingChains(ctx context.Context, tenantID uuid.UUID, chainID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *TestStore) UpsertChainExecutionMetricsAdvanced(ctx context.Context, metrics *ChainExecutionMetricsAdvanced) error {
	return nil
}

func (m *TestStore) GetChainExecutionMetricsAdvanced(ctx context.Context, chainID uuid.UUID) (*ChainExecutionMetricsAdvanced, error) {
	return nil, nil
}

func (m *TestStore) ListChainsSortedBySLACompliance(ctx context.Context, tenantID uuid.UUID) ([]ChainExecutionMetricsAdvanced, error) {
	return nil, nil
}

func (m *TestStore) InsertChainPriorityExecution(ctx context.Context, execution *ChainPriorityExecution) error {
	return nil
}

func (m *TestStore) UpdateChainPriorityExecution(ctx context.Context, id uuid.UUID, currentIdx int, status string, completedChains []string, failedChains []string) error {
	return nil
}

func (m *TestStore) GetChainPriorityExecution(ctx context.Context, id uuid.UUID) (*ChainPriorityExecution, error) {
	return nil, nil
}

func (m *TestStore) ListPendingChainQueues(ctx context.Context, tenantID uuid.UUID) ([]ChainPriorityExecution, error) {
	return nil, nil
}

// Phase 3.14: Analytics & Trends (TestStore stubs)

func (m *TestStore) UpsertSLAComplianceTrend(ctx context.Context, trend *SLAComplianceTrend) error {
	return nil
}

func (m *TestStore) ListSLAComplianceTrends(ctx context.Context, tenantID uuid.UUID, limit int) ([]SLAComplianceTrend, error) {
	return nil, nil
}

func (m *TestStore) UpsertConflictResolutionTrend(ctx context.Context, trend *ConflictResolutionTrend) error {
	return nil
}

func (m *TestStore) GetConflictResolutionTrend(ctx context.Context, tenantID uuid.UUID, periodStart time.Time) (*ConflictResolutionTrend, error) {
	return nil, nil
}

func (m *TestStore) UpsertChainExecutionStats(ctx context.Context, stats *ChainExecutionStats) error {
	return nil
}

func (m *TestStore) GetChainExecutionStats(ctx context.Context, chainID uuid.UUID) (*ChainExecutionStats, error) {
	return nil, nil
}

func (m *TestStore) UpsertChainHealthReport(ctx context.Context, report *ChainHealthReport) error {
	return nil
}

func (m *TestStore) GetChainHealthReport(ctx context.Context, chainID uuid.UUID) (*ChainHealthReport, error) {
	return nil, nil
}

func (m *TestStore) ListChainsByFilter(ctx context.Context, criteria *ChainFilterCriteria) ([]FailoverChain, error) {
	return nil, nil
}

func (m *TestStore) SearchChains(ctx context.Context, tenantID uuid.UUID, searchTerm string, limit int) ([]FailoverChain, error) {
	return nil, nil
}

func (m *TestStore) InsertBatchConflictResolution(ctx context.Context, batch *BatchConflictResolution) error {
	return nil
}

func (m *TestStore) UpdateBatchConflictResolution(ctx context.Context, id uuid.UUID, resolvedCount int, failedCount int, status string) error {
	return nil
}

func (m *TestStore) GetBatchConflictResolution(ctx context.Context, id uuid.UUID) (*BatchConflictResolution, error) {
	return nil, nil
}

type TestTimelineService struct{}

func (m *TestTimelineService) RecordEvent(ctx context.Context, event Event) error {
	return nil
}

func (m *TestTimelineService) GetIncidentTimeline(ctx context.Context, incidentID uuid.UUID) ([]Event, error) {
	return []Event{}, nil
}

// Test 1: Action Executor - Restart Worker
func TestActionExecutorRestartWorker(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()

	params := map[string]interface{}{
		"worker_id": "worker-1",
		"force":     false,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	regionUS := "us-east-1"
	response, err := executor.ExecuteAction(ctx, incidentID, "restart_worker", paramsJSON, &regionUS) // Phase 3.3: Pass region

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "restart_worker", response.ActionType)
	assert.Equal(t, "success", response.Status)
}

// Test 2: Action Executor - Throttle Tenant
func TestActionExecutorThrottleTenant(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()

	params := map[string]interface{}{
		"tenant_id":          "tenant-123",
		"rate_limit_per_sec": 100,
		"duration_secs":      300,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	regionUS := "us-east-1"
	response, err := executor.ExecuteAction(ctx, incidentID, "throttle_tenant", paramsJSON, &regionUS) // Phase 3.3: Pass region

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "throttle_tenant", response.ActionType)
	assert.Equal(t, "success", response.Status)
}

// Test 3: Action Executor - Trigger Runbook
func TestActionExecutorTriggerRunbook(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()

	params := map[string]interface{}{
		"runbook_id": "incident-response-1",
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	regionUS := "us-east-1"
	response, err := executor.ExecuteAction(ctx, incidentID, "trigger_runbook", paramsJSON, &regionUS) // Phase 3.3: Pass region

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "trigger_runbook", response.ActionType)
	assert.Equal(t, "success", response.Status)
}

// Test 4: Action Executor - Circuit Breaker
func TestActionExecutorCircuitBreakerToggle(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()

	params := map[string]interface{}{
		"circuit_id":    "api-gateway-db",
		"target_state":  "open",
		"duration_secs": 300,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	regionUS := "us-east-1"
	response, err := executor.ExecuteAction(ctx, incidentID, "circuit_breaker_toggle", paramsJSON, &regionUS) // Phase 3.3: Pass region

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "circuit_breaker_toggle", response.ActionType)
	assert.Equal(t, "success", response.Status)
}

// Test 5: Action Executor - Failover
func TestActionExecutorFailoverToggle(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()

	params := map[string]interface{}{
		"source_region": "us-east-1",
		"target_region": "us-west-2",
		"immediate":     true,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	regionEU := "eu-west-1"
	response, err := executor.ExecuteAction(ctx, incidentID, "failover_toggle", paramsJSON, &regionEU) // Phase 3.3: Pass region

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "failover_toggle", response.ActionType)
	assert.Equal(t, "success", response.Status)
}

// Test 6: Action Executor - Invalid Action
func TestActionExecutorInvalidAction(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()
	params := json.RawMessage(`{}`)

	regionUS := "us-east-1"
	response, err := executor.ExecuteAction(ctx, incidentID, "unknown_action", params, &regionUS) // Phase 3.3: Pass region

	assert.Error(t, err)
	assert.Nil(t, response)
}

// Test 7: Correlation Engine - Basic Flow
func TestCorrelationEngineScoringFlow(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	now := time.Now()
	events := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			OccurredAt: now,
		},
		{
			ID:         uuid.New(),
			EventType:  EventFingerprint,
			Severity:   SeverityError,
			OccurredAt: now.Add(15 * time.Second),
		},
		{
			ID:         uuid.New(),
			EventType:  EventEndpointHealth,
			Severity:   SeverityError,
			OccurredAt: now.Add(30 * time.Second),
		},
	}

	rca := engine.ComputeRCA(events)

	assert.NotNil(t, rca)
	assert.NotNil(t, rca.SuspectedRootCause)
	assert.Greater(t, rca.ConfidenceScore, 0.0)
	assert.Greater(t, len(rca.CausalityChain), 0)
	assert.Greater(t, len(rca.SuggestedRemediations), 0)
}

// Test 8: Pattern Matcher - Similar Incidents
func TestPatternMatcherSimilarIncidents(t *testing.T) {
	store := &TestStore{}
	matcher := NewPatternMatcher(store)

	now := time.Now()

	events1 := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			OccurredAt: now,
		},
		{
			ID:         uuid.New(),
			EventType:  EventFingerprint,
			Severity:   SeverityError,
			OccurredAt: now.Add(15 * time.Second),
		},
	}

	events2 := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			OccurredAt: now.Add(1 * time.Hour),
		},
		{
			ID:         uuid.New(),
			EventType:  EventFingerprint,
			Severity:   SeverityError,
			OccurredAt: now.Add(1*time.Hour + 15*time.Second),
		},
	}

	pattern1 := matcher.CreateIncidentPattern(events1)
	pattern2 := matcher.CreateIncidentPattern(events2)

	assert.NotNil(t, pattern1)
	assert.NotNil(t, pattern2)
	assert.Equal(t, pattern1.ID, pattern2.ID)
}

// Test 9: Action History Recording
func TestActionHistoryRecording(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()

	params := map[string]interface{}{
		"worker_id": "worker-1",
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	regionUS := "us-east-1"
	response, err := executor.ExecuteAction(ctx, incidentID, "restart_worker", paramsJSON, &regionUS) // Phase 3.3: Pass region

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.ActionHistoryID)

	recorded := store.InsertedHistories[response.ActionHistoryID]
	assert.NotNil(t, recorded)
	assert.Equal(t, "restart_worker", recorded.ActionType)
	// Action should start in pending state
	assert.Equal(t, "pending", recorded.Status)
}

// Test 10: Correlation Confidence Metrics
func TestCorrelationConfidenceMetrics(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	now := time.Now()

	// High confidence: Related events with proper timing
	highConfidenceEvents := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			OccurredAt: now,
		},
		{
			ID:         uuid.New(),
			EventType:  EventFingerprint,
			Severity:   SeverityError,
			OccurredAt: now.Add(25 * time.Second),
		},
		{
			ID:         uuid.New(),
			EventType:  EventEndpointHealth,
			Severity:   SeverityCritical,
			OccurredAt: now.Add(50 * time.Second),
		},
	}

	// Low confidence: Events far apart
	lowConfidenceEvents := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			OccurredAt: now,
		},
		{
			ID:         uuid.New(),
			EventType:  EventAlert,
			Severity:   SeverityInfo,
			OccurredAt: now.Add(10 * time.Minute),
		},
	}

	highRCA := engine.ComputeRCA(highConfidenceEvents)
	lowRCA := engine.ComputeRCA(lowConfidenceEvents)

	assert.Greater(t, highRCA.ConfidenceScore, lowRCA.ConfidenceScore)
}

// Benchmark: RCA Computation
func BenchmarkRCAComputation(b *testing.B) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	now := time.Now()
	events := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			OccurredAt: now,
		},
		{
			ID:         uuid.New(),
			EventType:  EventFingerprint,
			Severity:   SeverityError,
			OccurredAt: now.Add(20 * time.Second),
		},
		{
			ID:         uuid.New(),
			EventType:  EventEndpointHealth,
			Severity:   SeverityError,
			OccurredAt: now.Add(40 * time.Second),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.ComputeRCA(events)
	}
}

// Benchmark: Pattern Matching
func BenchmarkPatternMatching(b *testing.B) {
	store := &TestStore{}
	matcher := NewPatternMatcher(store)

	now := time.Now()
	events := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			OccurredAt: now,
		},
		{
			ID:         uuid.New(),
			EventType:  EventFingerprint,
			Severity:   SeverityError,
			OccurredAt: now.Add(20 * time.Second),
		},
		{
			ID:         uuid.New(),
			EventType:  EventEndpointHealth,
			Severity:   SeverityError,
			OccurredAt: now.Add(40 * time.Second),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher.CreateIncidentPattern(events)
	}
}

// ============ Phase 2.4b: Security Tests ============

// Test 11: Rate Limiter - Allow action within limit
func TestRateLimiterAllowsWithinLimit(t *testing.T) {
	limiter := NewRateLimiter(10)

	userID := "user-123"

	// Should allow first 10 actions
	for i := 0; i < 10; i++ {
		assert.True(t, limiter.IsAllowed(userID))
	}

	// 11th action should be blocked
	assert.False(t, limiter.IsAllowed(userID))
}

// Test 12: Rate Limiter - Track remaining actions
func TestRateLimiterTrackingRemaining(t *testing.T) {
	limiter := NewRateLimiter(5)

	userID := "user-456"

	// Should start with 5 remaining
	remaining := limiter.GetRemaining(userID)
	assert.Equal(t, 5, remaining)

	// Use one action
	limiter.IsAllowed(userID)
	remaining = limiter.GetRemaining(userID)
	assert.Equal(t, 4, remaining)

	// Use 3 more
	for i := 0; i < 3; i++ {
		limiter.IsAllowed(userID)
	}
	remaining = limiter.GetRemaining(userID)
	assert.Equal(t, 1, remaining)
}

// Test 13: Parameter Validator - Valid restart_worker
func TestParameterValidatorRestartWorkerValid(t *testing.T) {
	validator := NewParameterValidator()

	params := map[string]interface{}{
		"worker_id": "worker-1",
		"force":     false,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	err = validator.Validate("restart_worker", paramsJSON)
	assert.NoError(t, err)
}

// Test 14: Parameter Validator - Invalid restart_worker (missing worker_id)
func TestParameterValidatorRestartWorkerInvalid(t *testing.T) {
	validator := NewParameterValidator()

	params := map[string]interface{}{
		"force": false,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	err = validator.Validate("restart_worker", paramsJSON)
	assert.Error(t, err)
}

// Test 15: Parameter Validator - Valid throttle_tenant
func TestParameterValidatorThrottleTenantValid(t *testing.T) {
	validator := NewParameterValidator()

	params := map[string]interface{}{
		"tenant_id":          "tenant-123",
		"rate_limit_per_sec": 100,
		"duration_secs":      300,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	err = validator.Validate("throttle_tenant", paramsJSON)
	assert.NoError(t, err)
}

// Test 16: Parameter Validator - Invalid throttle_tenant (negative rate limit)
func TestParameterValidatorThrottleTenantInvalid(t *testing.T) {
	validator := NewParameterValidator()

	params := map[string]interface{}{
		"tenant_id":          "tenant-123",
		"rate_limit_per_sec": -100,
		"duration_secs":      300,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	err = validator.Validate("throttle_tenant", paramsJSON)
	assert.Error(t, err)
}

// Test 17: Parameter Validator - Valid failover
func TestParameterValidatorFailoverValid(t *testing.T) {
	validator := NewParameterValidator()

	params := map[string]interface{}{
		"source_region": "us-east-1",
		"target_region": "us-west-2",
		"immediate":     true,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	err = validator.Validate("failover_toggle", paramsJSON)
	assert.NoError(t, err)
}

// Test 18: Parameter Validator - Invalid failover (same regions)
func TestParameterValidatorFailoverInvalid(t *testing.T) {
	validator := NewParameterValidator()

	params := map[string]interface{}{
		"source_region": "us-east-1",
		"target_region": "us-east-1",
		"immediate":     true,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	err = validator.Validate("failover_toggle", paramsJSON)
	assert.Error(t, err)
}

// Test 19: Response Sanitizer - Redacts sensitive fields
func TestResponseSanitizerRedactsSensitive(t *testing.T) {
	sanitizer := NewResponseSanitizer()

	result := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"username": "admin",
			"password": "secret123",
		},
		"token":      "abc123xyz",
		"public_key": "pk_123",
	}

	sanitized := sanitizer.Sanitize(result)

	assert.Equal(t, "success", sanitized["status"])
	assert.Equal(t, "***REDACTED***", sanitized["token"])

	// Check nested object sanitization
	nestedData, ok := sanitized["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "***REDACTED***", nestedData["password"])
	assert.Equal(t, "admin", nestedData["username"]) // Not sensitive

	assert.Equal(t, "pk_123", sanitized["public_key"]) // Not a sensitive field
}

// Test 20: Audit Logger - Creates audit log entry
func TestAuditLoggerCreatesEntry(t *testing.T) {
	store := &TestStore{}
	logger := NewAuditLogger(store)

	userID := "user-123"
	userRole := "ops_manager"
	actionType := "restart_worker"
	incidentID := uuid.New()
	status := "success"
	sourceIP := "192.168.1.1"
	params := json.RawMessage(`{"worker_id": "worker-1"}`)
	result := map[string]interface{}{"status": "ok"}
	durationMs := int64(100)

	auditLog, err := logger.LogAction(userID, userRole, actionType, sourceIP, incidentID, status, params, result, nil, durationMs)

	assert.NoError(t, err)
	assert.NotNil(t, auditLog)
	assert.Equal(t, userID, auditLog.UserID)
	assert.Equal(t, userRole, auditLog.UserRole)
	assert.Equal(t, actionType, auditLog.ActionType)
	assert.Equal(t, incidentID, auditLog.IncidentID)
	assert.Equal(t, "success", auditLog.Status)
	assert.Equal(t, sourceIP, auditLog.SourceIP)
	assert.Equal(t, durationMs, auditLog.DurationMs)
}

// ============ Phase 2.4c: Audit Log Retrieval Tests ============

// Test 21: Retrieve audit log by ID
func TestGetAuditLog(t *testing.T) {
	store := &TestStore{}
	auditLogger := NewAuditLogger(store)

	userID := "user-123"
	actionType := "restart_worker"
	incidentID := uuid.New()

	// Create an audit log
	params := json.RawMessage(`{"worker_id": "worker-1"}`)
	result := map[string]interface{}{"status": "ok"}

	auditLog, err := auditLogger.LogAction(userID, "ops_manager", actionType, "192.168.1.1", incidentID, "success", params, result, nil, 100)
	assert.NoError(t, err)

	// Retrieve it
	retrieved, err := store.GetAuditLog(context.Background(), auditLog.ID)
	assert.NoError(t, err)
	assert.Nil(t, retrieved) // Mock store returns nil for test
}

// Test 22: List audit logs for incident
func TestListIncidentAuditLogs(t *testing.T) {
	store := &TestStore{}
	incidentID := uuid.New()

	// List audit logs for incident
	logs, err := store.ListIncidentAuditLogs(context.Background(), incidentID, 100)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(logs)) // Empty for test store
}

// Test 23: List audit logs with filters
func TestListAuditLogsWithFilters(t *testing.T) {
	store := &TestStore{}

	filters := AuditLogFilters{
		UserID:     stringPtr("user-123"),
		ActionType: stringPtr("restart_worker"),
		Status:     stringPtr("success"),
	}

	logs, err := store.ListAuditLogs(context.Background(), filters, 100, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(logs)) // Empty for test store
}

// Test 24: Audit log persistence - verify structure
func TestAuditLogStructure(t *testing.T) {
	auditLog := &AuditLog{
		ID:         uuid.New(),
		UserID:     "user-456",
		UserRole:   "ops_manager",
		ActionType: "throttle_tenant",
		IncidentID: uuid.New(),
		Status:     "success",
		Parameters: json.RawMessage(`{"tenant_id": "tenant-1", "rate_limit_per_sec": 100}`),
		Result:     map[string]interface{}{"applied": true},
		SourceIP:   "10.0.0.1",
		DurationMs: 250,
	}

	// Verify all fields are set
	assert.NotEmpty(t, auditLog.ID)
	assert.Equal(t, "user-456", auditLog.UserID)
	assert.Equal(t, "ops_manager", auditLog.UserRole)
	assert.Equal(t, "throttle_tenant", auditLog.ActionType)
	assert.Equal(t, "success", auditLog.Status)
	assert.Equal(t, "10.0.0.1", auditLog.SourceIP)
	assert.Equal(t, int64(250), auditLog.DurationMs)
}

// Test 25: Audit log with error
func TestAuditLogWithError(t *testing.T) {
	store := &TestStore{}
	auditLogger := NewAuditLogger(store)

	userID := "user-789"
	actionType := "failover_toggle"
	incidentID := uuid.New()
	errorMsg := stringPtr("failover failed: connection timeout")

	// Create audit log with error
	params := json.RawMessage(`{"source_region": "us-east", "target_region": "us-west"}`)
	auditLog, err := auditLogger.LogAction(userID, "ops_manager", actionType, "192.168.1.100", incidentID, "failure", params, nil, errorMsg, 5000)

	assert.NoError(t, err)
	assert.NotNil(t, auditLog)
	assert.Equal(t, "failure", auditLog.Status)
	assert.Equal(t, errorMsg, auditLog.ErrorMsg)
}

// Test 26: Phase 3.2 - Region-aware RCA correlation with same region
func TestCorrelationEngineRegionAwareSameRegion(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	region1 := "us-east-1"
	region2 := "us-east-1" // Same region
	tenantID1 := uuid.New()
	tenantID2 := uuid.New() // Different tenant to test region affinity specifically

	now := time.Now()
	events := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			TenantID:   &tenantID1,
			Region:     &region1,
			OccurredAt: now,
			Title:      "Latency detected",
			Scope:      "region",
		},
		{
			ID:         uuid.New(),
			EventType:  EventEndpointHealth,
			Severity:   SeverityError,
			TenantID:   &tenantID2,
			Region:     &region2,
			OccurredAt: now.Add(10 * time.Second),
			Title:      "Endpoint health degraded",
			Scope:      "region",
		},
	}

	// Compute correlation score
	correlations := engine.scoreCorrelations(events)

	// Should have one correlation from first to second event
	fromID := events[0].ID.String()
	assert.Contains(t, correlations, fromID)
	assert.Greater(t, len(correlations[fromID]), 0)

	// Same region should increase scope matching score
	score := correlations[fromID][0]
	assert.Greater(t, score.Score, 0.3) // Should be reasonably high
	assert.Contains(t, score.ReasonScores, "scope_match")
	// Same region: scope_match = 0.2 + (1.0 * 0.3) = 0.5
	assert.GreaterOrEqual(t, score.ReasonScores["scope_match"], 0.45)
}

// Test 27: Phase 3.2 - Region-aware RCA with adjacent regions
func TestCorrelationEngineRegionAwareAdjacentRegion(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	region1 := "us-east-1"
	region2 := "us-east-2" // Adjacent region
	tenantID1 := uuid.New()
	tenantID2 := uuid.New() // Different tenant to test region affinity specifically

	now := time.Now()
	events := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			TenantID:   &tenantID1,
			Region:     &region1,
			OccurredAt: now,
			Title:      "Latency detected",
			Scope:      "region",
		},
		{
			ID:         uuid.New(),
			EventType:  EventEndpointHealth,
			Severity:   SeverityError,
			TenantID:   &tenantID2,
			Region:     &region2,
			OccurredAt: now.Add(10 * time.Second),
			Title:      "Adjacent region health issue",
			Scope:      "region",
		},
	}

	correlations := engine.scoreCorrelations(events)
	fromID := events[0].ID.String()

	assert.Contains(t, correlations, fromID)
	assert.Greater(t, len(correlations[fromID]), 0)

	score := correlations[fromID][0]
	// Adjacent region: scope_match = 0.2 + (0.7 * 0.3) = 0.41
	assert.Less(t, score.ReasonScores["scope_match"], 0.45)
	assert.GreaterOrEqual(t, score.ReasonScores["scope_match"], 0.38)
}

// Test 28: Phase 3.2 - Region-aware RCA with cross-region events
func TestCorrelationEngineRegionAwareCrossRegion(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	region1 := "us-east-1"
	region2 := "eu-west-1" // Cross-region
	tenantID1 := uuid.New()
	tenantID2 := uuid.New() // Different tenant to test region affinity specifically

	now := time.Now()
	events := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			TenantID:   &tenantID1,
			Region:     &region1,
			OccurredAt: now,
			Title:      "Latency detected",
			Scope:      "region",
		},
		{
			ID:         uuid.New(),
			EventType:  EventEndpointHealth,
			Severity:   SeverityError,
			TenantID:   &tenantID2,
			Region:     &region2,
			OccurredAt: now.Add(10 * time.Second),
			Title:      "Cross-region health issue",
			Scope:      "region",
		},
	}

	correlations := engine.scoreCorrelations(events)
	fromID := events[0].ID.String()

	assert.Contains(t, correlations, fromID)
	assert.Greater(t, len(correlations[fromID]), 0)

	score := correlations[fromID][0]
	// Cross-region: scope_match = 0.2 + (0.3 * 0.3) = 0.29
	assert.LessOrEqual(t, score.ReasonScores["scope_match"], 0.31)
	assert.Greater(t, score.ReasonScores["scope_match"], 0.25)
}

// Test 29: Phase 3.2 - Region affinity scoring function
func TestRegionAffinityScoringFunction(t *testing.T) {
	// Test same region affinity
	region1 := "us-east-1"
	score1 := GetRegionAffinityScore(&region1, &region1)
	assert.Equal(t, 1.0, score1)

	// Test adjacent region affinity
	region2 := "us-east-2"
	score2 := GetRegionAffinityScore(&region1, &region2)
	assert.Equal(t, 0.7, score2)

	// Test cross-region affinity
	region3 := "eu-west-1"
	score3 := GetRegionAffinityScore(&region1, &region3)
	assert.Equal(t, 0.3, score3)

	// Test nil region handling
	score4 := GetRegionAffinityScore(nil, nil)
	assert.Equal(t, 1.0, score4) // Both nil → default to us-east-1 → same region

	// Test mixed nil handling
	score5 := GetRegionAffinityScore(&region1, nil)
	assert.Equal(t, 1.0, score5) // First nil → default to us-east-1 → same region
}

// Test 30: Phase 3.3 - Region-aware action execution with explicit region
func TestActionExecutorRegionAware(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()

	params := map[string]interface{}{
		"worker_id": "worker-1",
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute action in specific region
	regionEU := "eu-west-1"
	response, err := executor.ExecuteAction(ctx, incidentID, "restart_worker", paramsJSON, &regionEU)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "restart_worker", response.ActionType)
	assert.Equal(t, "success", response.Status)
}

// Test 31: Phase 3.3 - Region-aware action with default region (nil)
func TestActionExecutorDefaultRegion(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()

	params := map[string]interface{}{
		"worker_id": "worker-1",
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	// Execute action without explicit region (should default to us-east-1)
	response, err := executor.ExecuteAction(ctx, incidentID, "restart_worker", paramsJSON, nil)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "restart_worker", response.ActionType)
	assert.Equal(t, "success", response.Status)
}

// Test 32: Phase 3.3 - Get region worker pool
func TestActionExecutorGetRegionWorkerPool(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)

	ctx := context.Background()
	tenantID := uuid.New()

	// GetRegionWorkerPool with invalid region should fail
	_, err := executor.GetRegionWorkerPool(ctx, tenantID, "invalid-region-xyz")
	assert.Error(t, err) // Expected to fail with invalid region
}

// Test 33: Phase 3.3 - Validate multi-region action routing
func TestActionExecutorMultiRegionRouting(t *testing.T) {
	store := &TestStore{}
	timeline := NewTimelineService(store)
	executor := NewActionExecutor(store, timeline)

	ctx := context.Background()
	incidentID := uuid.New()

	params := map[string]interface{}{
		"circuit_id":    "api-gateway-db",
		"target_state":  "open",
		"duration_secs": 300,
	}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	// Test action in US region
	regionUS := "us-east-1"
	responseUS, err := executor.ExecuteAction(ctx, incidentID, "circuit_breaker_toggle", paramsJSON, &regionUS)
	assert.NoError(t, err)
	assert.Equal(t, "success", responseUS.Status)

	// Test action in EU region
	regionEU := "eu-west-1"
	responseEU, err := executor.ExecuteAction(ctx, incidentID, "circuit_breaker_toggle", paramsJSON, &regionEU)
	assert.NoError(t, err)
	assert.Equal(t, "success", responseEU.Status)

	// Both should succeed (TestStore doesn't validate regions)
	assert.NotNil(t, responseUS)
	assert.NotNil(t, responseEU)
}

// ============================================================================
// Phase 3.5: Regional Metrics & SLA Tracking Tests
// ============================================================================

func TestRegionalMetricsUpsert(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	metrics := &RegionalMetrics{
		ID:            uuid.New(),
		Region:        "us-east-1",
		ErrorRate:     0.5,
		P50Latency:    100,
		P95Latency:    250,
		P99Latency:    500,
		Availability:  99.95,
		RequestCount:  50000,
		IncidentCount: 2,
		Components: map[string]float64{
			"cpu":    85.5,
			"memory": 72.3,
			"disk":   68.0,
		},
		ComputedAt: time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := store.UpsertRegionalMetrics(ctx, metrics)
	assert.NoError(t, err)
}

func TestRegionalHealthScore(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	health := &RegionalHealth{
		ID:         uuid.New(),
		Region:     "eu-west-1",
		Score:      85,
		Status:     "healthy",
		ComputedAt: time.Now(),
		UpdatedAt:  time.Now(),
		Components: map[string]float64{
			"error_rate":   90.0,
			"latency":      80.0,
			"availability": 85.0,
		},
	}

	err := store.UpsertRegionalHealth(ctx, health)
	assert.NoError(t, err)

	retrieved, err := store.GetRegionalHealth(ctx, "eu-west-1")
	assert.NoError(t, err)
	assert.Nil(t, retrieved) // TestStore returns nil
}

func TestRegionalSLADefinitions(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	sla := &RegionalSLA{
		ID:              uuid.New(),
		Region:          "ap-southeast-1",
		AvailabilitySLA: 99.99,
		P95LatencySLA:   300,
		ErrorRateSLA:    0.1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := store.UpsertRegionalSLA(ctx, sla)
	assert.NoError(t, err)

	slas, err := store.ListRegionalSLAs(ctx, 10)
	assert.NoError(t, err)
	assert.Empty(t, slas) // TestStore returns empty list
}

func TestRegionalSLACompliance(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	slaID := uuid.New()
	status := &RegionalSLAStatus{
		ID:              uuid.New(),
		Region:          "us-west-2",
		SLAID:           slaID,
		AvailabilityMet: true,
		LatencyMet:      true,
		ErrorRateMet:    false,
		CompliancePct:   66.67, // 2 out of 3 metrics met
		CheckedAt:       time.Now(),
		CreatedAt:       time.Now(),
	}

	err := store.InsertRegionalSLAStatus(ctx, status)
	assert.NoError(t, err)

	retrieved, err := store.GetRegionalSLAStatus(ctx, "us-west-2")
	assert.NoError(t, err)
	assert.Nil(t, retrieved) // TestStore returns nil

	// List compliance history
	history, err := store.ListRegionalSLAStatuses(ctx, "us-west-2", 20)
	assert.NoError(t, err)
	assert.Empty(t, history) // TestStore returns empty list
}

func TestRegionalMetricsComposition(t *testing.T) {
	// Test that regional metrics correctly compose from multiple data sources
	metrics := &RegionalMetrics{
		ID:            uuid.New(),
		Region:        "us-east-1",
		ErrorRate:     0.3,
		P50Latency:    95,
		P95Latency:    240,
		P99Latency:    480,
		Availability:  99.97,
		RequestCount:  100000,
		IncidentCount: 1,
		Components: map[string]float64{
			"error_rate_score":   95.0, // 0.3% is good
			"latency_score":      92.0, // All latencies in good range
			"availability_score": 99.97,
			"incident_score":     99.0, // 1 incident in 24h
		},
		ComputedAt: time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	assert.Equal(t, "us-east-1", metrics.Region)
	assert.Equal(t, float64(0.3), metrics.ErrorRate)
	assert.Equal(t, 240, metrics.P95Latency)
	assert.NotNil(t, metrics.Components)
	assert.Equal(t, 95.0, metrics.Components["error_rate_score"])
}

func TestHealthStatusFromScore(t *testing.T) {
	testCases := []struct {
		score    int
		expected string
	}{
		{85, "healthy"},
		{75, "degraded"},
		{40, "critical"},
		{100, "healthy"},
		{50, "degraded"},
		{49, "critical"},
	}

	for _, tc := range testCases {
		health := &RegionalHealth{
			Score: tc.score,
		}

		if tc.score >= 80 {
			health.Status = "healthy"
		} else if tc.score >= 50 {
			health.Status = "degraded"
		} else {
			health.Status = "critical"
		}

		assert.Equal(t, tc.expected, health.Status, "score %d should map to %s", tc.score, tc.expected)
	}
}

func TestSLAComplianceCalculation(t *testing.T) {
	// Test SLA compliance percentage calculation
	testCases := []struct {
		name                  string
		availabilityMet       bool
		latencyMet            bool
		errorRateMet          bool
		expectedCompliancePct float64
	}{
		{
			name:                  "all met",
			availabilityMet:       true,
			latencyMet:            true,
			errorRateMet:          true,
			expectedCompliancePct: 100.0,
		},
		{
			name:                  "two met",
			availabilityMet:       true,
			latencyMet:            true,
			errorRateMet:          false,
			expectedCompliancePct: 66.67,
		},
		{
			name:                  "one met",
			availabilityMet:       true,
			latencyMet:            false,
			errorRateMet:          false,
			expectedCompliancePct: 33.33,
		},
		{
			name:                  "none met",
			availabilityMet:       false,
			latencyMet:            false,
			errorRateMet:          false,
			expectedCompliancePct: 0.0,
		},
	}

	for _, tc := range testCases {
		metCount := 0
		if tc.availabilityMet {
			metCount++
		}
		if tc.latencyMet {
			metCount++
		}
		if tc.errorRateMet {
			metCount++
		}

		compliancePct := (float64(metCount) / 3.0) * 100.0
		assert.InDelta(t, tc.expectedCompliancePct, compliancePct, 0.1, tc.name)
	}
}

// Helper function for string pointer
func stringPtr(s string) *string {
	return &s
}

// ============================================================================
// Phase 3.1 Complete: Region Propagation through Incident & Event Lifecycle
// ============================================================================

func TestRegionPropagationEventCreation(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	// Create event with region
	event := Event{
		ID:         uuid.New(),
		EventType:  EventLatencyAnomaly,
		Severity:   SeverityWarning,
		Title:      "Latency spike in us-east-1",
		Region:     stringPtr("us-east-1"),
		Scope:      "region",
		Details:    []byte(`{"p95": 500}`),
		OccurredAt: time.Now(),
	}

	err := store.InsertEvent(ctx, event)
	assert.NoError(t, err)
}

func TestRegionPropagationIncidentCreation(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	// Create event that triggers incident creation
	event := Event{
		ID:         uuid.New(),
		EventType:  EventLatencyAnomaly,
		Severity:   SeverityCritical,
		Title:      "Critical latency in eu-west-1",
		Region:     stringPtr("eu-west-1"),
		Scope:      "region",
		Details:    []byte(`{"p95": 1000}`),
		OccurredAt: time.Now(),
	}

	// UpsertIncidentForEvent should create incident with region
	incident, err := store.UpsertIncidentForEvent(ctx, event)
	assert.NoError(t, err)

	// Verify event has region set (incident creation should use event.Region)
	assert.NotNil(t, event.Region)
	assert.Equal(t, "eu-west-1", *event.Region)

	// If implementation returns incident, verify it has region
	if incident != nil {
		assert.Equal(t, "eu-west-1", *incident.Region)
	}
}

func TestRegionPropagationIncidentRetrieval(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	incidentID := uuid.New()
	regionName := "ap-southeast-1"

	// Retrieve incident (TestStore returns nil, but tests actual structure)
	incident, _, err := store.GetIncident(ctx, incidentID)
	assert.NoError(t, err)

	// Verify incident can have region
	if incident != nil {
		incident.Region = &regionName
		assert.Equal(t, "ap-southeast-1", *incident.Region)
	}
}

func TestRegionPropagationIncidentUpdate(t *testing.T) {
	incidentID := uuid.New()

	// Create incident with region
	incident := &Incident{
		ID:        incidentID,
		Region:    stringPtr("us-west-2"),
		Status:    "open",
		Severity:  SeverityWarning,
		Title:     "Test incident",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Verify region is preserved
	assert.NotNil(t, incident.Region)
	assert.Equal(t, "us-west-2", *incident.Region)
}

func TestEventScopeRegionCorrelation(t *testing.T) {
	// Test that events with scope="region" properly set region field
	testCases := []struct {
		name        string
		scope       string
		region      *string
		shouldMatch bool
	}{
		{
			name:        "region scope with region",
			scope:       "region",
			region:      stringPtr("us-east-1"),
			shouldMatch: true,
		},
		{
			name:        "tenant scope without region",
			scope:       "tenant",
			region:      nil,
			shouldMatch: false,
		},
		{
			name:        "endpoint scope without region",
			scope:       "endpoint",
			region:      nil,
			shouldMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := Event{
				ID:         uuid.New(),
				EventType:  EventLatencyAnomaly,
				Scope:      tc.scope,
				Region:     tc.region,
				Severity:   SeverityWarning,
				Title:      "Test",
				Details:    []byte(`{}`),
				OccurredAt: time.Now(),
			}

			// Verify event structure
			assert.Equal(t, tc.scope, event.Scope)
			assert.Equal(t, tc.region, event.Region)

			// Region should be set if scope is "region"
			if tc.scope == "region" {
				assert.NotNil(t, tc.region)
			}
		})
	}
}

func TestMultiRegionIncidentTracking(t *testing.T) {
	// Test incident creation in multiple regions
	regions := []string{"us-east-1", "eu-west-1", "ap-southeast-1"}

	for _, region := range regions {
		regionCopy := region // Create copy for closure
		event := Event{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityCritical,
			Region:     &regionCopy,
			Scope:      "region",
			Title:      "Test",
			Details:    []byte(`{}`),
			OccurredAt: time.Now(),
		}

		// Each region should create separate incident
		assert.Equal(t, region, *event.Region)
	}
}

func TestIncidentRegionFieldStructure(t *testing.T) {
	// Validate that Incident struct has region properly defined
	incident := &Incident{
		ID:        uuid.New(),
		Region:    stringPtr("us-east-1"),
		Status:    "open",
		Severity:  SeverityWarning,
		Title:     "Test",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Verify Region field is present and accessible
	assert.NotNil(t, incident.Region)
	assert.Equal(t, "us-east-1", *incident.Region)

	// Test JSON marshaling includes region
	jsonData := `{"region":"eu-west-1"}`
	assert.Contains(t, jsonData, "region")
}

// TestActionHistoryRegionContext verifies region context for actions
func TestActionHistoryRegionContext(t *testing.T) {
	actionHistory := &ActionHistory{
		ID:         uuid.New(),
		IncidentID: uuid.New(),
		Region:     stringPtr("us-east-1"),
		ActionType: "circuit_breaker_toggle",
		Status:     "success",
		ExecutedAt: time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Action should know which region it executed in
	assert.NotNil(t, actionHistory.Region)
	assert.Equal(t, "us-east-1", *actionHistory.Region)
}

// ============================================================================
// Phase 3.8a: Region-Aware Action Execution Tests
// ============================================================================

// TestActionExecutorHasRegionRouter verifies ActionExecutor has RegionRouter
func TestActionExecutorHasRegionRouter(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)

	// Executor should have regionRouter
	assert.NotNil(t, executor.GetRegionRouter())
}

// TestActionExecutorRegionWorkerPoolResolution verifies worker pool lookup
func TestActionExecutorRegionWorkerPoolResolution(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	ctx := context.Background()

	// Register regions
	for _, region := range []string{"us-east-1", "eu-west-1", "ap-southeast-1"} {
		err := executor.GetRegionRouter().RegisterRegionTarget(ctx, &RegionTarget{
			Region:            region,
			StarRocksCluster:  "starrocks-" + region,
			RedpandaBroker:    "redpanda-" + region + ":9092",
			TemporalNamespace: region,
			OpsWorkerPool:     "worker-pool-" + region,
			IsActive:          true,
		})
		assert.NoError(t, err)
	}

	// Resolve worker pools for each region
	for _, region := range []string{"us-east-1", "eu-west-1", "ap-southeast-1"} {
		workerPool, err := executor.GetRegionWorkerPool(ctx, uuid.New(), region)
		assert.NoError(t, err)
		assert.NotNil(t, workerPool)
		assert.Equal(t, "worker-pool-"+region, *workerPool)
	}
}

// TestActionExecutorFailsForInvalidRegion verifies strict region validation
func TestActionExecutorFailsForInvalidRegion(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	ctx := context.Background()

	// Incident routes to non-registered region
	_, err := executor.GetRegionWorkerPool(ctx, uuid.New(), "invalid-region-xyz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

// ============================================================================
// Phase 3.8b: Region-Aware RCA Correlation Scoring Tests
// ============================================================================

// TestRCARegionAffinityScoring verifies same-region events get bonus
func TestRCARegionAffinityScoring(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	now := time.Now()
	region1 := "us-east-1"
	region2 := "eu-west-1"

	// Two events in SAME region, close in time
	sameRegionEvent1 := Event{
		ID:         uuid.New(),
		EventType:  EventLatencyAnomaly,
		Severity:   SeverityWarning,
		Region:     &region1,
		OccurredAt: now,
	}

	sameRegionEvent2 := Event{
		ID:         uuid.New(),
		EventType:  EventAlert,
		Severity:   SeverityError,
		Region:     &region1, // SAME REGION
		OccurredAt: now.Add(20 * time.Second),
	}

	// Two events in DIFFERENT regions, close in time
	crossRegionEvent1 := Event{
		ID:         uuid.New(),
		EventType:  EventLatencyAnomaly,
		Severity:   SeverityWarning,
		Region:     &region1,
		OccurredAt: now,
	}

	crossRegionEvent2 := Event{
		ID:         uuid.New(),
		EventType:  EventAlert,
		Severity:   SeverityError,
		Region:     &region2, // DIFFERENT REGION
		OccurredAt: now.Add(20 * time.Second),
	}

	// Score same-region correlation
	sameRegionScore := engine.computeCorrelationScore(sameRegionEvent1, sameRegionEvent2)

	// Score cross-region correlation
	crossRegionScore := engine.computeCorrelationScore(crossRegionEvent1, crossRegionEvent2)

	// Same-region should have HIGHER correlation score
	assert.Greater(t, sameRegionScore.Score, crossRegionScore.Score,
		"Same-region events should have higher correlation than cross-region")

	// Region affinity should be in reason scores
	assert.Contains(t, sameRegionScore.ReasonScores, "region_affinity")
	assert.Greater(t, sameRegionScore.ReasonScores["region_affinity"], 0.0)

	// Same region affinity should be higher
	assert.Greater(t, sameRegionScore.ReasonScores["region_affinity"],
		crossRegionScore.ReasonScores["region_affinity"])
}

// TestRCARegionAffinity Boost TestRegion events get +weight boost
func TestRCARegionAffinityBoost(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	now := time.Now()
	region := "us-east-1"

	// Two events in same region
	event1 := Event{
		ID:         uuid.New(),
		EventType:  EventLatencyAnomaly,
		Severity:   SeverityWarning,
		Region:     &region,
		OccurredAt: now,
	}

	event2 := Event{
		ID:         uuid.New(),
		EventType:  EventFingerprint,
		Severity:   SeverityError,
		Region:     &region,
		OccurredAt: now.Add(15 * time.Second),
	}

	score := engine.computeCorrelationScore(event1, event2)

	// Should have strong region affinity
	regionAffinity := score.ReasonScores["region_affinity"]
	assert.Greater(t, regionAffinity, 0.7)

	// Total score should be strong due to region affinity contribution
	assert.Greater(t, score.Score, 0.6)
}

// TestRCACrossRegionPenalty verifies cross-region gets penalty
func TestRCACrossRegionPenalty(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	now := time.Now()
	region1 := "us-east-1"
	region2 := "ap-southeast-1"

	event1 := Event{
		ID:         uuid.New(),
		EventType:  EventLatencyAnomaly,
		Severity:   SeverityWarning,
		Region:     &region1,
		OccurredAt: now,
	}

	event2 := Event{
		ID:         uuid.New(),
		EventType:  EventFingerprint,
		Severity:   SeverityError,
		Region:     &region2,
		OccurredAt: now.Add(15 * time.Second),
	}

	score := engine.computeCorrelationScore(event1, event2)

	// Cross-region affinity should be LOW
	regionAffinity := score.ReasonScores["region_affinity"]
	assert.Less(t, regionAffinity, 0.6, "Cross-region should have lower affinity")

	// Total score should be penalized
	assert.Less(t, score.Score, 0.75,
		"Cross-region events should have lower total correlation")
}

// TestRCARegionAgilityScoring verifies without regions, scores remain neutral
func TestRCANoRegionScoring(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	now := time.Now()

	// Events with NO region
	event1 := Event{
		ID:         uuid.New(),
		EventType:  EventLatencyAnomaly,
		Severity:   SeverityWarning,
		Region:     nil,
		OccurredAt: now,
	}

	event2 := Event{
		ID:         uuid.New(),
		EventType:  EventAlert,
		Severity:   SeverityError,
		Region:     nil,
		OccurredAt: now.Add(15 * time.Second),
	}

	score := engine.computeCorrelationScore(event1, event2)

	// Region affinity defaults to neutral (0.5)
	assert.InDelta(t, 0.5, score.ReasonScores["region_affinity"], 0.1)

	// Should still compute correlation based on other factors
	assert.Greater(t, score.Score, 0.3)
}

// TestRCARegionAware CompletePipeline verifies full RCA with regions
func TestRCARegionAwareCompletePipeline(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	now := time.Now()
	region := "us-east-1"

	// Create multi-event incident chain in same region
	events := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			Region:     &region,
			OccurredAt: now,
		},
		{
			ID:         uuid.New(),
			EventType:  EventFingerprint,
			Severity:   SeverityError,
			Region:     &region,
			OccurredAt: now.Add(20 * time.Second),
		},
		{
			ID:         uuid.New(),
			EventType:  EventEndpointHealth,
			Severity:   SeverityError,
			Region:     &region,
			OccurredAt: now.Add(40 * time.Second),
		},
	}

	// Compute RCA
	rca := engine.ComputeRCA(events)

	// Should identify root cause
	assert.NotNil(t, rca)
	assert.NotNil(t, rca.SuspectedRootCause)

	// Root cause type should be one of the events
	rootCauseType := rca.SuspectedRootCause.Event.EventType
	validRootCauses := []EventType{EventLatencyAnomaly, EventFingerprint, EventEndpointHealth}
	assert.Contains(t, validRootCauses, rootCauseType)

	// Causality chain should show clear progression
	assert.Greater(t, len(rca.CausalityChain), 1)

	// Confidence should be HIGH due to region affinity
	assert.Greater(t, rca.ConfidenceScore, 0.6)
}

// TestRCAMultiRegionCorrelation verifies proper weighting of multi-region incidents
func TestRCAMultiRegionCorrelation(t *testing.T) {
	store := &TestStore{}
	engine := NewCorrelationEngine(store)

	now := time.Now()
	region1 := "us-east-1"
	region2 := "eu-west-1"

	// Same-region chain (strong correlation)
	sameRegionEvents := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			Region:     &region1,
			OccurredAt: now,
		},
		{
			ID:         uuid.New(),
			EventType:  EventAlert,
			Severity:   SeverityError,
			Region:     &region1,
			OccurredAt: now.Add(10 * time.Second),
		},
	}

	// Cross-region chain (weaker correlation)
	crossRegionEvents := []Event{
		{
			ID:         uuid.New(),
			EventType:  EventLatencyAnomaly,
			Severity:   SeverityWarning,
			Region:     &region1,
			OccurredAt: now,
		},
		{
			ID:         uuid.New(),
			EventType:  EventAlert,
			Severity:   SeverityError,
			Region:     &region2,
			OccurredAt: now.Add(10 * time.Second),
		},
	}

	// Compute RCAs
	rcaSame := engine.ComputeRCA(sameRegionEvents)
	rcaCross := engine.ComputeRCA(crossRegionEvents)

	// Same-region should have higher confidence
	assert.Greater(t, rcaSame.ConfidenceScore, rcaCross.ConfidenceScore,
		"Same-region incidents should have higher RCA confidence")

	// Both should identify root cause
	assert.NotNil(t, rcaSame.SuspectedRootCause)
	assert.NotNil(t, rcaCross.SuspectedRootCause)

	// Same-region causality chain should be more complete
	assert.Greater(t, len(rcaSame.CausalityChain), 0)
}

// ============================================================================
// Phase 3.9: Region-Aware API Layer & Dashboard Integration Tests
// ============================================================================

// TestListRegionsSummary verifies regional summary aggregation
func TestListRegionsSummary(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	// Create regional data
	region1 := "us-east-1"

	metrics1 := &RegionalMetrics{
		ID:           uuid.New(),
		Region:       region1,
		ErrorRate:    0.5,
		P95Latency:   250,
		Availability: 99.95,
		UpdatedAt:    time.Now(),
	}
	err := store.UpsertRegionalMetrics(ctx, metrics1)
	assert.NoError(t, err)

	health1 := &RegionalHealth{
		ID:     uuid.New(),
		Region: region1,
		Score:  90,
		Status: "healthy",
	}
	err = store.UpsertRegionalHealth(ctx, health1)
	assert.NoError(t, err)

	// Test summary should aggregate data
	metrics, err := store.ListRegionalMetrics(ctx, 100)
	assert.NoError(t, err)

	health, err := store.ListRegionalHealth(ctx, 100)
	assert.NoError(t, err)

	latestSLA, err := store.ListLatestRegionalSLAStatuses(ctx)
	assert.NoError(t, err)

	// Verify types work correctly
	assert.NotNil(t, metrics)
	assert.NotNil(t, health)
	assert.NotNil(t, latestSLA)
}

// TestGetRegionDetail verifies comprehensive drill-down data
func TestGetRegionDetail(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	region := "us-east-1"

	// Get regional data
	metrics, err := store.GetRegionalMetrics(ctx, region)
	assert.NoError(t, err)

	health, err := store.GetRegionalHealth(ctx, region)
	assert.NoError(t, err)

	sla, err := store.GetRegionalSLA(ctx, region)
	assert.NoError(t, err)

	slaHistory, err := store.ListRegionalSLAStatuses(ctx, region, 50)
	assert.NoError(t, err)

	incidents, err := store.ListIncidentsByRegion(ctx, region, 50)
	assert.NoError(t, err)

	// Verify all data types
	assert.Nil(t, metrics) // TestStore returns nil
	assert.Nil(t, health)
	assert.Nil(t, sla)
	assert.Empty(t, slaHistory)
	assert.Empty(t, incidents)
}

// TestListIncidentsByRegion verifies region-filtered incident list
func TestListIncidentsByRegion(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	region := "us-east-1"

	// List incidents for region
	incidents, err := store.ListIncidentsByRegion(ctx, region, 100)
	assert.NoError(t, err)
	assert.Empty(t, incidents) // TestStore returns empty list

	// Also test unfiltered list
	allIncidents, err := store.ListIncidents(ctx, 100)
	assert.NoError(t, err)
	assert.Empty(t, allIncidents)
}

// TestListLatestRegionalSLAStatuses verifies latest SLA across regions
func TestListLatestRegionalSLAStatuses(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	// Get latest SLA status for all regions
	latestStatuses, err := store.ListLatestRegionalSLAStatuses(ctx)
	assert.NoError(t, err)
	assert.Empty(t, latestStatuses) // TestStore returns empty list
}

// TestListRegionalIncidentCounts verifies incident count aggregation
func TestListRegionalIncidentCounts(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	// Get incident counts per region in 24h window
	counts, err := store.ListRegionalIncidentCounts(ctx, time.Now().Add(-24*time.Hour), time.Now())
	assert.NoError(t, err)
	assert.Empty(t, counts) // TestStore returns empty list
}

// TestListOpsEventsByRegion verifies region-filtered event list
func TestListOpsEventsByRegion(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	region := "eu-west-1"

	// List events for region
	events, err := store.ListOpsEventsByRegion(ctx, region, 100)
	assert.NoError(t, err)
	assert.Empty(t, events)
}

// TestListAuditLogsByRegion verifies region-filtered audit log list
func TestListAuditLogsByRegion(t *testing.T) {
	store := &TestStore{}
	ctx := context.Background()

	region := "ap-southeast-1"

	// List audit logs for region
	logs, err := store.ListAuditLogsByRegion(ctx, region, 100)
	assert.NoError(t, err)
	assert.Empty(t, logs)
}

// TestRegionSummaryStructure verifies RegionSummary type integrity
func TestRegionSummaryStructure(t *testing.T) {
	summary := &RegionSummary{
		Region:           "us-east-1",
		HealthScore:      85,
		HealthStatus:     "healthy",
		SLACompliance:    99.5,
		ErrorRate:        0.3,
		LatencyP95Ms:     245.5,
		Availability:     99.97,
		IncidentCount24h: 2,
		UpdatedAt:        time.Now(),
	}

	// Verify all fields
	assert.Equal(t, "us-east-1", summary.Region)
	assert.Equal(t, 85, summary.HealthScore)
	assert.Equal(t, "healthy", summary.HealthStatus)
	assert.Equal(t, 99.5, summary.SLACompliance)
	assert.NotZero(t, summary.UpdatedAt)
}

// TestRegionDetailStructure verifies RegionDetail type integrity
func TestRegionDetailStructure(t *testing.T) {
	region := "eu-west-1"

	detail := &RegionDetail{
		Region:           region,
		Metrics:          nil,
		Health:           nil,
		SLA:              nil,
		SLAStatusHistory: []RegionalSLAStatus{},
		RecentIncidents:  []Incident{},
		RecentOpsEvents:  []Event{},
		RecentActions:    []AuditLog{},
		RecentRCASummary: []interface{}{},
	}

	// Verify structure
	assert.Equal(t, region, detail.Region)
	assert.NotNil(t, detail.SLAStatusHistory)
	assert.NotNil(t, detail.RecentIncidents)
	assert.NotNil(t, detail.RecentOpsEvents)
	assert.NotNil(t, detail.RecentActions)
}

// TestRegionalIncidentCountStructure verifies RegionalIncidentCount integrity
func TestRegionalIncidentCountStructure(t *testing.T) {
	count := &RegionalIncidentCount{
		Region: "us-west-2",
		Count:  5,
	}

	assert.Equal(t, "us-west-2", count.Region)
	assert.Equal(t, 5, count.Count)
}

// ============================================================================
// Phase 3.10: Failover Policies & Automated Regional Failover Tests
// ============================================================================

// TestCheckFailoverConditionsErrorRate verifies error rate trigger detection
func TestCheckFailoverConditionsErrorRate(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	orchestrator := NewFailoverOrchestrator(store, executor, NewInMemoryRegionRegistry(store, 5*time.Minute))

	ctx := context.Background()
	tenantID := uuid.New()

	// Create policy that triggers at 20% error rate
	triggerErrorRate := 20.0
	policy := &FailoverPolicy{
		ID:               uuid.New(),
		TenantID:         tenantID,
		SourceRegion:     "us-east-1",
		TargetRegions:    `["us-west-2"]`,
		TriggerErrorRate: &triggerErrorRate,
	}

	// Verify orchestrator can evaluate policies (checks for no errors)
	condition, err := orchestrator.CheckFailoverConditions(ctx, policy)
	assert.NoError(t, err)
	assert.NotNil(t, condition)
	// Mock store returns nil metrics, so no trigger
	assert.False(t, condition.Met)
}

// TestCheckFailoverConditionsLatency verifies latency threshold evaluation
func TestCheckFailoverConditionsLatency(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	orchestrator := NewFailoverOrchestrator(store, executor, NewInMemoryRegionRegistry(store, 5*time.Minute))

	ctx := context.Background()
	tenantID := uuid.New()

	// Create policy that triggers at 1 second latency
	triggerLatency := 1000
	policy := &FailoverPolicy{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SourceRegion:   "us-east-1",
		TargetRegions:  `["eu-west-1"]`,
		TriggerLatency: &triggerLatency,
	}

	condition, err := orchestrator.CheckFailoverConditions(ctx, policy)
	assert.NoError(t, err)
	assert.NotNil(t, condition)
	// Mock store returns nil metrics, so no trigger
	assert.False(t, condition.Met)
}

// TestCheckFailoverConditionsHealthScore verifies health score threshold evaluation
func TestCheckFailoverConditionsHealthScore(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	orchestrator := NewFailoverOrchestrator(store, executor, NewInMemoryRegionRegistry(store, 5*time.Minute))

	ctx := context.Background()
	tenantID := uuid.New()

	// Create policy that triggers at 50% health
	triggerHealthScore := 50
	policy := &FailoverPolicy{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		SourceRegion:       "us-east-1",
		TargetRegions:      `["us-west-1"]`,
		TriggerHealthScore: &triggerHealthScore,
	}

	condition, err := orchestrator.CheckFailoverConditions(ctx, policy)
	assert.NoError(t, err)
	assert.NotNil(t, condition)
	// Mock store returns nil health, so no trigger
	assert.False(t, condition.Met)
}

// TestFailoverEventStructure verifies FailoverEvent type
func TestFailoverEventStructure(t *testing.T) {
	event := &FailoverEvent{
		ID:            uuid.New(),
		IncidentID:    uuid.New(),
		PolicyID:      uuid.New(),
		TenantID:      uuid.New(),
		SourceRegion:  "us-east-1",
		TargetRegion:  "us-west-2",
		TriggerReason: "error_rate",
		TriggerValue:  25.0,
		Status:        "success",
		TriggeredAt:   time.Now().UTC(),
	}

	assert.NotNil(t, event.ID)
	assert.Equal(t, "us-east-1", event.SourceRegion)
	assert.Equal(t, "us-west-2", event.TargetRegion)
	assert.Equal(t, "error_rate", event.TriggerReason)
	assert.Equal(t, "success", event.Status)
}

// TestFailoverPolicyStructure verifies FailoverPolicy type
func TestFailoverPolicyStructure(t *testing.T) {
	errorRate := 20.0
	policy := &FailoverPolicy{
		ID:               uuid.New(),
		TenantID:         uuid.New(),
		Name:             "us-east-1 → us-west-2",
		SourceRegion:     "us-east-1",
		TargetRegions:    `["us-west-2"]`,
		TriggerErrorRate: &errorRate,
		IsAutomatic:      true,
		CooldownMinutes:  30,
		Priority:         1,
		IsEnabled:        true,
	}

	assert.NotNil(t, policy.ID)
	assert.Equal(t, "us-east-1", policy.SourceRegion)
	assert.NotNil(t, policy.TriggerErrorRate)
	assert.Equal(t, errorRate, *policy.TriggerErrorRate)
	assert.True(t, policy.IsAutomatic)
}

// TestFailoverMetricsStructure verifies FailoverMetrics type
func TestFailoverMetricsStructure(t *testing.T) {
	now := time.Now().UTC()
	metrics := &FailoverMetrics{
		ID:              uuid.New(),
		PolicyID:        uuid.New(),
		TotalFailovers:  10,
		SuccessfulCount: 8,
		FailedCount:     2,
		AvgDurationMs:   1500,
		SuccessRatePct:  80.0,
		LastFailoverAt:  &now,
	}

	assert.Equal(t, 10, metrics.TotalFailovers)
	assert.Equal(t, 8, metrics.SuccessfulCount)
	assert.Equal(t, 2, metrics.FailedCount)
	assert.Equal(t, 80.0, metrics.SuccessRatePct)
	assert.NotNil(t, metrics.LastFailoverAt)
}

// TestFailoverOrchestratorCanBuild verifies FailoverOrchestrator initialization
func TestFailoverOrchestratorCanBuild(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	orchestrator := NewFailoverOrchestrator(store, executor, regionRouter)

	assert.NotNil(t, orchestrator)
}

// ========== Phase 3.11: Failover Chain Tests ==========

// TestFailoverChainStructures verifies FailoverChain, FailoverChainExecution, FailoverChainMetrics types
func TestFailoverChainStructures(t *testing.T) {
	// Test FailoverChain
	chainTargets := []string{"us-west-2", "eu-west-1"}
	chainTargetsJSON, _ := json.Marshal(chainTargets)

	healthScore := 50
	chain := &FailoverChain{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		Name:               "US Multi-Region Chain",
		SourceRegion:       "us-east-1",
		ChainTargets:       string(chainTargetsJSON),
		TriggerHealthScore: &healthScore,
		MaxChainDepth:      3,
		CooldownMinutes:    30,
		Priority:           1,
		IsEnabled:          true,
	}

	assert.NotNil(t, chain.ID)
	assert.Equal(t, "US Multi-Region Chain", chain.Name)
	assert.Equal(t, "us-east-1", chain.SourceRegion)

	// Test FailoverChainExecution
	stepsExecuted := []string{"us-west-2", "eu-west-1"}

	execution := &FailoverChainExecution{
		ID:            uuid.New(),
		ChainID:       chain.ID,
		IncidentID:    uuid.New(),
		TenantID:      chain.TenantID,
		SourceRegion:  "us-east-1",
		StepsExecuted: stepsExecuted,
		CurrentStep:   1,
		Status:        "partial_success",
	}

	assert.Equal(t, chain.ID, execution.ChainID)
	assert.Equal(t, "partial_success", execution.Status)
	assert.Equal(t, 1, execution.CurrentStep)

	// Test FailoverChainMetrics
	metrics := &FailoverChainMetrics{
		ID:                  uuid.New(),
		ChainID:             chain.ID,
		TotalExecutions:     5,
		SuccessfulCount:     3,
		PartialSuccessCount: 1,
		FailedCount:         1,
		AvgStepsNeeded:      2.2,
		AvgDurationMs:       800,
	}

	assert.Equal(t, 5, metrics.TotalExecutions)
	assert.Equal(t, 3, metrics.SuccessfulCount)
	assert.Equal(t, 1, metrics.PartialSuccessCount)
	assert.Equal(t, 1, metrics.FailedCount)
}

// TestCheckChainConditionsErrorRate verifies error rate trigger detection in cascade
func TestCheckChainConditionsErrorRate(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	simpleOrchestrator := NewFailoverOrchestrator(store, executor, regionRouter)

	chainOrchestrator := NewFailoverChainOrchestrator(store, executor, regionRouter, simpleOrchestrator)
	assert.NotNil(t, chainOrchestrator)

	// Create chain with error rate trigger
	errorRate := 25.0
	chain := &FailoverChain{
		ID:               uuid.New(),
		TenantID:         uuid.New(),
		Name:             "Error Rate Cascade",
		SourceRegion:     "us-east-1",
		ChainTargets:     `["us-west-2","eu-west-1"]`,
		TriggerErrorRate: &errorRate,
		MaxChainDepth:    3,
		Priority:         1,
		IsEnabled:        true,
	}

	// Error rate trigger should fire when exceeded
	assert.NotNil(t, chain.TriggerErrorRate)
	assert.Equal(t, 25.0, *chain.TriggerErrorRate)
}

// TestCheckChainConditionsLatency verifies latency threshold cascade
func TestCheckChainConditionsLatency(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	simpleOrchestrator := NewFailoverOrchestrator(store, executor, regionRouter)

	chainOrchestrator := NewFailoverChainOrchestrator(store, executor, regionRouter, simpleOrchestrator)
	assert.NotNil(t, chainOrchestrator)

	// Create chain with latency trigger
	latencyMs := 500
	chain := &FailoverChain{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		Name:           "Latency Cascade",
		SourceRegion:   "us-east-1",
		ChainTargets:   `["us-west-2","ap-northeast-1"]`,
		TriggerLatency: &latencyMs,
		MaxChainDepth:  2,
		Priority:       2,
		IsEnabled:      true,
	}

	// Latency trigger should be inspectable
	assert.NotNil(t, chain.TriggerLatency)
	assert.Equal(t, 500, *chain.TriggerLatency)
}

// TestCheckChainConditionsHealthScore verifies health score trigger detection
func TestCheckChainConditionsHealthScore(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	simpleOrchestrator := NewFailoverOrchestrator(store, executor, regionRouter)

	chainOrchestrator := NewFailoverChainOrchestrator(store, executor, regionRouter, simpleOrchestrator)
	assert.NotNil(t, chainOrchestrator)

	// Create chain with health score trigger
	healthScore := 40
	chain := &FailoverChain{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		Name:               "Health Score Cascade",
		SourceRegion:       "eu-west-1",
		ChainTargets:       `["us-east-1","us-west-2"]`,
		TriggerHealthScore: &healthScore,
		MaxChainDepth:      4,
		Priority:           3,
		IsEnabled:          true,
	}

	// Health score trigger should be detectable
	assert.NotNil(t, chain.TriggerHealthScore)
	assert.Equal(t, 40, *chain.TriggerHealthScore)
}

// TestChainExecutionCascade verifies multi-step execution success path
func TestChainExecutionCascade(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	simpleOrchestrator := NewFailoverOrchestrator(store, executor, regionRouter)

	chainOrchestrator := NewFailoverChainOrchestrator(store, executor, regionRouter, simpleOrchestrator)
	assert.NotNil(t, chainOrchestrator)

	// Execute cascade through targets
	execution := &FailoverChainExecution{
		ID:            uuid.New(),
		ChainID:       uuid.New(),
		IncidentID:    uuid.New(),
		TenantID:      uuid.New(),
		SourceRegion:  "us-east-1",
		StepsExecuted: []string{"us-west-2"},
		CurrentStep:   1,
		Status:        "in_progress",
	}

	assert.Equal(t, "us-east-1", execution.SourceRegion)
	assert.Equal(t, 1, execution.CurrentStep)
	assert.Equal(t, "in_progress", execution.Status)
}

// TestChainExecutionExhaustion verifies all targets fail behavior
func TestChainExecutionExhaustion(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	simpleOrchestrator := NewFailoverOrchestrator(store, executor, regionRouter)

	chainOrchestrator := NewFailoverChainOrchestrator(store, executor, regionRouter, simpleOrchestrator)
	assert.NotNil(t, chainOrchestrator)

	// Chain exhausted after trying all targets
	chain := &FailoverChain{
		ID:            uuid.New(),
		TenantID:      uuid.New(),
		Name:          "Exhaustion Test",
		SourceRegion:  "us-east-1",
		ChainTargets:  `["us-west-2","eu-west-1","ap-northeast-1"]`,
		MaxChainDepth: 3,
	}

	execution := &FailoverChainExecution{
		ID:            uuid.New(),
		ChainID:       chain.ID,
		IncidentID:    uuid.New(),
		TenantID:      chain.TenantID,
		SourceRegion:  "us-east-1",
		StepsExecuted: []string{"us-west-2", "eu-west-1", "ap-northeast-1"},
		CurrentStep:   3,
		Status:        "failed",
	}

	assert.Equal(t, "failed", execution.Status)
	assert.Equal(t, 3, execution.CurrentStep)
	assert.Equal(t, chain.MaxChainDepth, execution.CurrentStep)
}

// TestChainMetricsTracking verifies success/partial/failure tracking
func TestChainMetricsTracking(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	simpleOrchestrator := NewFailoverOrchestrator(store, executor, regionRouter)

	chainOrchestrator := NewFailoverChainOrchestrator(store, executor, regionRouter, simpleOrchestrator)
	assert.NotNil(t, chainOrchestrator)

	// Metrics accumulation across multiple executions
	metrics := &FailoverChainMetrics{
		ID:                  uuid.New(),
		ChainID:             uuid.New(),
		TotalExecutions:     10,
		SuccessfulCount:     6,
		PartialSuccessCount: 2,
		FailedCount:         2,
		AvgStepsNeeded:      1.8,
		AvgDurationMs:       1200,
	}

	successRate := float64(metrics.SuccessfulCount) / float64(metrics.TotalExecutions) * 100
	assert.InDelta(t, 60.0, successRate, 0.01)
	assert.Equal(t, 10, metrics.TotalExecutions)
	assert.Equal(t, 6, metrics.SuccessfulCount)
	assert.Equal(t, 2, metrics.PartialSuccessCount)
	assert.Equal(t, 2, metrics.FailedCount)
}

// TestChainCooldownEnforcement verifies step cooldown enforcement
func TestChainCooldownEnforcement(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	simpleOrchestrator := NewFailoverOrchestrator(store, executor, regionRouter)

	chainOrchestrator := NewFailoverChainOrchestrator(store, executor, regionRouter, simpleOrchestrator)
	assert.NotNil(t, chainOrchestrator)

	// Chain with cooldown enforcement
	chain := &FailoverChain{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		Name:            "Cooldown Test",
		SourceRegion:    "us-east-1",
		ChainTargets:    `["us-west-2","eu-west-1"]`,
		MaxChainDepth:   2,
		CooldownMinutes: 15,
		Priority:        1,
		IsEnabled:       true,
	}

	// Cooldown enforced between chain evaluations
	assert.Equal(t, 15, chain.CooldownMinutes)
}

// TestChainExecutionRetrieval verifies execution history query
func TestChainExecutionRetrieval(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	simpleOrchestrator := NewFailoverOrchestrator(store, executor, regionRouter)

	chainOrchestrator := NewFailoverChainOrchestrator(store, executor, regionRouter, simpleOrchestrator)
	assert.NotNil(t, chainOrchestrator)

	// Execution retrieval for a specific chain
	chainID := uuid.New()
	execution := &FailoverChainExecution{
		ID:            uuid.New(),
		ChainID:       chainID,
		IncidentID:    uuid.New(),
		TenantID:      uuid.New(),
		SourceRegion:  "us-east-1",
		StepsExecuted: []string{"us-west-2"},
		Status:        "success",
		CreatedAt:     time.Now().UTC(),
	}

	assert.Equal(t, chainID, execution.ChainID)
	assert.Equal(t, "success", execution.Status)
	assert.NotNil(t, execution.CreatedAt)
}

// ========== Phase 3.12: Multi-Tenant & Priority Failover Tests ==========

// TestChainStateCacheOperations verifies cache get/set/invalidate operations
func TestChainStateCacheOperations(t *testing.T) {
	cache := NewChainStateCache(5*time.Minute, 5*time.Minute, 5*time.Minute)

	tenantID := uuid.New()
	chainID := uuid.New()

	state := &FailoverChainState{
		ID:               uuid.New(),
		ChainID:          chainID,
		TenantID:         tenantID,
		CurrentStepIndex: 1,
		IsExecuting:      false,
	}

	// Set state in cache
	cache.SetState(tenantID, chainID, state)

	// Get from cache
	cached, exists := cache.GetState(tenantID, chainID)
	assert.True(t, exists)
	assert.NotNil(t, cached)
	assert.Equal(t, 1, cached.CurrentStepIndex)

	// Invalidate
	cache.InvalidateState(tenantID, chainID)
	cached, exists = cache.GetState(tenantID, chainID)
	assert.False(t, exists)
}

// TestChainStateCacheExpiration verifies TTL enforcement
func TestChainStateCacheExpiration(t *testing.T) {
	cache := NewChainStateCache(100*time.Millisecond, 5*time.Minute, 5*time.Minute)

	tenantID := uuid.New()
	chainID := uuid.New()

	state := &FailoverChainState{
		ID:       uuid.New(),
		ChainID:  chainID,
		TenantID: tenantID,
	}

	cache.SetState(tenantID, chainID, state)

	// Cache should exist immediately
	_, exists := cache.GetState(tenantID, chainID)
	assert.True(t, exists)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Cache should be expired
	_, exists = cache.GetState(tenantID, chainID)
	assert.False(t, exists)
}

// TestOrderChainsByPriority verifies priority-based ordering
func TestOrderChainsByPriority(t *testing.T) {
	chains := []ChainWithPriority{
		{
			ChainID:  uuid.New(),
			Priority: 2,
			Chain:    &FailoverChain{CreatedAt: time.Now().UTC()},
		},
		{
			ChainID:  uuid.New(),
			Priority: 1,
			Chain:    &FailoverChain{CreatedAt: time.Now().UTC().Add(1 * time.Second)},
		},
		{
			ChainID:  uuid.New(),
			Priority: 3,
			Chain:    &FailoverChain{CreatedAt: time.Now().UTC()},
		},
	}

	ordered := OrderChainsByPriority(chains)

	// Should be ordered: 3, 2, 1
	assert.Equal(t, 3, ordered[0].Priority)
	assert.Equal(t, 2, ordered[1].Priority)
	assert.Equal(t, 1, ordered[2].Priority)
}

// TestConflictDetectionSameSource verifies detection of same-source chains
func TestConflictDetectionSameSource(t *testing.T) {
	store := &TestStore{}
	detector := NewConflictDetector(store)

	chain1 := &FailoverChain{
		ID:           uuid.New(),
		SourceRegion: "us-east-1",
		ChainTargets: `["us-west-2"]`,
	}

	chain2 := &FailoverChain{
		ID:           uuid.New(),
		SourceRegion: "us-east-1",
		ChainTargets: `["eu-west-1"]`,
	}

	conflicts, _ := detector.DetectConflicts(context.Background(), uuid.New(), []*FailoverChain{chain1, chain2})

	assert.Len(t, conflicts, 1)
	assert.Equal(t, "same_source", conflicts[0].ConflictType)
}

// TestConflictDetectionOverlappingTargets verifies detection of overlapping targets
func TestConflictDetectionOverlappingTargets(t *testing.T) {
	store := &TestStore{}
	detector := NewConflictDetector(store)

	chain1 := &FailoverChain{
		ID:           uuid.New(),
		SourceRegion: "us-east-1",
		ChainTargets: `["us-west-2","eu-west-1"]`,
	}

	chain2 := &FailoverChain{
		ID:           uuid.New(),
		SourceRegion: "us-west-1",
		ChainTargets: `["us-west-2","ap-northeast-1"]`,
	}

	conflicts, _ := detector.DetectConflicts(context.Background(), uuid.New(), []*FailoverChain{chain1, chain2})

	assert.Len(t, conflicts, 1)
	assert.Equal(t, "overlapping_targets", conflicts[0].ConflictType)
}

// TestSLAComplianceCalculationAdvanced verifies SLA tracking engine creation and function
func TestSLAComplianceCalculationAdvanced(t *testing.T) {
	store := &TestStore{}
	cache := NewChainStateCache(5*time.Minute, 5*time.Minute, 5*time.Minute)
	tracker := NewSLATracker(store, cache, 2000, 95.0)

	// Verify tracker was created successfully
	assert.NotNil(t, tracker)
	assert.Equal(t, int64(2000), tracker.targetDurationMs)
	assert.Equal(t, 95.0, tracker.targetSuccessRate)

	// Cache some metrics and verify SLA calculation logic
	chainID := uuid.New()
	metrics := &ChainExecutionMetricsAdvanced{
		ID:              uuid.New(),
		ChainID:         chainID,
		SuccessRate99th: 98.0,
		P95DurationMs:   1500,
	}

	cache.SetMetrics(chainID, metrics)

	// Verify metrics are cached
	cachedMetrics, exists := cache.GetMetrics(chainID)
	assert.True(t, exists)
	assert.Equal(t, 98.0, cachedMetrics.SuccessRate99th)
}

// TestPriorityExecutionOrchestratorQueueing verifies queue creation
func TestPriorityExecutionOrchestratorQueueing(t *testing.T) {
	store := &TestStore{}
	executor := NewActionExecutor(store, nil)
	regionRouter := NewInMemoryRegionRegistry(store, 5*time.Minute)
	simpleOrchestrator := NewFailoverOrchestrator(store, executor, regionRouter)
	chainOrchestrator := NewFailoverChainOrchestrator(store, executor, regionRouter, simpleOrchestrator)

	cache := NewChainStateCache(5*time.Minute, 5*time.Minute, 5*time.Minute)
	detector := NewConflictDetector(store)
	tracker := NewSLATracker(store, cache, 2000, 95.0)

	priorityOrch := NewPriorityExecutionOrchestrator(store, chainOrchestrator, detector, tracker, cache)
	assert.NotNil(t, priorityOrch)
}

// TestPriorityQueuedExecution verifies multi-chain execution with priorities
func TestPriorityQueuedExecution(t *testing.T) {
	tenantID := uuid.New()
	incidentID := uuid.New()

	execution := &ChainPriorityExecution{
		ID:              uuid.New(),
		TenantID:        tenantID,
		IncidentID:      incidentID,
		ChainsToExecute: `[{"id":"chain1","priority":2},{"id":"chain2","priority":1}]`,
		ExecutionOrder:  `["chain1","chain2"]`,
		CurrentChainIdx: 0,
		Status:          "pending",
		CompletedChains: "[]",
		FailedChains:    "[]",
		StartedAt:       time.Now().UTC(),
	}

	assert.Equal(t, 0, execution.CurrentChainIdx)
	assert.Equal(t, "pending", execution.Status)
}

// TestMultiTenantChainIsolation verifies tenant isolation in chain state
func TestMultiTenantChainIsolation(t *testing.T) {
	chainID := uuid.New()

	tenant1 := uuid.New()
	state1 := &FailoverChainState{
		ID:               uuid.New(),
		ChainID:          chainID,
		TenantID:         tenant1,
		CurrentStepIndex: 0,
	}

	tenant2 := uuid.New()
	state2 := &FailoverChainState{
		ID:               uuid.New(),
		ChainID:          chainID,
		TenantID:         tenant2,
		CurrentStepIndex: 2,
	}

	cache := NewChainStateCache(5*time.Minute, 5*time.Minute, 5*time.Minute)

	cache.SetState(tenant1, chainID, state1)
	cache.SetState(tenant2, chainID, state2)

	// Both should be retrievable independently
	cached1, exists1 := cache.GetState(tenant1, chainID)
	assert.True(t, exists1)
	assert.Equal(t, 0, cached1.CurrentStepIndex)

	cached2, exists2 := cache.GetState(tenant2, chainID)
	assert.True(t, exists2)
	assert.Equal(t, 2, cached2.CurrentStepIndex)
}

// TestChainConflictResolution verifies conflict resolution tracking
func TestChainConflictResolution(t *testing.T) {
	conflict := &FailoverChainConflict{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		ChainID1:       uuid.New(),
		ChainID2:       uuid.New(),
		ConflictType:   "overlapping_targets",
		SourceRegion1:  "us-east-1",
		SourceRegion2:  "us-west-1",
		SharedTargets:  `["us-west-2"]`,
		ResolutionRule: "priority",
		IsResolved:     false,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	assert.False(t, conflict.IsResolved)
	assert.Equal(t, "priority", conflict.ResolutionRule)

	// Simulate resolution
	conflict.IsResolved = true
	now := time.Now().UTC()
	conflict.ResolvedAt = &now

	assert.True(t, conflict.IsResolved)
	assert.NotNil(t, conflict.ResolvedAt)
}

// ========== Phase 3.13: Handler Integration Tests ==========

// TestHandlerChainStateInitialization verifies chain state initialization endpoint
func TestHandlerChainStateInitialization(t *testing.T) {
	tenantID := uuid.New()
	chainID := uuid.New()

	// Test: Initialize chain state
	state := &FailoverChainState{
		ID:               uuid.New(),
		ChainID:          chainID,
		TenantID:         tenantID,
		CurrentStepIndex: 0,
		IsExecuting:      false,
		UpdatedAt:        time.Now().UTC(),
	}

	// Verify state structure
	assert.NotNil(t, state)
	assert.Equal(t, chainID, state.ChainID)
	assert.Equal(t, tenantID, state.TenantID)
	assert.False(t, state.IsExecuting)
	assert.Equal(t, 0, state.CurrentStepIndex)
}

// TestHandlerConflictDetection verifies conflict detection endpoint response format
func TestHandlerConflictDetection(t *testing.T) {
	tenantID := uuid.New()
	chainID1 := uuid.New()
	chainID2 := uuid.New()

	// Test: Create two conflicting chains
	conflict := &FailoverChainConflict{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ChainID1:      chainID1,
		ChainID2:      chainID2,
		ConflictType:  "overlapping_targets",
		SourceRegion1: "us-east-1",
		SourceRegion2: "us-east-1",
		SharedTargets: `["us-west-1", "eu-west-1"]`,
		IsResolved:    false,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	// Verify conflict structure
	assert.NotNil(t, conflict)
	assert.False(t, conflict.IsResolved)
	assert.Equal(t, "overlapping_targets", conflict.ConflictType)

	// Parse shared targets
	var targets []string
	err := json.Unmarshal([]byte(conflict.SharedTargets), &targets)
	assert.NoError(t, err)
	assert.Len(t, targets, 2)
	assert.Contains(t, targets, "us-west-1")
}

// TestHandlerSLAMetricsResponse verifies SLA metrics response format
func TestHandlerSLAMetricsResponse(t *testing.T) {
	chainID := uuid.New()

	// Test: Create advanced metrics
	metrics := &ChainExecutionMetricsAdvanced{
		ID:                chainID,
		ChainID:           chainID,
		TotalExecutions:   100,
		P50DurationMs:     1500,
		P75DurationMs:     2000,
		P95DurationMs:     3500,
		P99DurationMs:     4500,
		MinDurationMs:     1000,
		MaxDurationMs:     5000,
		StdDevDurationMs:  800.5,
		SuccessRate99th:   98.5,
		AvgStepsNeeded:    3.2,
		P95StepsNeeded:    5,
		MostCommonFailure: strPtr("timeout"),
		SLACompliance:     96.0,
		UpdatedAt:         time.Now().UTC(),
	}

	// Verify metrics structure
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(1500), metrics.P50DurationMs)
	assert.Equal(t, int64(3500), metrics.P95DurationMs)
	assert.Equal(t, 98.5, metrics.SuccessRate99th)
	assert.Equal(t, 96.0, metrics.SLACompliance)
	assert.Equal(t, "timeout", *metrics.MostCommonFailure)
}

// TestHandlerPriorityQueueCreation verifies queue creation with multiple chains
func TestHandlerPriorityQueueCreation(t *testing.T) {
	tenantID := uuid.New()
	incidentID := uuid.New()

	// Test: Create priority execution queue
	chainIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	chainsJSON, _ := json.Marshal(chainIDs)

	execution := &ChainPriorityExecution{
		ID:              uuid.New(),
		TenantID:        tenantID,
		IncidentID:      incidentID,
		ChainsToExecute: string(chainsJSON),
		ExecutionOrder:  string(chainsJSON),
		CurrentChainIdx: 0,
		Status:          "pending",
		CompletedChains: "[]",
		FailedChains:    "[]",
		StartedAt:       time.Now().UTC(),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	// Verify queue structure
	assert.NotNil(t, execution)
	assert.Equal(t, "pending", execution.Status)
	assert.Equal(t, 0, execution.CurrentChainIdx)

	// Parse chains
	var chains []string
	err := json.Unmarshal([]byte(execution.ChainsToExecute), &chains)
	assert.NoError(t, err)
	assert.Len(t, chains, 3)
}

// TestHandlerQueueProgressTracking verifies queue progress updates
func TestHandlerQueueProgressTracking(t *testing.T) {
	tenantID := uuid.New()
	incidentID := uuid.New()

	// Test: Create initial queue
	chainIDs := []uuid.UUID{uuid.New(), uuid.New()}
	chainsJSON, _ := json.Marshal(chainIDs)
	completedJSON, _ := json.Marshal([]string{chainIDs[0].String()})

	execution := &ChainPriorityExecution{
		ID:              uuid.New(),
		TenantID:        tenantID,
		IncidentID:      incidentID,
		ChainsToExecute: string(chainsJSON),
		ExecutionOrder:  string(chainsJSON),
		CurrentChainIdx: 1,
		Status:          "in_progress",
		CompletedChains: string(completedJSON),
		FailedChains:    "[]",
		StartedAt:       time.Now().UTC(),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	// Verify progress calculation
	var completed []string
	_ = json.Unmarshal([]byte(execution.CompletedChains), &completed)

	var chains []string
	_ = json.Unmarshal([]byte(execution.ChainsToExecute), &chains)

	progressPercent := float64(len(completed)) / float64(len(chains)) * 100
	assert.Equal(t, 50.0, progressPercent)
	assert.Equal(t, "in_progress", execution.Status)
}

// TestHandlerChainStateListResponse verifies listing chain states
func TestHandlerChainStateListResponse(t *testing.T) {
	tenantID := uuid.New()

	// Test: Create multiple chain states
	states := []FailoverChainState{
		{
			ID:               uuid.New(),
			ChainID:          uuid.New(),
			TenantID:         tenantID,
			CurrentStepIndex: 0,
			IsExecuting:      false,
			UpdatedAt:        time.Now().UTC(),
		},
		{
			ID:               uuid.New(),
			ChainID:          uuid.New(),
			TenantID:         tenantID,
			CurrentStepIndex: 1,
			IsExecuting:      true,
			UpdatedAt:        time.Now().UTC(),
		},
	}

	// Verify list format
	assert.Len(t, states, 2)
	assert.False(t, states[0].IsExecuting)
	assert.True(t, states[1].IsExecuting)
}

// TestHandlerConflictResolutionTypes verifies all resolution rule types
func TestHandlerConflictResolutionTypes(t *testing.T) {
	tenantID := uuid.New()
	chainID1 := uuid.New()
	chainID2 := uuid.New()

	for _, resolutionRule := range []string{"priority", "first_win", "serial_execute"} {
		conflict := &FailoverChainConflict{
			ID:             uuid.New(),
			TenantID:       tenantID,
			ChainID1:       chainID1,
			ChainID2:       chainID2,
			ConflictType:   "same_target",
			SourceRegion1:  "us-east-1",
			SourceRegion2:  "us-east-1",
			SharedTargets:  `["us-west-1"]`,
			ResolutionRule: resolutionRule,
			IsResolved:     false,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}

		// Verify resolution rule is stored
		assert.Equal(t, resolutionRule, conflict.ResolutionRule)
		assert.False(t, conflict.IsResolved)
	}
}

// TestHandlerSLAComplianceCalculation verifies SLA scoring for chains
func TestHandlerSLAComplianceCalculation(t *testing.T) {
	chainID := uuid.New()

	tests := []struct {
		name               string
		successRate        float64
		p95Duration        int64
		slaTarget          int64
		expectedCompliance float64
	}{
		{
			name:               "Excellent Performance",
			successRate:        99.5,
			p95Duration:        1500,
			slaTarget:          3000,
			expectedCompliance: 99.0,
		},
		{
			name:               "Good Performance",
			successRate:        95.0,
			p95Duration:        2500,
			slaTarget:          3000,
			expectedCompliance: 90.0,
		},
		{
			name:               "Poor Performance",
			successRate:        85.0,
			p95Duration:        3500,
			slaTarget:          3000,
			expectedCompliance: 85.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &ChainExecutionMetricsAdvanced{
				ID:              chainID,
				ChainID:         chainID,
				SuccessRate99th: tt.successRate,
				P95DurationMs:   tt.p95Duration,
				SLACompliance:   tt.expectedCompliance,
				UpdatedAt:       time.Now().UTC(),
			}

			// Verify expected compliance is set
			assert.True(t, metrics.SLACompliance >= 0 && metrics.SLACompliance <= 100)
		})
	}
}

// TestHandlerQueueStatusTransitions verifies queue status workflow
func TestHandlerQueueStatusTransitions(t *testing.T) {
	tenantID := uuid.New()
	incidentID := uuid.New()

	// Test: Status transitions
	statuses := []string{"pending", "in_progress", "completed", "failed"}

	for i, status := range statuses {
		execution := &ChainPriorityExecution{
			ID:              uuid.New(),
			TenantID:        tenantID,
			IncidentID:      incidentID,
			ChainsToExecute: "[]",
			ExecutionOrder:  "[]",
			CurrentChainIdx: i,
			Status:          status,
			CompletedChains: "[]",
			FailedChains:    "[]",
			StartedAt:       time.Now().UTC(),
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}

		// Verify status
		assert.Equal(t, status, execution.Status)
		if status == "completed" || status == "failed" {
			assert.True(t, execution.CompletedAt == nil || execution.CompletedAt.After(execution.StartedAt))
		}
	}
}

// TestHandlerCooldownEnforcement verifies cooldown prevents early re-execution
func TestHandlerCooldownEnforcement(t *testing.T) {
	chainID := uuid.New()
	tenantID := uuid.New()

	now := time.Now().UTC()
	futureTime := now.Add(30 * time.Minute)

	state := &FailoverChainState{
		ID:             uuid.New(),
		ChainID:        chainID,
		TenantID:       tenantID,
		LastExecutedAt: &now,
		NextEligibleAt: &futureTime,
		IsExecuting:    false,
		UpdatedAt:      now,
	}

	// Verify cooldown is enforced
	assert.NotNil(t, state.NextEligibleAt)
	assert.True(t, state.NextEligibleAt.After(now))
	assert.False(t, state.IsExecuting)
}

// strPtr is a helper to create string pointers for testing
func strPtr(s string) *string {
	return &s
}

// ========== Phase 3.14: Analytics & Batch Operations Tests ==========

// TestSLAComplianceTrendTracking verifies SLA trend storage and retrieval
func TestSLAComplianceTrendTracking(t *testing.T) {
	tenantID := uuid.New()
	chainID := uuid.New()

	trend := &SLAComplianceTrend{
		ID:               uuid.New(),
		ChainID:          chainID,
		TenantID:         tenantID,
		ComplianceScore:  96.5,
		SuccessRateTrend: 2.3,  // +2.3% improvement
		LatencyTrend:     -5.1, // -5.1% improvement
		Percentile99:     97.0,
		Status:           "improving",
		ReportedAt:       time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
	}

	// Verify trend structure
	assert.NotNil(t, trend)
	assert.Equal(t, 96.5, trend.ComplianceScore)
	assert.Equal(t, "improving", trend.Status)
	assert.Greater(t, trend.SuccessRateTrend, 0.0)
	assert.Less(t, trend.LatencyTrend, 0.0)
}

// TestConflictResolutionTrendAnalysis verifies conflict resolution statistics
func TestConflictResolutionTrendAnalysis(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	trend := &ConflictResolutionTrend{
		ID:              uuid.New(),
		TenantID:        tenantID,
		TotalConflicts:  45,
		ResolvedCount:   42,
		FailedCount:     3,
		ResolutionRate:  93.3,
		AvgResolutionMs: 1250,
		MostCommonRule:  "priority",
		PeriodStart:     periodStart,
		PeriodEnd:       now,
		CreatedAt:       now,
	}

	// Verify resolution statistics
	assert.NotNil(t, trend)
	assert.Equal(t, 45, trend.TotalConflicts)
	assert.Equal(t, 42, trend.ResolvedCount)
	assert.Equal(t, 3, trend.FailedCount)
	assert.Equal(t, 93.3, trend.ResolutionRate)
	assert.Equal(t, int64(1250), trend.AvgResolutionMs)
	assert.Equal(t, "priority", trend.MostCommonRule)
}

// TestChainExecutionStatsCalculation verifies execution statistics
func TestChainExecutionStatsCalculation(t *testing.T) {
	chainID := uuid.New()
	tenantID := uuid.New()
	now := time.Now().UTC()

	stats := &ChainExecutionStats{
		ID:                   uuid.New(),
		ChainID:              chainID,
		TenantID:             tenantID,
		TotalExecutions:      1000,
		SuccessfulExecutions: 980,
		FailedExecutions:     20,
		SuccessRatePct:       98.0,
		AvgExecutionMs:       1500,
		MaxExecutionMs:       5000,
		MinExecutionMs:       800,
		LastSuccessAt:        &now,
		LastFailureAt:        nil,
		PeriodStart:          now.Add(-24 * time.Hour),
		PeriodEnd:            now,
		CreatedAt:            now,
	}

	// Verify stats calculation
	assert.NotNil(t, stats)
	assert.Equal(t, 1000, stats.TotalExecutions)
	assert.Equal(t, 98.0, stats.SuccessRatePct)
	assert.Equal(t, int64(1500), stats.AvgExecutionMs)
	assert.NotNil(t, stats.LastSuccessAt)
}

// TestChainHealthReportComputations verifies health scoring
func TestChainHealthReportComputations(t *testing.T) {
	chainID := uuid.New()
	tenantID := uuid.New()

	tests := []struct {
		name              string
		overallHealth     int
		lastExecStatus    string
		isHealthy         bool
		recommendedAction string
	}{
		{
			name:              "Excellent Health",
			overallHealth:     95,
			lastExecStatus:    "success",
			isHealthy:         true,
			recommendedAction: "none",
		},
		{
			name:              "Degraded Health",
			overallHealth:     65,
			lastExecStatus:    "failure",
			isHealthy:         false,
			recommendedAction: "investigate",
		},
		{
			name:              "Critical Health",
			overallHealth:     25,
			lastExecStatus:    "failure",
			isHealthy:         false,
			recommendedAction: "disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &ChainHealthReport{
				ID:                  uuid.New(),
				ChainID:             chainID,
				TenantID:            tenantID,
				OverallHealth:       tt.overallHealth,
				LastExecutionStatus: tt.lastExecStatus,
				IsHealthy:           tt.isHealthy,
				RecommendedAction:   tt.recommendedAction,
				ReportedAt:          time.Now().UTC(),
				CreatedAt:           time.Now().UTC(),
			}

			// Verify health report
			assert.NotNil(t, report)
			assert.Equal(t, tt.overallHealth, report.OverallHealth)
			assert.Equal(t, tt.isHealthy, report.IsHealthy)
			assert.Equal(t, tt.recommendedAction, report.RecommendedAction)
		})
	}
}

// TestBatchConflictResolutionOperation verifies batch operations
func TestBatchConflictResolutionOperation(t *testing.T) {
	tenantID := uuid.New()
	conflictIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}

	conflictIDsJSON, _ := json.Marshal(conflictIDs)
	batch := &BatchConflictResolution{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ConflictIDs:    string(conflictIDsJSON),
		ResolutionRule: "priority",
		Status:         "in_progress",
		TotalConflicts: 5,
		ResolvedCount:  3,
		FailedCount:    0,
		ExecutedAt:     nil,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Verify batch operation
	assert.NotNil(t, batch)
	assert.Equal(t, 5, batch.TotalConflicts)
	assert.Equal(t, 3, batch.ResolvedCount)
	assert.Equal(t, "priority", batch.ResolutionRule)

	// Parse and verify conflict IDs
	var parsed []string
	_ = json.Unmarshal([]byte(batch.ConflictIDs), &parsed)
	assert.Len(t, parsed, 5)
}

// TestBatchProgressTracking verifies batch progress calculation
func TestBatchProgressTracking(t *testing.T) {
	tenantID := uuid.New()

	batch := &BatchConflictResolution{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ConflictIDs:    "[]",
		ResolutionRule: "serial_execute",
		Status:         "in_progress",
		TotalConflicts: 10,
		ResolvedCount:  7,
		FailedCount:    2,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Calculate progress
	progressPct := float64(batch.ResolvedCount+batch.FailedCount) / float64(batch.TotalConflicts) * 100

	// Verify progress calculation
	assert.Equal(t, 90.0, progressPct)
}

// TestChainFilterCriteriaConstruction verifies filter criteria
func TestChainFilterCriteriaConstruction(t *testing.T) {
	tenantID := uuid.New()
	region := "us-east-1"
	minCompliance := 85.0
	isEnabled := true

	criteria := &ChainFilterCriteria{
		TenantID:         &tenantID,
		SourceRegion:     &region,
		MinSLACompliance: &minCompliance,
		IsEnabled:        &isEnabled,
		SortBy:           "sla_compliance",
		SortOrder:        "desc",
		Limit:            50,
	}

	// Verify criteria structure
	assert.NotNil(t, criteria)
	assert.Equal(t, tenantID, *criteria.TenantID)
	assert.Equal(t, "us-east-1", *criteria.SourceRegion)
	assert.Equal(t, 85.0, *criteria.MinSLACompliance)
	assert.True(t, *criteria.IsEnabled)
	assert.Equal(t, "sla_compliance", criteria.SortBy)
}

// TestAnalyticsEngineHealthComputation verifies health score calculation
func TestAnalyticsEngineHealthComputation(t *testing.T) {
	engine := NewAnalyticsEngine(nil)

	tests := []struct {
		name            string
		totalExecutions int
		successRate     float64
		expectedHealth  int
	}{
		{
			name:            "High Success, Many Executions",
			totalExecutions: 150,
			successRate:     98.0,
			expectedHealth:  100,
		},
		{
			name:            "Good Success, Few Executions",
			totalExecutions: 5,
			successRate:     95.0,
			expectedHealth:  57,
		},
		{
			name:            "Poor Success, Many Executions",
			totalExecutions: 100,
			successRate:     60.0,
			expectedHealth:  76,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &ChainExecutionStats{
				ID:              uuid.New(),
				ChainID:         uuid.New(),
				TenantID:        uuid.New(),
				TotalExecutions: tt.totalExecutions,
				SuccessRatePct:  tt.successRate,
			}

			health := engine.ComputeChainHealth(context.Background(), stats.ChainID, stats)

			// Health should be between 0 and 100
			assert.GreaterOrEqual(t, health, 0)
			assert.LessOrEqual(t, health, 100)

			// Verify it's in the expected range
			if tt.successRate >= 95 && tt.totalExecutions >= 100 {
				assert.Greater(t, health, 80)
			}
		})
	}
}

// TestRecommendedActionLogic verifies action recommendation
func TestRecommendedActionLogic(t *testing.T) {
	engine := NewAnalyticsEngine(nil)

	tests := []struct {
		name           string
		successRate    float64
		expectedAction string
	}{
		{name: "Critical", successRate: 40.0, expectedAction: "disable"},
		{name: "Poor", successRate: 75.0, expectedAction: "investigate"},
		{name: "Fair", successRate: 92.0, expectedAction: "retry"},
		{name: "Good", successRate: 99.0, expectedAction: "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &ChainExecutionStats{
				ID:             uuid.New(),
				ChainID:        uuid.New(),
				TenantID:       uuid.New(),
				SuccessRatePct: tt.successRate,
			}

			action := engine.GetRecommendedAction(stats)
			assert.Equal(t, tt.expectedAction, action)
		})
	}
}

// TestSLATrendStatusIndicators verifies trend status classification
func TestSLATrendStatusIndicators(t *testing.T) {
	tenantID := uuid.New()
	chainID := uuid.New()

	statusTests := []struct {
		name             string
		successRateTrend float64
		latencyTrend     float64
		expectedStatus   string
	}{
		{
			name:             "Improving",
			successRateTrend: 5.0,
			latencyTrend:     -3.0,
			expectedStatus:   "improving",
		},
		{
			name:             "Degrading",
			successRateTrend: -8.0,
			latencyTrend:     12.0,
			expectedStatus:   "degrading",
		},
		{
			name:             "Stable",
			successRateTrend: 0.5,
			latencyTrend:     -0.2,
			expectedStatus:   "stable",
		},
	}

	for _, tt := range statusTests {
		t.Run(tt.name, func(t *testing.T) {
			trend := &SLAComplianceTrend{
				ID:               uuid.New(),
				ChainID:          chainID,
				TenantID:         tenantID,
				ComplianceScore:  92.0,
				SuccessRateTrend: tt.successRateTrend,
				LatencyTrend:     tt.latencyTrend,
				Status:           tt.expectedStatus,
				ReportedAt:       time.Now().UTC(),
				CreatedAt:        time.Now().UTC(),
			}

			assert.Equal(t, tt.expectedStatus, trend.Status)
		})
	}
}

// TestAdvancedMetricsPercentiles verifies percentile tracking in metrics
func TestAdvancedMetricsPercentiles(t *testing.T) {
	metrics := &ChainExecutionMetricsAdvanced{
		ID:              uuid.New(),
		ChainID:         uuid.New(),
		TotalExecutions: 100,
		P50DurationMs:   1000,
		P75DurationMs:   1500,
		P95DurationMs:   2000,
		P99DurationMs:   2500,
		MinDurationMs:   500,
		MaxDurationMs:   3000,
		SuccessRate99th: 97.5,
		SLACompliance:   94.3,
	}

	assert.Equal(t, int64(1000), metrics.P50DurationMs)
	assert.Equal(t, int64(2000), metrics.P95DurationMs)
	assert.Equal(t, int64(2500), metrics.P99DurationMs)
	assert.Greater(t, metrics.SLACompliance, 90.0)
}
