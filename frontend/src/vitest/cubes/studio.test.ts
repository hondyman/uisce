import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../features/cubes/cubeDefinitionApi', () => ({
  listCubes: vi.fn(),
  getCube: vi.fn(),
  createCube: vi.fn(),
  patchCube: vi.fn(),
  validateCube: vi.fn(),
  deployCube: vi.fn(),
  refreshCube: vi.fn(),
  listCubeMetrics: vi.fn(),
  getCubeImpact: vi.fn(),
  previewCubeImpact: vi.fn(),
  cascadeCube: vi.fn(),
}));

vi.mock('../../studio-core/binding/businessObjectApi', async (orig) => {
  const real = await orig<typeof import('../../studio-core/binding/businessObjectApi')>();
  return {
    ...real,
    listBusinessObjects: vi.fn(),
  };
});

import {
  createCube,
  deployCube,
  getCube,
  cascadeCube,
  getCubeImpact,
  listCubeMetrics,
  listCubes,
  patchCube,
  previewCubeImpact,
  refreshCube,
  validateCube,
} from '../../features/cubes/cubeDefinitionApi';
import { listBusinessObjects } from '../../studio-core/binding/businessObjectApi';
import { emptyCubeDraft } from '../../features/cubes/draft';
import '../../features/cubes/studio';
import { getDomainComponent, listDomainComponents } from '../../studio-core/components/registry';
import { getOperation } from '../../studio-core/operations/registry';

const api = {
  listCubes: listCubes as unknown as ReturnType<typeof vi.fn>,
  getCube: getCube as unknown as ReturnType<typeof vi.fn>,
  createCube: createCube as unknown as ReturnType<typeof vi.fn>,
  patchCube: patchCube as unknown as ReturnType<typeof vi.fn>,
  validateCube: validateCube as unknown as ReturnType<typeof vi.fn>,
  deployCube: deployCube as unknown as ReturnType<typeof vi.fn>,
  refreshCube: refreshCube as unknown as ReturnType<typeof vi.fn>,
  listCubeMetrics: listCubeMetrics as unknown as ReturnType<typeof vi.fn>,
  getCubeImpact: getCubeImpact as unknown as ReturnType<typeof vi.fn>,
  previewCubeImpact: previewCubeImpact as unknown as ReturnType<typeof vi.fn>,
  cascadeCube: cascadeCube as unknown as ReturnType<typeof vi.fn>,
  listBusinessObjects: listBusinessObjects as unknown as ReturnType<typeof vi.fn>,
};

const run = (id: string, params: Record<string, unknown> = {}) => getOperation(id)!.run(params);

const sampleCube = {
  id: 'c-1',
  tenantId: 't',
  name: 'Positions',
  description: 'd',
  boId: 'bo-pos',
  dimensions: [{ termNodeId: 'term-a' }],
  metricIds: ['m1'],
  grains: [['term-a']],
  materialization: { strategy: 'starrocks_mv' },
  federation: {},
  contractVersion: 2,
  contentHash: 'h',
  isCore: false,
  status: 'active',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-02T00:00:00Z',
};

beforeEach(() => Object.values(api).forEach((f) => f.mockReset()));

describe('cubes studio registration (PR3)', () => {
  it('registers every operation the designer host will use', () => {
    for (const id of [
      'cubes.list',
      'cubes.get',
      'cubes.editorStart',
      'cubes.patchDraft',
      'cubes.create',
      'cubes.patch',
      'cubes.validate',
      'cubes.deploy',
      'cubes.refresh',
      'cubes.metrics',
      'cubes.businessObjects',
      'cubes.impact',
      'cubes.impactPreview',
      'cubes.cascade',
    ]) {
      expect(getOperation(id), id).toBeTruthy();
    }
    expect(getOperation('cubes.list')?.kind).toBe('query');
    expect(getOperation('cubes.create')?.kind).toBe('mutation');
    expect(getOperation('cubes.validate')?.kind).toBe('mutation');
    expect(getOperation('cubes.impact')?.kind).toBe('query');
    expect(getOperation('cubes.impactPreview')?.kind).toBe('mutation');
    expect(getOperation('cubes.cascade')?.kind).toBe('mutation');
  });

  it('registers cubes.FederationEditor + cubes.ImpactPanel and never a full-page cubes.Designer', () => {
    const fed = getDomainComponent('cubes.FederationEditor');
    expect(fed?.domain).toBe('cubes');
    expect(fed?.inputs.map((i) => i.name)).toEqual(
      expect.arrayContaining(['primaryBoId', 'bos', 'federation', 'keySamples']),
    );
    expect(fed?.events.map((e) => e.name)).toEqual(['onChange', 'onKeySamplesChange']);
    const impact = getDomainComponent('cubes.ImpactPanel');
    expect(impact?.domain).toBe('cubes');
    expect(impact?.inputs.map((i) => i.name)).toEqual(
      expect.arrayContaining(['cubeId', 'draft', 'readOnly', 'title']),
    );
    expect(getDomainComponent('cubes.Designer')).toBeUndefined();
    expect(listDomainComponents().some((d) => d.id === 'cubes.Designer')).toBe(false);
  });
});

