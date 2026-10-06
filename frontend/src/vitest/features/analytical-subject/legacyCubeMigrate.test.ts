import { describe, it, expect } from 'vitest';
import {
  migrateLegacyCubeName,
  resolveReportCubeBinding,
  rewriteDataBindingsWithSubject,
  extractLegacyCubeName,
} from '../../../features/analytical-subject/legacyCubeMigrate';
import { cubeSubject, businessObjectSubject } from '../../../features/analytical-subject/types';

const cubes = [
  { id: 'cube-acct', name: 'oms.account', contractVersion: 2 },
  { id: 'cube-pos', name: 'oms.position', contractVersion: 1 },
];

describe('migrateLegacyCubeName', () => {
  it('resolves exact name to pinned subject', () => {
    const r = migrateLegacyCubeName('oms.account', cubes);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.migrated).toBe(true);
    expect(r.subject).toEqual(cubeSubject('cube-acct', 2));
    expect(r.cube?.id).toBe('cube-acct');
  });

  it('trims whitespace', () => {
    const r = migrateLegacyCubeName('  oms.position  ', cubes);
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.subject.cubeId).toBe('cube-pos');
  });

  it('fails closed on blank name', () => {
    const r = migrateLegacyCubeName('   ', cubes);
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.reason).toMatch(/blank/);
  });

  it('fails closed when unresolved', () => {
    const r = migrateLegacyCubeName('cash_flow.settlement', cubes);
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.reason).toMatch(/unresolved legacy cube name/);
  });

  it('fails closed when ambiguous', () => {
    const r = migrateLegacyCubeName('oms.account', [
      ...cubes,
      { id: 'cube-acct-2', name: 'oms.account', contractVersion: 3 },
    ]);
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.reason).toMatch(/ambiguous/);
  });

  it('fails closed on invalid contractVersion', () => {
    const r = migrateLegacyCubeName('bad', [
      { id: 'x', name: 'bad', contractVersion: 0 },
    ]);
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.reason).toMatch(/invalid contractVersion/);
  });
});

describe('resolveReportCubeBinding', () => {
  it('passes through already-pinned cube subject', () => {
    const pin = cubeSubject('cube-acct', 2);
    const r = resolveReportCubeBinding({ subject: pin, cube: 'oms.account' }, cubes);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.migrated).toBe(false);
    expect(r.subject).toEqual(pin);
  });

  it('passes through business_object subject', () => {
    const bo = businessObjectSubject('bo-1', 'bind-1');
    const r = resolveReportCubeBinding({ subject: bo }, cubes);
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.subject).toEqual(bo);
  });

  it('migrates legacy cube name when subject absent', () => {
    const r = resolveReportCubeBinding({ cube: 'oms.account' }, cubes);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.migrated).toBe(true);
    expect(r.subject).toEqual(cubeSubject('cube-acct', 2));
  });

  it('fails closed when neither subject nor name', () => {
    const r = resolveReportCubeBinding({ measures: [] }, cubes);
    expect(r.ok).toBe(false);
  });

  it('fails closed on blank cubeId in subject', () => {
    const r = resolveReportCubeBinding(
      { subject: { kind: 'cube', cubeId: '  ', contractVersion: 1 } },
      cubes,
    );
    expect(r.ok).toBe(false);
  });
});

describe('rewriteDataBindingsWithSubject', () => {
  it('drops legacy cube name and sets subject', () => {
    const next = rewriteDataBindingsWithSubject(
      {
        primary: {
          cube: 'oms.account',
          measures: ['SUM(x)'],
          dimensions: ['y'],
        },
      },
      cubeSubject('cube-acct', 2),
    );
    expect(next.primary.cube).toBeUndefined();
    expect(next.primary.subject).toEqual(cubeSubject('cube-acct', 2));
    expect(next.primary.measures).toEqual(['SUM(x)']);
  });
});

describe('extractLegacyCubeName', () => {
  it('returns trimmed name or null', () => {
    expect(extractLegacyCubeName({ cube: ' oms.account ' })).toBe('oms.account');
    expect(extractLegacyCubeName({ cube: '' })).toBeNull();
    expect(extractLegacyCubeName(null)).toBeNull();
  });
});
