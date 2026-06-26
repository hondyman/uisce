package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/observability"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"
)

// AccessIntelligenceService provides a unified interface for advanced access control features.
type AccessIntelligenceService struct {
	db            *sqlx.DB
	collabService *CollaborationService
	autoService   *AutomationService
	// Enhanced caching with sharding and versioning
	governanceCache *analytics.ShardedCache
	versionManager  *analytics.VersionManager
	// Performance enhancements
	tenantConfig *TenantConfigService
	jobQueue     *BackgroundJobQueue
	perfMonitor  *PerformanceMonitor
	// Object pools for GC optimization
	govCtxPool *sync.Pool
	// Async audit logging
	auditChan chan *AuditEvent
	auditWG   sync.WaitGroup
	// Concurrency control with per-tenant token buckets
	tokenBuckets map[string]chan struct{}
	tbMux        sync.RWMutex
	// Single-flight for cache stampede protection
	sfGroup singleflight.Group
	// Per-tenant circuit breakers
	circuitBreakers map[string]*CircuitBreaker
	cbMux           sync.RWMutex
	// Dynatrace manager for tracing
	dtManager *observability.DynatraceManager
	// Unified QoS manager
	qosManager *QoSManager
}

// AuditEvent represents an audit event for async logging
type AuditEvent struct {
	ID        uuid.UUID
	Request   models.EvaluateAccessRequest
	Response  models.EvaluateAccessResponse
	Timestamp time.Time
	TraceData json.RawMessage
}

// NewAccessIntelligenceService creates a new AccessIntelligenceService.
func NewAccessIntelligenceService(db *sqlx.DB, collabService *CollaborationService, autoService *AutomationService, dtManager *observability.DynatraceManager, perfMonitor *PerformanceMonitor) *AccessIntelligenceService {
	// Initialize sharded cache with 16 shards, 1000 max entries per shard
	cache := analytics.NewShardedCache(16, 1000)
	versionManager := analytics.NewVersionManager()
	tenantConfig := NewTenantConfigService(db)
	jobQueue := NewBackgroundJobQueue()

	// Initialize object pool for governance contexts
	govCtxPool := &sync.Pool{
		New: func() interface{} {
			return &analytics.GovernanceContext{
				AllowedMetrics:    make([]string, 0, 10),
				AllowedDimensions: make([]string, 0, 10),
				RequiredFilters:   make([]analytics.QueryFilter, 0, 5),
				AppliedPolicies:   make([]analytics.AppliedGovernancePolicy, 0, 5),
				AssetMappings:     make(map[string]string),
			}
		},
	}

	// Initialize async audit channel
	auditChan := make(chan *AuditEvent, 1000) // Buffered channel for backpressure

	service := &AccessIntelligenceService{
		db:              db,
		collabService:   collabService,
		autoService:     autoService,
		governanceCache: cache,
		versionManager:  versionManager,
		tenantConfig:    tenantConfig,
		jobQueue:        jobQueue,
		perfMonitor:     perfMonitor,
		govCtxPool:      govCtxPool,
		auditChan:       auditChan,
		tokenBuckets:    make(map[string]chan struct{}),
		circuitBreakers: make(map[string]*CircuitBreaker),
		// sfGroup is zero-value ready
		dtManager:  dtManager,
		qosManager: NewQoSManager(db),
	}

	// Start async audit workers
	for i := 0; i < 4; i++ { // 4 worker goroutines
		service.auditWG.Add(1)
		go service.auditWorker()
	}

	return service
}

