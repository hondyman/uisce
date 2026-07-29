package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type IdentityEnrichmentConfig struct {
	DB *security.ProfileService
}

func IdentityEnrichmentMiddleware(cfg IdentityEnrichmentConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authInfo, ok := security.AuthInfoFromContext(ctx)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			if cfg.DB == nil {
				next.ServeHTTP(w, r)
				return
			}

			tenantIDStr := ""
			if len(authInfo.TenantIDs) > 0 {
				tenantIDStr = authInfo.TenantIDs[0]
			}
			if tenantIDStr == "" {
				next.ServeHTTP(w, r)
				return
			}

			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			var idpGroups []string
			if claims, ok := authInfo.RawClaims.(*jwtmiddleware.JWTClaims); ok && claims != nil {
				idpGroups = claims.IdpGroups
			}

			if idpGroups == nil {
				idpGroups = []string{}
			}

			functionalRole, clearanceLevel, err := cfg.DB.EnrichSubjectAttributes(
				ctx, tenantID, authInfo.UserID, idpGroups,
			)
			if err != nil {
				http.Error(w, "identity enrichment failed", http.StatusInternalServerError)
				return
			}

			authInfo.FunctionalRole = functionalRole
			authInfo.ClearanceLevel = clearanceLevel

			ctx = security.WithAuthInfo(ctx, authInfo)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

type identityContextKey struct{}

func WithIdentityContext(ctx context.Context, identity IdentityContext) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}

type IdentityContext struct {
	FunctionalRole string
	ClearanceLevel string
	UserID         string
	TenantID       string
}

func IdentityContextFromContext(ctx context.Context) (IdentityContext, bool) {
	value := ctx.Value(identityContextKey{})
	identity, ok := value.(IdentityContext)
	return identity, ok
}

func ResolveIdentityContext(ctx context.Context) IdentityContext {
	if identity, ok := IdentityContextFromContext(ctx); ok {
		return identity
	}

	if authInfo, ok := security.AuthInfoFromContext(ctx); ok {
		tenantID := ""
		if len(authInfo.TenantIDs) > 0 {
			tenantID = authInfo.TenantIDs[0]
		}
		return IdentityContext{
			FunctionalRole: authInfo.FunctionalRole,
			ClearanceLevel: authInfo.ClearanceLevel,
			UserID:         authInfo.UserID,
			TenantID:       tenantID,
		}
	}

	return IdentityContext{
		FunctionalRole: "standard_guest",
		ClearanceLevel: "L1",
	}
}
