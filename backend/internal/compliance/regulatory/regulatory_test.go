package regulatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
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
	"github.com/hondyman/uisce/backend/internal/compliance/jobs"
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

	// Ensure Migrations 007 and 008 up are applied on the test DB
	ensureMigrationsApplied(t, db)

	return db
}

func ensureMigrationsApplied(t *testing.T, db *sql.DB) {
	t.Helper()
	migDir := filepath.Join("..", "..", "..", "db", "migrations")
	if _, err := os.Stat(migDir); err != nil {
		migDir = filepath.Join("backend", "db", "migrations")
	}

	for _, f := range []string{"20261218_007_regulatory_change_workflow.up.sql", "20261219_008_trigger_refactor_and_draft_guard.up.sql"} {
		upFile := filepath.Join(migDir, f)
		upContent, _ := os.ReadFile(upFile)
		_, _ = db.Exec(string(upContent))
	}
}

// Gate 2: Transition Legality Tested Both Ways
func TestRegulatoryWorkflow_TransitionLegality(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := NewService(db)

	// 1. Open Case (INTAKED)
	caseCode := fmt.Sprintf("RCC-TEST-LEG-%s", uuid.New().String()[:8])
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
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE case_id = $1", c.ID)
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

	// 7. Save draft for approval
	var dummyRuleID uuid.UUID
	err = db.QueryRowContext(ctx, "SELECT id FROM compliance.compliance_rule LIMIT 1").Scan(&dummyRuleID)
	require.NoError(t, err)

	dummyAST := map[string]interface{}{"type": "METRIC", "path": "pos.weight"}
	dummyParams := map[string]interface{}{"limit": "0.100000"}
	corpusRes, err := svc.ExecuteCorpusGate(ctx, c.ID, "steward_alice", []RuleDraft{
		{
			RuleID:        dummyRuleID,
			NewAST:        dummyAST,
			NewThresholds: dummyParams,
			NewCitation:   "Test Citation",
			TestCorpus: []drift.ScenarioTestCase{
				{
					CaseID:          "TC_TRANS",
					MetricSnapshots: map[string]interface{}{"pos.weight": decimal.RequireFromString("0.050000"), "limit": decimal.RequireFromString("0.100000")},
					ExpectedPassed:  true,
					ExpectedAction:  "APPROVED",
				},
			},
		},
	})
	require.NoError(t, err)
	require.True(t, corpusRes.AllPassed)

	// 8. Legal transition: UNDER_REVIEW -> APPROVED_FOR_PUBLISH
	err = svc.ApproveCase(ctx, c.ID, "steward_alice", "Corpus passed")
	require.NoError(t, err)

	// 9. Legal alternative terminal: APPROVED_FOR_PUBLISH -> REJECTED
	err = svc.RejectCase(ctx, c.ID, "steward_bob", "Overruled by chief compliance officer")
	require.NoError(t, err)

	updatedCase, err := svc.GetCase(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, StatusRejected, updatedCase.Status)
	t.Logf("State machine transition legality verified 100%% both positive and negative paths!")
}

