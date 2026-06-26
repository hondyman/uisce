package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// BulkheadConfig holds bulkhead isolation configuration
type BulkheadConfig struct {
	Name          string        // Identifier for this bulkhead
	MaxConcurrent int64         // Maximum concurrent operations
	QueueSize     int64         // Queue size for waiting operations
	WaitTimeout   time.Duration // Timeout for acquiring a permit
}

// BulkheadMetrics tracks bulkhead isolation statistics
type BulkheadMetrics struct {
	TotalRequests     int64         // Total requests received
	AllowedRequests   int64         // Requests allowed to execute
	QueuedRequests    int64         // Requests queued waiting for permit
	RejectedRequests  int64         // Requests rejected (queue full)
	CurrentConcurrent int64         // Current concurrent executions
	PeakConcurrent    int64         // Peak concurrent executions
	AvgWaitTime       time.Duration // Average wait time
	MaxWaitTime       time.Duration // Maximum wait time
	TotalWaitTime     time.Duration // Total wait time
}

// BulkheadIsolation implements bulkhead pattern for limiting concurrent operations
type BulkheadIsolation struct {
	config        BulkheadConfig
	semaphore     *Semaphore
	queue         chan struct{}
	metrics       BulkheadMetrics
	maxWaitTime   time.Duration
	avgWaitTime   time.Duration
	totalWaitTime time.Duration
	waitCount     int64
	mu            sync.RWMutex
}

// NewBulkheadIsolation creates a new bulkhead isolation instance
func NewBulkheadIsolation(config BulkheadConfig) *BulkheadIsolation {
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 10
	}
	if config.QueueSize == 0 {
		config.QueueSize = 100
	}
	if config.WaitTimeout == 0 {
		config.WaitTimeout = 5 * time.Second
	}

	return &BulkheadIsolation{
		config:        config,
		semaphore:     NewSemaphore(config.MaxConcurrent),
		queue:         make(chan struct{}, config.QueueSize),
		maxWaitTime:   0,
		avgWaitTime:   0,
		totalWaitTime: 0,
	}
}

// Execute runs a function with bulkhead isolation
func (bi *BulkheadIsolation) Execute(ctx context.Context, fn func(context.Context) error) error {
	atomic.AddInt64(&bi.metrics.TotalRequests, 1)

	// Try to acquire permit with timeout
	startTime := time.Now()
	acquired := bi.tryAcquireWithTimeout(ctx)
	waitTime := time.Since(startTime)

	if !acquired {
		atomic.AddInt64(&bi.metrics.RejectedRequests, 1)
		return fmt.Errorf("bulkhead %s: no permits available", bi.config.Name)
	}

	// Update wait time metrics
	bi.recordWaitTime(waitTime)
	atomic.AddInt64(&bi.metrics.AllowedRequests, 1)

	// Update current concurrent count
	currentConcurrent := atomic.AddInt64(&bi.metrics.CurrentConcurrent, 1)

	// Update peak concurrent
	for {
		currentPeak := atomic.LoadInt64(&bi.metrics.PeakConcurrent)
		if currentConcurrent <= currentPeak {
			break
		}
		if atomic.CompareAndSwapInt64(&bi.metrics.PeakConcurrent, currentPeak, currentConcurrent) {
			break
		}
	}

	defer func() {
		bi.semaphore.Release()
		atomic.AddInt64(&bi.metrics.CurrentConcurrent, -1)
	}()

	// Execute the function
	return fn(ctx)
}

// ExecuteAsync queues a function for execution with bulkhead isolation
func (bi *BulkheadIsolation) ExecuteAsync(ctx context.Context, fn func(context.Context) error, resultChan chan error) {
	atomic.AddInt64(&bi.metrics.TotalRequests, 1)

	// Try to queue the request
	select {
	case bi.queue <- struct{}{}:
		atomic.AddInt64(&bi.metrics.QueuedRequests, 1)

		// Process queued request
		go func() {
			defer func() { <-bi.queue }()

			if err := bi.Execute(ctx, fn); err != nil {
				select {
				case resultChan <- err:
				case <-ctx.Done():
				}
			} else {
				select {
				case resultChan <- nil:
				case <-ctx.Done():
				}
			}
		}()

	default:
		atomic.AddInt64(&bi.metrics.RejectedRequests, 1)
		select {
		case resultChan <- fmt.Errorf("bulkhead %s: queue full", bi.config.Name):
		case <-ctx.Done():
		}
	}
}

