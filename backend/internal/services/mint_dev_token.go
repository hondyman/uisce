package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DevTokenInput is the claim shape AuthContextMiddleware reads from a
// validated JWT. MintDevToken is the only approved forge helper for
// backend tests and cmd/devjwt — it builds this map and signs via
// SignToken (JWTManager). Knowing JWT_SECRET plus this helper is an
// auth-bypass kit; keep forge use in development/local/test only.
type DevTokenInput struct {
	UserID    string
	Email     string
	TenantIDs []string
	Roles     []string
	// TTL defaults to 1 hour when zero or negative.
	TTL time.Duration
}

// DevTokenClaims builds the MapClaims AuthContextMiddleware / ValidateToken
// expect. Prefer MintDevToken for a signed string.
func DevTokenClaims(in DevTokenInput) jwt.MapClaims {
	ttl := in.TTL
	if ttl <= 0 {
		ttl = time.Hour
	}
	now := time.Now()
	userID := strings.TrimSpace(in.UserID)
	tenantIDs := normalizeList(in.TenantIDs)
	roles := normalizeList(in.Roles)

	claims := jwt.MapClaims{
		"sub":     userID,
		"user_id": userID,
		"roles":   roles,
		"iat":     now.Unix(),
		"exp":     now.Add(ttl).Unix(),
	}
	if email := strings.TrimSpace(in.Email); email != "" {
		claims["email"] = email
	}
	if len(tenantIDs) > 0 {
		claims["tenant_ids"] = tenantIDs
		claims["tenant_id"] = tenantIDs[0]
	}
	return claims
}

// MintDevToken signs a development/test JWT with DevTokenClaims via SignToken.
// This is the single backend forge path; do not HS256 tokens ad hoc.
func (sm *SecurityManager) MintDevToken(in DevTokenInput) (string, error) {
	if sm == nil || sm.jwtManager == nil {
		return "", fmt.Errorf("security manager not initialized")
	}
	if strings.TrimSpace(in.UserID) == "" {
		return "", fmt.Errorf("user_id is required")
	}
	return sm.SignToken(DevTokenClaims(in))
}
