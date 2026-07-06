import { describe, it, expect } from 'vitest';
import {
  createEmptyQueryDef,
  getUsedAliases,
  makeAlias,
} from '../../../features/query-builder/types/queryDef';

describe('queryDef helpers', () => {
  it('creates an empty QueryDef', () => {
    const qd = createEmptyQueryDef({
      boId: 'bo-1',
      bindingId: 'binding-1',
      tenantId: 'tenant-1',
    });

    expect(qd.context).toEqual({
      boId: 'bo-1',
      bindingId: 'binding-1',
      tenantId: 'tenant-1',
    });
    expect(qd.query.dimensions).toEqual([]);
    expect(qd.query.measures).toEqual([]);
    expect(qd.query.filters).toEqual([]);
    expect(qd.query.limit).toBe(1000);
  });

  it('collects used aliases', () => {
    const aliases = getUsedAliases({
      dimensions: [{ termNodeId: 'a', alias: 'date' }],
      measures: [{ termNodeId: 'b', alias: 'revenue', agg: 'SUM' }],
      filters: [],
    });

    expect(aliases).toEqual(new Set(['date', 'revenue']));
  });

  it('generates unique aliases', () => {
    const used = new Set(['order_date', 'order_date_2']);
    expect(makeAlias('Order Date', used)).toBe('order_date_3');
  });

  it('sanitizes aliases', () => {
    expect(makeAlias('Revenue %', new Set())).toBe('revenue');
    expect(makeAlias('   ', new Set())).toBe('field');
  });
});
