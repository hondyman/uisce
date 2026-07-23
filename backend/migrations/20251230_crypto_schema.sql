-- Migration: Create Crypto Integration Schema
-- Description: Multi-chain wallet management, transactions, tax lots, and DeFi positions
-- Author: Semlayer Platform
-- Date: 2025-11-27

-- ============================================================================
-- CRYPTO WALLETS (Custody Accounts)
-- ============================================================================

-- Drop legacy tables to ensure new schema is applied
DROP TABLE IF EXISTS crypto_holdings CASCADE;
DROP TABLE IF EXISTS crypto_transactions CASCADE;
DROP TABLE IF EXISTS crypto_tax_lots CASCADE;
DROP TABLE IF EXISTS defi_positions CASCADE;
DROP TABLE IF EXISTS crypto_latest_prices CASCADE;
DROP TABLE IF EXISTS crypto_prices CASCADE;

CREATE TABLE IF NOT EXISTS crypto_wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    
    -- Custodian information
    custodian VARCHAR(50) CHECK (custodian IN (
        'COINBASE_PRIME',
        'FIREBLOCKS',
        'ANCHORAGE',
        'BITGO',
        'FIDELITY_DIGITAL',
        'SELF_CUSTODY'
    )),
    custodian_account_id VARCHAR(255), -- External account ID from custodian
    
    -- Blockchain details
    blockchain VARCHAR(50) NOT NULL CHECK (blockchain IN (
        'BITCOIN',
        'ETHEREUM',
        'SOLANA',
        'POLYGON',
        'ARBITRUM',
        'OPTIMISM'
    )),
    address TEXT NOT NULL,
    
    -- Wallet type and security
    wallet_type VARCHAR(50) DEFAULT 'MPC' CHECK (wallet_type IN (
        'HOT',
        'COLD',
        'MPC',
        'HARDWARE'
    )),
    
    -- Metadata
    label VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    whitelisted_addresses TEXT[], -- Array of approved withdrawal addresses
    daily_withdrawal_limit_usd DECIMAL(15,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT unique_wallet_address UNIQUE(blockchain, address, tenant_id)
);

-- Indexes
CREATE INDEX idx_crypto_wallets_client ON crypto_wallets(client_id, tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_crypto_wallets_custodian ON crypto_wallets(custodian) WHERE is_active = TRUE;
CREATE INDEX idx_crypto_wallets_blockchain ON crypto_wallets(blockchain) WHERE is_active = TRUE;

-- ============================================================================
-- CRYPTO HOLDINGS (Current Balances)
-- ============================================================================

CREATE TABLE IF NOT EXISTS crypto_holdings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES crypto_wallets(id) ON DELETE CASCADE,
    
    -- Asset identification
    asset_symbol VARCHAR(20) NOT NULL, -- BTC, ETH, SOL, USDC, etc.
    asset_name VARCHAR(100),
    asset_type VARCHAR(50) DEFAULT 'NATIVE' CHECK (asset_type IN (
        'NATIVE',      -- BTC, ETH, SOL
        'ERC20',       -- Ethereum tokens
        'SPL',         -- Solana tokens
        'BEP20'        -- Binance Smart Chain tokens
    )),
    contract_address TEXT, -- For tokens (null for native assets)
    decimals INTEGER DEFAULT 18,
    
    -- Balances
    quantity DECIMAL(30,18) NOT NULL CHECK (quantity >= 0),
    available_quantity DECIMAL(30,18) NOT NULL CHECK (available_quantity >= 0), -- Excludes locked/staked
    
    -- Valuation
    cost_basis_total DECIMAL(15,2) DEFAULT 0,
    average_cost_per_unit DECIMAL(15,8) GENERATED ALWAYS AS (
        CASE 
            WHEN quantity > 0 THEN cost_basis_total / quantity 
            ELSE 0 
        END
    ) STORED,
    
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_wallet_asset UNIQUE(wallet_id, asset_symbol, contract_address)
);

-- Indexes
CREATE INDEX idx_crypto_holdings_wallet ON crypto_holdings(wallet_id);
CREATE INDEX idx_crypto_holdings_asset ON crypto_holdings(asset_symbol);

