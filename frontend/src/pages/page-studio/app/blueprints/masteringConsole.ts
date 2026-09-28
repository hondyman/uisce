import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, CellSpec, ColumnDef, ConditionNode } from '../appModel';

/**
 * The mastering console (formerly the hand-built MasteringPage.tsx), rebuilt as a
 * Page Studio page: the acceptance test for the page application model.
 * Every behaviour of the hand-built page is here as data - entity picker,
 * policy and run buttons, the last-run alert, tabs with live counts and
 * conditional tabs, filtered grids with rich cells, row actions with
 * approval states, merge / keep-apart forms, and the record and price
 * drawers, policy dialog and run wizard - all built from studio blocks.
 */

// --- small builders ------------------------------------------------------------

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });
const any = (...conditions: ConditionNode[]): ConditionNode => ({ type: 'group', operator: 'OR', conditions });
const all = (...conditions: ConditionNode[]): ConditionNode => ({ type: 'group', operator: 'AND', conditions });
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });

// A record (not time-series) entity - also true while the profile loads, so
// record tabs do not flicker away. (The engine answers null != true as false.)
const RECORD = any(cond('queries.profile.data', 'is_empty'), cond('queries.profile.data.series', 'is_false'));
const SERIES = cond('queries.profile.data.series', 'is_true');
const E = '{{vars.entity}}';

const components: Record<string, ComponentDefinition> = {};
const nodes: Record<string, PageLayout['nodes']> = {};

function widget(id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}): string {
  components[id] = { id, type, props, ...extra };
  return id;
}
type NodeSpec = { type: 'Row' | 'Column' | 'Drawer' | 'Dialog' | 'TabSet'; children: string[]; style?: Record<string, string>; props?: Record<string, unknown> };
function layout(tree: string, root: string, spec: Record<string, NodeSpec>): PageLayout {
  nodes[tree] = Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }]));
  return { root, nodes: nodes[tree] };
}
const fit = { flex: '0 0 auto' };
const text = (field: string, more: Partial<Extract<CellSpec, { kind: 'text' }>> = {}): CellSpec => ({ kind: 'text', value: `{{row.${field}}}`, ...more });
const col = (id: string, header: string, cell: CellSpec | CellSpec[], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell, ...more };
// Opening a record shows its latest version.
const openGolden = (idField: string): Action[] => [set('goldenId', `{{row.${idField}}}`), set('goldenVersion')];
const byState = (state: string) => cond('row.state', 'equals', state);

const GOLDEN_STATUS = { PUBLISHED: 'success', REVIEW: 'warning', SUPERSEDED: 'default', DRAFT: 'info', RETRACTED: 'error' } as const;
const RUN_STATUS = { COMPLETED: 'success', PARTIAL: 'warning', RUNNING: 'info', FAILED: 'error' } as const;
const COUNT_KEYS = ['records', 'invalid', 'xref', 'deterministic', 'fuzzy', 'review', 'new', 'restated', 'rechecked', 'conflicts', 'published', 'held_for_review', 'unchanged', 'exceptions'];
const COUNT_COLORS = { invalid: 'error', conflicts: 'error', review: 'warning', held_for_review: 'warning', exceptions: 'warning', published: 'success' } as const;
const SEVERITY = { ERROR: 'error', '*': 'warning' } as const;

// --- page-wide: header, last run, the domain's drawers and dialogs --------------

widget('hdr', 'PageHeader', { icon: 'hub', title: 'mastering.title', subtitle: 'mastering.subtitle' }, { style: { flex: '1 1 320px' } });
widget('entity_select', 'VariableSelect', {
  variable: 'entity', label: 'mastering.entity', minWidth: 200,
  optionsFrom: { query: 'profiles', valueField: 'entity', labelField: 'display_name' },
  onChange: [set('goldenId')],
}, { style: fit });
widget('policy_btn', 'ActionButton', {
  label: '{{queries.policy.data.label}}', icon: 'gavel', variant: 'outlined', disabledWhen: { type: 'condition', field: 'queries.profile.data', operator: 'is_empty' },
  onClick: [set('policyOpen', true)],
}, { style: fit });
widget('run_btn', 'ActionButton', {
  label: 'mastering.runLoad', icon: 'play', variant: 'contained', disabledWhen: { type: 'condition', field: 'queries.profile.data', operator: 'is_empty' },
  onClick: [set('running', true)],
}, { style: fit });
widget('no_profiles', 'AlertBanner', { severity: 'info', text: 'mastering.noProfiles' }, {
  visibleWhen: cond('queries.profiles.data.length', 'equals', 0),
});
widget('last_run', 'AlertBanner', {
  severity: '{{vars.lastRun.status}}', severityMap: { FAILED: 'error', PARTIAL: 'warning', '*': 'success' },
  text: { t: '{{vars.lastRunMessage}}' },
  chips: { value: '{{vars.lastRun.counts}}', labelKey: 'mastering.counts.', colorMap: COUNT_COLORS, keys: COUNT_KEYS },
  detail: '{{vars.lastRun.error_detail}}',
  onClose: [set('lastRun')],
}, { visibleWhen: cond('vars.lastRun', 'is_not_empty') });
// --- The golden record drawer (provenance), built from studio blocks --------------

const GD = 'queries.goldenDetail.data';
const COMPARE = cond('vars.goldenView', 'equals', 'compare');
const VALUES = cond('vars.goldenView', 'not_equals', 'compare');
const heading = (id: string, t: unknown, visibleWhen?: ConditionNode) =>
  widget(id, 'TextBlock', { text: t, variant: 'subtitle2' }, visibleWhen ? { visibleWhen } : {});
