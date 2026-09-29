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

  reg('entity_select', 'VariableSelect', {
    variable: 'entity_domain',
    label: 'Entity Domain',
    emptyLabel: 'All Master Domains',
    minWidth: 190,
    options: [
      { value: 'security', label: 'Security Master (ISIN, Cap, Shares)' },
      { value: 'ratings', label: 'Credit & Sanctions Ratings' },
      { value: 'benchmarks', label: 'Benchmarks & Sectors (GICS, NAICS)' },
      { value: 'pricing', label: 'Evaluated Pricing (EOD, Yields)' },
      { value: 'party', label: 'Party & Legal Entity (LEI, Issuer)' },
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
        progressVariable: 'ingest_status',
        params: {
          universe_size: '{{vars.universe_size}}',
        },
        successMessage: 'MDM Multi-Vendor Ingestion & Scoring pipeline completed successfully.',
      },
    ],
  }, { style: fit });

  reg('pipeline_progress', 'ProgressBar', {
    variable: 'ingest_status',
    showSignal: true,
    color: 'primary',
  }, {
    visibleWhen: cond('vars.ingest_status', 'is_not_null'),
  });

  reg('pipeline_hud', 'AlertBanner', {
    severity: 'info',
    text: 'Tripartite Pipeline: Raw files → Apache Iceberg Parquet Lakehouse → Centralized Validation Engine → Staging DB → MDM Survivorship Mastering → Vendor Quality & Displacement Scoring Mart (StarRocks: mdm_analytics.vendor_substitution_daily).',
  });

  // Tab 1: Controls (Typeahead Search & Facets)
  reg('matrix_search', 'SearchInput', {
    variable: 'matrix_search',
    placeholder: 'Typeahead search attribute or vendor (e.g. CLOSING_PRICE, BBG)...',
    variant: 'typeahead',
    optionsFrom: 'queries.scorecard.data.substitution_matrix',
    optionsField: 'attribute_code',
    maxWidth: 420,
  });

  reg('tier_facets', 'FacetFilter', {
    variable: 'tier_filter',
    label: 'Attribute Tier',
    allLabel: 'All Tiers',
    facetField: 'tier',
    query: 'scorecard',
    rowsPath: 'substitution_matrix',
    options: [
      { value: '1', label: 'Tier 1 (Core Identity & Price)', color: 'warning' },
      { value: '2', label: 'Tier 2 (Risk & Classification)', color: 'primary' },
      { value: '3', label: 'Tier 3 (Reference & Descriptive)', color: 'default' },
    ],
  });

  // Tab 1: Sufficiency Matrix Grid
  reg('matrix_grid', 'DataGrid', {
    query: 'scorecard',
    rowsPath: 'substitution_matrix',
    rowKey: 'vendor_id,attribute_code',
    enableSearch: true,
    searchVariant: 'typeahead',
    searchOptionsField: 'attribute_code',
    searchVariable: 'matrix_search',
    searchFields: ['attribute_code', 'vendor_id', 'entity_domain'],
    filters: {
      vendor_id: 'vendor_filter',
      tier: 'tier_filter',
      entity_domain: 'entity_domain',
    },
    emptyText: 'No scoring data matching your search or filter criteria.',
    columns: [
      col('attr', 'Attribute', { kind: 'twoLine', primary: '{{row.attribute_code}}', secondary: 'Tier {{row.tier}}' }),
      col('domain', 'Domain', { kind: 'chip', value: '{{row.entity_domain}}', variant: 'outlined' }),
      col('vendor', 'Vendor', { kind: 'chip', value: '{{row.vendor_id}}', variant: 'outlined' }),
      col('tier', 'Tier', { kind: 'chip', value: '{{row.tier}}', colorMap: TIER_COLORS }),
      col('sufficiency', 'Substitution Rate', { kind: 'percent', value: '{{row.sufficiency_rate_pct}}' }, { align: 'right' }),
      col('coverage', 'Coverage', { kind: 'percent', value: '{{row.coverage_pct}}' }, { align: 'right' }),
      col('cond_suff', 'Conditional Sufficiency', { kind: 'percent', value: '{{row.conditional_sufficiency_pct}}' }, { align: 'right' }),
      col('solo_rate', 'Solo Rate', { kind: 'percent', value: '{{row.solo_rate_pct}}' }, { align: 'right' }),
      col('oer', 'Override Endorsement (OER)', { kind: 'percent', value: '{{row.override_endorsement_rate_pct}}' }, { align: 'right' }),
    ] satisfies ColumnDef[],
  });

  // Tab 2: Value for Money Frontier Grid & Entity Breakdown
  reg('frontier_grid', 'DataGrid', {
    query: 'scorecard',
    rowsPath: 'frontier_points',
    rowKey: 'vendor_id',
    enableSearch: true,
    searchVariant: 'typeahead',
    searchOptionsField: 'vendor_name',
    searchFields: ['vendor_name', 'vendor_id'],
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
      col('actions', 'Actions', {
        kind: 'actions',
        buttons: [
          {
            label: 'Edit Spend',
            icon: 'edit',
            variant: 'outlined',
            color: 'primary',
            onClick: [
              {
                kind: 'runOperation',
                operation: 'mdmScoring.updateSpend',
                form: {
                  title: 'Edit Annual Vendor Spend',
                  fields: [
                    {
                      name: 'annual_cost',
                      label: 'Annual Spend ($)',
                      kind: 'number',
                      step: 10000,
                      required: true,
                    },
                  ],
                },
                params: {
                  vendor_id: '{{row.vendor_id}}',
                  annual_cost: '{{form.annual_cost}}',
                },
                successMessage: 'Annual spend updated for {{row.vendor_name}}.',
              },
            ],
          },
        ],
      }),
    ] satisfies ColumnDef[],
  });

  reg('entity_breakdown_hdr', 'TextBlock', {
    text: 'Domain / Entity Breakdown: Vendor Strengths by Data Category',
    variant: 'h6',
  });

  reg('entity_breakdown_desc', 'TextBlock', {
    text: 'Vendors specialize in specific domains (e.g. S&P in ratings, Bloomberg in pricing, Refinitiv in party hierarchy). Compare non-aggregated standing per entity domain.',
    variant: 'body2',
    color: 'text.secondary',
  });

  reg('entity_breakdown_grid', 'DataGrid', {
    query: 'scorecard',
    rowsPath: 'entity_breakdowns',
    rowKey: 'vendor_id,entity_domain',
    enableSearch: true,
    searchVariant: 'typeahead',
    searchOptionsField: 'entity_domain',
    searchFields: ['vendor_name', 'vendor_id', 'entity_domain'],
    emptyText: 'No entity breakdown data available.',
    columns: [
      col('vendor', 'Vendor', { kind: 'twoLine', primary: '{{row.vendor_name}}', secondary: '{{row.vendor_id}}' }),
      col('domain', 'Entity Domain', {
        kind: 'chip',
        value: '{{row.entity_domain}}',
        colorMap: { security: 'primary', ratings: 'warning', benchmarks: 'secondary', pricing: 'info', party: 'default' },
      }),
      col('attrs', 'Attributes', { kind: 'number', value: '{{row.attribute_count}}' }, { align: 'right' }),
      col('cov', 'Coverage', { kind: 'percent', value: '{{row.coverage_pct}}' }, { align: 'right' }),
      col('suff', 'Sufficiency', { kind: 'percent', value: '{{row.sufficiency_rate_pct}}' }, { align: 'right' }),
      col('qual', 'Domain Quality', { kind: 'number', value: '{{row.quality_index}}' }, { align: 'right' }),
      col('cost', 'Allocated Spend ($)', { kind: 'number', value: '{{row.annual_cost}}' }, { align: 'right' }),
      col('cpp', 'Cost / Pt ($)', { kind: 'number', value: '{{row.cost_per_quality_point}}' }, { align: 'right' }),
      col('frontier', 'Frontier Status', { kind: 'chip', value: '{{row.is_on_frontier}}', colorMap: FRONTIER_COLORS }),
      col('actions', 'Actions', {
        kind: 'actions',
        buttons: [
          {
            label: 'Edit Domain Spend',
            icon: 'edit',
            variant: 'outlined',
            color: 'primary',
            onClick: [
              {
                kind: 'runOperation',
                operation: 'mdmScoring.updateSpend',
                form: {
                  title: 'Edit Domain Spend',
                  fields: [
                    {
                      name: 'annual_cost',
                      label: 'Domain Spend ($)',
                      kind: 'number',
                      step: 5000,
                      required: true,
                    },
                  ],
                },
                params: {
                  vendor_id: '{{row.vendor_id}}',
                  entity_domain: '{{row.entity_domain}}',
                  annual_cost: '{{form.annual_cost}}',
                },
                successMessage: 'Domain spend updated for {{row.vendor_name}} ({{row.entity_domain}}).',
              },
            ],
          },
        ],
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
    rowsPath: 'tier_summaries',
    rowKey: 'tier',
    enableSearch: true,
    searchVariant: 'typeahead',
    searchFields: ['tier'],
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
    rowsPath: 'residual_gaps',
    rowKey: 'attribute_code',
    enableSearch: true,
    searchVariant: 'typeahead',
    searchOptionsField: 'attribute_code',
    searchFields: ['attribute_code'],
    emptyText: 'No residual gaps detected for this removal scenario.',
    columns: [
      col('attr', 'Attribute Code', { kind: 'text', value: '{{row.attribute_code}}' }),
      col('tier', 'Tier', { kind: 'chip', value: '{{row.tier}}', colorMap: TIER_COLORS }),
      col('solo_rate', 'Solo Rate', { kind: 'percent', value: '{{row.solo_rate_pct}}' }, { align: 'right' }),
      col('records_lost', 'Records Lost Without Replacement', { kind: 'number', value: '{{row.records_lost}}' }, { align: 'right' }),
    ] satisfies ColumnDef[],
  });

  reg('disp_tco_table', 'DomainComponent', {
    component: 'mdmScoring.DisplacementTCOTable',
    inputs: {
      candidate_vendor_id: '{{vars.candidate_vendor}}',
      as_of: '{{vars.as_of}}',
    },
  });

  reg('radar_widget', 'DomainComponent', {
    component: 'mdmScoring.RadarScorecard',
    inputs: {
      as_of: '{{vars.as_of}}',
    },
  });

  reg('optimizer_widget', 'DomainComponent', {
    component: 'mdmScoring.BundleOptimizer',
    inputs: {},
  });

  // Tab 4: Tolerance Registry
  reg('tolerances_grid', 'DataGrid', {
    query: 'tolerances',
    emptyText: 'No tolerances registered.',
    enableSearch: true,
    searchVariant: 'typeahead',
    searchOptionsField: 'attribute_code',
    searchFields: ['attribute_code', 'match_type', 'description'],
    columns: [
      col('code', 'Attribute Code', { kind: 'text', value: '{{row.attribute_code}}' }),
      col('tier', 'Tier', { kind: 'chip', value: '{{row.tier}}', colorMap: TIER_COLORS }),
      col('match_type', 'Match Rule', { kind: 'text', value: '{{row.match_type}}' }),
      col('tolerance', 'Tolerance Value', { kind: 'number', value: '{{row.tolerance_val}}' }, { align: 'right' }),
      col('weight', 'Tier Weight', { kind: 'number', value: '{{row.tier_weight}}' }, { align: 'right' }),
      col('desc', 'Description', { kind: 'text', value: '{{row.description}}' }),
    ] satisfies ColumnDef[],
  });

  // Tab 5: Training & User Guide
  reg('training_alert', 'AlertBanner', {
    severity: 'info',
    text: 'User Training Guide: Learn how MDM Source Scoring evaluates market data feeds, eliminates vendor lock-in, and optimizes multi-million dollar data budgets.',
  });

  reg('training_sec1', 'TextBlock', {
    text: '1. What is Master Data Management (MDM) & Why Does This Page Exist?',
    variant: 'h6',
  });

  reg('training_text1', 'TextBlock', {
    text: 'Financial institutions subscribe to multiple market data vendors (Bloomberg, Refinitiv, FactSet, ICE, S&P) costing upwards of $5M–$20M annually. MDM continuously ingests all feeds, evaluates conflicting values against strict business rules, and crowns an authoritative "Golden Record" for each security and legal entity. This page gives data executives empirical metrics to challenge vendor contract renewals.',
    variant: 'body2',
  });

  reg('training_sec2', 'TextBlock', {
    text: '2. Metric Definitions Cheat Sheet',
    variant: 'h6',
  });

  reg('training_text2', 'TextBlock', {
    text: '• Substitution Rate (SR): The percentage of entities where a secondary vendor matches the golden record within tolerance. High SR means the vendor can seamlessly replace the incumbent.\n• Coverage: The percentage of your portfolio universe that the vendor provides data for.\n• Solo Rate (Holdout Risk): The percentage of entities where this vendor is the ONLY source. High solo rates indicate high lock-in that will create NULLs if dropped.\n• Override Endorsement Rate (OER): How often human portfolio managers intervened to pick this vendor over other candidates.\n• Value-for-Money Frontier: Plots composite data quality against annual licensing cost to find Pareto-optimal vendors.\n• 6-Pillar Radar Scorecard: Evaluates vendors across Sufficiency, Coverage, Delivery SLA, Restatement Stability, Steward Labor Friction, and Commercial Rights.',
    variant: 'body2',
  });

  reg('training_sec3', 'TextBlock', {
    text: '3. Entity Domain Strengths & Spend Allocation',
    variant: 'h6',
  });

  reg('training_text3', 'TextBlock', {
    text: 'Vendors are not equal across asset classes. In the Value for Money tab, examine the Domain Breakdown table:\n• Pricing: EOD prices and yields (Bloomberg & ICE excel).\n• Security Master: Instrument identifiers, ISINs, and shares (Bloomberg & Refinitiv).\n• Ratings: Credit ratings and sanctions flags (S&P Global excels with proprietary coverage).\n• Benchmarks: Index weights and industry sectors (FactSet & S&P Global).\n• Party / Legal Entity: LEIs, entity trees, and incorporation jurisdictions (Refinitiv PermID).\nClick "Edit Spend" on any vendor or domain row to simulate budget renegotiations in real time.',
    variant: 'body2',
  });

  // Layouts
  const filterBar = layout('filter_root', {
    filter_root: { type: 'Column', children: ['header_row', 'pipeline_progress', 'pipeline_hud'], style: { gap: '12px' } },
    header_row: { type: 'Row', children: ['hdr', 'universe_select', 'entity_select', 'vendor_filter', 'sync_mart_btn', 'run_pipeline_btn'], style: { alignItems: 'center', gap: '8px', flexWrap: 'wrap' } },
  });

  const matrixLayout = layout('matrix_root', {
    matrix_root: { type: 'Column', children: ['matrix_controls', 'matrix_grid'], style: { gap: '16px' } },
    matrix_controls: { type: 'Row', children: ['matrix_search', 'tier_facets'], style: { alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '16px' } },
  });

  const radarLayout = layout('radar_root', {
    radar_root: { type: 'Column', children: ['radar_widget'], style: { gap: '16px' } },
  });

  const optimizerLayout = layout('optimizer_root', {
    optimizer_root: { type: 'Column', children: ['optimizer_widget'], style: { gap: '16px' } },
  });

  const frontierLayout = layout('frontier_root', {
    frontier_root: { type: 'Column', children: ['frontier_grid', 'entity_breakdown_hdr', 'entity_breakdown_desc', 'entity_breakdown_grid'], style: { gap: '16px' } },
  });

  const displacementLayout = layout('disp_root', {
    disp_root: { type: 'Column', children: ['candidate_select', 'disp_tco_table', 'displacement_kpis', 'residual_gaps_grid'], style: { gap: '16px' } },
  });

  reg('multi_disp_widget', 'mdmScoring.MultiVendorDisplacement', {
    initial_dropped_vendors: ['BBG'],
    initial_replacement_vendors: ['ICE'],
    universe_size: '{{vars.universe_size}}',
  });

  const multiDisplacementLayout = layout('multi_disp_root', {
    multi_disp_root: { type: 'Column', children: ['multi_disp_widget'], style: { gap: '16px' } },
  });

  const tolerancesLayout = layout('tol_root', {
    tol_root: { type: 'Column', children: ['tolerances_grid'], style: { gap: '16px' } },
  });

  reg('shadow_widget', 'mdmScoring.ShadowComparator', {});

  const shadowLayout = layout('shadow_root', {
    shadow_root: { type: 'Column', children: ['shadow_widget'], style: { gap: '16px' } },
  });

  reg('trends_widget', 'mdmScoring.HistoricalTrendsViewer', {
    entity_domain: '{{vars.entity_domain}}',
  });

  const trendsLayout = layout('trends_root', {
    trends_root: { type: 'Column', children: ['trends_widget'], style: { gap: '16px' } },
  });

  const trainingLayout = layout('training_root', {
    training_root: { type: 'Column', children: ['training_alert', 'training_sec1', 'training_text1', 'training_sec2', 'training_text2', 'training_sec3', 'training_text3'], style: { gap: '16px' } },
  });

  const tabs: PageTab[] = [
    {
      id: 'matrix',
      label: 'Sufficiency Matrix',
      layout: matrixLayout,
    },
    {
      id: 'radar',
      label: '360° Radar Scorecard',
      layout: radarLayout,
    },
    {
      id: 'optimizer',
      label: 'Portfolio Optimizer',
      layout: optimizerLayout,
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
      id: 'multi_displacement',
      label: 'Multi-Vendor Exit Engine',
      layout: multiDisplacementLayout,
    },
    {
      id: 'shadow',
      label: 'Shadow Validation',
      layout: shadowLayout,
    },
    {
      id: 'trends',
      label: 'Historical Trends (3-Tier)',
      layout: trendsLayout,
    },
    {
      id: 'tolerances',
      label: 'Tolerance Registry',
      layout: tolerancesLayout,
    },
    {
      id: 'training',
      label: 'User Guide & Training',
      layout: trainingLayout,
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
        { name: 'entity_domain', default: '', url: true },
        { name: 'tier_filter', default: '', url: true },
        { name: 'matrix_search', default: '', url: true },
        { name: 'tab', default: 'matrix', url: true },
        { name: 'ingest_status', default: null, url: false },
      ],
      queries: [
        {
          id: 'scorecard',
          operation: 'mdmScoring.scorecard',
          params: {
            universe_size: '{{vars.universe_size}}',
            entity_domain: '{{vars.entity_domain}}',
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
