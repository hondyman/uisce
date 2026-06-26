package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// SemanticViewSchema represents a cached semantic view definition
type SemanticViewSchema struct {
	ViewID      string                 `json:"view_id"`
	ViewName    string                 `json:"view_name"`
	TenantID    string                 `json:"tenant_id"`
	Fields      []SemanticField        `json:"fields"`
	Metadata    map[string]interface{} `json:"metadata"`
	Version     int                    `json:"version"`
	PublishedAt time.Time              `json:"published_at"`
}

// SemanticField represents a field in a semantic view
type SemanticField struct {
	FieldName   string `json:"field_name"`
	FieldType   string `json:"field_type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

// SemanticViewCache provides caching for semantic view schemas
type SemanticViewCache struct {
	redisClient *redis.Client
	ctx         context.Context
	ttl         time.Duration
}

// NewSemanticViewCache creates a new semantic view cache
func NewSemanticViewCache(redisAddr, redisPassword string, redisDB int) (*SemanticViewCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed, semantic view caching disabled: %v", err)
		return &SemanticViewCache{
			redisClient: client,
			ctx:         ctx,
			ttl:         24 * time.Hour,
		}, nil
	}

	log.Printf("Semantic view cache initialized with Redis at %s", redisAddr)

	return &SemanticViewCache{
		redisClient: client,
		ctx:         ctx,
		ttl:         24 * time.Hour, // 24-hour TTL for semantic views
	}, nil
}

// GetSemanticView retrieves a semantic view from cache
func (svc *SemanticViewCache) GetSemanticView(tenantID, viewID string) (*SemanticViewSchema, error) {
	if svc.redisClient == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	key := svc.generateCacheKey(tenantID, viewID)

	val, err := svc.redisClient.Get(svc.ctx, key).Result()
	if err == redis.Nil {
		// Cache miss
		return nil, nil
	}
	if err != nil {
		log.Printf("Redis GET error for key %s: %v", key, err)
		return nil, err
	}

	var schema SemanticViewSchema
	if err := json.Unmarshal([]byte(val), &schema); err != nil {
		log.Printf("Failed to unmarshal semantic view from cache: %v", err)
		return nil, err
	}

	return &schema, nil
}

// SetSemanticView stores a semantic view in cache
func (svc *SemanticViewCache) SetSemanticView(schema *SemanticViewSchema) error {
	if svc.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := svc.generateCacheKey(schema.TenantID, schema.ViewID)

	data, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal semantic view: %w", err)
	}

	if err := svc.redisClient.Set(svc.ctx, key, data, svc.ttl).Err(); err != nil {
		log.Printf("Redis SET error for key %s: %v", key, err)
		return err
	}

	log.Printf("Cached semantic view: tenant=%s view=%s", schema.TenantID, schema.ViewID)
	return nil
}

// InvalidateSemanticView removes a semantic view from cache
func (svc *SemanticViewCache) InvalidateSemanticView(tenantID, viewID string) error {
	if svc.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := svc.generateCacheKey(tenantID, viewID)

	if err := svc.redisClient.Del(svc.ctx, key).Err(); err != nil {
		log.Printf("Redis DEL error for key %s: %v", key, err)
		return err
	}

	log.Printf("Invalidated semantic view cache: tenant=%s view=%s", tenantID, viewID)
	return nil
}

// InvalidateTenantViews removes all semantic views for a tenant
func (svc *SemanticViewCache) InvalidateTenantViews(tenantID string) error {
	if svc.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	pattern := fmt.Sprintf("semantic_view:%s:*", tenantID)

	iter := svc.redisClient.Scan(svc.ctx, 0, pattern, 0).Iterator()
	deletedCount := 0

	for iter.Next(svc.ctx) {
		key := iter.Val()
		if err := svc.redisClient.Del(svc.ctx, key).Err(); err != nil {
			log.Printf("Failed to delete key %s: %v", key, err)
		} else {
			deletedCount++
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan iteration error: %w", err)
	}

	log.Printf("Invalidated %d semantic view(s) for tenant %s", deletedCount, tenantID)
	return nil
}

// GetMultipleViews retrieves multiple semantic views in a single operation
func (svc *SemanticViewCache) GetMultipleViews(tenantID string, viewIDs []string) (map[string]*SemanticViewSchema, error) {
	if svc.redisClient == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	if len(viewIDs) == 0 {
		return make(map[string]*SemanticViewSchema), nil
	}

	// Build keys for pipeline
	keys := make([]string, len(viewIDs))
	for i, viewID := range viewIDs {
		keys[i] = svc.generateCacheKey(tenantID, viewID)
	}

	// Use pipeline for batch GET
	pipe := svc.redisClient.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(svc.ctx, key)
	}

	if _, err := pipe.Exec(svc.ctx); err != nil && err != redis.Nil {
		log.Printf("Pipeline exec error: %v", err)
	}

	// Parse results
	results := make(map[string]*SemanticViewSchema)

	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			// Cache miss for this view
			continue
		}
		if err != nil {
			log.Printf("Error retrieving view %s: %v", viewIDs[i], err)
			continue
		}

		var schema SemanticViewSchema
		if err := json.Unmarshal([]byte(val), &schema); err != nil {
			log.Printf("Failed to unmarshal view %s: %v", viewIDs[i], err)
			continue
		}

		results[viewIDs[i]] = &schema
	}

	log.Printf("Retrieved %d/%d semantic views from cache for tenant %s", len(results), len(viewIDs), tenantID)
	return results, nil
}

// GetCacheStats returns cache statistics
func (svc *SemanticViewCache) GetCacheStats() (map[string]interface{}, error) {
	if svc.redisClient == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	info, err := svc.redisClient.Info(svc.ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis stats: %w", err)
	}

	// Count semantic view keys
	pattern := "semantic_view:*"
	count := 0

	iter := svc.redisClient.Scan(svc.ctx, 0, pattern, 0).Iterator()
	for iter.Next(svc.ctx) {
		count++
	}

	if err := iter.Err(); err != nil {
		log.Printf("Scan error during stats: %v", err)
	}

	return map[string]interface{}{
		"redis_info":          info,
		"semantic_view_count": count,
		"cache_ttl_hours":     svc.ttl.Hours(),
	}, nil
}

// Close closes the Redis connection
func (svc *SemanticViewCache) Close() error {
	if svc.redisClient == nil {
		return nil
	}
	return svc.redisClient.Close()
}

// generateCacheKey generates a Redis key for a semantic view
func (svc *SemanticViewCache) generateCacheKey(tenantID, viewID string) string {
	return fmt.Sprintf("semantic_view:%s:%s", tenantID, viewID)
}
