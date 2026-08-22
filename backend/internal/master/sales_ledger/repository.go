package sales_ledger

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

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]SalesLedgerRecord, error) {
	query := `
		SELECT id, tenant_id, invoice_number, client_id, subtype_code,
		       billing_period_end, aum_basis_amount, effective_fee_bps, hwm_benchmark_nav, invoice_status,
		       created_at, updated_at, valid_from, valid_to
		FROM master.sales_ledger
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sales_ledger: %w", err)
	}
	defer rows.Close()

	var records []SalesLedgerRecord
	for rows.Next() {
		var rec SalesLedgerRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.InvoiceNumber, &rec.ClientID, &rec.SubtypeCode,
			&rec.BillingPeriodEnd, &rec.AUMBasisAmount, &rec.EffectiveFeeBPS,
			&rec.HWMBenchmarkNAV, &rec.InvoiceStatus,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sales_ledger: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*SalesLedgerRecord, error) {
	var rec SalesLedgerRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, invoice_number, client_id, subtype_code,
		       billing_period_end, aum_basis_amount, effective_fee_bps, hwm_benchmark_nav, invoice_status,
		       created_at, updated_at, valid_from, valid_to
		FROM master.sales_ledger
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.InvoiceNumber, &rec.ClientID, &rec.SubtypeCode,
		&rec.BillingPeriodEnd, &rec.AUMBasisAmount, &rec.EffectiveFeeBPS,
		&rec.HWMBenchmarkNAV, &rec.InvoiceStatus,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sales_ledger: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *SalesLedgerRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO master.sales_ledger (
			id, tenant_id, invoice_number, client_id, subtype_code,
			billing_period_end, aum_basis_amount, effective_fee_bps, hwm_benchmark_nav, invoice_status,
			created_at, updated_at, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.InvoiceNumber, rec.ClientID, rec.SubtypeCode,
		rec.BillingPeriodEnd, rec.AUMBasisAmount, rec.EffectiveFeeBPS,
		rec.HWMBenchmarkNAV, rec.InvoiceStatus,
	)
	if err != nil {
		return fmt.Errorf("failed to create sales_ledger: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE master.sales_ledger SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete sales_ledger: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
