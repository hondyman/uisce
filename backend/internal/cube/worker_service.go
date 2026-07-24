package cube

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WorkerPool represents a pool of refresh workers
type WorkerPool struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	Name               string          `json:"name" db:"name"`
	DisplayName        string          `json:"display_name" db:"display_name"`
	Description        string          `json:"description" db:"description"`
	Tier               string          `json:"tier" db:"tier"`
	MinWorkers         int             `json:"min_workers" db:"min_workers"`
	MaxWorkers         int             `json:"max_workers" db:"max_workers"`
	CurrentWorkers     int             `json:"current_workers" db:"current_workers"`
	TargetWorkers      int             `json:"target_workers" db:"target_workers"`
	MemoryLimitMB      int             `json:"memory_limit_mb" db:"memory_limit_mb"`
	CPULimitCores      float64         `json:"cpu_limit_cores" db:"cpu_limit_cores"`
	ConcurrentJobs     int             `json:"concurrent_jobs" db:"concurrent_jobs"`
	QueueSize          int             `json:"queue_size" db:"queue_size"`
	AutoScaleEnabled   bool            `json:"auto_scale_enabled" db:"auto_scale_enabled"`
	ScaleUpThreshold   float64         `json:"scale_up_threshold" db:"scale_up_threshold"`
	ScaleDownThreshold float64         `json:"scale_down_threshold" db:"scale_down_threshold"`
	ScaleCooldownSecs  int             `json:"scale_cooldown_seconds" db:"scale_cooldown_seconds"`
	Status             string          `json:"status" db:"status"`
	LastScaleAt        *time.Time      `json:"last_scale_at" db:"last_scale_at"`
	HealthCheckAt      *time.Time      `json:"health_check_at" db:"health_check_at"`
	Metadata           json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

// WorkerInstance represents a running worker
type WorkerInstance struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	PoolID         uuid.UUID       `json:"pool_id" db:"pool_id"`
	InstanceID     string          `json:"instance_id" db:"instance_id"`
	Hostname       string          `json:"hostname" db:"hostname"`
	IPAddress      string          `json:"ip_address" db:"ip_address"`
	Status         string          `json:"status" db:"status"`
	CurrentJobID   *uuid.UUID      `json:"current_job_id" db:"current_job_id"`
	JobsCompleted  int             `json:"jobs_completed" db:"jobs_completed"`
	JobsFailed     int             `json:"jobs_failed" db:"jobs_failed"`
	MemoryUsedMB   int             `json:"memory_used_mb" db:"memory_used_mb"`
	CPUUsedPercent float64         `json:"cpu_used_percent" db:"cpu_used_percent"`
	StartedAt      time.Time       `json:"started_at" db:"started_at"`
	LastHeartbeat  time.Time       `json:"last_heartbeat_at" db:"last_heartbeat_at"`
	LastJobAt      *time.Time      `json:"last_job_at" db:"last_job_at"`
	Metadata       json.RawMessage `json:"metadata" db:"metadata"`
}

