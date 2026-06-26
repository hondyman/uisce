package auth

import (
	"context"

	imodels "github.com/hondyman/semlayer/backend/internal/models"
)

// typed context key to avoid collisions
type ctxKey string

const userKey ctxKey = "semlayer_user"

// SetUserInContext returns a new context with the user attached.
func SetUserInContext(ctx context.Context, u imodels.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// GetUserFromContext retrieves the user from context. Second return value indicates presence.
func GetUserFromContext(ctx context.Context) (imodels.User, bool) {
	if v := ctx.Value(userKey); v != nil {
		if u, ok := v.(imodels.User); ok {
			return u, true
		}
	}
	return imodels.User{}, false
}

// FallbackUser returns a default steward user for development/testing purposes.
func FallbackUser() imodels.User {
	return imodels.User{
		ID:           "user-steward-1",
		Email:        "steward@example.com",
		Name:         "Default Steward",
		Role:         "Steward",
		Roles:        []string{"Steward"},
		Organization: "Default Organization",
		Permissions:  []string{"read", "write", "admin"},
		TenantID:     "tenant-default",
		Attributes: map[string]string{
			"region": "global",
		},
		IsCoreAdmin: false,
		IsActive:    true,
	}
}

// AllowedTenantsFromContext retrieves the list of tenants the user is allowed to access
func AllowedTenantsFromContext(ctx context.Context) []string {
	u, ok := GetUserFromContext(ctx)
	if !ok {
		return []string{}
	}
	return []string{u.TenantID}
}

// TenantIDFromContext retrieves the tenant ID from context
func TenantIDFromContext(ctx context.Context) string {
	u, ok := GetUserFromContext(ctx)
	if !ok {
		return ""
	}
	return u.TenantID
}

// RolesFromContext retrieves the roles from context
func RolesFromContext(ctx context.Context) []string {
	u, ok := GetUserFromContext(ctx)
	if !ok {
		return []string{}
	}
	return u.Roles
}
