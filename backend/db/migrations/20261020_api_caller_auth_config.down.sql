-- Migration: 20261020_api_caller_auth_config.down.sql
ALTER TABLE semantic.api_endpoints DROP COLUMN IF EXISTS auth_type;
ALTER TABLE semantic.api_endpoints DROP COLUMN IF EXISTS auth_config;
ALTER TABLE semantic.api_endpoints DROP COLUMN IF EXISTS auth_secret_id;
