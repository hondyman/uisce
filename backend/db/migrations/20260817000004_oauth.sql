-- Migration: Add OAuth2 refresh-flow columns to tenant_api_connections
-- Purpose: Persist OAuth client credentials, refresh token, token URL, and
--          last-known access-token expiry so the dispatcher can refresh
--          short-lived access tokens automatically instead of relying on a
--          static bearer pasted by the user.
-- Date: 2026-08-17
--
-- Column design:
--   * oauth_client_id              - public identifier, not encrypted
--   * oauth_client_secret_encrypted- long-lived secret (AES-256-GCM)
--   * oauth_refresh_token_encrypted- long-lived secret (AES-256-GCM)
--   * oauth_token_url              - public endpoint URL
--   * oauth_scopes                 - public space-separated scope list
--   * oauth_expires_at             - cache hint: last-known access-token
--                                    expiry; not authoritative (Redis is)
-- All encrypted columns use the same TokenEncryptor key as
-- auth_config_encrypted (API_TOKEN_ENCRYPTION_KEY).

BEGIN;

ALTER TABLE public.tenant_api_connections
    ADD COLUMN IF NOT EXISTS oauth_client_id               TEXT,
    ADD COLUMN IF NOT EXISTS oauth_client_secret_encrypted BYTEA,
    ADD COLUMN IF NOT EXISTS oauth_refresh_token_encrypted BYTEA,
    ADD COLUMN IF NOT EXISTS oauth_token_url               TEXT,
    ADD COLUMN IF NOT EXISTS oauth_scopes                  TEXT,
    ADD COLUMN IF NOT EXISTS oauth_expires_at              TIMESTAMPTZ;

COMMIT;
