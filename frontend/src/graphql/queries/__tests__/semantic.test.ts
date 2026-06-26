import { normalizeChart } from '../semantic';
import { describe, it, expect } from 'vitest';

describe('normalizeChart', () => {
  it('normalizes core_id and isCore from various shapes', () => {
    const chart = {
      nodes: [
        { id: '1', data: { nodeType: 'table', label: 'A', core_id: 123 } },
        { id: '2', data: { nodeType: 'table', label: 'B', data: { coreId: 'abc' } } },
        { id: '3', data: { nodeType: 'table', label: 'C', properties: { core_id: 'xyz' } } },
        { id: '4', data: { nodeType: 'table', label: 'D', catalog_defn: { core_id: 'zzz' } } },
        { id: '5', data: { nodeType: 'table', label: 'E', isCore: true } },
        { id: '6', data: { nodeType: 'table', label: 'F' } },
      ],
    } as any;

    const normalized = normalizeChart(chart) as any;
    const nodes = normalized.nodes;

    expect(nodes[0].data.core_id).toBe('123');
    expect(nodes[0].data.isCore).toBe(true);

    expect(nodes[1].data.core_id).toBe('abc');
    expect(nodes[1].data.isCore).toBe(true);

    expect(nodes[2].data.core_id).toBe('xyz');
    expect(nodes[2].data.isCore).toBe(true);

    expect(nodes[3].data.core_id).toBe('zzz');
    expect(nodes[3].data.isCore).toBe(true);

    expect(nodes[4].data.isCore).toBe(true);
    expect(nodes[4].data.core_id === undefined || nodes[4].data.core_id === null).toBeTruthy();

    expect(nodes[5].data.isCore).toBe(false);
  });
});
