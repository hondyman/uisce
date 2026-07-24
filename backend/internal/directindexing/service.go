package directindexing

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient interface for GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

// Account represents a direct indexing account
type Account struct {
	AccountID               uuid.UUID       `db:"account_id" json:"account_id"`
	ClientID                uuid.UUID       `db:"client_id" json:"client_id"`
	TenantID                uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	AccountNumber           string          `db:"account_number" json:"account_number"`
	AccountName             string          `db:"account_name" json:"account_name"`
	Custodian               string          `db:"custodian" json:"custodian"`
	BenchmarkIndex          string          `db:"benchmark_index" json:"benchmark_index"`
	TrackingMethod          string          `db:"tracking_method" json:"tracking_method"`
	CustomizationProfile    sql.NullString  `db:"customization_profile" json:"customization_profile"`
	TaxLotMethod            string          `db:"tax_lot_method" json:"tax_lot_method"`
	HarvestThresholdPct     float64         `db:"harvest_threshold_pct" json:"harvest_threshold_pct"`
	AutoHarvestEnabled      bool            `db:"auto_harvest_enabled" json:"auto_harvest_enabled"`
	FederalTaxBracket       sql.NullFloat64 `db:"federal_tax_bracket" json:"federal_tax_bracket"`
	StateTaxBracket         sql.NullFloat64 `db:"state_tax_bracket" json:"state_tax_bracket"`
	TotalMarketValue        float64         `db:"total_market_value" json:"total_market_value"`
	TotalCostBasis          float64         `db:"total_cost_basis" json:"total_cost_basis"`
	TotalUnrealizedGainLoss float64         `db:"total_unrealized_gain_loss" json:"total_unrealized_gain_loss"`
	YTDTaxLossHarvested     float64         `db:"ytd_tax_loss_harvested" json:"ytd_tax_loss_harvested"`
	YTDTaxSavings           float64         `db:"ytd_tax_savings" json:"ytd_tax_savings"`
	YTDReturnPct            sql.NullFloat64 `db:"ytd_return_pct" json:"ytd_return_pct"`
	YTDBenchmarkReturnPct   sql.NullFloat64 `db:"ytd_benchmark_return_pct" json:"ytd_benchmark_return_pct"`
	TrackingErrorPct        sql.NullFloat64 `db:"tracking_error_pct" json:"tracking_error_pct"`
	AccountStatus           string          `db:"account_status" json:"account_status"`
	InceptionDate           time.Time       `db:"inception_date" json:"inception_date"`
	LastRebalanceDate       sql.NullTime    `db:"last_rebalance_date" json:"last_rebalance_date"`
	NextRebalanceDate       sql.NullTime    `db:"next_rebalance_date" json:"next_rebalance_date"`
	CreatedAt               time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time       `db:"updated_at" json:"updated_at"`
}

// Holding represents a single stock holding
type Holding struct {
	HoldingID             uuid.UUID       `db:"holding_id" json:"holding_id"`
	AccountID             uuid.UUID       `db:"account_id" json:"account_id"`
	Ticker                string          `db:"ticker" json:"ticker"`
	CUSIP                 sql.NullString  `db:"cusip" json:"cusip"`
	SecurityName          sql.NullString  `db:"security_name" json:"security_name"`
	Sector                sql.NullString  `db:"sector" json:"sector"`
	SharesOwned           float64         `db:"shares_owned" json:"shares_owned"`
	AverageCostBasis      float64         `db:"average_cost_basis" json:"average_cost_basis"`
	CurrentPrice          float64         `db:"current_price" json:"current_price"`
	CurrentMarketValue    float64         `db:"current_market_value" json:"current_market_value"`
	PortfolioWeightPct    float64         `db:"portfolio_weight_pct" json:"portfolio_weight_pct"`
	BenchmarkWeightPct    float64         `db:"benchmark_weight_pct" json:"benchmark_weight_pct"`
	UnrealizedGainLoss    float64         `db:"unrealized_gain_loss" json:"unrealized_gain_loss"`
	UnrealizedGainLossPct sql.NullFloat64 `db:"unrealized_gain_loss_pct" json:"unrealized_gain_loss_pct"`
	HarvestEligible       bool            `db:"harvest_eligible" json:"harvest_eligible"`
	LastHarvestDate       sql.NullTime    `db:"last_harvest_date" json:"last_harvest_date"`
	EstimatedTaxSavings   sql.NullFloat64 `db:"estimated_tax_savings" json:"estimated_tax_savings"`
}

