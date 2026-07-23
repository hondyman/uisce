package scheduler_intelligence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository provides data access for scheduler intelligence
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new scheduler intelligence repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// Job CRUD
// ============================================================================

// CreateJob creates a new scheduled job
func (r *Repository) CreateJob(ctx context.Context, job *Job) error {
	query := `
		INSERT INTO scheduled_jobs (
			id, scope, tenant_id, parent_job_id, datasource_id, name, description, category, job_type,
			parameters, semantic_bindings, schedule_type, cron_expression, event_trigger,
			timezone, calendar_ids, blackout_windows, constraints, retry_policy,
			timeout_seconds, priority, risk_score, slo_critical, compliance_tags,
			pii_exposure_level, residency_rules, changeset_id, is_active, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29
		)
		RETURNING created_at, updated_at
	`

	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}

	return r.db.QueryRowxContext(ctx, query,
		job.ID, job.Scope, job.TenantID, job.ParentJobID, job.DatasourceID, job.Name, job.Description, job.Category, job.JobType,
		job.Parameters, job.SemanticBindings, job.ScheduleType, job.CronExpression, job.EventTrigger,
		job.Timezone, job.CalendarIDs, job.BlackoutWindows, job.Constraints, job.RetryPolicy,
		job.TimeoutSeconds, job.Priority, job.RiskScore, job.SLOCritical, job.ComplianceTags,
		job.PIIExposureLevel, job.ResidencyRules, job.ChangeSetID, job.IsActive, job.CreatedBy,
	).Scan(&job.CreatedAt, &job.UpdatedAt)
}

