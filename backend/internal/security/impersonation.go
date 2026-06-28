package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Impersonation Types
// ============================================================================

// ImpersonationMode controls what a global admin can do inside the session.
type ImpersonationMode string

const (
	// ModeReadOnly restricts the admin to read-only operations on client data.
	// This is the default and must be the starting mode for all sessions.
	ModeReadOnly ImpersonationMode = "read_only"

	// ModeBreakGlass permits state-changing operations on Business Objects.
	// Requires a valid support ticket reference and writes a BREAK_GLASS_ACTION
	// audit event for every mutation.
	ModeBreakGlass ImpersonationMode = "break_glass"
)

// ImpersonationEventType is the type written to platform_admin_audit.event_type.
type ImpersonationEventType string

const (
	EventImpersonationStart  ImpersonationEventType = "IMPERSONATION_START"
	EventImpersonationEnd    ImpersonationEventType = "IMPERSONATION_END"
	EventBreakGlassAction    ImpersonationEventType = "BREAK_GLASS_ACTION"
)

// MaxImpersonationDuration is the hard server-side cap on any impersonation window.
const MaxImpersonationDuration = 2 * time.Hour

// ImpersonationRequest is the validated input from the HTTP handler.
type ImpersonationRequest struct {
	// TargetTenantID is the UUID of the client tenant to impersonate.
	TargetTenantID uuid.UUID

	// Reason is a mandatory free-text justification. Minimum 10 characters.
	Reason string

	// TicketReference is the support/change ticket ID. Required for break_glass mode;
	// strongly recommended for read_only.
	TicketReference string

	// Mode controls access level during the impersonation window.
	Mode ImpersonationMode

	// Duration is the requested window length. Capped server-side to MaxImpersonationDuration.
	Duration time.Duration

	// IPAddress and UserAgent are populated by the HTTP handler from the request.
	IPAddress string
	UserAgent string
}

// ImpersonationSession is the resolved, validated session record that gets
// written to platform_admin_audit and embedded in the context token.
type ImpersonationSession struct {
	SessionID       uuid.UUID
	AdminUserID     string
	AdminEmail      string
	TargetTenantID  uuid.UUID
	Reason          string
	TicketReference string
	Mode            ImpersonationMode
	Duration        time.Duration
	ExpiresAt       time.Time
	IPAddress       string
	UserAgent       string
}

// ImpersonationContextToken is the signed platform-internal token returned to
// the frontend. The frontend attaches it as the Authorization header for all
// API calls during the impersonation window.
//
// The token explicitly carries a concrete, singular TenantID so that all
// downstream tenant isolation logic (RLS, security.BuildContext, ABAC) works
// identically to a normal tenant user — no special-casing required.
type ImpersonationContextToken struct {
	// AccessToken is the HMAC-SHA256 signed JSON payload (base64-encoded).
	AccessToken string `json:"access_token"`

	// TokenType is always "Bearer".
	TokenType string `json:"token_type"`

	// ExpiresAt is the absolute expiry time (UTC). The frontend must NOT use
	// this token after this timestamp.
	ExpiresAt time.Time `json:"expires_at"`

	// SessionID links this token to the platform_admin_audit rows.
	SessionID uuid.UUID `json:"session_id"`

	// TenantID is the concrete target tenant UUID.
	TenantID uuid.UUID `json:"tenant_id"`

	// Mode exposes the access level so the UI can render the correct banner.
	Mode ImpersonationMode `json:"mode"`
}

// impersonationTokenPayload is the internal JWT-like payload that gets HMAC-signed.
type impersonationTokenPayload struct {
	// Sub is the REAL admin's user ID (immutable, cannot be spoofed by frontend).
	Sub string `json:"sub"`

	// AdminEmail is the real admin's email address.
	AdminEmail string `json:"admin_email"`

	// TenantID is the target tenant's UUID.
	// This is what downstream middleware reads — identical to a normal user's token.
	TenantID string `json:"tenant_id"`

	// ImpersonationActive signals to middleware that this is an impersonation context.
	ImpersonationActive bool `json:"impersonation_active"`

	// SessionID links back to platform_admin_audit.
	SessionID string `json:"session_id"`

	// Mode is the operational mode for this session.
	Mode ImpersonationMode `json:"mode"`

	// ExpiresAt is unix epoch seconds.
	ExpiresAt int64 `json:"exp"`

	// IssuedAt is unix epoch seconds.
	IssuedAt int64 `json:"iat"`
}

