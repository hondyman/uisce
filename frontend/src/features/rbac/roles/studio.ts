import { registerOperations, type OperationDef } from '../../../studio-core/operations/registry';
import apiClient from '../../../utils/apiClient';
import {
  createRoleBody,
  fieldPermissionRow,
  matchesRole,
  toRoleRow,
  updateRoleBody,
  userRow,
  type RawRole,
  type RoleRow,
} from './logic';

/**
 * Role manager Page Studio operations (the rbac-roles page). Roles are the tenant's
 * own plus the shared gold-copy roles; the API decides what a caller may change.
 */

const DOMAIN = 'rbac';

async function loadRoles(): Promise<RoleRow[]> {
  const raw = await apiClient<RawRole[]>('/api/rbac/roles');
  return (Array.isArray(raw) ? raw : []).map(toRoleRow);
}

const operations: OperationDef[] = [
  {
    id: 'rbac.listRoles',
    domain: DOMAIN,
    kind: 'query',
    label: 'List roles',
    description: 'Roles visible to this tenant, filtered by name or key and by level.',
    params: [
      { name: 'q', type: 'string', required: false, description: 'Name or key contains this text.' },
      { name: 'level', type: 'string', required: false, description: 'Only roles at this level.' },
    ],
    rowsPath: 'rows',
    rowFields: [
      { name: 'id', type: 'string' },
      { name: 'role_key', type: 'string' },
      { name: 'role_name', type: 'string' },
      { name: 'description', type: 'string' },
      { name: 'role_level', type: 'string' },
      { name: 'level_label', type: 'string' },
      { name: 'origin_label', type: 'string' },
      { name: 'editable', type: 'boolean' },
    ],
    fields: [{ name: 'rows' }, { name: 'total', type: 'number' }],
    run: async (params) => {
      const q = String(params.q ?? '');
      const level = String(params.level ?? '');
      const rows = (await loadRoles()).filter((r) => matchesRole(r, q, level));
      return { rows, total: rows.length };
    },
  },
  {
    id: 'rbac.roleStart',
    domain: DOMAIN,
    kind: 'query',
    label: 'Start the role editor',
    description: 'The title and draft for the role editor or details: a blank draft for a new role, the stored values for an existing one.',
    params: [
      { name: 'id', type: 'string', required: false, description: 'The role to edit or show; empty for a new one.' },
      { name: 'open', type: 'boolean', required: false, description: 'Re-runs the query each time the dialog opens.' },
    ],
    fields: [{ name: 'title' }, { name: 'key' }, { name: 'draft', type: 'object' }],
    run: async (params) => {
      const id = params.id ? String(params.id) : '';
      // The key changes on every run, so the form re-seeds each time the dialog opens.
      if (!id) {
        return {
          title: 'New role',
          key: `new:${Date.now()}`,
          draft: { role_key: '', role_name: '', description: '', role_level: 'viewer' },
        };
      }
      const role = (await loadRoles()).find((r) => r.id === id);
      if (!role) throw new Error('This role no longer exists.');
      return {
        title: `Edit ${role.role_name}`,
        key: `${role.id}:${Date.now()}`,
        draft: {
          id: role.id,
          role_key: role.role_key,
          role_name: role.role_name,
          description: role.description,
          role_level: role.role_level,
        },
      };
    },
  },
  {
    id: 'rbac.saveRole',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Save role',
    description: 'Creates a role, or updates the name and description of an existing one.',
    params: [{ name: 'draft', type: 'object', required: true }],
    run: async (params) => {
      const d = (params.draft ?? {}) as Record<string, unknown>;
      const draft = {
        id: d.id ? String(d.id) : undefined,
        role_key: String(d.role_key ?? '').trim(),
        role_name: String(d.role_name ?? '').trim(),
        description: String(d.description ?? ''),
        role_level: String(d.role_level ?? 'viewer'),
      };
      if (!draft.role_name) throw new Error('A role needs a name.');
      if (draft.id) {
        await apiClient(`/api/rbac/roles/${draft.id}`, { method: 'PUT', body: JSON.stringify(updateRoleBody(draft)) });
      } else {
        if (!draft.role_key) throw new Error('A role needs a key.');
        await apiClient(`/api/rbac/roles`, { method: 'POST', body: JSON.stringify(createRoleBody(draft)) });
      }
      return { ok: true };
    },
  },
  {
    id: 'rbac.removeRole',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Remove role',
    description: 'Deactivates a role (a soft delete): it stops being listed and can no longer be assigned.',
    params: [{ name: 'id', type: 'string', required: true }],
    run: async (params) => {
      await apiClient(`/api/rbac/roles/${String(params.id)}`, { method: 'DELETE' });
      return { ok: true };
    },
  },
  {
    id: 'rbac.roleUsers',
    domain: DOMAIN,
    kind: 'query',
    label: 'Users in a role',
    description: 'The users holding a role in this tenant.',
    params: [{ name: 'roleId', type: 'string', required: true }],
    rowsPath: 'rows',
    rowFields: [
      { name: 'id', type: 'string' },
      { name: 'name', type: 'string' },
      { name: 'email', type: 'string' },
      { name: 'assigned_text', type: 'string' },
    ],
    fields: [{ name: 'rows' }, { name: 'total', type: 'number' }],
    run: async (params) => {
      const raw = await apiClient<Record<string, unknown>[]>(`/api/rbac/roles/${String(params.roleId)}/users`);
      const rows = (Array.isArray(raw) ? raw : []).map(userRow);
      return { rows, total: rows.length };
    },
  },
  {
    id: 'rbac.roleFieldPermissions',
    domain: DOMAIN,
    kind: 'query',
    label: 'Field permissions of a role',
    description: 'The field-level permissions granted to a role.',
    params: [{ name: 'roleId', type: 'string', required: true }],
    rowsPath: 'rows',
    rowFields: [
      { name: 'id', type: 'string' },
      { name: 'field', type: 'string' },
      { name: 'resource_type', type: 'string' },
      { name: 'permission_level', type: 'string' },
    ],
    fields: [{ name: 'rows' }, { name: 'total', type: 'number' }],
    run: async (params) => {
      const raw = await apiClient<Record<string, unknown>[]>(
        `/api/rbac/field-permissions?role_id=${encodeURIComponent(String(params.roleId))}`,
      );
      const rows = (Array.isArray(raw) ? raw : []).map(fieldPermissionRow);
      return { rows, total: rows.length };
    },
  },
];

registerOperations(operations);
