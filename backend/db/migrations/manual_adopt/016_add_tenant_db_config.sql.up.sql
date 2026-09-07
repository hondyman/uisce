-- Migration 016: Add Tenant DB Config
-- Adds db_connection_string to platform.tenants to support Database-per-Tenant architecture.

ALTER TABLE platform.tenants 
ADD COLUMN IF NOT EXISTS db_connection_string TEXT;

-- Optional: Add a flag to indicate if the tenant has a dedicated DB
ALTER TABLE platform.tenants 
ADD COLUMN IF NOT EXISTS has_dedicated_db BOOLEAN DEFAULT FALSE;
