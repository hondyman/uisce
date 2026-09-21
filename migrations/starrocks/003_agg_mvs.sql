-- StarRocks Materialized Views for trigger-derived CDC events.
--
-- These rollups are pre-aggregated by async MV refresh for fast OLAP queries
-- without scanning the underlying detailed rows. They sit alongside the base
-- tables defined in 002_agg_layer.sql.
--
-- Apply with:
--   docker exec -i starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root \
--     < migrations/starrocks/003_agg_mvs.sql
--
-- Refresh strategy: ASYNC EVERY (INTERVAL N MINUTE) — no query-time cost,
-- but data lags the base table by up to N minutes. For real-time accuracy,
-- query the base tables instead.

USE agg;

-- ============================================================================
-- 1. Screening funnel — opportunities by stage + average score per stage.
-- Powers "how many opportunities cleared screening this week" dashboards and
-- advisor performance attribution.
--
-- SOURCE OF TRUTH:
--   Post-cutover, `public.investment_opportunities.current_stage` is the
--   in-app system of record (writes come from the screening handler in
--   aggregate-consumer-alpha; the trigger was dropped). This MV *rolls up*
--   that source for analytics. The Postgres-side stage value is the
--   single authoritative row; this MV exists only to denormalize for fast
--   funnel queries.
--   Freshness depends on the consumer being healthy; if the consumer is
--   DLQ-bound or down, the funnel MV silently freezes (data is not wrong
--   but stale). Check the DLQ topic before investigating "MV is wrong".
-- ============================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS alpha_screening_funnel_mv
DISTRIBUTED BY HASH(client_id) BUCKETS 8
REFRESH ASYNC EVERY (INTERVAL 5 MINUTE)
AS
SELECT
    client_id,
    advisor_id,
    current_stage,
    opportunity_type,
    COUNT(*)                                            AS opportunity_count,
    SUM(CASE WHEN screening_passed IS TRUE THEN 1 ELSE 0 END) AS passed_count,
    AVG(screening_score)                                 AS avg_screening_score,
    AVG(target_irr_min)                                  AS avg_target_irr_min,
    SUM(CASE WHEN minimum_commitment IS NOT NULL THEN minimum_commitment ELSE 0 END) AS total_minimum_commitment,
    MAX(stage_updated_at)                                AS last_stage_transition_at
FROM alpha_investment_opportunities
GROUP BY client_id, advisor_id, current_stage, opportunity_type;

-- ============================================================================
-- 2. Rating distribution per template — top-rated templates rollup.
-- Powers "best-rated templates by category" leaderboards.
--
-- SOURCE OF TRUTH:
--   `public.process_templates.rating_average` / `rating_count` is
--   maintained in Postgres by the aggregate-consumer-alpha `rating_stats`
--   handler (replacing the dropped trigger). The MV aggregates the raw
--   `template_ratings` rows directly, so it includes ratings still
--   awaiting moderation_status = 'approved' and is the more accurate
--   "all reactions" view. For in-app display, the Postgres column on
--   process_templates is authoritative (approved-only). The MV's
--   10-minute refresh cadence is acceptable for the leaderboard use case.
-- ============================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS alpha_template_rating_distribution_mv
DISTRIBUTED BY HASH(template_id) BUCKETS 8
REFRESH ASYNC EVERY (INTERVAL 10 MINUTE)
AS
SELECT
    template_id,
    tenant_id,
    moderation_status,
    COUNT(*)                  AS rating_count,
    AVG(rating)               AS avg_rating,
    AVG(CASE WHEN helpful_count > 0
             THEN helpful_count::DOUBLE / (helpful_count + not_helpful_count + 1)
             ELSE NULL END)   AS helpful_ratio,
    SUM(helpful_count)        AS total_helpful,
    SUM(not_helpful_count)    AS total_not_helpful,
    COUNT(DISTINCT reviewer_name)        AS unique_reviewers
FROM alpha_template_ratings
GROUP BY template_id, tenant_id, moderation_status;

-- ============================================================================
-- 3. Crypto throughput — transaction volume / value per asset / day.
-- Powers portfolio analytics dashboards.
-- ============================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS alpha_crypto_throughput_mv
DISTRIBUTED BY HASH(asset_symbol) BUCKETS 8
REFRESH ASYNC EVERY (INTERVAL 5 MINUTE)
AS
SELECT
    asset_symbol,
    blockchain,
    txn_type,
    DATE_TRUNC('day', block_timestamp) AS bucket_day,
    COUNT(*)                            AS txn_count,
    SUM(quantity)                       AS total_quantity,
    SUM(fiat_value_usd)                 AS total_fiat_value_usd,
    SUM(fee_fiat_value_usd)             AS total_fee_usd,
    AVG(price_per_unit_usd)             AS avg_unit_price_usd
FROM alpha_crypto_transactions
WHERE block_timestamp IS NOT NULL
GROUP BY asset_symbol, blockchain, txn_type, DATE_TRUNC('day', block_timestamp);

-- ============================================================================
-- 4. Process execution latency — duration / error rate by workflow_type.
-- Powers SLA dashboards and workflow reliability monitoring.
--
-- The `duration` column in alpha_process_execution_metrics is the
-- Postgres `interval` type, which the stream loader decodes to integer
-- microseconds. Divide by 1000 below to get milliseconds.
-- ============================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS alpha_workflow_latency_mv
DISTRIBUTED BY HASH(workflow_type) BUCKETS 8
REFRESH ASYNC EVERY (INTERVAL 5 MINUTE)
AS
SELECT
    workflow_type,
    step_type,
    status,
    DATE_TRUNC('hour', start_time)       AS bucket_hour,
    COUNT(*)                             AS step_count,
    AVG(duration) / 1000.0               AS avg_duration_ms,
    PERCENTILE_APPROX(duration, 0.5)     / 1000.0 AS p50_duration_ms,
    PERCENTILE_APPROX(duration, 0.95)    / 1000.0 AS p95_duration_ms,
    PERCENTILE_APPROX(duration, 0.99)    / 1000.0 AS p99_duration_ms,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failure_count,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS success_count
FROM alpha_process_execution_metrics
WHERE start_time IS NOT NULL
GROUP BY workflow_type, step_type, status, DATE_TRUNC('hour', start_time);

-- ============================================================================
-- 5. Security identifier coverage — how many securities have each ID type.
-- Powers identifier-completeness dashboards.
--
-- The aggregate-consumer-crims identifier_cache handler replicates the
-- `isin/cusip/ticker` denorm columns on `crims.orm.security` from the
-- primary identifier rows here. This MV measures identifier-coverage
-- upstream; the orm.security denorm is a downstream effect, not a
-- authoritative stat.
-- ============================================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS crims_identifier_coverage_mv
DISTRIBUTED BY HASH(id_type) BUCKETS 4
REFRESH ASYNC EVERY (INTERVAL 30 MINUTE)
AS
SELECT
    id_type,
    COUNT(DISTINCT security_id)        AS security_count,
    SUM(CASE WHEN is_primary  THEN 1 ELSE 0 END) AS primary_count,
    SUM(CASE WHEN is_alternate THEN 1 ELSE 0 END) AS alternate_count,
    SUM(CASE WHEN effective_to IS NULL THEN 1 ELSE 0 END) AS active_count,
    COUNT(DISTINCT source)             AS source_count
FROM crims_security_identifier
GROUP BY id_type;
