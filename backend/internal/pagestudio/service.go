// Package pagestudio owns page_definitions reads used by HTTP Page Studio and MCP.
// MCP tools must call this service rather than assembling SQL themselves.
package pagestudio

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	dbpkg "github.com/hondyman/uisce/backend/internal/db"
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

// withTenant opens a tx, SET LOCALs tenant+gold GUCs (FORCE RLS), runs fn, commits.
func (s *Service) withTenant(ctx context.Context, tenantID uuid.UUID, fn func(tx *sqlx.Tx, gold uuid.UUID) error) error {
	if s == nil || s.db == nil {
		return fn(nil, uuid.Nil)
	}
	gold := goldcopy.ResolveTenantID(ctx, s.db)
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := dbpkg.ApplyTenantGUCs(ctx, tx.Tx, tenantID.String(), gold.String()); err != nil {
		return fmt.Errorf("tenant GUC: %w", err)
	}
	if err := fn(tx, gold); err != nil {
		return err
	}
	return tx.Commit()
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
	var pages []PageSummary
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx, gold uuid.UUID) error {
		return tx.SelectContext(ctx, &pages, `
			SELECT id::text, name, slug, COALESCE(status, '') AS status
			FROM public.page_definitions
			WHERE tenant_id = $1
			   OR (is_core = true AND tenant_id = $2)
			ORDER BY updated_at DESC
			LIMIT 100
		`, tenantID, gold)
	})
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
	var out *PageDetail
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx, gold uuid.UUID) error {
		page, err := getOneTx(ctx, tx, `
			SELECT id::text, name, slug, COALESCE(status, '') AS status,
			       layout, components, data_sources,
			       COALESCE(presentation_events, '[]'::jsonb) AS presentation_events,
			       COALESCE(filter_bar, '{}'::jsonb) AS filter_bar
			FROM public.page_definitions
			WHERE tenant_id = $1 AND (id::text = $2 OR slug = $3)
			LIMIT 1
		`, tenantID, pageID, slug)
		if err == nil {
			out = page
			return nil
		}
		if err != sql.ErrNoRows {
			return err
		}
		page, err = getOneTx(ctx, tx, `
			SELECT id::text, name, slug, COALESCE(status, '') AS status,
			       layout, components, data_sources,
			       COALESCE(presentation_events, '[]'::jsonb) AS presentation_events,
			       COALESCE(filter_bar, '{}'::jsonb) AS filter_bar
			FROM public.page_definitions
			WHERE is_core = true AND tenant_id = $1 AND (id::text = $2 OR slug = $3)
			LIMIT 1
		`, gold, pageID, slug)
		if err != nil {
			return err
		}
		out = page
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func getOneTx(ctx context.Context, tx *sqlx.Tx, query string, args ...interface{}) (*PageDetail, error) {
	var page PageDetail
	if err := tx.GetContext(ctx, &page, query, args...); err != nil {
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
	var pages []PageSummary
	err := s.withTenant(ctx, tenantID, func(tx *sqlx.Tx, gold uuid.UUID) error {
		return tx.SelectContext(ctx, &pages, `
			SELECT id::text, name, slug, COALESCE(status, '') AS status
			FROM public.page_definitions
			WHERE (tenant_id = $1 OR (is_core = true AND tenant_id = $4))
			  AND slug IN ($2, $3)
		`, tenantID, slugA, slugB, gold)
	})
	if err != nil {
		return nil, err
	}
	if pages == nil {
		pages = []PageSummary{}
	}
	return pages, nil
}
