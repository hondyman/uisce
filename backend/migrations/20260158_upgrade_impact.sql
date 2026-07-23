-- Migration: 20260158_upgrade_impact.sql
-- Goal: Store upgrade impact analysis for core-to-tenant reconciliations

CREATE TABLE IF NOT EXISTS semantic.upgrade_impacts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    core_page_id uuid NOT NULL REFERENCES semantic.pages(id),
    core_old_version int NOT NULL,
    core_new_version int NOT NULL,
    tenant_id text NOT NULL,
    overlay_page_id uuid NOT NULL REFERENCES semantic.page_overlays(id),
    summary text,
    conflicts jsonb NOT NULL DEFAULT '[]',
    inherited_changes jsonb NOT NULL DEFAULT '[]',
    new_core_components text[] NOT NULL DEFAULT '{}',
    removed_core_components text[] NOT NULL DEFAULT '{}',
    status text NOT NULL DEFAULT 'pending', -- pending, accepted, partially_applied, dismissed
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- Index for fast lookup by tenant and page
CREATE INDEX IF NOT EXISTS idx_upgrade_impacts_tenant_page ON semantic.upgrade_impacts(tenant_id, core_page_id);
