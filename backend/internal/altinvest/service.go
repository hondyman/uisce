package altinvest

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient interface for GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

type Service struct {
	DB           *sqlx.DB
	hasuraClient HasuraClient
}

func NewService(db *sqlx.DB) *Service {
	return &Service{DB: db}
}

func NewServiceWithHasura(db *sqlx.DB, hasuraClient HasuraClient) *Service {
	return &Service{
		DB:           db,
		hasuraClient: hasuraClient,
	}
}

// CreateInvestment creates a new alternative investment record
func (s *Service) CreateInvestment(ctx context.Context, inv *AlternativeInvestment) error {
	if inv.InvestmentID == uuid.Nil {
		inv.InvestmentID = uuid.New()
	}
	inv.CreatedAt = time.Now()
	inv.UpdatedAt = time.Now()

	err := s.createInvestmentRecord(ctx, inv)
	if err != nil {
		return fmt.Errorf("failed to create investment: %w", err)
	}
	return nil
}

// GetClientInvestments retrieves all alternative investments for a client
func (s *Service) GetClientInvestments(ctx context.Context, clientID uuid.UUID) ([]AlternativeInvestment, error) {
	investments, err := s.getClientInvestmentsRecords(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get investments: %w", err)
	}
	return investments, nil
}

// RecordCapitalCall records a new capital call
func (s *Service) RecordCapitalCall(ctx context.Context, call *CapitalCall) error {
	if call.CallID == uuid.Nil {
		call.CallID = uuid.New()
	}
	call.CreatedAt = time.Now()

	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert call
	err = s.insertCapitalCallRecord(ctx, tx, call)
	if err != nil {
		return fmt.Errorf("failed to record capital call: %w", err)
	}

	// Update investment totals if funded
	if call.Status == "FUNDED" {
		err = s.updateInvestmentTotalsRecord(ctx, tx, call.InvestmentID, call.AmountFunded)
		if err != nil {
			return fmt.Errorf("failed to update investment totals: %w", err)
		}
	}

	return tx.Commit()
}

// CalculateMetrics updates performance metrics (simplified)
func (s *Service) CalculateMetrics(ctx context.Context, investmentID uuid.UUID) error {
	// In a real system, this would use XIRR calculation and more complex logic.
	// Here we do basic TVPI/DPI updates based on current NAV and flows.

	err := s.calculateMetricsRecord(ctx, investmentID)
	if err != nil {
		return fmt.Errorf("failed to calculate metrics: %w", err)
	}
	return nil
}

// Helper methods for Hasura integration - SQL fallback for complex operations

func (s *Service) createInvestmentRecord(ctx context.Context, inv *AlternativeInvestment) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for complex struct with 24 fields
	query := `
		INSERT INTO alternative_investments (
investment_id, client_id, investment_type, fund_name, general_partner, vintage_year,
total_commitment_amount, unfunded_commitment, total_capital_called, total_distributions,
current_nav, nav_date, valuation_source, irr_since_inception, tvpi, dpi, rvpi, moic,
lock_up_end_date, redemption_notice_days, redemption_frequency, metadata, created_at, updated_at
) VALUES (
:investment_id, :client_id, :investment_type, :fund_name, :general_partner, :vintage_year,
:total_commitment_amount, :unfunded_commitment, :total_capital_called, :total_distributions,
:current_nav, :nav_date, :valuation_source, :irr_since_inception, :tvpi, :dpi, :rvpi, :moic,
:lock_up_end_date, :redemption_notice_days, :redemption_frequency, :metadata, :created_at, :updated_at
)`
	_, err := s.DB.NamedExecContext(ctx, query, inv)
	return err
}

func (s *Service) getClientInvestmentsRecords(ctx context.Context, clientID uuid.UUID) ([]AlternativeInvestment, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for SELECT * with many fields
	var investments []AlternativeInvestment
	err := s.DB.SelectContext(ctx, &investments, "SELECT * FROM alternative_investments WHERE client_id = $1", clientID)
	return investments, err
}

func (s *Service) insertCapitalCallRecord(ctx context.Context, tx *sqlx.Tx, call *CapitalCall) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for transaction INSERT
	query := `
		INSERT INTO capital_calls (
call_id, investment_id, notice_date, due_date, amount_requested, amount_funded,
status, funding_source_account, liquidity_check_passed, alert_sent_at, created_at
) VALUES (
:call_id, :investment_id, :notice_date, :due_date, :amount_requested, :amount_funded,
:status, :funding_source_account, :liquidity_check_passed, :alert_sent_at, :created_at
)`
	_, err := tx.NamedExecContext(ctx, query, call)
	return err
}

func (s *Service) updateInvestmentTotalsRecord(ctx context.Context, tx *sqlx.Tx, investmentID uuid.UUID, amountFunded float64) error {
	// TODO: Use HasuraClient for UPDATE when available
	// For now, use SQL fallback for UPDATE with arithmetic in transaction
	_, err := tx.ExecContext(ctx, `
		UPDATE alternative_investments 
		SET total_capital_called = total_capital_called + $1,
		    unfunded_commitment = unfunded_commitment - $1,
			updated_at = NOW()
		WHERE investment_id = $2
	`, amountFunded, investmentID)
	return err
}

func (s *Service) calculateMetricsRecord(ctx context.Context, investmentID uuid.UUID) error {
	// TODO: Use HasuraClient for UPDATE when available
	// For now, use SQL fallback for UPDATE with complex CASE expressions
	_, err := s.DB.ExecContext(ctx, `
		UPDATE alternative_investments
		SET 
			tvpi = CASE WHEN total_capital_called > 0 THEN (current_nav + total_distributions) / total_capital_called ELSE 0 END,
			dpi = CASE WHEN total_capital_called > 0 THEN total_distributions / total_capital_called ELSE 0 END,
			rvpi = CASE WHEN total_capital_called > 0 THEN current_nav / total_capital_called ELSE 0 END,
			moic = CASE WHEN total_capital_called > 0 THEN (current_nav + total_distributions) / total_capital_called ELSE 0 END,
			updated_at = NOW()
		WHERE investment_id = $1
	`, investmentID)
	return err
}
