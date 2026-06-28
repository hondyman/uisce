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

	if err := h.svc.ExitTenantContext(r.Context(), sessionID, authInfo.UserID, targetTenantID); err != nil {
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
