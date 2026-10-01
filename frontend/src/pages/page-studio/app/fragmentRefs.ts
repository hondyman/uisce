import type { PageFragment } from './fragment';

/**
 * Fragments by reference. A page (or another fragment) names a stored
 * fragment by slug and version rather than carrying a copy; resolving walks
 * the references and flattens them into one fragment, so the runtime and the
 * page checker see ordinary widgets, nodes, variables and queries. Versions
 * are exact (a fragment version is immutable, like a core page version), so a
 * page never changes under its author. Nesting is capped at MAX_FRAGMENT_DEPTH.
 * Pure: the caller supplies `lookup`.
 */

export const MAX_FRAGMENT_DEPTH = 2;

export interface FragmentRef {
  fragment: string;
  version: number;
}

/** A fragment as stored: its content plus the fragments it is built from. */
export interface StoredFragment extends PageFragment {
  uses?: FragmentRef[];
}

export type FragmentLookup = (ref: FragmentRef) => StoredFragment | undefined;

export interface Resolved {
  fragment: PageFragment;
  problems: string[];
}

const key = (r: FragmentRef) => `${r.fragment}@${r.version}`;

const empty = (): PageFragment => ({ components: {}, nodes: {}, variables: [], queries: [] });

/** Add `from` into `into`; a name both use is a problem, not a silent overwrite. */
function absorb(into: PageFragment, from: PageFragment, via: string, problems: string[]) {
  for (const [id, c] of Object.entries(from.components)) {
    if (into.components[id]) problems.push(`${via}: a widget "${id}" is already defined.`);
    else into.components[id] = c;
  }
  for (const [id, n] of Object.entries(from.nodes)) {
    if (into.nodes[id]) problems.push(`${via}: a layout node "${id}" is already defined.`);
    else into.nodes[id] = n;
  }
  for (const v of from.variables) {
    if (into.variables.some((x) => x.name === v.name)) problems.push(`${via}: a variable "${v.name}" is already defined.`);
    else into.variables.push(v);
  }
  for (const q of from.queries) {
    if (into.queries.some((x) => x.id === q.id)) problems.push(`${via}: a query "${q.id}" is already defined.`);
    else into.queries.push(q);
  }
}

function expand(ref: FragmentRef, lookup: FragmentLookup, depth: number, trail: string[], problems: string[]): PageFragment | undefined {
  const k = key(ref);
  const found = lookup(ref);
  if (!found) { problems.push(`The fragment "${ref.fragment}" version ${ref.version} does not exist.`); return undefined; }
  if (trail.includes(k)) { problems.push(`The fragment "${k}" uses itself (${[...trail, k].join(' → ')}).`); return undefined; }
  const out: PageFragment = { root: found.root, components: { ...found.components }, nodes: { ...found.nodes }, variables: [...found.variables], queries: [...found.queries] };
  for (const inner of found.uses ?? []) {
    if (depth >= MAX_FRAGMENT_DEPTH) { problems.push(`"${k}" uses "${key(inner)}", nested deeper than ${MAX_FRAGMENT_DEPTH} levels.`); continue; }
    const got = expand(inner, lookup, depth + 1, [...trail, k], problems);
    if (got) absorb(out, got, `"${k}" and "${key(inner)}"`, problems);
  }
  return out;
}

/** Flatten references into one fragment (depth 1 = referenced from a page, 2 = referenced from a fragment). */
export function resolveFragments(refs: FragmentRef[], lookup: FragmentLookup): Resolved {
  const problems: string[] = [];
  const merged = empty();
  for (const ref of refs) {
    const got = expand(ref, lookup, 1, [], problems);
    if (got) absorb(merged, got, `"${key(ref)}"`, problems);
  }
  return { fragment: merged, problems };
}
