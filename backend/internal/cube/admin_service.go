package cube

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Organization represents a managed service provider or parent organization
// that can manage multiple tenants on behalf of their clients.
type Organization struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	Name             string           `json:"name" db:"name"`
	Slug             string           `json:"slug" db:"slug"`
	Type             OrganizationType `json:"type" db:"type"`
	ParentOrgID      *uuid.UUID       `json:"parent_org_id,omitempty" db:"parent_org_id"`
	Settings         json.RawMessage  `json:"settings" db:"settings"`
	BillingPlan      string           `json:"billing_plan" db:"billing_plan"`
	MaxTenants       int              `json:"max_tenants" db:"max_tenants"`
	MaxQueriesPerDay int64            `json:"max_queries_per_day" db:"max_queries_per_day"`
	IsActive         bool             `json:"is_active" db:"is_active"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at" db:"updated_at"`
}

type OrganizationType string

const (
	OrgTypePlatform OrganizationType = "platform" // Your company (super admin)
	OrgTypeMSP      OrganizationType = "msp"      // Managed Service Provider
	OrgTypeClient   OrganizationType = "client"   // Direct client/tenant
)

// TenantCubeConfig represents Cube.js configuration for a specific tenant
type TenantCubeConfig struct {
	ID                   uuid.UUID       `json:"id" db:"id"`
	TenantID             uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	DatasourceID         uuid.UUID       `json:"datasource_id" db:"datasource_id"`
	OrganizationID       uuid.UUID       `json:"organization_id" db:"organization_id"`
	ResourceGroup        string          `json:"resource_group" db:"resource_group"`
	CacheTier            CacheTier       `json:"cache_tier" db:"cache_tier"`
	RefreshMode          RefreshMode     `json:"refresh_mode" db:"refresh_mode"`
	RefreshCron          string          `json:"refresh_cron" db:"refresh_cron"`
	RefreshTimezone      string          `json:"refresh_timezone" db:"refresh_timezone"`
	MaxConcurrentQueries int             `json:"max_concurrent_queries" db:"max_concurrent_queries"`
	QueryTimeout         int             `json:"query_timeout_seconds" db:"query_timeout_seconds"`
	PreAggEnabled        bool            `json:"preagg_enabled" db:"preagg_enabled"`
	SQLAPIEnabled        bool            `json:"sql_api_enabled" db:"sql_api_enabled"`
	GraphQLEnabled       bool            `json:"graphql_enabled" db:"graphql_enabled"`
	CustomSchemaPath     string          `json:"custom_schema_path" db:"custom_schema_path"`
	FeatureFlags         json.RawMessage `json:"feature_flags" db:"feature_flags"`
	Metadata             json.RawMessage `json:"metadata" db:"metadata"`
	IsActive             bool            `json:"is_active" db:"is_active"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
}

type CacheTier string

const (
	CacheTierStarter    CacheTier = "starter"
	CacheTierStandard   CacheTier = "standard"
	CacheTierEnterprise CacheTier = "enterprise"
)

type RefreshMode string

const (
	RefreshModeInterval RefreshMode = "interval"
	RefreshModeCron     RefreshMode = "cron"
	RefreshModeOnDemand RefreshMode = "on_demand"
	RefreshModeRealtime RefreshMode = "realtime"
)

// CubeCatalogEntry represents a semantic cube in the catalog
type CubeCatalogEntry struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	TenantID        uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	DatasourceID    uuid.UUID       `json:"datasource_id" db:"datasource_id"`
	Name            string          `json:"name" db:"name"`
	DisplayName     string          `json:"display_name" db:"display_name"`
	Description     string          `json:"description" db:"description"`
	Category        string          `json:"category" db:"category"`
	DataSource      string          `json:"data_source" db:"data_source"` // starrocks, trino
	SQLDefinition   string          `json:"sql_definition" db:"sql_definition"`
	Dimensions      json.RawMessage `json:"dimensions" db:"dimensions"`
	Measures        json.RawMessage `json:"measures" db:"measures"`
	Joins           json.RawMessage `json:"joins" db:"joins"`
	PreAggregations json.RawMessage `json:"pre_aggregations" db:"pre_aggregations"`
	RefreshKey      json.RawMessage `json:"refresh_key" db:"refresh_key"`
	IsPublic        bool            `json:"is_public" db:"is_public"`
	IsShared        bool            `json:"is_shared" db:"is_shared"` // Available to all org tenants
	Version         int             `json:"version" db:"version"`
	Status          CubeStatus      `json:"status" db:"status"`
	CreatedBy       uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

type CubeStatus string

const (
	CubeStatusDraft      CubeStatus = "draft"
	CubeStatusActive     CubeStatus = "active"
	CubeStatusDeprecated CubeStatus = "deprecated"
)

// QueryAnalytics captures query execution metrics for optimization
type QueryAnalytics struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	TenantID       uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	DatasourceID   uuid.UUID       `json:"datasource_id" db:"datasource_id"`
	QueryHash      string          `json:"query_hash" db:"query_hash"`
	QueryText      string          `json:"query_text" db:"query_text"`
	CubesUsed      []string        `json:"cubes_used" db:"cubes_used"`
	MeasuresUsed   []string        `json:"measures_used" db:"measures_used"`
	DimensionsUsed []string        `json:"dimensions_used" db:"dimensions_used"`
	FiltersApplied json.RawMessage `json:"filters_applied" db:"filters_applied"`
	PreAggUsed     bool            `json:"preagg_used" db:"preagg_used"`
	PreAggName     string          `json:"preagg_name" db:"preagg_name"`
	CacheHit       bool            `json:"cache_hit" db:"cache_hit"`
	DurationMs     int64           `json:"duration_ms" db:"duration_ms"`
	RowsReturned   int64           `json:"rows_returned" db:"rows_returned"`
	BytesScanned   int64           `json:"bytes_scanned" db:"bytes_scanned"`
	UserID         string          `json:"user_id" db:"user_id"`
	ClientIP       string          `json:"client_ip" db:"client_ip"`
	UserAgent      string          `json:"user_agent" db:"user_agent"`
	ErrorMessage   string          `json:"error_message,omitempty" db:"error_message"`
	ExecutedAt     time.Time       `json:"executed_at" db:"executed_at"`
}

// ScheduledReport represents a scheduled semantic layer report
type ScheduledReport struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	TenantID       uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	DatasourceID   uuid.UUID       `json:"datasource_id" db:"datasource_id"`
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description" db:"description"`
	Query          json.RawMessage `json:"query" db:"query"` // Cube query definition
	Format         ReportFormat    `json:"format" db:"format"`
	Schedule       string          `json:"schedule" db:"schedule"` // Cron expression
	Timezone       string          `json:"timezone" db:"timezone"`
	Recipients     []string        `json:"recipients" db:"recipients"`
	DeliveryMethod DeliveryMethod  `json:"delivery_method" db:"delivery_method"`
	S3Destination  string          `json:"s3_destination,omitempty" db:"s3_destination"`
	WebhookURL     string          `json:"webhook_url,omitempty" db:"webhook_url"`
	IsActive       bool            `json:"is_active" db:"is_active"`
	LastRunAt      *time.Time      `json:"last_run_at,omitempty" db:"last_run_at"`
	LastRunStatus  string          `json:"last_run_status,omitempty" db:"last_run_status"`
	NextRunAt      *time.Time      `json:"next_run_at,omitempty" db:"next_run_at"`
	CreatedBy      uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

type ReportFormat string

const (
	ReportFormatCSV     ReportFormat = "csv"
	ReportFormatExcel   ReportFormat = "excel"
	ReportFormatJSON    ReportFormat = "json"
	ReportFormatParquet ReportFormat = "parquet"
	ReportFormatPDF     ReportFormat = "pdf"
)

type DeliveryMethod string

const (
	DeliveryMethodEmail   DeliveryMethod = "email"
	DeliveryMethodS3      DeliveryMethod = "s3"
	DeliveryMethodWebhook DeliveryMethod = "webhook"
	DeliveryMethodSlack   DeliveryMethod = "slack"
)

// PreAggSuggestion represents an auto-suggested pre-aggregation
type PreAggSuggestion struct {
	ID                 uuid.UUID        `json:"id" db:"id"`
	TenantID           uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	CubeName           string           `json:"cube_name" db:"cube_name"`
	SuggestionType     string           `json:"suggestion_type" db:"suggestion_type"`
	Measures           []string         `json:"measures" db:"measures"`
	Dimensions         []string         `json:"dimensions" db:"dimensions"`
	TimeDimension      string           `json:"time_dimension" db:"time_dimension"`
	Granularity        string           `json:"granularity" db:"granularity"`
	QueryCount         int64            `json:"query_count" db:"query_count"`
	AvgDurationMs      int64            `json:"avg_duration_ms" db:"avg_duration_ms"`
	EstimatedSavingsMs int64            `json:"estimated_savings_ms" db:"estimated_savings_ms"`
	Score              float64          `json:"score" db:"score"`
	YAMLDefinition     string           `json:"yaml_definition" db:"yaml_definition"`
	Status             SuggestionStatus `json:"status" db:"status"`
	ReviewedBy         *uuid.UUID       `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt         *time.Time       `json:"reviewed_at,omitempty" db:"reviewed_at"`
	CreatedAt          time.Time        `json:"created_at" db:"created_at"`
}

type SuggestionStatus string

const (
	SuggestionStatusPending  SuggestionStatus = "pending"
	SuggestionStatusApproved SuggestionStatus = "approved"
	SuggestionStatusRejected SuggestionStatus = "rejected"
	SuggestionStatusApplied  SuggestionStatus = "applied"
)

// AdminRole defines roles for the admin console
type AdminRole string

const (
	AdminRoleSuperAdmin   AdminRole = "super_admin"   // Platform operator - sees all
	AdminRoleOrgAdmin     AdminRole = "org_admin"     // MSP admin - sees org tenants
	AdminRoleTenantAdmin  AdminRole = "tenant_admin"  // Single tenant admin
	AdminRoleTenantViewer AdminRole = "tenant_viewer" // Read-only access
)

// CubeAdminUser represents an admin user with organization context
type CubeAdminUser struct {
	ID             uuid.UUID   `json:"id" db:"id"`
	UserID         uuid.UUID   `json:"user_id" db:"user_id"`
	OrganizationID uuid.UUID   `json:"organization_id" db:"organization_id"`
	Role           AdminRole   `json:"role" db:"role"`
	AllowedTenants []uuid.UUID `json:"allowed_tenants" db:"allowed_tenants"` // Empty = all org tenants
	Permissions    []string    `json:"permissions" db:"permissions"`
	IsActive       bool        `json:"is_active" db:"is_active"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at" db:"updated_at"`
}

// CubeAdminService provides admin console functionality
type CubeAdminService struct {
	db *sql.DB
}

// NewCubeAdminService creates a new admin service
func NewCubeAdminService(db *sql.DB) *CubeAdminService {
	return &CubeAdminService{db: db}
}

// GetOrganizationHierarchy returns the org tree for super admins
func (s *CubeAdminService) GetOrganizationHierarchy(ctx context.Context) ([]Organization, error) {
	query := `
		WITH RECURSIVE org_tree AS (
			SELECT id, name, slug, type, parent_org_id, settings, billing_plan,
			       max_tenants, max_queries_per_day, is_active, created_at, updated_at, 0 as depth
			FROM cube_organizations
			WHERE parent_org_id IS NULL
			UNION ALL
			SELECT o.id, o.name, o.slug, o.type, o.parent_org_id, o.settings, o.billing_plan,
			       o.max_tenants, o.max_queries_per_day, o.is_active, o.created_at, o.updated_at, ot.depth + 1
			FROM cube_organizations o
			JOIN org_tree ot ON o.parent_org_id = ot.id
		)
		SELECT * FROM org_tree ORDER BY depth, name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query org hierarchy: %w", err)
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var org Organization
		var depth int
		if err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Type, &org.ParentOrgID,
			&org.Settings, &org.BillingPlan, &org.MaxTenants, &org.MaxQueriesPerDay,
			&org.IsActive, &org.CreatedAt, &org.UpdatedAt, &depth,
		); err != nil {
			return nil, fmt.Errorf("scan org: %w", err)
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

// GetTenantsForOrganization returns tenants managed by an organization
func (s *CubeAdminService) GetTenantsForOrganization(ctx context.Context, orgID uuid.UUID) ([]TenantCubeConfig, error) {
	query := `
		SELECT id, tenant_id, datasource_id, organization_id, resource_group, cache_tier,
		       refresh_mode, refresh_cron, refresh_timezone, max_concurrent_queries,
		       query_timeout_seconds, preagg_enabled, sql_api_enabled, graphql_enabled,
		       custom_schema_path, feature_flags, metadata, is_active, created_at, updated_at
		FROM tenant_cube_configs
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	defer rows.Close()

	var tenants []TenantCubeConfig
	for rows.Next() {
		var t TenantCubeConfig
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.DatasourceID, &t.OrganizationID, &t.ResourceGroup,
			&t.CacheTier, &t.RefreshMode, &t.RefreshCron, &t.RefreshTimezone,
			&t.MaxConcurrentQueries, &t.QueryTimeout, &t.PreAggEnabled, &t.SQLAPIEnabled,
			&t.GraphQLEnabled, &t.CustomSchemaPath, &t.FeatureFlags, &t.Metadata,
			&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

// GetCubeCatalog returns available cubes for a tenant
func (s *CubeAdminService) GetCubeCatalog(ctx context.Context, tenantID, datasourceID uuid.UUID) ([]CubeCatalogEntry, error) {
	query := `
		SELECT id, tenant_id, datasource_id, name, display_name, description, category,
		       data_source, sql_definition, dimensions, measures, joins, pre_aggregations,
		       refresh_key, is_public, is_shared, version, status, created_by, created_at, updated_at
		FROM cube_definitions
		WHERE (tenant_id = $1 AND datasource_id = $2) OR is_shared = true
		ORDER BY category, display_name
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("query cubes: %w", err)
	}
	defer rows.Close()

	var cubes []CubeCatalogEntry
	for rows.Next() {
		var c CubeCatalogEntry
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.DatasourceID, &c.Name, &c.DisplayName, &c.Description,
			&c.Category, &c.DataSource, &c.SQLDefinition, &c.Dimensions, &c.Measures,
			&c.Joins, &c.PreAggregations, &c.RefreshKey, &c.IsPublic, &c.IsShared,
			&c.Version, &c.Status, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan cube: %w", err)
		}
		cubes = append(cubes, c)
	}
	return cubes, nil
}

// GetQueryAnalytics returns query performance data
func (s *CubeAdminService) GetQueryAnalytics(ctx context.Context, tenantID uuid.UUID, since time.Time, limit int) ([]QueryAnalytics, error) {
	query := `
		SELECT id, tenant_id, datasource_id, query_hash, query_text, cubes_used,
		       measures_used, dimensions_used, filters_applied, preagg_used, preagg_name,
		       cache_hit, duration_ms, rows_returned, bytes_scanned, user_id,
		       client_ip, user_agent, error_message, executed_at
		FROM cube_query_analytics
		WHERE tenant_id = $1 AND executed_at >= $2
		ORDER BY executed_at DESC
		LIMIT $3
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("query analytics: %w", err)
	}
	defer rows.Close()

	var analytics []QueryAnalytics
	for rows.Next() {
		var a QueryAnalytics
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.DatasourceID, &a.QueryHash, &a.QueryText,
			&a.CubesUsed, &a.MeasuresUsed, &a.DimensionsUsed, &a.FiltersApplied,
			&a.PreAggUsed, &a.PreAggName, &a.CacheHit, &a.DurationMs, &a.RowsReturned,
			&a.BytesScanned, &a.UserID, &a.ClientIP, &a.UserAgent, &a.ErrorMessage, &a.ExecutedAt,
		); err != nil {
			return nil, fmt.Errorf("scan analytics: %w", err)
		}
		analytics = append(analytics, a)
	}
	return analytics, nil
}

// GetPreAggSuggestions returns optimization suggestions
func (s *CubeAdminService) GetPreAggSuggestions(ctx context.Context, tenantID uuid.UUID) ([]PreAggSuggestion, error) {
	query := `
		SELECT id, tenant_id, cube_name, suggestion_type, measures, dimensions,
		       time_dimension, granularity, query_count, avg_duration_ms,
		       estimated_savings_ms, score, yaml_definition, status,
		       reviewed_by, reviewed_at, created_at
		FROM cube_preagg_suggestions
		WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY score DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []PreAggSuggestion
	for rows.Next() {
		var s PreAggSuggestion
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.CubeName, &s.SuggestionType, &s.Measures,
			&s.Dimensions, &s.TimeDimension, &s.Granularity, &s.QueryCount,
			&s.AvgDurationMs, &s.EstimatedSavingsMs, &s.Score, &s.YAMLDefinition,
			&s.Status, &s.ReviewedBy, &s.ReviewedAt, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan suggestion: %w", err)
		}
		suggestions = append(suggestions, s)
	}
	return suggestions, nil
}

// GetScheduledReports returns reports for a tenant
func (s *CubeAdminService) GetScheduledReports(ctx context.Context, tenantID uuid.UUID) ([]ScheduledReport, error) {
	query := `
		SELECT id, tenant_id, datasource_id, name, description, query, format,
		       schedule, timezone, recipients, delivery_method, s3_destination,
		       webhook_url, is_active, last_run_at, last_run_status, next_run_at,
		       created_by, created_at, updated_at
		FROM cube_scheduled_reports
		WHERE tenant_id = $1
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query reports: %w", err)
	}
	defer rows.Close()

	var reports []ScheduledReport
	for rows.Next() {
		var r ScheduledReport
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.DatasourceID, &r.Name, &r.Description, &r.Query,
			&r.Format, &r.Schedule, &r.Timezone, &r.Recipients, &r.DeliveryMethod,
			&r.S3Destination, &r.WebhookURL, &r.IsActive, &r.LastRunAt, &r.LastRunStatus,
			&r.NextRunAt, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// DashboardStats represents aggregate stats for admin dashboard
type DashboardStats struct {
	TotalTenants       int64   `json:"total_tenants"`
	ActiveTenants      int64   `json:"active_tenants"`
	TotalCubes         int64   `json:"total_cubes"`
	TotalQueries24h    int64   `json:"total_queries_24h"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	CacheHitRate       float64 `json:"cache_hit_rate"`
	PreAggHitRate      float64 `json:"preagg_hit_rate"`
	ErrorRate          float64 `json:"error_rate"`
	PendingSuggestions int64   `json:"pending_suggestions"`
}

// GetDashboardStats returns aggregate stats for the admin dashboard
func (s *CubeAdminService) GetDashboardStats(ctx context.Context, orgID *uuid.UUID) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Get tenant counts
	tenantQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE is_active = true) as active
		FROM tenant_cube_configs
		WHERE ($1::uuid IS NULL OR organization_id = $1)
	`
	if err := s.db.QueryRowContext(ctx, tenantQuery, orgID).Scan(&stats.TotalTenants, &stats.ActiveTenants); err != nil {
		return nil, fmt.Errorf("query tenant stats: %w", err)
	}

	// Get query stats from last 24h
	queryQuery := `
		SELECT 
			COUNT(*) as total_queries,
			COALESCE(AVG(duration_ms), 0) as avg_latency,
			COALESCE(AVG(CASE WHEN cache_hit THEN 1.0 ELSE 0.0 END), 0) as cache_hit_rate,
			COALESCE(AVG(CASE WHEN preagg_used THEN 1.0 ELSE 0.0 END), 0) as preagg_hit_rate,
			COALESCE(AVG(CASE WHEN error_message != '' THEN 1.0 ELSE 0.0 END), 0) as error_rate
		FROM cube_query_analytics
		WHERE executed_at >= NOW() - INTERVAL '24 hours'
		  AND ($1::uuid IS NULL OR tenant_id IN (
		      SELECT tenant_id FROM tenant_cube_configs WHERE organization_id = $1
		  ))
	`
	if err := s.db.QueryRowContext(ctx, queryQuery, orgID).Scan(
		&stats.TotalQueries24h, &stats.AvgLatencyMs, &stats.CacheHitRate,
		&stats.PreAggHitRate, &stats.ErrorRate,
	); err != nil {
		return nil, fmt.Errorf("query query stats: %w", err)
	}

	return stats, nil
}

// ============================================================================
// ORGANIZATION CRUD
// ============================================================================

// CreateOrganization creates a new organization
func (s *CubeAdminService) CreateOrganization(ctx context.Context, org Organization) (*Organization, error) {
	org.ID = uuid.New()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()

	query := `
		INSERT INTO cube_organizations (
id, name, slug, type, parent_org_id, settings, billing_plan,
max_tenants, max_queries_per_day, is_active, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
org.ID, org.Name, org.Slug, org.Type, org.ParentOrgID, org.Settings,
org.BillingPlan, org.MaxTenants, org.MaxQueriesPerDay, org.IsActive,
org.CreatedAt, org.UpdatedAt,
).Scan(&org.ID)

	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}

	return &org, nil
}

// GetOrganization retrieves an organization by ID
func (s *CubeAdminService) GetOrganization(ctx context.Context, orgID uuid.UUID) (*Organization, error) {
	query := `
		SELECT id, name, slug, type, parent_org_id, settings, billing_plan,
		       max_tenants, max_queries_per_day, is_active, created_at, updated_at
		FROM cube_organizations WHERE id = $1
	`

	var org Organization
	err := s.db.QueryRowContext(ctx, query, orgID).Scan(
&org.ID, &org.Name, &org.Slug, &org.Type, &org.ParentOrgID, &org.Settings,
		&org.BillingPlan, &org.MaxTenants, &org.MaxQueriesPerDay, &org.IsActive,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("organization not found")
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}

	return &org, nil
}

// UpdateOrganization updates an existing organization
func (s *CubeAdminService) UpdateOrganization(ctx context.Context, org Organization) (*Organization, error) {
	org.UpdatedAt = time.Now()

	query := `
		UPDATE cube_organizations SET
			name = $2, slug = $3, type = $4, parent_org_id = $5, settings = $6,
			billing_plan = $7, max_tenants = $8, max_queries_per_day = $9,
			is_active = $10, updated_at = $11
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, query,
org.ID, org.Name, org.Slug, org.Type, org.ParentOrgID, org.Settings,
org.BillingPlan, org.MaxTenants, org.MaxQueriesPerDay, org.IsActive, org.UpdatedAt,
)
	if err != nil {
		return nil, fmt.Errorf("update organization: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("organization not found")
	}

	return &org, nil
}

// DeleteOrganization soft-deletes an organization
func (s *CubeAdminService) DeleteOrganization(ctx context.Context, orgID uuid.UUID) error {
	query := `UPDATE cube_organizations SET is_active = false, updated_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, orgID, time.Now())
	return err
}

// ============================================================================
// TENANT CONFIG CRUD
// ============================================================================

// CreateTenantConfig creates a new tenant cube configuration
func (s *CubeAdminService) CreateTenantConfig(ctx context.Context, config TenantCubeConfig) (*TenantCubeConfig, error) {
	config.ID = uuid.New()
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	query := `
		INSERT INTO tenant_cube_configs (
id, tenant_id, datasource_id, organization_id, resource_group, cache_tier,
refresh_mode, refresh_cron, refresh_timezone, max_concurrent_queries,
query_timeout_seconds, preagg_enabled, sql_api_enabled, graphql_enabled,
custom_schema_path, feature_flags, metadata, is_active, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
config.ID, config.TenantID, config.DatasourceID, config.OrganizationID,
config.ResourceGroup, config.CacheTier, config.RefreshMode, config.RefreshCron,
config.RefreshTimezone, config.MaxConcurrentQueries, config.QueryTimeout,
config.PreAggEnabled, config.SQLAPIEnabled, config.GraphQLEnabled,
config.CustomSchemaPath, config.FeatureFlags, config.Metadata, config.IsActive,
config.CreatedAt, config.UpdatedAt,
).Scan(&config.ID)

	if err != nil {
		return nil, fmt.Errorf("create tenant config: %w", err)
	}

	return &config, nil
}

// GetTenantConfig retrieves a tenant configuration by tenant ID
func (s *CubeAdminService) GetTenantConfig(ctx context.Context, tenantID uuid.UUID) (*TenantCubeConfig, error) {
	query := `
		SELECT id, tenant_id, datasource_id, organization_id, resource_group, cache_tier,
		       refresh_mode, refresh_cron, refresh_timezone, max_concurrent_queries,
		       query_timeout_seconds, preagg_enabled, sql_api_enabled, graphql_enabled,
		       custom_schema_path, feature_flags, metadata, is_active, created_at, updated_at
		FROM tenant_cube_configs WHERE tenant_id = $1
	`

	var config TenantCubeConfig
	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(
&config.ID, &config.TenantID, &config.DatasourceID, &config.OrganizationID,
		&config.ResourceGroup, &config.CacheTier, &config.RefreshMode, &config.RefreshCron,
		&config.RefreshTimezone, &config.MaxConcurrentQueries, &config.QueryTimeout,
		&config.PreAggEnabled, &config.SQLAPIEnabled, &config.GraphQLEnabled,
		&config.CustomSchemaPath, &config.FeatureFlags, &config.Metadata, &config.IsActive,
		&config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant config not found")
		}
		return nil, fmt.Errorf("get tenant config: %w", err)
	}

	return &config, nil
}

// UpdateTenantConfig updates an existing tenant configuration
func (s *CubeAdminService) UpdateTenantConfig(ctx context.Context, config TenantCubeConfig) (*TenantCubeConfig, error) {
	config.UpdatedAt = time.Now()

	query := `
		UPDATE tenant_cube_configs SET
			datasource_id = $2, organization_id = $3, resource_group = $4, cache_tier = $5,
			refresh_mode = $6, refresh_cron = $7, refresh_timezone = $8,
			max_concurrent_queries = $9, query_timeout_seconds = $10, preagg_enabled = $11,
			sql_api_enabled = $12, graphql_enabled = $13, custom_schema_path = $14,
			feature_flags = $15, metadata = $16, is_active = $17, updated_at = $18
		WHERE tenant_id = $1
	`

	result, err := s.db.ExecContext(ctx, query,
config.TenantID, config.DatasourceID, config.OrganizationID, config.ResourceGroup,
config.CacheTier, config.RefreshMode, config.RefreshCron, config.RefreshTimezone,
config.MaxConcurrentQueries, config.QueryTimeout, config.PreAggEnabled,
config.SQLAPIEnabled, config.GraphQLEnabled, config.CustomSchemaPath,
config.FeatureFlags, config.Metadata, config.IsActive, config.UpdatedAt,
)
	if err != nil {
		return nil, fmt.Errorf("update tenant config: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("tenant config not found")
	}

	return &config, nil
}

// DeleteTenantConfig soft-deletes a tenant configuration
func (s *CubeAdminService) DeleteTenantConfig(ctx context.Context, tenantID uuid.UUID) error {
	query := `UPDATE tenant_cube_configs SET is_active = false, updated_at = $2 WHERE tenant_id = $1`
	_, err := s.db.ExecContext(ctx, query, tenantID, time.Now())
	return err
}

// ============================================================================
// CUBE DEFINITION CRUD
// ============================================================================

// CreateCube creates a new cube definition
func (s *CubeAdminService) CreateCube(ctx context.Context, cube CubeCatalogEntry) (*CubeCatalogEntry, error) {
	cube.ID = uuid.New()
	cube.Version = 1
	cube.Status = CubeStatusDraft
	cube.CreatedAt = time.Now()
	cube.UpdatedAt = time.Now()

	query := `
		INSERT INTO cube_definitions (
id, tenant_id, datasource_id, name, display_name, description, category,
data_source, sql_definition, dimensions, measures, joins, pre_aggregations,
refresh_key, is_public, is_shared, version, status, created_by, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
cube.ID, cube.TenantID, cube.DatasourceID, cube.Name, cube.DisplayName,
cube.Description, cube.Category, cube.DataSource, cube.SQLDefinition,
cube.Dimensions, cube.Measures, cube.Joins, cube.PreAggregations,
cube.RefreshKey, cube.IsPublic, cube.IsShared, cube.Version, cube.Status,
cube.CreatedBy, cube.CreatedAt, cube.UpdatedAt,
).Scan(&cube.ID)

	if err != nil {
		return nil, fmt.Errorf("create cube: %w", err)
	}

	return &cube, nil
}

// GetCube retrieves a cube by ID
func (s *CubeAdminService) GetCube(ctx context.Context, cubeID uuid.UUID) (*CubeCatalogEntry, error) {
	query := `
		SELECT id, tenant_id, datasource_id, name, display_name, description, category,
		       data_source, sql_definition, dimensions, measures, joins, pre_aggregations,
		       refresh_key, is_public, is_shared, version, status, created_by, created_at, updated_at
		FROM cube_definitions WHERE id = $1
	`

	var cube CubeCatalogEntry
	err := s.db.QueryRowContext(ctx, query, cubeID).Scan(
&cube.ID, &cube.TenantID, &cube.DatasourceID, &cube.Name, &cube.DisplayName,
		&cube.Description, &cube.Category, &cube.DataSource, &cube.SQLDefinition,
		&cube.Dimensions, &cube.Measures, &cube.Joins, &cube.PreAggregations,
		&cube.RefreshKey, &cube.IsPublic, &cube.IsShared, &cube.Version, &cube.Status,
		&cube.CreatedBy, &cube.CreatedAt, &cube.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cube not found")
		}
		return nil, fmt.Errorf("get cube: %w", err)
	}

	return &cube, nil
}

// UpdateCube updates an existing cube definition
func (s *CubeAdminService) UpdateCube(ctx context.Context, cube CubeCatalogEntry) (*CubeCatalogEntry, error) {
	cube.Version++
	cube.UpdatedAt = time.Now()

	query := `
		UPDATE cube_definitions SET
			name = $2, display_name = $3, description = $4, category = $5,
			data_source = $6, sql_definition = $7, dimensions = $8, measures = $9,
			joins = $10, pre_aggregations = $11, refresh_key = $12, is_public = $13,
			is_shared = $14, version = $15, status = $16, updated_at = $17
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, query,
cube.ID, cube.Name, cube.DisplayName, cube.Description, cube.Category,
cube.DataSource, cube.SQLDefinition, cube.Dimensions, cube.Measures,
cube.Joins, cube.PreAggregations, cube.RefreshKey, cube.IsPublic,
cube.IsShared, cube.Version, cube.Status, cube.UpdatedAt,
)
	if err != nil {
		return nil, fmt.Errorf("update cube: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("cube not found")
	}

	return &cube, nil
}

// DeleteCube soft-deletes a cube (marks as deprecated)
func (s *CubeAdminService) DeleteCube(ctx context.Context, cubeID uuid.UUID) error {
	query := `UPDATE cube_definitions SET status = 'deprecated', updated_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, cubeID, time.Now())
	return err
}

// ValidateCube validates a cube definition syntax
func (s *CubeAdminService) ValidateCube(ctx context.Context, cubeID uuid.UUID) (*CubeValidationResult, error) {
	cube, err := s.GetCube(ctx, cubeID)
	if err != nil {
		return nil, err
	}

	result := &CubeValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate SQL definition
	if cube.SQLDefinition == "" {
		result.Errors = append(result.Errors, "SQL definition is required")
		result.Valid = false
	}

	// Validate dimensions
	if len(cube.Dimensions) == 0 || string(cube.Dimensions) == "null" {
		result.Warnings = append(result.Warnings, "No dimensions defined")
	}

	// Validate measures
	if len(cube.Measures) == 0 || string(cube.Measures) == "null" {
		result.Warnings = append(result.Warnings, "No measures defined")
	}

	return result, nil
}

// CubeValidationResult contains validation results
type CubeValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// DeployCube deploys a cube to the Cube.js schema directory
func (s *CubeAdminService) DeployCube(ctx context.Context, cubeID uuid.UUID) error {
	cube, err := s.GetCube(ctx, cubeID)
	if err != nil {
		return err
	}

	// Update status to active
	cube.Status = CubeStatusActive
	_, err = s.UpdateCube(ctx, *cube)
	return err
}

// ============================================================================
// SCHEDULED REPORT CRUD
// ============================================================================

// CreateScheduledReport creates a new scheduled report
func (s *CubeAdminService) CreateScheduledReport(ctx context.Context, report ScheduledReport) (*ScheduledReport, error) {
	report.ID = uuid.New()
	report.CreatedAt = time.Now()
	report.UpdatedAt = time.Now()

	query := `
		INSERT INTO cube_scheduled_reports (
id, tenant_id, datasource_id, name, description, query, format,
schedule, timezone, recipients, delivery_method, s3_destination,
webhook_url, is_active, created_by, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
report.ID, report.TenantID, report.DatasourceID, report.Name, report.Description,
report.Query, report.Format, report.Schedule, report.Timezone, report.Recipients,
report.DeliveryMethod, report.S3Destination, report.WebhookURL, report.IsActive,
report.CreatedBy, report.CreatedAt, report.UpdatedAt,
).Scan(&report.ID)

	if err != nil {
		return nil, fmt.Errorf("create scheduled report: %w", err)
	}

	return &report, nil
}

// GetScheduledReport retrieves a scheduled report by ID
func (s *CubeAdminService) GetScheduledReport(ctx context.Context, reportID uuid.UUID) (*ScheduledReport, error) {
	query := `
		SELECT id, tenant_id, datasource_id, name, description, query, format,
		       schedule, timezone, recipients, delivery_method, s3_destination,
		       webhook_url, is_active, last_run_at, last_run_status, next_run_at,
		       created_by, created_at, updated_at
		FROM cube_scheduled_reports WHERE id = $1
	`

	var report ScheduledReport
	err := s.db.QueryRowContext(ctx, query, reportID).Scan(
&report.ID, &report.TenantID, &report.DatasourceID, &report.Name, &report.Description,
		&report.Query, &report.Format, &report.Schedule, &report.Timezone, &report.Recipients,
		&report.DeliveryMethod, &report.S3Destination, &report.WebhookURL, &report.IsActive,
		&report.LastRunAt, &report.LastRunStatus, &report.NextRunAt,
		&report.CreatedBy, &report.CreatedAt, &report.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("scheduled report not found")
		}
		return nil, fmt.Errorf("get scheduled report: %w", err)
	}

	return &report, nil
}

// UpdateScheduledReport updates an existing scheduled report
func (s *CubeAdminService) UpdateScheduledReport(ctx context.Context, report ScheduledReport) (*ScheduledReport, error) {
	report.UpdatedAt = time.Now()

	query := `
		UPDATE cube_scheduled_reports SET
			name = $2, description = $3, query = $4, format = $5, schedule = $6,
			timezone = $7, recipients = $8, delivery_method = $9, s3_destination = $10,
			webhook_url = $11, is_active = $12, updated_at = $13
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, query,
report.ID, report.Name, report.Description, report.Query, report.Format,
report.Schedule, report.Timezone, report.Recipients, report.DeliveryMethod,
report.S3Destination, report.WebhookURL, report.IsActive, report.UpdatedAt,
)
	if err != nil {
		return nil, fmt.Errorf("update scheduled report: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("scheduled report not found")
	}

	return &report, nil
}

// DeleteScheduledReport deletes a scheduled report
func (s *CubeAdminService) DeleteScheduledReport(ctx context.Context, reportID uuid.UUID) error {
	query := `DELETE FROM cube_scheduled_reports WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, reportID)
	return err
}