// auditWorker processes audit events asynchronously
func (s *AccessIntelligenceService) auditWorker() {
	defer s.auditWG.Done()

	const batchSize = 100
	const batchTimeout = 5 * time.Second

	events := make([]*AuditEvent, 0, batchSize)
	timer := time.NewTimer(batchTimeout)
	defer timer.Stop()

	for {
		select {
		case event, ok := <-s.auditChan:
			if !ok { // Channel closed
				if len(events) > 0 {
					s.processBatch(events)
				}
				return
			}
			events = append(events, event)
			if len(events) >= batchSize {
				s.processBatch(events)
				events = make([]*AuditEvent, 0, batchSize) // Reset batch
				timer.Reset(batchTimeout)                  // Reset timer after processing
			}
		case <-timer.C:
			if len(events) > 0 {
				s.processBatch(events)
				events = make([]*AuditEvent, 0, batchSize) // Reset batch
			}
			timer.Reset(batchTimeout) // Reset timer for the next cycle
		}
	}
}

// processBatch handles the actual storage of audit events.
func (s *AccessIntelligenceService) processBatch(events []*AuditEvent) {
	for _, event := range events {
		// Implement actual database write with batching
		// Use batch insert for better performance
		eventJSON, _ := json.Marshal(event)
		fmt.Printf("[AccessIntelligence] Storing event: %s\n", string(eventJSON))

		// In production: Batch insert to database
		// query := `INSERT INTO access_events (tenant_id, user_id, resource, action, timestamp, metadata) VALUES ...`
		// Use sqlx.NamedExec or prepared statement with batch
	}

	fmt.Printf("[AccessIntelligence] Stored %d events\n", len(events))
}

// Shutdown gracefully shuts down the service
func (s *AccessIntelligenceService) Shutdown() {
	close(s.auditChan)
	s.auditWG.Wait()

	// Stop QoS manager goroutine pools
	if s.qosManager != nil {
		s.qosManager.Stop()
	}
}

// GetEffectiveClaims calculates a user's final permissions after applying all intelligence layers.
// This is the new, unified entry point for checking access.
func (s *AccessIntelligenceService) GetEffectiveClaims(ctx context.Context, userID, tenantID string) (claims []models.SemanticModelClaim, err error) {
	traceErr := s.dtManager.TraceFunc(ctx, "AccessIntelligenceService.GetEffectiveClaims", func(traceCtx context.Context) error {
		claims, err = s.getEffectiveClaimsInternal(traceCtx, userID, tenantID)
		return err
	}, attribute.String("user_id", userID), attribute.String("tenant_id", tenantID))

	return claims, traceErr
}

