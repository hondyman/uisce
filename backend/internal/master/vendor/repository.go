package vendor

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]VendorRecord, error) {
	query := `
		SELECT id, tenant_id, vendor_name, subtype_code, vendor_category, sla_tier,
		       soc2_certification_date, soc1_type2_on_file, billing_cycle,
		       created_at, updated_at, valid_from, valid_to
		FROM master.vendor
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list vendors: %w", err)
	}
	defer rows.Close()

	var records []VendorRecord
	for rows.Next() {
		var rec VendorRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.VendorName, &rec.SubtypeCode,
			&rec.VendorCategory, &rec.SLATier, &rec.SOC2CertDate,
			&rec.SOC1Type2OnFile, &rec.BillingCycle,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan vendor: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*VendorRecord, error) {
	var rec VendorRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, vendor_name, subtype_code, vendor_category, sla_tier,
		       soc2_certification_date, soc1_type2_on_file, billing_cycle,
		       created_at, updated_at, valid_from, valid_to
		FROM master.vendor
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.VendorName, &rec.SubtypeCode,
		&rec.VendorCategory, &rec.SLATier, &rec.SOC2CertDate,
		&rec.SOC1Type2OnFile, &rec.BillingCycle,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vendor: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *VendorRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO master.vendor (
			id, tenant_id, vendor_name, subtype_code, vendor_category, sla_tier,
			soc2_certification_date, soc1_type2_on_file, billing_cycle,
			created_at, updated_at, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.VendorName, rec.SubtypeCode,
		rec.VendorCategory, rec.SLATier, rec.SOC2CertDate,
		rec.SOC1Type2OnFile, rec.BillingCycle,
	)
	if err != nil {
		return fmt.Errorf("failed to create vendor: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE master.vendor SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete vendor: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}