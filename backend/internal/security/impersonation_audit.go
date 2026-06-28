package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PlatformAdminAuditLogger is the concrete synchronous OLTP implementation
// of ImpersonationAuditLogger. It writes directly to platform_admin_audit
// using the provided *sql.DB connection pool.
//
// CRITICAL: All methods write synchronously. There are no goroutines, channels,
// or message queues involved. If the DB write fails, the caller receives the
// error and must abort the triggering operation.
type PlatformAdminAuditLogger struct {
	db *sql.DB
}

// NewPlatformAdminAuditLogger constructs a PlatformAdminAuditLogger.
// The db must already be open and connected.
func NewPlatformAdminAuditLogger(db *sql.DB) *PlatformAdminAuditLogger {
	if db == nil {
		panic("security: PlatformAdminAuditLogger requires a non-nil *sql.DB")
	}
	return &PlatformAdminAuditLogger{db: db}
}

// LogStart writes an IMPERSONATION_START record to platform_admin_audit.
// This is called BEFORE the context token is issued. If this fails, the
// ContextExchangeService will abort and return an error — no token is issued.
func (l *PlatformAdminAuditLogger) LogStart(ctx context.Context, session ImpersonationSession) error {
	const q = `
		INSERT INTO platform_admin_audit (
			id,
			event_type,
			mode,
			admin_user_id,
			admin_email,
			target_tenant_id,
			session_id,
			reason,
			ticket_reference,
			duration_minutes,
			expires_at,
			ip_address,
			user_agent,
			action_detail,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)`

	durationMinutes := int(session.Duration.Minutes())

	_, err := l.db.ExecContext(
		ctx,
		q,
		uuid.New(),                    // $1  id
		EventImpersonationStart,       // $2  event_type
		string(session.Mode),          // $3  mode
		session.AdminUserID,           // $4  admin_user_id
		session.AdminEmail,            // $5  admin_email
		session.TargetTenantID,        // $6  target_tenant_id
		session.SessionID,             // $7  session_id
		session.Reason,                // $8  reason
		nullableStr(session.TicketReference), // $9  ticket_reference
		durationMinutes,               // $10 duration_minutes
		session.ExpiresAt,             // $11 expires_at
		nullableStr(session.IPAddress), // $12 ip_address
		nullableStr(session.UserAgent), // $13 user_agent
		nil,                           // $14 action_detail (nil for START events)
		time.Now().UTC(),              // $15 created_at
	)
	if err != nil {
		return fmt.Errorf("platform_admin_audit: failed to write IMPERSONATION_START for session %s: %w",
			session.SessionID, err)
	}
	return nil
}

