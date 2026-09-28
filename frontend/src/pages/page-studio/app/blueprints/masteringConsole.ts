import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, CellSpec, ColumnDef, ConditionNode } from '../appModel';

/**
 * The mastering console (formerly the hand-built MasteringPage.tsx), rebuilt as a
 * Page Studio page: the acceptance test for the page application model.
 * Every behaviour of the hand-built page is here as data - entity picker,
 * policy and run buttons, the last-run alert, tabs with live counts and
 * conditional tabs, filtered grids with rich cells, row actions with
 * approval states, merge / keep-apart forms, and the domain's own drawers
 * and dialogs placed as domain components.
 */

// --- small builders ------------------------------------------------------------

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });
const any = (...conditions: ConditionNode[]): ConditionNode => ({ type: 'group', operator: 'OR', conditions });
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
function layout(tree: string, root: string, spec: Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>): PageLayout {
  nodes[tree] = Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }]));
  return { root, nodes: nodes[tree] };
}
const fit = { flex: '0 0 auto' };
const text = (field: string, more: Partial<Extract<CellSpec, { kind: 'text' }>> = {}): CellSpec => ({ kind: 'text', value: `{{row.${field}}}`, ...more });
const col = (id: string, header: string, cell: CellSpec | CellSpec[], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell, ...more };
const openGolden = (idField: string): Action[] => [set('goldenId', `{{row.${idField}}}`)];
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
widget('golden_drawer', 'DomainComponent', {
  component: 'mastering.GoldenDrawer', inputs: { entity: E, id: '{{vars.goldenId}}' }, events: { close: [set('goldenId')] },
}, { visibleWhen: RECORD, style: fit });
widget('price_drawer', 'DomainComponent', {
  component: 'mastering.PriceDrawer', inputs: { entity: E, id: '{{vars.goldenId}}' }, events: { close: [set('goldenId')] },
}, { visibleWhen: SERIES, style: fit });
widget('run_dialog', 'DomainComponent', {
  component: 'mastering.RunDialog', inputs: { profile: '{{queries.profile.data}}', open: '{{vars.running}}' },
  events: {
    close: [set('running', false)],
    done: [set('running', false), set('lastRun', '{{event.run}}'), set('lastRunMessage', '{{event.messageKey}}'), set('tab', '{{event.tab}}')],
  },
}, { style: fit });
widget('policy_dialog', 'DomainComponent', {
  component: 'mastering.PolicyDialog', inputs: { entity: E, open: '{{vars.policyOpen}}' }, events: { close: [set('policyOpen', false)] },
}, { style: fit });

const filterBar = layout('top', 'top_root', {
  top_root: { type: 'Column', children: ['top_header', 'no_profiles', 'last_run', 'top_overlays'], style: { gap: '16px' } },
  top_header: { type: 'Row', children: ['hdr', 'entity_select', 'policy_btn', 'run_btn'], style: { alignItems: 'center' } },
  top_overlays: { type: 'Row', children: ['golden_drawer', 'price_drawer', 'run_dialog', 'policy_dialog'], style: { gap: '8px' } },
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
        { name: 'policyOpen', default: false },
        { name: 'running', default: false },
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
