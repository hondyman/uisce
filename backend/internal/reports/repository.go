package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/db/queries"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
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

	_, err = r.db.ExecContext(ctx, queries.InsertReportTemplate,
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

	_, err = r.db.ExecContext(ctx, queries.UpdateReportTemplate,
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
	query := `
		SELECT id, tenant_id, template_name, description, category,
		       layout_config,
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
		var layoutJSON []byte
		if err := rows.Scan(
			&tmpl.ID,
			&tmpl.TenantID,
			&tmpl.TemplateName,
			&tmpl.Description,
			&tmpl.Category,
			&layoutJSON,
			&tmpl.IsActive,
			&tmpl.IsPublic,
			&tmpl.CreatedAt,
			&tmpl.UpdatedAt,
			&tmpl.Version,
		); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		if len(layoutJSON) > 0 {
			_ = json.Unmarshal(layoutJSON, &tmpl.LayoutConfig)
		}
		templates = append(templates, tmpl)
	}

	return templates, nil
}

// DeleteTemplate removes a template by ID
func (r *Repository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
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


