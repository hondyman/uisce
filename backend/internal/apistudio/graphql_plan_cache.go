package apistudio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// CachedPlan is what's stored per plan key: the resolved SQL plus the
// field-masking metadata that was evaluated alongside it. The plan key
// (see GeneratePlanKey) includes the caller's role set, so a cache hit
// implies entitlement evaluation already ran and passed for that exact
// role set — it is NOT safe to cache SQL without the caller's roles baked
// into the key, since that would let a request from an unentitled caller
// be served another caller's already-authorized plan.
type CachedPlan struct {
	SQL          string              `json:"sql"`
	MaskedFields map[string][]string `json:"masked_fields,omitempty"`
}

// GetPlan retrieves a cached plan
func (c *GraphQLPlanCache) GetPlan(ctx context.Context, key string) (*CachedPlan, error) {
	if c.redisClient == nil {
		return nil, redis.Nil
	}
	raw, err := c.redisClient.Get(ctx, "gqlplan:"+key).Result()
	if err != nil {
		return nil, err
	}
	var plan CachedPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// SetPlan stores a plan in cache
func (c *GraphQLPlanCache) SetPlan(ctx context.Context, key string, plan CachedPlan) error {
	if c.redisClient == nil {
		return nil
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	// Default TTL 24 hours - invalidation handled by event logic (Phase 32 part 3)
	return c.redisClient.Set(ctx, "gqlplan:"+key, raw, 24*time.Hour).Err()
}

// GeneratePlanKey creates a deterministic hash for a query request.
//
// Filters must be included by value, not just by key: the resolved SQL bakes
// filter values in literally, so keying the cache on filter names alone lets
// a request with one filter value be served another request's cached SQL.
//
// Roles must be part of the key too: entitlement role gates and field
// masking are evaluated once per plan resolution, not on every cache hit,
// so two callers with different role sets must never share a cache entry.
func GeneratePlanKey(tenantID string, endpointID string, version int, measures []string, filters map[string]interface{}, roles []string) string {
	// Normalize inputs
	measures = append([]string(nil), measures...)
	sort.Strings(measures)

	filterKeys := make([]string, 0, len(filters))
	for k := range filters {
		filterKeys = append(filterKeys, k)
	}
	sort.Strings(filterKeys)

	filterParts := make([]string, 0, len(filterKeys))
	for _, k := range filterKeys {
		filterParts = append(filterParts, fmt.Sprintf("%s=%v", k, filters[k]))
	}

	roles = append([]string(nil), roles...)
	sort.Strings(roles)

	// Create composite string
	raw := fmt.Sprintf("%s|%s|%d|%s|%s|%s", tenantID, endpointID, version, measures, filterParts, roles)

	// Hash
	hasher := sha256.New()
	hasher.Write([]byte(raw))
	return hex.EncodeToString(hasher.Sum(nil))
}
