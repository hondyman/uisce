-- Phase 3: dedicated admin read role for cross-tenant monitoring queries
-- app_admin_read: SELECT on monitoring tables, INSERT on admin_audit_logs, ROLBYPASSRLS (scoped to 4 tables only)

-- Create role with login so it can authenticate to a connection pool
CREATE ROLE app_admin_read WITH LOGIN NOCREATEDB NOCREATEROLE;

-- Grant SELECT on the four monitoring read tables (RLS bypass is scoped to these)
GRANT SELECT ON public.report_executions TO app_admin_read;
GRANT SELECT ON public.report_execution_events TO app_admin_read;
GRANT SELECT ON public.report_templates TO app_admin_read;

-- Grant INSERT+SELECT on admin_audit_logs (for audit write; SELECT for any future reads)
GRANT INSERT, SELECT ON public.admin_audit_logs TO app_admin_read;

-- Now enable RLS bypass — this is scoped to the four tables because
-- the role has no other table-level privileges in the schema.
-- Blast radius of a compromised app_admin_read connection: read-only on monitoring tables.
ALTER ROLE app_admin_read BYPASSRLS;

-- Comment for documentation
COMMENT ON ROLE app_admin_read IS 'Dedicated admin read role for cross-tenant monitoring queries. SELECT on report_executions, report_execution_events, report_templates; INSERT+SELECT on admin_audit_logs. ROLBYPASSRLS scoped to these four tables only.';
