-- Rollback for audit_log table creation
DROP TABLE IF EXISTS ops_audit_log CASCADE;
