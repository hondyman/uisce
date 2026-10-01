import { describe, expect, it, beforeAll } from 'vitest';
import { registerOperations } from '../../studio-core/operations/registry';
import { generateFromOperations, GenerateError } from '../../pages/page-studio/app/generateFromOperations';
import { pageFromFragment } from '../../pages/page-studio/app/fragment';
import { checkPage } from '../../pages/page-studio/app/pageChecker';

const rowFields = [
  { name: 'id', type: 'string' as const }, { name: 'name', type: 'string' as const }, { name: 'size', type: 'number' as const },
  { name: 'active', type: 'boolean' as const }, { name: 'created_at', type: 'datetime' as const }, { name: 'meta', type: 'object' as const },
];
const run = async () => ({});

beforeAll(() => registerOperations([
  { id: 'gen.list', domain: 'gen', label: 'List', kind: 'query', params: [], run, rowsPath: 'rows', rowFields },
  { id: 'gen.flat', domain: 'gen', label: 'Flat', kind: 'query', params: [], run, fields: rowFields },
  { id: 'gen.bare', domain: 'gen', label: 'Bare', kind: 'query', params: [], run },
  { id: 'gen.create', domain: 'gen', label: 'Create', kind: 'mutation', params: [], run },
  { id: 'gen.update', domain: 'gen', label: 'Update', kind: 'mutation', params: [{ name: 'id', type: 'string', required: true }], run },
  { id: 'gen.remove', domain: 'gen', label: 'Remove', kind: 'mutation', params: [{ name: 'id', type: 'string', required: true }], run },
]));

const full = { id: 'widgets', title: 'Widgets', list: 'gen.list', create: 'gen.create', update: 'gen.update', remove: 'gen.remove' };
const check = (f: ReturnType<typeof generateFromOperations>) => checkPage(pageFromFragment(f, { name: 'T', slug: 't' }) as never);

describe('generateFromOperations', () => {
  it('builds a list, drawer and edit dialog the page checker accepts', () => {
    const f = generateFromOperations(full);
    expect(f.root).toBe('widgetsRoot');
    expect(Object.keys(f.nodes).sort()).toEqual(['widgetsDialog', 'widgetsDrawer', 'widgetsRoot', 'widgetsTop']);
    expect(check(f).filter((i) => i.severity === 'error')).toEqual([]);
    expect(check(f)).toEqual([]);
  });
  it('reads rows from rowsPath and columns from rowFields; the key is shown in the list but not asked for on create', () => {
    const f = generateFromOperations(full);
    const grid = f.components.widgetsGrid.props as { rowsPath: string; columns: { id: string }[] };
    expect(grid.rowsPath).toBe('rows');
    expect(grid.columns.map((c) => c.id)).toEqual(['id', 'name', 'size', 'active', 'created_at', 'meta', 'act'].filter((x) => x !== 'meta'));
    const form = f.components.widgetsForm.props as { fields: { name: string; readOnly?: boolean; kind: string }[] };
    expect(form.fields.map((x) => x.name)).toEqual(['id', 'name', 'size', 'active', 'meta']);
    expect(form.fields.find((x) => x.name === 'id')!.readOnly).toBe(true);
    expect(form.fields.find((x) => x.name === 'active')!.kind).toBe('switch');
    expect(form.fields.some((x) => x.name === 'created_at')).toBe(false);
  });
  it('falls back to fields for a flat list and omits rowsPath', () => {
    const f = generateFromOperations({ id: 'flat', title: 'Flat', list: 'gen.flat' });
    expect((f.components.flatGrid.props as Record<string, unknown>).rowsPath).toBeUndefined();
    expect(check(f)).toEqual([]);
  });
  it('offers only what has an operation', () => {
    const f = generateFromOperations({ id: 'ro', title: 'RO', list: 'gen.list' });
    expect(f.nodes.roDialog).toBeUndefined();
    expect(f.components.roNew).toBeUndefined();
    const buttons = ((f.components.roGrid.props as { columns: { cell?: { buttons?: { label: string }[] } }[] }).columns.at(-1)!.cell!.buttons!).map((b) => b.label);
    expect(buttons).toEqual(['View']);
    expect(check(f)).toEqual([]);
  });
  it('separates Create and Save when both operations exist', () => {
    const f = generateFromOperations(full);
    const labels = ((f.nodes.widgetsDialog.props as { buttons: { label: string }[] }).buttons).map((b) => b.label);
    expect(labels).toEqual(['Cancel', 'Save', 'Create']);
  });
  it('guards delete behind a confirmation', () => {
    const f = generateFromOperations(full);
    const del = ((f.components.widgetsGrid.props as { columns: { cell?: { buttons?: { label: string; onClick: { confirm?: unknown }[] }[] } }[] }).columns.at(-1)!.cell!.buttons!).find((b) => b.label === 'Delete')!;
    expect(del.onClick[0].confirm).toBeTruthy();
  });
  it('hides fields on request', () => {
    const f = generateFromOperations({ ...full, hide: ['size'] });
    expect(JSON.stringify(f)).not.toContain('row.size');
  });
  it('refuses with a reason it can state', () => {
    expect(() => generateFromOperations({ ...full, list: 'nope' })).toThrow(/not registered/);
    expect(() => generateFromOperations({ ...full, list: 'gen.create' })).toThrow(/must be a query/);
    expect(() => generateFromOperations({ ...full, create: 'gen.list' })).toThrow(/must be a mutation/);
    expect(() => generateFromOperations({ ...full, list: 'gen.bare' })).toThrow(/does not say what its rows carry/);
    expect(() => generateFromOperations({ ...full, keyField: 'uuid' })).toThrow(/identifies a row/);
    expect(() => generateFromOperations({ ...full, id: 'a b' })).toThrow(GenerateError);
  });
});
