package resilience

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

// RetryPolicy defines the retry strategy
type RetryPolicy struct {
	MaxAttempts       int           // Maximum number of retry attempts
	InitialBackoff    time.Duration // Initial backoff duration
	MaxBackoff        time.Duration // Maximum backoff duration
	BackoffMultiplier float64       // Multiplier for exponential backoff
	JitterFraction    float64       // Jitter as fraction of backoff (0-1)
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.1,
	}
}

// RetryMetrics tracks retry attempt statistics
type RetryMetrics struct {
	TotalAttempts     int64         // Total retry attempts made
	SuccessfulRetries int64         // Attempts that eventually succeeded
	ExhaustedRetries  int64         // Attempts that exhausted max retries
	TotalBackoffTime  time.Duration // Total time spent backing off
	LastAttemptTime   time.Time     // Time of last attempt
}

// RetryManager handles retry logic with exponential backoff and jitter
type RetryManager struct {
	policy  RetryPolicy
	metrics RetryMetrics
}

// NewRetryManager creates a new retry manager with the given policy
func NewRetryManager(policy RetryPolicy) *RetryManager {
	return &RetryManager{
		policy: policy,
	}
}

// Execute runs the given function with retry logic
// Returns the result of the last attempt and any error
func (rm *RetryManager) Execute(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	attempt := 0

	for {
		attempt++
		atomic.AddInt64(&rm.metrics.TotalAttempts, 1)

		// Check context before attempting
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled before attempt %d: %w", attempt, err)
		}

		// Execute the function
		err := fn(ctx)
		rm.metrics.LastAttemptTime = time.Now()

		if err == nil {
			// Success
			atomic.AddInt64(&rm.metrics.SuccessfulRetries, 1)
			return nil
		}

		lastErr = err

		// Check if we've exhausted retries
		if attempt >= rm.policy.MaxAttempts {
			atomic.AddInt64(&rm.metrics.ExhaustedRetries, 1)
			return fmt.Errorf("max retries exceeded (%d attempts): %w", attempt, lastErr)
		}

		// Calculate backoff with exponential growth and jitter
		backoffDuration := rm.calculateBackoff(attempt)

		// Add to metrics
		rm.metrics.TotalBackoffTime += backoffDuration

		// Wait before retrying
		select {
		case <-time.After(backoffDuration):
			// Continue to next attempt
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during backoff (attempt %d): %w", attempt, ctx.Err())
		}
	}
}

// ExecuteWithFallback runs the function with retry logic, using fallback if all retries fail
func (rm *RetryManager) ExecuteWithFallback(
	ctx context.Context,
	fn func(context.Context) error,
	fallback func(context.Context, error) error,
) error {
	err := rm.Execute(ctx, fn)
	if err != nil {
		return fallback(ctx, err)
	}
	return nil
}

// calculateBackoff calculates the backoff duration for attempt n
// Formula: min(maxBackoff, initialBackoff * multiplier^(n-1)) + jitter
func (rm *RetryManager) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: initial * multiplier^(attempt-1)
	exponentialBackoff := time.Duration(float64(rm.policy.InitialBackoff) * math.Pow(rm.policy.BackoffMultiplier, float64(attempt-1)))

	// Cap at max backoff
	if exponentialBackoff > rm.policy.MaxBackoff {
		exponentialBackoff = rm.policy.MaxBackoff
	}

	// Add jitter: random value between 0 and (backoff * jitterFraction)
	jitterAmount := time.Duration(float64(exponentialBackoff) * rm.policy.JitterFraction * rand.Float64())

	return exponentialBackoff + jitterAmount
}

// GetMetrics returns a snapshot of retry metrics
func (rm *RetryManager) GetMetrics() RetryMetrics {
	return RetryMetrics{
		TotalAttempts:     atomic.LoadInt64(&rm.metrics.TotalAttempts),
		SuccessfulRetries: atomic.LoadInt64(&rm.metrics.SuccessfulRetries),
		ExhaustedRetries:  atomic.LoadInt64(&rm.metrics.ExhaustedRetries),
		TotalBackoffTime:  rm.metrics.TotalBackoffTime,
		LastAttemptTime:   rm.metrics.LastAttemptTime,
	}
}

// Reset resets the retry manager metrics
func (rm *RetryManager) Reset() {
	atomic.StoreInt64(&rm.metrics.TotalAttempts, 0)
	atomic.StoreInt64(&rm.metrics.SuccessfulRetries, 0)
	atomic.StoreInt64(&rm.metrics.ExhaustedRetries, 0)
	rm.metrics.TotalBackoffTime = 0
}

// ExportMetrics returns Prometheus format metrics
func (rm *RetryManager) ExportMetrics(serviceName string) string {
	metrics := rm.GetMetrics()

	return fmt.Sprintf(
		`retry_total_attempts{service="%s"} %d
retry_successful{service="%s"} %d
retry_exhausted{service="%s"} %d
retry_total_backoff_seconds{service="%s"} %f
`,
		serviceName, metrics.TotalAttempts,
		serviceName, metrics.SuccessfulRetries,
		serviceName, metrics.ExhaustedRetries,
		serviceName, metrics.TotalBackoffTime.Seconds(),
	)
}

// IsRetryableError determines if an error should be retried
// Override this function to customize retry logic
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Retry on context deadline or timeout
	if err == context.DeadlineExceeded || err == context.Canceled {
		return false
	}

	// In production, you might check error type or code
	// For now, retry all errors except context-related
	return true
}

// ConditionalRetry runs the function with conditional retry
// Continues retrying only if the error is deemed retryable
func (rm *RetryManager) ConditionalRetry(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	attempt := 0

	for {
		attempt++
		atomic.AddInt64(&rm.metrics.TotalAttempts, 1)

		if err := ctx.Err(); err != nil {
			return err
		}

		err := fn(ctx)
		rm.metrics.LastAttemptTime = time.Now()

		if err == nil {
			atomic.AddInt64(&rm.metrics.SuccessfulRetries, 1)
			return nil
		}

		// Check if error is retryable
		if !IsRetryableError(err) {
			return err
		}

		lastErr = err

		// Check if we've exhausted retries
		if attempt >= rm.policy.MaxAttempts {
			atomic.AddInt64(&rm.metrics.ExhaustedRetries, 1)
			return fmt.Errorf("max retries exceeded (%d attempts): %w", attempt, lastErr)
		}

		// Calculate backoff
		backoffDuration := rm.calculateBackoff(attempt)
		rm.metrics.TotalBackoffTime += backoffDuration

		// Wait before retrying
		select {
		case <-time.After(backoffDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
