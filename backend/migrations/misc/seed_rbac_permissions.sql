-- ============================================================================
-- RBAC System - Seed Data
-- Standard Roles & Permissions for Fortune 500 Compliance
-- ============================================================================

-- ============================================================================
-- 1. SYSTEM PERMISSIONS
-- ============================================================================

-- Process Management Permissions
INSERT INTO bp_permissions (tenant_id, permission_key, permission_name, description, resource_type, action, is_system) VALUES
('00000000-0000-0000-0000-000000000000', 'process.read', 'View Processes', 'View process definitions and instances', 'process', 'read', true),
('00000000-0000-0000-0000-000000000000', 'process.create', 'Create Processes', 'Create new process definitions', 'process', 'create', true),
('00000000-0000-0000-0000-000000000000', 'process.update', 'Edit Processes', 'Modify existing process definitions', 'process', 'update', true),
('00000000-0000-0000-0000-000000000000', 'process.delete', 'Delete Processes', 'Remove process definitions (soft delete)', 'process', 'delete', true),
('00000000-0000-0000-0000-000000000000', 'process.execute', 'Execute Processes', 'Start process instances', 'process', 'execute', true),
('00000000-0000-0000-0000-000000000000', 'process.publish', 'Publish Processes', 'Publish process definitions to production', 'process', 'approve', true),

-- Step Permissions
('00000000-0000-0000-0000-000000000000', 'step.read', 'View Steps', 'View process steps', 'step', 'read', true),
('00000000-0000-0000-0000-000000000000', 'step.execute', 'Execute Steps', 'Execute assigned steps', 'step', 'execute', true),
('00000000-0000-0000-0000-000000000000', 'step.reassign', 'Reassign Steps', 'Reassign steps to other users', 'step', 'update', true),
('00000000-0000-0000-0000-000000000000', 'step.approve', 'Approve Steps', 'Approve step completion', 'step', 'approve', true),

-- Field Permissions (for field-level security)
('00000000-0000-0000-0000-000000000000', 'field.read.sensitive', 'View Sensitive Fields', 'View PII and sensitive data fields', 'field', 'read', true),
('00000000-0000-0000-0000-000000000000', 'field.update.sensitive', 'Edit Sensitive Fields', 'Modify PII and sensitive data', 'field', 'update', true),
('00000000-0000-0000-0000-000000000000', 'field.read.financial', 'View Financial Fields', 'View financial data', 'field', 'read', true),
('00000000-0000-0000-0000-000000000000', 'field.update.financial', 'Edit Financial Fields', 'Modify financial data', 'field', 'update', true),

-- Document Permissions
('00000000-0000-0000-0000-000000000000', 'document.read', 'View Documents', 'View attached documents', 'document', 'read', true),
('00000000-0000-0000-0000-000000000000', 'document.upload', 'Upload Documents', 'Upload new documents', 'document', 'create', true),
('00000000-0000-0000-0000-000000000000', 'document.delete', 'Delete Documents', 'Remove documents', 'document', 'delete', true),

-- Report Permissions
('00000000-0000-0000-0000-000000000000', 'report.read', 'View Reports', 'View reports and analytics', 'report', 'read', true),
('00000000-0000-0000-0000-000000000000', 'report.export', 'Export Reports', 'Export reports to PDF/Excel', 'report', 'read', true),
('00000000-0000-0000-0000-000000000000', 'report.create', 'Create Reports', 'Create custom reports', 'report', 'create', true),

-- Admin Permissions
('00000000-0000-0000-0000-000000000000', 'admin.users', 'Manage Users', 'Create and manage user accounts', 'admin', 'update', true),
('00000000-0000-0000-0000-000000000000', 'admin.roles', 'Manage Roles', 'Create and assign roles', 'admin', 'update', true),
('00000000-0000-0000-0000-000000000000', 'admin.permissions', 'Manage Permissions', 'Grant and revoke permissions', 'admin', 'update', true),
('00000000-0000-0000-0000-000000000000', 'admin.audit', 'View Audit Logs', 'Access audit and compliance logs', 'admin', 'read', true),
('00000000-0000-0000-0000-000000000000', 'admin.delegation', 'Manage Delegations', 'Create and manage approval delegations', 'admin', 'update', true),

-- Approval Permissions
('00000000-0000-0000-0000-000000000000', 'approval.level1', 'Level 1 Approval', 'Approve up to $10,000', 'approval', 'approve', true),
('00000000-0000-0000-0000-000000000000', 'approval.level2', 'Level 2 Approval', 'Approve up to $50,000', 'approval', 'approve', true),
('00000000-0000-0000-0000-000000000000', 'approval.level3', 'Level 3 Approval', 'Approve up to $100,000', 'approval', 'approve', true),
('00000000-0000-0000-0000-000000000000', 'approval.unlimited', 'Unlimited Approval', 'Approve any amount', 'approval', 'approve', true);

-- ============================================================================
-- 2. STANDARD ROLES
-- ============================================================================

