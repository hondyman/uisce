package cold_test

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

	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/hondyman/uisce/backend/internal/compliance/cold"
	"github.com/hondyman/uisce/backend/internal/compliance/jobs"
)

func TestCompliance_EndToEndCryptographicAuditHotWarmColdCLI(t *testing.T) {
	s3Endpoint := os.Getenv("AWS_S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "100.84.50.65:9000"
	}
	conn, err := net.DialTimeout("tcp", s3Endpoint, 2*time.Second)
	if err != nil {
		t.Skipf("MinIO at %s not reachable, skipping E2E audit test", s3Endpoint)
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

	tenantID := uuid.New()
	ruleID := uuid.New()

	// 1. Insert rule for tenant and rule version snapshot
	ruleQ := `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, pinned_core_version, drift_status,
			rule_code, name, rule_phase, severity, ast_condition,
			parameter_thresholds, priority, is_active, created_at, updated_at
		) VALUES (
			$1, $2, 'custom', 1, 'CURRENT',
			'E2E_AUDIT_RULE', 'E2E Audit Rule', 'PRE_TRADE', 'HARD_BLOCK', '{}'::jsonb,
			'{}'::jsonb, 10, true, NOW(), NOW()
		)
	`
	_, err = pgDB.Exec(ruleQ, ruleID, tenantID)
	require.NoError(t, err)

	expectedContentHash, err := canonical.ComputeRuleContentHashFromRaw([]byte("{}"), []byte("{}"), "E2E Test Citation")
	require.NoError(t, err)
	expectedBytecodeHash := canonical.ComputeBytecodeHash(nil)

	ruleVerQ := `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, content_hash, compiled_bytecode_hash, created_by
		) VALUES (
			$1, 1, $2, '{}'::jsonb, '{}'::jsonb,
			'E2E Test Citation', NOW(), $3, $4, 'test'
		) ON CONFLICT (rule_id, version) DO NOTHING
	`
	_, err = pgDB.Exec(ruleVerQ, ruleID, tenantID, expectedContentHash, expectedBytecodeHash)
	require.NoError(t, err)

	defer func() {
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_evaluation_event WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_watermark_checkpoint WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_archive_manifest WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_rule_version WHERE rule_id = $1", ruleID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_rule WHERE id = $1", ruleID)
	}()

	// 2. Insert 10 synthetic trade evaluations into Hot PG
	for i := 1; i <= 10; i++ {
		insertQ := `
			INSERT INTO compliance.compliance_evaluation_event (
				id, lineage_id, tenant_id, order_id, rule_id, rule_version, rule_content_hash,
				passed, action_taken, latency_micros, evaluation_hash,
				input_params, metric_snapshots, evaluated_at, created_at, ingest_lsn
			) VALUES (
				$1, $2, $3, $4, $5, 1, $6,
				true, 'APPROVED', 140, $7,
				'{"price":"185.500000","qty":"500.000000"}'::jsonb, '{"exposure":"92750.000000"}'::jsonb,
				NOW(), NOW(), (pg_current_wal_lsn() - '0/0'::pg_lsn)::bigint
			)
		`
		lineageID := uuid.New()
		orderID := uuid.New()
		evalHash := fmt.Sprintf("eval_hash_%s", lineageID.String())
		_, err := pgDB.Exec(insertQ, uuid.New(), lineageID, tenantID, orderID, ruleID, expectedContentHash, evalHash)
		require.NoError(t, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// 3. Hot -> Warm: Stream load to StarRocks Warm Tier via WarmTierLoader
	warmLoader := jobs.NewWarmTierLoader(jobs.WarmTierLoaderConfig{
		PostgresDB:        pgDB,
		StarRocksHTTPHost: "100.84.50.65",
		StarRocksHTTPPort: 8030,
		StarRocksDatabase: "oms",
		StarRocksTable:    "compliance_evaluations",
		BatchSize:         100,
	})

	warmResult, err := warmLoader.SyncTenantBatch(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, int64(10), warmResult.RowsLoaded)
	t.Logf("Hot -> Warm Sync PASSED: %d rows stream loaded to StarRocks, Certified LWM = %d",
		warmResult.RowsLoaded, warmResult.HighestLSN)

	// 4. Warm -> Cold: Seal immutable WORM Parquet slice to MinIO S3 via ColdArchivalWorker
	s3Client, err := cold.NewS3StorageClient(cold.S3Config{
		Endpoint:        s3Endpoint,
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UseSSL:          false,
		BucketName:      "compliance-cold-archive",
		RetentionYears:  15,
	})
	require.NoError(t, err)

	manifestRepo := cold.NewManifestRepository(pgDB)
	coldWorker := jobs.NewColdArchivalWorker(jobs.ColdArchivalWorkerConfig{
		PostgresDB:   pgDB,
		S3Client:     s3Client,
		ManifestRepo: manifestRepo,
		BucketName:   "compliance-cold-archive",
		BatchSize:    100,
	})

	coldResult, err := coldWorker.ArchiveTenantSlice(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, int64(10), coldResult.RowsArchived)
	require.NotEmpty(t, coldResult.MerkleRootHash)
	require.NotEmpty(t, coldResult.RuleRegistryMerkleRoot)
	require.NotEmpty(t, coldResult.RuleRegistryS3Key)
	require.NotEmpty(t, coldResult.ETag)
	t.Logf("Warm -> Cold WORM Archive PASSED: S3Key=%s, MerkleRoot=%s, RuleRegistryKey=%s, RuleRegistryMerkleRoot=%s, ETag=%s",
		coldResult.S3Key, coldResult.MerkleRootHash, coldResult.RuleRegistryS3Key, coldResult.RuleRegistryMerkleRoot, coldResult.ETag)

	// 5. Cold -> Verification CLI: Independently download Parquet and verify Merkle root
	downloadedParquet, err := s3Client.DownloadObject(ctx, coldResult.S3Bucket, coldResult.S3Key)
	require.NoError(t, err)

	verifier := cold.NewArchiveVerifier()
	report, err := verifier.VerifyParquetSlice(ctx, downloadedParquet, coldResult.MerkleRootHash)
	require.NoError(t, err)

	require.True(t, report.RootMatch, "Calculated Merkle root must match signed manifest root 100%%")
	require.True(t, report.InclusionProofs, "All leaf inclusion proofs must verify")
	require.Equal(t, "VERIFIED_TAMPER_EVIDENT_MATCH", report.AuditStatus)
	require.Equal(t, int64(10), report.RecordCount)

	// Verify Companion Rule Registry slice
	downloadedRuleParquet, err := s3Client.DownloadObject(ctx, coldResult.S3Bucket, coldResult.RuleRegistryS3Key)
	require.NoError(t, err)

	ruleReport, err := verifier.VerifyRuleRegistrySlice(ctx, downloadedRuleParquet, coldResult.RuleRegistryMerkleRoot)
	require.NoError(t, err)
	require.True(t, ruleReport.RootMatch, "Rule registry Merkle root must match signed manifest root 100%%")
	require.True(t, ruleReport.InclusionProofs, "Rule registry leaf inclusion proofs must verify")
	require.True(t, ruleReport.ContentHashesVerified, "Rule registry content hashes MUST derive 100%% from underlying Parquet AST bytes")
	require.Equal(t, "VERIFIED_TAMPER_EVIDENT_MATCH", ruleReport.AuditStatus)
	require.Equal(t, int64(1), ruleReport.RecordCount)

	t.Logf("END-TO-END CRYPTOGRAPHIC AUDIT PASSED: 100%% Merkle Match (Evaluations + Rule Registry) & 100%% Content Hash Derivation, Status = %s", report.AuditStatus)
}
