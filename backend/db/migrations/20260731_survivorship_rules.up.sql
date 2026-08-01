-- Migration: Multi-Source Golden Record Survivorship Engine (Pillar 2)
-- Date: 2026-07-31
-- Purpose: Persists per-tenant, per-business-object field-level survivorship
--          strategies used by mdm.SurvivorshipEngine to merge heterogeneous
--          source payloads (Bloomberg, Refinitiv, CRIMS, etc.) into a single
--          authoritative Golden Record consumed by vm.Project() → FastRecord.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'survivorship_strategy_enum') THEN
        CREATE TYPE survivorship_strategy_enum AS ENUM (
            'SOURCE_PRIORITY',
            'MOST_RECENT',
            'CONSERVATIVE_MIN',
            'CONSERVATIVE_MAX',
            'WEIGHTED_CONFIDENCE'
        );
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS public.survivorship_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    bo_id VARCHAR(128) NOT NULL REFERENCES public.legacy_business_objects(bo_id),
    field_name VARCHAR(100) NOT NULL,
    strategy survivorship_strategy_enum NOT NULL DEFAULT 'SOURCE_PRIORITY',
    priority_order TEXT[], -- e.g. ARRAY['BLOOMBERG', 'REFINITIV', 'CRIMS']
    max_stale_seconds INT DEFAULT 0, -- 0 = no staleness check
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT uk_tenant_bo_field_survivorship UNIQUE (tenant_id, bo_id, field_name)
);

CREATE INDEX IF NOT EXISTS idx_survivorship_lookup ON public.survivorship_rules(tenant_id, bo_id);