widget('gd_chips', 'KeyValue', { source: `{{${GD}}}`, columns: 1, items: [{ label: '', cell: { kind: 'chips', value: '{{data.header_chips}}' } }] });
widget('gd_historical', 'AlertBanner', {
  severity: 'info', text: `{{${GD}.viewing_text}}`, action: { label: 'mastering.golden.backToLatest', onClick: [set('goldenVersion')] },
}, { visibleWhen: cond(`${GD}.historical`, 'is_true') });
widget('gd_changes', 'TextBlock', { text: `{{${GD}.changes_text}}`, variant: 'caption', color: 'text.secondary' }, { visibleWhen: cond(`${GD}.changes_text`, 'is_not_empty') });
widget('gd_view', 'VariableSelect', {
  variable: 'goldenView', variant: 'toggle',
  options: [{ value: 'values', label: 'mastering.compare.values' }, { value: 'compare', label: 'mastering.compare.sideBySide' }],
}, { style: fit });
heading('gd_fields_title', `{{${GD}.fields_title}}`, VALUES);
const overrideForm: Action = {
  kind: 'runOperation', operation: 'mastering.overrides.propose',
  params: { entity: E, id: `{{${GD}.id}}`, attribute: '{{row.attribute}}', action: '{{form.action}}', value: '{{form.value}}', reason: '{{form.reason}}' },
  form: {
    title: '{{row.override_title}}', intro: '{{row.override_intro}}',
    notice: { severity: '{{row.override_severity}}', text: '{{row.override_notice}}' },
    fields: [
      { name: 'action', kind: 'radio', label: '', default: 'SET', visibleWhen: cond('row.overridden', 'is_true'),
        options: [{ value: 'SET', label: 'mastering.overrides.set' }, { value: 'CLEAR', label: 'mastering.overrides.clear' }] },
      { name: 'value', kind: 'text', label: 'mastering.overrides.newValue', default: '{{row.value_raw}}', visibleWhen: cond('form.action', 'not_equals', 'CLEAR') },
      { name: 'reason', kind: 'multiline', label: 'mastering.overrides.reason', required: true, helperText: 'mastering.overrides.reasonHelp' },
    ],
    submitLabel: '{{row.override_submit}}',
  },
};
widget('gd_fields', 'DataGrid', {
  query: 'goldenDetail', rowsPath: 'fields', progressOnFetch: true,
  columns: [
    col('attr', 'mastering.golden.attribute', text('attribute', { mono: true }), { nowrap: true }),
    col('value', 'mastering.golden.value', [text('value', { bold: true }), { kind: 'text', value: '{{row.was_text}}', caption: true, color: 'info' }]),
    col('source', 'mastering.golden.source', [
      { kind: 'chip', label: 'mastering.overrides.steward', color: 'secondary', visibleWhen: cond('row.overridden', 'is_true') },
      { kind: 'chip', value: '{{row.source}}', variant: 'outlined', visibleWhen: all(cond('row.overridden', 'is_false'), cond('row.source', 'is_not_empty')) },
    ], { nowrap: true }),
    col('confidence', 'mastering.golden.confidence', {
      kind: 'chip', value: '{{row.confidence_label}}', colorBy: '{{row.confidence_color}}', colorMap: { success: 'success', warning: 'warning', error: 'error' },
      variant: 'outlined', tooltip: 'mastering.golden.confidenceHelp',
    }),
    col('act', '', {
      kind: 'actions', buttons: [{
        label: { t: 'mastering.overrides.action', params: { field: '{{row.attribute}}' } }, icon: 'edit',
        visibleWhen: cond('row.can_override', 'is_true'), onClick: [overrideForm],
      }],
    }, { align: 'right', nowrap: true }),
  ] satisfies ColumnDef[],
  rowDetail: {
    rows: '{{row.competing}}', text: '{{row.reason}}', when: cond('row.has_decision', 'is_true'),
    columns: [
      col('src', 'mastering.golden.source', [text('source'), { kind: 'text', value: '{{row.source_key}}', mono: true, caption: true }]),
      col('val', 'mastering.golden.value', [
        { kind: 'text', value: '{{row.value}}', strike: '{{row.excluded}}' },
        { kind: 'text', value: '{{row.note}}', caption: true, color: 'warning' },
      ]),
      col('asof', 'mastering.golden.asOf', [
        { kind: 'datetime', value: '{{row.as_of}}' },
        { kind: 'chip', label: 'mastering.golden.stale', color: 'warning', variant: 'outlined', visibleWhen: cond('row.stale', 'is_true') },
      ], { nowrap: true, stackDirection: 'row' }),
    ],
  },
}, { visibleWhen: VALUES });
heading('gd_compare_title', `{{${GD}.compare_title}}`, COMPARE);
widget('gd_compare_help', 'TextBlock', { text: 'mastering.compare.help', variant: 'caption', color: 'text.secondary' }, { visibleWhen: COMPARE });
widget('gd_matrix', 'DataGrid', {
  query: 'goldenDetail', rowsPath: 'matrix.rows', stickyFirstColumn: true, maxHeight: 640,
  columns: [
    col('term', 'mastering.compare.term', [text('term', { bold: true }), { kind: 'text', value: '{{row.attribute}}', mono: true, caption: true }], { minWidth: 200 }),
    col('golden', 'mastering.compare.golden', [
      text('golden', { bold: true }),
      { kind: 'chip', label: 'mastering.overrides.steward', color: 'secondary', visibleWhen: cond('row.steward', 'is_true') },
    ], { minWidth: 170 }),
    col('selection', 'mastering.compare.selection', [
      { kind: 'chip', value: '{{row.strategy_label}}', colorBy: '{{row.steward}}', colorMap: { true: 'secondary', '*': 'default' }, variant: 'outlined', visibleWhen: cond('row.strategy', 'is_not_empty') },
      { kind: 'text', value: '{{row.rule}}', caption: true },
      { kind: 'text', value: '{{row.detail}}', caption: true },
    ], { minWidth: 260 }),
  ] satisfies ColumnDef[],
  dynamicColumns: {
    from: `{{${GD}.matrix.sources}}`, idField: 'code', header: '{{col.code}}', valuePath: 'by_source', insertAt: 2, minWidth: 150,
    cell: {
      kind: 'list', value: '{{value}}',
      item: { kind: 'text', value: '{{item.value}}', tone: '{{item.tone}}', strike: '{{item.strike}}', tooltip: '{{item.note}}' },
    },
  },
}, { visibleWhen: COMPARE });
heading('gd_ids_title', 'mastering.golden.identifiers');
widget('gd_ids', 'KeyValue', { source: `{{${GD}}}`, columns: 1, items: [{ label: '', cell: { kind: 'chips', value: '{{data.identifiers}}' } }] });
heading('gd_sources_title', 'mastering.golden.sources');
widget('gd_sources', 'DataGrid', {
  query: 'goldenDetail', rowsPath: 'sources_list',
  columns: [
    col('source', '', text('source')),
    col('key', '', text('source_key', { mono: true })),
    col('method', '', [{ kind: 'chip', value: '{{row.method}}', variant: 'outlined' }, { kind: 'percent', value: '{{row.score}}' }], { stackDirection: 'row', nowrap: true }),
    col('updated', '', { kind: 'datetime', value: '{{row.updated_at}}' }, { nowrap: true }),
  ] satisfies ColumnDef[],
});
heading('gd_versions_title', 'mastering.golden.versionsHelp');
widget('gd_versions', 'DataGrid', {
  query: 'goldenDetail', rowsPath: 'versions', onRowClick: [set('goldenVersion', '{{row.target}}')],
  columns: [
    col('v', '', [{ kind: 'text', value: 'v{{row.version}}' }, { kind: 'chip', label: 'mastering.golden.showing', visibleWhen: cond('row.showing', 'is_true') }], { stackDirection: 'row' }),
    col('status', '', { kind: 'chip', value: '{{row.status}}', labelKey: 'mastering.goldenStatus.', colorMap: GOLDEN_STATUS }),
    col('dq', '', text('dq_text')),
    col('at', '', { kind: 'datetime', value: '{{row.at}}' }, { nowrap: true }),
  ] satisfies ColumnDef[],
});
const HAS_OVERRIDES = cond(`${GD}.overrides.length`, 'not_equals', 0);
heading('gd_ovr_title', 'mastering.overrides.forRecord', HAS_OVERRIDES);
widget('gd_ovr', 'DataGrid', {
  query: 'goldenDetail', rowsPath: 'overrides',
  columns: [
    col('attr', '', text('attribute', { mono: true })),
    col('val', '', [
      { kind: 'text', text: 'mastering.overrides.clearDesc', visibleWhen: cond('row.action', 'equals', 'CLEAR') },
      { kind: 'text', value: '{{row.after_text}}', visibleWhen: cond('row.action', 'not_equals', 'CLEAR') },
    ]),
    col('state', '', { kind: 'chip', value: '{{row.status_view}}', label: '{{row.status_label}}', colorMap: { PENDING: 'warning', APPLIED: 'success', ACTIVE: 'success', REJECTED: 'error', '*': 'default' } }),
    col('who', '', { kind: 'text', value: '{{row.requested_by_name}} · {{row.reason}}', caption: true }),
  ] satisfies ColumnDef[],
}, { visibleWhen: HAS_OVERRIDES });
const HAS_EXCEPTIONS = cond(`${GD}.exceptions.length`, 'not_equals', 0);
heading('gd_exc_title', 'mastering.golden.openExceptions', HAS_EXCEPTIONS);
widget('gd_exc', 'DataGrid', {
  query: 'goldenDetail', rowsPath: 'exceptions',
  columns: [
    col('type', '', { kind: 'chip', value: '{{row.type}}', colorBy: '{{row.severity}}', colorMap: { ERROR: 'error', '*': 'warning' } }),
    col('desc', '', text('description')),
  ] satisfies ColumnDef[],
}, { visibleWhen: HAS_EXCEPTIONS });
// --- The golden price drawer ------------------------------------------------------
const PD = 'queries.priceDetail.data';
const DECIDE = `${PD}.decide`;
const priceOverride: Action = {
  kind: 'runOperation', operation: 'mastering.overrides.propose',
  params: { entity: E, id: `{{${DECIDE}.golden_id}}`, attribute: `{{${DECIDE}.attribute}}`, action: '{{form.action}}', value: '{{form.value}}', reason: '{{form.reason}}' },
  form: {
    title: `{{${DECIDE}.title}}`, intro: `{{${DECIDE}.intro}}`,
    notice: { severity: `{{${DECIDE}.severity}}`, text: `{{${DECIDE}.notice}}` },
    fields: [
      { name: 'action', kind: 'radio', label: '', default: 'SET', visibleWhen: cond(`${DECIDE}.has_active`, 'is_true'),
        options: [{ value: 'SET', label: 'mastering.overrides.set' }, { value: 'CLEAR', label: 'mastering.overrides.clear' }] },
      { name: 'value', kind: 'text', label: 'mastering.overrides.newValue', default: `{{${DECIDE}.value_raw}}`, visibleWhen: cond('form.action', 'not_equals', 'CLEAR') },
      { name: 'reason', kind: 'multiline', label: 'mastering.overrides.reason', required: true, helperText: 'mastering.overrides.reasonHelp' },
    ],
    submitLabel: `{{${DECIDE}.submit}}`,
  },
};
const CAN_DECIDE = cond(`${DECIDE}.can`, 'is_true');
widget('pd_value', 'TextBlock', { text: `{{${PD}.value_text}}`, variant: 'h4' }, { style: fit });
widget('pd_chips', 'KeyValue', { source: `{{${PD}}}`, columns: 1, items: [{ label: '', cell: { kind: 'chips', value: '{{data.header_chips}}' } }] }, { style: { flex: '1 1 200px' } });
widget('pd_decide_held', 'ActionButton', {
  label: `{{${DECIDE}.label}}`, variant: 'contained', disabledWhen: cond(`${DECIDE}.blocked`, 'is_true'), onClick: [priceOverride],
}, { visibleWhen: all(CAN_DECIDE, cond(`${DECIDE}.held`, 'is_true')), style: fit });
widget('pd_decide', 'ActionButton', {
  label: `{{${DECIDE}.label}}`, variant: 'outlined', disabledWhen: cond(`${DECIDE}.blocked`, 'is_true'), onClick: [priceOverride],
}, { visibleWhen: all(CAN_DECIDE, cond(`${DECIDE}.held`, 'is_false')), style: fit });
widget('pd_decide_state', 'KeyValue', {
  source: `{{${DECIDE}}}`, columns: 1, items: [{ label: '', cell: { kind: 'chips', value: '{{data.chips}}' } }],
}, { visibleWhen: all(CAN_DECIDE, cond(`${DECIDE}.chips.length`, 'not_equals', 0)), style: fit });
widget('pd_pending', 'TextBlock', { text: `{{${DECIDE}.pending_text}}`, variant: 'caption', color: 'text.secondary' },
  { visibleWhen: all(CAN_DECIDE, cond(`${DECIDE}.pending_text`, 'is_not_empty')), style: fit });