describe('cubes.patchDraft', () => {
  it('merges federation and key samples into the draft', async () => {
    const draft = emptyCubeDraft({ name: 'X', boId: 'account' });
    const federation = {
      sources: [{ boId: 'account', alias: 't0' }],
      joins: [],
    };
    const samples = [{ leftAlias: 't0', rightAlias: 't1', leftKeys: 1, rightKeys: 1, matched: 1 }];
    const r = (await run('cubes.patchDraft', {
      draft,
      federation,
      federationKeySamples: samples,
      patch: { description: 'merged' },
    })) as { draft: { federation: typeof federation; federationKeySamples: typeof samples; description: string } };
    expect(r.draft.federation).toEqual(federation);
    expect(r.draft.federationKeySamples).toEqual(samples);
    expect(r.draft.description).toBe('merged');
  });
});

describe('cubes.editorStart', () => {
  it('returns is_new draft when id is absent or new', async () => {
    const a = (await run('cubes.editorStart', {})) as { is_new: boolean; draft: { name: string }; key: string };
    const b = (await run('cubes.editorStart', { id: 'new' })) as { is_new: boolean };
    expect(a.is_new).toBe(true);
    expect(b.is_new).toBe(true);
    expect(a.key).toBe('new');
    expect(a.draft.name).toBe('');
    expect(api.getCube).not.toHaveBeenCalled();
  });

  it('loads an existing cube into draft', async () => {
    api.getCube.mockResolvedValue(sampleCube);
    const r = (await run('cubes.editorStart', { id: 'c-1' })) as {
      is_new: boolean;
      draft: { name: string; boId: string };
      title: string;
    };
    expect(r.is_new).toBe(false);
    expect(r.draft.name).toBe('Positions');
    expect(r.draft.boId).toBe('bo-pos');
    expect(r.title).toBe('Positions');
  });
});

