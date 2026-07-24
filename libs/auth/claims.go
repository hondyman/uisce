package auth

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID         string   `json:"user_id"`
	Email          string   `json:"email,omitempty"`
	TenantID       string   `json:"tenant_id,omitempty"`
	TenantIDs      []string `json:"tenant_ids,omitempty"`
	Roles          []string `json:"roles,omitempty"`
	IsActive       bool     `json:"is_active"`
	IsCoreAdmin    bool     `json:"is_core_admin,omitempty"`
	OrganizationID string   `json:"organization_id,omitempty"`
	jwt.RegisteredClaims
}

type ImpersonationInfo struct {
	Active    bool
	SessionID string
	Mode      string
	AdminRole string
	RealUserID string
}

type Identity struct {
	UserID    string
	Email     string
	TenantID  string
	TenantIDs []string
	Roles     []string

	IsGlobalAdmin bool

	Impersonation ImpersonationInfo
}

type contextKey struct{}

var claimsKey = contextKey{}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	v := ctx.Value(claimsKey)
	if v == nil {
		return nil, false
	}
	c, ok := v.(*Claims)
	return c, ok
}

type identityKey struct{}

func WithIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

func IdentityFromContext(ctx context.Context) (*Identity, bool) {
	v := ctx.Value(identityKey{})
	if v == nil {
		return nil, false
	}
	id, ok := v.(*Identity)
	return id, ok
}

func (c *Claims) HasRole(role string) bool {
	if c.IsCoreAdmin {
		return true
	}
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func (c *Claims) IsGlobalAdminRole() bool {
	if c.IsCoreAdmin {
		return true
	}
	for _, r := range c.Roles {
		if r == "global_admin" || r == "global_ops" {
			return true
		}
	}
	return false
}

func (c *Claims) CanAccessTenant(requestedTenantID string) bool {
	if c.IsGlobalAdminRole() {
		return true
	}
	if requestedTenantID != "" && requestedTenantID == c.TenantID {
		return true
	}
	for _, tid := range c.TenantIDs {
		if tid == requestedTenantID {
			return true
		}
	}
	return false
}

func NormalizeTenantIDs(values []string, fallback string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := TrimString(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 && TrimString(fallback) != "" {
		result = append(result, TrimString(fallback))
	}
	return result
}

func NormalizeRoles(values []string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := TrimString(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func TrimString(s string) string {
	return strings.TrimSpace(s)
}

func IsGlobalAdminFromRoles(roles []string) bool {
	for _, r := range NormalizeRoles(roles) {
		if r == "global_admin" || r == "global_ops" || r == "core_admin" || r == "is_core_admin" {
			return true
		}
	}
	return false
}
