package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// SemanticViewSchema represents a cached semantic view schema
type SemanticViewSchema struct {
	ViewID      string                 `json:"view_id"`
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Columns     []SemanticViewColumn   `json:"columns"`
	Metadata    map[string]interface{} `json:"metadata"`
	CachedAt    time.Time              `json:"cached_at"`
}

// SemanticViewColumn represents a column in a semantic view
type SemanticViewColumn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	IsMeasure   bool   `json:"is_measure"`
	IsDimension bool   `json:"is_dimension"`
}

// SemanticViewCache provides Redis-backed caching for semantic view schemas
type SemanticViewCache struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewSemanticViewCache creates a new SemanticViewCache
func NewSemanticViewCache(rdb *redis.Client, ttl time.Duration) *SemanticViewCache {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &SemanticViewCache{rdb: rdb, ttl: ttl}
}

// GetSemanticView retrieves a semantic view schema from Redis
func (c *SemanticViewCache) GetSemanticView(tenantID, viewID string) (*SemanticViewSchema, error) {
	if c == nil || c.rdb == nil {
		return nil, nil
	}
	key := fmt.Sprintf("semantic_view:%s:%s", tenantID, viewID)
	val, err := c.rdb.Get(context.Background(), key).Bytes()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var schema SemanticViewSchema
	if err := json.Unmarshal(val, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// SetSemanticView stores a semantic view schema in Redis
func (c *SemanticViewCache) SetSemanticView(tenantID, viewID string, schema *SemanticViewSchema) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	key := fmt.Sprintf("semantic_view:%s:%s", tenantID, viewID)
	val, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	return c.rdb.Set(context.Background(), key, val, c.ttl).Err()
}

// ListSemanticViews returns all cached semantic view schemas for a tenant
func (c *SemanticViewCache) ListSemanticViews(tenantID string) ([]*SemanticViewSchema, error) {
	if c == nil || c.rdb == nil {
		return []*SemanticViewSchema{}, nil
	}
	pattern := fmt.Sprintf("semantic_view:%s:*", tenantID)
	keys, err := c.rdb.Keys(context.Background(), pattern).Result()
	if err != nil {
		return nil, err
	}
	var schemas []*SemanticViewSchema
	for _, key := range keys {
		val, err := c.rdb.Get(context.Background(), key).Bytes()
		if err != nil {
			continue
		}
		var schema SemanticViewSchema
		if err := json.Unmarshal(val, &schema); err == nil {
			schemas = append(schemas, &schema)
		}
	}
	return schemas, nil
}

// InvalidateTenantViews removes all cached semantic view schemas for a tenant
func (c *SemanticViewCache) InvalidateTenantViews(tenantID string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	pattern := fmt.Sprintf("semantic_view:%s:*", tenantID)
	keys, err := c.rdb.Keys(context.Background(), pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.rdb.Del(context.Background(), keys...).Err()
	}
	return nil
}

// GetCacheStats returns basic stats about the semantic view cache
func (c *SemanticViewCache) GetCacheStats() (map[string]interface{}, error) {
	if c == nil || c.rdb == nil {
		return map[string]interface{}{"status": "disabled"}, nil
	}
	keys, err := c.rdb.Keys(context.Background(), "semantic_view:*").Result()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"total_keys":  len(keys),
		"ttl_seconds": c.ttl.Seconds(),
		"status":      "active",
	}, nil
}

type SemanticCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewSemanticCache(rdb *redis.Client, ttl time.Duration) *SemanticCache {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &SemanticCache{
		rdb: rdb,
		ttl: ttl,
	}
}

func ComputeASTHash(astPayload interface{}) (string, error) {
	bytes, err := json.Marshal(astPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AST for hashing: %w", err)
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}

func (c *SemanticCache) Get(ctx context.Context, astHash string, target interface{}) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, nil
	}
	val, err := c.rdb.Get(ctx, "semantic_ast:"+astHash).Bytes()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if err := json.Unmarshal(val, target); err != nil {
		return false, err
	}
	return true, nil
}

func (c *SemanticCache) Set(ctx context.Context, astHash string, result interface{}) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	val, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, "semantic_ast:"+astHash, val, c.ttl).Err()
}
