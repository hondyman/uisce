-- Migration: 017_factor_models.sql
-- Description: Creates tables for Factor Models, Definitions, and Returns.

-- 1. Factor Models Registry (e.g., "Fama-French 3-Factor", "Carhart 4-Factor")
CREATE TABLE IF NOT EXISTS factor_models (
    model_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Factor Definitions (e.g., "Mkt-RF", "SMB", "HML")
CREATE TABLE IF NOT EXISTS factor_definitions (
    factor_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES factor_models(model_id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT ux_factor_definitions_slug UNIQUE (model_id, slug)
);

-- 3. Factor Returns (Time-Series Data)
CREATE TABLE IF NOT EXISTS factor_returns (
    factor_id UUID NOT NULL REFERENCES factor_definitions(factor_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    return_value NUMERIC(19, 6) NOT NULL, -- High precision for returns
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    PRIMARY KEY (factor_id, date)
);

-- Index for efficient time-series querying
CREATE INDEX IF NOT EXISTS idx_factor_returns_date ON factor_returns (date);
