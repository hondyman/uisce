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
