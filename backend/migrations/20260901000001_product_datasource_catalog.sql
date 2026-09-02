-- 20260901000001_product_datasource_catalog.sql
-- Phase 1 of the connections/datasource redesign: formalize the gold-copy
-- tenant's product -> datasource-type allow-list as a real catalog table,
-- instead of it only existing as a convention in the gold-copy tenant's own
-- tenant_product_datasource rows. Purely additive: no FK enforcement yet.

-- Step 0: alpha_datasource.id has no PK/unique constraint today (nothing
-- currently FKs to it — joins only), which blocks referencing it below.
-- id values are already unique/non-null in practice; add the constraint.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'alpha_datasource' AND constraint_type = 'PRIMARY KEY'
    ) THEN
        ALTER TABLE alpha_datasource ADD PRIMARY KEY (id);
    END IF;
END $$;

-- Step 1: catalog table. Each row means "alpha_product_id may use
-- alpha_datasource_id" — the gold-copy tenant is the only one allowed to
-- add/remove rows here (enforced at the service layer in a later phase).
CREATE TABLE IF NOT EXISTS product_datasource_catalog (
    alpha_product_id    UUID NOT NULL REFERENCES alpha_product(id) ON DELETE CASCADE,
    alpha_datasource_id UUID NOT NULL REFERENCES alpha_datasource(id) ON DELETE CASCADE,
    is_required          BOOLEAN NOT NULL DEFAULT false,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (alpha_product_id, alpha_datasource_id)
);

-- Step 2: backfill the catalog from the gold-copy tenant's existing
-- tenant_product_datasource rows (its own configured datasources per
-- product are, by convention, what every tenant is licensed to use).
INSERT INTO product_datasource_catalog (alpha_product_id, alpha_datasource_id)
SELECT DISTINCT tp.alpha_product_id, tpd.alpha_datasource_id
FROM tenant_product_datasource tpd
JOIN tenant_product tp ON tp.id = tpd.tenant_product_id
JOIN tenants t ON t.id = tpd.tenant_id
WHERE t.gold_copy = true
  AND tpd.alpha_datasource_id IS NOT NULL
ON CONFLICT (alpha_product_id, alpha_datasource_id) DO NOTHING;

-- Step 3: denormalize alpha_product_id onto tenant_product_datasource so a
-- later phase can add a composite FK to the catalog without a join. Nullable
-- for now — no enforcement yet, this migration is report-only.
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS alpha_product_id UUID;

UPDATE tenant_product_datasource tpd
SET alpha_product_id = tp.alpha_product_id
FROM tenant_product tp
WHERE tpd.tenant_product_id = tp.id
  AND tpd.alpha_product_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_tpd_alpha_product ON tenant_product_datasource(alpha_product_id);

-- +goose Down
DROP INDEX IF EXISTS idx_tpd_alpha_product;
ALTER TABLE tenant_product_datasource DROP COLUMN IF EXISTS alpha_product_id;
DROP TABLE IF EXISTS product_datasource_catalog;
