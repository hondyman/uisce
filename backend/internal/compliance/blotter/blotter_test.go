package blotter

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/hondyman/uisce/backend/internal/compliance/testutil"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func getAlphaTestDB(tb testing.TB) *sql.DB {
	if tb != nil {
		tb.Helper()
	}
	return testutil.GetEphemeralTestDB(tb)
}

func TestBlotterService_ListAndEvidenceBundle(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	require.NoError(t, err)

	hub := NewWebSocketHub()
	go hub.Run()
	svc := NewService(db, hub)

	// Fetch UCITS_ISSUER_5 rule info
	var ruleID uuid.UUID
	var ruleCode, citation, contentHash string
	err = db.QueryRowContext(ctx, `
		SELECT r.id, r.rule_code, v.citation, v.content_hash
		FROM compliance.compliance_rule r
		JOIN compliance.compliance_rule_version v ON r.id = v.rule_id AND r.current_version = v.version
		WHERE r.tenant_id = $1 AND r.rule_code = 'UCITS_ISSUER_5'
		LIMIT 1
	`, goldTenant).Scan(&ruleID, &ruleCode, &citation, &contentHash)
	require.NoError(t, err)

	lineageID1 := uuid.New()
	orderID1 := uuid.New()
	evalHash1 := fmt.Sprintf("eval_hash_%s", uuid.New().String()[:8])

	lineageID2 := uuid.New()
	orderID2 := uuid.New()
	evalHash2 := fmt.Sprintf("eval_hash_%s", uuid.New().String()[:8])

	// Clean up inserted test evaluation events upon test completion
	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_evaluation_event WHERE lineage_id IN ($1, $2)", lineageID1, lineageID2)
	}()

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, order_id, rule_id, rule_version,
			passed, action_taken, latency_micros, rule_content_hash, evaluation_hash,
			input_params, metric_snapshots, evaluated_at, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, 1,
			false, 'BLOCKED', 420, $5, $6,
			'{"order_amount": 1000000}'::jsonb,
			'{"pos.issuer_pct": "0.054200", "issuer_limit_pct": "0.050000"}'::jsonb,
			now(), now()
		)
	`, lineageID1, goldTenant, orderID1, ruleID, contentHash, evalHash1)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, order_id, rule_id, rule_version,
			passed, action_taken, latency_micros, rule_content_hash, evaluation_hash,
			input_params, metric_snapshots, evaluated_at, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, 1,
			true, 'APPROVED', 310, $5, $6,
			'{"order_amount": 500000}'::jsonb,
			'{"pos.issuer_pct": "0.038000", "issuer_limit_pct": "0.050000"}'::jsonb,
			now(), now()
		)
	`, lineageID2, goldTenant, orderID2, ruleID, contentHash, evalHash2)
	require.NoError(t, err)

	// 1. Test ListEvaluations with filter
	res, err := svc.ListEvaluations(ctx, ListFilter{
		TenantID:    goldTenant,
		RuleCode:    "UCITS_ISSUER_5",
		ActionTaken: "BLOCKED",
		Page:        1,
		PageSize:    10,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, res.TotalCount, int64(1))
	require.NotEmpty(t, res.Data)
	require.Equal(t, "BLOCKED", res.Data[0].ActionTaken)
	require.Equal(t, "UCITS_ISSUER_5", res.Data[0].RuleCode)

	// 2. Test GetEvidenceBundleByLineageID
	bundle, err := svc.GetEvidenceBundleByLineageID(ctx, goldTenant, lineageID1)
	require.NoError(t, err)
	require.NotNil(t, bundle)
	require.Equal(t, lineageID1, bundle.Evaluation.LineageID)
	require.Equal(t, "BLOCKED", bundle.Evaluation.ActionTaken)
	require.Equal(t, "UCITS_ISSUER_5", bundle.Evaluation.RuleCode)
	require.True(t, bundle.IntegrityProof.ContentHashMatches, "Content hash must match Go RFC 8785 canonical recomputation")
	require.NotEmpty(t, bundle.NaturalLanguageExplanation)
	require.Contains(t, bundle.NaturalLanguageExplanation, "BLOCKED")
	require.Contains(t, bundle.NaturalLanguageExplanation, "UCITS")
	t.Logf("Generated Natural-Language Explanation:\n%s", bundle.NaturalLanguageExplanation)
	t.Logf("Decision Integrity Proof verified: %v", bundle.IntegrityProof)

	// Assert metric comparison structure
	require.NotEmpty(t, bundle.Metrics)
	foundIssuerMetric := false
	for _, m := range bundle.Metrics {
		if m.MetricPath == "pos.issuer_pct" {
			foundIssuerMetric = true
			require.True(t, m.Breached)
			require.Contains(t, m.Margin, "exceeded limit")
		}
	}
	require.True(t, foundIssuerMetric, "pos.issuer_pct metric must be parsed and flagged as breached")
}

