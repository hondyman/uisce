package validationsdk

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// RetryPolicy: exponential backoff with jitter. Both eval endpoints are
// read-only, so retries are always safe.
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultRetry() RetryPolicy {
	return RetryPolicy{MaxAttempts: 4, BaseDelay: 100 * time.Millisecond, MaxDelay: 2 * time.Second}
}

func (p RetryPolicy) backoff(attempt int) time.Duration {
	d := p.BaseDelay << attempt // 100ms, 200ms, 400ms...
	if d > p.MaxDelay {
		d = p.MaxDelay
	}
	return d + time.Duration(rand.Int63n(int64(d/4)+1)) // jitter
}

// CircuitBreaker: consecutive-failure breaker with half-open probing.
// Protects callers from hammering a degraded engine and failing fast instead.
type CircuitBreaker struct {
	mu               sync.Mutex
	failureThreshold int
	cooldown         time.Duration

	failures    int
	openedAt    time.Time
	halfOpenTry bool
}

func DefaultBreaker() *CircuitBreaker {
	return &CircuitBreaker{failureThreshold: 5, cooldown: 30 * time.Second}
}

var ErrCircuitOpen = errors.New("validation-sdk: circuit breaker open")

func (b *CircuitBreaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failures < b.failureThreshold {
		return nil
	}
	if time.Since(b.openedAt) < b.cooldown {
		return ErrCircuitOpen
	}
	// Half-open: allow one probe.
	if b.halfOpenTry {
		return ErrCircuitOpen
	}
	b.halfOpenTry = true
	return nil
}

func (b *CircuitBreaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.halfOpenTry = false
}

func (b *CircuitBreaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.halfOpenTry = false
	b.failures++
	if b.failures == b.failureThreshold {
		b.openedAt = time.Now()
	}
}

func withRetry(ctx context.Context, p RetryPolicy, br *CircuitBreaker, fn func() error) error {
	if br != nil {
		if err := br.Allow(); err != nil {
			return err
		}
	}
	var lastErr error
	for attempt := 0; attempt < p.MaxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(p.backoff(attempt - 1)):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		lastErr = fn()
		if lastErr == nil {
			if br != nil {
				br.RecordSuccess()
			}
			return nil
		}
		var apiErr *APIError
		// Retry only on transport/5xx; 4xx are deterministic client errors.
		if !errors.As(lastErr, &apiErr) || apiErr.StatusCode < 500 {
			break
		}
	}
	if br != nil {
		br.RecordFailure()
	}
	return lastErr
}

// APIError is a non-2xx response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("uisce validation API: status %d: %s", e.StatusCode, truncate(e.Body, 512))
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
