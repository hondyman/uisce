-- Migration: 20260910_001_report_personal_and_favorites.up.sql
-- Description:
-- 1. Add `is_personal` (boolean) and `created_by_id` (TEXT referencing app_user.id) to `report_templates`
-- 2. Add tenant-scoped case-insensitive partial unique index on `(tenant_id, LOWER(template_name)) WHERE is_active = true`
-- 3. Create dedicated `report_favorites` table for per-user favorites
--
-- Architectural & Design Notes:
-- - Legacy Favorites: Pre-existing favorite values stored inside `layout_config -> 'metadata' -> 'is_favorite'`
--   are tenant-wide and un-attributed to specific users. They are intentionally NOT migrated to `report_favorites`.
--   The per-user `report_favorites` table starts fresh for each authenticated user.
-- - Legacy Custom Reports: Existing custom report rows have `created_by_id = NULL`.
--   Sharing is intentionally disabled for them until they are claimed/re-saved or duplicated by their author.
-- - Deletions: Hard deletes (DELETE FROM report_templates) cascade via FKs. The unique index is declared
--   with `WHERE is_active = true` so that if soft-deletes or deactivations are ever employed, inactive template
--   names are immediately freed up for reuse without causing 409 conflicts.

-- 1. Extend report_templates
ALTER TABLE report_templates 
    ADD COLUMN IF NOT EXISTS is_personal BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS created_by_id TEXT REFERENCES app_user(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_report_templates_personal ON report_templates(tenant_id, is_personal);
CREATE INDEX IF NOT EXISTS idx_report_templates_creator ON report_templates(created_by_id);

-- 2. Tenant-scoped case-insensitive partial unique index on active report name
CREATE UNIQUE INDEX IF NOT EXISTS uq_report_templates_tenant_lower_name 
    ON report_templates(tenant_id, LOWER(template_name))
    WHERE is_active = true;

-- 3. Dedicated per-user favorites table
CREATE TABLE IF NOT EXISTS report_favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id TEXT NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES report_templates(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_report_favorites_user_template UNIQUE (tenant_id, user_id, template_id)
);

CREATE INDEX IF NOT EXISTS idx_report_favorites_user ON report_favorites(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_report_favorites_template ON report_favorites(template_id);