// Gate 3 & Finding 2: Full Happy-Path Publish with Cryptographic Approval Binding
func TestRegulatoryWorkflow_FullHappyPathPublish(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	svc := NewService(db)
	testCoreTenant := uuid.New()
	_, err := db.ExecContext(ctx, `
		INSERT INTO public.tenants (id, name, display_name, gold_copy)
		VALUES ($1, $2, $2, false)
	`, testCoreTenant, fmt.Sprintf("e2e_core_%s", testCoreTenant.String()[:8]))
	require.NoError(t, err)

	// Create an isolated Test Core Rule under testCoreTenant
	depRuleID := uuid.New()
	testRuleCode := fmt.Sprintf("TEST_UCITS_DEP_%d", time.Now().UnixNano()%1000000)
	origAST := []byte(`{"left":{"path":"position.depository_exposure_pct","type":"METRIC"},"operator":"GREATER_THAN","right":{"name":"single_depository_limit_pct","type":"PARAM"},"type":"COMPARISON"}`)
	origParams := []byte(`{"aggregate_across_accounts":true,"single_depository_limit_pct":"0.200000"}`)
	origCitation := "UCITS Directive 2009/65/EC Art. 52 § 1(b) Deposit Limit 20%"
	curVer := 1

	// Set up an extend tenant rule to assert drift flag fanout
	extendTenant := uuid.New()
	extendRuleID := uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO public.tenants (id, name, display_name, gold_copy)
		VALUES ($1, $2, $2, false)
	`, extendTenant, fmt.Sprintf("e2e_extend_%s", extendTenant.String()[:8]))
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "UPDATE compliance.compliance_rule SET valid_to = now(), is_active = false WHERE id IN ($1, $2)", depRuleID, extendRuleID)
		_, _ = db.ExecContext(context.Background(), "UPDATE public.tenants SET is_active = false WHERE id IN ($1, $2)", testCoreTenant, extendTenant)
	}()

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
			priority, is_active, current_version, ast_condition, parameter_thresholds,
			citation, compiled_bytecode, library_status, effective_from
		) VALUES (
			$1, $2, 'custom', $3, 'Test Deposit Limit', 'PRE_TRADE', 'HARD_BLOCK',
			100, true, 1, $4::jsonb, $5::jsonb, $6, ''::bytea, 'ACTIVE', '2026-01-01T00:00:00Z'
		)
	`, depRuleID, testCoreTenant, testRuleCode, string(origAST), string(origParams), origCitation)
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
	`, depRuleID, testCoreTenant, string(origAST), string(origParams), origCitation, coreHashV1, canonical.ComputeBytecodeHash(nil))
	require.NoError(t, err)

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

	caseCode := fmt.Sprintf("RCC-2027-%s", uuid.New().String()[:8])
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_notification WHERE payload->>'case_code' = $1", caseCode)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id IN (SELECT id FROM compliance.regulatory_change_case WHERE case_code = $1)", caseCode)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE rule_id IN ($1, $2)", extendRuleID, depRuleID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE case_code = $1", caseCode)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.governance_audit_event WHERE rule_id IN ($1, $2) OR steward_notes LIKE '%' || $3 || '%'", extendRuleID, depRuleID, caseCode)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule_version WHERE rule_id IN ($1, $2)", extendRuleID, depRuleID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule WHERE id IN ($1, $2) OR rule_code = $3", extendRuleID, depRuleID, testRuleCode)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM public.tenants WHERE id IN ($1, $2)", extendTenant, testCoreTenant)
	})

	// 1. Intake Case
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:        caseCode,
		Source:          SourceRegulatorPublication,
		SourceReference: "ESMA/2026/1102",
		Title:           "UCITS Deposit Concentration Limit Tightened from 20% to 15%",
		Description:     "ESMA Level 2 measures lowering single-bank deposit threshold to 15%",
		CreatedBy:       "steward_alice",
	})
	require.NoError(t, err)

	// 2. Triage Case
	err = svc.TriageCase(ctx, TriageRequest{
		CaseID:          c.ID,
		Classification:  ClassificationParameterChange,
		TriageNotes:     "Threshold parameter limit adjusted from 0.200000 to 0.150000",
		TriagedBy:       "steward_alice",
		AffectedRuleIDs: []uuid.UUID{depRuleID},
	})
	require.NoError(t, err)

	// 3. Start Review
	err = svc.StartReview(ctx, c.ID, "steward_alice", nil)
	require.NoError(t, err)

	// 4. Create Proposed Rule Draft (20% -> 15%)
	var newASTMap map[string]interface{}
	_ = json.Unmarshal(origAST, &newASTMap)
	newParamsMap := map[string]interface{}{
		"single_depository_limit_pct": "0.150000",
		"aggregate_across_accounts":   true,
	}
	newCitation := "UCITS Directive 2009/65/EC Art. 52 § 1(b) Deposit Limit 15% (Amended 2026)"
	futureEffective := time.Now().UTC().Add(30 * 24 * time.Hour)

	scenarioVector := drift.ScenarioTestCase{
		CaseID:      "TEST_UCITS_DEP_BOUNDARY",
		Description: "Boundary test at 15%",
		MetricSnapshots: map[string]interface{}{
			"depository_exposure_pct":     decimal.RequireFromString("0.150000"),
			"single_depository_limit_pct": decimal.RequireFromString("0.150000"),
		},
		ExpectedPassed: true,
		ExpectedAction: "APPROVED",
	}

	draft := RuleDraft{
		RuleID:        depRuleID,
		NewAST:        newASTMap,
		NewThresholds: newParamsMap,
		NewCitation:   newCitation,
		EffectiveFrom: futureEffective,
		TestCorpus:    []drift.ScenarioTestCase{scenarioVector},
	}

	// 5. Execute Corpus Gate (Persists draft & validates scenario)
	corpusResult, err := svc.ExecuteCorpusGate(ctx, c.ID, "steward_alice", []RuleDraft{draft})
	require.NoError(t, err)
	require.True(t, corpusResult.AllPassed)
	t.Logf("Corpus Gate passed 100%% (%d/%d tests)!", corpusResult.PassedCases, corpusResult.TotalCases)

	// 6. Approve Case (Cryptographically binds approved draft hash)
	err = svc.ApproveCase(ctx, c.ID, "steward_alice", "ESMA 15% deposit threshold change verified against scenario corpus")
	require.NoError(t, err)

	// Verify draft is marked approved in compliance.regulatory_draft_rule
	var draftApproved bool
	var draftHash string
	err = db.QueryRowContext(ctx, `
		SELECT is_approved, proposed_content_hash
		FROM compliance.regulatory_draft_rule
		WHERE case_id = $1 AND rule_id = $2
	`, c.ID, depRuleID).Scan(&draftApproved, &draftHash)
	require.NoError(t, err)
	require.True(t, draftApproved, "Draft rule must be marked is_approved = true")
	require.Len(t, draftHash, 64, "Approved draft content hash must be 64 characters")

	// 7. Publish Release (Asserts hash match against approved draft)
	pubReq := PublishRequest{
		CaseID:    c.ID,
		StewardID: "steward_alice",
		Drafts:    []RuleDraft{draft},
	}
	publishedEntries, err := svc.PublishRelease(ctx, pubReq)
	require.NoError(t, err)
	require.Len(t, publishedEntries, 1)
	require.Equal(t, 1, publishedEntries[0].FromVersion)
	require.Equal(t, 2, publishedEntries[0].ToVersion)
	require.Equal(t, draftHash, publishedEntries[0].ApprovedContentHash)
	require.Equal(t, draftHash, publishedEntries[0].PublishedContentHash)
	t.Logf("Core release published successfully! Version advanced: v1 -> v2, Hash: %s", publishedEntries[0].PublishedContentHash)

	// 8. Assert Extend-Tenant Drift Flag Fanout
	var extendDriftStatus string
	err = db.QueryRowContext(ctx, `
		SELECT drift_status FROM compliance.compliance_rule WHERE id = $1
	`, extendRuleID).Scan(&extendDriftStatus)
	require.NoError(t, err)
	require.Equal(t, "CORE_VERSION_UPDATED", extendDriftStatus)
	t.Logf("Extend tenant rule %s correctly drift-flagged with CORE_VERSION_UPDATED!", extendRuleID)

	// 9. Assert In-App Blotter Notification
	notifSvc := NewNotificationService(db)
	extendNotifs, err := notifSvc.GetUnreadNotifications(ctx, extendTenant)
	require.NoError(t, err)
	require.NotEmpty(t, extendNotifs)
	var foundDriftNotif bool
	for _, n := range extendNotifs {
		if n.Kind == NotificationDriftFlag {
			foundDriftNotif = true
			t.Logf("Extend tenant received unread DRIFT_FLAG notification: %q", n.Title)
			break
		}
	}
	require.True(t, foundDriftNotif, "Extend tenant should receive DRIFT_FLAG notification")

	// 10. Verify Point-in-Time Rule Loading
	loader := compliance.NewMultiTenantRuleLoader(db)
	rulesNow, err := loader.LoadTenantActiveRulesAsOf(ctx, testCoreTenant, time.Now().UTC())
	require.NoError(t, err)
	var foundDepRuleNow *compliance.ComplianceRuleRecord
	for i := range rulesNow {
		if rulesNow[i].RuleCode == testRuleCode {
			foundDepRuleNow = &rulesNow[i]
			break
		}
	}
	require.Nil(t, foundDepRuleNow, "Future rule should NOT be active before effective_from")

	rulesFuture, err := loader.LoadTenantActiveRulesAsOf(ctx, testCoreTenant, futureEffective.Add(time.Hour))
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

	// 11. Verify Governance Audit Event contains approved and published hashes
	var govNotes string
	err = db.QueryRowContext(ctx, `
		SELECT steward_notes FROM compliance.governance_audit_event
		WHERE event_type = 'REGULATORY_CHANGE_PUBLISHED' AND rule_id = $1
		ORDER BY created_at DESC LIMIT 1
	`, depRuleID).Scan(&govNotes)
	require.NoError(t, err)
	require.Contains(t, govNotes, draftHash)
	t.Logf("Governance audit event REGULATORY_CHANGE_PUBLISHED verified with cryptographic binding!")
}

// Finding 2 Adversarial Check: Unapproved / Tampered Draft Content Is Strictly Rejected on Publish
func TestRegulatoryWorkflow_ApprovalContentBinding_AdversarialTamper(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := NewService(db)

	// Setup case & rule
	caseCode := fmt.Sprintf("RCC-TAMPER-%s", uuid.New().String()[:8])
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:    caseCode,
		Source:      SourceRegulatorPublication,
		Title:       "Adversarial Tamper Test",
		Description: "Testing cryptographic approval binding",
	})
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	require.NoError(t, err)

	ruleID := uuid.New()
	ruleCode := fmt.Sprintf("TEST_TAMPER_%d", time.Now().UnixNano()%100000)
	astOrig := map[string]interface{}{"type": "METRIC", "path": "pos.weight"}
	paramsOrig := map[string]interface{}{"limit": "0.100000"}

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
			priority, is_active, current_version, ast_condition, parameter_thresholds,
			citation, compiled_bytecode, library_status
		) VALUES (
			$1, $2, 'inherit', $3, 'Tamper Rule', 'PRE_TRADE', 'HARD_BLOCK',
			100, true, 1, '{"type":"METRIC"}'::jsonb, '{"limit":"0.100000"}'::jsonb, 'Approved 10% Citation', ''::bytea, 'ACTIVE'
		)
	`, ruleID, goldTenant, ruleCode)
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.governance_audit_event WHERE rule_id = $1", ruleID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE rule_id = $1", ruleID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule_version WHERE rule_id = $1", ruleID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule WHERE id = $1", ruleID)
	}()

	_ = svc.TriageCase(ctx, TriageRequest{CaseID: c.ID, Classification: ClassificationParameterChange, TriagedBy: "steward"})
	_ = svc.StartReview(ctx, c.ID, "steward", nil)

	// 1. Run corpus gate and approve draft with 10% limit
	corpusRes, err := svc.ExecuteCorpusGate(ctx, c.ID, "steward", []RuleDraft{
		{
			RuleID:        ruleID,
			NewAST:        astOrig,
			NewThresholds: paramsOrig,
			NewCitation:   "Approved 10% Citation",
			TestCorpus: []drift.ScenarioTestCase{
				{
					CaseID:          "TC_BASE",
					MetricSnapshots: map[string]interface{}{"pos.weight": decimal.RequireFromString("0.050000"), "limit": decimal.RequireFromString("0.100000")},
					ExpectedPassed:  true,
					ExpectedAction:  "APPROVED",
				},
			},
		},
	})
	require.NoError(t, err)
	require.True(t, corpusRes.AllPassed)

	err = svc.ApproveCase(ctx, c.ID, "steward", "Approved 10%")
	require.NoError(t, err)

	// 2. Adversarial Attempt: Publish with altered 20% limit without re-running corpus / approval
	tamperedParams := map[string]interface{}{"limit": "0.200000"}
	tamperedDraft := RuleDraft{
		RuleID:        ruleID,
		NewAST:        astOrig,
		NewThresholds: tamperedParams,
		NewCitation:   "Approved 10% Citation",
	}

	_, err = svc.PublishRelease(ctx, PublishRequest{
		CaseID:    c.ID,
		StewardID: "malicious_actor",
		Drafts:    []RuleDraft{tamperedDraft},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "publish release blocked: published content hash")
	t.Logf("Cryptographic binding assertion PASSED: tampered publish payload strictly blocked: %v", err)
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

	caseCode := fmt.Sprintf("RCC-FAIL-%s", uuid.New().String()[:8])
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:        caseCode,
		Source:          SourceRegulatorPublication,
		SourceReference: "ESMA/2026/FAIL",
		Title:           "Failing Draft Case",
		Description:     "Testing corpus gate failure rejection",
		CreatedBy:       "steward_bob",
	})
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	var targetRuleID uuid.UUID
	var curASTBytes, curParamsBytes []byte
	var curCitation string
	err = db.QueryRowContext(ctx, `
		SELECT id, ast_condition, parameter_thresholds, citation
		FROM compliance.compliance_rule
		WHERE tenant_id = $1 AND rule_code = 'UCITS_DEPOSIT_20'
	`, goldTenant).Scan(&targetRuleID, &curASTBytes, &curParamsBytes, &curCitation)
	require.NoError(t, err)

	var astMap, paramsMap map[string]interface{}
	_ = json.Unmarshal(curASTBytes, &astMap)
	_ = json.Unmarshal(curParamsBytes, &paramsMap)

	// 1. Failing test vector
	failingVector := drift.ScenarioTestCase{
		CaseID:      "UCITS_DEPOSIT_20:FAILS_ON_PURPOSE",
		Description: "Failing test vector (exposure 25% > limit 15%)",
		MetricSnapshots: map[string]interface{}{
			"depository_exposure_pct":     decimal.RequireFromString("0.250000"),
			"single_depository_limit_pct": decimal.RequireFromString("0.150000"),
		},
		ExpectedPassed: true, // expects pass but will fail
		ExpectedAction: "APPROVED",
	}

	draftFailing := RuleDraft{
		RuleID:        targetRuleID,
		NewAST:        astMap,
		NewThresholds: paramsMap,
		NewCitation:   curCitation,
		TestCorpus:    []drift.ScenarioTestCase{failingVector},
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

// Gate 5 & Finding 1: Temporal SLA Escalation Does NOT Auto-Close Cases
type mockWorkflowActivities struct {
	escalatedCount int
	mockCase       RegulatoryChangeCase
}

func (m *mockWorkflowActivities) CreateCaseActivity(ctx context.Context, req IntakeRequest) (*RegulatoryChangeCase, error) {
	return &m.mockCase, nil
}

func (m *mockWorkflowActivities) EscalateCaseActivity(ctx context.Context, caseID uuid.UUID, reason string) error {
	m.escalatedCount++
	return nil
}

func (m *mockWorkflowActivities) RejectCaseActivity(ctx context.Context, caseID uuid.UUID, actor string, reason string) error {
	return nil
}

func TestRegulatoryWorkflow_TTLEscalation_NoAutoClose(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	caseID := uuid.New()
	mockActivities := &mockWorkflowActivities{
		mockCase: RegulatoryChangeCase{
			ID:       caseID,
			CaseCode: "RCC-TTL-ESCALATE-001",
			DueAt:    time.Now().Add(30 * 24 * time.Hour),
			Status:   StatusIntaked,
		},
	}

	env.RegisterActivity(mockActivities.CreateCaseActivity)
	env.RegisterActivity(mockActivities.EscalateCaseActivity)
	env.RegisterActivity(mockActivities.RejectCaseActivity)

	// Send an explicit steward reject signal 40 days in the future (proving the workflow stays OPEN after 30-day TTL)
	env.RegisterDelayedCallback(func() {
		// Verify escalation fired
		require.GreaterOrEqual(t, mockActivities.escalatedCount, 1, "SLA escalation must have fired at 30 days")

		// Case was NOT auto-closed: send human steward rejection signal at day 40
		env.SignalWorkflow(SignalRejectCase, RejectSignalPayload{
			Actor:  "steward_alice",
			Reason: "Manually rejected after review of escalated case",
		})
	}, 40*24*time.Hour)

	env.ExecuteWorkflow(RegulatoryChangeWorkflow, RegulatoryWorkflowInput{
		IntakeReq: IntakeRequest{
			CaseCode:    "RCC-TTL-ESCALATE-001",
			Title:       "Test SLA Escalation Without Auto-Close",
			Description: "SLA expiry must escalate alerts but never auto-close substantive cases",
			TTL:         30 * 24 * time.Hour,
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var res RegulatoryWorkflowResult
	err := env.GetWorkflowResult(&res)
	require.NoError(t, err)
	require.Equal(t, StatusRejected, res.Status)
	require.GreaterOrEqual(t, mockActivities.escalatedCount, 1, "Escalation activity must have executed without auto-closing the workflow")
	t.Logf("Temporal TTL Escalation verified: case escalated at 30d, remained OPEN, and was resolved by human steward at 40d!")
}

// Gate 6: Custom Rule Version Evolution & Structural Trigger
func TestRegulatoryWorkflow_CustomRuleVersionEvolution(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testTenant := uuid.New()
	ruleID := uuid.New()
	ruleCode := fmt.Sprintf("CUSTOM_RATE_LIMIT_%d", time.Now().UnixNano()%100000)

	astV1 := []byte(`{"left":{"path":"order.rate_per_sec","type":"METRIC"},"operator":"GREATER_THAN","right":{"name":"max_rate","type":"PARAM"},"type":"COMPARISON"}`)
	paramsV1 := []byte(`{"max_rate":"100"}`)
	citationV1 := "Tenant House Rule v1"
	hashV1, err := canonical.ComputeRuleContentHashFromRaw(astV1, paramsV1, citationV1)
	require.NoError(t, err)
	bytecodeHash := canonical.ComputeBytecodeHash(nil)

	// Insert custom rule v1
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
			priority, is_active, current_version, ast_condition, parameter_thresholds,
			citation, compiled_bytecode, library_status
		) VALUES (
			$1, $2, 'custom', $3, 'Custom Rate Limit', 'PRE_TRADE', 'HARD_BLOCK',
			95, true, 1, $4::jsonb, $5::jsonb, $6, ''::bytea, 'ACTIVE'
		)
	`, ruleID, testTenant, ruleCode, string(astV1), string(paramsV1), citationV1)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
			created_by, created_at
		) VALUES (
			$1, 1, $2, $3::jsonb, $4::jsonb,
			$5, now(), null, $6, $7, 'custom_steward', now()
		)
	`, ruleID, testTenant, string(astV1), string(paramsV1), citationV1, hashV1, bytecodeHash)
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "UPDATE compliance.compliance_rule SET valid_to = now() WHERE id = $1", ruleID)
	}()

	// 1. Transactional Evolution to v2
	paramsV2 := []byte(`{"max_rate":"150"}`)
	citationV2 := "Tenant House Rule v2 (Amended)"
	hashV2, err := canonical.ComputeRuleContentHashFromRaw(astV1, paramsV2, citationV2)
	require.NoError(t, err)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
			created_by, created_at
		) VALUES (
			$1, 2, $2, $3::jsonb, $4::jsonb,
			$5, now(), null, $6, $7, 'custom_steward', now()
		)
	`, ruleID, testTenant, string(astV1), string(paramsV2), citationV2, hashV2, bytecodeHash)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.compliance_rule
		SET 
			current_version = 2,
			parameter_thresholds = $1::jsonb,
			citation = $2,
			updated_at = now()
		WHERE id = $3
	`, string(paramsV2), citationV2, ruleID)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)
	t.Logf("Custom rule evolution to version 2 succeeded with current_version structural trigger validation!")

	// 2. Evaluation event referencing v2 passes FK constraint
	evalInput := canonical.EvaluationHashInput{
		LineageID:       uuid.New(),
		TenantID:        testTenant,
		RuleID:          ruleID,
		RuleVersion:     2,
		RuleContentHash: hashV2,
		ActionTaken:     "APPROVED",
		Passed:          true,
		InputParams: map[string]interface{}{
			"accountId": "acc-custom-1",
		},
		MetricSnapshots: map[string]interface{}{
			"rate": decimal.RequireFromString("120.000000"),
		},
	}
	evalHash, err := canonical.ComputeEvaluationHash(evalInput)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, rule_id, rule_version, rule_content_hash,
			action_taken, passed, evaluation_hash, latency_micros, evaluated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, 2, $4,
			'APPROVED', true, $5, 120, now()
		)
	`, evalInput.LineageID, testTenant, ruleID, hashV2, evalHash)
	require.NoError(t, err)
	t.Logf("Evaluation event for custom rule version 2 passed FK RESTRICT constraint cleanly!")
}

// Gate 7: Populated Structure-Aware Steward Diff View at UNDER_REVIEW
func TestRegulatoryWorkflow_StewardPresentationView(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := NewService(db)
	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	require.NoError(t, err)

	var ucits5ID uuid.UUID
	var curASTBytes, curParamsBytes []byte
	var curCitation string
	err = db.QueryRowContext(ctx, `
		SELECT id, ast_condition, parameter_thresholds, citation
		FROM compliance.compliance_rule
		WHERE tenant_id = $1 AND rule_code = 'UCITS_ISSUER_5'
	`, goldTenant).Scan(&ucits5ID, &curASTBytes, &curParamsBytes, &curCitation)
	require.NoError(t, err)

	caseCode := fmt.Sprintf("RCC-VIEW-%s", uuid.New().String()[:8])
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
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	_ = svc.TriageCase(ctx, TriageRequest{
		CaseID:          c.ID,
		Classification:  ClassificationParameterChange,
		TriageNotes:     "Lookthrough aggregation threshold clarified",
		TriagedBy:       "steward_alice",
		AffectedRuleIDs: []uuid.UUID{ucits5ID},
	})

	_ = svc.StartReview(ctx, c.ID, "steward_alice", nil)

	// Save draft with modified threshold & citation
	var proposedAST map[string]interface{}
	_ = json.Unmarshal(curASTBytes, &proposedAST)
	proposedParams := map[string]interface{}{
		"issuer_limit_pct":          "0.050000",
		"lookthrough":               true,
		"aggregate_across_accounts": true,
		"max_umbrella_layers":       "3",
	}
	proposedCitation := curCitation + " (Amended Lookthrough Guidelines 2026)"

	err = svc.SaveDrafts(ctx, c.ID, []RuleDraft{
		{
			RuleID:        ucits5ID,
			NewAST:        proposedAST,
			NewThresholds: proposedParams,
			NewCitation:   proposedCitation,
		},
	})
	require.NoError(t, err)

	view, err := svc.GetStewardTriageView(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, caseCode, view.CaseCode)
	require.Equal(t, StatusUnderReview, view.Status)
	require.NotEmpty(t, view.AffectedRules)
	require.NotEmpty(t, view.DiffViews, "UNDER_REVIEW view MUST have populated DiffViews for steward verification")

	diff := view.DiffViews[0]
	require.Equal(t, "UCITS_ISSUER_5", diff.RuleCode)
	require.Equal(t, "0.050000", diff.ProposedThresholds["issuer_limit_pct"])
	require.Equal(t, "3", diff.ProposedThresholds["max_umbrella_layers"])
	require.Equal(t, proposedCitation, diff.ProposedCitation)
	require.Len(t, diff.ProposedContentHash, 64)

	viewJSON, _ := json.MarshalIndent(view, "", "  ")
	t.Logf("Structured Steward Presentation View with Populated DiffViews (Page Designer Ready):\n%s", string(viewJSON))
}

// Handling of NEW_RULE_REQUIRED Classification
func TestRegulatoryWorkflow_NewRuleRequiredRouting(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := NewService(db)

	caseCode := fmt.Sprintf("RCC-NEW-RULE-%s", uuid.New().String()[:8])
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:    caseCode,
		Source:      SourceRegulatorPublication,
		Title:       "Crypto-Asset Concentration Mandate (MiCA II)",
		Description: "New regulatory limit for crypto-asset exposure requiring brand new rule creation",
		CreatedBy:   "steward_alice",
	})
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	err = svc.TriageCase(ctx, TriageRequest{
		CaseID:         c.ID,
		Classification: ClassificationNewRuleRequired,
		TriageNotes:    "Requires authoring of MICA_CRYPTO_LIMIT_10 in core seed and scenario corpus",
		TriagedBy:      "steward_alice",
	})
	require.NoError(t, err)

	updatedCase, err := svc.GetCase(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, StatusNewRuleBacklog, updatedCase.Status, "NEW_RULE_REQUIRED must transition case to NEW_RULE_BACKLOG")

	var eventCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM compliance.regulatory_case_event
		WHERE case_id = $1 AND event_type = 'NEW_RULE_ROUTED'
	`, c.ID).Scan(&eventCount)
	require.NoError(t, err)
	require.Equal(t, 1, eventCount)
	t.Logf("NEW_RULE_REQUIRED routing to NEW_RULE_BACKLOG verified!")
}