// getEffectiveClaimsInternal contains the core logic for GetEffectiveClaims, designed to be wrapped by a tracer.
func (s *AccessIntelligenceService) getEffectiveClaimsInternal(ctx context.Context, userID, tenantID string) ([]models.SemanticModelClaim, error) {
	// Get current versions for cache key
	claimsVersion := s.versionManager.GetClaimsVersion(tenantID)
	policyVersion := s.versionManager.GetPolicyVersion(tenantID)
	cacheKey := generateCacheKey(tenantID, userID, claimsVersion, policyVersion)

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("semlayer.tenant_id", tenantID),
		attribute.String("semlayer.user_id", userID),
		attribute.String("semlayer.cache.key", cacheKey),
	)

	// Get tenant QoS configuration
	tenantConfig, err := s.tenantConfig.GetConfig(tenantID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to get tenant config for %s: %v", tenantID, err)
		tenantConfig = s.tenantConfig.getDefaultConfig(tenantID) // Use default if not found
	}

	// Perform QoS checks
	if err := s.qosManager.CheckQoS(ctx, tenantID, tenantConfig); err != nil {
		span.SetAttributes(
			attribute.String("semlayer.qos.error", err.Error()),
		)
		// Check if it's a QoS error with HTTP status
		if qosErr, ok := err.(*QoSError); ok {
			span.SetAttributes(attribute.Int("semlayer.qos.http_status", qosErr.HTTPStatus))
		}
		return nil, err
	}

	span.SetAttributes(attribute.Bool("semlayer.qos.allowed", true))

	// Try cache first
	if entry, found := s.governanceCache.Get(cacheKey); found {
		logging.GetLogger().Sugar().Infof("AccessIntelligence: Cache HIT for user %s", userID)
		span.SetAttributes(attribute.Bool("semlayer.cache.hit", true))
		return s.convertGovernanceContextToClaims(entry.GovernanceContext), nil
	}

	logging.GetLogger().Sugar().Infof("AccessIntelligence: Cache MISS for user %s. Computing claims.", userID)
	span.SetAttributes(attribute.Bool("semlayer.cache.hit", false))

	// Use singleflight to ensure only one goroutine computes claims for a given key at a time.
	v, err, _ := s.sfGroup.Do(cacheKey, func() (interface{}, error) {
		// This function is executed only once for a given cacheKey while multiple goroutines might be waiting.
		baseClaims, err := s.collabService.GetEffectiveClaimsForUser(ctx, userID)
		if err != nil {
			return nil, err
		}

		// Mock filtering by tenant - in reality, claims would have a tenant_id.
		tenantClaims := append([]models.SemanticModelClaim{}, baseClaims...)

		// 3. Fetch and apply active JIT grants for the user. This is a key enhancement.
		jitGrants, err := s.listActiveJITGrants(ctx, userID)
		if err != nil {
			// Log the error but don't fail the whole operation. JIT grants are additive.
			logging.GetLogger().Sugar().Warnf("Failed to fetch JIT grants for user %s: %v", userID, err)
		} else {
			// Augment the base claims with the JIT claims.
			jitClaims, err := s.convertJITGrantsToClaims(ctx, jitGrants)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("Failed to convert JIT grants to claims for user %s: %v", userID, err)
			} else {
				// A real implementation would need a sophisticated merge strategy.
				tenantClaims = append(tenantClaims, jitClaims...)
				span.SetAttributes(attribute.Int("semlayer.claims.jit_count", len(jitClaims)))
			}
		}

		// Create governance context from claims
		govCtx := s.convertClaimsToGovernanceContext(tenantClaims, tenantID, userID)

		// Create the cache entry
		cacheEntry := &analytics.CacheEntry{
			GovernanceContext: govCtx,
			ClaimsVersion:     claimsVersion,
			PolicyVersion:     policyVersion,
			CreatedAt:         time.Now(),
			AccessedAt:        time.Now().UnixNano(),
		}

		// Cache the result for subsequent requests
		s.governanceCache.Put(cacheKey, cacheEntry, 5*time.Minute)

		return cacheEntry, nil
	})

	if err != nil {
		// Record QoS failure
		s.qosManager.RecordQoSResult(tenantID, false)
		return nil, fmt.Errorf("failed to compute effective claims via singleflight: %w", err)
	}

	// The result from singleflight is the cached entry.
	cachedEntry := v.(*analytics.CacheEntry)

	// Record QoS success
	s.qosManager.RecordQoSResult(tenantID, true)

	return s.convertGovernanceContextToClaims(cachedEntry.GovernanceContext), nil
}

// GrantClaim grants a claim, performing conflict checks and tenant binding.
func (s *AccessIntelligenceService) GrantClaim(ctx context.Context, req models.GrantClaimRequest, actorID string) (*models.SemanticModelClaim, *models.ClaimConflict, error) {
	// 1. Run conflict check before granting.
	conflicts, err := s.collabService.DetectClaimConflicts(ctx, req.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check for conflicts: %w", err)
	}

	for _, conflict := range conflicts {
		if conflict.ModelID == req.ModelID {
			logging.GetLogger().Sugar().Infof("AccessIntelligence: Conflict detected for user %s on model %s. Aborting grant.", req.UserID, req.ModelID)
			return nil, &conflict, nil // Return the detected conflict
		}
	}

	// 2. If no conflicts, grant the claim.
	logging.GetLogger().Sugar().Info("AccessIntelligence: No conflicts found. Granting claim.")
	claim, err := s.collabService.GrantDirectClaim(ctx, req.UserID, req.ModelID.String(), req.Permission, actorID)
	if err != nil {
		return nil, nil, err
	}

	// In a real app, you would also bind the tenant_id to the claim here.
	return claim, nil, nil
}

