package personnel

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

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]PersonnelRecord, error) {
	query := `
		SELECT id, tenant_id, full_name, email, subtype_code,
		       crd_number, series_licenses_held, supervisory_id, discretionary_authority_limit,
		       created_at, updated_at, valid_from, valid_to
		FROM master.personnel
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list personnel: %w", err)
	}
	defer rows.Close()

	var records []PersonnelRecord
	for rows.Next() {
		var rec PersonnelRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.FullName, &rec.Email, &rec.SubtypeCode,
			&rec.CRDNumber, &rec.SeriesLicensesHeld, &rec.SupervisoryID, &rec.DiscretionaryAuthorityLimit,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan personnel: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*PersonnelRecord, error) {
	var rec PersonnelRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, full_name, email, subtype_code,
		       crd_number, series_licenses_held, supervisory_id, discretionary_authority_limit,
		       created_at, updated_at, valid_from, valid_to
		FROM master.personnel
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.FullName, &rec.Email, &rec.SubtypeCode,
		&rec.CRDNumber, &rec.SeriesLicensesHeld, &rec.SupervisoryID, &rec.DiscretionaryAuthorityLimit,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get personnel: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *PersonnelRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO master.personnel (
			id, tenant_id, full_name, email, subtype_code,
			crd_number, series_licenses_held, supervisory_id, discretionary_authority_limit,
			created_at, updated_at, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.FullName, rec.Email, rec.SubtypeCode,
		rec.CRDNumber, rec.SeriesLicensesHeld, rec.SupervisoryID, rec.DiscretionaryAuthorityLimit,
	)
	if err != nil {
		return fmt.Errorf("failed to create personnel: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE master.personnel SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete personnel: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
