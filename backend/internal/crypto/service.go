package crypto

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

type AssetSymbol string

type RecordTransactionInput struct {
	ClientID        uuid.UUID
	TransactionType string
	AssetSymbol     string
	Quantity        decimal.Decimal
	PricePerUnit    decimal.Decimal
	Fee             decimal.Decimal
	TransactionDate time.Time
	Blockchain      string
	TxHash          string
}

type Service interface {
	GetClientHoldings(ctx context.Context, clientID uuid.UUID) ([]*Holding, error)
	GetHolding(ctx context.Context, holdingID uuid.UUID) (*Holding, error)
	RecordTransaction(ctx context.Context, input RecordTransactionInput) (*Transaction, error)
	GetClientTransactions(ctx context.Context, clientID uuid.UUID, limit int) ([]*Transaction, error)
	GetLatestPrice(ctx context.Context, symbol AssetSymbol) (decimal.Decimal, error)
	GetPriceHistory(ctx context.Context, symbol AssetSymbol, hours int) ([]*MarketTick, error)
	GetClientPortfolioSummary(ctx context.Context, clientID uuid.UUID) (*PortfolioSummary, error)
	GetAllocationPercentage(ctx context.Context, clientID uuid.UUID) (decimal.Decimal, error)
	IdentifyTaxLossOpportunities(ctx context.Context, clientID uuid.UUID, minLoss decimal.Decimal) ([]*TaxLossOpportunity, error)
}

type service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) Service {
	return &service{db: db}
}

type Holding struct {
	HoldingID          uuid.UUID       `db:"holding_id" json:"holdingId"`
	ClientID           uuid.UUID       `db:"client_id" json:"clientId"`
	AssetSymbol        string          `db:"asset_symbol" json:"assetSymbol"`
	Quantity           decimal.Decimal `db:"quantity" json:"quantity"`
	TotalCostBasis     decimal.Decimal `db:"total_cost_basis" json:"totalCostBasis"`
	AvgCostPerUnit     decimal.Decimal `db:"average_cost_per_unit" json:"avgCostPerUnit"`
	CurrentPriceUSD    decimal.Decimal `db:"current_price_usd" json:"currentPriceUsd"`
	CurrentValueUSD    decimal.Decimal `db:"current_value_usd" json:"currentValueUsd"`
	UnrealizedGainLoss decimal.Decimal `db:"unrealized_gain_loss" json:"unrealizedGainLoss"`
	CreatedAt          time.Time       `db:"created_at" json:"createdAt"`
}

type Transaction struct {
	TransactionID   uuid.UUID       `db:"transaction_id" json:"transactionId"`
	ClientID        uuid.UUID       `db:"client_id" json:"clientId"`
	TransactionType string          `db:"transaction_type" json:"transactionType"`
	TransactionDate time.Time       `db:"transaction_date" json:"transactionDate"`
	AssetSymbol     string          `db:"asset_symbol" json:"assetSymbol"`
	Quantity        decimal.Decimal `db:"quantity" json:"quantity"`
	PricePerUnitUSD decimal.Decimal `db:"price_per_unit_usd" json:"pricePerUnitUsd"`
	TotalValueUSD   decimal.Decimal `db:"total_value_usd" json:"totalValueUsd"`
	FeeUSD          decimal.Decimal `db:"fee_usd" json:"feeUsd"`
	CreatedAt       time.Time       `db:"created_at" json:"createdAt"`
}

type MarketTick struct {
	AssetSymbol  string          `db:"asset_symbol" json:"assetSymbol"`
	PriceUSD     decimal.Decimal `db:"price_usd" json:"priceUsd"`
	TimestampUTC time.Time       `db:"timestamp_utc" json:"timestampUtc"`
}

type PortfolioSummary struct {
	ClientID            uuid.UUID       `json:"clientId"`
	TotalCryptoValueUSD decimal.Decimal `json:"totalCryptoValueUsd"`
	AllocationPct       decimal.Decimal `json:"allocationPct"`
	UnrealizedGainLoss  decimal.Decimal `json:"unrealizedGainLoss"`
	TotalCostBasis      decimal.Decimal `json:"totalCostBasis"`
	UniqueAssets        int             `json:"uniqueAssets"`
}

type TaxLossOpportunity struct {
	HoldingID             uuid.UUID       `json:"holdingId"`
	AssetSymbol           string          `json:"assetSymbol"`
	Quantity              decimal.Decimal `json:"quantity"`
	UnrealizedLoss        decimal.Decimal `json:"unrealizedLoss"`
	EstimatedTaxSavings   decimal.Decimal `json:"estimatedTaxSavings"`
	ReplacementSuggestion string          `json:"replacementSuggestion"`
}

