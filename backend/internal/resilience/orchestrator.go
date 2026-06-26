package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ResilienceOrchestrator combines circuit breaker, retry, timeout, bulkhead, and rate limiting
type ResilienceOrchestrator struct {
	name                string
	circuitBreaker      *CircuitBreaker
	retryManager        *RetryManager
	timeoutManager      *TimeoutManager
	rateLimiter         *TokenBucketLimiter
	bulkhead            *BulkheadIsolation
	gracefulDegradation *GracefulDegradation
	mu                  sync.RWMutex
	metricsCollected    bool
}

// GracefulDegradation handles fallback strategies when system is degraded
type GracefulDegradation struct {
	enabled           bool
	fallbacks         map[string]func(context.Context, error) error
	degradationLevel  int // 0=normal, 1=partial, 2=degraded, 3=failed
	degradationReason string
	mu                sync.RWMutex
}

// NewResilienceOrchestrator creates a fully configured resilience orchestrator
func NewResilienceOrchestrator(
	name string,
	cbConfig CircuitBreakerConfig,
	retryPolicy RetryPolicy,
	timeoutConfig TimeoutConfig,
	rateLimitConfig RateLimitConfig,
	bulkheadConfig BulkheadConfig,
) *ResilienceOrchestrator {
	cbConfig.Name = name
	rateLimitConfig.Name = name
	bulkheadConfig.Name = name

	return &ResilienceOrchestrator{
		name:           name,
		circuitBreaker: NewCircuitBreaker(cbConfig),
		retryManager:   NewRetryManager(retryPolicy),
		timeoutManager: NewTimeoutManager(timeoutConfig),
		rateLimiter:    NewTokenBucketLimiter(rateLimitConfig),
		bulkhead:       NewBulkheadIsolation(bulkheadConfig),
		gracefulDegradation: &GracefulDegradation{
			enabled:   true,
			fallbacks: make(map[string]func(context.Context, error) error),
		},
	}
}

// Execute runs a function with all resilience patterns applied in order:
// 1. Rate limiting
// 2. Bulkhead isolation
// 3. Circuit breaker
// 4. Retry logic
// 5. Timeout management
// 6. Graceful degradation on failure
func (ro *ResilienceOrchestrator) Execute(ctx context.Context, fn func(context.Context) error, options ...ExecuteOption) error {
	opts := &executeOptions{}
	for _, opt := range options {
		opt(opts)
	}

	// Step 1: Rate limiting
	if !ro.rateLimiter.Allow() {
		return ro.handleDegradation(ctx, "rate_limit_exceeded", nil, opts)
	}

	// Step 2: Bulkhead isolation
	if err := ro.bulkhead.Execute(ctx, func(ctx context.Context) error {
		// Step 3: Circuit breaker
		cbErr := ro.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
			// Step 4: Retry logic with timeout
			retryErr := ro.retryManager.Execute(ctx, func(ctx context.Context) error {
				// Step 5: Timeout management
				return ro.timeoutManager.ExecuteWithTimeout(ctx, 0, fn)
			})

			if retryErr != nil && opts.fallbackName != "" {
				return ro.handleDegradation(ctx, opts.fallbackName, retryErr, opts)
			}

			return retryErr
		})

		if cbErr != nil && opts.fallbackName != "" {
			return ro.handleDegradation(ctx, opts.fallbackName, cbErr, opts)
		}

		return cbErr
	}); err != nil {
		return err
	}

	return nil
}

// ExecuteAsync runs a function asynchronously with resilience patterns
func (ro *ResilienceOrchestrator) ExecuteAsync(ctx context.Context, fn func(context.Context) error, resultChan chan error) {
	go func() {
		resultChan <- ro.Execute(ctx, fn)
	}()
}

// ExecuteWithFallback runs a function with a fallback strategy
func (ro *ResilienceOrchestrator) ExecuteWithFallback(
	ctx context.Context,
	fn func(context.Context) error,
	fallback func(context.Context, error) error,
) error {
	err := ro.Execute(ctx, fn)
	if err != nil {
		return fallback(ctx, err)
	}
	return nil
}

// handleDegradation handles degraded operations with fallback strategies
func (ro *ResilienceOrchestrator) handleDegradation(ctx context.Context, fallbackName string, originalErr error, opts *executeOptions) error {
	ro.gracefulDegradation.mu.RLock()
	fallback, exists := ro.gracefulDegradation.fallbacks[fallbackName]
	ro.gracefulDegradation.mu.RUnlock()

	if exists && fallback != nil {
		return fallback(ctx, originalErr)
	}

	// No fallback, return the original error
	if originalErr != nil {
		return fmt.Errorf("degradation: %s (original error: %w)", fallbackName, originalErr)
	}

	return fmt.Errorf("degradation: %s", fallbackName)
}

