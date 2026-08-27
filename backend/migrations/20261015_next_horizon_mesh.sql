-- 20261015_next_horizon_mesh.sql
-- Next-Horizon Apex Mesh: Mandate SMT Verifications & ZK Compliance Passports

CREATE SCHEMA IF NOT EXISTS privacy;
CREATE SCHEMA IF NOT EXISTS optimizer;
CREATE SCHEMA IF NOT EXISTS compliance;

CREATE TABLE IF NOT EXISTS compliance.mandate_smt_verifications (
    verification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    mandate_code VARCHAR(100) NOT NULL,
    is_satisfiable BOOLEAN NOT NULL,
    conflict_detected BOOLEAN NOT NULL,
    diagnostic_message TEXT NOT NULL,
    rules_payload JSONB NOT NULL,
    verified_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS privacy.zk_compliance_passports (
    passport_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    metric_name VARCHAR(100) NOT NULL,
    noisy_value NUMERIC(18, 6) NOT NULL,
    epsilon_used NUMERIC(6, 4) NOT NULL,
    sensitivity NUMERIC(10, 4) NOT NULL,
    zk_passport_hash VARCHAR(64) NOT NULL,
    certified_at TIMESTAMPTZ DEFAULT NOW()
);
