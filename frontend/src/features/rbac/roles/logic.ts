/**
 * Role manager logic, free of React and the network so it can be tested directly.
 * The API is the source of truth: a role's level is set when it is created (the
 * update endpoint does not change it), and only the tenant's own roles can be edited.
 */

export const ROLE_LEVELS = [
  { value: 'viewer', label: 'Viewer' },
  { value: 'editor', label: 'Editor' },
  { value: 'approver', label: 'Approver' },
  { value: 'admin', label: 'Admin' },
  { value: 'super_admin', label: 'Super Admin' },
] as const;

export function levelLabel(level: string): string {
  const hit = ROLE_LEVELS.find((l) => l.value === level);
  return hit ? hit.label : level.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

/** A role as GET /api/rbac/roles returns it. */
export interface RawRole {
  id: string;
  role_key: string;
  role_name: string;
  description?: string | null;
  role_level: string;
  is_active?: boolean;
  is_template?: boolean;
  /** gold_copy | extended | tenant (derived by the API). */
  origin?: string;
}

export interface RoleRow {
  id: string;
  role_key: string;
  role_name: string;
  description: string;
  role_level: string;
  level_label: string;
  origin: string;
  origin_label: string;
  /** Gold-copy roles are shared and read-only for a tenant. */
  editable: boolean;
}

const ORIGIN_LABELS: Record<string, string> = { gold_copy: 'Gold copy', extended: 'Extended', tenant: 'Tenant' };

export function toRoleRow(r: RawRole): RoleRow {
  const origin = r.origin || (r.is_template ? 'gold_copy' : 'tenant');
  return {
    id: r.id,
    role_key: r.role_key,
    role_name: r.role_name,
    description: r.description ?? '',
    role_level: r.role_level,
    level_label: levelLabel(r.role_level),
    origin,
    origin_label: ORIGIN_LABELS[origin] ?? origin,
    editable: origin !== 'gold_copy',
  };
}

/** Name or key contains the query (case-insensitive), and the level matches when one is chosen. */
export function matchesRole(row: RoleRow, q: string, level: string): boolean {
  const needle = q.trim().toLowerCase();
  const byText = !needle || row.role_name.toLowerCase().includes(needle) || row.role_key.toLowerCase().includes(needle);
  const byLevel = !level || row.role_level === level;
  return byText && byLevel;
}

export interface RoleDraft {
  id?: string;
  role_key: string;
  role_name: string;
  description: string;
  role_level: string;
}

/** Create body: key and level are set here and cannot change later. */
export function createRoleBody(d: RoleDraft) {
  return { role_key: d.role_key, role_name: d.role_name, description: d.description, role_level: d.role_level, permissions: [] as string[] };
}

/**
 * Update body: only the name and description. The status is not sent, so an edit
 * never reactivates a role, and the level is not sent because the API ignores it.
 */
export function updateRoleBody(d: RoleDraft) {
  return { role_name: d.role_name, description: d.description };
}

export interface UserRow {
  id: string;
  name: string;
  email: string;
  assigned_text: string;
}

export function userRow(u: Record<string, unknown>): UserRow {
  const name = String(u.name ?? '') || String(u.username ?? '');
  return {
    id: String(u.id ?? ''),
    name,
    email: String(u.email ?? ''),
    assigned_text: String(u.assigned_at ?? '').slice(0, 10),
  };
}

export interface FieldPermissionRow {
  id: string;
  field: string;
  resource_type: string;
  permission_level: string;
}

/** The API names the field by its term (display name, else term name, else node id). */
export function fieldPermissionRow(fp: Record<string, unknown>): FieldPermissionRow {
  const field = String(fp.term_display_name ?? '') || String(fp.term_name ?? '') || String(fp.term_node_id ?? '');
  return {
    id: String(fp.id ?? ''),
    field,
    resource_type: String(fp.resource_type ?? ''),
    permission_level: String(fp.permission_level ?? ''),
  };
}
