package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// ResolvedCalendar represents a fully resolved profile calendar
type ResolvedCalendar struct {
	TenantID    string      `json:"tenant_id"`
	Region      string      `json:"region"`
	ProfileName string      `json:"profile_name"`
	Holidays    []time.Time `json:"holidays"`
	Blackouts   []TimeRange `json:"blackouts"`
	Timezone    string      `json:"timezone"`
	ResolvedAt  time.Time   `json:"resolved_at"`
	Version     string      `json:"version"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Client manages Redis cache operations for calendar resolution
type Client struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
	logger *logrus.Entry

	// Metrics
	hits   *prometheus.CounterVec
	misses *prometheus.CounterVec
}

var (
	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_cache_hits_total",
			Help: "Total cache hits for resolved calendars",
		},
		[]string{"tenant_id", "region"},
	)
	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_cache_misses_total",
			Help: "Total cache misses for resolved calendars",
		},
		[]string{"tenant_id", "region"},
	)
)

// NewClient creates a new Redis cache client
func NewClient(addr, prefix string, ttl time.Duration, logger *logrus.Entry) *Client {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     20,
		MinIdleConns: 5,
		MaxRetries:   3,
	})

	return &Client{
		client: client,
		prefix: prefix,
		ttl:    ttl,
		logger: logger.WithField("component", "cache"),
		hits:   cacheHits,
		misses: cacheMisses,
	}
}

// Key generates a region-aware cache key
func (c *Client) Key(tenantID, region, profileName string) string {
	return fmt.Sprintf("%s:resolved:%s:%s:%s", c.prefix, tenantID, region, profileName)
}

// Get retrieves a resolved calendar from cache
func (c *Client) Get(ctx context.Context, tenantID, region, profileName string) (*ResolvedCalendar, error) {
	key := c.Key(tenantID, region, profileName)
	data, err := c.client.Get(ctx, key).Bytes()

	if err == redis.Nil {
		c.misses.WithLabelValues(tenantID, region).Inc()
		return nil, nil // Cache miss, not an error
	}
	if err != nil {
		c.logger.WithError(err).Warn("Redis get error")
		return nil, err
	}

	var rc ResolvedCalendar
	if err := json.Unmarshal(data, &rc); err != nil {
		c.logger.WithError(err).Warn("Cache unmarshal error")
		_ = c.client.Del(ctx, key) // Delete corrupted entry
		return nil, err
	}

	c.hits.WithLabelValues(tenantID, region).Inc()
	return &rc, nil
}

// Set stores a resolved calendar in cache (synchronous)
func (c *Client) Set(ctx context.Context, tenantID, region, profileName string, rc *ResolvedCalendar) error {
	key := c.Key(tenantID, region, profileName)
	data, err := json.Marshal(rc)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// SetAsync stores in cache without blocking the caller
func (c *Client) SetAsync(ctx context.Context, tenantID, region, profileName string, rc *ResolvedCalendar) {
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := c.Set(bgCtx, tenantID, region, profileName, rc); err != nil {
			c.logger.WithError(err).Debug("Async cache set failed")
		}
	}()
}

// Invalidate deletes a specific cache entry
func (c *Client) Invalidate(ctx context.Context, tenantID, region, profileName string) error {
	key := c.Key(tenantID, region, profileName)
	return c.client.Del(ctx, key).Err()
}

// InvalidateTenantProfiles invalidates all profiles for a tenant and region
func (c *Client) InvalidateTenantProfiles(ctx context.Context, tenantID, region string, profiles []string) error {
	for _, p := range profiles {
		_ = c.Invalidate(ctx, tenantID, region, p)
	}
	return c.PublishInvalidation(ctx, tenantID, region)
}

// PublishInvalidation sends Pub/Sub message for cross-instance cache sync
func (c *Client) PublishInvalidation(ctx context.Context, tenantID, region string) error {
	msg, _ := json.Marshal(map[string]string{"tenant_id": tenantID, "region": region})
	return c.client.Publish(ctx, fmt.Sprintf("%s:invalidations", c.prefix), msg).Err()
}

// SubscribeToInvalidations listens for cross-instance invalidation events
func (c *Client) SubscribeToInvalidations(ctx context.Context, invalidateFunc func(string, string)) {
	ch := c.client.Subscribe(ctx, fmt.Sprintf("%s:invalidations", c.prefix)).Channel()
	go func() {
		for msg := range ch {
			var data map[string]string
			if err := json.Unmarshal([]byte(msg.Payload), &data); err == nil {
				invalidateFunc(data["tenant_id"], data["region"])
			}
		}
	}()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Ping verifies connection to Redis
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Stats returns cache hit/miss statistics
func (c *Client) Stats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"connection":  "healthy",
		"ttl_seconds": int64(c.ttl.Seconds()),
		"prefix":      c.prefix,
	}
}
