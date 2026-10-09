import { describe, it, expect, vi, beforeEach } from 'vitest';
import apiClient from '../../utils/apiClient';
import { getOperation } from '../../studio-core/operations/registry';
import '../../features/rbac/roles/studio';
import {
  createRoleBody,
  fieldPermissionRow,
  matchesRole,
  toRoleRow,
  updateRoleBody,
  userRow,
  levelLabel,
} from '../../features/rbac/roles/logic';

vi.mock('../../utils/apiClient', () => ({ default: vi.fn() }));

const roles = [
  { id: 'r1', role_key: 'auditor', role_name: 'Auditor', description: 'Reads audit logs', role_level: 'viewer', is_active: true, origin: 'tenant' },
  { id: 'r2', role_key: 'controller', role_name: 'Controller', description: '', role_level: 'approver', is_active: true, origin: 'gold_copy', is_template: true },
  { id: 'r3', role_key: 'desk_lead', role_name: 'Desk lead', description: 'Runs the desk', role_level: 'super_admin', is_active: true, origin: 'extended' },
];

describe('role logic', () => {
  it('labels levels and derives row origin and editability', () => {
    expect(levelLabel('super_admin')).toBe('Super Admin');
    expect(toRoleRow(roles[1])).toMatchObject({ origin_label: 'Gold copy', editable: false, level_label: 'Approver' });
    expect(toRoleRow(roles[0])).toMatchObject({ origin_label: 'Tenant', editable: true });
    expect(toRoleRow({ id: 'x', role_key: 'k', role_name: 'K', role_level: 'viewer', is_template: true })).toMatchObject({ origin: 'gold_copy', editable: false });
  });

  it('matches on name or key and on level', () => {
    const rows = roles.map(toRoleRow);
    expect(rows.filter((r) => matchesRole(r, 'desk', ''))).toHaveLength(1);
    expect(rows.filter((r) => matchesRole(r, 'AUDIT', ''))).toHaveLength(1);
    expect(rows.filter((r) => matchesRole(r, '', 'approver'))).toHaveLength(1);
    expect(rows.filter((r) => matchesRole(r, 'zzz', ''))).toHaveLength(0);
  });

  it('update sends only name and description, so it never reactivates or changes the level', () => {
    const body = updateRoleBody({ id: 'r1', role_key: 'auditor', role_name: 'Auditor II', description: 'd', role_level: 'admin' });
    expect(body).toEqual({ role_name: 'Auditor II', description: 'd' });
    expect(body).not.toHaveProperty('is_active');
    expect(body).not.toHaveProperty('role_level');
  });

  it('create sends the key and level', () => {
    expect(createRoleBody({ role_key: 'k', role_name: 'K', description: '', role_level: 'editor' })).toMatchObject({ role_key: 'k', role_level: 'editor' });
  });

  it('maps users and field permissions to display rows', () => {
    expect(userRow({ id: 'u1', username: 'ann', name: '', email: 'a@x', assigned_at: '2026-03-01T10:00:00Z' })).toEqual({
      id: 'u1', name: 'ann', email: 'a@x', assigned_text: '2026-03-01',
    });
    expect(fieldPermissionRow({ id: 'p1', term_display_name: 'Account number', term_name: 'acct', term_node_id: 'n1', resource_type: 'bo', permission_level: 'read' })).toEqual({
      id: 'p1', field: 'Account number', resource_type: 'bo', permission_level: 'read',
    });
  });
});

