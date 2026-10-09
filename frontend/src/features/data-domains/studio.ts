import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { deleteDomain, listDomains, saveDomain, type DataDomainRecord } from './api';

/**
 * Data domains Page Studio operations (the core-domains page). Mutations that
 * the API refuses (a non-core administrator, a missing parent) surface as the
 * operation's error; the page shows it.
 */

const DOMAIN = 'domains';

/** A new slug: the API keeps slugs unique, so names alone would collide. */
function newSlug(): string {
  return typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : `domain-${Date.now().toString(36)}`;
}

/** True when `candidate` is `node` or sits beneath it. Walks parent links with a guard against loops. */
function isWithin(all: DataDomainRecord[], candidate: string, node: string): boolean {
  const byId = new Map(all.map((d) => [d.id, d]));
  const seen = new Set<string>();
  let cur: string | null = candidate;
  while (cur && !seen.has(cur)) {
    if (cur === node) return true;
    seen.add(cur);
    cur = byId.get(cur)?.parent_id ?? null;
  }
  return false;
}

const operations: OperationDef[] = [
  {
    id: 'domains.list',
    domain: DOMAIN,
    kind: 'query',
    label: 'List data domains',
    description: 'Every data domain, with its parent name and level.',
    params: [],
    rowsPath: 'rows',
    rowFields: [
      { name: 'id', type: 'string' },
      { name: 'name', type: 'string' },
      { name: 'slug', type: 'string' },
      { name: 'level', type: 'number' },
      { name: 'parent_id', type: 'string' },
      { name: 'parent_name', type: 'string' },
      { name: 'description', type: 'string' },
    ],
    fields: [{ name: 'rows' }, { name: 'total', type: 'number' }],
    run: async () => {
      const all = await listDomains();
      const byId = new Map(all.map((d) => [d.id, d]));
      const rows = all.map((d) => ({
        ...d,
        parent_name: d.parent_id ? byId.get(d.parent_id)?.name ?? d.parent_id : '',
      }));
      return { rows, total: rows.length };
    },
  },
  {
    id: 'domains.editorStart',
    domain: DOMAIN,
    kind: 'query',
    label: 'Start the domain editor',
    description: 'The title and draft for the domain editor: a blank draft for a new domain, the stored values for an existing one.',
    params: [
      { name: 'id', type: 'string', required: false, description: 'The domain to edit; empty for a new one.' },
      { name: 'open', type: 'boolean', required: false, description: 'Re-runs the query each time the editor opens.' },
    ],
    fields: [{ name: 'title' }, { name: 'key' }, { name: 'draft', type: 'object' }],
    run: async (params) => {
      const id = params.id ? String(params.id) : '';
      // The key changes on every run, so the form re-seeds each time the editor opens.
      if (!id) {
        return {
          title: 'New domain',
          key: `new:${Date.now()}`,
          draft: { name: '', slug: '', parent_id: null, description: '' },
        };
      }
      const d = (await listDomains()).find((r) => r.id === id);
      if (!d) throw new Error('This domain no longer exists.');
      return {
        title: `Edit ${d.name}`,
        key: `${d.id}:${Date.now()}`,
        draft: { id: d.id, name: d.name, slug: d.slug, parent_id: d.parent_id, description: d.description },
      };
    },
  },
  {
    id: 'domains.save',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Save data domain',
    description: 'Creates the domain, or updates it when the draft has an id. The level follows from the parent.',
    params: [{ name: 'draft', type: 'object', required: true }],
    run: async (params) => {
      const draft = (params.draft ?? {}) as {
        id?: unknown;
        name?: unknown;
        slug?: unknown;
        parent_id?: unknown;
        description?: unknown;
      };
      const name = String(draft.name ?? '').trim();
      if (!name) throw new Error('A domain needs a name.');
      const id = draft.id ? String(draft.id) : undefined;
      const parentId = draft.parent_id ? String(draft.parent_id) : null;

      const all = await listDomains();
      if (id && parentId) {
        if (parentId === id) throw new Error('A domain cannot be its own parent.');
        if (isWithin(all, parentId, id)) throw new Error('A domain cannot move under one of its own children.');
      }
      const parent = parentId ? all.find((d) => d.id === parentId) : undefined;
      if (parentId && !parent) throw new Error('The parent domain no longer exists.');
      const existing = id ? all.find((d) => d.id === id) : undefined;

      await saveDomain({
        id,
        name,
        slug: String(draft.slug ?? '').trim() || existing?.slug || newSlug(),
        parent_id: parentId,
        level: parent ? parent.level + 1 : 1,
        description: String(draft.description ?? ''),
      });
      return { ok: true };
    },
  },
  {
    id: 'domains.remove',
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Delete data domain',
    description: 'Deletes a domain.',
    params: [{ name: 'id', type: 'string', required: true }],
    run: async (params) => {
      await deleteDomain(String(params.id));
      return { ok: true };
    },
  },
];

registerOperations(operations);