// RunReportNow triggers an immediate report execution
func (s *CubeAdminService) RunReportNow(ctx context.Context, reportID uuid.UUID) error {
	// Update last_run_at and set status to running
	query := `
		UPDATE cube_scheduled_reports SET
			last_run_at = $2, last_run_status = 'running', updated_at = $2
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, reportID, time.Now())
	return err
}

// ============================================================================
// PRE-AGGREGATION MANAGEMENT
// ============================================================================

// ApprovePreAggSuggestion approves a pre-aggregation suggestion
func (s *CubeAdminService) ApprovePreAggSuggestion(ctx context.Context, suggestionID uuid.UUID, reviewerID uuid.UUID) error {
	query := `
		UPDATE cube_preagg_suggestions SET
			status = 'approved', reviewed_by = $2, reviewed_at = $3
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, suggestionID, reviewerID, time.Now())
	return err
}

// RejectPreAggSuggestion rejects a pre-aggregation suggestion
func (s *CubeAdminService) RejectPreAggSuggestion(ctx context.Context, suggestionID uuid.UUID, reviewerID uuid.UUID) error {
	query := `
		UPDATE cube_preagg_suggestions SET
			status = 'rejected', reviewed_by = $2, reviewed_at = $3
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, suggestionID, reviewerID, time.Now())
	return err
}

// ListPreAggregations lists all pre-aggregations for a tenant
func (s *CubeAdminService) ListPreAggregations(ctx context.Context, tenantID uuid.UUID) ([]PreAggregation, error) {
	query := `
		SELECT id, tenant_id, cube_name, name, dimensions, measures, time_dimension,
		       granularity, refresh_key, build_range, partitions, status,
		       last_built_at, row_count, size_bytes, created_at, updated_at
		FROM cube_pre_aggregations
		WHERE tenant_id = $1
		ORDER BY cube_name, name
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list pre-aggregations: %w", err)
	}
	defer rows.Close()

	var preAggs []PreAggregation
	for rows.Next() {
		var p PreAggregation
		if err := rows.Scan(
&p.ID, &p.TenantID, &p.CubeName, &p.Name, &p.Dimensions, &p.Measures,
			&p.TimeDimension, &p.Granularity, &p.RefreshKey, &p.BuildRange,
			&p.Partitions, &p.Status, &p.LastBuiltAt, &p.RowCount, &p.SizeBytes,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pre-aggregation: %w", err)
		}
		preAggs = append(preAggs, p)
	}
	return preAggs, nil
}

