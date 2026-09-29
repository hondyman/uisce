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
  friction_savings?: number;
  forfeited_sla_credits?: number;
  net_tco_benefit?: number;
}

export interface QualityComponents {
  sufficiency: number;
  coverage: number;
  sla: number;
  stability: number;
  friction: number;
  licensing: number;
}

export interface VendorDimensionProfile {
  vendor_id: string;
  vendor_name: string;
  annual_spend: number;
  components: QualityComponents;
  composite_quality: number;
  cost_per_quality_point: number;
  avg_delivery_lag_mins: number;
  sla_breach_count: number;
  total_feeds_received: number;
  total_revisions: number;
  revision_rate_pct: number;
  defect_tickets_count: number;
  investigation_hours: number;
  friction_cost: number;
  sla_credits: number;
  rights_score: number;
  is_baseline_seeded: boolean;
}

export interface ResidualCoverageGap {
  tier: number;
  target_coverage: number;
  achieved_coverage: number;
  missing_entity_count: number;
  notes: string;
}

export interface OptimalBundleRequest {
  target_t1_coverage?: number;
  target_t2_coverage?: number;
  target_t3_coverage?: number;
  max_budget?: number;
  mandatory_vendors?: string[];
  excluded_vendors?: string[];
  universe_size?: number;
}

export interface OptimalBundleResult {
  selected_vendors: string[];
  selected_vendor_names: string[];
  total_annual_cost: number;
  previous_total_cost: number;
  annual_savings: number;
  savings_pct: number;
  t1_coverage_achieved: number;
  t2_coverage_achieved: number;
  t3_coverage_achieved: number;
  was_relaxed: boolean;
  residual_gaps?: ResidualCoverageGap[];
  solver_execution_ms: number;
  solver_strategy: string;
  solver_partial?: boolean;
}

export interface WeightProfile {
  profile_id: number;
  profile_name: string;
  is_active: boolean;
  weight_suff: number;
  weight_cov: number;
  weight_sla: number;
  weight_stab: number;
  weight_oer: number;
  weight_lic: number;
  created_by: string;
  change_reason: string;
}

