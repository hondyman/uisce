import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { PageStudioApi } from '../../../api/pageStudio';
import type { PageFragment } from './fragment';
import { MAX_FRAGMENT_DEPTH, resolveFragments, withFragments, type FragmentRef, type StoredFragment } from './fragmentRefs';

/**
 * Bringing a page's fragments to where the page is used. The page names them
 * (app.fragments); this fetches those versions - and the ones they use, up to
 * the nesting cap - resolves them, and merges them into a copy of the page for
 * the runtime, the preview and the checker. The saved page is never changed.
 */

export type FragmentGetter = (ref: FragmentRef) => Promise<StoredFragment | undefined>;

const key = (r: FragmentRef) => `${r.fragment}@${r.version}`;

/** Fetch the referenced fragments and, one level further, the ones they use. A fetch that fails is left out and reported by the resolver as missing. */
export async function fetchFragmentClosure(refs: FragmentRef[], get: FragmentGetter): Promise<Map<string, StoredFragment>> {
  const found = new Map<string, StoredFragment>();
  let wave = refs;
  for (let depth = 1; depth <= MAX_FRAGMENT_DEPTH && wave.length; depth++) {
    const fresh = wave.filter((r, i) => !found.has(key(r)) && wave.findIndex((x) => key(x) === key(r)) === i);
    const got = await Promise.all(fresh.map(async (r) => [r, await get(r).catch(() => undefined)] as const));
    const next: FragmentRef[] = [];
    for (const [r, f] of got) {
      if (!f) continue;
      found.set(key(r), f);
      next.push(...(f.uses ?? []));
    }
    wave = next;
  }
  return found;
}

/** What the page's references resolve to: one flattened fragment and what went wrong. */
export async function resolvePageFragments(refs: FragmentRef[], get: FragmentGetter): Promise<{ fragment: PageFragment; problems: string[] }> {
  const closure = await fetchFragmentClosure(refs, get);
  return resolveFragments(refs, (r) => closure.get(key(r)));
}

/** The API's getter: a stored version's content becomes a StoredFragment. */
export const apiFragmentGetter: FragmentGetter = async (ref) => {
  const v = await PageStudioApi.getFragmentVersion(ref.fragment, ref.version);
  return v.content as unknown as StoredFragment;
};

type PageWithRefs = Parameters<typeof withFragments>[0];

/**
 * The page with its fragments applied. A page that names none is returned as
 * it is, at no cost. `pending` is true while they load, so a caller can hold
 * back (a checker would otherwise report the missing nodes for a moment).
 */
export function useResolvedPage<P extends PageWithRefs>(page: P): { page: P; problems: string[]; pending: boolean } {
  const refs = useMemo(() => page.app?.fragments ?? [], [page.app?.fragments]);
  const refsKey = JSON.stringify(refs);
  const query = useQuery({
    queryKey: ['page-studio', 'fragments', refsKey],
    queryFn: () => resolvePageFragments(refs, apiFragmentGetter),
    enabled: refs.length > 0,
    staleTime: 60_000, // a version is immutable, so a minute is conservative
  });
  return useMemo(() => {
    if (refs.length === 0) return { page, problems: [], pending: false };
    if (!query.data) return { page, problems: query.error ? [`Could not load the page's fragments: ${query.error instanceof Error ? query.error.message : 'unknown error'}`] : [], pending: query.isLoading };
    const merged = withFragments(page, query.data.fragment);
    return { page: merged.page, problems: [...query.data.problems, ...merged.problems], pending: false };
  }, [page, refs.length, query.data, query.error, query.isLoading]);
}

/** For code that loads a page once (not in a component): the page with its fragments applied. */
export async function applyPageFragments<P extends PageWithRefs>(page: P, get: FragmentGetter = apiFragmentGetter): Promise<{ page: P; problems: string[] }> {
  const refs = page.app?.fragments ?? [];
  if (refs.length === 0) return { page, problems: [] };
  const { fragment, problems } = await resolvePageFragments(refs, get);
  const merged = withFragments(page, fragment);
  return { page: merged.page, problems: [...problems, ...merged.problems] };
}