// PreAggDefinition represents a pre-aggregation configuration
type PreAggDefinition struct {
	ID                   uuid.UUID       `json:"id" db:"id"`
	TenantID             uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	DatasourceID         uuid.UUID       `json:"datasource_id" db:"datasource_id"`
	CubeName             string          `json:"cube_name" db:"cube_name"`
	PreAggName           string          `json:"preagg_name" db:"preagg_name"`
	Measures             []string        `json:"measures" db:"measures"`
	Dimensions           []string        `json:"dimensions" db:"dimensions"`
	TimeDimension        string          `json:"time_dimension" db:"time_dimension"`
	Granularity          string          `json:"granularity" db:"granularity"`
	PartitionGranularity string          `json:"partition_granularity" db:"partition_granularity"`
	RefreshKey           json.RawMessage `json:"refresh_key" db:"refresh_key"`
	ScheduledRefresh     bool            `json:"scheduled_refresh" db:"scheduled_refresh"`
	RefreshCron          string          `json:"refresh_cron" db:"refresh_cron"`
	RefreshIntervalMins  int             `json:"refresh_interval_minutes" db:"refresh_interval_minutes"`
	RefreshTimezone      string          `json:"refresh_timezone" db:"refresh_timezone"`
	ExternalStorage      bool            `json:"external_storage" db:"external_storage"`
	StorageEngine        string          `json:"storage_engine" db:"storage_engine"`
	TableName            string          `json:"table_name" db:"table_name"`
	Indexes              json.RawMessage `json:"indexes" db:"indexes"`
	BuildRangeStart      *time.Time      `json:"build_range_start" db:"build_range_start"`
	BuildRangeEnd        *time.Time      `json:"build_range_end" db:"build_range_end"`
	Priority             int             `json:"priority" db:"priority"`
	WorkerPoolID         *uuid.UUID      `json:"worker_pool_id" db:"worker_pool_id"`
	Status               string          `json:"status" db:"status"`
	LastBuildAt          *time.Time      `json:"last_build_at" db:"last_build_at"`
	LastBuildDurationMS  *int64          `json:"last_build_duration_ms" db:"last_build_duration_ms"`
	LastBuildRows        *int64          `json:"last_build_rows" db:"last_build_rows"`
	LastError            string          `json:"last_error" db:"last_error"`
	YAMLDefinition       string          `json:"yaml_definition" db:"yaml_definition"`
	Metadata             json.RawMessage `json:"metadata" db:"metadata"`
	CreatedBy            *uuid.UUID      `json:"created_by" db:"created_by"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
}

// PreAggJob represents a pre-aggregation build job
type PreAggJob struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	PreAggID         uuid.UUID       `json:"preagg_id" db:"preagg_id"`
	TenantID         uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	DatasourceID     uuid.UUID       `json:"datasource_id" db:"datasource_id"`
	JobType          string          `json:"job_type" db:"job_type"`
	PartitionKey     string          `json:"partition_key" db:"partition_key"`
	Priority         int             `json:"priority" db:"priority"`
	WorkerPoolID     *uuid.UUID      `json:"worker_pool_id" db:"worker_pool_id"`
	AssignedWorkerID *uuid.UUID      `json:"assigned_worker_id" db:"assigned_worker_id"`
	Status           string          `json:"status" db:"status"`
	ProgressPercent  int             `json:"progress_percent" db:"progress_percent"`
	CurrentStep      string          `json:"current_step" db:"current_step"`
	ScheduledAt      time.Time       `json:"scheduled_at" db:"scheduled_at"`
	QueuedAt         *time.Time      `json:"queued_at" db:"queued_at"`
	StartedAt        *time.Time      `json:"started_at" db:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at" db:"completed_at"`
	TimeoutAt        *time.Time      `json:"timeout_at" db:"timeout_at"`
	RowsProcessed    int64           `json:"rows_processed" db:"rows_processed"`
	BytesWritten     int64           `json:"bytes_written" db:"bytes_written"`
	DurationMS       *int64          `json:"duration_ms" db:"duration_ms"`
	RetryCount       int             `json:"retry_count" db:"retry_count"`
	MaxRetries       int             `json:"max_retries" db:"max_retries"`
	ErrorMessage     string          `json:"error_message" db:"error_message"`
	ErrorStack       string          `json:"error_stack" db:"error_stack"`
	BuildOptions     json.RawMessage `json:"build_options" db:"build_options"`
	ResultMetadata   json.RawMessage `json:"result_metadata" db:"result_metadata"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// PreAggPartition represents a partition of a pre-aggregation
type PreAggPartition struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	PreAggID        uuid.UUID  `json:"preagg_id" db:"preagg_id"`
	PartitionKey    string     `json:"partition_key" db:"partition_key"`
	Status          string     `json:"status" db:"status"`
	TableName       string     `json:"table_name" db:"table_name"`
	RowCount        int64      `json:"row_count" db:"row_count"`
	SizeBytes       int64      `json:"size_bytes" db:"size_bytes"`
	DataFrom        *time.Time `json:"data_from" db:"data_from"`
	DataTo          *time.Time `json:"data_to" db:"data_to"`
	BuiltAt         *time.Time `json:"built_at" db:"built_at"`
	ExpiresAt       *time.Time `json:"expires_at" db:"expires_at"`
	RefreshKeyValue string     `json:"refresh_key_value" db:"refresh_key_value"`
	BuildDurationMS *int64     `json:"build_duration_ms" db:"build_duration_ms"`
	LastError       string     `json:"last_error" db:"last_error"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// WorkerService provides worker management operations
type WorkerService struct {
	db *sql.DB
}

// NewWorkerService creates a new worker service
func NewWorkerService(db *sql.DB) *WorkerService {
	return &WorkerService{db: db}
}

// ListWorkerPools returns all worker pools
func (s *WorkerService) ListWorkerPools(ctx context.Context) ([]WorkerPool, error) {
	query := `
		SELECT id, name, display_name, description, tier,
			   min_workers, max_workers, current_workers, target_workers,
			   memory_limit_mb, cpu_limit_cores, concurrent_jobs, queue_size,
			   auto_scale_enabled, scale_up_threshold, scale_down_threshold, scale_cooldown_seconds,
			   status, last_scale_at, health_check_at, metadata, created_at, updated_at
		FROM cube_worker_pools
		ORDER BY tier, name`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list worker pools: %w", err)
	}
	defer rows.Close()

	var pools []WorkerPool
	for rows.Next() {
		var p WorkerPool
		err := rows.Scan(
			&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.Tier,
			&p.MinWorkers, &p.MaxWorkers, &p.CurrentWorkers, &p.TargetWorkers,
			&p.MemoryLimitMB, &p.CPULimitCores, &p.ConcurrentJobs, &p.QueueSize,
			&p.AutoScaleEnabled, &p.ScaleUpThreshold, &p.ScaleDownThreshold, &p.ScaleCooldownSecs,
			&p.Status, &p.LastScaleAt, &p.HealthCheckAt, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worker pool: %w", err)
		}
		pools = append(pools, p)
	}
	return pools, nil
}

