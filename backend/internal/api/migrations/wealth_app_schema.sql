-- Wealth Management Software Database Schema
-- PostgreSQL Database Schema for comprehensive wealth management platform

-- Enable UUID extension for generating UUIDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create ENUM types for better data integrity
CREATE TYPE user_status AS ENUM ('ACTIVE', 'INACTIVE', 'SUSPENDED');
CREATE TYPE asset_type AS ENUM ('STOCK', 'BOND', 'MUTUAL_FUND', 'ETF', 'COMMODITY', 'CRYPTOCURRENCY', 'REAL_ESTATE', 'CASH');
CREATE TYPE transaction_type AS ENUM ('BUY', 'SELL', 'DIVIDEND', 'INTEREST', 'FEE', 'TRANSFER_IN', 'TRANSFER_OUT');
CREATE TYPE transaction_status AS ENUM ('PENDING', 'COMPLETED', 'FAILED', 'CANCELLED');
CREATE TYPE order_type AS ENUM ('MARKET', 'LIMIT', 'STOP', 'STOP_LIMIT');
CREATE TYPE order_side AS ENUM ('BUY', 'SELL');
CREATE TYPE order_status AS ENUM ('PENDING', 'PARTIALLY_FILLED', 'FILLED', 'CANCELLED', 'REJECTED');
CREATE TYPE compliance_status AS ENUM ('COMPLIANT', 'NON_COMPLIANT', 'PENDING_REVIEW', 'REQUIRES_ACTION');
CREATE TYPE risk_tolerance AS ENUM ('CONSERVATIVE', 'MODERATE', 'AGGRESSIVE', 'VERY_AGGRESSIVE');

-- Users table - stores wealth managers and system users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    status user_status DEFAULT 'ACTIVE',
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Roles table - defines user roles in the system
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Permissions table - granular permissions
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User roles junction table
CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

-- Role permissions junction table
CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);

-- Clients table - high net worth individuals
CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- Wealth manager who manages this client
    client_code VARCHAR(20) UNIQUE NOT NULL, -- Unique client identifier
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    date_of_birth DATE,
    ssn VARCHAR(11), -- Encrypted in production
    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(50),
    zip_code VARCHAR(20),
    country VARCHAR(50) DEFAULT 'USA',
    primary_phone VARCHAR(20),
    secondary_phone VARCHAR(20),
    email VARCHAR(255),
    risk_tolerance risk_tolerance DEFAULT 'MODERATE',
    investment_goals TEXT,
    net_worth DECIMAL(15,2),
    annual_income DECIMAL(15,2),
    kyc_status compliance_status DEFAULT 'PENDING_REVIEW',
    kyc_completed_at TIMESTAMP WITH TIME ZONE,
    aml_status compliance_status DEFAULT 'PENDING_REVIEW',
    aml_completed_at TIMESTAMP WITH TIME ZONE,
    onboarding_completed BOOLEAN DEFAULT FALSE,
    status user_status DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Portfolios table - investment portfolios for clients