// AssignBundle assigns a claim bundle to a user. Mock implementation.
func (s *AccessIntelligenceService) AssignBundle(ctx context.Context, userID, bundleID, actorID string) error {
	logging.GetLogger().Sugar().Infof("AccessIntelligence: Actor %s assigning bundle %s to user %s", actorID, bundleID, userID)
	// Logic: Look up bundle, iterate its permissions, and grant each as a claim with source 'bundle:<id>'.
	return nil
}

// asyncLogDecision sends audit events to the async channel
func (s *AccessIntelligenceService) asyncLogDecision(_ context.Context, decisionID uuid.UUID, req models.EvaluateAccessRequest, resp *models.EvaluateAccessResponse, trace []map[string]interface{}) {
	traceJSON, _ := json.Marshal(trace)

	event := &AuditEvent{
		ID:        decisionID,
		Request:   req,
		Response:  *resp,
		Timestamp: time.Now(),
		TraceData: traceJSON,
	}

	select {
	case s.auditChan <- event:
		// Successfully queued for async processing
	default:
		// Channel is full, drop the audit event to prevent blocking
		logging.GetLogger().Sugar().Warnf("Audit channel full, dropping event for decision %s", decisionID)
	}
}

// logDecision is a helper to create and store the decision trace.
func (s *AccessIntelligenceService) logDecision(ctx context.Context, decisionID uuid.UUID, req models.EvaluateAccessRequest, decision, reason string, evaluatedClaims []map[string]interface{}) {
	resp := &models.EvaluateAccessResponse{
		Decision:   decision,
		Reason:     reason,
		DecisionID: decisionID,
	}
	s.asyncLogDecision(ctx, decisionID, req, resp, evaluatedClaims)
}

// --- Real-Time Evaluation Engine ---

// EvaluateAccess performs a real-time access check against the effective claims cache.
func (s *AccessIntelligenceService) EvaluateAccess(ctx context.Context, req models.EvaluateAccessRequest) (resp *models.EvaluateAccessResponse, err error) {
	start := time.Now()

	traceErr := s.dtManager.TraceFunc(ctx, "AccessIntelligenceService.EvaluateAccess", func(traceCtx context.Context) error {
		resp, err = s.evaluateAccessInternal(traceCtx, req)
		return err
	},
		attribute.String("user_id", req.UserID),
		attribute.String("tenant_id", req.TenantID),
		attribute.String("asset_id", req.AssetID),
		attribute.String("action", req.Action),
	)

	// Record performance metrics
	if s.perfMonitor != nil {
		duration := time.Since(start)
		success := err == nil && resp != nil && resp.Decision == "allow"
		s.perfMonitor.RecordTenantLatency(req.TenantID, duration, success)
	}

	return resp, traceErr
}

