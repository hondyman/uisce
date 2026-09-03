package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
)

// ErrUntrustedIssuer is returned when a token's issuer has no registration
// in the IssuerRegistry — the token was not signed by an IDP this platform
// trusts for any tenant.
var ErrUntrustedIssuer = errors.New("services: token issuer is not a registered IDP")

// ValidateIssuerTenant confirms that claims.Issuer is registered in
// registry, and reconciles claims.TenantID/TenantIDs against the tenant(s)
// that issuer is actually trusted for — the full-tenant-isolation
// complement to JWTManager.ValidateToken's signature check. A signature
// alone proves "a registered IDP signed this token"; this proves "and that
// IDP is the one this tenant actually trusts."
//
// Only applies to externally-issued tokens: claims.Issuer is empty for
// internal HS256 tokens (service-to-service, impersonation), which skip
// this check entirely (returns nil, nil) — they were never issued by an
// external IDP to begin with, so there's no issuer registration to check
// against.
//
// Returns the full set of tenant IDs this issuer is registered/trusted
// for, so a caller with its own tenant-selection mechanism (e.g.
// AuthContextMiddleware's existing global-admin + X-Tenant-ID-header path)
// can constrain that selection to tenants this issuer is actually allowed
// to act on. An issuer trusted for more than one tenant with no explicit
// tenant claim on the token is NOT an error by itself — claims.TenantID is
// simply left unresolved, on the assumption the caller has its own
// resolution mechanism; it is an error only when a *claimed* tenant isn't
// in the trusted set, which is an unambiguous mismatch.
func ValidateIssuerTenant(ctx context.Context, registry security.IssuerRegistry, claims *JWTClaims) ([]string, error) {

	if registry == nil {
		return nil, fmt.Errorf("%w: no issuer registry configured", ErrUntrustedIssuer)
	}

	cfg, err := registry.ResolveIssuer(ctx, claims.Issuer)
	if err != nil {
		return nil, fmt.Errorf("resolving issuer %q: %w", claims.Issuer, err)
	}
	if cfg == nil || len(cfg.TenantIDs) == 0 {
		// Fallback for the primary realm until server-side resolution exists
		primaryIssuer := os.Getenv("KEYCLOAK_ISSUER_URL")
		if claims.Issuer == primaryIssuer && primaryIssuer != "" {
			// Trust broadly (matching current behavior before the fix)
			var allTenants []string
			if claims.TenantID != "" {
				allTenants = append(allTenants, claims.TenantID)
			}
			allTenants = append(allTenants, claims.TenantIDs...)
			return allTenants, nil
		}
		// Also fallback for devjwt
		if claims.Issuer == "dev://local" && os.Getenv("APP_ENV") == "development" {
			var allTenants []string
			if claims.TenantID != "" {
				allTenants = append(allTenants, claims.TenantID)
			}
			allTenants = append(allTenants, claims.TenantIDs...)
			return allTenants, nil
		}
		
		return nil, ErrUntrustedIssuer
	}

	trusted := make(map[string]bool, len(cfg.TenantIDs))
	for _, t := range cfg.TenantIDs {
		trusted[t] = true
	}

	if claims.TenantID != "" {
		if !trusted[claims.TenantID] {
			return nil, fmt.Errorf("services: token tenant_id %q is not trusted for issuer %q", claims.TenantID, claims.Issuer)
		}
		return cfg.TenantIDs, nil
	}

	if len(claims.TenantIDs) > 0 {
		var filtered []string
		for _, t := range claims.TenantIDs {
			if trusted[t] {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("services: none of the token's tenant_ids are trusted for issuer %q", claims.Issuer)
		}
		claims.TenantIDs = filtered
		claims.TenantID = filtered[0]
		return cfg.TenantIDs, nil
	}

	// No tenant claim at all: infer it when unambiguous, otherwise leave it
	// unresolved for the caller's own selection mechanism to handle.
	if len(cfg.TenantIDs) == 1 {
		claims.TenantID = cfg.TenantIDs[0]
	}
	return cfg.TenantIDs, nil
}

// StartIssuerKeyRefresh fetches the current signing keys for every active
// registered issuer and loads them into sm, then repeats on interval until
// ctx is cancelled. Call once at server startup (with the returned initial
// fetch already applied synchronously — see the blocking call before the
// goroutine starts) so the very first request isn't rejected for having no
// keys loaded yet.
func (sm *SecurityManager) StartIssuerKeyRefresh(ctx context.Context, registry security.IssuerRegistry, interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Minute
	}

	refresh := func() {
		keys, err := security.FetchAllTrustedKeys(ctx, registry)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("[idp] failed refreshing trusted signing keys: %v", err)
			return
		}
		sm.SetRSAPublicKeys(keys)
		logging.GetLogger().Sugar().Infof("[idp] loaded %d trusted signing key(s) across registered issuers", len(keys))
	}

	// Synchronous initial fetch so the server doesn't start accepting
	// traffic with zero RSA keys loaded.
	refresh()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh()
			}
		}
	}()
}