// LogEnd writes an IMPERSONATION_END record to platform_admin_audit.
// The original session mode is recovered from the matching START row in the same
// transaction so the audit trail preserves whether the session was read-only or
// break-glass. Scope fields are propagated to the END row for completeness.
func (l *PlatformAdminAuditLogger) LogEnd(
	ctx context.Context,
	session ImpersonationSession,
) error {
	// Recover the original session mode from the START row.
	var originalMode string
	err := l.db.QueryRowContext(
		ctx,
		`SELECT mode FROM platform_admin_audit WHERE session_id = $1 AND event_type = $2 LIMIT 1`,
		session.SessionID, EventImpersonationStart,
	).Scan(&originalMode)
	if err != nil {
		// If we can't find the START row, fall back to the session mode.
		originalMode = string(session.Mode)
	}

	const q = `
		INSERT INTO platform_admin_audit (
			id,
			event_type,
			mode,
			admin_user_id,
			admin_email,
			target_tenant_id,
			session_id,
			reason,
			scope_kind,
			scope_id,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err = l.db.ExecContext(
		ctx,
		q,
		uuid.New(),                    // $1  id
		EventImpersonationEnd,         // $2  event_type
		originalMode,                  // $3  mode — recovered from START row
		session.AdminUserID,           // $4  admin_user_id
		session.AdminEmail,            // $5  admin_email
		session.TargetTenantID,        // $6  target_tenant_id
		session.SessionID,             // $7  session_id
		"Session terminated",          // $8  reason
		nullableStr(session.ScopeKind), // $9  scope_kind
		nullableUUID(session.ScopeID),  // $10 scope_id
		time.Now().UTC(),               // $11 created_at
	)
	if err != nil {
		return fmt.Errorf("platform_admin_audit: failed to write IMPERSONATION_END for session %s: %w",
			session.SessionID, err)
	}
	return nil
}

// nullableUUID converts a uuid.UUID to a nilable value for nullable UUID columns.
// We can't use sql.NullString directly because the postgres driver doesn't
// handle uuid.Nil → NULL via plain interface{} conversion.
func nullableUUID(id uuid.UUID) interface{} {
	if id == uuid.Nil {
		return nil
	}
	return id
}

// LogBreakGlassAction writes a BREAK_GLASS_ACTION record to platform_admin_audit.
// This must be called by every handler that performs a state-changing operation
// while an impersonation session is active in break_glass mode.
func (l *PlatformAdminAuditLogger) LogBreakGlassAction(
	ctx context.Context,
	sessionID uuid.UUID,
	adminUserID string,
	targetTenantID uuid.UUID,
	detail map[string]any,
) error {
	detailJSON, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("platform_admin_audit: failed to marshal action_detail: %w", err)
	}

	const q = `
		INSERT INTO platform_admin_audit (
			id,
			event_type,
			mode,
			admin_user_id,
			admin_email,
			target_tenant_id,
			session_id,
			reason,
			action_detail,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)`

	_, err = l.db.ExecContext(
		ctx,
		q,
		uuid.New(),           // $1  id
		EventBreakGlassAction, // $2  event_type
		string(ModeBreakGlass), // $3  mode
		adminUserID,          // $4  admin_user_id
		"",                   // $5  admin_email
		targetTenantID,       // $6  target_tenant_id
		sessionID,            // $7  session_id
		"Break-glass action", // $8  reason
		detailJSON,           // $9  action_detail
		time.Now().UTC(),     // $10 created_at
	)
	if err != nil {
		return fmt.Errorf("platform_admin_audit: failed to write BREAK_GLASS_ACTION for session %s: %w",
			sessionID, err)
	}
	return nil
}

// nullableStr converts an empty string to nil for nullable TEXT columns.
func nullableStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ListExpiredActiveSessions returns the START rows of impersonation sessions
// that have no matching END row AND whose expires_at is already in the past.
// Used by the background sweeper to write IMPERSONATION_EXPIRED audit rows so
// the audit log stays complete even when clients go offline without exiting.
//
// Bounded by a hard limit (100) to prevent memory blowups if the sweeper is
// offline for a long time and a backlog accumulates.
func (l *PlatformAdminAuditLogger) ListExpiredActiveSessions(ctx context.Context) ([]ImpersonationSession, error) {
	const q = `
		SELECT s.session_id, s.admin_user_id, s.admin_email,
		       s.target_tenant_id, s.mode, s.expires_at, s.created_at
		FROM platform_admin_audit s
		LEFT JOIN platform_admin_audit e
		  ON e.session_id = s.session_id AND e.event_type = $2
		WHERE s.event_type = $1
		  AND s.expires_at IS NOT NULL
		  AND s.expires_at < NOW()
		  AND e.id IS NULL
		ORDER BY s.expires_at ASC
		LIMIT 100
	`

	rows, err := l.db.QueryContext(ctx, q,
		EventImpersonationStart,
		EventImpersonationEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("platform_admin_audit: failed to query expired sessions: %w", err)
	}
	defer rows.Close()

	out := make([]ImpersonationSession, 0, 16)
	for rows.Next() {
		var (
			sid, adminID, adminEmail, tenantID, mode string
			expiresAt                               sql.NullTime
			createdAt                               time.Time
		)
		if err := rows.Scan(&sid, &adminID, &adminEmail, &tenantID, &mode, &expiresAt, &createdAt); err != nil {
			return nil, fmt.Errorf("platform_admin_audit: scan failed: %w", err)
		}
		sessionUUID, uuidErr := uuid.Parse(sid)
		if uuidErr != nil {
			continue
		}
		tenantUUID, _ := uuid.Parse(tenantID)
		parsedMode := ImpersonationMode(mode)
		var expiresTime time.Time
		if expiresAt.Valid {
			expiresTime = expiresAt.Time
		}
		out = append(out, ImpersonationSession{
			SessionID:      sessionUUID,
			AdminUserID:    adminID,
			AdminEmail:     adminEmail,
			TargetTenantID: tenantUUID,
			Mode:           parsedMode,
			ExpiresAt:      expiresTime,
			// StartedAt is captured via created_at; LogEnd handles the rest.
		})
	}
	return out, nil
}

// LogExpired writes an IMPERSONATION_EXPIRED row for a session that the client
// never explicitly exited. The event_type is new ("IMPERSONATION_EXPIRED") but
// we reuse the existing columns; the "reason" field carries "Session expired
// without explicit exit" so audit consumers can distinguish from manual exits.
func (l *PlatformAdminAuditLogger) LogExpired(ctx context.Context, session ImpersonationSession) error {
	const q = `
		INSERT INTO platform_admin_audit (
			id,
			event_type,
			mode,
			admin_user_id,
			admin_email,
			target_tenant_id,
			session_id,
			reason,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err := l.db.ExecContext(
		ctx,
		q,
		uuid.New(),                          // $1  id
		EventImpersonationExpired,           // $2  event_type
		string(session.Mode),                // $3  mode — preserve original
		session.AdminUserID,                 // $4  admin_user_id
		session.AdminEmail,                  // $5  admin_email
		session.TargetTenantID,              // $6  target_tenant_id
		session.SessionID,                   // $7  session_id
		"Session expired without explicit exit", // $8  reason
		time.Now().UTC(),                    // $9  created_at
	)
	if err != nil {
		return fmt.Errorf("platform_admin_audit: failed to write IMPERSONATION_EXPIRED for session %s: %w",
			session.SessionID, err)
	}
	return nil
}
