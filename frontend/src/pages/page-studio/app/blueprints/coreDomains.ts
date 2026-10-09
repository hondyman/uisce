import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import type { Action, ColumnDef, ConditionNode, FormFieldSpec } from '../appModel';

/**
 * Data domains (slug core-domains): the shared domain taxonomy, as Page Studio
 * configuration. Served at /core/domains via STUDIO_ROUTES. Create and edit run in
 * a dialog; delete asks first. Core administrators make changes: the API refuses
 * other callers and the page shows the refusal. Abbreviations are not on this
 * page; they stay at /core/abbreviations, which the menu links to.
 */

const op = (
  operation: string,
  params: Record<string, unknown>,
  onSuccess: Action[] = [],
  more: Record<string, unknown> = {},
): Action => ({ kind: 'runOperation', operation, params, onSuccess, ...more } as Action);
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const cond = (field: string, operator: string, value?: unknown): ConditionNode =>
  ({ type: 'condition', field, operator, value });
const col = (id: string, header: string, cell: ColumnDef['cell'] | ColumnDef['stack'], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell: cell as ColumnDef['cell'], ...more };

const fit = { flex: '0 0 auto' };

type NodeSpec = {
  type: 'Row' | 'Column' | 'Dialog';
  children: string[];
  style?: Record<string, string>;
  props?: Record<string, unknown>;
};
const layout = (root: string, spec: Record<string, NodeSpec>): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

const OPEN = cond('vars.domainOpen', 'is_true');
const close = [set('domainOpen', false), set('domainDraft')];

function coreDomainsPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const w = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => {
    components[id] = { id, type, props, ...extra };
  };

  w(
    'hdr',
    'PageHeader',
    {
      icon: 'hub',
      title: 'Data domains',
      subtitle: 'The domain taxonomy every tenant shares. Core administrators make changes.',
    },
    { style: { flex: '1 1 320px' } },
  );

  w(
    'new_btn',
    'ActionButton',
    {
      label: 'New domain',
      icon: 'add',
      variant: 'contained',
      onClick: [set('domainEditing', ''), set('domainOpen', true)],
    },
    { style: fit },
  );

  w('grid', 'DataGrid', {
    query: 'domains',
    rowsPath: 'rows',
    emptyText: 'No domains yet. Use "New domain" to add the first one.',
    columns: [
      col('name', 'Name', [
        { kind: 'text', value: '{{row.name}}', bold: true },
        { kind: 'text', value: '{{row.slug}}', caption: true },
      ]),
      col('level', 'Level', { kind: 'text', value: '{{row.level}}' }, { align: 'right' }),
      col('parent', 'Parent', { kind: 'text', value: '{{row.parent_name}}' }),
      col('description', 'Description', { kind: 'text', value: '{{row.description}}' }),
      col(
        'act',
        '',
        {
          kind: 'actions',
          buttons: [
            {
              label: 'Edit',
              icon: 'edit',
              onClick: [set('domainEditing', '{{row.id}}'), set('domainOpen', true)],
            },
            {
              label: 'Delete',
              icon: 'delete',
              onClick: [
                op('domains.remove', { id: '{{row.id}}' }, [], {
                  confirm: {
                    title: 'Delete this domain?',
                    text: '{{row.name}} will be removed.',
                    confirmLabel: 'Delete',
                  },
                  successMessage: 'Domain deleted',
                }),
              ],
            },
          ],
        },
        { align: 'right', nowrap: true },
      ),
    ],
  });

  const fields: FormFieldSpec[] = [
    { name: 'name', kind: 'text', label: 'Name', required: true, wide: true },
    { name: 'slug', kind: 'text', label: 'Slug', helperText: 'Leave blank to generate one.', wide: true },
    {
      name: 'parent_id',
      kind: 'select',
      label: 'Parent domain',
      helperText: 'Leave empty for a top-level domain.',
      optionsFrom: { query: 'domains', rowsPath: 'rows', valueField: 'id', labelField: 'name' },
    },
    { name: 'description', kind: 'multiline', label: 'Description', wide: true },
  ];
  w('domain_form', 'Form', {
    variable: 'domainDraft',
    fields,
    initFrom: '{{queries.domainStart.data.draft}}',
    seedKey: '{{queries.domainStart.data.key}}',
  });

  const main = layout('page_root', {
    page_root: {
      type: 'Column',
      children: ['top', 'grid', 'domain_dialog'],
      style: { gap: '16px' },
    },
    top: {
      type: 'Row',
      children: ['hdr', 'new_btn'],
      style: { alignItems: 'center', gap: '12px', flexWrap: 'wrap' },
    },
    domain_dialog: {
      type: 'Dialog',
      children: ['domain_form'],
      props: {
        title: '{{queries.domainStart.data.title}}',
        maxWidth: 'sm',
        openWhen: OPEN,
        onClose: close,
        buttons: [
          { label: 'Cancel', onClick: close },
          {
            label: 'Save',
            variant: 'contained',
            disabledWhen: cond('vars.domainDraft.name', 'is_empty'),
            onClick: [op('domains.save', { draft: '{{vars.domainDraft}}' }, close, { successMessage: 'Domain saved' })],
          },
        ],
      },
    },
  });

  return {
    name: 'Data domains',
    slug: 'core-domains',
    description:
      'The shared data-domain taxonomy: levels, parents, and descriptions. Core administrators make changes. Built in Page Studio.',
    version: 1,
    isCore: true,
    status: 'published',
    layout: main,
    tabs: [],
    filterBar: layout('fb_root', { fb_root: { type: 'Column', children: [], style: { gap: '0px' } } }),
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 1200, padding: 3 },
      variables: [
        { name: 'domainOpen', default: false, description: 'Whether the domain editor is open' },
        { name: 'domainEditing', default: '', description: 'The domain being edited; empty = a new one' },
        { name: 'domainDraft', description: 'The domain being edited' },
      ],
      queries: [
        { id: 'domains', operation: 'domains.list', params: {}, keepPrevious: true },
        {
          id: 'domainStart',
          operation: 'domains.editorStart',
          params: { id: '{{vars.domainEditing}}', open: '{{vars.domainOpen}}' },
          enabledWhen: OPEN,
        },
      ],
    },
  };
}

export const coreDomainsBlueprint = () => structuredClone(coreDomainsPage());