// ============================================================================
// AuditLogger Interface
// ============================================================================

// ImpersonationAuditLogger defines the synchronous OLTP audit interface.
// All implementations MUST write synchronously — no goroutines, no channels.
// If the write fails, the caller must abort the operation.
type ImpersonationAuditLogger interface {
	// LogStart records the intent to begin an impersonation session.
	// This MUST succeed before any context token is issued.
	LogStart(ctx context.Context, session ImpersonationSession) error

	// LogEnd records the termination of an impersonation session.
	LogEnd(ctx context.Context, sessionID uuid.UUID, adminUserID string, targetTenantID uuid.UUID) error

	// LogBreakGlassAction records a state-changing operation performed under break_glass mode.
	LogBreakGlassAction(ctx context.Context, sessionID uuid.UUID, adminUserID string, targetTenantID uuid.UUID, detail map[string]any) error
}

// ============================================================================
// ContextExchangeService
// ============================================================================

// ContextExchangeService orchestrates the safe, auditable transition of a
// global admin into an explicit, isolated tenant context.
//
// Design contract (Zero-Tolerance Security Mandate):
//  1. Admin must have the global_admin role in their primary JWT.
//  2. TargetTenantID must be a valid, non-nil UUID.
//  3. Audit log write MUST succeed before any token is issued. Fail = abort.
//  4. Issued token carries a CONCRETE tenant UUID — never a wildcard.
//  5. Maximum session duration is capped at MaxImpersonationDuration (2h).
type ContextExchangeService struct {
	audit ImpersonationAuditLogger
}

// NewContextExchangeService constructs a ContextExchangeService.
func NewContextExchangeService(audit ImpersonationAuditLogger) *ContextExchangeService {
	if audit == nil {
		panic("security: ContextExchangeService requires a non-nil ImpersonationAuditLogger")
	}
	return &ContextExchangeService{audit: audit}
}

// AssumeTenantContext validates the admin's identity, enforces governance rules,
// writes a synchronous audit record, and issues a scoped ImpersonationContextToken.
//
// The audit write happens BEFORE token issuance. If the DB write fails, the
// function returns an error and NO token is issued. This is intentional and
// preserves the invariant: "if a token exists, an audit record exists."
func (s *ContextExchangeService) AssumeTenantContext(
	ctx context.Context,
	adminUserID string,
	adminEmail string,
	adminRoles []string,
	req ImpersonationRequest,
) (*ImpersonationContextToken, error) {
	// ── 1. Enforce Zero-Tolerance Security Mandate ──────────────────────────
	if !isGlobalAdmin(adminRoles) {
		return nil, fmt.Errorf(
			"security violation: user %s lacks global_admin or global_ops role — impersonation denied",
			adminUserID,
		)
	}

	if req.TargetTenantID == uuid.Nil {
		return nil, errors.New("invalid operation: target_tenant_id cannot be nil")
	}

	if len(req.Reason) < 10 {
		return nil, errors.New("governance violation: reason must be at least 10 characters")
	}

	if req.Mode == ModeBreakGlass && req.TicketReference == "" {
		return nil, errors.New("governance violation: ticket_reference is mandatory for break_glass mode")
	}

	// ── 2. Resolve + cap the session window ─────────────────────────────────
	duration := req.Duration
	if duration <= 0 {
		duration = 30 * time.Minute // safe default
	}
	if duration > MaxImpersonationDuration {
		duration = MaxImpersonationDuration // hard cap — never negotiable
	}

	mode := req.Mode
	if mode == "" {
		mode = ModeReadOnly // always default to least privilege
	}

	sessionID := uuid.New()
	expiresAt := time.Now().UTC().Add(duration)

	session := ImpersonationSession{
		SessionID:       sessionID,
		AdminUserID:     adminUserID,
		AdminEmail:      adminEmail,
		TargetTenantID:  req.TargetTenantID,
		Reason:          req.Reason,
		TicketReference: req.TicketReference,
		Mode:            mode,
		Duration:        duration,
		ExpiresAt:       expiresAt,
		IPAddress:       req.IPAddress,
		UserAgent:       req.UserAgent,
	}

	// ── 3. Synchronous audit write BEFORE token issuance ────────────────────
	// Critical invariant: if this write fails, no token is issued.
	if err := s.audit.LogStart(ctx, session); err != nil {
		// Fail loudly and securely — do NOT issue a token if audit fails.
		return nil, fmt.Errorf(
			"critical invariant failure: could not commit audit trail record — impersonation aborted: %w",
			err,
		)
	}

	// ── 4. Sign and issue the context token ─────────────────────────────────
	tokenStr, err := signImpersonationToken(session)
	if err != nil {
		// Audit is already written — log this internal error but do not leak details.
		return nil, fmt.Errorf("token signing failed after successful audit write: %w", err)
	}

	return &ImpersonationContextToken{
		AccessToken: tokenStr,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		SessionID:   sessionID,
		TenantID:    req.TargetTenantID,
		Mode:        mode,
	}, nil
}

