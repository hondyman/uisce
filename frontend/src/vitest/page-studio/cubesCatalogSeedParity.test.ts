import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { cubesCatalogBlueprint } from '../../pages/page-studio/app/blueprints/cubesCatalog';

// The seed migration's JSON is generated from the blueprint. If either changes
// without the other, a fresh database serves a page that differs from the one the
// studio would build. This fails until the migration is regenerated.
const SEED = resolve(
  __dirname,
  '../../../../backend/db/migrations/20261220_001_seed_cubes_catalog_page.up.sql',
);

function block(sql: string, tag: string): unknown {
  const m = sql.match(new RegExp(`\\$${tag}\\$([\\s\\S]*?)\\$${tag}\\$`));
  if (!m) throw new Error(`no $${tag}$ block in the seed`);
  return JSON.parse(m[1]);
}

describe('Cubes catalog seed matches its blueprint', () => {
  const sql = readFileSync(SEED, 'utf8');
  const bp = cubesCatalogBlueprint();

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
    expect(bp.slug).toBe('cubes-catalog');
  });

  it('wires cubes.list and navigate to the coded designer paths', () => {
    expect(sql).toContain('cubes.list');
    expect(sql).toContain('/build/cubes/new');
    expect(sql).toContain('/build/cubes/{{row.id}}');
    expect(sql).toContain('cubes.deploy');
    expect(sql).toContain('cubes.refresh');
  });
});