widget('pd_historical', 'AlertBanner', {
  severity: 'info', text: `{{${PD}.viewing_text}}`, action: { label: 'mastering.golden.backToLatest', onClick: [set('goldenVersion')] },
}, { visibleWhen: cond(`${PD}.historical`, 'is_true') });
widget('pd_reason', 'TextBlock', { text: `{{${PD}.reason_text}}` });
widget('pd_prior', 'TextBlock', { text: `{{${PD}.prior_text}}`, color: 'text.secondary' }, { visibleWhen: cond(`${PD}.prior_text`, 'is_not_empty') });
heading('pd_quotes_title', 'mastering.prices.quotes');
widget('pd_quotes', 'DataGrid', {
  query: 'priceDetail', rowsPath: 'candidates', progressOnFetch: true,
  columns: [
    col('source', 'mastering.golden.source', [text('source'), { kind: 'chip', label: 'mastering.prices.won', color: 'primary', visibleWhen: cond('row.won', 'is_true') }],
      { nowrap: true, stackDirection: 'row' }),
    col('price', 'mastering.prices.price', [
      { kind: 'text', value: '{{row.value_text}}', strike: '{{row.out}}' }, { kind: 'text', value: '{{row.currency}}', caption: true },
    ], { align: 'right', nowrap: true, stackDirection: 'row' }),
    col('diff', 'mastering.prices.diff', { kind: 'delta', value: '{{row.diff_pct}}', warn: 1, bad: 5 }, { align: 'right' }),
    col('rank', 'mastering.prices.rank', text('rank_text'), { align: 'center' }),
    col('asof', 'mastering.golden.asOf', [
      { kind: 'datetime', value: '{{row.as_of}}' },
      { kind: 'chip', label: 'mastering.golden.stale', color: 'warning', variant: 'outlined', visibleWhen: cond('row.stale', 'is_true') },
    ], { nowrap: true, stackDirection: 'row' }),
    col('why', '', [
      { kind: 'text', value: '{{row.excluded_text}}', caption: true, color: 'warning' },
      { kind: 'text', value: '{{row.note}}', caption: true, color: 'warning' },
    ]),
  ] satisfies ColumnDef[],
});
widget('pd_ranking', 'TextBlock', { text: `{{${PD}.ranking_text}}`, variant: 'caption', color: 'text.secondary' });
const HAS_CONTROLS = cond(`${PD}.controls.length`, 'not_equals', 0);
heading('pd_controls_title', 'mastering.prices.controls', HAS_CONTROLS);
widget('pd_controls', 'KeyValue', { source: `{{${PD}}}`, columns: 1, items: [{ label: '', cell: { kind: 'chips', value: '{{data.controls}}' } }] }, { visibleWhen: HAS_CONTROLS });
const HAS_VARIANCES = cond(`${PD}.variances.length`, 'not_equals', 0);
heading('pd_var_title', 'mastering.prices.variances', HAS_VARIANCES);
widget('pd_var', 'DataGrid', {
  query: 'priceDetail', rowsPath: 'variances',
  columns: [
    col('a', '', text('a_text')),
    col('b', '', text('b_text')),
    col('pct', '', { kind: 'delta', value: '{{row.variance_pct}}', warn: 1, bad: 5 }, { align: 'right' }),
    col('sev', '', { kind: 'chip', value: '{{row.severity}}', colorMap: { WARNING: 'warning', '*': 'error' } }),
    col('status', '', text('status')),
  ] satisfies ColumnDef[],
}, { visibleWhen: HAS_VARIANCES });
const HAS_PRICE_EXC = cond(`${PD}.exceptions.length`, 'not_equals', 0);
heading('pd_exc_title', 'mastering.golden.openExceptions', HAS_PRICE_EXC);
widget('pd_exc', 'DataGrid', {
  query: 'priceDetail', rowsPath: 'exceptions',
  columns: [
    col('type', '', { kind: 'chip', value: '{{row.type}}', colorBy: '{{row.severity}}', colorMap: { ERROR: 'error', '*': 'warning' } }),
    col('desc', '', text('description')),
  ] satisfies ColumnDef[],
}, { visibleWhen: HAS_PRICE_EXC });
heading('pd_versions_title', 'mastering.golden.versionsHelp');
widget('pd_versions', 'DataGrid', {
  query: 'priceDetail', rowsPath: 'versions', onRowClick: [set('goldenVersion', '{{row.target}}')],
  columns: [
    col('v', '', [{ kind: 'text', value: 'v{{row.version}}' }, { kind: 'chip', label: 'mastering.golden.showing', visibleWhen: cond('row.showing', 'is_true') }], { stackDirection: 'row' }),
    col('value', '', text('value_text'), { align: 'right', nowrap: true }),
    col('winner', '', text('winner')),
    col('status', '', { kind: 'chip', value: '{{row.status}}', labelKey: 'mastering.goldenStatus.', colorMap: GOLDEN_STATUS }),
    col('at', '', { kind: 'datetime', value: '{{row.at}}' }, { nowrap: true }),
  ] satisfies ColumnDef[],
});
widget('pd_bitemporal', 'TextBlock', { text: 'mastering.prices.bitemporal', variant: 'caption', color: 'text.secondary' });

