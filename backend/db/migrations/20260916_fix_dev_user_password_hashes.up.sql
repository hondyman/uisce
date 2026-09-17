-- Idempotent upsert seed for the two dev users used by MCP auth work.
-- These rows previously existed via an unrecorded action; their
-- password_hash column held the literal placeholder 'testpass' which
-- fails bcrypt's length check on every login attempt. This migration
-- converts the unrecorded seeding into a recorded one and ensures both
-- rows carry a valid bcrypt hash on fresh rebuilds.
--
-- Column set mirrors the live rows captured 2026-09-16 (see
-- backend/docs/INCIDENT_REPORT_20260916_LOGIN_BCRYPT.md). The
-- tenant_id column is the load-bearing one — without it, login works
-- but every tenant-scoped endpoint returns 401.
--
-- No BEGIN/COMMIT: the migration runner (internal/migrations/runner.go)
-- wraps each file in its own transaction and rejects transaction-control
-- statements.

INSERT INTO public.app_user (
    id, email, username, display_name, name,
    is_active, password_hash, tenant_id, language, status, attributes,
    permissions, is_core_admin, created_at, updated_at
)
VALUES
    ('da83f01c-da3e-480c-bac0-5ace1a97bc6e', 'testuser@example.com',
     'testuser@example.com', 'Test User', 'Test User',
     true, '$2a$10$Ja1GX27lwmjc5oo/vnd93.uSZ9O2WWamgvzV/89flg02.5MMx1epS', '99e99e99-99e9-49e9-89e9-99e99e99e999',
     'en', 'active', '{}', '[]', false, NOW(), NOW()),
    ('811e9f41-622f-4ef0-90ad-1098e1407d85', 'testuser2@example.com',
     'testuser2@example.com', 'Test User 2', 'Test User 2',
     true, '$2a$10$Ja1GX27lwmjc5oo/vnd93.uSZ9O2WWamgvzV/89flg02.5MMx1epS', '99e99e99-99e9-49e9-89e9-99e99e99e999',
     'en', 'active', '{}', '[]', false, NOW(), NOW())
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    updated_at    = NOW();
