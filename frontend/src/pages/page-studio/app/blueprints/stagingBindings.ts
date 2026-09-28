import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, ColumnDef, ConditionNode } from '../appModel';

/**
 * Staging bindings (features/staging-bindings/StagingBindingsPage.tsx) as a
 * Page Studio page: which staging column each business object field is read
 * from, maker-checker. Bindings, approvals (with a waiting count) and
 * history tabs; proposals open the domain's own editor.
 */

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const byState = (state: string) => cond('row.state', 'equals', state);
const fit = { flex: '0 0 auto' };

const components: Record<string, ComponentDefinition> = {};
function widget(id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}): string {
  components[id] = { id, type, props, ...extra };
  return id;
}
function layout(root: string, spec: Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>): PageLayout {
  return { root, nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])) };
}

const STATUS = { pending: 'warning', applied: 'success', rejected: 'error', withdrawn: 'default' } as const;
const openEditor = (binding: unknown): Action[] => [set('editing', binding), set('editorOpen', true)];

// --- page-wide ---------------------------------------------------------------

widget('hdr', 'PageHeader', { icon: 'link', title: 'stagingBindings.title', subtitle: 'stagingBindings.subtitle' }, { style: { flex: '1 1 320px' } });
widget('new_btn', 'ActionButton', { label: 'stagingBindings.new', icon: 'add', variant: 'contained', onClick: openEditor(null) }, { style: fit });
widget('editor', 'DomainComponent', {
  component: 'stagingBindings.BindingEditor',
  inputs: { open: '{{vars.editorOpen}}', binding: '{{vars.editing}}' },
  events: { close: [set('editorOpen', false), set('editing')] },
}, { style: fit });

const filterBar = layout('top_root', {
  top_root: { type: 'Column', children: ['top_header', 'editor'], style: { gap: '8px' } },
  top_header: { type: 'Row', children: ['hdr', 'new_btn'], style: { alignItems: 'center' } },
});

// --- Bindings ------------------------------------------------------------------

widget('bindings_grid', 'DataGrid', {
  query: 'bindings', emptyText: 'stagingBindings.empty',
  columns: [
    { id: 'table', header: 'stagingBindings.columns.stagingTable', cell: { kind: 'text', value: '{{row.staging_table}}', mono: true } },
    { id: 'bo', header: 'stagingBindings.columns.businessObject', cell: { kind: 'twoLine', primary: '{{row.bo_name}}', secondary: '{{row.bo_key}}', secondaryMono: true } },
    {
      id: 'fields', header: 'stagingBindings.columns.fields', tooltip: '{{row.fields_tooltip}}',
      cell: { kind: 'chip', label: { t: 'stagingBindings.fieldCount', params: { count: '{{row.field_count}}' } } },
    },
    {
      id: 'origin', header: 'stagingBindings.columns.origin', stackDirection: 'row', stack: [
        { kind: 'chip', value: '{{row.origin}}', labelKey: 'stagingBindings.origin.', colorMap: { core: 'primary', '*': 'default' }, variant: 'outlined' },
        { kind: 'chip', label: 'stagingBindings.changePending', color: 'warning', visibleWhen: cond('row.pending', 'is_true') },
      ],
    },
    {
      id: 'updated', header: 'stagingBindings.columns.updated', nowrap: true, stackDirection: 'row', stack: [
        { kind: 'datetime', value: '{{row.updated_at}}' },
        { kind: 'text', value: '· v{{row.version}}', caption: true },
      ],
    },
    {
      id: 'act', align: 'right', nowrap: true, cell: {
        kind: 'actions', buttons: [
          { label: 'stagingBindings.proposeChange', onClick: openEditor('{{row}}') },
          {
            label: 'stagingBindings.proposeDelete', color: 'error', visibleWhen: cond('row.inherited', 'is_false'),
            onClick: [{
              kind: 'runOperation', operation: 'stagingBindings.proposeDelete',
              params: { bo_key: '{{row.bo_key}}', staging_table: '{{row.staging_table}}' },
              confirm: { title: 'stagingBindings.proposeDelete', text: { t: 'stagingBindings.confirmDelete', params: { table: '{{row.staging_table}}', bo: '{{row.bo_key}}' } } },
              onSuccess: [set('tab', 'approvals')],
            }],
          },
        ],
      },
    },
  ] satisfies ColumnDef[],
});

// --- Approvals -----------------------------------------------------------------