// TestBlotterService_JoinIntegrity_VersionAnchoringAndTamperRejection verifies:
// 1. Version Anchoring: When rule is at v2, evaluation for v1 resolves exact v1 snapshot logic and limits (never v2).
// 2. Fail-Closed Tamper Rejection: Spoofed / mismatched hash on evaluation event or snapshot fails loudly with ErrProvenanceVerificationFailed (never degrades silently).
func TestBlotterService_JoinIntegrity_VersionAnchoringAndTamperRejection(t *testing.T) {
	db := getAlphaTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	hub := NewWebSocketHub()
	go hub.Run()
	svc := NewService(db, hub)
	testTenant := uuid.New()
	testRuleID := uuid.New()
	testRuleCode := fmt.Sprintf("JOIN_TEST_%s", uuid.New().String()[:8])

	_, err := db.ExecContext(ctx, `
		INSERT INTO public.tenants (id, name, display_name, gold_copy)
		VALUES ($1, 'join_test_tenant', 'Join Test Tenant', false)
		ON CONFLICT (id) DO NOTHING
	`, testTenant)
	require.NoError(t, err)

	// Clean up all inserted test artifacts
	var lineageV1, lineageV2, lineageTampered uuid.UUID
	defer func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule_version WHERE tenant_id = $1", testTenant)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM compliance.compliance_rule WHERE tenant_id = $1", testTenant)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM public.tenants WHERE id = $1", testTenant)
	}()

	// 1. Seed Rule V1 (Limit: 5%)
	astV1 := `{"type":"METRIC","path":"pos.issuer_pct"}`
	paramsV1 := `{"issuer_limit_pct":"0.050000"}`
	citationV1 := "UCITS Directive 2009/65/EC Art. 52 (v1 Standard)"
	hashV1, err := canonical.ComputeRuleContentHashFromRaw([]byte(astV1), []byte(paramsV1), citationV1)
	require.NoError(t, err)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, rule_code, name, rule_phase, severity,
			priority, is_active, current_version, ast_condition, parameter_thresholds,
			citation, compiled_bytecode, library_status
		) VALUES (
			$1, $2, 'inherit', $3, 'Join Integrity Rule', 'PRE_TRADE', 'HARD_BLOCK',
			100, true, 1, $4::jsonb, $5::jsonb, $6, '\x00'::bytea, 'ACTIVE'
		)
	`, testRuleID, testTenant, testRuleCode, astV1, paramsV1, citationV1)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
			created_by, created_at
		) VALUES (
			$1, 1, $2, $3::jsonb, $4::jsonb,
			$5, now() - interval '10 days', null, $6, $7, 'seed', now() - interval '10 days'
		)
	`, testRuleID, testTenant, astV1, paramsV1, citationV1, hashV1, canonical.ComputeBytecodeHash(nil))
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	// 2. Advance Rule to V2 (Limit: 3% tightened)
	astV2 := astV1
	paramsV2 := `{"issuer_limit_pct":"0.030000"}`
	citationV2 := "UCITS Directive 2009/65/EC Art. 52 (v2 Amended Concentration 2026)"
	hashV2, err := canonical.ComputeRuleContentHashFromRaw([]byte(astV2), []byte(paramsV2), citationV2)
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
			$5, now(), null, $6, $7, 'seed', now()
		)
	`, testRuleID, testTenant, astV2, paramsV2, citationV2, hashV2, canonical.ComputeBytecodeHash(nil))
	require.NoError(t, err)

	_, err = tx2.ExecContext(ctx, `
		UPDATE compliance.compliance_rule
		SET current_version = 2,
		    parameter_thresholds = $2::jsonb,
		    citation = $3
		WHERE id = $1
	`, testRuleID, paramsV2, citationV2)
	require.NoError(t, err)
	require.NoError(t, tx2.Commit())

	// 3. Create Evaluation Event 1 (Evaluated historically against V1 when V1 was live)
	lineageV1 = uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, order_id, rule_id, rule_version,
			passed, action_taken, latency_micros, rule_content_hash, evaluation_hash,
			input_params, metric_snapshots, evaluated_at, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, gen_random_uuid(), $3, 1,
			true, 'APPROVED', 280, $4, 'eval_hash_v1',
			'{}'::jsonb,
			'{"pos.issuer_pct": "0.042000", "issuer_limit_pct": "0.050000"}'::jsonb,
			now() - interval '5 days', now() - interval '5 days'
		)
	`, lineageV1, testTenant, testRuleID, hashV1)
	require.NoError(t, err)

	// 4. Assert Evidence Bundle for V1 Event: MUST return exact V1 snapshot, NOT V2!
	bundleV1, err := svc.GetEvidenceBundleByLineageID(ctx, testTenant, lineageV1)
	require.NoError(t, err)
	require.NotNil(t, bundleV1)
	require.Equal(t, 1, bundleV1.RuleSnapshot.Version, "Evidence bundle MUST anchor strictly to historical version 1")
	require.Equal(t, hashV1, bundleV1.RuleSnapshot.ContentHash, "Must resolve V1 content hash")
	require.Equal(t, citationV1, bundleV1.RuleSnapshot.Citation, "Must resolve V1 citation")
	require.True(t, bundleV1.IntegrityProof.ContentHashMatches, "V1 hash must recompute and match 100%")
	require.Contains(t, bundleV1.NaturalLanguageExplanation, "v1")
	t.Logf("Scenario A: Version Anchoring Verified! Event under v1 resolved v1 citation: %s", bundleV1.RuleSnapshot.Citation)

	// 5. Create Evaluation Event 2 under V2
	lineageV2 = uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, order_id, rule_id, rule_version,
			passed, action_taken, latency_micros, rule_content_hash, evaluation_hash,
			input_params, metric_snapshots, evaluated_at, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, gen_random_uuid(), $3, 2,
			false, 'BLOCKED', 350, $4, 'eval_hash_v2',
			'{}'::jsonb,
			'{"pos.issuer_pct": "0.038000", "issuer_limit_pct": "0.030000"}'::jsonb,
			now(), now()
		)
	`, lineageV2, testTenant, testRuleID, hashV2)
	require.NoError(t, err)

	bundleV2, err := svc.GetEvidenceBundleByLineageID(ctx, testTenant, lineageV2)
	require.NoError(t, err)
	require.Equal(t, 2, bundleV2.RuleSnapshot.Version)
	require.Equal(t, hashV2, bundleV2.RuleSnapshot.ContentHash)
	require.Equal(t, citationV2, bundleV2.RuleSnapshot.Citation)

	// 6. Scenario B: Injected Tampered / Mismatched Event Hash (Spoofed Hash)
	lineageTampered = uuid.New()
	spoofedHash := "0000000000000000000000000000000000000000000000000000000000000000"
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, order_id, rule_id, rule_version,
			passed, action_taken, latency_micros, rule_content_hash, evaluation_hash,
			input_params, metric_snapshots, evaluated_at, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, gen_random_uuid(), $3, 1,
			true, 'APPROVED', 280, $4, 'eval_hash_tampered',
			'{}'::jsonb,
			'{"pos.issuer_pct": "0.040000"}'::jsonb,
			now(), now()
		)
	`, lineageTampered, testTenant, testRuleID, spoofedHash)
	require.NoError(t, err)

	// MUST fail loudly with ErrProvenanceVerificationFailed! Never degrade silently!
	bundleTampered, err := svc.GetEvidenceBundleByLineageID(ctx, testTenant, lineageTampered)
	require.Error(t, err, "Spoofed content hash on evaluation event MUST be rejected loudly")
	require.ErrorIs(t, err, ErrProvenanceVerificationFailed, "Error must wrap ErrProvenanceVerificationFailed")
	require.Nil(t, bundleTampered)
	t.Logf("Scenario B: Fail-Closed Tamper Rejection Verified: %v", err)
}

