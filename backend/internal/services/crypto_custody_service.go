package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/hondyman/semlayer/backend/internal/types"

	"github.com/google/uuid"
)

// CryptoCustodyService handles integration with crypto custodians
// This is a stub implementation ready for Coinbase Prime integration
type CryptoCustodyService struct {
	db *sql.DB
	// coinbasePrimeClient *coinbase.Client - will be added when integrating
}

// NewCryptoCustodyService creates a new crypto custody service
func NewCryptoCustodyService(db *sql.DB) *CryptoCustodyService {
	return &CryptoCustodyService{
		db: db,
	}
}

// CreateWallet creates a new crypto wallet
func (s *CryptoCustodyService) CreateWallet(ctx context.Context, wallet *types.CryptoWallet) error {
	query := `
		INSERT INTO crypto_wallets (
			tenant_id, client_id, custodian, custodian_account_id,
			blockchain, address, wallet_type, label,
			whitelisted_addresses, daily_withdrawal_limit_usd
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	return s.db.QueryRowContext(ctx, query,
		wallet.TenantID, wallet.ClientID, wallet.Custodian, wallet.CustodianAccountID,
		wallet.Blockchain, wallet.Address, wallet.WalletType, wallet.Label,
		wallet.WhitelistedAddresses, wallet.DailyWithdrawalLimitUSD,
	).Scan(&wallet.ID, &wallet.CreatedAt, &wallet.UpdatedAt)
}

// GetWallet retrieves a wallet by ID
func (s *CryptoCustodyService) GetWallet(ctx context.Context, walletID uuid.UUID) (*types.CryptoWallet, error) {
	query := `
		SELECT 
			id, tenant_id, client_id, custodian, custodian_account_id,
			blockchain, address, wallet_type, label, is_active,
			whitelisted_addresses, daily_withdrawal_limit_usd,
			created_at, updated_at, deleted_at
		FROM crypto_wallets
		WHERE id = $1 AND deleted_at IS NULL
	`

	wallet := &types.CryptoWallet{}
	err := s.db.QueryRowContext(ctx, query, walletID).Scan(
		&wallet.ID, &wallet.TenantID, &wallet.ClientID, &wallet.Custodian, &wallet.CustodianAccountID,
		&wallet.Blockchain, &wallet.Address, &wallet.WalletType, &wallet.Label, &wallet.IsActive,
		&wallet.WhitelistedAddresses, &wallet.DailyWithdrawalLimitUSD,
		&wallet.CreatedAt, &wallet.UpdatedAt, &wallet.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

// GetWalletsByClient retrieves all wallets for a client
func (s *CryptoCustodyService) GetWalletsByClient(ctx context.Context, clientID uuid.UUID) ([]*types.CryptoWallet, error) {
	query := `
		SELECT 
			id, tenant_id, client_id, custodian, custodian_account_id,
			blockchain, address, wallet_type, label, is_active,
			whitelisted_addresses, daily_withdrawal_limit_usd,
			created_at, updated_at
		FROM crypto_wallets
		WHERE client_id = $1 AND deleted_at IS NULL AND is_active = TRUE
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []*types.CryptoWallet
	for rows.Next() {
		wallet := &types.CryptoWallet{}
		err := rows.Scan(
			&wallet.ID, &wallet.TenantID, &wallet.ClientID, &wallet.Custodian, &wallet.CustodianAccountID,
			&wallet.Blockchain, &wallet.Address, &wallet.WalletType, &wallet.Label, &wallet.IsActive,
			&wallet.WhitelistedAddresses, &wallet.DailyWithdrawalLimitUSD,
			&wallet.CreatedAt, &wallet.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, wallet)
	}

	return wallets, rows.Err()
}

