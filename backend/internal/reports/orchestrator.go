package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ReportOrchestrator coordinates report generation
type ReportOrchestrator struct {
	db     *sql.DB
	hasura HasuraClient
}

// NewReportOrchestrator creates a new report orchestrator
func NewReportOrchestrator(db *sql.DB) *ReportOrchestrator {
	return &ReportOrchestrator{
		db: db,
	}
}

// NewReportOrchestratorWithHasura creates a new report orchestrator with Hasura support
func NewReportOrchestratorWithHasura(db *sql.DB, hasura HasuraClient) *ReportOrchestrator {
	return &ReportOrchestrator{
		db:     db,
		hasura: hasura,
	}
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

// updateExecutionMetrics updates execution metrics
// Hasura-first with SQL fallback
func (ro *ReportOrchestrator) updateExecutionMetrics(ctx context.Context, executionID uuid.UUID, outputURL string, sizeBytes, rowsProcessed, executionTimeMS int) error {
	if ro.hasura != nil {
		mutation := `
			mutation UpdateMetrics($id: uuid!, $outputURL: String!, $sizeBytes: Int!, $rows: Int!, $timeMS: Int!) {
				update_report_executions_by_pk(pk_columns: {id: $id}, _set: {
					output_url: $outputURL
					output_size_bytes: $sizeBytes
					rows_processed: $rows
					execution_time_ms: $timeMS
				}) {
					id
				}
			}
		`

		variables := map[string]interface{}{
			"id":        executionID,
			"outputURL": outputURL,
			"sizeBytes": sizeBytes,
			"rows":      rowsProcessed,
			"timeMS":    executionTimeMS,
		}

		_, err := ro.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
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

// getTemplate retrieves a report template
// Hasura-first with SQL fallback
func (ro *ReportOrchestrator) getTemplate(ctx context.Context, templateID, tenantID uuid.UUID) (*ReportTemplate, error) {
	if ro.hasura != nil {
		query := `
			query GetTemplate($id: uuid!, $tenantID: uuid!) {
				report_templates(where: {
					id: {_eq: $id}
					_or: [
						{tenant_id: {_eq: $tenantID}}
						{is_public: {_eq: true}}
					]
				}) {
					id
					tenant_id
					template_name
					description
					category
					semantic_view_ids
					layout_config
					parameter_schema
					is_active
					is_public
					created_at
					updated_at
					created_by
					version
				}
			}
		`

		variables := map[string]interface{}{
			"id":       templateID,
			"tenantID": tenantID,
		}

		result, err := ro.hasura.Query(query, variables)
		if err == nil {
			if templates, ok := result["report_templates"].([]interface{}); ok && len(templates) > 0 {
				if tplData, ok := templates[0].(map[string]interface{}); ok {
					template, err := parseTemplateFromHasura(tplData)
					if err == nil {
						return template, nil
					}
				}
			}
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
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

// listTemplates lists available report templates
// Hasura-first with SQL fallback
func (ro *ReportOrchestrator) listTemplates(ctx context.Context, tenantID uuid.UUID, category string) ([]ReportTemplate, error) {
	if ro.hasura != nil {
		whereClause := map[string]interface{}{
			"_or": []interface{}{
				map[string]interface{}{"tenant_id": map[string]interface{}{"_eq": tenantID}},
				map[string]interface{}{"is_public": map[string]interface{}{"_eq": true}},
			},
			"is_active": map[string]interface{}{"_eq": true},
		}

		if category != "" {
			whereClause["category"] = map[string]interface{}{"_eq": category}
		}

		query := `
			query ListTemplates($where: report_templates_bool_exp!) {
				report_templates(where: $where, order_by: {template_name: asc}) {
					id
					tenant_id
					template_name
					description
					category
					is_active
					is_public
					created_at
					version
				}
			}
		`

		variables := map[string]interface{}{
			"where": whereClause,
		}

		result, err := ro.hasura.Query(query, variables)
		if err == nil {
			if templates, ok := result["report_templates"].([]interface{}); ok {
				var out []ReportTemplate
				for _, t := range templates {
					if tplData, ok := t.(map[string]interface{}); ok {
						template, err := parseTemplateSummaryFromHasura(tplData)
						if err == nil {
							out = append(out, template)
						}
					}
				}
				return out, nil
			}
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
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

// getExecution retrieves a report execution
// Hasura-first with SQL fallback
func (ro *ReportOrchestrator) getExecution(ctx context.Context, executionID, tenantID uuid.UUID) (*ReportExecution, error) {
	if ro.hasura != nil {
		query := `
			query GetExecution($id: uuid!, $tenantID: uuid!) {
				report_executions_by_pk(id: $id) {
					id
					tenant_id
					template_id
					household_id
					parameters
					status
					error_message
					output_url
					output_size_bytes
					execution_time_ms
					rows_processed
					workflow_id
					run_id
					created_at
					completed_at
					created_by
				}
			}
		`

		variables := map[string]interface{}{
			"id":       executionID,
			"tenantID": tenantID,
		}

		result, err := ro.hasura.Query(query, variables)
		if err == nil {
			if execData, ok := result["report_executions_by_pk"].(map[string]interface{}); ok {
				// Check tenant match
				if tid, ok := execData["tenant_id"].(string); ok {
					if tid == tenantID.String() {
						execution, err := parseExecutionFromHasura(execData)
						if err == nil {
							return execution, nil
						}
					}
				}
			}
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
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

// createExecutionRecord inserts a new execution record
// Hasura-first with SQL fallback
func (ro *ReportOrchestrator) createExecutionRecord(ctx context.Context, execution *ReportExecution) error {
	if ro.hasura != nil {
		mutation := `
			mutation InsertExecution(
				$id: uuid!
				$tenantID: uuid!
				$templateID: uuid!
				$householdID: uuid
				$parameters: jsonb!
				$status: String!
				$createdAt: timestamptz!
				$createdBy: String
			) {
				insert_report_executions_one(object: {
					id: $id
					tenant_id: $tenantID
					template_id: $templateID
					household_id: $householdID
					parameters: $parameters
					status: $status
					created_at: $createdAt
					created_by: $createdBy
				}) {
					id
				}
			}
		`

		variables := map[string]interface{}{
			"id":          execution.ID,
			"tenantID":    execution.TenantID,
			"templateID":  execution.TemplateID,
			"householdID": execution.HouseholdID,
			"parameters":  execution.Parameters,
			"status":      execution.Status,
			"createdAt":   execution.CreatedAt,
			"createdBy":   execution.CreatedBy,
		}

		_, err := ro.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
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

// updateStatus updates the status of an execution
// Hasura-first with SQL fallback
func (ro *ReportOrchestrator) updateStatus(ctx context.Context, executionID uuid.UUID, status, errorMessage string, completedAt *time.Time) error {
	if ro.hasura != nil {
		mutation := `
			mutation UpdateStatus($id: uuid!, $status: String!, $errorMessage: String, $completedAt: timestamptz) {
				update_report_executions_by_pk(pk_columns: {id: $id}, _set: {
					status: $status
					error_message: $errorMessage
					completed_at: $completedAt
				}) {
					id
				}
			}
		`

		variables := map[string]interface{}{
			"id":           executionID,
			"status":       status,
			"errorMessage": errorMessage,
			"completedAt":  completedAt,
		}

		_, err := ro.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
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

// ============================================================================
// HELPER FUNCTIONS FOR PARSING HASURA RESPONSES
// ============================================================================

func parseTemplateFromHasura(data map[string]interface{}) (*ReportTemplate, error) {
	template := &ReportTemplate{}

	if id, ok := data["id"].(string); ok {
		template.ID, _ = uuid.Parse(id)
	}
	if tenantID, ok := data["tenant_id"].(string); ok {
		template.TenantID, _ = uuid.Parse(tenantID)
	}
	if name, ok := data["template_name"].(string); ok {
		template.TemplateName = name
	}
	if desc, ok := data["description"].(string); ok {
		template.Description = desc
	}
	if cat, ok := data["category"].(string); ok {
		template.Category = cat
	}
	if isActive, ok := data["is_active"].(bool); ok {
		template.IsActive = isActive
	}
	if isPublic, ok := data["is_public"].(bool); ok {
		template.IsPublic = isPublic
	}
	if version, ok := data["version"].(float64); ok {
		template.Version = int(version)
	}
	if createdBy, ok := data["created_by"].(string); ok {
		template.CreatedBy = createdBy
	}

	// Parse semantic view IDs
	if viewIDs, ok := data["semantic_view_ids"].([]interface{}); ok {
		template.SemanticViewIDs = make([]uuid.UUID, 0, len(viewIDs))
		for _, vid := range viewIDs {
			if idStr, ok := vid.(string); ok {
				if id, err := uuid.Parse(idStr); err == nil {
					template.SemanticViewIDs = append(template.SemanticViewIDs, id)
				}
			}
		}
	}

	// Parse JSONB fields
	if layoutConfig, ok := data["layout_config"].(map[string]interface{}); ok {
		template.LayoutConfig = layoutConfig
	}
	if paramSchema, ok := data["parameter_schema"].(map[string]interface{}); ok {
		template.ParameterSchema = paramSchema
	}

	return template, nil
}

func parseTemplateSummaryFromHasura(data map[string]interface{}) (ReportTemplate, error) {
	template := ReportTemplate{}

	if id, ok := data["id"].(string); ok {
		template.ID, _ = uuid.Parse(id)
	}
	if tenantID, ok := data["tenant_id"].(string); ok {
		template.TenantID, _ = uuid.Parse(tenantID)
	}
	if name, ok := data["template_name"].(string); ok {
		template.TemplateName = name
	}
	if desc, ok := data["description"].(string); ok {
		template.Description = desc
	}
	if cat, ok := data["category"].(string); ok {
		template.Category = cat
	}
	if isActive, ok := data["is_active"].(bool); ok {
		template.IsActive = isActive
	}
	if isPublic, ok := data["is_public"].(bool); ok {
		template.IsPublic = isPublic
	}
	if version, ok := data["version"].(float64); ok {
		template.Version = int(version)
	}

	return template, nil
}

func parseExecutionFromHasura(data map[string]interface{}) (*ReportExecution, error) {
	execution := &ReportExecution{}

	if id, ok := data["id"].(string); ok {
		execution.ID, _ = uuid.Parse(id)
	}
	if tenantID, ok := data["tenant_id"].(string); ok {
		execution.TenantID, _ = uuid.Parse(tenantID)
	}
	if templateID, ok := data["template_id"].(string); ok {
		execution.TemplateID, _ = uuid.Parse(templateID)
	}
	if householdID, ok := data["household_id"].(string); ok {
		hid, _ := uuid.Parse(householdID)
		execution.HouseholdID = &hid
	}
	if params, ok := data["parameters"].(map[string]interface{}); ok {
		execution.Parameters = params
	}
	if status, ok := data["status"].(string); ok {
		execution.Status = status
	}
	if errMsg, ok := data["error_message"].(string); ok {
		execution.ErrorMessage = errMsg
	}
	if outputURL, ok := data["output_url"].(string); ok {
		execution.OutputURL = outputURL
	}
	if sizeBytes, ok := data["output_size_bytes"].(float64); ok {
		execution.OutputSizeBytes = int(sizeBytes)
	}
	if timeMS, ok := data["execution_time_ms"].(float64); ok {
		execution.ExecutionTimeMS = int(timeMS)
	}
	if rows, ok := data["rows_processed"].(float64); ok {
		execution.RowsProcessed = int(rows)
	}
	if workflowID, ok := data["workflow_id"].(string); ok {
		execution.WorkflowID = workflowID
	}
	if runID, ok := data["run_id"].(string); ok {
		execution.RunID = runID
	}
	if createdBy, ok := data["created_by"].(string); ok {
		execution.CreatedBy = createdBy
	}

	return execution, nil
}