// Operational SLA Metrics View
func TestRegulatoryWorkflow_OperationalMetricsView(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := NewService(db)

	caseCode := fmt.Sprintf("RCC-OVERDUE-%s", uuid.New().String()[:8])
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:    caseCode,
		Source:      SourceScheduledReview,
		Title:       "Annual Review of CSDR Settlement Limits",
		Description: "Overdue review tracking test",
		CreatedBy:   "steward_alice",
	})
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, "UPDATE compliance.regulatory_change_case SET due_at = now() - interval '2 hours' WHERE id = $1", c.ID)
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	err = svc.EscalateCase(ctx, c.ID, "Past SLA deadline")
	require.NoError(t, err)

	cases, err := svc.GetUnaddressedCases(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, cases)

	var foundCase *UnaddressedRegulatoryCase
	for i := range cases {
		if cases[i].CaseCode == caseCode {
			foundCase = &cases[i]
			break
		}
	}
	require.NotNil(t, foundCase, "Overdue case must appear in compliance.v_unaddressed_regulatory_cases")
	require.True(t, foundCase.IsOverdue)
	require.True(t, foundCase.IsEscalated)
	require.Greater(t, foundCase.OverdueSeconds, int64(0))
	t.Logf("Operational metrics view verified: Case %s is overdue by %d seconds (escalated %d times)!", foundCase.CaseCode, foundCase.OverdueSeconds, foundCase.EscalationCount)
}

