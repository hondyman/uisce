package alternative_investment

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

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]AlternativeInvestmentRecord, error) {
	query := `
		SELECT id, tenant_id, investment_id, client_id, investment_type, fund_name, general_partner, vintage_year,
		       total_commitment_amount, unfunded_commitment, total_capital_called, total_distributions,
		       current_nav, nav_date, valuation_source,
		       irr_since_inception, tvpi, dpi, rvpi, moic,
		       lock_up_end_date, redemption_notice_days, redemption_frequency,
		       last_capital_call_date, last_distribution_date, last_k1_received_date,
		       subtype_code,
		       created_at, updated_at, valid_from, valid_to
		FROM altinv.alternative_investment
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alternative investments: %w", err)
	}
	defer rows.Close()

	var records []AlternativeInvestmentRecord
	for rows.Next() {
		var rec AlternativeInvestmentRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.InvestmentID, &rec.ClientID, &rec.InvestmentType, &rec.FundName, &rec.GeneralPartner, &rec.VintageYear,
			&rec.TotalCommitmentAmount, &rec.UnfundedCommitment, &rec.TotalCapitalCalled, &rec.TotalDistributions,
			&rec.CurrentNAV, &rec.NAVDate, &rec.ValuationSource,
			&rec.IRRSinceInception, &rec.TVPI, &rec.DPI, &rec.RVPI, &rec.MOIC,
			&rec.LockUpEndDate, &rec.RedemptionNoticeDays, &rec.RedemptionFrequency,
			&rec.LastCapitalCallDate, &rec.LastDistributionDate, &rec.LastK1ReceivedDate,
			&rec.SubtypeCode,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan alternative investment: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*AlternativeInvestmentRecord, error) {
	var rec AlternativeInvestmentRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, investment_id, client_id, investment_type, fund_name, general_partner, vintage_year,
		       total_commitment_amount, unfunded_commitment, total_capital_called, total_distributions,
		       current_nav, nav_date, valuation_source,
		       irr_since_inception, tvpi, dpi, rvpi, moic,
		       lock_up_end_date, redemption_notice_days, redemption_frequency,
		       last_capital_call_date, last_distribution_date, last_k1_received_date,
		       subtype_code,
		       created_at, updated_at, valid_from, valid_to
		FROM altinv.alternative_investment
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.InvestmentID, &rec.ClientID, &rec.InvestmentType, &rec.FundName, &rec.GeneralPartner, &rec.VintageYear,
		&rec.TotalCommitmentAmount, &rec.UnfundedCommitment, &rec.TotalCapitalCalled, &rec.TotalDistributions,
		&rec.CurrentNAV, &rec.NAVDate, &rec.ValuationSource,
		&rec.IRRSinceInception, &rec.TVPI, &rec.DPI, &rec.RVPI, &rec.MOIC,
		&rec.LockUpEndDate, &rec.RedemptionNoticeDays, &rec.RedemptionFrequency,
		&rec.LastCapitalCallDate, &rec.LastDistributionDate, &rec.LastK1ReceivedDate,
		&rec.SubtypeCode,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alternative investment: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *AlternativeInvestmentRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO altinv.alternative_investment (
			id, tenant_id, investment_id, client_id, investment_type, fund_name, general_partner, vintage_year,
			total_commitment_amount, unfunded_commitment, total_capital_called, total_distributions,
			current_nav, nav_date, valuation_source,
			irr_since_inception, tvpi, dpi, rvpi, moic,
			lock_up_end_date, redemption_notice_days, redemption_frequency,
			last_capital_call_date, last_distribution_date, last_k1_received_date,
			subtype_code,
			created_at, updated_at, valid_from
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,NOW(),NOW(),NOW()
		)`,
		rec.ID, rec.TenantID, rec.InvestmentID, rec.ClientID, rec.InvestmentType, rec.FundName, rec.GeneralPartner, rec.VintageYear,
		rec.TotalCommitmentAmount, rec.UnfundedCommitment, rec.TotalCapitalCalled, rec.TotalDistributions,
		rec.CurrentNAV, rec.NAVDate, rec.ValuationSource,
		rec.IRRSinceInception, rec.TVPI, rec.DPI, rec.RVPI, rec.MOIC,
		rec.LockUpEndDate, rec.RedemptionNoticeDays, rec.RedemptionFrequency,
		rec.LastCapitalCallDate, rec.LastDistributionDate, rec.LastK1ReceivedDate,
		rec.SubtypeCode,
	)
	if err != nil {
		return fmt.Errorf("failed to create alternative investment: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE altinv.alternative_investment SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete alternative investment: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}