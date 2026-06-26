package cache

import (
	"context"
)

// RedisPool provides backward-compatible Redis wrapper if needed
// Actual Client implementation is in calendar_cache.go
type RedisPool struct {
	*Client
}

// NewRedisClient creates a new Redis pool (deprecated, use NewClient instead)
func NewRedisClient(addr string) *Client {
	// Stub for backward compatibility - prefer NewClient
	return nil
}

// RedisClientInterface defines common Redis operations
type RedisClientInterface interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Del(ctx context.Context, key string) error
}
