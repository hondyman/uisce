package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"strings"

	"github.com/jmoiron/sqlx"
	kafka "github.com/segmentio/kafka-go"
)

// ============================================================================
// ASYNC VALIDATION SERVICE
// ============================================================================
// Non-blocking validation processing with event-driven architecture
// Enables Workday-like real-time validation feedback without blocking operations

// ValidationTask represents a single validation job
type ValidationTask struct {
	ID         string                 `json:"id"`
	EntityID   string                 `json:"entity_id"`
	EntityType string                 `json:"entity_type"`
	TenantID   string                 `json:"tenant_id"`
	EntityData map[string]interface{} `json:"entity_data"`
	RuleIDs    []string               `json:"rule_ids"`
	CreatedAt  time.Time              `json:"created_at"`
	Status     string                 `json:"status"` // pending, processing, completed, failed
	Priority   int                    `json:"priority"`
	Retries    int                    `json:"retries"`
}

// AsyncValidator provides non-blocking validation
type AsyncValidator interface {
	// Submit validation task to async queue
	SubmitValidationTask(ctx context.Context, task *ValidationTask) error

	// Get task status
	GetTaskStatus(ctx context.Context, taskID string) (string, error)

	// Get validation result once complete
	WaitForValidationResult(ctx context.Context, taskID string, timeout time.Duration) (*ValidationTaskResult, error)

	// Poll for result (non-blocking)
	GetValidationResult(ctx context.Context, taskID string) (*ValidationTaskResult, error)

	// Subscribe to validation events
	SubscribeToValidationEvents(ctx context.Context) (<-chan *ValidationEvent, error)

	// Process pending tasks (worker)
	ProcessValidationQueue(ctx context.Context, workerID string) error

	// Get validation queue stats
	GetQueueStats(ctx context.Context) (map[string]interface{}, error)
}

// ValidationTaskResult holds the outcome of a validation task
type ValidationTaskResult struct {
	TaskID      string                   `json:"task_id"`
	Status      string                   `json:"status"` // valid, warning, error, failed
	Errors      []map[string]interface{} `json:"errors"`
	Warnings    []map[string]interface{} `json:"warnings"`
	Summary     string                   `json:"summary"`
	ProcessedAt time.Time                `json:"processed_at"`
	Duration    time.Duration            `json:"duration"`
}

// ValidationEvent is emitted when validation completes
type ValidationEvent struct {
	TaskID     string                `json:"task_id"`
	EntityID   string                `json:"entity_id"`
	EntityType string                `json:"entity_type"`
	Status     string                `json:"status"`
	Result     *ValidationTaskResult `json:"result"`
	Timestamp  time.Time             `json:"timestamp"`
}

// AsyncValidatorImpl implements AsyncValidator
type AsyncValidatorImpl struct {
	db *sqlx.DB
	// Kafka writers for tasks and results
	taskWriter   *kafka.Writer
	resultWriter *kafka.Writer
	mu           sync.RWMutex
	tasks        map[string]*ValidationTask
	results      map[string]*ValidationTaskResult
	eventChan    chan *ValidationEvent
}

// NewAsyncValidator creates a new async validation service using Kafka brokers (comma-separated)
func NewAsyncValidator(db *sqlx.DB, kafkaBrokers string) (AsyncValidator, error) {
	if kafkaBrokers == "" {
		return nil, fmt.Errorf("kafka brokers not configured")
	}
	brokers := strings.Split(kafkaBrokers, ",")

	// Writer for task submission
	taskWriter := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	// Writer for results/events
	resultWriter := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	return &AsyncValidatorImpl{
		db:           db,
		taskWriter:   taskWriter,
		resultWriter: resultWriter,
		tasks:        make(map[string]*ValidationTask),
		results:      make(map[string]*ValidationTaskResult),
		eventChan:    make(chan *ValidationEvent, 100),
	}, nil
}

// ============================================================================
// Task Submission
// ============================================================================

// SubmitValidationTask submits a validation task to the queue
func (av *AsyncValidatorImpl) SubmitValidationTask(ctx context.Context, task *ValidationTask) error {
	task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	task.CreatedAt = time.Now()
	task.Status = "pending"

	// Store task
	av.mu.Lock()
	av.tasks[task.ID] = task
	av.mu.Unlock()

	// Serialize task
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Publish task to Kafka topic 'validation.tasks'
	msg := kafka.Message{
		Topic: "validation.tasks",
		Key:   []byte(task.ID),
		Value: taskJSON,
		Time:  time.Now(),
	}
	if err := av.taskWriter.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish task: %w", err)
	}

	log.Printf("[AsyncValidator] Task submitted: %s for %s %s", task.ID, task.EntityType, task.EntityID)
	return nil
}

// ============================================================================
// Result Retrieval
// ============================================================================

// GetTaskStatus returns the current status of a validation task
func (av *AsyncValidatorImpl) GetTaskStatus(ctx context.Context, taskID string) (string, error) {
	av.mu.RLock()
	task, exists := av.tasks[taskID]
	av.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("task not found: %s", taskID)
	}

	return task.Status, nil
}

// GetValidationResult polls for a completed validation result
func (av *AsyncValidatorImpl) GetValidationResult(ctx context.Context, taskID string) (*ValidationTaskResult, error) {
	av.mu.RLock()
	result, exists := av.results[taskID]
	av.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("result not available: %s", taskID)
	}

	return result, nil
}

