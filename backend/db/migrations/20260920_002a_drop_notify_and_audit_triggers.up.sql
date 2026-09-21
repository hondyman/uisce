-- Drop redundant triggers identified by pg_trigger audit 2026-09-20
-- Phase A: NOTIFY triggers + misnamed "audit" triggers (with zero-behavior-change replacements)
--
-- What this removes:
--   Phase 1a  audit_investment_opportunities  — in-DB audit writer; Debezium covers investment_opportunities CDC
--   Phase 1b  notify_security_change / notify_metrics_registry_changed  — NOTIFY-only; no active LISTEN consumers
--   Phase 1c  audit_uma_accounts / audit_uma_rebalance_requests / audit_validation_patterns  —
--               misnamed "audit" triggers; body is just NEW.updated_at = now(); no actual audit
--
-- What this keeps:
--   oms.default_order_status (4 triggers) — status defaults + limit_price/stop_price domain enforcement
--   screen_investment_opportunity — real scoring logic
--   create_default_template_permissions — seeds RBAC rows on template create
--   cube_*_version_trigger — correctly gated with IS DISTINCT FROM
--   update_template_rating_stats (3 triggers) — aggregate maintenance; keep for now
--   calculate_step_duration — pure NEW mutation
--   crypto_* triggers — user decision: keep in-DB for transactional consistency
--   sync_identifier_cache (crims.orm.security_identifier) — low-volume cross-table sync
--
-- What is NOT dropped (deferred decisions):
--   trigger_capital_call_liquidity_check — user deferred
--   trigger_create_version_on_update bug (versions on every UPDATE) — user deferred fix
--   semantic_cube_cache (Phase B)

BEGIN;

-- ============================================================================
-- Phase 1a: in-DB audit writer — replaced by Debezium CDC on investment_opportunities
-- workflow_audit_log table is the canonical app-level audit log; it stays and
-- continues to be written by audit/service.go, timeout_monitor.go, and others.
-- ============================================================================

DROP TRIGGER IF EXISTS audit_investment_opportunities
    ON public.investment_opportunities;
DROP FUNCTION  IF EXISTS public.log_workflow_audit();

-- ============================================================================
-- Phase 1b: NOTIFY-only triggers — no active LISTEN consumers confirmed; redundant
-- with Debezium CDC topics (orm_oms.security_user_fund_access, public.metrics_registry)
-- Verified: no LISTEN calls on these channels in codebase.
-- ============================================================================

DROP TRIGGER IF EXISTS trg_notify_security_change
    ON public.security_user_fund_access;
DROP FUNCTION  IF EXISTS public.notify_security_change();

DROP TRIGGER IF EXISTS metrics_registry_notify_trigger
    ON public.metrics_registry;
DROP FUNCTION  IF EXISTS public.notify_metrics_registry_changed();

-- ============================================================================
-- Phase 1c: misnamed "audit" triggers — body is only NEW.updated_at = now();
-- no actual audit behaviour.
--
-- IMPORTANT: replacement touchers are installed BEFORE the drop so updated_at
-- behaviour is preserved. public.update_updated_at_column() exists in this DB
-- (used by ~140 other triggers in alpha).
-- ============================================================================

-- Install standard updated_at touchers first (zero behaviour change)
CREATE TRIGGER update_uma_accounts_updated_at
    BEFORE UPDATE ON public.uma_accounts
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_uma_rebalance_requests_updated_at
    BEFORE UPDATE ON public.uma_rebalance_requests
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

CREATE TRIGGER update_validation_patterns_updated_at
    BEFORE UPDATE ON public.validation_patterns
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- Now drop the misnamed old functions (no behaviour change)
DROP TRIGGER IF EXISTS trigger_audit_uma_accounts
    ON public.uma_accounts;
DROP FUNCTION  IF EXISTS public.audit_uma_accounts();

DROP TRIGGER IF EXISTS trigger_audit_uma_rebalance_requests
    ON public.uma_rebalance_requests;
DROP FUNCTION  IF EXISTS public.audit_uma_rebalance_requests();

DROP TRIGGER IF EXISTS trigger_audit_validation_patterns
    ON public.validation_patterns;
DROP FUNCTION  IF EXISTS public.audit_validation_patterns();

COMMIT;
