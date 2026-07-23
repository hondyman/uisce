package cube

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkerOrchestrator manages worker pools and autoscaling
type WorkerOrchestrator struct {
	db            *sql.DB
	workerService *WorkerService
	mu            sync.RWMutex
	pools         map[uuid.UUID]*ManagedPool
	stopCh        chan struct{}
	wg            sync.WaitGroup

	// Configuration
	autoscaleInterval    time.Duration
	healthCheckInterval  time.Duration
	jobCleanupInterval   time.Duration
	metricsRetention     time.Duration
	staleWorkerThreshold time.Duration
}

// ManagedPool represents a worker pool under orchestration
type ManagedPool struct {
	Pool              *WorkerPool
	LastMetrics       *PoolMetrics
	LastScaleCheck    time.Time
	ScalingInProgress bool
}

// PoolMetrics represents current metrics for a worker pool
type PoolMetrics struct {
	QueueDepth       int
	RunningJobs      int
	AvgJobDurationMS float64
	AvgCPUPercent    float64
	AvgMemoryMB      float64
	HealthyWorkers   int
	UnhealthyWorkers int
	IdleWorkers      int
	BusyWorkers      int
	Throughput1h     int // Jobs completed in last hour
	FailureRate1h    float64
	CollectedAt      time.Time
}

// AutoscaleDecision represents a scaling decision
type AutoscaleDecision struct {
	PoolID         uuid.UUID
	DecisionType   string // scale_up, scale_down, no_change
	TriggerReason  string
	CurrentWorkers int
	TargetWorkers  int
	Metrics        *PoolMetrics
}

// NewWorkerOrchestrator creates a new worker orchestrator
func NewWorkerOrchestrator(db *sql.DB, workerService *WorkerService) *WorkerOrchestrator {
	return &WorkerOrchestrator{
		db:                   db,
		workerService:        workerService,
		pools:                make(map[uuid.UUID]*ManagedPool),
		stopCh:               make(chan struct{}),
		autoscaleInterval:    30 * time.Second,
		healthCheckInterval:  10 * time.Second,
		jobCleanupInterval:   5 * time.Minute,
		metricsRetention:     7 * 24 * time.Hour,
		staleWorkerThreshold: 2 * time.Minute,
	}
}

// Start begins the orchestration loops
func (o *WorkerOrchestrator) Start(ctx context.Context) error {
	log.Println("[ORCHESTRATOR] Starting worker orchestrator")

	// Load initial pool state
	if err := o.loadPools(ctx); err != nil {
		return fmt.Errorf("failed to load worker pools: %w", err)
	}

	// Start background loops
	o.wg.Add(4)
	go o.autoscaleLoop(ctx)
	go o.healthCheckLoop(ctx)
	go o.jobCleanupLoop(ctx)
	go o.metricsCollectionLoop(ctx)

	return nil
}

// Stop gracefully stops the orchestrator
func (o *WorkerOrchestrator) Stop() {
	log.Println("[ORCHESTRATOR] Stopping worker orchestrator")
	close(o.stopCh)
	o.wg.Wait()
	log.Println("[ORCHESTRATOR] Worker orchestrator stopped")
}

// loadPools loads all worker pools from the database
func (o *WorkerOrchestrator) loadPools(ctx context.Context) error {
	pools, err := o.workerService.ListWorkerPools(ctx)
	if err != nil {
		return err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	for i := range pools {
		pool := &pools[i]
		o.pools[pool.ID] = &ManagedPool{
			Pool:           pool,
			LastScaleCheck: time.Now(),
		}
	}

	log.Printf("[ORCHESTRATOR] Loaded %d worker pools", len(pools))
	return nil
}

// autoscaleLoop periodically evaluates autoscaling decisions
func (o *WorkerOrchestrator) autoscaleLoop(ctx context.Context) {
	defer o.wg.Done()
	ticker := time.NewTicker(o.autoscaleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.evaluateAutoscaling(ctx)
		}
	}
}

