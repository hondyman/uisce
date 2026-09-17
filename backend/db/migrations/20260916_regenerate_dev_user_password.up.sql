-- Regenerate dev user password hashes. The previous preimage
-- (16 chars from openssl rand -base64 12) was never recorded by
-- the operator and the temp file was deleted at cleanup; the only
-- surviving copy in the server log was truncated when the
-- request-trace middleware redaction landed (commit 58b2ae9f9).
-- Generating a fresh value with a fresh bcrypt hash. New preimage
-- captured to a temp file (operator records it externally before
-- the file is deleted).
--
-- Idempotent upsert: inserts on fresh rebuild (rows are otherwise
-- unrecorded), updates on current DB. Same shape as
-- 20260916_fix_dev_user_password_hashes.up.sql.
--
-- Reference: backend/docs/INCIDENT_REPORT_20260916_LOGIN_BCRYPT.md

INSERT INTO public.app_user (
    id, email, username, display_name, name,
    is_active, password_hash, tenant_id, language, status, attributes,
    permissions, is_core_admin, created_at, updated_at
)
VALUES
    ('da83f01c-da3e-480c-bac0-5ace1a97bc6e', 'testuser@example.com',
     'testuser@example.com', 'Test User', 'Test User',
     true, '$2a$10$n1btyilO6DUZ83bh/HP5n.ay6wYS1nqyv.bK5pgQPJAUX5A/.izNu', '99e99e99-99e9-49e9-89e9-99e99e99e999',
     'en', 'active', '{}', '[]', false, NOW(), NOW()),
    ('811e9f41-622f-4ef0-90ad-1098e1407d85', 'testuser2@example.com',
     'testuser2@example.com', 'Test User 2', 'Test User 2',
     true, '$2a$10$n1btyilO6DUZ83bh/HP5n.ay6wYS1nqyv.bK5pgQPJAUX5A/.izNu', '99e99e99-99e9-49e9-89e9-99e99e99e999',
     'en', 'active', '{}', '[]', false, NOW(), NOW())
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    updated_at    = NOW();
