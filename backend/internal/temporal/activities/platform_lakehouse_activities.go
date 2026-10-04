package activities

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
)

// Activities that provision the platform warehouse, ivy-control (ADR-032, ADR-047). The same steps as a tenant's, in
// the same order and with the same rules, over the platform's own registry row and credential path. There is no
// tenant here, so there is no tenant id, no tenant transaction, and nothing that reuses a tenant's credential space.

// PlatformRegistry is the platform warehouse's one registry row.
type PlatformRegistry interface {
	Get(ctx context.Context) (*registry.PlatformConfig, error)
	ConfigureRetention(ctx context.Context, days int) error
	MarkCredentialIssued(ctx context.Context) error
	MarkProvisioned(ctx context.Context, warehouseID uuid.UUID, kmsKeyID string, appliedDays int) error
}

// PlatformBuckets creates the ivy-control bucket.
type PlatformBuckets interface {
	EnsureControlBucket(ctx context.Context, spec iceberg.ControlBucketSpec) (*iceberg.TenantBucket, error)
}

// PlatformWarehouses creates the ivy-control warehouse.
type PlatformWarehouses interface {
	EnsureControlWarehouse(ctx context.Context, spec iceberg.ControlWarehouseSpec) (*iceberg.TenantWarehouse, error)
}

// PlatformLakehouseActivities holds the dependencies. Build it once per worker.
type PlatformLakehouseActivities struct {
	Registry    PlatformRegistry
	Keys        LakehouseKeys
	Buckets     PlatformBuckets
	Credentials infra.PlatformCredentials
	Warehouses  PlatformWarehouses
	S3Endpoint  string
	S3Region    string
}

// PlatformLakehouseInput says who asked and how long the bucket must hold its data. The retention is required and
// has no default: compliance-mode retention cannot be shortened, so the number is always an explicit decision.
type PlatformLakehouseInput struct {
	RetentionDays int
	ActorID       string
}

// PlatformLakehouseSpec is what the workflow needs from the registry.
type PlatformLakehouseSpec struct {
	RetentionDays      int
	AlreadyProvisioned bool
}

// ConfigurePlatformLakehouse records the retention and decides whether there is anything to do.
//
//   - Already provisioned and the request is no higher than what the bucket carries: nothing to do.
//   - Already provisioned and the request is HIGHER: refused, nothing written. Raising an existing bucket's default
//     retention is a separate operation this does not perform, and recording a number the bucket does not enforce
//     would be a record that lies.
//   - Otherwise the retention is recorded; a lower number than one already recorded is refused.
func (a *PlatformLakehouseActivities) ConfigurePlatformLakehouse(ctx context.Context, in PlatformLakehouseInput) (PlatformLakehouseSpec, error) {
	if in.RetentionDays < registry.MinRetentionDays || in.RetentionDays > registry.MaxRetentionDays {
		return PlatformLakehouseSpec{}, nonRetryable(errTypeInvalidInput, registry.ErrInvalidRetention)
	}
	cfg, err := a.Registry.Get(ctx)
	switch {
	case errors.Is(err, registry.ErrPlatformNotConfigured):
		// first run: recorded below
	case err != nil:
		return PlatformLakehouseSpec{}, fmt.Errorf("read the platform lakehouse registry: %w", err)
	case cfg.Provisioned:
		applied := 0
		if cfg.RetentionAppliedDays != nil {
			applied = *cfg.RetentionAppliedDays
		}
		if in.RetentionDays > applied {
			return PlatformLakehouseSpec{}, nonRetryable(errTypeNotReady,
				fmt.Errorf("the platform bucket already enforces %d days; raising an existing bucket's retention to %d is not done by provisioning", applied, in.RetentionDays))
		}
		return PlatformLakehouseSpec{AlreadyProvisioned: true}, nil
	}
	if err := a.Registry.ConfigureRetention(ctx, in.RetentionDays); err != nil {
		if errors.Is(err, registry.ErrRetentionLowered) || errors.Is(err, registry.ErrInvalidRetention) {
			return PlatformLakehouseSpec{}, nonRetryable(errTypeInvalidInput, err)
		}
		return PlatformLakehouseSpec{}, fmt.Errorf("record the platform retention: %w", err)
	}
	// ConfigureRetention leaves the recorded value equal to the request (a lower request was refused above), so the
	// request is what the bucket will be created with.
	return PlatformLakehouseSpec{RetentionDays: in.RetentionDays}, nil
}

