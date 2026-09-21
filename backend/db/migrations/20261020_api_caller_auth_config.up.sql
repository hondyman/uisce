-- Migration: 20261020_api_caller_auth_config.up.sql
-- Goal: Add auth configuration columns to semantic.api_endpoints so that
--   APICallerTransformer can make real HTTP calls with service-held credentials
--   instead of stamping {"verified": true} without making any network request.
--
-- auth_type:     'none' | 'api_key' | 'bearer' | 'oauth2_client_credentials' | 'basic_auth'
-- auth_config:   JSONB holding type-specific fields:
--   api_key:        {"header_name": "X-Api-Key", "key": "<encrypted>"}
--   bearer:         {"token": "<encrypted>"}
--   oauth2_client_credentials: {"client_id": "...", "client_secret": "<encrypted>", "token_url": "...", "scopes": "..."}
--   basic_auth:     {"username": "...", "password": "<encrypted>"}
-- auth_secret_id:  Infisical secret path for credentials that must never appear in plain text

BEGIN;

ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS auth_type text NOT NULL DEFAULT 'none';
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS auth_config jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS auth_secret_id text;

ALTER TABLE semantic.api_endpoints DROP CONSTRAINT IF EXISTS api_endpoints_auth_type_check;
ALTER TABLE semantic.api_endpoints ADD CONSTRAINT api_endpoints_auth_type_check
    CHECK (auth_type IN ('none', 'api_key', 'bearer', 'oauth2_client_credentials', 'basic_auth'));

COMMENT ON COLUMN semantic.api_endpoints.auth_type IS 'Authentication type for outbound calls from api_caller transformer';
COMMENT ON COLUMN semantic.api_endpoints.auth_config IS 'Encrypted credential configuration (API keys, OAuth2 tokens, etc.) — never expose in API responses';
COMMENT ON COLUMN semantic.api_endpoints.auth_secret_id IS 'Infisical secret path for credentials that must not appear in plain text';

COMMIT;
