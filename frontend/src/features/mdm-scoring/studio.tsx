import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { mdmScoringApi } from './api';

/**
 * MDM Source Scoring Page Studio Operations:
 * Exposes queries and mutations to the Page Studio runtime.
 * NO coded pages are used; everything is bound dynamically in the Page Designer.
 */

const operations: OperationDef[] = [
  {
    id: 'mdmScoring.scorecard',
    domain: 'mdm_scoring',
    kind: 'query',
    label: 'Vendor Scorecard & Substitution Matrix',
    description: 'Retrieves the complete vendor sufficiency matrix, solo rates, and value-for-money frontier.',
    params: [
      { name: 'universe_size', type: 'string', required: false },
      { name: 'as_of', type: 'string', required: false },
      { name: 'entity_domain', type: 'string', required: false },
    ],
    fields: [
      { name: 'substitution_matrix' },
      { name: 'frontier_points' },
      { name: 'entity_breakdowns' },
      { name: 'entity_domains' },
      { name: 'displacement_scenarios' },
      { name: 'universe_size', type: 'number' },
      { name: 'annual_spend_total', type: 'number' },
      { name: 'best_alt_sufficiency_t1', type: 'number' },
      { name: 'premium_solo_share_t1', type: 'number' },
      { name: 'displacement_readiness', type: 'number' },
    ],
    run: async (params) => {
      const uSize = params.universe_size ? parseInt(String(params.universe_size), 10) : 42000;
      const asOf = params.as_of ? String(params.as_of) : undefined;
      const entityDomain = params.entity_domain ? String(params.entity_domain) : undefined;
      return mdmScoringApi.scorecard(uSize, asOf, entityDomain);
    },
  },
  {
    id: 'mdmScoring.updateSpend',
    domain: 'mdm_scoring',
    kind: 'mutation',
    label: 'Update Vendor Annual Spend',
    description: 'Updates annual licensing cost for a vendor overall or for a specific entity domain (security, ratings, benchmarks, pricing, party).',
    params: [
      { name: 'vendor_id', type: 'string', required: true },
      { name: 'annual_cost', type: 'number', required: true },
      { name: 'entity_domain', type: 'string', required: false },
    ],
    invalidates: [['mdmScoring.scorecard'], ['mdmScoring.displacement']],
    run: async (params) => {
      const vendorId = String(params.vendor_id);
      const annualCost = Number(params.annual_cost);
      const entityDomain = params.entity_domain ? String(params.entity_domain) : undefined;
      return mdmScoringApi.updateSpend(vendorId, annualCost, entityDomain);
    },
  },
  {
    id: 'mdmScoring.tolerances',
    domain: 'mdm_scoring',
    kind: 'query',
    label: 'Attribute Tolerance Registry',
    description: 'Retrieves matching criteria (exact, Jaro-Winkler, numeric BP, % diff, date lag) and weights by tier.',
    params: [],
    fields: [
      { name: 'attribute_code' },
      { name: 'tier', type: 'number' },
      { name: 'match_type' },
      { name: 'tolerance_val', type: 'number' },
      { name: 'tier_weight', type: 'number' },
      { name: 'description' },
    ],
    run: async () => {
      const res = await mdmScoringApi.tolerances();
      return Object.values(res);
    },
  },
  {
    id: 'mdmScoring.displacement',
    domain: 'mdm_scoring',
    kind: 'query',
    label: 'Vendor Displacement Simulator',
    description: 'Simulates the survivorship outcome if a vendor is removed from the stack.',
    params: [
      { name: 'dropped_vendor_id', type: 'string', required: true },
    ],
    fields: [
      { name: 'dropped_vendor_id' },
      { name: 'dropped_vendor_name' },
      { name: 'annual_cost', type: 'number' },
      { name: 'displacement_readiness_pct', type: 'number' },
      { name: 'tier_summaries' },
      { name: 'residual_gaps' },
      { name: 'total_null_values', type: 'number' },
      { name: 'net_first_year_benefit', type: 'number' },
    ],
    run: async (params) => {
      const vendorId = params.dropped_vendor_id ? String(params.dropped_vendor_id) : 'BBG';
      return mdmScoringApi.displacement(vendorId);
    },
  },
  {
    id: 'mdmScoring.syncMart',
    domain: 'mdm_scoring',
    kind: 'mutation',
    label: 'Sync StarRocks Hot Mart',
    description: 'Flushes vendor substitution rollups to StarRocks mdm_analytics table.',
    params: [
      { name: 'as_of', type: 'string', required: false },
    ],
    run: async (params) => {
      const asOf = params.as_of ? String(params.as_of) : undefined;
      return mdmScoringApi.syncMart(asOf);
    },
  },
  {
    id: 'mdmScoring.runPipeline',
    domain: 'mdm_scoring',
    kind: 'mutation',
    label: 'Run Ingest & Scoring Pipeline',
    description: 'Executes the full-cycle Data Pipeline: Iceberg export + Staging load + Mastering + Vendor scoring.',
    params: [
      { name: 'pipeline_id', type: 'string', required: false },
      { name: 'universe_size', type: 'string', required: false },
    ],
    invalidates: [['mdmScoring.scorecard'], ['mdmScoring.displacement']],
    run: async (params, ctx) => {
      const pid = params.pipeline_id ? String(params.pipeline_id) : undefined;
      const uSize = params.universe_size ? parseInt(String(params.universe_size), 10) : 42000;

      ctx?.progress?.('Step 1/4: Ingesting raw market feeds (Bloomberg, Refinitiv, FactSet, ICE, SPG)...');
      await new Promise((r) => setTimeout(r, 600));

      ctx?.progress?.('Step 2/4: Validating centralized attribute tolerances against Lakehouse...');
      await new Promise((r) => setTimeout(r, 600));

      ctx?.progress?.('Step 3/4: Mastering golden records & calculating survivorship...');
      await new Promise((r) => setTimeout(r, 500));

      ctx?.progress?.('Step 4/4: Computing substitution matrix & synchronizing StarRocks hot mart...');
      const res = await mdmScoringApi.runPipeline(pid, uSize);

      ctx?.progress?.('Ingestion & Scoring completed successfully.');
      return res;
    },
  },
  {
    id: 'mdmScoring.dimensions',
    domain: 'mdm_scoring',
    kind: 'query',
    label: '6-Pillar Vendor Telemetry Dimensions',
    description: 'Retrieves normalized multi-dimensional vendor telemetry (sufficiency, coverage, SLA, stability, friction, licensing).',
    params: [
      { name: 'as_of', type: 'string', required: false },
    ],
    fields: [
      { name: 'vendor_id' },
      { name: 'vendor_name' },
      { name: 'annual_spend', type: 'number' },
      { name: 'components' },
      { name: 'composite_quality', type: 'number' },
      { name: 'cost_per_quality_point', type: 'number' },
      { name: 'avg_delivery_lag_mins', type: 'number' },
      { name: 'sla_breach_count', type: 'number' },
      { name: 'total_revisions', type: 'number' },
      { name: 'friction_cost', type: 'number' },
      { name: 'rights_score', type: 'number' },
    ],
    run: async (params) => {
      const asOf = params.as_of ? String(params.as_of) : undefined;
      return mdmScoringApi.dimensions(asOf);
    },
  },
  {
    id: 'mdmScoring.optimizeBundle',
    domain: 'mdm_scoring',
    kind: 'mutation',
    label: 'Solve Optimal Vendor Bundle',
    description: 'Executes the weighted set cover solver with coverage threshold constraints.',
    params: [
      { name: 'target_t1_coverage', type: 'number', required: false },
      { name: 'target_t2_coverage', type: 'number', required: false },
      { name: 'target_t3_coverage', type: 'number', required: false },
    ],
    run: async (params) => {
      return mdmScoringApi.optimizeBundle({
        target_t1_coverage: params.target_t1_coverage ? Number(params.target_t1_coverage) : 0.995,
        target_t2_coverage: params.target_t2_coverage ? Number(params.target_t2_coverage) : 0.950,
        target_t3_coverage: params.target_t3_coverage ? Number(params.target_t3_coverage) : 0.850,
      });
    },
  },
  {
    id: 'mdmScoring.weightProfiles',
    domain: 'mdm_scoring',
    kind: 'query',
    label: 'Scoring Weight Profiles',
    description: 'Lists all registered governance weight profiles.',
    params: [],
    run: async () => {
      return mdmScoringApi.weightProfiles();
    },
  },
  {
    id: 'mdmScoring.displacementMulti',
    domain: 'mdm_scoring',
    kind: 'mutation',
    label: 'Multi-Vendor Displacement TCO Simulation',
    description: 'Simulates dropping multiple vendors, adding replacements, and calculates combined TCO and tier coverage.',
    params: [
      { name: 'dropped_vendor_ids', type: 'object', required: true },
      { name: 'replacement_vendor_ids', type: 'object', required: false },
      { name: 'target_t1_coverage', type: 'number', required: false },
      { name: 'target_t2_coverage', type: 'number', required: false },
      { name: 'target_t3_coverage', type: 'number', required: false },
      { name: 'universe_size', type: 'number', required: false },
    ],
    run: async (params) => {
      return mdmScoringApi.displacementMulti({
        dropped_vendor_ids: Array.isArray(params.dropped_vendor_ids) ? (params.dropped_vendor_ids as string[]) : ['BBG'],
        replacement_vendor_ids: Array.isArray(params.replacement_vendor_ids) ? (params.replacement_vendor_ids as string[]) : undefined,
        target_t1_coverage: params.target_t1_coverage ? Number(params.target_t1_coverage) : undefined,
        target_t2_coverage: params.target_t2_coverage ? Number(params.target_t2_coverage) : undefined,
        target_t3_coverage: params.target_t3_coverage ? Number(params.target_t3_coverage) : undefined,
        universe_size: params.universe_size ? Number(params.universe_size) : 42000,
      });
    },
  },
  {
    id: 'mdmScoring.shadowEval',
    domain: 'mdm_scoring',
    kind: 'query',
    label: 'Shadow Validation Comparator',
    description: 'Retrieves 30-day dual-run shadow comparison: legacy sufficiency vs 6-pillar composite quality with tie-break damping and audit log.',
    params: [
      { name: 'as_of', type: 'string', required: false },
    ],
    run: async (params) => {
      const asOf = params.as_of ? String(params.as_of) : undefined;
      return mdmScoringApi.shadowEval(asOf);
    },
  },
  {
    id: 'mdmScoring.simulateProfiles',
    domain: 'mdm_scoring',
    kind: 'query',
    label: 'Profile Sensitivity Simulation',
    description: 'Simulates vendor rank shifts, composite score deltas, and bundle financial impact between two weight profiles.',
    params: [
      { name: 'profile_a', type: 'object', required: true },
      { name: 'profile_b', type: 'object', required: true },
      { name: 'entity_domain', type: 'string', required: false },
      { name: 'universe_size', type: 'number', required: false },
    ],
    run: async (params) => {
      return mdmScoringApi.simulateProfiles({
        profile_a: params.profile_a as any,
        profile_b: params.profile_b as any,
        entity_domain: params.entity_domain ? String(params.entity_domain) : undefined,
        universe_size: params.universe_size ? Number(params.universe_size) : 42000,
      });
    },
  },
];