func (s *service) GetClientHoldings(ctx context.Context, clientID uuid.UUID) ([]*Holding, error) {
	var holdings []*Holding
	query := `
		SELECT holding_id, client_id, asset_symbol, quantity, total_cost_basis,
		       average_cost_per_unit, current_price_usd, current_value_usd,
		       unrealized_gain_loss, created_at
		FROM crypto_holdings
		WHERE client_id = $1
		ORDER BY current_value_usd DESC
	`
	err := s.db.SelectContext(ctx, &holdings, query, clientID)
	return holdings, err
}

func (s *service) GetHolding(ctx context.Context, holdingID uuid.UUID) (*Holding, error) {
	var holding Holding
	query := `
		SELECT holding_id, client_id, asset_symbol, quantity, total_cost_basis,
		       average_cost_per_unit, current_price_usd, current_value_usd,
		       unrealized_gain_loss, created_at
		FROM crypto_holdings
		WHERE holding_id = $1
	`
	err := s.db.GetContext(ctx, &holding, query, holdingID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("holding not found")
	}
	return &holding, err
}

func (s *service) RecordTransaction(ctx context.Context, input RecordTransactionInput) (*Transaction, error) {
	txn := &Transaction{
		TransactionID:   uuid.New(),
		ClientID:        input.ClientID,
		TransactionType: input.TransactionType,
		TransactionDate: input.TransactionDate,
		AssetSymbol:     input.AssetSymbol,
		Quantity:        input.Quantity,
		PricePerUnitUSD: input.PricePerUnit,
		TotalValueUSD:   input.Quantity.Mul(input.PricePerUnit),
		FeeUSD:          input.Fee,
		CreatedAt:       time.Now(),
	}

	query := `
		INSERT INTO crypto_transactions (
			transaction_id, client_id, transaction_type, transaction_date,
			asset_symbol, quantity, price_per_unit_usd, total_value_usd,
			fee_usd, created_at
		) VALUES (
			:transaction_id, :client_id, :transaction_type, :transaction_date,
			:asset_symbol, :quantity, :price_per_unit_usd, :total_value_usd,
			:fee_usd, :created_at
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, txn)
	if err != nil {
		return nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	// Update holdings after transaction
	if err := s.updateHoldings(ctx, input.ClientID, input.AssetSymbol); err != nil {
		// Log error but don't fail transaction
		fmt.Printf("Warning: failed to update holdings: %v\n", err)
	}

	return txn, nil
}

func (s *service) updateHoldings(ctx context.Context, clientID uuid.UUID, assetSymbol string) error {
	// Recalculate holdings based on transactions
	query := `
		INSERT INTO crypto_holdings (
			holding_id, client_id, asset_symbol, quantity, total_cost_basis,
			average_cost_per_unit, current_price_usd, current_value_usd,
			unrealized_gain_loss, created_at
		)
		SELECT
			gen_random_uuid(),
			$1,
			$2,
			COALESCE(SUM(CASE 
				WHEN transaction_type IN ('BUY', 'RECEIVE') THEN quantity
				WHEN transaction_type IN ('SELL', 'SEND') THEN -quantity
				ELSE 0
			END), 0),
			COALESCE(SUM(CASE
				WHEN transaction_type IN ('BUY', 'RECEIVE') THEN total_value_usd
				WHEN transaction_type IN ('SELL', 'SEND') THEN -total_value_usd
				ELSE 0
			END), 0),
			CASE
				WHEN SUM(CASE WHEN transaction_type IN ('BUY', 'RECEIVE') THEN quantity ELSE 0 END) > 0
				THEN SUM(CASE WHEN transaction_type IN ('BUY', 'RECEIVE') THEN total_value_usd ELSE 0 END) /
				     SUM(CASE WHEN transaction_type IN ('BUY', 'RECEIVE') THEN quantity ELSE 0 END)
				ELSE 0
			END,
			0, 0, 0,
			NOW()
		FROM crypto_transactions
		WHERE client_id = $1 AND asset_symbol = $2
		ON CONFLICT (client_id, asset_symbol)
		DO UPDATE SET
			quantity = EXCLUDED.quantity,
			total_cost_basis = EXCLUDED.total_cost_basis,
			average_cost_per_unit = EXCLUDED.average_cost_per_unit
	`
	_, err := s.db.ExecContext(ctx, query, clientID, assetSymbol)
	return err
}

func (s *service) GetClientTransactions(ctx context.Context, clientID uuid.UUID, limit int) ([]*Transaction, error) {
	var transactions []*Transaction
	query := `
		SELECT transaction_id, client_id, transaction_type, transaction_date,
		       asset_symbol, quantity, price_per_unit_usd, total_value_usd,
		       fee_usd, created_at
		FROM crypto_transactions
		WHERE client_id = $1
		ORDER BY transaction_date DESC
		LIMIT $2
	`
	err := s.db.SelectContext(ctx, &transactions, query, clientID, limit)
	return transactions, err
}

func (s *service) GetLatestPrice(ctx context.Context, symbol AssetSymbol) (decimal.Decimal, error) {
	var price decimal.Decimal
	query := `SELECT vwap_price FROM crypto_vwap_prices WHERE asset_symbol = $1`
	err := s.db.GetContext(ctx, &price, query, string(symbol))
	if err == sql.ErrNoRows {
		return decimal.Zero, fmt.Errorf("price not found for %s", symbol)
	}
	return price, err
}

func (s *service) GetPriceHistory(ctx context.Context, symbol AssetSymbol, hours int) ([]*MarketTick, error) {
	var ticks []*MarketTick
	query := `
		SELECT asset_symbol, price_usd, timestamp_utc
		FROM crypto_market_data
		WHERE asset_symbol = $1
		  AND timestamp_utc >= NOW() - INTERVAL '1 hour' * $2
		ORDER BY timestamp_utc DESC
	`
	err := s.db.SelectContext(ctx, &ticks, query, string(symbol), hours)
	return ticks, err
}

func (s *service) GetClientPortfolioSummary(ctx context.Context, clientID uuid.UUID) (*PortfolioSummary, error) {
	var summary PortfolioSummary
	summary.ClientID = clientID

	query := `
		SELECT
			COALESCE(SUM(current_value_usd), 0) as total_crypto_value_usd,
			COALESCE(SUM(unrealized_gain_loss), 0) as unrealized_gain_loss,
			COALESCE(SUM(total_cost_basis), 0) as total_cost_basis,
			COUNT(DISTINCT asset_symbol) as unique_assets
		FROM crypto_holdings
		WHERE client_id = $1
	`

	err := s.db.GetContext(ctx, &summary, query, clientID)
	if err != nil {
		return nil, err
	}

	// Get allocation percentage
	alloc, err := s.GetAllocationPercentage(ctx, clientID)
	if err == nil {
		summary.AllocationPct = alloc
	}

	return &summary, nil
}

func (s *service) GetAllocationPercentage(ctx context.Context, clientID uuid.UUID) (decimal.Decimal, error) {
	var pct decimal.Decimal
	query := `SELECT COALESCE(get_crypto_allocation_pct($1), 0)`
	err := s.db.GetContext(ctx, &pct, query, clientID)
	if err != nil {
		// If function doesn't exist or error, return 0
		return decimal.Zero, nil
	}
	return pct, nil
}

func (s *service) IdentifyTaxLossOpportunities(ctx context.Context, clientID uuid.UUID, minLoss decimal.Decimal) ([]*TaxLossOpportunity, error) {
	var opportunities []*TaxLossOpportunity

	query := `
		SELECT
			holding_id,
			asset_symbol,
			quantity,
			ABS(unrealized_gain_loss) as unrealized_loss
		FROM crypto_holdings
		WHERE client_id = $1
		  AND unrealized_gain_loss < -$2
		ORDER BY unrealized_gain_loss ASC
	`

	err := s.db.SelectContext(ctx, &opportunities, query, clientID, minLoss)
	if err != nil {
		return nil, err
	}

	// Calculate tax savings and suggest replacements
	taxRate := decimal.NewFromFloat(0.37)
	replacements := map[string]string{
		"BTC":  "ETH",
		"ETH":  "BTC",
		"SOL":  "AVAX",
		"AVAX": "SOL",
		"USDC": "USDT",
		"USDT": "USDC",
	}

	for _, opp := range opportunities {
		opp.EstimatedTaxSavings = opp.UnrealizedLoss.Mul(taxRate)
		if replacement, ok := replacements[opp.AssetSymbol]; ok {
			opp.ReplacementSuggestion = replacement
		} else {
			opp.ReplacementSuggestion = "Similar asset to maintain exposure"
		}
	}

	return opportunities, nil
}