-- ============================================================================
-- CRYPTO TRANSACTIONS (All Blockchain Activity)
-- ============================================================================

CREATE TABLE IF NOT EXISTS crypto_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES crypto_wallets(id) ON DELETE CASCADE,
    
    -- Blockchain details
    blockchain VARCHAR(50) NOT NULL,
    txn_hash TEXT,
    block_number BIGINT,
    block_timestamp TIMESTAMPTZ,
    
    -- Transaction type
    txn_type VARCHAR(50) NOT NULL CHECK (txn_type IN (
        'BUY',
        'SELL',
        'TRANSFER_IN',
        'TRANSFER_OUT',
        'STAKE',
        'UNSTAKE',
        'REWARD',
        'AIRDROP',
        'SWAP',
        'FEE'
    )),
    
    -- Asset details
    asset_symbol VARCHAR(20) NOT NULL,
    contract_address TEXT,
    quantity DECIMAL(30,18) NOT NULL,
    
    -- USD valuation at time of transaction
    fiat_value_usd DECIMAL(15,2),
    price_per_unit_usd DECIMAL(15,8),
    
    -- Transaction fees
    fee_asset_symbol VARCHAR(20),
    fee_quantity DECIMAL(30,18) DEFAULT 0,
    fee_fiat_value_usd DECIMAL(15,2) DEFAULT 0,
    
    -- Addresses
    from_address TEXT,
    to_address TEXT,
    
    -- Status
    status VARCHAR(20) DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'CONFIRMED',
        'FAILED',
        'DROPPED'
    )),
    confirmations INTEGER DEFAULT 0,
    
    -- Tax tracking
    is_taxable BOOLEAN DEFAULT TRUE,
    tax_lot_method VARCHAR(20), -- FIFO, LIFO, HIFO, SPECIFIC_ID
    
    -- Metadata
    notes TEXT,
    external_id VARCHAR(255), -- ID from custodian or exchange
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_txn_hash UNIQUE(blockchain, txn_hash)
);

-- Indexes
CREATE INDEX idx_crypto_txns_wallet ON crypto_transactions(wallet_id, block_timestamp DESC);
CREATE INDEX idx_crypto_txns_hash ON crypto_transactions(txn_hash);
CREATE INDEX idx_crypto_txns_type ON crypto_transactions(txn_type, block_timestamp DESC);
CREATE INDEX idx_crypto_txns_status ON crypto_transactions(status) WHERE status = 'PENDING';
CREATE INDEX idx_crypto_txns_taxable ON crypto_transactions(wallet_id, is_taxable) WHERE is_taxable = TRUE;

-- ============================================================================
-- CRYPTO TAX LOTS (Cost Basis Tracking)
-- ============================================================================

CREATE TABLE IF NOT EXISTS crypto_tax_lots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES crypto_wallets(id) ON DELETE CASCADE,
    asset_symbol VARCHAR(20) NOT NULL,
    
    -- Acquisition
    acquisition_txn_id UUID REFERENCES crypto_transactions(id),
    acquisition_date TIMESTAMPTZ NOT NULL,
    acquisition_type VARCHAR(50), -- BUY, TRANSFER_IN, REWARD, AIRDROP
    quantity_acquired DECIMAL(30,18) NOT NULL,
    cost_basis_per_unit DECIMAL(15,8) NOT NULL,
    total_cost_basis DECIMAL(15,2) NOT NULL,
    
    -- Disposal
    disposal_txn_id UUID REFERENCES crypto_transactions(id),
    disposal_date TIMESTAMPTZ,
    disposal_type VARCHAR(50), -- SELL, TRANSFER_OUT, SWAP
    quantity_disposed DECIMAL(30,18) DEFAULT 0,
    disposal_proceeds DECIMAL(15,2) DEFAULT 0,
    
    -- Remaining
    quantity_remaining DECIMAL(30,18) GENERATED ALWAYS AS (
        quantity_acquired - quantity_disposed
    ) STORED,
    is_fully_disposed BOOLEAN GENERATED ALWAYS AS (
        quantity_acquired <= quantity_disposed
    ) STORED,
    
    -- Tax implications
    holding_period_days INTEGER,
    is_long_term BOOLEAN GENERATED ALWAYS AS (holding_period_days >= 365) STORED,
    realized_gain_loss DECIMAL(15,2),
    
    -- Wash sale tracking
    is_wash_sale BOOLEAN DEFAULT FALSE,
    wash_sale_disallowed_loss DECIMAL(15,2) DEFAULT 0,
    linked_wash_sale_lot_id UUID REFERENCES crypto_tax_lots(id),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_tax_lots_wallet_asset ON crypto_tax_lots(wallet_id, asset_symbol);
