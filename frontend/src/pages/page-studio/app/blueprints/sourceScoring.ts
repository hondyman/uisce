import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, CellSpec, ChipColor, ColumnDef, ConditionNode } from '../appModel';

/**
 * MDM Source Scoring & Vendor Displacement Page Blueprint:
 * Built entirely from the Page Studio application model.
 * 
 * Includes:
 * 1. Sufficiency Matrix (Substitution Rate, Coverage, Conditional Sufficiency, Solo Rate)
 * 2. Value for Money Frontier (Quality vs. Annual Cost, Non-dominated Envelope)
 * 3. Displacement Simulator (Vendor Removal Impact, Unchanged vs. Changed vs. Null)
 * 4. Tolerance Registry (Tiers 1-3, Match Types, Basis Points, Weights)
 */

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const fit = { flex: '0 0 auto' };

type Spec = Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>;
const layout = (root: string, spec: Spec): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

const col = (id: string, header: string, cell: CellSpec | CellSpec[], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell, ...more };

const TIER_COLORS: Record<string, ChipColor> = { '1': 'warning', '2': 'primary', '3': 'default' };
const FRONTIER_COLORS: Record<string, ChipColor> = { true: 'success', false: 'error', 'Optimal': 'success', 'Suboptimal': 'error' };

export function sourceScoringBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const reg = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}): string => {
    components[id] = { id, type, props, ...extra };
    return id;
  };

  // Header & Controls
  reg('hdr', 'PageHeader', {
    icon: 'analytics',
    title: 'Source Scoring & Vendor Displacement',
    subtitle: 'Evaluate vendor sufficiency, displacement readiness, and cost-efficiency frontier',
  }, { style: { flex: '1 1 320px' } });

  reg('universe_select', 'VariableSelect', {
    variable: 'universe_size',
    label: 'Universe Scope',
    minWidth: 160,
    options: [
      { value: '42000', label: '42,000 Entities (Full)' },
      { value: '10000', label: '10,000 Entities (Bake-off Sample)' },
      { value: '5000', label: '5,000 Entities (Core G10)' },
    ],
  }, { style: fit });

  reg('vendor_filter', 'VariableSelect', {
    variable: 'vendor_filter',
    label: 'Vendor Focus',
    emptyLabel: 'All Vendors',
    minWidth: 180,
    options: [
      { value: 'BBG', label: 'Bloomberg (Preference 1)' },
      { value: 'RFT', label: 'Refinitiv (LSEG)' },
      { value: 'FDS', label: 'FactSet' },
      { value: 'ICE', label: 'ICE Data Services' },
      { value: 'SPG', label: 'S&P Global MI' },
    ],
  }, { style: fit });

  reg('sync_mart_btn', 'ActionButton', {
    label: 'Sync StarRocks Hot Mart',
    icon: 'refresh',
    variant: 'outlined',
    onClick: [
      {
        kind: 'runOperation',
        operation: 'mdmScoring.syncMart',
        successMessage: 'StarRocks hot analytical mart synchronized.',
      },
    ],
  }, { style: fit });

  reg('run_pipeline_btn', 'ActionButton', {
    label: 'Run Ingest & Scoring Pipeline',
    icon: 'play',
    variant: 'contained',
    onClick: [
      {
        kind: 'runOperation',
        operation: 'mdmScoring.runPipeline',
        successMessage: 'MDM Multi-Vendor Ingestion & Scoring pipeline launched via Temporal.',
      },
    ],
  }, { style: fit });

  reg('pipeline_hud', 'AlertBanner', {
    severity: 'info',
    text: 'Tripartite Pipeline: Raw files → Apache Iceberg Parquet Lakehouse → Centralized Validation Engine → Staging DB → MDM Survivorship Mastering → Vendor Quality & Displacement Scoring Mart (StarRocks: mdm_analytics.vendor_substitution_daily).',
  });

  // Tab 1: Sufficiency Matrix Grid
  reg('matrix_grid', 'DataGrid', {
    query: 'scorecard',
    dataPath: 'substitution_matrix',
    emptyText: 'No scoring data available for this universe.',
    columns: [
      col('attr', 'Attribute', { kind: 'twoLine', primary: '{{row.attribute_code}}', secondary: 'Tier {{row.tier}}' }),
      col('vendor', 'Vendor', { kind: 'chip', value: '{{row.vendor_id}}', variant: 'outlined' }),
      col('tier', 'Tier', { kind: 'chip', value: '{{row.tier}}', colorMap: TIER_COLORS }),
      col('sufficiency', 'Substitution Rate', { kind: 'percent', value: '{{row.sufficiency_rate_pct}}' }, { align: 'right' }),
      col('coverage', 'Coverage', { kind: 'percent', value: '{{row.coverage_pct}}' }, { align: 'right' }),
      col('cond_suff', 'Conditional Sufficiency', { kind: 'percent', value: '{{row.conditional_sufficiency_pct}}' }, { align: 'right' }),
      col('solo_rate', 'Solo Rate', { kind: 'percent', value: '{{row.solo_rate_pct}}' }, { align: 'right' }),
      col('oer', 'Override Endorsement (OER)', { kind: 'percent', value: '{{row.override_endorsement_rate_pct}}' }, { align: 'right' }),
    ] satisfies ColumnDef[],
  });

  // Tab 2: Value for Money Frontier Grid
  reg('frontier_grid', 'DataGrid', {
    query: 'scorecard',
    dataPath: 'frontier_points',
    emptyText: 'No frontier calculations available.',
    columns: [
      col('vendor', 'Vendor', { kind: 'twoLine', primary: '{{row.vendor_name}}', secondary: '{{row.vendor_id}}' }),
      col('cost', 'Annual Spend ($)', { kind: 'number', value: '{{row.annual_cost}}' }, { align: 'right' }),
      col('quality', 'Composite Quality Index', { kind: 'number', value: '{{row.quality_index}}' }, { align: 'right' }),
      col('cost_per_pt', 'Cost per Quality Point ($)', { kind: 'number', value: '{{row.cost_per_quality_point}}' }, { align: 'right' }),
      col('on_frontier', 'Frontier Status', {
        kind: 'chip',
        value: '{{row.is_on_frontier}}',
        colorMap: FRONTIER_COLORS,
      }),
    ] satisfies ColumnDef[],
  });

  // Tab 3: Displacement Simulator
  reg('candidate_select', 'VariableSelect', {
    variable: 'candidate_vendor',
    label: 'Candidate for Removal',
    minWidth: 220,
    options: [
      { value: 'BBG', label: 'Bloomberg ($2,140,000/yr)' },
      { value: 'RFT', label: 'Refinitiv ($1,180,000/yr)' },
      { value: 'FDS', label: 'FactSet ($720,000/yr)' },
      { value: 'ICE', label: 'ICE Data ($540,000/yr)' },
      { value: 'SPG', label: 'S&P Global ($610,000/yr)' },
    ],
  }, { style: fit });

  reg('displacement_kpis', 'DataGrid', {
    query: 'displacement',
    dataPath: 'tier_summaries',
    emptyText: 'Select a candidate vendor to simulate removal.',
    columns: [
      col('tier', 'Attribute Tier', { kind: 'chip', value: '{{row.tier}}', colorMap: TIER_COLORS }),
      col('attrs', 'Attributes Count', { kind: 'number', value: '{{row.attributes_count}}' }, { align: 'right' }),
      col('unchanged', 'Unchanged (Takeover Match)', { kind: 'percent', value: '{{row.unchanged_pct}}' }, { align: 'right' }),
      col('changed', 'Changed (Within Tolerance)', { kind: 'percent', value: '{{row.changed_pct}}' }, { align: 'right' }),
      col('now_null', 'Now NULL (Sole Source Lost)', { kind: 'percent', value: '{{row.now_null_pct}}' }, { align: 'right' }),
    ] satisfies ColumnDef[],
  });

  reg('residual_gaps_grid', 'DataGrid', {
    query: 'displacement',
    dataPath: 'residual_gaps',
    emptyText: 'No residual gaps detected for this removal scenario.',
    columns: [
      col('attr', 'Attribute Code', { kind: 'text', value: '{{row.attribute_code}}' }),
      col('tier', 'Tier', { kind: 'chip', value: '{{row.tier}}', colorMap: TIER_COLORS }),
      col('solo_rate', 'Solo Rate', { kind: 'percent', value: '{{row.solo_rate_pct}}' }, { align: 'right' }),
      col('records_lost', 'Records Lost Without Replacement', { kind: 'number', value: '{{row.records_lost}}' }, { align: 'right' }),
    ] satisfies ColumnDef[],
  });

  // Tab 4: Tolerance Registry
  reg('tolerances_grid', 'DataGrid', {
    query: 'tolerances',
    emptyText: 'No tolerances registered.',
    columns: [
      col('code', 'Attribute Code', { kind: 'text', value: '{{row.attribute_code}}' }),
      col('tier', 'Tier', { kind: 'chip', value: '{{row.tier}}', colorMap: TIER_COLORS }),
      col('match_type', 'Match Rule', { kind: 'text', value: '{{row.match_type}}' }),
      col('tolerance', 'Tolerance Value', { kind: 'number', value: '{{row.tolerance_val}}' }, { align: 'right' }),
      col('weight', 'Tier Weight', { kind: 'number', value: '{{row.tier_weight}}' }, { align: 'right' }),
      col('desc', 'Description', { kind: 'text', value: '{{row.description}}' }),
    ] satisfies ColumnDef[],
  });

  // Layouts
  const filterBar = layout('filter_root', {
    filter_root: { type: 'Column', children: ['header_row', 'pipeline_hud'], style: { gap: '12px' } },
    header_row: { type: 'Row', children: ['hdr', 'universe_select', 'vendor_filter', 'sync_mart_btn', 'run_pipeline_btn'], style: { alignItems: 'center', gap: '8px' } },
  });

  const matrixLayout = layout('matrix_root', {
    matrix_root: { type: 'Column', children: ['matrix_grid'], style: { gap: '16px' } },
  });

  const frontierLayout = layout('frontier_root', {
    frontier_root: { type: 'Column', children: ['frontier_grid'], style: { gap: '16px' } },
  });

  const displacementLayout = layout('disp_root', {
    disp_root: { type: 'Column', children: ['candidate_select', 'displacement_kpis', 'residual_gaps_grid'], style: { gap: '16px' } },
  });

  const tolerancesLayout = layout('tol_root', {
    tol_root: { type: 'Column', children: ['tolerances_grid'], style: { gap: '16px' } },
  });

  const tabs: PageTab[] = [
    {
      id: 'matrix',
      label: 'Sufficiency Matrix',
      layout: matrixLayout,
    },
    {
      id: 'frontier',
      label: 'Value for Money',
      layout: frontierLayout,
    },
    {
      id: 'displacement',
      label: 'Displacement Simulator',
      layout: displacementLayout,
    },
    {
      id: 'tolerances',
      label: 'Tolerance Registry',
      layout: tolerancesLayout,
    },
  ];

  return structuredClone({
    name: 'Source scoring & displacement',
    slug: 'mdm-source-scoring',
    description: 'Vendor quality scoring, substitution rates, override endorsements, value-for-money efficient frontier, and displacement readiness simulation.',
    layout: tabs[0].layout,
    tabs,
    filterBar,
    components,
    dataSources: [],
    status: 'published' as const,
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 1400, padding: 3 },
      tabVariable: 'tab',
      variables: [
        { name: 'universe_size', default: '42000', url: true },
        { name: 'candidate_vendor', default: 'BBG', url: true },
        { name: 'vendor_filter', default: '', url: true },
        { name: 'tab', default: 'matrix', url: true },
      ],
      queries: [
        {
          id: 'scorecard',
          operation: 'mdmScoring.scorecard',
          params: {
            universe_size: '{{vars.universe_size}}',
          },
        },
        {
          id: 'tolerances',
          operation: 'mdmScoring.tolerances',
        },
        {
          id: 'displacement',
          operation: 'mdmScoring.displacement',
          params: {
            dropped_vendor_id: '{{vars.candidate_vendor}}',
          },
        },
      ],
    },
  });
}
