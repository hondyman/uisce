import React from 'react';
import { registerDomainComponents } from '../../studio-core/components/registry';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { useTenant } from '../../contexts/TenantContext';

import { TeamManager } from '../../components/RBAC/TeamManager';
import { UserManagerMasterDetail } from '../../components/RBAC/UserManager_MasterDetail';
import { UserRoleAssignmentStyled } from '../../components/RBAC/UserRoleAssignment_Styled';
import TenantUserAssignmentPage from './pages/TenantUserAssignmentPage';
import { DelegationManager } from '../../components/RBAC/DelegationManager';
import { FieldPermissionEditor } from '../../components/RBAC/FieldPermissionEditor';
import { RoleManagerStyled } from '../../components/RBAC/RoleManager_Styled';
import IPWhitelistManagementPage from '../fabric/pages/IPWhitelistManagementPage';
import MessageCatalogPage from '../message-catalog/MessageCatalogPage';
import SeedingPage from './pages/SeedingPage';

const DOMAIN = 'admin';

const operations: OperationDef[] = [
  {
    id: `${DOMAIN}.listAuditLog`,
    domain: DOMAIN,
    kind: 'query',
    label: 'List audit log entries',
    description: 'Lists system and tenant security audit logs.',
    params: [{ name: 'tenant_id', type: 'string' }],
    run: async () => ({ logs: [] }),
  },
];

registerOperations(operations);

function TeamManagerWrapper() {
  const { tenant, datasource } = useTenant();
  if (!tenant) return null;
  const effectiveDatasource = datasource || { id: 'none', source_name: 'Default' };
  return <TeamManager tenant={tenant} datasource={effectiveDatasource} />;
}

function UserManagerWrapper() {
  const { tenant, datasource } = useTenant();
  if (!tenant) return null;
  const effectiveDatasource = datasource || { id: 'none', source_name: 'Default' };
  return <UserManagerMasterDetail tenant={tenant as any} datasource={effectiveDatasource as any} />;
}

function UserRoleAssignmentWrapper() {
  const { tenant, datasource } = useTenant();
  if (!tenant) return null;
  const effectiveDatasource = datasource || { id: 'none', source_name: 'Default' };
  return <UserRoleAssignmentStyled tenant={tenant as any} datasource={effectiveDatasource as any} />;
}

function RoleManagerWrapper() {
  const { tenant, datasource } = useTenant();
  if (!tenant) return null;
  const effectiveDatasource = datasource || { id: 'none', source_name: 'Default' };
  return <RoleManagerStyled tenant={tenant as any} datasource={effectiveDatasource as any} />;
}

registerDomainComponents([
  {
    id: 'admin.TeamManager',
    domain: DOMAIN,
    label: 'Team Manager',
    description: 'Manage organizational teams, membership, and resource assignments.',
    inputs: [],
    events: [],
    render: () => <TeamManagerWrapper />,
  },
  {
    id: 'admin.UserManager',
    domain: DOMAIN,
    label: 'User Manager',
    description: 'Master-detail user management, profile configuration, and credentials.',
    inputs: [],
    events: [],
    render: () => <UserManagerWrapper />,
  },
  {
    id: 'admin.UserRoleAssignment',
    domain: DOMAIN,
    label: 'User Role Assignment',
    description: 'Assign and revoke system and application roles to users.',
    inputs: [],
    events: [],
    render: () => <UserRoleAssignmentWrapper />,
  },
  {
    id: 'admin.TenantUserAssignment',
    domain: DOMAIN,
    label: 'Tenant User Assignment',
    description: 'Multi-tenant membership and tenant switching authorization.',
    inputs: [],
    events: [],
    render: () => <TenantUserAssignmentPage />,
  },
  {
    id: 'admin.DelegationManager',
    domain: DOMAIN,
    label: 'Delegation Manager',
    description: 'Time-bound and approval-scoped user access delegations.',
    inputs: [],
    events: [],
    render: () => <DelegationManager />,
  },
  {
    id: 'admin.FieldPermissionEditor',
    domain: DOMAIN,
    label: 'Field Permission Editor',
    description: 'Fine-grained attribute and column-level permission matrix.',
    inputs: [],
    events: [],
    render: () => <FieldPermissionEditor />,
  },
  {
    id: 'admin.RoleManager',
    domain: DOMAIN,
    label: 'Role & Permission Manager',
    description: 'Define RBAC roles, permission sets, and capability grants.',
    inputs: [],
    events: [],
    render: () => <RoleManagerWrapper />,
  },
  {
    id: 'admin.IPWhitelist',
    domain: DOMAIN,
    label: 'IP Whitelist Manager',
    description: 'Configure network IP allowlists and CIDR ranges per tenant.',
    inputs: [],
    events: [],
    render: () => <IPWhitelistManagementPage />,
  },
  {
    id: 'admin.MessageCatalog',
    domain: DOMAIN,
    label: 'Message Catalog',
    description: 'Unified error code, localization, and message catalog editor.',
    inputs: [],
    events: [],
    render: () => <MessageCatalogPage />,
  },
  {
    id: 'admin.Seeding',
    domain: DOMAIN,
    label: 'System Seeding & Provisioning',
    description: 'Seed catalog metadata, demo business objects, and rule fixtures.',
    inputs: [],
    events: [],
    render: () => <SeedingPage />,
  },
]);
