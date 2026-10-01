-- 20261130_005_harden_mdm_survivorship_rls_and_pinning.down.sql
SET search_path = public;

DROP TABLE IF EXISTS public.mdm_batch_rule_snapshot CASCADE;
DROP INDEX IF EXISTS public.uq_semantic_survivorship_active;
DROP FUNCTION IF EXISTS public.get_core_tenant_id();
