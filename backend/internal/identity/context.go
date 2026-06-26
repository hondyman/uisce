package identity

import "context"

type ctxKey string

const (
	CtxActorIDKey  ctxKey = "actorID"
	CtxTenantIDKey ctxKey = "tenantID"
)

// IdentityContext represents the identity information for a request
type IdentityContext struct {
	TenantID string
	Roles    []string
}

func WithActorTenant(ctx context.Context, actorID, tenantID string) context.Context {
	ctx = context.WithValue(ctx, CtxActorIDKey, actorID)
	ctx = context.WithValue(ctx, CtxTenantIDKey, tenantID)
	return ctx
}

func ActorIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(CtxActorIDKey)
	s, ok := v.(string)
	return s, ok && s != ""
}

func TenantIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(CtxTenantIDKey)
	s, ok := v.(string)
	return s, ok && s != ""
}