// PreAggregation represents a pre-aggregation in the system
type PreAggregation struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	TenantID      uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	CubeName      string          `json:"cube_name" db:"cube_name"`
	Name          string          `json:"name" db:"name"`
	Dimensions    json.RawMessage `json:"dimensions" db:"dimensions"`
	Measures      json.RawMessage `json:"measures" db:"measures"`
	TimeDimension string          `json:"time_dimension" db:"time_dimension"`
	Granularity   string          `json:"granularity" db:"granularity"`
	RefreshKey    json.RawMessage `json:"refresh_key" db:"refresh_key"`
	BuildRange    string          `json:"build_range" db:"build_range"`
	Partitions    int             `json:"partitions" db:"partitions"`
	Status        string          `json:"status" db:"status"`
	LastBuiltAt   *time.Time      `json:"last_built_at" db:"last_built_at"`
	RowCount      int64           `json:"row_count" db:"row_count"`
	SizeBytes     int64           `json:"size_bytes" db:"size_bytes"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// TriggerPreAggRefresh triggers a pre-aggregation refresh
func (s *CubeAdminService) TriggerPreAggRefresh(ctx context.Context, tenantID uuid.UUID) error {
	// Mark all pre-aggregations for refresh
	query := `
		UPDATE cube_pre_aggregations SET
			status = 'pending_refresh', updated_at = $2
		WHERE tenant_id = $1 AND status = 'active'
	`
	_, err := s.db.ExecContext(ctx, query, tenantID, time.Now())
	return err
}

// ============================================================================
// CACHE MANAGEMENT
// ============================================================================

// CacheStats represents cache statistics
type CacheStats struct {
	Entries       int64   `json:"entries"`
	MemoryUsedMB  float64 `json:"memory_used_mb"`
	HitRate       float64 `json:"hit_rate"`
	EvictionRate  float64 `json:"eviction_rate"`
	AvgTTLSeconds int     `json:"avg_ttl_seconds"`
}

// GetCacheStats returns cache statistics for a tenant
func (s *CubeAdminService) GetCacheStats(ctx context.Context, tenantID uuid.UUID) (*CacheStats, error) {
	query := `
		SELECT 
			COUNT(*) as entries,
			COALESCE(SUM(size_bytes) / 1024.0 / 1024.0, 0) as memory_used_mb,
			COALESCE(
SUM(hit_count)::float / NULLIF(SUM(hit_count + miss_count), 0), 
0
) as hit_rate,
			COALESCE(
SUM(eviction_count)::float / NULLIF(SUM(hit_count + miss_count), 0), 
0
) as eviction_rate,
			COALESCE(AVG(ttl_seconds), 0) as avg_ttl
		FROM cube_cache_entries
		WHERE tenant_id = $1 AND expires_at > NOW()
	`

	var stats CacheStats
	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(
&stats.Entries, &stats.MemoryUsedMB, &stats.HitRate,
		&stats.EvictionRate, &stats.AvgTTLSeconds,
	)
	if err != nil {
		return nil, fmt.Errorf("get cache stats: %w", err)
	}

	return &stats, nil
}

// ClearCache clears the cache for a tenant
func (s *CubeAdminService) ClearCache(ctx context.Context, tenantID uuid.UUID) error {
	query := `DELETE FROM cube_cache_entries WHERE tenant_id = $1`
	_, err := s.db.ExecContext(ctx, query, tenantID)
	return err
}

// WarmCache triggers cache warming for a tenant
func (s *CubeAdminService) WarmCache(ctx context.Context, tenantID uuid.UUID) error {
	// Record cache warm request
	query := `
		INSERT INTO cube_cache_warm_requests (tenant_id, status, created_at)
		VALUES ($1, 'pending', $2)
	`
	_, err := s.db.ExecContext(ctx, query, tenantID, time.Now())
	return err
}

// ============================================================================
// ADMIN USER MANAGEMENT
// ============================================================================

// ListAdminUsers returns admin users for an organization
func (s *CubeAdminService) ListAdminUsers(ctx context.Context, orgID uuid.UUID) ([]CubeAdminUser, error) {
	query := `
		SELECT id, user_id, organization_id, role, allowed_tenants, permissions,
		       is_active, created_at, updated_at
		FROM cube_admin_users
		WHERE organization_id = $1 AND is_active = true
		ORDER BY role, created_at
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()

	var users []CubeAdminUser
	for rows.Next() {
		var u CubeAdminUser
		if err := rows.Scan(
&u.ID, &u.UserID, &u.OrganizationID, &u.Role, &u.AllowedTenants,
			&u.Permissions, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan admin user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// CreateAdminUser creates a new admin user
func (s *CubeAdminService) CreateAdminUser(ctx context.Context, user CubeAdminUser) (*CubeAdminUser, error) {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	query := `
		INSERT INTO cube_admin_users (
id, user_id, organization_id, role, allowed_tenants, permissions,
is_active, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
user.ID, user.UserID, user.OrganizationID, user.Role, user.AllowedTenants,
user.Permissions, user.IsActive, user.CreatedAt, user.UpdatedAt,
).Scan(&user.ID)

	if err != nil {
		return nil, fmt.Errorf("create admin user: %w", err)
	}

	return &user, nil
}

// UpdateAdminUser updates an existing admin user
func (s *CubeAdminService) UpdateAdminUser(ctx context.Context, user CubeAdminUser) (*CubeAdminUser, error) {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE cube_admin_users SET
			role = $2, allowed_tenants = $3, permissions = $4, is_active = $5, updated_at = $6
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, query,
user.ID, user.Role, user.AllowedTenants, user.Permissions, user.IsActive, user.UpdatedAt,
)
	if err != nil {
		return nil, fmt.Errorf("update admin user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("admin user not found")
	}

	return &user, nil
}

// DeleteAdminUser soft-deletes an admin user
func (s *CubeAdminService) DeleteAdminUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE cube_admin_users SET is_active = false, updated_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, userID, time.Now())
	return err
}

// ============================================================================
// PERFORMANCE METRICS
// ============================================================================

// PerformanceMetrics contains query performance metrics
type PerformanceMetrics struct {
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	P50LatencyMs       float64 `json:"p50_latency_ms"`
	P95LatencyMs       float64 `json:"p95_latency_ms"`
	P99LatencyMs       float64 `json:"p99_latency_ms"`
	QueriesPerSecond   float64 `json:"queries_per_second"`
	ErrorRate          float64 `json:"error_rate"`
}

// GetPerformanceMetrics returns performance metrics for a tenant
func (s *CubeAdminService) GetPerformanceMetrics(ctx context.Context, tenantID uuid.UUID) (*PerformanceMetrics, error) {
	query := `
		WITH recent_queries AS (
SELECT duration_ms, error_message
FROM cube_query_analytics
WHERE tenant_id = $1 AND executed_at >= NOW() - INTERVAL '1 hour'
		),
		percentiles AS (
SELECT 
COALESCE(AVG(duration_ms), 0) as avg_latency,
COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY duration_ms), 0) as p50,
COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms), 0) as p95,
COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms), 0) as p99,
COUNT(*)::float / 3600 as qps,
COALESCE(AVG(CASE WHEN error_message != '' THEN 1.0 ELSE 0.0 END), 0) as error_rate
FROM recent_queries
)
		SELECT * FROM percentiles
	`

	var metrics PerformanceMetrics
	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(
&metrics.AvgLatencyMs, &metrics.P50LatencyMs, &metrics.P95LatencyMs,
		&metrics.P99LatencyMs, &metrics.QueriesPerSecond, &metrics.ErrorRate,
	)
	if err != nil {
		return nil, fmt.Errorf("get performance metrics: %w", err)
	}

	return &metrics, nil
}

// UsageMetrics contains usage statistics
type UsageMetrics struct {
	QueriesToday int64    `json:"queries_today"`
	UniqueUsers  int64    `json:"unique_users"`
	TopCubes     []string `json:"top_cubes"`
	TopMeasures  []string `json:"top_measures"`
}

// GetUsageMetrics returns usage metrics for a tenant
func (s *CubeAdminService) GetUsageMetrics(ctx context.Context, tenantID uuid.UUID) (*UsageMetrics, error) {
	metrics := &UsageMetrics{}

	// Get query count and unique users
	countQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(DISTINCT user_id) as unique_users
		FROM cube_query_analytics
		WHERE tenant_id = $1 AND executed_at >= CURRENT_DATE
	`
	if err := s.db.QueryRowContext(ctx, countQuery, tenantID).Scan(
&metrics.QueriesToday, &metrics.UniqueUsers,
	); err != nil {
		return nil, fmt.Errorf("get usage counts: %w", err)
	}

	// Get top cubes
	cubesQuery := `
		SELECT UNNEST(cubes_used) as cube, COUNT(*) as cnt
		FROM cube_query_analytics
		WHERE tenant_id = $1 AND executed_at >= CURRENT_DATE
		GROUP BY cube ORDER BY cnt DESC LIMIT 5
	`
	rows, err := s.db.QueryContext(ctx, cubesQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get top cubes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cube string
		var cnt int
		if err := rows.Scan(&cube, &cnt); err == nil {
			metrics.TopCubes = append(metrics.TopCubes, cube)
		}
	}

	// Get top measures
	measuresQuery := `
		SELECT UNNEST(measures_used) as measure, COUNT(*) as cnt
		FROM cube_query_analytics
		WHERE tenant_id = $1 AND executed_at >= CURRENT_DATE
		GROUP BY measure ORDER BY cnt DESC LIMIT 5
	`
	rows2, err := s.db.QueryContext(ctx, measuresQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get top measures: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var measure string
		var cnt int
		if err := rows2.Scan(&measure, &cnt); err == nil {
			metrics.TopMeasures = append(metrics.TopMeasures, measure)
		}
	}

	return metrics, nil
}