// WaitForValidationResult blocks until validation completes or timeout
func (av *AsyncValidatorImpl) WaitForValidationResult(ctx context.Context, taskID string, timeout time.Duration) (*ValidationTaskResult, error) {
	deadline := time.Now().Add(timeout)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-time.After(time.Until(deadline)):
			return nil, fmt.Errorf("validation timeout: %s", taskID)

		case <-ticker.C:
			result, err := av.GetValidationResult(ctx, taskID)
			if err == nil {
				return result, nil
			}
		}
	}
}

// ============================================================================
// Event Streaming
// ============================================================================

// SubscribeToValidationEvents returns a channel for validation events (Kafka-backed)
func (av *AsyncValidatorImpl) SubscribeToValidationEvents(ctx context.Context) (<-chan *ValidationEvent, error) {
	if av.resultWriter == nil {
		return nil, fmt.Errorf("result writer not configured")
	}

	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     fmt.Sprintf("async-validator-sse-%d", time.Now().UnixNano()),
		Topic:       "validation.results",
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})

	eventChan := make(chan *ValidationEvent, 100)
	go func() {
		defer close(eventChan)
		defer r.Close()
		for {
			m, err := r.FetchMessage(ctx)
			if err != nil {
				return
			}
			var event ValidationEvent
			if err := json.Unmarshal(m.Value, &event); err == nil {
				eventChan <- &event
			}
			r.CommitMessages(ctx, m)
		}
	}()

	return eventChan, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// ============================================================================
// Queue Processing (Worker)
// ============================================================================

// ProcessValidationQueue processes pending validation tasks
func (av *AsyncValidatorImpl) ProcessValidationQueue(ctx context.Context, workerID string) error {
	log.Printf("[AsyncValidator] Worker %s started", workerID)

	// Use Kafka reader for tasks with groupID = workerID
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  workerID,
		Topic:    "validation.tasks",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer r.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			m, err := r.FetchMessage(ctx)
			if err != nil {
				continue
			}
			av.processValidationTask(ctx, m.Value)
			if err := r.CommitMessages(ctx, m); err != nil {
				log.Printf("failed to commit message: %v", err)
			}
		}
	}
}

// processValidationTask processes a single validation task
func (av *AsyncValidatorImpl) processValidationTask(ctx context.Context, payload []byte) {
	var task ValidationTask
	err := json.Unmarshal(payload, &task)
	if err != nil {
		log.Printf("[AsyncValidator] Failed to unmarshal task: %v", err)
		return
	}

	log.Printf("[AsyncValidator] Processing task: %s", task.ID)

	// Update task status
	av.mu.Lock()
	if _, ok := av.tasks[task.ID]; ok {
		av.tasks[task.ID].Status = "processing"
	} else {
		// ensure task exists in memory
		av.tasks[task.ID] = &task
		av.tasks[task.ID].Status = "processing"
	}
	av.mu.Unlock()

	startTime := time.Now()

	// Simulate validation (in real implementation, call validation service)
	result := &ValidationTaskResult{
		TaskID:      task.ID,
		Status:      "valid",
		Errors:      []map[string]interface{}{},
		Warnings:    []map[string]interface{}{},
		Summary:     "Validation passed",
		ProcessedAt: time.Now(),
		Duration:    time.Since(startTime),
	}

	// Store result
	av.mu.Lock()
	av.results[task.ID] = result
	av.tasks[task.ID].Status = "completed"
	av.mu.Unlock()

	// Emit event
	event := &ValidationEvent{
		TaskID:     task.ID,
		EntityID:   task.EntityID,
		EntityType: task.EntityType,
		Status:     result.Status,
		Result:     result,
		Timestamp:  time.Now(),
	}

	av.publishValidationEvent(ctx, event)

	log.Printf("[AsyncValidator] Task completed: %s", task.ID)
}

// publishValidationEvent publishes a validation event
func (av *AsyncValidatorImpl) publishValidationEvent(ctx context.Context, event *ValidationEvent) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return
	}

	routingKey := fmt.Sprintf("validation.%s", event.Status)
	msg := kafka.Message{
		Topic: "validation.results",
		Key:   []byte(routingKey),
		Value: eventJSON,
		Time:  time.Now(),
	}
	if av.resultWriter != nil {
		av.resultWriter.WriteMessages(ctx, msg)
	}
}

// ============================================================================
// Monitoring
// ============================================================================

// GetQueueStats returns queue statistics
func (av *AsyncValidatorImpl) GetQueueStats(ctx context.Context) (map[string]interface{}, error) {
	av.mu.RLock()
	defer av.mu.RUnlock()

	pending := 0
	processing := 0
	completed := 0

	for _, task := range av.tasks {
		switch task.Status {
		case "pending":
			pending++
		case "processing":
			processing++
		case "completed":
			completed++
		}
	}

	// Kafka does not expose queue depth in the same way as AMQP; return best-effort counts
	return map[string]interface{}{
		"queue_depth":   -1,
		"consumers":     -1,
		"pending":       pending,
		"processing":    processing,
		"completed":     completed,
		"total_tasks":   len(av.tasks),
		"total_results": len(av.results),
	}, nil
}
