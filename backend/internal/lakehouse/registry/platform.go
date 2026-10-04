package registry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// The platform warehouse ivy-control (ADR-045) has exactly one registry row, in public.platform_lakehouse. It is
// not a tenant, so it has no tenant transaction and no row-level security.

// PlatformConfig is the platform warehouse's provisioning state. It carries no secret.
type PlatformConfig struct {
	AuditRetentionDays   int
	RetentionAppliedDays *int
	KMSKeyID             *string
	WarehouseID          *uuid.UUID
	CredentialIssued     bool
	Provisioned          bool
	ProvisionedAt        *time.Time
}

// ErrPlatformNotConfigured: no retention has been chosen for the platform warehouse yet.
var ErrPlatformNotConfigured = errors.New("the platform warehouse has no retention configured")

// PlatformStore reads and writes the platform warehouse's row.
type PlatformStore struct{ db *sql.DB }

func NewPlatformStore(db *sql.DB) *PlatformStore { return &PlatformStore{db: db} }

// Get returns the row, or ErrPlatformNotConfigured.
func (s *PlatformStore) Get(ctx context.Context) (*PlatformConfig, error) {
	var c PlatformConfig
	var applied sql.NullInt32
	var kms sql.NullString
	var wh uuid.NullUUID
	var issued, prov sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT audit_retention_days, retention_applied_days, kms_key_id, lakekeeper_warehouse_id, credential_issued_at, provisioned_at
		  FROM public.platform_lakehouse WHERE name = 'ivy-control'`).
		Scan(&c.AuditRetentionDays, &applied, &kms, &wh, &issued, &prov)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPlatformNotConfigured
	}
	if err != nil {
		return nil, fmt.Errorf("read platform lakehouse: %w", err)
	}
	if applied.Valid {
		v := int(applied.Int32)
		c.RetentionAppliedDays = &v
	}
	if kms.Valid {
		c.KMSKeyID = &kms.String
	}
	if wh.Valid {
		c.WarehouseID = &wh.UUID
	}
	c.CredentialIssued = issued.Valid
	if prov.Valid {
		c.ProvisionedAt = &prov.Time
		c.Provisioned = true
	}
	return &c, nil
}

// ConfigureRetention records the retention the platform bucket is to carry. The first call creates the row. A
// later call may only keep or raise it: a lower value is ErrRetentionLowered and nothing is written, because
// compliance-mode retention cannot be shortened and a silently ignored lower number would hide a mistake.
func (s *PlatformStore) ConfigureRetention(ctx context.Context, days int) error {
	if days < MinRetentionDays || days > MaxRetentionDays {
		return ErrInvalidRetention
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("configure platform retention: %w", err)
	}
	defer tx.Rollback()
	var have sql.NullInt32
	err = tx.QueryRowContext(ctx, `SELECT audit_retention_days FROM public.platform_lakehouse WHERE name = 'ivy-control' FOR UPDATE`).Scan(&have)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err = tx.ExecContext(ctx, `INSERT INTO public.platform_lakehouse (name, bucket, audit_retention_days) VALUES ('ivy-control', 'ivy-control', $1)`, days); err != nil {
			return fmt.Errorf("configure platform retention: %w", err)
		}
	case err != nil:
		return fmt.Errorf("configure platform retention: %w", err)
	case int(have.Int32) > days:
		return ErrRetentionLowered
	case int(have.Int32) < days:
		if _, err = tx.ExecContext(ctx, `UPDATE public.platform_lakehouse SET audit_retention_days = $1, version = version + 1 WHERE name = 'ivy-control'`, days); err != nil {
			return fmt.Errorf("configure platform retention: %w", err)
		}
	}
	return tx.Commit()
}

// MarkCredentialIssued records, once, that the platform storage credential now exists.
func (s *PlatformStore) MarkCredentialIssued(ctx context.Context) error {
	res, err := s.db.ExecContext(ctx, `UPDATE public.platform_lakehouse SET credential_issued_at = COALESCE(credential_issued_at, now()) WHERE name = 'ivy-control'`)
	if err != nil {
		return fmt.Errorf("record platform credential issued: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrPlatformNotConfigured
	}
	return nil
}

// MarkProvisioned records the warehouse, the key and the retention the bucket was created with. A different
// warehouse id than the one already recorded is ErrWarehouseMismatch.
func (s *PlatformStore) MarkProvisioned(ctx context.Context, warehouseID uuid.UUID, kmsKeyID string, appliedDays int) error {
	if warehouseID == uuid.Nil || kmsKeyID == "" || appliedDays < MinRetentionDays {
		return errors.New("a warehouse id, a KMS key id and the retention the bucket was created with are required")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE public.platform_lakehouse
		   SET lakekeeper_warehouse_id = $1, kms_key_id = $2, retention_applied_days = GREATEST(COALESCE(retention_applied_days, 0), $3),
		       provisioned_at = COALESCE(provisioned_at, now()), version = version + 1
		 WHERE name = 'ivy-control' AND (lakekeeper_warehouse_id IS NULL OR lakekeeper_warehouse_id = $1)`,
		warehouseID, kmsKeyID, appliedDays)
	if err != nil {
		return fmt.Errorf("record platform provisioning: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, gerr := s.Get(ctx); errors.Is(gerr, ErrPlatformNotConfigured) {
			return ErrPlatformNotConfigured
		}
		return ErrWarehouseMismatch
	}
	return nil
}
