package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/db/queries"
	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CheckDuplicateName verifies whether an active report template with the given case-insensitive
// name already exists in the specified tenant (excluding the given report ID).
func (r *Repository) CheckDuplicateName(ctx context.Context, tenantID uuid.UUID, name string, excludeID uuid.UUID) error {
	query := `
		SELECT COUNT(*) 
		FROM report_templates 
		WHERE tenant_id = $1 
		  AND LOWER(template_name) = LOWER($2) 
		  AND is_active = true 
		  AND id != $3
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID, name, excludeID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check duplicate report name: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: report name '%s' already in use in this tenant", ErrConflict, name)
	}
	return nil
}

// CreateTemplate creates a new report template with duplicate pre-check and 23505 mapping.
func (r *Repository) CreateTemplate(ctx context.Context, template *ReportTemplate) error {
	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}

	if err := r.CheckDuplicateName(ctx, template.TenantID, template.TemplateName, template.ID); err != nil {
		return err
	}

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
		template.IsPersonal,
		template.CreatedByID,
		template.CreatedBy,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return fmt.Errorf("%w: duplicate report template name: %v", ErrConflict, err)
		}
		return fmt.Errorf("failed to create template: %w", err)
	}

	return nil
}

// UpdateTemplate updates an existing report template with duplicate pre-check, strict tenant scoping, and 23505 mapping.
func (r *Repository) UpdateTemplate(ctx context.Context, template *ReportTemplate) error {
	if err := r.CheckDuplicateName(ctx, template.TenantID, template.TemplateName, template.ID); err != nil {
		return err
	}

	layoutJSON, err := json.Marshal(template.LayoutConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal layout config: %w", err)
	}

	paramSchemaJSON, err := json.Marshal(template.ParameterSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal parameter schema: %w", err)
	}

	res, err := r.db.ExecContext(ctx, queries.UpdateReportTemplate,
		template.TemplateName,
		template.Description,
		template.Category,
		layoutJSON,
		paramSchemaJSON,
		template.IsActive,
		template.IsPersonal,
		template.ID,
		template.TenantID,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return fmt.Errorf("%w: duplicate report template name: %v", ErrConflict, err)
		}
		return fmt.Errorf("failed to update template: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: report template not found or unauthorized", ErrNotFound)
	}

	return nil
}

// GetTemplate retrieves a single template by ID
func (r *Repository) GetTemplate(ctx context.Context, id uuid.UUID) (*ReportTemplate, error) {
	query := `
		SELECT id, tenant_id, template_name, description, category,
		       semantic_view_ids, layout_config, parameter_schema,
		       is_active, is_public, is_personal, created_by_id, created_by,
		       created_at, updated_at, version
		FROM report_templates
		WHERE id = $1
	`

	var tmpl ReportTemplate
	var layoutJSON, paramJSON, viewsJSON []byte
	var createdByID sql.NullString
	var createdBy sql.NullString

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
		&tmpl.IsPersonal,
		&createdByID,
		&createdBy,
		&tmpl.CreatedAt,
		&tmpl.UpdatedAt,
		&tmpl.Version,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: report template not found: %s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	if createdByID.Valid {
		tmpl.CreatedByID = &createdByID.String
	}
	if createdBy.Valid {
		tmpl.CreatedBy = createdBy.String
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

// ResolveGoldCopyTenantID looks up the master tenant where gold_copy = true in public.tenants.
func (r *Repository) ResolveGoldCopyTenantID(ctx context.Context) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to resolve gold_copy tenant: %w", err)
	}
	return id, nil
}

// ListTemplatesScoped lists report templates for the caller with personal-report visibility filtering,
// core report inheritance from the server-resolved gold_copy tenant, and per-caller favorite state.
func (r *Repository) ListTemplatesScoped(ctx context.Context, tenantID uuid.UUID, callerUserID string) ([]ReportTemplate, error) {
	goldCopyID, err := r.ResolveGoldCopyTenantID(ctx)
	if err != nil {
		// If gold_copy tenant cannot be resolved, strictly isolate to the caller's tenant
		goldCopyID = tenantID
	}

	query := `
		SELECT t.id, t.tenant_id, t.template_name, t.description, t.category,
		       t.layout_config, t.parameter_schema,
		       t.is_active, t.is_public, t.is_personal, t.created_by_id, t.created_by,
		       t.created_at, t.updated_at, t.version,
		       (f.template_id IS NOT NULL) AS is_favorite
		FROM report_templates t
		LEFT JOIN report_favorites f 
		       ON f.template_id = t.id 
		      AND f.tenant_id = t.tenant_id 
		      AND f.user_id = $1
		WHERE t.tenant_id IN ($2, $3)
		  AND (t.is_personal = false OR t.created_by_id = $1)
		ORDER BY t.template_name
	`

	rows, err := r.db.QueryContext(ctx, query, callerUserID, tenantID, goldCopyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list scoped templates: %w", err)
	}
	defer rows.Close()


	var templates []ReportTemplate
	for rows.Next() {
		var tmpl ReportTemplate
		var layoutJSON, paramJSON []byte
		var createdByID sql.NullString
		var createdBy sql.NullString

		if err := rows.Scan(
			&tmpl.ID,
			&tmpl.TenantID,
			&tmpl.TemplateName,
			&tmpl.Description,
			&tmpl.Category,
			&layoutJSON,
			&paramJSON,
			&tmpl.IsActive,
			&tmpl.IsPublic,
			&tmpl.IsPersonal,
			&createdByID,
			&createdBy,
			&tmpl.CreatedAt,
			&tmpl.UpdatedAt,
			&tmpl.Version,
			&tmpl.IsFavorite,
		); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		if createdByID.Valid {
			tmpl.CreatedByID = &createdByID.String
		}
		if createdBy.Valid {
			tmpl.CreatedBy = createdBy.String
		}
		if len(layoutJSON) > 0 {
			_ = json.Unmarshal(layoutJSON, &tmpl.LayoutConfig)
		}
		if len(paramJSON) > 0 {
			_ = json.Unmarshal(paramJSON, &tmpl.ParameterSchema)
		}
		templates = append(templates, tmpl)
	}

	return templates, nil
}

// ListTemplates returns all active templates for backward compatibility.
func (r *Repository) ListTemplates(ctx context.Context) ([]ReportTemplate, error) {
	query := `
		SELECT id, tenant_id, template_name, description, category,
		       layout_config, parameter_schema,
		       is_active, is_public, is_personal, created_by_id, created_by,
		       created_at, updated_at, version, false AS is_favorite
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
		var layoutJSON, paramJSON []byte
		var createdByID sql.NullString
		var createdBy sql.NullString

		if err := rows.Scan(
			&tmpl.ID,
			&tmpl.TenantID,
			&tmpl.TemplateName,
			&tmpl.Description,
			&tmpl.Category,
			&layoutJSON,
			&paramJSON,
			&tmpl.IsActive,
			&tmpl.IsPublic,
			&tmpl.IsPersonal,
			&createdByID,
			&createdBy,
			&tmpl.CreatedAt,
			&tmpl.UpdatedAt,
			&tmpl.Version,
			&tmpl.IsFavorite,
		); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		if createdByID.Valid {
			tmpl.CreatedByID = &createdByID.String
		}
		if createdBy.Valid {
			tmpl.CreatedBy = createdBy.String
		}
		if len(layoutJSON) > 0 {
			_ = json.Unmarshal(layoutJSON, &tmpl.LayoutConfig)
		}
		if len(paramJSON) > 0 {
			_ = json.Unmarshal(paramJSON, &tmpl.ParameterSchema)
		}
		templates = append(templates, tmpl)
	}

	return templates, nil
}

