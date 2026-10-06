import { describe, expect, it } from 'vitest';
import { buildCubeFieldCatalog } from '../../../features/analytical-subject/fieldCatalog';
import { routeBadgeFromPreview } from '../../../features/analytical-subject/RouteBadge';
import type { CubeDefinition } from '../../../features/cubes/types';
import type { SemanticTermView } from '../../../features/query-builder/types/queryDef';

const cube: CubeDefinition = {
  id: 'c1',
  tenantId: 't1',
  name: 'sales_cube',
  boId: 'bo_sales',
  dimensions: [{ termNodeId: 'country' }],
  timeDimension: { termNodeId: 'order_date', defaultGrain: 'day' },
  metricIds: ['m_revenue'],
  grains: [['country', 'order_date']],
  materialization: {},
  federation: {},
  contractVersion: 2,
  contentHash: 'abc',
  isCore: false,
  status: 'active',
  createdAt: '',
  updatedAt: '',
};

const boTerms: SemanticTermView[] = [
  {
    termNodeId: 'country',
    termKey: 'country',
    termName: 'country',
    displayName: 'Country',
    role: 'DIMENSION',
    bindingStatus: 'RESOLVED',
  },
  {
    termNodeId: 'order_date',
    termKey: 'order_date',
    termName: 'order_date',
    displayName: 'Order Date',
    role: 'DIMENSION',
    bindingStatus: 'RESOLVED',
    dataType: 'date',
  },
];

describe('buildCubeFieldCatalog', () => {
  it('exposes dimensions and metrics with metric ids as measure termNodeIds', () => {
    const catalog = buildCubeFieldCatalog(cube, boTerms, [
      {
        id: 'm_revenue',
        name: 'Revenue',
        boId: 'bo_sales',
        decomposable: true,
        isCore: false,
        status: 'active',
      },
    ]);
    expect(catalog.map((t) => t.termNodeId)).toEqual(['country', 'order_date', 'm_revenue']);
    const measure = catalog.find((t) => t.role === 'MEASURE');
    expect(measure?.termNodeId).toBe('m_revenue');
    expect(measure?.displayName).toBe('Revenue');
  });
});

describe('routeBadgeFromPreview', () => {
  it('maps cubeHit to hot badge', () => {
    const badge = routeBadgeFromPreview({
      cubeHit: {
        cubeId: 'c1',
        cubeName: 'sales_cube',
        materialization: 'mv_x',
        servedFrom: 'hot',
        contractVersion: 2,
        stale: false,
      },
    });
    expect(badge?.servedFrom).toBe('hot');
    expect(badge?.contractVersion).toBe(2);
  });

  it('maps cubeMiss to raw badge', () => {
    const badge = routeBadgeFromPreview({ cubeMiss: 'no_cube_for_bo' });
    expect(badge?.servedFrom).toBe('raw');
    expect(badge?.missReason).toBe('no_cube_for_bo');
  });
});
