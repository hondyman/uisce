-- 20260727000025_add_tenant_id_to_datasource.sql
-- Adds tenant_id column to tenant_product_datasource and backfills from join chain.
-- This is a prerequisite for RLS policies that reference tenant_id on this table.

-- Step 1: Add tenant_id column (idempotent)
ALTER TABLE tenant_product_datasource ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Step 2: Backfill tenant_id from tenant_product → tenant_instance → tenants
-- Uses batched updates to avoid lock contention on large tables
DO $$
DECLARE
    batch_size INT := 1000;
    updated INT := batch_size;
BEGIN
    WHILE updated = batch_size LOOP
        UPDATE tenant_product_datasource tpd
        SET tenant_id = ti.tenant_id
        FROM tenant_product tp
        JOIN tenant_instance ti ON ti.id = tp.datasource_id
        WHERE tpd.tenant_product_id = tp.id
          AND tpd.tenant_id IS NULL
          AND tp.id IN (
              SELECT id FROM tenant_product LIMIT batch_size
          );
        GET DIAGNOSTICS updated = ROW_COUNT;
    END LOOP;
END $$;

-- Step 3: Add FK constraint (skip if already present or if column is NULL)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'tenant_product_datasource_tenant_id_fkey'
    ) THEN
        ALTER TABLE tenant_product_datasource
        ADD CONSTRAINT tenant_product_datasource_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

-- Step 4: Add index for RLS performance
CREATE INDEX IF NOT EXISTS idx_tpd_tenant ON tenant_product_datasource(tenant_id);

-- +goose Down
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'tenant_product_datasource_tenant_id_fkey') THEN
        ALTER TABLE tenant_product_datasource DROP CONSTRAINT tenant_product_datasource_tenant_id_fkey;
    END IF;
END $$;
ALTER TABLE tenant_product_datasource DROP COLUMN IF EXISTS tenant_id;
