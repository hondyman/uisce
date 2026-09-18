// Package pagestudio owns page_definitions reads used by HTTP Page Studio and MCP.
// MCP tools must call this service rather than assembling SQL themselves.
package pagestudio

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
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

// ListSummaries returns tenant-owned pages only.
//
// Predicate comparison (SL extract from MCP list_pages):
//
//	OLD (MCP): WHERE tenant_id = $1
//	NEW:       WHERE tenant_id = $1
//	DELTA:     none — gold-copy OR from PageStudioHandler.list deliberately deferred
//	           to a follow-up that widens with IDOR coverage (gold visible, tenant-B not).
func (s *Service) ListSummaries(ctx context.Context, tenantID uuid.UUID) ([]PageSummary, error) {
	if s == nil || s.db == nil {
		return []PageSummary{}, nil
	}
	var pages []PageSummary
	err := s.db.SelectContext(ctx, &pages, `
		SELECT id::text, name, slug, COALESCE(status, '') AS status
		FROM public.page_definitions
		WHERE tenant_id = $1
		ORDER BY updated_at DESC
		LIMIT 100
	`, tenantID)
	if err != nil {
		return nil, err
	}
	if pages == nil {
		pages = []PageSummary{}
	}
	return pages, nil
}

// GetByIDOrSlug returns one tenant-owned page.
//
// Predicate comparison (SL extract from MCP get_page):
//
//	OLD (MCP): WHERE tenant_id = $1 AND (id::text = $2 OR slug = $3)
//	NEW:       identical
//	DELTA:     none — handler gold-copy fallback (is_core + gold tenant) deferred.
func (s *Service) GetByIDOrSlug(ctx context.Context, tenantID uuid.UUID, pageID, slug string) (*PageDetail, error) {
	if s == nil || s.db == nil {
		return nil, sql.ErrNoRows
	}
	var page PageDetail
	err := s.db.GetContext(ctx, &page, `
		SELECT id::text, name, slug, COALESCE(status, '') AS status,
		       layout, components, data_sources,
		       COALESCE(presentation_events, '[]'::jsonb) AS presentation_events,
		       COALESCE(filter_bar, '{}'::jsonb) AS filter_bar
		FROM public.page_definitions
		WHERE tenant_id = $1 AND (id::text = $2 OR slug = $3)
		LIMIT 1
	`, tenantID, pageID, slug)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// ListBySlugs returns tenant-owned pages matching exactly two slugs.
//
// Predicate comparison (SL extract from MCP describe_oms_journey):
//
//	OLD (MCP): WHERE tenant_id = $1 AND slug IN ($2, $3)
//	NEW:       identical bind shape
//	DELTA:     none
func (s *Service) ListBySlugs(ctx context.Context, tenantID uuid.UUID, slugA, slugB string) ([]PageSummary, error) {
	if s == nil || s.db == nil {
		return []PageSummary{}, nil
	}
	var pages []PageSummary
	err := s.db.SelectContext(ctx, &pages, `
		SELECT id::text, name, slug, COALESCE(status, '') AS status
		FROM public.page_definitions
		WHERE tenant_id = $1 AND slug IN ($2, $3)
	`, tenantID, slugA, slugB)
	if err != nil {
		return nil, err
	}
	if pages == nil {
		pages = []PageSummary{}
	}
	return pages, nil
}
