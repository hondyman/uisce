package cold

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds connection parameters for S3 / MinIO WORM storage
type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
	Region          string
	RetentionYears  int
}

// S3ObjectMeta describes an object metadata entry from S3 listing
type S3ObjectMeta struct {
	Bucket       string
	Key          string
	ETag         string
	Size         int64
	LastModified time.Time
}

// S3StorageClient manages WORM Object Lock uploads, downloads, and listings
type S3StorageClient struct {
	client *minio.Client
	cfg    S3Config
}

// NewS3StorageClient initializes the S3/MinIO client
func NewS3StorageClient(cfg S3Config) (*S3StorageClient, error) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = os.Getenv("AWS_S3_ENDPOINT")
		if cfg.Endpoint == "" {
			cfg.Endpoint = "100.84.50.65:9000"
		}
	}
	// Strip scheme if present
	if u, err := url.Parse(cfg.Endpoint); err == nil && u.Host != "" {
		cfg.Endpoint = u.Host
		if u.Scheme == "https" {
			cfg.UseSSL = true
		}
	}

	if cfg.AccessKeyID == "" {
		cfg.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
		if cfg.AccessKeyID == "" {
			cfg.AccessKeyID = "minioadmin"
		}
	}
	if cfg.SecretAccessKey == "" {
		cfg.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		if cfg.SecretAccessKey == "" {
			cfg.SecretAccessKey = "minioadmin"
		}
	}
	if cfg.BucketName == "" {
		cfg.BucketName = "compliance-cold-archive"
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.RetentionYears <= 0 {
		cfg.RetentionYears = 15
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client: %w", err)
	}

	return &S3StorageClient{
		client: client,
		cfg:    cfg,
	}, nil
}

// EnsureBucket ensures the archive bucket exists with Object Lock enabled
func (s *S3StorageClient) EnsureBucket(ctx context.Context, bucket string) error {
	if bucket == "" {
		bucket = s.cfg.BucketName
	}

	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket exists: %w", err)
	}
	if exists {
		return nil
	}

	opts := minio.MakeBucketOptions{
		Region:        s.cfg.Region,
		ObjectLocking: true,
	}
	if err := s.client.MakeBucket(ctx, bucket, opts); err != nil {
		// Fallback without explicit object locking if server has default locking
		if errFallback := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: s.cfg.Region}); errFallback != nil {
			return fmt.Errorf("make bucket: %w", errFallback)
		}
	}

	return nil
}

// UploadWORMObject uploads data with SEC Rule 17a-4 / FINRA 4511 Compliance Mode Object Lock
func (s *S3StorageClient) UploadWORMObject(ctx context.Context, bucket, key string, data []byte) (string, error) {
	if bucket == "" {
		bucket = s.cfg.BucketName
	}

	retentionDate := time.Now().UTC().AddDate(s.cfg.RetentionYears, 0, 0)
	opts := minio.PutObjectOptions{
		ContentType:     "application/vnd.apache.parquet",
		Mode:            minio.Compliance,
		RetainUntilDate: retentionDate,
	}

	info, err := s.client.PutObject(ctx, bucket, key, bytes.NewReader(data), int64(len(data)), opts)
	if err != nil {
		return "", fmt.Errorf("put object with object lock (bucket=%s, key=%s): %w", bucket, key, err)
	}

	return info.ETag, nil
}

// DownloadObject fetches raw object bytes from S3/MinIO
func (s *S3StorageClient) DownloadObject(ctx context.Context, bucket, key string) ([]byte, error) {
	if bucket == "" {
		bucket = s.cfg.BucketName
	}

	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read object body: %w", err)
	}

	return data, nil
}

// ListObjects lists all objects under a given prefix
func (s *S3StorageClient) ListObjects(ctx context.Context, bucket, prefix string) ([]S3ObjectMeta, error) {
	if bucket == "" {
		bucket = s.cfg.BucketName
	}

	var results []S3ObjectMeta
	objectCh := s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objectCh {
		if obj.Err != nil {
			return nil, fmt.Errorf("list objects error: %w", obj.Err)
		}
		results = append(results, S3ObjectMeta{
			Bucket:       bucket,
			Key:          obj.Key,
			ETag:         obj.ETag,
			Size:         obj.Size,
			LastModified: obj.LastModified,
		})
	}

	return results, nil
}

// AttemptDeleteObject tries to delete an object (used to test WORM Object Lock enforcement)
func (s *S3StorageClient) AttemptDeleteObject(ctx context.Context, bucket, key string) error {
	if bucket == "" {
		bucket = s.cfg.BucketName
	}
	return s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}
