package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SagaStep represents a single step in a saga workflow
type SagaStep struct {
	Name             string
	Action           func(context.Context, interface{}) (interface{}, error)
	Compensation     func(context.Context, interface{}) error
	CompensationData interface{}
	Timeout          time.Duration
	MaxRetries       int
}

// SagaStepResult contains the result of executing a saga step
type SagaStepResult struct {
	StepName         string
	Status           string // "pending", "executing", "completed", "compensating", "compensated", "failed"
	Result           interface{}
	CompensationData interface{}
	Error            error
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
	AttemptCount     int
}

// SagaTransaction represents a complete saga workflow
type SagaTransaction struct {
	ID              string
	Steps           []SagaStep
	Results         map[string]*SagaStepResult
	Status          string // "pending", "executing", "completed", "compensating", "failed"
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	FailedStep      string
	CompensationLog []string
	mu              sync.RWMutex
}

// SagaCoordinator orchestrates distributed transactions across services
type SagaCoordinator struct {
	transactions     map[string]*SagaTransaction
	transactionMutex sync.RWMutex
	metrics          *SagaMetrics
	eventBus         SagaEventBus
	compensationReg  *CompensationRegistry
}

// SagaMetrics tracks saga execution metrics
type SagaMetrics struct {
	TotalSagas         int64
	SuccessfulSagas    int64
	FailedSagas        int64
	CompensatedSagas   int64
	AverageDuration    time.Duration
	AverageSteps       float64
	CompensationRate   float64
	CurrentActiveSagas int64
	mu                 sync.RWMutex
}

// SagaEventBus publishes saga lifecycle events
type SagaEventBus interface {
	PublishStepStarted(ctx context.Context, sagaID, stepName string) error
	PublishStepCompleted(ctx context.Context, sagaID, stepName string, result interface{}) error
	PublishStepFailed(ctx context.Context, sagaID, stepName string, err error) error
	PublishCompensationStarted(ctx context.Context, sagaID string) error
	PublishCompensationCompleted(ctx context.Context, sagaID string) error
}

// CompensationRegistry stores compensation functions for failed steps
type CompensationRegistry struct {
	compensations map[string]func(context.Context, interface{}) error
	mu            sync.RWMutex
}

// NewSagaCoordinator creates a new saga coordinator
func NewSagaCoordinator(eventBus SagaEventBus) *SagaCoordinator {
	return &SagaCoordinator{
		transactions:    make(map[string]*SagaTransaction),
		metrics:         &SagaMetrics{},
		eventBus:        eventBus,
		compensationReg: NewCompensationRegistry(),
	}
}

// NewCompensationRegistry creates a new compensation registry
func NewCompensationRegistry() *CompensationRegistry {
	return &CompensationRegistry{
		compensations: make(map[string]func(context.Context, interface{}) error),
	}
}

// RegisterCompensation registers a compensation function for a step
func (cr *CompensationRegistry) RegisterCompensation(
	stepName string,
	compensationFn func(context.Context, interface{}) error,
) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.compensations[stepName] = compensationFn
}

// GetCompensation retrieves a registered compensation function
func (cr *CompensationRegistry) GetCompensation(stepName string) func(context.Context, interface{}) error {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.compensations[stepName]
}

