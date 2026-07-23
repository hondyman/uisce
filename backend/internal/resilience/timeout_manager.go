package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// TimeoutConfig holds timeout management configuration
type TimeoutConfig struct {
	DefaultTimeout      time.Duration // Default timeout for operations
	MaxTimeout          time.Duration // Maximum allowed timeout
	DeadlinePropagation bool          // Propagate deadline to child contexts
	GracefulShutdown    time.Duration // Time for graceful shutdown
}

// TimeoutMetrics tracks timeout statistics
type TimeoutMetrics struct {
	TotalOperations  int64         // Total operations executed
	TimedOutCount    int64         // Operations that timed out
	CompletedCount   int64         // Operations that completed in time
	TotalTimeSpent   time.Duration // Total time spent in operations
	MaxOperationTime time.Duration // Maximum operation duration
	MinOperationTime time.Duration // Minimum operation duration
}

// TimeoutManager enforces timeout constraints with deadline propagation
type TimeoutManager struct {
	config           TimeoutConfig
	metrics          TimeoutMetrics
	activeContexts   map[string]*contextInfo
	activeContextsMu sync.RWMutex
	minOperationTime time.Duration
}

// contextInfo tracks context metadata
type contextInfo struct {
	id        string
	startTime time.Time
	deadline  time.Time
	cancelled bool
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager(config TimeoutConfig) *TimeoutManager {
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}
	if config.MaxTimeout == 0 {
		config.MaxTimeout = 5 * time.Minute
	}
	if config.GracefulShutdown == 0 {
		config.GracefulShutdown = 10 * time.Second
	}

	return &TimeoutManager{
		config:           config,
		activeContexts:   make(map[string]*contextInfo),
		minOperationTime: time.Duration(1<<63 - 1), // Max int64 as initial value
	}
}

// WithTimeout wraps a context with timeout and propagates deadline
func (tm *TimeoutManager) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	// Cap timeout at configured maximum
	if timeout > tm.config.MaxTimeout {
		timeout = tm.config.MaxTimeout
	}

	// If no timeout specified, use default
	if timeout == 0 {
		timeout = tm.config.DefaultTimeout
	}

	// Check if parent context already has a deadline
	if parentDeadline, ok := ctx.Deadline(); ok {
		now := time.Now()
		parentTimeLeft := parentDeadline.Sub(now)

		// Use the more restrictive deadline
		if parentTimeLeft < timeout {
			timeout = parentTimeLeft
		}
	}

	// Create timeout context
	timedCtx, cancel := context.WithTimeout(ctx, timeout)

	// Track the context
	contextID := fmt.Sprintf("ctx-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&counterID, 1))
	tm.activeContextsMu.Lock()
	tm.activeContexts[contextID] = &contextInfo{
		id:        contextID,
		startTime: time.Now(),
		deadline:  time.Now().Add(timeout),
	}
	tm.activeContextsMu.Unlock()

	// Wrap cancel to clean up tracking
	wrappedCancel := func() {
		tm.activeContextsMu.Lock()
		if info, exists := tm.activeContexts[contextID]; exists {
			info.cancelled = true
			operationTime := time.Since(info.startTime)
			tm.recordOperationTime(operationTime)
		}
		delete(tm.activeContexts, contextID)
		tm.activeContextsMu.Unlock()
		cancel()
	}

	return timedCtx, wrappedCancel
}

// ExecuteWithTimeout runs a function with timeout constraint
func (tm *TimeoutManager) ExecuteWithTimeout(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	timedCtx, cancel := tm.WithTimeout(ctx, timeout)
	defer cancel()

	atomic.AddInt64(&tm.metrics.TotalOperations, 1)
	startTime := time.Now()

	// Create a channel for the result
	type result struct {
		err error
	}
	resultChan := make(chan result, 1)

	// Run function in goroutine
	go func() {
		err := fn(timedCtx)
		resultChan <- result{err}
	}()

	// Wait for either result or timeout
	select {
	case res := <-resultChan:
		operationTime := time.Since(startTime)
		tm.recordOperationTime(operationTime)
		atomic.AddInt64(&tm.metrics.CompletedCount, 1)
		return res.err

	case <-timedCtx.Done():
		atomic.AddInt64(&tm.metrics.TimedOutCount, 1)
		return fmt.Errorf("operation timed out after %v: %w", timeout, timedCtx.Err())
	}
}

