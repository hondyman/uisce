package settlement

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

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]SettlementRecord, error) {
	query := `
		SELECT id, tenant_id, account_id, amount, currency, settlement_date, settlement_status, subtype_code,
		       ex_date, record_date, drip_reinvest_flag, tax_withholding_amount,
		       coupon_period_start, accrued_interest, payment_frequency,
		       call_notice_id, due_date, management_fee_portion, investment_portion,
		       return_of_capital, preferred_return, carried_interest_retained,
		       action_type_code, cash_in_lieu_amount, mandatory_flag,
		       fee_category, invoice_reference_id, vat_amount,
		       created_at, updated_at, valid_from, valid_to
		FROM cash_flow.settlement
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list settlements: %w", err)
	}
	defer rows.Close()

	var records []SettlementRecord
	for rows.Next() {
		var rec SettlementRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.AccountID, &rec.Amount, &rec.Currency,
			&rec.SettlementDate, &rec.SettlementStatus, &rec.SubtypeCode,
			&rec.ExDate, &rec.RecordDate, &rec.DripReinvestFlag, &rec.TaxWithholdingAmount,
			&rec.CouponPeriodStart, &rec.AccruedInterest, &rec.PaymentFrequency,
			&rec.CallNoticeID, &rec.DueDate, &rec.ManagementFeePortion, &rec.InvestmentPortion,
			&rec.ReturnOfCapital, &rec.PreferredReturn, &rec.CarriedInterestRetained,
			&rec.ActionTypeCode, &rec.CashInLieuAmount, &rec.MandatoryFlag,
			&rec.FeeCategory, &rec.InvoiceReferenceID, &rec.VATAmount,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan settlement: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*SettlementRecord, error) {
	var rec SettlementRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, account_id, amount, currency, settlement_date, settlement_status, subtype_code,
		       ex_date, record_date, drip_reinvest_flag, tax_withholding_amount,
		       coupon_period_start, accrued_interest, payment_frequency,
		       call_notice_id, due_date, management_fee_portion, investment_portion,
		       return_of_capital, preferred_return, carried_interest_retained,
		       action_type_code, cash_in_lieu_amount, mandatory_flag,
		       fee_category, invoice_reference_id, vat_amount,
		       created_at, updated_at, valid_from, valid_to
		FROM cash_flow.settlement
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.AccountID, &rec.Amount, &rec.Currency,
		&rec.SettlementDate, &rec.SettlementStatus, &rec.SubtypeCode,
		&rec.ExDate, &rec.RecordDate, &rec.DripReinvestFlag, &rec.TaxWithholdingAmount,
		&rec.CouponPeriodStart, &rec.AccruedInterest, &rec.PaymentFrequency,
		&rec.CallNoticeID, &rec.DueDate, &rec.ManagementFeePortion, &rec.InvestmentPortion,
		&rec.ReturnOfCapital, &rec.PreferredReturn, &rec.CarriedInterestRetained,
		&rec.ActionTypeCode, &rec.CashInLieuAmount, &rec.MandatoryFlag,
		&rec.FeeCategory, &rec.InvoiceReferenceID, &rec.VATAmount,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *SettlementRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cash_flow.settlement (
			id, tenant_id, account_id, amount, currency, settlement_date, settlement_status, subtype_code,
			ex_date, record_date, drip_reinvest_flag, tax_withholding_amount,
			coupon_period_start, accrued_interest, payment_frequency,
			call_notice_id, due_date, management_fee_portion, investment_portion,
			return_of_capital, preferred_return, carried_interest_retained,
			action_type_code, cash_in_lieu_amount, mandatory_flag,
			fee_category, invoice_reference_id, vat_amount,
			created_at, updated_at, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,NOW(),NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.AccountID, rec.Amount, rec.Currency,
		rec.SettlementDate, rec.SettlementStatus, rec.SubtypeCode,
		rec.ExDate, rec.RecordDate, rec.DripReinvestFlag, rec.TaxWithholdingAmount,
		rec.CouponPeriodStart, rec.AccruedInterest, rec.PaymentFrequency,
		rec.CallNoticeID, rec.DueDate, rec.ManagementFeePortion, rec.InvestmentPortion,
		rec.ReturnOfCapital, rec.PreferredReturn, rec.CarriedInterestRetained,
		rec.ActionTypeCode, rec.CashInLieuAmount, rec.MandatoryFlag,
		rec.FeeCategory, rec.InvoiceReferenceID, rec.VATAmount,
	)
	if err != nil {
		return fmt.Errorf("failed to create settlement: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE cash_flow.settlement SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete settlement: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}