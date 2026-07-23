package ops

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Phase 3.6/3.7: Region Routing Layer
// Control plane for tenant → region → infrastructure mappings
// ============================================================================

// RegionTarget represents a routable infrastructure target for a region
type RegionTarget struct {
	Region            string
	StarRocksCluster  string  // e.g., "starrocks-us-east-1"
	RedpandaBroker    string  // e.g., "kafka-us-east-1:9092"
	TemporalNamespace string  // e.g., "us-east-1"
	OpsWorkerPool     string  // e.g., "worker-pool-us-east-1"
	FailoverTarget    *string // e.g., "eu-west-1" if this region is down
	IsActive          bool
	LastHealthCheckAt time.Time
}

// TenantRouting represents how to route a specific tenant
type TenantRouting struct {
	TenantID        uuid.UUID
	PreferredRegion string                   // Primary region for this tenant
	AllowedRegions  []string                 // Regions tenant can use
	Targets         map[string]*RegionTarget // region -> target mappings
	UpdatedAt       time.Time
}

// RegionRouter defines the routing interface for the control plane
type RegionRouter interface {
	// Tenant routing
	GetTenantRegion(ctx context.Context, tenantID uuid.UUID) (string, error)
	GetTenantAllowedRegions(ctx context.Context, tenantID uuid.UUID) ([]string, error)
	SetTenantRegion(ctx context.Context, tenantID uuid.UUID, preferredRegion string, allowedRegions []string) error

	// Region target resolution
	GetRegionTarget(ctx context.Context, region string) (*RegionTarget, error)
	ListRegionTargets(ctx context.Context) (map[string]*RegionTarget, error)
	RegisterRegionTarget(ctx context.Context, target *RegionTarget) error

	// Routing decisions
	RouteForTenant(ctx context.Context, tenantID uuid.UUID) (*RegionTarget, error)
	RouteForIncident(ctx context.Context, incident *Incident) (*RegionTarget, error)
	RouteForEvent(ctx context.Context, event *Event) (*RegionTarget, error)

	// Failover logic
	GetFailoverTarget(ctx context.Context, region string) (*RegionTarget, error)
	MarkRegionDown(ctx context.Context, region string) error
	MarkRegionUp(ctx context.Context, region string) error
}

// InMemoryRegionRegistry implements RegionRouter with in-memory caching
// Syncs from Store on initialization and refreshes periodically
type InMemoryRegionRegistry struct {
	mu              sync.RWMutex
	tenantRouting   map[uuid.UUID]*TenantRouting // tenant_id -> routing
	regionTargets   map[string]*RegionTarget     // region_code -> target
	downRegions     map[string]bool              // region_code -> is_down
	lastRefresh     time.Time
	refreshInterval time.Duration
	store           Store
}

// NewInMemoryRegionRegistry creates a registry backed by the Store
func NewInMemoryRegionRegistry(store Store, refreshInterval time.Duration) *InMemoryRegionRegistry {
	if refreshInterval == 0 {
		refreshInterval = 5 * time.Minute // Default: refresh every 5 minutes
	}

	registry := &InMemoryRegionRegistry{
		tenantRouting:   make(map[uuid.UUID]*TenantRouting),
		regionTargets:   make(map[string]*RegionTarget),
		downRegions:     make(map[string]bool),
		refreshInterval: refreshInterval,
		store:           store,
	}

	// Register default regions for development/testing
	// These serve as fallbacks when regions are accessed before database initialization
	defaultRegions := []string{"us-east-1", "us-west-1", "us-west-2", "eu-west-1", "ap-southeast-1", "ap-northeast-1"}
	for _, region := range defaultRegions {
		registry.regionTargets[region] = &RegionTarget{
			Region:            region,
			StarRocksCluster:  fmt.Sprintf("starrocks-%s", region),
			RedpandaBroker:    fmt.Sprintf("redpanda-%s:9092", region),
			TemporalNamespace: region,
			OpsWorkerPool:     fmt.Sprintf("worker-pool-%s", region),
			IsActive:          true,
		}
	}

	return registry
}

