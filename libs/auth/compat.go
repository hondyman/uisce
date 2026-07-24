package auth

import (
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type LegacyConfig struct {
	SkipPaths map[string]bool
	Logger    any
}

type LegacyMiddleware struct {
	SkipPaths map[string]bool
	Logger    any
}

func NewLegacyMiddleware(skipPaths ...string) *LegacyMiddleware {
	m := &LegacyMiddleware{SkipPaths: make(map[string]bool)}
	for _, p := range skipPaths {
		m.SkipPaths[p] = true
	}
	return m
}

func (m *LegacyMiddleware) Handler(next http.Handler) http.Handler {
	secret, err := GetJWTSecret()
	if err != nil {
		panic("JWT_SECRET not configured")
	}
	verifier := NewJWTAndAPIKeyVerifier(Config{
		JWTSecret:   secret,
		SkipPaths:   m.SkipPaths,
		RequireAuth: true,
	})
	mw := HTTPMiddleware(verifier, MiddlewareOptions{})
	return mw(next)
}

func ValidateTokenFromRequest(r *http.Request) (*Claims, error) {
	secret, err := GetJWTSecret()
	if err != nil {
		return nil, err
	}
	return ValidateRequestToken(r, secret)
}

func ValidateToken(tokenString string) (*Claims, error) {
	secret, err := GetJWTSecret()
	if err != nil {
		return nil, err
	}
	return VerifyHS256(tokenString, secret)
}

func ExtractToken(r *http.Request) (string, error) {
	return ExtractBearerToken(r)
}

func GetClaimsFromContext(r *http.Request) *Claims {
	return GetClaimsFromRequest(r)
}

func GetUserIDFromContext(r *http.Request) string {
	return GetUserIDFromRequest(r)
}

func GetTenantIDFromContext(r *http.Request) string {
	return GetTenantIDFromRequest(r)
}

func HasRole(claims *Claims, role string) bool {
	if claims == nil {
		return false
	}
	return claims.HasRole(role)
}

func ValidateTenantAccess(claims *Claims, requestedTenantID string) error {
	if claims == nil {
		return ErrUnauthorized
	}
	if !claims.CanAccessTenant(requestedTenantID) {
		return ErrInsufficientAccess
	}
	return nil
}

func RequireTenant(next http.Handler) http.Handler {
	return RequireTenantHandler(next)
}

func RequireRole(role string, next http.Handler) http.Handler {
	return RequireRoleHandler(role, next)
}

func RequireRoles(roles []string, next http.Handler) http.Handler {
	return RequireRolesHandler(roles, next)
}

func ChiMiddleware() func(next http.Handler) http.Handler {
	secret, err := GetJWTSecret()
	if err != nil {
		panic("JWT_SECRET not configured")
	}
	verifier := NewJWTAndAPIKeyVerifier(Config{
		JWTSecret: secret,
		SkipPaths: map[string]bool{
			"/health":       true,
			"/api/auth/login":  true,
			"/api/auth/refresh": true,
			"/docs":         true,
			"/docs/*":       true,
		},
		RequireAuth: true,
	})
	return HTTPMiddleware(verifier, MiddlewareOptions{})
}

type OptionalJWTMiddleware struct {
	Logger any
}

func (m *OptionalJWTMiddleware) Handler(next http.Handler) http.Handler {
	secret, err := GetJWTSecret()
	if err != nil {
		panic("JWT_SECRET not configured")
	}
	verifier := NewJWTAndAPIKeyVerifier(Config{
		JWTSecret: secret,
	})
	return OptionalHTTPMiddleware(verifier, WithOptionalLogger(m.Logger.(*log.Logger)))(next)
}

func NewOptionalJWTMiddleware() *OptionalJWTMiddleware {
	return &OptionalJWTMiddleware{}
}

type JWTClaims = Claims

var _ = jwt.RegisteredClaims{}
