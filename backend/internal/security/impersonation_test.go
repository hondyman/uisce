package security

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

// recordingAudit is a test double that captures session and action records.
type recordingAudit struct {
	startCalled  bool
	actionCalled bool
	lastSession  ImpersonationSession
	lastAction   ImpersonationAction
}

func (r *recordingAudit) LogStart(_ context.Context, session ImpersonationSession) error {
	r.startCalled = true
	r.lastSession = session
	return nil
}

func (r *recordingAudit) LogEnd(_ context.Context, _ ImpersonationSession) error {
	return nil
}

func (r *recordingAudit) LogBreakGlassAction(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ map[string]any) error {
	return nil
}

func (r *recordingAudit) LogImpersonationAction(_ context.Context, _ *sql.Tx, action ImpersonationAction) error {
	r.actionCalled = true
	r.lastAction = action
	return nil
}

func (r *recordingAudit) ListExpiredActiveSessions(_ context.Context) ([]ImpersonationSession, error) {
	return nil, nil
}

func (r *recordingAudit) LogExpired(_ context.Context, _ ImpersonationSession) error {
	return nil
}

func TestImpersonationPolicy_ResolveAdminRole(t *testing.T) {
	p := ImpersonationPolicy{}

	cases := []struct {
		roles []string
		want  string
	}{
		{[]string{RoleHelpdesk, RoleGlobalAdmin}, RoleGlobalAdmin},
		{[]string{RoleGlobalOps, RoleHelpdesk}, RoleGlobalOps},
		{[]string{RoleProfessionalServices}, RoleProfessionalServices},
		{[]string{RoleHelpdesk}, RoleHelpdesk},
		{[]string{"tenant_admin"}, ""},
	}

	for _, tc := range cases {
		got := p.ResolveAdminRole(tc.roles)
		if got != tc.want {
			t.Errorf("ResolveAdminRole(%v) = %q, want %q", tc.roles, got, tc.want)
		}
	}
}

func TestImpersonationPolicy_CanImpersonate(t *testing.T) {
	p := ImpersonationPolicy{}

	for _, role := range []string{RoleGlobalAdmin, RoleGlobalOps, RoleHelpdesk, RoleProfessionalServices} {
		if !p.CanImpersonate(role) {
			t.Errorf("CanImpersonate(%q) = false, want true", role)
		}
	}

	if p.CanImpersonate("tenant_admin") {
		t.Error("CanImpersonate(\"tenant_admin\") = true, want false")
	}
}

func TestImpersonationPolicy_RoleMatrix(t *testing.T) {
	p := ImpersonationPolicy{}

	tests := []struct {
		role                   string
		wantScopes             []string
		wantModes              []ImpersonationMode
		wantMax                time.Duration
		readOnlyTicketRequired bool
	}{
		{RoleGlobalAdmin, []string{ScopeTenant, ScopeInstance, ScopeProduct, ScopeDatasource}, []ImpersonationMode{ModeReadOnly, ModeBreakGlass}, MaxImpersonationDuration, false},
		{RoleGlobalOps, []string{ScopeTenant, ScopeInstance, ScopeProduct, ScopeDatasource}, []ImpersonationMode{ModeReadOnly, ModeBreakGlass}, MaxImpersonationDuration, false},
		{RoleProfessionalServices, []string{ScopeTenant, ScopeInstance, ScopeProduct, ScopeDatasource}, []ImpersonationMode{ModeReadOnly, ModeBreakGlass}, MaxImpersonationDuration, true},
		{RoleHelpdesk, []string{ScopeTenant, ScopeInstance}, []ImpersonationMode{ModeReadOnly}, HelpdeskMaxDuration, true},
	}

	for _, tc := range tests {
		if got := p.AllowedScopes(tc.role); !stringSliceEqual(got, tc.wantScopes) {
			t.Errorf("AllowedScopes(%q) = %v, want %v", tc.role, got, tc.wantScopes)
		}
		if got := p.AllowedModes(tc.role); !modeSliceEqual(got, tc.wantModes) {
			t.Errorf("AllowedModes(%q) = %v, want %v", tc.role, got, tc.wantModes)
		}
		if got := p.MaxDuration(tc.role); got != tc.wantMax {
			t.Errorf("MaxDuration(%q) = %v, want %v", tc.role, got, tc.wantMax)
		}
		if got := p.RequiresTicket(tc.role, ModeReadOnly); got != tc.readOnlyTicketRequired {
			t.Errorf("RequiresTicket(%q, read_only) = %v, want %v", tc.role, got, tc.readOnlyTicketRequired)
		}
	}
}

