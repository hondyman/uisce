-- The calculations table backs models.Calculation / SemanticCalculationService
-- (backend/internal/analytics/semantic_calculation_service.go): catalog-node-
-- attached calculation definitions (formula, engine, materialization, etc.).
-- This table was never created by a migration even though CreateCalculation/
-- UpdateCalculation/ListCalculations/GetCalculationByName have referenced it
-- since introduction; 20260902_001_calculations_tier_persistence.up.sql adds
-- columns to it via ALTER TABLE and fails without this first.

CREATE TABLE IF NOT EXISTS calculations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id         UUID NOT NULL REFERENCES catalog_node(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    title           VARCHAR(255),
    description     TEXT,
    formula         TEXT NOT NULL,
    engine_type     VARCHAR(50),
    return_type     VARCHAR(50),
    arguments       JSONB NOT NULL DEFAULT '{}'::jsonb,
    category        VARCHAR(100),
    subcategory     VARCHAR(100),
    domain_id       UUID REFERENCES business_objects(id) ON DELETE SET NULL,
    execution_type  VARCHAR(50),
    engine          VARCHAR(50),
    is_materialized BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(name)
);

CREATE INDEX IF NOT EXISTS idx_calculations_node ON calculations(node_id);
CREATE INDEX IF NOT EXISTS idx_calculations_domain ON calculations(domain_id);

COMMENT ON TABLE calculations IS
  'Catalog-node-attached calculation definitions (name, formula, engine, materialization). See SemanticCalculationService.';
