import type { ComponentDefinition } from '../../../types/pageStudio';
import type { Action, CellSpec, ColumnDef, FormFieldSpec } from './appModel';
import type { PageFragment, FragmentNode } from './fragment';
import { getOperation, type OperationDef, type OperationField } from '../../../studio-core/operations/registry';

/**
 * Generate a page fragment from registered operations: a list, a detail
 * drawer, and a create/edit dialog, with delete behind a confirmation. It
 * reads what the operations say about themselves (rowsPath, rowFields /
 * fields, params) and writes ordinary page configuration - the result is
 * the same JSON a hand-built page holds, open to every editor afterwards.
 * Pure: no editor surface, no network.
 */

export interface GenerateSpec {
  /** Prefix for every widget, variable and query it creates (letters and digits). */
  id: string;
  title: string;
  subtitle?: string;
  /** The query operation that lists rows. */
  list: string;
  /** Mutations, each optional: without one, that action is not offered. */
  create?: string;
  update?: string;
  remove?: string;
  /** The field that identifies a row (default "id"). */
  keyField?: string;
  /** Row fields to leave out of the list, drawer and form. */
  hide?: string[];
}

export class GenerateError extends Error {}

const label = (f: OperationField) => f.label ?? f.name.replace(/[_-]+/g, ' ').replace(/^./, (c) => c.toUpperCase());
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const cond = (field: string, operator: string, value?: unknown) => ({ type: 'condition' as const, field, operator, value });

function need(id: string, kind: OperationDef['kind'], role: string): OperationDef {
  const op = getOperation(id);
  if (!op) throw new GenerateError(`The ${role} operation "${id}" is not registered.`);
  if (op.kind !== kind) throw new GenerateError(`The ${role} operation "${id}" is a ${op.kind}; it must be a ${kind}.`);
  return op;
}

function cellFor(f: OperationField): CellSpec {
  const value = `{{row.${f.name}}}`;
  if (f.type === 'datetime') return { kind: 'datetime', value };
  if (f.type === 'boolean') return { kind: 'chip', value, variant: 'outlined' };
  if (f.type === 'array') return { kind: 'chips', value };
  return { kind: 'text', value };
}

function formFieldFor(f: OperationField, keyField: string, editing: boolean): FormFieldSpec | undefined {
  if (f.type === 'datetime') return undefined; // set by the server
  const base = { name: f.name, label: label(f), wide: false };
  if (f.name === keyField) return editing ? { ...base, kind: 'text', readOnly: true } : undefined;
  switch (f.type) {
    case 'number': return { ...base, kind: 'number' };
    case 'boolean': return { ...base, kind: 'switch' };
    case 'object': case 'array': return { ...base, kind: 'json', wide: true };
    default: return { ...base, kind: 'text' };
  }
}