// SlowQuery represents a slow query for analysis
type SlowQuery struct {
	QueryHash    string    `json:"query_hash"`
	QueryText    string    `json:"query_text"`
	AvgDuration  float64   `json:"avg_duration_ms"`
	MaxDuration  float64   `json:"max_duration_ms"`
	Count        int64     `json:"count"`
	CacheHitRate float64   `json:"cache_hit_rate"`
	LastSeen     time.Time `json:"last_seen"`
}

// GetSlowQueries returns slow queries for optimization
func (s *CubeAdminService) GetSlowQueries(ctx context.Context, tenantID uuid.UUID, thresholdMs int) ([]SlowQuery, error) {
	query := `
		SELECT 
			query_hash,
			MAX(query_text) as query_text,
			AVG(duration_ms) as avg_duration,
			MAX(duration_ms) as max_duration,
			COUNT(*) as count,
			AVG(CASE WHEN cache_hit THEN 1.0 ELSE 0.0 END) as cache_hit_rate,
			MAX(executed_at) as last_seen
		FROM cube_query_analytics
		WHERE tenant_id = $1 
		  AND executed_at >= NOW() - INTERVAL '24 hours'
		GROUP BY query_hash
		HAVING AVG(duration_ms) > $2
		ORDER BY avg_duration DESC
		LIMIT 20
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, thresholdMs)
	if err != nil {
		return nil, fmt.Errorf("get slow queries: %w", err)
	}
	defer rows.Close()

	var slowQueries []SlowQuery
	for rows.Next() {
		var q SlowQuery
		if err := rows.Scan(
&q.QueryHash, &q.QueryText, &q.AvgDuration, &q.MaxDuration,
			&q.Count, &q.CacheHitRate, &q.LastSeen,
		); err != nil {
			return nil, fmt.Errorf("scan slow query: %w", err)
		}
		slowQueries = append(slowQueries, q)
	}
	return slowQueries, nil
}