// evaluateAccessInternal contains the core logic for EvaluateAccess, designed to be wrapped by a tracer.
func (s *AccessIntelligenceService) evaluateAccessInternal(ctx context.Context, req models.EvaluateAccessRequest) (*models.EvaluateAccessResponse, error) {
	// Get tenant QoS configuration
	tenantConfig, err := s.tenantConfig.GetConfig(req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}

	// Check circuit breaker for this tenant
	cb := s.getCircuitBreaker(req.TenantID)
	if err := cb.Call(func() error {
		// Check tenant-specific concurrency limits
		tokenBucket := s.getTokenBucket(req.TenantID, tenantConfig.ConcurrencyLimit)
		select {
		case tokenBucket <- struct{}{}:
			defer func() { <-tokenBucket }() // Release token
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Bucket is full, shed load
			return fmt.Errorf("tenant concurrency limit exceeded")
		}
	}); err != nil {
		return &models.EvaluateAccessResponse{
			Decision: "deny", Reason: "Service overloaded, please retry later", DecisionID: uuid.New(),
		}, nil
	}

	// 1. Get effective claims (this will hit the cache).
	claims, err := s.GetEffectiveClaims(ctx, req.UserID, req.TenantID)
	if err != nil {
		return nil, err
	}

	// This will hold the trace of evaluated claims
	var evaluatedClaimsForTrace []map[string]interface{}

	// 2. Perform the check.
	for _, claim := range claims {
		// Add every considered claim to the trace
		evaluatedClaimsForTrace = append(evaluatedClaimsForTrace, map[string]interface{}{
			"claim_id":   claim.ID,
			"source":     claim.GrantedBy,
			"permission": claim.Permission,
			"model_id":   claim.ModelID,
		})

		// This is a simplified check. A real engine would handle asset hierarchies (metric -> model).
		if claim.ModelID.String() == req.AssetID {
			if claim.Permission == req.Action || (req.Action == "query" && claim.Permission == "read") {

				decisionID := uuid.New()
				reason := fmt.Sprintf("Access allowed via claim ID %s (source: %s)", claim.ID, claim.GrantedBy)

				response := &models.EvaluateAccessResponse{
					Decision:     "allow",
					Reason:       reason,
					AllowedScope: claim.Scope,
					DecisionID:   decisionID,
				}

				// Dynatrace: Add attributes for the decision
				span := trace.SpanFromContext(ctx)
				span.SetAttributes(
					attribute.String("semlayer.decision", "allow"),
					attribute.String("semlayer.decision.reason", reason),
					attribute.String("semlayer.decision.source", claim.GrantedBy),
				)

				// Async audit logging
				s.asyncLogDecision(ctx, decisionID, req, response, evaluatedClaimsForTrace)

				return response, nil
			}
		}
	}

	// 3. If no matching claim is found, deny access.
	decisionID := uuid.New()
	reason := "No effective claim found for the requested asset and action."

	response := &models.EvaluateAccessResponse{
		Decision:   "deny",
		Reason:     reason,
		DecisionID: decisionID,
	}

	// Dynatrace: Add attributes for the decision
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("semlayer.decision", "deny"),
		attribute.String("semlayer.decision.reason", reason),
	)

	s.asyncLogDecision(ctx, decisionID, req, response, evaluatedClaimsForTrace)

	return response, nil
}

// RefreshClaimsCache invalidates and re-computes the claims for a user/tenant.
func (s *AccessIntelligenceService) RefreshClaimsCache(ctx context.Context, userID, tenantID string) error {
	// Increment claims version to invalidate all cached entries for this tenant
	s.versionManager.IncrementClaimsVersion(tenantID)

	logging.GetLogger().Sugar().Infof("AccessIntelligence: Invalidated cache for tenant %s", tenantID)
	// The cache will be repopulated on the next call to GetEffectiveClaims.
	return nil
}

// GetDecisionTrace retrieves the full technical trace for a decision. Mock implementation.
func (s *AccessIntelligenceService) GetDecisionTrace(ctx context.Context, decisionID uuid.UUID) (*models.AccessDecisionTrace, error) {
	// In a real app, this would SELECT from the access_decision_trace table.
	// For this mock, we'll construct a plausible trace.
	evaluatedClaimsJSON, _ := json.Marshal([]map[string]interface{}{
		{"source": "role:analyst", "modelId": "d1b6a5e0-9a9a-4b1a-8b0a-1b1b1b1b1b1b", "permissions": []string{"read"}, "result": "pass"},
		{"source": "manual_grant", "modelId": "avg_order_value", "permissions": []string{"update"}, "result": "deny_scope"},
	})
	matchedPoliciesJSON, _ := json.Marshal([]map[string]interface{}{
		{"policyId": "certified_update_requires_approval", "result": "fail"},
	})

	return &models.AccessDecisionTrace{
		ID:              uuid.New(),
		DecisionLogID:   decisionID,
		UserID:          "patrick",
		AssetID:         "avg_order_value",
		Action:          "update",
		Decision:        "deny",
		EvaluatedClaims: evaluatedClaimsJSON,
		MatchedPolicies: matchedPoliciesJSON,
		TenantScope:     "acme_corp",
		Reason:          "Access denied because your role grants read access to orders_view, but update access to avg_order_value is restricted by policy: Certified metrics require steward approval for updates.",
		EvaluatedAt:     time.Now().Add(-5 * time.Second),
	}, nil
}

