package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance/cold"
)

// QuarantineReport summarizes the results of a cold archive quarantine sweep
type QuarantineReport struct {
	TotalScanned int `json:"total_scanned"`
	Manifested   int `json:"manifested"`
	Quarantined  int `json:"quarantined"`
	DurationMs   int `json:"duration_ms"`
}

// QuarantineSweeper reconciles objects in S3 storage against compliance_archive_manifest
type QuarantineSweeper struct {
	s3Client     *cold.S3StorageClient
	manifestRepo *cold.ManifestRepository
	bucketName   string
}

// NewQuarantineSweeper creates a new QuarantineSweeper instance
func NewQuarantineSweeper(s3Client *cold.S3StorageClient, manifestRepo *cold.ManifestRepository, bucketName string) *QuarantineSweeper {
	if bucketName == "" {
		bucketName = "compliance-cold-archive"
	}
	return &QuarantineSweeper{
		s3Client:     s3Client,
		manifestRepo: manifestRepo,
		bucketName:   bucketName,
	}
}

// RunSweep iterates all objects in the bucket and logs any unmanifested object to compliance_orphan_object
func (s *QuarantineSweeper) RunSweep(ctx context.Context, prefix string) (*QuarantineReport, error) {
	start := time.Now()

	objects, err := s.s3Client.ListObjects(ctx, s.bucketName, prefix)
	if err != nil {
		return nil, fmt.Errorf("list s3 objects for sweep: %w", err)
	}

	report := &QuarantineReport{
		TotalScanned: len(objects),
	}

	for _, obj := range objects {
		manifested, err := s.manifestRepo.IsObjectManifested(ctx, obj.Bucket, obj.Key)
		if err != nil {
			return nil, fmt.Errorf("check object manifestation (key=%s): %w", obj.Key, err)
		}

		if manifested {
			report.Manifested++
		} else {
			notes := fmt.Sprintf("Unmanifested object discovered by QuarantineSweeper (size=%d bytes)", obj.Size)
			orphan := &cold.OrphanObjectRecord{
				ID:           uuid.New(),
				S3Bucket:     obj.Bucket,
				S3Key:        obj.Key,
				ETag:         obj.ETag,
				DiscoveredAt: time.Now().UTC(),
				Status:       cold.OrphanStatusQuarantined,
				TriageNotes:  &notes,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			}

			if err := s.manifestRepo.InsertOrphan(ctx, orphan); err != nil {
				return nil, fmt.Errorf("quarantine orphan object (key=%s): %w", obj.Key, err)
			}
			report.Quarantined++
		}
	}

	report.DurationMs = int(time.Since(start).Milliseconds())
	return report, nil
}
