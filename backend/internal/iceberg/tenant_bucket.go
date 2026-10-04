package iceberg

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/sse"
)

// ADR-032: a tenant's warehouse sits on its own bucket, created with Object Lock
// in compliance mode and default SSE-KMS under the tenant's own key.
//
// Compliance-mode retention cannot be shortened or removed once objects are
// locked, so this provisioner never changes an existing bucket's lock or
// encryption settings. It creates what is missing and refuses (with an error)
// to reconcile anything that differs. The retention period is a required input
// with no default: it is a regulatory decision, not a tunable.

// ErrBucketConflict marks a refusal to reconcile an existing bucket: it has no Object
// Lock, a non-compliance or shorter retention, or a different key. Retrying cannot
// fix it, and it must reach a human, so a caller treats it as non-retryable.
var ErrBucketConflict = errors.New("tenant bucket conflicts with the required settings")

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
	return p.ensureBucket(ctx, "tenant bucket", name, spec.KMSKeyID, spec.RetentionDays, spec.Region)
}

// ensureBucket is the one implementation of "WORM and encrypted under this key", shared by the tenant
// bucket and the platform's ivy-control bucket so the two can never differ in what they enforce. The name
// is always derived by the caller from a fixed rule; it is never taken from user input here.
func (p *TenantBucketProvisioner) ensureBucket(ctx context.Context, what, name, kmsKeyID string, retentionDays uint, region string) (*TenantBucket, error) {
	if kmsKeyID == "" {
		return nil, fmt.Errorf("%s: a KMS key is required", what)
	}
	if retentionDays == 0 {
		return nil, fmt.Errorf("%s: retention days is required and has no default", what)
	}

	exists, err := p.api.BucketExists(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%s: check %s: %w", what, name, err)
	}
	created := false
	if !exists {
		// Object Lock can only be requested here, at creation, and it also
		// enables versioning.
		if err := p.api.MakeBucket(ctx, name, minio.MakeBucketOptions{Region: region, ObjectLocking: true}); err != nil {
			return nil, fmt.Errorf("%s: create %s: %w", what, name, err)
		}
		created = true
	}

	if err := p.ensureObjectLock(ctx, name, retentionDays); err != nil {
		return nil, err
	}
	if err := p.ensureEncryption(ctx, name, kmsKeyID); err != nil {
		return nil, err
	}
	return &TenantBucket{Name: name, Created: created, RetentionDays: retentionDays, KMSKeyID: kmsKeyID}, nil
}

// ExtendTenantRetention raises the default Object Lock retention of an existing tenant bucket
// to at least `days`, and returns `days`: the retention the bucket now enforces by default is
// at least that. It only ever raises:
//   - a bucket already enforcing `days` or more is left untouched (a retry is a no-op);
//   - a bucket with no Object Lock, or in GOVERNANCE mode, is ErrBucketConflict and untouched;
//   - a missing bucket is ErrBucketConflict, because a tenant recorded as provisioned must
//     have one and that mismatch needs a person.
//
// What this changes: the DEFAULT retention, which applies to objects written from now on.
// Objects already in the bucket keep the retain-until date they were stamped with; extending
// them is a separate, per-object operation this does not perform.
func (p *TenantBucketProvisioner) ExtendTenantRetention(ctx context.Context, tenantID uuid.UUID, days uint) (uint, error) {
	name, err := TenantWarehouseName(tenantID)
	if err != nil {
		return 0, err
	}
	if days == 0 {
		return 0, errors.New("tenant bucket: retention days is required")
	}
	exists, err := p.api.BucketExists(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("tenant bucket: check %s: %w", name, err)
	}
	if !exists {
		return 0, fmt.Errorf("%w: %s does not exist but the tenant is recorded as provisioned", ErrBucketConflict, name)
	}

	enabled, mode, validity, unit, err := p.api.GetObjectLockConfig(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("tenant bucket: read object lock on %s: %w", name, err)
	}
	if enabled != "Enabled" {
		return 0, fmt.Errorf("%w: %s has no Object Lock", ErrBucketConflict, name)
	}
	if mode != nil && *mode != minio.Compliance {
		return 0, fmt.Errorf("%w: %s has %s retention, want COMPLIANCE; refusing to change it", ErrBucketConflict, name, *mode)
	}
	if mode != nil && retentionDays(validity, unit) >= days {
		return days, nil // already enforces at least this much
	}

	m, d, u := minio.Compliance, days, minio.Days
	if err := p.api.SetObjectLockConfig(ctx, name, &m, &d, &u); err != nil {
		return 0, fmt.Errorf("tenant bucket: extend retention on %s: %w", name, err)
	}
	// Do not trust the write: read it back, so the registry never records what the bucket
	// does not enforce.
	_, mode, validity, unit, err = p.api.GetObjectLockConfig(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("tenant bucket: verify retention on %s: %w", name, err)
	}
	if mode == nil || *mode != minio.Compliance || retentionDays(validity, unit) < days {
		return 0, fmt.Errorf("tenant bucket: %s did not accept %d days of retention", name, days)
	}
	return days, nil
}

func (p *TenantBucketProvisioner) ensureObjectLock(ctx context.Context, name string, wantDays uint) error {
	enabled, mode, validity, unit, err := p.api.GetObjectLockConfig(ctx, name)
	if err != nil {
		return fmt.Errorf("tenant bucket: read object lock on %s: %w", name, err)
	}
	if enabled != "Enabled" {
		// Not repairable here and not safe to continue: this bucket would hold
		// audit with no WORM guarantee.
		return fmt.Errorf("%w: %s exists without Object Lock; it cannot hold tenant audit and must be replaced", ErrBucketConflict, name)
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
		return fmt.Errorf("%w: %s has %s retention, want COMPLIANCE; refusing to change it", ErrBucketConflict, name, *mode)
	}
	have := retentionDays(validity, unit)
	if have < wantDays {
		return fmt.Errorf("%w: %s retains %d days, want at least %d; refusing to change an existing compliance retention", ErrBucketConflict, name, have, wantDays)
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
		return fmt.Errorf("%w: %s is encrypted under a different key; refusing to change it", ErrBucketConflict, name)
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
