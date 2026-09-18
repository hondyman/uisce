// Package boread is the thin read choke point for business_objects /
// business_object_fields / catalog_edge queries used by MCP adapters.
//
// # Platform gold-copy standard (decided at widen)
//
// Shared BO/catalog rows are those owned by goldcopy.ResolveTenantID
// (public.tenants.gold_copy = true). The transitional MCP nil-UUID OR is retired.
// Pages use the same resolver plus is_core (see pagestudio). RLS policies must
// encode this post-widen semantics, not the nil-UUID sentinel.
package boread

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/jmoiron/sqlx"
)

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

type Summary struct {
	ID          string `db:"id" json:"id"`
	Name        string `db:"name" json:"key"`
	DisplayName string `db:"display_name" json:"display_name"`
	Status      string `db:"status" json:"status"`
}

type Contract struct {
	Name        string `db:"name"`
	DisplayName string `db:"display_name"`
	Status      string `db:"status"`
}

type Term struct {
	TermKey     string `db:"term_key"`
	DisplayName string `db:"display_name"`
	Role        string `db:"role"`
}

type FieldSchema struct {
	Name        string `db:"name"`
	DisplayName string `db:"display_name"`
	DataType    string `db:"data_type"`
}

type Edge struct {
	EdgeTypeName string          `db:"edge_type_name"`
	Properties   json.RawMessage `db:"properties"`
}

// ListSummaries lists BOs for the caller plus the gold-copy tenant.
//
// Predicate comparison (gold-copy widen):
//
//	OLD: WHERE tenant_id = $1 OR tenant_id = nil-UUID
//	NEW: WHERE tenant_id = $1 OR tenant_id = $2  ($2 = goldcopy.ResolveTenantID)
//	DELTA: intentional — platform standard is tenants.gold_copy, not nil-UUID
func (s *Service) ListSummaries(ctx context.Context, tenantID uuid.UUID) ([]Summary, error) {
	if s == nil || s.db == nil {
		return []Summary{}, nil
	}
	gold := goldcopy.ResolveTenantID(ctx, s.db)
	var list []Summary
	err := s.db.SelectContext(ctx, &list, `
		SELECT id::text, COALESCE(name, '') AS name, COALESCE(display_name, name, '') AS display_name, COALESCE(status, '') AS status
		FROM public.business_objects
		WHERE tenant_id = $1 OR tenant_id = $2
		ORDER BY display_name
		LIMIT 200
	`, tenantID.String(), gold.String())
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Summary{}
	}
	return list, nil
}

