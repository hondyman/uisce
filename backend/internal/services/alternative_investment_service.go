package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/types"

	"github.com/google/uuid"
)

// AlternativeInvestmentService handles CRUD operations for alternative investments
type AlternativeInvestmentService struct {
	db *sql.DB
}

// NewAlternativeInvestmentService creates a new alternative investments service
func NewAlternativeInvestmentService(db *sql.DB) *AlternativeInvestmentService {
	return &AlternativeInvestmentService{db: db}
}

// CreateInvestment creates a new alternative investment
func (s *AlternativeInvestmentService) CreateInvestment(ctx context.Context, inv *types.AlternativeInvestment) error {
	query := `
		INSERT INTO alternative_investments (
			tenant_id, client_id, account_id, fund_name, fund_manager,
			asset_class, sub_strategy, vintage_year, commitment_amount,
			commitment_currency, management_fee_pct, performance_fee_pct,
			hurdle_rate_pct, has_high_water_mark, has_catch_up,
			tax_entity_type, inception_date, expected_term_years, maturity_date, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
		RETURNING id, created_at, updated_at, unfunded_commitment, current_nav, capital_called, capital_distributed
	`

	return s.db.QueryRowContext(ctx, query,
		inv.TenantID, inv.ClientID, inv.AccountID, inv.FundName, inv.FundManager,
		inv.AssetClass, inv.SubStrategy, inv.VintageYear, inv.CommitmentAmount,
		inv.CommitmentCurrency, inv.ManagementFeePct, inv.PerformanceFeePct,
		inv.HurdleRatePct, inv.HasHighWaterMark, inv.HasCatchUp,
		inv.TaxEntityType, inv.InceptionDate, inv.ExpectedTermYears, inv.MaturityDate, inv.Notes,
	).Scan(
		&inv.ID, &inv.CreatedAt, &inv.UpdatedAt, &inv.UnfundedCommitment,
		&inv.CurrentNAV, &inv.CapitalCalled, &inv.CapitalDistributed,
	)
}

// GetInvestment retrieves a single investment by ID
func (s *AlternativeInvestmentService) GetInvestment(ctx context.Context, tenantID, investmentID uuid.UUID) (*types.AlternativeInvestment, error) {
	query := `
		SELECT 
			id, tenant_id, client_id, account_id, fund_name, fund_manager,
			asset_class, sub_strategy, vintage_year, commitment_amount,
			commitment_currency, capital_called, capital_distributed, unfunded_commitment,
			current_nav, last_valuation_date, valuation_method,
			management_fee_pct, performance_fee_pct, hurdle_rate_pct,
			has_high_water_mark, has_catch_up, tax_entity_type,
			k1_received, k1_received_date, inception_date, expected_term_years,
			maturity_date, notes, created_at, updated_at
		FROM alternative_investments
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	inv := &types.AlternativeInvestment{}
	err := s.db.QueryRowContext(ctx, query, investmentID, tenantID).Scan(
		&inv.ID, &inv.TenantID, &inv.ClientID, &inv.AccountID, &inv.FundName, &inv.FundManager,
		&inv.AssetClass, &inv.SubStrategy, &inv.VintageYear, &inv.CommitmentAmount,
		&inv.CommitmentCurrency, &inv.CapitalCalled, &inv.CapitalDistributed, &inv.UnfundedCommitment,
		&inv.CurrentNAV, &inv.LastValuationDate, &inv.ValuationMethod,
		&inv.ManagementFeePct, &inv.PerformanceFeePct, &inv.HurdleRatePct,
		&inv.HasHighWaterMark, &inv.HasCatchUp, &inv.TaxEntityType,
		&inv.K1Received, &inv.K1ReceivedDate, &inv.InceptionDate, &inv.ExpectedTermYears,
		&inv.MaturityDate, &inv.Notes, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return inv, nil
}

// GetInvestmentsByClient retrieves all alternative investments for a client
func (s *AlternativeInvestmentService) GetInvestmentsByClient(ctx context.Context, tenantID, clientID uuid.UUID) ([]*types.AlternativeInvestment, error) {
	query := `
		SELECT 
			id, tenant_id, client_id, account_id, fund_name, fund_manager,
			asset_class, sub_strategy, vintage_year, commitment_amount,
			commitment_currency, capital_called, capital_distributed, unfunded_commitment,
			current_nav, last_valuation_date, valuation_method,
			management_fee_pct, performance_fee_pct, hurdle_rate_pct,
			has_high_water_mark, has_catch_up, tax_entity_type,
			k1_received, k1_received_date, inception_date, expected_term_years,
			maturity_date, notes, created_at, updated_at
		FROM alternative_investments
		WHERE tenant_id = $1 AND client_id = $2 AND deleted_at IS NULL
		ORDER BY inception_date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var investments []*types.AlternativeInvestment
	for rows.Next() {
		inv := &types.AlternativeInvestment{}
		err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.ClientID, &inv.AccountID, &inv.FundName, &inv.FundManager,
			&inv.AssetClass, &inv.SubStrategy, &inv.VintageYear, &inv.CommitmentAmount,
			&inv.CommitmentCurrency, &inv.CapitalCalled, &inv.CapitalDistributed, &inv.UnfundedCommitment,
			&inv.CurrentNAV, &inv.LastValuationDate, &inv.ValuationMethod,
			&inv.ManagementFeePct, &inv.PerformanceFeePct, &inv.HurdleRatePct,
			&inv.HasHighWaterMark, &inv.HasCatchUp, &inv.TaxEntityType,
			&inv.K1Received, &inv.K1ReceivedDate, &inv.InceptionDate, &inv.ExpectedTermYears,
			&inv.MaturityDate, &inv.Notes, &inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		investments = append(investments, inv)
	}

	return investments, rows.Err()
}