CREATE INDEX idx_tax_lots_acquisition ON crypto_tax_lots(acquisition_date);
CREATE INDEX idx_tax_lots_disposal ON crypto_tax_lots(disposal_date) WHERE disposal_date IS NOT NULL;
CREATE INDEX idx_tax_lots_remaining ON crypto_tax_lots(wallet_id, asset_symbol) 
    WHERE quantity_remaining > 0;
CREATE INDEX idx_tax_lots_wash_sale ON crypto_tax_lots(is_wash_sale) WHERE is_wash_sale = TRUE;

-- ============================================================================
-- CRYPTO PRICES (Real-Time Cache)
-- ============================================================================

CREATE TABLE IF NOT EXISTS crypto_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_symbol VARCHAR(20) NOT NULL,
    
    -- Price data
    price_usd DECIMAL(15,8) NOT NULL,
    market_cap_usd DECIMAL(20,2),
    volume_24h_usd DECIMAL(20,2),
    
    -- Change indicators
    change_1h_pct DECIMAL(8,4),
    change_24h_pct DECIMAL(8,4),
    change_7d_pct DECIMAL(8,4),
    
    -- High/Low
    high_24h_usd DECIMAL(15,8),
    low_24h_usd DECIMAL(15,8),
    
    -- Metadata
    source VARCHAR(50) DEFAULT 'COINGECKO', -- COINGECKO, COINMARKETCAP, BINANCE
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_price_point UNIQUE(asset_symbol, timestamp, source)
);

-- Indexes
CREATE INDEX idx_crypto_prices_symbol ON crypto_prices(asset_symbol, timestamp DESC);
CREATE INDEX idx_crypto_prices_timestamp ON crypto_prices(timestamp DESC);

-- Materialized view for latest prices (for performance)
CREATE MATERIALIZED VIEW crypto_latest_prices AS
SELECT DISTINCT ON (asset_symbol)
    asset_symbol,
    price_usd,
    market_cap_usd,
    volume_24h_usd,
    change_24h_pct,
    timestamp
FROM crypto_prices
ORDER BY asset_symbol, timestamp DESC;

CREATE UNIQUE INDEX idx_latest_prices_symbol ON crypto_latest_prices(asset_symbol);

-- ============================================================================
-- DEFI POSITIONS (Staking, Lending, LP)
-- ============================================================================

CREATE TABLE IF NOT EXISTS defi_positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES crypto_wallets(id) ON DELETE CASCADE,
    
    -- Protocol information
    protocol VARCHAR(100) NOT NULL, -- AAVE, COMPOUND, UNISWAP, LIDO, CURVE
    protocol_version VARCHAR(20),
    blockchain VARCHAR(50) NOT NULL,
    contract_address TEXT,
    
    -- Position type
    position_type VARCHAR(50) NOT NULL CHECK (position_type IN (
        'LENDING',
        'BORROWING',
        'STAKING',
        'LIQUIDITY_POOL',
        'YIELD_FARMING'
    )),
    
    -- Deposited assets
    asset_deposited VARCHAR(20) NOT NULL,
    quantity_deposited DECIMAL(30,18) NOT NULL,
    deposit_value_usd DECIMAL(15,2),
    deposit_date TIMESTAMPTZ NOT NULL,
    
    -- Borrowed assets (if applicable)
    asset_borrowed VARCHAR(20),
    quantity_borrowed DECIMAL(30,18),
    
    -- Current value
    current_value_usd DECIMAL(15,2),
    
    -- Rewards
    reward_asset_symbol VARCHAR(20),
    rewards_earned DECIMAL(30,18) DEFAULT 0,
    rewards_claimed DECIMAL(30,18) DEFAULT 0,
    unclaimed_rewards_usd DECIMAL(15,2) DEFAULT 0,
    
    -- Yield metrics
    apr DECIMAL(8,4), -- Annual Percentage Rate
    apy DECIMAL(8,4), -- Annual Percentage Yield (with compounding)
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    closed_date TIMESTAMPTZ,
    
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_defi_positions_wallet ON defi_positions(wallet_id) WHERE is_active = TRUE;
CREATE INDEX idx_defi_positions_protocol ON defi_positions(protocol, position_type);
CREATE INDEX idx_defi_positions_asset ON defi_positions(asset_deposited);