// GetJob retrieves a job by ID
func (r *Repository) GetJob(ctx context.Context, id uuid.UUID) (*Job, error) {
	var job Job
	query := `SELECT * FROM scheduled_jobs WHERE id = $1`
	err := r.db.GetContext(ctx, &job, query, id)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// GetJobByName retrieves a job by tenant and name (or global name)
func (r *Repository) GetJobByName(ctx context.Context, tenantID *uuid.UUID, name string) (*Job, error) {
	var job Job
	var query string
	var err error
	if tenantID == nil {
		query = `SELECT * FROM scheduled_jobs WHERE scope = 'GLOBAL' AND name = $1`
		err = r.db.GetContext(ctx, &job, query, name)
	} else {
		query = `SELECT * FROM scheduled_jobs WHERE tenant_id = $1 AND name = $2`
		err = r.db.GetContext(ctx, &job, query, tenantID, name)
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateJob updates an existing job
func (r *Repository) UpdateJob(ctx context.Context, job *Job) error {
	query := `
		UPDATE scheduled_jobs SET
			name = $2, description = $3, category = $4, job_type = $5,
			parameters = $6, semantic_bindings = $7, schedule_type = $8,
			cron_expression = $9, event_trigger = $10, timezone = $11,
			calendar_ids = $12, blackout_windows = $13, constraints = $14,
			retry_policy = $15, timeout_seconds = $16, priority = $17,
			risk_score = $18, slo_critical = $19, compliance_tags = $20,
			pii_exposure_level = $21, residency_rules = $22, is_active = $23,
			next_run_at = $24
		WHERE id = $1
		RETURNING updated_at
	`

	return r.db.QueryRowxContext(ctx, query,
		job.ID, job.Name, job.Description, job.Category, job.JobType,
		job.Parameters, job.SemanticBindings, job.ScheduleType,
		job.CronExpression, job.EventTrigger, job.Timezone,
		job.CalendarIDs, job.BlackoutWindows, job.Constraints,
		job.RetryPolicy, job.TimeoutSeconds, job.Priority,
		job.RiskScore, job.SLOCritical, job.ComplianceTags,
		job.PIIExposureLevel, job.ResidencyRules, job.IsActive,
		job.NextRunAt,
	).Scan(&job.UpdatedAt)
}

// DeleteJob soft-deletes a job by setting is_active to false
func (r *Repository) DeleteJob(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE scheduled_jobs SET is_active = false WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListJobs lists jobs with filters
func (r *Repository) ListJobs(ctx context.Context, filters JobListFilters) ([]Job, int, error) {
	var jobs []Job
	var total int

	baseQuery := `FROM scheduled_jobs WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filters.TenantID != "" {
		baseQuery += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, filters.TenantID)
		argIndex++
	}

	if filters.DatasourceID != "" {
		baseQuery += fmt.Sprintf(" AND datasource_id = $%d", argIndex)
		args = append(args, filters.DatasourceID)
		argIndex++
	}

	if filters.Scope != "" {
		baseQuery += fmt.Sprintf(" AND scope = $%d", argIndex)
		args = append(args, filters.Scope)
		argIndex++
	}

	if filters.Category != "" {
		baseQuery += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, filters.Category)
		argIndex++
	}

	if filters.IsActive != nil {
		baseQuery += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filters.IsActive)
		argIndex++
	}

	if filters.SLOCritical != nil {
		baseQuery += fmt.Sprintf(" AND slo_critical = $%d", argIndex)
		args = append(args, *filters.SLOCritical)
		argIndex++
	}

	// Get count
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get jobs
	selectQuery := "SELECT * " + baseQuery + " ORDER BY name"
	if filters.Limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}
	if filters.Offset > 0 {
		selectQuery += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	err = r.db.SelectContext(ctx, &jobs, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// UpdateJobLastRun updates the last run timestamp
func (r *Repository) UpdateJobLastRun(ctx context.Context, jobID uuid.UUID) error {
	query := `UPDATE scheduled_jobs SET last_run_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, jobID)
	return err
}

// GetTenantJobOverride looks for a tenant-specific override for a global job
func (r *Repository) GetTenantJobOverride(ctx context.Context, parentJobID uuid.UUID, tenantID uuid.UUID) (*Job, error) {
	var job Job
	query := `SELECT * FROM scheduled_jobs WHERE parent_job_id = $1 AND tenant_id = $2`
	err := r.db.GetContext(ctx, &job, query, parentJobID, tenantID)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// ============================================================================
// DAG CRUD
// ============================================================================

// CreateDAG creates a new DAG
func (r *Repository) CreateDAG(ctx context.Context, dag *DAG) error {
	query := `
		INSERT INTO scheduled_dags (
			id, scope, tenant_id, parent_dag_id, name, description, category, nodes, edges,
			semantic_bindings, schedule_type, cron_expression, calendar_ids, timezone,
			max_parallel_jobs, fail_fast, timeout_seconds,
			risk_score, slo_critical, changeset_id, is_active, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)
		RETURNING created_at, updated_at
	`

	if dag.ID == uuid.Nil {
		dag.ID = uuid.New()
	}

	return r.db.QueryRowxContext(ctx, query,
		dag.ID, dag.Scope, dag.TenantID, dag.ParentDAGID, dag.Name, dag.Description, dag.Category,
		dag.Nodes, dag.Edges, dag.SemanticBindings, dag.ScheduleType, dag.CronExpression,
		dag.CalendarIDs, dag.Timezone, dag.MaxParallelJobs, dag.FailFast,
		dag.TimeoutSeconds, dag.RiskScore, dag.SLOCritical, dag.ChangeSetID,
		dag.IsActive, dag.CreatedBy,
	).Scan(&dag.CreatedAt, &dag.UpdatedAt)
}

// GetDAG retrieves a DAG by ID
func (r *Repository) GetDAG(ctx context.Context, id uuid.UUID) (*DAG, error) {
	var dag DAG
	query := `SELECT * FROM scheduled_dags WHERE id = $1`
	err := r.db.GetContext(ctx, &dag, query, id)
	if err != nil {
		return nil, err
	}
	return &dag, nil
}

// UpdateDAG updates an existing DAG
func (r *Repository) UpdateDAG(ctx context.Context, dag *DAG) error {
	query := `
		UPDATE scheduled_dags SET
			name = $2, description = $3, category = $4, nodes = $5, edges = $6,
			semantic_bindings = $7, schedule_type = $8, cron_expression = $9,
			calendar_ids = $10, timezone = $11, max_parallel_jobs = $12,
			fail_fast = $13, timeout_seconds = $14, risk_score = $15,
			slo_critical = $16, is_active = $17, next_run_at = $18
		WHERE id = $1
		RETURNING updated_at
	`

	return r.db.QueryRowxContext(ctx, query,
		dag.ID, dag.Name, dag.Description, dag.Category, dag.Nodes, dag.Edges,
		dag.SemanticBindings, dag.ScheduleType, dag.CronExpression, dag.CalendarIDs, dag.Timezone,
		dag.MaxParallelJobs, dag.FailFast, dag.TimeoutSeconds,
		dag.RiskScore, dag.SLOCritical, dag.IsActive, dag.NextRunAt,
	).Scan(&dag.UpdatedAt)
}

// DeleteDAG soft-deletes a DAG
func (r *Repository) DeleteDAG(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE scheduled_dags SET is_active = false WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListDAGs lists DAGs for a tenant
func (r *Repository) ListDAGs(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]DAG, error) {
	var dags []DAG
	query := `SELECT * FROM scheduled_dags WHERE tenant_id = $1`
	if activeOnly {
		query += ` AND is_active = true`
	}
	query += ` ORDER BY name`

	err := r.db.SelectContext(ctx, &dags, query, tenantID)
	return dags, err
}

// ============================================================================
// Job Run CRUD
// ============================================================================

// CreateJobRun creates a new job run record
func (r *Repository) CreateJobRun(ctx context.Context, run *JobRun) error {
	query := `
		INSERT INTO job_runs (
			id, job_id, dag_run_id, tenant_id, temporal_workflow_id, temporal_run_id,
			task_queue, status, attempt_number, trigger_type, triggered_by,
			scheduled_at, input_parameters, semantic_bindings
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		RETURNING created_at
	`

	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}

	return r.db.QueryRowxContext(ctx, query,
		run.ID, run.JobID, run.DAGRunID, run.TenantID,
		run.TemporalWorkflowID, run.TemporalRunID, run.TaskQueue,
		run.Status, run.AttemptNumber, run.TriggerType, run.TriggeredBy,
		run.ScheduledAt, run.InputParameters, run.SemanticBindings,
	).Scan(&run.CreatedAt)
}

// GetJobRun retrieves a job run by ID
func (r *Repository) GetJobRun(ctx context.Context, id uuid.UUID) (*JobRun, error) {
	var run JobRun
	query := `SELECT * FROM job_runs WHERE id = $1`
	err := r.db.GetContext(ctx, &run, query, id)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// UpdateJobRunStatus updates the status of a job run
func (r *Repository) UpdateJobRunStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	query := `
		UPDATE job_runs SET
			status = $2,
			error_message = $3,
			completed_at = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') THEN NOW() ELSE completed_at END,
			duration_ms = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') AND started_at IS NOT NULL 
				THEN EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000 
				ELSE duration_ms END
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, errorMsg)
	return err
}

// UpdateJobRunStarted marks a job run as started
func (r *Repository) UpdateJobRunStarted(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE job_runs SET status = 'running', started_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// UpdateJobRunResult updates the result of a completed job run
func (r *Repository) UpdateJobRunResult(ctx context.Context, id uuid.UUID, result json.RawMessage) error {
	query := `UPDATE job_runs SET result = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, result)
	return err
}

// ListJobRuns lists job runs with filters
func (r *Repository) ListJobRuns(ctx context.Context, filters JobRunListFilters) ([]JobRun, error) {
	var runs []JobRun

	query := `SELECT * FROM job_runs WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filters.JobID != "" {
		query += fmt.Sprintf(" AND job_id = $%d", argIndex)
		args = append(args, filters.JobID)
		argIndex++
	}

	if filters.DAGRunID != "" {
		query += fmt.Sprintf(" AND dag_run_id = $%d", argIndex)
		args = append(args, filters.DAGRunID)
		argIndex++
	}

	if filters.TenantID != "" {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, filters.TenantID)
		argIndex++
	}

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filters.Status)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	err := r.db.SelectContext(ctx, &runs, query, args...)
	return runs, err
}

// ============================================================================
// DAG Run CRUD
// ============================================================================

// CreateDAGRun creates a new DAG run record
func (r *Repository) CreateDAGRun(ctx context.Context, run *DAGRun) error {
	query := `
		INSERT INTO dag_runs (
			id, dag_id, tenant_id, temporal_workflow_id, temporal_run_id,
			status, trigger_type, triggered_by, scheduled_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
		RETURNING created_at
	`

	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}

	return r.db.QueryRowxContext(ctx, query,
		run.ID, run.DAGID, run.TenantID, run.TemporalWorkflowID, run.TemporalRunID,
		run.Status, run.TriggerType, run.TriggeredBy, run.ScheduledAt, run.Metadata,
	).Scan(&run.CreatedAt)
}

// GetDAGRun retrieves a DAG run by ID
func (r *Repository) GetDAGRun(ctx context.Context, id uuid.UUID) (*DAGRun, error) {
	var run DAGRun
	query := `SELECT * FROM dag_runs WHERE id = $1`
	err := r.db.GetContext(ctx, &run, query, id)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// UpdateDAGRunStatus updates the status of a DAG run
func (r *Repository) UpdateDAGRunStatus(ctx context.Context, id uuid.UUID, status string, completedJobs, failedJobs, skippedJobs int, errorMsg *string) error {
	query := `
		UPDATE dag_runs SET
			status = $2,
			completed_jobs = $3,
			failed_jobs = $4,
			skipped_jobs = $5,
			error_message = $6,
			completed_at = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') THEN NOW() ELSE completed_at END,
			duration_ms = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') AND started_at IS NOT NULL 
				THEN EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000 
				ELSE duration_ms END
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, completedJobs, failedJobs, skippedJobs, errorMsg)
	return err
}

// ListDAGRuns lists DAG runs for a DAG
func (r *Repository) ListDAGRuns(ctx context.Context, dagID uuid.UUID, limit int) ([]DAGRun, error) {
	var runs []DAGRun
	query := `SELECT * FROM dag_runs WHERE dag_id = $1 ORDER BY created_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	err := r.db.SelectContext(ctx, &runs, query, dagID)
	return runs, err
}

// ============================================================================
// AI Suggestions
// ============================================================================

// CreateAISuggestion creates a new AI suggestion
func (r *Repository) CreateAISuggestion(ctx context.Context, suggestion *AISuggestion) error {
	query := `
		INSERT INTO scheduler_ai_suggestions (
			id, tenant_id, suggestion_type, target_type, target_id,
			title, description, impact_summary, risk_level, affected_tenants,
			proposed_changes, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		RETURNING created_at, updated_at
	`

	if suggestion.ID == uuid.Nil {
		suggestion.ID = uuid.New()
	}

	return r.db.QueryRowxContext(ctx, query,
		suggestion.ID, suggestion.TenantID, suggestion.SuggestionType,
		suggestion.TargetType, suggestion.TargetID, suggestion.Title,
		suggestion.Description, suggestion.ImpactSummary, suggestion.RiskLevel,
		suggestion.AffectedTenants, suggestion.ProposedChanges, suggestion.Status,
	).Scan(&suggestion.CreatedAt, &suggestion.UpdatedAt)
}

// GetPendingAISuggestions retrieves pending AI suggestions for a tenant
func (r *Repository) GetPendingAISuggestions(ctx context.Context, tenantID uuid.UUID) ([]AISuggestion, error) {
	var suggestions []AISuggestion
	query := `
		SELECT * FROM scheduler_ai_suggestions 
		WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &suggestions, query, tenantID)
	return suggestions, err
}

// UpdateAISuggestionStatus updates the status of an AI suggestion
func (r *Repository) UpdateAISuggestionStatus(ctx context.Context, id uuid.UUID, status string, dismissedReason *string, changesetID *uuid.UUID) error {
	query := `
		UPDATE scheduler_ai_suggestions SET
			status = $2,
			dismissed_reason = $3,
			changeset_id = $4
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, dismissedReason, changesetID)
	return err
}

// ============================================================================
// Governance - ChangeSets
// ============================================================================

// SaveChangeSet creates or updates a scheduler changeset
func (r *Repository) SaveChangeSet(ctx context.Context, cs *SchedulerChangeSet) error {
	query := `
		INSERT INTO scheduler_changesets (
			id, tenant_id, scope, type, title, description, author, status,
			target_type, target_id, diff, impact_analysis, ai_review, risk_score
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			impact_analysis = EXCLUDED.impact_analysis,
			ai_review = EXCLUDED.ai_review,
			risk_score = EXCLUDED.risk_score,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`

	return r.db.QueryRowxContext(ctx, query,
		cs.ID, cs.TenantID, cs.Scope, cs.Type, cs.Title, cs.Description, cs.Author, cs.Status,
		cs.TargetType, cs.TargetID, cs.Diff, cs.ImpactAnalysis, cs.AIReview, cs.RiskScore,
	).Scan(&cs.CreatedAt, &cs.UpdatedAt)
}

// GetChangeSet retrieves a changeset by ID
func (r *Repository) GetChangeSet(ctx context.Context, id uuid.UUID) (*SchedulerChangeSet, error) {
	var cs SchedulerChangeSet
	query := `SELECT * FROM scheduler_changesets WHERE id = $1`
	err := r.db.GetContext(ctx, &cs, query, id)
	return &cs, err
}

// ListChangeSets lists hangesets for a tenant
func (r *Repository) ListChangeSets(ctx context.Context, tenantID uuid.UUID, status *ChangeSetStatus) ([]SchedulerChangeSet, error) {
	var css []SchedulerChangeSet
	query := `SELECT * FROM scheduler_changesets WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(*status))
		argIndex++
	}

	if tenantID != uuid.Nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, tenantID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"
	err := r.db.SelectContext(ctx, &css, query, args...)
	return css, err
}

// SaveApproval saves an approval record
func (r *Repository) SaveApproval(ctx context.Context, app *ChangeSetApproval) error {
	query := `
		INSERT INTO scheduler_changeset_approvals (
			id, changeset_id, approver_id, approver_role, decision, comment
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		RETURNING created_at
	`
	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}
	return r.db.QueryRowxContext(ctx, query,
		app.ID, app.ChangeSetID, app.ApproverID, app.ApproverRole, app.Decision, app.Comment,
	).Scan(&app.CreatedAt)
}