// Opportunity represents a tax-loss harvesting opportunity
type Opportunity struct {
	OpportunityID           uuid.UUID       `db:"opportunity_id" json:"opportunity_id"`
	AccountID               uuid.UUID       `db:"account_id" json:"account_id"`
	HoldingID               uuid.UUID       `db:"holding_id" json:"holding_id"`
	Ticker                  string          `db:"ticker" json:"ticker"`
	SharesToSell            float64         `db:"shares_to_sell" json:"shares_to_sell"`
	CostBasisPerShare       float64         `db:"cost_basis_per_share" json:"cost_basis_per_share"`
	CurrentPrice            float64         `db:"current_price" json:"current_price"`
	UnrealizedLoss          float64         `db:"unrealized_loss" json:"unrealized_loss"`
	UnrealizedLossPct       float64         `db:"unrealized_loss_pct" json:"unrealized_loss_pct"`
	EstimatedTaxSavings     float64         `db:"estimated_tax_savings" json:"estimated_tax_savings"`
	TaxRateUsed             float64         `db:"tax_rate_used" json:"tax_rate_used"`
	ReplacementTicker       sql.NullString  `db:"replacement_ticker" json:"replacement_ticker"`
	ReplacementName         sql.NullString  `db:"replacement_name" json:"replacement_name"`
	CorrelationWithOriginal sql.NullFloat64 `db:"correlation_with_original" json:"correlation_with_original"`
	ReplacementShares       sql.NullFloat64 `db:"replacement_shares" json:"replacement_shares"`
	ReplacementCost         sql.NullFloat64 `db:"replacement_cost" json:"replacement_cost"`
	WashSaleRisk            bool            `db:"wash_sale_risk" json:"wash_sale_risk"`
	OpportunityStatus       string          `db:"opportunity_status" json:"opportunity_status"`
	DetectedAt              time.Time       `db:"detected_at" json:"detected_at"`
	ApprovedAt              sql.NullTime    `db:"approved_at" json:"approved_at"`
	ExecutedAt              sql.NullTime    `db:"executed_at" json:"executed_at"`
}

// Service provides direct indexing operations
type Service struct {
	db           *sqlx.DB
	hasuraClient HasuraClient
}

// NewService creates a new direct indexing service
func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// NewServiceWithHasura creates a service with Hasura GraphQL client
func NewServiceWithHasura(db *sqlx.DB, hasuraClient HasuraClient) *Service {
	return &Service{
		db:           db,
		hasuraClient: hasuraClient,
	}
}

// GetAccount retrieves an account by ID
func (s *Service) GetAccount(ctx context.Context, accountID uuid.UUID) (*Account, error) {
	return s.getAccountRecord(ctx, accountID)
}

// ListAccounts lists all accounts for a client
func (s *Service) ListAccounts(ctx context.Context, clientID uuid.UUID) ([]Account, error) {
	return s.listAccountsRecords(ctx, clientID)
}

// GetHoldings retrieves all holdings for an account
func (s *Service) GetHoldings(ctx context.Context, accountID uuid.UUID) ([]Holding, error) {
	return s.getHoldingsRecords(ctx, accountID)
}

// GetOpportunities retrieves pending harvest opportunities for an account
func (s *Service) GetOpportunities(ctx context.Context, accountID uuid.UUID, status string) ([]Opportunity, error) {
	return s.getOpportunitiesRecords(ctx, accountID, status)
}