export interface VendorEntityBreakdown {
  vendor_id: string;
  vendor_name: string;
  entity_domain: string;
  annual_cost: number;
  quality_index: number;
  cost_per_quality_point: number;
  sufficiency_rate_pct: number;
  coverage_pct: number;
  solo_rate_pct: number;
  is_on_frontier: boolean;
  attribute_count: number;
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
  entity_breakdowns: VendorEntityBreakdown[];
  entity_domains: string[];
  displacement_scenarios: VendorDisplacementResult[];
  dimension_profiles?: VendorDimensionProfile[];
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

export interface MultiVendorDisplacementRequest {
  dropped_vendor_ids: string[];
  replacement_vendor_ids?: string[];
  target_t1_coverage?: number;
  target_t2_coverage?: number;
  target_t3_coverage?: number;
  universe_size?: number;
}

export interface CombinedTCOBreakdown {
  license_savings_total: number;
  friction_delta_total: number;
  forfeited_sla_credits_total: number;
  remediation_drag_total: number;
  replacement_cost_delta: number;
  net_first_year_tco_benefit: number;
  net_annual_tco_benefit: number;
  payback_months: number;
}

export interface SingleVendorTCO {
  vendor_id: string;
  vendor_name: string;
  annual_savings: number;
  friction_change: number;
  sla_credits_lost: number;
  remediation_cost: number;
  net_tco_benefit: number;
  displacement_readiness_pct: number;
}

export interface TierCoverageDelta {
  tier: number;
  coverage_before: number;
  coverage_after: number;
  delta_pct: number;
  meets_threshold: boolean;
  threshold: number;
}

export interface ResidualGapAttribute {
  attribute_code: string;
  entity_domain: string;
  tier: number;
  entities_affected: number;
  sole_source_vendor_id: string;
  suggested_substitute_id?: string;
  estimated_sub_license_cost: number;
  is_tier1_critical: boolean;
}

export interface SubLicenseProposal {
  vendor_id: string;
  vendor_name: string;
  attributes_covered: string[];
  entities_covered: number;
  estimated_annual_cost: number;
  tier_priority: number;
}

export interface DisplacementGapReport {
  total_entities_affected: number;
  tier1_gaps: ResidualGapAttribute[];
  tier2_gaps: ResidualGapAttribute[];
  tier3_gaps: ResidualGapAttribute[];
  recommended_sub_licenses: SubLicenseProposal[];
  estimated_gap_remediation_cost: number;
  net_negotiation_leverage: number;
}

export interface MultiVendorDisplacementResult {
  dropped_vendor_ids: string[];
  replacement_vendor_ids: string[];
  combined_tco: CombinedTCOBreakdown;
  per_vendor_breakdown: SingleVendorTCO[];
  tier_coverage_deltas: TierCoverageDelta[];
  residual_gaps: ResidualGap[];
  gap_report?: DisplacementGapReport;
  solver_partial?: boolean;
  generated_at: string;
}

export type ShadowStabilityState = 'STABLE' | 'TIE_BOUND' | 'MOVED';
export type ShadowAttribution = 'COMPARABLE' | 'SCALED' | 'MODEL_CHANGED' | 'UNAVAILABLE';

export interface ShadowBasis {
  weights_changed: boolean;
  scale_factor: number;
  decomposition: string;
  old_ranking: 'VALID' | 'DEGENERATE';
  old_spread: number;
  input_basis: 'SAME_SNAPSHOT_RULER_ONLY' | 'HISTORICAL_SNAPSHOT';
}

export interface DroppedVendorInfo {
  vendor_id: string;
  reason: string;
}

export interface ShadowVendorComparison {
  vendor_id: string;
  vendor_name: string;
  old_score: number;
  new_score: number;
  old_score_normalized: number;
  new_score_normalized: number;
  old_rank: number;
  new_rank: number;
  rank_delta: number | null;
  score_delta: number;
  model_effect: number;
  input_effect: number;
  scale_factor: number;
  attribution: ShadowAttribution;
  neighbor_gap: number | null;
  stability: ShadowStabilityState;
  is_tie_stable?: boolean;
  safety_alert?: string;
}

export interface ShadowRunLogEntry {
  run_id: number;
  tenant_id: string;
  executed_at: string;
  candidate_vendor_ids: string[];
  dropped_vendor_ids: string[];
  replacement_vendor_ids: string[];
  universe_size: number;
  t1_concordance_delta: number;
  t2_concordance_delta: number;
  t3_concordance_delta: number;
  gross_annual_savings: number;
  net_tco_benefit: number;
  payback_months: number;
  solver_latency_ms: number;
  solver_strategy: string;
  solver_partial: boolean;
  status: string;
  metadata?: string;
}

export interface ShadowValidationReport {
  as_of_date: string;
  tenant_id: string;
  weight_profile_name: string;
  basis: ShadowBasis;
  is_stable: boolean;
  max_rank_delta: number;
  pairwise_agreement_pct: number;
  rank_inversion_rate_pct: number;
  comparisons: ShadowVendorComparison[];
  dropped_vendors?: DroppedVendorInfo[];
  recent_run_history?: ShadowRunLogEntry[];
}

export interface BoundaryConflict {
  vendor_id: string;
  date: string;
  tier_a: string;
  tier_b: string;
  value_a: number;
  value_b: number;
}

export interface TrendPoint {
  as_of_date: string;
  vendor_id: string;
  vendor_name: string;
  dimension: string;
  score_value: number;
  raw_metric_value?: number;
  tier1_coverage?: number;
  tier2_coverage?: number;
  tier3_coverage?: number;
  rank_position?: number;
  storage_tier: 'HOT' | 'WARM' | 'COLD';
}

export interface VendorTrendSeries {
  vendor_id: string;
  vendor_name: string;
  dimension: string;
  points: TrendPoint[];
  min_score: number;
  max_score: number;
  avg_score: number;
  avg_score_weighted: number;
  slope_per_30d: number;
  trend_direction: 'IMPROVING' | 'DECLINING' | 'STABLE' | 'INSUFFICIENT_DATA';
  trend_by_tier?: Record<string, string>;
}

export interface WatermarkBoundaries {
  hot_window_days: number;
  warm_window_days: number;
  hot_start?: string;
  hot_end?: string;
  warm_start?: string;
  warm_end?: string;
  cold_start?: string;
  cold_end?: string;
}

export interface TrendAnalysisReport {
  tenant_id: string;
  dimension: string;
  date_from: string;
  date_to: string;
  tiers_planned: string[];
  tiers_hit: string[];
  empty_tiers: string[];
  tiers_queried: string[];
  boundary_conflicts: BoundaryConflict[];
  boundary_conflict_count: number;
  series: VendorTrendSeries[];
  watermarks: WatermarkBoundaries;
  generated_at: string;
}

export const mdmScoringApi = {
  trends: (dimension = 'COMPOSITE', vendors?: string[], dateFrom?: string, dateTo?: string, entityDomain?: string) => {
    const params = new URLSearchParams();
    if (dimension) params.set('dimension', dimension);
    if (vendors && vendors.length > 0) params.set('vendors', vendors.join(','));
    if (dateFrom) params.set('date_from', dateFrom);
    if (dateTo) params.set('date_to', dateTo);
    if (entityDomain) params.set('entity_domain', entityDomain);
    return apiClient<TrendAnalysisReport>(`${BASE}/trends?${params.toString()}`);
  },
  shadowEval: (asOf?: string) =>
    apiClient<ShadowValidationReport>(`${BASE}/shadow-eval${asOf ? `?as_of=${asOf}` : ''}`),
  scorecard: (universeSize = 42000, asOf?: string, entityDomain?: string) =>
    apiClient<VendorScorecardReport>(
      `${BASE}/scorecard?universe_size=${universeSize}${asOf ? `&as_of=${asOf}` : ''}${entityDomain ? `&entity_domain=${encodeURIComponent(entityDomain)}` : ''}`
    ),
  dimensions: (asOf?: string) =>
    apiClient<VendorDimensionProfile[]>(`${BASE}/dimensions${asOf ? `?as_of=${asOf}` : ''}`),
  optimizeBundle: (req: OptimalBundleRequest) =>
    apiClient<OptimalBundleResult>(`${BASE}/optimize-bundle`, json(req)),
  weightProfiles: () =>
    apiClient<WeightProfile[]>(`${BASE}/weight-profiles`),
  saveWeightProfile: (profile: Partial<WeightProfile>) =>
    apiClient<WeightProfile>(`${BASE}/weight-profiles`, json(profile)),
  tolerances: () =>
    apiClient<Record<string, AttributeTolerance>>(`${BASE}/tolerances`),
  displacement: (droppedVendorId: string, hierarchy?: string[]) =>
    apiClient<VendorDisplacementResult>(`${BASE}/displacement`, json({
      dropped_vendor_id: droppedVendorId,
      vendor_hierarchy: hierarchy,
    })),
  displacementMulti: (req: MultiVendorDisplacementRequest) =>
    apiClient<MultiVendorDisplacementResult>(`${BASE}/displacement-multi`, json(req)),
  syncMart: (asOf?: string) =>
    apiClient<{ status: string; records_synced: number }>(`${BASE}/sync-mart${asOf ? `?as_of=${asOf}` : ''}`, { method: 'POST' }),
  updateSpend: (vendorId: string, annualCost: number, entityDomain?: string) =>
    apiClient<{ status: string; vendor_id: string; annual_cost: number; message: string }>(
      `${BASE}/spend`,
      json({ vendor_id: vendorId, annual_cost: annualCost, entity_domain: entityDomain })
    ),
  runPipeline: async (pipelineId?: string, universeSize = 42000) => {
    try {
      return await apiClient<{ run_id: string; status: string; records_ingested?: number; message?: string }>(
        `${BASE}/run-ingest`,
        json({ pipeline_id: pipelineId, universe_size: universeSize })
      );
    } catch {
      return apiClient<{ run_id: string; status: string }>(
        `/api/data-pipelines/${pipelineId || 'a11c0001-0001-4000-8000-000000000099'}/runs`,
        { method: 'POST' }
      );
    }
  },
};