// --- The override policy dialog ------------------------------------------------------
const CAN_EDIT = cond('queries.policy.data.can_edit', 'is_true');
const READ_ONLY = cond('queries.policy.data.can_edit', 'is_false');
const approvers = [1, 2, 3, 4, 5].map((n) => ({ value: String(n), label: String(n) }));
widget('pol_help', 'TextBlock', { text: 'mastering.policy.help', color: 'text.secondary' });
widget('pol_form', 'Form', {
  variable: 'policyDraft', initFrom: '{{queries.policy.data.policy}}',
  fields: [
    { name: 'mode', kind: 'radio', label: '', readOnlyWhen: READ_ONLY,
      options: [{ value: 'APPROVAL', label: 'mastering.policy.approval' }, { value: 'DIRECT', label: 'mastering.policy.direct' }] },
    { name: 'approvals_required', kind: 'select', label: 'mastering.policy.approvers', required: true, valueType: 'number', options: approvers,
      readOnlyWhen: READ_ONLY, visibleWhen: cond('form.mode', 'equals', 'APPROVAL') },
    { name: 'high_risk_attributes', kind: 'chips', label: 'mastering.policy.highRisk', helperText: 'mastering.policy.highRiskHelp',
      optionsFrom: { query: 'policy', rowsPath: 'attributes' }, readOnlyWhen: READ_ONLY, visibleWhen: cond('form.mode', 'equals', 'APPROVAL') },
    { name: 'high_risk_approvals', kind: 'select', label: 'mastering.policy.highRiskApprovers', required: true, valueType: 'number', options: approvers,
      readOnlyWhen: any(READ_ONLY, cond('form.high_risk_attributes', 'is_empty')), visibleWhen: cond('form.mode', 'equals', 'APPROVAL') },
  ],
});
widget('pol_direct', 'AlertBanner', { severity: 'warning', text: 'mastering.policy.directWarning' }, { visibleWhen: cond('vars.policyDraft.mode', 'equals', 'DIRECT') });
widget('pol_caption', 'TextBlock', { text: '{{queries.policy.data.caption}}', variant: 'caption', color: 'text.secondary' });
widget('pol_admins', 'AlertBanner', { severity: 'info', text: 'mastering.policy.adminsOnly' }, { visibleWhen: READ_ONLY });

// --- The run wizard: pick a load, preview (nothing kept), run ---------------------------
const RS = 'queries.runSetup.data';
const runParams = { entity: E, table: '{{vars.runForm.table}}', load_id: '{{vars.runForm.load_id}}', again: '{{vars.runForm.again}}' };
widget('run_form', 'Form', {
  variable: 'runForm', initFrom: `{{${RS}.initial}}`, columns: 2,
  fields: [
    { name: 'load_id', kind: 'select', label: 'mastering.runDialog.load', required: true, helperText: `{{${RS}.loads_help}}`,
      optionsFrom: { query: 'runSetup', rowsPath: 'loads', valueField: 'id', labelField: 'label', captionField: 'caption' } },
    { name: 'table', kind: 'select', label: 'mastering.runDialog.table', required: true, helperText: `{{${RS}.table_help}}`,
      optionsFrom: { query: 'runSetup', rowsPath: 'tables', valueField: 'value' } },
  ],
  // A preview is of the choice it was made for.
  onChange: [set('runPreview')],
});
widget('run_again_note', 'AlertBanner', { severity: 'info', text: 'mastering.runDialog.alreadyMastered' }, { visibleWhen: cond(`${RS}.already_mastered`, 'is_true') });
widget('run_again', 'Form', {
  variable: 'runForm', fields: [{ name: 'again', kind: 'switch', label: 'mastering.runDialog.again' }],
}, { visibleWhen: cond(`${RS}.already_mastered`, 'is_true') });
widget('run_stage', 'TextBlock', { text: '{{vars.runStage}}', variant: 'caption', color: 'text.secondary' }, { visibleWhen: cond('vars.runStage', 'is_not_empty') });
const PREVIEWED = cond('vars.runPreview', 'is_not_empty');
heading('run_preview_title', 'mastering.runDialog.previewTitle', PREVIEWED);
widget('run_preview_counts', 'KeyValue', { source: '{{vars.runPreview}}', columns: 1, items: [{ label: '', cell: { kind: 'chips', value: '{{data.chips}}' } }] }, { visibleWhen: PREVIEWED });
widget('run_preview_issues', 'KeyValue', {
  source: '{{vars.runPreview}}', columns: 1, items: [{
    label: '', visibleWhen: cond('data.exceptions.length', 'not_equals', 0),
    cell: {
      kind: 'list', value: '{{data.exceptions}}',
      item: { kind: 'chip', value: '{{item.code}}', colorBy: '{{item.severity}}', colorMap: { ERROR: 'error', '*': 'warning' }, caption: '{{item.message}}' },
    },
  }],
}, { visibleWhen: PREVIEWED });
widget('run_preview_unmastered', 'TextBlock', { text: '{{vars.runPreview.unmastered_text}}', variant: 'caption', color: 'text.secondary' },
  { visibleWhen: all(PREVIEWED, cond('vars.runPreview.unmastered_text', 'is_not_empty')) });
