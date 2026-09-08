package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
)

// AuthGateMode controls how AuthGateMiddleware reacts to a request that
// carries no security.AuthInfo (i.e. AuthContextMiddleware could not
// validate any credential) for a route that isn't on the allowlist.
type AuthGateMode string

const (
	// AuthGateOff disables the gate entirely - no logging, no enforcement.
	AuthGateOff AuthGateMode = "off"
	// AuthGateShadow logs what the gate would have blocked, but always
	// calls next.ServeHTTP. This is the discovery step: run it against the
	// e2e suite and a live frontend session, and the violation log becomes
	// the enforce PR's review artifact.
	AuthGateShadow AuthGateMode = "shadow"
	// AuthGateEnforce actually rejects unauthenticated requests to
	// non-allowlisted routes with 401.
	AuthGateEnforce AuthGateMode = "enforce"
)

// AuthGateConfig configures AuthGateMiddleware.
type AuthGateConfig struct {
	Mode AuthGateMode
	// Allowlist holds exact paths and path prefixes (a trailing "*" makes
	// an entry a prefix match, e.g. "/api/public/*") that may be reached
	// with no authentication. Kept intentionally small - see
	// backend/docs/DISCOVERY_UNAUTH_ROUTES.md and the shadow-mode
	// methodology in backend/docs/INCIDENT_REPORT_20260906.md. Additive
	// only: an entry is added when shadow mode observes a real
	// pre-authentication caller and a reason is written down, never by
	// default or by inertia.
	Allowlist []string
}

// AuthGateModeFromEnv reads AUTH_GATE_MODE ("off" | "shadow" | "enforce"),
// defaulting to "off" for any unset or unrecognized value so this gate is
// inert until deliberately turned on.
func AuthGateModeFromEnv() AuthGateMode {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_GATE_MODE"))) {
	case string(AuthGateShadow):
		return AuthGateShadow
	case string(AuthGateEnforce):
		return AuthGateEnforce
	default:
		return AuthGateOff
	}
}

func isAllowlisted(path string, allowlist []string) bool {
	for _, entry := range allowlist {
		if strings.HasSuffix(entry, "*") {
			if strings.HasPrefix(path, strings.TrimSuffix(entry, "*")) {
				return true
			}
			continue
		}
		if path == entry {
			return true
		}
	}
	return false
}

// AuthGateMiddleware is the route-layer authentication gate (Fix 2 in
// backend/docs/INCIDENT_REPORT_20260906.md's tenant-resolution sweep).
//
// It deliberately does not re-validate the JWT itself - that is
// AuthContextMiddleware's job, and duplicating it here would be exactly
// the "same responsibility, two owners" pattern that produced every
// finding in this sweep. This gate only asks the one question a
// default-deny posture needs: did AuthContextMiddleware manage to
// populate security.AuthInfo for this request, and if not, is the route
// one of the deliberately public ones?
//
// Must be registered with r.Use(...) AFTER appmid.AuthContextMiddleware
// so AuthInfo is already in context by the time this runs.
func AuthGateMiddleware(cfg AuthGateConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Mode == AuthGateOff {
				next.ServeHTTP(w, r)
				return
			}

			if isAllowlisted(r.URL.Path, cfg.Allowlist) {
				next.ServeHTTP(w, r)
				return
			}

			_, authed := security.AuthInfoFromContext(r.Context())
			if authed {
				next.ServeHTTP(w, r)
				return
			}

			// Violation: no AuthInfo, route not allowlisted. In shadow mode
			// this is pure observation - it must never change response
			// behavior, so the classified reading (expected-violation vs.
			// legitimate-pre-auth-traffic vs. a caller expecting the route
			// to already be gated) stays uncontaminated by the gate itself.
			logging.GetLogger().Sugar().Warnf(
				"[AuthGateMiddleware] mode=%s violation: %s %s has no AuthInfo and is not allowlisted",
				cfg.Mode, r.Method, r.URL.Path,
			)

			if cfg.Mode == AuthGateShadow {
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		})
	}
}
