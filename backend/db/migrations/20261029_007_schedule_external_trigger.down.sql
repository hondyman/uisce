DELETE FROM public.message_catalog WHERE set_nbr = 9200 AND message_nbr BETWEEN 18 AND 23;
DROP INDEX IF EXISTS public.uq_schedule_runs_idempotency;
ALTER TABLE public.schedule_runs DROP CONSTRAINT IF EXISTS schedule_runs_trigger_check;
ALTER TABLE public.schedule_runs ADD CONSTRAINT schedule_runs_trigger_check CHECK (trigger IN ('schedule', 'manual'));
ALTER TABLE public.schedule_runs DROP COLUMN IF EXISTS idempotency_key, DROP COLUMN IF EXISTS external_ref, DROP COLUMN IF EXISTS external_system;
ALTER TABLE public.schedules DROP CONSTRAINT IF EXISTS schedules_external_shape_check,
    DROP CONSTRAINT IF EXISTS schedules_trigger_mode_check, DROP COLUMN IF EXISTS trigger_mode;
