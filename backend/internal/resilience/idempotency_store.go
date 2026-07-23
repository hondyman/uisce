package resilience

import (
	"context"
	"crypto/md5"
	"fmt"
	"hash"
	"sync"
	"time"
)

// IdempotentOperation represents an operation that can be safely retried
type IdempotentOperation struct {
	ID            string
	OperationType string
	Payload       interface{}
	PayloadHash   string
	Status        string // "pending", "processing", "completed", "failed"
	Result        interface{}
	CreatedAt     time.Time
	CompletedAt   time.Time
	AttemptCount  int
	LastError     error
	TTL           time.Duration
}

// IdempotencyStore manages idempotent operation tracking
type IdempotencyStore struct {
	operations map[string]*IdempotentOperation
	mu         sync.RWMutex
	ttlTicker  *time.Ticker
	metrics    *IdempotencyMetrics
}

// IdempotencyMetrics tracks idempotency metrics
type IdempotencyMetrics struct {
	TotalOperations         int64
	DeduplicatedOperations  int64
	FailedOperations        int64
	SuccessfulOperations    int64
	AverageRetries          float64
	CurrentStoredOperations int64
	mu                      sync.RWMutex
}

// NewIdempotencyStore creates a new idempotency store
func NewIdempotencyStore() *IdempotencyStore {
	store := &IdempotencyStore{
		operations: make(map[string]*IdempotentOperation),
		metrics:    &IdempotencyMetrics{},
		ttlTicker:  time.NewTicker(1 * time.Minute),
	}

	// Start TTL cleanup goroutine
	go store.cleanupExpiredOperations()

	return store
}

// RecordOperation records a new operation for idempotency tracking
func (is *IdempotencyStore) RecordOperation(
	ctx context.Context,
	operationID string,
	operationType string,
	payload interface{},
	ttl time.Duration,
) (*IdempotentOperation, bool, error) {
	is.mu.Lock()
	defer is.mu.Unlock()

	// Check if operation already exists
	if existing, exists := is.operations[operationID]; exists {
		is.metrics.mu.Lock()
		is.metrics.DeduplicatedOperations++
		is.metrics.mu.Unlock()

		// If already completed, return the stored result
		if existing.Status == "completed" {
			return existing, true, nil
		}

		// If still processing, return error
		if existing.Status == "processing" {
			return existing, false, fmt.Errorf("operation %s already in progress", operationID)
		}

		// If failed, allow retry
		if existing.Status == "failed" {
			return existing, false, nil
		}
	}

	// Create new operation
	operation := &IdempotentOperation{
		ID:            operationID,
		OperationType: operationType,
		Payload:       payload,
		PayloadHash:   hashPayload(payload),
		Status:        "pending",
		CreatedAt:     time.Now(),
		TTL:           ttl,
		AttemptCount:  0,
	}

	is.operations[operationID] = operation

	is.metrics.mu.Lock()
	is.metrics.TotalOperations++
	is.metrics.CurrentStoredOperations = int64(len(is.operations))
	is.metrics.mu.Unlock()

	return operation, false, nil
}

// UpdateOperationStatus updates the status of an operation
func (is *IdempotencyStore) UpdateOperationStatus(
	operationID string,
	status string,
	result interface{},
	err error,
) error {
	is.mu.Lock()
	defer is.mu.Unlock()

	operation, exists := is.operations[operationID]
	if !exists {
		return fmt.Errorf("operation %s not found", operationID)
	}

	operation.Status = status
	operation.Result = result
	operation.LastError = err
	operation.AttemptCount++

	if status == "completed" {
		operation.CompletedAt = time.Now()
		is.metrics.mu.Lock()
		is.metrics.SuccessfulOperations++
		is.metrics.mu.Unlock()
	} else if status == "failed" {
		operation.CompletedAt = time.Now()
		is.metrics.mu.Lock()
		is.metrics.FailedOperations++
		is.metrics.mu.Unlock()
	}

	return nil
}

// GetOperation retrieves an operation by ID
func (is *IdempotencyStore) GetOperation(operationID string) (*IdempotentOperation, bool) {
	is.mu.RLock()
	defer is.mu.RUnlock()

	operation, exists := is.operations[operationID]
	return operation, exists
}

