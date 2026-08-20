package security

import (
	"context"
	"errors"
	"net/http"
)

// GetRequiredTenantID extracts tenant ID from JWT context, returns error if missing
func GetRequiredTenantID(ctx context.Context) (string, error) {
	auth, ok := AuthInfoFromContext(ctx)
	if !ok {
		return "", errors.New("authentication required")
	}
	if len(auth.TenantIDs) == 0 {
		return "", errors.New("no tenant in context")
	}
	return auth.TenantIDs[0], nil
}

// GetRequiredUserID extracts user ID from JWT context, returns error if missing
func GetRequiredUserID(ctx context.Context) (string, error) {
	auth, ok := AuthInfoFromContext(ctx)
	if !ok {
		return "", errors.New("authentication required")
	}
	if auth.UserID == "" {
		return "", errors.New("no user in context")
	}
	return auth.UserID, nil
}

// RequireAuth extracts auth info or returns 401
func RequireAuth(w http.ResponseWriter, r *http.Request) (AuthInfo, bool) {
	auth, ok := AuthInfoFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return AuthInfo{}, false
	}
	return auth, true
}

// RequireTenant extracts tenant ID or returns 401
func RequireTenant(w http.ResponseWriter, r *http.Request) (string, bool) {
	tenantID, err := GetRequiredTenantID(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return "", false
	}
	return tenantID, true
}

// RequireUser extracts user ID or returns 401
func RequireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, err := GetRequiredUserID(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return "", false
	}
	return userID, true
}
