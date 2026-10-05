package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance/cold"
)

// ColdArchivalWorkerConfig holds configuration for the WORM Cold Tier Archival Worker
type ColdArchivalWorkerConfig struct {
	PostgresDB   *sql.DB
	S3Client     *cold.S3StorageClient
	ManifestRepo *cold.ManifestRepository
	BucketName   string
	BatchSize    int
}

// ArchiveResult contains metadata describing a sealed archival slice
type ArchiveResult struct {
	TenantID       uuid.UUID `json:"tenant_id"`
	RowsArchived   int64     `json:"rows_archived"`
	StartLSN       int64     `json:"start_lsn"`
	EndLSN         int64     `json:"end_lsn"`
	MerkleRootHash string    `json:"merkle_root_hash"`
	S3Bucket       string    `json:"s3_bucket"`
	S3Key          string    `json:"s3_key"`
	ETag           string    `json:"etag"`
	FileSizeBytes  int64     `json:"file_size_bytes"`
	DurationMs     int64     `json:"duration_ms"`
}

// ColdArchivalWorker coordinates the LSN-watermarked assembly of immutable WORM cold archive slices
type ColdArchivalWorker struct {
	cfg          ColdArchivalWorkerConfig
	manifestRepo *cold.ManifestRepository
}

// NewColdArchivalWorker instantiates a new ColdArchivalWorker
func NewColdArchivalWorker(cfg ColdArchivalWorkerConfig) *ColdArchivalWorker {
	if cfg.BucketName == "" {
		cfg.BucketName = os.Getenv("COMPLIANCE_ARCHIVE_BUCKET")
		if cfg.BucketName == "" {
			cfg.BucketName = "compliance-cold-archive"
		}
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 5000
	}
	if cfg.ManifestRepo == nil {
		cfg.ManifestRepo = cold.NewManifestRepository(cfg.PostgresDB)
	}

	return &ColdArchivalWorker{
		cfg:          cfg,
		manifestRepo: cfg.ManifestRepo,
	}
}

// GetWatermark retrieves the certified LSN watermark for a tenant tier ('WARM' or 'COLD')
func (w *ColdArchivalWorker) GetWatermark(ctx context.Context, tenantID uuid.UUID, tier string) (int64, error) {
	var lsnInt int64
	query := `
		SELECT (current_lsn - '0/0'::pg_lsn)::bigint
		FROM compliance.compliance_watermark_checkpoint
		WHERE tenant_id = $1 AND tier = $2
	`
	err := w.cfg.PostgresDB.QueryRowContext(ctx, query, tenantID, tier).Scan(&lsnInt)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query %s watermark: %w", tier, err)
	}
	return lsnInt, nil
}

// UpdateColdWatermark checkpoints the certified COLD watermark in PostgreSQL
func (w *ColdArchivalWorker) UpdateColdWatermark(ctx context.Context, tenantID uuid.UUID, lsn int64) error {
	query := `
		INSERT INTO compliance.compliance_watermark_checkpoint (
			id, tenant_id, tier, current_lsn, certified_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, 'COLD', '0/0'::pg_lsn + $2, NOW(), NOW()
		)
		ON CONFLICT (tenant_id, tier) DO UPDATE SET
			current_lsn = EXCLUDED.current_lsn,
			certified_at = EXCLUDED.certified_at,
			updated_at = NOW()
	`
	_, err := w.cfg.PostgresDB.ExecContext(ctx, query, tenantID, lsn)
	if err != nil {
		return fmt.Errorf("update cold watermark: %w", err)
	}
	return nil
}

