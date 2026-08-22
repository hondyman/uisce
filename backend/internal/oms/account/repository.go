package account

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

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]AccountRecord, error) {
	query := `
		SELECT id, tenant_id, account_number, account_name, base_currency, status, subtype_code,
		       sponsor_id, mandate_type, erisa_flag, fee_schedule_code,
		       tax_id_type, citizenship, margin_agreement_flag, accredited_investor_status,
		       sponsor_firm, model_strategy_id, overlay_manager_id, rebalance_frequency,
		       trust_type, grantor_name, trustee_signatory_id, dissolution_date,
		       plan_type, vesting_schedule_code, rmd_eligible_flag, custodian_bank_id,
		       corporate_entity_id, treasury_signatory_group, wire_limit_daily,
		       created_at, updated_at, valid_from, valid_to
		FROM oms.account
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var records []AccountRecord
	for rows.Next() {
		var rec AccountRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.AccountNumber, &rec.AccountName, &rec.BaseCurrency,
			&rec.Status, &rec.SubtypeCode,
			&rec.SponsorID, &rec.MandateType, &rec.ErisaFlag, &rec.FeeScheduleCode,
			&rec.TaxIDType, &rec.Citizenship, &rec.MarginAgreementFlag, &rec.AccreditedInvestorStatus,
			&rec.SponsorFirm, &rec.ModelStrategyID, &rec.OverlayManagerID, &rec.RebalanceFrequency,
			&rec.TrustType, &rec.GrantorName, &rec.TrusteeSignatoryID, &rec.DissolutionDate,
			&rec.PlanType, &rec.VestingScheduleCode, &rec.RMDEligibleFlag, &rec.CustodianBankID,
			&rec.CorporateEntityID, &rec.TreasurySignatoryGroup, &rec.WireLimitDaily,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*AccountRecord, error) {
	var rec AccountRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, account_number, account_name, base_currency, status, subtype_code,
		       sponsor_id, mandate_type, erisa_flag, fee_schedule_code,
		       tax_id_type, citizenship, margin_agreement_flag, accredited_investor_status,
		       sponsor_firm, model_strategy_id, overlay_manager_id, rebalance_frequency,
		       trust_type, grantor_name, trustee_signatory_id, dissolution_date,
		       plan_type, vesting_schedule_code, rmd_eligible_flag, custodian_bank_id,
		       corporate_entity_id, treasury_signatory_group, wire_limit_daily,
		       created_at, updated_at, valid_from, valid_to
		FROM oms.account
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.AccountNumber, &rec.AccountName, &rec.BaseCurrency,
		&rec.Status, &rec.SubtypeCode,
		&rec.SponsorID, &rec.MandateType, &rec.ErisaFlag, &rec.FeeScheduleCode,
		&rec.TaxIDType, &rec.Citizenship, &rec.MarginAgreementFlag, &rec.AccreditedInvestorStatus,
		&rec.SponsorFirm, &rec.ModelStrategyID, &rec.OverlayManagerID, &rec.RebalanceFrequency,
		&rec.TrustType, &rec.GrantorName, &rec.TrusteeSignatoryID, &rec.DissolutionDate,
		&rec.PlanType, &rec.VestingScheduleCode, &rec.RMDEligibleFlag, &rec.CustodianBankID,
		&rec.CorporateEntityID, &rec.TreasurySignatoryGroup, &rec.WireLimitDaily,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *AccountRecord) error {
	rec.ID = uuid.New()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oms.account (
			id, tenant_id, account_number, account_name, base_currency, status, subtype_code,
			sponsor_id, mandate_type, erisa_flag, fee_schedule_code,
			tax_id_type, citizenship, margin_agreement_flag, accredited_investor_status,
			sponsor_firm, model_strategy_id, overlay_manager_id, rebalance_frequency,
			trust_type, grantor_name, trustee_signatory_id, dissolution_date,
			plan_type, vesting_schedule_code, rmd_eligible_flag, custodian_bank_id,
			corporate_entity_id, treasury_signatory_group, wire_limit_daily,
			created_at, updated_at, valid_from
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,NOW(),NOW(),NOW()
		)`,
		rec.ID, rec.TenantID, rec.AccountNumber, rec.AccountName, rec.BaseCurrency,
		rec.Status, rec.SubtypeCode,
		rec.SponsorID, rec.MandateType, rec.ErisaFlag, rec.FeeScheduleCode,
		rec.TaxIDType, rec.Citizenship, rec.MarginAgreementFlag, rec.AccreditedInvestorStatus,
		rec.SponsorFirm, rec.ModelStrategyID, rec.OverlayManagerID, rec.RebalanceFrequency,
		rec.TrustType, rec.GrantorName, rec.TrusteeSignatoryID, rec.DissolutionDate,
		rec.PlanType, rec.VestingScheduleCode, rec.RMDEligibleFlag, rec.CustodianBankID,
		rec.CorporateEntityID, rec.TreasurySignatoryGroup, rec.WireLimitDaily,
	)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE oms.account SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete account: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