// GetWorkerPool returns a specific worker pool
func (s *WorkerService) GetWorkerPool(ctx context.Context, poolID uuid.UUID) (*WorkerPool, error) {
	query := `
		SELECT id, name, display_name, description, tier,
			   min_workers, max_workers, current_workers, target_workers,
			   memory_limit_mb, cpu_limit_cores, concurrent_jobs, queue_size,
			   auto_scale_enabled, scale_up_threshold, scale_down_threshold, scale_cooldown_seconds,
			   status, last_scale_at, health_check_at, metadata, created_at, updated_at
		FROM cube_worker_pools
		WHERE id = $1`

	var p WorkerPool
	err := s.db.QueryRowContext(ctx, query, poolID).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Description, &p.Tier,
		&p.MinWorkers, &p.MaxWorkers, &p.CurrentWorkers, &p.TargetWorkers,
		&p.MemoryLimitMB, &p.CPULimitCores, &p.ConcurrentJobs, &p.QueueSize,
		&p.AutoScaleEnabled, &p.ScaleUpThreshold, &p.ScaleDownThreshold, &p.ScaleCooldownSecs,
		&p.Status, &p.LastScaleAt, &p.HealthCheckAt, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get worker pool: %w", err)
	}
	return &p, nil
}

// ScaleWorkerPool adjusts the target worker count
func (s *WorkerService) ScaleWorkerPool(ctx context.Context, poolID uuid.UUID, targetWorkers int) error {
	query := `
		UPDATE cube_worker_pools
		SET target_workers = $2, last_scale_at = NOW(), updated_at = NOW()
		WHERE id = $1`

	_, err := s.db.ExecContext(ctx, query, poolID, targetWorkers)
	if err != nil {
		return fmt.Errorf("failed to scale worker pool: %w", err)
	}
	return nil
}

