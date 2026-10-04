package activities

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/hondyman/uisce/backend/internal/lakehouse/infra"
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
	MarkProvisioned(ctx context.Context, tenantID, warehouseID uuid.UUID, kmsKeyID string, appliedDays int, actor registry.Actor) error
	// MarkRetentionApplied records that the bucket now enforces at least days by default.
	MarkRetentionApplied(ctx context.Context, tenantID uuid.UUID, days int, actor registry.Actor) error
	RecordRetentionSyncFailure(ctx context.Context, tenantID uuid.UUID, actor registry.Actor, step, reason string) error
	RecordProvisionFailure(ctx context.Context, tenantID uuid.UUID, actor registry.Actor, step, reason string) error
	// MarkCredentialIssued records, transactionally, that the storage credential now exists.
	MarkCredentialIssued(ctx context.Context, tenantID uuid.UUID) error

	// For the Iceberg audit copy (ADR-036).
	// AuditAfter returns entries after an id, oldest first (chain order).
	AuditAfter(ctx context.Context, tenantID uuid.UUID, afterID int64, limit int) ([]registry.AuditEntry, error)
	// MarkAuditCopied records, for operators only, how far the copy has got. It never decides what is shipped.
	MarkAuditCopied(ctx context.Context, tenantID uuid.UUID, throughID int64) error
	// ProvisionedTenants lists the tenants that have a warehouse to copy into.
	ProvisionedTenants(ctx context.Context) ([]uuid.UUID, error)
	// VerifyAudit recomputes alpha's own chain and returns the first id that does not verify, or nil.
	VerifyAudit(ctx context.Context, tenantID uuid.UUID) (*int64, error)
	// RecordAuditVerification replaces the recorded outcome of the last verification (ADR-044).
	RecordAuditVerification(ctx context.Context, tenantID uuid.UUID, v registry.AuditVerification) error
}

// LakehouseKeys manages the tenant's KMS key (KES).
type LakehouseKeys interface {
	// EnsureKey creates the named key if it does not exist and does nothing if it does.
	EnsureKey(ctx context.Context, name string) error
}

// LakehouseBuckets creates the tenant's WORM, encrypted bucket and raises its retention.
type LakehouseBuckets interface {
	EnsureTenantBucket(ctx context.Context, spec iceberg.TenantBucketSpec) (*iceberg.TenantBucket, error)
	// ExtendTenantRetention raises the default retention to at least days; it never lowers.
	ExtendTenantRetention(ctx context.Context, tenantID uuid.UUID, days uint) (uint, error)
}

// LakehouseCredentials issues and reads the tenant's bucket-scoped storage credential.
type LakehouseCredentials interface {
	// EnsureBucketCredential makes sure a credential limited to this tenant's bucket
	// exists in the secrets store. It is idempotent and returns nothing secret. mayMint is
	// false once the registry records that one was issued: then a credential that cannot be
	// read is infra.ErrCredentialLost, never a reason to issue a new one.
	EnsureBucketCredential(ctx context.Context, tenantID uuid.UUID, bucket string, mayMint bool) error
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
	// Destination is where the tenant's audit copy lives (StarRocks over the tenant's warehouse).
	Destination infra.AuditDestination
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

// infraErr wraps an error from an infrastructure adapter. Missing configuration cannot be
// fixed by retrying, so it fails fast with a readable reason; anything else is an outage
// worth retrying.
func infraErr(what string, err error) error {
	if errors.Is(err, infra.ErrNotConfigured) {
		return nonRetryable(errTypeNotReady, fmt.Errorf("%s: %w", what, err))
	}
	return fmt.Errorf("%s: %w", what, err)
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
		return "", infraErr("ensure KMS key", err)
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
		return infraErr("ensure bucket", err)
	}
	return nil
}

// EnsureLakehouseCredential makes sure the tenant's bucket-scoped credential exists in the
// secrets store. Nothing secret is returned.
//
// Whether a credential may be MINTED is decided by the registry, not by the secrets store:
// the store reports every failure, including an outage, as "not found", so it cannot prove a
// credential is absent. Once the registry says one was issued, "not found" is a non-retryable
// ErrCredentialLost that needs a person, because minting a new credential would strand the
// tenant's warehouse on the old, now-stale keys.
func (a *TenantLakehouseActivities) EnsureLakehouseCredential(ctx context.Context, in LakehouseProvisionInput) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	bucket, err := iceberg.TenantWarehouseName(id)
	if err != nil {
		return nonRetryable(errTypeInvalidInput, err)
	}
	cfg, err := a.Registry.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("read lakehouse registry: %w", err)
	}

	err = a.Credentials.EnsureBucketCredential(ctx, id, bucket, !cfg.CredentialIssued)
	if errors.Is(err, infra.ErrCredentialLost) {
		return nonRetryable(errTypeConflict, err)
	}
	if err != nil {
		return infraErr("ensure storage credential", err)
	}
	if !cfg.CredentialIssued {
		if err := a.Registry.MarkCredentialIssued(ctx, id); err != nil {
			return fmt.Errorf("record credential issued: %w", err)
		}
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
	if a.S3Endpoint == "" {
		return "", nonRetryable(errTypeNotReady, fmt.Errorf("%w: the object store endpoint (S3_ENDPOINT) is not set", infra.ErrNotConfigured))
	}
	key, secret, err := a.Credentials.Read(ctx, id)
	if err != nil {
		return "", infraErr("read storage credential", err)
	}
	wh, err := a.Warehouses.EnsureTenantWarehouse(ctx, iceberg.TenantWarehouseSpec{
		TenantID:        id,
		Region:          a.S3Region,
		Endpoint:        a.S3Endpoint,
		AccessKeyID:     key,
		SecretAccessKey: secret,
	})
	if err != nil {
		return "", infraErr("ensure warehouse", err)
	}
	return wh.ID, nil
}

