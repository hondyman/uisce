package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
)

// AdminImpersonateHandler handles the tenant impersonation context exchange.
// It exposes two endpoints:
//
//   - POST   /api/admin/impersonate         — assume a tenant context
//   - DELETE /api/admin/impersonate/:sessionId — exit a tenant context
//
// Both endpoints require the caller to present a valid primary JWT containing
// the global_admin or global_ops role. The Authorization header is validated
// by AuthContextMiddleware BEFORE this handler is invoked.
type AdminImpersonateHandler struct {
	svc    *security.ContextExchangeService
	logger *security.PlatformAdminAuditLogger
}

// NewAdminImpersonateHandler constructs the handler, wiring the DB-backed
// synchronous audit logger to the ContextExchangeService.
func NewAdminImpersonateHandler(db *sql.DB) *AdminImpersonateHandler {
	auditLogger := security.NewPlatformAdminAuditLogger(db)
	svc := security.NewContextExchangeService(auditLogger)
	return &AdminImpersonateHandler{
		svc:    svc,
		logger: auditLogger,
	}
}

// ============================================================================
// POST /api/admin/impersonate
// ============================================================================

type assumeContextRequest struct {
	TargetTenantID  string `json:"target_tenant_id"`
	Reason          string `json:"reason"`
	TicketReference string `json:"ticket_reference"`
	Mode            string `json:"mode"`             // "read_only" | "break_glass"
	DurationMinutes int    `json:"duration_minutes"` // 1–120, default 30
}

type assumeContextResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	SessionID   string    `json:"session_id"`
	TenantID    string    `json:"tenant_id"`
	Mode        string    `json:"mode"`
}

// AssumeContext handles POST /api/admin/impersonate.
func (h *AdminImpersonateHandler) AssumeContext(w http.ResponseWriter, r *http.Request) {
	// ── 1. Extract the caller's AuthInfo from context (set by AuthContextMiddleware) ──
	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// ── 2. Guard: must be a global admin ────────────────────────────────────
	// Defence-in-depth: AuthContextMiddleware also checks this, but we
	// enforce it here too to prevent routing misconfiguration bypass.
	if !authInfo.IsGlobalAdmin {
		writeJSONError(w, http.StatusForbidden,
			fmt.Sprintf("access denied: user %s does not hold global_admin or global_ops role", authInfo.UserID))
		return
	}

	// ── 3. Parse request body ────────────────────────────────────────────────
	var req assumeContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	targetID, err := uuid.Parse(req.TargetTenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "target_tenant_id must be a valid UUID")
		return
	}

	mode := security.ImpersonationMode(req.Mode)
	if mode == "" {
		mode = security.ModeReadOnly
	}
	if mode != security.ModeReadOnly && mode != security.ModeBreakGlass {
		writeJSONError(w, http.StatusBadRequest, "mode must be 'read_only' or 'break_glass'")
		return
	}

	durationMinutes := req.DurationMinutes
	if durationMinutes <= 0 {
		durationMinutes = 30
	}

	// Extract network metadata for the audit record
	ipAddr, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ipAddr == "" {
		ipAddr = r.RemoteAddr
	}
	// Respect X-Forwarded-For if behind a proxy
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			ipAddr = strings.TrimSpace(parts[0])
		}
	}

	// ── 4. Call ContextExchangeService (audit write + token issuance) ────────
	impReq := security.ImpersonationRequest{
		TargetTenantID:  targetID,
		Reason:          req.Reason,
		TicketReference: req.TicketReference,
		Mode:            mode,
		Duration:        time.Duration(durationMinutes) * time.Minute,
		IPAddress:       ipAddr,
		UserAgent:       r.Header.Get("User-Agent"),
	}

	// Extract admin email from the JWT (stored in standard OIDC claim)
	// The email is populated by the Keycloak OIDC token; fall back to UserID.
	adminEmail := r.Header.Get("X-User-Email")
	if adminEmail == "" {
		adminEmail = authInfo.UserID
	}

	token, err := h.svc.AssumeTenantContext(
		r.Context(),
		authInfo.UserID,
		adminEmail,
		authInfo.Roles,
		impReq,
	)
	if err != nil {
		// Log the error server-side; return a safe, non-leaking error to the client.
		// The error from AssumeTenantContext already contains enough context for server logs.
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "security violation") || strings.Contains(err.Error(), "governance violation") {
			code = http.StatusForbidden
		} else if strings.Contains(err.Error(), "invalid operation") {
			code = http.StatusBadRequest
		}
		writeJSONError(w, code, err.Error())
		return
	}

	// ── 5. Return the scoped context token ───────────────────────────────────
	resp := assumeContextResponse{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		ExpiresAt:   token.ExpiresAt,
		SessionID:   token.SessionID.String(),
		TenantID:    token.TenantID.String(),
		Mode:        string(token.Mode),
	}
	writeJSON(w, http.StatusOK, resp)
}

