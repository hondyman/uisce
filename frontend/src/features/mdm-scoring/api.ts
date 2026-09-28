import apiClient from '../../utils/apiClient';

export interface SubstitutionScore {
  vendor_id: string;
  attribute_code: string;
  tier: number;
  in_scope: number;
  available_count: number;
  valid_match_count: number;
  coverage_pct: number;
  sufficiency_rate_pct: number;
  conditional_sufficiency_pct: number;
  solo_rate_pct: number;
  solo_records_count: number;
  contribution_share_pct: number;
  override_endorsement_rate_pct: number;
}

export interface ValueForMoneyPoint {
  vendor_id: string;
  vendor_name: string;
  annual_cost: number;
  quality_index: number;
  marginal_quality_uplift: number;
  cost_per_quality_point: number;
  is_on_frontier: boolean;
}

export interface TierDisplacementSummary {
  tier: number;
  attributes_count: number;
  average_in_scope: number;
  unchanged_pct: number;
  changed_pct: number;
  now_null_pct: number;
}

export interface ResidualGap {
  attribute_code: string;
  tier: number;
  solo_rate_pct: number;
  records_lost: number;
}

export interface VendorDisplacementResult {
  dropped_vendor_id: string;
  dropped_vendor_name: string;
  annual_cost: number;
  displacement_readiness_pct: number;
  tier_summaries: TierDisplacementSummary[];
  residual_gaps: ResidualGap[];
  total_null_values: number;
  remediation_cost_est: number;
  net_first_year_benefit: number;
  cost_per_unique_value: number;
}

export interface VendorScorecardReport {
  as_of_date: string;
  universe_size: number;
  tiers_tracked: number;
  annual_spend_total: number;
  best_alt_sufficiency_t1: number;
  premium_solo_share_t1: number;
  displacement_readiness: number;
  substitution_matrix: SubstitutionScore[];
  frontier_points: ValueForMoneyPoint[];
  displacement_scenarios: VendorDisplacementResult[];
}

export interface AttributeTolerance {
  attribute_code: string;
  tier: number;
  match_type: string;
  tolerance_val: number;
  tier_weight: number;
  description: string;
}

const BASE = '/api/mdm/scoring';
const json = (body: unknown) => ({ method: 'POST', body: JSON.stringify(body), headers: { 'Content-Type': 'application/json' } });

export const mdmScoringApi = {
  scorecard: (universeSize = 42000, asOf?: string) =>
    apiClient<VendorScorecardReport>(`${BASE}/scorecard?universe_size=${universeSize}${asOf ? `&as_of=${asOf}` : ''}`),
  tolerances: () =>
    apiClient<Record<string, AttributeTolerance>>(`${BASE}/tolerances`),
  displacement: (droppedVendorId: string, hierarchy?: string[]) =>
    apiClient<VendorDisplacementResult>(`${BASE}/displacement`, json({
      dropped_vendor_id: droppedVendorId,
      vendor_hierarchy: hierarchy,
    })),
};