// ListWorkerInstances returns workers in a pool
func (s *WorkerService) ListWorkerInstances(ctx context.Context, poolID uuid.UUID) ([]WorkerInstance, error) {
	query := `
		SELECT id, pool_id, instance_id, hostname, ip_address,
			   status, current_job_id, jobs_completed, jobs_failed,
			   memory_used_mb, cpu_used_percent, started_at, last_heartbeat_at, last_job_at, metadata
		FROM cube_worker_instances
		WHERE pool_id = $1
		ORDER BY started_at DESC`

	rows, err := s.db.QueryContext(ctx, query, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}
	defer rows.Close()

	var workers []WorkerInstance
	for rows.Next() {
		var w WorkerInstance
		err := rows.Scan(
			&w.ID, &w.PoolID, &w.InstanceID, &w.Hostname, &w.IPAddress,
			&w.Status, &w.CurrentJobID, &w.JobsCompleted, &w.JobsFailed,
			&w.MemoryUsedMB, &w.CPUUsedPercent, &w.StartedAt, &w.LastHeartbeat, &w.LastJobAt, &w.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worker: %w", err)
		}
		workers = append(workers, w)
	}
	return workers, nil
}

// RegisterWorker registers a new worker instance
func (s *WorkerService) RegisterWorker(ctx context.Context, poolID uuid.UUID, instanceID, hostname, ipAddress string) (*WorkerInstance, error) {
	id := uuid.New()
	query := `
		INSERT INTO cube_worker_instances (id, pool_id, instance_id, hostname, ip_address, status)
		VALUES ($1, $2, $3, $4, $5, 'starting')
		RETURNING id, pool_id, instance_id, hostname, ip_address, status, 
				  jobs_completed, jobs_failed, memory_used_mb, cpu_used_percent,
				  started_at, last_heartbeat_at`

	var w WorkerInstance
	err := s.db.QueryRowContext(ctx, query, id, poolID, instanceID, hostname, ipAddress).Scan(
		&w.ID, &w.PoolID, &w.InstanceID, &w.Hostname, &w.IPAddress, &w.Status,
		&w.JobsCompleted, &w.JobsFailed, &w.MemoryUsedMB, &w.CPUUsedPercent,
		&w.StartedAt, &w.LastHeartbeat,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}
	return &w, nil
}

// UpdateWorkerHeartbeat updates worker heartbeat and metrics
func (s *WorkerService) UpdateWorkerHeartbeat(ctx context.Context, workerID uuid.UUID, status string, memoryMB int, cpuPercent float64) error {
	query := `
		UPDATE cube_worker_instances
		SET status = $2, memory_used_mb = $3, cpu_used_percent = $4, last_heartbeat_at = NOW()
		WHERE id = $1`

	_, err := s.db.ExecContext(ctx, query, workerID, status, memoryMB, cpuPercent)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}
	return nil
}

