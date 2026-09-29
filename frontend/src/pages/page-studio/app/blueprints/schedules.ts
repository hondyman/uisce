import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, CellSpec, ColumnDef, ConditionNode, FormFieldSpec, PageQuery, PageVariable } from '../appModel';

/**
 * The Schedules console (formerly the hand-built SchedulesPage) and the
 * schedule editor dialog, as Page Studio configuration. The editor is a
 * fragment any page with a schedulable target places (the pipeline editor
 * fixes its target to the pipeline); its draft -> timing rules live in the
 * schedules domain (features/schedules/studio.ts).
 */

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });
const all = (...conditions: ConditionNode[]): ConditionNode => ({ type: 'group', operator: 'AND', conditions });
const any = (...conditions: ConditionNode[]): ConditionNode => ({ type: 'group', operator: 'OR', conditions });
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const op = (operation: string, params: Record<string, unknown>, onSuccess: Action[] = [], more: Record<string, unknown> = {}): Action =>
  ({ kind: 'runOperation', operation, params, onSuccess, ...more } as Action);
const col = (id: string, header: string, cell: CellSpec | CellSpec[], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell, ...more };
const fit = { flex: '0 0 auto' };

type NodeSpec = { type: 'Row' | 'Column' | 'Dialog'; children: string[]; style?: Record<string, string>; props?: Record<string, unknown> };
export type Fragment = { components: Record<string, ComponentDefinition>; nodes: Record<string, NodeSpec>; variables: PageVariable[]; queries: PageQuery[] };

/**
 * The schedule editor dialog. Opens while vars.<open> holds; edits the
 * schedule vars.<editing> names (an id), or a new one. With `fixed`, the
 * target is set (kind, ref, name bindings) and not shown.
 */