// SimulateAccess performs a "what-if" access check with temporary claims.
func (s *AccessIntelligenceService) SimulateAccess(ctx context.Context, req models.SimulateAccessRequest) (*models.EvaluateAccessResponse, error) {
	// 1. Get the user's base effective claims (without caching, as simulation is a one-off).
	baseClaims, err := s.collabService.GetEffectiveClaimsForUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	// 2. Augment with simulated claims. This logic overrides existing claims for the same model.
	augmentedClaims := make(map[string]models.SemanticModelClaim)
	for _, claim := range baseClaims {
		key := fmt.Sprintf("%s-%s", claim.ModelID, claim.Permission)
		augmentedClaims[key] = claim
	}

	for _, simClaim := range req.SimulatedClaims {
		tempClaim := models.SemanticModelClaim{
			ID:         uuid.New(),
			UserID:     req.UserID,
			ModelID:    simClaim.ModelID,
			Permission: simClaim.Permission,
			GrantedBy:  "simulation",
			Status:     "active",
		}
		key := fmt.Sprintf("%s-%s", tempClaim.ModelID, tempClaim.Permission)
		augmentedClaims[key] = tempClaim
	}

	var finalSimulatedClaims []models.SemanticModelClaim
	for _, claim := range augmentedClaims {
		finalSimulatedClaims = append(finalSimulatedClaims, claim)
	}

	// 3. Create an equivalent EvaluateAccessRequest for logging.
	evalReq := models.EvaluateAccessRequest{
		UserID: req.UserID, TenantID: req.TenantID, AssetID: req.AssetID, Action: req.Action,
	}
	var evaluatedClaimsForTrace []map[string]interface{}

	// 4. Perform the check against the augmented claims.
	for _, claim := range finalSimulatedClaims {
		evaluatedClaimsForTrace = append(evaluatedClaimsForTrace, map[string]interface{}{"claim_id": claim.ID, "source": claim.GrantedBy, "permission": claim.Permission, "model_id": claim.ModelID})
		if claim.ModelID.String() == req.AssetID && (claim.Permission == req.Action || (req.Action == "query" && claim.Permission == "read")) {
			decisionID := uuid.New()
			reason := fmt.Sprintf("Access would be ALLOWED via claim from source: %s", claim.GrantedBy)
			s.logDecision(ctx, decisionID, evalReq, "allow", reason, evaluatedClaimsForTrace)
			return &models.EvaluateAccessResponse{
				Decision: "allow", Reason: reason, AllowedScope: claim.Scope, DecisionID: decisionID,
			}, nil
		}
	}

	// 5. If no match, deny.
	decisionID := uuid.New()
	reason := "No effective or simulated claim found for the requested asset and action."
	s.logDecision(ctx, decisionID, evalReq, "deny", reason, evaluatedClaimsForTrace)
	return &models.EvaluateAccessResponse{
		Decision: "deny", Reason: reason, DecisionID: decisionID,
	}, nil
}