// WaitWithTimeout waits for a channel with timeout
func (tm *TimeoutManager) WaitWithTimeout(ctx context.Context, timeout time.Duration, ch <-chan interface{}) (interface{}, error) {
	timedCtx, cancel := tm.WithTimeout(ctx, timeout)
	defer cancel()

	atomic.AddInt64(&tm.metrics.TotalOperations, 1)
	startTime := time.Now()

	select {
	case value := <-ch:
		operationTime := time.Since(startTime)
		tm.recordOperationTime(operationTime)
		atomic.AddInt64(&tm.metrics.CompletedCount, 1)
		return value, nil

	case <-timedCtx.Done():
		atomic.AddInt64(&tm.metrics.TimedOutCount, 1)
		return nil, fmt.Errorf("wait timed out after %v: %w", timeout, timedCtx.Err())
	}
}

// RecordOperationTime records an operation duration for metrics
func (tm *TimeoutManager) recordOperationTime(duration time.Duration) {
	tm.metrics.TotalTimeSpent += duration

	// Update max operation time
	for {
		currentMax := atomic.LoadInt64((*int64)(&tm.metrics.MaxOperationTime))
		newMax := duration.Nanoseconds()
		if newMax <= currentMax {
			break
		}
		if atomic.CompareAndSwapInt64((*int64)(&tm.metrics.MaxOperationTime), currentMax, newMax) {
			break
		}
	}

	// Update min operation time
	if duration < tm.minOperationTime {
		tm.minOperationTime = duration
	}
}

// GetMetrics returns timeout metrics snapshot
func (tm *TimeoutManager) GetMetrics() TimeoutMetrics {
	tm.activeContextsMu.RLock()
	defer tm.activeContextsMu.RUnlock()

	metrics := tm.metrics
	metrics.TotalOperations = atomic.LoadInt64(&tm.metrics.TotalOperations)
	metrics.TimedOutCount = atomic.LoadInt64(&tm.metrics.TimedOutCount)
	metrics.CompletedCount = atomic.LoadInt64(&tm.metrics.CompletedCount)
	metrics.MinOperationTime = tm.minOperationTime

	return metrics
}

// GetActiveContextCount returns the number of active contexts
func (tm *TimeoutManager) GetActiveContextCount() int {
	tm.activeContextsMu.RLock()
	defer tm.activeContextsMu.RUnlock()
	return len(tm.activeContexts)
}

// GetContextsNearDeadline returns contexts that are within 10% of their deadline
func (tm *TimeoutManager) GetContextsNearDeadline() []contextInfo {
	tm.activeContextsMu.RLock()
	defer tm.activeContextsMu.RUnlock()

	var nearDeadline []contextInfo
	now := time.Now()

	for _, info := range tm.activeContexts {
		timeLeft := info.deadline.Sub(now)
		totalTime := info.deadline.Sub(info.startTime)
		percentLeft := float64(timeLeft) / float64(totalTime)

		if percentLeft < 0.1 { // Within 10% of deadline
			nearDeadline = append(nearDeadline, *info)
		}
	}

	return nearDeadline
}

// Reset resets the timeout manager metrics
func (tm *TimeoutManager) Reset() {
	atomic.StoreInt64(&tm.metrics.TotalOperations, 0)
	atomic.StoreInt64(&tm.metrics.TimedOutCount, 0)
	atomic.StoreInt64(&tm.metrics.CompletedCount, 0)
	tm.metrics.TotalTimeSpent = 0
	tm.metrics.MaxOperationTime = 0
	tm.minOperationTime = time.Duration(1<<63 - 1)
}

// ExportMetrics returns Prometheus format metrics
func (tm *TimeoutManager) ExportMetrics(serviceName string) string {
	metrics := tm.GetMetrics()

	avgTime := time.Duration(0)
	if metrics.TotalOperations > 0 {
		avgTime = metrics.TotalTimeSpent / time.Duration(metrics.TotalOperations)
	}

	return fmt.Sprintf(
		`timeout_total_operations{service="%s"} %d
timeout_timed_out_count{service="%s"} %d
timeout_completed_count{service="%s"} %d
timeout_rate{service="%s"} %f
timeout_total_time_seconds{service="%s"} %f
timeout_avg_operation_seconds{service="%s"} %f
timeout_max_operation_seconds{service="%s"} %f
timeout_active_contexts{service="%s"} %d
`,
		serviceName, metrics.TotalOperations,
		serviceName, metrics.TimedOutCount,
		serviceName, metrics.CompletedCount,
		serviceName, float64(metrics.TimedOutCount)/float64(metrics.TotalOperations),
		serviceName, metrics.TotalTimeSpent.Seconds(),
		serviceName, avgTime.Seconds(),
		serviceName, metrics.MaxOperationTime.Seconds(),
		serviceName, tm.GetActiveContextCount(),
	)
}

var counterID int64 = 0
