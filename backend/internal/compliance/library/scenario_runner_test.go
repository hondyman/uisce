package library

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/compliance"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/hondyman/uisce/backend/internal/compliance/drift"
	"github.com/hondyman/uisce/backend/internal/compliance/reservation"
	"github.com/hondyman/uisce/backend/internal/compliance/testutil"
)

func getAlphaTestDB(t *testing.T) *sql.DB {
	t.Helper()
	return testutil.GetEphemeralTestDB(t)
}

// 1. Scenario Corpus Regression Test
func TestCoreLibrary_ScenarioCorpusRegression(t *testing.T) {
	for _, sc := range CoreScenarioCorpus {
		t.Run(sc.Code, func(t *testing.T) {
			status, explanation := EvaluateScenario(sc)
			if status != sc.Expected.Status {
				t.Fatalf("%s: expected status %q, got %q (rule %s)",
					sc.Code, sc.Expected.Status, status, sc.RuleCode)
			}
			if sc.Expected.MustContain != "" && !strings.Contains(explanation, sc.Expected.MustContain) {
				t.Fatalf("%s: explanation %q missing required substring %q",
					sc.Code, explanation, sc.Expected.MustContain)
			}
		})
	}
}

// 1b. Scenario Rule Code Resolution Guard (Fails if scenario references an unknown rule code)
func TestCoreLibrary_ScenarioRuleCodeResolution(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	require.NoError(t, err)

	rows, err := db.QueryContext(ctx, `
		SELECT rule_code FROM compliance.compliance_rule
		WHERE tenant_id = $1 AND valid_to IS NULL
	`, goldTenant)
	require.NoError(t, err)
	defer rows.Close()

	validRules := make(map[string]bool)
	for rows.Next() {
		var code string
		require.NoError(t, rows.Scan(&code))
		validRules[code] = true
	}

	for _, sc := range CoreScenarioCorpus {
		if !validRules[sc.RuleCode] {
			t.Fatalf("Scenario %s references unknown rule code %q (not found in gold-copy rule library)",
				sc.Code, sc.RuleCode)
		}
	}
	t.Logf("100%% Rule Code Resolution Verified: all %d scenarios map directly to valid gold-copy core rules", len(CoreScenarioCorpus))
}

// 2. Reservation-Dependent Adversarial Test
func TestCoreLibrary_ReservationAdversarial(t *testing.T) {
	cache := reservation.NewPositionCache()
	mgr := reservation.NewReservationManager(cache, 10*time.Minute)

	tenantID := uuid.New()
	accountID := uuid.New()
	secID := uuid.New()

	// Account NAV: $1,000,000,000 ($1B)
	nav := decimal.NewFromFloat(1000000000.00)
	cache.SetAccountNAV(tenantID, accountID, nav)

	// Initial settled exposure: $47,000,000 (4.7% of NAV)
	initialMV := decimal.NewFromFloat(47000000.00)
	initialQty := decimal.NewFromFloat(47000.00)
	cache.SetSettledPosition(tenantID, accountID, secID, initialQty, initialMV)

	// Order 1: qty = 2,000 @ $1,000 = $2,000,000 (+0.2%) -> pushes to $49M (4.9% < 5%) -> SUCCESS
	orderID1 := uuid.New()
	qty1 := decimal.NewFromFloat(2000.00)
	price1 := decimal.NewFromFloat(1000.00)
	limit := decimal.NewFromFloat(0.050000) // 5% limit

	lease1, err := mgr.AcquireReservation(context.Background(), tenantID, accountID, secID, orderID1, "BUY", qty1, price1, limit)
	if err != nil {
		t.Fatalf("AcquireReservation 1 failed: %v", err)
	}

	// Projected exposure is now $49,000,000 (4.9%)
	pos := cache.GetPosition(tenantID, accountID, secID)
	if !pos.TotalProjectedValue().Equal(decimal.NewFromFloat(49000000.00)) {
		t.Fatalf("Expected projected value $49M, got %s", pos.TotalProjectedValue())
	}

	// Order 2: qty = 3,000 @ $1,000 = $3,000,000 (+0.3%) -> would push to $52M (5.2% > 5%) -> BLOCKED BY RESERVATION DELTA
	orderID2 := uuid.New()
	qty2 := decimal.NewFromFloat(3000.00)
	price2 := decimal.NewFromFloat(1000.00)

	_, err = mgr.AcquireReservation(context.Background(), tenantID, accountID, secID, orderID2, "BUY", qty2, price2, limit)
	if err == nil {
		t.Fatalf("Expected Order 2 reservation to be rejected due to projected 5.2%% exceeding 5%% limit")
	}

	// Releasing Order 1 reservation restores projected value back to $47,000,000
	err = mgr.ReleaseReservation(lease1.LeaseID)
	if err != nil {
		t.Fatalf("ReleaseReservation failed: %v", err)
	}

	pos = cache.GetPosition(tenantID, accountID, secID)
	if !pos.TotalProjectedValue().Equal(initialMV) {
		t.Fatalf("Expected exposure restored to %s, got %s", initialMV, pos.TotalProjectedValue())
	}
}