// GetGovernanceCockpitSnapshot aggregates data for the main governance dashboard. Mock implementation.
func (s *AccessIntelligenceService) GetGovernanceCockpitSnapshot(ctx context.Context, tenantID string) (*models.GovernanceCockpitSnapshot, error) {
	// In a real app, this would be a series of optimized queries. Here we'll call other mock services.

	// 1. Get health score components (mocked)
	healthScore := models.GovernanceHealthScore{
		Score:             88.2,
		CertifiedCoverage: 92.3,
		ClaimAlignment:    85.1,
		UsageCoverage:     78.9,
		RiskExposure:      5.5,
	}

	// 2. Get data from collab service
	driftedClaims, _ := s.collabService.DetectClaimDrift(ctx)
	conflicts, _ := s.collabService.DetectClaimConflicts(ctx, "patrick")
	policies, _ := s.collabService.ListAccessPolicies(ctx)
	simulations, _ := s.collabService.ListClaimSimulations(ctx)
	suppressed, _ := s.collabService.ListAlertsByStatus(ctx, "suppressed")
	escalated, _ := s.collabService.ListAlertsByStatus(ctx, "escalated")
	autoLogs, _ := s.autoService.ListAutomationLogs(ctx)

	snapshot := &models.GovernanceCockpitSnapshot{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		Timestamp:            time.Now(),
		HealthScore:          healthScore,
		ConflictCount:        len(conflicts),
		DriftCount:           len(driftedClaims),
		PolicyCount:          len(policies),
		SimulationCount:      len(simulations),
		SuppressedAlertCount: len(suppressed),
		EscalatedAlertCount:  len(escalated),
		AutomationStatus:     s.autoService.GetStatus(),
		AutoResolvedCount:    len(autoLogs),
	}

	return snapshot, nil
}

// convertClaimsToGovernanceContext converts claims to governance context
func (s *AccessIntelligenceService) convertClaimsToGovernanceContext(claims []models.SemanticModelClaim, tenantID, userID string) *analytics.GovernanceContext {
	govCtx := s.govCtxPool.Get().(*analytics.GovernanceContext)
	govCtx.UserID = userID
	govCtx.TenantID = tenantID
	govCtx.Datasource = "" // Will be set by caller

	// Reset slices
	govCtx.AllowedMetrics = govCtx.AllowedMetrics[:0]
	govCtx.AllowedDimensions = govCtx.AllowedDimensions[:0]
	govCtx.RequiredFilters = govCtx.RequiredFilters[:0]
	govCtx.AppliedPolicies = govCtx.AppliedPolicies[:0]

	// Clear and rebuild asset mappings
	for k := range govCtx.AssetMappings {
		delete(govCtx.AssetMappings, k)
	}

	// Convert claims to governance context
	for _, claim := range claims {
		// Parse claim scopes to determine allowed metrics/dimensions
		for _, scope := range claim.Scope {
			if strings.HasPrefix(scope, "metric:") {
				metric := strings.TrimPrefix(scope, "metric:")
				govCtx.AllowedMetrics = append(govCtx.AllowedMetrics, metric)
			} else if strings.HasPrefix(scope, "dimension:") {
				dimension := strings.TrimPrefix(scope, "dimension:")
				govCtx.AllowedDimensions = append(govCtx.AllowedDimensions, dimension)
			}
		}
	}

	return govCtx
}

// convertGovernanceContextToClaims converts governance context back to claims (simplified)
func (s *AccessIntelligenceService) convertGovernanceContextToClaims(govCtx *analytics.GovernanceContext) []models.SemanticModelClaim {
	// This is a simplified conversion - in practice you'd need to maintain claim metadata
	claims := make([]models.SemanticModelClaim, 0, len(govCtx.AllowedMetrics)+len(govCtx.AllowedDimensions))

	tenantUUID, _ := uuid.Parse(govCtx.TenantID) // Parse string to UUID

	for _, metric := range govCtx.AllowedMetrics {
		claims = append(claims, models.SemanticModelClaim{
			UserID:     govCtx.UserID,
			TenantID:   tenantUUID,
			Scope:      []string{"metric:" + metric},
			Status:     "active",
			Permission: "read",
		})
	}

	for _, dimension := range govCtx.AllowedDimensions {
		claims = append(claims, models.SemanticModelClaim{
			UserID:     govCtx.UserID,
			TenantID:   tenantUUID,
			Scope:      []string{"dimension:" + dimension},
			Status:     "active",
			Permission: "read",
		})
	}

	return claims
}

