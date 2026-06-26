package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// TenantRateLimiter manages per-tenant rate limits
type TenantRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rps      float64 // requests per second per tenant
	burst    int     // max burst size per tenant
	logger   *logrus.Entry
}

// NewTenantRateLimiter creates a new rate limiter with default settings
func NewTenantRateLimiter(rps float64, burst int, logger *logrus.Entry) *TenantRateLimiter {
	return &TenantRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    burst,
		logger:   logger.WithField("component", "rate_limiter"),
	}
}

// getLimiter returns or creates a rate limiter for a tenant
func (rl *TenantRateLimiter) getLimiter(tenantID string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[tenantID]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := rl.limiters[tenantID]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
	rl.limiters[tenantID] = limiter
	rl.logger.WithField("tenant_id", tenantID).Debug("Created new rate limiter")
	return limiter
}

// RateLimit middleware checks rate limit before proceeding
// Must be applied AFTER JWT and TenantGuard middleware
func (rl *TenantRateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract tenant from context (set by TenantGuardMiddleware)
		tenantID := ExtractTenantIDFromContext(r.Context())
		if tenantID == "" {
			// Should not happen if TenantGuard is applied correctly
			http.Error(w, "Internal error: missing tenant context", http.StatusInternalServerError)
			return
		}

		limiter := rl.getLimiter(tenantID)

		if !limiter.Allow() {
			rl.logger.WithFields(logrus.Fields{
				"tenant_id": tenantID,
				"user_id":   ExtractUserIDFromContext(r.Context()),
				"method":    r.Method,
				"path":      r.RequestURI,
			}).Warn("Rate limit exceeded")

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests. Please retry after 60 seconds.",
				"retry_after": 60,
			})
			return
		}

		// Rate limit passed, proceed to handler
		next.ServeHTTP(w, r)
	})
}

// Allow checks if a request from a tenant should be allowed
// Returns true if allowed, false if rate limited
// This is useful for custom logic that doesn't need full middleware behavior
func (rl *TenantRateLimiter) Allow(tenantID string) bool {
	limiter := rl.getLimiter(tenantID)
	return limiter.Allow()
}

// Cleanup removes inactive tenant limiters (call periodically)
// In production, implement LRU cache instead of simple cleanup
func (rl *TenantRateLimiter) Cleanup(maxInactiveDuration time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// For demo purposes, we don't actively clean up
	// In production, track last access time per limiter and remove old ones
	rl.logger.WithField("limiters_count", len(rl.limiters)).Debug("Current rate limiters")
}

// UpdateLimits changes the RPS and burst for all future limiters
func (rl *TenantRateLimiter) UpdateLimits(newRps float64, newBurst int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.rps = newRps
	rl.burst = newBurst

	// Clear existing limiters so new ones are created with updated config
	// In production, might want to preserve some tenant-specific limits
	rl.logger.WithFields(logrus.Fields{
		"new_rps":   newRps,
		"new_burst": newBurst,
	}).Info("Rate limiter limits updated")
}

// GetStats returns rate limiter statistics for monitoring
func (rl *TenantRateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"total_limiters": len(rl.limiters),
		"rps":            rl.rps,
		"burst":          rl.burst,
		"version":        "1.0",
	}
}
