package events

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

/**
 * Phase 3.5: Rate Limiting
 * Per-tenant quotas and fairness controls
 */

// QuotaConfig defines limits for a tenant
type QuotaConfig struct {
	TenantID          string
	EventsPerMinute   int64
	MaxConcurrentSubs int
	BurstAllowance    int64 // Extra events allowed in burst
	ResetInterval     time.Duration
}

// TokenBucket implements token bucket rate limiting algorithm
type TokenBucket struct {
	maxTokens    float64
	tokens       float64
	lastRefillNS int64
	refillRate   float64 // tokens per nanosecond
	burstAllowed float64
	mu           sync.Mutex
}

// NewTokenBucket creates a rate limiter for the given quota
func NewTokenBucket(eventsPerMinute int64, burst int64) *TokenBucket {
	maxTokens := float64(eventsPerMinute)
	burstAllowed := float64(burst)
	tb := &TokenBucket{
		maxTokens:    maxTokens,
		tokens:       maxTokens + burstAllowed, // Start with full capacity including burst
		refillRate:   maxTokens / 60e9,         // tokens per nanosecond
		burstAllowed: burstAllowed,
		lastRefillNS: time.Now().UnixNano(),
	}
	return tb
}

// TryConsume attempts to consume 1 token. Returns true if allowed, false if rate limited
func (tb *TokenBucket) TryConsume() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now().UnixNano()
	elapsed := float64(now - tb.lastRefillNS)

	// Refill tokens
	tokensToAdd := elapsed * tb.refillRate
	tb.tokens = min(tb.tokens+tokensToAdd, tb.maxTokens+tb.burstAllowed)

	tb.lastRefillNS = now

	// Try to consume
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// AvailableTokens returns current token count
func (tb *TokenBucket) AvailableTokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now().UnixNano()
	elapsed := float64(now - tb.lastRefillNS)

	// Refill tokens
	tokensToAdd := elapsed * tb.refillRate
	tb.tokens = min(tb.tokens+tokensToAdd, tb.maxTokens+tb.burstAllowed)

	tb.lastRefillNS = now

	return tb.tokens
}

// Reset clears the token bucket
func (tb *TokenBucket) Reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.tokens = tb.maxTokens
	tb.lastRefillNS = time.Now().UnixNano()
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// QuotaManager manages rate limiting for multiple tenants
type QuotaManager struct {
	buckets     map[string]*TokenBucket
	configs     map[string]*QuotaConfig
	subscribers map[string]int // tenant -> subscriber count
	mu          sync.RWMutex

	globalEventCount atomic.Int64
	globalErrorCount atomic.Int64
}

// NewQuotaManager creates a quota manager
func NewQuotaManager() *QuotaManager {
	return &QuotaManager{
		buckets:     make(map[string]*TokenBucket),
		configs:     make(map[string]*QuotaConfig),
		subscribers: make(map[string]int),
	}
}

// RegisterTenant adds a tenant with quota configuration
func (qm *QuotaManager) RegisterTenant(config *QuotaConfig) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if config.EventsPerMinute <= 0 {
		return fmt.Errorf("EventsPerMinute must be > 0")
	}
	if config.MaxConcurrentSubs <= 0 {
		return fmt.Errorf("MaxConcurrentSubs must be > 0")
	}

	qm.configs[config.TenantID] = config
	qm.buckets[config.TenantID] = NewTokenBucket(
		config.EventsPerMinute,
		config.BurstAllowance,
	)
	qm.subscribers[config.TenantID] = 0

	return nil
}

// CheckEventQuota checks if tenant can publish an event
func (qm *QuotaManager) CheckEventQuota(tenantID string) bool {
	qm.mu.RLock()
	bucket, exists := qm.buckets[tenantID]
	qm.mu.RUnlock()

	if !exists {
		// Unregistered tenant denied
		qm.globalErrorCount.Add(1)
		return false
	}

	if !bucket.TryConsume() {
		qm.globalErrorCount.Add(1)
		return false
	}

	qm.globalEventCount.Add(1)
	return true
}

// CheckSubscriberQuota checks if tenant can add another subscriber
func (qm *QuotaManager) CheckSubscriberQuota(tenantID string) bool {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	config, exists := qm.configs[tenantID]
	if !exists {
		return false
	}

	if qm.subscribers[tenantID] >= config.MaxConcurrentSubs {
		return false
	}

	qm.subscribers[tenantID]++
	return true
}

// ReleaseSubscriber decrements subscriber count
func (qm *QuotaManager) ReleaseSubscriber(tenantID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if count, exists := qm.subscribers[tenantID]; exists && count > 0 {
		qm.subscribers[tenantID]--
	}
}