// tryAcquireWithTimeout attempts to acquire a permit with timeout
func (bi *BulkheadIsolation) tryAcquireWithTimeout(ctx context.Context) bool {
	select {
	case <-time.After(bi.config.WaitTimeout):
		return false
	case <-ctx.Done():
		return false
	default:
		// Try non-blocking acquire
		if bi.semaphore.TryAcquire() {
			return true
		}

		// Wait for permit or timeout
		select {
		case <-time.After(bi.config.WaitTimeout):
			return false
		case <-ctx.Done():
			return false
		default:
			bi.semaphore.Acquire()
			return true
		}
	}
}

// recordWaitTime records a wait time measurement
func (bi *BulkheadIsolation) recordWaitTime(waitTime time.Duration) {
	bi.mu.Lock()
	defer bi.mu.Unlock()

	bi.totalWaitTime += waitTime
	bi.waitCount++

	if bi.waitCount > 0 {
		bi.avgWaitTime = bi.totalWaitTime / time.Duration(bi.waitCount)
	}

	if waitTime > bi.maxWaitTime {
		bi.maxWaitTime = waitTime
	}
}

// GetMetrics returns bulkhead metrics snapshot
func (bi *BulkheadIsolation) GetMetrics() BulkheadMetrics {
	bi.mu.RLock()
	defer bi.mu.RUnlock()

	metrics := bi.metrics
	metrics.TotalRequests = atomic.LoadInt64(&bi.metrics.TotalRequests)
	metrics.AllowedRequests = atomic.LoadInt64(&bi.metrics.AllowedRequests)
	metrics.QueuedRequests = atomic.LoadInt64(&bi.metrics.QueuedRequests)
	metrics.RejectedRequests = atomic.LoadInt64(&bi.metrics.RejectedRequests)
	metrics.CurrentConcurrent = atomic.LoadInt64(&bi.metrics.CurrentConcurrent)
	metrics.PeakConcurrent = atomic.LoadInt64(&bi.metrics.PeakConcurrent)
	metrics.AvgWaitTime = bi.avgWaitTime
	metrics.MaxWaitTime = bi.maxWaitTime
	metrics.TotalWaitTime = bi.totalWaitTime

	return metrics
}

// GetAvailablePermits returns the number of available permits
func (bi *BulkheadIsolation) GetAvailablePermits() int {
	return bi.semaphore.CurrentPermits()
}

// Reset resets bulkhead metrics
func (bi *BulkheadIsolation) Reset() {
	bi.mu.Lock()
	defer bi.mu.Unlock()

	atomic.StoreInt64(&bi.metrics.TotalRequests, 0)
	atomic.StoreInt64(&bi.metrics.AllowedRequests, 0)
	atomic.StoreInt64(&bi.metrics.QueuedRequests, 0)
	atomic.StoreInt64(&bi.metrics.RejectedRequests, 0)
	atomic.StoreInt64(&bi.metrics.CurrentConcurrent, 0)
	atomic.StoreInt64(&bi.metrics.PeakConcurrent, 0)

	bi.maxWaitTime = 0
	bi.avgWaitTime = 0
	bi.totalWaitTime = 0
	bi.waitCount = 0
}

// ExportMetrics returns Prometheus format metrics
func (bi *BulkheadIsolation) ExportMetrics() string {
	metrics := bi.GetMetrics()

	allowRate := float64(0)
	rejectRate := float64(0)
	if metrics.TotalRequests > 0 {
		allowRate = float64(metrics.AllowedRequests) / float64(metrics.TotalRequests)
		rejectRate = float64(metrics.RejectedRequests) / float64(metrics.TotalRequests)
	}

	return fmt.Sprintf(
		`bulkhead_total_requests{name="%s"} %d
bulkhead_allowed_requests{name="%s"} %d
bulkhead_queued_requests{name="%s"} %d
bulkhead_rejected_requests{name="%s"} %d
bulkhead_allow_rate{name="%s"} %f
bulkhead_reject_rate{name="%s"} %f
bulkhead_current_concurrent{name="%s"} %d
bulkhead_peak_concurrent{name="%s"} %d
bulkhead_max_concurrent{name="%s"} %d
bulkhead_avg_wait_ms{name="%s"} %f
bulkhead_max_wait_ms{name="%s"} %f
`,
		bi.config.Name, metrics.TotalRequests,
		bi.config.Name, metrics.AllowedRequests,
		bi.config.Name, metrics.QueuedRequests,
		bi.config.Name, metrics.RejectedRequests,
		bi.config.Name, allowRate,
		bi.config.Name, rejectRate,
		bi.config.Name, metrics.CurrentConcurrent,
		bi.config.Name, metrics.PeakConcurrent,
		bi.config.Name, bi.config.MaxConcurrent,
		bi.config.Name, float64(metrics.AvgWaitTime.Milliseconds()),
		bi.config.Name, float64(metrics.MaxWaitTime.Milliseconds()),
	)
}
