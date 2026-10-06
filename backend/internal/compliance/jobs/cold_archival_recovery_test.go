package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/hondyman/uisce/backend/internal/compliance/cold"
)

func TestColdArchivalWorker_MultiBatchContiguityAndQuarantineRecovery(t *testing.T) {
	// 1. Check live connectivity
	s3Endpoint := os.Getenv("AWS_S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "100.84.50.65:9000"
	}
	conn, err := net.DialTimeout("tcp", s3Endpoint, 2*time.Second)
	if err != nil {
		t.Skipf("MinIO at %s not reachable, skipping live archival worker test", s3Endpoint)
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

	tenantID := uuid.New()
	ruleID := uuid.New()

	// Insert test compliance rule and version snapshot
	ruleQ := `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, pinned_core_version, drift_status,
			rule_code, name, rule_phase, severity, ast_condition,
			parameter_thresholds, priority, is_active, created_at, updated_at
		) VALUES (
			$1, $2, 'custom', 1, 'CURRENT',
			'COLD_ARCHIVE_TEST_RULE', 'Cold Archive Test Rule', 'PRE_TRADE', 'HARD_BLOCK', '{}'::jsonb,
			'{}'::jsonb, 10, true, NOW(), NOW()
		)
	`
	_, err = pgDB.Exec(ruleQ, ruleID, tenantID)
	require.NoError(t, err)

	contentHash1, err := canonical.ComputeRuleContentHashFromRaw([]byte("{}"), []byte("{}"), "Test Citation")
	require.NoError(t, err)
	bytecodeHash1 := canonical.ComputeBytecodeHash(nil)

	ruleVerQ := `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, content_hash, compiled_bytecode_hash, created_by
		) VALUES (
			$1, 1, $2, '{}'::jsonb, '{}'::jsonb,
			'Test Citation', NOW(), $3, $4, 'test'
		) ON CONFLICT (rule_id, version) DO NOTHING
	`
	_, err = pgDB.Exec(ruleVerQ, ruleID, tenantID, contentHash1, bytecodeHash1)
	require.NoError(t, err)

	defer func() {
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_evaluation_event WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_watermark_checkpoint WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_archive_manifest WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_rule_version WHERE rule_id = $1", ruleID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_rule WHERE id = $1", ruleID)
	}()

	worker := NewColdArchivalWorker(ColdArchivalWorkerConfig{
		PostgresDB:   pgDB,
		S3Client:     s3Client,
		ManifestRepo: manifestRepo,
		BucketName:   "compliance-cold-archive",
		BatchSize:    100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// 2. Batch 1: Insert 5 events with LSN 101..105
	baseLSN := int64(46000000000)
	for i := 1; i <= 5; i++ {
		insertQ := `
			INSERT INTO compliance.compliance_evaluation_event (
				id, lineage_id, tenant_id, order_id, rule_id, rule_version, rule_content_hash,
				passed, action_taken, latency_micros, evaluation_hash,
				input_params, metric_snapshots, evaluated_at, created_at, ingest_lsn
			) VALUES (
				$1, $2, $3, $4, $5, 1, $6,
				true, 'APPROVED', 150, $7,
				'{"batch": 1}'::jsonb, '{"metric": 100}'::jsonb,
				NOW(), NOW(), $8
			)
		`
		lineageID := uuid.New()
		orderID := uuid.New()
		evalHash := fmt.Sprintf("hash_%s", lineageID.String())
		lsn := baseLSN + int64(i)
		_, err := pgDB.Exec(insertQ, uuid.New(), lineageID, tenantID, orderID, ruleID, contentHash1, evalHash, lsn)
		require.NoError(t, err)
	}

	// Set Certified Warm Watermark for Batch 1 to baseLSN + 5
	warmLWM1 := baseLSN + 5
	_, err = pgDB.Exec(`
		INSERT INTO compliance.compliance_watermark_checkpoint (
			id, tenant_id, tier, current_lsn, certified_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, 'WARM', '0/0'::pg_lsn + $2, NOW(), NOW()
		) ON CONFLICT (tenant_id, tier) DO UPDATE SET current_lsn = EXCLUDED.current_lsn
	`, tenantID, warmLWM1)
	require.NoError(t, err)

	// Execute Batch 1 Archival
	res1, err := worker.ArchiveTenantSlice(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, int64(5), res1.RowsArchived)
	require.Equal(t, baseLSN+1, res1.StartLSN)
	require.Equal(t, baseLSN+5, res1.EndLSN)
	require.NotEmpty(t, res1.MerkleRootHash)
	t.Logf("Batch 1 Archived: %d rows, LSN [%d, %d], MerkleRoot=%s",
		res1.RowsArchived, res1.StartLSN, res1.EndLSN, res1.MerkleRootHash)

	// 3. Batch 2: Insert next 5 events with contiguous LSN 106..110
	for i := 6; i <= 10; i++ {
		insertQ := `
			INSERT INTO compliance.compliance_evaluation_event (
				id, lineage_id, tenant_id, order_id, rule_id, rule_version, rule_content_hash,
				passed, action_taken, latency_micros, evaluation_hash,
				input_params, metric_snapshots, evaluated_at, created_at, ingest_lsn
			) VALUES (
				$1, $2, $3, $4, $5, 1, $6,
				true, 'APPROVED', 150, $7,
				'{"batch": 2}'::jsonb, '{"metric": 200}'::jsonb,
				NOW(), NOW(), $8
			)
		`
		lineageID := uuid.New()
		orderID := uuid.New()
		evalHash := fmt.Sprintf("hash_%s", lineageID.String())
		lsn := baseLSN + int64(i)
		_, err := pgDB.Exec(insertQ, uuid.New(), lineageID, tenantID, orderID, ruleID, contentHash1, evalHash, lsn)
		require.NoError(t, err)
	}

	// Set Certified Warm Watermark for Batch 2 to baseLSN + 10
	warmLWM2 := baseLSN + 10
	_, err = pgDB.Exec(`
		UPDATE compliance.compliance_watermark_checkpoint 
		SET current_lsn = '0/0'::pg_lsn + $2, updated_at = NOW()
		WHERE tenant_id = $1 AND tier = 'WARM'
	`, tenantID, warmLWM2)
	require.NoError(t, err)

	// Execute Batch 2 Archival
	res2, err := worker.ArchiveTenantSlice(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, int64(5), res2.RowsArchived)
	require.Equal(t, baseLSN+6, res2.StartLSN, "Batch 2 must start exactly at previous Batch 1 EndLSN + 1 (Contiguity)")
	require.Equal(t, baseLSN+10, res2.EndLSN)
	t.Logf("Batch 2 Contiguous Archival PASSED: LSN [%d, %d]", res2.StartLSN, res2.EndLSN)

	// 4. Test Audit Gap Detection: Artificial non-contiguous batch
	lastManifest, err := manifestRepo.GetLastManifest(ctx, tenantID)
	require.NoError(t, err)
	gapErr := cold.VerifyContiguity(lastManifest, baseLSN+15, baseLSN+20)
	require.Error(t, gapErr)
	require.Contains(t, gapErr.Error(), "manifest audit gap detected")
	t.Logf("Manifest Audit Gap Detection correctly rejected non-contiguous range: %v", gapErr)

	// 5. Test Quarantine Sweeper against orphan unmanifested object
	orphanKey := fmt.Sprintf("cold-archive/tenant_%s/year=2026/month=10/orphan_unmanifested_%d.parquet",
		tenantID.String(), time.Now().UnixNano())
	orphanData := []byte("ORPHAN_UNMANIFESTED_CRASH_RECOVERY_PAYLOAD")

	_, err = s3Client.UploadWORMObject(ctx, "compliance-cold-archive", orphanKey, orphanData)
	require.NoError(t, err)

	defer func() {
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_orphan_object WHERE s3_key = $1", orphanKey)
	}()

	sweeper := NewQuarantineSweeper(s3Client, manifestRepo, "compliance-cold-archive")
	sweepReport, err := sweeper.RunSweep(ctx, fmt.Sprintf("cold-archive/tenant_%s", tenantID.String()))
	require.NoError(t, err)
	require.GreaterOrEqual(t, sweepReport.Manifested, 2, "Must identify at least 2 manifested batches")
	require.GreaterOrEqual(t, sweepReport.Quarantined, 1, "Must quarantine unmanifested orphan object")

	t.Logf("Quarantine Sweeper PASSED: Total=%d, Manifested=%d, Quarantined=%d",
		sweepReport.TotalScanned, sweepReport.Manifested, sweepReport.Quarantined)

	// Verify database entry in compliance_orphan_object
	var orphanStatus string
	err = pgDB.QueryRowContext(ctx, "SELECT status FROM compliance.compliance_orphan_object WHERE s3_key = $1", orphanKey).Scan(&orphanStatus)
	require.NoError(t, err)
	require.Equal(t, "QUARANTINED", orphanStatus)

	t.Logf("Cold Archival Worker, Multi-Batch Contiguity, and Quarantine Sweeper verified end-to-end!")
}

func TestColdArchivalWorker_CrashBeforeManifestCommit_RecoveryResumeAndCLIVerification(t *testing.T) {
	s3Endpoint := os.Getenv("AWS_S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "100.84.50.65:9000"
	}
	conn, err := net.DialTimeout("tcp", s3Endpoint, 2*time.Second)
	if err != nil {
		t.Skipf("MinIO at %s not reachable, skipping crash recovery test", s3Endpoint)
		return
	}
	conn.Close()

	homeDir, _ := os.UserHomeDir()
	dsn := fmt.Sprintf("postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=verify-full&sslrootcert=%s/.uisce/certs/ca.crt&sslcert=%s/.uisce/certs/postgres-client.crt&sslkey=%s/.uisce/certs/postgres-client.key", homeDir, homeDir, homeDir)
	pgDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer pgDB.Close()

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

	tenantID := uuid.New()
	ruleID := uuid.New()

	// Insert test compliance rule and version snapshot
	ruleQ := `
		INSERT INTO compliance.compliance_rule (
			id, tenant_id, inherit_mode, pinned_core_version, drift_status,
			rule_code, name, rule_phase, severity, ast_condition,
			parameter_thresholds, priority, is_active, created_at, updated_at
		) VALUES (
			$1, $2, 'custom', 1, 'CURRENT',
			'RECOVERY_CRASH_TEST_RULE', 'Recovery Crash Test Rule', 'PRE_TRADE', 'HARD_BLOCK', '{}'::jsonb,
			'{}'::jsonb, 10, true, NOW(), NOW()
		)
	`
	_, err = pgDB.Exec(ruleQ, ruleID, tenantID)
	require.NoError(t, err)

	contentHash2, err := canonical.ComputeRuleContentHashFromRaw([]byte("{}"), []byte("{}"), "Crash Recovery Citation")
	require.NoError(t, err)
	bytecodeHash2 := canonical.ComputeBytecodeHash(nil)

	ruleVerQ := `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
			citation, effective_from, content_hash, compiled_bytecode_hash, created_by
		) VALUES (
			$1, 1, $2, '{}'::jsonb, '{}'::jsonb,
			'Crash Recovery Citation', NOW(), $3, $4, 'test'
		) ON CONFLICT (rule_id, version) DO NOTHING
	`
	_, err = pgDB.Exec(ruleVerQ, ruleID, tenantID, contentHash2, bytecodeHash2)
	require.NoError(t, err)

	defer func() {
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_evaluation_event WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_watermark_checkpoint WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_archive_manifest WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_orphan_object WHERE tenant_id = $1", tenantID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_rule_version WHERE rule_id = $1", ruleID)
		_, _ = pgDB.Exec("DELETE FROM compliance.compliance_rule WHERE id = $1", ruleID)
	}()

	worker := NewColdArchivalWorker(ColdArchivalWorkerConfig{
		PostgresDB:   pgDB,
		S3Client:     s3Client,
		ManifestRepo: manifestRepo,
		BucketName:   "compliance-cold-archive",
		BatchSize:    100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// 1. Insert 5 events (LSN 501..505) into Hot PG
	baseLSN := int64(48000000000)
	var records []cold.CanonicalRecord
	for i := 1; i <= 5; i++ {
		insertQ := `
			INSERT INTO compliance.compliance_evaluation_event (
				id, lineage_id, tenant_id, order_id, rule_id, rule_version, rule_content_hash,
				passed, action_taken, latency_micros, evaluation_hash,
				input_params, metric_snapshots, evaluated_at, created_at, ingest_lsn
			) VALUES (
				$1, $2, $3, $4, $5, 1, $6,
				true, 'APPROVED', 150, $7,
				'{"recovery_test": true}'::jsonb, '{"metric": 500}'::jsonb,
				NOW(), NOW(), $8
			)
		`
		lineageID := uuid.New()
		orderID := uuid.New()
		evalHash := fmt.Sprintf("hash_%s", lineageID.String())
		lsn := baseLSN + int64(i)
		_, err := pgDB.Exec(insertQ, uuid.New(), lineageID, tenantID, orderID, ruleID, contentHash2, evalHash, lsn)
		require.NoError(t, err)

		records = append(records, cold.CanonicalRecord{
			LineageID:       lineageID.String(),
			EvaluatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
			TenantID:        tenantID.String(),
			OrderID:         orderID.String(),
			RuleID:          ruleID.String(),
			RuleVersion:     1,
			RuleContentHash: contentHash2,
			ActionTaken:     "APPROVED",
			Passed:          true,
			LatencyMicros:   150,
			EvaluationHash:  evalHash,
			IngestLSN:       lsn,
			InputParams:     `{"recovery_test":true}`,
			MetricSnapshots: `{"metric":500}`,
			CreatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
		})
	}

	// Set Certified Warm Watermark to baseLSN + 5
	warmLWM := baseLSN + 5
	_, err = pgDB.Exec(`
		INSERT INTO compliance.compliance_watermark_checkpoint (
			id, tenant_id, tier, current_lsn, certified_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, 'WARM', '0/0'::pg_lsn + $2, NOW(), NOW()
		) ON CONFLICT (tenant_id, tier) DO UPDATE SET current_lsn = EXCLUDED.current_lsn
	`, tenantID, warmLWM)
	require.NoError(t, err)

	// 2. SIMULATE WORKER CRASH:
	// Worker uploads Parquet bytes to S3 but crashes BEFORE writing manifest or updating ColdLWM!
	parquetBytes, _, _, err := cold.WriteCanonicalParquet(records)
	require.NoError(t, err)

	s3Key := fmt.Sprintf("cold-archive/tenant_%s/year=2026/month=10/evaluations_%d_%d.parquet",
		tenantID.String(), baseLSN+1, baseLSN+5)

	orphanETag, orphanVersionID, err := s3Client.UploadWORMObjectWithVersion(ctx, "compliance-cold-archive", s3Key, parquetBytes)
	require.NoError(t, err)
	t.Logf("Simulated Crash: WORM Object uploaded to S3 (Key=%s, VersionID=%s), Manifest NOT written", s3Key, orphanVersionID)

	// 3. Quarantine Sweeper runs during startup sweep
	sweeper := NewQuarantineSweeper(s3Client, manifestRepo, "compliance-cold-archive")
	sweepReport, err := sweeper.RunSweep(ctx, fmt.Sprintf("cold-archive/tenant_%s", tenantID.String()))
	require.NoError(t, err)
	require.Equal(t, 1, sweepReport.Quarantined, "Sweeper must quarantine the unmanifested S3 object")

	var orphanStatus string
	err = pgDB.QueryRowContext(ctx, "SELECT status FROM compliance.compliance_orphan_object WHERE s3_key = $1", s3Key).Scan(&orphanStatus)
	require.NoError(t, err)
	require.Equal(t, "QUARANTINED", orphanStatus)
	t.Logf("Orphan object successfully quarantined: status=%s, ETag=%s", orphanStatus, orphanETag)

	// 4. PIPELINE RESUME / WORKER RECOVERY RUN:
	// Worker restarts and executes ArchiveTenantSlice for this tenant
	resRecovered, err := worker.ArchiveTenantSlice(ctx, tenantID)
	require.NoError(t, err, "Worker recovery run must succeed without deadlocking")
	require.Equal(t, int64(5), resRecovered.RowsArchived)
	require.Equal(t, baseLSN+1, resRecovered.StartLSN)
	require.Equal(t, baseLSN+5, resRecovered.EndLSN)
	t.Logf("Worker Recovery Succeeded: Manifest sealed, ColdLWM advanced to %d", resRecovered.EndLSN)

	// Verify manifest entry is now committed
	manifest, err := manifestRepo.GetLastManifest(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, manifest)
	require.Equal(t, resRecovered.MerkleRootHash, manifest.MerkleRootHash)

	// 5. Subsequent Batch Archival: Verify Contiguity Invariant Holds Across Recovery
	for i := 6; i <= 10; i++ {
		insertQ := `
			INSERT INTO compliance.compliance_evaluation_event (
				id, lineage_id, tenant_id, order_id, rule_id, rule_version, rule_content_hash,
				passed, action_taken, latency_micros, evaluation_hash,
				input_params, metric_snapshots, evaluated_at, created_at, ingest_lsn
			) VALUES (
				$1, $2, $3, $4, $5, 1, $6,
				true, 'APPROVED', 150, $7,
				'{"batch": 2}'::jsonb, '{"metric": 600}'::jsonb,
				NOW(), NOW(), $8
			)
		`
		lineageID := uuid.New()
		orderID := uuid.New()
		evalHash := fmt.Sprintf("hash_%s", lineageID.String())
		lsn := baseLSN + int64(i)
		_, err := pgDB.Exec(insertQ, uuid.New(), lineageID, tenantID, orderID, ruleID, contentHash2, evalHash, lsn)
		require.NoError(t, err)
	}

	_, err = pgDB.Exec(`
		UPDATE compliance.compliance_watermark_checkpoint 
		SET current_lsn = '0/0'::pg_lsn + $2, updated_at = NOW()
		WHERE tenant_id = $1 AND tier = 'WARM'
	`, tenantID, baseLSN+10)
	require.NoError(t, err)

	resNext, err := worker.ArchiveTenantSlice(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, baseLSN+6, resNext.StartLSN, "Contiguity MUST hold: next batch starts at recovered EndLSN + 1")
	require.Equal(t, baseLSN+10, resNext.EndLSN)
	t.Logf("Post-Recovery Contiguity PASSED: Batch 2 LSN [%d, %d]", resNext.StartLSN, resNext.EndLSN)

	// 6. CLI Cryptographic Verification of the Recovered Archive
	downloadedParquet, err := s3Client.DownloadObject(ctx, manifest.S3Bucket, manifest.S3Key)
	require.NoError(t, err)

	verifier := cold.NewArchiveVerifier()
	verReport, err := verifier.VerifyParquetSlice(ctx, downloadedParquet, manifest.MerkleRootHash)
	require.NoError(t, err)
	require.True(t, verReport.RootMatch, "Recovered archive slice MUST verify 100%% against Merkle root")
	require.True(t, verReport.InclusionProofs)
	require.Equal(t, "VERIFIED_TAMPER_EVIDENT_MATCH", verReport.AuditStatus)

	t.Logf("CRASH-RECOVERY, PIPELINE RESUME & CLI VERIFICATION VERIFIED 100%%!")
}
