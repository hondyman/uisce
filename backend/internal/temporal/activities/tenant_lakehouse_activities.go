package activities

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"go.temporal.io/sdk/temporal"
)

// Activities for provisioning a tenant's one Iceberg warehouse (ADR-032): KMS key,
// WORM bucket, bucket-scoped credential, Lakekeeper warehouse, then the registry row.
//
// Rules these activities keep:
//   - No credential ever appears in an argument or a result. Temporal stores both in
//     workflow history, in the clear. An activity that needs the credential reads it
//     from the secrets store itself.
//   - Every step is idempotent, so a retry or a re-run completes what is missing.
//   - There is no compensation. The bucket holds WORM audit and is never deleted by a
//     failed run; the next run picks up where this one stopped (forward recovery).
//   - A condition a retry cannot fix (retention not set, a bucket that conflicts with
//     the required settings) is non-retryable, so it fails fast and reaches a human.
//
// These are deliberately NOT registered as BP-Designer "safe" activities: a designed
// business process must never be able to call them.

// LakehouseRegistry is the part of registry.Store these activities use.
type LakehouseRegistry interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*registry.Config, error)
	MarkProvisioned(ctx context.Context, tenantID, warehouseID uuid.UUID, kmsKeyID string, actor registry.Actor) error
	RecordProvisionFailure(ctx context.Context, tenantID uuid.UUID, actor registry.Actor, step, reason string) error
}

// LakehouseKeys manages the tenant's KMS key (KES).
type LakehouseKeys interface {
	// EnsureKey creates the named key if it does not exist and does nothing if it does.
	EnsureKey(ctx context.Context, name string) error
}

// LakehouseBuckets creates the tenant's WORM, encrypted bucket.
type LakehouseBuckets interface {
	EnsureTenantBucket(ctx context.Context, spec iceberg.TenantBucketSpec) (*iceberg.TenantBucket, error)
}

// LakehouseCredentials issues and reads the tenant's bucket-scoped storage credential.
type LakehouseCredentials interface {
	// EnsureBucketCredential makes sure a credential limited to this tenant's bucket
	// exists in the secrets store. It is idempotent and returns nothing secret.
	EnsureBucketCredential(ctx context.Context, tenantID uuid.UUID, bucket string) error
	// Read returns the stored credential. Only an activity that must hand it to a
	// storage client may call it; it must never be returned from an activity.
	Read(ctx context.Context, tenantID uuid.UUID) (accessKeyID, secretAccessKey string, err error)
}

// LakehouseWarehouses creates the tenant's Lakekeeper warehouse.
type LakehouseWarehouses interface {
	EnsureTenantWarehouse(ctx context.Context, spec iceberg.TenantWarehouseSpec) (*iceberg.TenantWarehouse, error)
}

// TenantLakehouseActivities holds the dependencies. Build it once per worker.
type TenantLakehouseActivities struct {
	Registry    LakehouseRegistry
	Keys        LakehouseKeys
	Buckets     LakehouseBuckets
	Credentials LakehouseCredentials
	Warehouses  LakehouseWarehouses
	// S3Endpoint and S3Region locate the object store for the Lakekeeper warehouse.
	S3Endpoint string
	S3Region   string
}

// LakehouseProvisionInput identifies the tenant and who asked. It carries no secret.
type LakehouseProvisionInput struct {
	TenantID  string
	ActorID   string
	ActorRole string
}

// LakehouseSpec is what the workflow needs from the registry.
type LakehouseSpec struct {
	RetentionDays      int
	AlreadyProvisioned bool
}

// Error types, so a workflow or an operator can tell the cases apart.
const (
	errTypeInvalidInput = "LakehouseInvalidInput"
	errTypeNotReady     = "LakehouseNotReady"
	errTypeConflict     = "LakehouseConflict"
)

func nonRetryable(kind string, err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), kind, err)
}

func (in LakehouseProvisionInput) tenant() (uuid.UUID, error) {
	id, err := uuid.Parse(in.TenantID)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, nonRetryable(errTypeInvalidInput, fmt.Errorf("invalid tenant id %q", in.TenantID))
	}
	return id, nil
}

func (in LakehouseProvisionInput) actor() registry.Actor {
	return registry.Actor{ID: in.ActorID, Role: in.ActorRole}
}

// LoadLakehouseSpec reads the tenant's registry entry and refuses to go on unless the
// tenant is configured with an audit retention and is still waiting to be provisioned.
func (a *TenantLakehouseActivities) LoadLakehouseSpec(ctx context.Context, in LakehouseProvisionInput) (LakehouseSpec, error) {
	id, err := in.tenant()
	if err != nil {
		return LakehouseSpec{}, err
	}
	cfg, err := a.Registry.Get(ctx, id)
	if errors.Is(err, registry.ErrTenantNotFound) {
		return LakehouseSpec{}, nonRetryable(errTypeInvalidInput, err)
	}
	if err != nil {
		return LakehouseSpec{}, fmt.Errorf("read lakehouse registry: %w", err)
	}
	if cfg.Provisioned {
		return LakehouseSpec{AlreadyProvisioned: true}, nil
	}
	if !cfg.Configured || cfg.AuditRetentionDays == nil {
		return LakehouseSpec{}, nonRetryable(errTypeNotReady, registry.ErrNotConfigured)
	}
	if cfg.LifecycleState != "provisioning" {
		return LakehouseSpec{}, nonRetryable(errTypeNotReady,
			fmt.Errorf("%w: state is %q", registry.ErrInvalidState, cfg.LifecycleState))
	}
	return LakehouseSpec{RetentionDays: *cfg.AuditRetentionDays}, nil
}

