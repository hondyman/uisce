-- StarRocks OLAP layer for trigger-derived CDC events.
--
-- The alpha-trg and crims-trg Debezium connectors feed 15 topics covering:
--   alpha_trg.public.template_ratings
--   alpha_trg.public.semantic_query_templates
--   alpha_trg.public.cube_custom_models
--   alpha_trg.public.cube_security_policies
--   alpha_trg.public.investment_opportunities
--   alpha_trg.public.process_execution_metrics
--   alpha_trg.public.security_user_fund_access
--   alpha_trg.public.metrics_registry
--   alpha_trg.public.crypto_transactions
--   alpha_trg.public.crypto_market_data
--   alpha_trg.public.crypto_prices
--   alpha_trg.public.uma_accounts
--   alpha_trg.public.uma_rebalance_requests
--   alpha_trg.public.validation_patterns
--   crims_trg.orm.security_identifier
--
-- Stream-loader services consume each topic and stream-load the after-image
-- row into one table each. This file defines the destination tables in the
-- `agg` database (separate from `oms` for the OMS execution pipeline).
--
-- Apply with:
--   docker exec -i starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root \
--     < migrations/starrocks/002_agg_layer.sql
--
-- Table-naming convention: <source_schema>_<source_table> so topic-to-table
-- mapping is unambiguous (e.g. alpha_trg.public.investment_opportunities →
-- alpha_investment_opportunities in StarRocks).
--
-- ENGINE = olap with PRIMARY KEY (not DUPLICATE KEY) is required so the
-- stream-loader can emit StarRocks DELETE for op=d events. With DUPLICATE
-- KEY model, deletes would land as no-op upserts, leaving phantom rows that
-- the Postgres source no longer has. PRIMARY KEY with `__op=1` column in
-- the stream load applies a true delete on PK match.
--
-- PK columns (verified against Postgres pg_index on 2026-09-21):
--   alpha_investment_opportunities       → opportunity_id  (verified Postgres PK)
--   alpha_process_execution_metrics      → id
--   alpha_template_ratings               → id
--   alpha_crypto_transactions            → id
--   alpha_crypto_market_data             → tick_id
--   alpha_semantic_query_templates       → id
--   alpha_cube_custom_models             → id
--   alpha_cube_security_policies         → id
--   alpha_security_user_fund_access      → id
--   alpha_metrics_registry               → id
--   alpha_uma_accounts                   → id
--   alpha_uma_rebalance_requests         → id
--   alpha_validation_patterns            → id
--   alpha_crypto_prices                  → id
--   crims_security_identifier            → id
-- Stream-loader compose env var PRIMARY_KEY_COLUMN points each loader at
-- the correct PK (defaults to "id"; investment opportunities uses
-- "opportunity_id" explicitly).
--
-- Delete shape (TBD — pending integration test; see RUNBOOK.md):
--   The stream-loader sends StarRocks stream-load DELETE for op=d events
--   via the columns-projection pattern:
--     columns: __op=1,<pk_col>
--     body:    {"<pk_col>": "<value>"}
--   The integration test in RUNBOOK.md (StarRocks delete integration test)
--   determines which DDL shape pairs with this projection:
--     - Variant 1b winner: add `__op INT DEFAULT 0` to each table DDL
--     - Variant 2 winner:  no `__op` column needed in DDL
--   After the test completes, replace this block with the proven pattern
--   AND paste the exact curl that worked in RUNBOOK.md so the oms loader
--   conversion later (post-cutover) can copy verbatim.
--   Until then: deletes sent through the loader WILL land as nothing —
--   the loader's columns projection will be rejected by StarRocks without
--   the matching DDL/config, which is a safe failure mode (rows go stale
--   instead of corrupting).
--
-- Note: the existing oms.* tables (defined outside this file) are still
-- DUPLICATE KEY — the orm_oms pipeline predates the delete-handling fix
-- and doesn't yet emit deletes. The PK-column convention documented
-- above applies to oms loader conversion as well. Post-cutover backlog
-- item: convert oms.* to PRIMARY KEY + add delete handling.

-- ============================================================================
-- Database
-- ============================================================================
CREATE DATABASE IF NOT EXISTS agg;
USE agg;

