import { computeQuickFixActions } from '../MonacoCodeEditor';

describe('Monaco completions & insert descriptors', () => {
  it('computeQuickFixActions provides insert descriptor for missing datasource', () => {
    const markers = [{ code: 'MISSING_DATASOURCE', message: 'missing datasource' }];
    const actions = computeQuickFixActions(markers, '{}', 'json');
    const a = actions.find((x: any) => x.title && x.title.includes('datasource'));
    expect(a).toBeDefined();
    expect(a.insert).toBeDefined();
    expect(a.insert.key).toBe('tenant_instance_id');
  });

  it('computeQuickFixActions derives path from marker.source for nested keys', () => {
    const markers = [{ code: 'MISSING_JOIN', message: 'missing join', source: 'cubes.orders.joins.customer' }];
    const actions = computeQuickFixActions(markers, '{"cubes":{}}', 'json');
    const a = actions.find((x: any) => x.title && x.title.includes('Scaffold join'));
    expect(a).toBeDefined();
    expect(a.insert).toBeDefined();
    // path should be all segments except last
    expect(Array.isArray(a.insert.path)).toBe(true);
    expect(a.insert.path.join('.')).toBe('cubes.orders.joins');
    expect(a.insert.key).toBe('joins');
  });
});
