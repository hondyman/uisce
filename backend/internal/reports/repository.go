package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// Repository handles database operations for reports
type Repository struct {
	db     *sql.DB
	hasura HasuraClient
}

// NewRepository creates a new reports repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// NewRepositoryWithHasura creates a new reports repository with Hasura support
func NewRepositoryWithHasura(db *sql.DB, hasura HasuraClient) *Repository {
	return &Repository{db: db, hasura: hasura}
}

// CreateTemplate creates a new report template
func (r *Repository) CreateTemplate(ctx context.Context, template *ReportTemplate) error {
	layoutJSON, err := json.Marshal(template.LayoutConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal layout config: %w", err)
	}

	paramSchemaJSON, err := json.Marshal(template.ParameterSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal parameter schema: %w", err)
	}

	if r.hasura != nil {
		err := r.createTemplateWithHasura(ctx, template, layoutJSON, paramSchemaJSON)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	query := `
		INSERT INTO report_templates (
			id, tenant_id, template_name, description, category,
			layout_config, parameter_schema, is_active, is_public
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.ExecContext(ctx, query,
		template.ID,
		template.TenantID,
		template.TemplateName,
		template.Description,
		template.Category,
		layoutJSON,
		paramSchemaJSON,
		template.IsActive,
		template.IsPublic,
	)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// UpdateTemplate updates an existing report template
func (r *Repository) UpdateTemplate(ctx context.Context, template *ReportTemplate) error {
	layoutJSON, err := json.Marshal(template.LayoutConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal layout config: %w", err)
	}

	paramSchemaJSON, err := json.Marshal(template.ParameterSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal parameter schema: %w", err)
	}

	if r.hasura != nil {
		err := r.updateTemplateWithHasura(ctx, template, layoutJSON, paramSchemaJSON)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	query := `
		UPDATE report_templates
		SET template_name = $1,
		    description = $2,
		    category = $3,
		    layout_config = $4,
		    parameter_schema = $5,
		    is_active = $6,
		    updated_at = NOW()
		WHERE id = $7 AND tenant_id = $8
	`

	_, err = r.db.ExecContext(ctx, query,
		template.TemplateName,
		template.Description,
		template.Category,
		layoutJSON,
		paramSchemaJSON,
		template.IsActive,
		template.ID,
		template.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	return nil
}

// GetTemplate retrieves a single template by ID
func (r *Repository) GetTemplate(ctx context.Context, id uuid.UUID) (*ReportTemplate, error) {
	if r.hasura != nil {
		tmpl, err := r.getTemplateWithHasura(ctx, id)
		if err == nil {
			return tmpl, nil
		}
		fmt.Printf("Hasura query failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	query := `
		SELECT id, tenant_id, template_name, description, category,
		       semantic_view_ids, layout_config, parameter_schema,
		       is_active, is_public, created_at, updated_at, created_by, version
		FROM report_templates
		WHERE id = $1
	`

	var tmpl ReportTemplate
	var layoutJSON, paramJSON, viewsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tmpl.ID,
		&tmpl.TenantID,
		&tmpl.TemplateName,
		&tmpl.Description,
		&tmpl.Category,
		&viewsJSON,
		&layoutJSON,
		&paramJSON,
		&tmpl.IsActive,
		&tmpl.IsPublic,
		&tmpl.CreatedAt,
		&tmpl.UpdatedAt,
		&tmpl.CreatedBy,
		&tmpl.Version,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report template not found: %s", id)
		}
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	if len(layoutJSON) > 0 {
		if err := json.Unmarshal(layoutJSON, &tmpl.LayoutConfig); err != nil {
			return nil, fmt.Errorf("failed to parse layout config: %w", err)
		}
	}
	if len(paramJSON) > 0 {
		if err := json.Unmarshal(paramJSON, &tmpl.ParameterSchema); err != nil {
			return nil, fmt.Errorf("failed to parse parameter schema: %w", err)
		}
	}
	if len(viewsJSON) > 0 {
		var rawIDs []string
		if err := json.Unmarshal(viewsJSON, &rawIDs); err == nil {
			tmpl.SemanticViewIDs = make([]uuid.UUID, 0, len(rawIDs))
			for _, raw := range rawIDs {
				if idVal, err := uuid.Parse(raw); err == nil {
					tmpl.SemanticViewIDs = append(tmpl.SemanticViewIDs, idVal)
				}
			}
		}
	}

	return &tmpl, nil
}

// ListTemplates returns all templates ordered by name
func (r *Repository) ListTemplates(ctx context.Context) ([]ReportTemplate, error) {
	if r.hasura != nil {
		templates, err := r.listTemplatesWithHasura(ctx)
		if err == nil {
			return templates, nil
		}
		fmt.Printf("Hasura query failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	query := `
		SELECT id, tenant_id, template_name, description, category,
		       is_active, is_public, created_at, updated_at, version
		FROM report_templates
		ORDER BY template_name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []ReportTemplate
	for rows.Next() {
		var tmpl ReportTemplate
		if err := rows.Scan(
			&tmpl.ID,
			&tmpl.TenantID,
			&tmpl.TemplateName,
			&tmpl.Description,
			&tmpl.Category,
			&tmpl.IsActive,
			&tmpl.IsPublic,
			&tmpl.CreatedAt,
			&tmpl.UpdatedAt,
			&tmpl.Version,
		); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, tmpl)
	}

	return templates, nil
}

