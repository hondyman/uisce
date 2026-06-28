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
	jwt.RegisteredClaims
}

// ExtractToken extracts the JWT token from the Authorization header
func ExtractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}

	token := parts[1]
	if token == "" {
		return "", errors.New("empty token")
	}

	return token, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*JWTClaims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET not configured")
	}

	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
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
	// Admin users can access any tenant
	if claims.IsCoreAdmin {
		return nil
	}

	// Check if user's tenant IDs include the requested tenant
	if requestedTenantID != "" && requestedTenantID == claims.TenantID {
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
	// Admin users have all roles
	if claims.IsCoreAdmin {
		return true
	}

	// Check roles array
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}

	return false
}