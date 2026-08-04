-- 20260805000001_marketplace_unified_schema.up.sql
-- Unifies the three fragmented marketplace schemas into one RBAC-ecosystem-grade store.
-- Consolidates: RBAC artifacts (marketplace_listings/artifacts/installs from scripts/setup_marketplace_schema.sql),
--               rule/calculation packs (marketplace_items from migrations/004_marketplace_tables.sql),
--               and integrations (marketplace_integrations) under a single kind enum.
--
-- Security properties added:
--   - status CHECK constraint so listings can be suspended/retired (not just soft-deleted)
--   - idempotency table to make publish + install safe for retries and HTTP-replay
--   - installed_marketplace_artifacts replaces the unsafe ON CONFLICT(role_key) DO UPDATE install path

BEGIN;

-- ── 1. marketplace_listings ────────────────────────────────────────────────────
ALTER TABLE marketplace_listings
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'rbac'
        CHECK (kind IN ('rbac','rules_calculations','integration','bundle')),
    ADD COLUMN IF NOT EXISTS billing_period TEXT NOT NULL DEFAULT 'one_time'
        CHECK (billing_period IN ('one_time','monthly','annual')),
    ADD COLUMN IF NOT EXISTS publisher_payout_account TEXT,
    ADD COLUMN IF NOT EXISTS publisher_actor_id UUID,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published'
        CHECK (status IN ('draft','published','suspended','retired')),
    ADD COLUMN IF NOT EXISTS version_count INT NOT NULL DEFAULT 1;

-- Defensively promote existing rows to the default kind so the CHECK passes.
UPDATE marketplace_listings SET kind = 'rbac' WHERE kind IS NULL;
UPDATE marketplace_listings SET status = 'published' WHERE status IS NULL;
UPDATE marketplace_listings SET billing_period = 'one_time' WHERE billing_period IS NULL;

ALTER TABLE marketplace_listings
    ALTER COLUMN kind SET NOT NULL,
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN billing_period SET NOT NULL,
    ALTER COLUMN version_count SET NOT NULL DEFAULT 1;

-- ── 2. marketplace_artifacts ────────────────────────────────────────────────────
-- Ensure payload is JSONB and add versioning columns.
ALTER TABLE marketplace_artifacts
    ALTER COLUMN payload TYPE JSONB USING payload::JSONB,
    ADD COLUMN IF NOT EXISTS artifact_version TEXT NOT NULL DEFAULT '1.0.0',
    ADD COLUMN IF NOT EXISTS min_platform_version TEXT,
    ADD COLUMN IF NOT EXISTS canonical_sha256 TEXT,
    ADD COLUMN IF NOT EXISTS is_latest BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE marketplace_artifacts SET artifact_version = '1.0.0' WHERE artifact_version IS NULL;
UPDATE marketplace_artifacts SET is_latest = TRUE WHERE is_latest IS NULL;

ALTER TABLE marketplace_artifacts
    ALTER COLUMN artifact_version SET NOT NULL,
    ALTER COLUMN is_latest SET NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_artifacts_listing_version
    ON marketplace_artifacts(listing_id, artifact_version);
CREATE INDEX IF NOT EXISTS idx_artifacts_latest
    ON marketplace_artifacts(listing_id) WHERE is_latest = TRUE;

-- ── 3. installed_marketplace_artifacts ────────────────────────────────────────
-- Template-subscription model: installing a listing creates a subscription record,
-- NOT a cloned row in bp_roles.  The RBAC engine already falls back to Gold Copy
-- when a role_key is not found in the tenant's local bp_roles table, so the
-- subscription activates automatically without data duplication.
CREATE TABLE IF NOT EXISTS installed_marketplace_artifacts (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                    UUID NOT NULL,
    listing_id                   UUID NOT NULL REFERENCES marketplace_listings(id) ON DELETE CASCADE,
    artifact_type                TEXT NOT NULL
        CHECK (artifact_type IN ('rbac_role','rbac_role_pack','abac_policy',
                                 'rules_calculation','integration','bundle')),
    artifact_key                 TEXT NOT NULL,   -- the role_key / policy_key this tenant subscribes to
    artifact_version             TEXT NOT NULL,
    installed_by_actor_id        UUID NOT NULL,
    installed_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One active subscription per (tenant, listing).  Re-installing re-activates if retired.
    UNIQUE (tenant_id, listing_id)
);

CREATE INDEX IF NOT EXISTS idx_installed_tenant    ON installed_marketplace_artifacts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_installed_listing    ON installed_marketplace_artifacts(listing_id);
CREATE INDEX IF NOT EXISTS idx_installed_artifact_key ON installed_marketplace_artifacts(artifact_key);

-- ── 4. marketplace_idempotency ─────────────────────────────────────────────────
-- Guarantees exactly-once publish + install semantics for HTTP retries.
CREATE TABLE IF NOT EXISTS marketplace_idempotency (
    key             TEXT PRIMARY KEY,       -- client-supplied Idempotency-Key
    tenant_id       UUID NOT NULL,
    operation       TEXT NOT NULL CHECK (operation IN ('publish','install')),
    request_hash    BYTEA NOT NULL,         -- SHA-256 of canonicalized request body
    response_status INT NOT NULL,
    response_body   JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours'
);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON marketplace_idempotency(expires_at);

-- ── 5. Seed Gold Copy marketplace roles ───────────────────────────────────────
-- These are Gold Copy template roles (is_template = true, gold_copy tenant).
-- They define the three marketplace permission levels.
DO $$
DECLARE
    gold_copy_tid UUID;
BEGIN
    SELECT id INTO gold_copy_tid FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF gold_copy_tid IS NULL THEN
        RAISE NOTICE 'Gold Copy tenant not found – skipping marketplace role seed. Run tenant seed first.';
        RETURN;
    END IF;

    INSERT INTO bp_roles (tenant_id, role_key, role_name, role_type, role_level, is_active, is_template, description)
    VALUES
        (gold_copy_tid, 'marketplace_publisher', 'Marketplace Publisher',
         'SYSTEM', 'GOLD_COPY', true, true,
         'Can publish custom roles and policies to the Uisce Marketplace.'),

        (gold_copy_tid, 'marketplace_admin', 'Marketplace Administrator',
         'SYSTEM', 'GOLD_COPY', true, true,
         'Can review, approve, suspend, and retire marketplace listings. Required for product-evolution.'),

        (gold_copy_tid, 'marketplace_auditor', 'Marketplace Auditor',
         'SYSTEM', 'GOLD_COPY', true, true,
         'Read-only access to marketplace listings, installs, and product-evolution telemetry.')
    ON CONFLICT (role_key) DO NOTHING;
END $$;

-- ── 6. marketplace_listings indexes & audit helper ────────────────────────────
CREATE INDEX IF NOT EXISTS idx_listings_kind        ON marketplace_listings(kind);
CREATE INDEX IF NOT EXISTS idx_listings_status      ON marketplace_listings(status);
CREATE INDEX IF NOT EXISTS idx_listings_publisher   ON marketplace_listings(publisher_tenant_id);
CREATE INDEX IF NOT EXISTS idx_listings_installs    ON marketplace_listings(installs_count DESC);

-- Retired listings are hidden from browse by default; make the predicate selective.
CREATE INDEX IF NOT EXISTS idx_listings_browse
    ON marketplace_listings(kind, status, installs_count DESC)
    WHERE status = 'published';

COMMIT;