// GetStats returns current quota statistics
func (qm *QuotaManager) GetStats(tenantID string) map[string]interface{} {
	qm.mu.RLock()
	config, configExists := qm.configs[tenantID]
	subCount := qm.subscribers[tenantID]
	bucket, bucketExists := qm.buckets[tenantID]
	qm.mu.RUnlock()

	if !configExists {
		return map[string]interface{}{"error": "tenant not found"}
	}

	result := map[string]interface{}{
		"tenant_id":             tenantID,
		"events_per_minute":     config.EventsPerMinute,
		"max_concurrent_subs":   config.MaxConcurrentSubs,
		"current_subscribers":   subCount,
		"burst_allowance":       config.BurstAllowance,
		"available_tokens":      0.0,
		"subscriber_quota_used": float64(subCount) / float64(config.MaxConcurrentSubs) * 100,
	}

	if bucketExists {
		result["available_tokens"] = bucket.AvailableTokens()
	}

	return result
}

// GetGlobalStats returns global statistics
func (qm *QuotaManager) GetGlobalStats() map[string]interface{} {
	qm.mu.RLock()
	tenantCount := len(qm.configs)
	totalSubs := 0
	maxSubs := 0
	for _, sub := range qm.subscribers {
		totalSubs += sub
	}
	for _, config := range qm.configs {
		maxSubs += config.MaxConcurrentSubs
	}
	qm.mu.RUnlock()

	return map[string]interface{}{
		"total_tenants":       tenantCount,
		"total_subscribers":   totalSubs,
		"max_subscribers":     maxSubs,
		"global_events":       qm.globalEventCount.Load(),
		"global_errors":       qm.globalErrorCount.Load(),
		"subscribers_percent": float64(totalSubs) / float64(maxSubs) * 100,
	}
}

// UpdateQuota adjusts rates for a tenant
func (qm *QuotaManager) UpdateQuota(tenantID string, eventsPerMinute, maxConcurrentSubs int64, burst int64) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	config, exists := qm.configs[tenantID]
	if !exists {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	if eventsPerMinute > 0 {
		config.EventsPerMinute = eventsPerMinute
		qm.buckets[tenantID] = NewTokenBucket(eventsPerMinute, burst)
	}
	if maxConcurrentSubs > 0 {
		config.MaxConcurrentSubs = int(maxConcurrentSubs)
	}
	if burst >= 0 {
		config.BurstAllowance = burst
	}

	return nil
}

// RateLimitedEventStreamBroker wraps EventStreamBroker with quota management
type RateLimitedEventStreamBroker struct {
	broker       *EventStreamBroker
	quotaManager *QuotaManager
	defaultQuota *QuotaConfig
	mu           sync.RWMutex
}

// NewRateLimitedEventStreamBroker creates a rate-limited broker
func NewRateLimitedEventStreamBroker(bufferSize int, quotaManager *QuotaManager) *RateLimitedEventStreamBroker {
	return &RateLimitedEventStreamBroker{
		broker:       NewEventStreamBroker(bufferSize),
		quotaManager: quotaManager,
		defaultQuota: &QuotaConfig{
			EventsPerMinute:   1000,
			MaxConcurrentSubs: 100,
			BurstAllowance:    200,
		},
	}
}

// Publish sends an event with quota checking
func (rlb *RateLimitedEventStreamBroker) Publish(ctx context.Context, tenantID string, event *StreamedEvent) error {
	// Check quota
	if !rlb.quotaManager.CheckEventQuota(tenantID) {
		return fmt.Errorf("rate limit exceeded for tenant: %s", tenantID)
	}

	// Publish to underlying broker
	return rlb.broker.PublishEvent(ctx, event)
}

// Subscribe with quota checking
func (rlb *RateLimitedEventStreamBroker) Subscribe(ctx context.Context, tenantID string, regions []string) (*EventSubscriber, error) {
	// Check subscriber quota
	if !rlb.quotaManager.CheckSubscriberQuota(tenantID) {
		return nil, fmt.Errorf("subscriber limit exceeded for tenant: %s", tenantID)
	}

	// Subscribe to underlying broker
	sub, err := rlb.broker.Subscribe(ctx, tenantID, regions)
	if err != nil {
		rlb.quotaManager.ReleaseSubscriber(tenantID)
		return nil, err
	}

	return sub, nil
}

// Unsubscribe with quota cleanup
func (rlb *RateLimitedEventStreamBroker) Unsubscribe(subscriberID string, tenantID string) error {
	err := rlb.broker.Unsubscribe(subscriberID)
	rlb.quotaManager.ReleaseSubscriber(tenantID)
	return err
}

// Stop gracefully shuts down the broker
func (rlb *RateLimitedEventStreamBroker) Stop() {
	_ = rlb.broker.Stop()
}

// GetQuotaStats returns stats for a tenant
func (rlb *RateLimitedEventStreamBroker) GetQuotaStats(tenantID string) map[string]interface{} {
	return rlb.quotaManager.GetStats(tenantID)
}

// GetGlobalQuotaStats returns global quota stats
func (rlb *RateLimitedEventStreamBroker) GetGlobalQuotaStats() map[string]interface{} {
	return rlb.quotaManager.GetGlobalStats()
}

// RegisterTenant registers quota for a tenant
func (rlb *RateLimitedEventStreamBroker) RegisterTenant(config *QuotaConfig) error {
	return rlb.quotaManager.RegisterTenant(config)
}
