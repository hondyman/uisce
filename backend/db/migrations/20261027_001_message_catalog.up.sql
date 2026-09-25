-- Message Catalog: the one source of every user-facing error, warning and
-- message (PeopleSoft Message Catalog style: message set + number, %1..%9
-- parameters, one row per language).
--
-- message_sets / message_catalog / tenant_message_catalog were created by
-- hand on alpha; this migration makes them reproducible (IF NOT EXISTS) and
-- adds what the catalog editor and runtime need:
--   * user_action: the translatable "what you can do" text shown to users.
--     description stays the internal explanation (admins, error bot) and is
--     never sent to clients.
--   * updated_by on both catalog tables.
--   * message_catalog_changes: maker-checker log. Every edit, through the UI
--     or the API, is a change row; it is applied immediately only where the
--     scope does not require a second approver, and it is the audit trail
--     either way.
--
-- Language codes are the app's BCP-47 locales (en, es, fr, de, pt-BR, ja,
-- zh-CN, ar); 20261027_002 converts the legacy ENG rows.

CREATE TABLE IF NOT EXISTS public.message_sets (
    set_nbr integer PRIMARY KEY,
    set_name text NOT NULL,
    module text NOT NULL,
    set_type text NOT NULL DEFAULT 'core',
    description text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.message_catalog (
    set_nbr integer NOT NULL,
    message_nbr integer NOT NULL,
    language_cd text NOT NULL DEFAULT 'en',
    severity text NOT NULL DEFAULT 'Error',
    message_text text NOT NULL,
    description text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (set_nbr, message_nbr, language_cd)
);

CREATE TABLE IF NOT EXISTS public.tenant_message_catalog (
    tenant_id text NOT NULL,
    set_nbr integer NOT NULL,
    message_nbr integer NOT NULL,
    language_cd text NOT NULL DEFAULT 'en',
    severity text NOT NULL DEFAULT 'Error',
    message_text text NOT NULL,
    description text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, set_nbr, message_nbr, language_cd)
);

ALTER TABLE public.tenant_message_catalog ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_message_catalog FORCE ROW LEVEL SECURITY;
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public'
                   AND tablename = 'tenant_message_catalog' AND policyname = 'tenant_isolation_policy') THEN
        CREATE POLICY tenant_isolation_policy ON public.tenant_message_catalog
            USING (tenant_id = current_setting('uisce.current_tenant', true));
    END IF;
END $$;

ALTER TABLE public.message_catalog
    ADD COLUMN IF NOT EXISTS user_action text,
    ADD COLUMN IF NOT EXISTS updated_by text;
ALTER TABLE public.tenant_message_catalog
    ADD COLUMN IF NOT EXISTS user_action text,
    ADD COLUMN IF NOT EXISTS updated_by text;

CREATE INDEX IF NOT EXISTS idx_tenant_message_catalog_key
    ON public.tenant_message_catalog (tenant_id, set_nbr, message_nbr);

CREATE TABLE IF NOT EXISTS public.message_catalog_changes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- NULL: a change to the core catalog (platform administrators only).
    tenant_id text,
    set_nbr integer NOT NULL,
    message_nbr integer NOT NULL,
    language_cd text NOT NULL,
    action text NOT NULL CHECK (action IN ('upsert', 'delete')),
    severity text,
    message_text text,
    description text,
    user_action text,
    -- The row as it was when the change was proposed (NULL: did not exist).
    -- Approval re-checks it, so a change can never overwrite an edit made
    -- after it was proposed.
    before jsonb,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'applied', 'rejected', 'withdrawn')),
    requested_by text NOT NULL,
    requested_at timestamptz NOT NULL DEFAULT now(),
    reason text,
    reviewed_by text,
    reviewed_at timestamptz,
    review_comment text,
    applied_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_message_catalog_changes_status
    ON public.message_catalog_changes (tenant_id, status, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_catalog_changes_key
    ON public.message_catalog_changes (set_nbr, message_nbr, requested_at DESC);

ALTER TABLE public.message_catalog_changes ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.message_catalog_changes FORCE ROW LEVEL SECURITY;
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public'
                   AND tablename = 'message_catalog_changes' AND policyname = 'tenant_isolation_policy') THEN
        -- Core changes (tenant_id NULL) are read and written by the
        -- application only for platform administrators; tenants see their own.
        CREATE POLICY tenant_isolation_policy ON public.message_catalog_changes
            USING (tenant_id IS NULL OR tenant_id = current_setting('uisce.current_tenant', true));
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        -- Core catalog writes happen only through an approved change
        -- (platform administrators, maker-checker), enforced by the app.
        GRANT SELECT ON public.message_sets TO app_user;
        GRANT SELECT, INSERT, UPDATE, DELETE ON public.message_catalog, public.tenant_message_catalog,
            public.message_catalog_changes TO app_user;
    END IF;
END $$;

COMMENT ON TABLE public.message_catalog_changes IS 'Maker-checker log for message catalog edits: every edit is a change row, applied on approval (or at once where approval is not required)';
COMMENT ON COLUMN public.message_catalog.description IS 'Internal explanation for administrators and the error bot; never sent to clients';
COMMENT ON COLUMN public.message_catalog.user_action IS 'Translatable "what you can do" text shown to users with the message';