describe('rbac.* Page Studio operations', () => {
  beforeEach(() => {
    vi.mocked(apiClient).mockReset();
  });

  it('lists roles with the filters applied', async () => {
    vi.mocked(apiClient).mockResolvedValueOnce(roles);
    const out = (await getOperation('rbac.listRoles')!.run({ q: 'desk' })) as { rows: { role_name: string }[]; total: number };
    expect(out.total).toBe(1);
    expect(out.rows[0].role_name).toBe('Desk lead');
  });

  it('starts a blank draft for a new role and the stored values for an existing one', async () => {
    const blank = (await getOperation('rbac.roleStart')!.run({ id: '' })) as { title: string; draft: { role_level: string } };
    expect(blank.title).toBe('New role');
    expect(blank.draft.role_level).toBe('viewer');

    vi.mocked(apiClient).mockResolvedValueOnce(roles);
    const edit = (await getOperation('rbac.roleStart')!.run({ id: 'r1' })) as { title: string; draft: Record<string, unknown> };
    expect(edit.title).toBe('Edit Auditor');
    expect(edit.draft).toMatchObject({ id: 'r1', role_key: 'auditor', role_level: 'viewer' });
  });

  it('saves an edit with a PUT that carries only the name and description', async () => {
    vi.mocked(apiClient).mockResolvedValueOnce({ status: 'updated' });
    await getOperation('rbac.saveRole')!.run({ draft: { id: 'r1', role_key: 'auditor', role_name: 'Auditor II', description: 'd', role_level: 'admin' } });
    const [url, init] = vi.mocked(apiClient).mock.calls[0];
    expect(url).toBe('/api/rbac/roles/r1');
    expect(init).toMatchObject({ method: 'PUT' });
    expect(JSON.parse(String((init as RequestInit).body))).toEqual({ role_name: 'Auditor II', description: 'd' });
  });

  it('creates with a POST that carries the key and level', async () => {
    vi.mocked(apiClient).mockResolvedValueOnce({ id: 'new', status: 'created' });
    await getOperation('rbac.saveRole')!.run({ draft: { role_key: 'ops', role_name: 'Ops', description: '', role_level: 'editor' } });
    const [url, init] = vi.mocked(apiClient).mock.calls[0];
    expect(url).toBe('/api/rbac/roles');
    expect(init).toMatchObject({ method: 'POST' });
    expect(JSON.parse(String((init as RequestInit).body))).toMatchObject({ role_key: 'ops', role_level: 'editor', permissions: [] });
  });

  it('refuses a role without a name, or a new role without a key, before any write', async () => {
    await expect(getOperation('rbac.saveRole')!.run({ draft: { role_key: 'x', role_name: ' ' } })).rejects.toThrow('needs a name');
    await expect(getOperation('rbac.saveRole')!.run({ draft: { role_key: '', role_name: 'X' } })).rejects.toThrow('needs a key');
    expect(apiClient).not.toHaveBeenCalled();
  });

  it('removes a role with a DELETE', async () => {
    vi.mocked(apiClient).mockResolvedValueOnce({ status: 'deleted' });
    await getOperation('rbac.removeRole')!.run({ id: 'r1' });
    expect(vi.mocked(apiClient).mock.calls[0][0]).toBe('/api/rbac/roles/r1');
    expect(vi.mocked(apiClient).mock.calls[0][1]).toMatchObject({ method: 'DELETE' });
  });

  it('loads a role\'s users and field permissions', async () => {
    vi.mocked(apiClient).mockResolvedValueOnce([{ id: 'u1', username: 'ann', name: 'Ann', email: 'a@x', assigned_at: '2026-03-01T00:00:00Z' }]);
    const users = (await getOperation('rbac.roleUsers')!.run({ roleId: 'r1' })) as { rows: { name: string }[] };
    expect(users.rows[0].name).toBe('Ann');
    expect(vi.mocked(apiClient).mock.calls[0][0]).toBe('/api/rbac/roles/r1/users');

    vi.mocked(apiClient).mockResolvedValueOnce([{ id: 'p1', term_name: 'acct', resource_type: 'bo', permission_level: 'write' }]);
    const perms = (await getOperation('rbac.roleFieldPermissions')!.run({ roleId: 'r1' })) as { rows: { field: string }[] };
    expect(perms.rows[0].field).toBe('acct');
    expect(vi.mocked(apiClient).mock.calls[1][0]).toBe('/api/rbac/field-permissions?role_id=r1');
  });
});
