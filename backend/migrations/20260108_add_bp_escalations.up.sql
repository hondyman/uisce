ALTER TABLE IF EXISTS business_process_step
ADD COLUMN IF NOT EXISTS escalations JSONB;
