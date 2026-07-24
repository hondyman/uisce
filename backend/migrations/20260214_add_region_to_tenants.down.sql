-- Remove region column from tenants table
ALTER TABLE tenants 
DROP COLUMN IF EXISTS region;
