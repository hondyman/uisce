import schema from '../../../schema/cube-semantic.json';
import { computeQuickFixActions } from '../MonacoCodeEditor';

describe('cube semantic schema', () => {
  it('loads schema and has expected top-level keys', () => {
    expect(schema).toBeDefined();
    expect(schema.properties).toBeDefined();
    expect(schema.properties.tenant_instance_id).toBeDefined();
    expect(schema.properties.joins).toBeDefined();
  });

  it('computeQuickFixActions unaffected by schema load', () => {
    const markers = [{ code: 'MISSING_DATASOURCE', message: 'missing datasource' }];
    const actions = computeQuickFixActions(markers, '{}', 'json');
    expect(actions.some((a: any) => a.title && a.title.includes('datasource'))).toBe(true);
  });
});
