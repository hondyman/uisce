package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type InProcessSyncExecutor struct {
	db *sql.DB
}

func NewTestInProcessExecutor(db *sql.DB) *InProcessSyncExecutor {
	return &InProcessSyncExecutor{db: db}
}

func (e *InProcessSyncExecutor) ExecuteReport(ctx context.Context, tmpl *ReportTemplate, params map[string]interface{}) (*ScheduleExecutionResult, error) {
	execID := uuid.New()
	outputURL := fmt.Sprintf("s3://reports/%s/%s/%s.pdf", tmpl.TenantID, tmpl.ID, execID)
	paramsJSON, _ := json.Marshal(params)

	var reqBy *string
	if tmpl.CreatedByID != nil && *tmpl.CreatedByID != "" {
		reqBy = tmpl.CreatedByID
	} else if tmpl.CreatedBy != "" {
		reqBy = &tmpl.CreatedBy
	}

	lineageJSON, _ := json.Marshal(map[string]interface{}{
		"synthetic":    true,
		"engine":       "in_process_sync",
		"note":         "Test-only in-process executor; production uses TemporalReportExecutor",
	})

	query := `
		INSERT INTO public.report_executions (
			id, tenant_id, template_id, report_key, status, parameters,
			output_format, output_url, rows_processed, execution_time_ms,
			requested_by, lineage, created_at, completed_at
		) VALUES (
			$1, $2, $3, $4, 'synthetic', $5,
			'pdf', $6, 10, 250,
			$7, $8, NOW(), NOW()
		) RETURNING id, status, output_url, COALESCE(requested_by, '')
	`

	var res ScheduleExecutionResult
	res.TenantID = tmpl.TenantID
	res.TemplateID = tmpl.ID

	err := e.db.QueryRowContext(ctx, query,
		execID, tmpl.TenantID, tmpl.ID, tmpl.TemplateName, paramsJSON,
		outputURL, reqBy, lineageJSON,
	).Scan(&res.ExecutionID, &res.Status, &res.OutputURL, &res.RequestedBy)

	if err != nil {
		return nil, fmt.Errorf("execute report record failed: %w", err)
	}

	return &res, nil
}
