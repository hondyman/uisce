package security

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

func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]SecurityRecord, error) {
	query := `
		SELECT id, tenant_id, security_name, identifier_type, identifier_value, subtype_code,
		       ticker, isin, voting_rights_type, dividend_currency,
		       coupon_rate, maturity_date, day_count_convention, inflation_protected_flag,
		       credit_rating_sp, call_date, conversion_ratio, seniority_level,
		       pool_number, factor_current, prepayment_speed_cpr, tranche_tier,
		       contract_size, strike_price, put_call_indicator, exchange_mic,
		       isda_agreement_id, fixed_rate, floating_index_name, counterparty_lei,
		       created_at, updated_at, valid_from, valid_to
		FROM oms.security
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtypeCode != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtypeCode)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list securities: %w", err)
	}
	defer rows.Close()

	var records []SecurityRecord
	for rows.Next() {
		var rec SecurityRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.SecurityName, &rec.IdentifierType, &rec.IdentifierValue,
			&rec.SubtypeCode,
			&rec.Ticker, &rec.ISIN, &rec.VotingRightsType, &rec.DividendCurrency,
			&rec.CouponRate, &rec.MaturityDate, &rec.DayCountConvention, &rec.InflationProtectedFlag,
			&rec.CreditRatingSP, &rec.CallDate, &rec.ConversionRatio, &rec.SeniorityLevel,
			&rec.PoolNumber, &rec.FactorCurrent, &rec.PrepaymentSpeedCPR, &rec.TrancheTier,
			&rec.ContractSize, &rec.StrikePrice, &rec.PutCallIndicator, &rec.ExchangeMIC,
			&rec.ISDAAgreementID, &rec.FixedRate, &rec.FloatingIndexName, &rec.CounterpartyLEI,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan security: %w", err)
		}
		records = append(records, rec)
	}

	return records, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id uuid.UUID) (*SecurityRecord, error) {
	var rec SecurityRecord
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, security_name, identifier_type, identifier_value, subtype_code,
		       ticker, isin, voting_rights_type, dividend_currency,
		       coupon_rate, maturity_date, day_count_convention, inflation_protected_flag,
		       credit_rating_sp, call_date, conversion_ratio, seniority_level,
		       pool_number, factor_current, prepayment_speed_cpr, tranche_tier,
		       contract_size, strike_price, put_call_indicator, exchange_mic,
		       isda_agreement_id, fixed_rate, floating_index_name, counterparty_lei,
		       created_at, updated_at, valid_from, valid_to
		FROM oms.security
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.SecurityName, &rec.IdentifierType, &rec.IdentifierValue,
		&rec.SubtypeCode,
		&rec.Ticker, &rec.ISIN, &rec.VotingRightsType, &rec.DividendCurrency,
		&rec.CouponRate, &rec.MaturityDate, &rec.DayCountConvention, &rec.InflationProtectedFlag,
		&rec.CreditRatingSP, &rec.CallDate, &rec.ConversionRatio, &rec.SeniorityLevel,
		&rec.PoolNumber, &rec.FactorCurrent, &rec.PrepaymentSpeedCPR, &rec.TrancheTier,
		&rec.ContractSize, &rec.StrikePrice, &rec.PutCallIndicator, &rec.ExchangeMIC,
		&rec.ISDAAgreementID, &rec.FixedRate, &rec.FloatingIndexName, &rec.CounterpartyLEI,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get security: %w", err)
	}
	return &rec, nil
}

func (r *Repository) Create(ctx context.Context, rec *SecurityRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oms.security (
			id, tenant_id, security_name, identifier_type, identifier_value, subtype_code,
			ticker, isin, voting_rights_type, dividend_currency,
			coupon_rate, maturity_date, day_count_convention, inflation_protected_flag,
			credit_rating_sp, call_date, conversion_ratio, seniority_level,
			pool_number, factor_current, prepayment_speed_cpr, tranche_tier,
			contract_size, strike_price, put_call_indicator, exchange_mic,
			isda_agreement_id, fixed_rate, floating_index_name, counterparty_lei,
			created_at, updated_at, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,NOW(),NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.SecurityName, rec.IdentifierType, rec.IdentifierValue,
		rec.SubtypeCode,
		rec.Ticker, rec.ISIN, rec.VotingRightsType, rec.DividendCurrency,
		rec.CouponRate, rec.MaturityDate, rec.DayCountConvention, rec.InflationProtectedFlag,
		rec.CreditRatingSP, rec.CallDate, rec.ConversionRatio, rec.SeniorityLevel,
		rec.PoolNumber, rec.FactorCurrent, rec.PrepaymentSpeedCPR, rec.TrancheTier,
		rec.ContractSize, rec.StrikePrice, rec.PutCallIndicator, rec.ExchangeMIC,
		rec.ISDAAgreementID, rec.FixedRate, rec.FloatingIndexName, rec.CounterpartyLEI,
	)
	if err != nil {
		return fmt.Errorf("failed to create security: %w", err)
	}
	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE oms.security SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete security: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