// GetBalances retrieves current balances for a wallet
func (s *CryptoCustodyService) GetBalances(ctx context.Context, walletID uuid.UUID) ([]types.CryptoHolding, error) {
	query := `
		SELECT 
			id, wallet_id, asset_symbol, asset_name, asset_type,
			contract_address, decimals, quantity, available_quantity,
			cost_basis_total, average_cost_per_unit, last_updated
		FROM crypto_holdings
		WHERE wallet_id = $1 AND quantity > 0
		ORDER BY asset_symbol
	`

	rows, err := s.db.QueryContext(ctx, query, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []types.CryptoHolding
	for rows.Next() {
		var h types.CryptoHolding
		err := rows.Scan(
			&h.ID, &h.WalletID, &h.AssetSymbol, &h.AssetName, &h.AssetType,
			&h.ContractAddress, &h.Decimals, &h.Quantity, &h.AvailableQuantity,
			&h.CostBasisTotal, &h.AverageCostPerUnit, &h.LastUpdated,
		)
		if err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}

	return holdings, rows.Err()
}

// SubmitTransaction submits a crypto transaction
func (s *CryptoCustodyService) SubmitTransaction(ctx context.Context, txn *types.CryptoTransaction) error {
	// For now, auto-approve in development mode
	env := os.Getenv("ENVIRONMENT")
	if env == "production" || env == "prod" {
		return fmt.Errorf("auto-approval is disabled in production")
	}
	fmt.Printf("[CryptoCustody] Withdrawal request %s approved automatically (development mode)\n", txn.ID) // Assuming txn.ID can be used as withdrawalID

	// In production, this would:
	// 1. Call custodian API (Coinbase Prime, Fireblocks, etc.)
	// 2. Submit withdrawal for approval workflow
	// 3. Track approval status
	// 4. Return approval confirmation

	query := `
		INSERT INTO crypto_transactions (
			wallet_id, blockchain, txn_type, asset_symbol, contract_address,
			quantity, fiat_value_usd, price_per_unit_usd,
			fee_asset_symbol, fee_quantity, fee_fiat_value_usd,
			from_address, to_address, status, is_taxable, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		RETURNING id, created_at, updated_at
	`

	return s.db.QueryRowContext(ctx, query,
		txn.WalletID, txn.Blockchain, txn.TxnType, txn.AssetSymbol, txn.ContractAddress,
		txn.Quantity, txn.FiatValueUSD, txn.PricePerUnitUSD,
		txn.FeeAssetSymbol, txn.FeeQuantity, txn.FeeFiatValueUSD,
		txn.FromAddress, txn.ToAddress, txn.Status, txn.IsTaxable, txn.Notes,
	).Scan(&txn.ID, &txn.CreatedAt, &txn.UpdatedAt)
}

// ConfirmTransaction marks a transaction as confirmed
func (s *CryptoCustodyService) ConfirmTransaction(ctx context.Context, txnID uuid.UUID, txnHash string, blockNumber int64, blockTimestamp time.Time) error {
	query := `
		UPDATE crypto_transactions
		SET 
			status = 'CONFIRMED',
			txn_hash = $1,
			block_number = $2,
			block_timestamp = $3,
			confirmations = 1,
			updated_at = NOW()
		WHERE id = $4
	`

	result, err := s.db.ExecContext(ctx, query, txnHash, blockNumber, blockTimestamp, txnID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	// Note: Trigger will automatically update holdings

	return nil
}

// GetTransactions retrieves transactions for a wallet
func (s *CryptoCustodyService) GetTransactions(ctx context.Context, walletID uuid.UUID, limit int) ([]types.CryptoTransaction, error) {
	query := `
		SELECT 
			id, wallet_id, blockchain, txn_hash, block_number, block_timestamp,
			txn_type, asset_symbol, contract_address, quantity,
			fiat_value_usd, price_per_unit_usd,
			fee_asset_symbol, fee_quantity, fee_fiat_value_usd,
			from_address, to_address, status, confirmations,
			is_taxable, tax_lot_method, notes, external_id,
			created_at, updated_at
		FROM crypto_transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, walletID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []types.CryptoTransaction
	for rows.Next() {
		var txn types.CryptoTransaction
		err := rows.Scan(
			&txn.ID, &txn.WalletID, &txn.Blockchain, &txn.TxnHash, &txn.BlockNumber, &txn.BlockTimestamp,
			&txn.TxnType, &txn.AssetSymbol, &txn.ContractAddress, &txn.Quantity,
			&txn.FiatValueUSD, &txn.PricePerUnitUSD,
			&txn.FeeAssetSymbol, &txn.FeeQuantity, &txn.FeeFiatValueUSD,
			&txn.FromAddress, &txn.ToAddress, &txn.Status, &txn.Confirmations,
			&txn.IsTaxable, &txn.TaxLotMethod, &txn.Notes, &txn.ExternalID,
			&txn.CreatedAt, &txn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, txn)
	}

	return transactions, rows.Err()
}

// SyncTransactionsFromCustodian syncs transactions from custodian API
func (s *CryptoCustodyService) SyncTransactionsFromCustodian(ctx context.Context, walletID uuid.UUID, since time.Time) error {
	// Stub implementation
	// In production, this would:
	// 1. Call Coinbase Prime API to get transactions since `since` timestamp
	// 2. For each transaction from API, check if exists in our DB by external_id
	// 3. If not exists, insert new transaction
	// 4. Update transaction status if changed

	return nil
}

// GetTransactionStatus retrieves transaction status from blockchain
func (s *CryptoCustodyService) GetTransactionStatus(ctx context.Context, blockchain string, txnHash string) (string, int, error) {
	// For now, return mock transaction data
	fmt.Printf("[CryptoCustody] Fetching transaction %s from blockchain explorer (mock)\n", txnHash)

	// In production, would call Etherscan, Blockchair, or similar:
	// GET /api/v1/tx/{txHash}

	// Stub implementation
	// In production, this would call:
	// - Etherscan API for Ethereum
	// - Blockchain.info for Bitcoin
	// - Solscan for Solana

	return "CONFIRMED", 12, nil
}

// ReconcileBalances reconciles database balances with custodian balances
func (s *CryptoCustodyService) ReconcileBalances(ctx context.Context, walletID uuid.UUID, asset string, address string) error {
	// Implement reconciliation logic
	// Compare custodian balance with blockchain balance

	// Get custodian balance (mock for now)
	fmt.Printf("[CryptoCustody] Fetching custodian balance for wallet %s, asset %s\n", walletID, asset)
	custodianBalance := "2.0" // Mock value

	// Get blockchain balance (mock for now)
	fmt.Printf("[CryptoCustody] Fetching blockchain balance for address %s\n", address)
	blockchainBalance := "2.0" // Mock value

	// Compare and log discrepancies
	if custodianBalance != blockchainBalance {
		fmt.Printf("[WARNING] Balance mismatch for %s: Custodian=%s, Blockchain=%s\n",
			asset, custodianBalance, blockchainBalance)
		// In production: Create alert, update reconciliation table
		// _, err := s.db.ExecContext(ctx, `INSERT INTO reconciliation_discrepancies ...`)
	} else {
		fmt.Printf("[CryptoCustody] Balances match for %s: %s\n", asset, custodianBalance)
	}

	return nil
}