// ExecuteSaga orchestrates the execution of a saga with full compensation support
func (sc *SagaCoordinator) ExecuteSaga(ctx context.Context, sagaID string, steps []SagaStep) error {
	saga := &SagaTransaction{
		ID:        sagaID,
		Steps:     steps,
		Results:   make(map[string]*SagaStepResult),
		Status:    "executing",
		StartTime: time.Now(),
	}

	sc.transactionMutex.Lock()
	sc.transactions[sagaID] = saga
	sc.transactionMutex.Unlock()

	sc.metrics.mu.Lock()
	sc.metrics.TotalSagas++
	sc.metrics.CurrentActiveSagas++
	sc.metrics.mu.Unlock()

	// Execute each step in sequence
	for i, step := range steps {
		result := sc.executeStep(ctx, saga, step, i)
		saga.mu.Lock()
		saga.Results[step.Name] = result
		saga.mu.Unlock()

		if result.Error != nil {
			saga.mu.Lock()
			saga.Status = "compensating"
			saga.FailedStep = step.Name
			saga.mu.Unlock()

			// Trigger compensation for completed steps
			err := sc.compensate(ctx, saga)
			if err != nil {
				saga.mu.Lock()
				saga.Status = "failed"
				saga.mu.Unlock()
				return fmt.Errorf("saga %s failed at step %s with compensation error: %w", sagaID, step.Name, err)
			}

			saga.mu.Lock()
			saga.Status = "failed"
			saga.mu.Unlock()
			return fmt.Errorf("saga %s failed at step %s: %w", sagaID, step.Name, result.Error)
		}
	}

	saga.mu.Lock()
	saga.Status = "completed"
	saga.EndTime = time.Now()
	saga.Duration = saga.EndTime.Sub(saga.StartTime)
	saga.mu.Unlock()

	sc.metrics.mu.Lock()
	sc.metrics.SuccessfulSagas++
	sc.metrics.CurrentActiveSagas--
	sc.metrics.mu.Unlock()

	return nil
}

// executeStep executes a single saga step with retries
func (sc *SagaCoordinator) executeStep(
	ctx context.Context,
	saga *SagaTransaction,
	step SagaStep,
	stepIndex int,
) *SagaStepResult {
	result := &SagaStepResult{
		StepName:     step.Name,
		Status:       "pending",
		StartTime:    time.Now(),
		AttemptCount: 0,
	}

	// Create timeout context if specified
	stepCtx := ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	// Attempt execution with retries
	for attempt := 0; attempt <= step.MaxRetries; attempt++ {
		result.AttemptCount = attempt + 1
		result.Status = "executing"

		// Publish step started event
		if sc.eventBus != nil {
			sc.eventBus.PublishStepStarted(ctx, saga.ID, step.Name)
		}

		// Execute the action
		stepResult, err := step.Action(stepCtx, nil)
		if err == nil {
			result.Status = "completed"
			result.Result = stepResult
			result.CompensationData = stepResult
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)

			// Publish step completed event
			if sc.eventBus != nil {
				sc.eventBus.PublishStepCompleted(ctx, saga.ID, step.Name, stepResult)
			}

			return result
		}

		result.Error = err

		// Don't retry if context canceled or deadline exceeded
		if err == context.Canceled || err == context.DeadlineExceeded {
			result.Status = "failed"
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}

		// Publish step failed event if this is the last attempt
		if attempt == step.MaxRetries {
			if sc.eventBus != nil {
				sc.eventBus.PublishStepFailed(ctx, saga.ID, step.Name, err)
			}
		}

		// Exponential backoff between retries
		if attempt < step.MaxRetries {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return result
			}
		}
	}

	result.Status = "failed"
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	return result
}

// compensate executes compensation logic for completed steps in reverse order
func (sc *SagaCoordinator) compensate(ctx context.Context, saga *SagaTransaction) error {
	saga.mu.Lock()
	defer saga.mu.Unlock()

	if sc.eventBus != nil {
		sc.eventBus.PublishCompensationStarted(ctx, saga.ID)
	}

	// Execute compensation in reverse order
	for i := len(saga.Steps) - 1; i >= 0; i-- {
		step := saga.Steps[i]
		result, exists := saga.Results[step.Name]

		// Only compensate steps that were successfully completed
		if !exists || result.Status != "completed" {
			continue
		}

		// Mark as compensating
		result.Status = "compensating"
		saga.CompensationLog = append(saga.CompensationLog, fmt.Sprintf("compensating %s", step.Name))

		// Execute compensation
		var err error
		if step.Compensation != nil {
			err = step.Compensation(ctx, result.CompensationData)
		}

		if err != nil {
			saga.CompensationLog = append(saga.CompensationLog, fmt.Sprintf("compensation failed for %s: %v", step.Name, err))
			return fmt.Errorf("compensation failed for step %s: %w", step.Name, err)
		}

		result.Status = "compensated"
		saga.CompensationLog = append(saga.CompensationLog, fmt.Sprintf("successfully compensated %s", step.Name))
	}

	if sc.eventBus != nil {
		sc.eventBus.PublishCompensationCompleted(ctx, saga.ID)
	}

	sc.metrics.mu.Lock()
	sc.metrics.CompensatedSagas++
	sc.metrics.mu.Unlock()

	return nil
}