export function scheduleEditor(opts: { open: string; editing?: string; fixed?: { kind: string; ref: string; name: string } }): Fragment {
  const D = 'vars.schedDraft';
  const START = 'queries.schedStart.data';
  const close = [set(opts.open, false), set('schedDraft')];
  const TIMETABLE = cond('form.mode', 'not_equals', 'external');
  const EXTERNAL = cond('form.mode', 'equals', 'external');
  const heading = (name: string, label: string, visibleWhen?: ConditionNode): FormFieldSpec => ({ name, kind: 'note', label, wide: true, visibleWhen });
  const fields: FormFieldSpec[] = [
    { name: 'name', kind: 'text', label: 'schedules.editor.name', wide: true },
    ...(opts.fixed ? [] : [
      { name: 'kind', kind: 'select', label: 'schedules.editor.kind', required: true, optionsFrom: { query: 'schedKinds', valueField: 'value', labelField: 'label' } },
      { name: 'ref', kind: 'select', label: 'schedules.editor.target', required: true, resetOn: ['kind'], readOnlyWhen: cond('form.kind', 'is_empty'),
        optionsFrom: { query: 'schedTargets', valueField: 'value', labelField: 'label' } },
    ] as FormFieldSpec[]),
    heading('_how', 'schedules.editor.mode'),
    { name: 'mode', kind: 'radio', label: '', wide: true, options: [{ value: 'timetable', label: 'schedules.modes.timetable' }, { value: 'external', label: 'schedules.modes.external' }] },
    { name: '_external', kind: 'note', label: 'schedules.editor.externalHelp', severity: 'info', wide: true, visibleWhen: EXTERNAL },
    { name: '_command', kind: 'note', label: '{{form.command}}', wide: true, visibleWhen: all(EXTERNAL, cond('form.command', 'is_not_empty')) },
    heading('_when', 'schedules.editor.when'),
    { name: 'preset', kind: 'select', label: 'schedules.editor.repeat', required: true, visibleWhen: TIMETABLE, optionsFrom: { query: 'schedChoices', rowsPath: 'presets', valueField: 'value', labelField: 'label' } },
    { name: 'cron', kind: 'text', label: 'schedules.editor.cron', helperText: 'schedules.editor.cronHelp', visibleWhen: all(TIMETABLE, cond('form.preset', 'equals', 'custom')) },
    { name: 'time', kind: 'time', label: 'schedules.editor.time', visibleWhen: all(TIMETABLE, cond('form.preset', 'not_equals', 'custom')) },
    { name: 'weekday', kind: 'select', label: 'schedules.editor.weekday', required: true, visibleWhen: all(TIMETABLE, cond('form.preset', 'equals', 'weekly')),
      optionsFrom: { query: 'schedChoices', rowsPath: 'weekdays', valueField: 'value', labelField: 'label' } },
    { name: 'bd', kind: 'number', label: 'schedules.editor.businessDay', helperText: 'schedules.editor.businessDayHelp', step: 1,
      visibleWhen: all(TIMETABLE, cond('form.preset', 'equals', 'monthly_bd')) },
    { name: 'zone', kind: 'select', label: 'schedules.editor.timeZone', required: true, optionsFrom: { query: 'schedChoices', rowsPath: 'zones', valueField: 'value', labelField: 'label' } },
    heading('_cal', 'schedules.editor.calendar'),
    { name: 'calendar', kind: 'select', label: 'schedules.editor.calendar', required: true, optionsFrom: { query: 'schedCalendars', valueField: 'value', labelField: 'label' } },
    { name: 'rule', kind: 'radio', label: '', readOnlyWhen: cond('form.calendar', 'is_empty'), visibleWhen: all(TIMETABLE, cond('form.preset', 'not_equals', 'monthly_bd')),
      options: [{ value: 'none', label: 'schedules.rules.none' }, { value: 'skip', label: 'schedules.rules.skip' }, { value: 'next_business_day', label: 'schedules.rules.next_business_day' }] },
    { name: 'rule_ext', kind: 'radio', label: '', readOnlyWhen: cond('form.calendar', 'is_empty'), visibleWhen: EXTERNAL,
      options: [{ value: 'none', label: 'schedules.rules.none' }, { value: 'skip', label: 'schedules.rules.skip' }] },
    { name: '_pick', kind: 'note', label: 'schedules.editor.pickCalendar', severity: 'info', wide: true, visibleWhen: cond('queries.schedPreview.data.needs_calendar', 'is_true') },
    heading('_next', 'schedules.editor.nextRuns', TIMETABLE),
  ];
  const components: Record<string, ComponentDefinition> = {
    sched_form: {
      id: 'sched_form', type: 'Form', props: {
        variable: 'schedDraft', fields, initFrom: `{{${START}.draft}}`, seedKey: `{{${START}.key}}`,
        // The custom timetable follows the preset until it is edited.
        onChange: [op('schedules.normalize', { draft: '{{form}}' }, [set('schedDraft', '{{result}}')])],
      },
    },
    sched_next: {
      id: 'sched_next', type: 'DataGrid', visibleWhen: cond(`${D}.mode`, 'not_equals', 'external'), props: {
        query: 'schedPreview', rowsPath: 'rows', emptyText: '—',
        columns: [col('run', '', [
          { kind: 'chip', value: '{{row.action}}', label: '{{row.action_label}}', colorMap: { run: 'success', wait: 'warning', '*': 'default' }, variant: 'outlined' },
          { kind: 'text', value: '{{row.when}}', strike: '{{row.skipped}}' },
          { kind: 'text', value: '{{row.moved_to}}', color: 'warning', visibleWhen: cond('row.moved_to', 'is_not_empty') },
          { kind: 'chip', label: 'schedules.editor.halfDay', visibleWhen: cond('row.half_day', 'is_true') },
          { kind: 'text', value: '{{row.reason}}', caption: true, visibleWhen: cond('row.reason', 'is_not_empty') },
        ], { stackDirection: 'row' })],
      },
    },
    sched_enabled: {
      id: 'sched_enabled', type: 'Form', props: { variable: 'schedDraft', fields: [{ name: 'enabled', kind: 'switch', label: 'schedules.editor.enabled' }] },
    },
  };
  const nodes: Record<string, NodeSpec> = {
    sched_dialog: {
      type: 'Dialog', children: ['sched_body'], props: {
        title: `{{${START}.title}}`, maxWidth: 'md', openWhen: cond(`vars.${opts.open}`, 'is_true'), onClose: close,
        buttons: [
          { label: 'schedules.cancel', onClick: close },
          {
            label: 'schedules.save', variant: 'contained',
            disabledWhen: any(cond(`${D}.name`, 'is_empty'), cond(`${D}.kind`, 'is_empty'), cond(`${D}.ref`, 'is_empty'), cond('queries.schedPreview.data.needs_calendar', 'is_true')),
            onClick: [op('schedules.save', { draft: `{{${D}}}` }, close)],
          },
        ],
      },
    },
    sched_body: { type: 'Column', children: ['sched_form', 'sched_next', 'sched_enabled'], style: { gap: '12px' } },
  };
  const ED = opts.editing ? `{{vars.${opts.editing}}}` : '';
  return {
    components, nodes,
    variables: [
      { name: opts.open, default: false },
      ...(opts.editing ? [{ name: opts.editing, description: 'The schedule being edited; empty = a new one' }] : []),
      { name: 'schedDraft', description: 'The schedule being edited' },
    ],
    queries: [
      {
        id: 'schedStart', operation: 'schedules.editorStart', enabledWhen: cond(`vars.${opts.open}`, 'is_true'),
        params: opts.fixed ? { ...opts.fixed, open: `{{vars.${opts.open}}}` } : { id: ED, open: `{{vars.${opts.open}}}` },
      },
      { id: 'schedKinds', operation: 'schedules.kinds', params: {} },
      { id: 'schedTargets', operation: 'schedules.targets', params: { kind: `{{${D}.kind}}` }, enabledWhen: cond(`vars.${opts.open}`, 'is_true'), keepPrevious: true },
      { id: 'schedCalendars', operation: 'schedules.calendars', params: {}, enabledWhen: cond(`vars.${opts.open}`, 'is_true') },
      { id: 'schedChoices', operation: 'schedules.choices', params: { zone: `{{${D}.zone}}` }, keepPrevious: true },
      { id: 'schedPreview', operation: 'schedules.preview', params: { draft: `{{${D}}}` }, debounceMs: 350, keepPrevious: true, enabledWhen: cond(`vars.${opts.open}`, 'is_true') },
    ],
  };
}

