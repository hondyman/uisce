package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// OAuthRateLimitMiddleware adds rate limiting to OAuth endpoints
func OAuthRateLimitMiddleware(next http.Handler) http.Handler {
	// 5 requests per minute per IP for OAuth endpoints
	// Note: In visual studio code environment, IP tracking might be tricky, limiting globally for now or using a simpler approach
	// For production, this should be keyed by IP or UserID if available.
	// This is a simple per-instance limiter.
	limiter := rate.NewLimiter(rate.Every(12*time.Second), 5)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "rate_limit_exceeded",
				"message": "Too many OAuth requests. Please retry after 60 seconds.",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
