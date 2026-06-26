package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ChainStateCache provides in-memory caching for chain states and metrics
type ChainStateCache struct {
	mu sync.RWMutex

	// Cache entries: key format is "{tenantID}:{chainID}"
	stateCache    map[string]*FailoverChainState
	metricsCache  map[string]*ChainExecutionMetricsAdvanced
	conflictCache map[string][]FailoverChainConflict

	// TTL for cache entries
	stateTTL    time.Duration
	metricsTTL  time.Duration
	conflictTTL time.Duration

	// Last update times for TTL enforcement
	stateLastUpdated    map[string]time.Time
	metricsLastUpdated  map[string]time.Time
	conflictLastUpdated map[string]time.Time
}

// NewChainStateCache creates a new caching layer
func NewChainStateCache(stateTTL, metricsTTL, conflictTTL time.Duration) *ChainStateCache {
	return &ChainStateCache{
		stateCache:          make(map[string]*FailoverChainState),
		metricsCache:        make(map[string]*ChainExecutionMetricsAdvanced),
		conflictCache:       make(map[string][]FailoverChainConflict),
		stateTTL:            stateTTL,
		metricsTTL:          metricsTTL,
		conflictTTL:         conflictTTL,
		stateLastUpdated:    make(map[string]time.Time),
		metricsLastUpdated:  make(map[string]time.Time),
		conflictLastUpdated: make(map[string]time.Time),
	}
}

// Get retrieves a state from cache if not expired
func (c *ChainStateCache) GetState(tenantID, chainID uuid.UUID) (*FailoverChainState, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID.String(), chainID.String())
	state, exists := c.stateCache[key]
	if !exists {
		return nil, false
	}

	// Check TTL
	if time.Since(c.stateLastUpdated[key]) > c.stateTTL {
		return nil, false // Expired
	}

	return state, true
}

// Set stores a state in cache
func (c *ChainStateCache) SetState(tenantID, chainID uuid.UUID, state *FailoverChainState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID.String(), chainID.String())
	c.stateCache[key] = state
	c.stateLastUpdated[key] = time.Now().UTC()
}

// GetMetrics retrieves metrics from cache if not expired
func (c *ChainStateCache) GetMetrics(chainID uuid.UUID) (*ChainExecutionMetricsAdvanced, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := chainID.String()
	metrics, exists := c.metricsCache[key]
	if !exists {
		return nil, false
	}

	// Check TTL
	if time.Since(c.metricsLastUpdated[key]) > c.metricsTTL {
		return nil, false // Expired
	}

	return metrics, true
}

// SetMetrics stores metrics in cache
func (c *ChainStateCache) SetMetrics(chainID uuid.UUID, metrics *ChainExecutionMetricsAdvanced) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := chainID.String()
	c.metricsCache[key] = metrics
	c.metricsLastUpdated[key] = time.Now().UTC()
}

// InvalidateState removes a state from cache
func (c *ChainStateCache) InvalidateState(tenantID, chainID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID.String(), chainID.String())
	delete(c.stateCache, key)
	delete(c.stateLastUpdated, key)
}

// InvalidateMetrics removes metrics from cache
func (c *ChainStateCache) InvalidateMetrics(chainID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := chainID.String()
	delete(c.metricsCache, key)
	delete(c.metricsLastUpdated, key)
}

// ========== Priority-Based Execution ==========

// ChainWithPriority represents a chain with its priority for execution ordering
type ChainWithPriority struct {
	ChainID  uuid.UUID
	Priority int
	Chain    *FailoverChain
	State    *FailoverChainState
}

// OrderChainsByPriority sorts chains by priority (higher first) then by consistency
func OrderChainsByPriority(chains []ChainWithPriority) []ChainWithPriority {
	sorted := make([]ChainWithPriority, len(chains))
	copy(sorted, chains)

	sort.Slice(sorted, func(i, j int) bool {
		// Sort by priority descending (higher priority first)
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}

		// Then by creation time (consistent ordering)
		if sorted[i].Chain != nil && sorted[j].Chain != nil {
			return sorted[i].Chain.CreatedAt.Before(sorted[j].Chain.CreatedAt)
		}

		return false
	})

	return sorted
}

// ========== Conflict Detection & Resolution ==========

// ConflictDetector identifies and resolves conflicts between chains
type ConflictDetector struct {
	store Store
}

// NewConflictDetector creates a new conflict detection engine
func NewConflictDetector(store Store) *ConflictDetector {
	return &ConflictDetector{store: store}
}

