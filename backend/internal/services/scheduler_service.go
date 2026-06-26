package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/tenant"
	"github.com/robfig/cron/v3"
)

// ScheduleType represents the type of schedule
type ScheduleType string

const (
	ScheduleTypeOnce    ScheduleType = "once"
	ScheduleTypeDaily   ScheduleType = "daily"
	ScheduleTypeWeekly  ScheduleType = "weekly"
	ScheduleTypeMonthly ScheduleType = "monthly"
	ScheduleCron        ScheduleType = "cron"
)

// ScheduleStatus represents schedule status
type ScheduleStatus string

const (
	ScheduleStatusActive    ScheduleStatus = "active"
	ScheduleStatusPaused    ScheduleStatus = "paused"
	ScheduleStatusCompleted ScheduleStatus = "completed"
	ScheduleStatusFailed    ScheduleStatus = "failed"
	ScheduleStatusDisabled  ScheduleStatus = "disabled"
)

// ScheduledJob represents a scheduled job
type ScheduledJob struct {
	ID             uuid.UUID              `json:"id"`
	TenantID       uuid.UUID              `json:"tenant_id"`
	OperationType  string                 `json:"operation_type"`
	JobTemplate    map[string]interface{} `json:"job_template"`
	ScheduleType   ScheduleType           `json:"schedule_type"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        *time.Time             `json:"end_time,omitempty"`
	CronExpression string                 `json:"cron_expression,omitempty"`
	Timezone       string                 `json:"timezone"`
	MaxRunDuration int                    `json:"max_run_duration,omitempty"`
	RetryOnFailure bool                   `json:"retry_on_failure"`
	MaxRetries     int                    `json:"max_retries"`
	Status         ScheduleStatus         `json:"status"`
	IsActive       bool                   `json:"is_active"`
	LastRunAt      *time.Time             `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time             `json:"next_run_at,omitempty"`
	RunCount       int                    `json:"run_count"`
	SuccessCount   int                    `json:"success_count"`
	FailureCount   int                    `json:"failure_count"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Priority       int                    `json:"priority"`
	CreatedBy      uuid.UUID              `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// SchedulerService manages scheduled job execution
type SchedulerService interface {
	// CreateSchedule creates a new scheduled job
	CreateSchedule(ctx context.Context, job *ScheduledJob) (uuid.UUID, error)

	// GetSchedule retrieves a scheduled job
	GetSchedule(ctx context.Context, scheduleID uuid.UUID) (*ScheduledJob, error)

	// ListSchedules lists all schedules for a tenant
	ListSchedules(ctx context.Context, tenantID uuid.UUID) ([]*ScheduledJob, error)

	// UpdateSchedule updates a scheduled job
	UpdateSchedule(ctx context.Context, job *ScheduledJob) error

	// PauseSchedule pauses execution of a schedule
	PauseSchedule(ctx context.Context, scheduleID uuid.UUID) error

	// ResumeSchedule resumes execution
	ResumeSchedule(ctx context.Context, scheduleID uuid.UUID) error

	// DeleteSchedule removes a schedule
	DeleteSchedule(ctx context.Context, scheduleID uuid.UUID) error

	// GetNextDueJobs gets jobs that are due to run
	GetNextDueJobs(ctx context.Context) ([]*ScheduledJob, error)

	// RecordRun records a scheduled job execution
	RecordRun(ctx context.Context, scheduleID uuid.UUID, jobID *uuid.UUID, status string, errorMsg string) error
}

// PostgresSchedulerService implements SchedulerService
type PostgresSchedulerService struct {
	db   *sql.DB
	cron *cron.Cron
}

// NewPostgresSchedulerService creates a new scheduler service
func NewPostgresSchedulerService(db *sql.DB) *PostgresSchedulerService {
	return &PostgresSchedulerService{
		db:   db,
		cron: cron.New(),
	}
}

// Start begins the scheduler background loop
func (s *PostgresSchedulerService) Start(ctx context.Context, jobQueue JobQueue) error {
	s.cron.Start()

	// Add cron entry to check and execute due jobs every minute
	_, err := s.cron.AddFunc("@every 1m", func() {
		s.executeDueJobs(context.Background(), jobQueue)
	})

	return err
}

// Stop stops the scheduler
func (s *PostgresSchedulerService) Stop() {
	s.cron.Stop()
}

// CreateSchedule creates a new scheduled job
func (s *PostgresSchedulerService) CreateSchedule(ctx context.Context, job *ScheduledJob) (uuid.UUID, error) {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}

	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = time.Now()
	}

	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to extract tenant: %w", err)
	}

	// Calculate next run time
	nextRun, err := s.calculateNextRunTime(job.ScheduleType, job.StartTime, job.CronExpression, job.Timezone)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to calculate next run time: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return uuid.Nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	// Convert job template to JSON
	templateJSON, err := json.Marshal(job.JobTemplate)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal job template: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO edm.scheduled_jobs (
			id, tenant_id, operation_type, job_template, schedule_type, start_time,
			end_time, cron_expression, timezone, max_run_duration, retry_on_failure,
			max_retries, status, is_active, name, description, priority, created_by,
			next_run_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW(), NOW())
	`,
		job.ID, tenantID, job.OperationType, templateJSON, job.ScheduleType, job.StartTime,
		job.EndTime, job.CronExpression, job.Timezone, job.MaxRunDuration, job.RetryOnFailure,
		job.MaxRetries, ScheduleStatusActive, true, job.Name, job.Description, job.Priority,
		job.CreatedBy, nextRun,
	)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert schedule: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return job.ID, nil
}

// GetSchedule retrieves a scheduled job
func (s *PostgresSchedulerService) GetSchedule(ctx context.Context, scheduleID uuid.UUID) (*ScheduledJob, error) {
	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tenant: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	job := &ScheduledJob{}
	var templateJSON []byte

	err = tx.QueryRowContext(ctx, `
		SELECT id, tenant_id, operation_type, job_template, schedule_type, start_time,
		       end_time, cron_expression, timezone, max_run_duration, retry_on_failure,
		       max_retries, status, is_active, name, description, priority, created_by,
		       last_run_at, next_run_at, run_count, success_count, failure_count,
		       created_at, updated_at
		FROM edm.scheduled_jobs
		WHERE id = $1 AND tenant_id = $2
	`, scheduleID, tenantID).Scan(
		&job.ID, &job.TenantID, &job.OperationType, &templateJSON, &job.ScheduleType,
		&job.StartTime, &job.EndTime, &job.CronExpression, &job.Timezone, &job.MaxRunDuration,
		&job.RetryOnFailure, &job.MaxRetries, &job.Status, &job.IsActive, &job.Name,
		&job.Description, &job.Priority, &job.CreatedBy, &job.LastRunAt, &job.NextRunAt,
		&job.RunCount, &job.SuccessCount, &job.FailureCount, &job.CreatedAt, &job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query schedule: %w", err)
	}

	// Unmarshal job template
	if templateJSON != nil {
		json.Unmarshal(templateJSON, &job.JobTemplate)
	}

	tx.Commit()
	return job, nil
}

// ListSchedules lists all schedules for a tenant
func (s *PostgresSchedulerService) ListSchedules(ctx context.Context, tenantID uuid.UUID) ([]*ScheduledJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, tenant_id, operation_type, schedule_type, status, is_active,
		       name, priority, created_at, next_run_at, run_count, success_count
		FROM edm.scheduled_jobs
		WHERE tenant_id = $1 AND status != 'deleted'
		ORDER BY is_active DESC, next_run_at ASC
	`, tenantID)

	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	schedules := []*ScheduledJob{}
	for rows.Next() {
		job := &ScheduledJob{}
		err := rows.Scan(
			&job.ID, &job.TenantID, &job.OperationType, &job.ScheduleType, &job.Status,
			&job.IsActive, &job.Name, &job.Priority, &job.CreatedAt, &job.NextRunAt,
			&job.RunCount, &job.SuccessCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, job)
	}

	tx.Commit()
	return schedules, nil
}

