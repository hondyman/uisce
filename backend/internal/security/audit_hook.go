package security

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// LogBOActionIfImpersonating writes a synchronous micro-audit record when the
// request context carries an active impersonation session and the action is a
// Business Object state change. It should be called inside the transaction that
// mutates the BO so the audit row commits atomically.
//
// The function is a no-op (returns nil) when impersonation is not active, so
// callers may invoke it unconditionally for every BO transition.
func LogBOActionIfImpersonating(
	ctx context.Context,
	policy ImpersonationPolicy,
	audit ImpersonationAuditLogger,
	tx *sql.Tx,
	boKey string,
	boInstanceID string,
	transition string,
	payload []byte,
) error {
	if audit == nil {
		return errors.New("impersonation audit logger is not configured")
	}

	secCtx, ok := FromContext(ctx)
	if !ok || secCtx == nil || !secCtx.ImpersonationActive {
		return nil
	}

	if secCtx.ImpersonationSessionID == "" {
		return errors.New("impersonation active but session_id missing from security context")
	}

	// Break-glass actions are only audited when the role is allowed to perform
	// them on the target BO. Read-only sessions are audited for visibility but
	// should not be making state-changing calls in the first place.
	if secCtx.ImpersonationMode == string(ModeBreakGlass) {
		if !policy.CanBreakGlassForBO(secCtx.ImpersonationAdminRole, boKey) {
			return fmt.Errorf(
				"security violation: role %s is not permitted break_glass on business object %s",
				secCtx.ImpersonationAdminRole, boKey,
			)
		}
	}

	impersonationID, err := uuid.Parse(secCtx.ImpersonationSessionID)
	if err != nil {
		return fmt.Errorf("invalid impersonation session_id in context: %w", err)
	}

	tenantID, err := uuid.Parse(secCtx.TenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant_id in security context: %w", err)
	}

	action := ImpersonationAction{
		ImpersonationID: impersonationID,
		TargetTenantID:  tenantID,
		BOKey:           boKey,
		BOInstanceID:    boInstanceID,
		StateTransition: transition,
		PayloadSnapshot: payload,
	}

	if err := audit.LogImpersonationAction(ctx, tx, action); err != nil {
		return fmt.Errorf("aborting transaction: impersonation action audit failed: %w", err)
	}
	return nil
}
