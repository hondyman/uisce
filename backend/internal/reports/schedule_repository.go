package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DefaultReportExecutor creates execution records in report_executions and generates a snapshot path.
//
// PLACEHOLDER NOTE: This default implementation uses synthetic metrics (rows_processed=10,
// execution_time_ms=250) and a deterministic s3:// output path as an integration bridge.
// In Phase 3, this executor will be replaced or wrapped by the full ReportOrchestrator
// and ReportBurstOrchestrator runtime pipelines.
type DefaultReportExecutor struct {
	db *sql.DB
}

func NewDefaultReportExecutor(db *sql.DB) *DefaultReportExecutor {
	return &DefaultReportExecutor{db: db}
}

func (e *DefaultReportExecutor) ExecuteReport(ctx context.Context, tmpl *ReportTemplate, params map[string]interface{}) (*ScheduleExecutionResult, error) {
	execID := uuid.New()
	outputURL := fmt.Sprintf("s3://reports/%s/%s/%s.pdf", tmpl.TenantID, tmpl.ID, execID)
	paramsJSON, _ := json.Marshal(params)

	var reqBy *string
	if tmpl.CreatedByID != nil && *tmpl.CreatedByID != "" {
		reqBy = tmpl.CreatedByID
	} else if tmpl.CreatedBy != "" {
		reqBy = &tmpl.CreatedBy
	}

	query := `
		INSERT INTO public.report_executions (
			id, tenant_id, template_id, report_key, status, parameters,
			output_format, output_url, rows_processed, execution_time_ms,
			requested_by, created_at, completed_at
		) VALUES (
			$1, $2, $3, $4, 'completed', $5,
			'pdf', $6, 10, 250,
			$7, NOW(), NOW()
		) RETURNING id, status, output_url, COALESCE(requested_by, '')
	`

	var res ScheduleExecutionResult
	res.TenantID = tmpl.TenantID
	res.TemplateID = tmpl.ID

	err := e.db.QueryRowContext(ctx, query,
		execID, tmpl.TenantID, tmpl.ID, tmpl.TemplateName, paramsJSON,
		outputURL, reqBy,
	).Scan(&res.ExecutionID, &res.Status, &res.OutputURL, &res.RequestedBy)

	if err != nil {
		return nil, fmt.Errorf("execute report record failed: %w", err)
	}

	return &res, nil
}

// CreateSchedule creates a new schedule linked to a template after validating the visibility predicate.
func (r *Repository) CreateSchedule(ctx context.Context, tenantID uuid.UUID, callerUserID string, input CreateScheduleInput) (*ReportSchedule, error) {
	goldCopyTenantID, err := r.ResolveGoldCopyTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve gold copy tenant failed: %w", err)
	}

	// 1. Enforce Template Visibility Predicate
	// Target template must be active and either belong to caller's tenant or gold copy.
	// Clean IN ($2, $3) predicate without zero-UUID wildcards.
	// If it's a personal report, caller MUST be the owner (created_by_id).
	var tmpl ReportTemplate
	tmplQuery := `
		SELECT id, tenant_id, template_name, is_active, is_personal, created_by_id
		FROM public.report_templates
		WHERE id = $1
		  AND is_active = true
		  AND tenant_id IN ($2, $3)
		  AND (is_personal = false OR (created_by_id IS NOT NULL AND created_by_id = $4))
	`
	err = r.db.QueryRowContext(ctx, tmplQuery, input.TemplateID, tenantID, goldCopyTenantID, callerUserID).Scan(
		&tmpl.ID, &tmpl.TenantID, &tmpl.TemplateName, &tmpl.IsActive, &tmpl.IsPersonal, &tmpl.CreatedByID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("verify template visibility failed: %w", err)
	}

	// 2. Insert Schedule Record
	scheduleID := uuid.New()
	notificationJSON, _ := json.Marshal(map[string]bool{
		"in_app": input.NotifyInApp,
		"email":  input.NotifyEmail,
	})

	insertQuery := `
		INSERT INTO public.report_schedules (
			id, tenant_id, report_definition_id, owner_id, schedule_name,
			cron_expression, region, calendar_id, start_of_day_time,
			unscheduled_behavior, business_day_offset, burst_dimension,
			export_format, notification_channels, is_active, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, COALESCE(NULLIF($7, ''), 'us-west'), $8, COALESCE(NULLIF($9, '')::time, '08:00:00'::time),
			COALESCE(NULLIF($10, ''), 'SKIP'), $11, COALESCE(NULLIF($12, ''), 'client_id'),
			COALESCE(NULLIF($13, ''), 'PDF'), $14, true, NOW()
		)
		RETURNING id, tenant_id, report_definition_id, owner_id, schedule_name,
		          cron_expression, region, calendar_id, start_of_day_time::text,
		          unscheduled_behavior, business_day_offset, burst_dimension,
		          export_format, notification_channels, is_active, deleted_at,
		          last_run_at, next_run_at, created_at
	`

	var s ReportSchedule
	var notifBytes []byte
	err = r.db.QueryRowContext(ctx, insertQuery,
		scheduleID, tenantID, input.TemplateID, callerUserID, input.ScheduleName,
		input.CronExpression, input.Region, input.CalendarID, input.StartOfDayTime,
		input.UnscheduledBehavior, input.BusinessDayOffset, input.BurstDimension,
		input.ExportFormat, notificationJSON,
	).Scan(
		&s.ID, &s.TenantID, &s.ReportDefinitionID, &s.OwnerID, &s.ScheduleName,
		&s.CronExpression, &s.Region, &s.CalendarID, &s.StartOfDayTime,
		&s.UnscheduledBehavior, &s.BusinessDayOffset, &s.BurstDimension,
		&s.ExportFormat, &notifBytes, &s.IsActive, &s.DeletedAt,
		&s.LastRunAt, &s.NextRunAt, &s.CreatedAt,
	)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" { // unique_violation
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create schedule failed: %w", err)
	}

	_ = json.Unmarshal(notifBytes, &s.NotificationChannels)
	return &s, nil
}