func TestBlotterWebSocketHub_Broadcast(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	tenantA := uuid.New()
	tenantB := uuid.New()

	clientA := &Client{
		hub:      hub,
		send:     make(chan []byte, 10),
		tenantID: tenantA,
	}
	clientB := &Client{
		hub:      hub,
		send:     make(chan []byte, 10),
		tenantID: tenantB,
	}

	hub.register <- clientA
	hub.register <- clientB
	time.Sleep(20 * time.Millisecond)

	// Broadcast to Tenant A
	eventA := EvaluationEventRecord{
		ID:          uuid.New(),
		LineageID:   uuid.New(),
		TenantID:    tenantA,
		RuleCode:    "UCITS_ISSUER_5",
		ActionTaken: "BLOCKED",
	}
	hub.BroadcastEvaluation(tenantA, eventA)

	// Client A should receive message
	select {
	case msg := <-clientA.send:
		require.Contains(t, string(msg), "UCITS_ISSUER_5")
		require.Contains(t, string(msg), "BLOCKED")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client A timed out waiting for broadcast")
	}

	// Client B should NOT receive message (Tenant isolation)
	select {
	case msg := <-clientB.send:
		t.Fatalf("Tenant isolation breach: Client B received message intended for Tenant A: %s", string(msg))
	case <-time.After(50 * time.Millisecond):
		// Success: no message received by Client B
	}
}