// UpdateInvestment updates an existing investment
func (s *AlternativeInvestmentService) UpdateInvestment(ctx context.Context, inv *types.AlternativeInvestment) error {
	query := `
		UPDATE alternative_investments
		SET 
			fund_name = $1, fund_manager = $2, sub_strategy = $3,
			current_nav = $4, last_valuation_date = $5, valuation_method = $6,
			management_fee_pct = $7, performance_fee_pct = $8, hurdle_rate_pct = $9,
			k1_received = $10, k1_received_date = $11,
			notes = $12, updated_at = NOW()
		WHERE id = $13 AND tenant_id = $14 AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query,
		inv.FundName, inv.FundManager, inv.SubStrategy,
		inv.CurrentNAV, inv.LastValuationDate, inv.ValuationMethod,
		inv.ManagementFeePct, inv.PerformanceFeePct, inv.HurdleRatePct,
		inv.K1Received, inv.K1ReceivedDate,
		inv.Notes, inv.ID, inv.TenantID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("investment not found")
	}

	return nil
}

// DeleteInvestment soft-deletes an investment
func (s *AlternativeInvestmentService) DeleteInvestment(ctx context.Context, tenantID, investmentID uuid.UUID) error {
	query := `
		UPDATE alternative_investments
		SET deleted_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, investmentID, tenantID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("investment not found")
	}

	return nil
}

// RecordCapitalCall records a new capital call notice
func (s *AlternativeInvestmentService) RecordCapitalCall(ctx context.Context, call *types.CapitalCall) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert capital call
	query := `
		INSERT INTO capital_calls (
			investment_id, call_number, call_date, due_date, amount_requested,
			notice_document_id
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at, status, amount_funded
	`

	err = tx.QueryRowContext(ctx, query,
		call.InvestmentID, call.CallNumber, call.CallDate, call.DueDate,
		call.AmountRequested, call.NoticeDocumentID,
	).Scan(&call.ID, &call.CreatedAt, &call.UpdatedAt, &call.Status, &call.AmountFunded)
	if err != nil {
		return err
	}

	// Perform liquidity check
	err = s.checkLiquidity(ctx, tx, call)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// checkLiquidity performs liquidity analysis for a capital call