// evaluateAutoscaling checks each pool and makes scaling decisions
func (o *WorkerOrchestrator) evaluateAutoscaling(ctx context.Context) {
	o.mu.RLock()
	poolsCopy := make([]*ManagedPool, 0, len(o.pools))
	for _, p := range o.pools {
		poolsCopy = append(poolsCopy, p)
	}
	o.mu.RUnlock()

	for _, mp := range poolsCopy {
		if !mp.Pool.AutoScaleEnabled {
			continue
		}

		// Check cooldown period
		cooldown := time.Duration(mp.Pool.ScaleCooldownSecs) * time.Second
		if time.Since(mp.LastScaleCheck) < cooldown {
			continue
		}

		// Collect metrics
		metrics, err := o.collectPoolMetrics(ctx, mp.Pool.ID)
		if err != nil {
			log.Printf("[ORCHESTRATOR] Failed to collect metrics for pool %s: %v", mp.Pool.Name, err)
			continue
		}

		// Make scaling decision
		decision := o.makeScalingDecision(mp.Pool, metrics)

		// Log and execute decision
		if decision.DecisionType != "no_change" {
			log.Printf("[ORCHESTRATOR] Pool %s: %s from %d to %d workers (reason: %s)",
				mp.Pool.Name, decision.DecisionType, decision.CurrentWorkers, decision.TargetWorkers, decision.TriggerReason)

			if err := o.executeScalingDecision(ctx, decision); err != nil {
				log.Printf("[ORCHESTRATOR] Failed to execute scaling for pool %s: %v", mp.Pool.Name, err)
			}
		}

		// Update last check time
		o.mu.Lock()
		if p, ok := o.pools[mp.Pool.ID]; ok {
			p.LastScaleCheck = time.Now()
			p.LastMetrics = metrics
		}
		o.mu.Unlock()
	}
}

