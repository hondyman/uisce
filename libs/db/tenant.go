package database

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/db/queries"
)

type contextKey string

const tenantKey contextKey = "tenant_id"

var ErrNoTenant = errors.New("no tenant context set")

func WithTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

func TenantFromContext(ctx context.Context) (uuid.UUID, error) {
	tenant, ok := ctx.Value(tenantKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrNoTenant
	}
	return tenant, nil
}

func (p *Pool) SetTenantSetting(ctx context.Context, tenantID uuid.UUID) error {
	_, err := p.Exec(ctx, queries.SetTenantGUC, tenantID.String())
	return err
}
