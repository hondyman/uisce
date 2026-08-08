-- Seed Tenant Provisioning Workflow Template for BP Designer
-- This creates a process template that can be used to provision new tenant instances

-- Insert step types for tenant provisioning
INSERT INTO process_step_types (key, label, description, icon_svg, default_data, is_system)
VALUES
('tenant_register', 'Register Tenant', 'Register a new tenant in the platform', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2z"/><path d="M12 8v8"/><path d="M8 12h8"/></svg>', '{}', true),
('tenant_instance_register', 'Register Instance', 'Register a new tenant instance', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><circle cx="6" cy="6" r="1"/><circle cx="6" cy="18" r="1"/></svg>', '{}', true),
('tenant_database_create', 'Create Database', 'Create a new database for the tenant', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/><path d="M3 12c0 1.66 4 3 9 3s9-1.34 9-3"/></svg>', '{}', true),
('tenant_schema_clone', 'Clone Schema', 'Clone gold copy schema to new database', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="8" y="2" width="8" height="4" rx="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><path d="M12 11h4"/></svg>', '{}', true),
('tenant_namespace_create', 'Create Namespace', 'Create Lakekeeper namespace for tenant', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>', '{}', true),
('tenant_products_clone', 'Clone Products', 'Clone gold copy products to tenant', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><rect x="8" y="2" width="8" height="4" rx="1"/></svg>', '{}', true),
('tenant_emit_event', 'Emit Event', 'Emit provisioning complete event', '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M22 17H2a3 3 0 0 0 3-3V9a7 7 0 0 1 14 0v5a3 3 0 0 0 3 3zm-8.27 4a2 2 0 0 1-3.46 0"/></svg>', '{}', true)
ON CONFLICT (key) DO NOTHING;

-- Insert the tenant provisioning process template
INSERT INTO processes (id, name, description, version, nodes, edges, status, tenant_id, datasource_id)
VALUES (
  'proc-tenant-provisioning',
  'Tenant Provisioning',
  'Provisions a complete tenant instance including database, schema clone, Lakekeeper namespace, and product cloning from gold copy',
  1,
  '[
    {
      "id": "node-1",
      "type": "tenant_register",
      "data": {
        "label": "Register Tenant",
        "activityName": "RegisterTenant",
        "description": "Register a new tenant in the platform"
      },
      "position": {"x": 100, "y": 100}
    },
    {
      "id": "node-2",
      "type": "tenant_instance_register",
      "data": {
        "label": "Register Instance",
        "activityName": "RegisterInstance",
        "description": "Register a new tenant instance"
      },
      "position": {"x": 100, "y": 200}
    },
    {
      "id": "node-3",
      "type": "tenant_database_create",
      "data": {
        "label": "Create Database",
        "activityName": "CreateTenantDatabase",
        "description": "Create a new database for the tenant"
      },
      "position": {"x": 100, "y": 300}
    },
    {
      "id": "node-4",
      "type": "tenant_schema_clone",
      "data": {
        "label": "Clone Schema",
        "activityName": "CloneSchemaFromGoldCopy",
        "description": "Clone gold copy schema to new database"
      },
      "position": {"x": 100, "y": 400}
    },
    {
      "id": "node-5",
      "type": "tenant_namespace_create",
      "data": {
        "label": "Create Namespace",
        "activityName": "CreateLakekeeperNamespace",
        "description": "Create Lakekeeper namespace for tenant"
      },
      "position": {"x": 100, "y": 500}
    },
    {
      "id": "node-6",
      "type": "tenant_products_clone",
      "data": {
        "label": "Clone Products",
        "activityName": "CloneGoldCopyProducts",
        "description": "Clone gold copy products to tenant"
      },
      "position": {"x": 100, "y": 600}
    },
    {
      "id": "node-7",
      "type": "tenant_emit_event",
      "data": {
        "label": "Emit Event",
        "activityName": "EmitProvisioningEvent",
        "description": "Emit provisioning complete event"
      },
      "position": {"x": 100, "y": 700}
    }
  ]'::jsonb,
  '[
    {"id": "edge-1", "source": "node-1", "target": "node-2"},
    {"id": "edge-2", "source": "node-2", "target": "node-3"},
    {"id": "edge-3", "source": "node-3", "target": "node-4"},
    {"id": "edge-4", "source": "node-4", "target": "node-5"},
    {"id": "edge-5", "source": "node-5", "target": "node-6"},
    {"id": "edge-6", "source": "node-6", "target": "node-7"}
  ]'::jsonb,
  'published',
  NULL,
  NULL
) ON CONFLICT (id) DO NOTHING;