const decide = (operation: string): Action[] => [{ kind: 'runOperation', operation, params: { id: '{{row.id}}', comment: '{{rowState.comment}}' } }];
widget('approvals_grid', 'DataGrid', {
  query: 'pending', emptyText: 'stagingBindings.approvals.none',
  columns: [
    {
      id: 'binding', header: 'stagingBindings.columns.stagingTable', stack: [
        { kind: 'text', value: '{{row.staging_table}}', mono: true, bold: true },
        { kind: 'text', value: '→ {{row.bo_key}}', caption: true },
      ],
    },
    {
      id: 'change', header: 'stagingBindings.history.change', stack: [
        { kind: 'text', text: 'stagingBindings.approvals.deleteBinding', color: 'error', visibleWhen: cond('row.is_delete', 'is_true') },
        { kind: 'chips', value: '{{row.diff}}' },
        { kind: 'text', text: 'stagingBindings.approvals.noChange', caption: true, visibleWhen: { type: 'group', operator: 'AND', conditions: [cond('row.is_delete', 'is_false'), cond('row.diff_count', 'equals', 0)] } },
        { kind: 'text', value: '“{{row.reason}}”', caption: true, visibleWhen: cond('row.reason', 'is_not_empty') },
      ],
    },
    {
      id: 'proposed', header: 'stagingBindings.history.maker', nowrap: true, stack: [
        { kind: 'text', value: '{{row.requested_by_label}}' },
        { kind: 'datetime', value: '{{row.requested_at}}', caption: true },
      ],
    },
    {
      id: 'decide', align: 'right', nowrap: true, minWidth: 240, stack: [
        { kind: 'text', text: 'stagingBindings.approvals.yours', caption: true, visibleWhen: byState('mine') },
        { kind: 'input', name: 'comment', placeholder: 'stagingBindings.approvals.comment', visibleWhen: byState('can_vote') },
        {
          kind: 'actions', buttons: [
            { label: 'stagingBindings.approvals.withdraw', color: 'inherit', visibleWhen: byState('mine'), onClick: [{ kind: 'runOperation', operation: 'stagingBindings.withdraw', params: { id: '{{row.id}}' } }] },
            { label: 'stagingBindings.approvals.approve', variant: 'contained', visibleWhen: byState('can_vote'), onClick: decide('stagingBindings.approve') },
            { label: 'stagingBindings.approvals.reject', color: 'error', visibleWhen: byState('can_vote'), onClick: decide('stagingBindings.reject') },
          ],
        },
      ],
    },
  ] satisfies ColumnDef[],
});

// --- History -------------------------------------------------------------------

widget('history_grid', 'DataGrid', {
  query: 'history',
  columns: [
    { id: 'when', header: 'stagingBindings.history.when', nowrap: true, cell: { kind: 'datetime', value: '{{row.requested_at}}' } },
    {
      id: 'table', header: 'stagingBindings.columns.stagingTable', stack: [
        { kind: 'text', value: '{{row.staging_table}}', mono: true },
        { kind: 'text', value: '→ {{row.bo_key}}', caption: true },
      ],
    },
    {
      id: 'change', header: 'stagingBindings.history.change', stack: [
        { kind: 'text', text: 'stagingBindings.history.delete', visibleWhen: cond('row.is_delete', 'is_true') },
        { kind: 'text', text: { t: 'stagingBindings.fieldCount', params: { count: '{{row.field_count}}' } }, visibleWhen: cond('row.is_delete', 'is_false') },
      ],
    },
    { id: 'status', header: 'stagingBindings.history.status', cell: { kind: 'chip', value: '{{row.status}}', labelKey: 'stagingBindings.status.', colorMap: STATUS } },
    { id: 'maker', header: 'stagingBindings.history.maker', field: 'requested_by_label' },
    {
      id: 'checker', header: 'stagingBindings.history.checker', stack: [
        { kind: 'text', value: '{{row.reviewed_by_label}}' },
        { kind: 'text', value: '“{{row.review_comment}}”', caption: true, visibleWhen: cond('row.review_comment', 'is_not_empty') },
      ],
    },
  ] satisfies ColumnDef[],
});

const one = (id: string, child: string): PageLayout => layout(`${id}_root`, { [`${id}_root`]: { type: 'Column', children: [child] } });
const tabs: PageTab[] = [
  { id: 'bindings', label: 'stagingBindings.tabs.bindings', layout: one('bindings', 'bindings_grid') },
  { id: 'approvals', label: 'stagingBindings.tabs.approvals', badge: '{{queries.pending.data.length}}', layout: one('approvals', 'approvals_grid') },
  { id: 'history', label: 'stagingBindings.tabs.history', layout: one('history', 'history_grid') },
];

export function stagingBindingsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return structuredClone({
    name: 'Staging bindings',
    slug: 'staging-bindings',
    description: 'Which staging column each business object field is read from - the one vendor-to-field mapping for mastering. Changes need a second administrator. Built in Page Studio.',
    layout: tabs[0].layout,
    tabs,
    filterBar,
    components,
    dataSources: [],
    status: 'draft' as const,
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 1400, padding: 3 },
      tabVariable: 'tab',
      variables: [
        { name: 'tab', default: 'bindings', url: true },
        { name: 'editorOpen', default: false },
        { name: 'editing', description: 'The binding being changed; empty = a new one' },
      ],
      queries: [
        { id: 'bindings', operation: 'stagingBindings.list', params: {} },
        { id: 'pending', operation: 'stagingBindings.changes', params: { status: 'pending' } },
        { id: 'history', operation: 'stagingBindings.changes', params: { status: '' }, enabledWhen: cond('vars.tab', 'equals', 'history') },
      ],
    },
  });
}
