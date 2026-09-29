-- 20261118_004_mdm_scoring_expansion.up.sql
-- MDM Vendor Scoring Expansion: Telemetry, SLA, Stability, Friction, Contract Rights, and Weight Profiles

CREATE SCHEMA IF NOT EXISTS mdm_eval;

-- 1. Tenant-Configurable Scoring Settings & Labor Rates
CREATE TABLE IF NOT EXISTS mdm_eval.scoring_settings (
    tenant_id UUID PRIMARY KEY,
    hourly_labor_rate NUMERIC(10,2) NOT NULL DEFAULT 150.00,
    stability_decay_k NUMERIC(8,2) NOT NULL DEFAULT 50.0,
    friction_budget NUMERIC(12,2) NOT NULL DEFAULT 100000.00,
    cold_start_days INT NOT NULL DEFAULT 30,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Weight Profile Governance & Audit Trail
CREATE TABLE IF NOT EXISTS mdm_eval.scoring_weight_profiles (
    profile_id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    profile_name VARCHAR(64) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    weight_suff NUMERIC(4,3) NOT NULL CHECK (weight_suff >= 0.0 AND weight_suff <= 1.0),
    weight_cov NUMERIC(4,3) NOT NULL CHECK (weight_cov >= 0.0 AND weight_cov <= 1.0),
    weight_sla NUMERIC(4,3) NOT NULL CHECK (weight_sla >= 0.0 AND weight_sla <= 1.0),
    weight_stab NUMERIC(4,3) NOT NULL CHECK (weight_stab >= 0.0 AND weight_stab <= 1.0),
    weight_oer NUMERIC(4,3) NOT NULL CHECK (weight_oer >= 0.0 AND weight_oer <= 1.0),
    weight_lic NUMERIC(4,3) NOT NULL CHECK (weight_lic >= 0.0 AND weight_lic <= 1.0),
    created_by VARCHAR(64) NOT NULL,
    change_reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_weights_sum CHECK (
        ABS((weight_suff + weight_cov + weight_sla + weight_stab + weight_oer + weight_lic) - 1.000) < 0.001
    )
);

CREATE INDEX IF NOT EXISTS ix_weight_profiles_tenant_active 
    ON mdm_eval.scoring_weight_profiles (tenant_id, is_active);

-- 3. Feed Arrival & SLA Telemetry
CREATE TABLE IF NOT EXISTS mdm_eval.vendor_feed_log (
    log_id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    vendor_id VARCHAR(32) NOT NULL,
    entity_domain VARCHAR(32) NOT NULL CHECK (entity_domain IN ('security', 'ratings', 'benchmarks', 'pricing', 'party')),
    feed_name VARCHAR(128) NOT NULL,
    as_of_date DATE NOT NULL,
    arrival_ts TIMESTAMPTZ NOT NULL,
    sla_cutoff_ts TIMESTAMPTZ NOT NULL,
    delivery_lag_mins INT NOT NULL,
    sla_breached BOOLEAN NOT NULL DEFAULT FALSE,
    records_received INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_vendor_feed_log_vendor_date 
    ON mdm_eval.vendor_feed_log (tenant_id, vendor_id, as_of_date);

-- 4. Vendor Restatement Log (Vendor-Initiated Revisions)
CREATE TABLE IF NOT EXISTS mdm_eval.vendor_revision_log (
    revision_id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    vendor_id VARCHAR(32) NOT NULL,
    entity_id BIGINT NOT NULL,
    attribute_code VARCHAR(64) NOT NULL,
    entity_domain VARCHAR(32) NOT NULL CHECK (entity_domain IN ('security', 'ratings', 'benchmarks', 'pricing', 'party')),
    as_of_date DATE NOT NULL,
    original_value TEXT NOT NULL,
    revised_value TEXT NOT NULL,
    pct_change NUMERIC(12,6),
    hours_to_revision NUMERIC(8,2) NOT NULL,
    revision_ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ix_vendor_rev_log_vendor_date 
    ON mdm_eval.vendor_revision_log (tenant_id, vendor_id, as_of_date);
CREATE INDEX IF NOT EXISTS ix_vendor_rev_log_entity_attr 
    ON mdm_eval.vendor_revision_log (entity_id, attribute_code);

-- 5. Operational Stewardship Friction Tracker
CREATE TABLE IF NOT EXISTS mdm_eval.vendor_operational_friction (
    metric_id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    vendor_id VARCHAR(32) NOT NULL,
    as_of_date DATE NOT NULL,
    defect_tickets_count INT NOT NULL DEFAULT 0,
    investigation_hours NUMERIC(8,2) NOT NULL DEFAULT 0.0,
    avg_mttr_hours NUMERIC(8,2) NOT NULL DEFAULT 0.0,
    contract_sla_credits NUMERIC(12,2) NOT NULL DEFAULT 0.0,
    calculated_friction_cost NUMERIC(12,2) NOT NULL DEFAULT 0.0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_vendor_friction_tenant_vendor_date UNIQUE (tenant_id, vendor_id, as_of_date)
);

CREATE INDEX IF NOT EXISTS ix_vendor_friction_tenant_vendor 
    ON mdm_eval.vendor_operational_friction (tenant_id, vendor_id, as_of_date);

-- 6. Commercial Contract Rights Matrix
CREATE TABLE IF NOT EXISTS mdm_eval.vendor_contract_rights (
    contract_id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    vendor_id VARCHAR(32) NOT NULL,
    derived_data_rights INT NOT NULL CHECK (derived_data_rights BETWEEN 0 AND 100),
    client_redistribution INT NOT NULL CHECK (client_redistribution BETWEEN 0 AND 100),
    external_web_rights INT NOT NULL CHECK (external_web_rights BETWEEN 0 AND 100),
    unbundled_api_access BOOLEAN NOT NULL DEFAULT TRUE,
    cancellation_notice_days INT NOT NULL DEFAULT 90,
    contract_start_date DATE NOT NULL,
    contract_end_date DATE NOT NULL,
    composite_rights_score NUMERIC(5,2) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_vendor_contract_tenant_vendor UNIQUE (tenant_id, vendor_id)
);

-- Enable Row-Level Security
ALTER TABLE mdm_eval.scoring_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_eval.scoring_weight_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_eval.vendor_feed_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_eval.vendor_revision_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_eval.vendor_operational_friction ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_eval.vendor_contract_rights ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'scoring_settings' AND policyname = 'tenant_isolation_settings') THEN
        CREATE POLICY tenant_isolation_settings ON mdm_eval.scoring_settings
            USING (tenant_id = public.uisce_get_current_tenant())
            WITH CHECK (tenant_id = public.uisce_get_current_tenant());
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'scoring_weight_profiles' AND policyname = 'tenant_isolation_weights') THEN
        CREATE POLICY tenant_isolation_weights ON mdm_eval.scoring_weight_profiles
            USING (tenant_id = public.uisce_get_current_tenant())
            WITH CHECK (tenant_id = public.uisce_get_current_tenant());
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'vendor_feed_log' AND policyname = 'tenant_isolation_feed_log') THEN
        CREATE POLICY tenant_isolation_feed_log ON mdm_eval.vendor_feed_log
            USING (tenant_id = public.uisce_get_current_tenant())
            WITH CHECK (tenant_id = public.uisce_get_current_tenant());
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'vendor_revision_log' AND policyname = 'tenant_isolation_revisions') THEN
        CREATE POLICY tenant_isolation_revisions ON mdm_eval.vendor_revision_log
            USING (tenant_id = public.uisce_get_current_tenant())
            WITH CHECK (tenant_id = public.uisce_get_current_tenant());
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'vendor_operational_friction' AND policyname = 'tenant_isolation_friction') THEN
        CREATE POLICY tenant_isolation_friction ON mdm_eval.vendor_operational_friction
            USING (tenant_id = public.uisce_get_current_tenant())
            WITH CHECK (tenant_id = public.uisce_get_current_tenant());
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'vendor_contract_rights' AND policyname = 'tenant_isolation_rights') THEN
        CREATE POLICY tenant_isolation_rights ON mdm_eval.vendor_contract_rights
            USING (tenant_id = public.uisce_get_current_tenant())
            WITH CHECK (tenant_id = public.uisce_get_current_tenant());
    END IF;
