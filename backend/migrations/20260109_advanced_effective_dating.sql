-- Add history_mode to business_objects
ALTER TABLE business_objects ADD COLUMN history_mode VARCHAR(50) DEFAULT 'EXPLICIT_RANGE';

-- Add comments for documentation
COMMENT ON COLUMN business_objects.history_mode IS 'History tracking mode: EXPLICIT_RANGE or EVENT_LOG';
