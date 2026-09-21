-- REPLICA IDENTITY FULL on tables where the CDC consumer needs the BEFORE image
-- (the consumer's dedupe-on-source_lsn logic + per-handler state comparisons rely on
--  Debezium emitting the full OLD row in before:).
--
-- Tables where REPLICA IDENTITY FULL is REQUIRED (consumer logic reads OLD.* or
--  transitions keyed on OLD state):
--   crypto_transactions           — handler compares OLD.status vs NEW.status to
--                                   detect the CONFIRMED transition
--   template_ratings              — handler recomputes rating_average/count on the
--                                   moderation_status='approved' subset
--   semantic_query_templates      — handler checks OLD.* IS DISTINCT FROM NEW.*
--                                   to decide whether to version
--   investment_opportunities      — screening handler reads OLD.current_stage to
--                                   auto-advance only on INTAKE
--   process_execution_metrics     — step_duration handler compared OLD.end_time
--                                   IS NULL vs NEW.end_time IS NOT NULL
--                                   (NB: this handler was dropped from final scope;
--                                   REPLICA IDENTITY FULL still desirable for
--                                   future CDC consumers on this table)
--
-- Tables where REPLICA IDENTITY DEFAULT (f) is sufficient (event-only sources,
--  no OLD comparison in handler logic):
--   crypto_market_data, crypto_prices, cube_custom_models, cube_security_policies,
--   security_user_fund_access, metrics_registry, uma_accounts, uma_rebalance_requests,
--   validation_patterns
--
-- Note on cost: REPLICA IDENTITY FULL costs a few extra bytes per UPDATE/DELETE WAL
--  record (the full OLD tuple). At current zero-volume tables this is irrelevant.
--
-- Also creates the narrow publication consumed by the alpha-trg connector.
--  Mirrors the pattern of orm_cdc_publication / iam_security_publication:
--  one intent-specific publication rather than reusing dbz_publication_alpha
--  (which covers 1491 tables).

BEGIN;

-- ---------------------------------------------------------------------------
-- REPLICA IDENTITY FULL
-- ---------------------------------------------------------------------------

ALTER TABLE public.crypto_transactions        REPLICA IDENTITY FULL;
ALTER TABLE public.template_ratings           REPLICA IDENTITY FULL;
ALTER TABLE public.semantic_query_templates   REPLICA IDENTITY FULL;
ALTER TABLE public.investment_opportunities   REPLICA IDENTITY FULL;
ALTER TABLE public.process_execution_metrics  REPLICA IDENTITY FULL;

-- ---------------------------------------------------------------------------
-- Publication
-- ---------------------------------------------------------------------------

DROP PUBLICATION IF EXISTS dropped_trigger_source_publication;

CREATE PUBLICATION dropped_trigger_source_publication FOR TABLE
    public.crypto_transactions,
    public.crypto_market_data,
    public.crypto_prices,
    public.template_ratings,
    public.semantic_query_templates,
    public.cube_custom_models,
    public.cube_security_policies,
    public.investment_opportunities,
    public.process_execution_metrics,
    public.security_user_fund_access,
    public.metrics_registry,
    public.uma_accounts,
    public.uma_rebalance_requests,
    public.validation_patterns;

COMMIT;
