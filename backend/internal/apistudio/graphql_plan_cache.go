package apistudio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
)

// GraphQLPlanCache handles caching of resolved SQL plans
type GraphQLPlanCache struct {
	redisClient *redis.Client
}

// NewGraphQLPlanCache creates a new plan cache
func NewGraphQLPlanCache(client *redis.Client) *GraphQLPlanCache {
	return &GraphQLPlanCache{redisClient: client}
}

// GetPlan retrieves a cached SQL plan
func (c *GraphQLPlanCache) GetPlan(ctx context.Context, key string) (string, error) {
	if c.redisClient == nil {
		return "", redis.Nil
	}
	return c.redisClient.Get(ctx, "gqlplan:"+key).Result()
}

// SetPlan stores a SQL plan in cache
func (c *GraphQLPlanCache) SetPlan(ctx context.Context, key string, plan string) error {
	if c.redisClient == nil {
		return nil
	}
	// Default TTL 24 hours - invalidation handled by event logic (Phase 32 part 3)
	return c.redisClient.Set(ctx, "gqlplan:"+key, plan, 24*time.Hour).Err()
}

// GeneratePlanKey creates a deterministic hash for a query request
func GeneratePlanKey(tenantID string, endpointID string, version int, measures []string, filterKeys []string) string {
	// Normalize inputs
	sort.Strings(measures)
	sort.Strings(filterKeys)

	// Create composite string
	raw := fmt.Sprintf("%s|%s|%d|%s|%s", tenantID, endpointID, version, measures, filterKeys)

	// Hash
	hasher := sha256.New()
	hasher.Write([]byte(raw))
	return hex.EncodeToString(hasher.Sum(nil))
}
