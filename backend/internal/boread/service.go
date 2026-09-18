// Package boread is the thin read choke point for business_objects /
// business_object_fields / catalog_edge queries used by MCP adapters.
//
// # MCP vs HTTP gold-copy divergence (named decision, open until widen)
//
// MCP reads in this package use the nil-UUID OR
// (tenant_id = caller OR tenant_id = '00000000-…').
// HTTP BusinessObjectService paths use different gold semantics:
//   - GetBusinessObject: requireAccess + tenants.gold_copy lookup
//   - ListBusinessObjects: tenant-only (no gold OR)
//   - ListBusinessObjectsLegacy: EXISTS(gold_copy) + set_config('app.tenant_id')
//     — LIVE on api.Server listBusinessObjects (api.go) and a cousin set_config
//     on getBusinessObjectByID. RLS design must account for pre-existing
//     session-config tenancy, not assume it introduces the only one.
//
// Until the gold-copy-widen commit, preserving MCP predicates means the two
// surfaces can see different rows on the same tables. Widen must decide
// deliberately (adopt HTTP richer semantics vs keep MCP nil-UUID OR) for
// both pages and BOs — not accumulate divergence by deference.
//
// Why not metadata.BusinessObjectService yet: the three bullets above are
// evidence-backed rejections (commit 1/5), not preferences.
package boread

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GlobalTenantNilUUID is the sentinel MCP queries use for shared/gold rows.
// Distinct from tenants.gold_copy=true lookup used by BusinessObjectService.
const GlobalTenantNilUUID = "00000000-0000-0000-0000-000000000000"

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

// ListSummaries lists BOs for the tenant including the nil-UUID global sentinel.
//
// Predicate comparison (SL extract from MCP list_business_objects):
//
//	OLD: WHERE tenant_id = $1 OR tenant_id = '00000000-…'
//	NEW: identical
//	DELTA: none — NOT ListBusinessObjects (tenant-only) and NOT Legacy (EXISTS gold_copy).
func (s *Service) ListSummaries(ctx context.Context, tenantID uuid.UUID) ([]Summary, error) {
	if s == nil || s.db == nil {
		return []Summary{}, nil
	}
	var list []Summary
	err := s.db.SelectContext(ctx, &list, `
		SELECT id::text, COALESCE(name, '') AS name, COALESCE(display_name, name, '') AS display_name, COALESCE(status, '') AS status
		FROM public.business_objects
		WHERE tenant_id = $1 OR tenant_id = '`+GlobalTenantNilUUID+`'
		ORDER BY display_name
		LIMIT 200
	`, tenantID.String())
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Summary{}
	}
	return list, nil
}

// GetContract loads one BO contract row by id or name with nil-UUID global OR.
//
// Predicate comparison (SL extract from MCP get_business_object_contract):
//
//	OLD: WHERE (id = $1 OR name = $2) AND (tenant_id = $3 OR tenant_id = nil-UUID)
//	NEW: identical
//	DELTA: none — NOT GetBusinessObject (requireAccess + gold_copy tenant fallback).
func (s *Service) GetContract(ctx context.Context, tenantID uuid.UUID, boID uuid.UUID, boKey string) (*Contract, error) {
	if s == nil || s.db == nil {
		return nil, sql.ErrNoRows
	}
	var c Contract
	err := s.db.GetContext(ctx, &c, `
		SELECT name, display_name, status
		FROM public.business_objects
		WHERE (id = $1 OR name = $2) AND (tenant_id = $3 OR tenant_id = '`+GlobalTenantNilUUID+`')
		LIMIT 1
	`, boID.String(), boKey, tenantID.String())
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListTerms returns field/term rows for a BO id or key.
//
// Predicate comparison (SL extract from MCP get_bo_terms):
//
//	OLD: business_object_id = $1 OR id IN (SELECT … name = $2 AND tenant OR nil-UUID)
//	NEW: identical
//	DELTA: none
func (s *Service) ListTerms(ctx context.Context, tenantID uuid.UUID, boID, boKey string) ([]Term, error) {
	if s == nil || s.db == nil {
		return []Term{}, nil
	}
	var terms []Term
	err := s.db.SelectContext(ctx, &terms, `
		SELECT COALESCE(term_key, name, '') AS term_key,
		       COALESCE(display_name, term_key, name, '') AS display_name,
		       COALESCE(role, '') AS role
		FROM public.business_object_fields
		WHERE business_object_id::text = $1 OR business_object_id IN (
			SELECT id FROM public.business_objects WHERE name = $2 AND (tenant_id = $3 OR tenant_id = '`+GlobalTenantNilUUID+`')
		)
		LIMIT 200
	`, boID, boKey, tenantID.String())
	if err != nil {
		return nil, err
	}
	if terms == nil {
		terms = []Term{}
	}
	return terms, nil
}

// ResolveEdge looks up one catalog_edge between two nodes.
//
// Predicate comparison (SL extract from MCP resolve_relationship_path):
//
//	OLD: source_id/target_id + (tenant_id = $3 OR nil-UUID)
//	NEW: identical
//	DELTA: none — NOT GetBusinessObjectRelationships (FK graph, different shape).
func (s *Service) ResolveEdge(ctx context.Context, tenantID, sourceID, targetID uuid.UUID) (*Edge, error) {
	if s == nil || s.db == nil {
		return nil, sql.ErrNoRows
	}
	var e Edge
	err := s.db.GetContext(ctx, &e, `
		SELECT edge_type_name, properties
		FROM public.catalog_edge
		WHERE source_id = $1 AND target_id = $2
		  AND (tenant_id = $3::text OR tenant_id = '`+GlobalTenantNilUUID+`')
		LIMIT 1
	`, sourceID.String(), targetID.String(), tenantID.String())
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

// Search finds business_objects by name/display_name ILIKE for the tenant.
//
// This is a new service contract (extraction), not a delegation — there is no
// existing BusinessObjectService search with this shape. discovery/search is
// rejected: it queries discovery_candidates, not business_objects.
//
// Predicate comparison (SL extract from MCP search_catalog):
//
//	OLD (tool SQL): (tenant_id = $1 OR nil-UUID) AND (name ILIKE … OR display_name ILIKE …)
//	NEW:            identical bind shape and WHERE (new boread.Search)
//	DELTA:          none (predicate copied; contract newly owned by boread)
func (s *Service) Search(ctx context.Context, tenantID uuid.UUID, query string) ([]SearchMatch, error) {
	if s == nil || s.db == nil {
		return []SearchMatch{}, nil
	}
	var matches []SearchMatch
	err := s.db.SelectContext(ctx, &matches, `
		SELECT id::text, COALESCE(name,'') AS name, COALESCE(display_name, name, '') AS display_name
		FROM public.business_objects
		WHERE (tenant_id = $1 OR tenant_id = '`+GlobalTenantNilUUID+`')
		  AND (name ILIKE '%' || $2 || '%' OR display_name ILIKE '%' || $2 || '%')
		ORDER BY display_name
		LIMIT 50
	`, tenantID.String(), query)
	if err != nil {
		return nil, err
	}
	if matches == nil {
		matches = []SearchMatch{}
	}
	return matches, nil
}
