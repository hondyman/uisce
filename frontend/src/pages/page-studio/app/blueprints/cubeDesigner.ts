import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, ConditionNode, FormFieldSpec } from '../appModel';

/**
 * Cube designer host (slug cube-designer): composed Page Studio page for
 * /build/cubes/new and /build/cubes/:id. Tabs + Form/ops + cubes.FederationEditor
 * + cubes.ImpactPanel DomainComponents — no full-page cubes.Designer DC.
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
const any = (...conditions: ConditionNode[]): ConditionNode =>
  ({ type: 'group', operator: 'OR', conditions });

const fit = { flex: '0 0 auto' };

type NodeSpec = {
  type: 'Row' | 'Column' | 'TabSet';
  children: string[];
  style?: Record<string, string>;
  props?: Record<string, unknown>;
};
const layout = (root: string, spec: Record<string, NodeSpec>): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

function cubeDesignerPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const ID = '{{route.id}}';
  const NEW = cond('queries.load.data.is_new', 'is_true');
  const EXISTING = cond('queries.load.data.is_new', 'is_false');
  const dirty = [set('dirty', true)];

  const components: Record<string, ComponentDefinition> = {};
  const w = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => {
    components[id] = { id, type, props, ...extra };
  };

  // --- toolbar -----------------------------------------------------------------
  w('back', 'ActionButton', {
    label: 'Cubes', icon: 'back', variant: 'text',
    onClick: [{ kind: 'navigate', to: '/build/cubes' }],
  }, { style: fit });

  w('title', 'TextBlock', {
    text: '{{queries.load.data.title}}',
    variant: 'h6',
  }, { style: { flex: '1 1 240px' } });

  w('unsaved', 'ActionButton', {
    label: 'unsaved', variant: 'chip', onClick: [],
  }, { style: fit, visibleWhen: cond('vars.dirty', 'is_true') });

  w('info_banner', 'AlertBanner', {
    severity: 'info', text: '{{vars.info}}',
  }, { visibleWhen: cond('vars.info', 'is_not_empty') });

  w('error_banner', 'AlertBanner', {
    severity: 'error', text: '{{vars.error}}',
  }, { visibleWhen: cond('vars.error', 'is_not_empty') });

  w('validate_btn', 'ActionButton', {
    label: 'Validate', icon: 'check', variant: 'text',
    disabledWhen: NEW,
    tooltip: 'Save the cube first, then validate (API requires an id).',
    onClick: [
      op(
        'cubes.validate',
        {
          id: ID,
          draft: '{{vars.draft}}',
          federationKeySamples: '{{vars.draft.federationKeySamples}}',
        },
        [
          set('validation', '{{result}}'),
          set('error'),
          {
            kind: 'notify',
            severity: 'success',
            text: 'Validation ok',
            when: cond('result.ok', 'is_true'),
          },
          {
            kind: 'notify',
            severity: 'warning',
            text: 'Validation reported issues — see federation/structural flags on the result',
            when: cond('result.ok', 'is_false'),
          },
        ],
      ),
    ],
  }, { style: fit, visibleWhen: EXISTING });

  w('deploy_btn', 'ActionButton', {
    label: 'Deploy', icon: 'play', variant: 'text',
    disabledWhen: any(NEW, cond('vars.dirty', 'is_true')),
    onClick: [
      op(
        'cubes.deploy',
        { id: ID, force: false },
        [{ kind: 'notify', severity: 'success', text: 'Deploy started' }],
        {
          confirm: {
            title: 'Deploy this cube?',
            text: 'Start materialization for the current contract (force=false).',
            confirmLabel: 'Deploy',
          },
        },
      ),
    ],
  }, { style: fit, visibleWhen: EXISTING });

  w('refresh_btn', 'ActionButton', {
    label: 'Refresh', icon: 'refresh', variant: 'text',
    disabledWhen: any(NEW, cond('vars.dirty', 'is_true')),
    onClick: [
      op(
        'cubes.refresh',
        { id: ID, force: false },
        [{ kind: 'notify', severity: 'success', text: 'Refresh requested' }],
        {
          confirm: {
            title: 'Refresh materialization?',
            text: 'force=false may no-op when already dual-committed.',
            confirmLabel: 'Refresh',
          },
        },
      ),
    ],
  }, { style: fit, visibleWhen: EXISTING });

  w('impact_btn', 'ActionButton', {
    label: 'Impact', icon: 'insights', variant: 'text',
    tooltip: 'Composition, consumers, and dry-run change preview',
    onClick: [set('tab', 'impact')],
  }, { style: fit, visibleWhen: EXISTING });

  w('save_create', 'ActionButton', {
    label: 'Save', icon: 'save', variant: 'contained',
    disabledWhen: cond('vars.dirty', 'is_false'),
    onClick: [
      op('cubes.create', { draft: '{{vars.draft}}' }, [
        set('dirty', false),
        set('draft', '{{result.draft}}'),
        set('error'),
        { kind: 'notify', severity: 'success', text: 'Cube created' },
        { kind: 'navigate', to: '/build/cubes/{{result.id}}' },
      ]),
    ],
  }, { style: fit, visibleWhen: NEW });

  w('save_patch', 'ActionButton', {
    label: 'Save', icon: 'save', variant: 'contained',
    disabledWhen: cond('vars.dirty', 'is_false'),
    onClick: [
      op('cubes.patch', { id: ID, draft: '{{vars.draft}}' }, [
        set('dirty', false),
        set('draft', '{{result.draft}}'),
        set('error'),
        { kind: 'notify', severity: 'success', text: 'Saved' },
      ]),
    ],
  }, { style: fit, visibleWhen: EXISTING });

  // --- overview ----------------------------------------------------------------
  const overviewFields: FormFieldSpec[] = [
    { name: 'name', kind: 'text', label: 'Name', required: true },
    { name: 'description', kind: 'multiline', label: 'Description' },
    {
      name: 'boId',
      kind: 'select',
      label: 'Business Object',
      required: true,
      optionsFrom: { query: 'bos', rowsPath: 'rows', valueField: 'key', labelField: 'displayName' },
    },
  ];
  w('overview_form', 'Form', {
    variable: 'draft',
    fields: overviewFields,
    initFrom: '{{queries.load.data.draft}}',
    seedKey: '{{queries.load.data.key}}',
    onChange: dirty,
  });

  w('overview_hint', 'TextBlock', {
    text: 'Logical BO key is stored on the cube. Changing BO clears authored axes in the coded designer; here clear Dimensions / Metrics / Grains tabs after a BO change.',
    variant: 'caption',
    color: 'text.secondary',
  });

  // --- dimensions --------------------------------------------------------------
  w('dims_help', 'TextBlock', {
    text: 'Ordered dimension surface. Each axis must appear in at least one grain. Choose terms from the selected Business Object.',
    color: 'text.secondary',
  });
  w('dims_form', 'Form', {
    variable: 'draft',
    fields: [
      {
        name: 'dimensions',
        kind: 'rows',
        label: 'Dimensions',
        addLabel: 'Add dimension',
        rowFields: [
          {
            name: 'termNodeId',
            kind: 'select',
            label: 'Dimension Term',
            optionsFrom: {
              query: 'dimensions',
              rowsPath: 'rows',
              valueField: 'termNodeId',
              labelField: 'displayName',
            },
          },
        ],
      },
    ] satisfies FormFieldSpec[],
    onChange: dirty,
  });

  // --- metrics -----------------------------------------------------------------
  w('metrics_help', 'TextBlock', {
    text: 'Governed metrics only (metric_definition). Options follow the selected Business Object.',
    color: 'text.secondary',
  });
  w('metrics_form', 'Form', {
    variable: 'draft',
    fields: [
      {
        name: 'metricIds',
        kind: 'chips',
        label: 'Governed Metrics',
        optionsFrom: {
          query: 'metrics',
          rowsPath: 'rows',
          valueField: 'id',
          labelField: 'name',
        },
      },
    ] satisfies FormFieldSpec[],
    onChange: dirty,
  });

  // --- grains ------------------------------------------------------------------
  w('grains_help', 'TextBlock', {
    text: 'Each grain is one materialization shape: an array of dimension term ids (JSON array of string arrays).',
    color: 'text.secondary',
  });
  w('grains_form', 'Form', {
    variable: 'draft',
    fields: [
      { name: 'grains', kind: 'json', label: 'Grains', wide: true },
    ] satisfies FormFieldSpec[],
    onChange: dirty,
  });

  // --- federation DomainComponent ----------------------------------------------
  w('fed_dc', 'DomainComponent', {
    component: 'cubes.FederationEditor',
    inputs: {
      primaryBoId: '{{vars.draft.boId}}',
      bos: '{{queries.bos.data.bos}}',
      federation: '{{vars.draft.federation}}',
      keySamples: '{{vars.draft.federationKeySamples}}',
    },
    events: {
      onChange: [
        op(
          'cubes.patchDraft',
          { draft: '{{vars.draft}}', federation: '{{event.federation}}' },
          [set('draft', '{{result.draft}}'), set('dirty', true)],
        ),
      ],
      onKeySamplesChange: [
        op(
          'cubes.patchDraft',
          { draft: '{{vars.draft}}', federationKeySamples: '{{event.keySamples}}' },
          [set('draft', '{{result.draft}}'), set('dirty', true)],
        ),
      ],
    },
  });

  // --- materialization ---------------------------------------------------------
  w('mat_form', 'Form', {
    variable: 'mat',
    initFrom: '{{vars.draft.materialization}}',
    seedKey: '{{queries.load.data.key}}',
    fields: [
      {
        name: 'strategy',
        kind: 'select',
        label: 'Strategy',
        options: [
          { value: 'starrocks_mv', label: 'starrocks_mv' },
          { value: 'aggregate_table', label: 'aggregate_table' },
        ],
      },
      {
        name: 'stalePolicy',
        kind: 'select',
        label: 'Stale policy',
        options: [
          { value: 'serve_with_flag', label: 'serve_with_flag' },
          { value: 'force_raw_fallback', label: 'force_raw_fallback' },
        ],
      },
      {
        name: 'refreshStrategy',
        kind: 'select',
        label: 'Refresh strategy',
        options: [
          { value: 'manual', label: 'manual' },
          { value: 'schedule', label: 'schedule' },
        ],
      },
    ] satisfies FormFieldSpec[],
    onChange: [
      op(
        'cubes.patchDraft',
        { draft: '{{vars.draft}}', patch: { materialization: '{{form}}' } },
        [set('draft', '{{result.draft}}'), set('mat', '{{form}}'), set('dirty', true)],
      ),
    ],
  });
  w('mat_engines', 'TextBlock', {
    text: 'Hot engine: {{vars.draft.materialization.hotEngine}} · Cold engine: {{vars.draft.materialization.coldEngine}}',
    variant: 'caption',
    color: 'text.secondary',
  });

  // --- versions ----------------------------------------------------------------
  w('versions_text', 'TextBlock', {
    text: 'Current contract_version: {{queries.load.data.contractVersion}}. Additive edits use Save (PATCH). Breaking grain/dimension/metric/federation changes require POST /api/cubes/{id}/versions. Use the Impact tab to preview consumers before archive or publish.',
  });

  // --- impact DomainComponent (A3 dry-run) ------------------------------------
  w('impact_new_hint', 'AlertBanner', {
    severity: 'info',
    text: 'Save the cube first to assess consumers and preview archive / contract publish.',
  }, { visibleWhen: NEW });
  w('impact_dc', 'DomainComponent', {
    component: 'cubes.ImpactPanel',
    inputs: {
      cubeId: ID,
      draft: '{{vars.draft}}',
      title: 'Impact — {{queries.load.data.title}}',
    },
  }, { visibleWhen: EXISTING });

  const tabs: PageTab[] = [{
    id: 'designer',
    label: 'Designer',
    layout: layout('root', {
      root: {
        type: 'Column',
        children: ['toolbar', 'banners', 'body'],
        style: { gap: '0', height: 'calc(100vh - 64px)' },
      },
      toolbar: {
        type: 'Row',
        children: ['back', 'title', 'unsaved', 'actions'],
        style: {
          alignItems: 'center',
          gap: '8px',
          padding: '8px 16px',
          borderBottom: '1px solid rgba(128,128,128,0.3)',
          flexWrap: 'wrap',
        },
      },
      actions: {
        type: 'Row',
        children: ['validate_btn', 'deploy_btn', 'refresh_btn', 'impact_btn', 'save_create', 'save_patch'],
        style: { alignItems: 'center', gap: '8px', marginLeft: 'auto', flex: '0 0 auto' },
      },
      banners: {
        type: 'Column',
        children: ['error_banner', 'info_banner'],
        style: { gap: '8px', padding: '0 16px' },
      },
      body: {
        type: 'Column',
        children: ['tabs'],
        style: { flex: '1 1 0', minHeight: '0', overflowY: 'auto', padding: '16px' },
      },
      tabs: {
        type: 'TabSet',
        children: [
          'tab_overview',
          'tab_dimensions',
          'tab_metrics',
          'tab_grains',
          'tab_federation',
          'tab_materialization',
          'tab_versions',
          'tab_impact',
        ],
        props: {
          variable: 'tab',
          tabs: [
            { id: 'overview', label: 'Overview' },
            { id: 'dimensions', label: 'Dimensions' },
            { id: 'metrics', label: 'Metrics' },
            { id: 'grains', label: 'Grains' },
            { id: 'federation', label: 'Federation' },
            { id: 'materialization', label: 'Materialization' },
            { id: 'versions', label: 'Versions' },
            { id: 'impact', label: 'Impact' },
          ],
        },
      },
      tab_overview: {
        type: 'Column',
        children: ['overview_form', 'overview_hint'],
        style: { gap: '12px', maxWidth: '560px' },
      },
      tab_dimensions: {
        type: 'Column',
        children: ['dims_help', 'dims_form'],
        style: { gap: '12px', maxWidth: '720px' },
      },
      tab_metrics: {
        type: 'Column',
        children: ['metrics_help', 'metrics_form'],
        style: { gap: '12px', maxWidth: '720px' },
      },
      tab_grains: {
        type: 'Column',
        children: ['grains_help', 'grains_form'],
        style: { gap: '12px', maxWidth: '720px' },
      },
      tab_federation: {
        type: 'Column',
        children: ['fed_dc'],
        style: { gap: '12px' },
      },
      tab_materialization: {
        type: 'Column',
        children: ['mat_form', 'mat_engines'],
        style: { gap: '12px', maxWidth: '480px' },
      },
      tab_versions: {
        type: 'Column',
        children: ['versions_text'],
        style: { gap: '12px', maxWidth: '640px' },
      },
      tab_impact: {
        type: 'Column',
        children: ['impact_new_hint', 'impact_dc'],
        style: { gap: '12px', maxWidth: '960px' },
      },
    }),
  }];

  return {
    name: 'Cube designer',
    slug: 'cube-designer',
    description:
      'Compose a cube contract: overview, dimensions, metrics, grains, federation, materialization, and impact. Served at /build/cubes/new and /build/cubes/:id.',
    version: 2,
    isCore: true,
    status: 'published',
    layout: tabs[0].layout,
    tabs,
    filterBar: layout('fb_root', { fb_root: { type: 'Column', children: [], style: { gap: '0px' } } }),
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none' as const,
      surface: { padding: 0 },
      tabVariable: 'tab',
      variables: [
        { name: 'draft', initFrom: { query: 'load', path: 'draft' }, description: 'Working cube draft' },
        { name: 'mat', description: 'Materialization form slice mirrored into draft.materialization' },
        { name: 'dirty', default: false, description: 'Unsaved changes' },
        { name: 'tab', default: 'overview', description: 'Active designer tab' },
        { name: 'validation', description: 'Last validate response / banner payload' },
        { name: 'info', description: 'Info banner' },
        { name: 'error', description: 'Error banner' },
      ],
      queries: [
        { id: 'load', operation: 'cubes.editorStart', params: { id: ID } },
        { id: 'bos', operation: 'cubes.businessObjects', params: {} },
        {
          id: 'dimensions',
          operation: 'cubes.dimensions',
          params: { boId: '{{vars.draft.boId}}' },
          keepPrevious: true,
        },
        {
          id: 'metrics',
          operation: 'cubes.metrics',
          params: { boId: '{{vars.draft.boId}}' },
          keepPrevious: true,
        },
      ],
    },
  };
}

export const cubeDesignerBlueprint = () => structuredClone(cubeDesignerPage());