const closeRun = [set('running', false), set('runForm'), set('runPreview')];

const filterBar = layout('top', 'top_root', {
  top_root: { type: 'Column', children: ['top_header', 'no_profiles', 'last_run', 'top_overlays'], style: { gap: '16px' } },
  top_header: { type: 'Row', children: ['hdr', 'entity_select', 'policy_btn', 'run_btn'], style: { alignItems: 'center' } },
  top_overlays: { type: 'Row', children: ['golden_drawer', 'price_drawer', 'run_dialog', 'policy_dialog'], style: { gap: '8px' } },
  price_drawer: {
    type: 'Drawer', children: ['pd_body'], props: {
      title: `{{${PD}.title}}`, subtitle: `{{${PD}.subtitle}}`, width: 760,
      openWhen: all(SERIES, cond('vars.goldenId', 'is_not_empty')),
      onClose: [set('goldenId'), set('goldenVersion')],
    },
  },
  pd_body: {
    type: 'Column', style: { gap: '12px' }, children: [
      'pd_head', 'pd_decide_row', 'pd_historical', 'pd_reason', 'pd_prior', 'pd_quotes_title', 'pd_quotes', 'pd_ranking',
      'pd_controls_title', 'pd_controls', 'pd_var_title', 'pd_var', 'pd_exc_title', 'pd_exc', 'pd_versions_title', 'pd_versions', 'pd_bitemporal',
    ],
  },
  pd_head: { type: 'Row', children: ['pd_value', 'pd_chips'], style: { alignItems: 'center', gap: '12px' } },
  pd_decide_row: { type: 'Row', children: ['pd_decide_held', 'pd_decide', 'pd_decide_state', 'pd_pending'], style: { alignItems: 'center', gap: '8px' } },
  policy_dialog: {
    type: 'Dialog', children: ['pol_body'], props: {
      title: 'mastering.policy.title', maxWidth: 'sm',
      openWhen: cond('vars.policyOpen', 'is_true'),
      onClose: [set('policyOpen', false)],
      buttons: [
        { label: 'mastering.cancel', visibleWhen: CAN_EDIT, onClick: [set('policyOpen', false)] },
        { label: 'mastering.close', visibleWhen: READ_ONLY, onClick: [set('policyOpen', false)] },
        {
          label: 'mastering.policy.save', variant: 'contained', visibleWhen: CAN_EDIT, disabledWhen: cond('vars.policyDraft', 'is_empty'),
          onClick: [{ kind: 'runOperation', operation: 'mastering.policy.save', params: { entity: E, policy: '{{vars.policyDraft}}' }, onSuccess: [set('policyOpen', false)] }],
        },
      ],
    },
  },
  pol_body: { type: 'Column', style: { gap: '16px' }, children: ['pol_help', 'pol_form', 'pol_direct', 'pol_caption', 'pol_admins'] },
  run_dialog: {
    type: 'Dialog', children: ['run_body'], props: {
      title: { t: 'mastering.runDialog.title', params: { entity: '{{queries.profile.data.display_name}}' } }, maxWidth: 'md',
      openWhen: cond('vars.running', 'is_true'),
      onClose: closeRun,
      buttons: [
        { label: 'mastering.cancel', onClick: closeRun },
        {
          label: 'mastering.runDialog.preview', disabledWhen: cond(`${RS}.ready`, 'is_false'),
          onClick: [{ kind: 'runOperation', operation: 'mastering.runs.preview', params: runParams, onSuccess: [set('runPreview', '{{result}}')] }],
        },
        {
          label: 'mastering.runDialog.run', variant: 'contained', disabledWhen: cond(`${RS}.can_run`, 'is_false'),
          onClick: [{
            kind: 'runOperation', operation: 'mastering.runs.run', params: runParams, progressVariable: 'runStage',
            onSuccess: [...closeRun, set('lastRun', '{{result.run}}'), set('lastRunMessage', '{{result.messageKey}}'), set('tab', '{{result.tab}}')],
          }],
        },
      ],
    },
  },
  run_body: {
    type: 'Column', style: { gap: '16px' }, children: [
      'run_form', 'run_again_note', 'run_again', 'run_stage', 'run_preview_title', 'run_preview_counts', 'run_preview_issues', 'run_preview_unmastered',
    ],
  },
  golden_drawer: {
    type: 'Drawer', children: ['gd_body'], props: {
      title: `{{${GD}.title}}`, subtitle: `{{${GD}.code}}`, width: `{{${GD}.width}}`,
      openWhen: all(RECORD, cond('vars.goldenId', 'is_not_empty')),
      onClose: [set('goldenId'), set('goldenVersion')],
    },
  },
  gd_body: {
    type: 'Column', style: { gap: '16px' }, children: [
      'gd_chips', 'gd_historical', 'gd_changes', 'gd_view', 'gd_fields_title', 'gd_fields', 'gd_compare_title', 'gd_compare_help', 'gd_matrix',
      'gd_ids_title', 'gd_ids', 'gd_sources_title', 'gd_sources', 'gd_versions_title', 'gd_versions', 'gd_ovr_title', 'gd_ovr', 'gd_exc_title', 'gd_exc',
    ],
  },
});

// --- Golden records -------------------------------------------------------------