-- VIEWER ROLE (Read-only)
INSERT INTO bp_roles (tenant_id, datasource_id, role_key, role_name, description, role_type, role_level) VALUES
('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', 'viewer', 'Viewer', 'Read-only access to view processes and reports', 'system', 'viewer');

-- EDITOR ROLE (Read-write, no approval)
INSERT INTO bp_roles (tenant_id, datasource_id, role_key, role_name, description, role_type, role_level) VALUES
('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', 'editor', 'Editor', 'Create and edit processes, execute steps', 'system', 'editor');

-- APPROVER ROLE (Editor + Level 1 approval)
INSERT INTO bp_roles (tenant_id, datasource_id, role_key, role_name, description, role_type, role_level) VALUES
('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', 'approver', 'Approver', 'Editor permissions plus approval authority', 'system', 'editor');

-- ADMIN ROLE (Full access except system config)
INSERT INTO bp_roles (tenant_id, datasource_id, role_key, role_name, description, role_type, role_level) VALUES
('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', 'admin', 'Administrator', 'Full access to manage processes, users, and settings', 'system', 'admin');

-- SUPER ADMIN ROLE (Full system access)
INSERT INTO bp_roles (tenant_id, datasource_id, role_key, role_name, description, role_type, role_level) VALUES
('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', 'super_admin', 'Super Administrator', 'Unrestricted access to all system features', 'system', 'super_admin');

-- COMPLIANCE OFFICER ROLE (Read-only + audit access)
INSERT INTO bp_roles (tenant_id, datasource_id, role_key, role_name, description, role_type, role_level) VALUES
('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', 'compliance_officer', 'Compliance Officer', 'Read access plus audit log viewing', 'system', 'viewer');

-- PROCESS OWNER ROLE (Full control of assigned processes)
INSERT INTO bp_roles (tenant_id, datasource_id, role_key, role_name, description, role_type, role_level) VALUES
('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', 'process_owner', 'Process Owner', 'Full control over assigned processes', 'system', 'editor');

-- ============================================================================
-- 3. ROLE-PERMISSION MAPPINGS
-- ============================================================================

-- VIEWER PERMISSIONS
INSERT INTO bp_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM bp_roles r, bp_permissions p
WHERE r.role_key = 'viewer'
  AND p.permission_key IN (
    'process.read',
    'step.read',
    'document.read',
    'report.read'
  );

-- EDITOR PERMISSIONS
INSERT INTO bp_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM bp_roles r, bp_permissions p
WHERE r.role_key = 'editor'
  AND p.permission_key IN (
    'process.read',
    'process.create',
    'process.update',
    'process.execute',
    'step.read',
    'step.execute',
    'step.reassign',
    'document.read',
    'document.upload',
    'report.read',
    'report.export'
  );

-- APPROVER PERMISSIONS (Editor + Approval)
INSERT INTO bp_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM bp_roles r, bp_permissions p
WHERE r.role_key = 'approver'
  AND p.permission_key IN (
    'process.read',
    'process.create',
    'process.update',
    'process.execute',
    'step.read',
    'step.execute',
    'step.reassign',
    'step.approve',
    'document.read',
    'document.upload',
    'report.read',
    'report.export',
    'approval.level1'
  );

-- ADMIN PERMISSIONS (Almost everything)
INSERT INTO bp_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM bp_roles r, bp_permissions p
WHERE r.role_key = 'admin'
  AND p.permission_key NOT IN ('approval.unlimited'); -- All except unlimited approval

-- SUPER ADMIN PERMISSIONS (Everything)
INSERT INTO bp_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM bp_roles r, bp_permissions p
WHERE r.role_key = 'super_admin';

-- COMPLIANCE OFFICER PERMISSIONS
INSERT INTO bp_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM bp_roles r, bp_permissions p
WHERE r.role_key = 'compliance_officer'
  AND p.permission_key IN (
    'process.read',
    'step.read',
    'document.read',
    'report.read',
    'report.export',
    'admin.audit'
  );

-- PROCESS OWNER PERMISSIONS
INSERT INTO bp_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM bp_roles r, bp_permissions p
WHERE r.role_key = 'process_owner'
  AND p.permission_key IN (
    'process.read',
    'process.create',
    'process.update',
    'process.delete',
    'process.execute',
    'process.publish',
    'step.read',
    'step.execute',
    'step.reassign',
    'step.approve',
    'document.read',
    'document.upload',
    'document.delete',
    'report.read',
    'report.export',
    'report.create',
    'approval.level2'
  );

-- ============================================================================
-- 4. FIELD-LEVEL PERMISSIONS (Sensitive Data)
-- ============================================================================

-- VIEWER: No access to sensitive fields
INSERT INTO bp_field_permissions (tenant_id, datasource_id, role_id, resource_type, field_name, permission_level)
SELECT '00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', r.id, 'process', field_name, 'none'
FROM bp_roles r,
     (VALUES ('ssn'), ('tax_id'), ('bank_account'), ('credit_card')) AS fields(field_name)
WHERE r.role_key = 'viewer';

