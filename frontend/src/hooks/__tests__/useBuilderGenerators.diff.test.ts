import { describe, it, expect } from 'vitest';
import { computeCustomModelDiff } from '../useBuilderGenerators';

function makeModel(name: string, opts: Partial<any> = {}) {
  return {
    name,
    model_key: name,
    is_custom: false,
    dimensions: [],
    measures: [],
    filters: [],
    joins: [],
    resolved_config: undefined as any,
    ...opts,
  } as any;
}

describe('computeCustomModelDiff', () => {
  it('emits only changed fields vs core for dimensions/measures/filters/joins', () => {
    const parent = makeModel('core.orders', {
      resolved_config: {
        name: 'orders',
        dimensions: [{ name: 'order_id', title: 'Order ID', type: 'number', sourceTable: 'orders', sourceColumn: 'id' }],
        measures: [{ name: 'count', type: 'count' }],
        filters: [{ name: 'status', type: 'string', values: ['A','B'] }],
        joins: [{ name: 'orders_products', leftTable: 'orders', rightTable: 'products', relationship: 'belongsTo' }],
      },
    });

    const customSelected = { model_key: 'custom.orders', is_custom: true, parent_model_key: parent.model_key } as any;

    const semanticModel = {
      name: 'orders',
      is_custom: true,
      dimensions: [
        // same as core -> omitted
        { id: '1', is_custom: true, name: 'order_id', title: 'Order ID', type: 'number', sourceTable: 'orders', sourceColumn: 'id' },
        // changed title only -> include only title
        { id: '2', name: 'customer_id', title: 'Customer ID', type: 'number', sourceTable: 'orders', sourceColumn: 'customer_id' },
      ],
      measures: [
        { id: 'm1', name: 'count', type: 'count' }, // same as core -> omitted
        { id: 'm2', name: 'revenue', type: 'sum', sql: 'orders.amount' }, // new -> include full allowed
      ],
      filters: [
        { id: 'f1', name: 'status', type: 'string', values: ['A','B','C'] }, // changed values
      ],
      joins: [
        { id: 'j1', name: 'orders_products', leftTable: 'orders', rightTable: 'products', relationship: 'belongsTo' }, // same
        { id: 'j2', name: 'orders_customers', leftTable: 'orders', rightTable: 'customers', relationship: 'belongsTo' }, // new
      ],
    } as any;

    const diff = computeCustomModelDiff({
      selectedModel: customSelected,
      parentResolvedConfig: (parent as any).resolved_config,
      currentConfig: semanticModel,
    }) as any;

    expect(diff.extends).toBe(parent.model_key);
    // dimensions: includes customer_id only (new); order_id same as core should not emit a diff entry
    expect(diff.dimensions.find((d: any) => d.name === 'customer_id')).toBeTruthy();
    expect(diff.dimensions.find((d: any) => d.name === 'order_id')).toBeFalsy();

    // measures: include revenue only, not count (same)
    expect(diff.measures.find((m: any) => m.name === 'revenue')).toBeTruthy();
    expect(diff.measures.find((m: any) => m.name === 'count')).toBeFalsy();

    // filters: status changed -> include with values field
  const status = diff.filters.find((f: any) => f.name === 'status');
    expect(status).toBeTruthy();
    expect(status.values).toEqual(['A','B','C']);

    // joins: include orders_customers only
    expect(diff.joins.find((j: any) => j.name === 'orders_customers')).toBeTruthy();
    expect(diff.joins.find((j: any) => j.name === 'orders_products')).toBeFalsy();
  });
});
