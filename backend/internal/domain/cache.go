package domain

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// DecisionCache provides caching for access control decisions
type DecisionCache interface {
	Get(ctx context.Context, key string) (*CachedDecision, bool)
	Set(ctx context.Context, key string, decision *CachedDecision) error
	Invalidate(ctx context.Context, pattern string) error
}

type CachedDecision struct {
	Allow   bool
	Reason  string
	Claims  []EffectiveClaim
	Matched []map[string]any
	Scopes  []string
	Expires time.Time
}

// RedisDecisionCache implements DecisionCache using Redis
type RedisDecisionCache struct {
	// Redis client would go here
	_ttl time.Duration
}

func (c *RedisDecisionCache) Get(ctx context.Context, key string) (*CachedDecision, bool) {
	// Redis GET implementation
	return nil, false // Placeholder
}

func (c *RedisDecisionCache) Set(ctx context.Context, key string, decision *CachedDecision) error {
	// Redis SET with TTL implementation
	return nil // Placeholder
}

func (c *RedisDecisionCache) Invalidate(ctx context.Context, pattern string) error {
	// Redis DEL pattern implementation
	return nil // Placeholder
}

// CachedEvaluator wraps an evaluator with caching
type CachedEvaluator struct {
	Evaluator Evaluator
	Cache     DecisionCache
	Logger    *slog.Logger
}

func (ce *CachedEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (bool, string, []EffectiveClaim, error) {
	cacheKey := fmt.Sprintf("eval:%s:%s:%s:%s", req.UserID, req.TenantID, req.AssetID, req.Action)

	// Check cache first
	if cached, found := ce.Cache.Get(ctx, cacheKey); found && time.Now().Before(cached.Expires) {
		ce.Logger.Info("cache hit for evaluation", "key", cacheKey)
		return cached.Allow, cached.Reason, cached.Claims, nil
	}

	// Cache miss - evaluate
	allow, reason, claims, err := ce.Evaluator.Evaluate(ctx, req)
	if err != nil {
		return false, "", nil, err
	}

	// Cache the result
	cached := &CachedDecision{
		Allow:   allow,
		Reason:  reason,
		Claims:  claims,
		Expires: time.Now().Add(5 * time.Minute), // 5 minute TTL
	}

	if err := ce.Cache.Set(ctx, cacheKey, cached); err != nil {
		ce.Logger.Warn("failed to cache decision", "error", err)
	}

	return allow, reason, claims, nil
}
