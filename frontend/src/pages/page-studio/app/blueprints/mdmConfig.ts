import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, CellSpec, ColumnDef, ConditionNode } from '../appModel';

/**
 * Mastering configuration as Page Studio pages: the Vendor registry, Source
 * hierarchy and Match rules (features/mastering/configStudio.tsx). One
 * builder: header, entity picker, the rows as mastering reads them (the
 * gold copy's, read-only, and the tenant's own overrides), approvals and
 * history. Every change is proposed and applied only when a second
 * administrator approves.
 */

type Kind = 'source_system' | 'source_priority' | 'match_rule';

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });
const all = (...conditions: ConditionNode[]): ConditionNode => ({ type: 'group', operator: 'AND', conditions });
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const byState = (state: string) => cond('row.state', 'equals', state);
const CAN_EDIT = cond('queries.table.data.can_edit', 'is_true');
const fit = { flex: '0 0 auto' };

type Spec = Record<string, { type: 'Row' | 'Column' | 'Dialog'; children: string[]; style?: Record<string, string>; props?: Record<string, unknown> }>;
const layout = (root: string, spec: Spec): PageLayout => ({ root, nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])) });
const col = (id: string, header: string, cell: CellSpec | CellSpec[], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell, ...more };

const STATE = { own: 'success', core: 'primary', overridden: 'default' } as const;
const STATUS = { pending: 'warning', applied: 'success', rejected: 'error', withdrawn: 'default' } as const;

const openEditor = (mode: 'new' | 'edit' | 'override', row: unknown = null): Action[] => [set('editRow', row), set('editorMode', mode), set('editorOpen', true)];

/** Where a row came from, and what is waiting on it. */
const originCol = col('origin', 'Origin', [
  { kind: 'chip', value: '{{row.state}}', label: '{{row.state_label}}', colorMap: STATE, variant: 'outlined' },
  { kind: 'chip', label: 'Change pending', color: 'warning', visibleWhen: cond('row.pending', 'is_true') },
  { kind: 'chip', label: 'Inactive', color: 'default', visibleWhen: cond('row.active', 'is_false') },
], { stackDirection: 'row', nowrap: true });

function actionsCol(kind: Kind, entity: string): ColumnDef {
  return col('act', '', {
    kind: 'actions', buttons: [
      { label: 'Propose change', visibleWhen: all(CAN_EDIT, byState('own')), onClick: openEditor('edit', '{{row}}') },
      {
        label: 'Propose removal', color: 'error', visibleWhen: all(CAN_EDIT, byState('own')), onClick: [{
          kind: 'runOperation', operation: 'mdmConfig.proposeRemove', params: { kind, entity, id: '{{row.id}}', reason: '{{form.reason}}' },
          form: {
            title: 'Propose removing this row', intro: 'Another administrator must approve before it is removed.',
            fields: [{ name: 'reason', kind: 'multiline', label: 'Reason (shown to the approver)' }], submitLabel: 'Send for approval',
          },
          successMessage: 'Sent for approval', onSuccess: [set('tab', 'approvals')],
        }],
      },
      { label: 'Override', visibleWhen: all(CAN_EDIT, byState('core')), onClick: openEditor('override', '{{row}}') },
    ],
  }, { align: 'right', nowrap: true });
}

function approvalsGrid(): Record<string, unknown> {
  const decide = (operation: string): Action[] => [{ kind: 'runOperation', operation, params: { id: '{{row.id}}', comment: '{{rowState.comment}}' } }];
  return {
    query: 'pending', emptyText: 'Nothing waiting for approval.',
    columns: [
      col('subject', 'Row', { kind: 'twoLine', primary: '{{row.subject}}', secondary: '{{row.what}}' }),
      col('change', 'Change', [
        { kind: 'text', text: 'Remove this row', color: 'error', visibleWhen: cond('row.action', 'equals', 'delete') },
        { kind: 'chips', value: '{{row.lines}}' },
        { kind: 'text', value: '“{{row.reason}}”', caption: true, visibleWhen: cond('row.reason', 'is_not_empty') },
      ]),
      col('proposed', 'Proposed by', [{ kind: 'text', value: '{{row.requested_by_label}}' }, { kind: 'datetime', value: '{{row.requested_at}}', caption: true }], { nowrap: true }),
      col('decide', '', [
        { kind: 'text', text: 'Your proposal - another administrator approves it.', caption: true, visibleWhen: byState('mine') },
        { kind: 'text', text: 'Waiting for an administrator.', caption: true, visibleWhen: byState('waiting') },
        { kind: 'input', name: 'comment', placeholder: 'Comment', visibleWhen: byState('can_decide') },
        {
          kind: 'actions', buttons: [
            { label: 'Withdraw', color: 'inherit', visibleWhen: byState('mine'), onClick: decide('mdmConfig.withdraw') },
            { label: 'Approve', variant: 'contained', visibleWhen: byState('can_decide'), onClick: decide('mdmConfig.approve') },
            { label: 'Reject', color: 'error', visibleWhen: byState('can_decide'), onClick: decide('mdmConfig.reject') },
          ],
        },
      ], { align: 'right', nowrap: true, minWidth: 240 }),
    ] satisfies ColumnDef[],
  };
}

