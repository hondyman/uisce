package security

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// hookRecordingAudit captures LogImpersonationAction calls.
type hookRecordingAudit struct {
	lastAction ImpersonationAction
	returnErr  error
}

func (h *hookRecordingAudit) LogStart(_ context.Context, _ ImpersonationSession) error          { return nil }
func (h *hookRecordingAudit) LogEnd(_ context.Context, _ ImpersonationSession) error            { return nil }
func (h *hookRecordingAudit) LogBreakGlassAction(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ map[string]any) error {
	return nil
}
func (h *hookRecordingAudit) LogImpersonationAction(_ context.Context, _ *sql.Tx, action ImpersonationAction) error {
	h.lastAction = action
	return h.returnErr
}
func (h *hookRecordingAudit) ListExpiredActiveSessions(_ context.Context) ([]ImpersonationSession, error) {
	return nil, nil
}
func (h *hookRecordingAudit) LogExpired(_ context.Context, _ ImpersonationSession) error { return nil }

func TestLogBOActionIfImpersonating_NoOpWhenNotImpersonating(t *testing.T) {
	audit := &hookRecordingAudit{}
	ctx := context.Background()

	err := LogBOActionIfImpersonating(ctx, ImpersonationPolicy{}, audit, nil, "trade", "inst-1", "UPDATE", []byte(`{}`))
	if err != nil {
		t.Fatalf("expected no-op, got error: %v", err)
	}
	if audit.lastAction.BOKey != "" {
		t.Error("expected no audit action when not impersonating")
	}
}

func TestLogBOActionIfImpersonating_LogsWhenImpersonating(t *testing.T) {
	audit := &hookRecordingAudit{}
	sessionID := uuid.New()
	tenantID := uuid.New()

	secCtx := &Context{
		TenantID:               tenantID.String(),
		ImpersonationActive:    true,
		ImpersonationSessionID: sessionID.String(),
		ImpersonationMode:      string(ModeReadOnly),
		ImpersonationAdminRole: RoleGlobalAdmin,
	}
	ctx := WithContext(context.Background(), secCtx)

	payload := []byte(`{"field":"value"}`)
	err := LogBOActionIfImpersonating(ctx, ImpersonationPolicy{}, audit, nil, "trade", "inst-1", "UPDATE", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if audit.lastAction.ImpersonationID != sessionID {
		t.Errorf("ImpersonationID = %v, want %v", audit.lastAction.ImpersonationID, sessionID)
	}
	if audit.lastAction.TargetTenantID != tenantID {
		t.Errorf("TargetTenantID = %v, want %v", audit.lastAction.TargetTenantID, tenantID)
	}
	if audit.lastAction.BOKey != "trade" {
		t.Errorf("BOKey = %q, want trade", audit.lastAction.BOKey)
	}
	if audit.lastAction.BOInstanceID != "inst-1" {
		t.Errorf("BOInstanceID = %q, want inst-1", audit.lastAction.BOInstanceID)
	}
	if string(audit.lastAction.PayloadSnapshot) != string(payload) {
		t.Errorf("PayloadSnapshot = %q, want %q", audit.lastAction.PayloadSnapshot, payload)
	}
}

func TestLogBOActionIfImpersonating_UsesProvidedTx(t *testing.T) {
	audit := &hookRecordingAudit{}
	sessionID := uuid.New()
	tenantID := uuid.New()

	secCtx := &Context{
		TenantID:               tenantID.String(),
		ImpersonationActive:    true,
		ImpersonationSessionID: sessionID.String(),
		ImpersonationMode:      string(ModeReadOnly),
		ImpersonationAdminRole: RoleGlobalAdmin,
	}
	ctx := WithContext(context.Background(), secCtx)

	// A non-nil tx pointer proves the hook forwards tx to the audit logger.
	var fakeTx *sql.Tx
	err := LogBOActionIfImpersonating(ctx, ImpersonationPolicy{}, audit, fakeTx, "trade", "inst-1", "UPDATE", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if audit.lastAction.ImpersonationID != sessionID {
		t.Errorf("ImpersonationID = %v, want %v", audit.lastAction.ImpersonationID, sessionID)
	}
}

func TestLogBOActionIfImpersonating_AbortsOnAuditFailure(t *testing.T) {
	audit := &hookRecordingAudit{returnErr: errors.New("audit failed")}
	sessionID := uuid.New()
	tenantID := uuid.New()

	secCtx := &Context{
		TenantID:               tenantID.String(),
		ImpersonationActive:    true,
		ImpersonationSessionID: sessionID.String(),
		ImpersonationMode:      string(ModeReadOnly),
		ImpersonationAdminRole: RoleGlobalAdmin,
	}
	ctx := WithContext(context.Background(), secCtx)

	err := LogBOActionIfImpersonating(ctx, ImpersonationPolicy{}, audit, nil, "trade", "inst-1", "UPDATE", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error when audit fails, got nil")
	}
}

func TestLogBOActionIfImpersonating_BreakGlassRoleRestriction(t *testing.T) {
	audit := &hookRecordingAudit{}
	policy := ImpersonationPolicy{ProfessionalServicesBreakGlassBOKeys: []string{"catalog_mapping"}}

	sessionID := uuid.New()
	tenantID := uuid.New()

	secCtx := &Context{
		TenantID:               tenantID.String(),
		ImpersonationActive:    true,
		ImpersonationSessionID: sessionID.String(),
		ImpersonationMode:      string(ModeBreakGlass),
		ImpersonationAdminRole: RoleProfessionalServices,
	}
	ctx := WithContext(context.Background(), secCtx)

	// Allowed BO.
	err := LogBOActionIfImpersonating(ctx, policy, audit, nil, "catalog_mapping", "inst-1", "UPDATE", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error for allowed BO: %v", err)
	}

	// Disallowed BO.
	err = LogBOActionIfImpersonating(ctx, policy, audit, nil, "trade", "inst-1", "UPDATE", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for disallowed break_glass BO, got nil")
	}
}
