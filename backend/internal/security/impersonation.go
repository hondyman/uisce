package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
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
	EventImpersonationStart   ImpersonationEventType = "IMPERSONATION_START"
	EventImpersonationEnd     ImpersonationEventType = "IMPERSONATION_END"
	EventBreakGlassAction     ImpersonationEventType = "BREAK_GLASS_ACTION"
	EventImpersonationExpired ImpersonationEventType = "IMPERSONATION_EXPIRED"
)

// Admin impersonation roles.
const (
	RoleGlobalAdmin          = "global_admin"
	RoleGlobalOps            = "global_ops"
	RoleHelpdesk             = "helpdesk"
	RoleProfessionalServices = "professional_services"
)

// MaxImpersonationDuration is the hard server-side cap on any impersonation window.
const MaxImpersonationDuration = 2 * time.Hour

// HelpdeskMaxDuration is the hard cap for helpdesk impersonation sessions.
const HelpdeskMaxDuration = 30 * time.Minute

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

	// ScopeKind narrows the impersonation to a subset of the target tenant.
	// Allowed values depend on the admin role (see ImpersonationPolicy).
	ScopeKind string

	// ScopeID is the concrete UUID of the scoped resource when ScopeKind is
	// instance, product, or datasource. Empty for tenant-wide access.
	ScopeID uuid.UUID

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
	AdminRole       string
	TargetTenantID  uuid.UUID
	Reason          string
	TicketReference string
	Mode            ImpersonationMode
	Duration        time.Duration
	ExpiresAt       time.Time
	IPAddress       string
	UserAgent       string
	// ScopeKind restricts the impersonation to a subset of the target tenant.
	// Defaults to "tenant" (full access); narrower scopes are "instance",
	// "product", or "datasource". The END audit row carries this for completeness
	// even though the START row is the source of truth for the audit invariant.
	ScopeKind string
	ScopeID   uuid.UUID
}

// ImpersonationAction is the synchronous micro-audit record written for each
// Business Object state change performed during an impersonation window.
type ImpersonationAction struct {
	ImpersonationID uuid.UUID
	TargetTenantID  uuid.UUID
	BOKey           string
	BOInstanceID    string
	StateTransition string
	PayloadSnapshot []byte
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

	// AdminRole is the role used to initiate the impersonation session.
	AdminRole string `json:"admin_role"`

	// TenantID is the target tenant's UUID.
	// This is what downstream middleware reads — identical to a normal user's token.
	TenantID string `json:"tenant_id"`

	// ImpersonationActive signals to middleware that this is an impersonation context.
	ImpersonationActive bool `json:"impersonation_active"`

	// SessionID links back to platform_admin_audit.
	SessionID string `json:"session_id"`

	// Scope narrows impersonation to a subset of the target tenant.
	ScopeKind string `json:"scope_kind"` // "tenant" | "instance" | "product" | "datasource"
	ScopeID   string `json:"scope_id"`   // UUID of the scoped resource; empty = tenant-wide

	// RealRoles preserves the admin's actual roles so downstream auth can distinguish
	// global_admin vs global_ops even during impersonation.
	RealRoles []string `json:"real_roles"`

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
	// The session parameter carries all fields including scope_kind/scope_id and the
	// original mode (which is recovered from the matching START row to preserve
	// the audit invariant "the END row reflects the original session mode").
	LogEnd(ctx context.Context, session ImpersonationSession) error

	// LogBreakGlassAction records a state-changing operation performed under break_glass mode.
	// Deprecated: use LogImpersonationAction for per-BO micro-audit.
	LogBreakGlassAction(ctx context.Context, sessionID uuid.UUID, adminUserID string, targetTenantID uuid.UUID, detail map[string]any) error

	// LogImpersonationAction writes a per-BO action record inside the caller's
	// transaction. If tx is nil the implementation falls back to its own pool,
	// but callers performing state mutations should always pass the active tx
	// so the audit row commits atomically with the business change.
	LogImpersonationAction(ctx context.Context, tx *sql.Tx, action ImpersonationAction) error

	// ListExpiredActiveSessions returns START rows that have no matching END row
	// AND whose expires_at is in the past. Used by the background sweeper to write
	// IMPERSONATION_EXPIRED audit rows for sessions that the client never exited.
	ListExpiredActiveSessions(ctx context.Context) ([]ImpersonationSession, error)

	// LogExpired writes an IMPERSONATION_EXPIRED row for a session whose expires_at
	// has passed without the client calling DELETE. Written by the background sweeper.
	LogExpired(ctx context.Context, session ImpersonationSession) error
}

// ============================================================================
// Role-based impersonation policy
// ============================================================================

