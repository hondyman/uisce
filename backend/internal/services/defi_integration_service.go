package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/types"

	"github.com/google/uuid"
)

// DeFiIntegrationService handles DeFi protocol position tracking
type DeFiIntegrationService struct {
	db *sql.DB
	// ethClient *ethclient.Client - will be added for on-chain queries
}

// NewDeFiIntegrationService creates a new DeFi integration service
func NewDeFiIntegrationService(db *sql.DB) *DeFiIntegrationService {
	return &DeFiIntegrationService{
		db: db,
	}
}

// RecordPosition creates or updates a DeFi position
func (s *DeFiIntegrationService) RecordPosition(ctx context.Context, pos *types.DeFiPosition) error {
	query := `
		INSERT INTO defi_positions (
			wallet_id, protocol, protocol_version, blockchain, contract_address,
			position_type, asset_deposited, quantity_deposited, deposit_value_usd,
			deposit_date, asset_borrowed, quantity_borrowed, current_value_usd,
			reward_asset_symbol, rewards_earned, rewards_claimed, unclaimed_rewards_usd,
			apr, apy
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19
		)
		ON CONFLICT (id) DO UPDATE SET
			quantity_deposited = EXCLUDED.quantity_deposited,
			current_value_usd = EXCLUDED.current_value_usd,
			rewards_earned = EXCLUDED.rewards_earned,
			rewards_claimed = EXCLUDED.rewards_claimed,
			unclaimed_rewards_usd = EXCLUDED.unclaimed_rewards_usd,
			apr = EXCLUDED.apr,
			apy = EXCLUDED.apy,
			last_updated = NOW()
		RETURNING id, created_at, last_updated
	`

	return s.db.QueryRowContext(ctx, query,
		pos.WalletID, pos.Protocol, pos.ProtocolVersion, pos.Blockchain, pos.ContractAddress,
		pos.PositionType, pos.AssetDeposited, pos.QuantityDeposited, pos.DepositValueUSD,
		pos.DepositDate, pos.AssetBorrowed, pos.QuantityBorrowed, pos.CurrentValueUSD,
		pos.RewardAssetSymbol, pos.RewardsEarned, pos.RewardsClaimed, pos.UnclaimedRewardsUSD,
		pos.APR, pos.APY,
	).Scan(&pos.ID, &pos.CreatedAt, &pos.LastUpdated)
}

// GetPositionsByWallet retrieves all DeFi positions for a wallet
func (s *DeFiIntegrationService) GetPositionsByWallet(ctx context.Context, walletID uuid.UUID) ([]types.DeFiPosition, error) {
	query := `
		SELECT 
			id, wallet_id, protocol, protocol_version, blockchain, contract_address,
			position_type, asset_deposited, quantity_deposited, deposit_value_usd,
			deposit_date, asset_borrowed, quantity_borrowed, current_value_usd,
			reward_asset_symbol, rewards_earned, rewards_claimed, unclaimed_rewards_usd,
			apr, apy, is_active, closed_date, last_updated, created_at
		FROM defi_positions
		WHERE wallet_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []types.DeFiPosition
	for rows.Next() {
		var pos types.DeFiPosition
		err := rows.Scan(
			&pos.ID, &pos.WalletID, &pos.Protocol, &pos.ProtocolVersion, &pos.Blockchain, &pos.ContractAddress,
			&pos.PositionType, &pos.AssetDeposited, &pos.QuantityDeposited, &pos.DepositValueUSD,
			&pos.DepositDate, &pos.AssetBorrowed, &pos.QuantityBorrowed, &pos.CurrentValueUSD,
			&pos.RewardAssetSymbol, &pos.RewardsEarned, &pos.RewardsClaimed, &pos.UnclaimedRewardsUSD,
			&pos.APR, &pos.APY, &pos.IsActive, &pos.ClosedDate, &pos.LastUpdated, &pos.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, rows.Err()
}

// ClosePosition marks a DeFi position as closed
func (s *DeFiIntegrationService) ClosePosition(ctx context.Context, positionID uuid.UUID) error {
	query := `
		UPDATE defi_positions
		SET 
			is_active = FALSE,
			closed_date = NOW()
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, positionID)
	return err
}

// SyncAavePosition syncs an Aave lending/borrowing position
// TODO: Implement actual on-chain data fetching via eth RPC
func (s *DeFiIntegrationService) SyncAavePosition(ctx context.Context, walletID uuid.UUID, walletAddress string) error {
	// For now, log the sync operation
	fmt.Printf("[DeFi] Fetching Aave position for wallet %s (mock)\n", walletAddress)

	// In production, this would:
	// 1. Call Aave smart contracts to get user's deposits and borrows
	// 2. Calculate current APY from contract data
	// 3. Get accrued rewards
	// 4. Update or create position in database via RecordPosition

	return nil
}

// SyncUniswapLP syncs Uniswap liquidity pool position
// TODO: Implement actual on-chain data fetching
func (s *DeFiIntegrationService) SyncUniswapLP(ctx context.Context, walletID uuid.UUID, walletAddress string) error {
	// For now, log the sync operation
	fmt.Printf("[DeFi] Fetching Uniswap LP for wallet %s (mock)\n", walletAddress)

	// In production:
	// 1. Query Uniswap v2/v3 contracts for user's LP positions
	// 2. Calculate current value and impermanent loss
	// 3. Get accrued fees/rewards
	// 4. Update or create position via RecordPosition

	return nil
}

// CalculateTotalDeFiValue calculates total value locked in DeFi for a client
func (s *DeFiIntegrationService) CalculateTotalDeFiValue(ctx context.Context, clientID uuid.UUID) (float64, error) {
	query := `
		SELECT COALESCE(SUM(current_value_usd), 0)
		FROM defi_positions dp
		JOIN crypto_wallets cw ON dp.wallet_id = cw.id
		WHERE cw.client_id = $1 AND dp.is_active = TRUE
	`

	var total float64
	err := s.db.QueryRowContext(ctx, query, clientID).Scan(&total)
	return total, err
}

// GetRewardsEarned gets total unclaimed rewards for a client
func (s *DeFiIntegrationService) GetRewardsEarned(ctx context.Context, clientID uuid.UUID) (float64, error) {
	query := `
		SELECT COALESCE(SUM(unclaimed_rewards_usd), 0)
		FROM defi_positions dp
		JOIN crypto_wallets cw ON dp.wallet_id = cw.id
		WHERE cw.client_id = $1 AND dp.is_active = TRUE
	`

	var total float64
	err := s.db.QueryRowContext(ctx, query, clientID).Scan(&total)
	return total, err
}