// getCircuitBreaker retrieves or creates a circuit breaker for a tenant
func (s *AccessIntelligenceService) getCircuitBreaker(tenantID string) *CircuitBreaker {
	s.cbMux.RLock()
	cb, ok := s.circuitBreakers[tenantID]
	s.cbMux.RUnlock()

	if !ok {
		cb = NewCircuitBreaker(5, 30*time.Second)
		s.cbMux.Lock()
		s.circuitBreakers[tenantID] = cb
		s.cbMux.Unlock()
	}

	return cb
}

// getTokenBucket retrieves or creates a token bucket for a tenant
func (s *AccessIntelligenceService) getTokenBucket(tenantID string, capacity int) chan struct{} {
	s.tbMux.RLock()
	bucket, ok := s.tokenBuckets[tenantID]
	s.tbMux.RUnlock()

	if !ok {
		bucket = make(chan struct{}, capacity)
		s.tbMux.Lock()
		s.tokenBuckets[tenantID] = bucket
		s.tbMux.Unlock()
	}

	return bucket
}

// listActiveJITGrants retrieves all active JIT grants for a user.
func (s *AccessIntelligenceService) listActiveJITGrants(ctx context.Context, userID string) ([]models.JITAddonGrant, error) {
	var grants []models.JITAddonGrant
	query := `SELECT id, user_id, bundle_id, granted_by, granted_at, expires_at, reason, status 
	          FROM jit_addon_grant 
	          WHERE user_id = $1 AND status = 'active' AND expires_at > NOW()`
	err := s.db.SelectContext(ctx, &grants, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active JIT grants for user %s: %w", userID, err)
	}
	return grants, nil
}

// convertJITGrantsToClaims fetches the claims from the micro-bundles associated with the JIT grants.
func (s *AccessIntelligenceService) convertJITGrantsToClaims(ctx context.Context, grants []models.JITAddonGrant) ([]models.SemanticModelClaim, error) {
	if len(grants) == 0 {
		return nil, nil
	}

	bundleIDs := make([]uuid.UUID, len(grants))
	grantMap := make(map[uuid.UUID]models.JITAddonGrant)
	for i, grant := range grants {
		bundleIDs[i] = grant.BundleID
		grantMap[grant.BundleID] = grant
	}

	var bundles []models.MicroBundle
	query, args, err := sqlx.In("SELECT id, claims FROM micro_bundle WHERE id IN (?)", bundleIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build query for micro-bundles: %w", err)
	}
	query = s.db.Rebind(query)
	if err := s.db.SelectContext(ctx, &bundles, query, args...); err != nil {
		return nil, fmt.Errorf("failed to fetch micro-bundles: %w", err)
	}

	var jitClaims []models.SemanticModelClaim
	for _, bundle := range bundles {
		// In a real implementation, you would parse bundle.Claims (JSONB) into SemanticModelClaim structs.
		// For this example, we'll create a placeholder claim.
		grant := grantMap[bundle.ID]
		jitClaims = append(jitClaims, models.SemanticModelClaim{UserID: grant.UserID, ModelID: uuid.New(), Permission: "read", GrantedBy: fmt.Sprintf("jit_grant:%s", grant.ID), ExpiresAt: &grant.ExpiresAt, Status: "active"})
	}

	return jitClaims, nil
}

func generateCacheKey(tenantID, userID string, claimsVersion, policyVersion int64) string {
	return fmt.Sprintf("claims:%s:%s:%d:%d", tenantID, userID, claimsVersion, policyVersion)
}
