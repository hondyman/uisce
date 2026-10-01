import React from 'react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../api/pageStudio', () => ({
  PageStudioApi: {
    getFragmentVersion: vi.fn(async (slug: string, version: number) => {
      if (slug === 'orders') return { slug, version, name: 'Orders', contentHash: 'h', content: { root: 'ordersRoot', components: { ordersW: { id: 'ordersW', type: 'TextBlock', props: {} } }, nodes: { ordersRoot: { type: 'Column', children: ['ordersW'] } }, variables: [{ name: 'ordersV' }], queries: [] } };
      throw new Error('404');
    }),
  },
}));
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fetchFragmentClosure, resolvePageFragments, useResolvedPage, type FragmentGetter } from '../../pages/page-studio/app/fragmentLoad';
import type { FragmentRef, StoredFragment } from '../../pages/page-studio/app/fragmentRefs';

const frag = (id: string, uses?: FragmentRef[]): StoredFragment => ({
  root: `${id}Root`, uses,
  components: { [`${id}W`]: { id: `${id}W`, type: 'TextBlock', props: {} } },
  nodes: { [`${id}Root`]: { type: 'Column', children: [`${id}W`] } },
  variables: [{ name: `${id}V` }], queries: [],
});
const fake = (...fs: [string, number, StoredFragment][]): FragmentGetter & ReturnType<typeof vi.fn> =>
  vi.fn(async (r: FragmentRef) => fs.find(([s, v]) => s === r.fragment && v === r.version)?.[2]) as never;

describe('fetchFragmentClosure', () => {
  it('fetches each version once, and the fragments they use', async () => {
    const get = fake(['a', 1, frag('a', [{ fragment: 'b', version: 1 }])], ['b', 1, frag('b')]);
    const got = await fetchFragmentClosure([{ fragment: 'a', version: 1 }, { fragment: 'a', version: 1 }, { fragment: 'b', version: 1 }], get);
    expect([...got.keys()].sort()).toEqual(['a@1', 'b@1']);
    expect(get).toHaveBeenCalledTimes(2); // a once (listed twice), b once (listed and used)
  });
  it('stops at the nesting cap instead of walking forever', async () => {
    const get = fake(['a', 1, frag('a', [{ fragment: 'b', version: 1 }])], ['b', 1, frag('b', [{ fragment: 'c', version: 1 }])], ['c', 1, frag('c')]);
    const got = await fetchFragmentClosure([{ fragment: 'a', version: 1 }], get);
    expect([...got.keys()].sort()).toEqual(['a@1', 'b@1']); // c is three levels down and not fetched
  });
  it('leaves out a fetch that fails, so the resolver can name it missing', async () => {
    const get: FragmentGetter = async () => { throw new Error('404'); };
    const { problems } = await resolvePageFragments([{ fragment: 'gone', version: 3 }], get);
    expect(problems[0]).toMatch(/"gone" version 3 does not exist/);
  });
});

describe('useResolvedPage', () => {
  const wrap = (ui: React.ReactElement) => render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>,
  );
  const Probe = ({ page }: { page: Parameters<typeof useResolvedPage>[0] }) => {
    const r = useResolvedPage(page);
    return <div data-testid="out">{JSON.stringify({ widgets: Object.keys(r.page.components ?? {}), pending: r.pending, problems: r.problems })}</div>;
  };
  const out = () => JSON.parse(screen.getByTestId('out').textContent ?? '{}') as { widgets: string[]; pending: boolean; problems: string[] };

  it('returns a page that names no fragments untouched, with nothing pending', () => {
    wrap(<Probe page={{ components: { own: {} }, app: { variables: [], queries: [] } }} />);
    expect(out()).toEqual({ widgets: ['own'], pending: false, problems: [] });
  });

  it('is pending while the fragments load, then returns the page with them applied', async () => {
    wrap(<Probe page={{ components: { own: {} }, app: { variables: [], queries: [], fragments: [{ fragment: 'orders', version: 1 }] } }} />);
    expect(out().pending).toBe(true);
    await screen.findByText(/ordersW/);
    expect(out()).toEqual({ widgets: ['own', 'ordersW'], pending: false, problems: [] });
  });

  it('keeps the page and says why when a fragment cannot be loaded', async () => {
    wrap(<Probe page={{ components: { own: {} }, app: { variables: [], queries: [], fragments: [{ fragment: 'missing', version: 2 }] } }} />);
    await screen.findByText(/does not exist/);
    expect(out().widgets).toEqual(['own']);
    expect(out().pending).toBe(false);
  });
});
