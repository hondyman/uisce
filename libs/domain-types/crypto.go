package types

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// CRYPTO WALLETS
// ============================================================================

// CryptoWallet represents a blockchain wallet/account
type CryptoWallet struct {
	ID       uuid.UUID `json:"id" db:"id"`
	TenantID uuid.UUID `json:"tenantId" db:"tenant_id"`
	ClientID uuid.UUID `json:"clientId" db:"client_id"`

	// Custodian
	Custodian          string  `json:"custodian" db:"custodian"`
	CustodianAccountID *string `json:"custodianAccountId,omitempty" db:"custodian_account_id"`

	// Blockchain
	Blockchain string `json:"blockchain" db:"blockchain"`
	Address    string `json:"address" db:"address"`

	// Security
	WalletType              string   `json:"walletType" db:"wallet_type"`
	Label                   *string  `json:"label,omitempty" db:"label"`
	IsActive                bool     `json:"isActive" db:"is_active"`
	WhitelistedAddresses    []string `json:"whitelistedAddresses,omitempty" db:"whitelisted_addresses"`
	DailyWithdrawalLimitUSD *float64 `json:"dailyWithdrawalLimitUsd,omitempty" db:"daily_withdrawal_limit_usd"`

	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// ============================================================================
// CRYPTO HOLDINGS
// ============================================================================

// CryptoHolding represents a current balance of a crypto asset
type CryptoHolding struct {
	ID       uuid.UUID `json:"id" db:"id"`
	WalletID uuid.UUID `json:"walletId" db:"wallet_id"`

	// Asset
	AssetSymbol     string  `json:"assetSymbol" db:"asset_symbol"`
	AssetName       *string `json:"assetName,omitempty" db:"asset_name"`
	AssetType       string  `json:"assetType" db:"asset_type"`
	ContractAddress *string `json:"contractAddress,omitempty" db:"contract_address"`
	Decimals        int     `json:"decimals" db:"decimals"`

	// Balances
	Quantity          float64 `json:"quantity" db:"quantity"`
	AvailableQuantity float64 `json:"availableQuantity" db:"available_quantity"`

	// Valuation
	CostBasisTotal     float64 `json:"costBasisTotal" db:"cost_basis_total"`
	AverageCostPerUnit float64 `json:"averageCostPerUnit" db:"average_cost_per_unit"`

	LastUpdated time.Time `json:"lastUpdated" db:"last_updated"`
}

// ============================================================================
// CRYPTO TRANSACTIONS
// ============================================================================

// CryptoTransaction represents a blockchain transaction
type CryptoTransaction struct {
	ID       uuid.UUID `json:"id" db:"id"`
	WalletID uuid.UUID `json:"walletId" db:"wallet_id"`

	// Blockchain
	Blockchain     string     `json:"blockchain" db:"blockchain"`
	TxnHash        *string    `json:"txnHash,omitempty" db:"txn_hash"`
	BlockNumber    *int64     `json:"blockNumber,omitempty" db:"block_number"`
	BlockTimestamp *time.Time `json:"blockTimestamp,omitempty" db:"block_timestamp"`

	// Type
	TxnType string `json:"txnType" db:"txn_type"`

	// Asset
	AssetSymbol     string  `json:"assetSymbol" db:"asset_symbol"`
	ContractAddress *string `json:"contractAddress,omitempty" db:"contract_address"`
	Quantity        float64 `json:"quantity" db:"quantity"`

	// Valuation
	FiatValueUSD    *float64 `json:"fiatValueUsd,omitempty" db:"fiat_value_usd"`
	PricePerUnitUSD *float64 `json:"pricePerUnitUsd,omitempty" db:"price_per_unit_usd"`

	// Fees
	FeeAssetSymbol  *string `json:"feeAssetSymbol,omitempty" db:"fee_asset_symbol"`
	FeeQuantity     float64 `json:"feeQuantity" db:"fee_quantity"`
	FeeFiatValueUSD float64 `json:"feeFiatValueUsd" db:"fee_fiat_value_usd"`

	// Addresses
	FromAddress *string `json:"fromAddress,omitempty" db:"from_address"`
	ToAddress   *string `json:"toAddress,omitempty" db:"to_address"`

	// Status
	Status        string `json:"status" db:"status"`
	Confirmations int    `json:"confirmations" db:"confirmations"`

	// Tax
	IsTaxable    bool    `json:"isTaxable" db:"is_taxable"`
	TaxLotMethod *string `json:"taxLotMethod,omitempty" db:"tax_lot_method"`

	// Metadata
	Notes      *string `json:"notes,omitempty" db:"notes"`
	ExternalID *string `json:"externalId,omitempty" db:"external_id"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// ============================================================================
// TAX LOTS
// ============================================================================

// CryptoTaxLot represents a tax lot for cost basis tracking
type CryptoTaxLot struct {
	ID          uuid.UUID `json:"id" db:"id"`
	WalletID    uuid.UUID `json:"walletId" db:"wallet_id"`
	AssetSymbol string    `json:"assetSymbol" db:"asset_symbol"`

	// Acquisition
	AcquisitionTxnID *uuid.UUID `json:"acquisitionTxnId,omitempty" db:"acquisition_txn_id"`
	AcquisitionDate  time.Time  `json:"acquisitionDate" db:"acquisition_date"`
	AcquisitionType  *string    `json:"acquisitionType,omitempty" db:"acquisition_type"`
	QuantityAcquired float64    `json:"quantityAcquired" db:"quantity_acquired"`
	CostBasisPerUnit float64    `json:"costBasisPerUnit" db:"cost_basis_per_unit"`
	TotalCostBasis   float64    `json:"totalCostBasis" db:"total_cost_basis"`

	// Disposal
	DisposalTxnID    *uuid.UUID `json:"disposalTxnId,omitempty" db:"disposal_txn_id"`
	DisposalDate     *time.Time `json:"disposalDate,omitempty" db:"disposal_date"`
	DisposalType     *string    `json:"disposalType,omitempty" db:"disposal_type"`
	QuantityDisposed float64    `json:"quantityDisposed" db:"quantity_disposed"`
	DisposalProceeds float64    `json:"disposalProceeds" db:"disposal_proceeds"`

	// Remaining
	QuantityRemaining float64 `json:"quantityRemaining" db:"quantity_remaining"`
	IsFullyDisposed   bool    `json:"isFullyDisposed" db:"is_fully_disposed"`

	// Tax
	HoldingPeriodDays *int     `json:"holdingPeriodDays,omitempty" db:"holding_period_days"`
	IsLongTerm        *bool    `json:"isLongTerm,omitempty" db:"is_long_term"`
	RealizedGainLoss  *float64 `json:"realizedGainLoss,omitempty" db:"realized_gain_loss"`

	// Wash sale
	IsWashSale             bool       `json:"isWashSale" db:"is_wash_sale"`
	WashSaleDisallowedLoss float64    `json:"washSaleDisallowedLoss" db:"wash_sale_disallowed_loss"`
	LinkedWashSaleLotID    *uuid.UUID `json:"linkedWashSaleLotId,omitempty" db:"linked_wash_sale_lot_id"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// ============================================================================
// PRICES
// ============================================================================

// CryptoPrice represents a price snapshot for an asset
type CryptoPrice struct {
	ID          uuid.UUID `json:"id" db:"id"`
	AssetSymbol string    `json:"assetSymbol" db:"asset_symbol"`

	// Price
	PriceUSD     float64  `json:"priceUsd" db:"price_usd"`
	MarketCapUSD *float64 `json:"marketCapUsd,omitempty" db:"market_cap_usd"`
	Volume24hUSD *float64 `json:"volume24hUsd,omitempty" db:"volume_24h_usd"`

	// Changes
	Change1hPct  *float64 `json:"change1hPct,omitempty" db:"change_1h_pct"`
	Change24hPct *float64 `json:"change24hPct,omitempty" db:"change_24h_pct"`
	Change7dPct  *float64 `json:"change7dPct,omitempty" db:"change_7d_pct"`

	// High/Low
	High24hUSD *float64 `json:"high24hUsd,omitempty" db:"high_24h_usd"`
	Low24hUSD  *float64 `json:"low24hUsd,omitempty" db:"low_24h_usd"`

	Source    string    `json:"source" db:"source"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

// ============================================================================
// DEFI POSITIONS
// ============================================================================

// DeFiPosition represents a DeFi protocol position
type DeFiPosition struct {
	ID       uuid.UUID `json:"id" db:"id"`
	WalletID uuid.UUID `json:"walletId" db:"wallet_id"`

	// Protocol
	Protocol        string  `json:"protocol" db:"protocol"`
	ProtocolVersion *string `json:"protocolVersion,omitempty" db:"protocol_version"`
	Blockchain      string  `json:"blockchain" db:"blockchain"`
	ContractAddress *string `json:"contractAddress,omitempty" db:"contract_address"`

	// Type
	PositionType string `json:"positionType" db:"position_type"`

	// Deposited
	AssetDeposited    string    `json:"assetDeposited" db:"asset_deposited"`
	QuantityDeposited float64   `json:"quantityDeposited" db:"quantity_deposited"`
	DepositValueUSD   *float64  `json:"depositValueUsd,omitempty" db:"deposit_value_usd"`
	DepositDate       time.Time `json:"depositDate" db:"deposit_date"`

	// Borrowed (if applicable)
	AssetBorrowed    *string  `json:"assetBorrowed,omitempty" db:"asset_borrowed"`
	QuantityBorrowed *float64 `json:"quantityBorrowed,omitempty" db:"quantity_borrowed"`

	// Current
	CurrentValueUSD *float64 `json:"currentValueUsd,omitempty" db:"current_value_usd"`

	// Rewards
	RewardAssetSymbol   *string  `json:"rewardAssetSymbol,omitempty" db:"reward_asset_symbol"`
	RewardsEarned       float64  `json:"rewardsEarned" db:"rewards_earned"`
	RewardsClaimed      float64  `json:"rewardsClaimed" db:"rewards_claimed"`
	UnclaimedRewardsUSD *float64 `json:"unclaimedRewardsUsd,omitempty" db:"unclaimed_rewards_usd"`

	// Yield
	APR *float64 `json:"apr,omitempty" db:"apr"`
	APY *float64 `json:"apy,omitempty" db:"apy"`

	// Status
	IsActive   bool       `json:"isActive" db:"is_active"`
	ClosedDate *time.Time `json:"closedDate,omitempty" db:"closed_date"`

	LastUpdated time.Time `json:"lastUpdated" db:"last_updated"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

// ============================================================================
// HELPER TYPES
// ============================================================================

// TaxLotDisposal represents the result of disposing crypto with tax lot tracking
type TaxLotDisposal struct {
	DisposedLots      []CryptoTaxLot `json:"disposedLots"`
	TotalGainLoss     float64        `json:"totalGainLoss"`
	ShortTermGainLoss float64        `json:"shortTermGainLoss"`
	LongTermGainLoss  float64        `json:"longTermGainLoss"`
}

// GainLossReport represents capital gains/losses for a tax year
type GainLossReport struct {
	TaxYear          int     `json:"taxYear"`
	TotalShortTerm   float64 `json:"totalShortTerm"`
	TotalLongTerm    float64 `json:"totalLongTerm"`
	TotalNet         float64 `json:"totalNet"`
	TransactionCount int     `json:"transactionCount"`
}

// Form8949Entry represents a single line on IRS Form 8949
type Form8949Entry struct {
	AssetSymbol        string    `json:"assetSymbol"`
	DateAcquired       time.Time `json:"dateAcquired"`
	DateSold           time.Time `json:"dateSold"`
	Quantity           float64   `json:"quantity"`
	Proceeds           float64   `json:"proceeds"`
	CostBasis          float64   `json:"costBasis"`
	GainLoss           float64   `json:"gainLoss"`
	IsLongTerm         bool      `json:"isLongTerm"`
	WashSaleAdjustment float64   `json:"washSaleAdjustment"`
}

// WashSale represents detected wash sale
type WashSale struct {
	SaleLotID      uuid.UUID `json:"saleLotId"`
	PurchaseLotID  uuid.UUID `json:"purchaseLotId"`
	AssetSymbol    string    `json:"assetSymbol"`
	SaleDate       time.Time `json:"saleDate"`
	PurchaseDate   time.Time `json:"purchaseDate"`
	DisallowedLoss float64   `json:"disallowedLoss"`
	DaysBetween    int       `json:"daysBetween"`
}
