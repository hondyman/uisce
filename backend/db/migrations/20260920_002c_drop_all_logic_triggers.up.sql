-- Drop all remaining non-updated_at triggers from alpha + crims
-- Per directive: "only want triggers for updated_at; all others evaluated for CDC"
--
-- Execution order: 002a → 002b → 002d (oms status defaults) → 002e (oms CHECKs) → 002c → crims
--
-- Guards in place before this runs:
--   002d:  status_id column DEFAULTs on oms.orders/slice/allocation/settlement
--   002e:  CHECK constraints for limit_price/stop_price vs order_type on oms.orders
--          (replicate the RAISE EXCEPTION rejection that default_order_status did)
--
-- crypto_holdings aggregates: this IS the drop point.
--   trigger_update_holdings and trigger_update_crypto_valuations are removed here.
--   crypto_holdings becomes CDC-managed via Redpanda stream consumer with accepted
--   eventual-consistency lag (seconds). If any read path requires the aggregate
--   immediately after a confirmed transaction, that read will get stale data —
--   acceptable per user decision; revisit if that assumption changes.
--
-- Screening handler (investment_opportunities): STRICTEST PARITY CHECK REQUIRED.
--   The screening handler is the only consumer whose output feeds business
--   decisions (screening_passed, stage advancement). Unlike version/rating
--   handlers where a miss is a staleness window, a wrong screening score is
--   a data integrity issue. The handler implements self-write echo filtering
--   via ChangedColumns detection; parity window must verify that:
--     1. screening_score matches the Go scoring formula for every row
--     2. current_stage transitions INTAKE→INITIAL_SCREEN are identical to
--        what the trigger produced
--     3. No CDC feedback loops (consumer writes must not re-trigger itself)
--
-- Functions dropped (alpha + crims):
--   oms.default_order_status
--   public.update_holdings_on_transaction
--   public.update_crypto_holding_valuation
--   public.refresh_latest_prices
--   public.screen_investment_opportunity
--   vend.create_default_template_permissions
--   vend.create_template_version_on_update
--   public.cube_custom_model_version_trigger
--   public.cube_security_policy_version_trigger
--   public.update_template_rating_stats
--   public.calculate_step_duration
--   public.check_capital_call_liquidity
--   orm.sync_identifier_cache  (crims)
--
-- Authoritative function bodies: backend/db/migrations/_reference/20260920_002_trigger_fn_sources.sql

BEGIN;

-- ============================================================================
-- oms: status-default triggers with domain enforcement (RAISE EXCEPTION on bad data)
-- ============================================================================

DROP TRIGGER IF EXISTS trg_orders_default_status      ON oms.orders;
DROP TRIGGER IF EXISTS trg_order_slice_default_status ON oms.order_slice;
DROP TRIGGER IF EXISTS trg_allocation_default_status  ON oms.allocation;
DROP TRIGGER IF EXISTS trg_settlement_default_status  ON oms.settlement;
DROP FUNCTION  IF EXISTS oms.default_order_status();

-- ============================================================================
-- crypto: holdings aggregate maintenance (eventual consistency via CDC)
-- ============================================================================

DROP TRIGGER IF EXISTS trigger_update_holdings ON public.crypto_transactions;
DROP FUNCTION  IF EXISTS public.update_holdings_on_transaction();

DROP TRIGGER IF EXISTS trigger_update_crypto_valuations ON public.crypto_market_data;
DROP FUNCTION  IF EXISTS public.update_crypto_holding_valuation();

DROP TRIGGER IF EXISTS trigger_refresh_prices ON public.crypto_prices;
DROP FUNCTION  IF EXISTS public.refresh_latest_prices();

-- ============================================================================
-- investment_opportunities: real scoring logic
-- ============================================================================

DROP TRIGGER IF EXISTS trg_screen_investment_opportunity ON public.investment_opportunities;
DROP FUNCTION  IF EXISTS public.screen_investment_opportunity();

-- ============================================================================
-- semantic_query_templates: RBAC seeding + version-on-update
-- ============================================================================

DROP TRIGGER IF EXISTS trigger_create_default_permissions ON public.semantic_query_templates;
DROP FUNCTION  IF EXISTS vend.create_default_template_permissions();

DROP TRIGGER IF EXISTS trigger_create_version_on_update ON public.semantic_query_templates;
DROP FUNCTION  IF EXISTS vend.create_template_version_on_update();

-- ============================================================================
-- cube: version-bump triggers (correctly gated; still dropping per directive)
-- ============================================================================

DROP TRIGGER IF EXISTS cube_custom_model_version ON public.cube_custom_models;
DROP FUNCTION  IF EXISTS public.cube_custom_model_version_trigger();

DROP TRIGGER IF EXISTS cube_security_policy_version ON public.cube_security_policies;
DROP FUNCTION  IF EXISTS public.cube_security_policy_version_trigger();

-- ============================================================================
-- template_ratings: aggregate maintenance (3 triggers, 1 function)
-- ============================================================================

DROP TRIGGER IF EXISTS template_rating_inserted ON public.template_ratings;
DROP TRIGGER IF EXISTS template_rating_updated  ON public.template_ratings;
DROP TRIGGER IF EXISTS template_rating_deleted  ON public.template_ratings;
DROP FUNCTION  IF EXISTS public.update_template_rating_stats();

-- ============================================================================
-- process_execution_metrics: duration computation
-- ============================================================================
--
-- DROPPED: process_metrics handler (handler 5 in the CDC consumer plan).
--
-- Decision basis: grep of the Go service layer confirmed that
-- backend/internal/services/scheduler_service.go:RecordStepCompletion already
-- computes duration = end_time - start_time explicitly in its UPDATE query:
--
--   UPDATE process_execution_metrics
--   SET duration = EXTRACT(EPOCH FROM (end_time - start_time))
--
-- The trigger was therefore dead code — it would have fired after the same
-- UPDATE that already set the column, doing a no-op UPDATE that re-emitted a
-- CDC event with no actual change to the row. Dropping it removes a redundant
-- DB round-trip and eliminates one source of CDC noise without any behavioral
-- change.
--
-- Strictest parity check in the go-live window: diff process_execution_metrics.duration
-- per pipeline execution vs the consumer-recomputed value from the same source
-- data. Expected: zero diffs.

DROP TRIGGER IF EXISTS calculate_process_execution_metrics_duration ON public.process_execution_metrics;
DROP FUNCTION  IF EXISTS public.calculate_step_duration();

-- ============================================================================
-- capital_calls: stub liquidity check (always passes; deferred decision overridden)
-- ============================================================================

DROP TRIGGER IF EXISTS trigger_capital_call_liquidity_check ON public.capital_calls;
DROP FUNCTION  IF EXISTS public.check_capital_call_liquidity();

-- ============================================================================
-- crims: identifier cache sync
-- NOTE: run against crims database separately — not part of alpha migration
-- ============================================================================
-- Run this against crims:
--   DROP TRIGGER IF EXISTS trg_sec_ident_cache ON orm.security_identifier;
--   DROP FUNCTION  IF EXISTS orm.sync_identifier_cache();

COMMIT;
