import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, CellSpec, ColumnDef, ConditionNode } from '../appModel';

/**
 * Data pipelines and the pipeline editor as Page Studio pages (they replaced
 * the hand-built list and editor pages) - the ingest step of mastering. The
 * list is studio widgets over the dataPipelines.list operation. The editor is
 * built from blocks too: the graph on the Canvas widget, each step's settings
 * a Form shaped by the domain (fieldsFrom), problems / preview / runs in a
 * tab set; the domain's operations (features/data-pipelines/editorStudio.ts)
 * decide what steps are and do.
 */

type Spec = Record<string, { type: 'Row' | 'Column' | 'TabSet'; children: string[]; style?: Record<string, string>; props?: Record<string, unknown> }>;
const layout = (root: string, spec: Spec): PageLayout => ({ root, nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])) });
const fit = { flex: '0 0 auto' };

function listPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {
    hdr: {
      id: 'hdr', type: 'PageHeader', style: { flex: '1 1 320px' }, props: {
        icon: 'pipeline', title: 'Data pipelines',
        subtitle: 'Load files and business objects, check them against your rules, and write them to business objects, staging tables or files - no code.',
      },
    },
    new_btn: {
      id: 'new_btn', type: 'ActionButton', style: fit,
      props: { label: 'New pipeline', icon: 'add', variant: 'contained', onClick: [{ kind: 'navigate', to: '/data/pipelines/new' }] },
    },
    grid: {
      id: 'grid', type: 'DataGrid', props: {
        query: 'pipelines',
        emptyText: 'No pipelines yet. Start from a blank canvas with New pipeline, or describe what you want to the assistant.',
        onRowClick: [{ kind: 'navigate', to: '/data/pipelines/{{row.id}}' }],
        columns: [
          { id: 'name', header: 'Pipeline', cell: { kind: 'twoLine', primary: '{{row.name}}', secondary: '{{row.summary}}' } },
          { id: 'steps', header: 'Steps', align: 'right', cell: { kind: 'number', value: '{{row.steps}}' } },
          { id: 'edited', header: 'Edited', nowrap: true, cell: { kind: 'datetime', value: '{{row.last_modified_at}}' } },
          {
            id: 'act', align: 'right', nowrap: true, cell: {
              kind: 'actions', buttons: [
                { label: 'Open', onClick: [{ kind: 'navigate', to: '/data/pipelines/{{row.id}}' }] },
                {
                  label: 'Delete', color: 'error', onClick: [{
                    kind: 'runOperation', operation: 'dataPipelines.delete', params: { id: '{{row.id}}' },
                    confirm: { title: 'Delete "{{row.name}}"?', text: 'The pipeline is removed; its run history is kept.', confirmLabel: 'Delete' },
                    successMessage: 'Pipeline deleted',
                  }],
                },
              ],
            },
          },
        ] satisfies ColumnDef[],
      },
    },
  };
  const tabs: PageTab[] = [{
    id: 'pipelines', label: 'Pipelines',
    layout: layout('root', {
      root: { type: 'Column', children: ['top', 'grid'], style: { gap: '16px' } },
      top: { type: 'Row', children: ['hdr', 'new_btn'], style: { alignItems: 'center' } },
    }),
  }];
  return {
    name: 'Data pipelines',
    slug: 'data-pipelines',
    description: 'Every data pipeline: load files and business objects, check them, write them where mastering reads them. Built in Page Studio.',
    layout: tabs[0].layout,
    tabs,
    components,
    dataSources: [],
    status: 'draft' as const,
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 1200, padding: 3 },
      queries: [{ id: 'pipelines', operation: 'dataPipelines.list', params: {} }],
    },
  };
}

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });
const any = (...conditions: ConditionNode[]): ConditionNode => ({ type: 'group', operator: 'OR', conditions });
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const op = (operation: string, params: Record<string, unknown>, onSuccess: Action[] = [], more: Record<string, unknown> = {}): Action =>
  ({ kind: 'runOperation', operation, params, onSuccess, ...more } as Action);
const col = (id: string, header: string, cell: CellSpec | CellSpec[], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell, ...more };
const txt = (field: string, more: Record<string, unknown> = {}): CellSpec => ({ kind: 'text', value: `{{row.${field}}}`, ...more } as CellSpec);
/** Text coloured by the row's tone (error / warning). */
const toned = (field: string): CellSpec[] => [
  { kind: 'text', value: `{{row.${field}}}`, color: 'error', visibleWhen: cond('row.tone', 'equals', 'error') },
  { kind: 'text', value: `{{row.${field}}}`, color: 'warning', visibleWhen: cond('row.tone', 'not_equals', 'error') },
];

function editorPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const ID = '{{route.id}}';
  const NEW = cond('queries.load.data.is_new', 'is_true');
  const INVALID = cond('queries.check.data.valid', 'is_false');
  const edited = [set('dirty', true), set('preview')];
  const components: Record<string, ComponentDefinition> = {};
  const w = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => {
    components[id] = { id, type, props, ...extra };
  };

  // --- toolbar ---------------------------------------------------------------
  w('back', 'ActionButton', { label: 'Pipelines', icon: 'back', variant: 'text', onClick: [{ kind: 'navigate', to: '/data/pipelines' }] }, { style: fit });
  w('name', 'SearchInput', { variable: 'name', variant: 'title', label: 'pipeline name', onChange: [set('dirty', true)] }, { style: { flex: '1 1 260px' } });
  w('unsaved', 'ActionButton', { label: 'unsaved', variant: 'chip', onClick: [] }, { style: fit, visibleWhen: cond('vars.dirty', 'is_true') });
  w('status', 'ActionButton', {
    label: '{{queries.check.data.status_label}}', variant: 'chip', chipColor: '{{queries.check.data.status_color}}', onClick: [set('tab', 'problems')],
  }, { style: fit });
  w('assistant_open', 'ActionButton', { label: 'Assistant', icon: 'assistant', variant: 'outlined', color: 'secondary', onClick: [set('assistantOpen', true)] },
    { style: fit, visibleWhen: cond('vars.assistantOpen', 'is_false') });
  w('assistant_close', 'ActionButton', { label: 'Assistant', icon: 'assistant', variant: 'contained', color: 'secondary', onClick: [set('assistantOpen', false)] },
    { style: fit, visibleWhen: cond('vars.assistantOpen', 'is_true') });
  w('preview_btn', 'ActionButton', {
    label: 'Preview', icon: 'preview', variant: 'text', disabledWhen: INVALID,
    tooltip: '{{queries.check.data.preview_tip}}',
    onClick: [op('dataPipelines.preview', { spec: '{{vars.spec}}' }, [set('preview', '{{result}}'), set('previewNode', ''), set('tab', 'preview')])],
  }, { style: fit });
  w('save_btn', 'ActionButton', {
    label: 'Save', icon: 'save', variant: 'text', disabledWhen: cond('vars.dirty', 'is_false'),
    onClick: [op('dataPipelines.save', { id: ID, name: '{{vars.name}}', spec: '{{vars.spec}}' }, [
      set('dirty', false),
      { kind: 'notify', severity: 'success', text: 'Saved' },
      { kind: 'navigate', to: '/data/pipelines/{{result.id}}', when: cond('result.created', 'is_true') },
    ])],
  }, { style: fit });
  w('schedule_btn', 'ActionButton', {
    label: '{{queries.schedule.data.label}}', icon: 'schedule', variant: 'text', disabledWhen: NEW,
    tooltip: '{{queries.check.data.schedule_tip}}', onClick: [set('scheduleOpen', true)],
  }, { style: fit });
  w('run_btn', 'ActionButton', {
    label: 'Run', icon: 'play', variant: 'contained', tooltip: '{{queries.check.data.run_tip}}',
    disabledWhen: any(NEW, INVALID, cond('vars.activeRun', 'is_not_empty')),
    onClick: [op('dataPipelines.startRun', { id: ID, name: '{{vars.name}}', spec: '{{vars.spec}}', dirty: '{{vars.dirty}}' },
      [set('activeRun', '{{result.run_id}}'), set('dirty', false), set('tab', 'runs')])],
  }, { style: fit, visibleWhen: cond('vars.activeRun', 'is_empty') });
  w('running_btn', 'ActionButton', { label: 'Running…', icon: 'play', variant: 'contained', disabledWhen: cond('vars.activeRun', 'is_not_empty'), onClick: [] },
    { style: fit, visibleWhen: cond('vars.activeRun', 'is_not_empty') });

  // --- canvas ------------------------------------------------------------------
  w('canvas', 'Canvas', {
    variable: 'spec', selectedVariable: 'selected', singleInput: true, height: '100%',
    node: {
      title: '{{extra.title}}', subtitle: '{{extra.summary}}', placeholder: 'Click to configure', icon: '{{extra.icon}}',
      category: '{{extra.category}}', errors: '{{extra.issues}}', chips: '{{extra.chips}}',
    },
    nodeData: '{{queries.nodes.data}}',
    categories: { source: { color: 'info', inputs: false }, step: { color: 'secondary' }, destination: { color: 'success', outputs: false } },
    animatedWhen: cond('vars.preview', 'is_not_empty'),
    palette: {
      query: 'kinds', groupBy: 'category', label: '{{item.label}}', description: '{{item.description}}', icon: '{{item.icon}}', category: '{{item.category}}',
      groups: [{ id: 'source', label: 'Read from' }, { id: 'step', label: 'Check & shape' }, { id: 'destination', label: 'Write to' }],
      disabledWhen: cond('item.available', 'is_false'),
    },
    onAdd: [op('dataPipelines.addStep', { item: '{{item}}', spec: '{{graph}}', selected: '{{selected}}' },
      [set('spec', '{{result.spec}}'), set('selected', '{{result.id}}'), ...edited])],
    onChange: [set('dirty', true)],
    emptyText: 'Start with a source on the left',
    emptyHint: '…or open the Assistant and describe what you want to load.',
  }, { style: { flex: '1 1 0' } });

  // --- the selected step's settings --------------------------------------------
  w('step_type', 'TextBlock', { text: '{{queries.stepForm.data.type_label}}', variant: 'overline' }, { style: { flex: '1 1 auto' } });
  w('step_close', 'ActionButton', { label: 'Close', variant: 'text', onClick: [set('selected')] }, { style: fit });
  w('step_form', 'Form', {
    variable: 'stepDraft', fieldsFrom: '{{queries.stepForm.data.fields}}', initFrom: '{{queries.stepForm.data.initial}}',
    // The step on show, from the form's own data (kept while the next one loads).
    seedKey: '{{queries.stepForm.data.id}}',
    onChange: [op('dataPipelines.applyStep', { spec: '{{vars.spec}}', id: '{{queries.stepForm.data.id}}', form: '{{form}}', info: '{{vars.stepInfo}}' },
      [set('spec', '{{result.spec}}'), set('stepDraft', '{{result.form}}'), ...edited])],
  });

  // --- the assistant and the schedule (domain components for now) ----------------
  w('assistant', 'DomainComponent', {
    component: 'dataPipelines.Assistant', inputs: { spec: '{{vars.spec}}', selected: '{{vars.selected}}', preview: '{{vars.preview}}' },
    events: { apply: [set('spec', '{{event.spec}}'), set('selected', '{{event.focus}}'), ...edited], close: [set('assistantOpen', false)] },
  }, { style: { flex: '0 0 380px' }, visibleWhen: cond('vars.assistantOpen', 'is_true') });
  w('schedule', 'DomainComponent', {
    component: 'schedules.TargetSchedule', inputs: { kind: 'data_pipeline', ref: ID, name: '{{vars.name}}', open: '{{vars.scheduleOpen}}' },
    events: { close: [set('scheduleOpen', false)] },
  }, { style: fit });

  // --- problems, preview, runs ------------------------------------------------------
  w('problems', 'DataGrid', {
    query: 'check', rowsPath: 'issues', emptyText: '{{queries.check.data.empty_text}}',
    onRowClick: [{ ...set('selected', '{{row.node_id}}'), when: cond('row.node_id', 'is_not_empty') }],
    columns: [col('message', '', [txt('message'), txt('step', { caption: true })])],
  });
  const PV = 'queries.previewView.data';
  w('pv_none', 'TextBlock', { text: 'Press Preview to run the pipeline on the first 100 rows without writing anything.', color: 'text.secondary' },
    { visibleWhen: cond(`${PV}.state`, 'equals', 'none') });
  w('pv_error', 'AlertBanner', { severity: 'error', text: `{{${PV}.error}}` }, { visibleWhen: cond(`${PV}.error`, 'is_not_empty') });
  w('pv_totals', 'KeyValue', { source: `{{${PV}}}`, columns: 1, items: [{ label: '', cell: { kind: 'chips', value: '{{data.totals}}' } }] },
    { visibleWhen: cond(`${PV}.state`, 'equals', 'done') });
  w('pv_steps', 'VariableSelect', { variable: 'previewNode', variant: 'toggle', optionsFrom: { query: 'previewView', rowsPath: 'steps', valueField: 'value', labelField: 'label' } },
    { visibleWhen: cond(`${PV}.state`, 'equals', 'done') });
  w('pv_rejects', 'DataGrid', {
    query: 'previewView', rowsPath: 'rejects',
    columns: [col('row', 'Row', txt('row')), col('step', 'Step', txt('step')), col('why', 'Why', toned('reason'))],
  }, { visibleWhen: cond(`${PV}.rejects.length`, 'greater_than', 0) });
  w('pv_caption', 'TextBlock', { text: `{{${PV}.sample_caption}}`, variant: 'caption', color: 'text.secondary' },
    { visibleWhen: cond(`${PV}.sample_caption`, 'is_not_empty') });
  w('pv_rows', 'DataGrid', {
    query: 'previewView', rowsPath: 'rows', columns: [col('num', '#', txt('num'))],
    dynamicColumns: { from: `{{${PV}.columns}}`, idField: 'code', header: '{{col.code}}', valuePath: 'cells', cell: { kind: 'text', value: '{{value}}' } },
  }, { visibleWhen: cond(`${PV}.rows.length`, 'greater_than', 0) });
  w('runs', 'DataGrid', {
    query: 'runs', rowsPath: 'runs', emptyText: 'No runs yet.', onRowClick: [set('openRun', '{{row.id}}')],
    columns: [col('run', '', [
      { kind: 'chip', value: '{{row.status_label}}', colorBy: '{{row.status_color}}',
        colorMap: { success: 'success', warning: 'warning', error: 'error', info: 'info', '*': 'default' } },
      txt('started'), txt('counts', { caption: true }),
    ])],
  }, { style: { flex: '0 0 320px' } });
  const RD = 'queries.runDetail.data';
  w('run_steps', 'DataGrid', {
    query: 'runDetail', rowsPath: 'steps',
    columns: [
      col('step', 'Step', txt('step')), col('in', 'In', txt('in')), col('out', 'Out', txt('out')), col('rejected', 'Rejected', txt('rejected')),
      col('time', 'Time', txt('time')), col('error', '', { kind: 'text', value: '{{row.error}}', caption: true, color: 'error' }),
    ],
  });
  w('run_mastering', 'DataGrid', {
    query: 'runDetail', rowsPath: 'mastering',
    columns: [
      col('m', '', [
        { kind: 'chip', value: '{{row.label}}', colorBy: '{{row.color}}', colorMap: { success: 'success', warning: 'warning', error: 'error' } },
        txt('text'),
      ], { stackDirection: 'row' }),
      col('open', '', { kind: 'actions', buttons: [{ label: '{{row.link_label}}', onClick: [{ kind: 'navigate', to: '/data/mastering' }] }] }, { align: 'right' }),
    ],
  }, { visibleWhen: cond(`${RD}.mastering.length`, 'greater_than', 0) });
  w('run_errors', 'DataGrid', { query: 'runDetail', rowsPath: 'errors', columns: [col('e', '', toned('text'))] },
    { visibleWhen: cond(`${RD}.errors.length`, 'greater_than', 0) });

  const nowrap = { flexWrap: 'nowrap', gap: '0' };
  const tabs: PageTab[] = [{
    id: 'editor', label: 'Editor',
    layout: layout('root', {
      root: { type: 'Column', children: ['toolbar', 'main', 'schedule'], style: { ...nowrap, height: 'calc(100vh - 64px)' } },
      toolbar: {
        type: 'Row', children: ['back', 'name', 'unsaved', 'status', 'actions'],
        style: { alignItems: 'center', gap: '8px', padding: '8px 16px', borderBottom: '1px solid rgba(128,128,128,0.3)' },
      },
      actions: { type: 'Row', children: ['assistant_open', 'assistant_close', 'preview_btn', 'save_btn', 'schedule_btn', 'run_btn', 'running_btn'],
        style: { alignItems: 'center', gap: '8px', marginLeft: 'auto', flex: '0 0 auto' } },
      main: { type: 'Row', children: ['center', 'step_panel', 'assistant'], style: { ...nowrap, flex: '1 1 0', minHeight: '0' } },
      center: { type: 'Column', children: ['canvas', 'bottom'], style: { ...nowrap, flex: '1 1 0', minWidth: '280px' } },
      bottom: {
        type: 'Column', children: ['bottom_tabs'],
        style: { ...nowrap, height: '280px', flex: '0 0 280px', overflowY: 'auto', padding: '0 8px', borderTop: '1px solid rgba(128,128,128,0.3)' },
      },
      bottom_tabs: {
        type: 'TabSet', children: ['tab_problems', 'tab_preview', 'tab_runs'],
        props: {
          variable: 'tab',
          tabs: [
            { id: 'problems', label: 'Problems', badge: '{{queries.check.data.count}}' },
            { id: 'preview', label: 'Preview' },
            { id: 'runs', label: 'Runs', visibleWhen: cond('queries.load.data.is_new', 'is_false') },
          ],
        },
      },
      tab_problems: { type: 'Column', children: ['problems'] },
      tab_preview: { type: 'Column', children: ['pv_none', 'pv_error', 'pv_totals', 'pv_steps', 'pv_rejects', 'pv_caption', 'pv_rows'], style: { gap: '8px' } },
      tab_runs: { type: 'Row', children: ['runs', 'run_detail'], style: { ...nowrap, gap: '8px', alignItems: 'flex-start' } },
      run_detail: { type: 'Column', children: ['run_steps', 'run_mastering', 'run_errors'], style: { gap: '8px', flex: '1 1 0', minWidth: '0' } },
      step_panel: {
        type: 'Column', children: ['step_head', 'step_form'],
        props: { visibleWhen: cond('vars.selected', 'is_not_empty') },
        style: { ...nowrap, flex: '0 0 420px', overflowY: 'auto', padding: '8px 16px 16px', gap: '8px', borderLeft: '1px solid rgba(128,128,128,0.3)' },
      },
      step_head: { type: 'Row', children: ['step_type', 'step_close'], style: { alignItems: 'center', ...nowrap } },
    }),
  }];
  return {
    name: 'Data pipeline editor',
    slug: 'data-pipeline-editor',
    description: 'The visual pipeline editor (/data/pipelines/:id): canvas, step settings, problems, preview and runs - built in Page Studio (Canvas widget, forms shaped by the domain).',
    layout: tabs[0].layout,
    tabs,
    components,
    dataSources: [],
    status: 'draft' as const,
    app: {
      chrome: 'none' as const,
      surface: { padding: 0 },
      variables: [
        { name: 'name', initFrom: { query: 'load', path: 'name' }, description: 'The pipeline\'s name' },
        { name: 'spec', initFrom: { query: 'load', path: 'spec' }, description: 'The pipeline: nodes (steps) and edges' },
        { name: 'dirty', default: false, description: 'Unsaved changes' },
        { name: 'selected', description: 'The step whose settings show' },
        { name: 'stepDraft', description: 'The selected step\'s settings form' },
        { name: 'stepInfo', description: 'What the selected step shows beside its form (a file read, a suggestion)' },
        { name: 'tab', default: 'problems' },
        { name: 'preview', description: 'The last preview (cleared by an edit)' },
        { name: 'previewNode', default: '', description: 'The step whose rows the preview shows; empty = all' },
        { name: 'activeRun', description: 'A run being followed' },
        { name: 'openRun', description: 'The run shown in detail' },
        { name: 'assistantOpen', default: false },
        { name: 'scheduleOpen', default: false },
      ],
      queries: [
        { id: 'load', operation: 'dataPipelines.load', params: { id: ID } },
        { id: 'kinds', operation: 'dataPipelines.stepKinds', params: {} },
        { id: 'check', operation: 'dataPipelines.check', params: { spec: '{{vars.spec}}', is_new: '{{queries.load.data.is_new}}' }, debounceMs: 400, keepPrevious: true },
        {
          id: 'nodes', operation: 'dataPipelines.nodeView', keepPrevious: true,
          params: { spec: '{{vars.spec}}', kinds: '{{queries.kinds.data}}', issues: '{{queries.check.data.issues}}', preview: '{{vars.preview}}' },
        },
        {
          id: 'stepForm', operation: 'dataPipelines.stepForm', keepPrevious: true, enabledWhen: cond('vars.selected', 'is_not_empty'),
          params: { spec: '{{vars.spec}}', id: '{{vars.selected}}', info: '{{vars.stepInfo}}', kinds: '{{queries.kinds.data}}', issues: '{{queries.check.data.issues}}' },
        },
        { id: 'previewView', operation: 'dataPipelines.previewView', params: { preview: '{{vars.preview}}', step: '{{vars.previewNode}}', spec: '{{vars.spec}}' } },
        {
          id: 'runs', operation: 'dataPipelines.runs', params: { id: ID, active: '{{vars.activeRun}}' },
          refetchWhile: cond('vars.activeRun', 'is_not_empty'),
          onChange: [
            { kind: 'notify', severity: 'info', text: '{{data.done_text}}', when: cond('data.active_done', 'is_true') },
            { ...set('activeRun'), when: cond('data.active_done', 'is_true') },
          ],
        },
        { id: 'runDetail', operation: 'dataPipelines.runDetail', params: { run: '{{vars.openRun}}', spec: '{{vars.spec}}' }, enabledWhen: cond('vars.openRun', 'is_not_empty') },
        { id: 'schedule', operation: 'schedules.forTarget', params: { kind: 'data_pipeline', ref: ID } },
      ],
    },
  };
}

export const dataPipelinesBlueprint = () => structuredClone(listPage());
export const dataPipelineEditorBlueprint = () => structuredClone(editorPage());