// EnsureLakehouseKey makes sure the tenant's KMS key exists and returns its id. The key
// is named for the tenant's warehouse, so the name is derived and never supplied.
func (a *TenantLakehouseActivities) EnsureLakehouseKey(ctx context.Context, in LakehouseProvisionInput) (string, error) {
	id, err := in.tenant()
	if err != nil {
		return "", err
	}
	name, err := iceberg.TenantWarehouseName(id)
	if err != nil {
		return "", nonRetryable(errTypeInvalidInput, err)
	}
	if err := a.Keys.EnsureKey(ctx, name); err != nil {
		return "", fmt.Errorf("ensure KMS key: %w", err)
	}
	return name, nil
}

// EnsureLakehouseBucket creates or verifies the tenant's WORM bucket. A bucket that
// conflicts with the required settings is not retried: it needs a person.
func (a *TenantLakehouseActivities) EnsureLakehouseBucket(ctx context.Context, in LakehouseProvisionInput, kmsKeyID string, retentionDays int) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	if retentionDays < registry.MinRetentionDays {
		return nonRetryable(errTypeInvalidInput, registry.ErrInvalidRetention)
	}
	_, err = a.Buckets.EnsureTenantBucket(ctx, iceberg.TenantBucketSpec{
		TenantID:      id,
		KMSKeyID:      kmsKeyID,
		RetentionDays: uint(retentionDays),
		Region:        a.S3Region,
	})
	if errors.Is(err, iceberg.ErrBucketConflict) {
		return nonRetryable(errTypeConflict, err)
	}
	if err != nil {
		return fmt.Errorf("ensure bucket: %w", err)
	}
	return nil
}

// EnsureLakehouseCredential makes sure the tenant's bucket-scoped credential exists in
// the secrets store. Nothing secret is returned.
func (a *TenantLakehouseActivities) EnsureLakehouseCredential(ctx context.Context, in LakehouseProvisionInput) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	bucket, err := iceberg.TenantWarehouseName(id)
	if err != nil {
		return nonRetryable(errTypeInvalidInput, err)
	}
	if err := a.Credentials.EnsureBucketCredential(ctx, id, bucket); err != nil {
		return fmt.Errorf("ensure storage credential: %w", err)
	}
	return nil
}

// EnsureLakehouseWarehouse creates the tenant's Lakekeeper warehouse over its own bucket,
// reading the credential from the secrets store here and returning only the warehouse id.
func (a *TenantLakehouseActivities) EnsureLakehouseWarehouse(ctx context.Context, in LakehouseProvisionInput) (string, error) {
	id, err := in.tenant()
	if err != nil {
		return "", err
	}
	key, secret, err := a.Credentials.Read(ctx, id)
	if err != nil {
		return "", fmt.Errorf("read storage credential: %w", err)
	}
	wh, err := a.Warehouses.EnsureTenantWarehouse(ctx, iceberg.TenantWarehouseSpec{
		TenantID:        id,
		Region:          a.S3Region,
		Endpoint:        a.S3Endpoint,
		AccessKeyID:     key,
		SecretAccessKey: secret,
	})
	if err != nil {
		return "", fmt.Errorf("ensure warehouse: %w", err)
	}
	return wh.ID, nil
}

// MarkLakehouseProvisioned records the warehouse and key in the registry and audits it.
func (a *TenantLakehouseActivities) MarkLakehouseProvisioned(ctx context.Context, in LakehouseProvisionInput, warehouseID, kmsKeyID string) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	wh, err := uuid.Parse(warehouseID)
	if err != nil {
		return nonRetryable(errTypeInvalidInput, fmt.Errorf("warehouse id %q is not a UUID", warehouseID))
	}
	err = a.Registry.MarkProvisioned(ctx, id, wh, kmsKeyID, in.actor())
	switch {
	case errors.Is(err, registry.ErrWarehouseMismatch):
		return nonRetryable(errTypeConflict, err)
	case errors.Is(err, registry.ErrNotConfigured), errors.Is(err, registry.ErrInvalidState):
		return nonRetryable(errTypeNotReady, err)
	case err != nil:
		return fmt.Errorf("record provisioning: %w", err)
	}
	return nil
}

// RecordLakehouseFailure writes the reason a run stopped into the audit trail.
func (a *TenantLakehouseActivities) RecordLakehouseFailure(ctx context.Context, in LakehouseProvisionInput, step, reason string) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	return a.Registry.RecordProvisionFailure(ctx, id, in.actor(), step, reason)
}