// EnsurePlatformKey makes sure the platform's KMS key exists and returns its id, which is the bucket's name.
func (a *PlatformLakehouseActivities) EnsurePlatformKey(ctx context.Context) (string, error) {
	if err := a.Keys.EnsureKey(ctx, iceberg.ControlWarehouseName); err != nil {
		return "", infraErr("ensure KMS key", err)
	}
	return iceberg.ControlWarehouseName, nil
}

// EnsurePlatformBucket creates or verifies the WORM bucket. A bucket that conflicts is not retried: it needs a person.
func (a *PlatformLakehouseActivities) EnsurePlatformBucket(ctx context.Context, kmsKeyID string, retentionDays int) error {
	if retentionDays < registry.MinRetentionDays {
		return nonRetryable(errTypeInvalidInput, registry.ErrInvalidRetention)
	}
	_, err := a.Buckets.EnsureControlBucket(ctx, iceberg.ControlBucketSpec{
		KMSKeyID: kmsKeyID, RetentionDays: uint(retentionDays), Region: a.S3Region,
	})
	if errors.Is(err, iceberg.ErrBucketConflict) {
		return nonRetryable(errTypeConflict, err)
	}
	if err != nil {
		return infraErr("ensure bucket", err)
	}
	return nil
}

// EnsurePlatformCredential makes sure the credential scoped to ivy-control exists. Whether one may be MINTED is
// decided by the registry row, not the secrets store, exactly as for a tenant: the store reports every failure,
// outages included, as "not found", so it cannot prove absence. Once the registry says one was issued, a missing
// credential is a non-retryable ErrCredentialLost that needs a person.
func (a *PlatformLakehouseActivities) EnsurePlatformCredential(ctx context.Context) error {
	cfg, err := a.Registry.Get(ctx)
	if err != nil {
		if errors.Is(err, registry.ErrPlatformNotConfigured) {
			return nonRetryable(errTypeNotReady, err)
		}
		return fmt.Errorf("read the platform lakehouse registry: %w", err)
	}
	err = a.Credentials.EnsurePlatformCredential(ctx, !cfg.CredentialIssued)
	if errors.Is(err, infra.ErrCredentialLost) {
		return nonRetryable(errTypeConflict, err)
	}
	if err != nil {
		return infraErr("ensure storage credential", err)
	}
	if !cfg.CredentialIssued {
		if err := a.Registry.MarkCredentialIssued(ctx); err != nil {
			return fmt.Errorf("record credential issued: %w", err)
		}
	}
	return nil
}

// EnsurePlatformWarehouse creates the Lakekeeper warehouse over the ivy-control bucket, reading the credential here
// and returning only the warehouse id.
func (a *PlatformLakehouseActivities) EnsurePlatformWarehouse(ctx context.Context) (string, error) {
	if a.S3Endpoint == "" {
		return "", nonRetryable(errTypeNotReady, fmt.Errorf("%w: the object store endpoint (S3_ENDPOINT) is not set", infra.ErrNotConfigured))
	}
	key, secret, err := a.Credentials.ReadPlatform(ctx)
	if err != nil {
		return "", infraErr("read storage credential", err)
	}
	wh, err := a.Warehouses.EnsureControlWarehouse(ctx, iceberg.ControlWarehouseSpec{
		Region: a.S3Region, Endpoint: a.S3Endpoint, AccessKeyID: key, SecretAccessKey: secret,
	})
	if err != nil {
		return "", infraErr("ensure warehouse", err)
	}
	return wh.ID, nil
}

// MarkPlatformProvisioned records the warehouse, the key and the retention the bucket was created with.
func (a *PlatformLakehouseActivities) MarkPlatformProvisioned(ctx context.Context, warehouseID, kmsKeyID string, appliedDays int) error {
	wh, err := uuid.Parse(warehouseID)
	if err != nil {
		return nonRetryable(errTypeInvalidInput, fmt.Errorf("warehouse id %q is not a UUID", warehouseID))
	}
	err = a.Registry.MarkProvisioned(ctx, wh, kmsKeyID, appliedDays)
	switch {
	case errors.Is(err, registry.ErrWarehouseMismatch):
		return nonRetryable(errTypeConflict, err)
	case errors.Is(err, registry.ErrPlatformNotConfigured):
		return nonRetryable(errTypeNotReady, err)
	case err != nil:
		return fmt.Errorf("record provisioning: %w", err)
	}
	return nil
}
