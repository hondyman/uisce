package middleware

import (
	"net/http"
	"os"
	"strconv"
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
	//
	// This is exact-path-or-prefix matching only. The moment this list
	// grows past a handful of literal paths, it will need chi-route-
	// pattern awareness (e.g. matching "/api/bo/{boKey}/records" as a
	// pattern rather than one prefix per concrete boKey) - decide that
	// matching contract before the list grows, not after the first
	// pattern-shaped entry is needed under time pressure.
	Allowlist []string
}

// AuthGateModeFromEnv reads AUTH_GATE_MODE ("off" | "shadow" | "enforce").
// Unset (empty) is treated as "off" - the gate must be inert until someone
// deliberately turns it on. An unrecognized non-empty value is NOT treated
// the same way: it panics at startup rather than silently falling back to
// "off".
//
// This asymmetry is deliberate. A silent fail-open on typos (e.g.
// AUTH_GATE_MODE=enforc) is the same "control that exists on paper" failure
// this sweep has already caught three times - branch protection's
// enforce_admins defaulting to false, AuthContextMiddleware's header
// rewrite being mistaken for a real gate, and CI's red baseline making a
// broken check look present. Once "enforce" is the intended production
// mode, a typo that silently degrades to "off" means the operator believes
// the platform is protected and it isn't - so a loud startup crash is the
// correct failure mode here, not a quiet default.
func AuthGateModeFromEnv() AuthGateMode {
	raw := strings.TrimSpace(os.Getenv("AUTH_GATE_MODE"))
	if raw == "" {
		return AuthGateOff
	}
	switch strings.ToLower(raw) {
	case string(AuthGateOff):
		return AuthGateOff
	case string(AuthGateShadow):
		return AuthGateShadow
	case string(AuthGateEnforce):
		return AuthGateEnforce
	default:
		panic("AUTH_GATE_MODE has an unrecognized value " + strconv.Quote(raw) +
			" - must be unset, \"off\", \"shadow\", or \"enforce\". Refusing to silently " +
			"fall back to \"off\": once this gate is meant to be enforcing, a typo that " +
			"silently disables it is worse than a startup crash.")
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

// statusCapturingWriter records the status code a handler wrote without
// altering anything about the write itself - WriteHeader and Write are
// both forwarded unchanged. Used only to enrich the shadow-mode violation
// log line; must never change response bytes, headers, or timing in any
// observable way. See TestAuthGate_Shadow_NeverAltersResponse, which pins
// this property directly rather than trusting this comment.
type statusCapturingWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusCapturingWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusCapturingWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
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

			if cfg.Mode == AuthGateEnforce {
				logging.GetLogger().Sugar().Warnf(
					"[AuthGateMiddleware] mode=enforce violation: %s %s has no AuthInfo and is not allowlisted - rejected",
					r.Method, r.URL.Path,
				)
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			// Shadow mode: pure observation. Wrap the writer only to learn
			// the status the handler itself produced - every byte, header,
			// and status code the wrapper sees is forwarded unchanged, so
			// this adds a log line and nothing else. The captured status is
			// what turns the log into a mechanical classification: "200 +
			// no-auth + not-allowlisted" is the real exposure list, "401 +
			// no-auth" is a handler that already gates itself (redundant
			// once this gate enforces, not currently unsafe).
			capture := &statusCapturingWriter{ResponseWriter: w}
			next.ServeHTTP(capture, r)
			status := capture.status
			if status == 0 {
				status = http.StatusOK
			}
			logging.GetLogger().Sugar().Warnf(
				"[AuthGateMiddleware] mode=shadow violation: %s %s has no AuthInfo and is not allowlisted (handler responded %d)",
				r.Method, r.URL.Path, status,
			)
		})
	}
}
