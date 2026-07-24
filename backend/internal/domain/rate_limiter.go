package domain

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter provides rate limiting for access control evaluations
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, time.Duration, error)
}

// TokenBucketRateLimiter implements token bucket algorithm
type TokenBucketRateLimiter struct {
	capacity   int64
	refillRate float64 // tokens per second
	tokens     map[string]*bucket
	mu         sync.RWMutex
}

type bucket struct {
	tokens     int64
	lastRefill time.Time
}

func NewTokenBucketRateLimiter(capacity int64, refillRate float64) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		capacity:   capacity,
		refillRate: refillRate,
		tokens:     make(map[string]*bucket),
	}
}

func (rl *TokenBucketRateLimiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.tokens[key]
	if !exists {
		b = &bucket{tokens: rl.capacity, lastRefill: now}
		rl.tokens[key] = b
	}

	// Refill tokens
	elapsed := now.Sub(b.lastRefill).Seconds()
	tokensToAdd := int64(elapsed * rl.refillRate)
	if tokensToAdd > 0 {
		b.tokens = min(b.tokens+tokensToAdd, rl.capacity)
		b.lastRefill = now
	}

	if b.tokens > 0 {
		b.tokens--
		return true, 0, nil
	}

	// Calculate wait time for next token
	waitTime := time.Duration((1 / rl.refillRate) * float64(time.Second))
	return false, waitTime, nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// SlidingWindowRateLimiter implements sliding window algorithm
type SlidingWindowRateLimiter struct {
	windowSize  time.Duration
	maxRequests int64
	requests    map[string][]time.Time
	mu          sync.RWMutex
}

func NewSlidingWindowRateLimiter(windowSize time.Duration, maxRequests int64) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requests:    make(map[string][]time.Time),
	}
}

func (rl *SlidingWindowRateLimiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	// Clean old requests
	requests := rl.requests[key]
	validRequests := make([]time.Time, 0, len(requests))
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	if int64(len(validRequests)) >= rl.maxRequests {
		// Calculate wait time until oldest request expires
		if len(validRequests) > 0 {
			waitTime := validRequests[0].Add(rl.windowSize).Sub(now)
			return false, waitTime, nil
		}
		return false, rl.windowSize, nil
	}

	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests
	return true, 0, nil
}

// RateLimitedEvaluator wraps an evaluator with rate limiting
type RateLimitedEvaluator struct {
	Evaluator   Evaluator
	RateLimiter RateLimiter
}

func (rle *RateLimitedEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (bool, string, []EffectiveClaim, error) {
	key := fmt.Sprintf("eval:%s:%s", req.UserID, req.TenantID)

	allowed, waitTime, err := rle.RateLimiter.Allow(ctx, key)
	if err != nil {
		return false, "rate limiter error", nil, err
	}

	if !allowed {
		return false, fmt.Sprintf("rate limit exceeded, try again in %v", waitTime), nil, nil
	}

	return rle.Evaluator.Evaluate(ctx, req)
}
