// Package tiles implements the FIX-specific data-pipeline tiles defined
// in HANDOFF_FIX_OVER_PIPELINE.md §9.
//
// Each tile is a pure function with the signature:
//
//	Transform(ctx context.Context, records []Record) ([]Record, []string, error)
//
// matching the pattern of internal/datapipeline/transforms.go so they
// can be wired into the data-pipeline engine's `executeTransform` switch
// when the data-pipeline worktree (nifty-greider-015b86) lands.
//
// Tiles are deliberately decoupled from the data-pipeline package so
// they can be unit-tested in isolation and reused in any pipeline
// engine, including the legacy pkg/workflows system if a migration is
// ever needed.
//
// GSIFI compliance: every tile that reads from fix_tenant_config or
// fix_tenant_tag_mapping MUST apply the gold-copy OR-clause in SQL
// (the helper uisce_get_current_tenant() in the RLS policy already
// enforces single-tenant scoping for normal sessions; the application
// OR-clause is for Gold Copy inheritance).
//
// TODO(worktree-merge): when the data-pipeline worktree lands, these
// tiles must be relocated into backend/internal/datapipeline/transforms.go
// (and friends) — registered in the engine's executeTransform /
// executeSource / executeLoader switches so they execute inside the
// real pipeline DAG, not as a standalone Go package. Without that move,
// the FIX flow "bypasses the data-pipeline layer entirely — which is
// functionally the old way with better structure" (user feedback,
// session-2026-09-13). The standalone package here is interim only;
// the doc §13 build step 11 is the trigger for the move.
package tiles

import (
	"context"
)

// Record is the in-flight PipelineRecord shape that flows between tiles.
// Mirrors the JSON shape used by backend/internal/datapipeline/model.go.
// Kept here as a flat alias so this package doesn't depend on that
// worktree-only package.
type Record = map[string]any

// TenantContext is passed through ctx.Value so tiles can read the
// tenant boundary without each tile having to thread it through the
// records. This is the GSIFI anchor — every SQL query uses it.
type TenantContext struct {
	TenantID string
	BrokerID string
}

// WithTenant returns a derived context carrying t.
func WithTenant(ctx context.Context, t TenantContext) context.Context {
	return context.WithValue(ctx, tenantKey{}, t)
}

// TenantFromContext returns the TenantContext stored in ctx, or the
// zero value if none is set.
func TenantFromContext(ctx context.Context) TenantContext {
	if t, ok := ctx.Value(tenantKey{}).(TenantContext); ok {
		return t
	}
	return TenantContext{}
}

type tenantKey struct{}

// TileFunc is the common signature for every FIX tile. It mirrors the
// `(records, errors, err)` return shape of internal/datapipeline/transforms.go.
type TileFunc func(ctx context.Context, records []Record) ([]Record, []string, error)

// Identity returns its input unchanged. Useful as a default for tiles
// in a config that haven't been wired up yet.
func Identity(ctx context.Context, records []Record) ([]Record, []string, error) {
	return records, nil, nil
}
