-- Phase 1: tenant isolation + integrity on the live Order chain.
-- Additive only. Does not DROP SCHEMA orm. Does not create security/FIX/SWIFT.
-- BO CRUD already injects tenant_id when the column exists (bo_crud_handler.go).

DO $$
DECLARE
  gold uuid;
BEGIN
  SELECT id INTO gold FROM public.tenants WHERE gold_copy = true LIMIT 1;
  IF gold IS NULL THEN
    gold := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
  END IF;

  -- tenant_id on every live orm table
  ALTER TABLE orm.account ADD COLUMN IF NOT EXISTS tenant_id uuid;
  ALTER TABLE orm.broker ADD COLUMN IF NOT EXISTS tenant_id uuid;
  ALTER TABLE orm."order" ADD COLUMN IF NOT EXISTS tenant_id uuid;
  ALTER TABLE orm.order_allocation ADD COLUMN IF NOT EXISTS tenant_id uuid;
  ALTER TABLE orm.placement ADD COLUMN IF NOT EXISTS tenant_id uuid;
  ALTER TABLE orm.execution ADD COLUMN IF NOT EXISTS tenant_id uuid;
  ALTER TABLE orm.execution_allocation ADD COLUMN IF NOT EXISTS tenant_id uuid;

  UPDATE orm.account SET tenant_id = gold WHERE tenant_id IS NULL;
  UPDATE orm.broker SET tenant_id = gold WHERE tenant_id IS NULL;
  UPDATE orm."order" SET tenant_id = gold WHERE tenant_id IS NULL;
  UPDATE orm.order_allocation SET tenant_id = gold WHERE tenant_id IS NULL;
  UPDATE orm.placement SET tenant_id = gold WHERE tenant_id IS NULL;
  UPDATE orm.execution SET tenant_id = gold WHERE tenant_id IS NULL;
  UPDATE orm.execution_allocation SET tenant_id = gold WHERE tenant_id IS NULL;

  ALTER TABLE orm.account ALTER COLUMN tenant_id SET NOT NULL;
  ALTER TABLE orm.broker ALTER COLUMN tenant_id SET NOT NULL;
  ALTER TABLE orm."order" ALTER COLUMN tenant_id SET NOT NULL;
  ALTER TABLE orm.order_allocation ALTER COLUMN tenant_id SET NOT NULL;
  ALTER TABLE orm.placement ALTER COLUMN tenant_id SET NOT NULL;
  ALTER TABLE orm.execution ALTER COLUMN tenant_id SET NOT NULL;
  ALTER TABLE orm.execution_allocation ALTER COLUMN tenant_id SET NOT NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_orm_account_tenant ON orm.account (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_broker_tenant ON orm.broker (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_tenant ON orm."order" (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_allocation_tenant ON orm.order_allocation (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_placement_tenant ON orm.placement (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_execution_tenant ON orm.execution (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_execution_allocation_tenant ON orm.execution_allocation (tenant_id);

-- Empty ClOrdID is NULL so the unique index can ignore it.
UPDATE orm.placement SET fix_clordid = NULL WHERE fix_clordid IS NOT NULL AND btrim(fix_clordid) = '';

-- Seed data reused CLORD-1 on several placements; uniquify extras so FIX correlation can be unique per tenant.
UPDATE orm.placement p
SET fix_clordid = p.fix_clordid || '-' || substr(p.id::text, 1, 8)
WHERE p.id IN (
  SELECT id FROM (
    SELECT id,
           row_number() OVER (PARTITION BY fix_clordid ORDER BY created_at, id) AS rn
    FROM orm.placement
    WHERE fix_clordid IS NOT NULL
  ) d
  WHERE d.rn > 1
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_orm_placement_tenant_clordid
  ON orm.placement (tenant_id, fix_clordid)
  WHERE fix_clordid IS NOT NULL;

-- One live order had leaves_qty 42 after a full fill; align residual before the CHECK.
UPDATE orm."order"
SET leaves_qty = target_qty - executed_qty
WHERE leaves_qty IS DISTINCT FROM target_qty - executed_qty;

ALTER TABLE orm."order" DROP CONSTRAINT IF EXISTS chk_order_leaves;
ALTER TABLE orm."order" ADD CONSTRAINT chk_order_leaves
  CHECK (leaves_qty = target_qty - executed_qty AND leaves_qty >= 0 AND executed_qty >= 0);

ALTER TABLE orm.placement DROP CONSTRAINT IF EXISTS chk_placement_leaves;
ALTER TABLE orm.placement ADD CONSTRAINT chk_placement_leaves
  CHECK (leaves_qty = routed_qty - executed_qty AND leaves_qty >= 0 AND executed_qty >= 0);

-- Status CHECKs wait until Phase 5 (workflow owns the machine). Live values include
-- DRAFT, ALLOCATED, FULLY_ROUTED, PARTIALLY_ALLOCATED, ROUTED — do not reject them here.