func (s *AlternativeInvestmentService) checkLiquidity(ctx context.Context, tx *sql.Tx, call *types.CapitalCall) error {
	// Get client ID from investment
	var clientID uuid.UUID
	err := tx.QueryRowContext(ctx, `
		SELECT client_id FROM alternative_investments WHERE id = $1
	`, call.InvestmentID).Scan(&clientID)
	if err != nil {
		return err
	}

	// Calculate available liquid cash (assuming accounts table exists)
	var totalCash sql.NullFloat64
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(balance), 0)
		FROM accounts
		WHERE client_id = $1
		  AND account_type IN ('CHECKING', 'SAVINGS', 'MONEY_MARKET')
		  AND deleted_at IS NULL
	`, clientID).Scan(&totalCash)
	if err != nil {
		// If accounts table doesn't exist yet, skip liquidity check
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	liquidCash := totalCash.Float64
	required := call.AmountRequested

	// Determine status
	var status string
	if liquidCash > required*1.5 {
		status = "SUFFICIENT"
	} else if liquidCash > required*1.1 {
		status = "MARGINAL"
	} else {
		status = "INSUFFICIENT"
	}

	// Update capital call with liquidity status
	_, err = tx.ExecContext(ctx, `
		UPDATE capital_calls
		SET liquidity_check_status = $1
		WHERE id = $2
	`, status, call.ID)
	if err != nil {
		return err
	}

	call.LiquidityCheckStatus = &status

	if status == "INSUFFICIENT" {
		// Trigger alert if insufficient - integrate with notification system
		fmt.Printf("[AltInvestment] ALERT: Insufficient funds for capital call %s. Required: %.2f, Available: %.2f\n",
			call.ID, call.AmountRequested, liquidCash)

		// In production: Send notification via engagement notification service
		// notificationPayload := map[string]interface{}{
		//     "type": "capital_call_insufficient_funds",
		//     "investment_id": call.InvestmentID,
		//     "amount_due": call.AmountRequested,
		//     "available": liquidCash,
		//     "shortfall": call.AmountRequested - liquidCash,
		// }
		// s.notificationSvc.Send(ctx, notificationPayload)

		// For now, we'll return an error to prevent funding if insufficient
		return fmt.Errorf("insufficient funds: need %.2f, have %.2f", call.AmountRequested, liquidCash)
	}

	return nil
}

// FundCapitalCall marks a capital call as funded
func (s *AlternativeInvestmentService) FundCapitalCall(ctx context.Context, callID uuid.UUID, amount float64, fundedDate time.Time) error {
	query := `
		UPDATE capital_calls
		SET 
			status = CASE 
				WHEN $1 >= amount_requested THEN 'FUNDED'
				ELSE 'PARTIALLY_FUNDED'
			END,
			amount_funded = $1,
			funded_date = $2,
			updated_at = NOW()
		WHERE id = $3
	`

	result, err := s.db.ExecContext(ctx, query, amount, fundedDate, callID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("capital call not found")
	}

	// Note: Trigger will automatically update investment.capital_called

	return nil
}

// RecordDistribution records a distribution from an investment
func (s *AlternativeInvestmentService) RecordDistribution(ctx context.Context, dist *types.CapitalDistribution) error {
	query := `
		INSERT INTO capital_distributions (
			investment_id, distribution_date, amount, distribution_type,
			is_recallable, ordinary_income, long_term_capital_gain,
			short_term_capital_gain, return_of_capital, notice_document_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
		RETURNING id, created_at
	`

	err := s.db.QueryRowContext(ctx, query,
		dist.InvestmentID, dist.DistributionDate, dist.Amount, dist.DistributionType,
		dist.IsRecallable, dist.OrdinaryIncome, dist.LongTermCapitalGain,
		dist.ShortTermCapitalGain, dist.ReturnOfCapital, dist.NoticeDocumentID,
	).Scan(&dist.ID, &dist.CreatedAt)

	// Note: Trigger will automatically update investment.capital_distributed

	return err
}

// GetCapitalCallsByInvestment retrieves all capital calls for an investment
func (s *AlternativeInvestmentService) GetCapitalCallsByInvestment(ctx context.Context, investmentID uuid.UUID) ([]*types.CapitalCall, error) {
	query := `
		SELECT 
			id, investment_id, call_number, call_date, due_date, amount_requested,
			status, amount_funded, funded_date, liquidity_check_status,
			recommended_funding_source_account_id, alert_sent, alert_sent_at,
			notice_document_id, created_at, updated_at
		FROM capital_calls
		WHERE investment_id = $1
		ORDER BY call_number ASC
	`

	rows, err := s.db.QueryContext(ctx, query, investmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []*types.CapitalCall
	for rows.Next() {
		call := &types.CapitalCall{}
		err := rows.Scan(
			&call.ID, &call.InvestmentID, &call.CallNumber, &call.CallDate, &call.DueDate,
			&call.AmountRequested, &call.Status, &call.AmountFunded, &call.FundedDate,
			&call.LiquidityCheckStatus, &call.RecommendedFundingSourceAccount,
			&call.AlertSent, &call.AlertSentAt, &call.NoticeDocumentID,
			&call.CreatedAt, &call.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		calls = append(calls, call)
	}

	return calls, rows.Err()
}

// GetDistributionsByInvestment retrieves all distributions for an investment
func (s *AlternativeInvestmentService) GetDistributionsByInvestment(ctx context.Context, investmentID uuid.UUID) ([]*types.CapitalDistribution, error) {
	query := `
		SELECT 
			id, investment_id, distribution_date, amount, distribution_type,
			is_recallable, ordinary_income, long_term_capital_gain,
			short_term_capital_gain, return_of_capital, notice_document_id, created_at
		FROM capital_distributions
		WHERE investment_id = $1
		ORDER BY distribution_date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, investmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var distributions []*types.CapitalDistribution
	for rows.Next() {
		dist := &types.CapitalDistribution{}
		err := rows.Scan(
			&dist.ID, &dist.InvestmentID, &dist.DistributionDate, &dist.Amount,
			&dist.DistributionType, &dist.IsRecallable, &dist.OrdinaryIncome,
			&dist.LongTermCapitalGain, &dist.ShortTermCapitalGain,
			&dist.ReturnOfCapital, &dist.NoticeDocumentID, &dist.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		distributions = append(distributions, dist)
	}

	return distributions, rows.Err()
}
