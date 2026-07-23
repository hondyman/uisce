package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/singleflight"
)

// TenantQoSConfig defines Quality of Service settings per tenant
type TenantQoSConfig struct {
	TenantID         string
	Tier             TenantTier
	ConcurrencyLimit int
	TokenRate        int // tokens per second
	BurstTokens      int
	CPULimit         float64 // percentage
	MemoryLimit      int64   // bytes
	CacheTTL         time.Duration
	Priority         int // higher = more priority
	Features         FeatureFlags
}

// FeatureFlags controls per-tenant feature availability
type FeatureFlags struct {
	AutomationAutoApply    bool
	ConversationalFeatures bool
	AdvancedAnalytics      bool
	CustomIntegrations     bool
}

// TenantTier represents service tiers
type TenantTier int

const (
	TierBronze TenantTier = iota
	TierSilver
	TierGold
)

// EffectiveClaims represents precomputed claims for a user
type EffectiveClaims struct {
	UserID     string
	TenantID   string
	Claims     []Claim
	Version    uint64
	ComputedAt time.Time
}

// Claim represents an access claim
type Claim struct {
	ID         string
	Permission string
	Resource   string
}

// ECStore implements sharded cache for EffectiveClaims
type ECStore struct {
	shards [256]shard
}

type shard struct {
	mu   sync.RWMutex
	data map[string]*EffectiveClaims
}

// NewECStore creates a new sharded EffectiveClaims store
func NewECStore() *ECStore {
	store := &ECStore{}
	for i := range store.shards {
		store.shards[i].data = make(map[string]*EffectiveClaims)
	}
	return store
}

// Get retrieves EffectiveClaims from the cache
func (s *ECStore) Get(key string) (*EffectiveClaims, bool) {
	sh := &s.shards[xxhash.Sum64String(key)&0xFF]
	sh.mu.RLock()
	ec, ok := sh.data[key]
	sh.mu.RUnlock()
	return ec, ok
}

// Set stores EffectiveClaims in the cache
func (s *ECStore) Set(key string, ec *EffectiveClaims) {
	sh := &s.shards[xxhash.Sum64String(key)&0xFF]
	sh.mu.Lock()
	sh.data[key] = ec
	sh.mu.Unlock()
}

// AISvc represents the enhanced Access Intelligence Service
type AISvc struct {
	db              *sqlx.DB
	cache           *ECStore
	tenants         map[string]*TenantQoSConfig
	tenantMux       sync.RWMutex
	sfGroup         singleflight.Group
	circuitBreakers map[string]*CircuitBreaker
	cbMux           sync.RWMutex
}

// CircuitBreaker implements per-tenant circuit breaker pattern
type CircuitBreaker struct {
	mu          sync.RWMutex
	failures    int
	lastFailure time.Time
	state       CircuitState
	threshold   int
	timeout     time.Duration
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
		state:     StateClosed,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) < cb.timeout {
			return fmt.Errorf("circuit breaker is open")
		}
		cb.state = StateHalfOpen
	}

	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.threshold {
			cb.state = StateOpen
		}
		return err
	}

	// Success - reset on half-open, keep closed
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failures = 0
	}
	return nil
}

// NewAISvc creates a new Access Intelligence Service
func NewAISvc(db *sqlx.DB) *AISvc {
	return &AISvc{
		db:              db,
		cache:           NewECStore(),
		tenants:         make(map[string]*TenantQoSConfig),
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
}

// GetTenantConfig retrieves QoS configuration for a tenant
func (svc *AISvc) GetTenantConfig(tenantID string) *TenantQoSConfig {
	svc.tenantMux.RLock()
	config, ok := svc.tenants[tenantID]
	svc.tenantMux.RUnlock()

	if !ok {
		// Default to Bronze tier
		config = &TenantQoSConfig{
			TenantID:         tenantID,
			Tier:             TierBronze,
			ConcurrencyLimit: 10,
			TokenRate:        100,
			BurstTokens:      200,
			CPULimit:         10.0,
			MemoryLimit:      100 * 1024 * 1024, // 100MB
			CacheTTL:         5 * time.Minute,
			Priority:         1,
			Features: FeatureFlags{
				AutomationAutoApply:    false,
				ConversationalFeatures: true,
				AdvancedAnalytics:      false,
				CustomIntegrations:     false,
			},
		}
		svc.tenantMux.Lock()
		svc.tenants[tenantID] = config
		svc.tenantMux.Unlock()
	}

	return config
}

// RefreshEC refreshes EffectiveClaims with singleflight deduplication
func (svc *AISvc) RefreshEC(ctx context.Context, tenant, user string, claimVer, policyVer uint64) (*EffectiveClaims, error) {
	key := fmt.Sprintf("%s|%s|%d|%d", tenant, user, claimVer, policyVer)

	// Try cache first
	if ec, ok := svc.cache.Get(key); ok {
		return ec, nil
	}

	// Use singleflight to prevent stampedes
	v, err, shared := svc.sfGroup.Do(key, func() (interface{}, error) {
		return svc.computeEffectiveClaims(ctx, tenant, user)
	})

	if err != nil {
		return nil, err
	}

	ec := v.(*EffectiveClaims)

	// Only cache if this goroutine did the computation
	if !shared {
		svc.cache.Set(key, ec)
	}

	return ec, nil
}

// computeEffectiveClaims computes effective claims from database
func (svc *AISvc) computeEffectiveClaims(_ context.Context, tenant, user string) (*EffectiveClaims, error) {
	// This would query the database for effective claims
	// Implementation depends on your database schema
	claims := []Claim{
		{ID: "claim1", Permission: "read", Resource: "metric:*"},
		{ID: "claim2", Permission: "query", Resource: "dashboard:*"},
	}

	return &EffectiveClaims{
		UserID:     user,
		TenantID:   tenant,
		Claims:     claims,
		Version:    1,
		ComputedAt: time.Now(),
	}, nil
}

// Evaluate performs access evaluation with performance targets
func (svc *AISvc) Evaluate(ctx context.Context, tenant, user, resource, action string) (bool, string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		// Record metrics for p50/p95/p99 tracking
		_ = duration // In real implementation, send to metrics collector
	}()

	// Get tenant config for QoS
	config := svc.GetTenantConfig(tenant)

	// Check circuit breaker
	cb := svc.getCircuitBreaker(tenant)
	if err := cb.Call(func() error {
		// Simulate evaluation logic with QoS limits
		if config.ConcurrencyLimit > 0 {
			// In real implementation, check active requests against limit
		}
		return nil
	}); err != nil {
		return false, "service unavailable", err
	}

	// Get effective claims (this will hit cache or compute)
	ec, err := svc.RefreshEC(ctx, tenant, user, 1, 1)
	if err != nil {
		return false, "computation failed", err
	}

	// Evaluate access
	for _, claim := range ec.Claims {
		if claim.Resource == resource || claim.Resource == "*" {
			if claim.Permission == action || claim.Permission == "admin" {
				return true, "access granted", nil
			}
		}
	}

	return false, "access denied", nil
}

// getCircuitBreaker retrieves or creates a circuit breaker for a tenant
func (svc *AISvc) getCircuitBreaker(tenantID string) *CircuitBreaker {
	svc.cbMux.RLock()
	cb, ok := svc.circuitBreakers[tenantID]
	svc.cbMux.RUnlock()

	if !ok {
		cb = NewCircuitBreaker(5, 30*time.Second)
		svc.cbMux.Lock()
		svc.circuitBreakers[tenantID] = cb
		svc.cbMux.Unlock()
	}

	return cb
}
