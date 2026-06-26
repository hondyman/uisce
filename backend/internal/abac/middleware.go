package abac

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// PolicyCacheMiddleware provides caching for ABAC policy evaluations
type PolicyCacheMiddleware struct {
	redisClient *redis.Client
	ctx         context.Context
	ttl         time.Duration
	enabled     bool
}

// NewPolicyCacheMiddleware creates a new ABAC policy cache middleware
func NewPolicyCacheMiddleware(redisAddr, redisPassword string, redisDB int) *PolicyCacheMiddleware {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx := context.Background()

	// Test connection
	enabled := true
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed, ABAC policy caching disabled: %v", err)
		enabled = false
	} else {
		log.Printf("ABAC policy cache middleware initialized with Redis at %s", redisAddr)
	}

	return &PolicyCacheMiddleware{
		redisClient: client,
		ctx:         ctx,
		ttl:         5 * time.Minute, // 5-minute TTL for policies
		enabled:     enabled,
	}
}

// CachedPolicy represents a cached policy evaluation result
type CachedPolicy struct {
	TenantID     string    `json:"tenant_id"`
	DatasourceID string    `json:"datasource_id"`
	Subject      string    `json:"subject"`
	Action       string    `json:"action"`
	Resource     string    `json:"resource"`
	Decision     string    `json:"decision"`
	Reason       string    `json:"reason"`
	PolicyID     string    `json:"policy_id"`
	CachedAt     time.Time `json:"cached_at"`
}

// GetCachedPolicyDecision retrieves a cached policy evaluation
func (pcm *PolicyCacheMiddleware) GetCachedPolicyDecision(tenantID, datasourceID, subject, action, resource string) (*CachedPolicy, bool) {
	if !pcm.enabled || pcm.redisClient == nil {
		return nil, false
	}

	key := pcm.generatePolicyCacheKey(tenantID, datasourceID, subject, action, resource)

	val, err := pcm.redisClient.Get(pcm.ctx, key).Result()
	if err == redis.Nil {
		// Cache miss
		return nil, false
	}
	if err != nil {
		log.Printf("Redis GET error for key %s: %v", key, err)
		return nil, false
	}

	var cached CachedPolicy
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		log.Printf("Failed to unmarshal cached policy: %v", err)
		return nil, false
	}

	// Check if cache is still valid (not expired beyond TTL)
	if time.Since(cached.CachedAt) > pcm.ttl {
		// Expired, invalidate
		pcm.redisClient.Del(pcm.ctx, key)
		return nil, false
	}

	return &cached, true
}

// SetCachedPolicyDecision stores a policy evaluation in cache
func (pcm *PolicyCacheMiddleware) SetCachedPolicyDecision(policy *CachedPolicy) error {
	if !pcm.enabled || pcm.redisClient == nil {
		return nil // Silently skip if caching is disabled
	}

	policy.CachedAt = time.Now()
	key := pcm.generatePolicyCacheKey(policy.TenantID, policy.DatasourceID, policy.Subject, policy.Action, policy.Resource)

	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	if err := pcm.redisClient.Set(pcm.ctx, key, data, pcm.ttl).Err(); err != nil {
		log.Printf("Redis SET error for key %s: %v", key, err)
		return err
	}

	return nil
}

// InvalidatePolicyCache removes all cached policies (use when policy changes)
func (pcm *PolicyCacheMiddleware) InvalidatePolicyCache(tenantID, datasourceID string) error {
	if !pcm.enabled || pcm.redisClient == nil {
		return nil
	}

	pattern := fmt.Sprintf("abac_policy:%s:%s:*", tenantID, datasourceID)

	iter := pcm.redisClient.Scan(pcm.ctx, 0, pattern, 0).Iterator()
	deletedCount := 0

	for iter.Next(pcm.ctx) {
		key := iter.Val()
		if err := pcm.redisClient.Del(pcm.ctx, key).Err(); err != nil {
			log.Printf("Failed to delete key %s: %v", key, err)
		} else {
			deletedCount++
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan iteration error: %w", err)
	}

	log.Printf("Invalidated %d ABAC policy cache(s) for tenant %s datasource %s", deletedCount, tenantID, datasourceID)
	return nil
}

// InvalidateSpecificPolicy invalidates a cache entry for a specific evaluation
func (pcm *PolicyCacheMiddleware) InvalidateSpecificPolicy(tenantID, datasourceID, subject, action, resource string) error {
	if !pcm.enabled || pcm.redisClient == nil {
		return nil
	}

	key := pcm.generatePolicyCacheKey(tenantID, datasourceID, subject, action, resource)

	if err := pcm.redisClient.Del(pcm.ctx, key).Err(); err != nil {
		log.Printf("Redis DEL error for key %s: %v", key, err)
		return err
	}

	log.Printf("Invalidated specific ABAC policy cache: tenant=%s subject=%s action=%s resource=%s",
		tenantID, subject, action, resource)
	return nil
}

// Middleware returns a Chi-compatible middleware function that caches policy evaluations
func (pcm *PolicyCacheMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache for the /api/abac/evaluate endpoint
			if r.URL.Path != "/api/abac/evaluate" || r.Method != "POST" {
				next.ServeHTTP(w, r)
				return
			}

			claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
			datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

			if !pcm.enabled || tenantID == "" || datasourceID == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Parse request body to extract evaluation params
			var evalReq struct {
				Subject  string `json:"subject"`
				Action   string `json:"action"`
				Resource string `json:"resource"`
			}

			// Read body into buffer (since we might need to read it twice)
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				next.ServeHTTP(w, r) // Can't cache, proceed normally
				return
			}
			r.Body.Close()

			// Set body back for downstream handlers
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if err := json.Unmarshal(bodyBytes, &evalReq); err != nil {
				next.ServeHTTP(w, r) // Can't parse, proceed normally
				return
			}

			// Check cache
			if cached, found := pcm.GetCachedPolicyDecision(tenantID, datasourceID, evalReq.Subject, evalReq.Action, evalReq.Resource); found {
				// Cache hit! Return cached result
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"decision":   cached.Decision,
					"reason":     cached.Reason,
					"timestamp":  cached.CachedAt,
					"policy_id":  cached.PolicyID,
					"from_cache": true,
				})
				return
			}

			// Cache miss, proceed to handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetCacheStats returns cache statistics
func (pcm *PolicyCacheMiddleware) GetCacheStats() (map[string]interface{}, error) {
	if !pcm.enabled || pcm.redisClient == nil {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	// Count ABAC policy keys
	pattern := "abac_policy:*"
	count := 0

	iter := pcm.redisClient.Scan(pcm.ctx, 0, pattern, 0).Iterator()
	for iter.Next(pcm.ctx) {
		count++
	}

	if err := iter.Err(); err != nil {
		log.Printf("Scan error during stats: %v", err)
	}

	return map[string]interface{}{
		"enabled":            true,
		"policy_cache_count": count,
		"cache_ttl_minutes":  pcm.ttl.Minutes(),
	}, nil
}

// Close closes the Redis connection
func (pcm *PolicyCacheMiddleware) Close() error {
	if pcm.redisClient == nil {
		return nil
	}
	return pcm.redisClient.Close()
}

// generatePolicyCacheKey generates a Redis key for policy evaluation result
func (pcm *PolicyCacheMiddleware) generatePolicyCacheKey(tenantID, datasourceID, subject, action, resource string) string {
	return fmt.Sprintf("abac_policy:%s:%s:%s:%s:%s", tenantID, datasourceID, subject, action, resource)
}
