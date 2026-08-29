package middleware

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/hondyman/uisce/backend/internal/identity"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
)

// TenantProvisioningConfig configures TenantProvisioningMiddleware.
type TenantProvisioningConfig struct {
	DB             *sql.DB
	DomainResolver *security.TenantDomainService
}

// TenantProvisioningMiddleware auto-provisions a client user's tenant
// assignment on their first authenticated request, resolved from their
// verified email domain — never from a client-controlled claim or header.
//
// It only acts when the caller has ZERO existing security.tenant_domains
// rows are irrelevant here; the check is against public.user_tenant. A user
// with any existing user_tenant row (a single client-user assignment, or
// several professional-services assignments) is left completely untouched —
// this middleware only ever fires for a true first-ever login, so it can
// never override or narrow an existing multi-tenant assignment set.
//
// Global admins and active impersonation sessions are exempt: staff identity
// is provisioned explicitly by an admin (see internal/handlers/admin_tenant_access_handler.go),
// never by domain match.
//
// SECURITY: if the domain has no registered tenant, or the email isn't
// verified, the request is rejected with 403. There is no default tenant to
// fall back to — see security.ErrDomainNotRegistered.
func TenantProvisioningMiddleware(cfg TenantProvisioningConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authInfo, ok := security.AuthInfoFromContext(r.Context())
			if !ok || authInfo.UserID == "" {
				next.ServeHTTP(w, r)
				return
			}
			if authInfo.IsGlobalAdmin || authInfo.ImpersonationActive {
				next.ServeHTTP(w, r)
				return
			}
			if len(authInfo.TenantIDs) > 0 {
				// Token or a prior lookup already carries a tenant. Nothing to do.
				next.ServeHTTP(w, r)
				return
			}
			if cfg.DB == nil || cfg.DomainResolver == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			log := logging.GetLogger().Sugar()

			var existingCount int
			if err := cfg.DB.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM public.user_tenant WHERE user_id = $1`, authInfo.UserID,
			).Scan(&existingCount); err != nil {
				log.Warnf("[TenantProvisioningMiddleware] user_tenant lookup failed for user=%s: %v", authInfo.UserID, err)
				next.ServeHTTP(w, r)
				return
			}
			if existingCount > 0 {
				// User already has assignment(s) — e.g. professional-services
				// staff with several tenants — but the token happened to carry
				// no tenant claim on this request. Leave tenant resolution to
				// whatever mechanism already handles that (route param, header
				// validated against their assignment set, etc). Auto-provisioning
				// must never guess which of several assignments to pick.
				next.ServeHTTP(w, r)
				return
			}

			claims, _ := authInfo.RawClaims.(*services.JWTClaims)
			if claims == nil || claims.Email == "" {
				http.Error(w, "Forbidden: no tenant assignment and no verified email to provision one from", http.StatusForbidden)
				return
			}
			if !claims.EmailVerified {
				http.Error(w, "Forbidden: email must be verified before tenant access can be provisioned", http.StatusForbidden)
				return
			}

			resolvedTenantID, err := cfg.DomainResolver.ResolveTenantByEmailDomain(ctx, claims.Email)
			if err != nil {
				if errors.Is(err, security.ErrDomainNotRegistered) {
					http.Error(w, "Your organization isn't set up yet. Contact support.", http.StatusForbidden)
					return
				}
				log.Warnf("[TenantProvisioningMiddleware] domain resolution failed for user=%s: %v", authInfo.UserID, err)
				http.Error(w, "Forbidden: could not resolve tenant", http.StatusForbidden)
				return
			}

			tenantID := resolvedTenantID.String()
			if _, err := cfg.DB.ExecContext(ctx, `
				INSERT INTO public.user_tenant (user_id, tenant_id, access_role, created_at, updated_at)
				VALUES ($1, $2, 'client_user', NOW(), NOW())
				ON CONFLICT (user_id, tenant_id) DO NOTHING
			`, authInfo.UserID, tenantID); err != nil {
				log.Errorf("[TenantProvisioningMiddleware] failed to provision user_tenant for user=%s tenant=%s: %v", authInfo.UserID, tenantID, err)
				http.Error(w, "Forbidden: could not provision tenant access", http.StatusForbidden)
				return
			}
			log.Infof("[TenantProvisioningMiddleware] auto-provisioned user=%s into tenant=%s via verified email domain", authInfo.UserID, tenantID)

			authInfo.TenantIDs = []string{tenantID}
			ctx = security.WithAuthInfo(ctx, authInfo)
			ctx = identity.WithActorTenant(ctx, authInfo.UserID, tenantID)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
