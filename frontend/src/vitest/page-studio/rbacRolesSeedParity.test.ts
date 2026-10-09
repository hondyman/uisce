import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { rbacRolesBlueprint } from '../../pages/page-studio/app/blueprints/rbacRoles';

// The seed migration's JSON is generated from the blueprint. If either changes
// without the other, a fresh database serves a page that differs from the one the
// studio would build. This fails until the migration is regenerated.
const SEED = resolve(__dirname, '../../../../backend/db/migrations/20261226_004_seed_rbac_roles_page.up.sql');

function block(sql: string, tag: string): unknown {
  const m = sql.match(new RegExp(`\\$${tag}\\$([\\s\\S]*?)\\$${tag}\\$`));
  if (!m) throw new Error(`no $${tag}$ block in the seed`);
  return JSON.parse(m[1]);
}

describe('Roles seed matches its blueprint', () => {
  const sql = readFileSync(SEED, 'utf8');
  const bp = rbacRolesBlueprint();

  it.each([
    ['layout', 'layout'],
    ['tabs', 'tabs'],
    ['comp', 'components'],
    ['ds', 'dataSources'],
    ['pe', 'presentationEvents'],
    ['fb', 'filterBar'],
    ['app', 'app'],
  ])('%s block equals the blueprint %s', (tag, key) => {
    expect(block(sql, tag)).toEqual((bp as unknown as Record<string, unknown>)[key]);
  });

  it('seeds a core, published page under the blueprint slug in the gold copy', () => {
    expect(sql).toContain(`'${bp.slug}'`);
    expect(sql).toContain("'99e99e99-99e9-49e9-89e9-99e99e99e999'");
    expect(sql).toMatch(/true, 'published'/);
    expect(bp.slug).toBe('rbac-roles');
  });

  it('wires the role operations the blueprint uses', () => {
    for (const operation of ['rbac.listRoles', 'rbac.roleStart', 'rbac.saveRole', 'rbac.removeRole', 'rbac.roleUsers', 'rbac.roleFieldPermissions']) {
      expect(sql).toContain(operation);
    }
  });

  it('keeps the old sample-role fallback out of the page', () => {
    expect(sql).not.toMatch(/role_key.*administrator/i);
  });
});
