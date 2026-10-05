package blotter

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func getAlphaTestDB(t *testing.T) *sql.DB {
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

	// Insert test evaluation events
	lineageID1 := uuid.New()
	orderID1 := uuid.New()
	evalHash1 := fmt.Sprintf("eval_hash_%s", uuid.New().String()[:8])

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

	lineageID2 := uuid.New()
	orderID2 := uuid.New()
	evalHash2 := fmt.Sprintf("eval_hash_%s", uuid.New().String()[:8])

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
