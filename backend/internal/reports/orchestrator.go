package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ReportOrchestrator struct {
	db *sql.DB
}

func NewReportOrchestrator(db *sql.DB) *ReportOrchestrator {
	return &ReportOrchestrator{db: db}
}

// GenerateReport executes a report template (synchronous version)
func (ro *ReportOrchestrator) GenerateReport(ctx context.Context, templateID, tenantID uuid.UUID, params map[string]interface{}) (*ReportExecution, error) {
	startTime := time.Now()

	// Create execution record
	execution := &ReportExecution{
		ID:         uuid.New(),
		TenantID:   tenantID,
		TemplateID: templateID,
		Parameters: params,
		Status:     "running",
		CreatedAt:  time.Now(),
	}

	if householdID, ok := params["household_id"].(string); ok {
		hid, err := uuid.Parse(householdID)
		if err == nil {
			execution.HouseholdID = &hid
		}
	}

	// Insert execution record
	if err := ro.createExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	// Fetch template
	template, err := ro.GetTemplate(ctx, templateID, tenantID)
	if err != nil {
		ro.updateExecutionStatus(ctx, execution.ID, "failed", err.Error(), nil)
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	// Generate report (stub for now - will be implemented with PDF library)
	outputURL, sizeBytes, rowsProcessed, err := ro.generatePDF(ctx, template, params)
	if err != nil {
		ro.updateExecutionStatus(ctx, execution.ID, "failed", err.Error(), nil)
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Update execution with results
	executionTimeMS := int(time.Since(startTime).Milliseconds())
	completedAt := time.Now()

	execution.Status = "completed"
	execution.OutputURL = outputURL
	execution.OutputSizeBytes = sizeBytes
	execution.RowsProcessed = rowsProcessed
	execution.ExecutionTimeMS = executionTimeMS
	execution.CompletedAt = &completedAt

	if err := ro.updateExecutionStatus(ctx, execution.ID, "completed", "", &completedAt); err != nil {
		return nil, fmt.Errorf("failed to update execution status: %w", err)
	}

	// Update metrics
	if err := ro.updateExecutionMetrics(ctx, execution.ID, outputURL, sizeBytes, rowsProcessed, executionTimeMS); err != nil {
		return nil, fmt.Errorf("failed to update execution metrics: %w", err)
	}

	return execution, nil
}

// GetTemplate retrieves a report template
func (ro *ReportOrchestrator) GetTemplate(ctx context.Context, templateID, tenantID uuid.UUID) (*ReportTemplate, error) {
	return ro.getTemplate(ctx, templateID, tenantID)
}

// ListTemplates lists available report templates
func (ro *ReportOrchestrator) ListTemplates(ctx context.Context, tenantID uuid.UUID, category string) ([]ReportTemplate, error) {
	return ro.listTemplates(ctx, tenantID, category)
}

// GetExecution retrieves a report execution
func (ro *ReportOrchestrator) GetExecution(ctx context.Context, executionID, tenantID uuid.UUID) (*ReportExecution, error) {
	return ro.getExecution(ctx, executionID, tenantID)
}

// createExecution inserts a new execution record
func (ro *ReportOrchestrator) createExecution(ctx context.Context, execution *ReportExecution) error {
	return ro.createExecutionRecord(ctx, execution)
}

// updateExecutionStatus updates the status of an execution
func (ro *ReportOrchestrator) updateExecutionStatus(ctx context.Context, executionID uuid.UUID, status, errorMessage string, completedAt *time.Time) error {
	return ro.updateStatus(ctx, executionID, status, errorMessage, completedAt)
}

// generatePDF is a stub for PDF generation (to be implemented)
func (ro *ReportOrchestrator) generatePDF(ctx context.Context, template *ReportTemplate, params map[string]interface{}) (outputURL string, sizeBytes int, rowsProcessed int, err error) {
	// TODO: Implement actual PDF generation with gofpdf or similar
	// For now, return placeholder values

	// This is where we would:
	// 1. Query semantic views based on template.SemanticViewIDs
	// 2. Transform data according to template.LayoutConfig
	// 3. Generate PDF using library like gofpdf
	// 4. Upload to S3/GCS
	// 5. Return signed URL

	outputURL = fmt.Sprintf("/tmp/reports/%s.pdf", uuid.New().String())
	sizeBytes = 1024 * 100 // 100 KB placeholder
	rowsProcessed = 42     // Placeholder

	return outputURL, sizeBytes, rowsProcessed, nil
}

// ============================================================================
// HASURA-FIRST HELPERS
// ============================================================================

func (ro *ReportOrchestrator) updateExecutionMetrics(ctx context.Context, executionID uuid.UUID, outputURL string, sizeBytes, rowsProcessed, executionTimeMS int) error {
	query := `
		UPDATE report_executions
		SET output_url = $1,
		    output_size_bytes = $2,
		    rows_processed = $3,
		    execution_time_ms = $4
		WHERE id = $5
	`
	_, err := ro.db.ExecContext(ctx, query, outputURL, sizeBytes, rowsProcessed, executionTimeMS, executionID)
	return err
}

func (ro *ReportOrchestrator) getTemplate(ctx context.Context, templateID, tenantID uuid.UUID) (*ReportTemplate, error) {
	query := `
		SELECT id, tenant_id, template_name, description, category,
		       semantic_view_ids, layout_config, parameter_schema,
		       is_active, is_public, created_at, updated_at, created_by, version
		FROM report_templates
		WHERE id = $1 AND (tenant_id = $2 OR is_public = true)
	`

	var template ReportTemplate
	var layoutConfigJSON, paramSchemaJSON []byte
	var semanticViewIDsJSON []byte

	err := ro.db.QueryRowContext(ctx, query, templateID, tenantID).Scan(
		&template.ID,
		&template.TenantID,
		&template.TemplateName,
		&template.Description,
		&template.Category,
		&semanticViewIDsJSON,
		&layoutConfigJSON,
		&paramSchemaJSON,
		&template.IsActive,
		&template.IsPublic,
		&template.CreatedAt,
		&template.UpdatedAt,
		&template.CreatedBy,
		&template.Version,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found: %s", templateID)
		}
		return nil, fmt.Errorf("failed to query template: %w", err)
	}

	// Parse JSON fields
	if err := json.Unmarshal(layoutConfigJSON, &template.LayoutConfig); err != nil {
		return nil, fmt.Errorf("failed to parse layout config: %w", err)
	}
	if err := json.Unmarshal(paramSchemaJSON, &template.ParameterSchema); err != nil {
		return nil, fmt.Errorf("failed to parse parameter schema: %w", err)
	}

	// Parse semantic view IDs array
	var viewIDStrings []string
	if err := json.Unmarshal(semanticViewIDsJSON, &viewIDStrings); err != nil {
		return nil, fmt.Errorf("failed to parse semantic view IDs: %w", err)
	}
	template.SemanticViewIDs = make([]uuid.UUID, len(viewIDStrings))
	for i, idStr := range viewIDStrings {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid semantic view ID: %s", idStr)
		}
		template.SemanticViewIDs[i] = id
	}

	return &template, nil
}

