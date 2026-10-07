import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { complianceHubBlueprint } from '../../pages/page-studio/app/blueprints/complianceHub';

const SEED = resolve(__dirname, '../../../../backend/db/migrations/20261224_021_seed_compliance_hub_page.up.sql');

describe('Compliance Hub seed matches its blueprint', () => {
  const sql = readFileSync(SEED, 'utf8');
  const bp = complianceHubBlueprint();

  it('seeds a core, published page under the blueprint slug in the gold copy', () => {
    expect(sql).toContain(`'${bp.slug}'`);
    expect(sql).toContain("'99e99e99-99e9-49e9-89e9-99e99e99e999'");
    expect(sql).toMatch(/true,\s*'published'/);
  });

  it('contains all 7 tab layouts and domain components', () => {
    expect(sql).toContain('compliance.RuleLibraryExplorer');
    expect(sql).toContain('compliance.RuleActivationMatrix');
    expect(sql).toContain('compliance.DecisionBlotter');
    expect(sql).toContain('compliance.SurveillanceQueue');
    expect(sql).toContain('compliance.RegulatoryQueue');
    expect(sql).toContain('compliance.Calendar');
    expect(sql).toContain('compliance.LimitDashboard');
  });
});