-- EDITOR: Read-only on sensitive fields (masked)
INSERT INTO bp_field_permissions (tenant_id, datasource_id, role_id, resource_type, field_name, permission_level)
SELECT '00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', r.id, 'process', field_name, 'mask'
FROM bp_roles r,
     (VALUES ('ssn'), ('tax_id'), ('bank_account'), ('credit_card')) AS fields(field_name)
WHERE r.role_key = 'editor';

-- ADMIN: Full access to sensitive fields
INSERT INTO bp_field_permissions (tenant_id, datasource_id, role_id, resource_type, field_name, permission_level)
SELECT '00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', r.id, 'process', field_name, 'write'
FROM bp_roles r,
     (VALUES ('ssn'), ('tax_id'), ('bank_account'), ('credit_card')) AS fields(field_name)
WHERE r.role_key IN ('admin', 'super_admin');

-- ============================================================================
-- 5. FIELD MASKING RULES (PII/Sensitive Data)
-- ============================================================================

-- SSN Masking
INSERT INTO bp_field_masking_rules (tenant_id, datasource_id, resource_type, field_name, masking_type, masking_pattern, unmasked_roles)
SELECT 
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'process',
    'ssn',
    'partial',
    'XXX-XX-####',
    ARRAY(SELECT id FROM bp_roles WHERE role_key IN ('admin', 'super_admin', 'compliance_officer'));

-- Tax ID Masking
INSERT INTO bp_field_masking_rules (tenant_id, datasource_id, resource_type, field_name, masking_type, masking_pattern, unmasked_roles)
SELECT 
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'process',
    'tax_id',
    'partial',
    'XX-XXXXXXX',
    ARRAY(SELECT id FROM bp_roles WHERE role_key IN ('admin', 'super_admin', 'compliance_officer'));

-- Bank Account Masking
INSERT INTO bp_field_masking_rules (tenant_id, datasource_id, resource_type, field_name, masking_type, masking_pattern, unmasked_roles)
SELECT 
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'process',
    'bank_account',
    'partial',
    'XXXX-####',
    ARRAY(SELECT id FROM bp_roles WHERE role_key IN ('admin', 'super_admin'));

-- Credit Card Masking
INSERT INTO bp_field_masking_rules (tenant_id, datasource_id, resource_type, field_name, masking_type, masking_pattern, unmasked_roles)
SELECT 
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'process',
    'credit_card',
    'partial',
    'XXXX-XXXX-XXXX-####',
    ARRAY(SELECT id FROM bp_roles WHERE role_key IN ('admin', 'super_admin'));

-- Email Partial Masking
INSERT INTO bp_field_masking_rules (tenant_id, datasource_id, resource_type, field_name, masking_type, masking_pattern, unmasked_roles)
SELECT 
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'process',
    'email',
    'partial',
    'X***@domain.com',
    ARRAY(SELECT id FROM bp_roles WHERE role_key IN ('admin', 'super_admin', 'process_owner'));

-- ============================================================================
-- 6. SAMPLE TEAMS
-- ============================================================================

-- Finance Team
INSERT INTO bp_teams (tenant_id, datasource_id, team_key, team_name, description, team_type)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'finance',
    'Finance Team',
    'Financial operations and approvals',
    'functional'
);

-- HR Team
INSERT INTO bp_teams (tenant_id, datasource_id, team_key, team_name, description, team_type)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'hr',
    'Human Resources',
    'Employee lifecycle and benefits management',
    'functional'
);

-- Operations Team
INSERT INTO bp_teams (tenant_id, datasource_id, team_key, team_name, description, team_type)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'operations',
    'Operations Team',
    'Day-to-day business operations',
    'functional'
);

-- Compliance Team
INSERT INTO bp_teams (tenant_id, datasource_id, team_key, team_name, description, team_type)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'compliance',
    'Compliance & Audit',
    'Regulatory compliance and audit oversight',
    'functional'
);

-- ============================================================================
-- 7. VERIFICATION QUERIES
-- ============================================================================

-- Verify roles created
SELECT role_key, role_name, role_level, 
       (SELECT COUNT(*) FROM bp_role_permissions WHERE role_id = bp_roles.id) as permission_count
FROM bp_roles
ORDER BY role_level, role_key;

-- Verify permissions created
SELECT COUNT(*) as total_permissions,
       COUNT(DISTINCT resource_type) as resource_types,
       COUNT(DISTINCT action) as action_types
FROM bp_permissions;

-- Verify field-level permissions
SELECT r.role_key, COUNT(*) as field_permission_count
FROM bp_field_permissions fp
JOIN bp_roles r ON fp.role_id = r.id
GROUP BY r.role_key
ORDER BY field_permission_count DESC;

-- Verify masking rules
SELECT field_name, masking_type, masking_pattern,
       (SELECT COUNT(*) FROM unnest(unmasked_roles)) as unmasked_role_count
FROM bp_field_masking_rules
ORDER BY field_name;

-- Verify teams
SELECT team_key, team_name, team_type
FROM bp_teams
ORDER BY team_key;