// Initialize loads regional metadata from the Store
func (r *InMemoryRegionRegistry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load all region targets from RegionRouting table
	configs, err := r.store.ListRegionConfigs(ctx, true) // active regions only
	if err != nil {
		return fmt.Errorf("failed to load region configs: %w", err)
	}

	// Build region targets from configs and routings
	for _, config := range configs {
		target := &RegionTarget{
			Region:            config.RegionCode,
			IsActive:          config.IsActive,
			LastHealthCheckAt: time.Now(),
		}

		// TODO: Load actual cluster/broker/namespace mappings from metadata tables
		// For now, use convention-based naming
		target.StarRocksCluster = fmt.Sprintf("starrocks-%s", config.RegionCode)
		target.RedpandaBroker = fmt.Sprintf("redpanda-%s:9092", config.RegionCode)
		target.TemporalNamespace = config.RegionCode
		target.OpsWorkerPool = fmt.Sprintf("worker-pool-%s", config.RegionCode)

		r.regionTargets[config.RegionCode] = target
	}

	r.lastRefresh = time.Now()
	return nil
}

// GetTenantRegion returns the primary region for a tenant
func (r *InMemoryRegionRegistry) GetTenantRegion(ctx context.Context, tenantID uuid.UUID) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routing, exists := r.tenantRouting[tenantID]
	if !exists {
		// Default to us-east-1 if tenant not explicitly routed
		return "us-east-1", nil
	}

	return routing.PreferredRegion, nil
}

// GetTenantAllowedRegions returns all regions a tenant can use
func (r *InMemoryRegionRegistry) GetTenantAllowedRegions(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routing, exists := r.tenantRouting[tenantID]
	if !exists {
		// Default: tenant can use any active region
		regions := []string{}
		for region, target := range r.regionTargets {
			if target.IsActive {
				regions = append(regions, region)
			}
		}
		return regions, nil
	}

	return routing.AllowedRegions, nil
}

// SetTenantRegion sets routing for a specific tenant
func (r *InMemoryRegionRegistry) SetTenantRegion(ctx context.Context, tenantID uuid.UUID, preferredRegion string, allowedRegions []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate preferred region exists
	target, exists := r.regionTargets[preferredRegion]
	if !exists {
		return fmt.Errorf("region %s not found in registry", preferredRegion)
	}

	routing := &TenantRouting{
		TenantID:        tenantID,
		PreferredRegion: preferredRegion,
		AllowedRegions:  allowedRegions,
		Targets:         make(map[string]*RegionTarget),
		UpdatedAt:       time.Now(),
	}

	// Build targets for all allowed regions
	for _, region := range allowedRegions {
		if t, exists := r.regionTargets[region]; exists {
			routing.Targets[region] = t
		}
	}

	// Always add preferred region
	routing.Targets[preferredRegion] = target

	r.tenantRouting[tenantID] = routing

	// TODO: Persist to Store: r.store.SetTenantRegionRouting(ctx, ...)

	return nil
}

// GetRegionTarget returns the infrastructure target for a region
func (r *InMemoryRegionRegistry) GetRegionTarget(ctx context.Context, region string) (*RegionTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	target, exists := r.regionTargets[region]
	if !exists {
		return nil, fmt.Errorf("region %s not registered", region)
	}

	if !target.IsActive {
		return nil, fmt.Errorf("region %s is not active", region)
	}

	return target, nil
}

// ListRegionTargets returns all region targets
func (r *InMemoryRegionRegistry) ListRegionTargets(ctx context.Context) (map[string]*RegionTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*RegionTarget)
	for region, target := range r.regionTargets {
		result[region] = target
	}

	return result, nil
}