// RegisterFallback registers a fallback strategy for a named scenario
func (ro *ResilienceOrchestrator) RegisterFallback(name string, fallback func(context.Context, error) error) {
	ro.gracefulDegradation.mu.Lock()
	defer ro.gracefulDegradation.mu.Unlock()
	ro.gracefulDegradation.fallbacks[name] = fallback
}

// GetCircuitBreakerState returns the current circuit breaker state
func (ro *ResilienceOrchestrator) GetCircuitBreakerState() string {
	return ro.circuitBreaker.GetStateName()
}

// GetMetrics returns a comprehensive metrics report
func (ro *ResilienceOrchestrator) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"name":            ro.name,
		"circuit_breaker": ro.circuitBreaker.GetMetrics(),
		"retry":           ro.retryManager.GetMetrics(),
		"timeout":         ro.timeoutManager.GetMetrics(),
		"rate_limit":      ro.rateLimiter.GetMetrics(),
		"bulkhead":        ro.bulkhead.GetMetrics(),
	}
}

// ExportMetrics returns all metrics in Prometheus format
func (ro *ResilienceOrchestrator) ExportMetrics() string {
	result := ""
	result += ro.circuitBreaker.ExportMetrics()
	result += ro.retryManager.ExportMetrics(ro.name)
	result += ro.timeoutManager.ExportMetrics(ro.name)
	result += ro.rateLimiter.ExportMetrics()
	result += ro.bulkhead.ExportMetrics()

	return result
}

// GetDegradationLevel returns current degradation level (0=normal, 3=failed)
func (ro *ResilienceOrchestrator) GetDegradationLevel() int {
	ro.gracefulDegradation.mu.RLock()
	defer ro.gracefulDegradation.mu.RUnlock()
	return ro.gracefulDegradation.degradationLevel
}

// GetDegradationReason returns the reason for degradation
func (ro *ResilienceOrchestrator) GetDegradationReason() string {
	ro.gracefulDegradation.mu.RLock()
	defer ro.gracefulDegradation.mu.RUnlock()
	return ro.gracefulDegradation.degradationReason
}

// SetDegradationLevel sets the degradation level with a reason
func (ro *ResilienceOrchestrator) SetDegradationLevel(level int, reason string) {
	ro.gracefulDegradation.mu.Lock()
	defer ro.gracefulDegradation.mu.Unlock()
	ro.gracefulDegradation.degradationLevel = level
	ro.gracefulDegradation.degradationReason = reason
}

// HealthCheck returns a health status summary
func (ro *ResilienceOrchestrator) HealthCheck() HealthStatus {
	cbMetrics := ro.circuitBreaker.GetMetrics()
	rlMetrics := ro.rateLimiter.GetMetrics()
	bulkheadMetrics := ro.bulkhead.GetMetrics()

	status := HealthStatus{
		ServiceName:         ro.name,
		Timestamp:           time.Now(),
		CircuitBreakerState: ro.circuitBreaker.GetStateName(),
		CircuitBreakerOpen:  ro.circuitBreaker.GetState() == StateOpen,
		FailureRate:         cbMetrics.FailureRate,
		RateLimit: struct {
			DenyRate float64
		}{
			DenyRate: float64(rlMetrics.DeniedRequests) / float64(rlMetrics.TotalRequests+1),
		},
		Bulkhead: struct {
			RejectionRate     float64
			CurrentConcurrent int64
		}{
			RejectionRate:     float64(bulkheadMetrics.RejectedRequests) / float64(bulkheadMetrics.TotalRequests+1),
			CurrentConcurrent: bulkheadMetrics.CurrentConcurrent,
		},
		DegradationLevel:  ro.GetDegradationLevel(),
		DegradationReason: ro.GetDegradationReason(),
	}

	// Determine overall health
	if ro.circuitBreaker.GetState() == StateOpen {
		status.Health = "unhealthy"
	} else if cbMetrics.FailureRate > 0.1 || rlMetrics.DeniedRequests > 100 {
		status.Health = "degraded"
	} else {
		status.Health = "healthy"
	}

	return status
}

// HealthStatus represents the health of a resilient service
type HealthStatus struct {
	ServiceName         string
	Timestamp           time.Time
	Health              string // healthy, degraded, unhealthy
	CircuitBreakerState string
	CircuitBreakerOpen  bool
	FailureRate         float64
	RateLimit           struct {
		DenyRate float64
	}
	Bulkhead struct {
		RejectionRate     float64
		CurrentConcurrent int64
	}
	DegradationLevel  int
	DegradationReason string
}

// ExecuteOption is a functional option for Execute method
type ExecuteOption func(*executeOptions)

// executeOptions holds options for Execute method
type executeOptions struct {
	fallbackName string
}

// WithFallback sets the fallback strategy for degradation
func WithFallback(name string) ExecuteOption {
	return func(opts *executeOptions) {
		opts.fallbackName = name
	}
}
