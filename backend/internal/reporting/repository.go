package reporting

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles database operations for reporting
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new reporting repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// REPORT DEFINITIONS
// ============================================================================

// CreateDefinition creates a new report definition
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation CreateReportDefinition($object: report_definitions_insert_input!) {
//	  insert_report_definitions_one(object: $object) {
//	    id
//	    tenant_id
//	    report_key
//	    display_name
//	    definition
//	    parameters_schema
//	    ...
//	  }
//	}
//
// Note: JSONB fields for definition, tags, output_formats, parameters_schema
func (r *Repository) CreateDefinition(ctx context.Context, def *ReportDefinition) error {
	// Serialize JSON fields
	if def.Definition != nil {
		data, err := json.Marshal(def.Definition)
		if err != nil {
			return fmt.Errorf("failed to marshal definition: %w", err)
		}
		def.DefinitionJSON = data
	}

	if len(def.Tags) > 0 {
		data, err := json.Marshal(def.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}
		def.TagsJSON = data
	}

	if len(def.OutputFormats) > 0 {
		data, err := json.Marshal(def.OutputFormats)
		if err != nil {
			return fmt.Errorf("failed to marshal output_formats: %w", err)
		}
		def.OutputFormatsJSON = data
	}

	if len(def.ParametersSchema) > 0 {
		data, err := json.Marshal(def.ParametersSchema)
		if err != nil {
			return fmt.Errorf("failed to marshal parameters_schema: %w", err)
		}
		def.ParametersJSON = data
	}

	query := `
		INSERT INTO report_definitions (
			id, tenant_id, tenant_datasource_id, report_key, display_name, description,
			category, tags, report_type, output_formats, definition, parameters_schema,
			semantic_cube_id, semantic_query, version, is_current, previous_version_id,
			is_core, base_report_id, status, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)`

	if def.ID == uuid.Nil {
		def.ID = uuid.New()
	}

	_, err := r.db.ExecContext(ctx, query,
		def.ID, def.TenantID, def.TenantDatasourceID, def.ReportKey, def.DisplayName, def.Description,
		def.Category, def.TagsJSON, def.ReportType, def.OutputFormatsJSON, def.DefinitionJSON, def.ParametersJSON,
		def.SemanticCubeID, def.SemanticQuery, def.Version, def.IsCurrent, def.PreviousVersionID,
		def.IsCore, def.BaseReportID, def.Status, def.CreatedBy,
	)
	return err
}

// GetDefinition retrieves a report definition by ID
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetReportDefinition($id: uuid!) {
//	  report_definitions_by_pk(id: $id) { ... }
//	}
func (r *Repository) GetDefinition(ctx context.Context, id uuid.UUID) (*ReportDefinition, error) {
	var def ReportDefinition
	query := `SELECT * FROM report_definitions WHERE id = $1`

	err := r.db.GetContext(ctx, &def, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Deserialize JSON fields
	if err := r.deserializeDefinition(&def); err != nil {
		return nil, err
	}

	return &def, nil
}

// GetDefinitionByKey retrieves a report definition by key
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetReportDefinitionByKey($tenant_id: uuid!, $datasource_id: uuid!, $report_key: String!) {
//	  report_definitions(where: {tenant_id: {_eq: $tenant_id}, tenant_datasource_id: {_eq: $datasource_id}, report_key: {_eq: $report_key}, is_current: {_eq: true}}, limit: 1) { ... }
//	}
func (r *Repository) GetDefinitionByKey(ctx context.Context, tenantID, datasourceID uuid.UUID, reportKey string) (*ReportDefinition, error) {
	var def ReportDefinition
	query := `
		SELECT * FROM report_definitions 
		WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND report_key = $3 AND is_current = true`

	err := r.db.GetContext(ctx, &def, query, tenantID, datasourceID, reportKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := r.deserializeDefinition(&def); err != nil {
		return nil, err
	}

	return &def, nil
}

// ListDefinitions lists report definitions for a tenant
// TODO: Migrate to Hasura GraphQL query with optional filters:
//
//	query ListReportDefinitions($tenant_id: uuid!, $datasource_id: uuid!, $category: String, $status: String, $is_core: Boolean) {
//	  report_definitions(where: {tenant_id: {_eq: $tenant_id}, tenant_datasource_id: {_eq: $datasource_id}, is_current: {_eq: true}, category: {_eq: $category}, status: {_eq: $status}, is_core: {_eq: $is_core}}, order_by: {display_name: asc}) { ... }
//	}
//
// Note: Dynamic WHERE clause construction with optional category, status, is_core filters
func (r *Repository) ListDefinitions(ctx context.Context, tenantID, datasourceID uuid.UUID, filters map[string]interface{}) ([]ReportDefinition, error) {
	query := `
		SELECT * FROM report_definitions 
		WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND is_current = true`

	args := []interface{}{tenantID, datasourceID}
	argIdx := 3

	if category, ok := filters["category"].(string); ok && category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if isCore, ok := filters["is_core"].(bool); ok {
		query += fmt.Sprintf(" AND is_core = $%d", argIdx)
		args = append(args, isCore)
	}

	query += " ORDER BY display_name"

	var defs []ReportDefinition
	err := r.db.SelectContext(ctx, &defs, query, args...)
	if err != nil {
		return nil, err
	}

	// Deserialize JSON fields
	for i := range defs {
		if err := r.deserializeDefinition(&defs[i]); err != nil {
			return nil, err
		}
	}

	return defs, nil
}

// UpdateDefinition updates a report definition
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation UpdateReportDefinition($id: uuid!, $_set: report_definitions_set_input!) {
//	  update_report_definitions_by_pk(pk_columns: {id: $id}, _set: $_set) { ... }
//	}
func (r *Repository) UpdateDefinition(ctx context.Context, def *ReportDefinition) error {
	// Serialize JSON fields
	if def.Definition != nil {
		data, err := json.Marshal(def.Definition)
		if err != nil {
			return fmt.Errorf("failed to marshal definition: %w", err)
		}
		def.DefinitionJSON = data
	}

	if len(def.Tags) > 0 {
		data, err := json.Marshal(def.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}
		def.TagsJSON = data
	}

	query := `
		UPDATE report_definitions SET
			display_name = $2, description = $3, category = $4, tags = $5,
			definition = $6, parameters_schema = $7, semantic_query = $8,
			status = $9, updated_by = $10, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		def.ID, def.DisplayName, def.Description, def.Category, def.TagsJSON,
		def.DefinitionJSON, def.ParametersJSON, def.SemanticQuery,
		def.Status, def.UpdatedBy,
	)
	return err
}

// DeleteDefinition soft-deletes a report definition
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation DeleteReportDefinition($id: uuid!) {
//	  update_report_definitions_by_pk(pk_columns: {id: $id}, _set: {status: "deleted", is_current: false}) { id }
//	}
func (r *Repository) DeleteDefinition(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE report_definitions SET status = 'deleted', is_current = false WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// PublishDefinition publishes a report definition
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation PublishReportDefinition($id: uuid!, $user_id: uuid!) {
//	  update_report_definitions_by_pk(pk_columns: {id: $id}, _set: {status: "published", published_at: "now()", published_by: $user_id}) { ... }
//	}
func (r *Repository) PublishDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE report_definitions SET 
			status = 'published', published_at = NOW(), published_by = $2
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}

// ============================================================================
// REPORT EXTENSIONS
// ============================================================================

// CreateExtension creates a new report extension
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation CreateReportExtension($object: report_extensions_insert_input!) {
//	  insert_report_extensions_one(object: $object) { ... }
//	}
//
// Note: Tenant customizations of core reports with overrides, additions, removals
func (r *Repository) CreateExtension(ctx context.Context, ext *ReportExtension) error {
	query := `
		INSERT INTO report_extensions (
			id, tenant_id, tenant_datasource_id, base_report_id, extension_key, extension_name,
			description, extension_definition, overrides, additions, removals, parameter_defaults,
			version, is_current, core_version_target, status, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)`

	if ext.ID == uuid.Nil {
		ext.ID = uuid.New()
	}

	_, err := r.db.ExecContext(ctx, query,
		ext.ID, ext.TenantID, ext.TenantDatasourceID, ext.BaseReportID, ext.ExtensionKey, ext.ExtensionName,
		ext.Description, ext.ExtensionDefinition, ext.Overrides, ext.Additions, ext.Removals, ext.ParameterDefaults,
		ext.Version, ext.IsCurrent, ext.CoreVersionTarget, ext.Status, ext.CreatedBy,
	)
	return err
}

// GetExtension retrieves a report extension by ID
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetReportExtension($id: uuid!) {
//	  report_extensions_by_pk(id: $id) { ... }
//	}
func (r *Repository) GetExtension(ctx context.Context, id uuid.UUID) (*ReportExtension, error) {
	var ext ReportExtension
	query := `SELECT * FROM report_extensions WHERE id = $1`

	err := r.db.GetContext(ctx, &ext, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ext, nil
}

// ListExtensions lists extensions for a base report
// TODO: Migrate to Hasura GraphQL query:
//
//	query ListReportExtensions($tenant_id: uuid!, $datasource_id: uuid!, $base_report_id: uuid!) {
//	  report_extensions(where: {tenant_id: {_eq: $tenant_id}, tenant_datasource_id: {_eq: $datasource_id}, base_report_id: {_eq: $base_report_id}, is_current: {_eq: true}}, order_by: {extension_name: asc}) { ... }
//	}
func (r *Repository) ListExtensions(ctx context.Context, tenantID, datasourceID, baseReportID uuid.UUID) ([]ReportExtension, error) {
	query := `
		SELECT * FROM report_extensions 
		WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND base_report_id = $3 AND is_current = true
		ORDER BY extension_name`

	var exts []ReportExtension
	err := r.db.SelectContext(ctx, &exts, query, tenantID, datasourceID, baseReportID)
	return exts, err
}

// ListAllExtensions lists all extensions for a tenant
// TODO: Migrate to Hasura GraphQL query:
//
//	query ListAllReportExtensions($tenant_id: uuid!, $datasource_id: uuid!) {
//	  report_extensions(where: {tenant_id: {_eq: $tenant_id}, tenant_datasource_id: {_eq: $datasource_id}, is_current: {_eq: true}}, order_by: {extension_name: asc}) { ... }
//	}
func (r *Repository) ListAllExtensions(ctx context.Context, tenantID, datasourceID uuid.UUID) ([]ReportExtension, error) {
	query := `
		SELECT * FROM report_extensions 
		WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND is_current = true
		ORDER BY extension_name`

	var exts []ReportExtension
	err := r.db.SelectContext(ctx, &exts, query, tenantID, datasourceID)
	return exts, err
}

// ============================================================================
// REPORT INSTANCES
// ============================================================================

// CreateInstance creates a new report instance
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation CreateReportInstance($object: report_instances_insert_input!) {
//	  insert_report_instances_one(object: $object) { ... }
//	}
//
// Note: Report generation job tracking with merged definition, context, parameters
func (r *Repository) CreateInstance(ctx context.Context, inst *ReportInstance) error {
	query := `
		INSERT INTO report_instances (
			id, tenant_id, tenant_datasource_id, report_definition_id, report_extension_id,
			merged_definition, context_type, context_id, context_name, parameters,
			output_format, status, requested_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	if inst.ID == uuid.Nil {
		inst.ID = uuid.New()
	}

	_, err := r.db.ExecContext(ctx, query,
		inst.ID, inst.TenantID, inst.TenantDatasourceID, inst.ReportDefinitionID, inst.ReportExtensionID,
		inst.MergedDefinition, inst.ContextType, inst.ContextID, inst.ContextName, inst.Parameters,
		inst.OutputFormat, inst.Status, inst.RequestedBy,
	)
	return err
}

// GetInstance retrieves a report instance by ID
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetReportInstance($id: uuid!) {
//	  report_instances_by_pk(id: $id) { ... }
//	}
func (r *Repository) GetInstance(ctx context.Context, id uuid.UUID) (*ReportInstance, error) {
	var inst ReportInstance
	query := `SELECT * FROM report_instances WHERE id = $1`

	err := r.db.GetContext(ctx, &inst, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &inst, nil
}

// UpdateInstanceStatus updates an instance status
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation UpdateInstanceStatus($id: uuid!, $status: String!, $error_message: String) {
//	  update_report_instances_by_pk(pk_columns: {id: $id}, _set: {status: $status, error_message: $error_message}) { ... }
//	}
func (r *Repository) UpdateInstanceStatus(ctx context.Context, id uuid.UUID, status string, errorMsg string) error {
	query := `UPDATE report_instances SET status = $2, error_message = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status, errorMsg)
	return err
}

// UpdateInstanceComplete marks an instance as complete
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation UpdateInstanceComplete($id: uuid!, $output_url: String!, $metadata: jsonb!, $generation_time_ms: Int!) {
//	  update_report_instances_by_pk(pk_columns: {id: $id}, _set: {status: "completed", output_url: $output_url, output_metadata: $metadata, generation_time_ms: $generation_time_ms, completed_at: "now()"}) { ... }
//	}
func (r *Repository) UpdateInstanceComplete(ctx context.Context, id uuid.UUID, outputURL string, metadata json.RawMessage, generationTimeMs int) error {
	query := `
		UPDATE report_instances SET 
			status = 'completed', output_url = $2, output_metadata = $3,
			generation_time_ms = $4, completed_at = NOW()
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, outputURL, metadata, generationTimeMs)
	return err
}

// ListInstances lists report instances
// TODO: Migrate to Hasura GraphQL query:
//
//	query ListReportInstances($tenant_id: uuid!, $datasource_id: uuid!, $limit: Int!) {
//	  report_instances(where: {tenant_id: {_eq: $tenant_id}, tenant_datasource_id: {_eq: $datasource_id}}, order_by: {requested_at: desc}, limit: $limit) { ... }
//	}
func (r *Repository) ListInstances(ctx context.Context, tenantID, datasourceID uuid.UUID, limit int) ([]ReportInstance, error) {
	query := `
		SELECT * FROM report_instances 
		WHERE tenant_id = $1 AND tenant_datasource_id = $2
		ORDER BY requested_at DESC
		LIMIT $3`

	var instances []ReportInstance
	err := r.db.SelectContext(ctx, &instances, query, tenantID, datasourceID, limit)
	return instances, err
}

// ============================================================================
// REPORT SCHEDULES
// ============================================================================

// CreateSchedule creates a new report schedule
// TODO: Migrate to Hasura GraphQL mutation:
//
//	mutation CreateReportSchedule($object: report_schedules_insert_input!) {
//	  insert_report_schedules_one(object: $object) { ... }
//	}
//
// Note: Cron-based scheduled report generation with context query for dynamic recipients
func (r *Repository) CreateSchedule(ctx context.Context, sched *ReportSchedule) error {
	query := `
		INSERT INTO report_schedules (
			id, tenant_id, tenant_datasource_id, report_definition_id, report_extension_id,
			schedule_name, description, cron_expression, timezone, parameters_template,
			context_type, context_query, fixed_context_id, output_formats, delivery_config,
			is_active, next_run_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)`

	if sched.ID == uuid.Nil {
		sched.ID = uuid.New()
	}

	_, err := r.db.ExecContext(ctx, query,
		sched.ID, sched.TenantID, sched.TenantDatasourceID, sched.ReportDefinitionID, sched.ReportExtensionID,
		sched.ScheduleName, sched.Description, sched.CronExpression, sched.Timezone, sched.ParametersTemplate,
		sched.ContextType, sched.ContextQuery, sched.FixedContextID, sched.OutputFormats, sched.DeliveryConfig,
		sched.IsActive, sched.NextRunAt, sched.CreatedBy,
	)
	return err
}

// GetSchedule retrieves a schedule by ID
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetReportSchedule($id: uuid!) {
//	  report_schedules_by_pk(id: $id) { ... }
//	}
func (r *Repository) GetSchedule(ctx context.Context, id uuid.UUID) (*ReportSchedule, error) {
	var sched ReportSchedule
	query := `SELECT * FROM report_schedules WHERE id = $1`

	err := r.db.GetContext(ctx, &sched, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &sched, nil
}

// ListSchedules lists schedules for a tenant
// TODO: Migrate to Hasura GraphQL query:
//
//	query ListReportSchedules($tenant_id: uuid!, $datasource_id: uuid!) {
//	  report_schedules(where: {tenant_id: {_eq: $tenant_id}, tenant_datasource_id: {_eq: $datasource_id}}, order_by: {schedule_name: asc}) { ... }
//	}
func (r *Repository) ListSchedules(ctx context.Context, tenantID, datasourceID uuid.UUID) ([]ReportSchedule, error) {
	query := `
		SELECT * FROM report_schedules 
		WHERE tenant_id = $1 AND tenant_datasource_id = $2
		ORDER BY schedule_name`

	var schedules []ReportSchedule
	err := r.db.SelectContext(ctx, &schedules, query, tenantID, datasourceID)
	return schedules, err
}

// GetDueSchedules gets schedules that need to run
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetDueSchedules($now: timestamptz!) {
//	  report_schedules(where: {is_active: {_eq: true}, next_run_at: {_lte: $now}}, order_by: {next_run_at: asc}) { ... }
//	}
func (r *Repository) GetDueSchedules(ctx context.Context) ([]ReportSchedule, error) {
	query := `
		SELECT * FROM report_schedules 
		WHERE is_active = true AND next_run_at <= $1
		ORDER BY next_run_at`

	var schedules []ReportSchedule
	err := r.db.SelectContext(ctx, &schedules, query, time.Now())
	return schedules, err
}

// UpdateScheduleRun updates schedule after a run
// TODO: Migrate to Hasura GraphQL mutation with _inc:
//
//	mutation UpdateScheduleRun($id: uuid!, $status: String!, $error: String, $next_run: timestamptz) {
//	  update_report_schedules_by_pk(pk_columns: {id: $id}, _set: {last_run_at: "now()", last_run_status: $status, last_run_error: $error, next_run_at: $next_run}, _inc: {run_count: 1}) { ... }
//	}
func (r *Repository) UpdateScheduleRun(ctx context.Context, id uuid.UUID, status string, errMsg string, nextRun *time.Time) error {
	query := `
		UPDATE report_schedules SET 
			last_run_at = NOW(), last_run_status = $2, last_run_error = $3,
			next_run_at = $4, run_count = run_count + 1
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status, errMsg, nextRun)
	return err
}

// ============================================================================
// REPORT PACKAGES
// ============================================================================

// GetPackage retrieves a provisioning package by key
// TODO: Migrate to Hasura GraphQL query:
//
//	query GetReportPackage($package_key: String!) {
//	  report_packages(where: {package_key: {_eq: $package_key}, is_active: {_eq: true}}, limit: 1) { ... }
//	}
func (r *Repository) GetPackage(ctx context.Context, packageKey string) (*ReportPackage, error) {
	var pkg ReportPackage
	query := `SELECT * FROM report_packages WHERE package_key = $1 AND is_active = true`

	err := r.db.GetContext(ctx, &pkg, query, packageKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &pkg, nil
}

// ListPackages lists all available packages
// TODO: Migrate to Hasura GraphQL query:
//
//	query ListReportPackages {
//	  report_packages(where: {is_active: {_eq: true}}, order_by: {display_name: asc}) { ... }
//	}
func (r *Repository) ListPackages(ctx context.Context) ([]ReportPackage, error) {
	query := `SELECT * FROM report_packages WHERE is_active = true ORDER BY display_name`

	var packages []ReportPackage
	err := r.db.SelectContext(ctx, &packages, query)
	return packages, err
}

// ============================================================================
// HELPERS
// ============================================================================

func (r *Repository) deserializeDefinition(def *ReportDefinition) error {
	if len(def.DefinitionJSON) > 0 {
		var layout ReportLayout
		if err := json.Unmarshal(def.DefinitionJSON, &layout); err != nil {
			return fmt.Errorf("failed to unmarshal definition: %w", err)
		}
		def.Definition = &layout
	}

	if len(def.TagsJSON) > 0 {
		var tags []string
		if err := json.Unmarshal(def.TagsJSON, &tags); err != nil {
			return fmt.Errorf("failed to unmarshal tags: %w", err)
		}
		def.Tags = tags
	}

	if len(def.OutputFormatsJSON) > 0 {
		var formats []string
		if err := json.Unmarshal(def.OutputFormatsJSON, &formats); err != nil {
			return fmt.Errorf("failed to unmarshal output_formats: %w", err)
		}
		def.OutputFormats = formats
	}

	if len(def.ParametersJSON) > 0 {
		var params []Parameter
		if err := json.Unmarshal(def.ParametersJSON, &params); err != nil {
			return fmt.Errorf("failed to unmarshal parameters_schema: %w", err)
		}
		def.ParametersSchema = params
	}

	return nil
}