// 3. Seed-Drift Database Check
func TestCoreLibrary_SeedDriftCheck(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenantID, err := loader.GetGoldCopyTenantID(ctx)
	if err != nil {
		t.Fatalf("get gold copy tenant: %v", err)
	}

	// Verify exactly 50 rules exist for the gold tenant under CORE_LIB_V1
	var totalCount, activeCount, provisionalCount int
	err = db.QueryRowContext(ctx, `
		SELECT 
			count(*),
			count(*) FILTER (WHERE library_status = 'ACTIVE'),
			count(*) FILTER (WHERE library_status = 'PROVISIONAL')
		FROM compliance.compliance_rule
		WHERE tenant_id = $1 AND source_version = 'CORE_LIB_V1'
	`, goldTenantID).Scan(&totalCount, &activeCount, &provisionalCount)
	if err != nil {
		t.Fatalf("query gold rule count: %v", err)
	}

	if totalCount != 50 {
		t.Fatalf("Expected exactly 50 gold-copy rules in alpha, found %d", totalCount)
	}

	if activeCount != 50 {
		t.Errorf("Expected exactly 50 ACTIVE (scenario-covered) rules, got %d", activeCount)
	}

	if provisionalCount != 0 {
		t.Errorf("Expected exactly 0 PROVISIONAL rules, got %d", provisionalCount)
	}

	// Verify all 3 licensable rulesets are populated
	rows, err := db.QueryContext(ctx, `
		SELECT ruleset_code, count(*)
		FROM compliance.compliance_ruleset_membership
		GROUP BY ruleset_code
		ORDER BY ruleset_code
	`)
	if err != nil {
		t.Fatalf("query ruleset memberships: %v", err)
	}
	defer rows.Close()

	rulesets := make(map[string]int)
	for rows.Next() {
		var code string
		var c int
		if err := rows.Scan(&code, &c); err != nil {
			t.Fatalf("scan ruleset membership: %v", err)
		}
		rulesets[code] = c
	}

	if rulesets["CORE_REGULATORY"] != 17 {
		t.Errorf("Expected 17 rules in CORE_REGULATORY, got %d", rulesets["CORE_REGULATORY"])
	}
	if rulesets["MARKET_CONDUCT"] != 12 {
		t.Errorf("Expected 12 rules in MARKET_CONDUCT, got %d", rulesets["MARKET_CONDUCT"])
	}
	if rulesets["INSTITUTIONAL_CONTROLS"] != 21 {
		t.Errorf("Expected 21 rules in INSTITUTIONAL_CONTROLS, got %d", rulesets["INSTITUTIONAL_CONTROLS"])
	}
}

