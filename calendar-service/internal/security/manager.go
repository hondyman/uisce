package security

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Manager handles security operations for the calendar service
// Aligns with backend SecurityManager pattern
type Manager struct {
	jwtSecret []byte
	logger    interface{} // Can be replaced with proper logger
}

// NewManager creates a new security manager
func NewManager(jwtSecret string) *Manager {
	return &Manager{
		jwtSecret: []byte(jwtSecret),
	}
}

// TokenClaims represents the claims in a JWT token
type TokenClaims struct {
	UserID       string
	Email        string
	TenantID     string
	TenantIDs    []string
	Roles        []string
	Permissions  []string
	Organization string
	IsCoreAdmin  bool
	JTI          string
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

// ValidateToken validates a JWT token and returns claims
func (m *Manager) ValidateToken(tokenString string) (*TokenClaims, error) {
	// Accept optional "Bearer " prefix
	if strings.HasPrefix(strings.ToLower(tokenString), "bearer ") {
		tokenString = strings.TrimSpace(tokenString[7:])
	}

	if tokenString == "" {
		return nil, fmt.Errorf("empty token")
	}

	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	// Extract user ID - try common claim names
	var userID string
	if id, ok := claims["user_id"].(string); ok && id != "" {
		userID = id
	} else if id, ok := claims["sub"].(string); ok && id != "" {
		userID = id
	} else if id, ok := claims["uid"].(string); ok && id != "" {
		userID = id
	}

	if userID == "" {
		return nil, fmt.Errorf("token missing user identifier")
	}

	// Extract tenant info
	tenantID, _ := claims["tenant_id"].(string)
	email, _ := claims["email"].(string)
	organization, _ := claims["organization"].(string)
	isCoreAdmin, _ := claims["is_core_admin"].(bool)
	jti, _ := claims["jti"].(string)

	// Parse roles
	roles := parseStringListClaim(claims["roles"])
	if len(roles) == 0 {
		if role, ok := claims["role"].(string); ok && role != "" {
			roles = []string{role}
		}
	}

	// Parse permissions
	permissions := parseStringListClaim(claims["permissions"])

	// Parse tenant_ids
	tenantIDs := parseStringListClaim(claims["tenant_ids"])
	if len(tenantIDs) == 0 && tenantID != "" {
		tenantIDs = []string{tenantID}
	}

	// Extract issued at and expires at
	issuedAt := time.Now()
	if iat, ok := claims["iat"].(float64); ok {
		issuedAt = time.Unix(int64(iat), 0)
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	if exp, ok := claims["exp"].(float64); ok {
		expiresAt = time.Unix(int64(exp), 0)
	}

	return &TokenClaims{
		UserID:       userID,
		Email:        email,
		TenantID:     tenantID,
		TenantIDs:    tenantIDs,
		Roles:        roles,
		Permissions:  permissions,
		Organization: organization,
		IsCoreAdmin:  isCoreAdmin,
		JTI:          jti,
		IssuedAt:     issuedAt,
		ExpiresAt:    expiresAt,
	}, nil
}

// HasRole checks if the claims contain a specific role
func (tc *TokenClaims) HasRole(role string) bool {
	if tc.IsCoreAdmin {
		return true
	}
	for _, r := range tc.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the claims contain a specific permission
func (tc *TokenClaims) HasPermission(permission string) bool {
	if tc.IsCoreAdmin {
		return true
	}
	for _, p := range tc.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasTenantAccess checks if user has access to specific tenant
func (tc *TokenClaims) HasTenantAccess(tenantID string) bool {
	if tc.IsCoreAdmin {
		return true
	}
	if tc.TenantID == tenantID {
		return true
	}
	for _, t := range tc.TenantIDs {
		if t == tenantID {
			return true
		}
	}
	return false
}

// parseStringListClaim extracts a list of strings from claim value
func parseStringListClaim(value interface{}) []string {
	var result []string

	if value == nil {
		return result
	}

	// If it's already a slice
	if slice, ok := value.([]interface{}); ok {
		for _, item := range slice {
			if str, ok := item.(string); ok && str != "" {
				result = append(result, str)
			}
		}
		return result
	}

	// If it's a single string
	if str, ok := value.(string); ok && str != "" {
		result = append(result, str)
	}

	return result
}
