-- Recovers the public.app_user table's DDL into the managed migration chain.
--
-- This table has never had a migration anywhere in git history — it was
-- created directly against alpha by hand at some point before this
-- recovery, the same "created in the database, not via migrations" gap
-- 20260906_001_extend_public_users_view.up.sql already documents for the
-- public.users view built on top of it (that view's `FROM app_user` is why
-- this file must sort before it — 000 vs 001 on the same date).
--
-- Columns, constraints, indexes, RLS policy, and app_user-role grants below
-- are captured verbatim from alpha's live table via information_schema /
-- pg_constraint / pg_indexes / pg_policy, not guessed. Discovered while
-- building the ephemeral CI database (Feature A): backend/internal/reports's
-- gated test suite inserts into this table, which a fresh schema build
-- would otherwise be missing entirely.

CREATE TABLE IF NOT EXISTS public.app_user (
    id                text                     NOT NULL,
    email             text                     NOT NULL,
    display_name      text,
    created_at        timestamptz              NOT NULL DEFAULT now(),
    is_active         boolean                  NOT NULL DEFAULT true,
    name              character varying,
    role              character varying,
    username          character varying,
    organization      character varying,
    permissions       jsonb                    DEFAULT '[]'::jsonb,
    is_core_admin     boolean                  DEFAULT false,
    password_hash     text,
    salt              text,
    tenant_id         text,
    language          character varying        DEFAULT 'en'::character varying,
    last_login_time   timestamptz,
    user_preferences  jsonb                    DEFAULT '{}'::jsonb,
    updated_at_time   timestamptz              DEFAULT now(),
    first_name        character varying,
    last_name         character varying,
    status            text                     DEFAULT 'active'::text,
    updated_at        timestamptz              DEFAULT now(),
    attributes        jsonb                    NOT NULL DEFAULT '{}'::jsonb,

    CONSTRAINT app_user_pkey PRIMARY KEY (id),
    CONSTRAINT app_user_email_key UNIQUE (email),
    CONSTRAINT app_user_username_key UNIQUE (username)
);

CREATE INDEX IF NOT EXISTS idx_app_user_active ON public.app_user (is_active) WHERE (is_active = true);
CREATE INDEX IF NOT EXISTS idx_app_user_email ON public.app_user (email);
CREATE INDEX IF NOT EXISTS idx_users_language ON public.app_user (language);

ALTER TABLE public.app_user ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.app_user FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policy WHERE polname = 'tenant_isolation_policy' AND polrelid = 'public.app_user'::regclass
    ) THEN
        CREATE POLICY tenant_isolation_policy ON public.app_user
            USING (tenant_id = current_setting('uisce.current_tenant', true));
    END IF;
END $$;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.app_user TO app_user;
