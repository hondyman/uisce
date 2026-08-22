package customer

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

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]CustomerRecord, error) {
	query := `
		SELECT id, tenant_id, customer_name, subtype_code, lei_code, kyc_status,
		       suitability_profile, relationship_tier, parent_group_id,
		       created_at, updated_at, valid_from, valid_to
		FROM master.customer
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}
	defer rows.Close()

	var records []CustomerRecord
	for rows.Next() {
		var rec CustomerRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.CustomerName, &rec.SubtypeCode,
			&rec.LEICode, &rec.KYCStatus, &rec.SuitabilityProfile,
			&rec.RelationshipTier, &rec.ParentGroupID,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*CustomerRecord, error) {
	var rec CustomerRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, customer_name, subtype_code, lei_code, kyc_status,
		       suitability_profile, relationship_tier, parent_group_id,
		       created_at, updated_at, valid_from, valid_to
		FROM master.customer
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.CustomerName, &rec.SubtypeCode,
		&rec.LEICode, &rec.KYCStatus, &rec.SuitabilityProfile,
		&rec.RelationshipTier, &rec.ParentGroupID,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *CustomerRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO master.customer (
			id, tenant_id, customer_name, subtype_code, lei_code, kyc_status,
			suitability_profile, relationship_tier, parent_group_id,
			created_at, updated_at, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.CustomerName, rec.SubtypeCode,
		rec.LEICode, rec.KYCStatus, rec.SuitabilityProfile,
		rec.RelationshipTier, rec.ParentGroupID,
	)
	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE master.customer SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete customer: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}