END $$;

-- 7. Seed Initial Baseline Configuration for Gold Copy Tenant
INSERT INTO mdm_eval.scoring_settings (tenant_id, hourly_labor_rate, stability_decay_k, friction_budget, cold_start_days)
VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', 150.00, 50.00, 100000.00, 30)
ON CONFLICT (tenant_id) DO NOTHING;

INSERT INTO mdm_eval.scoring_weight_profiles (
    tenant_id, profile_name, is_active,
    weight_suff, weight_cov, weight_sla, weight_stab, weight_oer, weight_lic,
    created_by, change_reason
) VALUES (
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'Balanced Institutional Standard',
    TRUE,
    0.300, 0.200, 0.150, 0.150, 0.100, 0.100,
    'system_seed',
    'Standard procurement & stewardship multi-dimensional weight profile v2.1'
) ON CONFLICT DO NOTHING;

INSERT INTO mdm_eval.vendor_contract_rights (
    tenant_id, vendor_id, derived_data_rights, client_redistribution, external_web_rights,
    unbundled_api_access, cancellation_notice_days, contract_start_date, contract_end_date, composite_rights_score
) VALUES 
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'BBG', 85, 70, 50, FALSE, 90, '2026-01-01', '2026-12-31', 69.25),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'RFT', 90, 80, 65, TRUE,  60, '2026-01-01', '2026-12-31', 83.25),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'FDS', 80, 75, 60, TRUE,  60, '2026-01-01', '2026-12-31', 77.75),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'ICE', 85, 85, 70, TRUE,  30, '2026-01-01', '2026-12-31', 86.50),
    ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'SPG', 70, 60, 40, TRUE,  90, '2026-01-01', '2026-12-31', 64.50)
ON CONFLICT (tenant_id, vendor_id) DO NOTHING;
