-- 20261206_002_tenant_lakehouse_audit.up.sql
--
-- Append-only, hash-chained audit trail for a tenant's lakehouse configuration
-- (retention, provisioning), written in the same transaction as the change.
--
-- Why a dedicated table rather than the audit outbox: the outbox's table
-- (public.catalog_outbox_events) is defined in a migration file without the
-- .up.sql suffix, so the runner never applies it, and TransactionalOutboxManager
-- has no production constructor. Routing this through it would silently audit
-- nothing (ADR evidence rule: a "wired" claim needs a production call site).
--
-- Each row carries prev_hash and hash. The trigger, not the client, computes both,
-- so a caller cannot forge or skip a link. This is tamper-EVIDENT (verify
-- recomputes the chain), not tamper-PROOF: the hash is unkeyed, and a role that can
-- rewrite the whole table could rewrite the chain. alpha is the system of record
-- and nothing is ever dropped from this table (ADR-029); the lake may additionally
-- hold an immutable copy under Object Lock (ADR-032).
--
-- Additive: new table and functions only.

CREATE SEQUENCE IF NOT EXISTS public.tenant_lakehouse_audit_id_seq;

CREATE TABLE IF NOT EXISTS public.tenant_lakehouse_audit (
    -- No default: the insert trigger assigns the id inside the per-tenant lock.
    id          BIGINT PRIMARY KEY,
    -- No foreign key: the audit record must outlive the tenant row.
    tenant_id   UUID        NOT NULL,
    at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id    TEXT        NOT NULL,
    actor_role  TEXT        NOT NULL DEFAULT '',
    action      TEXT        NOT NULL CHECK (action IN (
                    'configured', 'retention_extended', 'provision_requested',
                    'provisioned', 'provision_failed', 'retention_applied',
                    'retention_sync_failed', 'state_changed')),
    before      JSONB,
    after       JSONB,
    prev_hash   TEXT        NOT NULL,
    hash        TEXT        NOT NULL
);

ALTER SEQUENCE public.tenant_lakehouse_audit_id_seq OWNED BY public.tenant_lakehouse_audit.id;

CREATE INDEX IF NOT EXISTS idx_tenant_lakehouse_audit_tenant
    ON public.tenant_lakehouse_audit (tenant_id, id);

-- The one definition of the hash, shared by the insert trigger and verify. The
-- timestamp is rendered in UTC so the result does not depend on a session's
-- time zone.
CREATE OR REPLACE FUNCTION public.tenant_lakehouse_audit_hash(
    p_prev TEXT, p_tenant UUID, p_at TIMESTAMPTZ, p_actor TEXT, p_action TEXT,
    p_before JSONB, p_after JSONB
) RETURNS TEXT AS $$
    SELECT encode(sha256(convert_to(
        p_prev || '|' || p_tenant::text || '|' ||
        to_char(p_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') || '|' ||
        p_actor || '|' || p_action || '|' ||
        coalesce(p_before::text, '') || '|' || coalesce(p_after::text, ''),
        'UTF8')), 'hex')
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION public.tenant_lakehouse_audit_before_insert() RETURNS trigger AS $$
BEGIN
    -- Serialize writers per tenant, and take the id and timestamp INSIDE the lock,
    -- so that id order is chain order. A column default would assign the id before
    -- the lock and let two writers link out of order.
    PERFORM pg_advisory_xact_lock(hashtextextended('tenant_lakehouse_audit:' || NEW.tenant_id::text, 0));
    NEW.id := nextval('public.tenant_lakehouse_audit_id_seq');
    NEW.at := clock_timestamp();

    SELECT a.hash INTO NEW.prev_hash
      FROM public.tenant_lakehouse_audit a
     WHERE a.tenant_id = NEW.tenant_id
     ORDER BY a.id DESC LIMIT 1;
    IF NEW.prev_hash IS NULL THEN
        NEW.prev_hash := repeat('0', 64);
    END IF;

    NEW.hash := public.tenant_lakehouse_audit_hash(
        NEW.prev_hash, NEW.tenant_id, NEW.at, NEW.actor_id, NEW.action, NEW.before, NEW.after);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tenant_lakehouse_audit_insert ON public.tenant_lakehouse_audit;
CREATE TRIGGER trg_tenant_lakehouse_audit_insert
BEFORE INSERT ON public.tenant_lakehouse_audit
FOR EACH ROW EXECUTE FUNCTION public.tenant_lakehouse_audit_before_insert();

CREATE OR REPLACE FUNCTION public.tenant_lakehouse_audit_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'tenant_lakehouse_audit is append-only';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tenant_lakehouse_audit_immutable ON public.tenant_lakehouse_audit;
CREATE TRIGGER trg_tenant_lakehouse_audit_immutable
BEFORE UPDATE OR DELETE ON public.tenant_lakehouse_audit
FOR EACH ROW EXECUTE FUNCTION public.tenant_lakehouse_audit_append_only();

DROP TRIGGER IF EXISTS trg_tenant_lakehouse_audit_no_truncate ON public.tenant_lakehouse_audit;
CREATE TRIGGER trg_tenant_lakehouse_audit_no_truncate
BEFORE TRUNCATE ON public.tenant_lakehouse_audit
FOR EACH STATEMENT EXECUTE FUNCTION public.tenant_lakehouse_audit_append_only();

-- Returns the id of the first row whose link or hash does not verify, or NULL if
-- the tenant's whole chain is intact. Runs as the caller, so under RLS it sees
-- only the tenant set in uisce.current_tenant.
CREATE OR REPLACE FUNCTION public.tenant_lakehouse_audit_verify(p_tenant UUID) RETURNS BIGINT AS $$
DECLARE
    r RECORD;
    expected_prev TEXT := repeat('0', 64);
BEGIN
    FOR r IN SELECT * FROM public.tenant_lakehouse_audit WHERE tenant_id = p_tenant ORDER BY id LOOP
        IF r.prev_hash <> expected_prev THEN
            RETURN r.id;
        END IF;
        IF r.hash <> public.tenant_lakehouse_audit_hash(
               r.prev_hash, r.tenant_id, r.at, r.actor_id, r.action, r.before, r.after) THEN
            RETURN r.id;
        END IF;
        expected_prev := r.hash;
    END LOOP;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- Fail-closed tenant isolation, same shape as 20261016_001.
ALTER TABLE public.tenant_lakehouse_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_lakehouse_audit FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_lakehouse_audit_isolation_policy ON public.tenant_lakehouse_audit;
CREATE POLICY tenant_lakehouse_audit_isolation_policy ON public.tenant_lakehouse_audit
    FOR ALL
    USING (tenant_id = uisce_get_current_tenant())
    WITH CHECK (tenant_id = uisce_get_current_tenant());

COMMENT ON TABLE public.tenant_lakehouse_audit IS
    'Append-only hash-chained audit of a tenant''s lakehouse configuration. The trigger computes prev_hash/hash. Tamper-evident, not tamper-proof; alpha is the system of record; the lake may hold an immutable copy (ADR-032).';