/** The list, drawer, and edit dialog for a set of operations, as a fragment whose root places all three. */
export function generateFromOperations(spec: GenerateSpec): PageFragment {
  if (!/^[A-Za-z][A-Za-z0-9]*$/.test(spec.id)) throw new GenerateError('The id must be letters and digits, starting with a letter.');
  const list = need(spec.list, 'query', 'list');
  const create = spec.create ? need(spec.create, 'mutation', 'create') : undefined;
  const update = spec.update ? need(spec.update, 'mutation', 'update') : undefined;
  const remove = spec.remove ? need(spec.remove, 'mutation', 'delete') : undefined;
  const keyField = spec.keyField ?? 'id';
  // An envelope result's `fields` describe the envelope, so rows come from rowFields; a flat list may use fields.
  const fields = list.rowsPath ? list.rowFields ?? [] : list.rowFields ?? list.fields ?? [];
  const shown = fields.filter((f) => !(spec.hide ?? []).includes(f.name));
  if (!shown.length) throw new GenerateError(`"${spec.list}" does not say what its rows carry (rowFields or fields), so there is nothing to show.`);
  if (!fields.some((f) => f.name === keyField)) throw new GenerateError(`Its rows have no "${keyField}" field; say which field identifies a row (keyField).`);

  const p = spec.id;
  const v = { selected: `${p}Selected`, editing: `${p}Editing`, draft: `${p}Draft`, open: `${p}EditOpen` };
  const q = `${p}List`;
  const components: Record<string, ComponentDefinition> = {};
  const nodes: Record<string, FragmentNode> = {};
  const w = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => { components[id] = { id, type, props, ...extra }; };

  // Row actions: view always; edit and delete when there is an operation for them.
  const buttons: Record<string, unknown>[] = [{ label: 'View', icon: 'view', onClick: [set(v.selected, '{{row}}')] }];
  if (update) buttons.push({ label: 'Edit', icon: 'edit', onClick: [set(v.editing, '{{row}}'), set(v.open, true)] });
  if (remove) {
    buttons.push({
      label: 'Delete', icon: 'delete',
      onClick: [{
        kind: 'runOperation', operation: remove.id, params: { [keyField]: `{{row.${keyField}}}` }, onSuccess: [],
        confirm: { title: 'Delete', text: `Delete this ${spec.title.toLowerCase()} record?`, confirmLabel: 'Delete' },
      } as Action],
    });
  }
  const columns: ColumnDef[] = [
    ...shown.filter((f) => f.type !== 'object').map((f) => ({ id: f.name, header: label(f), cell: cellFor(f) }) as ColumnDef),
    { id: 'act', header: '', cell: { kind: 'actions', buttons } as unknown as CellSpec, align: 'right', nowrap: true },
  ];

  w(`${p}Header`, 'PageHeader', { title: spec.title, subtitle: spec.subtitle ?? list.description ?? '' }, { style: { flex: '1 1 320px' } });
  if (create) w(`${p}New`, 'ActionButton', { label: 'New', icon: 'add', variant: 'contained', onClick: [set(v.editing, null), set(v.open, true)] }, { style: { flex: '0 0 auto' } });
  w(`${p}Grid`, 'DataGrid', { query: q, ...(list.rowsPath ? { rowsPath: list.rowsPath } : {}), emptyText: 'Nothing here yet.', columns });

  // Drawer: every field of the selected row.
  w(`${p}Detail`, 'KeyValue', {
    source: `{{vars.${v.selected}}}`, columns: 1,
    items: shown.map((f) => ({ label: label(f), cell: { ...cellFor(f), value: `{{data.${f.name}}}` } })),
  });
  nodes[`${p}Drawer`] = {
    type: 'Drawer', children: [`${p}Detail`],
    props: { title: spec.title, width: 560, openWhen: cond(`vars.${v.selected}`, 'is_not_empty'), onClose: [set(v.selected, null)] },
  };

  const children = [`${p}Top`, `${p}Grid`, `${p}Drawer`];
  nodes[`${p}Top`] = { type: 'Row', children: [`${p}Header`, ...(create ? [`${p}New`] : [])], style: { alignItems: 'center' } };

  // Edit dialog: create and update share one form; the key is read-only when editing.
  const saveOp = update ?? create;
  if (create || update) {
    const editingNow = cond(`vars.${v.editing}`, 'is_not_empty');
    const formFields = shown.map((f) => formFieldFor(f, keyField, !!update)).filter((f): f is FormFieldSpec => !!f);
    w(`${p}Form`, 'Form', { variable: v.draft, fields: formFields, initFrom: `{{vars.${v.editing}}}`, seedKey: `{{vars.${v.editing}.${keyField}}}${v.open}` });
    const close = [set(v.open, false), set(v.editing, null), set(v.draft, null)];
    const save = (o: OperationDef): Action => ({ kind: 'runOperation', operation: o.id, params: { ...(o === update ? { [keyField]: `{{vars.${v.editing}.${keyField}}}` } : {}), ...Object.fromEntries(formFields.filter((f) => f.name !== keyField).map((f) => [f.name, `{{vars.${v.draft}.${f.name}}}`])) }, onSuccess: close } as Action);
    const buttonsDlg: Record<string, unknown>[] = [{ label: 'Cancel', onClick: close }];
    if (create && update) {
      buttonsDlg.push({ label: 'Save', variant: 'contained', visibleWhen: editingNow, onClick: [save(update)] });
      buttonsDlg.push({ label: 'Create', variant: 'contained', visibleWhen: cond(`vars.${v.editing}`, 'is_empty'), onClick: [save(create)] });
    } else buttonsDlg.push({ label: create ? 'Create' : 'Save', variant: 'contained', onClick: [save(saveOp!)] });
    nodes[`${p}Dialog`] = {
      type: 'Dialog', children: [`${p}Form`],
      props: { title: spec.title, maxWidth: 'sm', openWhen: cond(`vars.${v.open}`, 'is_true'), onClose: close, buttons: buttonsDlg },
    };
    children.push(`${p}Dialog`);
  }
  nodes[`${p}Root`] = { type: 'Column', children, style: { gap: '16px' } };

  return {
    root: `${p}Root`, components, nodes,
    variables: [
      { name: v.selected, description: 'The row shown in the detail drawer' },
      ...(create || update ? [{ name: v.editing, description: 'The row being edited; empty = a new one' }, { name: v.draft, description: 'The form being edited' }, { name: v.open, default: false }] : []),
    ],
    queries: [{ id: q, operation: list.id, params: {}, keepPrevious: true }],
  };
}