// ExecuteIdempotently executes an operation with idempotency guarantee
func (is *IdempotencyStore) ExecuteIdempotently(
	ctx context.Context,
	operationID string,
	operationType string,
	payload interface{},
	ttl time.Duration,
	operation func(context.Context) (interface{}, error),
) (interface{}, error) {
	// Check if this operation was already processed
	op, isDuplicate, err := is.RecordOperation(ctx, operationID, operationType, payload, ttl)
	if err != nil {
		return nil, err
	}

	// If duplicate and already completed, return cached result
	if isDuplicate {
		return op.Result, op.LastError
	}

	// Mark as processing
	is.mu.Lock()
	op.Status = "processing"
	is.mu.Unlock()

	// Execute the operation
	result, execErr := operation(ctx)

	// Update status based on result
	if execErr != nil {
		is.UpdateOperationStatus(operationID, "failed", result, execErr)
		return result, execErr
	}

	is.UpdateOperationStatus(operationID, "completed", result, nil)
	return result, nil
}

// GetOperationResult retrieves the result of a completed operation
func (is *IdempotencyStore) GetOperationResult(operationID string) (interface{}, error) {
	is.mu.RLock()
	defer is.mu.RUnlock()

	operation, exists := is.operations[operationID]
	if !exists {
		return nil, fmt.Errorf("operation %s not found", operationID)
	}

	if operation.Status != "completed" {
		return nil, fmt.Errorf("operation %s not yet completed (status: %s)", operationID, operation.Status)
	}

	return operation.Result, operation.LastError
}

// cleanupExpiredOperations removes operations that have expired
func (is *IdempotencyStore) cleanupExpiredOperations() {
	for range is.ttlTicker.C {
		is.mu.Lock()
		now := time.Now()
		removed := 0

		for id, op := range is.operations {
			expiryTime := op.CreatedAt.Add(op.TTL)
			if now.After(expiryTime) {
				delete(is.operations, id)
				removed++
			}
		}

		is.metrics.mu.Lock()
		is.metrics.CurrentStoredOperations = int64(len(is.operations))
		is.metrics.mu.Unlock()

		is.mu.Unlock()
	}
}

// GetMetrics returns idempotency metrics
func (is *IdempotencyStore) GetMetrics() *IdempotencyMetrics {
	is.metrics.mu.RLock()
	defer is.metrics.mu.RUnlock()

	metricsCopy := &IdempotencyMetrics{
		TotalOperations:         is.metrics.TotalOperations,
		DeduplicatedOperations:  is.metrics.DeduplicatedOperations,
		FailedOperations:        is.metrics.FailedOperations,
		SuccessfulOperations:    is.metrics.SuccessfulOperations,
		AverageRetries:          is.metrics.AverageRetries,
		CurrentStoredOperations: is.metrics.CurrentStoredOperations,
	}

	return metricsCopy
}

// ExportMetrics exports idempotency metrics in Prometheus format
func (is *IdempotencyStore) ExportMetrics() string {
	is.metrics.mu.RLock()
	defer is.metrics.mu.RUnlock()

	deduplicationRate := 0.0
	if is.metrics.TotalOperations > 0 {
		deduplicationRate = float64(is.metrics.DeduplicatedOperations) / float64(is.metrics.TotalOperations)
	}

	successRate := 0.0
	if is.metrics.SuccessfulOperations+is.metrics.FailedOperations > 0 {
		successRate = float64(is.metrics.SuccessfulOperations) / float64(is.metrics.SuccessfulOperations+is.metrics.FailedOperations)
	}

	return fmt.Sprintf(`
# Idempotency Store Metrics
idempotency_total_operations %d
idempotency_deduplicated_operations %d
idempotency_failed_operations %d
idempotency_successful_operations %d
idempotency_deduplication_rate %.4f
idempotency_success_rate %.4f
idempotency_current_stored_operations %d
`,
		is.metrics.TotalOperations,
		is.metrics.DeduplicatedOperations,
		is.metrics.FailedOperations,
		is.metrics.SuccessfulOperations,
		deduplicationRate,
		successRate,
		is.metrics.CurrentStoredOperations,
	)
}

// hashPayload creates a hash of the operation payload
func hashPayload(payload interface{}) string {
	var h hash.Hash = md5.New()
	fmt.Fprintf(h, "%v", payload)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// PruneOperations manually prunes old operations
func (is *IdempotencyStore) PruneOperations(maxAge time.Duration) int {
	is.mu.Lock()
	defer is.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, op := range is.operations {
		if op.CompletedAt.Before(cutoff) {
			delete(is.operations, id)
			removed++
		}
	}

	is.metrics.mu.Lock()
	is.metrics.CurrentStoredOperations = int64(len(is.operations))
	is.metrics.mu.Unlock()

	return removed
}