// MarkLakehouseProvisioned records the warehouse, the key and the retention the bucket was
// created with in the registry, and audits it.
func (a *TenantLakehouseActivities) MarkLakehouseProvisioned(ctx context.Context, in LakehouseProvisionInput, warehouseID, kmsKeyID string, appliedDays int) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	wh, err := uuid.Parse(warehouseID)
	if err != nil {
		return nonRetryable(errTypeInvalidInput, fmt.Errorf("warehouse id %q is not a UUID", warehouseID))
	}
	err = a.Registry.MarkProvisioned(ctx, id, wh, kmsKeyID, appliedDays, in.actor())
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

// RetentionTarget is what a retention reconcile needs from the registry.
type RetentionTarget struct {
	DesiredDays int
	// Pending is false when the bucket already enforces what the registry wants.
	Pending bool
}

// LoadRetentionTarget reads what the registry wants and whether the bucket already enforces
// it. Only a provisioned tenant has a bucket to reconcile.
func (a *TenantLakehouseActivities) LoadRetentionTarget(ctx context.Context, in LakehouseProvisionInput) (RetentionTarget, error) {
	id, err := in.tenant()
	if err != nil {
		return RetentionTarget{}, err
	}
	cfg, err := a.Registry.Get(ctx, id)
	if errors.Is(err, registry.ErrTenantNotFound) {
		return RetentionTarget{}, nonRetryable(errTypeInvalidInput, err)
	}
	if err != nil {
		return RetentionTarget{}, fmt.Errorf("read lakehouse registry: %w", err)
	}
	if !cfg.Provisioned {
		return RetentionTarget{}, nonRetryable(errTypeNotReady,
			fmt.Errorf("%w: nothing is provisioned to apply retention to", registry.ErrInvalidState))
	}
	if cfg.AuditRetentionDays == nil {
		return RetentionTarget{}, nonRetryable(errTypeNotReady, registry.ErrNotConfigured)
	}
	return RetentionTarget{DesiredDays: *cfg.AuditRetentionDays, Pending: cfg.RetentionPending}, nil
}

// ExtendBucketRetention raises the tenant bucket's default retention to at least days. It never
// lowers; a bucket that is not compliance-locked is a conflict that needs a person.
func (a *TenantLakehouseActivities) ExtendBucketRetention(ctx context.Context, in LakehouseProvisionInput, days int) (int, error) {
	id, err := in.tenant()
	if err != nil {
		return 0, err
	}
	if days < registry.MinRetentionDays {
		return 0, nonRetryable(errTypeInvalidInput, registry.ErrInvalidRetention)
	}
	got, err := a.Buckets.ExtendTenantRetention(ctx, id, uint(days))
	if errors.Is(err, iceberg.ErrBucketConflict) {
		return 0, nonRetryable(errTypeConflict, err)
	}
	if err != nil {
		return 0, infraErr("extend bucket retention", err)
	}
	return int(got), nil
}

// MarkRetentionApplied records in the registry, with an audit entry, that the bucket enforces
// days. The registry only raises it and never lets it exceed what is wanted.
func (a *TenantLakehouseActivities) MarkRetentionApplied(ctx context.Context, in LakehouseProvisionInput, days int) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	err = a.Registry.MarkRetentionApplied(ctx, id, days, in.actor())
	switch {
	case errors.Is(err, registry.ErrNotConfigured), errors.Is(err, registry.ErrInvalidState), errors.Is(err, registry.ErrInvalidRetention):
		return nonRetryable(errTypeNotReady, err)
	case err != nil:
		return fmt.Errorf("record applied retention: %w", err)
	}
	return nil
}

// RecordRetentionSyncFailure writes the reason a reconcile stopped into the audit trail.
func (a *TenantLakehouseActivities) RecordRetentionSyncFailure(ctx context.Context, in LakehouseProvisionInput, step, reason string) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	return a.Registry.RecordRetentionSyncFailure(ctx, id, in.actor(), step, reason)
}

// RecordLakehouseFailure writes the reason a run stopped into the audit trail.
func (a *TenantLakehouseActivities) RecordLakehouseFailure(ctx context.Context, in LakehouseProvisionInput, step, reason string) error {
	id, err := in.tenant()
	if err != nil {
		return err
	}
	return a.Registry.RecordProvisionFailure(ctx, id, in.actor(), step, reason)
}
