-- Set all tenants to us-west region
UPDATE tenants 
SET region = 'us-west'
WHERE region IS NULL;
