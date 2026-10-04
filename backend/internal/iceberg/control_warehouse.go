package iceberg

import (
	"context"
	"errors"
	"strings"
)

// ADR-032: ivy-control is the platform's own warehouse for cross-tenant audit and metadata history. It
// belongs to no tenant. It is created exactly like a tenant's (its own bucket, Object Lock in compliance
// mode, default SSE-KMS under its own key, a storage credential scoped to that one bucket) by the same
// two implementations, so the platform's storage can never be weaker than a tenant's.

// ControlWarehouseName is the warehouse name and the bucket name of the platform warehouse. It can never
// collide with a tenant's, whose names always start with tenantWarehousePrefix.
const ControlWarehouseName = "ivy-control"

// ControlBucketSpec describes the platform bucket. The KMS key and the retention are required and have no
// default: a platform audit bucket is never created unencrypted, and the retention is a regulatory
// decision. Compliance-mode retention cannot be shortened once objects are locked.
type ControlBucketSpec struct {
	KMSKeyID      string
	RetentionDays uint
	Region        string
}

// EnsureControlBucket creates the ivy-control bucket if absent and verifies it is WORM and encrypted under
// the given key. Safe to re-run; it never changes an existing bucket's lock or encryption and refuses
// (ErrBucketConflict) to reconcile anything that differs.
func (p *TenantBucketProvisioner) EnsureControlBucket(ctx context.Context, spec ControlBucketSpec) (*TenantBucket, error) {
	return p.ensureBucket(ctx, "control bucket", ControlWarehouseName, spec.KMSKeyID, spec.RetentionDays, spec.Region)
}

// ControlWarehouseSpec describes the platform warehouse. The credential is scoped to the ivy-control
// bucket only, supplied by the caller from the secrets store, never read from the provisioner's shared
// configuration and never logged.
type ControlWarehouseSpec struct {
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

// EnsureControlWarehouse returns the platform warehouse, creating it if absent. Idempotent and safe
// against a concurrent creator. The bucket must already exist.
func (p *LakekeeperProvisioner) EnsureControlWarehouse(ctx context.Context, spec ControlWarehouseSpec) (*TenantWarehouse, error) {
	if strings.HasPrefix(ControlWarehouseName, tenantWarehousePrefix) {
		return nil, errors.New("control warehouse: its name must never fall in the tenant namespace")
	}
	return p.ensureWarehouse(ctx, "control warehouse", ControlWarehouseName, spec.Region, spec.Endpoint, spec.AccessKeyID, spec.SecretAccessKey)
}