// GetApprovals retrieves approvals for a changeset
func (r *Repository) GetApprovals(ctx context.Context, changesetID uuid.UUID) ([]ChangeSetApproval, error) {
	var apps []ChangeSetApproval
	query := `SELECT * FROM scheduler_changeset_approvals WHERE changeset_id = $1 ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &apps, query, changesetID)
	return apps, err
}

// GetDownstreamJobs retrieves jobs that depend on the given job
func (r *Repository) GetDownstreamJobs(ctx context.Context, jobID uuid.UUID) ([]Job, error) {
	var jobs []Job
	query := `
		SELECT j.* 
		FROM scheduled_jobs j
		JOIN job_dependencies d ON j.id = d.job_id
		WHERE d.depends_on_job_id = $1 AND j.is_active = true
	`
	err := r.db.SelectContext(ctx, &jobs, query, jobID)
	return jobs, err
}

// GetDAGsContainingJob retrieves DAGs that include the given job in their nodes
func (r *Repository) GetDAGsContainingJob(ctx context.Context, jobID uuid.UUID) ([]DAG, error) {
	var dags []DAG
	// Note: We use JSONB containment operator @> to check if any node has the job_id
	query := `
		SELECT * 
		FROM scheduled_dags 
		WHERE nodes @> $1::jsonb AND is_active = true
	`
	// Correctly format the JSON query parameter
	// nodes is [{id, job_id, ...}]
	// We check for any element in the array matching {"job_id": "..."}
	jobIDStr := fmt.Sprintf(`[{"job_id": "%s"}]`, jobID.String())
	err := r.db.SelectContext(ctx, &dags, query, jobIDStr)
	return dags, err
}
