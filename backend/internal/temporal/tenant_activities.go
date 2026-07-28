package temporal

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type TenantActivities struct {
	DB         *sql.DB
	S3Client   *s3.Client
	HttpClient *http.Client
}

type TenantProvisionParams struct {
	TenantID  string
	TenantKey string
}

// 1. PostgreSQL Tenant Creation & Compensation
func (a *TenantActivities) CreatePostgresTenant(ctx context.Context, p TenantProvisionParams) error {
	if a.DB == nil {
		return nil
	}
	query := `INSERT INTO tenants (id, key, status) VALUES ($1, $2, 'PROVISIONING') ON CONFLICT (id) DO NOTHING`
	_, err := a.DB.ExecContext(ctx, query, p.TenantID, p.TenantKey)
	if err != nil {
		return fmt.Errorf("failed to insert tenant into postgres: %w", err)
	}
	return nil
}

func (a *TenantActivities) RollbackPostgresTenant(ctx context.Context, p TenantProvisionParams) error {
	if a.DB == nil {
		return nil
	}
	query := `DELETE FROM tenants WHERE id = $1`
	_, err := a.DB.ExecContext(ctx, query, p.TenantID)
	return err
}

// 2. MinIO S3 Object Storage Prefix Initialization & Compensation
func (a *TenantActivities) InitializeMinIOPrefix(ctx context.Context, p TenantProvisionParams) error {
	if a.S3Client == nil {
		return nil
	}
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "iceberg-warehouse"
	}
	key := fmt.Sprintf("%s/.keep", p.TenantKey)

	_, err := a.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   nil,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize s3 prefix for tenant %s: %w", p.TenantKey, err)
	}
	return nil
}

func (a *TenantActivities) RollbackMinIOPrefix(ctx context.Context, p TenantProvisionParams) error {
	if a.S3Client == nil {
		return nil
	}
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "iceberg-warehouse"
	}
	key := fmt.Sprintf("%s/.keep", p.TenantKey)

	_, err := a.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

// 3. Apache Polaris Catalog Provisioning & Compensation
func (a *TenantActivities) ProvisionPolarisCatalog(ctx context.Context, p TenantProvisionParams) error {
	return nil
}

func (a *TenantActivities) DeprovisionPolarisCatalog(ctx context.Context, p TenantProvisionParams) error {
	return nil
}

// 4. Lakehouse Maintenance Activities
func (a *TenantActivities) ExpireIcebergSnapshots(ctx context.Context, catalogName string) error {
	return nil
}

func (a *TenantActivities) RemoveOrphanFiles(ctx context.Context, catalogName string) error {
	return nil
}

func (a *TenantActivities) CompactManifests(ctx context.Context, catalogName string) error {
	return nil
}
