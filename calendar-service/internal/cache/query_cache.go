package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// QueryCache implements a generic Redis cache for API requests and database queries
type QueryCache struct {
	client *Client
	logger *logrus.Entry
	prefix string
	ttl    time.Duration
}

func NewQueryCache(client *Client, ttl time.Duration, logger *logrus.Entry) *QueryCache {
	return &QueryCache{
		client: client,
		logger: logger.WithField("component", "query_cache"),
		prefix: "qcache:",
		ttl:    ttl,
	}
}

// Get retrieves a cached item
func (q *QueryCache) Get(ctx context.Context, key string, result interface{}) (bool, error) {
	fullKey := q.prefix + key
	data, err := q.client.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		return false, nil // Cache miss
	}

	if err := json.Unmarshal(data, result); err != nil {
		q.logger.WithError(err).Warn("Failed to unmarshal cached data")
		return false, err
	}
	return true, nil
}

// Set stores an item in the cache
func (q *QueryCache) Set(ctx context.Context, key string, data interface{}) error {
	fullKey := q.prefix + key
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	if err := q.client.client.Set(ctx, fullKey, bytes, q.ttl).Err(); err != nil {
		q.logger.WithError(err).Warn("Failed to set cache")
		return err
	}
	return nil
}

// Invalidate removes items by prefix
func (q *QueryCache) Invalidate(ctx context.Context, prefix string) error {
	// Use Redis SCAN to delete keys matching the prefix
	pattern := q.prefix + prefix + "*"

	iter := q.client.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := q.client.client.Del(ctx, iter.Val()).Err(); err != nil {
			q.logger.WithError(err).Warnf("Failed to delete key %s", iter.Val())
		}
	}
	if err := iter.Err(); err != nil {
		q.logger.WithError(err).Warn("Error while scanning for cache invalidation")
		return err
	}
	return nil
}
