-- Private Markets Database Schema
-- This file contains the database tables for the Private Markets Explorer

-- Ensure the shared users table provides the columns required by Private Markets flows
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS role VARCHAR(50) DEFAULT 'lp',
    ADD COLUMN IF NOT EXISTS organization VARCHAR(255),
    ADD COLUMN IF NOT EXISTS permissions JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS is_core_admin BOOLEAN DEFAULT false;

ALTER TABLE public.users
    ADD CONSTRAINT IF NOT EXISTS users_role_check CHECK (role IN ('lp', 'gp', 'fof', 'steward'));

CREATE INDEX IF NOT EXISTS idx_users_email ON public.users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON public.users(role);

-- Bundles table for configuration management
CREATE TABLE IF NOT EXISTS private_markets_bundles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bundle_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    audience VARCHAR(50) NOT NULL CHECK (audience IN ('lp', 'gp', 'fof')),
    version VARCHAR(50) NOT NULL,
    modules JSONB DEFAULT '[]'::jsonb,
    metrics JSONB DEFAULT '[]'::jsonb,
    governance JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Funds table for private markets fund data
CREATE TABLE IF NOT EXISTS private_markets_funds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fund_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    vintage INTEGER NOT NULL,
    manager VARCHAR(255) NOT NULL,
    strategy VARCHAR(255) NOT NULL,
    geography VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'liquidated', 'realizing')),
    description TEXT,
    target_size DECIMAL(20,2),
    committed_capital DECIMAL(20,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Fund metrics table for performance data
CREATE TABLE IF NOT EXISTS private_markets_fund_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fund_id VARCHAR(100) NOT NULL REFERENCES private_markets_funds(fund_id) ON DELETE CASCADE,
    as_of_date DATE NOT NULL,
    tvpi DECIMAL(10,4),
    rvpi DECIMAL(10,4),
    irr DECIMAL(10,6),
    xirr DECIMAL(10,6),
    pme DECIMAL(10,4),
    paid_in_capital DECIMAL(20,2),
    distributions DECIMAL(20,2),
    residual_value DECIMAL(20,2),
    nav DECIMAL(20,2),
    dpi DECIMAL(10,4),
    multiple DECIMAL(10,4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    UNIQUE(fund_id, as_of_date)
);

CREATE TABLE IF NOT EXISTS private_markets_user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    bundle_id VARCHAR(100) NOT NULL,
    dashboard_config JSONB DEFAULT '{}'::jsonb,
    favorite_funds JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_pm_bundles_audience ON private_markets_bundles(audience);
CREATE INDEX IF NOT EXISTS idx_pm_funds_manager ON private_markets_funds(manager);
CREATE INDEX IF NOT EXISTS idx_pm_funds_strategy ON private_markets_funds(strategy);
CREATE INDEX IF NOT EXISTS idx_pm_fund_metrics_fund_id ON private_markets_fund_metrics(fund_id);
CREATE INDEX IF NOT EXISTS idx_pm_fund_metrics_date ON private_markets_fund_metrics(as_of_date);

-- User authentication tables
CREATE TABLE IF NOT EXISTS private_markets_user_auth (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    salt VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    UNIQUE(user_id)
);

CREATE TABLE IF NOT EXISTS private_markets_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    refresh_token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    refresh_expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE TABLE IF NOT EXISTS private_markets_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_revoked BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    revoked_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for authentication performance
CREATE INDEX IF NOT EXISTS idx_pm_user_auth_user_id ON private_markets_user_auth(user_id);
CREATE INDEX IF NOT EXISTS idx_pm_sessions_user_id ON private_markets_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_pm_sessions_refresh_token ON private_markets_sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_pm_sessions_expires_at ON private_markets_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_pm_refresh_tokens_user_id ON private_markets_refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_pm_refresh_tokens_token ON private_markets_refresh_tokens(token);
CREATE INDEX IF NOT EXISTS idx_pm_refresh_tokens_expires_at ON private_markets_refresh_tokens(expires_at);

INSERT INTO public.users (username, email, name, role, organization, permissions)
VALUES
    ('john.doe@sample.com', 'john.doe@sample.com', 'John Doe', 'lp', 'Sample LP Organization', '["read", "write", "admin"]'),
    ('jane.smith@sample.com', 'jane.smith@sample.com', 'Jane Smith', 'gp', 'Sample GP Organization', '["read", "write"]'),
    ('bob.wilson@sample.com', 'bob.wilson@sample.com', 'Bob Wilson', 'fof', 'Sample FoF Organization', '["read"]'),
    ('alice.brown@sample.com', 'alice.brown@sample.com', 'Alice Brown', 'steward', 'Data Governance Team', '["read", "write", "admin", "steward"]')
ON CONFLICT (username) DO NOTHING;

UPDATE public.users
SET is_core_admin = true
WHERE email = 'admin@example.com';

INSERT INTO private_markets_bundles (bundle_id, name, audience, version, modules, metrics, governance) VALUES
('lp_private_markets_bundle', 'LP Private Markets Bundle', 'lp', '1.0.0',
 '[{"id": "fund-selector", "name": "Fund Selector", "type": "selector", "config": {"multiSelect": true}}, {"id": "irr-curve", "name": "IRR Curve Chart", "type": "chart", "config": {"timeRange": "5y"}}, {"id": "j-curve", "name": "J-Curve Plot", "type": "chart", "config": {"showBenchmark": true}}, {"id": "benchmark-comparison", "name": "Benchmark Comparison", "type": "comparison", "config": {"indices": ["S&P 500", "NASDAQ"]}}, {"id": "liquidity-panel", "name": "Liquidity Panel", "type": "panel", "config": {"showProjections": true}}]'::jsonb,
 '[{"id": "tvpi", "name": "TVPI", "type": "ratio", "formula": "(distributions + residual_value) / paid_in_capital"}, {"id": "irr", "name": "IRR", "type": "percentage", "formula": "XIRR(cash_flows, dates)"}, {"id": "pme", "name": "PME", "type": "ratio", "formula": "PME(cash_flows, benchmark)"}]'::jsonb,
 '{"status": "active", "steward_group": "data-stewards", "schema_hash": "abc123", "sla": {"refresh_frequency": "daily", "max_latency": "4h"}}'::jsonb),

('gp_private_markets_bundle', 'GP Private Markets Bundle', 'gp', '1.0.0',
 '[{"id": "deployment-pacing", "name": "Deployment Pacing Chart", "type": "chart", "config": {"targetPacing": "24months"}}, {"id": "irr-nav-tracking", "name": "IRR/NAV Tracking", "type": "tracking", "config": {"frequency": "quarterly"}}, {"id": "fee-analysis", "name": "Fee Analysis", "type": "analysis", "config": {"feeTypes": ["management", "performance"]}}, {"id": "value-attribution", "name": "Value Attribution", "type": "attribution", "config": {"methodology": "brinson"}}, {"id": "exit-analysis", "name": "Exit Analysis", "type": "analysis", "config": {"exitTypes": ["ipo", "merger", "sale"]}}]'::jsonb,
 '[{"id": "dpi", "name": "DPI", "type": "ratio", "formula": "distributions / paid_in_capital"}, {"id": "rvpi", "name": "RVPI", "type": "ratio", "formula": "residual_value / paid_in_capital"}, {"id": "tvpi", "name": "TVPI", "type": "ratio", "formula": "dpi + rvpi"}]'::jsonb,
 '{"status": "active", "steward_group": "gp-stewards", "schema_hash": "def456", "sla": {"refresh_frequency": "weekly", "max_latency": "24h"}}'::jsonb),

('fof_private_markets_bundle', 'FoF Private Markets Bundle', 'fof', '1.0.0',
 '[{"id": "portfolio-overview", "name": "Portfolio Overview", "type": "overview", "config": {"groupBy": "strategy"}}, {"id": "manager-performance", "name": "Manager Performance", "type": "performance", "config": {"benchmark": true}}, {"id": "allocation-analysis", "name": "Allocation Analysis", "type": "analysis", "config": {"dimensions": ["geography", "vintage"]}}, {"id": "risk-attribution", "name": "Risk Attribution", "type": "attribution", "config": {"method": "factor"}}]'::jsonb,
 '[{"id": "portfolio-irr", "name": "Portfolio IRR", "type": "percentage", "formula": "weighted_average(irr)"}, {"id": "diversification", "name": "Diversification Score", "type": "score", "formula": "1 - concentration_ratio"}, {"id": "alpha", "name": "Alpha vs Benchmark", "type": "percentage", "formula": "irr - benchmark_irr"}]'::jsonb,
 '{"status": "active", "steward_group": "fof-stewards", "schema_hash": "ghi789", "sla": {"refresh_frequency": "monthly", "max_latency": "48h"}}'::jsonb)
ON CONFLICT (bundle_id) DO NOTHING;

INSERT INTO private_markets_funds (fund_id, name, vintage, manager, strategy, geography, status, description, target_size, committed_capital) VALUES
('fund-1', 'Tech Growth Fund III', 2020, 'TechVentures Capital', 'Venture Capital', 'North America', 'active', 'Focused on high-growth technology companies', 500000000, 450000000),
('fund-2', 'Infrastructure Partners II', 2019, 'InfraCapital', 'Infrastructure', 'Europe', 'active', 'European infrastructure investments', 800000000, 750000000),
('fund-3', 'Real Estate Fund IV', 2021, 'PropertyPartners', 'Real Estate', 'Asia Pacific', 'active', 'Asian real estate development', 600000000, 550000000),
('fund-4', 'Healthcare Innovation Fund', 2022, 'MedTech Ventures', 'Healthcare', 'Global', 'active', 'Global healthcare technology investments', 400000000, 380000000)
ON CONFLICT (fund_id) DO NOTHING;
INSERT INTO private_markets_fund_metrics (fund_id, as_of_date, tvpi, rvpi, irr, xirr, pme, paid_in_capital, distributions, residual_value, nav, dpi) VALUES
('fund-1', CURRENT_DATE, 1.85, 1.23, 0.156, 0.142, 1.12, 100000000, 85000000, 123000000, 123000000, 0.85),
('fund-2', CURRENT_DATE, 1.65, 1.45, 0.123, 0.118, 1.08, 100000000, 65000000, 145000000, 145000000, 0.65),
('fund-3', CURRENT_DATE, 1.92, 1.67, 0.145, 0.138, 1.15, 100000000, 92000000, 167000000, 167000000, 0.92),
('fund-4', CURRENT_DATE, 2.05, 1.89, 0.178, 0.165, 1.22, 100000000, 105000000, 189000000, 189000000, 1.05)
ON CONFLICT (fund_id, as_of_date) DO NOTHING;
