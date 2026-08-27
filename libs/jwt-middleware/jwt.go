package jwtmiddleware

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents standard JWT claims used across all services
type JWTClaims struct {
	UserID         string   `json:"user_id"`
	Email          string   `json:"email"`
	TenantID       string   `json:"tenant_id"`
	TenantIDs      []string `json:"tenant_ids,omitempty"`
	Roles          []string `json:"roles,omitempty"`
	IsActive       bool     `json:"is_active"`
	IsCoreAdmin    bool     `json:"is_core_admin"`
	OrganizationID string   `json:"organization_id,omitempty"`
	IdpGroups      []string `json:"idp_groups,omitempty"`
	jwt.RegisteredClaims
}

// ExtractToken extracts the JWT token strictly from the Authorization header (GSIFI compliance)
func ExtractToken(r *http.Request) (string, error) {
	// GSIFI Mandate: Reject any authentication credentials attempted via URL query strings or cookies in cleartext
	if r.URL.Query().Get("token") != "" || r.URL.Query().Get("access_token") != "" || r.URL.Query().Get("jwt") != "" {
		return "", errors.New("GSIFI compliance violation: transmitting authentication tokens via URL query parameters is prohibited")
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("empty token")
	}

	return token, nil
}

// ValidateToken validates a JWT token and returns the claims with strict signature checks
func ValidateToken(tokenString string) (*JWTClaims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET not configured")
	}

	// GSIFI Minimum Entropy Rule: Secret must be at least 32 characters (256 bits) in production
	if os.Getenv("ENV") == "production" && len(secret) < 32 {
		return nil, errors.New("GSIFI compliance violation: JWT_SECRET must have at least 256 bits of entropy in production")
	}

	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Strict signing method enforcement: prevent 'none' or asymmetric mismatch attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// GSIFI Inactive User Check
	if !claims.IsActive && claims.UserID != "" {
		return nil, errors.New("user account is inactive or disabled")
	}

	return claims, nil
}

// ValidateTokenFromRequest extracts and validates a JWT token from HTTP request
func ValidateTokenFromRequest(r *http.Request) (*JWTClaims, error) {
	token, err := ExtractToken(r)
	if err != nil {
		return nil, err
	}

	return ValidateToken(token)
}

// ValidateTenantAccess checks if the user has access to the requested tenant
func ValidateTenantAccess(claims *JWTClaims, requestedTenantID string) error {
	if requestedTenantID == "" {
		return errors.New("requested tenant_id cannot be empty")
	}

	// Admin users can access any tenant
	if claims.IsCoreAdmin {
		return nil
	}

	// Check if user's tenant IDs include the requested tenant
	if requestedTenantID == claims.TenantID {
		return nil
	}

	// Check against tenant_ids array
	for _, tid := range claims.TenantIDs {
		if tid == requestedTenantID {
			return nil
		}
	}

	return fmt.Errorf("user does not have access to tenant %s", requestedTenantID)
}

// HasRole checks if the user has a specific role
func HasRole(claims *JWTClaims, role string) bool {
	if claims.IsCoreAdmin {
		return true
	}

	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}

	return false
}