// GetSagaStatus returns the current status of a saga
func (sc *SagaCoordinator) GetSagaStatus(sagaID string) (*SagaTransaction, error) {
	sc.transactionMutex.RLock()
	defer sc.transactionMutex.RUnlock()

	saga, exists := sc.transactions[sagaID]
	if !exists {
		return nil, fmt.Errorf("saga %s not found", sagaID)
	}

	saga.mu.RLock()
	defer saga.mu.RUnlock()

	// Return a copy to avoid race conditions
	sagaCopy := &SagaTransaction{
		ID:              saga.ID,
		Status:          saga.Status,
		StartTime:       saga.StartTime,
		EndTime:         saga.EndTime,
		Duration:        saga.Duration,
		FailedStep:      saga.FailedStep,
		CompensationLog: make([]string, len(saga.CompensationLog)),
		Results:         make(map[string]*SagaStepResult),
	}
	copy(sagaCopy.CompensationLog, saga.CompensationLog)

	for k, v := range saga.Results {
		sagaCopy.Results[k] = v
	}

	return sagaCopy, nil
}

// GetMetrics returns saga coordinator metrics
func (sc *SagaCoordinator) GetMetrics() *SagaMetrics {
	sc.metrics.mu.RLock()
	defer sc.metrics.mu.RUnlock()

	metricsCopy := &SagaMetrics{
		TotalSagas:         sc.metrics.TotalSagas,
		SuccessfulSagas:    sc.metrics.SuccessfulSagas,
		FailedSagas:        sc.metrics.FailedSagas,
		CompensatedSagas:   sc.metrics.CompensatedSagas,
		AverageDuration:    sc.metrics.AverageDuration,
		AverageSteps:       sc.metrics.AverageSteps,
		CompensationRate:   sc.metrics.CompensationRate,
		CurrentActiveSagas: sc.metrics.CurrentActiveSagas,
	}

	return metricsCopy
}

// ExportMetrics exports saga metrics in Prometheus format
func (sc *SagaCoordinator) ExportMetrics() string {
	sc.metrics.mu.RLock()
	defer sc.metrics.mu.RUnlock()

	totalFailed := sc.metrics.FailedSagas + sc.metrics.CompensatedSagas
	compensationRate := 0.0
	if sc.metrics.TotalSagas > 0 {
		compensationRate = float64(sc.metrics.CompensatedSagas) / float64(sc.metrics.TotalSagas)
	}

	return fmt.Sprintf(`
# Saga Coordinator Metrics
saga_total_transactions %d
saga_successful_transactions %d
saga_failed_transactions %d
saga_compensated_transactions %d
saga_compensation_rate %.4f
saga_current_active %d
saga_success_rate %.4f
`,
		sc.metrics.TotalSagas,
		sc.metrics.SuccessfulSagas,
		totalFailed,
		sc.metrics.CompensatedSagas,
		compensationRate,
		sc.metrics.CurrentActiveSagas,
		float64(sc.metrics.SuccessfulSagas)/float64(sc.metrics.TotalSagas+1),
	)
}

// CleanupOldSagas removes sagas older than the specified duration
func (sc *SagaCoordinator) CleanupOldSagas(maxAge time.Duration) int {
	sc.transactionMutex.Lock()
	defer sc.transactionMutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, saga := range sc.transactions {
		saga.mu.RLock()
		endTime := saga.EndTime
		status := saga.Status
		saga.mu.RUnlock()

		if (status == "completed" || status == "failed") && endTime.Before(cutoff) {
			delete(sc.transactions, id)
			removed++
		}
	}

	return removed
}
