-- Migration: Product-Tenant Relationship Refactor
-- Changes tenant_product to link to tenant instead of tenant_instance
-- Adds datasource_id to tenant_product_datasource for instance-specific datasources



-- ============================================
-- STEP 1: Add tenant_id to tenant_product
-- ============================================
ALTER TABLE tenant_product 
    ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- ============================================
-- STEP 2: Add product_id column (references products table)
-- ============================================
ALTER TABLE tenant_product
    ADD COLUMN IF NOT EXISTS product_id UUID;

-- ============================================
-- STEP 3: Backfill tenant_id from tenant_instance
-- ============================================
UPDATE tenant_product tp
SET tenant_id = ti.tenant_id
FROM tenant_instance ti
WHERE tp.datasource_id = ti.id
AND tp.tenant_id IS NULL;

-- ============================================
-- STEP 4: Add NOT NULL constraint after backfill (only if column has data)
-- ============================================
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM tenant_product WHERE tenant_id IS NOT NULL LIMIT 1) THEN
        ALTER TABLE tenant_product ALTER COLUMN tenant_id SET NOT NULL;
    ELSIF NOT EXISTS (SELECT 1 FROM tenant_product LIMIT 1) THEN
        -- Table is empty, safe to set NOT NULL
        ALTER TABLE tenant_product ALTER COLUMN tenant_id SET NOT NULL;
    END IF;
END $$;

-- ============================================
-- STEP 5: Add datasource_id to tenant_product_datasource
-- ============================================
ALTER TABLE tenant_product_datasource
    ADD COLUMN IF NOT EXISTS datasource_id UUID;

-- Add foreign key constraint
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'tenant_product_datasource_instance_fk'
    ) THEN
        ALTER TABLE tenant_product_datasource
            ADD CONSTRAINT tenant_product_datasource_instance_fk 
            FOREIGN KEY (datasource_id) REFERENCES tenant_instance(id) 
            ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED;
    END IF;
END $$;

-- ============================================
-- STEP 6: Backfill datasource_id in datasources from tenant_product
-- ============================================
UPDATE tenant_product_datasource tpd
SET datasource_id = tp.datasource_id
FROM tenant_product tp
WHERE tpd.tenant_product_id = tp.id
AND tpd.datasource_id IS NULL;

-- ============================================
-- STEP 7: Add foreign key from tenant_product to tenants
-- ============================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'tenant_product_tenant_id_fk'
    ) THEN
        ALTER TABLE tenant_product
            ADD CONSTRAINT tenant_product_tenant_id_fk 
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) 
            ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED;
    END IF;
END $$;

-- ============================================
-- STEP 8: Add foreign key from tenant_product to products table
-- Note: Only add if products table exists with the expected structure
-- ============================================
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_schema = 'public' AND table_name = 'products'
    ) THEN
        -- Check if constraint already exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name = 'tenant_product_products_fk'
        ) THEN
            -- The products table has composite PK (tenant_id, product_id), so we reference just product_id
            -- Assuming gold copy products have their own tenant_id
            ALTER TABLE tenant_product
                ADD CONSTRAINT tenant_product_products_fk 
                FOREIGN KEY (product_id) REFERENCES products(product_id) 
                ON DELETE SET NULL ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    ELSE
        RAISE NOTICE 'Products table not found - skipping foreign key constraint';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not create products FK: % - this is optional', SQLERRM;
END $$;

-- ============================================
-- STEP 9: Create new unique constraint (tenant_id, product_id)
-- ============================================
DO $$
BEGIN
    -- First, try to drop old constraint if it exists
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'tenant_product_uniq'
    ) THEN
        ALTER TABLE tenant_product DROP CONSTRAINT tenant_product_uniq;
    END IF;
    
    -- Add new unique constraint only if product_id column is populated
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'tenant_product_tenant_product_uniq'
    ) THEN
        -- Only create if we have both columns with data
        IF EXISTS (SELECT 1 FROM tenant_product WHERE tenant_id IS NOT NULL AND product_id IS NOT NULL LIMIT 1) 
           OR NOT EXISTS (SELECT 1 FROM tenant_product LIMIT 1) THEN
            ALTER TABLE tenant_product
                ADD CONSTRAINT tenant_product_tenant_product_uniq UNIQUE (tenant_id, product_id);
        ELSE
            RAISE NOTICE 'Skipping unique constraint - product_id not yet populated';
        END IF;
    END IF;
END $$;

-- ============================================
-- STEP 10: Update unique constraint on datasources to include datasource_id
-- ============================================
DO $$
BEGIN
    -- Check if old constraint exists
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'tenant_product_datasource_source_uniq'
    ) THEN
        ALTER TABLE tenant_product_datasource DROP CONSTRAINT tenant_product_datasource_source_uniq;
    END IF;
    
    -- Add new constraint including datasource_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'tenant_product_datasource_instance_source_uniq'
    ) THEN
        ALTER TABLE tenant_product_datasource
            ADD CONSTRAINT tenant_product_datasource_instance_source_uniq 
            UNIQUE (tenant_product_id, datasource_id, source_name);
    END IF;
END $$;

-- ============================================
-- STEP 11: Create index for common queries
-- ============================================
CREATE INDEX IF NOT EXISTS idx_tenant_product_tenant_id ON tenant_product(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_product_datasource_instance_id ON tenant_product_datasource(datasource_id);