widget('golden_search', 'SearchInput', { variable: 'goldenQ', placeholder: 'mastering.golden.search', maxWidth: 420 });
widget('golden_status', 'VariableSelect', {
  variable: 'goldenStatus', label: 'mastering.golden.status', emptyLabel: 'mastering.all', minWidth: 180,
  options: ['PUBLISHED', 'REVIEW'].map((s) => ({ value: s, label: `mastering.goldenStatus.${s}` })),
}, { style: fit });
widget('golden_grid', 'DataGrid', {
  query: 'golden', emptyText: 'mastering.golden.empty', onRowClick: openGolden('id'),
  columns: [
    col('code', 'mastering.golden.code', text('code', { mono: true }), { nowrap: true }),
    col('name', 'mastering.golden.name', text('name')),
    col('status', 'mastering.golden.status', { kind: 'chip', value: '{{row.status}}', labelKey: 'mastering.goldenStatus.', colorMap: GOLDEN_STATUS, caption: 'v{{row.version}}' }),
    col('dq', 'mastering.golden.dqShort', text('dq_score'), { align: 'right' }),
    col('identity', 'mastering.golden.identityShort', { kind: 'percent', value: '{{row.identity_confidence}}' }, { align: 'right' }),
    col('sources', 'mastering.golden.sources', { kind: 'chip', value: '{{row.winning_label}}', tooltip: '{{row.winning_tooltip}}' }),
    col('updated', 'mastering.golden.updated', { kind: 'datetime', value: '{{row.updated_at}}' }, { nowrap: true }),
  ] satisfies ColumnDef[],
});

// --- Golden prices (time series) --------------------------------------------------

widget('price_date', 'VariableSelect', {
  variable: 'priceDate', label: 'mastering.prices.date', minWidth: 170, optionsFrom: { query: 'prices', rowsPath: 'dates' },
}, { style: fit });
widget('price_search', 'SearchInput', { variable: 'priceQ', placeholder: 'mastering.prices.search', maxWidth: 360 });
widget('price_type', 'VariableSelect', {
  variable: 'priceType', label: 'mastering.prices.type', emptyLabel: 'mastering.all', minWidth: 170,
  options: ['LAST', 'OFFICIAL_CLOSE', 'BID', 'MID', 'ASK', 'NAV', 'EVALUATED'].map((p) => ({ value: p, label: p })),
}, { style: fit });
widget('price_status', 'VariableSelect', {
  variable: 'priceStatus', label: 'mastering.golden.status', emptyLabel: 'mastering.all', minWidth: 160,
  options: ['PUBLISHED', 'REVIEW'].map((s) => ({ value: s, label: `mastering.goldenStatus.${s}` })),
}, { style: fit });
widget('price_check', 'ActionButton', {
  label: 'mastering.prices.completeness.check', variant: 'outlined',
  disabledWhen: cond('queries.prices.data.date', 'is_empty'),
  onClick: [{
    kind: 'runOperation', operation: 'mastering.prices.completeness', params: { entity: E, date: '{{queries.prices.data.date}}' },
    onSuccess: [set('completeness', '{{result}}')],
  }],
}, { style: fit });
widget('price_completeness', 'AlertBanner', {
  severity: '{{vars.completeness.severity}}', text: '{{vars.completeness.message}}',
  chips: { value: '{{vars.completeness.gaps}}' }, onClose: [set('completeness')],
}, { visibleWhen: cond('vars.completeness', 'is_not_empty') });
widget('price_grid', 'DataGrid', {
  query: 'prices', rowsPath: 'prices', emptyText: 'mastering.prices.empty', progressOnFetch: true, onRowClick: openGolden('id'),
  columns: [
    col('instrument', 'mastering.prices.instrument', { kind: 'twoLine', primary: '{{row.name}}', secondary: '{{row.code}}', secondaryMono: true }),
    col('type', 'mastering.prices.type', text('price_type', { mono: true })),
    col('price', 'mastering.prices.price', { kind: 'number', value: '{{row.value}}', suffix: '{{row.currency}}' }, { align: 'right', nowrap: true }),
    col('change', 'mastering.prices.change', { kind: 'delta', value: '{{row.change_pct}}', warn: 3, bad: 10 }, { align: 'right' }),
    col('winner', 'mastering.prices.winner', { kind: 'chip', value: '{{row.winner_label}}', variant: 'outlined', tooltip: '{{row.sources_help}}' }),
    col('spread', 'mastering.prices.spread', { kind: 'delta', value: '{{row.variance_pct}}' }, { align: 'right' }),
    col('status', 'mastering.golden.status', [
      { kind: 'chip', value: '{{row.status}}', labelKey: 'mastering.goldenStatus.', colorMap: GOLDEN_STATUS },
      { kind: 'chip', label: 'mastering.golden.stale', color: 'warning', variant: 'outlined', visibleWhen: cond('row.is_stale', 'is_true') },
      { kind: 'text', value: 'v{{row.version}}', caption: true },
    ], { stackDirection: 'row', nowrap: true }),
    col('updated', 'mastering.golden.updated', { kind: 'datetime', value: '{{row.updated_at}}' }, { nowrap: true }),
  ] satisfies ColumnDef[],
});

// --- Runs ---------------------------------------------------------------------

widget('runs_grid', 'DataGrid', {
  query: 'runs', emptyText: 'mastering.runs.empty',
  columns: [
    col('started', 'mastering.runs.started', { kind: 'datetime', value: '{{row.started_at}}' }, { nowrap: true }),
    col('status', 'mastering.runs.status', [
      { kind: 'chip', value: '{{row.status}}', labelKey: 'mastering.runStatus.', colorMap: RUN_STATUS },
      { kind: 'text', value: '{{row.error_detail}}', caption: true, color: 'error' },
    ]),
    col('trigger', 'mastering.runs.trigger', { kind: 'text', text: 'mastering.trigger.{{row.trigger}}' }),
    col('outcome', 'mastering.runs.outcome', { kind: 'chips', value: '{{row.counts}}', labelKey: 'mastering.counts.', colorMap: COUNT_COLORS, keys: COUNT_KEYS }),
    col('by', 'mastering.runs.by', text('started_by')),
  ] satisfies ColumnDef[],
});

// --- Exceptions -----------------------------------------------------------------

const EXC_OPEN = any(cond('vars.exceptionStatus', 'is_empty'), cond('vars.exceptionStatus', 'in', ['OPEN', 'IN_REVIEW']));
const resolveAs = (status: 'RESOLVED' | 'WAIVED'): Action[] => [{
  kind: 'runOperation', operation: 'mastering.exceptions.resolve', params: { entity: E, id: '{{row.id}}', status },
}];
widget('exc_status', 'VariableSelect', {
  variable: 'exceptionStatus', label: 'mastering.exceptions.status', emptyLabel: 'mastering.exceptions.open', minWidth: 200,
  options: ['RESOLVED', 'WAIVED'].map((s) => ({ value: s, label: `mastering.exceptionStatus.${s}` })),
}, { style: fit });
widget('exc_grid', 'DataGrid', {
  query: 'exceptions', emptyText: 'mastering.exceptions.empty',
  columns: [
    col('type', 'mastering.exceptions.type', { kind: 'chip', value: '{{row.type}}', labelKey: 'mastering.exceptionType.', colorBy: '{{row.severity}}', colorMap: SEVERITY }),
    col('description', 'mastering.exceptions.description', text('description')),
    col('record', 'mastering.exceptions.record', [
      { kind: 'link', label: 'mastering.exceptions.openGolden', onClick: openGolden('golden_id'), visibleWhen: cond('row.golden_id', 'is_not_empty') },
      { kind: 'text', value: '{{row.source_ref}}', mono: true, visibleWhen: cond('row.golden_id', 'is_empty') },
    ], { nowrap: true }),
    col('detected', 'mastering.exceptions.detected', { kind: 'datetime', value: '{{row.detected_at}}' }, { nowrap: true }),
    col('actions', '', {
      kind: 'actions', buttons: [
        { label: 'mastering.exceptions.resolve', onClick: resolveAs('RESOLVED') },
        { label: 'mastering.exceptions.waive', color: 'inherit', onClick: resolveAs('WAIVED') },
      ],
    }, { align: 'right', nowrap: true, visibleWhen: EXC_OPEN }),
  ] satisfies ColumnDef[],
});