function historyGrid(): Record<string, unknown> {
  return {
    query: 'history', emptyText: 'No changes yet.',
    columns: [
      col('when', 'When', { kind: 'datetime', value: '{{row.requested_at}}' }, { nowrap: true }),
      col('subject', 'Row', { kind: 'twoLine', primary: '{{row.subject}}', secondary: '{{row.what}}' }),
      col('change', 'Change', [
        { kind: 'text', text: 'Removed', visibleWhen: cond('row.action', 'equals', 'delete') },
        { kind: 'chips', value: '{{row.lines}}' },
      ]),
      col('status', 'Status', { kind: 'chip', value: '{{row.status}}', colorMap: STATUS }),
      col('maker', 'Proposed by', { kind: 'text', value: '{{row.requested_by_label}}' }),
      col('checker', 'Decided by', [
        { kind: 'text', value: '{{row.reviewed_by_label}}' },
        { kind: 'text', value: '“{{row.review_comment}}”', caption: true, visibleWhen: cond('row.review_comment', 'is_not_empty') },
      ]),
    ] satisfies ColumnDef[],
  };
}

interface PageSpec {
  kind: Kind;
  slug: string;
  name: string;
  title: string;
  subtitle: string;
  description: string;
  icon: string;
  perEntity: boolean;
  columns: ColumnDef[];
  newLabel: string;
}

function configPage(s: PageSpec): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const entity = s.perEntity ? '{{vars.entity}}' : '';
  const components: Record<string, ComponentDefinition> = {};
  const widget = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => {
    components[id] = { id, type, props, ...extra };
    return id;
  };
  widget('hdr', 'PageHeader', { icon: s.icon, title: s.title, subtitle: s.subtitle }, { style: { flex: '1 1 320px' } });
  if (s.perEntity) {
    widget('entity_select', 'VariableSelect', {
      variable: 'entity', label: 'Entity', minWidth: 200, optionsFrom: { query: 'entities', valueField: 'entity', labelField: 'display_name' },
    }, { style: fit });
  }
  widget('new_btn', 'ActionButton', { label: s.newLabel, icon: 'add', variant: 'contained', onClick: openEditor('new') }, { style: fit, visibleWhen: CAN_EDIT });
  widget('read_only', 'AlertBanner', {
    severity: 'info', text: 'Read-only: only administrators propose configuration changes, and a second administrator approves them.',
  }, { visibleWhen: cond('queries.table.data.can_edit', 'is_false') });
  // The row editor: a dialog whose form the domain shapes from the table's columns.
  widget('editor_override', 'AlertBanner', {
    severity: 'info', text: 'This row comes from the gold copy. Your version replaces it in your environment only; the gold copy is unchanged.',
  }, { visibleWhen: cond('queries.editor.data.override', 'is_true') });
  widget('editor_form', 'Form', { variable: 'editDraft', fieldsFrom: '{{queries.editor.data.fields}}', initFrom: '{{queries.editor.data.initial}}' });
  widget('editor_note', 'TextBlock', { text: 'Nothing changes until another administrator approves.', variant: 'caption', color: 'text.secondary' });
  widget('grid', 'DataGrid', {
    query: 'table', rowsPath: 'rows', emptyText: 'No rows.',
    columns: [...s.columns, originCol, actionsCol(s.kind, entity)],
  });
  widget('approvals_grid', 'DataGrid', approvalsGrid());
  widget('history_grid', 'DataGrid', historyGrid());

  const header = ['hdr', ...(s.perEntity ? ['entity_select'] : []), 'new_btn'];
  const filterBar = layout('top_root', {
    top_root: { type: 'Column', children: ['top_header', 'read_only', 'editor'], style: { gap: '12px' } },
    editor: {
      type: 'Dialog', children: ['editor_body'], props: {
        title: '{{queries.editor.data.title}}', maxWidth: 'sm',
        openWhen: cond('vars.editorOpen', 'is_true'), onClose: [set('editorOpen', false)],
        buttons: [
          { label: 'Cancel', onClick: [set('editorOpen', false)] },
          {
            label: 'Send for approval', variant: 'contained', disabledWhen: cond('queries.editor.data', 'is_empty'),
            onClick: [{
              kind: 'runOperation', operation: 'mdmConfig.propose',
              params: { kind: s.kind, entity, mode: '{{vars.editorMode}}', id: '{{vars.editRow.id}}', values: '{{vars.editDraft}}', original: '{{queries.editor.data.initial}}' },
              onSuccess: [set('editorOpen', false), set('tab', 'approvals')],
            }],
          },
        ],
      },
    },
    editor_body: { type: 'Column', children: ['editor_override', 'editor_form', 'editor_note'], style: { gap: '16px' } },
    top_header: { type: 'Row', children: header, style: { alignItems: 'center' } },
  });
  const one = (id: string, child: string) => layout(`${id}_root`, { [`${id}_root`]: { type: 'Column', children: [child] } });
  const tabs: PageTab[] = [
    { id: 'rows', label: 'Current', layout: one('rows', 'grid') },
    { id: 'approvals', label: 'Approvals', badge: '{{queries.pending.data.length}}', layout: one('approvals', 'approvals_grid') },
    { id: 'history', label: 'History', layout: one('history', 'history_grid') },
  ];
  const needEntity = s.perEntity ? { enabledWhen: cond('vars.entity', 'is_not_empty') } : {};
  return structuredClone({
    name: s.name,
    slug: s.slug,
    description: s.description,
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
        ...(s.perEntity ? [{ name: 'entity', url: true, initFrom: { query: 'entities', path: '0.entity' } }] : []),
        { name: 'tab', default: 'rows', url: true },
        { name: 'editorOpen', default: false },
        { name: 'editorMode', default: 'new' },
        { name: 'editRow' },
        { name: 'editDraft', description: 'The row being proposed' },
      ],
      queries: [
        ...(s.perEntity ? [{ id: 'entities', operation: 'mdmConfig.entities', params: { kind: s.kind } }] : []),
        { id: 'table', operation: 'mdmConfig.table', params: { kind: s.kind, entity }, ...needEntity },
        { id: 'pending', operation: 'mdmConfig.changes', params: { kind: s.kind, entity, status: 'pending' }, ...needEntity },
        {
          id: 'editor', operation: 'mdmConfig.editor', params: { kind: s.kind, entity, mode: '{{vars.editorMode}}', id: '{{vars.editRow.id}}' },
          enabledWhen: cond('vars.editorOpen', 'is_true'),
        },
        { id: 'history', operation: 'mdmConfig.changes', params: { kind: s.kind, entity, status: '' }, enabledWhen: cond('vars.tab', 'equals', 'history') },
      ],
    },
  });
}