// DeregisterWorker removes a worker instance
func (s *WorkerService) DeregisterWorker(ctx context.Context, workerID uuid.UUID) error {
	query := `DELETE FROM cube_worker_instances WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, workerID)
	if err != nil {
		return fmt.Errorf("failed to deregister worker: %w", err)
	}
	return nil
}

// ListPreAggDefinitions returns pre-aggregation definitions for a tenant
func (s *WorkerService) ListPreAggDefinitions(ctx context.Context, tenantID, datasourceID uuid.UUID) ([]PreAggDefinition, error) {
	query := `
		SELECT id, tenant_id, datasource_id, cube_name, preagg_name,
			   measures, dimensions, time_dimension, granularity, partition_granularity,
			   refresh_key, scheduled_refresh, refresh_cron, refresh_interval_minutes, refresh_timezone,
			   external_storage, storage_engine, table_name, indexes,
			   build_range_start, build_range_end, priority, worker_pool_id,
			   status, last_build_at, last_build_duration_ms, last_build_rows, last_error,
			   yaml_definition, metadata, created_by, created_at, updated_at
		FROM cube_preagg_definitions
		WHERE tenant_id = $1 AND datasource_id = $2
		ORDER BY cube_name, preagg_name`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pre-aggregations: %w", err)
	}
	defer rows.Close()

	var defs []PreAggDefinition
	for rows.Next() {
		var d PreAggDefinition
		err := rows.Scan(
			&d.ID, &d.TenantID, &d.DatasourceID, &d.CubeName, &d.PreAggName,
			&d.Measures, &d.Dimensions, &d.TimeDimension, &d.Granularity, &d.PartitionGranularity,
			&d.RefreshKey, &d.ScheduledRefresh, &d.RefreshCron, &d.RefreshIntervalMins, &d.RefreshTimezone,
			&d.ExternalStorage, &d.StorageEngine, &d.TableName, &d.Indexes,
			&d.BuildRangeStart, &d.BuildRangeEnd, &d.Priority, &d.WorkerPoolID,
			&d.Status, &d.LastBuildAt, &d.LastBuildDurationMS, &d.LastBuildRows, &d.LastError,
			&d.YAMLDefinition, &d.Metadata, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pre-aggregation: %w", err)
		}
		defs = append(defs, d)
	}
	return defs, nil
}

// CreatePreAggDefinition creates a new pre-aggregation definition
func (s *WorkerService) CreatePreAggDefinition(ctx context.Context, def *PreAggDefinition) error {
	def.ID = uuid.New()
	query := `
		INSERT INTO cube_preagg_definitions (
			id, tenant_id, datasource_id, cube_name, preagg_name,
			measures, dimensions, time_dimension, granularity, partition_granularity,
			refresh_key, scheduled_refresh, refresh_cron, refresh_interval_minutes, refresh_timezone,
			external_storage, storage_engine, priority, worker_pool_id, yaml_definition, metadata, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)`

	_, err := s.db.ExecContext(ctx, query,
		def.ID, def.TenantID, def.DatasourceID, def.CubeName, def.PreAggName,
		def.Measures, def.Dimensions, def.TimeDimension, def.Granularity, def.PartitionGranularity,
		def.RefreshKey, def.ScheduledRefresh, def.RefreshCron, def.RefreshIntervalMins, def.RefreshTimezone,
		def.ExternalStorage, def.StorageEngine, def.Priority, def.WorkerPoolID, def.YAMLDefinition, def.Metadata, def.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create pre-aggregation: %w", err)
	}
	return nil
}

// UpdatePreAggStatus updates pre-aggregation status after a build
func (s *WorkerService) UpdatePreAggStatus(ctx context.Context, preAggID uuid.UUID, status string, durationMS, rows int64, lastError string) error {
	query := `
		UPDATE cube_preagg_definitions
		SET status = $2, last_build_at = NOW(), last_build_duration_ms = $3, 
			last_build_rows = $4, last_error = $5, updated_at = NOW()
		WHERE id = $1`

	_, err := s.db.ExecContext(ctx, query, preAggID, status, durationMS, rows, lastError)
	if err != nil {
		return fmt.Errorf("failed to update pre-aggregation status: %w", err)
	}
	return nil
}

// EnqueueJob creates a new pre-aggregation job
func (s *WorkerService) EnqueueJob(ctx context.Context, job *PreAggJob) error {
	job.ID = uuid.New()
	query := `
		INSERT INTO cube_preagg_jobs (
			id, preagg_id, tenant_id, datasource_id, job_type, partition_key,
			priority, worker_pool_id, status, scheduled_at, max_retries, build_options
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending', $9, $10, $11)`

	_, err := s.db.ExecContext(ctx, query,
		job.ID, job.PreAggID, job.TenantID, job.DatasourceID, job.JobType, job.PartitionKey,
		job.Priority, job.WorkerPoolID, job.ScheduledAt, job.MaxRetries, job.BuildOptions,
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}
	return nil
}

// ClaimJob assigns a pending job to a worker
func (s *WorkerService) ClaimJob(ctx context.Context, workerID, poolID uuid.UUID) (*PreAggJob, error) {
	// Use CTE with FOR UPDATE SKIP LOCKED for concurrent-safe claiming
	query := `
		WITH next_job AS (
			SELECT id FROM cube_preagg_jobs
			WHERE status = 'pending'
			  AND (worker_pool_id IS NULL OR worker_pool_id = $2)
			ORDER BY priority DESC, scheduled_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE cube_preagg_jobs j
		SET status = 'running', assigned_worker_id = $1, started_at = NOW(), queued_at = NOW()
		FROM next_job
		WHERE j.id = next_job.id
		RETURNING j.id, j.preagg_id, j.tenant_id, j.datasource_id, j.job_type, j.partition_key,
				  j.priority, j.build_options`

	var job PreAggJob
	err := s.db.QueryRowContext(ctx, query, workerID, poolID).Scan(
		&job.ID, &job.PreAggID, &job.TenantID, &job.DatasourceID, &job.JobType, &job.PartitionKey,
		&job.Priority, &job.BuildOptions,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No job available
	}
	if err != nil {
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}
	job.AssignedWorkerID = &workerID
	job.Status = "running"
	return &job, nil
}

// CompleteJob marks a job as completed
func (s *WorkerService) CompleteJob(ctx context.Context, jobID uuid.UUID, rowsProcessed, bytesWritten int64, metadata json.RawMessage) error {
	query := `
		UPDATE cube_preagg_jobs
		SET status = 'completed', completed_at = NOW(), 
			duration_ms = EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000,
			rows_processed = $2, bytes_written = $3, result_metadata = $4, progress_percent = 100
		WHERE id = $1`

	_, err := s.db.ExecContext(ctx, query, jobID, rowsProcessed, bytesWritten, metadata)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}
	return nil
}

