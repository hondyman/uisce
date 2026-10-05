package regulatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"github.com/hondyman/uisce/backend/internal/compliance"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/hondyman/uisce/backend/internal/compliance/drift"
)

func getAlphaTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("ALPHA_DSN")
	if dsn == "" {
		home, _ := os.UserHomeDir()
		caPath := filepath.Join(home, ".uisce/certs/ca.crt")
		certPath := filepath.Join(home, ".uisce/certs/postgres-client.crt")
		keyPath := filepath.Join(home, ".uisce/certs/postgres-client.key")

		if _, err := os.Stat(caPath); err == nil {
			dsn = "host=100.84.50.65 port=5432 user=postgres password=postgres dbname=alpha sslmode=verify-full sslrootcert=" + caPath + " sslcert=" + certPath + " sslkey=" + keyPath
		}
	}

	if dsn == "" {
		t.Skip("ALPHA_DSN not set and mTLS certificates not found; skipping live DB test")
		return nil
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Failed to open connection to alpha: %v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("Cannot ping alpha database: %v", err)
		return nil
	}

	return db
}

// Gate 2: Transition Legality Tested Both Ways
func TestRegulatoryWorkflow_TransitionLegality(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := NewService(db)

	// 1. Open Case (INTAKED)
	caseCode := fmt.Sprintf("RCC-TEST-LEG-%d", time.Now().UnixNano()%1000000)
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:        caseCode,
		Source:          SourceRegulatorPublication,
		SourceReference: "ESMA/2026/QA-110",
		Title:           "Test Transition Legality",
		Description:     "Verifying state machine transition guards",
		CreatedBy:       "steward_alice",
	})
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	// 2. Illegal direct transition: INTAKED -> PUBLISHED must fail
	_, err = db.ExecContext(ctx, "UPDATE compliance.regulatory_change_case SET status = 'PUBLISHED' WHERE id = $1", c.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal case transition INTAKED -> PUBLISHED")
	t.Logf("Illegal transition INTAKED -> PUBLISHED correctly rejected by trigger: %v", err)

	// 3. Illegal triage: TRIAGED with classification = NULL must fail
	_, err = db.ExecContext(ctx, "UPDATE compliance.regulatory_change_case SET status = 'TRIAGED', classification = NULL WHERE id = $1", c.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "triage requires classification")
	t.Logf("Triage without classification correctly rejected: %v", err)

	// 4. Legal transition: INTAKED -> TRIAGED
	err = svc.TriageCase(ctx, TriageRequest{
		CaseID:         c.ID,
		Classification: ClassificationParameterChange,
		TriageNotes:    "Parameter adjustment required",
		TriagedBy:      "steward_alice",
	})
	require.NoError(t, err)

	// 5. Illegal transition: TRIAGED -> APPROVED_FOR_PUBLISH must fail
	_, err = db.ExecContext(ctx, "UPDATE compliance.regulatory_change_case SET status = 'APPROVED_FOR_PUBLISH' WHERE id = $1", c.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal case transition TRIAGED -> APPROVED_FOR_PUBLISH")
	t.Logf("Illegal transition TRIAGED -> APPROVED_FOR_PUBLISH correctly rejected: %v", err)

	// 6. Legal transition: TRIAGED -> UNDER_REVIEW
	err = svc.StartReview(ctx, c.ID, "steward_alice", nil)
	require.NoError(t, err)

	// 7. Legal transition: UNDER_REVIEW -> APPROVED_FOR_PUBLISH
	err = svc.ApproveCase(ctx, c.ID, "steward_alice", "Corpus passed")
	require.NoError(t, err)

	// 8. Legal alternative terminal: APPROVED_FOR_PUBLISH -> REJECTED
	err = svc.RejectCase(ctx, c.ID, "steward_bob", "Overruled by chief compliance officer")
	require.NoError(t, err)

	updatedCase, err := svc.GetCaseByID(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, StatusRejected, updatedCase.Status)
	t.Logf("State machine transition legality verified 100%% both positive and negative paths!")
}

// Gate 3: Full Happy-Path Publish & Temporal Workflow
func TestRegulatoryWorkflow_FullHappyPathPublish(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	svc := NewService(db)
	loader := compliance.NewMultiTenantRuleLoader(db)

	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	require.NoError(t, err)

	// Create an isolated Test Core Rule under goldTenant
	depRuleID := uuid.New()
	testRuleCode := fmt.Sprintf("TEST_UCITS_DEP_%d", time.Now().UnixNano()%1000000)
	origAST := []byte(`{"left":{"path":"position.depository_exposure_pct","type":"METRIC"},"operator":"GREATER_THAN","right":{"name":"single_depository_limit_pct","type":"PARAM"},"type":"COMPARISON"}`)
	origParams := []byte(`{"aggregate_across_accounts":true,"single_depository_limit_pct":"0.200000"}`)
	origCitation := "UCITS Directive 2009/65/EC Art. 52 § 1(b) Deposit Limit 20%"
	curVer := 1

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
			priority, is_active, current_version, ast_condition, parameter_thresholds,
			citation, compiled_bytecode, library_status, effective_from
		) VALUES (
			$1, $2, 'inherit', $3, 'Test Deposit Limit', 'PRE_TRADE', 'HARD_BLOCK',
			100, true, 1, $4::jsonb, $5::jsonb, $6, ''::bytea, 'ACTIVE', '2026-01-01T00:00:00Z'
		)
	`, depRuleID, goldTenant, testRuleCode, string(origAST), string(origParams), origCitation)
	require.NoError(t, err)

	coreHashV1, err := canonical.ComputeRuleContentHashFromRaw(origAST, origParams, origCitation)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
			created_by, created_at
		) VALUES (
			$1, 1, $2, $3::jsonb, $4::jsonb,
			$5, '2026-01-01T00:00:00Z', null, $6, $7, 'seed', now()
		)
	`, depRuleID, goldTenant, string(origAST), string(origParams), origCitation, coreHashV1, canonical.ComputeBytecodeHash(nil))
	require.NoError(t, err)

	// Set up an extend tenant rule to assert drift flag fanout
	extendTenant := uuid.New()
	extendRuleID := uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, core_rule_id, inherit_mode, pinned_core_version, drift_status,
			rule_code, name, rule_phase, severity, priority, is_active, current_version,
			ast_condition, parameter_thresholds, citation, compiled_bytecode
		) VALUES (
			$1, $2, $3, 'extend', $4, 'CURRENT',
			$5, 'Custom Deposit Limit', 'PRE_TRADE', 'HARD_BLOCK', 100, true, 1,
			$6::jsonb, $7::jsonb, $8, ''::bytea
		)
	`, extendRuleID, extendTenant, depRuleID, curVer, testRuleCode, string(origAST), string(origParams), origCitation)
	require.NoError(t, err)

	// Also insert version 1 snapshot for extend rule
	extendHash, _ := canonical.ComputeRuleContentHashFromRaw(origAST, origParams, origCitation)
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
			created_by, created_at
		) VALUES (
			$1, 1, $2, $3::jsonb, $4::jsonb,
			$5, now(), null, $6, $7, 'seed', now()
		)
	`, extendRuleID, extendTenant, string(origAST), string(origParams), origCitation, extendHash, canonical.ComputeBytecodeHash(nil))
	require.NoError(t, err)

	defer func() {
		// Soft-delete test rules to isolate database state
		_, _ = db.ExecContext(context.Background(), "UPDATE compliance.compliance_rule SET valid_to = now() WHERE rule_code = $1 OR id IN ($2, $3)", testRuleCode, extendRuleID, depRuleID)
	}()

	// 1. Intake Case
	caseCode := fmt.Sprintf("RCC-2027-%d", time.Now().UnixNano()%100000)
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:        caseCode,
		Source:          SourceRegulatorPublication,
		SourceReference: "ESMA/2026/1102",
		Title:           "UCITS Deposit Concentration Limit Tightened from 20% to 15%",
		Description:     "ESMA Level 2 measures lowering single-bank deposit threshold to 15%",
		CreatedBy:       "steward_alice",
	})
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_notification WHERE payload->>'case_code' = $1", caseCode)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.governance_audit_event WHERE steward_notes LIKE '%' || $1 || '%'", caseCode)
	}()

	// 2. Triage Case
	err = svc.TriageCase(ctx, TriageRequest{
		CaseID:          c.ID,
		Classification:  ClassificationParameterChange,
		TriageNotes:     "Threshold parameter limit adjusted from 0.200000 to 0.150000",
		TriagedBy:       "steward_alice",
		AffectedRuleIDs: []uuid.UUID{depRuleID},
	})
	require.NoError(t, err)

	// 3. Review Case & Render Diff
	var origASTMap, origParamsMap map[string]interface{}
	_ = json.Unmarshal(origAST, &origASTMap)
	_ = json.Unmarshal(origParams, &origParamsMap)

	newParamsMap := map[string]interface{}{
		"single_depository_limit_pct": "0.150000",
		"aggregate_across_accounts":   true,
	}
	newCitation := "UCITS Directive 2009/65/EC Art. 52 § 1(b) (as amended by ESMA/2026/1102)"
	futureEffective := time.Now().UTC().Add(30 * 24 * time.Hour) // Effective in 30 days

	diff := drift.CompareAST(origASTMap, origASTMap)
	diffView := RuleDiffView{
		RuleID:             depRuleID,
		RuleCode:           testRuleCode,
		Diff:               diff,
		OldThresholds:      origParamsMap,
		NewThresholds:      newParamsMap,
		OldCitation:        origCitation,
		NewCitation:        newCitation,
		HasBreakingChanges: false,
	}

	err = svc.StartReview(ctx, c.ID, "steward_alice", []RuleDiffView{diffView})
	require.NoError(t, err)

	// 4. Execute Corpus Gate
	passingCorpus := []drift.ScenarioTestCase{
		{
			CaseID:      "UCITS_DEP15_PASS",
			Description: "Deposit exposure at 14% (<15%) -> PASS",
			InputParams: map[string]interface{}{"accountId": "acc-dep-1"},
			MetricSnapshots: map[string]interface{}{
				"position.depository_exposure_pct": decimal.RequireFromString("0.140000"),
				"single_depository_limit_pct":      decimal.RequireFromString("0.150000"),
			},
			ExpectedPassed: true,
			ExpectedAction: "APPROVED",
		},
		{
			CaseID:      "UCITS_DEP15_FAIL",
			Description: "Deposit exposure at 16% (>15%) -> FAIL",
			InputParams: map[string]interface{}{"accountId": "acc-dep-2"},
			MetricSnapshots: map[string]interface{}{
				"position.depository_exposure_pct": decimal.RequireFromString("0.160000"),
				"single_depository_limit_pct":      decimal.RequireFromString("0.150000"),
			},
			ExpectedPassed: false,
			ExpectedAction: "BLOCKED",
		},
	}

	draft := RuleDraft{
		RuleID:        depRuleID,
		NewAST:        origASTMap,
		NewThresholds: newParamsMap,
		NewCitation:   newCitation,
		EffectiveFrom: futureEffective,
		TestCorpus:    passingCorpus,
	}

	corpusRes, err := svc.ExecuteCorpusGate(ctx, c.ID, "steward_alice", []RuleDraft{draft})
	require.NoError(t, err)
	require.True(t, corpusRes.AllPassed)

	// 5. Approve Case
	err = svc.ApproveCase(ctx, c.ID, "steward_alice", "Corpus gate passed 2/2 tests")
	require.NoError(t, err)

	// 6. Publish Release
	pubEntries, err := svc.PublishRelease(ctx, PublishRequest{
		CaseID:    c.ID,
		StewardID: "steward_alice",
		Drafts:    []RuleDraft{draft},
	})
	require.NoError(t, err)
	require.Len(t, pubEntries, 1)
	require.Equal(t, curVer+1, pubEntries[0].ToVersion)
	t.Logf("Core release published successfully! Version advanced: v%d -> v%d, Hash: %s",
		pubEntries[0].FromVersion, pubEntries[0].ToVersion, pubEntries[0].ContentHash)

	// 7. Verify Extend Tenant was Drift Flagged
	var driftStatus string
	err = db.QueryRowContext(ctx, "SELECT drift_status FROM compliance.compliance_rule WHERE id = $1", extendRuleID).Scan(&driftStatus)
	require.NoError(t, err)
	require.Equal(t, "CORE_VERSION_UPDATED", driftStatus)
	t.Logf("Extend tenant rule %s correctly drift-flagged with CORE_VERSION_UPDATED!", extendRuleID)

	// 8. Verify Notifications Generated
	notifSvc := NewNotificationService(db)
	extendNotifs, err := notifSvc.GetUnreadNotifications(ctx, extendTenant)
	require.NoError(t, err)
	require.NotEmpty(t, extendNotifs)
	require.Equal(t, NotificationDriftFlag, extendNotifs[0].Kind)
	t.Logf("Extend tenant received unread DRIFT_FLAG notification: %q", extendNotifs[0].Title)

	// 9. Verify Point-in-Time Rule Loading
	// As of NOW (before future effective_from): new version is not effective yet
	rulesNow, err := loader.LoadTenantActiveRulesAsOf(ctx, goldTenant, time.Now().UTC())
	require.NoError(t, err)
	var foundDepRuleNow *compliance.ComplianceRuleRecord
	for i := range rulesNow {
		if rulesNow[i].RuleCode == testRuleCode {
			foundDepRuleNow = &rulesNow[i]
			break
		}
	}
	require.Nil(t, foundDepRuleNow, "Future rule should NOT be active before effective_from")

	// As of futureEffective: new version takes effect
	rulesFuture, err := loader.LoadTenantActiveRulesAsOf(ctx, goldTenant, futureEffective.Add(time.Hour))
	require.NoError(t, err)
	var foundDepRuleFuture *compliance.ComplianceRuleRecord
	for i := range rulesFuture {
		if rulesFuture[i].RuleCode == testRuleCode {
			foundDepRuleFuture = &rulesFuture[i]
			break
		}
	}
	require.NotNil(t, foundDepRuleFuture, "New version MUST be active at effective_from")
	require.Equal(t, "0.150000", foundDepRuleFuture.ParameterThresholds["single_depository_limit_pct"])
	t.Logf("Point-in-Time verification PASSED: Core library cleanly effective-dated (inactive before %s, active after)!", futureEffective.Format(time.RFC3339))

	// 10. Verify Governance Audit Event
	var govCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM compliance.governance_audit_event
		WHERE event_type = 'REGULATORY_CHANGE_PUBLISHED' AND steward_notes LIKE '%' || $1 || '%'
	`, caseCode).Scan(&govCount)
	require.NoError(t, err)
	require.Equal(t, 1, govCount)
	t.Logf("Governance audit event REGULATORY_CHANGE_PUBLISHED verified!")
}

// Gate 4: Corpus Gate Rejection Tests
func TestRegulatoryWorkflow_CorpusGateRejection(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := NewService(db)
	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	require.NoError(t, err)

	// 1. Failing corpus test case
	var depRuleID uuid.UUID
	err = db.QueryRowContext(ctx, "SELECT id FROM compliance.compliance_rule WHERE tenant_id = $1 AND rule_code = 'UCITS_DEPOSIT_20'", goldTenant).Scan(&depRuleID)
	require.NoError(t, err)

	failingCorpus := []drift.ScenarioTestCase{
		{
			CaseID:      "UCITS_DEP_FAIL_TEST",
			Description: "Intentionally broken expectation",
			InputParams: map[string]interface{}{"accountId": "acc-1"},
			MetricSnapshots: map[string]interface{}{
				"position.depository_exposure_pct": decimal.RequireFromString("0.160000"),
				"single_depository_limit_pct":      decimal.RequireFromString("0.150000"),
			},
			ExpectedPassed: true, // Broken expectation: 16% should NOT pass 15% limit
			ExpectedAction: "APPROVED",
		},
	}

	caseCode := fmt.Sprintf("RCC-CORPUS-FAIL-%d", time.Now().UnixNano()%100000)
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:    caseCode,
		Source:      SourceInternal,
		Title:       "Corpus Gate Failure Test",
		Description: "Testing failing corpus validation",
	})
	require.NoError(t, err)
	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	draftFailing := RuleDraft{
		RuleID:     depRuleID,
		TestCorpus: failingCorpus,
	}

	_, err = svc.ExecuteCorpusGate(ctx, c.ID, "steward_bob", []RuleDraft{draftFailing})
	require.Error(t, err)
	require.Contains(t, err.Error(), "corpus gate validation failed")
	t.Logf("Corpus gate correctly rejected failing test vector: %v", err)

	// 2. Reject SEMANTIC_CHANGE on PROVISIONAL rule
	var provRuleID uuid.UUID
	err = db.QueryRowContext(ctx, `
		SELECT id FROM compliance.compliance_rule
		WHERE tenant_id = $1 AND rule_code = 'ORDER_RATE_LIMIT' AND library_status = 'PROVISIONAL'
	`, goldTenant).Scan(&provRuleID)
	require.NoError(t, err)

	draftProv := RuleDraft{
		RuleID: provRuleID,
	}
	_, err = svc.ExecuteCorpusGate(ctx, c.ID, "steward_bob", []RuleDraft{draftProv})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PROVISIONAL library status and cannot undergo regulatory modification")
	t.Logf("Corpus gate correctly blocked modification of PROVISIONAL rule: %v", err)
}

type mockWorkflowActivities struct {
	escalatedCalled bool
	mockCase        RegulatoryChangeCase
}

func (m *mockWorkflowActivities) CreateCaseActivity(ctx context.Context, req IntakeRequest) (*RegulatoryChangeCase, error) {
	return &m.mockCase, nil
}

func (m *mockWorkflowActivities) EscalateCaseActivity(ctx context.Context, cID uuid.UUID) error {
	m.escalatedCalled = true
	return nil
}

func (m *mockWorkflowActivities) RejectCaseActivity(ctx context.Context, cID uuid.UUID, actor, reason string) error {
	return nil
}

// Gate 5: TTL Escalation via Temporal Test Environment Clock
func TestRegulatoryWorkflow_TTLEscalation(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	caseID := uuid.New()
	caseCode := "RCC-TTL-ESCALATE-001"
	dueAt := time.Now().Add(30 * 24 * time.Hour)

	mockCase := RegulatoryChangeCase{
		ID:       caseID,
		CaseCode: caseCode,
		DueAt:    dueAt,
		Status:   StatusIntaked,
	}

	mockAct := &mockWorkflowActivities{mockCase: mockCase}
	env.RegisterActivity(mockAct.CreateCaseActivity)
	env.RegisterActivity(mockAct.EscalateCaseActivity)
	env.RegisterActivity(mockAct.RejectCaseActivity)

	// Delayed signal sending: send Reject after 35 days (after the 30-day TTL expires)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalRejectCase, RejectSignalPayload{
			Actor:  "steward_ttl_sweeper",
			Reason: "Closing expired case",
		})
	}, 35*24*time.Hour)

	// Execute workflow
	input := RegulatoryWorkflowInput{
		IntakeReq: IntakeRequest{
			CaseCode: caseCode,
			Title:    "TTL Escalation Test",
		},
	}
	env.ExecuteWorkflow(RegulatoryChangeWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.True(t, mockAct.escalatedCalled, "EscalateCaseActivity MUST be invoked when case is untouched past due_at TTL")
	t.Logf("Temporal TTL Escalation Timer verified without real-time sleep!")
}

// Gate 6: Custom-Rule Evolution under Corrected Trigger
func TestRegulatoryWorkflow_CustomRuleVersionEvolution(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	customTenant := uuid.New()
	customRuleID := uuid.New()
	ruleCode := "CUSTOM_VOLATILITY_CAP"
	astV1 := `{"left":{"path":"instrument.volatility_30d","type":"METRIC"},"operator":"GREATER_THAN","right":{"name":"max_vol","type":"PARAM"},"type":"COMPARISON"}`
	paramsV1 := `{"max_vol":"0.400000"}`
	citationV1 := "Custom Mandate Volatility Cap v1"

	hashV1, err := canonical.ComputeRuleContentHashFromRaw([]byte(astV1), []byte(paramsV1), citationV1)
	require.NoError(t, err)

	// 1. Insert Custom Rule V1
	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = tx1.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
			priority, is_active, current_version, ast_condition, parameter_thresholds,
			citation, compiled_bytecode
		) VALUES (
			$1, $2, 'custom', $3, 'Custom Vol Cap', 'PRE_TRADE', 'HARD_BLOCK',
			100, true, 1, $4::jsonb, $5::jsonb, $6, ''::bytea
		)
	`, customRuleID, customTenant, ruleCode, astV1, paramsV1, citationV1)
	require.NoError(t, err)

	_, err = tx1.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
			created_by, created_at
		) VALUES (
			$1, 1, $2, $3::jsonb, $4::jsonb,
			$5, now(), null, $6, $7, 'custom_steward', now()
		)
	`, customRuleID, customTenant, astV1, paramsV1, citationV1, hashV1, canonical.ComputeBytecodeHash(nil))
	require.NoError(t, err)

	require.NoError(t, tx1.Commit())

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_evaluation_event WHERE tenant_id = $1", customTenant)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule WHERE id = $1", customRuleID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule_version WHERE rule_id = $1", customRuleID)
	}()

	// 2. Evolve Custom Rule to V2 (current_version = 2) in Transaction
	paramsV2 := `{"max_vol":"0.350000"}`
	citationV2 := "Custom Mandate Volatility Cap v2 (Tightened)"
	hashV2, err := canonical.ComputeRuleContentHashFromRaw([]byte(astV1), []byte(paramsV2), citationV2)
	require.NoError(t, err)

	tx2, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = tx2.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
			created_by, created_at
		) VALUES (
			$1, 2, $2, $3::jsonb, $4::jsonb,
			$5, now(), null, $6, $7, 'custom_steward', now()
		)
	`, customRuleID, customTenant, astV1, paramsV2, citationV2, hashV2, canonical.ComputeBytecodeHash(nil))
	require.NoError(t, err)

	_, err = tx2.ExecContext(ctx, `
		UPDATE compliance.compliance_rule
		SET 
			current_version = 2,
			parameter_thresholds = $1::jsonb,
			citation = $2,
			updated_at = now()
		WHERE id = $3
	`, paramsV2, citationV2, customRuleID)
	require.NoError(t, err)

	require.NoError(t, tx2.Commit())
	t.Logf("Custom rule evolution to version 2 succeeded with current_version structural trigger validation!")

	// 3. Emit evaluation event referencing custom rule version 2
	evalInput := canonical.EvaluationHashInput{
		LineageID:       uuid.New(),
		TenantID:        customTenant,
		RuleID:          customRuleID,
		RuleVersion:     2,
		RuleContentHash: hashV2,
		ActionTaken:     "APPROVED",
		Passed:          true,
		InputParams: map[string]interface{}{
			"accountId": "acc-custom-1",
		},
		MetricSnapshots: map[string]interface{}{
			"instrument.volatility_30d": decimal.RequireFromString("0.320000"),
		},
	}
	evalHash, err := canonical.ComputeEvaluationHash(evalInput)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, rule_id, rule_version, rule_content_hash,
			action_taken, passed, latency_micros, evaluation_hash, evaluated_at,
			input_params, metric_snapshots
		) VALUES (
			gen_random_uuid(), $1, $2, $3, 2, $4,
			'APPROVED', true, 110, $5, now(),
			'{"accountId":"acc-custom-1"}'::jsonb, '{"instrument.volatility_30d":"0.320000"}'::jsonb
		)
	`, evalInput.LineageID, customTenant, customRuleID, hashV2, evalHash)
	require.NoError(t, err)
	t.Logf("Evaluation event for custom rule version 2 passed FK RESTRICT constraint cleanly!")
}

// Steward Presentation View Output Test
func TestRegulatoryWorkflow_StewardPresentationView(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := NewService(db)
	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	require.NoError(t, err)

	var ruleID uuid.UUID
	err = db.QueryRowContext(ctx, "SELECT id FROM compliance.compliance_rule WHERE tenant_id = $1 AND rule_code = 'UCITS_ISSUER_5'", goldTenant).Scan(&ruleID)
	require.NoError(t, err)

	caseCode := fmt.Sprintf("RCC-VIEW-%d", time.Now().UnixNano()%100000)
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:        caseCode,
		Source:          SourceRegulatorPublication,
		SourceReference: "https://www.esma.europa.eu/guidelines/2026/ucits-5",
		Title:           "UCITS Single-Issuer 5% Clarification",
		Description:     "Clarifying lookthrough treatment on multi-layered umbrella funds",
		CreatedBy:       "steward_alice",
	})
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	err = svc.TriageCase(ctx, TriageRequest{
		CaseID:          c.ID,
		Classification:  ClassificationInterpretationOnly,
		TriageNotes:     "Updated legal citation reference and interpretation note",
		TriagedBy:       "steward_alice",
		AffectedRuleIDs: []uuid.UUID{ruleID},
	})
	require.NoError(t, err)

	view, err := svc.GetStewardTriageView(ctx, caseCode)
	require.NoError(t, err)
	require.Equal(t, caseCode, view.CaseCode)
	require.Equal(t, StatusTriaged, view.Status)
	require.Len(t, view.AffectedRules, 1)
	require.Equal(t, "UCITS_ISSUER_5", view.AffectedRules[0].RuleCode)
	require.Contains(t, view.AvailableSignals, "StartReview()")

	viewJSON, _ := json.MarshalIndent(view, "", "  ")
	t.Logf("Structured Steward Presentation View (Page Designer Ready):\n%s", string(viewJSON))
}