// SetFavorite idempotently favorites a report template for a user within their tenant.
// Uses INSERT ... SELECT FROM report_templates to guarantee the template belongs to the caller's tenant.
func (r *Repository) SetFavorite(ctx context.Context, tenantID uuid.UUID, userID string, templateID uuid.UUID) error {
	query := `
		INSERT INTO report_favorites (tenant_id, user_id, template_id)
		SELECT $1, $2, id FROM report_templates
		WHERE id = $3 AND tenant_id = $1
		ON CONFLICT (tenant_id, user_id, template_id) DO NOTHING
	`
	res, err := r.db.ExecContext(ctx, query, tenantID, userID, templateID)
	if err != nil {
		return fmt.Errorf("failed to set favorite: %w", err)
	}

	// Verify template exists in this tenant
	var exists bool
	err = r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM report_templates WHERE id = $1 AND tenant_id = $2)`, templateID, tenantID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to verify template: %w", err)
	}
	if !exists {
		return fmt.Errorf("%w: report template %s not found in tenant %s", ErrNotFound, templateID, tenantID)
	}

	_ = res
	return nil
}

// RemoveFavorite idempotently removes a report template favorite for a user within their tenant.
func (r *Repository) RemoveFavorite(ctx context.Context, tenantID uuid.UUID, userID string, templateID uuid.UUID) error {
	query := `
		DELETE FROM report_favorites
		WHERE tenant_id = $1 AND user_id = $2 AND template_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, tenantID, userID, templateID)
	if err != nil {
		return fmt.Errorf("failed to remove favorite: %w", err)
	}
	return nil
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
		return fmt.Errorf("%w: report template not found: %s", ErrNotFound, id)
	}

	return nil
}
