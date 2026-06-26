-- Migration: add ip whitelist entries with label + assignment table

CREATE TABLE IF NOT EXISTS tenant_ip_whitelist_entries (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    ip_address varchar(45) NOT NULL,
    label varchar(255) NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL
);

-- Ensure ip_address is unique so we can upsert by ip_address
CREATE UNIQUE INDEX IF NOT EXISTS idx_tiw_entries_ip_address ON tenant_ip_whitelist_entries (ip_address);

CREATE TABLE IF NOT EXISTS tenant_ip_whitelist_assignments (
    whitelist_id uuid NOT NULL REFERENCES tenant_ip_whitelist_entries(id) ON DELETE CASCADE,
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT tenant_ip_whitelist_assignments_pkey PRIMARY KEY (whitelist_id, tenant_id)
);

-- Backfill: if legacy table exists, migrate its rows into new schema
DO $$
BEGIN
    IF to_regclass('public.tenant_ip_whitelist') IS NOT NULL THEN
        -- Insert distinct ip addresses
        INSERT INTO tenant_ip_whitelist_entries (ip_address, created_at, updated_at)
        SELECT DISTINCT ip_address, now(), now() FROM tenant_ip_whitelist
        ON CONFLICT (ip_address) DO NOTHING;

        -- Link assignments
    INSERT INTO tenant_ip_whitelist_assignments (whitelist_id, tenant_id, created_at)
    SELECT e.id, tw.tenant_id::uuid, now()
    FROM tenant_ip_whitelist tw
    JOIN tenant_ip_whitelist_entries e ON e.ip_address = tw.ip_address
    ON CONFLICT DO NOTHING;

        -- Keep legacy table for now (for rollback) - not dropping here
    END IF;
END$$;
