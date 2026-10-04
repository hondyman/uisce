package iceberg

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ADR-032: every tenant has exactly one Iceberg warehouse, over its own bucket.
// The database enforces the one-to-one rule (public.tenant_lakehouse, with
// CHECKs deriving both names from tenant_id); this file is the only place that
// derives the name in Go, and it must stay identical to those CHECKs.

const tenantWarehousePrefix = "ivy-t-"

// warehouseKeyPrefix is the key prefix inside the tenant's own bucket.
const warehouseKeyPrefix = "warehouse"

// TenantWarehouseName is the warehouse name and the bucket name for a tenant:
// "ivy-t-" plus the tenant UUID without hyphens (38 chars, inside the 63-char
// bucket limit). It must match the CHECK constraints in
// 20261206_001_tenant_datasource_binding.up.sql.
func TenantWarehouseName(tenantID uuid.UUID) (string, error) {
	if tenantID == uuid.Nil {
		return "", errors.New("tenant warehouse: tenant id is required")
	}
	return tenantWarehousePrefix + strings.ReplaceAll(tenantID.String(), "-", ""), nil
}

// TenantWarehouseSpec describes the one warehouse of one tenant. The credential
// is the tenant's own storage credential, scoped to its own bucket, supplied by
// the caller from the secrets store. It is never read from the provisioner's
// shared configuration and never logged.
type TenantWarehouseSpec struct {
	TenantID        uuid.UUID
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

// TenantWarehouse is the result of EnsureTenantWarehouse. It carries no secret.
type TenantWarehouse struct {
	ID      string // Lakekeeper's warehouse id
	Name    string
	Bucket  string
	Created bool // false when the warehouse already existed
}

// EnsureTenantWarehouse returns the tenant's warehouse, creating it if absent.
// It is idempotent, and safe against a concurrent creator: a create that fails
// because another caller won the race is resolved by looking the warehouse up
// again.
//
// The warehouse always points at the tenant's own bucket. The provisioner's
// shared s3Bucket is deliberately not consulted, so a misconfigured default can
// never place one tenant's warehouse in another's storage.
//
// The bucket must already exist; creating it, with Object Lock, is a separate
// provisioning step.
func (p *LakekeeperProvisioner) EnsureTenantWarehouse(ctx context.Context, spec TenantWarehouseSpec) (*TenantWarehouse, error) {
	name, err := TenantWarehouseName(spec.TenantID)
	if err != nil {
		return nil, err
	}
	return p.ensureWarehouse(ctx, "tenant warehouse", name, spec.Region, spec.Endpoint, spec.AccessKeyID, spec.SecretAccessKey)
}

// ensureWarehouse is the one implementation of "a warehouse over its own bucket", shared by tenant
// warehouses and the platform's ivy-control warehouse. The name is derived by the caller from a fixed
// rule, and the bucket is always the warehouse's own name.
func (p *LakekeeperProvisioner) ensureWarehouse(ctx context.Context, what, name, region, endpoint, accessKeyID, secretAccessKey string) (*TenantWarehouse, error) {
	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("%s: a storage credential scoped to its own bucket is required", what)
	}
	if endpoint == "" {
		return nil, fmt.Errorf("%s: storage endpoint is required", what)
	}
	if region == "" {
		region = "us-east-1"
	}

	id, _, err := p.GetWarehouseByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%s: look up %s: %w", what, name, err)
	}
	if id != "" {
		return &TenantWarehouse{ID: id, Name: name, Bucket: name}, nil
	}

	payload := map[string]interface{}{
		"warehouse-name": name,
		"storage-profile": map[string]interface{}{
			"type":              "s3",
			"bucket":            name,
			"key-prefix":        warehouseKeyPrefix,
			"region":            region,
			"sts-enabled":       false,
			"endpoint":          endpoint,
			"path-style-access": true,
		},
		"storage-credential": map[string]interface{}{
			"type":              "s3",
			"credential-type":   "access-key",
			"access-key-id":     accessKeyID,
			"secret-access-key": secretAccessKey,
		},
	}

	createErr := p.CreateWarehouse(ctx, payload)

	// Look up regardless of createErr: if we lost a race, the warehouse exists
	// and the create error is expected. The error is wrapped below only if the
	// warehouse is still absent, so it can never leak into a success.
	id, status, err := p.GetWarehouseByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%s: look up %s after create: %w", what, name, err)
	}
	if id == "" {
		if createErr != nil {
			return nil, fmt.Errorf("%s: create %s: %w", what, name, createErr)
		}
		return nil, fmt.Errorf("%s: %s not found after create (status %d)", what, name, status)
	}
	return &TenantWarehouse{ID: id, Name: name, Bucket: name, Created: createErr == nil}, nil
}