// Webhook Delivery & HMAC-SHA256 Signature Verification with Retry
func TestRegulatoryWorkflow_WebhookDeliveryAndSignatureVerification(t *testing.T) {
	var attempts int32
	secret := []byte("super-secure-webhook-secret-key-12345")
	payload := []byte(`{"event":"REGULATORY_CHANGE_PUBLISHED","case_code":"RCC-2026-001","rule_code":"UCITS_ISSUER_5","new_version":2}`)

	// Mock receiver server that fails on attempt 1 with 500, succeeds on attempt 2
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentAttempt := atomic.AddInt32(&attempts, 1)
		sig := r.Header.Get("X-Uisce-Signature-SHA256")

		if !VerifyWebhookSignature(secret, payload, sig) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid signature"))
			return
		}

		if currentAttempt == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("transient server error"))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("accepted"))
	}))
	defer server.Close()

	notifSvc := NewNotificationService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := notifSvc.DispatchWebhookWithRetry(ctx, server.URL, secret, payload, 3, 20*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(&attempts), "Webhook dispatcher must retry on transient 500 error and succeed on second attempt")
	t.Logf("Webhook delivery & HMAC-SHA256 signature verification with retry verified 100%%!")
}

// Step 1: Rule Snapshot Reconciler Sweep Test (Detects injected corruption & alerts)
func TestRuleSnapshotReconciler_DetectsTamperedHash(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reconciler := jobs.NewRuleSnapshotReconciler(db)

	// 1. Initial baseline run across all 50 rules
	initialReport, err := reconciler.ReconcileAll(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, initialReport.TotalScanned, 50)
	require.Equal(t, 0, initialReport.Mismatched, "Initial baseline should have 0 hash mismatches")
	t.Logf("Baseline snapshot reconciler sweep verified: %d/%d rules match Go canonical hash!", initialReport.Matched, initialReport.TotalScanned)

	// 2. Insert test rule with deliberately tampered snapshot hash on an isolated test tenant
	testTenant := uuid.New()
	testRuleID := uuid.New()
	testRuleCode := fmt.Sprintf("TEST_RECON_%d", time.Now().UnixNano()%100000)

	_, err = db.ExecContext(ctx, `
		INSERT INTO public.tenants (id, name, display_name, gold_copy)
		VALUES ($1, $2, $2, false)
	`, testTenant, fmt.Sprintf("recon_%s", testTenant.String()[:8]))
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
			priority, is_active, current_version, ast_condition, parameter_thresholds,
			citation, compiled_bytecode, library_status
		) VALUES (
			$1, $2, 'inherit', $3, 'Reconcile Test Rule', 'PRE_TRADE', 'HARD_BLOCK',
			100, true, 1, '{"type":"METRIC","path":"pos.weight"}'::jsonb, '{"limit":"0.100000"}'::jsonb, 'Citation', '\x00'::bytea, 'ACTIVE'
		)
	`, testRuleID, testTenant, testRuleCode)
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_notification WHERE kind = 'SYSTEM' AND title LIKE '%CRITICAL: Rule Version Hash Mismatch%'")
		_, _ = db.ExecContext(context.Background(), "UPDATE compliance.compliance_rule SET valid_to = now(), is_active = false WHERE id = $1", testRuleID)
		_, _ = db.ExecContext(context.Background(), "UPDATE public.tenants SET is_active = false WHERE id = $1", testTenant)
	}()

	tamperedHash := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, content_hash, compiled_bytecode_hash, created_by
		) VALUES (
			$1, 1, $2, '{"type":"METRIC","path":"pos.weight"}'::jsonb, '{"limit":"0.100000"}'::jsonb,
			'Citation', now(), $3, $3, 'tester'
		)
	`, testRuleID, testTenant, tamperedHash)
	require.NoError(t, err)

	// 3. Re-run sweep and assert detection of injected divergence
	tamperedReport, err := reconciler.ReconcileAll(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, tamperedReport.Mismatched, 1, "Reconciler sweep must detect injected hash divergence")

	foundMismatch := false
	for _, m := range tamperedReport.Mismatches {
		if m.RuleID == testRuleID {
			foundMismatch = true
			require.Equal(t, tamperedHash, m.StoredHash)
			require.NotEmpty(t, m.ComputedHash)
			require.NotEqual(t, tamperedHash, m.ComputedHash)
			break
		}
	}
	require.True(t, foundMismatch, "Reconciler must pinpoint the exact corrupted rule version")

	// 4. Verify critical alert notification was written to compliance_notification
	var notifCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM compliance.compliance_notification
		WHERE kind = 'SYSTEM' AND title LIKE '%CRITICAL: Rule Version Hash Mismatch%'
	`).Scan(&notifCount)
	require.NoError(t, err)
	require.GreaterOrEqual(t, notifCount, 1, "Critical incident notification must be logged into compliance_notification table")

	// 5. Verify Watchdog Liveness Gauges & Targeted Debezium Event Verification
	lastSweep, durationMs, totalSweeps := reconciler.GetWatchdogMetrics()
	require.False(t, lastSweep.IsZero(), "Watchdog last_sweep timestamp must be set")
	require.GreaterOrEqual(t, durationMs, int64(0))
	require.GreaterOrEqual(t, totalSweeps, uint64(1))
	require.False(t, reconciler.IsWatchdogStale(15*time.Minute), "Reconciler must NOT be stale immediately after sweep")

	eventMismatch, err := reconciler.ReconcileRuleEvent(ctx, testRuleID)
	require.NoError(t, err)
	require.NotNil(t, eventMismatch, "Debezium targeted single-rule reconciliation must detect corruption immediately")
	require.Equal(t, testRuleID, eventMismatch.RuleID)

	t.Logf("RuleSnapshotReconciler detection, watchdog liveness metrics, and Debezium event sweep fully verified!")
}

// Step 2: Corpus-Run -> Approval Binding (Modifying draft after corpus run prevents approval without re-run & emits DRAFT_REVISED)
func TestRegulatoryWorkflow_CorpusApprovalBinding_Tamper(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := NewService(db)

	caseCode := fmt.Sprintf("RCC-BIND-%s", uuid.New().String()[:8])
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:    caseCode,
		Source:      SourceRegulatorPublication,
		Title:       "Corpus Approval Binding Test",
		Description: "Verify approval rejects modified draft post-corpus execution",
	})
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	ruleID := uuid.New()
	ruleCode := fmt.Sprintf("TEST_BIND_%s", uuid.New().String()[:8])
	astOrig := map[string]interface{}{"type": "METRIC", "path": "pos.issuer_pct"}
	paramsOrig := map[string]interface{}{"issuer_limit_pct": "0.050000"}

	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
			priority, is_active, current_version, ast_condition, parameter_thresholds,
			citation, compiled_bytecode, library_status
		) VALUES (
			$1, $2, 'inherit', $3, 'Bind Test Rule', 'PRE_TRADE', 'HARD_BLOCK',
			100, true, 1, $4::jsonb, $5::jsonb, 'Citation 1', '\x00'::bytea, 'ACTIVE'
		)
	`, ruleID, goldTenant, ruleCode, `{"type":"METRIC","path":"pos.issuer_pct"}`, `{"issuer_limit_pct":"0.050000"}`)
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.governance_audit_event WHERE rule_id = $1", ruleID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE rule_id = $1", ruleID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule_version WHERE rule_id = $1", ruleID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule WHERE id = $1", ruleID)
	}()

	_ = svc.TriageCase(ctx, TriageRequest{CaseID: c.ID, Classification: ClassificationParameterChange, TriagedBy: "steward"})
	_ = svc.StartReview(ctx, c.ID, "steward", nil)

	// 1. Run corpus on initial draft (5% limit)
	initialDraft := RuleDraft{
		RuleID:        ruleID,
		NewAST:        astOrig,
		NewThresholds: paramsOrig,
		NewCitation:   "Citation 1",
		TestCorpus: []drift.ScenarioTestCase{
			{
				CaseID:          "TC_1",
				MetricSnapshots: map[string]interface{}{"pos.issuer_pct": decimal.RequireFromString("0.040000"), "issuer_limit_pct": decimal.RequireFromString("0.050000")},
				ExpectedPassed:  true,
				ExpectedAction:  "APPROVED",
			},
		},
	}
	res, err := svc.ExecuteCorpusGate(ctx, c.ID, "steward", []RuleDraft{initialDraft})
	require.NoError(t, err)
	require.True(t, res.AllPassed)

	// 2. Tamper draft in storage without re-running corpus (update to 3% limit)
	tamperedParams := map[string]interface{}{"issuer_limit_pct": "0.030000"}
	err = svc.SaveDrafts(ctx, c.ID, []RuleDraft{
		{
			RuleID:        ruleID,
			NewAST:        astOrig,
			NewThresholds: tamperedParams,
			NewCitation:   "Citation 1",
		},
	})
	require.NoError(t, err)

	// 3. Verify DRAFT_REVISED audit event was automatically emitted by the database trigger
	var revisedCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM compliance.regulatory_case_event
		WHERE case_id = $1 AND event_type = 'DRAFT_REVISED'
	`, c.ID).Scan(&revisedCount)
	require.NoError(t, err)
	require.Equal(t, 1, revisedCount, "Modifying draft post-corpus execution must write a DRAFT_REVISED audit event")
	t.Logf("DRAFT_REVISED audit event verified in append-only ledger!")

	// 4. Attempt to approve case without re-running corpus gate -> must fail!
	err = svc.ApproveCase(ctx, c.ID, "steward", "Approval with stale corpus run")
	require.Error(t, err)
	require.Contains(t, err.Error(), "approval blocked: draft content for rule")
	require.Contains(t, err.Error(), "has changed since last corpus execution")
	t.Logf("Corpus-Run -> Approval binding verified: %v", err)
}

// Step 2: Draft Update Guard Trigger (Cannot mutate proposed content on an approved draft)
func TestRegulatoryWorkflow_DraftUpdateGuardTrigger(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	caseID := uuid.New()
	var ruleID uuid.UUID
	err := db.QueryRowContext(ctx, "SELECT id FROM compliance.compliance_rule WHERE rule_code = 'UCITS_ISSUER_5'").Scan(&ruleID)
	require.NoError(t, err)

	// Direct test on regulatory_draft_rule table trigger
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_change_case (id, case_code, source, title, description, due_at, created_by)
		VALUES ($1, $2, 'INTERNAL', 'Trigger Test Case', 'Desc', now() + interval '30 days', 'tester')
	`, caseID, fmt.Sprintf("RCC-TRG-%s", uuid.New().String()[:8]))
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE case_id = $1", caseID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", caseID)
	}()

	astBytes := []byte(`{"type":"METRIC","path":"pos.exposure"}`)
	paramBytes := []byte(`{"limit":"0.100000"}`)
	h, _ := canonical.ComputeRuleContentHashFromRaw(astBytes, paramBytes, "Cit")

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.regulatory_draft_rule (
			id, case_id, rule_id, proposed_ast, proposed_parameter_thresholds,
			proposed_citation, proposed_content_hash, is_approved
		) VALUES (
			gen_random_uuid(), $1, $2, $3::jsonb, $4::jsonb, 'Cit', $5, true
		)
	`, caseID, ruleID, string(astBytes), string(paramBytes), h)
	require.NoError(t, err)

	// Attempting in-place modification of proposed_parameter_thresholds on approved draft must fail
	_, err = db.ExecContext(ctx, `
		UPDATE compliance.regulatory_draft_rule
		SET proposed_parameter_thresholds = '{"limit":"0.200000"}'::jsonb
		WHERE case_id = $1 AND rule_id = $2
	`, caseID, ruleID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Audit Violation: Cannot modify proposed content on an approved regulatory draft")
	t.Logf("Draft modification guard trigger verified: illegal update blocked on approved draft!")
}

// Step 2: Threshold-Aware Diff Summary in GetStewardReviewView
func TestRegulatoryWorkflow_ThresholdDiffSummary(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := NewService(db)

	var depRuleID uuid.UUID
	err := db.QueryRowContext(ctx, "SELECT id FROM compliance.compliance_rule WHERE rule_code = 'UCITS_ISSUER_5'").Scan(&depRuleID)
	require.NoError(t, err)

	caseCode := fmt.Sprintf("RCC-DIFF-%s", uuid.New().String()[:8])
	c, err := svc.CreateCase(ctx, IntakeRequest{
		CaseCode:    caseCode,
		Source:      SourceRegulatorPublication,
		Title:       "Threshold Diff Summary Test",
		Description: "Testing threshold enumeration in diff view",
	})
	require.NoError(t, err)

	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_case_event WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_draft_rule WHERE case_id = $1", c.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.regulatory_change_case WHERE id = $1", c.ID)
	}()

	_ = svc.TriageCase(ctx, TriageRequest{CaseID: c.ID, Classification: ClassificationParameterChange, TriagedBy: "steward", AffectedRuleIDs: []uuid.UUID{depRuleID}})
	_ = svc.StartReview(ctx, c.ID, "steward", nil)

	// Fetch current rule AST and parameters
	var astBytes []byte
	var citation string
	err = db.QueryRowContext(ctx, "SELECT ast_condition, citation FROM compliance.compliance_rule WHERE id = $1", depRuleID).Scan(&astBytes, &citation)
	require.NoError(t, err)

	var astMap map[string]interface{}
	_ = json.Unmarshal(astBytes, &astMap)

	// Propose changed thresholds with UNCHANGED AST (0.050000 -> 0.040000)
	proposedParams := map[string]interface{}{
		"issuer_limit_pct":          "0.040000",
		"lookthrough":               true,
		"aggregate_across_accounts": true,
	}

	err = svc.SaveDrafts(ctx, c.ID, []RuleDraft{
		{
			RuleID:        depRuleID,
			NewAST:        astMap,
			NewThresholds: proposedParams,
			NewCitation:   citation,
		},
	})
	require.NoError(t, err)

	// Get Steward Review View
	view, err := svc.GetStewardReviewView(ctx, c.ID)
	require.NoError(t, err)
	require.NotEmpty(t, view.DiffViews)

	diffView := view.DiffViews[0]
	require.NotNil(t, diffView.Diff)
	require.Contains(t, diffView.Diff.SummaryText, "Threshold modifications: Param 'issuer_limit_pct': '0.050000' -> '0.040000'")
	t.Logf("Threshold-aware diff summary verified: %q", diffView.Diff.SummaryText)
}

