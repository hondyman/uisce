package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestWarmTierLoader_EndToEndBatchSyncAndIdempotency(t *testing.T) {
	// 1. Check connectivity to StarRocks & PG
	srHost := os.Getenv("STARROCKS_HTTP_HOST")
	if srHost == "" {
		srHost = "100.84.50.65"
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:8030", srHost), 2*time.Second)
	if err != nil {
		t.Skipf("StarRocks at %s:8030 not reachable, skipping live warm tier sync test", srHost)
		return
	}
	conn.Close()

	homeDir, _ := os.UserHomeDir()
	dsn := fmt.Sprintf("postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=verify-full&sslrootcert=%s/.uisce/certs/ca.crt&sslcert=%s/.uisce/certs/postgres-client.crt&sslkey=%s/.uisce/certs/postgres-client.key", homeDir, homeDir, homeDir)
	pgDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Postgres alpha not reachable: %v", err)
		return
	}
	defer pgDB.Close()

	if err := pgDB.Ping(); err != nil {
		t.Skipf("Postgres alpha ping failed: %v", err)
		return
	}

	srMySQLDSN := fmt.Sprintf("root:@tcp(%s:9030)/oms", srHost)
	srDB, err := sql.Open("mysql", srMySQLDSN)
	if err != nil {
		t.Skipf("StarRocks MySQL connection failed: %v", err)
		return
	}
	defer srDB.Close()

	if err := srDB.Ping(); err != nil {
		t.Skipf("StarRocks MySQL ping failed: %v", err)
		return
	}

	tenantID := uuid.New()
	ruleID := uuid.New()

	// Insert test compliance rule for this isolated tenant
	ruleQ := `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, pinned_core_version, drift_status,
			rule_code, name, rule_phase, severity, ast_condition,
			parameter_thresholds, priority, is_active, created_at, updated_at
		) VALUES (
			$1, $2, 'custom', 1, 'CURRENT',
			'WARM_TIER_TEST_RULE', 'Warm Tier Test Rule', 'PRE_TRADE', 'HARD_BLOCK', '{}'::jsonb,
			'{}'::jsonb, 10, true, NOW(), NOW()
		)
	`
	_, err = pgDB.Exec(ruleQ, ruleID, tenantID)
	require.NoError(t, err)

	defer func() {
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_evaluation_event WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_watermark_checkpoint WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_rule WHERE id = $1", ruleID)
		_, _ = srDB.Exec("DELETE FROM oms.compliance_evaluations WHERE tenant_id = ?", tenantID.String())
	}()

	// 2. Insert 5 evaluation events into Hot Postgres
	lineageIDs := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		lineageIDs[i] = uuid.New()
		orderID := uuid.New()
		insertQ := `
			INSERT INTO compliance.compliance_evaluation_event (
				id, lineage_id, tenant_id, order_id, rule_id, rule_version,
				passed, action_taken, latency_micros, evaluation_hash,
				input_params, metric_snapshots, evaluated_at, created_at, ingest_lsn
			) VALUES (
				$1, $2, $3, $4, $5, 1,
				true, 'APPROVED', 150, $6,
				'{"test": true}'::jsonb, '{"metric": 100}'::jsonb,
				NOW(), NOW(), (pg_current_wal_lsn() - '0/0'::pg_lsn)::bigint
			)
		`
		evalHash := fmt.Sprintf("hash_%s", lineageIDs[i].String())
		_, err := pgDB.Exec(insertQ, uuid.New(), lineageIDs[i], tenantID, orderID, ruleID, evalHash)
		require.NoError(t, err)
	}

	// 3. Instantiate WarmTierLoader and sync batch
	loader := NewWarmTierLoader(WarmTierLoaderConfig{
		PostgresDB:        pgDB,
		StarRocksHTTPHost: srHost,
		StarRocksHTTPPort: 8030,
		StarRocksDatabase: "oms",
		StarRocksTable:    "compliance_evaluations",
		BatchSize:         100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := loader.SyncTenantBatch(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, int64(5), result.RowsLoaded, "Expected exactly 5 rows loaded to StarRocks")
	require.Greater(t, result.HighestLSN, int64(0))

	t.Logf("Warm Tier Stream Load succeeded: TxnID=%d, Loaded=%d, HighestLSN=%d in %dms",
		result.TxnID, result.RowsLoaded, result.HighestLSN, result.DurationMs)

	// 4. Verify Watermark Checkpoint table in PostgreSQL
	watermarkLSN, err := loader.GetWatermark(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, result.HighestLSN, watermarkLSN, "Watermark checkpoint must match HighestLSN")

	// 5. Query StarRocks MySQL and verify 5 rows exist for this tenant
	var srCount int
	err = srDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM oms.compliance_evaluations WHERE tenant_id = ?", tenantID.String()).Scan(&srCount)
	require.NoError(t, err)
	require.Equal(t, 5, srCount, "StarRocks must have exactly 5 rows for this tenant")

	// 6. Test Idempotency: Re-syncing the same batch or streaming existing records again
	// Reset watermark to 0 to simulate replay
	err = loader.UpdateWatermark(ctx, tenantID, 0)
	require.NoError(t, err)

	replayResult, err := loader.SyncTenantBatch(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, int64(5), replayResult.RowsLoaded)

	// In StarRocks, Primary Key table dedup ensures count remains 5
	err = srDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM oms.compliance_evaluations WHERE tenant_id = ?", tenantID.String()).Scan(&srCount)
	require.NoError(t, err)
	require.Equal(t, 5, srCount, "StarRocks Primary Key deduplication ensures row count remains exactly 5 after replay")

	var distinctCount int
	err = srDB.QueryRowContext(ctx, "SELECT COUNT(DISTINCT lineage_id) FROM oms.compliance_evaluations WHERE tenant_id = ?", tenantID.String()).Scan(&distinctCount)
	require.NoError(t, err)
	require.Equal(t, 5, distinctCount, "DISTINCT lineage_id must match COUNT(*)")

	t.Logf("Warm Tier batch sync and idempotent deduplication verified end-to-end!")
}
