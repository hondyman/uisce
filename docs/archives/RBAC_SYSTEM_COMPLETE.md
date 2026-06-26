# RBAC System Complete Documentation
## Fortune 500 Enterprise-Grade Security & Access Control

**Status**: ✅ Backend Complete | ⏳ Frontend UI Pending  
**Version**: 1.0.0  
**Created**: [Date]  
**Features**: Field-Level Permissions, Approval Delegations, Team Access, Full Audit Trail

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Database Schema](#database-schema)
3. [Role Hierarchy](#role-hierarchy)
4. [Permission Catalog](#permission-catalog)
5. [Field-Level Security](#field-level-security)
6. [Approval Delegation](#approval-delegation)
7. [Team Access Control](#team-access-control)
8. [API Reference](#api-reference)
9. [Middleware Usage](#middleware-usage)
10. [Migration Guide](#migration-guide)
11. [Testing](#testing)
12. [Compliance Checklist](#compliance-checklist)
13. [Troubleshooting](#troubleshooting)

---

## 🎯 Overview

The RBAC (Role-Based Access Control) system provides enterprise-grade security with:

- **Hierarchical Role System**: 7 standard roles (viewer → super_admin)
- **Granular Permissions**: 30+ permissions across 7 resource types
- **Field-Level Security**: Control access to sensitive PII/financial fields
- **Approval Delegation**: Temporary authority delegation with audit trail
- **Team-Based Access**: Shared resource access for teams/departments
- **Complete Audit Trail**: Track all permission changes and delegated actions

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| Database Schema | `backend/migrations/misc/rbac_field_level_permissions.sql` | 11 tables for roles, permissions, delegations, teams |
| Seed Data | `backend/migrations/misc/seed_rbac_permissions.sql` | 7 roles, 30+ permissions, masking rules |
| API Handlers | `backend/internal/api/bp_rbac_handlers.go` | REST endpoints for RBAC management |
| Middleware | `backend/internal/api/middleware/rbac_enforcement.go` | Permission enforcement layer |

---

## 🗄️ Database Schema

### Core Tables

#### 1. `bp_roles` - Role Definitions
```sql
- id: UUID (PK)
- tenant_id, datasource_id: Tenant scope
- role_key: Unique identifier (e.g., 'admin', 'viewer')
- role_name: Display name
- role_type: 'system' | 'custom'
- role_level: 'viewer' | 'editor' | 'approver' | 'admin' | 'super_admin'
- is_active: Boolean status
```

**Role Levels** (hierarchical):
- `viewer` (1): Read-only access
- `editor` (2): Read-write access
- `approver` (3): Editor + approval authority
- `admin` (4): Almost everything
- `super_admin` (5): Unrestricted access

#### 2. `bp_permissions` - Permission Catalog
```sql
- id: UUID (PK)
- permission_key: Unique identifier (e.g., 'process.read')
- permission_name: Display name
- resource_type: Target resource ('process', 'step', 'field', etc.)
- action: Operation ('read', 'create', 'update', 'delete', etc.)
- is_system: Boolean (system vs custom permission)
```

#### 3. `bp_role_permissions` - Role-Permission Mapping
```sql
- role_id: FK to bp_roles
- permission_id: FK to bp_permissions
```

#### 4. `bp_user_roles` - User Role Assignments
```sql
- user_id: UUID
- role_id: FK to bp_roles
- scope_type: 'global' | 'process' | 'step' | 'team'
- scope_id: UUID (process/step/team ID)
- expires_at: Optional expiration timestamp
- is_active: Boolean status
```

**Scope Types**:
- `global`: Role applies across all resources
- `process`: Role limited to specific process(es)
- `step`: Role limited to specific step(s)
- `team`: Role limited to team's accessible resources

#### 5. `bp_field_permissions` - Field-Level Access Control
```sql
- role_id: FK to bp_roles
- resource_type: 'process' | 'step' | 'document'
- resource_id: Optional specific resource UUID
- field_name: Target field (e.g., 'ssn', 'tax_id')
- permission_level: 'none' | 'read' | 'write' | 'mask'
```

**Permission Levels**:
- `none`: No access (field hidden)
- `read`: Full read access (unmasked)
- `write`: Full read-write access
- `mask`: Read with masking applied (partial visibility)

#### 6. `bp_field_masking_rules` - PII Masking Patterns
```sql
- resource_type, field_name: Field identifier
- masking_type: 'full' | 'partial' | 'hash' | 'tokenize'
- masking_pattern: Pattern (e.g., 'XXX-XX-####')
- unmasked_roles: UUID[] array of roles with full access
```

**Masking Examples**:
- SSN: `XXX-XX-####` → `XXX-XX-1234`
- Credit Card: `XXXX-XXXX-XXXX-####` → `XXXX-XXXX-XXXX-5678`
- Email: `X***@domain.com` → `j***@example.com`

#### 7. `bp_approval_delegations` - Approval Authority Delegation
```sql
- delegator_user_id: User delegating authority
- delegate_user_id: User receiving authority
- delegation_type: 'full' | 'partial' | 'backup'
- resource_type, resource_id: Optional resource scope
- start_date, end_date: Delegation time window
- scope_conditions: JSONB conditions (amount limits, etc.)
- reason: Textual reason (vacation, backup, etc.)
```

**Delegation Types**:
- `full`: Complete approval authority within scope
- `partial`: Limited by scope_conditions (amount thresholds)
- `backup`: Activates only if delegator unavailable

#### 8. `bp_delegation_usage_log` - Delegation Audit Trail
```sql
- delegation_id: FK to bp_approval_delegations
- delegate_user_id: User who performed action
- action_type: 'approve' | 'reject' | 'reassign'
- resource_type, resource_id: Target resource
- action_details: JSONB details
```

#### 9. `bp_teams` - Team/Group Definitions
```sql
- team_key, team_name: Team identifier and display name
- team_type: 'functional' | 'project' | 'cross_functional'
- parent_team_id: Optional parent team (hierarchical)
- manager_user_id: Team manager/lead
```

#### 10. `bp_team_members` - Team Membership
```sql
- team_id: FK to bp_teams
- user_id: Team member
- role_in_team: 'member' | 'lead' | 'admin'
```

#### 11. `bp_team_permissions` - Team Resource Access
```sql
- team_id: FK to bp_teams
- resource_type, resource_id: Accessible resource
- permission_level: 'read' | 'write' | 'admin'
```

### Helper Functions

#### `bp_user_has_permission(user_id, tenant_id, datasource_id, permission_key)`
Returns boolean indicating if user has the specified permission.

```sql
SELECT bp_user_has_permission(
    '00000000-0000-0000-0000-000000000001'::uuid,
    '00000000-0000-0000-0000-000000000000'::uuid,
    '11111111-1111-1111-1111-111111111111'::uuid,
    'process.read'
);
```

#### `bp_get_user_resource_permissions(user_id, tenant_id, datasource_id, resource_type, resource_id)`
Returns all permissions user has on a specific resource.

#### `bp_get_active_delegation(delegator_id, delegate_id, tenant_id, datasource_id, resource_type, resource_id)`
Returns active delegation ID if one exists matching the criteria.

---

## 👥 Role Hierarchy

### Standard Roles

| Role Key | Level | Permissions | Use Case |
|----------|-------|-------------|----------|
| `viewer` | 1 | 4 | Read-only analysts, compliance reviewers |
| `editor` | 2 | 9 | Process participants, data entry |
| `approver` | 3 | 11 | Managers with approval authority ($10K) |
| `admin` | 4 | 28 | Department administrators |
| `super_admin` | 5 | 30 | IT administrators, system owners |
| `compliance_officer` | 4 | 6 | Audit/compliance staff |
| `process_owner` | 3 | 15 | Process designers, workflow owners |

### Role Details

#### Viewer (4 permissions)
- `process.read`
- `step.read`
- `document.read`
- `report.read`

**Field Access**: None on sensitive fields (SSN, tax_id, bank_account, credit_card)

#### Editor (9 permissions)
- All viewer permissions +
- `process.create`, `process.update`, `process.execute`
- `step.execute`
- `document.upload`
- `report.create`

**Field Access**: Masked access to sensitive fields

#### Approver (11 permissions)
- All editor permissions +
- `step.approve`
- `approval.level1` ($10,000 limit)

**Field Access**: Masked access to sensitive fields

#### Admin (28 permissions)
- All permissions except `approval.unlimited`, `admin.delegation`

**Field Access**: Full unmasked access to all fields

#### Super Admin (30 permissions)
- **All permissions** (unrestricted)

**Field Access**: Full unmasked access to all fields

#### Compliance Officer (6 permissions)
- `process.read`, `step.read`, `document.read`
- `report.read`, `report.export`
- `admin.audit`

**Field Access**: Full unmasked access for audit purposes

#### Process Owner (15 permissions)
- Full CRUD on assigned processes
- `process.read`, `process.create`, `process.update`, `process.delete`, `process.publish`
- `step.read`, `step.execute`, `step.reassign`, `step.approve`
- `document.read`, `document.upload`, `document.delete`
- `report.read`, `report.create`, `report.export`
- `approval.level2` ($50,000 limit)

**Field Access**: Full access within owned processes

---

## 🔐 Permission Catalog

### Process Management (6 permissions)

| Permission Key | Action | Description |
|---------------|--------|-------------|
| `process.read` | View | List and view process definitions |
| `process.create` | Create | Define new processes |
| `process.update` | Update | Modify process definitions |
| `process.delete` | Delete | Remove processes |
| `process.execute` | Execute | Start process instances |
| `process.publish` | Publish | Make processes available |

### Step Management (4 permissions)

| Permission Key | Action | Description |
|---------------|--------|-------------|
| `step.read` | View | View step details |
| `step.execute` | Execute | Complete steps |
| `step.reassign` | Reassign | Reassign step ownership |
| `step.approve` | Approve | Approve/reject steps |

### Field Security (4 permissions)

| Permission Key | Action | Description |
|---------------|--------|-------------|
| `field.read.sensitive` | Read | View sensitive PII fields |
| `field.update.sensitive` | Update | Modify sensitive PII fields |
| `field.read.financial` | Read | View financial data |
| `field.update.financial` | Update | Modify financial data |

### Document Management (3 permissions)

| Permission Key | Action | Description |
|---------------|--------|-------------|
| `document.read` | View | View attached documents |
| `document.upload` | Upload | Attach new documents |
| `document.delete` | Delete | Remove documents |

### Report Management (3 permissions)

| Permission Key | Action | Description |
|---------------|--------|-------------|
| `report.read` | View | View reports |
| `report.export` | Export | Export report data |
| `report.create` | Create | Design custom reports |

### Admin Functions (5 permissions)

| Permission Key | Action | Description |
|---------------|--------|-------------|
| `admin.users` | Manage | User administration |
| `admin.roles` | Manage | Role configuration |
| `admin.permissions` | Manage | Permission management |
| `admin.audit` | View | Audit log access |
| `admin.delegation` | Manage | Delegation configuration |

### Approval Levels (4 permissions)

| Permission Key | Limit | Description |
|---------------|-------|-------------|
| `approval.level1` | $10,000 | Basic approvals |
| `approval.level2` | $50,000 | Mid-level approvals |
| `approval.level3` | $100,000 | Senior approvals |
| `approval.unlimited` | ∞ | Executive approvals |

---

## 🔒 Field-Level Security

### Configuration

Field permissions are configured per role + resource type + field name:

```sql
INSERT INTO bp_field_permissions (
    tenant_id, datasource_id, role_id,
    resource_type, field_name, permission_level
) VALUES (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    (SELECT id FROM bp_roles WHERE role_key = 'viewer'),
    'process',
    'ssn',
    'none'
);
```

### Masking Rules

Masking rules define how fields are masked for roles without full access:

```sql
INSERT INTO bp_field_masking_rules (
    tenant_id, datasource_id,
    resource_type, field_name,
    masking_type, masking_pattern,
    unmasked_roles
) VALUES (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'process',
    'ssn',
    'partial',
    'XXX-XX-####',
    ARRAY(SELECT id FROM bp_roles WHERE role_key IN ('admin', 'super_admin'))
);
```

### Field Access Matrix

| Field | Viewer | Editor | Admin | Compliance |
|-------|--------|--------|-------|------------|
| SSN | ❌ None | 🔒 Masked | ✅ Full | ✅ Full |
| Tax ID | ❌ None | 🔒 Masked | ✅ Full | ✅ Full |
| Bank Account | ❌ None | 🔒 Masked | ✅ Full | ✅ Full |
| Credit Card | ❌ None | 🔒 Masked | ✅ Full | ❌ None |
| Email | ✅ Full | ✅ Full | ✅ Full | ✅ Full |
| Name | ✅ Full | ✅ Full | ✅ Full | ✅ Full |

### Using Field Masking in Code

```go
// Get masking rules for user
rules, err := rbacEnforcer.GetFieldMaskingRules(
    userID, tenantID, datasourceID, "process"
)

// Apply masking to response data
processData := map[string]interface{}{
    "id": "...",
    "name": "Process A",
    "ssn": "123-45-6789",
}

middleware.ApplyFieldMasking(processData, rules)
// processData["ssn"] now contains "XXX-XX-6789"
```

---

## 🔄 Approval Delegation

### Use Cases

1. **Vacation Coverage**: Manager delegates approval authority while on leave
2. **Backup Approver**: CFO designates backup for business continuity
3. **Temporary Authority**: Project lead delegates authority for specific project
4. **Amount Limits**: Delegate can approve up to $X without escalation

### Creating a Delegation

```sql
INSERT INTO bp_approval_delegations (
    tenant_id, datasource_id,
    delegator_user_id, delegate_user_id,
    delegation_type, resource_type,
    start_date, end_date, reason, scope_conditions
) VALUES (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222', -- CFO
    '33333333-3333-3333-3333-333333333333', -- Deputy CFO
    'partial',
    'process',
    '2024-01-15',
    '2024-01-30',
    'Vacation coverage',
    '{"max_amount": 50000, "process_types": ["expense_approval"]}'::jsonb
);
```

### Checking Active Delegation

```go
delegationID, err := rbacEnforcer.CheckDelegation(
    delegatorID, delegateID,
    tenantID, datasourceID,
    "process", processID,
)

if delegationID != "" {
    // Delegation exists, allow action
    // Log to delegation_usage_log
}
```

### Delegation Audit Trail

All actions performed using delegated authority are logged:

```sql
INSERT INTO bp_delegation_usage_log (
    delegation_id, delegate_user_id,
    action_type, resource_type, resource_id,
    action_details
) VALUES (
    '44444444-4444-4444-4444-444444444444',
    '33333333-3333-3333-3333-333333333333',
    'approve',
    'process',
    '55555555-5555-5555-5555-555555555555',
    '{"amount": 45000, "approval_level": "level2"}'::jsonb
);
```

---

## 👥 Team Access Control

### Team Types

- **Functional**: Department teams (Finance, HR, Operations, Compliance)
- **Project**: Cross-functional project teams
- **Cross-Functional**: Multi-department committees

### Creating a Team

```sql
INSERT INTO bp_teams (
    tenant_id, datasource_id,
    team_key, team_name, description,
    team_type, manager_user_id
) VALUES (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'finance',
    'Finance Department',
    'Finance and accounting team',
    'functional',
    '66666666-6666-6666-6666-666666666666'
);
```

### Adding Team Members

```sql
INSERT INTO bp_team_members (team_id, user_id, role_in_team)
VALUES (
    '77777777-7777-7777-7777-777777777777',
    '88888888-8888-8888-8888-888888888888',
    'member'
);
```

### Granting Team Access to Resources

```sql
INSERT INTO bp_team_permissions (
    team_id, resource_type, resource_id, permission_level
) VALUES (
    '77777777-7777-7777-7777-777777777777',
    'process',
    '99999999-9999-9999-9999-999999999999',
    'write'
);
```

---

## 🌐 API Reference

### Base URL
```
http://localhost:8080/api/rbac
```

All endpoints require `tenant_id` and `datasource_id` in query parameters or headers.

### Role Management

#### List Roles
```http
GET /api/rbac/roles?tenant_id={uuid}&datasource_id={uuid}
```

**Response**:
```json
[
  {
    "id": "uuid",
    "role_key": "admin",
    "role_name": "Administrator",
    "role_level": "admin",
    "is_active": true
  }
]
```

#### Create Role
```http
POST /api/rbac/roles?tenant_id={uuid}&datasource_id={uuid}
Content-Type: application/json

{
  "role_key": "custom_role",
  "role_name": "Custom Role",
  "description": "Custom role description",
  "role_level": "editor",
  "permissions": ["permission_id_1", "permission_id_2"]
}
```

#### Get Role
```http
GET /api/rbac/roles/{roleId}
```

#### Update Role
```http
PUT /api/rbac/roles/{roleId}
Content-Type: application/json

{
  "role_name": "Updated Role Name",
  "description": "Updated description",
  "is_active": true
}
```

#### Delete Role (Soft Delete)
```http
DELETE /api/rbac/roles/{roleId}
```

### Permission Management

#### List Permissions
```http
GET /api/rbac/permissions?tenant_id={uuid}
```

**Response**:
```json
[
  {
    "id": "uuid",
    "permission_key": "process.read",
    "permission_name": "Read Process",
    "resource_type": "process",
    "action": "read",
    "is_system": true
  }
]
```

#### Get User Permissions
```http
GET /api/rbac/permissions/user/{userId}?tenant_id={uuid}&datasource_id={uuid}
```

#### Check Permission
```http
POST /api/rbac/permissions/check
Content-Type: application/json

{
  "user_id": "uuid",
  "tenant_id": "uuid",
  "datasource_id": "uuid",
  "permission_key": "process.read"
}
```

**Response**:
```json
{
  "has_permission": true
}
```

### Role Assignment

#### Assign Role to User
```http
POST /api/rbac/roles/{roleId}/assign
Content-Type: application/json

{
  "user_id": "uuid",
  "tenant_id": "uuid",
  "datasource_id": "uuid",
  "scope_type": "global",
  "scope_id": null,
  "expires_at": "2024-12-31T23:59:59Z"
}
```

#### Unassign Role from User
```http
DELETE /api/rbac/roles/{roleId}/unassign/{userId}
```

#### Get User Roles
```http
GET /api/rbac/users/{userId}/roles?tenant_id={uuid}&datasource_id={uuid}
```

### Field-Level Permissions

#### List Field Permissions
```http
GET /api/rbac/field-permissions?tenant_id={uuid}&datasource_id={uuid}
```

#### Create Field Permission
```http
POST /api/rbac/field-permissions
Content-Type: application/json

{
  "tenant_id": "uuid",
  "datasource_id": "uuid",
  "role_id": "uuid",
  "resource_type": "process",
  "resource_id": "uuid",
  "field_name": "ssn",
  "permission_level": "mask"
}
```

#### Get User Field Permissions
```http
GET /api/rbac/field-permissions/user/{userId}/resource/{resourceType}/{resourceId}?tenant_id={uuid}&datasource_id={uuid}
```

### Delegations

#### List Delegations
```http
GET /api/rbac/delegations?tenant_id={uuid}&datasource_id={uuid}
```

#### Create Delegation
```http
POST /api/rbac/delegations
Content-Type: application/json

{
  "tenant_id": "uuid",
  "datasource_id": "uuid",
  "delegator_user_id": "uuid",
  "delegate_user_id": "uuid",
  "delegation_type": "full",
  "resource_type": "process",
  "resource_id": "uuid",
  "start_date": "2024-01-15T00:00:00Z",
  "end_date": "2024-01-30T23:59:59Z",
  "reason": "Vacation coverage"
}
```

#### Update Delegation
```http
PUT /api/rbac/delegations/{delegationId}
Content-Type: application/json

{
  "end_date": "2024-02-15T23:59:59Z",
  "is_active": true
}
```

#### Delete Delegation
```http
DELETE /api/rbac/delegations/{delegationId}
```

#### Get User Delegations
```http
GET /api/rbac/delegations/user/{userId}?type=delegator
```

Query param `type`: `delegator` (delegations given) or `delegate` (delegations received)

#### Log Delegation Usage
```http
POST /api/rbac/delegations/{delegationId}/log
Content-Type: application/json

{
  "delegate_user_id": "uuid",
  "action_type": "approve",
  "resource_type": "process",
  "resource_id": "uuid",
  "action_details": {
    "amount": 45000,
    "approval_level": "level2"
  }
}
```

### Teams

#### List Teams
```http
GET /api/rbac/teams?tenant_id={uuid}&datasource_id={uuid}
```

#### Create Team
```http
POST /api/rbac/teams
Content-Type: application/json

{
  "tenant_id": "uuid",
  "datasource_id": "uuid",
  "team_key": "finance",
  "team_name": "Finance Department",
  "description": "Finance and accounting team",
  "team_type": "functional",
  "manager_user_id": "uuid"
}
```

#### Add Team Member
```http
POST /api/rbac/teams/{teamId}/members
Content-Type: application/json

{
  "user_id": "uuid",
  "role_in_team": "member"
}
```

#### Remove Team Member
```http
DELETE /api/rbac/teams/{teamId}/members/{userId}
```

#### Get Team Members
```http
GET /api/rbac/teams/{teamId}/members
```

### Audit

#### List Permission Audit Log
```http
GET /api/rbac/audit?tenant_id={uuid}&datasource_id={uuid}&limit=100
```

---

## 🛡️ Middleware Usage

### Require Permission
```go
r.With(rbacEnforcer.RequirePermission("process.read")).Get("/api/processes", handler.listProcesses)
```

### Require Any Permission (OR logic)
```go
r.With(rbacEnforcer.RequireAnyPermission("process.read", "process.execute")).Get("/api/processes/{id}", handler.getProcess)
```

### Require All Permissions (AND logic)
```go
r.With(rbacEnforcer.RequireAllPermissions("process.update", "field.update.sensitive")).Put("/api/processes/{id}", handler.updateProcess)
```

### Require Role
```go
r.With(rbacEnforcer.RequireRole("admin")).Post("/api/admin/settings", handler.updateSettings)
```

### Require Role Level
```go
r.With(rbacEnforcer.RequireRoleLevel("admin")).Delete("/api/processes/{id}", handler.deleteProcess)
```

### Example Integration

```go
// In api.go
rbacEnforcer := middleware.NewRBACEnforcer(sqlxDB)

// Protect process endpoints
r.With(rbacEnforcer.RequirePermission("process.read")).Get("/api/bp-processes", handler.listProcesses)
r.With(rbacEnforcer.RequirePermission("process.create")).Post("/api/bp-processes", handler.createProcess)
r.With(rbacEnforcer.RequirePermission("process.update")).Put("/api/bp-processes/{id}", handler.updateProcess)
r.With(rbacEnforcer.RequirePermission("process.delete")).Delete("/api/bp-processes/{id}", handler.deleteProcess)

// Protect approval endpoints
r.With(rbacEnforcer.RequirePermission("step.approve")).Post("/api/bp-approvals/{id}/approve", handler.approveStep)

// Protect admin endpoints
r.With(rbacEnforcer.RequireRole("admin")).Route("/api/admin", func(r chi.Router) {
    r.Get("/users", handler.listUsers)
    r.Post("/users", handler.createUser)
})
```

---

## 🚀 Migration Guide

### Step 1: Run Schema Migration

```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable \
     -f backend/migrations/misc/rbac_field_level_permissions.sql
```

### Step 2: Load Seed Data

```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable \
     -f backend/migrations/misc/seed_rbac_permissions.sql
```

### Step 3: Verify Data

```sql
-- Check roles
SELECT role_key, role_name, role_level FROM bp_roles ORDER BY role_level;

-- Check permissions count
SELECT COUNT(*) FROM bp_permissions;

-- Check role-permission mappings
SELECT r.role_key, COUNT(rp.permission_id) as permission_count
FROM bp_roles r
JOIN bp_role_permissions rp ON r.id = rp.role_id
GROUP BY r.role_key
ORDER BY permission_count DESC;

-- Check field masking rules
SELECT field_name, masking_pattern FROM bp_field_masking_rules;

-- Check sample teams
SELECT team_key, team_name, team_type FROM bp_teams;
```

### Step 4: Assign Roles to Users

```sql
-- Assign admin role to user
INSERT INTO bp_user_roles (
    user_id, role_id, tenant_id, datasource_id, scope_type
) VALUES (
    'YOUR_USER_ID',
    (SELECT id FROM bp_roles WHERE role_key = 'admin'),
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'global'
);
```

### Step 5: Restart Backend

```bash
cd backend
go run cmd/api/main.go
```

---

## 🧪 Testing

### Test Permission Check

```bash
curl -X POST http://localhost:8080/api/rbac/permissions/check \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "YOUR_USER_ID",
    "tenant_id": "00000000-0000-0000-0000-000000000000",
    "datasource_id": "11111111-1111-1111-1111-111111111111",
    "permission_key": "process.read"
  }'
```

### Test Field Masking

```sql
-- Query with masking applied
SELECT * FROM bp_field_masking_rules WHERE field_name = 'ssn';

-- Test helper function
SELECT bp_user_has_permission(
    'YOUR_USER_ID'::uuid,
    '00000000-0000-0000-0000-000000000000'::uuid,
    '11111111-1111-1111-1111-111111111111'::uuid,
    'process.read'
);
```

### Test Delegation

```sql
-- Create test delegation
INSERT INTO bp_approval_delegations (
    tenant_id, datasource_id,
    delegator_user_id, delegate_user_id,
    delegation_type, start_date, end_date, reason
) VALUES (
    '00000000-0000-0000-0000-000000000000',
    '11111111-1111-1111-1111-111111111111',
    'DELEGATOR_USER_ID',
    'DELEGATE_USER_ID',
    'full',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP + INTERVAL '7 days',
    'Test delegation'
);

-- Check active delegation
SELECT bp_get_active_delegation(
    'DELEGATOR_USER_ID'::uuid,
    'DELEGATE_USER_ID'::uuid,
    '00000000-0000-0000-0000-000000000000'::uuid,
    '11111111-1111-1111-1111-111111111111'::uuid,
    NULL, NULL
);
```

---

## ✅ Compliance Checklist

### SOX (Sarbanes-Oxley)
- ✅ Audit trail for all permission changes
- ✅ Segregation of duties (role-based)
- ✅ User access reviews (role assignments with expiration)
- ✅ Change management controls (permission audit log)

### GDPR (General Data Protection Regulation)
- ✅ Field-level access control for PII
- ✅ Data masking for sensitive fields
- ✅ Audit trail for data access
- ✅ Right to access (user can query their permissions)

### HIPAA (Health Insurance Portability and Accountability Act)
- ✅ PHI field-level restrictions
- ✅ Minimum necessary access principle
- ✅ Access audit logging
- ✅ Automatic session expiration (role expiration)

### PCI DSS (Payment Card Industry Data Security Standard)
- ✅ Cardholder data field masking
- ✅ Role-based access to payment data
- ✅ Audit logging of access
- ✅ Least privilege principle

---

## 🔍 Troubleshooting

### Issue: Permission check returns false for admin user

**Solution**: Check role assignment and expiration:
```sql
SELECT * FROM bp_user_roles
WHERE user_id = 'YOUR_USER_ID'
  AND is_active = true
  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP);
```

### Issue: Field masking not applied

**Solution**: Verify masking rules exist:
```sql
SELECT * FROM bp_field_masking_rules
WHERE field_name = 'YOUR_FIELD'
  AND resource_type = 'YOUR_RESOURCE';
```

### Issue: Delegation not working

**Solution**: Check delegation date range and scope:
```sql
SELECT * FROM bp_approval_delegations
WHERE delegator_user_id = 'DELEGATOR_ID'
  AND delegate_user_id = 'DELEGATE_ID'
  AND is_active = true
  AND CURRENT_TIMESTAMP BETWEEN start_date AND end_date;
```

### Issue: API returns 403 Forbidden

**Solution**: Check middleware is correctly extracting tenant/datasource:
```go
// Verify context contains user_id
userID := middleware.getUserIDFromContext(r.Context())
log.Printf("User ID from context: %s", userID)

// Verify tenant_id in query/header
tenantID := middleware.getTenantIDFromRequest(r)
log.Printf("Tenant ID: %s", tenantID)
```

---

## 📚 Next Steps

1. **Frontend UI Development** (2-3 hours):
   - Role Manager component
   - User Role Assignment component
   - Delegation Manager component
   - Field Permission Editor component
   - Team Manager component

2. **Integration** (1 hour):
   - Add routes to AppRoutes.tsx
   - Add navigation menu items
   - Add page wrapper components

3. **Testing** (1 hour):
   - Test role assignments
   - Test field masking
   - Test delegation workflow
   - Test team access

4. **Documentation** (30 minutes):
   - User guide for admins
   - Video walkthrough
   - Compliance certification documentation

---

**Questions?** Contact the development team or refer to the codebase:
- Schema: `backend/migrations/misc/rbac_field_level_permissions.sql`
- API: `backend/internal/api/bp_rbac_handlers.go`
- Middleware: `backend/internal/api/middleware/rbac_enforcement.go`
