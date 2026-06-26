package temporal

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// WorkerRegistry manages all regional priority-based workers
// For each (region, priority_tier) combination, maintains a dedicated worker pool
type WorkerRegistry struct {
	client      client.Client
	logger      *slog.Logger
	workers     map[string]worker.Worker
	mu          sync.RWMutex
	workerCount int
}

// NewWorkerRegistry creates a new worker registry
func NewWorkerRegistry(c client.Client, logger *slog.Logger) *WorkerRegistry {
	return &WorkerRegistry{
		client:  c,
		logger:  logger,
		workers: make(map[string]worker.Worker),
	}
}

// RegisterRegionalWorkers creates and registers workers for all priority tiers in a region
// This is called once per region during startup
func (wr *WorkerRegistry) RegisterRegionalWorkers(region string, pools map[PriorityTier]WorkerPoolConfig) error {
	normalized := normalizeRegion(region)

	wr.logger.Info("Registering regional workers",
		"region", normalized,
		"tiers", len(pools),
	)

	for tier, poolConfig := range pools {
		queueName := fmt.Sprintf("%s-%s-queue", normalized, tier)

		wr.logger.Info("Creating worker pool",
			"queue", queueName,
			"tier", tier,
			"max_workflows", poolConfig.MaxConcurrentWorkflows,
			"max_activities", poolConfig.MaxConcurrentActivities,
			"pollers", poolConfig.PollerCount,
		)

		// Create worker options from pool config
		opts := worker.Options{
			MaxConcurrentActivityExecutionSize:     poolConfig.MaxConcurrentActivities,
			MaxConcurrentWorkflowTaskExecutionSize: poolConfig.MaxConcurrentWorkflows,
			WorkerActivitiesPerSecond:              poolConfig.WorkerActivitiesPerSecond,
		}

		// Create worker for this queue
		w := worker.New(wr.client, queueName, opts)

		// Store worker
		wr.mu.Lock()
		wr.workers[queueName] = w
		wr.workerCount++
		wr.mu.Unlock()

		wr.logger.Info("Registered worker",
			"queue", queueName,
			"total_workers", wr.workerCount,
		)
	}

	return nil
}

// RegisterWorkflow registers a workflow to all workers
func (wr *WorkerRegistry) RegisterWorkflow(wf interface{}) {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	for queueName, w := range wr.workers {
		w.RegisterWorkflow(wf)
		wr.logger.Debug("Registered workflow", "queue", queueName)
	}
}

// RegisterActivity registers an activity to all workers
func (wr *WorkerRegistry) RegisterActivity(act interface{}) {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	for queueName, w := range wr.workers {
		w.RegisterActivity(act)
		wr.logger.Debug("Registered activity", "queue", queueName)
	}
}

// StartAll starts all registered workers
// Blocks until context is cancelled or error occurs
func (wr *WorkerRegistry) StartAll(ctx context.Context) error {
	wr.mu.RLock()
	workers := make([]worker.Worker, 0, len(wr.workers))
	queueNames := make([]string, 0, len(wr.workers))
	for queueName, w := range wr.workers {
		workers = append(workers, w)
		queueNames = append(queueNames, queueName)
	}
	wr.mu.RUnlock()

	if len(workers) == 0 {
		return fmt.Errorf("no workers registered")
	}

	wr.logger.Info("Starting all workers",
		"count", len(workers),
		"queues", len(queueNames),
	)

	// Start all workers in goroutines
	errorChan := make(chan error, len(workers))
	doneChan := make(chan struct{}, len(workers))

	for i, w := range workers {
		go func(workerIdx int, worker worker.Worker, queueName string) {
			wr.logger.Info("Worker goroutine starting", "queue", queueName, "index", workerIdx)
			if err := worker.Start(); err != nil {
				errorChan <- fmt.Errorf("worker %s failed: %w", queueName, err)
			}
			doneChan <- struct{}{}
		}(i, w, queueNames[i])
	}

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		wr.logger.Info("Context cancelled, stopping workers")
		wr.StopAll()
		return ctx.Err()
	case err := <-errorChan:
		wr.logger.Error("Worker error", "err", err)
		wr.StopAll()
		return err
	}
}

// StopAll gracefully stops all workers
func (wr *WorkerRegistry) StopAll() {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	wr.logger.Info("Stopping all workers", "count", len(wr.workers))

	for queueName, w := range wr.workers {
		w.Stop()
		wr.logger.Debug("Stopped worker", "queue", queueName)
	}
}

// GetWorkerCount returns the total number of registered workers
func (wr *WorkerRegistry) GetWorkerCount() int {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	return wr.workerCount
}

// ListQueues returns all registered queue names
func (wr *WorkerRegistry) ListQueues() []string {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	queues := make([]string, 0, len(wr.workers))
	for queueName := range wr.workers {
		queues = append(queues, queueName)
	}
	return queues
}
