package middleware

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters map[string]*rate.Limiter
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimiter) GetLimiter(key string, rps float64) *rate.Limiter {
	limiter, exists := rl.Get(key)
	if exists {
		return limiter
	}
	limiter = rate.NewLimiter(rate.Limit(rps), int(rps)*2)
	rl.limiters[key] = limiter
	return limiter
}

func (rl *RateLimiter) Get(key string) (*rate.Limiter, bool) {
	limiter, exists := rl.limiters[key]
	return limiter, exists
}

func RateLimitMiddleware(limiter *RateLimiter, devBypass DevBypassConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[RATE LIMIT] Path: %s", c.Request.URL.Path)

		if ShouldBypassAuth(c, devBypass) {
			p := c.Request.URL.Path
			if p == "/api/tenants" || strings.HasPrefix(p, "/api/tenants/") ||
				p == "/api/ip-whitelist" || strings.HasPrefix(p, "/api/ip-whitelist/") {
				c.Next()
				return
			}
		}

		var clientID string
		if tenantID, ok := c.Get("semlayer_tenant_id"); ok {
			if tid, ok := tenantID.(string); ok && tid != "" {
				clientID = "tenant:" + tid
			}
		}
		if clientID == "" {
			clientID = c.GetHeader("X-API-Key")
			if clientID == "" {
				clientID = c.ClientIP()
			}
		}

		rateLimiter := limiter.GetLimiter(clientID, 60.0)

		if !rateLimiter.Allow() {
			c.JSON(429, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": "60",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