func (ro *ReportOrchestrator) listTemplates(ctx context.Context, tenantID uuid.UUID, category string) ([]ReportTemplate, error) {
	query := `
		SELECT id, tenant_id, template_name, description, category,
		       is_active, is_public, created_at, version
		FROM report_templates
		WHERE (tenant_id = $1 OR is_public = true)
		  AND is_active = true
	`
	args := []interface{}{tenantID}

	if category != "" {
		query += " AND category = $2"
		args = append(args, category)
	}

	query += " ORDER BY template_name"

	rows, err := ro.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []ReportTemplate
	for rows.Next() {
		var t ReportTemplate
		if err := rows.Scan(
			&t.ID,
			&t.TenantID,
			&t.TemplateName,
			&t.Description,
			&t.Category,
			&t.IsActive,
			&t.IsPublic,
			&t.CreatedAt,
			&t.Version,
		); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, t)
	}

	return templates, nil
}

func (ro *ReportOrchestrator) getExecution(ctx context.Context, executionID, tenantID uuid.UUID) (*ReportExecution, error) {
	query := `
		SELECT id, tenant_id, template_id, household_id, parameters,
		       status, error_message, output_url, output_size_bytes,
		       execution_time_ms, rows_processed, workflow_id, run_id,
		       created_at, completed_at, created_by
		FROM report_executions
		WHERE id = $1 AND tenant_id = $2
	`

	var execution ReportExecution
	var parametersJSON []byte
	var householdID *uuid.UUID

	err := ro.db.QueryRowContext(ctx, query, executionID, tenantID).Scan(
		&execution.ID,
		&execution.TenantID,
		&execution.TemplateID,
		&householdID,
		&parametersJSON,
		&execution.Status,
		&execution.ErrorMessage,
		&execution.OutputURL,
		&execution.OutputSizeBytes,
		&execution.ExecutionTimeMS,
		&execution.RowsProcessed,
		&execution.WorkflowID,
		&execution.RunID,
		&execution.CreatedAt,
		&execution.CompletedAt,
		&execution.CreatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("execution not found: %s", executionID)
		}
		return nil, fmt.Errorf("failed to query execution: %w", err)
	}

	execution.HouseholdID = householdID

	if err := json.Unmarshal(parametersJSON, &execution.Parameters); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	return &execution, nil
}

func (ro *ReportOrchestrator) createExecutionRecord(ctx context.Context, execution *ReportExecution) error {
	query := `
		INSERT INTO report_executions (
			id, tenant_id, template_id, household_id, parameters,
			status, created_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	paramsJSON, err := json.Marshal(execution.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	_, err = ro.db.ExecContext(ctx, query,
		execution.ID,
		execution.TenantID,
		execution.TemplateID,
		execution.HouseholdID,
		paramsJSON,
		execution.Status,
		execution.CreatedAt,
		execution.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to insert execution: %w", err)
	}

	return nil
}

func (ro *ReportOrchestrator) updateStatus(ctx context.Context, executionID uuid.UUID, status, errorMessage string, completedAt *time.Time) error {
	query := `
		UPDATE report_executions
		SET status = $1,
		    error_message = $2,
		    completed_at = $3
		WHERE id = $4
	`

	_, err := ro.db.ExecContext(ctx, query, status, errorMessage, completedAt, executionID)
	if err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	return nil
}
