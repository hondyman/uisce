package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// TemplateStore - Semantic Query Template Storage Layer
// ============================================================================

type TemplateStore struct {
	db *sql.DB
}

// NewTemplateStore creates a new template store
func NewTemplateStore(db *sql.DB) *TemplateStore {
	return &TemplateStore{db: db}
}

// ============================================================================
// Create & Update Operations
// ============================================================================

// Create inserts a new template and returns the ID
func (ts *TemplateStore) Create(ctx context.Context, t *SemanticQueryTemplate) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}

	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	// Serialize query and parameters to JSONB
	queryBytes, _ := json.Marshal(t.SemanticQuery)
	paramsBytes, _ := json.Marshal(t.Parameters)
	tagsArray := pq.Array(t.Tags)

	err := ts.db.QueryRowContext(ctx, `
		INSERT INTO semantic_query_templates 
		(id, tenant_id, name, description, datasource, version, semantic_query, parameters, 
		 created_by, created_at, updated_at, visibility, tags, deprecated)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`,
		t.ID, t.TenantID, t.Name, t.Description, t.Datasource, t.Version,
		queryBytes, paramsBytes, t.CreatedBy, t.CreatedAt, t.UpdatedAt,
		t.Visibility, tagsArray, t.Deprecated,
	).Scan(&t.ID)

	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	// Create version 1 entry
	if err := ts.createVersion(ctx, t, 1, "Initial template creation"); err != nil {
		log.Printf("Warning: failed to create template version: %v", err)
	}

	log.Printf("Created template: id=%s name=%s datasource=%s", t.ID, t.Name, t.Datasource)
	return nil
}

// Update modifies an existing template and creates a new version
func (ts *TemplateStore) Update(ctx context.Context, id string, t *SemanticQueryTemplate, changeMessage string) error {
	t.UpdatedAt = time.Now()

	queryBytes, _ := json.Marshal(t.SemanticQuery)
	paramsBytes, _ := json.Marshal(t.Parameters)
	tagsArray := pq.Array(t.Tags)

	result, err := ts.db.ExecContext(ctx, `
		UPDATE semantic_query_templates
		SET name = $1, description = $2, semantic_query = $3, parameters = $4,
		    updated_at = $5, visibility = $6, tags = $7, deprecated = $8,
		    deprecation_reason = $9
		WHERE id = $1
	`,
		id, t.Name, t.Description, queryBytes, paramsBytes,
		t.UpdatedAt, t.Visibility, tagsArray, t.Deprecated, t.DeprecationReason,
	)

	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("template not found: %s", id)
	}

	// Get current version number to create next version
	var currentVersion int
	ts.db.QueryRowContext(ctx, `
		SELECT MAX(version) FROM semantic_query_template_versions WHERE template_id = $1
	`, id).Scan(&currentVersion)

	nextVersion := currentVersion + 1

	// Create version entry
	if err := ts.createVersion(ctx, t, nextVersion, changeMessage); err != nil {
		log.Printf("Warning: failed to create template version: %v", err)
	}

	log.Printf("Updated template: id=%s version=%d message=%s", id, nextVersion, changeMessage)
	return nil
}

// Delete marks a template as deprecated or hard-deletes it
func (ts *TemplateStore) Delete(ctx context.Context, id string, hardDelete bool) error {
	if hardDelete {
		// Hard delete (for admin cleanup)
		result, err := ts.db.ExecContext(ctx, `
			DELETE FROM semantic_query_templates WHERE id = $1
		`, id)

		if err != nil {
			return fmt.Errorf("failed to delete template: %w", err)
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			return fmt.Errorf("template not found: %s", id)
		}

		log.Printf("Hard-deleted template: id=%s", id)
	} else {
		// Soft delete (mark deprecated)
		now := time.Now()
		_, err := ts.db.ExecContext(ctx, `
			UPDATE semantic_query_templates
			SET deprecated = true, deprecated_at = $1, deprecation_reason = $2
			WHERE id = $3
		`, now, "User-deprecated", id)

		if err != nil {
			return fmt.Errorf("failed to deprecate template: %w", err)
		}

		log.Printf("Deprecated template: id=%s", id)
	}

	return nil
}

// ============================================================================
// Read Operations
// ============================================================================

