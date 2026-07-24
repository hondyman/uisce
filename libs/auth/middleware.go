package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type MiddlewareOptions struct {
	SkipPaths      map[string]bool
	AllowDevHeader bool
	RequireAuth    bool
	OnAuthFailure  func(w http.ResponseWriter, r *http.Request, err error)
	OnAuthSuccess  func(r *http.Request, claims *Claims, identity *Identity)
}

func HTTPMiddleware(verifier *JWTAndAPIKeyVerifier, opts MiddlewareOptions) func(http.Handler) http.Handler {
	if opts.OnAuthFailure == nil {
		opts.OnAuthFailure = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, `{"error": "unauthorized", "message": "`+err.Error()+`"}`, http.StatusUnauthorized)
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, identity, err := verifier.VerifyRequestAndBuildIdentity(r)
			if err != nil {
				opts.OnAuthFailure(w, r, err)
				return
			}
			if claims == nil && opts.RequireAuth {
				opts.OnAuthFailure(w, r, ErrUnauthorized)
				return
			}

			if claims != nil {
				ctx := r.Context()
				ctx = WithClaims(ctx, claims)
				ctx = WithIdentity(ctx, identity)
				if identity != nil && identity.TenantID != "" {
					r.Header.Set("X-Tenant-ID", identity.TenantID)
				}
				if identity != nil && identity.UserID != "" {
					r.Header.Set("X-User-ID", identity.UserID)
				}
				r = r.WithContext(ctx)
				if opts.OnAuthSuccess != nil {
					opts.OnAuthSuccess(r, claims, identity)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireTenantHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := ClaimsFromContext(r.Context())
		if claims == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant_id")
		}
		if tenantID == "" {
			http.Error(w, `{"error": "tenant_id is required"}`, http.StatusBadRequest)
			return
		}
		if !claims.CanAccessTenant(tenantID) {
			http.Error(w, `{"error": "forbidden", "message": "tenant access denied"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireRoleHandler(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := ClaimsFromContext(r.Context())
		if claims == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if !claims.HasRole(role) {
			http.Error(w, `{"error": "forbidden", "message": "insufficient permissions"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireRolesHandler(roles []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := ClaimsFromContext(r.Context())
		if claims == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ok := false
		for _, role := range roles {
			if claims.HasRole(role) {
				ok = true
				break
			}
		}
		if !ok {
			http.Error(w, `{"error": "forbidden", "message": "insufficient permissions"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireImpersonationModeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IdentityFromContext(r.Context())
		if !ok || id == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if !id.Impersonation.Active {
			http.Error(w, `{"error": "forbidden", "message": "impersonation required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetClaimsFromRequest(r *http.Request) *Claims {
	claims, _ := ClaimsFromContext(r.Context())
	return claims
}

func GetIdentityFromRequest(r *http.Request) *Identity {
	id, _ := IdentityFromContext(r.Context())
	return id
}

func GetTenantIDFromRequest(r *http.Request) string {
	if id, ok := IdentityFromContext(r.Context()); ok && id != nil {
		return id.TenantID
	}
	if claims, ok := ClaimsFromContext(r.Context()); ok && claims != nil {
		return claims.TenantID
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID != "" {
		return tenantID
	}
	return r.URL.Query().Get("tenant_id")
}

func GetUserIDFromRequest(r *http.Request) string {
	if id, ok := IdentityFromContext(r.Context()); ok && id != nil {
		return id.UserID
	}
	if claims, ok := ClaimsFromContext(r.Context()); ok && claims != nil {
		return claims.UserID
	}
	return r.Header.Get("X-User-ID")
}

func HasRoleInRequest(r *http.Request, role string) bool {
	if claims := GetClaimsFromRequest(r); claims != nil {
		return claims.HasRole(role)
	}
	return false
}

func IsGlobalAdminInRequest(r *http.Request) bool {
	if claims := GetClaimsFromRequest(r); claims != nil {
		return claims.IsGlobalAdminRole()
	}
	return false
}

type OptionalMiddleware struct {
	Verifier       Verifier
	Logger         *log.Logger
	OnAuthenticated func(r *http.Request, claims *Claims)
}

func OptionalHTTPMiddleware(verifier Verifier, opts ...OptionalMiddlewareOption) func(http.Handler) http.Handler {
	om := &OptionalMiddleware{Verifier: verifier}
	for _, o := range opts {
		o(om)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				claims, err := verifier.VerifyRequest(r)
				if err == nil && claims != nil {
					ctx := r.Context()
					ctx = WithClaims(ctx, claims)
					identity := &Identity{
						UserID:    claims.UserID,
						Email:     claims.Email,
						TenantID:  claims.TenantID,
						TenantIDs: NormalizeTenantIDs(claims.TenantIDs, claims.TenantID),
						Roles:     NormalizeRoles(claims.Roles),
						IsGlobalAdmin: IsGlobalAdminFromRoles(claims.Roles),
					}
					ctx = WithIdentity(ctx, identity)
					if om.Logger != nil {
						om.Logger.Printf("auth: optional JWT validated for user=%s tenant=%s", claims.UserID, claims.TenantID)
					}
					if om.OnAuthenticated != nil {
						om.OnAuthenticated(r, claims)
					}
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

type OptionalMiddlewareOption func(*OptionalMiddleware)

func WithOptionalLogger(logger *log.Logger) OptionalMiddlewareOption {
	return func(o *OptionalMiddleware) { o.Logger = logger }
}

func WithOptionalOnAuthenticated(fn func(r *http.Request, claims *Claims)) OptionalMiddlewareOption {
	return func(o *OptionalMiddleware) { o.OnAuthenticated = fn }
}

func WriteError(w http.ResponseWriter, err error, status int) {
	body, _ := json.Marshal(map[string]string{"error": err.Error()})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body)
}

func CheckTenant(tenantID string, claims *Claims) error {
	if claims == nil {
		return ErrUnauthorized
	}
	if !claims.CanAccessTenant(tenantID) {
		return ErrInsufficientAccess
	}
	return nil
}

func CheckRole(role string, claims *Claims) error {
	if claims == nil {
		return ErrUnauthorized
	}
	if !claims.HasRole(role) {
		return ErrInsufficientAccess
	}
	return nil
}

func CheckRoles(roles []string, claims *Claims) error {
	if claims == nil {
		return ErrUnauthorized
	}
	for _, role := range roles {
		if claims.HasRole(role) {
			return nil
		}
	}
	return ErrInsufficientAccess
}
