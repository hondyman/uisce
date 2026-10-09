import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { fabricSettingsBlueprint } from '../../pages/page-studio/app/blueprints/fabricSettings';
import { getDomainComponent } from '../../studio-core/components/registry';
import '../../features/platform-settings/studio';

// The seed migration's JSON is generated from the blueprint. If either changes
// without the other, a fresh database serves a page that differs from the one the
// studio would build. This fails until the migration is regenerated.
const SEED = resolve(
  __dirname,
  '../../../../backend/db/migrations/20261226_003_seed_fabric_settings_page.up.sql',
);

function block(sql: string, tag: string): unknown {
  const m = sql.match(new RegExp(`\\$${tag}\\$([\\s\\S]*?)\\$${tag}\\$`));
  if (!m) throw new Error(`no $${tag}$ block in the seed`);
  return JSON.parse(m[1]);
}

describe('Settings seed matches its blueprint', () => {
  const sql = readFileSync(SEED, 'utf8');
  const bp = fabricSettingsBlueprint();

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
    expect(bp.slug).toBe('fabric-settings');
  });

  it('places the registered Appearance component', () => {
    expect(sql).toContain('"type": "platformSettings.Appearance"');
    expect(getDomainComponent('platformSettings.Appearance')).toBeDefined();
  });
});