// ExecuteHarvest executes a harvest opportunity
func (s *Service) ExecuteHarvest(ctx context.Context, opportunityID uuid.UUID, approvedBy uuid.UUID) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update opportunity status
	err = s.updateOpportunityStatusRecord(ctx, tx, opportunityID, approvedBy)
	if err != nil {
		return err
	}

	// Get opportunity details for wash sale tracker
	opp, err := s.getOpportunityDetailsRecord(ctx, tx, opportunityID)
	if err != nil {
		return err
	}

	// Create wash sale tracker entry
	err = s.createWashSaleTrackerRecord(ctx, tx, opp)
	if err != nil {
		return err
	}

	// Update account YTD metrics
	err = s.updateAccountYTDMetricsRecord(ctx, tx, opp)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// DismissOpportunity dismisses a harvest opportunity
func (s *Service) DismissOpportunity(ctx context.Context, opportunityID uuid.UUID, reason string) error {
	return s.dismissOpportunityRecord(ctx, opportunityID, reason)
}

// GetPerformanceMetrics retrieves performance metrics for an account
func (s *Service) GetPerformanceMetrics(ctx context.Context, accountID uuid.UUID) (map[string]interface{}, error) {
	account, err := s.getAccountRecord(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Get holdings count
	holdingsCount, err := s.getHoldingsCountRecord(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Get pending opportunities count
	pendingOppsCount, pendingOppsSavings, err := s.getPendingOpportunitiesStatsRecord(ctx, accountID)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"account_id":                 account.AccountID,
		"account_name":               account.AccountName,
		"total_market_value":         account.TotalMarketValue,
		"total_cost_basis":           account.TotalCostBasis,
		"total_unrealized_gain_loss": account.TotalUnrealizedGainLoss,
		"ytd_tax_loss_harvested":     account.YTDTaxLossHarvested,
		"ytd_tax_savings":            account.YTDTaxSavings,
		"ytd_return_pct":             account.YTDReturnPct,
		"ytd_benchmark_return_pct":   account.YTDBenchmarkReturnPct,
		"tracking_error_pct":         account.TrackingErrorPct,
		"holdings_count":             holdingsCount,
		"pending_opportunities":      pendingOppsCount,
		"pending_tax_savings":        pendingOppsSavings,
	}, nil
}

// Helper methods for Hasura integration - SQL fallback for complex operations

func (s *Service) getAccountRecord(ctx context.Context, accountID uuid.UUID) (*Account, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for SELECT * with many fields
	var account Account
	query := `SELECT * FROM direct_index_accounts WHERE account_id = $1`
	err := s.db.GetContext(ctx, &account, query, accountID)
	return &account, err
}

func (s *Service) listAccountsRecords(ctx context.Context, clientID uuid.UUID) ([]Account, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for SELECT * with ORDER BY
	var accounts []Account
	query := `
		SELECT * FROM direct_index_accounts
		WHERE client_id = $1
		ORDER BY account_name
	`
	err := s.db.SelectContext(ctx, &accounts, query, clientID)
	return accounts, err
}

func (s *Service) getHoldingsRecords(ctx context.Context, accountID uuid.UUID) ([]Holding, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for SELECT * with ORDER BY
	var holdings []Holding
	query := `
		SELECT * FROM direct_index_holdings
		WHERE account_id = $1
		ORDER BY current_market_value DESC
	`
	err := s.db.SelectContext(ctx, &holdings, query, accountID)
	return holdings, err
}

func (s *Service) getOpportunitiesRecords(ctx context.Context, accountID uuid.UUID, status string) ([]Opportunity, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for dynamic WHERE clause
	var opportunities []Opportunity
	query := `
		SELECT * FROM tax_loss_opportunities
		WHERE account_id = $1
	`
	args := []interface{}{accountID}
	if status != "" {
		query += ` AND opportunity_status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY estimated_tax_savings DESC`
	err := s.db.SelectContext(ctx, &opportunities, query, args...)
	return opportunities, err
}

func (s *Service) updateOpportunityStatusRecord(ctx context.Context, tx *sqlx.Tx, opportunityID uuid.UUID, approvedBy uuid.UUID) error {
	// TODO: Use HasuraClient for UPDATE when available
	// For now, use SQL fallback for multi-field UPDATE in transaction
	_, err := tx.ExecContext(ctx, `
		UPDATE tax_loss_opportunities
		SET opportunity_status = 'EXECUTED',
		    approved_at = NOW(),
		    approved_by = $1,
		    executed_at = NOW()
		WHERE opportunity_id = $2
		AND opportunity_status = 'PENDING'
	`, approvedBy, opportunityID)
	return err
}

func (s *Service) getOpportunityDetailsRecord(ctx context.Context, tx *sqlx.Tx, opportunityID uuid.UUID) (*Opportunity, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for SELECT * in transaction
	var opp Opportunity
	err := tx.GetContext(ctx, &opp, `
		SELECT * FROM tax_loss_opportunities WHERE opportunity_id = $1
	`, opportunityID)
	if err != nil {
		return nil, err
	}
	return &opp, nil
}

func (s *Service) createWashSaleTrackerRecord(ctx context.Context, tx *sqlx.Tx, opp *Opportunity) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for INSERT with date intervals
	_, err := tx.ExecContext(ctx, `
		INSERT INTO wash_sale_tracker (
account_id, ticker, sale_date, shares_sold, sale_price,
realized_loss, wash_window_start, wash_window_end
) VALUES (
$1, $2, CURRENT_DATE, $3, $4, $5,
CURRENT_DATE - INTERVAL '30 days',
CURRENT_DATE + INTERVAL '30 days'
)
	`, opp.AccountID, opp.Ticker, opp.SharesToSell, opp.CurrentPrice, opp.UnrealizedLoss)
	return err
}

func (s *Service) updateAccountYTDMetricsRecord(ctx context.Context, tx *sqlx.Tx, opp *Opportunity) error {
	// TODO: Use HasuraClient for UPDATE when available
	// For now, use SQL fallback for UPDATE with arithmetic
	_, err := tx.ExecContext(ctx, `
		UPDATE direct_index_accounts
		SET ytd_tax_loss_harvested = ytd_tax_loss_harvested + $1,
		    ytd_tax_savings = ytd_tax_savings + $2,
		    ytd_realized_losses = ytd_realized_losses + $1,
		    updated_at = NOW()
		WHERE account_id = $3
	`, -opp.UnrealizedLoss, opp.EstimatedTaxSavings, opp.AccountID)
	return err
}

func (s *Service) dismissOpportunityRecord(ctx context.Context, opportunityID uuid.UUID, reason string) error {
	// TODO: Use HasuraClient for UPDATE when available
	// For now, use SQL fallback for multi-field UPDATE
	_, err := s.db.ExecContext(ctx, `
		UPDATE tax_loss_opportunities
		SET opportunity_status = 'DISMISSED',
		    dismissal_reason = $1,
		    expired_at = NOW()
		WHERE opportunity_id = $2
	`, reason, opportunityID)
	return err
}

func (s *Service) getHoldingsCountRecord(ctx context.Context, accountID uuid.UUID) (int, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for COUNT aggregate
	var count int
	err := s.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM direct_index_holdings WHERE account_id = $1
	`, accountID)
	return count, err
}

func (s *Service) getPendingOpportunitiesStatsRecord(ctx context.Context, accountID uuid.UUID) (int, float64, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for multiple aggregates
	var count int
	var savings float64
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(estimated_tax_savings), 0)
		FROM tax_loss_opportunities
		WHERE account_id = $1 AND opportunity_status = 'PENDING'
	`, accountID).Scan(&count, &savings)
	return count, savings, err
}