// GetContract loads one BO by id or name for caller or gold tenant.
//
// Predicate comparison (gold-copy widen):
//
//	OLD: (id OR name) AND (tenant OR nil-UUID)
//	NEW: (id OR name) AND (tenant OR gold)
//	DELTA: intentional
func (s *Service) GetContract(ctx context.Context, tenantID uuid.UUID, boID uuid.UUID, boKey string) (*Contract, error) {
	if s == nil || s.db == nil {
		return nil, sql.ErrNoRows
	}
	gold := goldcopy.ResolveTenantID(ctx, s.db)
	var c Contract
	err := s.db.GetContext(ctx, &c, `
		SELECT name, display_name, status
		FROM public.business_objects
		WHERE (id = $1 OR name = $2) AND (tenant_id = $3 OR tenant_id = $4)
		LIMIT 1
	`, boID.String(), boKey, tenantID.String(), gold.String())
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListTerms returns field/term rows for a BO id or key (caller or gold BO).
//
// Predicate comparison (gold-copy widen):
//
//	OLD: … tenant OR nil-UUID in subquery
//	NEW: … tenant OR gold in subquery
//	DELTA: intentional
func (s *Service) ListTerms(ctx context.Context, tenantID uuid.UUID, boID, boKey string) ([]Term, error) {
	if s == nil || s.db == nil {
		return []Term{}, nil
	}
	gold := goldcopy.ResolveTenantID(ctx, s.db)
	var terms []Term
	err := s.db.SelectContext(ctx, &terms, `
		SELECT COALESCE(term_key, name, '') AS term_key,
		       COALESCE(display_name, term_key, name, '') AS display_name,
		       COALESCE(role, '') AS role
		FROM public.business_object_fields
		WHERE business_object_id::text = $1 OR business_object_id IN (
			SELECT id FROM public.business_objects WHERE name = $2 AND (tenant_id = $3 OR tenant_id = $4)
		)
		LIMIT 200
	`, boID, boKey, tenantID.String(), gold.String())
	if err != nil {
		return nil, err
	}
	if terms == nil {
		terms = []Term{}
	}
	return terms, nil
}

// ResolveEdge looks up one catalog_edge between two nodes (caller or gold).
//
// Predicate comparison (gold-copy widen):
//
//	OLD: tenant_id = $3 OR nil-UUID
//	NEW: tenant_id = $3 OR gold
//	DELTA: intentional
func (s *Service) ResolveEdge(ctx context.Context, tenantID, sourceID, targetID uuid.UUID) (*Edge, error) {
	if s == nil || s.db == nil {
		return nil, sql.ErrNoRows
	}
	gold := goldcopy.ResolveTenantID(ctx, s.db)
	var e Edge
	err := s.db.GetContext(ctx, &e, `
		SELECT edge_type_name, properties
		FROM public.catalog_edge
		WHERE source_id = $1 AND target_id = $2
		  AND (tenant_id = $3::text OR tenant_id = $4)
		LIMIT 1
	`, sourceID.String(), targetID.String(), tenantID.String(), gold.String())
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ResolveBOIDByKey resolves a BO id from tenant + name (no nil-UUID OR).
//
// Predicate comparison (SL extract from MCP get_bo_schema key lookup):
//
//	OLD: WHERE tenant_id = $1 AND name = $2
//	NEW: identical
//	DELTA: none
func (s *Service) ResolveBOIDByKey(ctx context.Context, tenantID uuid.UUID, boKey string) (string, error) {
	if s == nil || s.db == nil {
		return "", sql.ErrNoRows
	}
	var id string
	err := s.db.GetContext(ctx, &id, `
		SELECT id::text FROM public.business_objects
		WHERE tenant_id = $1 AND name = $2
		LIMIT 1
	`, tenantID, boKey)
	return id, err
}

// ListFieldSchema returns form fields for a BO under the calling tenant.
//
// Predicate comparison (SL extract from MCP get_bo_schema):
//
//	OLD: WHERE tenant_id = $1 AND bo_id::text = $2
//	NEW: identical
//	DELTA: none
func (s *Service) ListFieldSchema(ctx context.Context, tenantID uuid.UUID, boID string) ([]FieldSchema, error) {
	if s == nil || s.db == nil {
		return []FieldSchema{}, nil
	}
	var fields []FieldSchema
	err := s.db.SelectContext(ctx, &fields, `
		SELECT COALESCE(technical_name, field_name) AS name,
		       COALESCE(NULLIF(display_name, ''), field_name) AS display_name,
		       COALESCE(data_type, 'text') AS data_type
		FROM public.business_object_fields
		WHERE tenant_id = $1 AND bo_id::text = $2
		ORDER BY display_order NULLS LAST, field_name
	`, tenantID, boID)
	if err != nil {
		return nil, err
	}
	if fields == nil {
		fields = []FieldSchema{}
	}
	return fields, nil
}

// SearchMatch is one hit from Search.
type SearchMatch struct {
	ID          string `db:"id" json:"id"`
	Name        string `db:"name" json:"key"`
	DisplayName string `db:"display_name" json:"display_name"`
}

// Search is the MCP catalog search contract: tenant-or-gold, ILIKE on
// name/display_name, LIMIT 50. Not discovery/search (discovery_candidates).
//
// Predicate comparison (gold-copy widen):
//
//	OLD: tenant OR nil-UUID
//	NEW: tenant OR goldcopy.ResolveTenantID
//	DELTA: intentional
func (s *Service) Search(ctx context.Context, tenantID uuid.UUID, query string) ([]SearchMatch, error) {
	if s == nil || s.db == nil {
		return []SearchMatch{}, nil
	}
	gold := goldcopy.ResolveTenantID(ctx, s.db)
	var matches []SearchMatch
	err := s.db.SelectContext(ctx, &matches, `
		SELECT id::text, COALESCE(name,'') AS name, COALESCE(display_name, name, '') AS display_name
		FROM public.business_objects
		WHERE (tenant_id = $1 OR tenant_id = $2)
		  AND (name ILIKE '%' || $3 || '%' OR display_name ILIKE '%' || $3 || '%')
		ORDER BY display_name
		LIMIT 50
	`, tenantID.String(), gold.String(), query)
	if err != nil {
		return nil, err
	}
	if matches == nil {
		matches = []SearchMatch{}
	}
	return matches, nil
}
