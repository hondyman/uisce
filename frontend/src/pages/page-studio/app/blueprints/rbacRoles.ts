import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import type { Action, ColumnDef, ConditionNode, FormFieldSpec } from '../appModel';

/**
 * Roles (slug rbac-roles), served at /admin/rbac/roles via STUDIO_ROUTES. Lists the
 * roles a tenant sees (its own and the shared gold-copy roles), creates and edits
 * roles in a dialog, shows a role's users and field permissions, and removes a role
 * (a soft delete). Gold-copy roles are read-only here. The key and level are fixed
 * once a role exists: the API does not change them on update.
 */

const op = (
  operation: string,
  params: Record<string, unknown>,
  onSuccess: Action[] = [],
  more: Record<string, unknown> = {},
): Action => ({ kind: 'runOperation', operation, params, onSuccess, ...more } as Action);
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const cond = (field: string, operator: string, value?: unknown): ConditionNode =>
  ({ type: 'condition', field, operator, value });
const col = (id: string, header: string, cell: ColumnDef['cell'] | ColumnDef['stack'], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell: cell as ColumnDef['cell'], ...more };

const fit = { flex: '0 0 auto' };

type NodeSpec = {
  type: 'Row' | 'Column' | 'Dialog';
  children: string[];
  style?: Record<string, string>;
  props?: Record<string, unknown>;
};
const layout = (root: string, spec: Record<string, NodeSpec>): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

const LEVEL_OPTIONS = [
  { value: 'viewer', label: 'Viewer' },
  { value: 'editor', label: 'Editor' },
  { value: 'approver', label: 'Approver' },
  { value: 'admin', label: 'Admin' },
  { value: 'super_admin', label: 'Super Admin' },
];

const EDITOR_OPEN = cond('vars.roleOpen', 'is_true');
const VIEW_OPEN = cond('vars.roleViewOpen', 'is_true');
const closeEditor = [set('roleOpen', false), set('roleDraft')];
const closeView = [set('roleViewOpen', false), set('roleViewId')];
const EXISTING = cond('vars.roleDraft.id', 'is_not_empty');

function rolesPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const w = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => {
    components[id] = { id, type, props, ...extra };
  };

  w(
    'hdr',
    'PageHeader',
    {
      icon: 'gavel',
      title: 'Roles',
      subtitle: 'Roles for this tenant. Gold-copy roles are shared and read-only here.',
    },
    { style: { flex: '1 1 320px' } },
  );

  w(
    'new_btn',
    'ActionButton',
    {
      label: 'New role',
      icon: 'add',
      variant: 'contained',
      onClick: [set('roleEditing', ''), set('roleOpen', true)],
    },
    { style: fit },
  );

  w('q', 'SearchInput', { variable: 'roleSearch', placeholder: 'Search roles', maxWidth: 520 }, { style: { flex: '1 1 260px' } });
  w(
    'level',
    'VariableSelect',
    {
      variable: 'roleLevel',
      label: '',
      emptyLabel: 'All levels',
      minWidth: 170,
      options: LEVEL_OPTIONS,
    },
    { style: fit },
  );

  w('grid', 'DataGrid', {
    query: 'roles',
    rowsPath: 'rows',
    emptyText: 'No roles match. Use "New role" to add one.',
    columns: [
      col('name', 'Role', [
        { kind: 'text', value: '{{row.role_name}}', bold: true },
        { kind: 'text', value: '{{row.role_key}}', caption: true },
      ]),
      col('level', 'Level', { kind: 'chip', value: '{{row.level_label}}', variant: 'outlined' }),
      col('origin', 'Origin', { kind: 'chip', value: '{{row.origin_label}}', variant: 'outlined' }),
      col('description', 'Description', { kind: 'text', value: '{{row.description}}' }),
      col(
        'act',
        '',
        {
          kind: 'actions',
          buttons: [
            {
              label: 'View',
              icon: 'preview',
              onClick: [set('roleViewId', '{{row.id}}'), set('roleViewOpen', true)],
            },
            {
              label: 'Edit',
              icon: 'edit',
              visibleWhen: cond('row.editable', 'is_true'),
              onClick: [set('roleEditing', '{{row.id}}'), set('roleOpen', true)],
            },
            {
              label: 'Remove',
              icon: 'delete',
              visibleWhen: cond('row.editable', 'is_true'),
              onClick: [
                op('rbac.removeRole', { id: '{{row.id}}' }, [], {
                  confirm: {
                    title: 'Remove this role?',
                    text: '{{row.role_name}} will stop being listed and can no longer be assigned.',
                    confirmLabel: 'Remove',
                  },
                  successMessage: 'Role removed',
                }),
              ],
            },
          ],
        },
        { align: 'right', nowrap: true },
      ),
    ],
  });

  const roleFields: FormFieldSpec[] = [
    {
      name: 'role_key',
      kind: 'text',
      label: 'Role key',
      required: true,
      wide: true,
      readOnlyWhen: EXISTING,
      helperText: 'Cannot be changed once the role exists.',
    },
    { name: 'role_name', kind: 'text', label: 'Role name', required: true, wide: true },
    { name: 'description', kind: 'multiline', label: 'Description', wide: true },
    {
      name: 'role_level',
      kind: 'select',
      label: 'Level',
      required: true,
      readOnlyWhen: EXISTING,
      helperText: 'Set when the role is created.',
      options: LEVEL_OPTIONS,
    },
  ];
  w('role_form', 'Form', {
    variable: 'roleDraft',
    fields: roleFields,
    initFrom: '{{queries.roleStart.data.draft}}',
    seedKey: '{{queries.roleStart.data.key}}',
  });

  // The details dialog reads the same draft, but every field is read-only.
  const viewFields: FormFieldSpec[] = [
    { name: 'role_key', kind: 'text', label: 'Role key', wide: true, readOnlyWhen: cond('vars.roleViewOpen', 'is_true') },
    { name: 'role_name', kind: 'text', label: 'Role name', wide: true, readOnlyWhen: cond('vars.roleViewOpen', 'is_true') },
    { name: 'description', kind: 'multiline', label: 'Description', wide: true, readOnlyWhen: cond('vars.roleViewOpen', 'is_true') },
    {
      name: 'role_level',
      kind: 'select',
      label: 'Level',
      options: LEVEL_OPTIONS,
      readOnlyWhen: cond('vars.roleViewOpen', 'is_true'),
    },
  ];
  w('view_form', 'Form', {
    variable: 'roleViewDraft',
    fields: viewFields,
    initFrom: '{{queries.roleView.data.draft}}',
    seedKey: '{{queries.roleView.data.key}}',
  });

  w('view_users', 'DataGrid', {
    query: 'roleUsers',
    rowsPath: 'rows',
    emptyText: 'No users assigned to this role.',
    columns: [
      col('user', 'User', [
        { kind: 'text', value: '{{row.name}}', bold: true },
        { kind: 'text', value: '{{row.email}}', caption: true },
      ]),
      col('assigned', 'Assigned', { kind: 'text', value: '{{row.assigned_text}}' }, { nowrap: true }),
    ],
  });

  w('view_perms', 'DataGrid', {
    query: 'roleFieldPerms',
    rowsPath: 'rows',
    emptyText: 'No field permissions configured.',
    columns: [
      col('field', 'Field', { kind: 'text', value: '{{row.field}}', bold: true }),
      col('resource', 'Resource', { kind: 'text', value: '{{row.resource_type}}' }),
      col('level', 'Permission', {
        kind: 'chip',
        value: '{{row.permission_level}}',
        colorMap: { write: 'success', read: 'info', '*': 'warning' },
        variant: 'outlined',
      }),
    ],
  });

  const main = layout('page_root', {
    page_root: {
      type: 'Column',
      children: ['top', 'filters', 'grid', 'role_dialog', 'view_dialog'],
      style: { gap: '16px' },
    },
    top: {
      type: 'Row',
      children: ['hdr', 'new_btn'],
      style: { alignItems: 'center', gap: '12px', flexWrap: 'wrap' },
    },
    filters: {
      type: 'Row',
      children: ['q', 'level'],
      style: { alignItems: 'center', gap: '12px', flexWrap: 'wrap' },
    },
    role_dialog: {
      type: 'Dialog',
      children: ['role_form'],
      props: {
        title: '{{queries.roleStart.data.title}}',
        maxWidth: 'sm',
        openWhen: EDITOR_OPEN,
        onClose: closeEditor,
        buttons: [
          { label: 'Cancel', onClick: closeEditor },
          {
            label: 'Save',
            variant: 'contained',
            disabledWhen: cond('vars.roleDraft.role_name', 'is_empty'),
            onClick: [op('rbac.saveRole', { draft: '{{vars.roleDraft}}' }, closeEditor, { successMessage: 'Role saved' })],
          },
        ],
      },
    },
    view_dialog: {
      type: 'Dialog',
      children: ['view_form', 'view_users', 'view_perms'],
      props: {
        title: 'Role details',
        maxWidth: 'md',
        openWhen: VIEW_OPEN,
        onClose: closeView,
        buttons: [
          { label: 'Close', onClick: closeView },
          { label: 'Assign users', onClick: [{ kind: 'navigate', to: '/admin/rbac/user-roles' }] },
        ],
      },
    },
  });

  return {
    name: 'Roles',
    slug: 'rbac-roles',
    description:
      'Roles for this tenant: create and edit roles, see who holds them and what field permissions they grant. Built in Page Studio.',
    version: 1,
    isCore: true,
    status: 'published',
    layout: main,
    tabs: [],
    filterBar: layout('fb_root', { fb_root: { type: 'Column', children: [], style: { gap: '0px' } } }),
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 1200, padding: 3 },
      variables: [
        { name: 'roleSearch', default: '', description: 'Role name or key filter' },
        { name: 'roleLevel', default: '', description: 'Role level filter' },
        { name: 'roleOpen', default: false, description: 'Whether the role editor is open' },
        { name: 'roleEditing', default: '', description: 'The role being edited; empty = a new one' },
        { name: 'roleDraft', description: 'The role being edited' },
        { name: 'roleViewOpen', default: false, description: 'Whether the role details are open' },
        { name: 'roleViewId', default: '', description: 'The role shown in the details' },
        { name: 'roleViewDraft', description: 'The role shown in the details (read-only)' },
      ],
      queries: [
        {
          id: 'roles',
          operation: 'rbac.listRoles',
          params: { q: '{{vars.roleSearch}}', level: '{{vars.roleLevel}}' },
          keepPrevious: true,
        },
        {
          id: 'roleStart',
          operation: 'rbac.roleStart',
          params: { id: '{{vars.roleEditing}}', open: '{{vars.roleOpen}}' },
          enabledWhen: EDITOR_OPEN,
        },
        {
          id: 'roleView',
          operation: 'rbac.roleStart',
          params: { id: '{{vars.roleViewId}}', open: '{{vars.roleViewOpen}}' },
          enabledWhen: VIEW_OPEN,
        },
        {
          id: 'roleUsers',
          operation: 'rbac.roleUsers',
          params: { roleId: '{{vars.roleViewId}}' },
          enabledWhen: VIEW_OPEN,
        },
        {
          id: 'roleFieldPerms',
          operation: 'rbac.roleFieldPermissions',
          params: { roleId: '{{vars.roleViewId}}' },
          enabledWhen: VIEW_OPEN,
        },
      ],
    },
  };
}

export const rbacRolesBlueprint = () => structuredClone(rolesPage());
