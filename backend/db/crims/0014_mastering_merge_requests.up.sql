-- 0014_mastering_merge_requests.up.sql
-- Merges follow the entity's override policy (mdm.mastering_policy): under APPROVAL a steward's
-- merge is a request that N people other than the requester approve before it happens; under
-- DIRECT it happens at once. Run against crims. Same RLS as every mdm table. Additive.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0014_mastering_merge_requests.up.sql

\set ON_ERROR_STOP on
BEGIN;

CREATE TABLE IF NOT EXISTS mdm.golden_merge_request (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL,
    entity_cd           varchar(40) NOT NULL,
    candidate_id        uuid NOT NULL,                 -- mdm.<prefix>_match_candidate
    keep                varchar(1) NOT NULL DEFAULT 'a',
    note                text,
    approvals_required  int NOT NULL,
    status              varchar(20) NOT NULL DEFAULT 'PENDING',
    requested_by        text NOT NULL,
    requested_by_name   text,
    requested_at        timestamptz NOT NULL DEFAULT now(),
    decided_at          timestamptz,
    CONSTRAINT golden_merge_request_keep_ck CHECK (keep IN ('a', 'b')),
    CONSTRAINT golden_merge_request_status_ck CHECK (status IN ('PENDING', 'APPLIED', 'REJECTED', 'WITHDRAWN'))
);
CREATE UNIQUE INDEX IF NOT EXISTS golden_merge_request_pending_uq
    ON mdm.golden_merge_request (tenant_id, entity_cd, candidate_id) WHERE status = 'PENDING';

CREATE TABLE IF NOT EXISTS mdm.golden_merge_vote (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL,
    request_id    uuid NOT NULL REFERENCES mdm.golden_merge_request(id) ON DELETE CASCADE,
    approver      text NOT NULL,
    approver_name text,
    decision      varchar(10) NOT NULL,
    comment       text,
    decided_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT golden_merge_vote_uq UNIQUE (request_id, approver),
    CONSTRAINT golden_merge_vote_ck CHECK (decision IN ('APPROVE', 'REJECT'))
);

DO $rls$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['golden_merge_request', 'golden_merge_vote'] LOOP
        EXECUTE format('ALTER TABLE mdm.%I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE mdm.%I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_read', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR SELECT
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
                OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid,
                                         '00000000-0000-0000-0000-000000000001'::uuid)))$p$, t || '_tenant_read', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON mdm.%I', t || '_tenant_write', t);
        EXECUTE format($p$CREATE POLICY %I ON mdm.%I AS PERMISSIVE FOR ALL
            USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
            WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))$p$, t || '_tenant_write', t);
    END LOOP;
END
$rls$;

COMMIT;
