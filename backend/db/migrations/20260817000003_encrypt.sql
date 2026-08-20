-- Migration: Encrypt tenant_api_connections.auth_config at rest
-- Purpose: Replace plaintext JSONB credential store with AES-256-GCM encrypted BYTEA
-- Date: 2026-08-17
--
-- NOTE: This migration DROPS the existing auth_config JSONB column. Any rows that
-- were saved before this migration will lose their stored credentials. As of the
-- date of this migration, the API dispatcher has only just shipped and no real
-- tenant credentials exist; the loss is acceptable. Tenants that did save
-- placeholder credentials must re-enter them after deploy.
--
-- The new column holds base64 (URL-safe, no padding) ciphertext from
-- security.TokenEncryptor.Encrypt(). The plaintext shape is a JSON object:
--   { "token": "...", "username": "...", "password": "...",
--     "api_key": "...", "header_name": "...", "client_id": "...",
--     "client_secret": "...", "refresh_token": "...", "scopes": "..." }
-- Decryption happens inside api_dispatcher.go (GetTenantConnection,
-- ExecuteEndpoint) before use.

BEGIN;

ALTER TABLE public.tenant_api_connections
    ADD COLUMN IF NOT EXISTS auth_config_encrypted BYTEA;

ALTER TABLE public.tenant_api_connections
    DROP COLUMN IF EXISTS auth_config;

COMMIT;