-- ============================================================================
-- CRYPTO ADDRESS WHITELIST (Security)
-- ============================================================================

CREATE TABLE IF NOT EXISTS crypto_address_whitelist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES crypto_wallets(id) ON DELETE CASCADE,
    
    -- Whitelisted address
    blockchain VARCHAR(50) NOT NULL,
    address TEXT NOT NULL,
    label VARCHAR(255),
    
    -- Approval
    approved_by UUID, -- User ID who approved
    approved_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Limits
    daily_limit_usd DECIMAL(15,2),
    transaction_limit_usd DECIMAL(15,2),
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_whitelist_address UNIQUE(wallet_id, blockchain, address)
);

-- Indexes
CREATE INDEX idx_whitelist_wallet ON crypto_address_whitelist(wallet_id) WHERE is_active = TRUE;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update updated_at timestamp
CREATE TRIGGER update_crypto_wallets_updated_at
    BEFORE UPDATE ON crypto_wallets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_crypto_transactions_updated_at
    BEFORE UPDATE ON crypto_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Auto-update holdings on transaction confirmation
CREATE OR REPLACE FUNCTION update_holdings_on_transaction()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'CONFIRMED' AND (OLD.status IS NULL OR OLD.status != 'CONFIRMED') THEN
        -- Update or insert holding based on transaction type
        IF NEW.txn_type IN ('BUY', 'TRANSFER_IN', 'REWARD', 'AIRDROP', 'STAKE') THEN
            INSERT INTO crypto_holdings (wallet_id, asset_symbol, contract_address, quantity, available_quantity)
            VALUES (NEW.wallet_id, NEW.asset_symbol, NEW.contract_address, NEW.quantity, NEW.quantity)
            ON CONFLICT (wallet_id, asset_symbol, COALESCE(contract_address, ''))
            DO UPDATE SET
                quantity = crypto_holdings.quantity + NEW.quantity,
                available_quantity = crypto_holdings.available_quantity + NEW.quantity,
                last_updated = NOW();
        
        ELSIF NEW.txn_type IN ('SELL', 'TRANSFER_OUT', 'UNSTAKE') THEN
            UPDATE crypto_holdings
            SET 
                quantity = quantity - NEW.quantity,
                available_quantity = available_quantity - NEW.quantity,
                last_updated = NOW()
            WHERE wallet_id = NEW.wallet_id 
              AND asset_symbol = NEW.asset_symbol
              AND COALESCE(contract_address, '') = COALESCE(NEW.contract_address, '');
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_holdings
    AFTER INSERT OR UPDATE ON crypto_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_holdings_on_transaction();

-- Refresh materialized view on price insert
CREATE OR REPLACE FUNCTION refresh_latest_prices()
RETURNS TRIGGER AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY crypto_latest_prices;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_refresh_prices
    AFTER INSERT ON crypto_prices
    FOR EACH STATEMENT
    EXECUTE FUNCTION refresh_latest_prices();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE crypto_wallets IS 
'Multi-chain wallet management with support for institutional custodians (Coinbase Prime, Fireblocks) and self-custody';

COMMENT ON TABLE crypto_tax_lots IS 
'Tax lot tracking for IRS Form 8949 reporting with support for FIFO, LIFO, HIFO, and Specific Identification methods';

COMMENT ON TABLE defi_positions IS 
'DeFi protocol positions including Aave lending, Uniswap LP, and Lido staking';

COMMENT ON COLUMN crypto_tax_lots.is_wash_sale IS 
'IRS wash sale rule: if substantially identical security purchased within 30 days before or after a loss sale';
