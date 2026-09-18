// Package pagestudio owns page_definitions reads used by HTTP Page Studio and MCP.
// MCP tools must call this service rather than assembling SQL themselves.
package pagestudio

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/jmoiron/sqlx"
)

// Service is the choke point for page_definitions access.
type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// PageSummary is the list_pages / journey wire shape.
type PageSummary struct {
	ID     string `db:"id" json:"id"`
	Name   string `db:"name" json:"name"`
	Slug   string `db:"slug" json:"slug"`
	Status string `db:"status" json:"status"`
}

// PageDetail is the get_page wire shape (layout payload included).
type PageDetail struct {
	ID                 string          `db:"id" json:"id"`
	Name               string          `db:"name" json:"name"`
	Slug               string          `db:"slug" json:"slug"`
	Status             string          `db:"status" json:"status"`
	Layout             json.RawMessage `db:"layout" json:"layout"`
	Components         json.RawMessage `db:"components" json:"components"`
	DataSources        json.RawMessage `db:"data_sources" json:"dataSources"`
	PresentationEvents json.RawMessage `db:"presentation_events" json:"presentationEvents"`
	FilterBar          json.RawMessage `db:"filter_bar" json:"filterBar"`
}

// ListSummaries returns tenant pages plus gold-copy core pages.
//
// Predicate comparison (gold-copy widen):
//
//	OLD (MCP SL extract): WHERE tenant_id = $1
//	NEW:                  WHERE tenant_id = $1 OR (is_core = true AND tenant_id = $2)
//	                      ($2 = goldcopy.ResolveTenantID)
//	DELTA:                intentional — adopt PageStudioHandler.list gold semantics
func (s *Service) ListSummaries(ctx context.Context, tenantID uuid.UUID) ([]PageSummary, error) {
	if s == nil || s.db == nil {
		return []PageSummary{}, nil
	}
	gold := goldcopy.ResolveTenantID(ctx, s.db)
	var pages []PageSummary
	err := s.db.SelectContext(ctx, &pages, `
		SELECT id::text, name, slug, COALESCE(status, '') AS status
		FROM public.page_definitions
		WHERE tenant_id = $1
		   OR (is_core = true AND tenant_id = $2)
		ORDER BY updated_at DESC
		LIMIT 100
	`, tenantID, gold)
	if err != nil {
		return nil, err
	}
	if pages == nil {
		pages = []PageSummary{}
	}
	return pages, nil
}

// GetByIDOrSlug returns a tenant page, falling back to gold-copy core.
//
// Predicate comparison (gold-copy widen):
//
//	OLD: WHERE tenant_id = $1 AND (id OR slug)
//	NEW: try tenant match; on miss, id/slug AND is_core AND tenant_id = gold
//	DELTA: intentional — adopt PageStudioHandler.get / getBySlug fallback
func (s *Service) GetByIDOrSlug(ctx context.Context, tenantID uuid.UUID, pageID, slug string) (*PageDetail, error) {
	if s == nil || s.db == nil {
		return nil, sql.ErrNoRows
	}
	page, err := s.getOne(ctx, `
		SELECT id::text, name, slug, COALESCE(status, '') AS status,
		       layout, components, data_sources,
		       COALESCE(presentation_events, '[]'::jsonb) AS presentation_events,
		       COALESCE(filter_bar, '{}'::jsonb) AS filter_bar
		FROM public.page_definitions
		WHERE tenant_id = $1 AND (id::text = $2 OR slug = $3)
		LIMIT 1
	`, tenantID, pageID, slug)
	if err == nil {
		return page, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	gold := goldcopy.ResolveTenantID(ctx, s.db)
	return s.getOne(ctx, `
		SELECT id::text, name, slug, COALESCE(status, '') AS status,
		       layout, components, data_sources,
		       COALESCE(presentation_events, '[]'::jsonb) AS presentation_events,
		       COALESCE(filter_bar, '{}'::jsonb) AS filter_bar
		FROM public.page_definitions
		WHERE is_core = true AND tenant_id = $1 AND (id::text = $2 OR slug = $3)
		LIMIT 1
	`, gold, pageID, slug)
}

func (s *Service) getOne(ctx context.Context, query string, args ...interface{}) (*PageDetail, error) {
	var page PageDetail
	if err := s.db.GetContext(ctx, &page, query, args...); err != nil {
		return nil, err
	}
	return &page, nil
}

// ListBySlugs returns tenant or gold-core pages matching two slugs.
//
// Predicate comparison (gold-copy widen):
//
//	OLD: WHERE tenant_id = $1 AND slug IN ($2, $3)
//	NEW: WHERE (tenant_id = $1 OR (is_core AND tenant_id = $4)) AND slug IN ($2, $3)
//	DELTA: intentional — journey can resolve gold core blotter/ticket pages
func (s *Service) ListBySlugs(ctx context.Context, tenantID uuid.UUID, slugA, slugB string) ([]PageSummary, error) {
	if s == nil || s.db == nil {
		return []PageSummary{}, nil
	}
	gold := goldcopy.ResolveTenantID(ctx, s.db)
	var pages []PageSummary
	err := s.db.SelectContext(ctx, &pages, `
		SELECT id::text, name, slug, COALESCE(status, '') AS status
		FROM public.page_definitions
		WHERE (tenant_id = $1 OR (is_core = true AND tenant_id = $4))
		  AND slug IN ($2, $3)
	`, tenantID, slugA, slugB, gold)
	if err != nil {
		return nil, err
	}
	if pages == nil {
		pages = []PageSummary{}
	}
	return pages, nil
}