-- ============================================================================
-- 1. Investment Opportunities (screening funnel)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_investment_opportunities (
    opportunity_id              VARCHAR(36),       -- uuid -> VARCHAR(36)
    client_id                   VARCHAR(36),
    advisor_id                  VARCHAR(36),
    opportunity_type            VARCHAR(50),
    fund_name                   VARCHAR(65533),    -- text -> VARCHAR(MAX)
    general_partner             VARCHAR(65533),
    strategy                     VARCHAR(100),
    sub_strategy                VARCHAR(100),
    vintage_year                 INT,
    fund_size                    DECIMAL(15, 2),
    minimum_commitment           DECIMAL(15, 2),
    target_commitment            DECIMAL(15, 2),
    target_irr_min               DECIMAL(5, 2),
    target_irr_max               DECIMAL(5, 2),
    target_tvpi_min              DECIMAL(5, 2),
    target_vintage_year_range    JSON,               -- jsonb -> JSON
    max_leverage_ratio           DECIMAL(5, 2),
    manager_aum_min              DECIMAL(15, 2),
    track_record_years_min       INT,
    screening_passed             BOOLEAN,
    screening_reasons            ARRAY<VARCHAR(65533)>,  -- text[] -> ARRAY<VARCHAR>
    screening_score              DECIMAL(5, 2),
    screening_completed_at       DATETIME,
    current_stage                VARCHAR(50),
    stage_updated_at             DATETIME,
    stage_history                JSON,
    advisor_notes                VARCHAR(65533),
    updated_at                   DATETIME
) ENGINE = olap
PRIMARY KEY (opportunity_id)
DISTRIBUTED BY HASH(opportunity_id) BUCKETS 16
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 2. Process Execution Metrics (workflow telemetry)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_process_execution_metrics (
    id                  VARCHAR(36),
    workflow_id         VARCHAR(255),
    workflow_type       VARCHAR(255),
    tenant_id           VARCHAR(36),
    step_name           VARCHAR(255),
    step_type           VARCHAR(100),
    start_time          DATETIME,
    end_time            DATETIME,
    duration            BIGINT,     -- interval → microseconds
    status              VARCHAR(50),
    error_message       VARCHAR(65533),
    resource_usage      JSON,
    updated_at          DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(id) BUCKETS 16
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 3. Template Ratings (UX feedback)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_template_ratings (
    id                  VARCHAR(36),
    template_id         VARCHAR(36),
    tenant_id           VARCHAR(36),
    datasource_id       VARCHAR(36),
    rating              INT,
    review_text         VARCHAR(65533),
    review_title        VARCHAR(255),
    reviewer_name       VARCHAR(255),
    reviewer_role       VARCHAR(100),
    helpful_count       INT,
    not_helpful_count   INT,
    is_verified_user    BOOLEAN,
    is_moderated        BOOLEAN,
    moderation_status   VARCHAR(50),
    created_at          DATETIME,
    updated_at          DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(id) BUCKETS 8
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 4. Crypto Transactions (portfolio analytics)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_crypto_transactions (
    id                  VARCHAR(36),
    wallet_id           VARCHAR(36),
    blockchain          VARCHAR(50),
    txn_hash            VARCHAR(65533),
    block_number        BIGINT,
    block_timestamp     DATETIME,
    txn_type            VARCHAR(50),
    asset_symbol        VARCHAR(20),
    contract_address    VARCHAR(65533),
    quantity            DECIMAL(30, 18),
    fiat_value_usd      DECIMAL(15, 2),
    price_per_unit_usd  DECIMAL(15, 8),
    fee_asset_symbol    VARCHAR(20),
    fee_quantity        DECIMAL(30, 18),
    fee_fiat_value_usd  DECIMAL(15, 2),
    from_address        VARCHAR(65533),
    to_address          VARCHAR(65533),
    created_at          DATETIME,
    updated_at          DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(wallet_id) BUCKETS 16
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 5. Crypto Market Data (price timeseries)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_crypto_market_data (
    tick_id                  VARCHAR(36),
    asset_symbol             VARCHAR(20),
    price_usd                DECIMAL(15, 8),
    volume_24h               DECIMAL(18, 2),
    market_cap               DECIMAL(18, 2),
    price_change_24h_pct     DECIMAL(6, 4),
    price_change_7d_pct      DECIMAL(6, 4),
    open_price               DECIMAL(15, 8),
    high_price               DECIMAL(15, 8),
    low_price                DECIMAL(15, 8),
    close_price              DECIMAL(15, 8),
    data_source              VARCHAR(50),
    tick_timestamp           DATETIME,
    updated_at               DATETIME
) ENGINE = olap
PRIMARY KEY (tick_id)
DISTRIBUTED BY HASH(asset_symbol) BUCKETS 8
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 6. Semantic Query Templates
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_semantic_query_templates (
    id                  VARCHAR(36),
    tenant_id           VARCHAR(36),
    name                VARCHAR(255),
    description         VARCHAR(65533),
    datasource          VARCHAR(255),
    version             VARCHAR(50),
    semantic_query      JSON,
    parameters          JSON,
    visibility          VARCHAR(50),
    tags                ARRAY<VARCHAR(65533)>,
    deprecated          BOOLEAN,
    deprecation_reason  VARCHAR(65533),
    created_by          VARCHAR(255),
    created_at          DATETIME,
    updated_by          VARCHAR(255),
    updated_at          DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(id) BUCKETS 8
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 7. Cube Custom Models (configuration registry)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_cube_custom_models (
    id              VARCHAR(36),
    tenant_id       VARCHAR(36),
    datasource_id   VARCHAR(36),
    core_model_id   VARCHAR(36),
    name            VARCHAR(255),
    description     VARCHAR(65533),
    extension_type  VARCHAR(50),
    custom_config   JSON,
    version         INT,
    is_active       BOOLEAN,
    created_by      VARCHAR(36),
    created_at      DATETIME,
    updated_at      DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(id) BUCKETS 8
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 8. Cube Security Policies (RBAC rules)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_cube_security_policies (
    id              VARCHAR(36),
    tenant_id       VARCHAR(36),
    name            VARCHAR(255),
    description     VARCHAR(65533),
    policy_type     VARCHAR(50),
    priority        INT,
    enabled         BOOLEAN,
    target_cubes    JSON,
    target_members  JSON,
    conditions      JSON,
    effects         JSON,
    version         INT,
    created_by      VARCHAR(36),
    created_at      DATETIME,
    updated_at      DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(id) BUCKETS 8
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 9. Security User-Fund Access (denorm cache)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_security_user_fund_access (
    id                  VARCHAR(36),
    tenant_id           VARCHAR(36),
    user_id             VARCHAR(65533),
    fund_node_id        VARCHAR(36),
    access_source       VARCHAR(65533),
    source_fund_node_id VARCHAR(36),
    granted_at          DATETIME,
    updated_at          DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(tenant_id, user_id) BUCKETS 8
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 10. Metrics Registry (formula catalogue)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_metrics_registry (
    id                  INT,
    node_id             VARCHAR(255),
    schema_domain       VARCHAR(100),
    category            VARCHAR(100),
    description         VARCHAR(65533),
    formula_type        VARCHAR(50),
    formula             VARCHAR(65533),
    arguments           JSON,
    badge               VARCHAR(10),
    function_class      VARCHAR(50),
    functions_used      ARRAY<VARCHAR(65533)>,
    governance_status   VARCHAR(50),
    updated_at          DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(id) BUCKETS 4
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 11. UMA Accounts (managed-account billing)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_uma_accounts (
    id                  VARCHAR(36),
    tenant_id           VARCHAR(36),
    datasource_id       VARCHAR(36),
    name                VARCHAR(255),
    status              VARCHAR(50),
    aum                 DECIMAL(19, 2),
    target_allocation   JSON,
    updated_at          DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(tenant_id) BUCKETS 8
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 12. UMA Rebalance Requests
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_uma_rebalance_requests (
    id              VARCHAR(36),
    tenant_id       VARCHAR(36),
    datasource_id   VARCHAR(36),
    uma_account_id  VARCHAR(36),
    request_type    VARCHAR(50),
    reason          VARCHAR(65533),
    initiated_by    VARCHAR(36),
    updated_at      DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(uma_account_id) BUCKETS 8
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 13. Validation Patterns (catalog of regex rules)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_validation_patterns (
    id              VARCHAR(36),
    tenant_id       VARCHAR(36),
    name            VARCHAR(65533),
    description     VARCHAR(65533),
    regex_pattern   VARCHAR(65533),
    category        VARCHAR(65533),
    examples        ARRAY<VARCHAR(65533)>,
    is_active       BOOLEAN,
    created_at      DATETIME,
    updated_at      DATETIME,
    created_by      VARCHAR(36)
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(tenant_id) BUCKETS 8
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 14. Crypto Prices (one-row-per-asset)
-- ============================================================================
CREATE TABLE IF NOT EXISTS alpha_crypto_prices (
    id                  VARCHAR(36),
    asset_symbol        VARCHAR(20),
    price_usd           DECIMAL(15, 8),
    market_cap_usd      DECIMAL(20, 2),
    volume_24h_usd      DECIMAL(20, 2),
    change_1h_pct       DECIMAL(8, 4),
    change_24h_pct      DECIMAL(8, 4),
    updated_at          DATETIME
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(asset_symbol) BUCKETS 4
PROPERTIES ("replication_num" = "1");

-- ============================================================================
-- 15. Security Identifier (crims) — referenced by identifier_cache handler
-- ============================================================================
CREATE TABLE IF NOT EXISTS crims_security_identifier (
    id                  VARCHAR(36),
    security_id         VARCHAR(36),
    id_type             VARCHAR(20),
    id_value            VARCHAR(100),
    is_primary          BOOLEAN,
    is_alternate        BOOLEAN,
    effective_from      DATE,
    effective_to        DATE,
    source              VARCHAR(50),
    custom_attributes   JSON,
    created_at          DATETIME,
    updated_at          DATETIME,
    tenant_id           VARCHAR(36)
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(security_id) BUCKETS 8
PROPERTIES ("replication_num" = "1");
