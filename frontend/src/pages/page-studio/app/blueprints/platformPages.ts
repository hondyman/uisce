import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import '../../../../features/admin/studio';

type Spec = Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>;
const layout = (root: string, spec: Spec): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

function makeSingleDCBlueprint(opts: {
  name: string;
  slug: string;
  description: string;
  icon: string;
  dcId: string;
  title: string;
  subtitle: string;
}): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {
    header: {
      id: 'header',
      type: 'PageHeader',
      props: {
        icon: opts.icon,
        title: opts.title,
        subtitle: opts.subtitle,
      },
    },
    dc: {
      id: 'dc',
      type: 'DomainComponent',
      props: {
        component: opts.dcId,
        inputs: {},
      },
    },
  };

  const l = layout('root', {
    root: { type: 'Column', children: ['dc'], style: { gap: '16px' } },
  });

  const fb = layout('fb_root', {
    fb_root: { type: 'Row', children: ['header'], style: { alignItems: 'center', width: '100%' } },
  });

  return {
    name: opts.name,
    slug: opts.slug,
    description: opts.description,
    version: 1,
    isCore: true,
    status: 'published',
    layout: l,
    filterBar: fb,
    tabs: [],
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none',
      surface: { maxWidth: 1600, padding: 3 },
      variables: [],
      queries: [],
    },
  };
}

export function teamsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Teams',
    slug: 'platform-teams',
    description: 'Manage organizational teams, membership, and resource assignments',
    icon: 'groups',
    dcId: 'admin.TeamManager',
    title: 'Teams',
    subtitle: 'Organizational structures, team leadership, and member rosters',
  });
}

export function usersBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Users',
    slug: 'platform-users',
    description: 'User accounts, profile management, and credentials',
    icon: 'person',
    dcId: 'admin.UserManager',
    title: 'Users',
    subtitle: 'User directory, status management, and credential provisioning',
  });
}

export function userRolesBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'User Roles',
    slug: 'platform-user-roles',
    description: 'Assign and manage RBAC role bindings for users',
    icon: 'assignmentInd',
    dcId: 'admin.UserRoleAssignment',
    title: 'User Roles',
    subtitle: 'RBAC role assignments and access scope mapping',
  });
}

export function userTenantsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'User Tenants',
    slug: 'platform-user-tenants',
    description: 'Multi-tenant memberships and tenant access assignment',
    icon: 'domain',
    dcId: 'admin.TenantUserAssignment',
    title: 'User Tenants',
    subtitle: 'Cross-tenant user access authorizations',
  });
}

export function delegationsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Delegations',
    slug: 'platform-delegations',
    description: 'Time-bound and approval-scoped user access delegations',
    icon: 'swapHoriz',
    dcId: 'admin.DelegationManager',
    title: 'Delegations',
    subtitle: 'Temporary access delegations and out-of-office coverage',
  });
}

export function fieldPermissionsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Field Permissions',
    slug: 'platform-field-permissions',
    description: 'Fine-grained attribute and column-level permission matrix',
    icon: 'lock',
    dcId: 'admin.FieldPermissionEditor',
    title: 'Field Permissions',
    subtitle: 'Attribute masking, column visibility, and edit restrictions',
  });
}

export function rolesBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Roles & Permissions',
    slug: 'platform-roles',
    description: 'Define RBAC roles, permission sets, and capability grants',
    icon: 'adminPanelSettings',
    dcId: 'admin.RoleManager',
    title: 'Roles & Permissions',
    subtitle: 'System roles, functional permissions, and entitlement policies',
  });
}

export function ipWhitelistBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'IP Whitelist',
    slug: 'platform-ip-whitelist',
    description: 'Configure network IP allowlists and CIDR ranges per tenant',
    icon: 'security',
    dcId: 'admin.IPWhitelist',
    title: 'IP Whitelist',
    subtitle: 'Network access controls and CIDR boundary rules',
  });
}

export function messageCatalogBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'Message Catalog',
    slug: 'platform-message-catalog',
    description: 'Unified error code, localization, and message catalog editor',
    icon: 'translate',
    dcId: 'admin.MessageCatalog',
    title: 'Message Catalog',
    subtitle: 'Governed message keys, multi-language translations, and approval changesets',
  });
}

export function seedingBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return makeSingleDCBlueprint({
    name: 'System Seeding',
    slug: 'platform-seeding',
    description: 'Seed catalog metadata, demo business objects, and rule fixtures',
    icon: 'bolt',
    dcId: 'admin.Seeding',
    title: 'System Seeding',
    subtitle: 'Environment provisioning, fixture deployment, and catalog initializers',
  });
}
