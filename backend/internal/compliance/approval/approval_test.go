package approval

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/testsuite"
)

type mockResManager struct {
	mu          sync.Mutex
	parked      map[uuid.UUID]bool
	unparked    map[uuid.UUID]bool
	released    map[uuid.UUID]bool
}

func newMockResManager() *mockResManager {
	return &mockResManager{
		parked:   make(map[uuid.UUID]bool),
		unparked: make(map[uuid.UUID]bool),
		released: make(map[uuid.UUID]bool),
	}
}

func (m *mockResManager) ParkReservation(leaseID uuid.UUID, approvalTTL time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parked[leaseID] = true
	return nil
}

func (m *mockResManager) UnparkReservation(leaseID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unparked[leaseID] = true
	return nil
}

func (m *mockResManager) ReleaseReservation(leaseID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.released[leaseID] = true
	return nil
}

func TestOrderApprovalWorkflow_ApprovedBySignal(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockRM := newMockResManager()
	activities := ApprovalActivities{ResMgr: mockRM}
	env.RegisterActivity(activities.ParkReservationActivity)
	env.RegisterActivity(activities.ResolveReservationActivity)
	env.RegisterActivity(activities.NotifyComplianceOfficersActivity)
	env.RegisterActivity(activities.RecordApprovalDecisionActivity)

	tenantID := uuid.New()
	accountID := uuid.New()
	orderID := uuid.New()
	leaseID := uuid.New()
	lineageID := uuid.New()
	ruleID := uuid.New()

	input := OrderApprovalInput{
		TenantID:  tenantID,
		AccountID: accountID,
		OrderID:   orderID,
		LeaseID:   leaseID,
		LineageID: lineageID,
		BreachedRules: []BreachedRuleSummary{
			{
				RuleID:         ruleID,
				RuleCode:       "R_CONCENTRATION_LIMIT",
				Severity:       "APPROVAL_REQUIRED",
				ThresholdValue: decimal.RequireFromString("0.05"),
				Message:        "Position exceeds 5% limit",
			},
		},
		TTL:         15 * time.Minute,
		RequestedAt: time.Now().UTC(),
	}

	// Register delayed signal callback (Approve after 1 minute)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalOrderApprove, ApproveSignalPayload{
			ApproverID: "compliance_officer_alice",
			Reason:     "Approved portfolio manager exemption under mandate X",
			ApprovedAt: time.Now().UTC(),
		})
	}, 1*time.Minute)

	env.ExecuteWorkflow(OrderApprovalWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("Workflow did not complete")
	}

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("Workflow failed with error: %v", err)
	}

	var result OrderApprovalResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatalf("Failed to get workflow result: %v", err)
	}

	if result.Status != "APPROVED" {
		t.Errorf("Expected status APPROVED, got %s", result.Status)
	}
	if result.DecidedBy != "compliance_officer_alice" {
		t.Errorf("Expected DecidedBy compliance_officer_alice, got %s", result.DecidedBy)
	}
	if !mockRM.parked[leaseID] {
		t.Errorf("Expected lease to be parked on start")
	}
	if !mockRM.unparked[leaseID] {
		t.Errorf("Expected lease to be unparked on approval")
	}
}

func TestOrderApprovalWorkflow_RejectedBySignal(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockRM := newMockResManager()
	activities := ApprovalActivities{ResMgr: mockRM}
	env.RegisterActivity(activities.ParkReservationActivity)
	env.RegisterActivity(activities.ResolveReservationActivity)
	env.RegisterActivity(activities.NotifyComplianceOfficersActivity)
	env.RegisterActivity(activities.RecordApprovalDecisionActivity)

	tenantID := uuid.New()
	accountID := uuid.New()
	orderID := uuid.New()
	leaseID := uuid.New()
	lineageID := uuid.New()

	input := OrderApprovalInput{
		TenantID:    tenantID,
		AccountID:   accountID,
		OrderID:     orderID,
		LeaseID:     leaseID,
		LineageID:   lineageID,
		TTL:         10 * time.Minute,
		RequestedAt: time.Now().UTC(),
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalOrderReject, RejectSignalPayload{
			RejecterID: "compliance_officer_bob",
			Reason:     "Exceeds mandate limit without valid sponsor waiver",
			RejectedAt: time.Now().UTC(),
		})
	}, 2*time.Minute)

	env.ExecuteWorkflow(OrderApprovalWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("Workflow did not complete")
	}

	var result OrderApprovalResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatalf("Failed to get workflow result: %v", err)
	}

	if result.Status != "REJECTED" {
		t.Errorf("Expected status REJECTED, got %s", result.Status)
	}
	if !mockRM.released[leaseID] {
		t.Errorf("Expected lease to be released on rejection")
	}
}

func TestOrderApprovalWorkflow_ExpiredOnTTL(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	mockRM := newMockResManager()
	activities := ApprovalActivities{ResMgr: mockRM}
	env.RegisterActivity(activities.ParkReservationActivity)
	env.RegisterActivity(activities.ResolveReservationActivity)
	env.RegisterActivity(activities.NotifyComplianceOfficersActivity)
	env.RegisterActivity(activities.RecordApprovalDecisionActivity)

	tenantID := uuid.New()
	accountID := uuid.New()
	orderID := uuid.New()
	leaseID := uuid.New()
	lineageID := uuid.New()

	input := OrderApprovalInput{
		TenantID:    tenantID,
		AccountID:   accountID,
		OrderID:     orderID,
		LeaseID:     leaseID,
		LineageID:   lineageID,
		TTL:         5 * time.Minute,
		RequestedAt: time.Now().UTC(),
	}

	// No signal sent — let timer expire
	env.ExecuteWorkflow(OrderApprovalWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("Workflow did not complete")
	}

	var result OrderApprovalResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatalf("Failed to get workflow result: %v", err)
	}

	if result.Status != "EXPIRED" {
		t.Errorf("Expected status EXPIRED, got %s", result.Status)
	}
	if !mockRM.released[leaseID] {
		t.Errorf("Expected lease to be released on TTL expiry")
	}
}