func TestContextExchangeService_AssumeTenantContext_RoleRestrictions(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-unit-tests")

	audit := &recordingAudit{}
	svc := NewContextExchangeService(audit, ImpersonationPolicy{})

	target := uuid.New()
	baseReq := ImpersonationRequest{
		TargetTenantID:  target,
		Reason:          "support investigation for ticket 12345",
		TicketReference: "TICKET-12345",
		Mode:            ModeReadOnly,
		Duration:        15 * time.Minute,
	}

	// Helpdesk cannot use break_glass.
	_, err := svc.AssumeTenantContext(context.Background(), "helpdesk-user", "helpdesk@example.com", []string{RoleHelpdesk}, ImpersonationRequest{
		TargetTenantID:  target,
		Reason:          "support investigation for ticket 12345",
		TicketReference: "TICKET-12345",
		Mode:            ModeBreakGlass,
		Duration:        15 * time.Minute,
	})
	if err == nil {
		t.Fatal("expected error for helpdesk break_glass, got nil")
	}

	// Helpdesk requires a ticket even in read_only.
	_, err = svc.AssumeTenantContext(context.Background(), "helpdesk-user", "helpdesk@example.com", []string{RoleHelpdesk}, ImpersonationRequest{
		TargetTenantID: target,
		Reason:         "support investigation for ticket 12345",
		Mode:           ModeReadOnly,
		Duration:       15 * time.Minute,
	})
	if err == nil {
		t.Fatal("expected error for helpdesk without ticket, got nil")
	}

	// Helpdesk valid read_only session.
	token, err := svc.AssumeTenantContext(context.Background(), "helpdesk-user", "helpdesk@example.com", []string{RoleHelpdesk}, baseReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == nil {
		t.Fatal("expected token, got nil")
	}
	if !audit.startCalled {
		t.Fatal("expected audit LogStart to be called")
	}
	if audit.lastSession.AdminRole != RoleHelpdesk {
		t.Errorf("AdminRole = %q, want %q", audit.lastSession.AdminRole, RoleHelpdesk)
	}
	if audit.lastSession.ScopeKind != ScopeTenant {
		t.Errorf("ScopeKind = %q, want %q", audit.lastSession.ScopeKind, ScopeTenant)
	}

	// Helpdesk cannot request product scope.
	_, err = svc.AssumeTenantContext(context.Background(), "helpdesk-user", "helpdesk@example.com", []string{RoleHelpdesk}, ImpersonationRequest{
		TargetTenantID: target,
		Reason:         "support investigation for ticket 12345",
		TicketReference: "TICKET-12345",
		Mode:           ModeReadOnly,
		Duration:       15 * time.Minute,
		ScopeKind:      ScopeProduct,
	})
	if err == nil {
		t.Fatal("expected error for helpdesk product scope, got nil")
	}

	// Global admin can request break_glass without ticket? No, break_glass still needs ticket.
	_, err = svc.AssumeTenantContext(context.Background(), "admin-user", "admin@example.com", []string{RoleGlobalAdmin}, ImpersonationRequest{
		TargetTenantID: target,
		Reason:         "emergency remediation",
		Mode:           ModeBreakGlass,
		Duration:       15 * time.Minute,
	})
	if err == nil {
		t.Fatal("expected error for break_glass without ticket, got nil")
	}
}

func TestContextExchangeService_AssumeTenantContext_DurationCap(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-unit-tests")

	audit := &recordingAudit{}
	svc := NewContextExchangeService(audit, ImpersonationPolicy{})
	target := uuid.New()

	// Helpdesk duration capped at 30 minutes.
	_, err := svc.AssumeTenantContext(context.Background(), "helpdesk-user", "helpdesk@example.com", []string{RoleHelpdesk}, ImpersonationRequest{
		TargetTenantID:  target,
		Reason:          "support investigation for ticket 12345",
		TicketReference: "TICKET-12345",
		Mode:            ModeReadOnly,
		Duration:        2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if audit.lastSession.Duration != HelpdeskMaxDuration {
		t.Errorf("Duration = %v, want %v", audit.lastSession.Duration, HelpdeskMaxDuration)
	}
}

func TestImpersonationPolicy_CanBreakGlassForBO(t *testing.T) {
	// With an explicit allow-list, professional_services is restricted.
	restricted := ImpersonationPolicy{
		ProfessionalServicesBreakGlassBOKeys: []string{"catalog_mapping", "semantic_term"},
	}

	if !restricted.CanBreakGlassForBO(RoleGlobalAdmin, "trade") {
		t.Error("global_admin should be allowed break_glass on any BO")
	}
	if restricted.CanBreakGlassForBO(RoleHelpdesk, "catalog_mapping") {
		t.Error("helpdesk should never be allowed break_glass")
	}
	if !restricted.CanBreakGlassForBO(RoleProfessionalServices, "catalog_mapping") {
		t.Error("professional_services should be allowed break_glass on catalog_mapping")
	}
	if restricted.CanBreakGlassForBO(RoleProfessionalServices, "trade") {
		t.Error("professional_services should not be allowed break_glass on trade when allow-list is set")
	}

	// With no allow-list, professional_services is unrestricted for tenant administration.
	unrestricted := ImpersonationPolicy{}
	if !unrestricted.CanBreakGlassForBO(RoleProfessionalServices, "trade") {
		t.Error("professional_services should be allowed break_glass on any BO when no allow-list is configured")
	}
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func modeSliceEqual(a, b []ImpersonationMode) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
