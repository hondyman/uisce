package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

// CircuitBreakerConfig holds configuration for circuit breaker behavior
type CircuitBreakerConfig struct {
	Name              string        // Unique name for this circuit breaker
	FailureThreshold  int64         // Number of failures before opening
	SuccessThreshold  int64         // Number of successes to close from half-open
	Timeout           time.Duration // Time to wait before attempting half-open
	HalfOpenMaxCalls  int64         // Max concurrent calls in half-open state
	FailureRateThresh float64       // Failure rate (0-1) threshold to open circuit
}

// CircuitBreakerMetrics holds metrics for observability
type CircuitBreakerMetrics struct {
	TotalCalls      int64      // Total calls handled
	SuccessfulCalls int64      // Successful calls
	FailedCalls     int64      // Failed calls
	RejectedCalls   int64      // Calls rejected (circuit open)
	LastStateChange time.Time  // Last state transition
	CurrentState    string     // Current state (closed/open/half-open)
	FailureRate     float64    // Current failure rate (0-1)
	StateChanges    []StateLog // History of state changes
}

// StateLog records when and why state changed
type StateLog struct {
	Timestamp   time.Time
	OldState    CircuitBreakerState
	NewState    CircuitBreakerState
	Reason      string
	FailureRate float64
}

// CircuitBreaker implements the circuit breaker resilience pattern
type CircuitBreaker struct {
	config            CircuitBreakerConfig
	state             CircuitBreakerState
	failureCount      int64
	successCount      int64
	totalCount        int64
	halfOpenCallCount int64
	lastFailureTime   time.Time
	lastStateChangeAt time.Time
	metrics           CircuitBreakerMetrics
	mu                sync.RWMutex
	stateChangeMu     sync.Mutex
	stateChan         chan CircuitBreakerState
	halfOpenSemaphore *Semaphore
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.HalfOpenMaxCalls == 0 {
		config.HalfOpenMaxCalls = 3
	}
	if config.FailureRateThresh == 0 {
		config.FailureRateThresh = 0.5 // 50%
	}

	cb := &CircuitBreaker{
		config:            config,
		state:             StateClosed,
		lastStateChangeAt: time.Now(),
		stateChan:         make(chan CircuitBreakerState, 10),
		halfOpenSemaphore: NewSemaphore(config.HalfOpenMaxCalls),
		metrics: CircuitBreakerMetrics{
			CurrentState:    "closed",
			LastStateChange: time.Now(),
			StateChanges:    []StateLog{},
		},
	}

	return cb
}

// Execute runs the given function through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	cb.mu.RLock()
	currentState := cb.state
	cb.mu.RUnlock()

	// Check if circuit should be opened based on failure rate
	if currentState == StateClosed {
		cb.mu.RLock()
		failureRate := cb.calculateFailureRate()
		cb.mu.RUnlock()

		if failureRate > cb.config.FailureRateThresh && cb.totalCount > 0 {
			cb.openCircuit("failure_rate_threshold_exceeded")
			currentState = StateOpen
		}
	}

	switch currentState {
	case StateClosed:
		return cb.executeClosed(ctx, fn)
	case StateOpen:
		return cb.executeOpen(ctx)
	case StateHalfOpen:
		return cb.executeHalfOpen(ctx, fn)
	}

	return fmt.Errorf("unknown circuit breaker state: %d", currentState)
}

// executeClosed handles execution when circuit is closed (normal operation)
func (cb *CircuitBreaker) executeClosed(ctx context.Context, fn func(context.Context) error) error {
	err := fn(ctx)
	atomic.AddInt64(&cb.totalCount, 1)

	if err != nil {
		atomic.AddInt64(&cb.failureCount, 1)
		cb.mu.Lock()
		cb.lastFailureTime = time.Now()
		cb.mu.Unlock()

		// Check if failure threshold reached
		if atomic.LoadInt64(&cb.failureCount) >= cb.config.FailureThreshold {
			cb.openCircuit("failure_threshold_exceeded")
		}
	} else {
		atomic.AddInt64(&cb.successCount, 1)
		// Reset failure count on success
		atomic.StoreInt64(&cb.failureCount, 0)
	}

	return err
}

// executeOpen handles execution when circuit is open (failing fast)
func (cb *CircuitBreaker) executeOpen(ctx context.Context) error {
	cb.mu.RLock()
	timeSinceOpen := time.Since(cb.lastStateChangeAt)
	cb.mu.RUnlock()

	atomic.AddInt64(&cb.metrics.RejectedCalls, 1)

	// Check if timeout elapsed to attempt half-open
	if timeSinceOpen >= cb.config.Timeout {
		cb.transitionToHalfOpen()
		return fmt.Errorf("circuit breaker open (attempting recovery)")
	}

	return fmt.Errorf("circuit breaker open, next attempt in %v", cb.config.Timeout-timeSinceOpen)
}

// executeHalfOpen handles execution when circuit is half-open (testing recovery)
func (cb *CircuitBreaker) executeHalfOpen(ctx context.Context, fn func(context.Context) error) error {
	// Acquire permit from semaphore (limits concurrent calls in half-open)
	if !cb.halfOpenSemaphore.TryAcquire() {
		atomic.AddInt64(&cb.metrics.RejectedCalls, 1)
		return fmt.Errorf("circuit breaker half-open, max concurrent calls exceeded")
	}
	defer cb.halfOpenSemaphore.Release()

	err := fn(ctx)
	atomic.AddInt64(&cb.totalCount, 1)
	atomic.AddInt64(&cb.halfOpenCallCount, 1)

	if err != nil {
		atomic.AddInt64(&cb.failureCount, 1)
		// One failure reopens the circuit
		if atomic.LoadInt64(&cb.failureCount) >= 1 {
			cb.openCircuit("half_open_request_failed")
		}
	} else {
		atomic.AddInt64(&cb.successCount, 1)

		// Sufficient successes close the circuit
		if atomic.LoadInt64(&cb.successCount) >= cb.config.SuccessThreshold {
			cb.closeCircuit()
		}
	}

	return err
}

