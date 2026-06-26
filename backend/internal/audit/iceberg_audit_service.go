//go:build !no_parquet

package audit

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/segmentio/parquet-go"
)

// IcebergAuditService handles writing audit logs to object storage in Parquet format
type IcebergAuditService struct {
	client     *minio.Client
	bucketName string
}

// AuditRecord represents the schema for the audit log
type AuditRecord struct {
	EventID      string `parquet:"event_id"`
	EventType    string `parquet:"event_type"`
	TenantID     string `parquet:"tenant_id"`
	ConnectionID string `parquet:"connection_id"`
	Action       string `parquet:"action"`
	UserID       string `parquet:"user_id"`
	Timestamp    int64  `parquet:"timestamp_ms"` // Epoch millis
	Data         string `parquet:"data_json"`    // JSON string info
}

func NewIcebergAuditService() (*IcebergAuditService, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	if accessKeyID == "" {
		accessKeyID = "minioadmin"
	}
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
	if secretAccessKey == "" {
		secretAccessKey = "minioadmin"
	}
	bucketName := os.Getenv("AUDIT_BUCKET")
	if bucketName == "" {
		bucketName = "audit-logs"
	}

	useSSL := false // Dev env usually false

	// Initialize MinIO client object
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &IcebergAuditService{
		client:     minioClient,
		bucketName: bucketName,
	}, nil
}

func (s *IcebergAuditService) WriteEvent(ctx context.Context, event events.GoldCopyConnectionEvent) error {
	// Create Audit Record
	userID := ""
	if event.UserID != nil {
		userID = *event.UserID
	}

	record := AuditRecord{
		EventID:      event.EventID,
		EventType:    string(event.EventType),
		TenantID:     event.TenantID,
		ConnectionID: event.ConnectionID,
		Action:       event.Action,
		UserID:       userID,
		Timestamp:    event.Timestamp.UnixMilli(),
		Data:         fmt.Sprintf("%v", event.ConnectionData), // Ideally marshal to JSON
	}

	// Write to Parquet Buffer
	var buf bytes.Buffer
	writer := parquet.NewWriter(&buf)
	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write parquet record: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close parquet writer: %w", err)
	}

	// Determine object path (Hive Partitioning: year=YYYY/month=MM/day=DD)
	t := event.Timestamp
	objectPath := fmt.Sprintf("year=%d/month=%02d/day=%02d/%s.parquet",
		t.Year(), t.Month(), t.Day(), event.EventID)

	// Upload to MinIO
	// Ensure bucket exists first? (Optimization: assume exists or check once)
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	_, err = s.client.PutObject(ctx, s.bucketName, objectPath, &buf, int64(buf.Len()), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("failed to upload audit log: %w", err)
	}

	return nil
}
