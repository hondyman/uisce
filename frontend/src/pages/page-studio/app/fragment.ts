import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../types/pageStudio';
import type { PageQuery, PageVariable } from './appModel';

/**
 * A page fragment: a reusable piece of a page - widgets, the layout nodes
 * that arrange them, and the variables and queries they need - that stands
 * on its own. The schedule editor dialog is one; a list with a detail drawer
 * and an edit dialog generated from an operation is another. A fragment is
 * data (the same JSON a page holds), so it can be generated, saved, shared
 * and, later, referenced by id and versioned like any core object.
 */

export type FragmentNode = {
  type: 'Row' | 'Column' | 'Dialog' | 'Drawer' | 'TabSet';
  children: string[];
  style?: Record<string, string>;
  props?: Record<string, unknown>;
};

export interface PageFragment {
  /** The layout node (or widget) to place in a page; absent when the page places each node itself. */
  root?: string;
  components: Record<string, ComponentDefinition>;
  nodes: Record<string, FragmentNode>;
  variables: PageVariable[];
  queries: PageQuery[];
}

type PageSeed = Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'>;

/** A page that is just this fragment: one tab holding its root. */
export function pageFromFragment(fragment: PageFragment, meta: { name: string; slug: string; description?: string; tabLabel?: string }): PageSeed {
  if (!fragment.root) throw new Error('A fragment needs a root to become a page.');
  const layout: PageLayout = {
    root: fragment.root,
    nodes: Object.fromEntries(Object.entries(fragment.nodes).map(([id, n]) => [id, { id, ...n }])) as PageLayout['nodes'],
  };
  return {
    name: meta.name,
    slug: meta.slug,
    description: meta.description ?? '',
    layout,
    tabs: [{ id: 'main', label: meta.tabLabel ?? meta.name, layout }],
    components: structuredClone(fragment.components),
    dataSources: [],
    status: 'draft',
    app: { chrome: 'none', surface: { maxWidth: 1400, padding: 3 }, variables: structuredClone(fragment.variables), queries: structuredClone(fragment.queries) },
  } as PageSeed;
}

/** What stops a fragment being merged into a page: names it would take that the page already uses. */
export function mergeConflicts(page: Pick<CorePageDefinition, 'components' | 'app'>, fragment: PageFragment): string[] {
  const out: string[] = [];
  for (const id of Object.keys(fragment.components)) if (page.components?.[id]) out.push(`a widget "${id}"`);
  for (const v of fragment.variables) if (page.app?.variables?.some((x) => x.name === v.name)) out.push(`a variable "${v.name}"`);
  for (const q of fragment.queries) if (page.app?.queries?.some((x) => x.id === q.id)) out.push(`a query "${q.id}"`);
  return out;
}