// ============================================================================
// DELETE /api/admin/impersonate/:sessionId
// ============================================================================

// ExitContext handles DELETE /api/admin/impersonate/{sessionId}.
func (h *AdminImpersonateHandler) ExitContext(w http.ResponseWriter, r *http.Request) {
	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Extract sessionId from URL path using chi's URLParam helper.
	// chi.URLParam properly handles URL-decoding and trailing slashes.
	sessionIDStr := chi.URLParam(r, "sessionId")

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid session_id in path")
		return
	}

	// Look up the target tenant ID from the audit row (the session token contained it).
	// We do NOT rely on a client-supplied query param because that would let any caller
	// claim any tenant -- instead, recover the target from the platform_admin_audit row.
	targetTenantID, lookupErr := h.lookupSessionTenant(r.Context(), sessionID)
	if lookupErr != nil {
		writeJSONError(w, http.StatusInternalServerError,
			"failed to look up session tenant: "+lookupErr.Error())
		return
	}

	if err := h.svc.ExitTenantContext(r.Context(), security.ImpersonationSession{
		SessionID:      sessionID,
		AdminUserID:    authInfo.UserID,
		AdminEmail:     authInfo.UserID, // email not preserved on AuthInfo; fall back to UserID
		TargetTenantID: targetTenantID,
		Mode:           security.ModeReadOnly, // placeholder, recovered from START row
		ExpiresAt:      time.Now().UTC(),
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to record session end: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// lookupSessionTenant fetches the target tenant ID for a given session from
// platform_admin_audit via the ContextExchangeService. This allows ExitContext to work
// even after the impersonation token has been revoked client-side.
func (h *AdminImpersonateHandler) lookupSessionTenant(ctx context.Context, sessionID uuid.UUID) (uuid.UUID, error) {
	if h.svc == nil {
		return uuid.Nil, errors.New("context exchange service is not configured")
	}
	return h.svc.LookupSessionTenant(ctx, sessionID)
}

// ============================================================================
// Helpers
// ============================================================================
//
// writeJSON and writeJSONError live in source_preference_handler.go (declared earlier in
// the package). They are intentionally NOT redeclared here to avoid "redeclared in this
// block" errors at compile time.

// writeJSONError is a thin wrapper around the package-level writeJSON. Declared here so
// the rest of this file can use the standard "writeJSONError(w, code, msg)" pattern.
func writeJSONError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// =============================================================================
// Active / recent session queries
// =============================================================================

// ActiveImpersonationSession is one row returned by ListActiveSessions.
// Represents a session that has been started (audit START row) but has not yet
// been ended (no matching END row) AND whose expires_at is still in the future.
type ActiveImpersonationSession struct {
	SessionID      string    `json:"session_id"`
	AdminUserID    string    `json:"admin_user_id"`
	AdminEmail     string    `json:"admin_email"`
	TargetTenantID string    `json:"target_tenant_id"`
	Mode           string    `json:"mode"`
	ScopeKind      string    `json:"scope_kind,omitempty"`
	ScopeID        string    `json:"scope_id,omitempty"`
	Reason         string    `json:"reason"`
	StartedAt      time.Time `json:"started_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// ListActiveSessions handles GET /api/admin/impersonate/sessions/active
// Returns the impersonation sessions for the calling admin that are still
// active (no END row, expires_at > NOW). Used by the frontend to populate
// the "Active session" indicator and to prevent re-entering an existing context.
func (h *AdminImpersonateHandler) ListActiveSessions(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil || h.svc.AuditDB() == nil {
		writeJSONError(w, http.StatusInternalServerError, "audit logger not configured")
		return
	}
	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Find START events for this admin that have no matching END event AND
	// whose expires_at is still in the future. We use a LEFT JOIN + GROUP BY
	// approach to find sessions that haven't ended.
	const q = `
		SELECT s.session_id, s.admin_user_id, s.admin_email,
		       s.target_tenant_id, s.mode, s.scope_kind, s.scope_id,
		       s.reason, s.expires_at, MIN(s.created_at) AS started_at
		FROM platform_admin_audit s
		LEFT JOIN platform_admin_audit e
		  ON e.session_id = s.session_id AND e.event_type = $2
		WHERE s.event_type = $1
		  AND s.admin_user_id = $3
		  AND s.expires_at IS NOT NULL
		  AND s.expires_at > NOW()
		  AND e.id IS NULL
		GROUP BY s.session_id, s.admin_user_id, s.admin_email,
		         s.target_tenant_id, s.mode, s.scope_kind, s.scope_id,
		         s.reason, s.expires_at
		ORDER BY started_at DESC
		LIMIT 10
	`

	rows, err := h.svc.AuditDB().QueryContext(r.Context(), q,
		security.EventImpersonationStart,
		security.EventImpersonationEnd,
		authInfo.UserID,
	)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to query sessions: "+err.Error())
		return
	}
	defer rows.Close()

	out := make([]ActiveImpersonationSession, 0, 4)
	for rows.Next() {
		var s ActiveImpersonationSession
		var mode, scopeKind sql.NullString
		var scopeID sql.NullString
		var expiresAt sql.NullTime
		if err := rows.Scan(&s.SessionID, &s.AdminUserID, &s.AdminEmail,
			&s.TargetTenantID, &mode, &scopeKind, &scopeID,
			&s.Reason, &expiresAt, &s.StartedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
			return
		}
		s.Mode = mode.String
		s.ScopeKind = scopeKind.String
		s.ScopeID = scopeID.String
		if expiresAt.Valid {
			s.ExpiresAt = expiresAt.Time
		}
		out = append(out, s)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"active_sessions": out,
		"count":           len(out),
	})
}

// RecentImpersonationSession is one row returned by ListRecentSessions.
// Represents a distinct (admin, tenant) pair from the audit history, regardless
// of whether the session is still active. Used to populate the picker's
// "Recent sessions" strip on the server side as a fallback for the
// client-side localStorage cache (e.g. on a new browser).
type RecentImpersonationSession struct {
	TenantID    string    `json:"tenant_id"`
	TenantName  string    `json:"tenant_name,omitempty"`
	AdminUserID string    `json:"admin_user_id"`
	Mode        string    `json:"mode"`
	LastUsedAt  time.Time `json:"last_used_at"`
}

// ListRecentSessions handles GET /api/admin/impersonate/sessions/recent
// Returns the last 5 distinct tenants the admin has impersonated, ordered by
// most recent START row. The tenant display name is joined from the tenants table
// when available.
func (h *AdminImpersonateHandler) ListRecentSessions(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil || h.svc.AuditDB() == nil {
		writeJSONError(w, http.StatusInternalServerError, "audit logger not configured")
		return
	}
	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	const q = `
		SELECT s.target_tenant_id,
		       COALESCE(t.name, '') AS tenant_name,
		       s.admin_user_id,
		       s.mode,
		       MAX(s.created_at) AS last_used_at
		FROM platform_admin_audit s
		LEFT JOIN tenants t ON t.id = s.target_tenant_id
		WHERE s.event_type = $1
		  AND s.admin_user_id = $2
		GROUP BY s.target_tenant_id, t.name, s.admin_user_id, s.mode
		ORDER BY last_used_at DESC
		LIMIT 5
	`

	rows, err := h.svc.AuditDB().QueryContext(r.Context(), q,
		security.EventImpersonationStart,
		authInfo.UserID,
	)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to query recent sessions: "+err.Error())
		return
	}
	defer rows.Close()

	out := make([]RecentImpersonationSession, 0, 5)
	for rows.Next() {
		var s RecentImpersonationSession
		var mode sql.NullString
		var tenantName sql.NullString
		if err := rows.Scan(&s.TenantID, &tenantName, &s.AdminUserID, &mode, &s.LastUsedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
			return
		}
		s.TenantName = tenantName.String
		s.Mode = mode.String
		out = append(out, s)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"recent_sessions": out,
		"count":           len(out),
	})
}
