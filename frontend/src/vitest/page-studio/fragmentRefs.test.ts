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

import { withFragments } from '../../pages/page-studio/app/fragmentRefs';
import { checkPage } from '../../pages/page-studio/app/pageChecker';

describe('withFragments', () => {
  const page = () => ({
    components: { own: { id: 'own', type: 'TextBlock', props: { text: 'hi' } } },
    layout: { root: 'top', nodes: { top: { id: 'top', type: 'Column', children: ['own', 'aRoot'] } } },
    tabs: [] as { layout?: { root?: string; nodes?: Record<string, unknown> } }[],
    app: { variables: [{ name: 'mine' }], queries: [] as { id: string }[], fragments: [{ fragment: 'a', version: 1 }] },
  });
  const { fragment } = resolveFragments([{ fragment: 'a', version: 1 }], store(['a', 1, frag('a')]));

  it('makes the fragment part of the page for the runtime and the checker, without changing the page', () => {
    const p = page();
    const before = JSON.stringify(p);
    const merged = withFragments(p, fragment);
    expect(merged.problems).toEqual([]);
    expect(Object.keys(merged.page.components!).sort()).toEqual(['aW', 'own']);
    expect(Object.keys(merged.page.layout!.nodes!).sort()).toEqual(['aRoot', 'top']);
    expect(merged.page.app!.variables!.map((v) => v.name)).toEqual(['mine', 'aV']);
    expect(merged.page.app!.queries!.map((q) => q.id)).toEqual(['aQ']);
    expect(JSON.stringify(p)).toBe(before); // saving the page never writes fragment content into it
  });

  it('turns a "layout node does not exist" error into a clean page once the fragment is applied', () => {
    const p = page();
    expect(checkPage(p as never).some((i) => i.code === 'missing-node')).toBe(true);
    const merged = withFragments(p, fragment).page;
    expect(checkPage(merged as never).filter((i) => i.code === 'missing-node')).toEqual([]);
  });

  it('is available to every tab, and names what the page already uses instead of replacing it', () => {
    const p = page();
    p.tabs = [{ layout: { root: 't', nodes: { t: { id: 't', type: 'Column', children: [] } } } }];
    p.components = { ...p.components, aW: { id: 'aW', type: 'TextBlock', props: {} } } as never;
    const merged = withFragments(p, fragment);
    expect(Object.keys(merged.page.tabs![0].layout!.nodes!)).toContain('aRoot');
    expect(merged.problems.join()).toMatch(/already has a widget "aW"/);
    expect((merged.page.components as Record<string, { props: unknown }>).aW.props).toEqual({}); // the page's own, kept
  });
});