// --- Review (possible duplicates) --------------------------------------------------

const decide = (merge: boolean): Action => ({
  kind: 'runOperation', operation: 'mastering.candidates.decide',
  params: { entity: E, id: '{{row.id}}', merge, keep: merge ? '{{form.keep}}' : 'a', note: '{{form.note}}' },
  successMessage: '{{result.message}}',
  form: merge ? {
    title: 'mastering.review.mergeTitle', intro: 'mastering.review.mergeHelp',
    notice: {
      severity: 'info', visibleWhen: cond('queries.policy.data.policy.mode', 'equals', 'APPROVAL'),
      text: { t: 'mastering.review.needsApproval', params: { count: '{{queries.policy.data.policy.approvals_required}}' } },
    },
    fields: [
      {
        name: 'keep', kind: 'radio', label: '', default: 'a', options: [
          { value: 'a', label: { t: 'mastering.review.keep', params: { code: '{{row.a_code}}', name: '{{row.a_name}}' } } },
          { value: 'b', label: { t: 'mastering.review.keep', params: { code: '{{row.b_code}}', name: '{{row.b_name}}' } } },
        ],
      },
      { name: 'note', kind: 'multiline', label: 'mastering.review.note', helperText: 'mastering.review.noteHelp' },
    ],
    submitLabel: 'mastering.review.confirmMerge',
  } : {
    title: 'mastering.review.notSameTitle', intro: 'mastering.review.notSameHelp',
    fields: [{ name: 'note', kind: 'multiline', label: 'mastering.review.note', helperText: 'mastering.review.noteHelp' }],
    submitLabel: 'mastering.review.confirmNotSame', submitColor: 'inherit',
  },
});
const voteMerge = (approve: boolean): Action[] => [{
  kind: 'runOperation', operation: 'mastering.merges.vote', params: { entity: E, id: '{{row.merge_request_id}}', approve }, successMessage: '{{result.message}}',
}];
widget('review_help', 'AlertBanner', { severity: 'info', text: 'mastering.review.help' });
widget('review_grid', 'DataGrid', {
  query: 'candidates', emptyText: 'mastering.review.empty',
  columns: [
    col('a', 'mastering.review.existing', { kind: 'link', label: '{{row.a_code}}', after: '{{row.a_name}}', onClick: openGolden('a') }),
    col('b', 'mastering.review.new', { kind: 'link', label: '{{row.b_code}}', after: '{{row.b_name}}', onClick: openGolden('b') }),
    col('score', 'mastering.review.score', { kind: 'percent', value: '{{row.score}}' }, { align: 'right' }),
    col('rule', 'mastering.review.rule', text('rule')),
    col('decide', '', [
      { kind: 'text', value: '{{row.merge_caption}}', caption: true },
      { kind: 'text', text: 'mastering.overrides.youVoted', caption: true, visibleWhen: byState('voted') },
      {
        kind: 'actions', buttons: [
          { label: 'mastering.overrides.withdraw', color: 'inherit', visibleWhen: byState('mine'),
            onClick: [{ kind: 'runOperation', operation: 'mastering.merges.withdraw', params: { entity: E, id: '{{row.merge_request_id}}' } }] },
          { label: 'mastering.review.approveMerge', variant: 'contained', visibleWhen: byState('can_vote'), onClick: voteMerge(true) },
          { label: 'mastering.overrides.reject', color: 'error', visibleWhen: byState('can_vote'), onClick: voteMerge(false) },
          { label: 'mastering.review.merge', variant: 'outlined', visibleWhen: byState('open'), onClick: [decide(true)] },
          { label: 'mastering.review.notSame', color: 'inherit', visibleWhen: byState('open'), onClick: [decide(false)] },
        ],
      },
    ], { align: 'right', nowrap: true }),
  ] satisfies ColumnDef[],
});

// --- Overrides --------------------------------------------------------------------

const voteOverride = (approve: boolean): Action[] => [{
  kind: 'runOperation', operation: 'mastering.overrides.vote', params: { entity: E, id: '{{row.id}}', approve, comment: '{{rowState.comment}}' },
}];
widget('ovr_status', 'VariableSelect', {
  variable: 'overrideStatus', label: 'mastering.overrides.show', minWidth: 200,
  options: [
    { value: 'PENDING', label: 'mastering.overrides.status.PENDING' },
    { value: 'ACTIVE', label: 'mastering.overrides.active' },
    { value: '', label: 'mastering.all' },
  ],
}, { style: fit });
widget('ovr_grid', 'DataGrid', {
  query: 'overrides', emptyText: 'mastering.overrides.empty',
  columns: [
    col('record', 'mastering.overrides.record', [
      { kind: 'link', label: '{{row.record_label}}', onClick: openGolden('open_ref') },
      { kind: 'text', value: '{{row.golden_name}}', caption: true },
    ]),
    col('change', 'mastering.overrides.change', [
      { kind: 'text', value: '{{row.attribute}}', mono: true },
      { kind: 'text', text: 'mastering.overrides.clearDesc', visibleWhen: cond('row.action', 'equals', 'CLEAR') },
      { kind: 'diff', before: '{{row.before_text}}', after: '{{row.after_text}}', visibleWhen: cond('row.action', 'not_equals', 'CLEAR') },
    ]),
    col('reason', 'mastering.overrides.reason', text('reason'), { minWidth: 160 }),
    col('requested', 'mastering.overrides.requested', [
      text('requested_by_name'),
      { kind: 'datetime', value: '{{row.requested_at}}', caption: true },
    ], { nowrap: true }),
    col('state', 'mastering.overrides.state', [
      { kind: 'chip', value: '{{row.status_view}}', label: '{{row.status_label}}', colorMap: { PENDING: 'warning', APPLIED: 'success', ACTIVE: 'success', REJECTED: 'error', '*': 'default' } },
      { kind: 'text', value: '{{row.approvals_text}}', caption: true },
      { kind: 'text', value: '{{row.voters}}', caption: true },
    ]),
    col('act', '', [
      { kind: 'text', text: 'mastering.overrides.youVoted', caption: true, visibleWhen: byState('voted') },
      { kind: 'input', name: 'comment', placeholder: 'mastering.overrides.comment', visibleWhen: byState('can_vote') },
      {
        kind: 'actions', buttons: [
          { label: 'mastering.overrides.withdraw', color: 'inherit', visibleWhen: byState('mine'),
            onClick: [{ kind: 'runOperation', operation: 'mastering.overrides.withdraw', params: { entity: E, id: '{{row.id}}' } }] },
          { label: 'mastering.overrides.approve', variant: 'contained', visibleWhen: byState('can_vote'), onClick: voteOverride(true) },
          { label: 'mastering.overrides.reject', color: 'error', visibleWhen: byState('can_vote'), onClick: voteOverride(false) },
        ],
      },
    ], { align: 'right', nowrap: true, minWidth: 220 }),
  ] satisfies ColumnDef[],
});

