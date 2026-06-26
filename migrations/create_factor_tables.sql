-- Phase 9A: Factor Model Database Schema

-- Create factor_models table
CREATE TABLE IF NOT EXISTS factor_models (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- 'fama_french', 'barra', 'custom', 'pca'
    factors JSONB NOT NULL, -- List of factor names
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create factor_exposures table
CREATE TABLE IF NOT EXISTS factor_exposures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL,
    portfolio_id TEXT NOT NULL,
    model_id UUID NOT NULL REFERENCES factor_models(id),
    as_of_date DATE NOT NULL,
    exposures JSONB NOT NULL, -- {factor: {contribution, significance, p_value}}
    narratives JSONB, -- {factor: narrative_text}
    statistics JSONB, -- {alpha, r_squared, adj_r_squared}
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create factor_returns_cache table (for caching factor time series)
CREATE TABLE IF NOT EXISTS factor_returns_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_id UUID NOT NULL REFERENCES factor_models(id),
    factor_name TEXT NOT NULL,
    as_of_date DATE NOT NULL,
    return_value NUMERIC NOT NULL, -- Factor return (decimal)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(model_id, factor_name, as_of_date)
);

-- Create factor_attribution table (for factor attribution analysis)
CREATE TABLE IF NOT EXISTS factor_attribution (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL,
    portfolio_id TEXT NOT NULL,
    model_id UUID NOT NULL REFERENCES factor_models(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    total_return NUMERIC NOT NULL,
    explained_return NUMERIC NOT NULL,
    unexplained_return NUMERIC NOT NULL,
    factor_contributions JSONB NOT NULL, -- {factor: contribution}
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_factor_exposures_tenant 
    ON factor_exposures(tenant_id, as_of_date DESC);

CREATE INDEX IF NOT EXISTS idx_factor_exposures_portfolio 
    ON factor_exposures(portfolio_id, as_of_date DESC);

CREATE INDEX IF NOT EXISTS idx_factor_exposures_model 
    ON factor_exposures(model_id, as_of_date DESC);

CREATE INDEX IF NOT EXISTS idx_factor_returns_cache_lookup 
    ON factor_returns_cache(model_id, factor_name, as_of_date);

CREATE INDEX IF NOT EXISTS idx_factor_attribution_tenant 
    ON factor_attribution(tenant_id, end_date DESC);

CREATE INDEX IF NOT EXISTS idx_factor_attribution_portfolio 
    ON factor_attribution(portfolio_id, end_date DESC);

-- Create view for latest exposures by portfolio
CREATE OR REPLACE VIEW latest_factor_exposures AS
SELECT DISTINCT ON (portfolio_id, model_id)
    id,
    tenant_id,
    portfolio_id,
    model_id,
    as_of_date,
    exposures,
    narratives,
    statistics,
    created_at
FROM factor_exposures
ORDER BY portfolio_id, model_id, as_of_date DESC;

COMMENT ON VIEW latest_factor_exposures IS 'Latest factor exposures for each portfolio/model combination';

-- Insert Fama-French 5-Factor model
INSERT INTO factor_models (name, type, factors, description) VALUES (
    'Fama-French 5-Factor (US)',
    'fama_french',
    '["Market", "SMB", "HML", "RMW", "CMA"]'::jsonb,
    'Fama-French 5-factor model: Market Risk, Size, Value, Profitability, Investment'
) ON CONFLICT DO NOTHING;

-- Insert placeholder custom PCA model
INSERT INTO factor_models (name, type, factors, description) VALUES (
    'Custom PCA 3-Factor',
    'custom',
    '["PC1", "PC2", "PC3"]'::jsonb,
    'Proprietary principal component analysis-based factors'
) ON CONFLICT DO NOTHING;