describe('cubes.list / create / validate / businessObjects', () => {
  it('lists cubes as rows', async () => {
    api.listCubes.mockResolvedValue({ cubes: [sampleCube], scope: 'all' });
    const r = (await run('cubes.list', { scope: 'all' })) as { rows: { id: string }[] };
    expect(api.listCubes).toHaveBeenCalledWith(expect.objectContaining({ scope: 'all' }));
    expect(r.rows[0].id).toBe('c-1');
  });

  it('create posts draft payload and returns id', async () => {
    api.createCube.mockResolvedValue(sampleCube);
    const draft = emptyCubeDraft({ name: 'Positions', boId: 'bo-pos', metricIds: ['m1'] });
    const r = (await run('cubes.create', { draft })) as { id: string };
    expect(api.createCube).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Positions', boId: 'bo-pos', metricIds: ['m1'] }),
    );
    expect(r.id).toBe('c-1');
  });

  it('validate forwards federationKeySamples from draft', async () => {
    api.validateCube.mockResolvedValue({ ok: true, structuralOk: true, metricsOk: true });
    const draft = emptyCubeDraft({
      name: 'X',
      boId: 'bo',
      federation: {
        sources: [{ boId: 'bo', alias: 't0' }],
        joins: [],
      },
      federationKeySamples: [
        { leftAlias: 't0', rightAlias: 't1', leftKeys: 3, rightKeys: 3, matched: 3 },
      ],
    });
    await run('cubes.validate', { id: 'c-1', draft });
    expect(api.validateCube).toHaveBeenCalledWith(
      'c-1',
      expect.objectContaining({
        federationKeySamples: [
          expect.objectContaining({ leftAlias: 't0', matched: 3 }),
        ],
      }),
    );
  });

  it('businessObjects wraps listBusinessObjects', async () => {
    api.listBusinessObjects.mockResolvedValue([
      { id: '1', key: 'account', name: 'account', displayName: 'Account' },
    ]);
    const r = (await run('cubes.businessObjects', {})) as { bos: { key: string }[]; rows: unknown[] };
    expect(r.bos[0].key).toBe('account');
    expect(r.rows).toHaveLength(1);
  });

  it('deploy and refresh pass force/grain', async () => {
    api.deployCube.mockResolvedValue({ cube_id: 'c-1', force: true, starts: [] });
    api.refreshCube.mockResolvedValue({ cube_id: 'c-1', force: false, starts: [] });
    await run('cubes.deploy', { id: 'c-1', force: true, grain: ['d1'] });
    await run('cubes.refresh', { id: 'c-1', grain: 'd1,d2' });
    expect(api.deployCube).toHaveBeenCalledWith('c-1', { grain: ['d1'], force: true });
    expect(api.refreshCube).toHaveBeenCalledWith('c-1', { grain: ['d1', 'd2'], force: false });
  });

  it('metrics lists by boId', async () => {
    api.listCubeMetrics.mockResolvedValue([{ id: 'm1', name: 'AUM' }]);
    const r = (await run('cubes.metrics', { boId: 'bo-1' })) as { metrics: { id: string }[] };
    expect(api.listCubeMetrics).toHaveBeenCalledWith('bo-1');
    expect(r.metrics[0].id).toBe('m1');
  });

  it('patch updates an existing cube', async () => {
    api.patchCube.mockResolvedValue({ cube: sampleCube });
    const draft = emptyCubeDraft({ name: 'Positions', boId: 'bo-pos' });
    await run('cubes.patch', { id: 'c-1', draft });
    expect(api.patchCube).toHaveBeenCalledWith('c-1', expect.objectContaining({ name: 'Positions' }));
  });
});

describe('cubes.impact / impactPreview', () => {
  it('loads inventory and preview via API helpers', async () => {
    api.getCubeImpact.mockResolvedValueOnce({
      cubeId: 'c-1',
      composition: { name: 'Positions' },
      consumers: [],
      summary: { consumerCount: 0, blockingCount: 0, warningCount: 0, physicalGrainCount: 0 },
    });
    const inv = (await run('cubes.impact', { id: 'c-1' })) as { cubeId: string };
    expect(inv.cubeId).toBe('c-1');
    expect(api.getCubeImpact).toHaveBeenCalledWith('c-1', { includePhysical: true });

    api.previewCubeImpact.mockResolvedValueOnce({
      changeClass: 'archive',
      confirmToken: 'tok',
      allowedModes: ['fail_closed'],
      recommendedMode: 'fail_closed',
      breakReasons: [],
      blockingCount: 0,
    });
    const prev = (await run('cubes.impactPreview', { id: 'c-1', action: 'archive' })) as {
      changeClass: string;
    };
    expect(prev.changeClass).toBe('archive');
    expect(api.previewCubeImpact).toHaveBeenCalledWith('c-1', { action: 'archive', patch: undefined });

    api.cascadeCube.mockResolvedValueOnce({
      changeClass: 'archive',
      mode: 'fail_closed',
      action: 'archive',
      cube: { id: 'c-1', status: 'archived' },
      consumersAffected: [{ kind: 'cube', id: 'c-1', action: 'archived' }],
    });
    const casc = (await run('cubes.cascade', {
      id: 'c-1',
      action: 'archive',
      confirmToken: 'tok',
      mode: 'fail_closed',
    })) as { mode: string };
    expect(casc.mode).toBe('fail_closed');
    expect(api.cascadeCube).toHaveBeenCalledWith('c-1', {
      action: 'archive',
      confirmToken: 'tok',
      mode: 'fail_closed',
      patch: undefined,
    });
  });
});
