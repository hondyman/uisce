package iceberg

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/sse"
)

// ADR-031: a tenant's warehouse sits on its own bucket, created with Object Lock
// in compliance mode and default SSE-KMS under the tenant's own key.
//
// Compliance-mode retention cannot be shortened or removed once objects are
// locked, so this provisioner never changes an existing bucket's lock or
// encryption settings. It creates what is missing and refuses (with an error)
// to reconcile anything that differs. The retention period is a required input
// with no default: it is a regulatory decision, not a tunable.

// sseNotFoundCode is the S3 error code for "no default encryption configured".
const sseNotFoundCode = "ServerSideEncryptionConfigurationNotFoundError"

// bucketAPI is the subset of *minio.Client this file uses, so the provisioner
// can be tested without a MinIO server.
type bucketAPI interface {
	BucketExists(ctx context.Context, bucket string) (bool, error)
	MakeBucket(ctx context.Context, bucket string, opts minio.MakeBucketOptions) error
	GetObjectLockConfig(ctx context.Context, bucket string) (string, *minio.RetentionMode, *uint, *minio.ValidityUnit, error)
	SetObjectLockConfig(ctx context.Context, bucket string, mode *minio.RetentionMode, validity *uint, unit *minio.ValidityUnit) error
	GetBucketEncryption(ctx context.Context, bucket string) (*sse.Configuration, error)
	SetBucketEncryption(ctx context.Context, bucket string, cfg *sse.Configuration) error
}

// TenantBucketProvisioner creates tenant buckets.
type TenantBucketProvisioner struct {
	api bucketAPI
}

// NewTenantBucketProvisioner wraps a MinIO client authenticated as the platform
// provisioning identity (not a tenant identity).
func NewTenantBucketProvisioner(client *minio.Client) *TenantBucketProvisioner {
	return &TenantBucketProvisioner{api: client}
}

// TenantBucketSpec describes the one bucket of one tenant.
type TenantBucketSpec struct {
	TenantID uuid.UUID
	// KMSKeyID is the tenant's own KMS key. Required: a regulated tenant's
	// bucket is never created unencrypted, and the key is what offboarding
	// destroys.
	KMSKeyID string
	// RetentionDays is the default Object Lock retention, in days. Required and
	// has no default.
	RetentionDays uint
	Region        string
}

// TenantBucket is the result of EnsureTenantBucket.
type TenantBucket struct {
	Name          string
	Created       bool
	RetentionDays uint
	KMSKeyID      string
}

// EnsureTenantBucket creates the tenant's bucket if absent and verifies it is
// WORM and encrypted under the tenant's key. Safe to re-run: a partial earlier
// run is completed, and an already-correct bucket is left untouched.
func (p *TenantBucketProvisioner) EnsureTenantBucket(ctx context.Context, spec TenantBucketSpec) (*TenantBucket, error) {
	name, err := TenantWarehouseName(spec.TenantID)
	if err != nil {
		return nil, err
	}
	if spec.KMSKeyID == "" {
		return nil, errors.New("tenant bucket: a per-tenant KMS key is required")
	}
	if spec.RetentionDays == 0 {
		return nil, errors.New("tenant bucket: retention days is required and has no default")
	}

	exists, err := p.api.BucketExists(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("tenant bucket: check %s: %w", name, err)
	}
	created := false
	if !exists {
		// Object Lock can only be requested here, at creation, and it also
		// enables versioning.
		if err := p.api.MakeBucket(ctx, name, minio.MakeBucketOptions{Region: spec.Region, ObjectLocking: true}); err != nil {
			return nil, fmt.Errorf("tenant bucket: create %s: %w", name, err)
		}
		created = true
	}

	if err := p.ensureObjectLock(ctx, name, spec.RetentionDays); err != nil {
		return nil, err
	}
	if err := p.ensureEncryption(ctx, name, spec.KMSKeyID); err != nil {
		return nil, err
	}
	return &TenantBucket{Name: name, Created: created, RetentionDays: spec.RetentionDays, KMSKeyID: spec.KMSKeyID}, nil
}

func (p *TenantBucketProvisioner) ensureObjectLock(ctx context.Context, name string, wantDays uint) error {
	enabled, mode, validity, unit, err := p.api.GetObjectLockConfig(ctx, name)
	if err != nil {
		return fmt.Errorf("tenant bucket: read object lock on %s: %w", name, err)
	}
	if enabled != "Enabled" {
		// Not repairable here and not safe to continue: this bucket would hold
		// audit with no WORM guarantee.
		return fmt.Errorf("tenant bucket: %s exists without Object Lock; it cannot hold tenant audit and must be replaced", name)
	}

	if mode == nil {
		m := minio.Compliance
		d := wantDays
		u := minio.Days
		if err := p.api.SetObjectLockConfig(ctx, name, &m, &d, &u); err != nil {
			return fmt.Errorf("tenant bucket: set retention on %s: %w", name, err)
		}
		return nil
	}

	if *mode != minio.Compliance {
		return fmt.Errorf("tenant bucket: %s has %s retention, want COMPLIANCE; refusing to change it", name, *mode)
	}
	have := retentionDays(validity, unit)
	if have < wantDays {
		return fmt.Errorf("tenant bucket: %s retains %d days, want at least %d; refusing to change an existing compliance retention", name, have, wantDays)
	}
	return nil
}

func (p *TenantBucketProvisioner) ensureEncryption(ctx context.Context, name, kmsKey string) error {
	cfg, err := p.api.GetBucketEncryption(ctx, name)
	if err != nil {
		if minio.ToErrorResponse(err).Code != sseNotFoundCode {
			return fmt.Errorf("tenant bucket: read encryption on %s: %w", name, err)
		}
		cfg = nil
	}

	if cfg != nil && len(cfg.Rules) > 0 {
		have := cfg.Rules[0].Apply.KmsMasterKeyID
		if have == kmsKey {
			return nil
		}
		// Objects already written are under the other key; changing the default
		// does not re-encrypt them and would split the bucket across two keys.
		return fmt.Errorf("tenant bucket: %s is encrypted under a different key; refusing to change it", name)
	}

	if err := p.api.SetBucketEncryption(ctx, name, sse.NewConfigurationSSEKMS(kmsKey)); err != nil {
		return fmt.Errorf("tenant bucket: set encryption on %s: %w", name, err)
	}
	return nil
}

// retentionDays converts a default-retention period to days. Years are counted
// as 365 days; this is used only to compare against a required minimum.
func retentionDays(validity *uint, unit *minio.ValidityUnit) uint {
	if validity == nil || unit == nil {
		return 0
	}
	if *unit == minio.Years {
		return *validity * 365
	}
	return *validity
}
