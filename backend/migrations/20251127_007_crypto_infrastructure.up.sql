-- Phase 7A: Crypto Infrastructure Foundation
-- Institutional-grade digital asset management

-- Qualified custodian integrations
CREATE TABLE IF NOT EXISTS crypto_custody_integrations (
    integration_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    custodian_name VARCHAR(100) NOT NULL, -- 'Anchorage Digital', 'Coinbase Custody', 'Fidelity Digital Assets'
    custodian_type VARCHAR(50) NOT NULL, -- 'QUALIFIED_CUSTODIAN', 'EXCHANGE', 'SELF_CUSTODY'
    
    -- Regulatory compliance flags
    sec_qualified_custodian BOOLEAN DEFAULT FALSE, -- SEC Rule 206(4)-2
    finra_approved BOOLEAN DEFAULT FALSE,
    sipc_insured BOOLEAN DEFAULT FALSE,
    crime_insurance_limit DECIMAL(15,2),
    
    -- Technical integration
    api_endpoint TEXT,
    api_key_encrypted TEXT, -- Store encrypted
    supports_in_kind_transfers BOOLEAN DEFAULT FALSE,
    settlement_time_hours INTEGER DEFAULT 24,
    
    -- Asset support
    supported_assets TEXT[], -- ['BTC', 'ETH', 'SOL', 'USDC', etc.]
    supports_staking BOOLEAN DEFAULT FALSE,
    supports_defi_protocols BOOLEAN DEFAULT FALSE,
    
    -- Status
    compliance_reviewed_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Client crypto allocation policies
CREATE TABLE IF NOT EXISTS crypto_allocations (
    allocation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    
    -- Portfolio construction rules
    target_crypto_allocation_pct DECIMAL(5,2), -- e.g., 0.05 = 5% of total portfolio
    max_crypto_allocation_pct DECIMAL(5,2), -- Hard limit
    
    -- Asset-level constraints
    max_single_asset_pct DECIMAL(5,2), -- Prevent over-concentration
    allowed_assets TEXT[], -- Whitelist
    prohibited_assets TEXT[], -- Blacklist (e.g., privacy coins)
    
    -- Risk management
    require_qualified_custody BOOLEAN DEFAULT TRUE,
    allow_staking BOOLEAN DEFAULT FALSE,
    allow_defi BOOLEAN DEFAULT FALSE,
    max_defi_allocation_pct DECIMAL(5,2),
    
    -- Rebalancing
    rebalance_threshold_pct DECIMAL(5,2) DEFAULT 0.05, -- Auto-rebalance at 5% drift
    last_rebalanced_at TIMESTAMPTZ,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Crypto holdings (real-time positions)
CREATE TABLE IF NOT EXISTS crypto_holdings (
    holding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    account_id UUID REFERENCES accounts(id),
    custody_integration_id UUID REFERENCES crypto_custody_integrations(integration_id),
    
    -- Asset details
    asset_symbol VARCHAR(20) NOT NULL, -- 'BTC', 'ETH', 'SOL', etc.
    asset_name VARCHAR(100),
    quantity DECIMAL(28,18) NOT NULL, -- High precision for small amounts
    
    -- Cost basis tracking (FIFO, LIFO, HIFO)
    cost_basis_method VARCHAR(20) DEFAULT 'FIFO',
    total_cost_basis DECIMAL(15,2),
    average_cost_per_unit DECIMAL(15,8),
    
    -- Current valuation
    current_price_usd DECIMAL(15,8),
    current_value_usd DECIMAL(15,2),
    unrealized_gain_loss DECIMAL(15,2),
    
    -- Staking info
    is_staked BOOLEAN DEFAULT FALSE,
    staking_yield_apy DECIMAL(5,4), -- e.g., 0.045 = 4.5% APY
    staked_quantity DECIMAL(28,18),
    
    -- Custody info
    custody_wallet_address TEXT,
    custody_account_ref TEXT, -- Custodian's internal reference
    
    -- Timestamps
    acquisition_date TIMESTAMPTZ,
    last_price_update TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(client_id, asset_symbol, custody_integration_id)
);

-- Crypto transactions (full history for tax reporting)
CREATE TABLE IF NOT EXISTS crypto_transactions (
    transaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    holding_id UUID REFERENCES crypto_holdings(holding_id),
    
    -- Transaction details
    transaction_type VARCHAR(50) NOT NULL, -- 'BUY', 'SELL', 'TRANSFER_IN', 'TRANSFER_OUT', 'STAKING_REWARD', 'AIRDROP', 'FORK'
    transaction_date TIMESTAMPTZ NOT NULL,
    
    -- Asset and quantity
    asset_symbol VARCHAR(20) NOT NULL,
    quantity DECIMAL(28,18) NOT NULL,
    
    -- Pricing
    price_per_unit_usd DECIMAL(15,8),
    total_value_usd DECIMAL(15,2),
    fee_usd DECIMAL(10,2),
    
    -- Cost basis for tax reporting
    cost_basis_usd DECIMAL(15,2),
    proceeds_usd DECIMAL(15,2),
    gain_loss_usd DECIMAL(15,2),
    
    -- Tax classification
    holding_period_days INTEGER,
    is_short_term BOOLEAN, -- <= 365 days
    is_long_term BOOLEAN, -- > 365 days
    acquisition_date TIMESTAMPTZ, -- For matching with sales
    
    -- Blockchain details
    blockchain VARCHAR(50), -- 'ETHEREUM', 'BITCOIN', 'SOLANA', etc.
    transaction_hash TEXT,
    block_number BIGINT,
    
    -- Custodian reference
    custody_integration_id UUID REFERENCES crypto_custody_integrations(integration_id),
    custody_transaction_ref TEXT,
    
    -- Metadata
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Real-time market data (streaming ticks)
CREATE TABLE IF NOT EXISTS crypto_market_data (
    tick_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_symbol VARCHAR(20) NOT NULL,
    
    -- Price data
    price_usd DECIMAL(15,8) NOT NULL,
    volume_24h DECIMAL(18,2),
    market_cap DECIMAL(18,2),
    
    -- Change metrics
    price_change_24h_pct DECIMAL(6,4),
    price_change_7d_pct DECIMAL(6,4),
    
    -- OHLCV (for charting)
    open_price DECIMAL(15,8),
    high_price DECIMAL(15,8),
    low_price DECIMAL(15,8),
    close_price DECIMAL(15,8),
    
    -- Data source
    data_source VARCHAR(50), -- 'COINBASE', 'KRAKEN', 'BINANCE_US', 'GEMINI'
    
    -- Timestamp with microsecond precision
    timestamp_utc TIMESTAMPTZ DEFAULT NOW(),
    timestamp_micros BIGINT DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000000,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast price lookups
CREATE INDEX IF NOT EXISTS idx_crypto_market_data_symbol_time ON crypto_market_data(asset_symbol, timestamp_utc DESC);
CREATE INDEX IF NOT EXISTS idx_crypto_market_data_source ON crypto_market_data(data_source, asset_symbol);

-- Tokenized assets registry (future-proofing)
CREATE TABLE IF NOT EXISTS tokenized_assets (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Asset identification
    asset_type VARCHAR(50), -- 'TOKENIZED_BOND', 'TOKENIZED_EQUITY', 'TOKENIZED_FUND', 'TOKENIZED_REAL_ESTATE'
    asset_name VARCHAR(200) NOT NULL,
    ticker_symbol VARCHAR(20),
    
    -- Traditional securities info
    cusip VARCHAR(9),
    isin VARCHAR(12),
    issuer_name VARCHAR(200),
    
    -- Blockchain details
    blockchain VARCHAR(50) NOT NULL, -- 'ETHEREUM', 'POLYGON', 'STELLAR', 'AVALANCHE'
    smart_contract_address TEXT NOT NULL,
    token_standard VARCHAR(20), -- 'ERC-20', 'ERC-1400', 'ERC-3643'
    
    -- Financial details
    coupon_rate DECIMAL(6,4), -- For bonds
    maturity_date DATE, -- For bonds
    dividend_yield DECIMAL(6,4), -- For equities
    nav_per_token DECIMAL(15,8), -- For funds
    
    -- Settlement advantages
    settlement_time VARCHAR(50), -- '24/7_INSTANT', 'T+0', 'T+1'
    fractional_ownership BOOLEAN DEFAULT TRUE,
    minimum_investment DECIMAL(15,2),
    
    -- Secondary market
    secondary_trading_enabled BOOLEAN DEFAULT FALSE,
    liquidity_pool_address TEXT,
    
    -- Compliance
    accredited_investor_only BOOLEAN DEFAULT TRUE,
    qualified_purchaser_only BOOLEAN DEFAULT FALSE,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Crypto tax lot tracking (for wash sale and specific lot identification)
CREATE TABLE IF NOT EXISTS crypto_tax_lots (
    lot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    holding_id UUID REFERENCES crypto_holdings(holding_id),
    
    -- Lot details
    asset_symbol VARCHAR(20) NOT NULL,
    quantity DECIMAL(28,18) NOT NULL,
    remaining_quantity DECIMAL(28,18) NOT NULL, -- After partial sales
    
    -- Cost basis
    cost_basis_per_unit DECIMAL(15,8),
    total_cost_basis DECIMAL(15,2),
    
    -- Acquisition
    acquisition_date TIMESTAMPTZ NOT NULL,
    acquisition_method VARCHAR(50), -- 'PURCHASE', 'TRANSFER', 'STAKING_REWARD', 'AIRDROP', 'FORK'
    
    -- Tax reporting
    is_consumed BOOLEAN DEFAULT FALSE, -- Fully sold
    wash_sale_disallowed BOOLEAN DEFAULT FALSE, -- Note: Crypto currently exempt from wash-sale
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Views for reporting

-- VWAP (Volume-Weighted Average Price) calculation
CREATE OR REPLACE VIEW crypto_vwap_prices AS
SELECT 
    asset_symbol,
    AVG(price_usd) as vwap_price,
    MAX(timestamp_utc) as as_of_time,
    COUNT(*) as tick_count,
    ARRAY_AGG(DISTINCT data_source) as sources
FROM crypto_market_data
WHERE timestamp_utc >= NOW() - INTERVAL '1 hour'
GROUP BY asset_symbol;

-- Client crypto portfolio summary
DO $$
DECLARE
  id_col TEXT;
  name_col TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'clients') THEN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'clients' AND column_name = 'id') THEN
      id_col := 'id';
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'clients' AND column_name = 'client_id') THEN
      id_col := 'client_id';
    ELSE
      -- fallback to id (will cause view to be empty if neither exist)
      id_col := 'id';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'clients' AND column_name = 'client_name') THEN
      name_col := 'client_name';
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'clients' AND column_name = 'name') THEN
      name_col := 'name';
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'clients' AND column_name = 'display_name') THEN
      name_col := 'display_name';
    ELSE
      name_col := 'name';
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'crypto_holdings' AND column_name = 'client_id') THEN
      EXECUTE format(
        'CREATE OR REPLACE VIEW client_crypto_portfolios AS
         SELECT c.%I AS client_id, c.%I AS client_name,
                COUNT(DISTINCT h.asset_symbol) as unique_assets,
                SUM(h.current_value_usd) as total_crypto_value,
                SUM(h.unrealized_gain_loss) as total_unrealized_gain_loss,
                SUM(h.total_cost_basis) as total_cost_basis,
                MAX(h.last_price_update) as last_updated
         FROM clients c
         JOIN crypto_holdings h ON c.%I = h.client_id
         GROUP BY c.%I, c.%I', id_col, name_col, id_col, id_col, name_col
      );
    ELSE
      EXECUTE format(
        'CREATE OR REPLACE VIEW client_crypto_portfolios AS
         SELECT c.%I AS client_id, c.%I AS client_name,
                COUNT(DISTINCT h.asset_symbol) as unique_assets,
                COALESCE(SUM(h.quantity * COALESCE(p.price_usd, 0)), 0) as total_crypto_value,
                COALESCE(SUM(h.quantity * COALESCE(p.price_usd, 0)) - SUM(h.cost_basis_total), 0) as total_unrealized_gain_loss,
                SUM(h.cost_basis_total) as total_cost_basis,
                MAX(h.last_updated) as last_updated
         FROM clients c
         JOIN crypto_wallets w ON c.%I = w.client_id
         JOIN crypto_holdings h ON w.id = h.wallet_id
         LEFT JOIN crypto_latest_prices p ON h.asset_symbol = p.asset_symbol
         GROUP BY c.%I, c.%I', id_col, name_col, id_col, id_col, name_col
      );
    END IF;
  ELSE
    EXECUTE 'CREATE OR REPLACE VIEW client_crypto_portfolios AS SELECT NULL::uuid AS client_id, ''''::text AS client_name, 0::int AS unique_assets, 0::numeric AS total_crypto_value, 0::numeric AS total_unrealized_gain_loss, 0::numeric AS total_cost_basis, NULL::timestamptz AS last_updated WHERE FALSE';
  END IF;
END$$;

-- Indexes for performance
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'crypto_holdings' AND column_name = 'client_id') THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_crypto_holdings_client ON crypto_holdings(client_id)';
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'crypto_transactions' AND column_name = 'client_id') THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_crypto_transactions_client_date ON crypto_transactions(client_id, transaction_date DESC)';
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'crypto_tax_lots' AND column_name = 'client_id') THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_crypto_tax_lots_client_asset ON crypto_tax_lots(client_id, asset_symbol)';
  END IF;
END$$;
CREATE INDEX IF NOT EXISTS idx_crypto_holdings_asset ON crypto_holdings(asset_symbol);
CREATE INDEX IF NOT EXISTS idx_crypto_transactions_asset ON crypto_transactions(asset_symbol);

-- Trigger for updating crypto holdings valuation
CREATE OR REPLACE FUNCTION update_crypto_holding_valuation()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE crypto_holdings
    SET 
        current_price_usd = NEW.price_usd,
        current_value_usd = quantity * NEW.price_usd,
        unrealized_gain_loss = (quantity * NEW.price_usd) - total_cost_basis,
        last_price_update = NEW.timestamp_utc
    WHERE asset_symbol = NEW.asset_symbol;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_crypto_valuations ON crypto_market_data;
CREATE TRIGGER trigger_update_crypto_valuations
    AFTER INSERT ON crypto_market_data
    FOR EACH ROW
    EXECUTE FUNCTION update_crypto_holding_valuation();

-- Function to calculate crypto allocation percentage
CREATE OR REPLACE FUNCTION get_crypto_allocation_pct(p_client_id UUID)
RETURNS DECIMAL AS $$
DECLARE
    v_total_portfolio DECIMAL;
    v_crypto_value DECIMAL;
BEGIN
    -- Get total portfolio value (including traditional assets)
    SELECT SUM(current_value) INTO v_total_portfolio
    FROM portfolio_holdings
    WHERE client_id = p_client_id;
    
    -- Get crypto value
    SELECT SUM(current_value_usd) INTO v_crypto_value
    FROM crypto_holdings
    WHERE client_id = p_client_id;
    
    IF v_total_portfolio > 0 THEN
        RETURN (COALESCE(v_crypto_value, 0) / v_total_portfolio) * 100;
    ELSE
        RETURN 0;
    END IF;
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE crypto_custody_integrations IS 'Qualified custodian integrations for SEC compliance';
COMMENT ON TABLE crypto_holdings IS 'Real-time crypto positions with cost basis tracking';
COMMENT ON TABLE crypto_transactions IS 'Full transaction history for IRS Form 8949 generation';
COMMENT ON TABLE crypto_market_data IS 'Streaming market data ticks for real-time valuation';
COMMENT ON TABLE tokenized_assets IS 'Tokenized securities and real-world assets on blockchain';