// collectPoolMetrics gathers current metrics for a worker pool
func (o *WorkerOrchestrator) collectPoolMetrics(ctx context.Context, poolID uuid.UUID) (*PoolMetrics, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN j.status = 'pending' THEN 1 ELSE 0 END), 0) as queue_depth,
			COALESCE(SUM(CASE WHEN j.status = 'running' THEN 1 ELSE 0 END), 0) as running_jobs,
			COALESCE(AVG(j.duration_ms) FILTER (WHERE j.status = 'completed' AND j.completed_at > NOW() - INTERVAL '1 hour'), 0) as avg_duration_ms,
			COALESCE(AVG(w.cpu_used_percent), 0) as avg_cpu,
			COALESCE(AVG(w.memory_used_mb), 0) as avg_memory,
			COALESCE(SUM(CASE WHEN w.status = 'idle' THEN 1 ELSE 0 END), 0) as idle_workers,
			COALESCE(SUM(CASE WHEN w.status = 'busy' THEN 1 ELSE 0 END), 0) as busy_workers,
			COALESCE(SUM(CASE WHEN w.status = 'healthy' OR w.status = 'idle' OR w.status = 'busy' THEN 1 ELSE 0 END), 0) as healthy_workers,
			COALESCE(SUM(CASE WHEN w.status = 'unhealthy' THEN 1 ELSE 0 END), 0) as unhealthy_workers,
			COALESCE(COUNT(j2.id) FILTER (WHERE j2.status = 'completed' AND j2.completed_at > NOW() - INTERVAL '1 hour'), 0) as throughput_1h,
			COALESCE(
				COUNT(j2.id) FILTER (WHERE j2.status = 'failed' AND j2.completed_at > NOW() - INTERVAL '1 hour')::float /
				NULLIF(COUNT(j2.id) FILTER (WHERE j2.completed_at > NOW() - INTERVAL '1 hour'), 0),
				0
			) as failure_rate_1h
		FROM cube_worker_pools p
		LEFT JOIN cube_preagg_jobs j ON p.id = j.worker_pool_id AND j.status IN ('pending', 'running')
		LEFT JOIN cube_preagg_jobs j2 ON p.id = j2.worker_pool_id
		LEFT JOIN cube_worker_instances w ON p.id = w.pool_id
		WHERE p.id = $1
		GROUP BY p.id`

	var m PoolMetrics
	err := o.db.QueryRowContext(ctx, query, poolID).Scan(
		&m.QueueDepth, &m.RunningJobs, &m.AvgJobDurationMS,
		&m.AvgCPUPercent, &m.AvgMemoryMB,
		&m.IdleWorkers, &m.BusyWorkers, &m.HealthyWorkers, &m.UnhealthyWorkers,
		&m.Throughput1h, &m.FailureRate1h,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	m.CollectedAt = time.Now()
	return &m, nil
}

// makeScalingDecision determines if scaling is needed
func (o *WorkerOrchestrator) makeScalingDecision(pool *WorkerPool, metrics *PoolMetrics) *AutoscaleDecision {
	decision := &AutoscaleDecision{
		PoolID:         pool.ID,
		DecisionType:   "no_change",
		CurrentWorkers: pool.CurrentWorkers,
		TargetWorkers:  pool.CurrentWorkers,
		Metrics:        metrics,
	}

	totalWorkers := metrics.HealthyWorkers
	if totalWorkers == 0 {
		totalWorkers = pool.CurrentWorkers
	}

	// Calculate utilization based on queue depth and running jobs
	queueUtilization := float64(metrics.QueueDepth+metrics.RunningJobs) / float64(pool.ConcurrentJobs*max(totalWorkers, 1))
	cpuUtilization := metrics.AvgCPUPercent / 100.0

	// Use the higher of queue or CPU utilization
	effectiveUtilization := maxFloat64(queueUtilization, cpuUtilization)

	// Scale up conditions
	if effectiveUtilization >= pool.ScaleUpThreshold && pool.CurrentWorkers < pool.MaxWorkers {
		// Calculate target based on queue depth
		neededCapacity := float64(metrics.QueueDepth+metrics.RunningJobs) / pool.ScaleUpThreshold
		targetWorkers := int(neededCapacity/float64(pool.ConcurrentJobs)) + 1
		targetWorkers = min(targetWorkers, pool.MaxWorkers)
		targetWorkers = max(targetWorkers, pool.CurrentWorkers+1) // Scale up by at least 1

		decision.DecisionType = "scale_up"
		decision.TargetWorkers = targetWorkers
		decision.TriggerReason = fmt.Sprintf("utilization %.1f%% >= threshold %.1f%%", effectiveUtilization*100, pool.ScaleUpThreshold*100)
		return decision
	}

	// Scale down conditions
	if effectiveUtilization <= pool.ScaleDownThreshold && pool.CurrentWorkers > pool.MinWorkers {
		// Only scale down if we have idle workers
		if metrics.IdleWorkers > 0 {
			targetWorkers := max(pool.CurrentWorkers-metrics.IdleWorkers, pool.MinWorkers)
			targetWorkers = max(targetWorkers, pool.CurrentWorkers-1) // Scale down by at most 1 at a time

			decision.DecisionType = "scale_down"
			decision.TargetWorkers = targetWorkers
			decision.TriggerReason = fmt.Sprintf("utilization %.1f%% <= threshold %.1f%%, %d idle workers",
				effectiveUtilization*100, pool.ScaleDownThreshold*100, metrics.IdleWorkers)
			return decision
		}
	}

	return decision
}

// executeScalingDecision executes the scaling decision
func (o *WorkerOrchestrator) executeScalingDecision(ctx context.Context, decision *AutoscaleDecision) error {
	// Record the decision
	metricsJSON, _ := json.Marshal(decision.Metrics)
	_, err := o.db.ExecContext(ctx, `
		INSERT INTO cube_autoscale_decisions 
		(worker_pool_id, decision_type, trigger_reason, current_workers, target_workers, metrics_snapshot, executed)
		VALUES ($1, $2, $3, $4, $5, $6, true)`,
		decision.PoolID, decision.DecisionType, decision.TriggerReason,
		decision.CurrentWorkers, decision.TargetWorkers, metricsJSON)
	if err != nil {
		log.Printf("[ORCHESTRATOR] Failed to record autoscale decision: %v", err)
	}

	// Update target workers in pool
	return o.workerService.ScaleWorkerPool(ctx, decision.PoolID, decision.TargetWorkers)
}

// healthCheckLoop monitors worker health
func (o *WorkerOrchestrator) healthCheckLoop(ctx context.Context) {
	defer o.wg.Done()
	ticker := time.NewTicker(o.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.checkWorkerHealth(ctx)
		}
	}
}

// checkWorkerHealth marks stale workers as unhealthy
func (o *WorkerOrchestrator) checkWorkerHealth(ctx context.Context) {
	// Mark workers as unhealthy if heartbeat is stale
	result, err := o.db.ExecContext(ctx, `
		UPDATE cube_worker_instances
		SET status = 'unhealthy'
		WHERE status IN ('idle', 'busy', 'healthy')
		  AND last_heartbeat_at < NOW() - $1::interval`,
		fmt.Sprintf("%d seconds", int(o.staleWorkerThreshold.Seconds())))
	if err != nil {
		log.Printf("[ORCHESTRATOR] Failed to update stale workers: %v", err)
		return
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		log.Printf("[ORCHESTRATOR] Marked %d workers as unhealthy (stale heartbeat)", affected)
	}

	// Remove workers that have been unhealthy for too long
	result, err = o.db.ExecContext(ctx, `
		DELETE FROM cube_worker_instances
		WHERE status = 'unhealthy'
		  AND last_heartbeat_at < NOW() - INTERVAL '10 minutes'`)
	if err != nil {
		log.Printf("[ORCHESTRATOR] Failed to remove dead workers: %v", err)
		return
	}

	affected, _ = result.RowsAffected()
	if affected > 0 {
		log.Printf("[ORCHESTRATOR] Removed %d dead workers", affected)
	}

	// Update pool current worker counts
	_, err = o.db.ExecContext(ctx, `
		UPDATE cube_worker_pools p
		SET current_workers = (
			SELECT COUNT(*) FROM cube_worker_instances w
			WHERE w.pool_id = p.id AND w.status IN ('idle', 'busy', 'healthy')
		), health_check_at = NOW()`)
	if err != nil {
		log.Printf("[ORCHESTRATOR] Failed to update pool worker counts: %v", err)
	}
}

// jobCleanupLoop cleans up old jobs
func (o *WorkerOrchestrator) jobCleanupLoop(ctx context.Context) {
	defer o.wg.Done()
	ticker := time.NewTicker(o.jobCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.cleanupJobs(ctx)
		}
	}
}

// cleanupJobs handles stuck and expired jobs
func (o *WorkerOrchestrator) cleanupJobs(ctx context.Context) {
	// Reset stuck jobs (running too long)
	result, err := o.db.ExecContext(ctx, `
		UPDATE cube_preagg_jobs
		SET status = CASE WHEN retry_count < max_retries THEN 'pending' ELSE 'failed' END,
			assigned_worker_id = NULL,
			error_message = 'Job timeout - reset for retry',
			retry_count = retry_count + 1
		WHERE status = 'running'
		  AND started_at < NOW() - INTERVAL '2 hours'`)
	if err != nil {
		log.Printf("[ORCHESTRATOR] Failed to reset stuck jobs: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		if affected > 0 {
			log.Printf("[ORCHESTRATOR] Reset %d stuck jobs", affected)
		}
	}

	// Release jobs assigned to dead workers
	result, err = o.db.ExecContext(ctx, `
		UPDATE cube_preagg_jobs j
		SET status = 'pending',
			assigned_worker_id = NULL,
			error_message = 'Worker died - reset for retry'
		WHERE j.status = 'running'
		  AND j.assigned_worker_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM cube_worker_instances w
			WHERE w.id = j.assigned_worker_id
			  AND w.status IN ('idle', 'busy', 'healthy')
		  )`)
	if err != nil {
		log.Printf("[ORCHESTRATOR] Failed to release orphaned jobs: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		if affected > 0 {
			log.Printf("[ORCHESTRATOR] Released %d orphaned jobs", affected)
		}
	}
}

// metricsCollectionLoop collects metrics for historical analysis
func (o *WorkerOrchestrator) metricsCollectionLoop(ctx context.Context) {
	defer o.wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.recordPoolMetrics(ctx)
		}
	}
}

// recordPoolMetrics records current metrics to the time-series table
func (o *WorkerOrchestrator) recordPoolMetrics(ctx context.Context) {
	o.mu.RLock()
	poolsCopy := make([]*ManagedPool, 0, len(o.pools))
	for _, p := range o.pools {
		poolsCopy = append(poolsCopy, p)
	}
	o.mu.RUnlock()

	for _, mp := range poolsCopy {
		metrics, err := o.collectPoolMetrics(ctx, mp.Pool.ID)
		if err != nil {
			continue
		}

		// Store metrics for baselines
		hour := time.Now().Hour()
		_, err = o.db.ExecContext(ctx, `
			INSERT INTO cube_worker_baselines (worker_pool_id, metric_type, time_bucket, bucket_key, avg_value, sample_count, last_updated_at)
			VALUES 
				($1, 'queue_depth', 'hour', $2, $3, 1, NOW()),
				($1, 'throughput', 'hour', $2, $4, 1, NOW()),
				($1, 'cpu_percent', 'hour', $2, $5, 1, NOW()),
				($1, 'memory_mb', 'hour', $2, $6, 1, NOW())
			ON CONFLICT (worker_pool_id, metric_type, time_bucket, bucket_key)
			DO UPDATE SET
				avg_value = (cube_worker_baselines.avg_value * cube_worker_baselines.sample_count + EXCLUDED.avg_value) / (cube_worker_baselines.sample_count + 1),
				sample_count = cube_worker_baselines.sample_count + 1,
				last_updated_at = NOW()`,
			mp.Pool.ID, fmt.Sprintf("%d", hour),
			float64(metrics.QueueDepth), float64(metrics.Throughput1h),
			metrics.AvgCPUPercent, metrics.AvgMemoryMB)
		if err != nil {
			log.Printf("[ORCHESTRATOR] Failed to record metrics for pool %s: %v", mp.Pool.Name, err)
		}
	}
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
