-- What a pipeline run produced beyond rows: e.g. the mastering runs its master steps started
-- ({"mastering": [{node_id, entity, load_run_id, run_id, status, ...}]}). Additive.
ALTER TABLE public.data_pipeline_runs ADD COLUMN IF NOT EXISTS outputs jsonb NOT NULL DEFAULT '{}'::jsonb;
