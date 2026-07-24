package auth

import (
	"context"
	"net/http"
	"strings"
)

type Config struct {
	SkipPaths       map[string]bool
	AllowDevHeader  bool
	RequireAuth     bool
	JWTSecret       []byte
	JWKSURL         string
	APIKeyValidator func(ctx context.Context, key string) (*Claims, bool)
	OnAuthenticated func(ctx context.Context, claims *Claims, identity *Identity)
}

type Verifier interface {
	VerifyRequest(r *http.Request) (*Claims, error)
}

type JWTAndAPIKeyVerifier struct {
	HS256Verifier  *HS256Verifier
	RS256Verifier  *RS256Verifier
	APIKeyValidator func(ctx context.Context, key string) (*Claims, bool)
	AllowDevHeader  bool
	RequireAuth     bool
	SkipPaths       map[string]bool
	OnAuthenticated func(ctx context.Context, claims *Claims, identity *Identity)
}

func NewJWTAndAPIKeyVerifier(cfg Config) *JWTAndAPIKeyVerifier {
	var hs256 *HS256Verifier
	var rs256 *RS256Verifier
	if len(cfg.JWTSecret) > 0 {
		hs256 = NewHS256Verifier(cfg.JWTSecret)
	}
	if cfg.JWKSURL != "" {
		rs256 = NewRS256Verifier(cfg.JWKSURL)
	}
	return &JWTAndAPIKeyVerifier{
		HS256Verifier:   hs256,
		RS256Verifier:   rs256,
		APIKeyValidator: cfg.APIKeyValidator,
		AllowDevHeader:  cfg.AllowDevHeader,
		RequireAuth:     cfg.RequireAuth,
		SkipPaths:       cfg.SkipPaths,
		OnAuthenticated: cfg.OnAuthenticated,
	}
}

func (v *JWTAndAPIKeyVerifier) VerifyRequest(r *http.Request) (*Claims, error) {
	if v.SkipPaths != nil && v.SkipPaths[r.URL.Path] {
		return nil, nil
	}

	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" && v.APIKeyValidator != nil {
		if claims, ok := v.APIKeyValidator(r.Context(), apiKey); ok {
			return claims, nil
		}
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		if v.AllowDevHeader {
			if uid := r.Header.Get("X-User-ID"); uid != "" {
				claims := &Claims{UserID: uid, IsActive: true}
				return claims, nil
			}
		}
		if v.RequireAuth {
			return nil, ErrMissingToken
		}
		return nil, nil
	}

	stripped := strings.TrimSpace(authHeader)
	if strings.HasPrefix(strings.ToLower(stripped), "bearer ") {
		stripped = strings.TrimSpace(stripped[7:])
	}
	if stripped == "" {
		if v.RequireAuth {
			return nil, ErrMissingToken
		}
		return nil, nil
	}

	if v.HS256Verifier != nil {
		if claims, err := v.HS256Verifier.Verify(stripped); err == nil {
			return claims, nil
		}
	}
	if v.RS256Verifier != nil {
		if claims, err := v.RS256Verifier.Verify(stripped); err == nil {
			return claims, nil
		}
	}

	return nil, ErrTokenInvalid
}

func (v *JWTAndAPIKeyVerifier) VerifyRequestAndBuildIdentity(r *http.Request) (*Claims, *Identity, error) {
	claims, err := v.VerifyRequest(r)
	if err != nil {
		return nil, nil, err
	}
	if claims == nil && v.RequireAuth {
		return nil, nil, ErrUnauthorized
	}

	var identity *Identity
	if claims != nil {
		identity = &Identity{
			UserID:    claims.UserID,
			Email:     claims.Email,
			TenantID:  claims.TenantID,
			TenantIDs: NormalizeTenantIDs(claims.TenantIDs, claims.TenantID),
			Roles:     NormalizeRoles(claims.Roles),
			IsGlobalAdmin: IsGlobalAdminFromRoles(claims.Roles),
		}
		if v.OnAuthenticated != nil {
			v.OnAuthenticated(r.Context(), claims, identity)
		}
	}
	return claims, identity, nil
}
