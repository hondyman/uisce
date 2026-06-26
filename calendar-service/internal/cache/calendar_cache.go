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

// ResolvedCalendar represents a cached, fully-resolved calendar after merging all profile calendars
type ResolvedCalendar struct {
	TenantID    string      `json:"tenant_id"`
	Region      string      `json:"region"` // Global Distribution Support
	ProfileName string      `json:"profile_name"`
	Holidays    []time.Time `json:"holidays"`
	Blackouts   []TimeRange `json:"blackouts"`
	Timezone    string      `json:"timezone"`
	ResolvedAt  time.Time   `json:"resolved_at"`
	Version     string      `json:"version"` // Hash for invalidation tracking
}

// TimeRange represents a blocked/unavailable time period
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Client manages Redis caching for resolved calendars
// Provides cache-aside pattern with graceful fallback and pub/sub invalidation
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
		prometheus.CounterOpts{Name: "calendar_cache_hits_total", Help: "Cache hits"},
		[]string{"tenant_id", "region"},
	)
	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{Name: "calendar_cache_misses_total", Help: "Cache misses"},
		[]string{"tenant_id", "region"},
	)
)

// NewClient creates a new cache instance connected to Redis
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

// Key generates region-aware cache key
func (c *Client) Key(tenantID, region, profileName string) string {
	return fmt.Sprintf("%s:resolved:%s:%s:%s", c.prefix, tenantID, region, profileName)
}

// Get retrieves a resolved calendar from cache
// Returns (nil, nil) on cache miss - caller should compute and Set()
// Returns (nil, error) on Redis error - caller should compute from DB
func (c *Client) Get(ctx context.Context, tenantID, region, profileName string) (*ResolvedCalendar, error) {
	key := c.Key(tenantID, region, profileName)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		c.misses.WithLabelValues(tenantID, region).Inc()
		return nil, nil
	}
	if err != nil {
		c.logger.WithError(err).Warn("Redis get error")
		return nil, err
	}

	var rc ResolvedCalendar
	if err := json.Unmarshal(data, &rc); err != nil {
		c.logger.WithError(err).Warn("Cache unmarshal error")
		_ = c.client.Del(ctx, key) // Delete corrupted
		return nil, err
	}

	c.hits.WithLabelValues(tenantID, region).Inc()
	return &rc, nil
}

// Set stores in cache (async safe)
func (c *Client) Set(ctx context.Context, tenantID, region, profileName string, rc *ResolvedCalendar) error {
	key := c.Key(tenantID, region, profileName)
	data, err := json.Marshal(rc)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// SetAsync stores without blocking
func (c *Client) SetAsync(ctx context.Context, tenantID, region, profileName string, rc *ResolvedCalendar) {
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := c.Set(bgCtx, tenantID, region, profileName, rc); err != nil {
			c.logger.WithError(err).Debug("Async cache set failed")
		}
	}()
}

// Invalidate deletes key
func (c *Client) Invalidate(ctx context.Context, tenantID, region, profileName string) error {
	key := c.Key(tenantID, region, profileName)
	return c.client.Del(ctx, key).Err()
}

// InvalidateTenantProfiles invalidates all profiles for a tenant (used by CDC)
func (c *Client) InvalidateTenantProfiles(ctx context.Context, tenantID, region string, profiles []string) {
	for _, p := range profiles {
		_ = c.Invalidate(ctx, tenantID, region, p)
	}
	// Publish for cross-instance invalidation
	_ = c.PublishInvalidation(ctx, tenantID, region)
}

// PublishInvalidation sends Pub/Sub message
func (c *Client) PublishInvalidation(ctx context.Context, tenantID, region string) error {
	msg, _ := json.Marshal(map[string]string{"tenant_id": tenantID, "region": region})
	return c.client.Publish(ctx, fmt.Sprintf("%s:invalidations", c.prefix), msg).Err()
}

// SubscribeToInvalidations listens for cross-instance invalidation
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

// GetString retrieves a simple string value from cache (for profile name mappings)
func (c *Client) GetString(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		c.logger.WithError(err).WithField("key", key).Warn("Redis get string error")
		return "", err
	}
	return val, nil
}

// SetString stores a simple string value in cache (for profile name mappings)
func (c *Client) SetString(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// SetStringAsync stores a simple string value without blocking
func (c *Client) SetStringAsync(ctx context.Context, key string, value string, ttl time.Duration) {
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := c.SetString(bgCtx, key, value, ttl); err != nil {
			c.logger.WithError(err).WithField("key", key).Debug("Async cache set string failed")
		}
	}()
}

// DelString deletes a simple string value from cache
func (c *Client) DelString(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Ping verifies Redis connectivity
func (c *Client) Ping(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return c.client.Ping(ctx).Err()
}

// Close cleans up Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// NewCalendarCache compatibility function
func NewCalendarCache(addr, prefix string, ttl time.Duration, logger *logrus.Entry) *Client {
	return NewClient(addr, prefix, ttl, logger)
}

// CalendarCache compatibility type
type CalendarCache = Client
