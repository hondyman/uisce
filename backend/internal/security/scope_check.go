package security

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)
// ErrImpersonationScopeViolation is returned when a request resolves to a resource
// OUTSIDE the admin's chosen impersonation scope. This is a security-critical signal
// and triggers an audit row of event_type 'SCOPE_VIOLATION_BLOCKED'.
var ErrImpersonationScopeViolation = errors.New("impersonation scope violation: request outside chosen scope")

// ScopeKind constants — must match the CHECK constraint in the migration.
const (
	ScopeTenant     = "tenant"
	ScopeInstance   = "instance"
	ScopeProduct    = "product"
	ScopeDatasource = "datasource"
)

// logger can write an actionable record.
type ScopeViolationDetails struct {
	AdminUserID       string
	SessionID         uuid.UUID
	TargetTenantID    uuid.UUID
	ChosenScopeKind   string
	ChosenScopeID     uuid.UUID
	ResolvedTenantID  uuid.UUID
	ResolvedInstID    string
	ResolvedProdID    string
	ResolvedDsID      string
	RequestPath       string
	RequestMethod     string
	IPAddress         string
	Reason            string // e.g. "read of datasources outside scope"
}

// IsWithinScope returns true if the resolved datasource/instance/product hierarchy
// is contained within the chosen impersonation scope.
//
// Scope semantics:
//   - scope_kind == "tenant"  → always allowed (full tenant access).
//   - scope_kind == "instance"  → resolved.instance_id must equal scope_id.
//   - scope_kind == "product"   → resolved.product_id must equal scope_id.
//   - scope_kind == "datasource" → resolved.datasource_id must equal scope_id.
//   - Empty scope_kind is treated as "tenant" (defence in depth — should never
//     happen because the service defaults to "tenant" but we don't want a missing
//     scope_kind to silently widen access).
func IsWithinScope(scopeKind, scopeID string, resolved ResolvedDatasource) bool {
	switch scopeKind {
	case "", ScopeTenant:
		return true
	case ScopeInstance:
		return resolved.InstanceID == scopeID
	case ScopeProduct:
		return resolved.ProductID == scopeID
	case ScopeDatasource:
		return resolved.DatasourceID == scopeID
	default:
		// Unknown scope_kind — DENY by default.
		return false
	}
}

// ValidateScope returns nil if the request is within scope, or a wrapped
// ErrImpersonationScopeViolation with diagnostic context otherwise.
//
// Use this in BuildContext after resolver.Resolve so we can include the resolved
// hierarchy in the error message for audit trail clarity.
func ValidateScope(scopeKind, scopeID string, resolved ResolvedDatasource) error {
	if IsWithinScope(scopeKind, scopeID, resolved) {
		return nil
	}
	return fmt.Errorf("%w: scope_kind=%q scope_id=%q but resolved to tenant=%s instance=%s product=%s datasource=%s",
		ErrImpersonationScopeViolation, scopeKind, scopeID,
		resolved.TenantID, resolved.InstanceID, resolved.ProductID, resolved.DatasourceID)
}

// IsImpersonationScopeViolation reports whether err is a scope violation (or wraps one).
// Used by handlers to decide whether to write a SCOPE_VIOLATION_BLOCKED audit row.
func IsImpersonationScopeViolation(err error) bool {
	return errors.Is(err, ErrImpersonationScopeViolation)
}

// =============================================================================
// Context plumbing for scope
// =============================================================================

// ImpersonationScopeContext is the in-memory representation of the impersonation
// scope, attached to the request context by AuthContextMiddleware so that
// BuildContext can perform scope enforcement without having to re-parse the token.
type ImpersonationScopeContext struct {
	Kind string // "tenant" | "instance" | "product" | "datasource"
	ID   string // UUID of the scoped resource; empty when Kind = "tenant"
}

// type alias for the context key (unexported so external packages can't mint it).
type impersonationScopeKey struct{}

// WithImpersonationScope attaches the chosen scope to the request context. Called
// by AuthContextMiddleware after validating an impersonation token.
func WithImpersonationScope(ctx context.Context, scope ImpersonationScopeContext) context.Context {
	return context.WithValue(ctx, impersonationScopeKey{}, &scope)
}

// ImpersonationScopeFromContext returns the scope attached to ctx, or a default
// (tenant-wide) scope if none is present.
func ImpersonationScopeFromContext(ctx context.Context) ImpersonationScopeContext {
	v := ctx.Value(impersonationScopeKey{})
	if v == nil {
		return ImpersonationScopeContext{Kind: ScopeTenant}
	}
	s, ok := v.(*ImpersonationScopeContext)
	if !ok || s == nil {
		return ImpersonationScopeContext{Kind: ScopeTenant}
	}
	return *s
}