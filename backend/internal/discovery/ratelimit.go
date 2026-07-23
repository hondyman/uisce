package discovery

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements token bucket algorithm for rate limiting
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*TokenBucket
	maxRate  int           // requests per second
	maxBurst int           // burst capacity
	ttl      time.Duration // cleanup ttl for inactive buckets
}

// TokenBucket tracks rate for a single user/key
type TokenBucket struct {
	tokens     float64
	lastRefill time.Time
	rate       float64 // tokens per second
	capacity   float64
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter with specified config
func NewRateLimiter(requestsPerSecond, burstCapacity int, cleanupTTL time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*TokenBucket),
		maxRate:  requestsPerSecond,
		maxBurst: burstCapacity,
		ttl:      cleanupTTL,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request should be allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			tokens:     float64(rl.maxBurst),
			lastRefill: time.Now(),
			rate:       float64(rl.maxRate),
			capacity:   float64(rl.maxBurst),
		}
		rl.buckets[key] = bucket
	}
	rl.mu.Unlock()

	return bucket.Allow()
}

// Allow checks if token is available in the bucket
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	// Refill tokens based on elapsed time
	newTokens := tb.tokens + elapsed*tb.rate
	if newTokens > tb.capacity {
		tb.tokens = tb.capacity
	} else {
		tb.tokens = newTokens
	}

	if tb.tokens >= 1.0 {
		tb.tokens--
		return true
	}

	return false
}

// cleanup removes inactive buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, bucket := range rl.buckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastRefill) > rl.ttl {
				delete(rl.buckets, key)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware returns HTTP middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user identifier from header or IP
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				userID = r.RemoteAddr
			}

			// Check rate limit
			if !limiter.Allow(userID) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintf(w, `{"error": "Rate limit exceeded: %d req/sec per user"}`, limiter.maxRate)
				return
			}

			// Add rate limit headers to response
			w.Header().Set("X-Rate-Limit-Limit", fmt.Sprintf("%d", limiter.maxRate))
			w.Header().Set("X-Rate-Limit-Remaining", "available")
			w.Header().Set("X-Rate-Limit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

			next.ServeHTTP(w, r)
		})
	}
}
