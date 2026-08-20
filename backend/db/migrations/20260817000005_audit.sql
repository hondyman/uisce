-- Migration: Create api_dispatch_audit_log
-- Purpose: Persist a row for every API dispatcher call so operators can
--          answer "who called what, when, with what result" without grepping
--          application logs. The dispatcher writes via a fire-and-forget
--          goroutine so the user-facing HTTP response is never blocked on
--          this table.
-- Date: 2026-08-17
--
-- Column design:
--   * tenant_id              - tenant scoping + RLS key
--   * user_id                - nullable: may be an automated dispatch
--   * api_datasource_id      - the catalog node id of the API service
--   * api_endpoint_id        - the catalog node id of the specific endpoint
--   * method                 - HTTP verb actually used
--   * path                   - final URL path that was called
--   * status_code            - upstream's HTTP response status (or 0 on network error)
--   * duration_ms            - elapsed time in milliseconds
--   * success                - true on 2xx responses, false otherwise
--   * record_count           - number of parsed JSON records in the response
--   * error                  - text of any error (HTTP failure, parse failure,
--                              OAuth refresh failure, etc.); "" on success
--   * request_params         - JSON snapshot of path_params + query_params + headers
--   * created_at             - indexed for descending "recent calls" queries

BEGIN;

CREATE TABLE IF NOT EXISTS public.api_dispatch_audit_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id             UUID,
    api_datasource_id   UUID NOT NULL,
    api_endpoint_id     UUID NOT NULL,
    method              TEXT NOT NULL,
    path                TEXT NOT NULL,
    status_code         INT NOT NULL DEFAULT 0,
    duration_ms         BIGINT NOT NULL DEFAULT 0,
    success             BOOLEAN NOT NULL DEFAULT false,
    record_count        INT NOT NULL DEFAULT 0,
    error               TEXT NOT NULL DEFAULT '',
    request_params      JSONB DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_endpoint_recent
    ON public.api_dispatch_audit_log(tenant_id, api_endpoint_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_recent
    ON public.api_dispatch_audit_log(tenant_id, created_at DESC);

ALTER TABLE public.api_dispatch_audit_log ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_audit_select
    ON public.api_dispatch_audit_log
    FOR SELECT
    USING (tenant_id::text = current_setting('app.current_tenant', true));

CREATE POLICY tenant_isolation_audit_insert
    ON public.api_dispatch_audit_log
    FOR INSERT
    WITH CHECK (tenant_id::text = current_setting('app.current_tenant', true));

COMMIT;