const text = (field: string, more: Partial<Extract<CellSpec, { kind: 'text' }>> = {}): CellSpec => ({ kind: 'text', value: `{{row.${field}}}`, ...more });

export const sourceHierarchyBlueprint = () => configPage({
  kind: 'source_priority', slug: 'mdm-source-hierarchy', name: 'Source hierarchy', icon: 'hub',
  title: 'Source hierarchy', subtitle: 'Which source wins, per field group (and per price type for prices). Lower priority wins; your rows come before the gold copy\'s.',
  description: 'Per mastered entity: the source ranking survivorship uses by field group, scoped by product type, asset class or price type. Maker-checker. Built in Page Studio.',
  perEntity: true, newLabel: 'Propose a ranking',
  columns: [
    col('group', 'Group', text('group', { bold: true, mono: true })),
    col('scope', 'Scope', text('scope_label')),
    col('source', 'Source', { kind: 'chip', value: '{{row.source}}', variant: 'outlined' }),
    col('priority', 'Priority', { kind: 'number', value: '{{row.priority}}' }, { align: 'right' }),
    col('settings', 'Settings', { kind: 'chips', value: '{{row.settings}}' }),
  ],
});

export const matchRulesBlueprint = () => configPage({
  kind: 'match_rule', slug: 'mdm-match-rules', name: 'Match rules', icon: 'review',
  title: 'Match rules', subtitle: 'How incoming records find their golden record: exact identifiers first, then weighted fuzzy keys with auto-match and review thresholds.',
  description: 'Per record entity: deterministic and fuzzy match keys, thresholds and priority. Your rule with the same code replaces the gold copy\'s. Maker-checker. Built in Page Studio.',
  perEntity: true, newLabel: 'Propose a rule',
  columns: [
    col('rule', 'Rule', { kind: 'twoLine', primary: '{{row.rule_cd}}', secondary: '{{row.rule_name}}' }),
    col('scope', 'Scope', text('scope_label')),
    col('exact', 'Exact keys', { kind: 'chips', value: '{{row.deterministic}}' }),
    col('fuzzy', 'Fuzzy keys', { kind: 'chips', value: '{{row.fuzzy}}' }),
    col('auto', 'Auto-match ≥', { kind: 'number', value: '{{row.threshold_auto_match}}', digits: 2 }, { align: 'right' }),
    col('review', 'Review ≥', { kind: 'number', value: '{{row.threshold_review}}', digits: 2 }, { align: 'right' }),
    col('priority', 'Priority', { kind: 'number', value: '{{row.priority}}' }, { align: 'right' }),
  ],
});

export const vendorRegistryBlueprint = () => configPage({
  kind: 'source_system', slug: 'mdm-vendors', name: 'Vendor registry', icon: 'storage',
  title: 'Vendor registry', subtitle: 'The one list of data sources every mastered entity uses - Bloomberg is one source across securities, prices and products.',
  description: 'The sources (vendors, internal systems) mastering ranks and matches on. Maker-checker. Built in Page Studio.',
  perEntity: false, newLabel: 'Propose a source',
  columns: [
    col('code', 'Code', text('code', { mono: true, bold: true })),
    col('name', 'Name', text('display_name')),
    col('feed', 'Feed type', text('feed_type')),
    col('desc', 'Description', text('description', { caption: true })),
  ],
});