// RegisterRegionTarget adds a region to the registry
func (r *InMemoryRegionRegistry) RegisterRegionTarget(ctx context.Context, target *RegionTarget) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if target.Region == "" {
		return fmt.Errorf("region target must have a region")
	}

	target.LastHealthCheckAt = time.Now()
	r.regionTargets[target.Region] = target

	// TODO: Persist to Store: r.store.CreateRegionConfig(ctx, ...)

	return nil
}

// RouteForTenant determines the best region target for a tenant
func (r *InMemoryRegionRegistry) RouteForTenant(ctx context.Context, tenantID uuid.UUID) (*RegionTarget, error) {
	preferredRegion, err := r.GetTenantRegion(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return r.RouteToRegion(ctx, preferredRegion)
}

// RouteForIncident routes based on incident's region
func (r *InMemoryRegionRegistry) RouteForIncident(ctx context.Context, incident *Incident) (*RegionTarget, error) {
	if incident.Region == nil {
		// Fallback to us-east-1
		return r.RouteToRegion(ctx, "us-east-1")
	}

	return r.RouteToRegion(ctx, *incident.Region)
}

// RouteForEvent routes based on event's region
func (r *InMemoryRegionRegistry) RouteForEvent(ctx context.Context, event *Event) (*RegionTarget, error) {
	if event.Region == nil {
		// Fallback to us-east-1
		return r.RouteToRegion(ctx, "us-east-1")
	}

	return r.RouteToRegion(ctx, *event.Region)
}

// RouteToRegion is the core routing logic: region → target
// With failover support
func (r *InMemoryRegionRegistry) RouteToRegion(ctx context.Context, region string) (*RegionTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	target, exists := r.regionTargets[region]
	if !exists {
		return nil, fmt.Errorf("region %s not found", region)
	}

	// Check if region is down
	if r.downRegions[region] {
		// Try failover
		if target.FailoverTarget != nil {
			failoverTarget, exists := r.regionTargets[*target.FailoverTarget]
			if exists && !r.downRegions[*target.FailoverTarget] {
				return failoverTarget, nil
			}
		}

		return nil, fmt.Errorf("region %s is down and no active failover available", region)
	}

	return target, nil
}

// GetFailoverTarget returns the next region if current region fails
func (r *InMemoryRegionRegistry) GetFailoverTarget(ctx context.Context, region string) (*RegionTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	target, exists := r.regionTargets[region]
	if !exists {
		return nil, fmt.Errorf("region %s not found", region)
	}

	if target.FailoverTarget == nil {
		return nil, fmt.Errorf("region %s has no failover target configured", region)
	}

	failoverTarget, exists := r.regionTargets[*target.FailoverTarget]
	if !exists {
		return nil, fmt.Errorf("failover region %s not found", *target.FailoverTarget)
	}

	return failoverTarget, nil
}

// MarkRegionDown marks a region as unhealthy
func (r *InMemoryRegionRegistry) MarkRegionDown(ctx context.Context, region string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.regionTargets[region]; !exists {
		return fmt.Errorf("region %s not found", region)
	}

	r.downRegions[region] = true

	// TODO: Log this event, alert operators
	// TODO: Persist to RegionalHealth: status = "critical"

	return nil
}

// MarkRegionUp marks a region as healthy
func (r *InMemoryRegionRegistry) MarkRegionUp(ctx context.Context, region string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.regionTargets[region]; !exists {
		return fmt.Errorf("region %s not found", region)
	}

	delete(r.downRegions, region)

	// TODO: Log this event
	// TODO: Persist to RegionalHealth: status = "healthy"

	return nil
}

// IsRegionDown returns whether a region is marked as down
func (r *InMemoryRegionRegistry) IsRegionDown(region string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.downRegions[region]
}

// RefreshIfNeeded reloads regional metadata if refresh interval has elapsed
func (r *InMemoryRegionRegistry) RefreshIfNeeded(ctx context.Context) error {
	r.mu.RLock()
	if time.Since(r.lastRefresh) < r.refreshInterval {
		r.mu.RUnlock()
		return nil
	}
	r.mu.RUnlock()

	return r.Initialize(ctx)
}