// PauseSchedule pauses a scheduled job
func (s *PostgresSchedulerService) PauseSchedule(ctx context.Context, scheduleID uuid.UUID) error {
	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to extract tenant: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return fmt.Errorf("failed to set RLS context: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE edm.scheduled_jobs
		SET status = $1, is_active = false, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, ScheduleStatusPaused, scheduleID, tenantID)

	if err != nil {
		return fmt.Errorf("failed to pause schedule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return tx.Commit()
}

// ResumeSchedule resumes a scheduled job
func (s *PostgresSchedulerService) ResumeSchedule(ctx context.Context, scheduleID uuid.UUID) error {
	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to extract tenant: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return fmt.Errorf("failed to set RLS context: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE edm.scheduled_jobs
		SET status = $1, is_active = true, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, ScheduleStatusActive, scheduleID, tenantID)

	if err != nil {
		return fmt.Errorf("failed to resume schedule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return tx.Commit()
}

// DeleteSchedule deletes a scheduled job
func (s *PostgresSchedulerService) DeleteSchedule(ctx context.Context, scheduleID uuid.UUID) error {
	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to extract tenant: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return fmt.Errorf("failed to set RLS context: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM edm.scheduled_jobs
		WHERE id = $1 AND tenant_id = $2
	`, scheduleID, tenantID)

	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return tx.Commit()
}

// UpdateSchedule updates a scheduled job
func (s *PostgresSchedulerService) UpdateSchedule(ctx context.Context, job *ScheduledJob) error {
	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to extract tenant: %w", err)
	}

	job.UpdatedAt = time.Now()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return fmt.Errorf("failed to set RLS context: %w", err)
	}

	templateJSON, _ := json.Marshal(job.JobTemplate)

	_, err = tx.ExecContext(ctx, `
		UPDATE edm.scheduled_jobs
		SET name = $1, description = $2, status = $3, priority = $4,
		    job_template = $5, updated_at = NOW()
		WHERE id = $6 AND tenant_id = $7
	`, job.Name, job.Description, job.Status, job.Priority, templateJSON, job.ID, tenantID)

	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	return tx.Commit()
}

// GetNextDueJobs gets jobs due to run in the next minute
func (s *PostgresSchedulerService) GetNextDueJobs(ctx context.Context) ([]*ScheduledJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, operation_type, job_template, schedule_type,
		       start_time, cron_expression, timezone, name, priority, created_by
		FROM edm.next_scheduled_jobs
		LIMIT 100
	`)

	if err != nil {
		return nil, fmt.Errorf("failed to query next due jobs: %w", err)
	}
	defer rows.Close()

	jobs := []*ScheduledJob{}
	for rows.Next() {
		job := &ScheduledJob{}
		var templateJSON []byte

		err := rows.Scan(
			&job.ID, &job.TenantID, &job.OperationType, &templateJSON, &job.ScheduleType,
			&job.StartTime, &job.CronExpression, &job.Timezone, &job.Name, &job.Priority, &job.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		if templateJSON != nil {
			json.Unmarshal(templateJSON, &job.JobTemplate)
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// RecordRun records a scheduled job execution
func (s *PostgresSchedulerService) RecordRun(ctx context.Context, scheduleID uuid.UUID, jobID *uuid.UUID, status string, errorMsg string) error {
	tenantID, _ := tenant.ExtractTenantFromContext(ctx)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tenant.SetRLSContext(ctx, tx, tenantID.String())

	_, err = tx.ExecContext(ctx, `
		SELECT edm.record_scheduled_run($1, $2, $3, $4)
	`, scheduleID, jobID, status, errorMsg)

	return tx.Commit()
}

// Helper functions

func (s *PostgresSchedulerService) calculateNextRunTime(scheduleType ScheduleType, startTime time.Time, cronExpr string, timezone string) (*time.Time, error) {
	loc := time.UTC
	if tz, err := time.LoadLocation(timezone); err == nil {
		loc = tz
	}

	var nextRun time.Time

	switch scheduleType {
	case ScheduleTypeOnce:
		nextRun = startTime
	case ScheduleTypeDaily:
		nextRun = startTime.AddDate(0, 0, 1)
	case ScheduleTypeWeekly:
		nextRun = startTime.AddDate(0, 0, 7)
	case ScheduleTypeMonthly:
		nextRun = startTime.AddDate(0, 1, 0)
	case ScheduleCron:
		if cronExpr == "" {
			return nil, fmt.Errorf("cron expression required for cron schedule type")
		}
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(cronExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid cron expression: %w", err)
		}
		nextRun = schedule.Next(time.Now().In(loc))
	default:
		return nil, fmt.Errorf("unknown schedule type: %s", scheduleType)
	}

	return &nextRun, nil
}

func (s *PostgresSchedulerService) executeDueJobs(ctx context.Context, jobQueue JobQueue) {
	jobs, err := s.GetNextDueJobs(ctx)
	if err != nil {
		return
	}

	for _, job := range jobs {
		payload, _ := json.Marshal(job.JobTemplate)
		_, err := jobQueue.Enqueue(ctx, &models.AsyncJob{
			TenantID:      job.TenantID.String(),
			OperationType: models.OperationType(job.OperationType),
			Payload:       payload,
			CreatedBy:     job.CreatedBy.String(),
		})

		if err != nil {
			s.RecordRun(ctx, job.ID, nil, "failed", err.Error())
			continue
		}
	}
}