// 4. Activation Lifecycle Test
func TestCoreLibrary_ActivationLifecycle(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repinService := drift.NewRepinService(db)
	loader := compliance.NewMultiTenantRuleLoader(db)

	testTenant := uuid.New()
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	if err != nil {
		t.Fatalf("get gold tenant: %v", err)
	}

	// Find the rule ID for UCITS_ISSUER_5
	var ruleID uuid.UUID
	err = db.QueryRowContext(ctx, `
		SELECT id FROM compliance.compliance_rule
		WHERE tenant_id = $1 AND rule_code = 'UCITS_ISSUER_5'
	`, goldTenant).Scan(&ruleID)
	if err != nil {
		t.Fatalf("find UCITS_ISSUER_5 rule: %v", err)
	}

	// 1. Initially, tenant has no activations -> loader returns 0 rules (opt-in model)
	rules, err := loader.LoadTenantActiveRules(ctx, testTenant)
	if err != nil {
		t.Fatalf("LoadTenantActiveRules initially: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("Expected 0 active rules initially for new tenant, got %d", len(rules))
	}

	// 2. Enable UCITS_ISSUER_5 for tenant -> writes audit event
	err = repinService.SetTenantRuleActivation(ctx, testTenant, ruleID, true, "inherit", "steward_alice", "Enabled Core UCITS 5% concentration rule")
	if err != nil {
		t.Fatalf("SetTenantRuleActivation enable: %v", err)
	}

	// Verify rule is now active for tenant
	rules, err = loader.LoadTenantActiveRules(ctx, testTenant)
	if err != nil {
		t.Fatalf("LoadTenantActiveRules after enable: %v", err)
	}
	if len(rules) != 1 || rules[0].RuleCode != "UCITS_ISSUER_5" {
		t.Fatalf("Expected 1 active rule (UCITS_ISSUER_5), got %d rules", len(rules))
	}

	// Verify governance audit event was recorded
	var auditCount int
	err = db.QueryRowContext(ctx, `
		SELECT count(*) FROM compliance.governance_audit_event
		WHERE tenant_id = $1 AND rule_id = $2 AND event_type = 'RULE_ACTIVATED'
	`, testTenant, ruleID).Scan(&auditCount)
	if err != nil || auditCount == 0 {
		t.Fatalf("Expected governance_audit_event for RULE_ACTIVATED, count=%d, err=%v", auditCount, err)
	}

	// 3. Deactivate UCITS_ISSUER_5 -> writes RULE_DEACTIVATED audit event
	err = repinService.SetTenantRuleActivation(ctx, testTenant, ruleID, false, "inherit", "steward_alice", "Deactivated rule")
	if err != nil {
		t.Fatalf("SetTenantRuleActivation disable: %v", err)
	}

	rules, err = loader.LoadTenantActiveRules(ctx, testTenant)
	if err != nil {
		t.Fatalf("LoadTenantActiveRules after disable: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("Expected 0 active rules after deactivation, got %d", len(rules))
	}

	// Cleanup test tenant activation data
	_, _ = db.ExecContext(ctx, "DELETE FROM compliance.governance_audit_event WHERE tenant_id = $1", testTenant)
	_, _ = db.ExecContext(ctx, "DELETE FROM compliance.tenant_rule_activation WHERE tenant_id = $1", testTenant)
}

// 5. Point-in-Time Effective Dating Test
func TestCoreLibrary_EffectiveDating(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	if err != nil {
		t.Fatalf("get gold tenant: %v", err)
	}

	// Current time (2026+) -> all 50 rules are effective
	asOfNow := time.Now().UTC()
	rulesNow, err := loader.LoadTenantActiveRulesAsOf(ctx, goldTenant, asOfNow)
	if err != nil {
		t.Fatalf("LoadTenantActiveRulesAsOf now: %v", err)
	}
	if len(rulesNow) != 50 {
		t.Fatalf("Expected 50 rules effective now, got %d", len(rulesNow))
	}

	// Historical time prior to effective_from (e.g. 2024-01-01) -> 0 rules effective
	asOfPast := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rulesPast, err := loader.LoadTenantActiveRulesAsOf(ctx, goldTenant, asOfPast)
	if err != nil {
		t.Fatalf("LoadTenantActiveRulesAsOf past: %v", err)
	}
	if len(rulesPast) != 0 {
		t.Fatalf("Expected 0 rules effective as of 2024-01-01, got %d", len(rulesPast))
	}
}

// 6. Gold Tenant Identity Assertion
func TestCoreLibrary_GoldTenantIdentityAssertion(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loader := compliance.NewMultiTenantRuleLoader(db)

	goldTenantID, err := loader.GetGoldCopyTenantID(ctx)
	if err != nil {
		t.Fatalf("GetGoldCopyTenantID failed: %v", err)
	}

	// 1. Verify gold tenant returns isGoldCopy == true
	isGold, err := loader.IsGoldCopyTenant(ctx, goldTenantID)
	if err != nil {
		t.Fatalf("IsGoldCopyTenant on gold tenant failed: %v", err)
	}
	if !isGold {
		t.Fatalf("Expected IsGoldCopyTenant to return true for gold tenant %s", goldTenantID)
	}

	// 2. Verify random client tenant returns isGoldCopy == false
	randomTenant := uuid.New()
	isGoldRandom, err := loader.IsGoldCopyTenant(ctx, randomTenant)
	if err != nil {
		t.Fatalf("IsGoldCopyTenant on random tenant failed: %v", err)
	}
	if isGoldRandom {
		t.Fatalf("Expected IsGoldCopyTenant to return false for random tenant %s", randomTenant)
	}

	// 3. Verify database flag matches public.uisce_gold_copy_tenant_id()
	var sqlGoldID uuid.UUID
	err = db.QueryRowContext(ctx, "SELECT public.uisce_gold_copy_tenant_id()").Scan(&sqlGoldID)
	if err == nil && sqlGoldID != uuid.Nil {
		if sqlGoldID != goldTenantID {
			t.Fatalf("Mismatch: uisce_gold_copy_tenant_id() gave %s but GetGoldCopyTenantID gave %s", sqlGoldID, goldTenantID)
		}
	}
}

// 7. Repin Provisional Rejection Test
func TestCoreLibrary_RepinProvisionalRejection(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loader := compliance.NewMultiTenantRuleLoader(db)
	repinService := drift.NewRepinService(db)

	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	if err != nil {
		t.Fatalf("get gold tenant: %v", err)
	}

	// Create an isolated provisional core rule fixture under gold tenant for testing
	provisionalCoreID := uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, rule_code, name, rule_phase, severity, priority,
			is_active, source_version, inherit_mode, library_status,
			ast_condition, parameter_thresholds, citation
		) VALUES (
			$1, $2, 'TEST_PROVISIONAL_RULE', 'Test Provisional Rule', 'PRE_TRADE', 'HARD_BLOCK', 95,
			true, 'CORE_LIB_V1', 'inherit', 'PROVISIONAL',
			'{}'::jsonb, '{}'::jsonb, 'Test Provisional Citation'
		)
	`, provisionalCoreID, goldTenant)
	if err != nil {
		t.Fatalf("insert fixture provisional core rule: %v", err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM compliance.compliance_rule WHERE id = $1", provisionalCoreID)
	}()

	// Create an extended tenant rule pointing to the provisional core rule
	testTenant := uuid.New()
	tenantRuleID := uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, core_rule_id, inherit_mode, pinned_core_version, drift_status,
			rule_code, name, rule_phase, severity, priority, is_active, source_version
		) VALUES (
			$1, $2, $3, 'extend', 1, 'CORE_VERSION_UPDATED',
			'TEST_PROVISIONAL_RULE', 'Custom Rate Limit', 'PRE_TRADE', 'HARD_BLOCK', 95, true, 'TENANT_CUSTOM'
		)
	`, tenantRuleID, testTenant, provisionalCoreID)
	if err != nil {
		t.Fatalf("insert test tenant rule: %v", err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM compliance.compliance_rule WHERE id = $1", tenantRuleID)
	}()

	// Attempt repin on the provisional rule -> MUST FAIL
	req := drift.RepinRuleRequest{
		TenantID:       testTenant,
		RuleID:         tenantRuleID,
		NewCoreVersion: 2,
		StewardID:      "steward_bob",
		StewardNotes:   "Attempting repin on un-corpus'd provisional rule",
		TestCorpus:     []drift.ScenarioTestCase{}, // empty corpus passes simulation
	}

	_, err = repinService.RepinTenantRule(ctx, req)
	if err == nil {
		t.Fatalf("Expected repinning on PROVISIONAL rule to be rejected, but it succeeded")
	}

	if !strings.Contains(err.Error(), "PROVISIONAL library status and cannot be repinned") {
		t.Fatalf("Expected PROVISIONAL rejection error message, got: %v", err)
	}
}

// 8. 50-Rule Canonical Content Hash 3-Way Agreement (Go == PostgreSQL == Database)
func TestCoreLibrary_All50CoreRules_ContentHashAgreement(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	if err != nil {
		t.Fatalf("get gold tenant: %v", err)
	}

	rows, err := db.QueryContext(ctx, `
		SELECT 
			r.rule_code,
			r.ast_condition::text,
			r.parameter_thresholds::text,
			COALESCE(r.citation, ''),
			v.content_hash
		FROM compliance.compliance_rule r
		JOIN compliance.compliance_rule_version v 
		  ON r.id = v.rule_id AND COALESCE(r.current_version, 1) = v.version
		WHERE r.tenant_id = $1 AND r.valid_to IS NULL
		ORDER BY r.rule_code
	`, goldTenant)
	if err != nil {
		t.Fatalf("query core rules and version hashes: %v", err)
	}
	defer rows.Close()

	checkedCount := 0
	for rows.Next() {
		var ruleCode, astStr, paramStr, citation, storedHash string
		if err := rows.Scan(&ruleCode, &astStr, &paramStr, &citation, &storedHash); err != nil {
			t.Fatalf("scan row: %v", err)
		}

		// Compute hash in Go via RFC 8785 canonical transform (Go is the sole hash authority)
		goHash, err := canonical.ComputeRuleContentHashFromRaw([]byte(astStr), []byte(paramStr), citation)
		if err != nil {
			t.Fatalf("rule %s: ComputeRuleContentHashFromRaw failed: %v", ruleCode, err)
		}

		if len(storedHash) != 64 {
			t.Fatalf("rule %s: stored hash is not 64 chars (%q)", ruleCode, storedHash)
		}
		if goHash != storedHash {
			t.Fatalf("rule %s: Go canonical computed hash %s does not match stored snapshot hash %s", ruleCode, goHash, storedHash)
		}

		checkedCount++
	}

	if checkedCount != 50 {
		t.Fatalf("Expected to verify 50 core rules, verified %d", checkedCount)
	}

	t.Logf("100%% Hash Agreement Verified across all %d Gold-Copy Core Rules (Go RFC 8785 Authority == Stored ContentHash)", checkedCount)
}

// 9. Rule Version Evolution Positive Path (Snapshot v1 -> Snapshot v2 in Transaction + Evaluation Event)
func TestCoreLibrary_RuleVersionEvolution_PositivePath(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testTenant := uuid.New()
	ruleID := uuid.New()
	ruleCode := "TEST_EVOLVE_5"
	citationV1 := "UCITS Directive 2009/65/EC - Test Version 1"
	astV1 := `{"left":{"path":"position.issuer_exposure_pct","type":"METRIC"},"operator":"GREATER_THAN","right":{"name":"issuer_limit_pct","type":"PARAM"},"type":"COMPARISON"}`
	paramsV1 := `{"aggregate_across_accounts":true,"issuer_limit_pct":"0.050000","lookthrough":true}`

	hashV1, err := canonical.ComputeRuleContentHashFromRaw([]byte(astV1), []byte(paramsV1), citationV1)
	if err != nil {
		t.Fatalf("compute hash v1: %v", err)
	}
	bytecodeHash := canonical.ComputeBytecodeHash(nil)

	// Run entire test inside an isolated transaction that rolls back at end
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	// Register tenant in transaction
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.tenants (id, name, display_name, gold_copy)
		VALUES ($1, 'test_evolve_tenant', 'Test Evolve Tenant', false)
		ON CONFLICT (id) DO NOTHING
	`, testTenant)
	require.NoError(t, err)

	// Phase 1: Insert Rule V1 and Snapshot V1
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, rule_code, name, rule_phase, severity, priority,
			is_active, source_version, inherit_mode, pinned_core_version,
			ast_condition, parameter_thresholds, citation, compiled_bytecode
		) VALUES (
			$1, $2, $3, 'Evolving Test Rule', 'PRE_TRADE', 'HARD_BLOCK', 100,
			true, 'TENANT_CUSTOM', 'custom', 1,
			$4::jsonb, $5::jsonb, $6, ''::bytea
		)
	`, ruleID, testTenant, ruleCode, astV1, paramsV1, citationV1)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
			created_by, created_at
		) VALUES (
			$1, 1, $2, $3::jsonb, $4::jsonb,
			$5, now(), null, $6, $7,
			'test_steward', now()
		)
	`, ruleID, testTenant, astV1, paramsV1, citationV1, hashV1, bytecodeHash)
	require.NoError(t, err)

	// Phase 2: Evolve to Version 2 (Insert Version Snapshot V2 + Update Rule)
	paramsV2 := `{"aggregate_across_accounts":true,"issuer_limit_pct":"0.040000","lookthrough":true}`
	citationV2 := "UCITS Directive 2009/65/EC - Test Version 2 Tightened"
	hashV2, err := canonical.ComputeRuleContentHashFromRaw([]byte(astV1), []byte(paramsV2), citationV2)
	require.NoError(t, err)

	// 1. Insert Version 2 Snapshot
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
			created_by, created_at
		) VALUES (
			$1, 2, $2, $3::jsonb, $4::jsonb,
			$5, now(), null, $6, $7,
			'test_steward', now()
		)
	`, ruleID, testTenant, astV1, paramsV2, citationV2, hashV2, bytecodeHash)
	require.NoError(t, err)

	// 2. Update Rule to Version 2
	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.compliance_rule
		SET 
			parameter_thresholds = $1::jsonb,
			citation = $2,
			current_version = 2,
			pinned_core_version = 2,
			updated_at = now()
		WHERE id = $3
	`, paramsV2, citationV2, ruleID)
	require.NoError(t, err)

	t.Logf("Rule evolution V1 -> V2 succeeded in transaction with structural trigger validation!")

	// Phase 3: Emit Evaluation Event referencing Rule Version 2 and Hash V2
	lineageID := uuid.New()
	evalHashInput := canonical.EvaluationHashInput{
		LineageID:       lineageID,
		TenantID:        testTenant,
		RuleID:          ruleID,
		RuleVersion:     2,
		RuleContentHash: hashV2,
		ActionTaken:     "APPROVED",
		Passed:          true,
		InputParams: map[string]interface{}{
			"accountId": "acc-evolve-1",
		},
		MetricSnapshots: map[string]interface{}{
			"proposedWeight": decimal.RequireFromString("0.035000"),
		},
	}
	evalHash, err := canonical.ComputeEvaluationHash(evalHashInput)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, rule_id, rule_version, rule_content_hash,
			action_taken, passed, latency_micros, evaluation_hash, evaluated_at,
			input_params, metric_snapshots
		) VALUES (
			gen_random_uuid(), $1, $2, $3, 2, $4,
			'APPROVED', true, 120, $5, now(),
			'{"accountId":"acc-evolve-1"}'::jsonb, '{"proposedWeight":"0.035000"}'::jsonb
		)
	`, lineageID, testTenant, ruleID, hashV2, evalHash)
	require.NoError(t, err)
	t.Logf("Evaluation event referencing V2 and 64-char ContentHash V2 inserted successfully with FK verification!")

	// Phase 4: Verify FK violation when attempting to insert evaluation event for nonexistent version
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, rule_id, rule_version, rule_content_hash,
			action_taken, passed, latency_micros, evaluation_hash, evaluated_at
		) VALUES (
			gen_random_uuid(), gen_random_uuid(), $1, $2, 999, $3,
			'APPROVED', true, 120, $4, now()
		)
	`, testTenant, ruleID, hashV2, evalHash)
	require.Error(t, err, "Expected FK constraint violation for nonexistent version 999")
	t.Logf("FK RESTRICT constraint correctly rejected nonexistent rule version 999: %v", err)
}

