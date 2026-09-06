-- Migration: force RLS on 14 tenant-bearing tables that have RLS enabled
-- but not forced (table owner can still bypass policies).
--
-- Identified via:
--   SELECT c.relname FROM pg_class c JOIN pg_namespace n ON c.relnamespace=n.oid
--   WHERE n.nspname='public' AND c.relkind='r'
--     AND c.relrowsecurity AND NOT c.relforcerowsecurity;
--
-- Of the 14, four carry a 'tenant_id' column and represent tenant-scoped
-- data that a table owner could currently bypass:
--   - calc_fields
--   - notification_outbox
--   - okf_concept_manifest
--   - semantic_term_tags
--
-- The remaining 10 are deliberately admin/global tables (tenants, audit
-- logs, impersonation, role definitions, etc.); forcing them would block
-- legitimate cross-tenant admin operations. This migration only forces the
-- four tenant-bearing tables — a follow-up migration (separate file)
-- will address the broader 56-row gap list with policy authoring.

BEGIN;

ALTER TABLE public.calc_fields          FORCE ROW LEVEL SECURITY;
ALTER TABLE public.notification_outbox  FORCE ROW LEVEL SECURITY;
ALTER TABLE public.okf_concept_manifest FORCE ROW LEVEL SECURITY;
ALTER TABLE public.semantic_term_tags   FORCE ROW LEVEL SECURITY;

COMMIT;