// ExitTenantContext records the end of an impersonation session.
// The frontend should call this when the admin clicks "Exit Impersonation"
// or when the session timer expires client-side.
func (s *ContextExchangeService) ExitTenantContext(
	ctx context.Context,
	sessionID uuid.UUID,
	adminUserID string,
	targetTenantID uuid.UUID,
) error {
	return s.audit.LogEnd(ctx, sessionID, adminUserID, targetTenantID)
}

// ============================================================================
// Token Signing (HMAC-SHA256 — internal platform token, not a Keycloak token)
// ============================================================================

// signImpersonationToken creates a base64-encoded HMAC-SHA256 signed payload.
// The secret is taken from IMPERSONATION_TOKEN_SECRET env var, falling back
// to JWT_SECRET as a secondary option. Both must be set in production.
func signImpersonationToken(session ImpersonationSession) (string, error) {
	secret := os.Getenv("IMPERSONATION_TOKEN_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		return "", errors.New("IMPERSONATION_TOKEN_SECRET (or JWT_SECRET) is not configured")
	}

	payload := impersonationTokenPayload{
		Sub:                 session.AdminUserID,
		AdminEmail:          session.AdminEmail,
		TenantID:            session.TargetTenantID.String(),
		ImpersonationActive: true,
		SessionID:           session.SessionID.String(),
		Mode:                session.Mode,
		ExpiresAt:           session.ExpiresAt.Unix(),
		IssuedAt:            time.Now().UTC().Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token payload: %w", err)
	}

	// HMAC-SHA256 the payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	sig := mac.Sum(nil)

	// Encode as: base64(payload).base64(signature)
	encoded := base64.RawURLEncoding.EncodeToString(payloadBytes) +
		"." +
		base64.RawURLEncoding.EncodeToString(sig)

	return encoded, nil
}

// ValidateImpersonationToken parses and verifies an impersonation context token.
// Returns the payload if valid, error if expired or signature mismatch.
func ValidateImpersonationToken(tokenStr string) (*impersonationTokenPayload, error) {
	secret := os.Getenv("IMPERSONATION_TOKEN_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		return nil, errors.New("IMPERSONATION_TOKEN_SECRET (or JWT_SECRET) is not configured")
	}

	// Split into payload.signature
	dotIdx := -1
	for i := len(tokenStr) - 1; i >= 0; i-- {
		if tokenStr[i] == '.' {
			dotIdx = i
			break
		}
	}
	if dotIdx < 0 {
		return nil, errors.New("invalid impersonation token format")
	}

	payloadB64 := tokenStr[:dotIdx]
	sigB64 := tokenStr[dotIdx+1:]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}

	expectedSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token signature: %w", err)
	}

	// Verify HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBytes)
	actualSig := mac.Sum(nil)

	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("impersonation token signature verification failed")
	}

	var payload impersonationTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token payload: %w", err)
	}

	// Check expiry
	if time.Now().UTC().Unix() > payload.ExpiresAt {
		return nil, errors.New("impersonation token has expired")
	}

	return &payload, nil
}

// ============================================================================
// Helpers
// ============================================================================

// isGlobalAdmin returns true if any of the provided roles grants global admin access.
func isGlobalAdmin(roles []string) bool {
	for _, r := range roles {
		if r == "global_admin" || r == "global_ops" {
			return true
		}
	}
	return false
}
