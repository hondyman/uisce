import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { cubeDesignerBlueprint } from '../../pages/page-studio/app/blueprints/cubeDesigner';

// The seed migration's JSON is generated from the blueprint. If either changes
// without the other, a fresh database serves a page that differs from the one the
// studio would build. This fails until the migration is regenerated.
const SEED = resolve(
  __dirname,
  '../../../../backend/db/migrations/20261221_001_seed_cube_designer_page.up.sql',
);

function block(sql: string, tag: string): unknown {
  const m = sql.match(new RegExp(`\\$${tag}\\$([\\s\\S]*?)\\$${tag}\\$`));
  if (!m) throw new Error(`no $${tag}$ block in the seed`);
  return JSON.parse(m[1]);
}

describe('Cube designer seed matches its blueprint', () => {
  const sql = readFileSync(SEED, 'utf8');
  const bp = cubeDesignerBlueprint();

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
    expect(bp.slug).toBe('cube-designer');
  });

  it('wires editorStart, FederationEditor, ImpactPanel, create→navigate, and designer tabs', () => {
    expect(sql).toContain('cubes.editorStart');
    expect(sql).toContain('cubes.FederationEditor');
    expect(sql).toContain('cubes.ImpactPanel');
    expect(sql).toContain('cubes.create');
    expect(sql).toContain('/build/cubes/{{result.id}}');
    expect(sql).toContain('cubes.patchDraft');
    expect(sql).toContain('"overview"');
    expect(sql).toContain('"federation"');
    expect(sql).toContain('"materialization"');
    expect(sql).toContain('"impact"');
  });
});
