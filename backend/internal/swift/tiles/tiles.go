package tiles

import (
	"context"
)

// Record is the in-flight settlement record between tiles.
type Record = map[string]any

// TenantContext carries GSIFI boundary through ctx.Value.
type TenantContext struct {
	TenantID    string
	CustodianID string
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

// TileFunc signature
type TileFunc func(ctx context.Context, records []Record) ([]Record, []string, error)

// Identity returns its input unchanged.
func Identity(ctx context.Context, records []Record) ([]Record, []string, error) {
	return records, nil, nil
}

// MTParser parses MT raw bytes into tag->value map.
type MTParser interface {
	Parse(raw []byte) (map[string]string, error)
}

// MXParser parses MX raw bytes into field->value map.
type MXParser interface {
	Parse(raw []byte) (map[string]string, error)
}
