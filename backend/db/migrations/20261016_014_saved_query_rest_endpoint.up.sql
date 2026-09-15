-- Extends data_explorer.saved_query (added by 20260826_001, previously wired
-- to a completely no-op backend - handlers.SavedQueryHandler's methods were
-- literal empty function bodies) with the columns the real implementation
-- needs, and disables its row-level security policy.
--
-- Why disable RLS: the policy added by 20260826_001 gates every row on
-- current_setting('app.current_tenant_id', true) - a Postgres session GUC
-- that is supposed to be set per-request. The only code that ever sets it
-- (backend/internal/middleware/session_auth.go) calls
-- `SET LOCAL app.current_tenant_id = $1` as a single ExecContext call with
-- no surrounding transaction; SET LOCAL's scope is "until end of the
-- current transaction", and with no explicit BEGIN, Go's sql package runs
-- that Exec as its own auto-committing statement - so the setting is
-- already gone by the time the connection (routed through a pool that may
-- hand out a different connection per query) runs the next query anyway.
-- In practice this GUC is never actually in effect, so as written the
-- policy silently returns zero rows for every SELECT and rejects every
-- INSERT/UPDATE, regardless of tenant. Real tenant isolation in this
-- codebase is enforced by explicit `WHERE tenant_id = $1` in application
-- code (see boresolver's InjectTenantScopingToGraph, and every other
-- tenant-scoped handler) - this table's handler (querybuilder.SavedQueryHandler)
-- follows that same, actually-working pattern instead.
ALTER TABLE data_explorer.saved_query DISABLE ROW LEVEL SECURITY;

ALTER TABLE data_explorer.saved_query
    ADD COLUMN IF NOT EXISTS chart_type TEXT NOT NULL DEFAULT 'bar';
