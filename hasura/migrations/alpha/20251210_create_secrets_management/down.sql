-- Rollback secrets management schema

DROP VIEW IF EXISTS secrets_needing_rotation;
DROP TABLE IF EXISTS secret_access_log;
DROP TABLE IF EXISTS secret_version;
DROP TABLE IF EXISTS secret_policy;
DROP TABLE IF EXISTS secret_metadata;