// DeleteTemplate removes a template by ID
func (r *Repository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	if r.hasura != nil {
		err := r.deleteTemplateWithHasura(ctx, id)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	result, err := r.db.ExecContext(ctx, `DELETE FROM report_templates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete count: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("report template not found: %s", id)
	}

	return nil
}

// createTemplateWithHasura creates a report template using Hasura GraphQL
func (r *Repository) createTemplateWithHasura(ctx context.Context, template *ReportTemplate, layoutJSON, paramSchemaJSON []byte) error {
	mutation := `
		mutation CreateReportTemplate($object: report_templates_insert_input!) {
			insert_report_templates_one(object: $object) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"object": map[string]interface{}{
			"id":               template.ID.String(),
			"tenant_id":        template.TenantID,
			"template_name":    template.TemplateName,
			"description":      template.Description,
			"category":         template.Category,
			"layout_config":    json.RawMessage(layoutJSON),
			"parameter_schema": json.RawMessage(paramSchemaJSON),
			"is_active":        template.IsActive,
			"is_public":        template.IsPublic,
		},
	}

	_, err := r.hasura.Mutate(mutation, variables)
	return err
}

// updateTemplateWithHasura updates a report template using Hasura GraphQL
func (r *Repository) updateTemplateWithHasura(ctx context.Context, template *ReportTemplate, layoutJSON, paramSchemaJSON []byte) error {
	mutation := `
		mutation UpdateReportTemplate($id: uuid!, $tenant_id: String, $updates: report_templates_set_input!) {
			update_report_templates(
where: {
id: {_eq: $id},
tenant_id: {_eq: $tenant_id}
},
_set: $updates
) {
				affected_rows
			}
		}
	`

	variables := map[string]interface{}{
		"id":        template.ID.String(),
		"tenant_id": template.TenantID,
		"updates": map[string]interface{}{
			"template_name":    template.TemplateName,
			"description":      template.Description,
			"category":         template.Category,
			"layout_config":    json.RawMessage(layoutJSON),
			"parameter_schema": json.RawMessage(paramSchemaJSON),
			"is_active":        template.IsActive,
		},
	}

	_, err := r.hasura.Mutate(mutation, variables)
	return err
}

// getTemplateWithHasura retrieves a report template using Hasura GraphQL
func (r *Repository) getTemplateWithHasura(ctx context.Context, id uuid.UUID) (*ReportTemplate, error) {
	query := `
		query GetReportTemplate($id: uuid!) {
			report_templates_by_pk(id: $id) {
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
		"id": id.String(),
	}

	result, err := r.hasura.Query(query, variables)
	if err != nil {
		return nil, err
	}

	tmplData, ok := result["report_templates_by_pk"].(map[string]interface{})
	if !ok || tmplData == nil {
		return nil, fmt.Errorf("report template not found: %s", id)
	}

	tmpl := &ReportTemplate{}

	// Parse UUID
	if idStr := getString(tmplData, "id"); idStr != "" {
		if parsed, err := uuid.Parse(idStr); err == nil {
			tmpl.ID = parsed
		}
	}

	if tenantIDStr := getString(tmplData, "tenant_id"); tenantIDStr != "" {
		if parsed, err := uuid.Parse(tenantIDStr); err == nil {
			tmpl.TenantID = parsed
		}
	}

	tmpl.TemplateName = getString(tmplData, "template_name")
	tmpl.Description = getString(tmplData, "description")
	tmpl.Category = getString(tmplData, "category")
	tmpl.IsActive = getBool(tmplData, "is_active")
	tmpl.IsPublic = getBool(tmplData, "is_public")
	tmpl.CreatedBy = getString(tmplData, "created_by")
	tmpl.Version = getInt(tmplData, "version")

	// Parse timestamps
	tmpl.CreatedAt = parseTime(tmplData, "created_at")
	tmpl.UpdatedAt = parseTime(tmplData, "updated_at")

	// Parse JSONB fields
	if layoutData, ok := tmplData["layout_config"]; ok && layoutData != nil {
		layoutJSON, err := json.Marshal(layoutData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal layout config: %w", err)
		}
		if err := json.Unmarshal(layoutJSON, &tmpl.LayoutConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal layout config: %w", err)
		}
	}

	if paramData, ok := tmplData["parameter_schema"]; ok && paramData != nil {
		paramJSON, err := json.Marshal(paramData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameter schema: %w", err)
		}
		if err := json.Unmarshal(paramJSON, &tmpl.ParameterSchema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameter schema: %w", err)
		}
	}

	// Parse semantic_view_ids array
	if viewsData, ok := tmplData["semantic_view_ids"]; ok && viewsData != nil {
		if viewsArr, ok := viewsData.([]interface{}); ok {
			tmpl.SemanticViewIDs = make([]uuid.UUID, 0, len(viewsArr))
			for _, v := range viewsArr {
				if vStr, ok := v.(string); ok {
					if parsed, err := uuid.Parse(vStr); err == nil {
						tmpl.SemanticViewIDs = append(tmpl.SemanticViewIDs, parsed)
					}
				}
			}
		}
	}

	return tmpl, nil
}

// listTemplatesWithHasura retrieves all report templates using Hasura GraphQL
func (r *Repository) listTemplatesWithHasura(ctx context.Context) ([]ReportTemplate, error) {
	query := `
		query ListReportTemplates {
			report_templates(order_by: {template_name: asc}) {
				id
				tenant_id
				template_name
				description
				category
				is_active
				is_public
				created_at
				updated_at
				version
			}
		}
	`

	result, err := r.hasura.Query(query, nil)
	if err != nil {
		return nil, err
	}

	tmplList, ok := result["report_templates"].([]interface{})
	if !ok {
		return []ReportTemplate{}, nil
	}

	templates := make([]ReportTemplate, 0, len(tmplList))
	for _, item := range tmplList {
		tmplData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		tmpl := ReportTemplate{
			TemplateName: getString(tmplData, "template_name"),
			Description:  getString(tmplData, "description"),
			Category:     getString(tmplData, "category"),
			IsActive:     getBool(tmplData, "is_active"),
			IsPublic:     getBool(tmplData, "is_public"),
			Version:      getInt(tmplData, "version"),
			CreatedAt:    parseTime(tmplData, "created_at"),
			UpdatedAt:    parseTime(tmplData, "updated_at"),
		}

		// Parse UUIDs
		if idStr := getString(tmplData, "id"); idStr != "" {
			if parsed, err := uuid.Parse(idStr); err == nil {
				tmpl.ID = parsed
			}
		}
		if tenantIDStr := getString(tmplData, "tenant_id"); tenantIDStr != "" {
			if parsed, err := uuid.Parse(tenantIDStr); err == nil {
				tmpl.TenantID = parsed
			}
		}

		templates = append(templates, tmpl)
	}

	return templates, nil
}

// deleteTemplateWithHasura deletes a report template using Hasura GraphQL
func (r *Repository) deleteTemplateWithHasura(ctx context.Context, id uuid.UUID) error {
	mutation := `
		mutation DeleteReportTemplate($id: uuid!) {
			delete_report_templates(where: {id: {_eq: $id}}) {
				affected_rows
			}
		}
	`

	variables := map[string]interface{}{
		"id": id.String(),
	}

	result, err := r.hasura.Mutate(mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	// Check affected rows
	if deleteData, ok := result["delete_report_templates"].(map[string]interface{}); ok {
		if affectedRows := getInt(deleteData, "affected_rows"); affectedRows == 0 {
			return fmt.Errorf("report template not found: %s", id)
		}
	}

	return nil
}

// Helper functions for type extraction
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	if val, ok := data[key]; ok && val != nil {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case int64:
			return int(v)
		}
	}
	return 0
}

func getBool(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok && val != nil {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func parseTime(data map[string]interface{}, key string) time.Time {
	if val, ok := data[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}
