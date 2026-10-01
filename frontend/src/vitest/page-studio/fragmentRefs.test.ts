import { describe, expect, it } from 'vitest';
import { resolveFragments, type StoredFragment, type FragmentRef } from '../../pages/page-studio/app/fragmentRefs';

const frag = (id: string, uses?: FragmentRef[]): StoredFragment => ({
  root: `${id}Root`, uses,
  components: { [`${id}W`]: { id: `${id}W`, type: 'TextBlock', props: {} } },
  nodes: { [`${id}Root`]: { type: 'Column', children: [`${id}W`] } },
  variables: [{ name: `${id}V` }], queries: [{ id: `${id}Q`, operation: 'x.y', params: {} }],
});
const store = (...fs: [string, number, StoredFragment][]) => (r: FragmentRef) => fs.find(([s, v]) => s === r.fragment && v === r.version)?.[2];

describe('resolveFragments', () => {
  it('flattens a reference into widgets, nodes, variables and queries', () => {
    const { fragment, problems } = resolveFragments([{ fragment: 'a', version: 1 }], store(['a', 1, frag('a')]));
    expect(problems).toEqual([]);
    expect(Object.keys(fragment.components)).toEqual(['aW']);
    expect(fragment.variables.map((v) => v.name)).toEqual(['aV']);
    expect(fragment.queries.map((q) => q.id)).toEqual(['aQ']);
  });
  it('uses the exact version asked for', () => {
    const v1 = frag('a'); const v2 = { ...frag('a'), variables: [{ name: 'newer' }] };
    const lookup = store(['a', 1, v1], ['a', 2, v2]);
    expect(resolveFragments([{ fragment: 'a', version: 1 }], lookup).fragment.variables[0].name).toBe('aV');
    expect(resolveFragments([{ fragment: 'a', version: 2 }], lookup).fragment.variables[0].name).toBe('newer');
  });
  it('resolves one level of nesting and refuses a second', () => {
    const lookup = store(['a', 1, frag('a', [{ fragment: 'b', version: 1 }])], ['b', 1, frag('b', [{ fragment: 'c', version: 1 }])], ['c', 1, frag('c')]);
    const ok = resolveFragments([{ fragment: 'b', version: 1 }], store(['b', 1, frag('b', [{ fragment: 'c', version: 1 }])], ['c', 1, frag('c')]));
    expect(ok.problems).toEqual([]);
    expect(Object.keys(ok.fragment.components).sort()).toEqual(['bW', 'cW']);
    const deep = resolveFragments([{ fragment: 'a', version: 1 }], lookup);
    expect(deep.problems.join()).toMatch(/nested deeper than 2/);
  });
  it('names a missing fragment, a cycle, and a clash', () => {
    expect(resolveFragments([{ fragment: 'zz', version: 1 }], () => undefined).problems[0]).toMatch(/does not exist/);
    const loop = store(['a', 1, frag('a', [{ fragment: 'a', version: 1 }])]);
    expect(resolveFragments([{ fragment: 'a', version: 1 }], loop).problems.join()).toMatch(/uses itself/);
    const dup = resolveFragments([{ fragment: 'a', version: 1 }, { fragment: 'a2', version: 1 }], store(['a', 1, frag('a')], ['a2', 1, frag('a')]));
    expect(dup.problems.join()).toMatch(/widget "aW" is already defined/);
  });
});
