package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Name           string        // Identifier for this rate limiter
	RequestsPerSec float64       // Number of requests allowed per second
	BurstSize      int64         // Number of requests allowed in a burst
	WindowSize     time.Duration // Time window for measuring rate
}

// RateLimitMetrics tracks rate limiting statistics
type RateLimitMetrics struct {
	TotalRequests   int64     // Total requests processed
	AllowedRequests int64     // Requests allowed through
	DeniedRequests  int64     // Requests denied (rate limited)
	CurrentRate     float64   // Current requests per second
	BurstCapacity   int64     // Current burst capacity remaining
	LastUpdateTime  time.Time // Last time metrics were updated
}

// TokenBucketLimiter implements token bucket rate limiting algorithm
type TokenBucketLimiter struct {
	config         RateLimitConfig
	tokens         float64   // Current tokens in bucket
	lastRefillTime time.Time // Last time bucket was refilled
	mu             sync.Mutex
	metrics        RateLimitMetrics
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(config RateLimitConfig) *TokenBucketLimiter {
	if config.RequestsPerSec == 0 {
		config.RequestsPerSec = 100
	}
	if config.BurstSize == 0 {
		config.BurstSize = int64(config.RequestsPerSec * 2)
	}
	if config.WindowSize == 0 {
		config.WindowSize = 1 * time.Second
	}

	return &TokenBucketLimiter{
		config:         config,
		tokens:         float64(config.BurstSize),
		lastRefillTime: time.Now(),
		metrics: RateLimitMetrics{
			BurstCapacity:  config.BurstSize,
			LastUpdateTime: time.Now(),
		},
	}
}

// Allow checks if a request should be allowed through the rate limiter
func (tbl *TokenBucketLimiter) Allow() bool {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	tbl.refillTokens()

	if tbl.tokens >= 1.0 {
		tbl.tokens -= 1.0
		atomic.AddInt64(&tbl.metrics.TotalRequests, 1)
		atomic.AddInt64(&tbl.metrics.AllowedRequests, 1)
		tbl.metrics.BurstCapacity = int64(tbl.tokens)
		return true
	}

	atomic.AddInt64(&tbl.metrics.TotalRequests, 1)
	atomic.AddInt64(&tbl.metrics.DeniedRequests, 1)
	return false
}

// AllowN checks if n requests should be allowed through
func (tbl *TokenBucketLimiter) AllowN(n int64) bool {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	tbl.refillTokens()

	if tbl.tokens >= float64(n) {
		tbl.tokens -= float64(n)
		atomic.AddInt64(&tbl.metrics.TotalRequests, n)
		atomic.AddInt64(&tbl.metrics.AllowedRequests, n)
		tbl.metrics.BurstCapacity = int64(tbl.tokens)
		return true
	}

	atomic.AddInt64(&tbl.metrics.TotalRequests, n)
	atomic.AddInt64(&tbl.metrics.DeniedRequests, n)
	return false
}

// WaitForToken waits until a token is available (up to timeout)
func (tbl *TokenBucketLimiter) WaitForToken(ctx context.Context, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for {
		if tbl.Allow() {
			return true
		}

		// Calculate wait time
		tbl.mu.Lock()
		tbl.refillTokens()
		tokensNeeded := 1.0 - tbl.tokens
		timeToWait := time.Duration(float64(time.Second) * (tokensNeeded / tbl.config.RequestsPerSec))
		tbl.mu.Unlock()

		if time.Now().Add(timeToWait).After(deadline) {
			return false
		}

		select {
		case <-time.After(timeToWait):
			continue
		case <-ctx.Done():
			return false
		}
	}
}

// ExecuteWithRateLimit runs a function if rate limit allows, otherwise returns error
func (tbl *TokenBucketLimiter) ExecuteWithRateLimit(ctx context.Context, fn func(context.Context) error) error {
	if !tbl.Allow() {
		return fmt.Errorf("rate limit exceeded for %s", tbl.config.Name)
	}

	return fn(ctx)
}

// ExecuteWithRateLimitWait runs a function, waiting if necessary for rate limit
func (tbl *TokenBucketLimiter) ExecuteWithRateLimitWait(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	if !tbl.WaitForToken(ctx, timeout) {
		return fmt.Errorf("rate limit wait timeout for %s", tbl.config.Name)
	}

	return fn(ctx)
}

// refillTokens adds tokens based on elapsed time
func (tbl *TokenBucketLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(tbl.lastRefillTime)
	tokensToAdd := elapsed.Seconds() * tbl.config.RequestsPerSec

	tbl.tokens = tbl.tokens + tokensToAdd

	// Cap at burst size
	if tbl.tokens > float64(tbl.config.BurstSize) {
		tbl.tokens = float64(tbl.config.BurstSize)
	}

	tbl.lastRefillTime = now
	tbl.metrics.LastUpdateTime = now

	// Update current rate
	total := atomic.LoadInt64(&tbl.metrics.TotalRequests)
	if total > 0 && now.Unix() > 0 {
		elapsedSeconds := now.Sub(time.Unix(0, 0)).Seconds()
		if elapsedSeconds > 0 {
			tbl.metrics.CurrentRate = float64(total) / elapsedSeconds
		}
	}
}

// GetMetrics returns rate limit metrics snapshot
func (tbl *TokenBucketLimiter) GetMetrics() RateLimitMetrics {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	metrics := tbl.metrics
	metrics.TotalRequests = atomic.LoadInt64(&tbl.metrics.TotalRequests)
	metrics.AllowedRequests = atomic.LoadInt64(&tbl.metrics.AllowedRequests)
	metrics.DeniedRequests = atomic.LoadInt64(&tbl.metrics.DeniedRequests)
	metrics.BurstCapacity = int64(tbl.tokens)

	return metrics
}

// Reset resets the rate limiter
func (tbl *TokenBucketLimiter) Reset() {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	tbl.tokens = float64(tbl.config.BurstSize)
	tbl.lastRefillTime = time.Now()
	atomic.StoreInt64(&tbl.metrics.TotalRequests, 0)
	atomic.StoreInt64(&tbl.metrics.AllowedRequests, 0)
	atomic.StoreInt64(&tbl.metrics.DeniedRequests, 0)
	tbl.metrics.CurrentRate = 0
}

// ExportMetrics returns Prometheus format metrics
func (tbl *TokenBucketLimiter) ExportMetrics() string {
	metrics := tbl.GetMetrics()

	allowRate := float64(0)
	if metrics.TotalRequests > 0 {
		allowRate = float64(metrics.AllowedRequests) / float64(metrics.TotalRequests)
	}
	denyRate := float64(0)
	if metrics.TotalRequests > 0 {
		denyRate = float64(metrics.DeniedRequests) / float64(metrics.TotalRequests)
	}

	return fmt.Sprintf(
		`rate_limit_total_requests{name="%s"} %d
rate_limit_allowed_requests{name="%s"} %d
rate_limit_denied_requests{name="%s"} %d
rate_limit_allow_rate{name="%s"} %f
rate_limit_deny_rate{name="%s"} %f
rate_limit_current_rate{name="%s"} %f
rate_limit_burst_capacity_remaining{name="%s"} %d
`,
		tbl.config.Name, metrics.TotalRequests,
		tbl.config.Name, metrics.AllowedRequests,
		tbl.config.Name, metrics.DeniedRequests,
		tbl.config.Name, allowRate,
		tbl.config.Name, denyRate,
		tbl.config.Name, metrics.CurrentRate,
		tbl.config.Name, metrics.BurstCapacity,
	)
}

// RateLimiterGroup manages multiple rate limiters
type RateLimiterGroup struct {
	limiters map[string]*TokenBucketLimiter
	mu       sync.RWMutex
}

// NewRateLimiterGroup creates a new rate limiter group
func NewRateLimiterGroup() *RateLimiterGroup {
	return &RateLimiterGroup{
		limiters: make(map[string]*TokenBucketLimiter),
	}
}

// AddLimiter adds a rate limiter to the group
func (rlg *RateLimiterGroup) AddLimiter(name string, limiter *TokenBucketLimiter) {
	rlg.mu.Lock()
	defer rlg.mu.Unlock()
	rlg.limiters[name] = limiter
}

// GetLimiter retrieves a rate limiter by name
func (rlg *RateLimiterGroup) GetLimiter(name string) *TokenBucketLimiter {
	rlg.mu.RLock()
	defer rlg.mu.RUnlock()
	return rlg.limiters[name]
}

// CheckAll checks if all requests in the group should be allowed
func (rlg *RateLimiterGroup) CheckAll() bool {
	rlg.mu.RLock()
	defer rlg.mu.RUnlock()

	for _, limiter := range rlg.limiters {
		if !limiter.Allow() {
			return false
		}
	}

	return true
}

// ExportAllMetrics returns metrics for all limiters in the group
func (rlg *RateLimiterGroup) ExportAllMetrics() string {
	rlg.mu.RLock()
	defer rlg.mu.RUnlock()

	result := ""
	for _, limiter := range rlg.limiters {
		result += limiter.ExportMetrics() + "\n"
	}

	return result
}
