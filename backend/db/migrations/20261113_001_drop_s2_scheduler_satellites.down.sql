-- Recreate empty shells only; row data is not recovered.

CREATE TABLE IF NOT EXISTS public.scheduled_reports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    created_at timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.scheduled_report_executions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_report_id uuid NOT NULL REFERENCES public.scheduled_reports(id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.scheduler_ai_suggestions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    suggestion_type varchar(50) NOT NULL,
    title varchar(255) NOT NULL,
    proposed_changes jsonb NOT NULL DEFAULT '{}'::jsonb,
    status varchar(20) DEFAULT 'pending',
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.scheduler_changesets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid,
    title text,
    created_at timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.scheduler_changeset_approvals (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    changeset_id uuid NOT NULL REFERENCES public.scheduler_changesets(id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT now()
);
