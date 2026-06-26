package apistudio

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles persistence for API Studio entities
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// SaveEndpoint saves an API endpoint
func (r *Repository) SaveEndpoint(ctx context.Context, ep *APIEndpoint) error {
	query := `
		INSERT INTO semantic.api_endpoints (
			id, env, tenant_id, name, path, method, type, bo_name, fields, filters, pagination, auth_policy, version,
			status, semantic_version, previous_version_id, owner_team, deprecated_at, retired_at, request_schema_id, response_schema_id,
			created_at, created_by
		) VALUES (
			:id, :env, :tenant_id, :name, :path, :method, :type, :bo_name, :fields, :filters, :pagination, :auth_policy, :version,
			:status, :semantic_version, :previous_version_id, :owner_team, :deprecated_at, :retired_at, :request_schema_id, :response_schema_id,
			:created_at, :created_by
		) ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			path = EXCLUDED.path,
			method = EXCLUDED.method,
			type = EXCLUDED.type,
			bo_name = EXCLUDED.bo_name,
			fields = EXCLUDED.fields,
			filters = EXCLUDED.filters,
			pagination = EXCLUDED.pagination,
			auth_policy = EXCLUDED.auth_policy,
			version = EXCLUDED.version,
			status = EXCLUDED.status,
			semantic_version = EXCLUDED.semantic_version,
			previous_version_id = EXCLUDED.previous_version_id,
			owner_team = EXCLUDED.owner_team,
			deprecated_at = EXCLUDED.deprecated_at,
			retired_at = EXCLUDED.retired_at,
			request_schema_id = EXCLUDED.request_schema_id,
			response_schema_id = EXCLUDED.response_schema_id
	`
	_, err := r.db.NamedExecContext(ctx, query, ep)
	return err
}

// GetEndpoint retrieves an endpoint by ID
func (r *Repository) GetEndpoint(ctx context.Context, id uuid.UUID) (*APIEndpoint, error) {
	var ep APIEndpoint
	err := r.db.GetContext(ctx, &ep, "SELECT * FROM semantic.api_endpoints WHERE id=$1", id)
	return &ep, err
}

// FindByPath matches an incoming request to an endpoint
func (r *Repository) FindByPath(ctx context.Context, method, path, env, tenantID string) (*APIEndpoint, error) {
	var ep APIEndpoint
	// In production, we'd use a more sophisticated path matcher (e.g., Radix tree or regex)
	// For now, exact match or simple prefix
	query := `
		SELECT * FROM semantic.api_endpoints 
		WHERE method=$1 AND path=$2 AND env=$3 AND tenant_id=$4
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &ep, query, method, path, env, tenantID)
	return &ep, err
}

// ListEndpoints returns all endpoints for a tenant
func (r *Repository) ListEndpoints(ctx context.Context, env, tenantID string) ([]APIEndpoint, error) {
	var eps []APIEndpoint
	err := r.db.SelectContext(ctx, &eps, "SELECT * FROM semantic.api_endpoints WHERE env=$1 AND tenant_id=$2", env, tenantID)
	return eps, err
}

// SaveCatalog saves an API catalog
func (r *Repository) SaveCatalog(ctx context.Context, cat *APICatalog) error {
	query := `
		INSERT INTO semantic.api_catalogs (id, env, tenant_id, name, description, created_at, created_by)
		VALUES (:id, :env, :tenant_id, :name, :description, :created_at, :created_by)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description
	`
	_, err := r.db.NamedExecContext(ctx, query, cat)
	return err
}

// GetCatalog retrieves a catalog by ID
func (r *Repository) GetCatalog(ctx context.Context, id uuid.UUID) (*APICatalog, error) {
	var cat APICatalog
	err := r.db.GetContext(ctx, &cat, "SELECT * FROM semantic.api_catalogs WHERE id=$1", id)
	return &cat, err
}

// GetCatalogEntry retrieves an entry and its configuration
func (r *Repository) GetCatalogEntry(ctx context.Context, catalogID, endpointID uuid.UUID) (*APICatalogEntry, error) {
	var entry APICatalogEntry
	err := r.db.GetContext(ctx, &entry, "SELECT * FROM semantic.api_catalog_entries WHERE catalog_id=$1 AND endpoint_id=$2", catalogID, endpointID)
	return &entry, err
}

// ListCatalogEntries fetches endpoints for a catalog
func (r *Repository) ListCatalogEntries(ctx context.Context, catalogID uuid.UUID) ([]APICatalogEntry, error) {
	var entries []APICatalogEntry
	err := r.db.SelectContext(ctx, &entries, "SELECT * FROM semantic.api_catalog_entries WHERE catalog_id=$1", catalogID)
	return entries, err
}

// APITelemetry represents a usage record
type APITelemetry struct {
	ID           uuid.UUID  `id`
	APIID        uuid.UUID  `api_id`
	Env          string     `env`
	TenantID     *uuid.UUID `tenant_id`
	ClientType   string     `client_type`
	StatusCode   int        `status_code`
	LatencyMs    int        `latency_ms`
	ErrorMessage *string    `error_message`
	RequestedAt  time.Time  `requested_at`
}

// LogTelemetry persists a usage record
func (r *Repository) LogTelemetry(ctx context.Context, t *APITelemetry) error {
	query := `
		INSERT INTO api_telemetry (api_id, env, tenant_id, client_type, status_code, latency_ms, error_message, requested_at)
		VALUES (:api_id, :env, :tenant_id, :client_type, :status_code, :latency_ms, :error_message, :requested_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, t)
	return err
}
