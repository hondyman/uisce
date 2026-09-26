import { describe, expect, it } from 'vitest';
import { downstreamSink, fieldsIn, fieldsOut, missingRequired } from '../../features/data-pipelines/fields';
import type { Spec } from '../../features/data-pipelines/api';

const spec: Spec = {
  version: 1,
  nodes: [
    { id: 'f', type: 'file_source', config: { uri: 'x.csv', format: 'csv', columns: [
      { name: 'FSYM_ID', type: 'string' }, { name: 'AUM', type: 'string' }, { name: 'INCEPTION', type: 'string' }] } },
    { id: 'v', type: 'validate', config: { required: ['FSYM_ID'] } },
    { id: 'm', type: 'map', config: { fields: [
      { from: 'FSYM_ID', to: 'fsym_id' }, { from: 'AUM', to: 'aum', transform: 'to_number' }] } },
    { id: 'r', type: 'rule_check', config: { rule_ids: [] } },
    { id: 's', type: 'bo_sink', config: { bo_key: 'fund' } },
  ],
  edges: [{ from: 'f', to: 'v' }, { from: 'v', to: 'm' }, { from: 'm', to: 'r' }, { from: 'r', to: 's' }],
};
const none = () => undefined;

describe('field flow', () => {
  it('passes source fields through checks', () => {
    expect(fieldsIn(spec, 'm', none).map(c => c.name)).toEqual(['FSYM_ID', 'AUM', 'INCEPTION']);
  });
  it('map outputs only mapped fields, typed by transform', () => {
    expect(fieldsOut(spec, 'm', none)).toEqual([
      { name: 'fsym_id', type: 'string' }, { name: 'aum', type: 'decimal' }]);
    expect(fieldsIn(spec, 's', none).map(c => c.name)).toEqual(['fsym_id', 'aum']);
  });
  it('keep_unmapped carries the rest', () => {
    const s2: Spec = { ...spec, nodes: spec.nodes.map(n => n.id === 'm'
      ? { ...n, config: { ...(n.config as object), keep_unmapped: true } } as typeof n : n) };
    expect(fieldsOut(s2, 'm', none).map(c => c.name)).toEqual(['fsym_id', 'aum', 'INCEPTION']);
  });
  it('uses the lookup for BO sources', () => {
    const s3: Spec = { version: 1, nodes: [{ id: 'b', type: 'bo_source', config: { bo_key: 'fund' } }], edges: [] };
    expect(fieldsOut(s3, 'b', () => [{ name: 'aum', type: 'decimal' }])).toEqual([{ name: 'aum', type: 'decimal' }]);
  });
  it('finds the sink a map feeds, through checks', () => {
    expect(downstreamSink(spec, 'm')?.id).toBe('s');
  });
  it('does not loop on a cycle', () => {
    const cyc: Spec = { version: 1, nodes: [
      { id: 'a', type: 'validate', config: {} }, { id: 'b', type: 'validate', config: {} }],
      edges: [{ from: 'a', to: 'b' }, { from: 'b', to: 'a' }] };
    expect(fieldsOut(cyc, 'a', none)).toEqual([]);
  });
  it('reports required targets left unmapped', () => {
    expect(missingRequired(['aum'], [{ name: 'aum', required: true }, { name: 'fsym_id', required: true }, { name: 'x' }])
      .map(t => t.name)).toEqual(['fsym_id']);
  });
});
