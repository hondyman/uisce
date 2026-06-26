-- Revert uisce tenant gold copy status
UPDATE tenants 
SET gold_copy = false 
WHERE code = 'uisce';