// Get retrieves a single template by ID
func (ts *TemplateStore) Get(ctx context.Context, id string) (*SemanticQueryTemplate, error) {
	row := ts.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, name, description, datasource, version, 
		       semantic_query, parameters, created_by, created_at, updated_at,
		       visibility, tags, deprecated, deprecated_at, deprecation_reason
		FROM semantic_query_templates
		WHERE id = $1
	`, id)

	return ts.scanTemplate(row)
}

// List retrieves templates matching the given criteria
func (ts *TemplateStore) List(ctx context.Context, tenantID string, params *TemplateListQueryParams) ([]*SemanticQueryTemplate, error) {
	query := `
		SELECT id, tenant_id, name, description, datasource, version, 
		       semantic_query, parameters, created_by, created_at, updated_at,
		       visibility, tags, deprecated, deprecated_at, deprecation_reason
		FROM semantic_query_templates
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}

	// Add filters
	argNum := 2
	if params.Datasource != "" {
		query += fmt.Sprintf(" AND datasource = $%d", argNum)
		args = append(args, params.Datasource)
		argNum++
	}

	if params.Version != "" {
		query += fmt.Sprintf(" AND version = $%d", argNum)
		args = append(args, params.Version)
		argNum++
	}

	if params.CreatedBy != "" {
		query += fmt.Sprintf(" AND created_by = $%d", argNum)
		args = append(args, params.CreatedBy)
		argNum++
	}

	if params.Tag != "" {
		query += fmt.Sprintf(" AND $%d = ANY(tags)", argNum)
		args = append(args, params.Tag)
		argNum++
	}

	if params.Visibility != "" {
		query += fmt.Sprintf(" AND visibility = $%d", argNum)
		args = append(args, params.Visibility)
		argNum++
	}

	if !params.ShowDeprecated {
		query += " AND NOT deprecated"
	}

	query += " ORDER BY created_at DESC"

	// Pagination
	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", params.Limit)
	}
	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", params.Offset)
	}

	rows, err := ts.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []*SemanticQueryTemplate
	for rows.Next() {
		t, err := ts.scanTemplate(rows)
		if err != nil {
			log.Printf("Error scanning template: %v", err)
			continue
		}
		templates = append(templates, t)
	}

	return templates, rows.Err()
}

// GetByDatasourceVersion retrieves templates for a specific datasource/version combo
func (ts *TemplateStore) GetByDatasourceVersion(ctx context.Context, tenantID, datasource, version string) ([]*SemanticQueryTemplate, error) {
	params := &TemplateListQueryParams{
		Datasource: datasource,
		Version:    version,
		Limit:      100,
	}
	return ts.List(ctx, tenantID, params)
}

// scanTemplate reads a template from a SQL row
func (ts *TemplateStore) scanTemplate(row interface {
	Scan(...interface{}) error
}) (*SemanticQueryTemplate, error) {
	var t SemanticQueryTemplate
	var queryBytes, paramsBytes []byte
	var tags pq.StringArray

	err := row.Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Description, &t.Datasource, &t.Version,
		&queryBytes, &paramsBytes, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
		&t.Visibility, &tags, &t.Deprecated, &t.DeprecatedAt, &t.DeprecationReason,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to scan template: %w", err)
	}

	// Deserialize query and parameters
	t.SemanticQuery = &SemanticQuery{}
	json.Unmarshal(queryBytes, t.SemanticQuery)

	json.Unmarshal(paramsBytes, &t.Parameters)
	t.Tags = []string(tags)

	return &t, nil
}

// ============================================================================
// Versioning Operations
// ============================================================================

