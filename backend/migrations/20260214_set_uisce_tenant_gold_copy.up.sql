-- Set uisce tenant as gold copy to enable BO fallback
UPDATE tenants 
SET gold_copy = true 
WHERE LOWER(name) LIKE '%uisce%' OR LOWER(display_name) LIKE '%uisce%';


