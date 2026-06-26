package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RevocationStore interface {
	Revoke(ctx context.Context, jti string, exp time.Time) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

// InMemoryRevocationStore is a simple in-memory map with TTL semantics for tests.
type InMemoryRevocationStore struct {
	m map[string]time.Time
}

func NewInMemoryRevocationStore() *InMemoryRevocationStore {
	return &InMemoryRevocationStore{m: make(map[string]time.Time)}
}

func (s *InMemoryRevocationStore) Revoke(ctx context.Context, jti string, exp time.Time) error {
	s.m[jti] = exp
	return nil
}

func (s *InMemoryRevocationStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if exp, ok := s.m[jti]; ok {
		if time.Now().Before(exp) {
			return true, nil
		}
		// expired entry; cleanup
		delete(s.m, jti)
		return false, nil
	}
	return false, nil
}

// RedisRevocationStore is a placeholder implementation using Redis (recommended for prod).
type RedisRevocationStore struct {
	rdb *redis.Client
}

func NewRedisRevocationStore(addr string) *RedisRevocationStore {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	return &RedisRevocationStore{rdb: rdb}
}

func (r *RedisRevocationStore) Revoke(ctx context.Context, jti string, exp time.Time) error {
	ttl := time.Until(exp)
	return r.rdb.Set(ctx, jti, "revoked", ttl).Err()
}

func (r *RedisRevocationStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	v, err := r.rdb.Get(ctx, jti).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == "revoked", nil
}
