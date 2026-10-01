-- 20261130_008_create_bp_trigger_subscriptions.up.sql
-- Lookup table for cross-tenant event trigger subscriptions (used by bpbridge and compiler).
--
-- ARCHITECTURAL DECISION & RLS EXEMPTION:
-- 1. This table is deliberately NOT tenant-RLS'd: it serves as a lightweight routing index
--    mapping incoming Kafka/Redpanda events to candidate tenant workflow definitions.
-- 2. bpbridge performs cross-tenant event matching against this index before dispatching.
-- 3. WRITE ACCESS: Compiler only, maintained during compile/publish operations.
-- 4. DATA ISOLATION: This table contains ONLY routing metadata (topic, event_type, ops, version).
--    Tenant business data, full step graphs, payloads, and extension definitions MUST NEVER
--    be stored in this table; all business definitions remain strictly isolated in public.bp_process_definition.
SET search_path = public;

CREATE TABLE IF NOT EXISTS public.bp_trigger_subscriptions (
    tenant_id      UUID NOT NULL,
    process_id     VARCHAR(255) NOT NULL,
    trigger_type   VARCHAR(20) NOT NULL,          -- 'event'
    topic          VARCHAR(255) NOT NULL,
    event_type     VARCHAR(255),                  -- optional filter
    ops            JSONB,                          -- CDC op filter
    version        INT NOT NULL,                   -- definition version this came from
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, process_id, topic)
);

CREATE INDEX IF NOT EXISTS idx_bp_trigger_sub_topic ON public.bp_trigger_subscriptions (topic);

-- Formalize app_user grants
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON public.bp_trigger_subscriptions TO app_user;
    END IF;
END $$;