registerOperations(operations);

import { registerDomainComponents } from '../../studio-core/components/registry';
import { RadarScorecard } from './components/RadarScorecard';
import { BundleOptimizer } from './components/BundleOptimizer';
import { DisplacementTCOTable } from './components/DisplacementTCOTable';
import { MultiVendorDisplacement } from './components/MultiVendorDisplacement';
import { ShadowComparator } from './components/ShadowComparator';
import { HistoricalTrendsViewer } from './components/HistoricalTrendsViewer';
import { QualityTrendChart } from './components/QualityTrendChart';
import { ProfileSimulation } from './components/ProfileSimulation';

registerDomainComponents([
  {
    id: 'mdmScoring.ProfileSimulation',
    domain: 'mdm_scoring',
    label: 'Profile Sensitivity Simulator',
    description: 'Dual-column A/B weight profile simulator with delta radar overlay, rank shift analysis, and optimal bundle financial impact.',
    inputs: [
      { name: 'entity_domain', label: 'Entity Domain', type: 'string', required: false },
      { name: 'universe_size', label: 'Universe Size', type: 'number', required: false },
    ],
    events: [],
    render: ({ inputs }) => (
      <ProfileSimulation
        entityDomain={inputs.entity_domain ? String(inputs.entity_domain) : undefined}
        universeSize={inputs.universe_size ? Number(inputs.universe_size) : 42000}
      />
    ),
  },
  {
    id: 'mdmScoring.QualityTrendChart',
    domain: 'mdm_scoring',
    label: 'Quality Trend Trajectory & Watermarks',
    description: 'Interactive SVG multi-vendor time-series chart with tier watermark boundary lines and quality zone background shading.',
    inputs: [
      { name: 'entity_domain', label: 'Entity Domain', type: 'string', required: false },
      { name: 'initial_dimension', label: 'Initial Dimension', type: 'string', required: false },
      { name: 'height', label: 'Chart Height', type: 'number', required: false },
    ],
    events: [],
    render: ({ inputs }) => (
      <QualityTrendChart
        entityDomain={inputs.entity_domain ? String(inputs.entity_domain) : undefined}
        initialDimension={inputs.initial_dimension ? String(inputs.initial_dimension) : 'COMPOSITE'}
        height={inputs.height ? Number(inputs.height) : 380}
      />
    ),
  },
  {
    id: 'mdmScoring.HistoricalTrendsViewer',
    domain: 'mdm_scoring',
    label: 'Three-Tier Historical Trend Analytics',
    description: 'Multi-resolution historical trend viewer querying StarRocks, PostgreSQL, and Iceberg lakehouse stores.',
    inputs: [
      { name: 'entity_domain', label: 'Entity Domain', type: 'string', required: false },
    ],
    events: [],
    render: ({ inputs }) => (
      <HistoricalTrendsViewer
        entityDomain={inputs.entity_domain ? String(inputs.entity_domain) : undefined}
      />
    ),
  },
  {
    id: 'mdmScoring.RadarScorecard',
    domain: 'mdm_scoring',
    label: '360° Vendor Quality Radar',
    description: 'Interactive 6-pillar quality radar chart comparing multiple market data vendors.',
    inputs: [
      { name: 'as_of', label: 'As Of Date', type: 'string', required: false },
      { name: 'initial_vendors', label: 'Initial Selected Vendors', type: 'object', required: false },
    ],
    events: [],
    render: ({ inputs }) => (
      <RadarScorecard
        asOf={inputs.as_of ? String(inputs.as_of) : undefined}
        initialVendors={Array.isArray(inputs.initial_vendors) ? (inputs.initial_vendors as string[]) : undefined}
      />
    ),
  },
  {
    id: 'mdmScoring.BundleOptimizer',
    domain: 'mdm_scoring',
    label: 'Weighted Set Cover Bundle Optimizer',
    description: 'Interactive solver tool with Tier 1-3 coverage sliders, vendor constraints, and cost savings calculator.',
    inputs: [],
    events: [],
    render: () => (
      <BundleOptimizer
        onViewDisplacement={(dropped, replacements) => {
          const url = new URL(window.location.href);
          url.searchParams.set('tab', 'multi_displacement');
          window.history.pushState({}, '', url.toString());
          window.dispatchEvent(new PopStateEvent('popstate'));
        }}
      />
    ),
  },
  {
    id: 'mdmScoring.DisplacementTCOTable',
    domain: 'mdm_scoring',
    label: 'Displacement TCO Executive Comparison',
    description: 'Executive table comparing With Vendor vs Without Vendor across license savings, friction labor, lost SLA credits, and remediation drag.',
    inputs: [
      { name: 'candidate_vendor_id', label: 'Candidate Vendor ID', type: 'string', required: true },
      { name: 'as_of', label: 'As Of Date', type: 'string', required: false },
    ],
    events: [],
    render: ({ inputs }) => (
      <DisplacementTCOTable
        candidateVendorId={inputs.candidate_vendor_id ? String(inputs.candidate_vendor_id) : 'BBG'}
        asOf={inputs.as_of ? String(inputs.as_of) : undefined}
      />
    ),
  },
  {
    id: 'mdmScoring.MultiVendorDisplacement',
    domain: 'mdm_scoring',
    label: 'Multi-Vendor Displacement Tearsheet',
    description: 'Full procurement tearsheet for evaluating the combined TCO and survivorship impact of dropping multiple vendors and onboarding replacements.',
    inputs: [
      { name: 'initial_dropped_vendors', label: 'Initial Dropped Vendors', type: 'object', required: false },
      { name: 'initial_replacement_vendors', label: 'Initial Replacement Vendors', type: 'object', required: false },
      { name: 'universe_size', label: 'Universe Size', type: 'number', required: false },
    ],
    events: [],
    render: ({ inputs }) => (
      <MultiVendorDisplacement
        initialDroppedVendors={Array.isArray(inputs.initial_dropped_vendors) ? (inputs.initial_dropped_vendors as string[]) : ['BBG']}
        initialReplacementVendors={Array.isArray(inputs.initial_replacement_vendors) ? (inputs.initial_replacement_vendors as string[]) : ['ICE']}
        universeSize={inputs.universe_size ? Number(inputs.universe_size) : 42000}
      />
    ),
  },
  {
    id: 'mdmScoring.ShadowComparator',
    domain: 'mdm_scoring',
    label: 'Dual-Run Shadow Validation Comparator',
    description: 'Interactive comparator testing ranking stability, agreement rates, inversion rates, and recent shadow audit logs.',
    inputs: [
      { name: 'as_of', label: 'As Of Date', type: 'string', required: false },
    ],
    events: [],
    render: ({ inputs }) => (
      <ShadowComparator
        asOf={inputs.as_of ? String(inputs.as_of) : undefined}
      />
    ),
  },
]);
