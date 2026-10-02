-- Slice 5 follow-up: drop retired Scheduler Intelligence satellite tables.
--
-- KEPT as the job_dag / scheduled-job backing store for /api/schedules:
--   scheduled_jobs, scheduled_dags, job_runs, dag_runs, job_dependencies
--
-- DROPPED (HTTP already 410; zero rows on alpha):
--   scheduler_ai_suggestions, scheduler_changeset_*, scheduled_reports*

DROP TABLE IF EXISTS public.scheduler_changeset_approvals;
DROP TABLE IF EXISTS public.scheduler_changesets;
DROP TABLE IF EXISTS public.scheduler_ai_suggestions;
DROP TABLE IF EXISTS public.scheduled_report_executions;
DROP TABLE IF EXISTS public.scheduled_reports;

COMMENT ON TABLE public.scheduled_jobs IS
  'Job definitions for schedule target kind job_dag (and future job). Owned by the one scheduler (/api/schedules); S2 /scheduler HTTP is retired.';
COMMENT ON TABLE public.scheduled_dags IS
  'DAG definitions for schedule target kind job_dag. Owned by the one scheduler (/api/schedules); S2 /scheduler HTTP is retired.';
COMMENT ON TABLE public.job_runs IS
  'Run history for scheduled_jobs (job_dag node executions).';
COMMENT ON TABLE public.dag_runs IS
  'Run history for scheduled_dags (job_dag schedule firings).';
