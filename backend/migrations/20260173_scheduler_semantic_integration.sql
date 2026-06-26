-- Migration to add semantic bindings to scheduler objects (tolerant to table name differences)
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='scheduler_jobs') THEN
    ALTER TABLE scheduler_jobs ADD COLUMN IF NOT EXISTS semantic_bindings JSONB NOT NULL DEFAULT '{}'::jsonb;
    CREATE INDEX IF NOT EXISTS idx_jobs_semantic_gin ON scheduler_jobs USING GIN (semantic_bindings);
  ELSIF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='scheduled_jobs') THEN
    ALTER TABLE scheduled_jobs ADD COLUMN IF NOT EXISTS semantic_bindings JSONB NOT NULL DEFAULT '{}'::jsonb;
    CREATE INDEX IF NOT EXISTS idx_jobs_semantic_gin ON scheduled_jobs USING GIN (semantic_bindings);
  ELSE
    RAISE NOTICE 'No jobs table found (scheduler_jobs/scheduled_jobs); skipping semantic_bindings for jobs';
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='scheduler_dags') THEN
    ALTER TABLE scheduler_dags ADD COLUMN IF NOT EXISTS semantic_bindings JSONB NOT NULL DEFAULT '{}'::jsonb;
    CREATE INDEX IF NOT EXISTS idx_dags_semantic_gin ON scheduler_dags USING GIN (semantic_bindings);
  ELSIF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='scheduled_dags') THEN
    ALTER TABLE scheduled_dags ADD COLUMN IF NOT EXISTS semantic_bindings JSONB NOT NULL DEFAULT '{}'::jsonb;
    CREATE INDEX IF NOT EXISTS idx_dags_semantic_gin ON scheduled_dags USING GIN (semantic_bindings);
  ELSE
    RAISE NOTICE 'No dags table found (scheduler_dags/scheduled_dags); skipping semantic_bindings for dags';
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='scheduler_job_runs') THEN
    ALTER TABLE scheduler_job_runs ADD COLUMN IF NOT EXISTS semantic_bindings JSONB NOT NULL DEFAULT '{}'::jsonb;
    CREATE INDEX IF NOT EXISTS idx_runs_semantic_gin ON scheduler_job_runs USING GIN (semantic_bindings);
  ELSIF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='job_runs') THEN
    ALTER TABLE job_runs ADD COLUMN IF NOT EXISTS semantic_bindings JSONB NOT NULL DEFAULT '{}'::jsonb;
    CREATE INDEX IF NOT EXISTS idx_runs_semantic_gin ON job_runs USING GIN (semantic_bindings);
  ELSE
    RAISE NOTICE 'No job_runs table found (scheduler_job_runs/job_runs); skipping semantic_bindings for runs';
  END IF;
END
$do$;
