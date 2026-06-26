package apistudio

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// RateLimiter manages request quotas per tenant
type RateLimiter struct {
	redisClient *redis.Client
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{redisClient: client}
}

// Allow checks if the tenant has sufficient quota and consumes it
// Uses a simple fixed window or sliding window counter for MVP
// Key: ratelimit:{tenantID}:{minute_timestamp}
func (l *RateLimiter) Allow(ctx context.Context, tenantID string, cost int) (bool, error) {
	if l.redisClient == nil {
		return true, nil // Open by default if no redis
	}

	key := fmt.Sprintf("ratelimit:%s:%d", tenantID, time.Now().Unix()/60) // Per minute bucket

	// Increment
	count, err := l.redisClient.IncrBy(ctx, key, int64(cost)).Result()
	if err != nil {
		return false, err
	}

	// Set expiry if first
	if count == int64(cost) {
		l.redisClient.Expire(ctx, key, 2*time.Minute)
	}

	// Limit: Hardcoded 1000 per minute for MVP
	// In reality, this should be fetched from Tenant Config (AppDB or Cache)
	limit := int64(1000)

	if count > limit {
		return false, nil
	}

	return true, nil
}

// Middleware returns a middleware that enforces rate limits
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		// If no tenant context, we might skip or apply global limit.
		// For now, skip.
		if tenantID == "" {
			next.ServeHTTP(w, r)
			return
		}

		allowed, err := l.Allow(r.Context(), tenantID, 1)
		if err != nil {
			// Log error but fail safe (or closed)
			fmt.Printf("Rate limit error: %v\n", err)
		}

		if !allowed {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
