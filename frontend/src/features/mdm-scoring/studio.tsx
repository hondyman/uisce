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
    ],
    fields: [
      { name: 'substitution_matrix' },
      { name: 'frontier_points' },
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
      return mdmScoringApi.scorecard(uSize, asOf);
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
    ],
    run: async (params) => {
      const pid = params.pipeline_id ? String(params.pipeline_id) : undefined;
      return mdmScoringApi.runPipeline(pid);
    },
  },
];

registerOperations(operations);
