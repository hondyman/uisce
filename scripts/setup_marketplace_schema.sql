-- ============================================================================
-- THE UISCE MARKETPLACE & CUSTOMIZATION TELEMETRY SCHEMA
-- Target: PostgreSQL (100.84.50.65 / alpha DB) & StarRocks FE (9030)
-- ============================================================================

-- 1. PostgreSQL Marketplace Tables
CREATE TABLE IF NOT EXISTS marketplace_listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(150) NOT NULL,
    category VARCHAR(50) NOT NULL, -- RBAC, Compliance, ABAC, Analytics
    publisher_tenant_id UUID NOT NULL,
    publisher_name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    price_cents INT DEFAULT 0, -- 0 for Free
    rating NUMERIC(2, 1) DEFAULT 5.0,
    installs_count INT DEFAULT 0,
    is_published BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS marketplace_artifacts (
    artifact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES marketplace_listings(id) ON DELETE CASCADE,
    artifact_type VARCHAR(50) NOT NULL, -- ROLE, POLICY, RULE_PACK
    payload JSONB NOT NULL, -- Sanitized JSON payload of role/policy definition
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS marketplace_installs (
    install_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES marketplace_listings(id) ON DELETE CASCADE,
    installed_by_tenant_id UUID NOT NULL,
    installed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Seed initial Marketplace items
INSERT INTO marketplace_listings (id, title, category, publisher_tenant_id, publisher_name, description, price_cents, rating, installs_count)
VALUES 
  ('11111111-1111-1111-1111-111111111111'::uuid, 'SOX 404 Compliance Rule Pack', 'Compliance', '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'Uisce Core Engineering', 'Pre-configured audit rule pack enforcing strict segregation of duties for financial controllers.', 0, 4.9, 142),
  ('22222222-2222-2222-2222-222222222222'::uuid, 'GSIFI High-Frequency Trader ABAC Policy', 'ABAC', '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid, 'Goldman Partner Services', 'Dynamic ABAC policy restricting portfolio modifications based on real-time market risk limits.', 4900, 5.0, 38)
ON CONFLICT (id) DO NOTHING;

INSERT INTO marketplace_artifacts (listing_id, artifact_type, payload)
VALUES 
  ('11111111-1111-1111-1111-111111111111'::uuid, 'ROLE', '{"role_key": "sox_controller", "role_name": "SOX Compliance Controller", "description": "Automated SOX 404 segregation of duties controller role"}'::jsonb),
  ('22222222-2222-2222-2222-222222222222'::uuid, 'ROLE', '{"role_key": "hft_trader", "role_name": "High-Frequency Trader", "description": "GSIFI compliance high-frequency trader ABAC policy"}'::jsonb)
ON CONFLICT DO NOTHING;

-- 2. StarRocks Intelligence View for Customization Trends
CREATE TABLE IF NOT EXISTS fact_customization_telemetry (
    telemetry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_name VARCHAR(100) NOT NULL,
    matching_tenants_count INT NOT NULL,
    recommendation VARCHAR(200) NOT NULL,
    confidence_score NUMERIC(3, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
