package cold

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ManifestStatus represents the lifecycle state of a cold archive manifest entry
type ManifestStatus string

const (
	ManifestStatusPendingSeal ManifestStatus = "PENDING_SEAL"
	ManifestStatusSealed      ManifestStatus = "SEALED"
	ManifestStatusQuarantined ManifestStatus = "QUARANTINED"
	ManifestStatusVerified    ManifestStatus = "VERIFIED"
)

// OrphanStatus represents the triage status of an unmanifested S3 object
type OrphanStatus string

const (
	OrphanStatusQuarantined   OrphanStatus = "QUARANTINED"
	OrphanStatusInvestigating OrphanStatus = "INVESTIGATING"
	OrphanStatusResolved      OrphanStatus = "RESOLVED"
	OrphanStatusDiscarded     OrphanStatus = "DISCARDED"
)

// ArchiveManifestRecord models the database row in compliance.compliance_archive_manifest
type ArchiveManifestRecord struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	LWMStartLSN    int64          `json:"lwm_start_lsn"`
	LWMEndLSN      int64          `json:"lwm_end_lsn"`
	MerkleRootHash string         `json:"merkle_root_hash"`
	S3Bucket       string         `json:"s3_bucket"`
	S3Key          string         `json:"s3_key"`
	ETag           string         `json:"etag"`
	RecordCount    int64          `json:"record_count"`
	FileSizeBytes  int64          `json:"file_size_bytes"`
	Status         ManifestStatus `json:"status"`
	SealedAt       time.Time      `json:"sealed_at"`
	CreatedAt      time.Time      `json:"created_at"`
}

// OrphanObjectRecord models the database row in compliance.compliance_orphan_object
type OrphanObjectRecord struct {
	ID           uuid.UUID    `json:"id"`
	TenantID     *uuid.UUID   `json:"tenant_id,omitempty"`
	S3Bucket     string       `json:"s3_bucket"`
	S3Key        string       `json:"s3_key"`
	ETag         string       `json:"etag"`
	DiscoveredAt time.Time    `json:"discovered_at"`
	Status       OrphanStatus `json:"status"`
	TriageNotes  *string      `json:"triage_notes,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ManifestRepository provides database operations for cold archive manifests and quarantine triage
type ManifestRepository struct {
	db *sql.DB
}

// NewManifestRepository creates a new ManifestRepository
func NewManifestRepository(db *sql.DB) *ManifestRepository {
	return &ManifestRepository{db: db}
}

// GetLastManifest retrieves the most recent sealed manifest for a given tenant ordered by lwm_end_lsn DESC
func (r *ManifestRepository) GetLastManifest(ctx context.Context, tenantID uuid.UUID) (*ArchiveManifestRecord, error) {
	query := `
		SELECT 
			id, tenant_id, 
			(lwm_start_lsn - '0/0'::pg_lsn)::bigint, 
			(lwm_end_lsn - '0/0'::pg_lsn)::bigint,
			merkle_root_hash, s3_bucket, s3_key, etag, record_count, file_size_bytes,
			status, sealed_at, created_at
		FROM compliance.compliance_archive_manifest
		WHERE tenant_id = $1 AND status IN ('SEALED', 'VERIFIED')
		ORDER BY lwm_end_lsn DESC
		LIMIT 1
	`
	var m ArchiveManifestRecord
	var statusStr string

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&m.ID, &m.TenantID, &m.LWMStartLSN, &m.LWMEndLSN,
		&m.MerkleRootHash, &m.S3Bucket, &m.S3Key, &m.ETag, &m.RecordCount, &m.FileSizeBytes,
		&statusStr, &m.SealedAt, &m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query last manifest: %w", err)
	}
	m.Status = ManifestStatus(statusStr)
	return &m, nil
}

// VerifyContiguity enforces strict audit contiguity without gaps or overlaps (newStart == lastEnd + 1)
func VerifyContiguity(lastManifest *ArchiveManifestRecord, newStartLSN, newEndLSN int64) error {
	if newStartLSN > newEndLSN {
		return fmt.Errorf("invalid lsn batch range: start_lsn %d > end_lsn %d", newStartLSN, newEndLSN)
	}

	if lastManifest == nil {
		// Genesis batch for this tenant
		return nil
	}

	expectedStart := lastManifest.LWMEndLSN + 1
	if newStartLSN != expectedStart {
		if newStartLSN <= lastManifest.LWMEndLSN {
			return fmt.Errorf("manifest overlap detected: new batch start_lsn %d <= previous batch end_lsn %d (expected %d)",
				newStartLSN, lastManifest.LWMEndLSN, expectedStart)
		}
		return fmt.Errorf("manifest audit gap detected: new batch start_lsn %d > previous batch end_lsn %d (expected %d, gap size %d)",
			newStartLSN, lastManifest.LWMEndLSN, expectedStart, newStartLSN-expectedStart)
	}

	return nil
}

// InsertManifest writes a sealed archive manifest row
func (r *ManifestRepository) InsertManifest(ctx context.Context, m *ArchiveManifestRecord) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.Status == "" {
		m.Status = ManifestStatusSealed
	}
	if m.SealedAt.IsZero() {
		m.SealedAt = time.Now().UTC()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO compliance.compliance_archive_manifest (
			id, tenant_id, lwm_start_lsn, lwm_end_lsn, merkle_root_hash,
			s3_bucket, s3_key, etag, record_count, file_size_bytes,
			status, sealed_at, created_at
		) VALUES (
			$1, $2, '0/0'::pg_lsn + $3, '0/0'::pg_lsn + $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.TenantID, m.LWMStartLSN, m.LWMEndLSN, m.MerkleRootHash,
		m.S3Bucket, m.S3Key, m.ETag, m.RecordCount, m.FileSizeBytes,
		string(m.Status), m.SealedAt, m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert archive manifest: %w", err)
	}
	return nil
}

// IsObjectManifested checks if an S3 bucket/key pair is recorded in the manifest repository
func (r *ManifestRepository) IsObjectManifested(ctx context.Context, bucket, key string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM compliance.compliance_archive_manifest WHERE s3_bucket = $1 AND s3_key = $2`
	err := r.db.QueryRowContext(ctx, query, bucket, key).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check object manifested: %w", err)
	}
	return count > 0, nil
}

// InsertOrphan logs an unmanifested or orphaned S3 object to compliance.compliance_orphan_object
func (r *ManifestRepository) InsertOrphan(ctx context.Context, orphan *OrphanObjectRecord) error {
	if orphan.ID == uuid.Nil {
		orphan.ID = uuid.New()
	}
	if orphan.Status == "" {
		orphan.Status = OrphanStatusQuarantined
	}
	if orphan.DiscoveredAt.IsZero() {
		orphan.DiscoveredAt = time.Now().UTC()
	}
	if orphan.CreatedAt.IsZero() {
		orphan.CreatedAt = time.Now().UTC()
	}
	if orphan.UpdatedAt.IsZero() {
		orphan.UpdatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO compliance.compliance_orphan_object (
			id, tenant_id, s3_bucket, s3_key, etag,
			discovered_at, status, triage_notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10
		)
		ON CONFLICT (s3_bucket, s3_key) DO UPDATE SET
			etag = EXCLUDED.etag,
			status = EXCLUDED.status,
			triage_notes = EXCLUDED.triage_notes,
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query,
		orphan.ID, orphan.TenantID, orphan.S3Bucket, orphan.S3Key, orphan.ETag,
		orphan.DiscoveredAt, string(orphan.Status), orphan.TriageNotes, orphan.CreatedAt, orphan.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert orphan object: %w", err)
	}
	return nil
}