// openCircuit transitions circuit to open state
func (cb *CircuitBreaker) openCircuit(reason string) {
	cb.stateChangeMu.Lock()
	defer cb.stateChangeMu.Unlock()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen {
		return // Already open
	}

	oldState := cb.state
	cb.state = StateOpen
	cb.lastStateChangeAt = time.Now()

	failureRate := cb.calculateFailureRate()
	cb.metrics.StateChanges = append(cb.metrics.StateChanges, StateLog{
		Timestamp:   time.Now(),
		OldState:    oldState,
		NewState:    StateOpen,
		Reason:      reason,
		FailureRate: failureRate,
	})
	cb.metrics.CurrentState = "open"
	cb.metrics.LastStateChange = time.Now()
	cb.metrics.FailureRate = failureRate

	select {
	case cb.stateChan <- StateOpen:
	default:
	}
}

// transitionToHalfOpen transitions circuit to half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.stateChangeMu.Lock()
	defer cb.stateChangeMu.Unlock()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state != StateOpen {
		return // Only transition from open
	}

	oldState := cb.state
	cb.state = StateHalfOpen
	cb.lastStateChangeAt = time.Now()
	atomic.StoreInt64(&cb.failureCount, 0)
	atomic.StoreInt64(&cb.successCount, 0)
	atomic.StoreInt64(&cb.halfOpenCallCount, 0)

	cb.metrics.StateChanges = append(cb.metrics.StateChanges, StateLog{
		Timestamp: time.Now(),
		OldState:  oldState,
		NewState:  StateHalfOpen,
		Reason:    "timeout_elapsed",
	})
	cb.metrics.CurrentState = "half-open"
	cb.metrics.LastStateChange = time.Now()

	select {
	case cb.stateChan <- StateHalfOpen:
	default:
	}
}

// closeCircuit transitions circuit to closed state
func (cb *CircuitBreaker) closeCircuit() {
	cb.stateChangeMu.Lock()
	defer cb.stateChangeMu.Unlock()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateClosed {
		return // Already closed
	}

	oldState := cb.state
	cb.state = StateClosed
	cb.lastStateChangeAt = time.Now()
	atomic.StoreInt64(&cb.failureCount, 0)
	atomic.StoreInt64(&cb.successCount, 0)

	cb.metrics.StateChanges = append(cb.metrics.StateChanges, StateLog{
		Timestamp: time.Now(),
		OldState:  oldState,
		NewState:  StateClosed,
		Reason:    "recovered",
	})
	cb.metrics.CurrentState = "closed"
	cb.metrics.LastStateChange = time.Now()

	select {
	case cb.stateChan <- StateClosed:
	default:
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStateName returns the string representation of current state
func (cb *CircuitBreaker) GetStateName() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	switch cb.state {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// GetMetrics returns a snapshot of circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	metrics := cb.metrics
	metrics.TotalCalls = atomic.LoadInt64(&cb.totalCount)
	metrics.SuccessfulCalls = atomic.LoadInt64(&cb.successCount)
	metrics.FailedCalls = atomic.LoadInt64(&cb.failureCount)
	metrics.RejectedCalls = atomic.LoadInt64(&cb.metrics.RejectedCalls)
	metrics.FailureRate = cb.calculateFailureRate()

	return metrics
}

// StateChan returns a channel that receives state change notifications
func (cb *CircuitBreaker) StateChan() <-chan CircuitBreakerState {
	return cb.stateChan
}

// calculateFailureRate returns the current failure rate (0-1)
func (cb *CircuitBreaker) calculateFailureRate() float64 {
	if cb.totalCount == 0 {
		return 0
	}
	return float64(cb.failureCount) / float64(cb.totalCount)
}

// Reset resets the circuit breaker to initial state
func (cb *CircuitBreaker) Reset() {
	cb.stateChangeMu.Lock()
	defer cb.stateChangeMu.Unlock()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.StoreInt64(&cb.failureCount, 0)
	atomic.StoreInt64(&cb.successCount, 0)
	atomic.StoreInt64(&cb.totalCount, 0)
	atomic.StoreInt64(&cb.halfOpenCallCount, 0)
	atomic.StoreInt64(&cb.metrics.RejectedCalls, 0)

	cb.state = StateClosed
	cb.lastStateChangeAt = time.Now()
	cb.metrics.CurrentState = "closed"
	cb.metrics.FailureRate = 0
	cb.metrics.StateChanges = []StateLog{}
}

// ExportMetrics returns Prometheus format metrics
func (cb *CircuitBreaker) ExportMetrics() string {
	metrics := cb.GetMetrics()
	stateValue := int64(cb.GetState())

	return fmt.Sprintf(
		`circuit_breaker_state{name="%s"} %d
circuit_breaker_total_calls{name="%s"} %d
circuit_breaker_successful_calls{name="%s"} %d
circuit_breaker_failed_calls{name="%s"} %d
circuit_breaker_rejected_calls{name="%s"} %d
circuit_breaker_failure_rate{name="%s"} %f
circuit_breaker_state_change_timestamp{name="%s"} %d
`,
		cb.config.Name, stateValue,
		cb.config.Name, metrics.TotalCalls,
		cb.config.Name, metrics.SuccessfulCalls,
		cb.config.Name, metrics.FailedCalls,
		cb.config.Name, metrics.RejectedCalls,
		cb.config.Name, metrics.FailureRate,
		cb.config.Name, metrics.LastStateChange.Unix(),
	)
}