// ArchiveTenantSlice extracts a contiguous LSN slice up to the certified Warm LWM and seals it into WORM S3
func (w *ColdArchivalWorker) ArchiveTenantSlice(ctx context.Context, tenantID uuid.UUID) (*ArchiveResult, error) {
	start := time.Now()

	// 1. Check watermarks: Cold LWM (last archived) vs Certified Warm LWM
	coldLWM, err := w.GetWatermark(ctx, tenantID, "COLD")
	if err != nil {
		return nil, err
	}

	warmLWM, err := w.GetWatermark(ctx, tenantID, "WARM")
	if err != nil {
		return nil, err
	}

	if warmLWM <= coldLWM {
		// Nothing new certified in Warm Tier to archive
		return &ArchiveResult{
			TenantID:     tenantID,
			RowsArchived: 0,
			StartLSN:     coldLWM,
			EndLSN:       coldLWM,
			DurationMs:   time.Since(start).Milliseconds(),
		}, nil
	}

	// 2. Retrieve previous manifest to guarantee contiguity
	lastManifest, err := w.manifestRepo.GetLastManifest(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("retrieve last manifest: %w", err)
	}

	// 3. Query records from Hot/Warm database strictly within (coldLWM, warmLWM]
	query := `
		SELECT 
			lineage_id, evaluated_at, tenant_id, order_id, rule_id, rule_version,
			action_taken, passed, latency_micros, evaluation_hash, ingest_lsn,
			COALESCE(input_params, '{}'::jsonb), COALESCE(metric_snapshots, '{}'::jsonb), created_at
		FROM compliance.compliance_evaluation_event
		WHERE tenant_id = $1
		  AND ingest_lsn > $2
		  AND ingest_lsn <= $3
		ORDER BY ingest_lsn ASC
		LIMIT $4
	`

	rows, err := w.cfg.PostgresDB.QueryContext(ctx, query, tenantID, coldLWM, warmLWM, w.cfg.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("query records for archival: %w", err)
	}
	defer rows.Close()

	var records []cold.CanonicalRecord
	for rows.Next() {
		var r cold.CanonicalRecord
		var evaluatedAt, createdAt time.Time
		var lineageID, tenantIDStr, ruleID string
		var orderID sql.NullString
		var inputParams, metricSnapshots []byte

		if err := rows.Scan(
			&lineageID, &evaluatedAt, &tenantIDStr, &orderID, &ruleID, &r.RuleVersion,
			&r.ActionTaken, &r.Passed, &r.LatencyMicros, &r.EvaluationHash, &r.IngestLSN,
			&inputParams, &metricSnapshots, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan archival row: %w", err)
		}

		r.LineageID = lineageID
		r.TenantID = tenantIDStr
		r.RuleID = ruleID
		if orderID.Valid {
			r.OrderID = orderID.String
		}
		r.EvaluatedAt = evaluatedAt.UTC().Format(time.RFC3339Nano)
		r.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		r.InputParams = string(inputParams)
		r.MetricSnapshots = string(metricSnapshots)

		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate archival rows: %w", err)
	}

	if len(records) == 0 {
		// Advance cold watermark to warmLWM if interval contained no rows for this tenant
		if err := w.UpdateColdWatermark(ctx, tenantID, warmLWM); err != nil {
			return nil, err
		}
		return &ArchiveResult{
			TenantID:     tenantID,
			RowsArchived: 0,
			StartLSN:     coldLWM,
			EndLSN:       warmLWM,
			DurationMs:   time.Since(start).Milliseconds(),
		}, nil
	}

	batchStartLSN := records[0].IngestLSN
	batchEndLSN := records[len(records)-1].IngestLSN

	// 4. Verify strict manifest contiguity
	if err := cold.VerifyContiguity(lastManifest, batchStartLSN, batchEndLSN); err != nil {
		return nil, fmt.Errorf("manifest contiguity verification failed: %w", err)
	}

	// 5. Assemble deterministic Parquet file and Merkle Root
	parquetBytes, summary, _, err := cold.WriteCanonicalParquet(records)
	if err != nil {
		return nil, fmt.Errorf("assemble canonical parquet: %w", err)
	}

	// 6. Format deterministic S3 object key
	evalTime, _ := time.Parse(time.RFC3339Nano, records[0].EvaluatedAt)
	s3Key := fmt.Sprintf("cold-archive/tenant_%s/year=%04d/month=%02d/evaluations_%d_%d.parquet",
		tenantID.String(), evalTime.Year(), int(evalTime.Month()), batchStartLSN, batchEndLSN)

	// Ensure bucket exists
	if err := w.cfg.S3Client.EnsureBucket(ctx, w.cfg.BucketName); err != nil {
		return nil, fmt.Errorf("ensure archive bucket: %w", err)
	}

	// 7. Upload to S3/MinIO with WORM Compliance Mode Object Lock
	etag, err := w.cfg.S3Client.UploadWORMObject(ctx, w.cfg.BucketName, s3Key, parquetBytes)
	if err != nil {
		return nil, fmt.Errorf("upload worm object to s3: %w", err)
	}

	// 8. Atomically insert Manifest Record
	manifestID := uuid.New()
	manifestRecord := &cold.ArchiveManifestRecord{
		ID:             manifestID,
		TenantID:       tenantID,
		LWMStartLSN:    batchStartLSN,
		LWMEndLSN:      batchEndLSN,
		MerkleRootHash: summary.MerkleRoot,
		S3Bucket:       w.cfg.BucketName,
		S3Key:          s3Key,
		ETag:           etag,
		RecordCount:    summary.RowCount,
		FileSizeBytes:  summary.FileSizeBytes,
		Status:         cold.ManifestStatusSealed,
		SealedAt:       time.Now().UTC(),
		CreatedAt:      time.Now().UTC(),
	}

	if err := w.manifestRepo.InsertManifest(ctx, manifestRecord); err != nil {
		return nil, fmt.Errorf("commit archive manifest: %w", err)
	}

	// 9. Update certified COLD watermark
	if err := w.UpdateColdWatermark(ctx, tenantID, batchEndLSN); err != nil {
		return nil, fmt.Errorf("checkpoint cold watermark: %w", err)
	}

	return &ArchiveResult{
		TenantID:       tenantID,
		RowsArchived:   summary.RowCount,
		StartLSN:       batchStartLSN,
		EndLSN:         batchEndLSN,
		MerkleRootHash: summary.MerkleRoot,
		S3Bucket:       w.cfg.BucketName,
		S3Key:          s3Key,
		ETag:           etag,
		FileSizeBytes:  summary.FileSizeBytes,
		DurationMs:     time.Since(start).Milliseconds(),
	}, nil
}
