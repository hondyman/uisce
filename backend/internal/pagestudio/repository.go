package pagestudio

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles persistence for Page Studio entities
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// SavePage inserts or updates a core page
func (r *Repository) SavePage(ctx context.Context, p *CorePage) error {
	query := `
		INSERT INTO semantic.pages (id, env, tenant_id, name, slug, description, layout, components, data_bindings, visibility, version, created_by, updated_at)
		VALUES (:id, :env, :tenant_id, :name, :slug, :description, :layout, :components, :data_bindings, :visibility, :version, :created_by, NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			slug = EXCLUDED.slug,
			description = EXCLUDED.description,
			layout = EXCLUDED.layout,
			components = EXCLUDED.components,
			data_bindings = EXCLUDED.data_bindings,
			visibility = EXCLUDED.visibility,
			version = EXCLUDED.version,
			updated_at = NOW()
	`
	_, err := r.db.NamedExecContext(ctx, query, p)
	return err
}

// GetPage retrieves a core page by ID
func (r *Repository) GetPage(ctx context.Context, id uuid.UUID) (*CorePage, error) {
	var p CorePage
	err := r.db.GetContext(ctx, &p, "SELECT * FROM semantic.pages WHERE id=$1", id)
	return &p, err
}

// GetPageBySlug retrieves a core page by slug
func (r *Repository) GetPageBySlug(ctx context.Context, slug string, env string) (*CorePage, error) {
	var p CorePage
	err := r.db.GetContext(ctx, &p, "SELECT * FROM semantic.pages WHERE slug=$1 AND env=$2", slug, env)
	return &p, err
}

// ListPages lists all core pages for an environment
func (r *Repository) ListPages(ctx context.Context, env string) ([]CorePage, error) {
	var pages []CorePage
	err := r.db.SelectContext(ctx, &pages, "SELECT * FROM semantic.pages WHERE env=$1 AND tenant_id IS NULL", env)
	return pages, err
}

// SaveOverlay inserts or updates a page overlay
func (r *Repository) SaveOverlay(ctx context.Context, o *PageOverlay) error {
	query := `
		INSERT INTO semantic.page_overlays (id, parent_id, env, tenant_id, overrides, version, created_by, updated_at)
		VALUES (:id, :parent_id, :env, :tenant_id, :overrides, :version, :created_by, NOW())
		ON CONFLICT (parent_id, tenant_id, env) DO UPDATE SET
			overrides = EXCLUDED.overrides,
			version = EXCLUDED.version,
			updated_at = NOW()
	`
	_, err := r.db.NamedExecContext(ctx, query, o)
	return err
}

// GetOverlay retrieves an overlay for a core page and tenant
func (r *Repository) GetOverlay(ctx context.Context, parentID uuid.UUID, tenantID string, env string) (*PageOverlay, error) {
	var o PageOverlay
	err := r.db.GetContext(ctx, &o, "SELECT * FROM semantic.page_overlays WHERE parent_id=$1 AND tenant_id=$2 AND env=$3", parentID, tenantID, env)
	return &o, err
}

// SaveUpgradeImpact inserts or updates an upgrade impact analysis
func (r *Repository) SaveUpgradeImpact(ctx context.Context, i *UpgradeImpact) error {
	query := `
		INSERT INTO semantic.upgrade_impacts (
			id, core_page_id, core_old_version, core_new_version, tenant_id, overlay_page_id,
			summary, conflicts, inherited_changes, new_core_components, removed_core_components,
			status, updated_at
		) VALUES (
			:id, :core_page_id, :core_old_version, :core_new_version, :tenant_id, :overlay_page_id,
			:summary, :conflicts, :inherited_changes, :new_core_components, :removed_core_components,
			:status, NOW()
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = NOW()
	`
	_, err := r.db.NamedExecContext(ctx, query, i)
	return err
}

// GetUpgradeImpacts retrieves pending impacts for a tenant
func (r *Repository) GetUpgradeImpacts(ctx context.Context, tenantID string) ([]UpgradeImpact, error) {
	var impacts []UpgradeImpact
	err := r.db.SelectContext(ctx, &impacts, "SELECT * FROM semantic.upgrade_impacts WHERE tenant_id=$1 AND status='pending'", tenantID)
	return impacts, err
}

// ListOverlaysForPage retrieves all overlays across all tenants for a specific core page
func (r *Repository) ListOverlaysForPage(ctx context.Context, parentID uuid.UUID) ([]PageOverlay, error) {
	var overlays []PageOverlay
	err := r.db.SelectContext(ctx, &overlays, "SELECT * FROM semantic.page_overlays WHERE parent_id=$1", parentID)
	return overlays, err
}

// GetUpgradeImpact retrieves a specific impact record by ID
func (r *Repository) GetUpgradeImpact(ctx context.Context, id uuid.UUID) (*UpgradeImpact, error) {
	var i UpgradeImpact
	err := r.db.GetContext(ctx, &i, "SELECT * FROM semantic.upgrade_impacts WHERE id=$1", id)
	return &i, err
}