// --- the console ---------------------------------------------------------------------

function schedulesPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const w = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => { components[id] = { id, type, props, ...extra }; };
  const openEditor = (id: unknown): Action[] => [set('editing', id), set('editorOpen', true)];

  w('hdr', 'PageHeader', { icon: 'schedule', title: 'schedules.title', subtitle: 'schedules.subtitle' }, { style: { flex: '1 1 320px' } });
  w('new_btn', 'ActionButton', { label: 'schedules.new', icon: 'add', variant: 'contained', onClick: openEditor('') }, { style: fit });

  // Schedules.
  const FILTERED = any(cond('vars.q', 'is_not_empty'), cond('vars.kind', 'is_not_empty'), cond('vars.state', 'is_not_empty'));
  w('q', 'SearchInput', { variable: 'q', placeholder: 'schedules.filters.searchSchedules', maxWidth: 520 }, { style: { flex: '1 1 260px' } });
  w('kind', 'VariableSelect', { variable: 'kind', label: '', emptyLabel: 'schedules.filters.allKinds', minWidth: 170, optionsFrom: { query: 'kinds', valueField: 'value', labelField: 'label' } }, { style: fit });
  w('state', 'VariableSelect', {
    variable: 'state', label: '', emptyLabel: 'schedules.filters.allStates', minWidth: 170,
    options: [{ value: 'active', label: 'schedules.filters.active' }, { value: 'paused', label: 'schedules.filters.paused' }],
  }, { style: fit });
  w('clear', 'ActionButton', { label: 'schedules.filters.clear', variant: 'text', onClick: [set('q', ''), set('kind', ''), set('state', '')] },
    { style: fit, visibleWhen: FILTERED });
  w('grid', 'DataGrid', {
    query: 'schedules', rowsPath: 'rows', emptyText: '{{queries.schedules.data.empty_text}}',
    columns: [
      col('name', 'schedules.columns.name', [{ kind: 'text', value: '{{row.name}}', bold: true }, { kind: 'text', value: '{{row.updated_text}}', caption: true }]),
      col('runs', 'schedules.columns.runs', { kind: 'chip', value: '{{row.kind_label}}', variant: 'outlined' }),
      col('when', 'schedules.columns.when', { kind: 'text', value: '{{row.when_text}}' }),
      col('enabled', 'schedules.columns.enabled', {
        kind: 'toggle', value: '{{row.enabled}}', label: 'schedules.columns.enabled',
        onChange: [op('schedules.setActive', { id: '{{row.id}}', active: '{{value}}' })],
      }),
      col('act', '', {
        kind: 'actions', buttons: [
          { label: 'schedules.runNow', icon: 'play', onClick: [op('schedules.runNow', { id: '{{row.id}}' }, [set('tab', 'runs')])] },
          { label: 'schedules.edit', icon: 'edit', onClick: openEditor('{{row.id}}') },
          {
            label: 'schedules.delete', icon: 'delete',
            onClick: [op('schedules.remove', { id: '{{row.id}}' }, [], {
              confirm: { title: 'schedules.delete', text: { t: 'schedules.confirmDelete', params: { name: '{{row.name}}' } }, confirmLabel: 'schedules.delete' },
            })],
          },
        ],
      }, { align: 'right', nowrap: true }),
    ] satisfies ColumnDef[],
  });

  // Run history.
  const RUN_FILTERED = any(cond('vars.runQ', 'is_not_empty'), cond('vars.runStatus', 'is_not_empty'), cond('vars.runKind', 'is_not_empty'),
    cond('vars.runFrom', 'is_not_empty'), cond('vars.runTo', 'is_not_empty'));
  w('run_q', 'SearchInput', { variable: 'runQ', placeholder: 'schedules.filters.searchRuns', maxWidth: 420 }, { style: { flex: '1 1 220px' } });
  w('run_status', 'VariableSelect', {
    variable: 'runStatus', label: '', emptyLabel: 'schedules.filters.allStatuses', minWidth: 150,
    options: ['running', 'succeeded', 'failed', 'skipped'].map((x) => ({ value: x, label: `schedules.status.${x}` })),
  }, { style: fit });
  w('run_kind', 'VariableSelect', { variable: 'runKind', label: '', emptyLabel: 'schedules.filters.allKinds', minWidth: 170, optionsFrom: { query: 'kinds', valueField: 'value', labelField: 'label' } }, { style: fit });
  w('run_from', 'SearchInput', { variable: 'runFrom', variant: 'date', label: 'schedules.filters.from' }, { style: fit });
  w('run_to', 'SearchInput', { variable: 'runTo', variant: 'date', label: 'schedules.filters.to' }, { style: fit });
  w('run_clear', 'ActionButton', {
    label: 'schedules.filters.clear', variant: 'text', onClick: [set('runQ', ''), set('runStatus', ''), set('runKind', ''), set('runFrom', ''), set('runTo', '')],
  }, { style: fit, visibleWhen: RUN_FILTERED });
  w('runs', 'DataGrid', {
    query: 'runs', rowsPath: 'rows', emptyText: '{{queries.runs.data.empty_text}}', progressOnFetch: true,
    columns: [
      col('when', 'schedules.runs.when', { kind: 'text', value: '{{row.when_text}}' }, { nowrap: true }),
      col('schedule', 'schedules.runs.schedule', [
        { kind: 'text', value: '{{row.schedule_name}}' },
        { kind: 'chips', value: '{{row.chips}}', tooltip: '{{row.key_text}}' } as CellSpec,
      ], { stackDirection: 'row' }),
      col('status', 'schedules.runs.status', { kind: 'chip', value: '{{row.status}}', label: '{{row.status_label}}', colorMap: { succeeded: 'success', failed: 'error', running: 'info', '*': 'default' } }),
      col('result', 'schedules.runs.result', [
        { kind: 'text', value: '{{row.result_text}}', color: 'error', visibleWhen: cond('row.tone', 'equals', 'error') },
        { kind: 'text', value: '{{row.result_text}}', visibleWhen: cond('row.tone', 'not_equals', 'error') },
      ]),
      col('dl', '', {
        kind: 'actions', buttons: [{ label: 'schedules.runs.download', icon: 'download', visibleWhen: cond('row.has_output', 'is_true'),
          onClick: [op('schedules.download', { id: '{{row.id}}' })] }],
      }, { align: 'right' }),
    ] satisfies ColumnDef[],
    // A failed run's reason, under it.
    rowDetail: { text: '{{row.reason}}', severity: 'error', when: cond('row.status', 'equals', 'failed') },
  });

  const ed = scheduleEditor({ open: 'editorOpen', editing: 'editing' });
  Object.assign(components, ed.components);
  const layout = (root: string, spec: Record<string, NodeSpec>): PageLayout => ({ root, nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])) });
  const tabs: PageTab[] = [
    {
      id: 'schedules', label: 'schedules.tabs.schedules',
      layout: layout('s_root', {
        s_root: { type: 'Column', children: ['s_filters', 'grid'] },
        s_filters: { type: 'Row', children: ['q', 'kind', 'state', 'clear'], style: { alignItems: 'center' } },
      }),
    },
    {
      id: 'runs', label: 'schedules.tabs.runs',
      layout: layout('r_root', {
        r_root: { type: 'Column', children: ['r_filters', 'runs'] },
        r_filters: { type: 'Row', children: ['run_q', 'run_status', 'run_kind', 'run_from', 'run_to', 'run_clear'], style: { alignItems: 'center' } },
      }),
    },
  ];
  const filterBar = layout('top_root', {
    top_root: { type: 'Column', children: ['top_header', 'sched_dialog'], style: { gap: '8px' } },
    top_header: { type: 'Row', children: ['hdr', 'new_btn'], style: { alignItems: 'center' } },
    ...ed.nodes,
  });
  return {
    name: 'Schedules',
    slug: 'schedules',
    description: 'The platform scheduler: reports, saved queries, pipelines and mastering on one timetable, with one run history and business calendars. Built in Page Studio.',
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
        { name: 'tab', default: 'schedules', url: true },
        { name: 'q', default: '' }, { name: 'kind', default: '' }, { name: 'state', default: '' },
        { name: 'runQ', default: '' }, { name: 'runStatus', default: '' }, { name: 'runKind', default: '' }, { name: 'runFrom', default: '' }, { name: 'runTo', default: '' },
        ...ed.variables,
      ],
      queries: [
        { id: 'kinds', operation: 'schedules.kinds', params: {} },
        { id: 'schedules', operation: 'schedules.list', params: { q: '{{vars.q}}', kind: '{{vars.kind}}', state: '{{vars.state}}' }, keepPrevious: true },
        {
          id: 'runs', operation: 'schedules.runs', keepPrevious: true,
          params: { q: '{{vars.runQ}}', status: '{{vars.runStatus}}', kind: '{{vars.runKind}}', from: '{{vars.runFrom}}', to: '{{vars.runTo}}' },
          enabledWhen: cond('vars.tab', 'equals', 'runs'), refetchWhile: cond('vars.tab', 'equals', 'runs'), refetchMs: 10000, debounceMs: 300,
        },
        ...ed.queries,
      ],
    },
  };
}

export const schedulesBlueprint = () => structuredClone(schedulesPage());