CREATE TABLE portfolios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES clients(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    portfolio_type VARCHAR(50) DEFAULT 'INVESTMENT', -- INVESTMENT, RETIREMENT, TRUST, etc.
    base_currency VARCHAR(3) DEFAULT 'USD',
    inception_date DATE DEFAULT CURRENT_DATE,
    target_allocation JSONB, -- JSON object storing target asset allocation percentages
    benchmark_symbol VARCHAR(20), -- Benchmark index for performance comparison
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Assets master table - reference data for all tradeable assets
CREATE TABLE assets_master (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    asset_type asset_type NOT NULL,
    exchange VARCHAR(50),
    currency VARCHAR(3) DEFAULT 'USD',
    sector VARCHAR(100),
    industry VARCHAR(100),
    country VARCHAR(50),
    isin VARCHAR(12), -- International Securities Identification Number
    cusip VARCHAR(9), -- Committee on Uniform Securities Identification Procedures
    description TEXT,
    is_tradeable BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Portfolio holdings - current positions in portfolios
CREATE TABLE portfolio_holdings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID REFERENCES portfolios(id) ON DELETE CASCADE,
    asset_id UUID REFERENCES assets_master(id) ON DELETE RESTRICT,
    quantity DECIMAL(15,6) NOT NULL DEFAULT 0,
    average_cost DECIMAL(15,4), -- Average cost basis per unit
    market_value DECIMAL(15,2), -- Current market value (quantity * current_price)
    unrealized_pnl DECIMAL(15,2), -- Unrealized profit/loss
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Transactions table - all financial transactions
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID REFERENCES portfolios(id) ON DELETE CASCADE,
    asset_id UUID REFERENCES assets_master(id) ON DELETE RESTRICT,
    transaction_type transaction_type NOT NULL,
    quantity DECIMAL(15,6),
    price_per_unit DECIMAL(15,4),
    total_amount DECIMAL(15,2) NOT NULL,
    fees DECIMAL(15,2) DEFAULT 0,
    net_amount DECIMAL(15,2), -- total_amount - fees
    currency VARCHAR(3) DEFAULT 'USD',
    transaction_date TIMESTAMP WITH TIME ZONE NOT NULL,
    settlement_date DATE,
    status transaction_status DEFAULT 'PENDING',
    order_id UUID, -- Reference to orders table if applicable
    external_transaction_id VARCHAR(100), -- ID from external trading system
    notes TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Orders table - trading orders
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID REFERENCES portfolios(id) ON DELETE CASCADE,
    asset_id UUID REFERENCES assets_master(id) ON DELETE RESTRICT,
    order_type order_type NOT NULL,
    side order_side NOT NULL,
    quantity DECIMAL(15,6) NOT NULL,
    price DECIMAL(15,4), -- NULL for market orders
    stop_price DECIMAL(15,4), -- For stop orders
    filled_quantity DECIMAL(15,6) DEFAULT 0,
    remaining_quantity DECIMAL(15,6),
    average_fill_price DECIMAL(15,4),
    status order_status DEFAULT 'PENDING',
    time_in_force VARCHAR(10) DEFAULT 'DAY', -- DAY, GTC (Good Till Cancelled), IOC (Immediate or Cancel)
    external_order_id VARCHAR(100), -- ID from external trading system
    placed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    filled_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Market data table - caches market prices and data
CREATE TABLE market_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID REFERENCES assets_master(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    last_price DECIMAL(15,4),
    bid_price DECIMAL(15,4),
    ask_price DECIMAL(15,4),
    open_price DECIMAL(15,4),
    high_price DECIMAL(15,4),
    low_price DECIMAL(15,4),
    close_price DECIMAL(15,4),
    volume BIGINT,
    market_cap DECIMAL(20,2),
    pe_ratio DECIMAL(8,2),
    dividend_yield DECIMAL(5,4),
    fifty_two_week_high DECIMAL(15,4),
    fifty_two_week_low DECIMAL(15,4),
    data_source VARCHAR(50), -- Bloomberg, Yahoo, Alpha Vantage, etc.
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Compliance records table
CREATE TABLE compliance_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID REFERENCES clients(id) ON DELETE CASCADE,
    portfolio_id UUID REFERENCES portfolios(id) ON DELETE SET NULL,
    transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    record_type VARCHAR(50) NOT NULL, -- KYC, AML, TRADE_REVIEW, SUITABILITY, etc.
    status compliance_status DEFAULT 'PENDING_REVIEW',
    risk_score INTEGER, -- 1-100 risk score
    details JSONB, -- Flexible storage for compliance-specific data
    rules_applied TEXT[], -- Array of compliance rules that were checked
    violations TEXT[], -- Array of any violations found
    remediation_actions TEXT[], -- Required actions to address violations
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    due_date DATE,
    completed_at TIMESTAMP WITH TIME ZONE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Audit trail table - tracks all system changes
CREATE TABLE audit_trail (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    table_name VARCHAR(50) NOT NULL,
    record_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL, -- INSERT, UPDATE, DELETE
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Performance tracking table
CREATE TABLE portfolio_performance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID REFERENCES portfolios(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    total_value DECIMAL(15,2) NOT NULL,
    daily_return DECIMAL(8,6), -- Daily return percentage
    cumulative_return DECIMAL(8,6), -- Cumulative return since inception
    benchmark_return DECIMAL(8,6), -- Benchmark return for comparison
    alpha DECIMAL(8,6), -- Alpha vs benchmark
    beta DECIMAL(8,6), -- Beta vs benchmark
    sharpe_ratio DECIMAL(8,6),
    volatility DECIMAL(8,6), -- Annualized volatility
    max_drawdown DECIMAL(8,6),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(portfolio_id, date)
);

-- Notifications table
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    client_id UUID REFERENCES clients(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- TRADE_EXECUTED, COMPLIANCE_ALERT, PERFORMANCE_UPDATE, etc.
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    priority VARCHAR(20) DEFAULT 'MEDIUM', -- LOW, MEDIUM, HIGH, URGENT
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    data JSONB, -- Additional structured data
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_clients_user_id ON clients(user_id);
CREATE INDEX idx_clients_client_code ON clients(client_code);
CREATE INDEX idx_portfolios_client_id ON portfolios(client_id);
CREATE INDEX idx_portfolio_holdings_portfolio_id ON portfolio_holdings(portfolio_id);
CREATE INDEX idx_portfolio_holdings_asset_id ON portfolio_holdings(asset_id);
CREATE INDEX idx_transactions_portfolio_id ON transactions(portfolio_id);
CREATE INDEX idx_transactions_asset_id ON transactions(asset_id);
CREATE INDEX idx_transactions_date ON transactions(transaction_date);
CREATE INDEX idx_orders_portfolio_id ON orders(portfolio_id);
CREATE INDEX idx_orders_asset_id ON orders(asset_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_market_data_symbol ON market_data(symbol);
CREATE INDEX idx_market_data_asset_id ON market_data(asset_id);
CREATE INDEX idx_compliance_records_client_id ON compliance_records(client_id);
CREATE INDEX idx_compliance_records_status ON compliance_records(status);
CREATE INDEX idx_audit_trail_user_id ON audit_trail(user_id);
CREATE INDEX idx_audit_trail_table_record ON audit_trail(table_name, record_id);
CREATE INDEX idx_portfolio_performance_portfolio_date ON portfolio_performance(portfolio_id, date);
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;

-- Create triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers to relevant tables
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_clients_updated_at BEFORE UPDATE ON clients FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_portfolios_updated_at BEFORE UPDATE ON portfolios FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_assets_master_updated_at BEFORE UPDATE ON assets_master FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- Generic Business Object Instances Table (Metadata-Driven Storage)
CREATE TABLE bo_instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL, -- Redundant but good for safety
    business_object_id TEXT NOT NULL,
    business_object_key TEXT NOT NULL,
    datasource_id TEXT,
    subtype_id TEXT,
    subtype_key TEXT,
    core_field_values JSONB DEFAULT '{}',
    custom_field_values JSONB DEFAULT '{}',
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_modified_by TEXT,
    last_modified_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bo_instances_bo_key ON bo_instances(business_object_key);
CREATE INDEX idx_bo_instances_tenant_bo ON bo_instances(tenant_id, business_object_key);