func TestDMABypassManager_LifecycleAndSingleUse(t *testing.T) {
	mgr := NewDMABypassManager("v1", []byte("super-secret-hmac-key-for-test-32b"))

	tenantID := uuid.New()
	accountID := uuid.New()
	ruleID := uuid.New()

	// 1. Issue Token
	token := mgr.IssueToken(tenantID, accountID, ruleID, "compliance_lead_carol", "Pre-market DMA basket authorization", 5*time.Minute)
	if token == nil || token.TokenID == uuid.Nil {
		t.Fatalf("Expected valid token, got nil")
	}

	// 2. First consumption: must succeed
	ctx := context.Background()
	err := mgr.ValidateAndConsume(ctx, token.TokenID, tenantID, accountID, ruleID)
	if err != nil {
		t.Fatalf("First ValidateAndConsume failed: %v", err)
	}

	// 3. Second consumption attempt: must be strictly rejected (Single Use)
	err = mgr.ValidateAndConsume(ctx, token.TokenID, tenantID, accountID, ruleID)
	if err != ErrBypassTokenUsed {
		t.Fatalf("Expected ErrBypassTokenUsed on second attempt, got: %v", err)
	}

	// 4. Scope mismatch attempt on new token
	token2 := mgr.IssueToken(tenantID, accountID, ruleID, "compliance_lead_carol", "Test scope", 5*time.Minute)
	wrongAccount := uuid.New()
	err = mgr.ValidateAndConsume(ctx, token2.TokenID, tenantID, wrongAccount, ruleID)
	if err != ErrBypassTokenScopeFail {
		t.Fatalf("Expected ErrBypassTokenScopeFail for wrong account, got: %v", err)
	}

	// 5. Expired token attempt
	token3 := mgr.IssueToken(tenantID, accountID, ruleID, "compliance_lead_carol", "Test expiry", -1*time.Minute)
	err = mgr.ValidateAndConsume(ctx, token3.TokenID, tenantID, accountID, ruleID)
	if err != ErrBypassTokenExpired {
		t.Fatalf("Expected ErrBypassTokenExpired for expired token, got: %v", err)
	}

	// 6. Verify audit log integrity
	logs := mgr.GetAuditLogs()
	if len(logs) != 4 {
		t.Fatalf("Expected 4 audit logs, got %d", len(logs))
	}
	if !logs[0].Success || logs[1].Success || logs[2].Success || logs[3].Success {
		t.Errorf("Audit log success flags incorrect: %+v", logs)
	}
}

type mockDurableAuditSink struct {
	mu   sync.Mutex
	logs []DMABypassAuditLog
}

func (s *mockDurableAuditSink) LogBypassAttempt(ctx context.Context, log DMABypassAuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, log)
	return nil
}

func TestDMABypassManager_KeyRotationAndDurableAudit(t *testing.T) {
	mgr := NewDMABypassManager("key-v1", []byte("initial-secret-key-32-bytes-v1"))
	sink := &mockDurableAuditSink{}
	mgr.SetDurableLogger(sink)

	tenantID := uuid.New()
	accountID := uuid.New()
	ruleID := uuid.New()

	// Token 1 issued with key-v1
	t1 := mgr.IssueToken(tenantID, accountID, ruleID, "auditor_dave", "Token 1", 10*time.Minute)
	if t1.KeyID != "key-v1" {
		t.Errorf("Expected KeyID key-v1, got %s", t1.KeyID)
	}

	// Rotate key: register key-v2 and promote to active
	mgr.AddKey("key-v2", []byte("rotated-secret-key-32-bytes-v2"))
	err := mgr.SetActiveKey("key-v2")
	if err != nil {
		t.Fatalf("SetActiveKey failed: %v", err)
	}

	// Token 2 issued with key-v2
	t2 := mgr.IssueToken(tenantID, accountID, ruleID, "auditor_dave", "Token 2", 10*time.Minute)
	if t2.KeyID != "key-v2" {
		t.Errorf("Expected KeyID key-v2, got %s", t2.KeyID)
	}

	// Validate Token 1 (signed with key-v1): must still validate via key ring
	ctx := context.Background()
	err = mgr.ValidateAndConsume(ctx, t1.TokenID, tenantID, accountID, ruleID)
	if err != nil {
		t.Fatalf("Token 1 validation with rotated key ring failed: %v", err)
	}

	// Validate Token 2 (signed with key-v2): must validate
	err = mgr.ValidateAndConsume(ctx, t2.TokenID, tenantID, accountID, ruleID)
	if err != nil {
		t.Fatalf("Token 2 validation failed: %v", err)
	}

	// Verify durable audit sink received both events
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.logs) != 2 {
		t.Fatalf("Expected 2 durable audit logs, got %d", len(sink.logs))
	}
	if sink.logs[0].KeyID != "key-v1" || sink.logs[1].KeyID != "key-v2" {
		t.Errorf("Durable audit logs recorded incorrect key IDs: %+v", sink.logs)
	}
}