// --- Tabs -----------------------------------------------------------------------

const filters = (id: string, children: string[]) => ({ [id]: { type: 'Row' as const, children, style: { alignItems: 'center' } } });
const tabs: PageTab[] = [
  {
    id: 'golden', label: 'mastering.tabs.golden', visibleWhen: RECORD,
    layout: layout('golden', 'golden_root', {
      golden_root: { type: 'Column', children: ['golden_filters', 'golden_grid'] },
      ...filters('golden_filters', ['golden_search', 'golden_status']),
    }),
  },
  {
    id: 'prices', label: 'mastering.tabs.prices', visibleWhen: SERIES,
    layout: layout('prices', 'prices_root', {
      prices_root: { type: 'Column', children: ['price_filters', 'price_completeness', 'price_grid'] },
      ...filters('price_filters', ['price_date', 'price_search', 'price_type', 'price_status', 'price_check']),
    }),
  },
  { id: 'runs', label: 'mastering.tabs.runs', layout: layout('runs', 'runs_root', { runs_root: { type: 'Column', children: ['runs_grid'] } }) },
  {
    id: 'exceptions', label: 'mastering.tabs.exceptions', badge: '{{queries.exceptionsOpen.data.length}}',
    layout: layout('exceptions', 'exc_root', {
      exc_root: { type: 'Column', children: ['exc_filters', 'exc_grid'] },
      ...filters('exc_filters', ['exc_status']),
    }),
  },
  {
    id: 'review', label: 'mastering.tabs.review', badge: '{{queries.toReview.data}}', visibleWhen: RECORD,
    layout: layout('review', 'review_root', { review_root: { type: 'Column', children: ['review_help', 'review_grid'] } }),
  },
  {
    id: 'overrides', label: 'mastering.tabs.overrides', badge: '{{queries.toDecide.data}}',
    layout: layout('overrides', 'ovr_root', {
      ovr_root: { type: 'Column', children: ['ovr_filters', 'ovr_grid'] },
      ...filters('ovr_filters', ['ovr_status']),
    }),
  },
];

const q = (id: string, operation: string, params: Record<string, unknown> = { entity: E }, more: Record<string, unknown> = {}) => ({ id, operation, params, ...more });

export function masteringConsoleBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  return structuredClone({
    name: 'Mastering console',
    slug: 'mastering-console',
    description: 'Golden records built from every source - where each value came from, what needs a steward. Built in Page Studio.',
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
        { name: 'entity', url: true, initFrom: { query: 'profiles', path: '0.entity' }, description: 'The mastered entity on show' },
        { name: 'tab', default: 'golden', url: true },
        { name: 'goldenId', url: true, description: 'The record open in the drawer' },
        { name: 'goldenVersion', description: 'The version the drawer shows; empty = the latest' },
        { name: 'goldenView', default: 'values', description: 'Values or side by side' },
        { name: 'policyOpen', default: false },
        { name: 'policyDraft', description: 'The policy being edited' },
        { name: 'running', default: false },
        { name: 'runForm', description: 'The load and staging table chosen to master' },
        { name: 'runPreview', description: 'What a run would do (the last preview)' },
        { name: 'runStage', description: 'The stage a run has reached' },
        { name: 'lastRun' },
        { name: 'lastRunMessage', default: '' },
        { name: 'goldenQ', default: '' },
        { name: 'goldenStatus', default: '' },
        { name: 'priceDate', initFrom: { query: 'prices', path: 'date' } },
        { name: 'priceQ', default: '' },
        { name: 'priceType', default: '' },
        { name: 'priceStatus', default: '' },
        { name: 'exceptionStatus', default: '' },
        { name: 'overrideStatus', default: 'PENDING' },
        { name: 'completeness', description: 'The last completeness check for the date shown' },
      ],
      queries: [
        q('profiles', 'mastering.profiles', {}),
        q('profile', 'mastering.profile'),
        q('policy', 'mastering.policy'),
        q('goldenDetail', 'mastering.golden.detail', { entity: E, id: '{{vars.goldenId}}', version: '{{vars.goldenVersion}}', view: '{{vars.goldenView}}' },
          { enabledWhen: all(RECORD, cond('vars.goldenId', 'is_not_empty')), keepPrevious: true }),
        q('priceDetail', 'mastering.price.detail', { entity: E, id: '{{vars.goldenId}}', version: '{{vars.goldenVersion}}' },
          { enabledWhen: all(SERIES, cond('vars.goldenId', 'is_not_empty')), keepPrevious: true }),
        q('runSetup', 'mastering.runs.setup', {
          entity: E, bo_key: '{{queries.profile.data.bo_key}}', load_id: '{{vars.runForm.load_id}}', table: '{{vars.runForm.table}}', again: '{{vars.runForm.again}}',
        }, { enabledWhen: cond('vars.running', 'is_true'), keepPrevious: true }),
        q('golden', 'mastering.golden.list', { entity: E, q: '{{vars.goldenQ}}', status: '{{vars.goldenStatus}}' }, { enabledWhen: RECORD }),
        q('prices', 'mastering.prices.list', {
          entity: E, date: '{{vars.priceDate}}', q: '{{vars.priceQ}}', price_type: '{{vars.priceType}}', status: '{{vars.priceStatus}}',
        }, { enabledWhen: SERIES, keepPrevious: true }),
        q('runs', 'mastering.runs.list'),
        // Same params as `exceptions` with no filter, so the two share one request.
        q('exceptionsOpen', 'mastering.exceptions.list', { entity: E, status: '' }),
        q('exceptions', 'mastering.exceptions.list', { entity: E, status: '{{vars.exceptionStatus}}' }),
        q('candidates', 'mastering.candidates.list', { entity: E }, { enabledWhen: RECORD }),
        q('toReview', 'mastering.candidates.toReview', { entity: E }, { enabledWhen: RECORD }),
        q('overrides', 'mastering.overrides.list', { entity: E, status: '{{vars.overrideStatus}}' }),
        q('toDecide', 'mastering.overrides.toDecide'),
      ],
    },
  });
}