// ImpersonationPolicy encodes the governance matrix for the three supported
// impersonation roles: global_admin, helpdesk, and professional_services.
// The zero value is safe and uses the default matrix.
type ImpersonationPolicy struct {
	// ProfessionalServicesBreakGlassBOKeys lists the business object keys for
	// which professional_services may use break_glass mode. Global admins have
	// no such restriction; helpdesk is never allowed break_glass.
	ProfessionalServicesBreakGlassBOKeys []string
}

// ResolveAdminRole picks the most privileged impersonation role from the
// caller's JWT roles. Priority: global_admin > global_ops > professional_services > helpdesk.
func (p ImpersonationPolicy) ResolveAdminRole(roles []string) string {
	for _, r := range roles {
		if r == RoleGlobalAdmin {
			return RoleGlobalAdmin
		}
	}
	for _, r := range roles {
		if r == RoleGlobalOps {
			return RoleGlobalOps
		}
	}
	for _, r := range roles {
		if r == RoleProfessionalServices {
			return RoleProfessionalServices
		}
	}
	for _, r := range roles {
		if r == RoleHelpdesk {
			return RoleHelpdesk
		}
	}
	return ""
}

// CanImpersonate reports whether the supplied role is allowed to assume a tenant context.
func (p ImpersonationPolicy) CanImpersonate(role string) bool {
	switch role {
	case RoleGlobalAdmin, RoleGlobalOps, RoleHelpdesk, RoleProfessionalServices:
		return true
	default:
		return false
	}
}

// AllowedScopes returns the scope kinds permitted for a role.
func (p ImpersonationPolicy) AllowedScopes(role string) []string {
	switch role {
	case RoleGlobalAdmin, RoleGlobalOps, RoleProfessionalServices:
		// Professional services administers the tenant, so it receives the same
		// scope latitude as global admins. Helpdesk remains more narrowly scoped.
		return []string{ScopeTenant, ScopeInstance, ScopeProduct, ScopeDatasource}
	case RoleHelpdesk:
		return []string{ScopeTenant, ScopeInstance}
	default:
		return nil
	}
}

// AllowedModes returns the impersonation modes permitted for a role.
func (p ImpersonationPolicy) AllowedModes(role string) []ImpersonationMode {
	switch role {
	case RoleGlobalAdmin, RoleGlobalOps, RoleProfessionalServices:
		return []ImpersonationMode{ModeReadOnly, ModeBreakGlass}
	case RoleHelpdesk:
		return []ImpersonationMode{ModeReadOnly}
	default:
		return nil
	}
}

// MaxDuration returns the hard cap for an impersonation session for a role.
func (p ImpersonationPolicy) MaxDuration(role string) time.Duration {
	switch role {
	case RoleHelpdesk:
		return HelpdeskMaxDuration
	default:
		return MaxImpersonationDuration
	}
}

// RequiresTicket reports whether a ticket reference is mandatory for the role/mode.
func (p ImpersonationPolicy) RequiresTicket(role string, mode ImpersonationMode) bool {
	if mode == ModeBreakGlass {
		return true
	}
	// Helpdesk and professional_services must always provide a ticket, even in read_only.
	return role == RoleHelpdesk || role == RoleProfessionalServices
}

