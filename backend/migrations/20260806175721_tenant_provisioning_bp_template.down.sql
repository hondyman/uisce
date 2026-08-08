-- Rollback tenant provisioning BP template

DELETE FROM processes WHERE id = 'proc-tenant-provisioning';

DELETE FROM process_step_types WHERE key IN (
  'tenant_register',
  'tenant_instance_register',
  'tenant_database_create',
  'tenant_schema_clone',
  'tenant_namespace_create',
  'tenant_products_clone',
  'tenant_emit_event'
);
