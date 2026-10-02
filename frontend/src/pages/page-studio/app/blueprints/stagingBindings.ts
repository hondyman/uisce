import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, ColumnDef, ConditionNode } from '../appModel';

/**
 * Staging bindings (formerly the hand-built StagingBindingsPage.tsx) as a
 * Page Studio page: which staging column each business object field is read
 * from, maker-checker. Bindings, approvals (with a waiting count) and
 * history tabs; proposals are made in a dialog built from studio blocks - a
 * form whose field -> column map is a `map` field with suggestions.
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
function layout(root: string, spec: Record<string, { type: 'Row' | 'Column' | 'Dialog'; children: string[]; style?: Record<string, string>; props?: Record<string, unknown> }>): PageLayout {
  return { root, nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])) };
}

const STATUS = { pending: 'warning', applied: 'success', rejected: 'error', withdrawn: 'default' } as const;
const openEditor = (binding: unknown, title: string): Action[] => [set('editing', binding), set('editorTitle', title), set('sbHints'), set('editorOpen', true)];

// --- page-wide ---------------------------------------------------------------

widget('hdr', 'PageHeader', { icon: 'link', title: 'stagingBindings.title', subtitle: 'stagingBindings.subtitle' }, { style: { flex: '1 1 320px' } });
widget('new_btn', 'ActionButton', { label: 'stagingBindings.new', icon: 'add', variant: 'contained', onClick: openEditor(null, 'stagingBindings.editor.newTitle') }, { style: fit });

// --- The binding editor: which staging column each field is read from ---------------
const EDITING = cond('vars.editing.id', 'is_not_empty');
const closeEditor = [set('editorOpen', false), set('editing'), set('sbHints')];
widget('sb_maker_checker', 'AlertBanner', { severity: 'info', text: 'stagingBindings.editor.makerChecker' });
widget('sb_form', 'Form', {
  variable: 'sbDraft', initFrom: '{{vars.editing}}', columns: 2,
  fields: [
    { name: 'bo_key', kind: 'select', label: 'stagingBindings.columns.businessObject', readOnlyWhen: EDITING,
      optionsFrom: { query: 'sbBos', valueField: 'name', labelField: 'label' } },
    { name: 'staging_table', kind: 'select', label: 'stagingBindings.columns.stagingTable', readOnlyWhen: EDITING,
      optionsFrom: { query: 'sbTables', valueField: 'table', labelField: 'table' } },
    {
      name: 'fields', kind: 'map', label: '', wide: true, resetOn: ['bo_key', 'staging_table'],
      visibleWhen: { type: 'group', operator: 'AND', conditions: [cond('form.bo_key', 'is_not_empty'), cond('form.staging_table', 'is_not_empty')] },
      optionsFrom: { query: 'sbColumns', valueField: 'name', labelField: 'name' },
      map: {
        rows: '{{queries.sbRows.data.rows}}', hints: '{{vars.sbHints}}', placeholder: 'stagingBindings.editor.notBound',
        keyHeader: 'stagingBindings.editor.field', valueHeader: 'stagingBindings.editor.column',
        groups: [
          {
            id: 'fields', headers: true,
            title: { t: 'stagingBindings.editor.mapping', params: { bound: '{{map.bound}}', total: '{{map.total}}' } },
            action: {
              label: 'stagingBindings.editor.suggest',
              onClick: [{ kind: 'runOperation', operation: 'stagingBindings.suggest', params: { draft: '{{vars.sbDraft}}' },
                onSuccess: [set('sbDraft', '{{result.draft}}'), set('sbHints', '{{result.hints}}')] }],
            },
          },
          { id: 'keys', title: 'stagingBindings.editor.masteringKeys', help: 'stagingBindings.editor.masteringKeysHelp' },
        ],
        add: [
          {
            prefix: 'id:', group: 'keys', code: true, label: 'stagingBindings.editor.addIdentifier', inputLabel: 'stagingBindings.editor.identifierType',
            options: ['ISIN', 'CUSIP', 'SEDOL', 'LEI', 'FIGI', 'BLOOMBERG_ID', 'TICKER', 'RIC', 'PROVIDER_CODE', 'INTERNAL'],
            rowLabel: { t: 'stagingBindings.editor.identifier', params: { type: '{{item.type}}' } }, removeLabel: 'stagingBindings.editor.remove',
          },
          {
            prefix: 'value:', group: 'keys', code: true, label: 'stagingBindings.editor.addPriceColumn', inputLabel: 'stagingBindings.editor.priceType',
            helperText: 'stagingBindings.editor.priceTypeHelp', visibleWhen: cond('queries.sbRows.data.offers_values', 'is_true'),
            options: ['LAST', 'OFFICIAL_CLOSE', 'BID', 'ASK', 'MID', 'NAV', 'EVALUATED', 'CLEAN_PRICE', 'DIRTY_PRICE', 'SETTLEMENT'],
            rowLabel: { t: 'stagingBindings.editor.seriesValue', params: { type: '{{item.type}}' } }, removeLabel: 'stagingBindings.editor.remove',
          },
        ],
      },
    },
    { name: 'reason', kind: 'multiline', label: 'stagingBindings.editor.reason', helperText: 'stagingBindings.editor.reasonHelp', wide: true },
  ],
});

const filterBar = layout('top_root', {
  top_root: { type: 'Column', children: ['top_header', 'editor'], style: { gap: '8px' } },
  editor: {
    type: 'Dialog', children: ['editor_body'], props: {
      title: { t: '{{vars.editorTitle}}' }, maxWidth: 'md',
      openWhen: cond('vars.editorOpen', 'is_true'), onClose: closeEditor,
      buttons: [
        { label: 'stagingBindings.cancel', onClick: closeEditor },
        {
          label: 'stagingBindings.editor.propose', variant: 'contained',
          disabledWhen: { type: 'group', operator: 'OR', conditions: [
            cond('vars.sbDraft.bo_key', 'is_empty'), cond('vars.sbDraft.staging_table', 'is_empty'), cond('vars.sbDraft.fields', 'is_empty'),
          ] },
          onClick: [{ kind: 'runOperation', operation: 'stagingBindings.propose', params: { draft: '{{vars.sbDraft}}' }, onSuccess: closeEditor }],
        },
      ],
    },
  },
  editor_body: { type: 'Column', children: ['sb_maker_checker', 'sb_form'], style: { gap: '16px' } },
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
          { label: 'stagingBindings.proposeChange', onClick: openEditor('{{row}}', 'stagingBindings.editor.editTitle') },
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
        { name: 'editorTitle', default: 'stagingBindings.editor.newTitle' },
        { name: 'sbDraft', description: 'The proposal: bo_key, staging_table, fields (field -> column), reason' },
        { name: 'sbHints', description: 'Suggested columns per field (from Suggest)' },
      ],
      queries: [
        { id: 'bindings', operation: 'stagingBindings.list', params: {} },
        { id: 'pending', operation: 'stagingBindings.changes', params: { status: 'pending' } },
        { id: 'sbBos', operation: 'stagingBindings.businessObjects', params: {}, enabledWhen: cond('vars.editorOpen', 'is_true') },
        { id: 'sbTables', operation: 'stagingBindings.stagingTables', params: {}, enabledWhen: cond('vars.editorOpen', 'is_true') },
        { id: 'sbRows', operation: 'stagingBindings.editorRows', params: { bo_key: '{{vars.sbDraft.bo_key}}' }, enabledWhen: cond('vars.editorOpen', 'is_true') },
        { id: 'sbColumns', operation: 'stagingBindings.tableColumns', params: { table: '{{vars.sbDraft.staging_table}}' }, enabledWhen: cond('vars.editorOpen', 'is_true') },
        { id: 'history', operation: 'stagingBindings.changes', params: { status: '' }, enabledWhen: cond('vars.tab', 'equals', 'history') },
      ],
    },
  });
}