// FailJob marks a job as failed
func (s *WorkerService) FailJob(ctx context.Context, jobID uuid.UUID, errorMsg, errorStack string) error {
	query := `
		UPDATE cube_preagg_jobs
		SET status = CASE WHEN retry_count < max_retries THEN 'pending' ELSE 'failed' END,
			completed_at = NOW(),
			duration_ms = EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000,
			error_message = $2, error_stack = $3, retry_count = retry_count + 1,
			assigned_worker_id = NULL
		WHERE id = $1`

	_, err := s.db.ExecContext(ctx, query, jobID, errorMsg, errorStack)
	if err != nil {
		return fmt.Errorf("failed to fail job: %w", err)
	}
	return nil
}

// ListJobs returns jobs with optional filters
func (s *WorkerService) ListJobs(ctx context.Context, tenantID uuid.UUID, status string, limit int) ([]PreAggJob, error) {
	query := `
		SELECT id, preagg_id, tenant_id, datasource_id, job_type, partition_key,
			   priority, worker_pool_id, assigned_worker_id, status, progress_percent, current_step,
			   scheduled_at, queued_at, started_at, completed_at, timeout_at,
			   rows_processed, bytes_written, duration_ms, retry_count, max_retries,
			   error_message, build_options, result_metadata, created_at
		FROM cube_preagg_jobs
		WHERE tenant_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := s.db.QueryContext(ctx, query, tenantID, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []PreAggJob
	for rows.Next() {
		var j PreAggJob
		err := rows.Scan(
			&j.ID, &j.PreAggID, &j.TenantID, &j.DatasourceID, &j.JobType, &j.PartitionKey,
			&j.Priority, &j.WorkerPoolID, &j.AssignedWorkerID, &j.Status, &j.ProgressPercent, &j.CurrentStep,
			&j.ScheduledAt, &j.QueuedAt, &j.StartedAt, &j.CompletedAt, &j.TimeoutAt,
			&j.RowsProcessed, &j.BytesWritten, &j.DurationMS, &j.RetryCount, &j.MaxRetries,
			&j.ErrorMessage, &j.BuildOptions, &j.ResultMetadata, &j.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// GetJobQueueStats returns queue statistics
func (s *WorkerService) GetJobQueueStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'queued') as queued,
			COUNT(*) FILTER (WHERE status = 'running') as running,
			COUNT(*) FILTER (WHERE status = 'completed' AND completed_at > NOW() - INTERVAL '1 hour') as completed_1h,
			COUNT(*) FILTER (WHERE status = 'failed' AND completed_at > NOW() - INTERVAL '1 hour') as failed_1h,
			AVG(duration_ms) FILTER (WHERE status = 'completed' AND completed_at > NOW() - INTERVAL '1 hour') as avg_duration_ms
		FROM cube_preagg_jobs`

	var pending, queued, running, completed1h, failed1h int
	var avgDuration sql.NullFloat64
	err := s.db.QueryRowContext(ctx, query).Scan(&pending, &queued, &running, &completed1h, &failed1h, &avgDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	return map[string]interface{}{
		"pending":         pending,
		"queued":          queued,
		"running":         running,
		"completed_1h":    completed1h,
		"failed_1h":       failed1h,
		"avg_duration_ms": avgDuration.Float64,
	}, nil
}

// ListPartitions returns partitions for a pre-aggregation
func (s *WorkerService) ListPartitions(ctx context.Context, preAggID uuid.UUID) ([]PreAggPartition, error) {
	query := `
		SELECT id, preagg_id, partition_key, status, table_name, row_count, size_bytes,
			   data_from, data_to, built_at, expires_at, refresh_key_value,
			   build_duration_ms, last_error, created_at, updated_at
		FROM cube_preagg_partitions
		WHERE preagg_id = $1
		ORDER BY partition_key DESC`

	rows, err := s.db.QueryContext(ctx, query, preAggID)
	if err != nil {
		return nil, fmt.Errorf("failed to list partitions: %w", err)
	}
	defer rows.Close()

	var parts []PreAggPartition
	for rows.Next() {
		var p PreAggPartition
		err := rows.Scan(
			&p.ID, &p.PreAggID, &p.PartitionKey, &p.Status, &p.TableName, &p.RowCount, &p.SizeBytes,
			&p.DataFrom, &p.DataTo, &p.BuiltAt, &p.ExpiresAt, &p.RefreshKeyValue,
			&p.BuildDurationMS, &p.LastError, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan partition: %w", err)
		}
		parts = append(parts, p)
	}
	return parts, nil
}

// GetPreAggDefinition retrieves a pre-aggregation by ID
func (s *WorkerService) GetPreAggDefinition(ctx context.Context, defID uuid.UUID) (*PreAggDefinition, error) {
	query := `
		SELECT id, tenant_id, datasource_id, cube_name, preagg_name, measures, dimensions,
		       time_dimension, granularity, partition_granularity, refresh_key,
		       scheduled_refresh, refresh_cron, refresh_interval_minutes, refresh_timezone,
		       external_storage, storage_engine, table_name, indexes,
		       build_range_start, build_range_end, priority, worker_pool_id,
		       status, last_build_at, last_build_duration_ms, last_build_rows, last_error,
		       yaml_definition, metadata, created_by, created_at, updated_at
		FROM cube_preagg_definitions
		WHERE id = $1`

	var def PreAggDefinition
	var measures, dimensions []byte
	err := s.db.QueryRowContext(ctx, query, defID).Scan(
		&def.ID, &def.TenantID, &def.DatasourceID, &def.CubeName, &def.PreAggName,
		&measures, &dimensions,
		&def.TimeDimension, &def.Granularity, &def.PartitionGranularity, &def.RefreshKey,
		&def.ScheduledRefresh, &def.RefreshCron, &def.RefreshIntervalMins, &def.RefreshTimezone,
		&def.ExternalStorage, &def.StorageEngine, &def.TableName, &def.Indexes,
		&def.BuildRangeStart, &def.BuildRangeEnd, &def.Priority, &def.WorkerPoolID,
		&def.Status, &def.LastBuildAt, &def.LastBuildDurationMS, &def.LastBuildRows, &def.LastError,
		&def.YAMLDefinition, &def.Metadata, &def.CreatedBy, &def.CreatedAt, &def.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pre-aggregation definition not found: %s", defID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pre-aggregation definition: %w", err)
	}

	// Parse array fields
	_ = json.Unmarshal(measures, &def.Measures)
	_ = json.Unmarshal(dimensions, &def.Dimensions)

	return &def, nil
}

