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
  listCubeMetrics,
  listCubes,
  patchCube,
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
      'cubes.create',
      'cubes.patch',
      'cubes.validate',
      'cubes.deploy',
      'cubes.refresh',
      'cubes.metrics',
      'cubes.businessObjects',
    ]) {
      expect(getOperation(id), id).toBeTruthy();
    }
    expect(getOperation('cubes.list')?.kind).toBe('query');
    expect(getOperation('cubes.create')?.kind).toBe('mutation');
    expect(getOperation('cubes.validate')?.kind).toBe('mutation');
  });

  it('registers cubes.FederationEditor and never a full-page cubes.Designer', () => {
    const fed = getDomainComponent('cubes.FederationEditor');
    expect(fed?.domain).toBe('cubes');
    expect(fed?.inputs.map((i) => i.name)).toEqual(
      expect.arrayContaining(['primaryBoId', 'bos', 'federation', 'keySamples']),
    );
    expect(fed?.events.map((e) => e.name)).toEqual(['onChange', 'onKeySamplesChange']);
    expect(getDomainComponent('cubes.Designer')).toBeUndefined();
    expect(listDomainComponents().some((d) => d.id === 'cubes.Designer')).toBe(false);
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