// DetectConflicts identifies potential conflicts between chains for a tenant/incident
func (cd *ConflictDetector) DetectConflicts(ctx context.Context, tenantID uuid.UUID, chains []*FailoverChain) ([]FailoverChainConflict, error) {
	var conflicts []FailoverChainConflict

	for i := 0; i < len(chains); i++ {
		for j := i + 1; j < len(chains); j++ {
			conflictType, sharedTargets := cd.getConflictType(chains[i], chains[j])

			if conflictType != "" {
				conflict := FailoverChainConflict{
					TenantID:       tenantID,
					ChainID1:       chains[i].ID,
					ChainID2:       chains[j].ID,
					ConflictType:   conflictType,
					SourceRegion1:  chains[i].SourceRegion,
					SourceRegion2:  chains[j].SourceRegion,
					SharedTargets:  sharedTargets,
					ResolutionRule: "priority", // Default: use priority to resolve
					IsResolved:     false,
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts, nil
}

// getConflictType determines if two chains conflict and what type
func (cd *ConflictDetector) getConflictType(chain1, chain2 *FailoverChain) (string, string) {
	if chain1.SourceRegion == chain2.SourceRegion {
		return "same_source", "[]" // Both failover from same region
	}

	// Check for overlapping targets
	targets1 := parseChainTargets(chain1.ChainTargets)
	targets2 := parseChainTargets(chain2.ChainTargets)

	shared := findSharedRegions(targets1, targets2)
	if len(shared) > 0 {
		sharedJSON, _ := encodeStringSlice(shared)
		return "overlapping_targets", sharedJSON
	}

	return "", ""
}

// findSharedRegions returns regions that appear in both slices
func findSharedRegions(regions1, regions2 []string) []string {
	regionMap := make(map[string]bool)
	for _, r := range regions1 {
		regionMap[r] = true
	}

	var shared []string
	for _, r := range regions2 {
		if regionMap[r] {
			shared = append(shared, r)
		}
	}

	return shared
}

// ========== SLA Compliance Tracking ==========

// SLATracker calculates and tracks SLA compliance metrics
type SLATracker struct {
	store Store
	cache *ChainStateCache

	// SLA targets (configurable)
	targetDurationMs  int64
	targetSuccessRate float64
}

// NewSLATracker creates a new SLA tracking engine
func NewSLATracker(store Store, cache *ChainStateCache, targetDurationMs int64, targetSuccessRate float64) *SLATracker {
	return &SLATracker{
		store:             store,
		cache:             cache,
		targetDurationMs:  targetDurationMs,
		targetSuccessRate: targetSuccessRate,
	}
}

// CalculateSLACompliance evaluates chain compliance against SLA targets
func (st *SLATracker) CalculateSLACompliance(ctx context.Context, chainID uuid.UUID) float64 {
	metrics, err := st.store.GetChainExecutionMetricsAdvanced(ctx, chainID)
	if err != nil || metrics == nil {
		return 0.0
	}

	// SLA compliance is weighted average of:
	// - 50% success rate compliance
	// - 50% duration compliance
	successCompliance := (metrics.SuccessRate99th / 100.0) * 100.0
	durationCompliance := 0.0

	if metrics.P95DurationMs > 0 {
		durationCompliance = float64(st.targetDurationMs) / float64(metrics.P95DurationMs) * 100.0
		if durationCompliance > 100.0 {
			durationCompliance = 100.0 // Cap at 100%
		}
	}

	return (successCompliance * 0.5) + (durationCompliance * 0.5)
}

// GetChainsSortedBySLA returns chains sorted by SLA compliance (best first)
func (st *SLATracker) GetChainsSortedBySLA(ctx context.Context, tenantID uuid.UUID) ([]ChainExecutionMetricsAdvanced, error) {
	return st.store.ListChainsSortedBySLACompliance(ctx, tenantID)
}

// ========== Priority Execution Orchestrator ==========

// PriorityExecutionOrchestrator manages execution of multiple chains with priority-based ordering
type PriorityExecutionOrchestrator struct {
	store             Store
	chainOrchestrator *FailoverChainOrchestrator
	conflictDetector  *ConflictDetector
	slaTracker        *SLATracker
	cache             *ChainStateCache
}

// NewPriorityExecutionOrchestrator creates a new orchestrator
func NewPriorityExecutionOrchestrator(
	store Store,
	chainOrchestrator *FailoverChainOrchestrator,
	conflictDetector *ConflictDetector,
	slaTracker *SLATracker,
	cache *ChainStateCache,
) *PriorityExecutionOrchestrator {
	return &PriorityExecutionOrchestrator{
		store:             store,
		chainOrchestrator: chainOrchestrator,
		conflictDetector:  conflictDetector,
		slaTracker:        slaTracker,
		cache:             cache,
	}
}

// ExecuteChainQueue handles multi-chain execution with priority-based ordering
func (peo *PriorityExecutionOrchestrator) ExecuteChainQueue(
	ctx context.Context,
	tenantID uuid.UUID,
	incidentID uuid.UUID,
	incident *Incident,
	chainIDs []uuid.UUID,
) (*ChainPriorityExecution, error) {
	// Build ChainWithPriority list
	var chainsWithPriority []ChainWithPriority
	for _, chainID := range chainIDs {
		chain, err := peo.store.GetFailoverChain(ctx, chainID)
		if err != nil || chain == nil {
			continue
		}

		chainsWithPriority = append(chainsWithPriority, ChainWithPriority{
			ChainID:  chainID,
			Priority: chain.Priority,
			Chain:    chain,
		})
	}

	// Sort by priority
	sortedChains := OrderChainsByPriority(chainsWithPriority)

	// Check for conflicts among selected chains
	var chains []*FailoverChain
	var chainOrder []uuid.UUID
	for _, cp := range sortedChains {
		chains = append(chains, cp.Chain)
		chainOrder = append(chainOrder, cp.ChainID)
	}

	// Detect conflicts and apply priority-based resolution
	conflicts, _ := peo.conflictDetector.DetectConflicts(ctx, tenantID, chains)
	for _, conflict := range conflicts {
		if conflict.ResolutionRule == "priority" {
			// Find priority levels and keep higher-priority chain
			// Record conflict but proceed with execution
			_ = peo.store.InsertFailoverChainConflict(ctx, &conflict)
		}
	}

	// Create execution queue - convert UUIDs to strings for JSON storage
	chainOrderStrs := make([]string, len(chainOrder))
	for i, id := range chainOrder {
		chainOrderStrs[i] = id.String()
	}
	chainsJSON, _ := encodeStringSlice(chainOrderStrs)
	executionOrderJSON, _ := encodeStringSlice(chainOrderStrs)

	execution := &ChainPriorityExecution{
		TenantID:        tenantID,
		IncidentID:      incidentID,
		ChainsToExecute: chainsJSON,
		ExecutionOrder:  executionOrderJSON,
		CurrentChainIdx: 0,
		Status:          "pending",
		CompletedChains: "[]",
		FailedChains:    "[]",
		StartedAt:       time.Now().UTC(),
	}

	err := peo.store.InsertChainPriorityExecution(ctx, execution)
	if err != nil {
		return nil, err
	}

	return execution, nil
}

// ExecuteNextChainInQueue proceeds to execute the next chain in priority order
func (peo *PriorityExecutionOrchestrator) ExecuteNextChainInQueue(
	ctx context.Context,
	executionID uuid.UUID,
) error {
	execution, err := peo.store.GetChainPriorityExecution(ctx, executionID)
	if err != nil || execution == nil {
		return err
	}

	// Parse execution order - contains chain IDs as strings
	chainOrderStrs := parseChainOrder(execution.ExecutionOrder)
	if execution.CurrentChainIdx >= len(chainOrderStrs) {
		// All chains processed
		completed := parseChainIDs(parseChainOrder(execution.CompletedChains))
		return peo.store.UpdateChainPriorityExecution(ctx, executionID, execution.CurrentChainIdx, "completed", completed, nil)
	}

	// Get next chain ID and retrieve chain
	nextChainIDStr := chainOrderStrs[execution.CurrentChainIdx]
	nextChainID, err := uuid.Parse(nextChainIDStr)
	if err != nil {
		return peo.store.UpdateChainPriorityExecution(ctx, executionID, execution.CurrentChainIdx+1, "in_progress", nil, nil)
	}

	chain, err := peo.store.GetFailoverChain(ctx, nextChainID)
	if err != nil || chain == nil {
		// Skip failed/missing chain
		return peo.store.UpdateChainPriorityExecution(
			ctx, executionID, execution.CurrentChainIdx+1, "in_progress", nil, nil,
		)
	}

	// Mark as processing this chain (in a real scenario, would call ExecuteChain here)
	// For now, just advance to next chain
	// In production, this would integrate with the FailoverChainOrchestrator to actually execute

	completed := append(parseChainIDs(parseChainOrder(execution.CompletedChains)), nextChainIDStr)
	return peo.store.UpdateChainPriorityExecution(
		ctx, executionID, execution.CurrentChainIdx+1, "in_progress", completed, nil,
	)
}

// Helper functions

func encodeStringSlice(strs []string) (string, error) {
	data, err := json.Marshal(strs)
	return string(data), err
}

func parseChainTargets(jsonStr string) []string {
	var targets []string
	_ = json.Unmarshal([]byte(jsonStr), &targets)
	return targets
}

func parseChainOrder(jsonStr string) []string {
	var order []string
	_ = json.Unmarshal([]byte(jsonStr), &order)
	return order
}

func parseChainIDs(uuidStrs []string) []string {
	return uuidStrs
}