// ListSchedulesForTemplate retrieves active schedules for a template in a tenant.
func (r *Repository) ListSchedulesForTemplate(ctx context.Context, tenantID uuid.UUID, callerUserID string, templateID uuid.UUID) ([]ReportSchedule, error) {
	goldCopyTenantID, err := r.ResolveGoldCopyTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve gold copy tenant failed: %w", err)
	}

	// Verify template visibility first using clean IN ($2, $3) predicate
	var dummy int
	tmplQuery := `
		SELECT 1 FROM public.report_templates
		WHERE id = $1
		  AND is_active = true
		  AND tenant_id IN ($2, $3)
		  AND (is_personal = false OR (created_by_id IS NOT NULL AND created_by_id = $4))
	`
	if err := r.db.QueryRowContext(ctx, tmplQuery, templateID, tenantID, goldCopyTenantID, callerUserID).Scan(&dummy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("verify template visibility failed: %w", err)
	}

	query := `
		SELECT id, tenant_id, report_definition_id, owner_id, schedule_name,
		       cron_expression, region, calendar_id, start_of_day_time::text,
		       unscheduled_behavior, business_day_offset, burst_dimension,
		       export_format, notification_channels, is_active, deleted_at,
		       last_run_at, next_run_at, created_at
		FROM public.report_schedules
		WHERE tenant_id = $1
		  AND report_definition_id = $2
		  AND is_active = true
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, templateID)
	if err != nil {
		return nil, fmt.Errorf("list schedules failed: %w", err)
	}
	defer rows.Close()

	schedules := make([]ReportSchedule, 0)
	for rows.Next() {
		var s ReportSchedule
		var notifBytes []byte
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.ReportDefinitionID, &s.OwnerID, &s.ScheduleName,
			&s.CronExpression, &s.Region, &s.CalendarID, &s.StartOfDayTime,
			&s.UnscheduledBehavior, &s.BusinessDayOffset, &s.BurstDimension,
			&s.ExportFormat, &notifBytes, &s.IsActive, &s.DeletedAt,
			&s.LastRunAt, &s.NextRunAt, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schedule row failed: %w", err)
		}
		_ = json.Unmarshal(notifBytes, &s.NotificationChannels)
		schedules = append(schedules, s)
	}

	return schedules, nil
}

// GetSchedule retrieves a single schedule by ID ensuring tenant scoping and excluding soft-deleted records.
func (r *Repository) GetSchedule(ctx context.Context, tenantID uuid.UUID, scheduleID uuid.UUID) (*ReportSchedule, error) {
	query := `
		SELECT id, tenant_id, report_definition_id, owner_id, schedule_name,
		       cron_expression, region, calendar_id, start_of_day_time::text,
		       unscheduled_behavior, business_day_offset, burst_dimension,
		       export_format, notification_channels, is_active, deleted_at,
		       last_run_at, next_run_at, created_at
		FROM public.report_schedules
		WHERE id = $1
		  AND tenant_id = $2
		  AND is_active = true
		  AND deleted_at IS NULL
	`

	var s ReportSchedule
	var notifBytes []byte
	err := r.db.QueryRowContext(ctx, query, scheduleID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.ReportDefinitionID, &s.OwnerID, &s.ScheduleName,
		&s.CronExpression, &s.Region, &s.CalendarID, &s.StartOfDayTime,
		&s.UnscheduledBehavior, &s.BusinessDayOffset, &s.BurstDimension,
		&s.ExportFormat, &notifBytes, &s.IsActive, &s.DeletedAt,
		&s.LastRunAt, &s.NextRunAt, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get schedule failed: %w", err)
	}

	_ = json.Unmarshal(notifBytes, &s.NotificationChannels)
	return &s, nil
}

// DeleteSchedule performs a soft delete on a schedule (is_active = false, deleted_at = NOW()).
// Restricts operation to schedule owner or tenant admin.
func (r *Repository) DeleteSchedule(ctx context.Context, tenantID uuid.UUID, callerUserID string, isAdmin bool, scheduleID uuid.UUID) error {
	var ownerID string
	checkQuery := `
		SELECT owner_id
		FROM public.report_schedules
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	err := r.db.QueryRowContext(ctx, checkQuery, scheduleID, tenantID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("check schedule owner failed: %w", err)
	}

	if !isAdmin && ownerID != callerUserID {
		return ErrForbidden
	}

	updateQuery := `
		UPDATE public.report_schedules
		SET is_active = false, deleted_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`
	res, err := r.db.ExecContext(ctx, updateQuery, scheduleID, tenantID)
	if err != nil {
		return fmt.Errorf("soft delete schedule failed: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// TriggerScheduleRun triggers execution of a scheduled report.
// RESTRICTION: Only schedule owner or tenant admin may trigger.
// EXECUTION INVARIANT: The run always executes under the template owner's identity
// (template.tenant_id, template.created_by_id), NOT the HTTP caller's credentials.
// SCOPING INVARIANT: The template is read with tenant-scoping against the caller tenant OR
// gold copy tenant, guaranteeing execution context cannot be hijacked across foreign tenants.
func (r *Repository) TriggerScheduleRun(
	ctx context.Context,
	tenantID uuid.UUID,
	callerUserID string,
	isAdmin bool,
	scheduleID uuid.UUID,
	executor ReportExecutor,
) (*ScheduleExecutionResult, error) {
	// 1. Fetch schedule and check caller permission
	var s struct {
		ID                 uuid.UUID
		TenantID           uuid.UUID
		ReportDefinitionID uuid.UUID
		OwnerID            string
	}
	querySched := `
		SELECT id, tenant_id, report_definition_id, owner_id
		FROM public.report_schedules
		WHERE id = $1 AND tenant_id = $2 AND is_active = true AND deleted_at IS NULL
	`
	err := r.db.QueryRowContext(ctx, querySched, scheduleID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.ReportDefinitionID, &s.OwnerID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get schedule for trigger failed: %w", err)
	}

	if !isAdmin && s.OwnerID != callerUserID {
		return nil, ErrForbidden
	}

	// 2. Fetch the linked template to extract its execution context & owner.
	// Explicitly scope template fetch to either the schedule's tenant or gold copy tenant.
	goldCopyTenantID, err := r.ResolveGoldCopyTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve gold copy tenant failed: %w", err)
	}

	var tmpl ReportTemplate
	var viewsJSON, layoutJSON, paramJSON []byte
	var createdByID, createdBy sql.NullString
	tmplQuery := `
		SELECT id, tenant_id, template_name, description, category,
		       semantic_view_ids, layout_config, parameter_schema,
		       is_active, is_public, is_personal, created_by_id, created_by,
		       created_at, updated_at, version
		FROM public.report_templates
		WHERE id = $1
		  AND is_active = true
		  AND tenant_id IN ($2, $3)
	`
	err = r.db.QueryRowContext(ctx, tmplQuery, s.ReportDefinitionID, s.TenantID, goldCopyTenantID).Scan(
		&tmpl.ID, &tmpl.TenantID, &tmpl.TemplateName, &tmpl.Description, &tmpl.Category,
		&viewsJSON, &layoutJSON, &paramJSON,
		&tmpl.IsActive, &tmpl.IsPublic, &tmpl.IsPersonal, &createdByID, &createdBy,
		&tmpl.CreatedAt, &tmpl.UpdatedAt, &tmpl.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: linked report template not found or inaccessible in tenant scope", ErrNotFound)
		}
		return nil, fmt.Errorf("fetch linked template failed: %w", err)
	}
	if createdByID.Valid {
		tmpl.CreatedByID = &createdByID.String
	}
	if createdBy.Valid {
		tmpl.CreatedBy = createdBy.String
	}

	// 3. Delegate to executor (DefaultReportExecutor or mock passed from test)
	if executor == nil {
		executor = NewDefaultReportExecutor(r.db)
	}
	res, err := executor.ExecuteReport(ctx, &tmpl, map[string]interface{}{
		"triggered_at": time.Now().Format(time.RFC3339),
		"schedule_id":  s.ID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("execute report failed: %w", err)
	}

	// 4. Update schedule last_run_at
	_, _ = r.db.ExecContext(ctx, `
		UPDATE public.report_schedules
		SET last_run_at = NOW()
		WHERE id = $1
	`, s.ID)

	// 5. Write cache metadata entry with 24h TTL
	cacheKey := fmt.Sprintf("sched_%s", s.ID)
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO public.report_cache_metadata (
			id, tenant_id, template_id, cache_key, output_url, expires_at, created_at, hit_count, last_accessed_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, NOW() + INTERVAL '24 hours', NOW(), 1, NOW()
		)
		ON CONFLICT (template_id, cache_key) DO UPDATE
		SET output_url = EXCLUDED.output_url,
		    expires_at = EXCLUDED.expires_at,
		    hit_count = public.report_cache_metadata.hit_count + 1,
		    last_accessed_at = NOW()
	`, tmpl.TenantID, tmpl.ID, cacheKey, res.OutputURL)
	if err != nil {
		log.Printf("[WARN] failed to update report_cache_metadata for template %s schedule %s: %v", tmpl.ID, s.ID, err)
	}

	return res, nil
}
