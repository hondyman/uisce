-- migrations/045_observability.sql

CREATE TABLE IF NOT EXISTS edm.wasm_module_version (
    wasm_version_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_name       TEXT NOT NULL,
    version           TEXT NOT NULL,
    build_hash        TEXT NOT NULL,
    build_time        TIMESTAMP NOT NULL,
    artifact_uri      TEXT NOT NULL,
    checksum_sha256   TEXT NOT NULL,
    is_active         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMP NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ux_wasm_module_version_name_version
    ON edm.wasm_module_version (module_name, version);


CREATE TABLE IF NOT EXISTS edm.rule_lineage (
    rule_lineage_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    etl_run_id      UUID NOT NULL REFERENCES edm.etl_run(run_id),
    rule_id         UUID NOT NULL, -- references edm.compliance_rule
    portfolio_id    UUID NOT NULL,
    valuation_date  DATE NOT NULL,
    status          TEXT NOT NULL, -- PASS | FAIL
    metric_value    NUMERIC,
    threshold_value NUMERIC
);

CREATE INDEX idx_rule_lineage_rule
    ON edm.rule_lineage (rule_id, valuation_date DESC);

CREATE INDEX idx_rule_lineage_portfolio
    ON edm.rule_lineage (portfolio_id, valuation_date DESC);


CREATE TABLE IF NOT EXISTS edm.scenario_lineage (
    scenario_lineage_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    etl_run_id          UUID NOT NULL REFERENCES edm.etl_run(run_id),
    scenario_id         UUID NOT NULL, -- references edm.risk_scenario
    portfolio_id        UUID NOT NULL,
    valuation_date      DATE NOT NULL,
    pnl                 NUMERIC
);

CREATE INDEX idx_scenario_lineage_scenario
    ON edm.scenario_lineage (scenario_id, valuation_date DESC);
