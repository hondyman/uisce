-- Migration: Extend user_personalization_profiles with pinned items and UI preferences
-- Part of Phase 2: PersonalizationService

ALTER TABLE user_personalization_profiles
    ADD COLUMN IF NOT EXISTS pinned_bo_keys TEXT[] DEFAULT ARRAY[]::TEXT[],
    ADD COLUMN IF NOT EXISTS quick_launch_filters JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS default_page_layout VARCHAR(100) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS default_dashboard_domain VARCHAR(50) DEFAULT 'PORTFOLIO';

COMMENT ON COLUMN user_personalization_profiles.pinned_bo_keys IS
    'User-pinned business object keys shown in the omnibox quick-launch';
COMMENT ON COLUMN user_personalization_profiles.quick_launch_filters IS
    'Saved filter presets keyed by BO key for quick re-application';
COMMENT ON COLUMN user_personalization_profiles.default_page_layout IS
    'Preferred page layout template (e.g. grid, canvas, dashboard)';
COMMENT ON COLUMN user_personalization_profiles.default_dashboard_domain IS
    'Preferred dashboard domain (PORTFOLIO, COMPLIANCE, RISK, ENGINEERING)';

CREATE INDEX IF NOT EXISTS idx_profile_pinned_bo
    ON user_personalization_profiles USING GIN (pinned_bo_keys)
    WHERE array_length(pinned_bo_keys, 1) IS NOT NULL;
