package cbo

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Ping(ctx context.Context) error
	Close() error
}

type redisClientWrapper struct {
	client *redis.Client
}

func NewRedisClient(redisURL string) (RedisClient, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}
	client := redis.NewClient(opt)
	return &redisClientWrapper{client: client}, nil
}

func (r *redisClientWrapper) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *redisClientWrapper) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *redisClientWrapper) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisClientWrapper) Close() error {
	return r.client.Close()
}

type noopRedisClient struct{}

func NewNoopRedisClient() RedisClient {
	return &noopRedisClient{}
}

func (n *noopRedisClient) Get(ctx context.Context, key string) (string, error) {
	return "", redis.Nil
}

func (n *noopRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

func (n *noopRedisClient) Ping(ctx context.Context) error {
	return nil
}

func (n *noopRedisClient) Close() error {
	return nil
}
