-- Migration: 20261020_001_oms_security_search_trgm.up.sql
-- Purpose: Support high-performance (<100ms) prefix, exact, and token search across
--          operational securities in oms.security for the Workstation Command Bar.
--
-- Note on edm.security_master:
--   Search targets oms.security (the operational multi-tenant security master used by
--   the OMS runtime). edm.security_master is a separate gold-copy enterprise data model
--   table out of scope for workstation execution.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- B-Tree indexes for exact symbol/ISIN matching and tenant filtering
CREATE INDEX IF NOT EXISTS idx_oms_security_tenant_ticker
    ON oms.security (tenant_id, UPPER(ticker))
    WHERE ticker IS NOT NULL AND valid_to IS NULL;

CREATE INDEX IF NOT EXISTS idx_oms_security_tenant_isin
    ON oms.security (tenant_id, UPPER(isin))
    WHERE isin IS NOT NULL AND valid_to IS NULL;

-- GIN trigram indexes for prefix and token similarity search
CREATE INDEX IF NOT EXISTS idx_oms_security_ticker_trgm
    ON oms.security USING gin (ticker gin_trgm_ops)
    WHERE ticker IS NOT NULL AND valid_to IS NULL;

CREATE INDEX IF NOT EXISTS idx_oms_security_name_trgm
    ON oms.security USING gin (security_name gin_trgm_ops)
    WHERE valid_to IS NULL;