// CanBreakGlassForBO reports whether the role may perform break_glass on the
// given business object key. Global admins and professional_services are
// unrestricted (the latter because it administers the tenant for the duration
// of the session). Helpdesk is never allowed break_glass. If a deployment wants
// to restrict professional_services further, populate ProfessionalServicesBreakGlassBOKeys.
func (p ImpersonationPolicy) CanBreakGlassForBO(role string, boKey string) bool {
	switch role {
	case RoleGlobalAdmin, RoleGlobalOps:
		return true
	case RoleHelpdesk:
		return false
	case RoleProfessionalServices:
		// Unrestricted by default so professional_services can administer any BO.
		// An explicit allow-list narrows the set when desired.
		if len(p.ProfessionalServicesBreakGlassBOKeys) == 0 {
			return true
		}
		for _, allowed := range p.ProfessionalServicesBreakGlassBOKeys {
			if allowed == boKey {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ============================================================================
// ContextExchangeService
// ============================================================================

// ContextExchangeService orchestrates the safe, auditable transition of an
// authorized platform operator into an explicit, isolated tenant context.
//
// Design contract (Zero-Tolerance Security Mandate):
//  1. Admin must hold one of the supported impersonation roles in their primary JWT.
//  2. TargetTenantID must be a valid, non-nil UUID.
//  3. Audit log write MUST succeed before any token is issued. Fail = abort.
//  4. Issued token carries a CONCRETE tenant UUID — never a wildcard.
//  5. Maximum session duration is capped per role by ImpersonationPolicy.
type ContextExchangeService struct {
	audit  ImpersonationAuditLogger
	policy ImpersonationPolicy
}

// AuditDB returns the underlying *sql.DB of the audit logger, when the logger is
// backed by PlatformAdminAuditLogger. Returns nil otherwise. This is the only
// way handlers in other packages can run read-only audit queries (e.g. listing
// active sessions) without going through the audit-logger interface.
func (s *ContextExchangeService) AuditDB() *sql.DB {
	if s == nil || s.audit == nil {
		return nil
	}
	pa, ok := s.audit.(*PlatformAdminAuditLogger)
	if !ok || pa == nil {
		return nil
	}
	return pa.db
}

// LookupSessionTenant fetches the target tenant ID for a session from the audit log.
// Used by ExitContext to recover the target tenant even after the impersonation
// token has been revoked client-side (the audit row is the durable record).
func (s *ContextExchangeService) LookupSessionTenant(ctx context.Context, sessionID uuid.UUID) (uuid.UUID, error) {
	if s.audit == nil {
		return uuid.Nil, errors.New("audit logger is not configured")
	}
	pa, ok := s.audit.(*PlatformAdminAuditLogger)
	if !ok {
		return uuid.Nil, errors.New("audit logger does not support session tenant lookup")
	}
	if pa.db == nil {
		return uuid.Nil, errors.New("audit database is not configured")
	}
	var targetTenantID uuid.UUID
	err := pa.db.QueryRowContext(ctx,
		`SELECT target_tenant_id FROM platform_admin_audit
		 WHERE session_id = $1 AND event_type = $2 LIMIT 1`,
		sessionID, EventImpersonationStart,
	).Scan(&targetTenantID)
	if err != nil {
		return uuid.Nil, err
	}
	return targetTenantID, nil
}

// NewContextExchangeService constructs a ContextExchangeService.
// If policy is the zero value, the default role matrix is used.
func NewContextExchangeService(audit ImpersonationAuditLogger, policy ImpersonationPolicy) *ContextExchangeService {
	if audit == nil {
		panic("security: ContextExchangeService requires a non-nil ImpersonationAuditLogger")
	}
	return &ContextExchangeService{audit: audit, policy: policy}
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
	// ── 1. Resolve role and enforce role-based governance ───────────────────
	adminRole := s.policy.ResolveAdminRole(adminRoles)
	if adminRole == "" || !s.policy.CanImpersonate(adminRole) {
		return nil, fmt.Errorf(
			"security violation: user %s lacks an authorized impersonation role — impersonation denied",
			adminUserID,
		)
	}

	if req.TargetTenantID == uuid.Nil {
		return nil, errors.New("invalid operation: target_tenant_id cannot be nil")
	}

	if len(req.Reason) < 10 {
		return nil, errors.New("governance violation: reason must be at least 10 characters")
	}

	// ── 2. Enforce mode restrictions per role ───────────────────────────────
	mode := req.Mode
	if mode == "" {
		mode = ModeReadOnly // always default to least privilege
	}
	if !containsImpersonationMode(s.policy.AllowedModes(adminRole), mode) {
		return nil, fmt.Errorf(
			"governance violation: role %s is not permitted to use impersonation mode %s",
			adminRole, mode,
		)
	}

	if s.policy.RequiresTicket(adminRole, mode) && req.TicketReference == "" {
		return nil, fmt.Errorf(
			"governance violation: ticket_reference is mandatory for role %s in mode %s",
			adminRole, mode,
		)
	}

	// ── 3. Enforce scope restrictions per role ──────────────────────────────
	scopeKind := req.ScopeKind
	if scopeKind == "" {
		scopeKind = ScopeTenant
	}
	if !containsString(s.policy.AllowedScopes(adminRole), scopeKind) {
		return nil, fmt.Errorf(
			"governance violation: role %s is not permitted to use scope_kind %s",
			adminRole, scopeKind,
		)
	}

	// ── 4. Resolve + cap the session window per role ────────────────────────
	duration := req.Duration
	if duration <= 0 {
		duration = 30 * time.Minute // safe default
	}
	maxDuration := s.policy.MaxDuration(adminRole)
	if duration > maxDuration {
		duration = maxDuration // hard cap — never negotiable
	}

	sessionID := uuid.New()
	expiresAt := time.Now().UTC().Add(duration)

	session := ImpersonationSession{
		SessionID:       sessionID,
		AdminUserID:     adminUserID,
		AdminEmail:      adminEmail,
		AdminRole:       adminRole,
		TargetTenantID:  req.TargetTenantID,
		Reason:          req.Reason,
		TicketReference: req.TicketReference,
		Mode:            mode,
		Duration:        duration,
		ExpiresAt:       expiresAt,
		IPAddress:       req.IPAddress,
		UserAgent:       req.UserAgent,
		ScopeKind:       scopeKind,
		ScopeID:         req.ScopeID,
	}

	// ── 5. Synchronous audit write BEFORE token issuance ────────────────────
	// Critical invariant: if this write fails, no token is issued.
	if err := s.audit.LogStart(ctx, session); err != nil {
		// Fail loudly and securely — do NOT issue a token if audit fails.
		return nil, fmt.Errorf(
			"critical invariant failure: could not commit audit trail record — impersonation aborted: %w",
			err,
		)
	}

	// ── 6. Sign and issue the context token ─────────────────────────────────
	tokenStr, err := signImpersonationToken(session, adminRoles)
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
	session ImpersonationSession,
) error {
	return s.audit.LogEnd(ctx, session)
}

// SubjectAttributes identifies an operator requesting impersonation authority.
type SubjectAttributes struct {
	UserID       string
	OperatorRole string
}

// queryRowContext matches the subset of *sql.DB and *sql.Tx used for lease
// lookups. Both concrete types implement it, so VerifyImpersonationAuthority can
// run inside an existing transaction or fall back to the audit database pool.
type queryRowContext interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Boundary errors returned by VerifyImpersonationAuthority and the transition
// guard helpers used by BO state-machine handlers.
var (
	// ErrImpersonationLeaseViolation is returned when helpdesk or
	// professional_services operators try to access a tenant for which they have
	// no active staff_tenant_assignments lease.
	ErrImpersonationLeaseViolation = errors.New("impersonation lease violation: no active assignment for target tenant")

	// ErrImpersonationWriteForbidden is returned when an impersonation session
	// that is not in break_glass mode (or whose role cannot break glass on the
	// target BO) attempts a state-changing operation.
	ErrImpersonationWriteForbidden = errors.New("impersonation write forbidden: operator role or session mode does not permit state mutation")
)

// VerifyImpersonationAuthority validates that subject is allowed to assume a
// tenant context for the given target tenant.
//
// Global admins and global ops are permitted directly. Helpdesk and
// professional_services operators must hold an active row in
// staff_tenant_assignments that matches both their user ID and the exact target
// tenant, with expires_at in the future.
//
// The optional tx lets callers run the authority check inside an existing
// business transaction so the lease state is evaluated at the same snapshot as
// the operation it protects.
func (s *ContextExchangeService) VerifyImpersonationAuthority(ctx context.Context, tx *sql.Tx, subject SubjectAttributes, targetTenantID uuid.UUID) error {
	if targetTenantID == uuid.Nil {
		return errors.New("invalid operation: target_tenant_id cannot be nil")
	}
	if subject.UserID == "" {
		return errors.New("invalid operation: subject user_id cannot be empty")
	}
	if !s.policy.CanImpersonate(subject.OperatorRole) {
		return fmt.Errorf("security violation: role %s is not permitted to impersonate", subject.OperatorRole)
	}

	switch subject.OperatorRole {
	case RoleGlobalAdmin, RoleGlobalOps:
		return nil
	case RoleHelpdesk, RoleProfessionalServices:
		// These roles require an active lease; continue below.
	default:
		return fmt.Errorf("security violation: role %s is not permitted to impersonate", subject.OperatorRole)
	}

	var executor queryRowContext = s.AuditDB()
	if tx != nil {
		executor = tx
	}
	if executor == nil {
		return errors.New("impersonation authority check requires a database connection")
	}

	const q = `
		SELECT COUNT(*)
		FROM staff_tenant_assignments
		WHERE operator_user_id = $1
		  AND target_tenant_id = $2
		  AND expires_at > CURRENT_TIMESTAMP
	`
	var count int
	if err := executor.QueryRowContext(ctx, q, subject.UserID, targetTenantID.String()).Scan(&count); err != nil {
		return fmt.Errorf("impersonation authority check failed: %w", err)
	}
	if count == 0 {
		return ErrImpersonationLeaseViolation
	}
	return nil
}

// ============================================================================
// Token Signing (HMAC-SHA256 — internal platform token, not a Keycloak token)
// ============================================================================

// signImpersonationToken creates a base64-encoded HMAC-SHA256 signed payload.
// The secret is taken from IMPERSONATION_TOKEN_SECRET env var, falling back
// to JWT_SECRET as a secondary option. Both must be set in production.
func signImpersonationToken(session ImpersonationSession, realRoles []string) (string, error) {
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
		AdminRole:           session.AdminRole,
		TenantID:            session.TargetTenantID.String(),
		ImpersonationActive: true,
		SessionID:           session.SessionID.String(),
		ScopeKind:           session.ScopeKind,
		ScopeID:             session.ScopeID.String(),
		RealRoles:           realRoles,
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

// containsImpersonationMode reports whether modes contains the requested mode.
func containsImpersonationMode(modes []ImpersonationMode, mode ImpersonationMode) bool {
	for _, m := range modes {
		if m == mode {
			return true
		}
	}
	return false
}