// UpdatePreAggDefinition updates a pre-aggregation definition
func (s *WorkerService) UpdatePreAggDefinition(ctx context.Context, def *PreAggDefinition) error {
	measures, _ := json.Marshal(def.Measures)
	dimensions, _ := json.Marshal(def.Dimensions)

	query := `
		UPDATE cube_preagg_definitions SET
			cube_name = $2, preagg_name = $3, measures = $4, dimensions = $5,
			time_dimension = $6, granularity = $7, partition_granularity = $8,
			refresh_key = $9, scheduled_refresh = $10, refresh_cron = $11,
			refresh_interval_minutes = $12, refresh_timezone = $13,
			external_storage = $14, storage_engine = $15, table_name = $16,
			indexes = $17, build_range_start = $18, build_range_end = $19,
			priority = $20, worker_pool_id = $21, yaml_definition = $22, metadata = $23,
			updated_at = NOW()
		WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query,
		def.ID, def.CubeName, def.PreAggName, measures, dimensions,
		def.TimeDimension, def.Granularity, def.PartitionGranularity,
		def.RefreshKey, def.ScheduledRefresh, def.RefreshCron,
		def.RefreshIntervalMins, def.RefreshTimezone,
		def.ExternalStorage, def.StorageEngine, def.TableName,
		def.Indexes, def.BuildRangeStart, def.BuildRangeEnd,
		def.Priority, def.WorkerPoolID, def.YAMLDefinition, def.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update pre-aggregation definition: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("pre-aggregation definition not found: %s", def.ID)
	}
	return nil
}

// DeletePreAggDefinition deletes a pre-aggregation definition
func (s *WorkerService) DeletePreAggDefinition(ctx context.Context, defID uuid.UUID) error {
	// First delete all partitions
	_, err := s.db.ExecContext(ctx, "DELETE FROM cube_preagg_partitions WHERE preagg_id = $1", defID)
	if err != nil {
		return fmt.Errorf("failed to delete partitions: %w", err)
	}

	// Delete all jobs
	_, err = s.db.ExecContext(ctx, "DELETE FROM cube_preagg_jobs WHERE preagg_id = $1", defID)
	if err != nil {
		return fmt.Errorf("failed to delete jobs: %w", err)
	}

	// Delete the definition
	result, err := s.db.ExecContext(ctx, "DELETE FROM cube_preagg_definitions WHERE id = $1", defID)
	if err != nil {
		return fmt.Errorf("failed to delete pre-aggregation definition: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("pre-aggregation definition not found: %s", defID)
	}
	return nil
}

// GetJob retrieves a job by ID
func (s *WorkerService) GetJob(ctx context.Context, jobID uuid.UUID) (*PreAggJob, error) {
	query := `
		SELECT id, preagg_id, tenant_id, datasource_id, job_type, partition_key,
		       priority, worker_pool_id, assigned_worker_id, status, progress_percent, current_step,
		       scheduled_at, queued_at, started_at, completed_at, timeout_at,
		       rows_processed, bytes_written, duration_ms, retry_count, max_retries,
		       error_message, error_stack, build_options, result_metadata, created_at
		FROM cube_preagg_jobs
		WHERE id = $1`

	var job PreAggJob
	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.PreAggID, &job.TenantID, &job.DatasourceID, &job.JobType, &job.PartitionKey,
		&job.Priority, &job.WorkerPoolID, &job.AssignedWorkerID, &job.Status, &job.ProgressPercent, &job.CurrentStep,
		&job.ScheduledAt, &job.QueuedAt, &job.StartedAt, &job.CompletedAt, &job.TimeoutAt,
		&job.RowsProcessed, &job.BytesWritten, &job.DurationMS, &job.RetryCount, &job.MaxRetries,
		&job.ErrorMessage, &job.ErrorStack, &job.BuildOptions, &job.ResultMetadata, &job.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	return &job, nil
}

// CancelJob cancels a pending or running job
func (s *WorkerService) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	query := `
		UPDATE cube_preagg_jobs SET
			status = 'cancelled',
			completed_at = NOW(),
			error_message = 'Cancelled by user',
			updated_at = NOW()
		WHERE id = $1 AND status IN ('pending', 'queued', 'running')`

	result, err := s.db.ExecContext(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("job not found or already completed: %s", jobID)
	}
	return nil
}

// RetryJob retries a failed job
func (s *WorkerService) RetryJob(ctx context.Context, jobID uuid.UUID) error {
	// First get the job to check status and retry count
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	if job.Status != "failed" && job.Status != "cancelled" {
		return fmt.Errorf("can only retry failed or cancelled jobs, current status: %s", job.Status)
	}

	// Reset job status and increment retry count
	query := `
		UPDATE cube_preagg_jobs SET
			status = 'pending',
			retry_count = retry_count + 1,
			started_at = NULL,
			completed_at = NULL,
			worker_id = NULL,
			error_message = NULL,
			error_stack = NULL,
			updated_at = NOW()
		WHERE id = $1`

	_, err = s.db.ExecContext(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("failed to retry job: %w", err)
	}
	return nil
}