// createVersion creates a new version entry for a template
func (ts *TemplateStore) createVersion(ctx context.Context, t *SemanticQueryTemplate, version int, changeMessage string) error {
	queryBytes, _ := json.Marshal(t.SemanticQuery)
	paramsBytes, _ := json.Marshal(t.Parameters)

	_, err := ts.db.ExecContext(ctx, `
		INSERT INTO semantic_query_template_versions
		(version_id, template_id, version, name, semantic_query, parameters, 
		 created_at, created_by, change_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		uuid.New().String(), t.ID, version, t.Name,
		queryBytes, paramsBytes, time.Now(), t.CreatedBy, changeMessage,
	)

	return err
}

// GetVersion retrieves a specific version of a template
func (ts *TemplateStore) GetVersion(ctx context.Context, templateID string, versionNum int) (*TemplateVersion, error) {
	row := ts.db.QueryRowContext(ctx, `
		SELECT version_id, template_id, version, name, semantic_query, parameters,
		       created_at, created_by, change_message, is_promoted, promoted_at
		FROM semantic_query_template_versions
		WHERE template_id = $1 AND version = $2
	`, templateID, versionNum)

	var tv TemplateVersion
	var queryBytes, paramsBytes []byte

	err := row.Scan(
		&tv.VersionID, &tv.TemplateID, &tv.Version, &tv.Name,
		&queryBytes, &paramsBytes, &tv.CreatedAt, &tv.CreatedBy,
		&tv.ChangeMessage, &tv.IsPromoted, &tv.PromotedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("version not found")
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	tv.SemanticQuery = &SemanticQuery{}
	json.Unmarshal(queryBytes, tv.SemanticQuery)
	json.Unmarshal(paramsBytes, &tv.Parameters)

	return &tv, nil
}

// ListVersions retrieves all versions of a template
func (ts *TemplateStore) ListVersions(ctx context.Context, templateID string) ([]*TemplateVersion, error) {
	rows, err := ts.db.QueryContext(ctx, `
		SELECT version_id, template_id, version, name, semantic_query, parameters,
		       created_at, created_by, change_message, is_promoted, promoted_at
		FROM semantic_query_template_versions
		WHERE template_id = $1
		ORDER BY version DESC
	`, templateID)

	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []*TemplateVersion
	for rows.Next() {
		var tv TemplateVersion
		var queryBytes, paramsBytes []byte

		if err := rows.Scan(
			&tv.VersionID, &tv.TemplateID, &tv.Version, &tv.Name,
			&queryBytes, &paramsBytes, &tv.CreatedAt, &tv.CreatedBy,
			&tv.ChangeMessage, &tv.IsPromoted, &tv.PromotedAt,
		); err != nil {
			log.Printf("Error scanning version: %v", err)
			continue
		}

		tv.SemanticQuery = &SemanticQuery{}
		json.Unmarshal(queryBytes, tv.SemanticQuery)
		json.Unmarshal(paramsBytes, &tv.Parameters)

		versions = append(versions, &tv)
	}

	return versions, rows.Err()
}

// ============================================================================
// Permission Operations
// ============================================================================

// SetPermission sets RBAC permissions for a template
func (ts *TemplateStore) SetPermission(ctx context.Context, perm *TemplatePermission) error {
	_, err := ts.db.ExecContext(ctx, `
		INSERT INTO semantic_query_template_permissions
		(template_id, role, can_run, can_edit, can_delete, can_promote)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (template_id, role) 
		DO UPDATE SET can_run = $3, can_edit = $4, can_delete = $5, can_promote = $6
	`,
		perm.TemplateID, perm.Role, perm.CanRun, perm.CanEdit, perm.CanDelete, perm.CanPromote,
	)

	return err
}

// GetPermission retrieves permissions for a template and role
func (ts *TemplateStore) GetPermission(ctx context.Context, templateID, role string) (*TemplatePermission, error) {
	row := ts.db.QueryRowContext(ctx, `
		SELECT template_id, role, can_run, can_edit, can_delete, can_promote
		FROM semantic_query_template_permissions
		WHERE template_id = $1 AND role = $2
	`, templateID, role)

	var perm TemplatePermission
	err := row.Scan(&perm.TemplateID, &perm.Role, &perm.CanRun, &perm.CanEdit, &perm.CanDelete, &perm.CanPromote)

	if err != nil {
		if err == sql.ErrNoRows {
			// Default permissions if not found
			return &TemplatePermission{
				TemplateID: templateID,
				Role:       role,
				CanRun:     true,
				CanEdit:    false,
				CanDelete:  false,
				CanPromote: false,
			}, nil
		}
		return nil, err
	}

	return &perm, nil
}

// ============================================================================
// Statistics & Metrics
// ============================================================================

// GetTemplateStats returns usage statistics for a template
type TemplateStats struct {
	TemplateID  string    `json:"template_id"`
	RunCount    int       `json:"run_count"`
	LastRunAt   *time.Time `json:"last_run_at"`
	Viewers     int       `json:"viewers"`
	AvgRunTime  int64     `json:"avg_run_time_ms"`
}

// GetStats retrieves usage statistics for a template
func (ts *TemplateStore) GetStats(ctx context.Context, templateID string) (*TemplateStats, error) {
	row := ts.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as run_count,
			MAX(executed_at) as last_run_at,
			AVG(duration_ms) as avg_run_time
		FROM semantic_query_template_executions
		WHERE template_id = $1
	`, templateID)

	var stats TemplateStats
	stats.TemplateID = templateID

	err := row.Scan(&stats.RunCount, &stats.LastRunAt, &stats.AvgRunTime)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return &stats, nil
}
