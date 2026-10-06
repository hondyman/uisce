import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { lakehouseStatusBlueprint } from '../../pages/page-studio/app/blueprints/lakehouseStatus';

// The seed migration's JSON is generated from the blueprint. If either changes
// without the other, a fresh database serves a page that differs from the one the
// studio would build. This fails until the migration is regenerated.
const SEED = resolve(__dirname, '../../../../backend/db/migrations/20261006_001_seed_lakehouse_status_page.up.sql');

function block(sql: string, tag: string): unknown {
  const m = sql.match(new RegExp(`\\$${tag}\\$([\\s\\S]*?)\\$${tag}\\$`));
  if (!m) throw new Error(`no $${tag}$ block in the seed`);
  return JSON.parse(m[1]);
}

describe('Lakehouse status seed matches its blueprint', () => {
  const sql = readFileSync(SEED, 'utf8');
  const bp = lakehouseStatusBlueprint();

  it.each([
    ['layout', 'layout'], ['tabs', 'tabs'], ['comp', 'components'], ['ds', 'dataSources'],
    ['pe', 'presentationEvents'], ['fb', 'filterBar'], ['app', 'app'],
  ])('%s block equals the blueprint %s', (tag, key) => {
    expect(block(sql, tag)).toEqual((bp as unknown as Record<string, unknown>)[key]);
  });

  it('seeds a core, published page under the blueprint slug in the gold copy', () => {
    expect(sql).toContain(`'${bp.slug}'`);
    expect(sql).toContain("'99e99e99-99e9-49e9-89e9-99e99e99e999'");
    expect(sql).toMatch(/true, 'published'/);
  });

  it('keeps the menu entry gated to platform operators', () => {
    // The leaf is gated by required_entitlement. The System parent is created by
    // 20261207_001_seed_system_lakehouse_page.up.sql and gates itself — this migration
    // only references it.
    expect(sql).toContain("'PLATFORM_OPERATOR'");
    expect(sql).not.toMatch(/'BASE_USER'/);
  });
});