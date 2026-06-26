/**
 * Alternative Investment Advisor Dashboard Types
 * 
 * Comprehensive TypeScript interfaces for the Alternative Investment
 * allocation and monitoring system.
 */

// ============================================================
// Core Enums and Status Types
// ============================================================

export type InvestmentStage = 
  | 'sourced'
  | 'screening'
  | 'due_diligence'
  | 'committee_review'
  | 'approved'
  | 'rejected'
  | 'invested';

export type RiskLevel = 'low' | 'moderate' | 'high' | 'very_high';

export type AssetClass = 
  | 'private_equity'
  | 'venture_capital'
  | 'real_estate'
  | 'hedge_funds'
  | 'private_credit'
  | 'infrastructure'
  | 'commodities'
  | 'other';

export type RebalanceTriggerType =
  | 'threshold_breach'
  | 'capital_event'
  | 'market_dislocation'
  | 'scheduled'
  | 'client_request';

export type ComplianceFilingStatus =
  | 'pending'
  | 'submitted'
  | 'acknowledged'
  | 'deficient'
  | 'completed';

export type ActionPriority = 'urgent' | 'high' | 'normal' | 'low';

// ============================================================
// Pipeline & Due Diligence Types
// ============================================================

export interface InvestmentOpportunity {
  id: string;
  fund_name: string;
  asset_class: AssetClass;
  vintage_year: number;
  target_size_millions: number;
  minimum_investment: number;
  expected_irr: number;
  expected_moic: number;
  investment_period_years: number;
  fund_life_years: number;
  management_fee_percent: number;
  carried_interest_percent: number;
  hurdle_rate_percent: number;
  stage: InvestmentStage;
  source: string;
  gp_track_record: GPTrackRecord;
  key_terms: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface GPTrackRecord {
  previous_funds: number;
  total_aum_billions: number;
  avg_net_irr: number;
  avg_moic: number;
  loss_ratio: number;
  team_tenure_years: number;
}

export interface AdvisorReview {
  id: string;
  opportunity_id: string;
  advisor_id: string;
  review_stage: InvestmentStage;
  recommendation: 'approve' | 'reject' | 'needs_info';
  risk_assessment: RiskAssessment;
  fit_score: number; // 0-100
  notes: string;
  attachments: ReviewAttachment[];
  created_at: string;
}

export interface RiskAssessment {
  overall_risk: RiskLevel;
  liquidity_risk: RiskLevel;
  market_risk: RiskLevel;
  operational_risk: RiskLevel;
  concentration_risk: RiskLevel;
  key_concerns: string[];
  mitigating_factors: string[];
}

export interface ReviewAttachment {
  id: string;
  filename: string;
  document_type: 'ppm' | 'lpa' | 'financials' | 'dd_report' | 'other';
  uploaded_at: string;
}

export interface CommitteePackage {
  id: string;
  opportunity_id: string;
  meeting_date: string;
  status: 'draft' | 'submitted' | 'reviewed' | 'approved' | 'rejected';
  executive_summary: string;
  investment_thesis: string;
  risk_factors: string[];
  recommendation: 'strong_buy' | 'buy' | 'hold' | 'pass';
  proposed_allocation_millions: number;
  voting_results?: VotingResults;
  created_at: string;
}

export interface VotingResults {
  votes_for: number;
  votes_against: number;
  abstentions: number;
  conditions: string[];
  final_decision: 'approved' | 'rejected' | 'tabled';
}

// ============================================================
// Allocation & Portfolio Types
// ============================================================

export interface AllocationRecommendation {
  id: string;
  portfolio_id: string;
  opportunity_id: string;
  recommended_amount: number;
  current_exposure_percent: number;
  post_allocation_exposure_percent: number;
  risk_contribution: number;
  diversification_benefit: number;
  fit_rationale: string[];
  constraints_satisfied: ConstraintCheck[];
  model_confidence: number;
  advisor_override?: AdvisorOverride;
  status: 'pending' | 'approved' | 'rejected' | 'executed';
  created_at: string;
}

export interface ConstraintCheck {
  constraint_name: string;
  constraint_type: 'regulatory' | 'policy' | 'client' | 'risk';
  current_value: number;
  limit_value: number;
  passed: boolean;
  headroom: number;
}

export interface AdvisorOverride {
  advisor_id: string;
  override_amount: number;
  reason: string;
  override_date: string;
}

export interface PortfolioAllocation {
  asset_class: AssetClass;
  target_percent: number;
  current_percent: number;
  variance_percent: number;
  market_value: number;
  positions: PositionSummary[];
}

export interface PositionSummary {
  investment_id: string;
  investment_name: string;
  commitment: number;
  called_capital: number;
  distributions: number;
  nav: number;
  irr: number;
  moic: number;
  vintage_year: number;
}

// ============================================================
// Rebalancing Types
// ============================================================

export interface RebalanceTrigger {
  id: string;
  portfolio_id: string;
  trigger_type: RebalanceTriggerType;
  severity: 'info' | 'warning' | 'critical';
  asset_class_affected: AssetClass;
  current_weight: number;
  target_weight: number;
  deviation_percent: number;
  suggested_actions: RebalanceAction[];
  auto_executable: boolean;
  acknowledged: boolean;
  created_at: string;
}

export interface RebalanceAction {
  action_type: 'increase' | 'decrease' | 'hold';
  investment_id?: string;
  investment_name?: string;
  amount: number;
  rationale: string;
  execution_priority: number;
  estimated_timeline_days: number;
}

// ============================================================
// Monitoring & Reporting Types
// ============================================================

export interface QuarterlyReview {
  id: string;
  portfolio_id: string;
  review_period: string; // e.g., "2024-Q3"
  total_commitment: number;
  total_called: number;
  total_distributed: number;
  total_nav: number;
  period_irr: number;
  inception_irr: number;
  moic: number;
  dpi: number;
  rvpi: number;
  tvpi: number;
  performance_by_asset_class: AssetClassPerformance[];
  top_performers: InvestmentPerformance[];
  underperformers: InvestmentPerformance[];
  capital_activity: CapitalActivity[];
  outlook: string;
  generated_at: string;
}

export interface AssetClassPerformance {
  asset_class: AssetClass;
  weight: number;
  irr: number;
  moic: number;
  benchmark_irr: number;
  relative_performance: number;
}

export interface InvestmentPerformance {
  investment_id: string;
  investment_name: string;
  irr: number;
  moic: number;
  attribution_contribution: number;
}

export interface CapitalActivity {
  investment_id: string;
  investment_name: string;
  activity_type: 'capital_call' | 'distribution' | 'recallable';
  amount: number;
  date: string;
}

// ============================================================
// Compliance Types
// ============================================================

export interface ComplianceFiling {
  id: string;
  portfolio_id: string;
  filing_type: 'form_pf' | 'form_adv' | 'schedule_k1' | 'annual_report' | 'other';
  reporting_period: string;
  deadline: string;
  status: ComplianceFilingStatus;
  submitted_at?: string;
  data_snapshot: Record<string, unknown>;
  validation_errors: ValidationError[];
  created_at: string;
}

export interface ValidationError {
  field: string;
  error_code: string;
  message: string;
  severity: 'error' | 'warning';
}

// ============================================================
// Dashboard Aggregate Types
// ============================================================

export interface AlternativesAdvisorDashboard {
  pipeline_overview: PipelineOverview;
  portfolio_health: PortfolioHealth;
  performance_attribution: PerformanceAttribution;
  risk_monitoring: RiskMonitoring;
  next_actions: NextAction[];
  last_updated: string;
}

export interface PipelineOverview {
  total_opportunities: number;
  by_stage: StageCount[];
  total_pipeline_value_millions: number;
  avg_time_to_decision_days: number;
  conversion_rate: number;
  recent_activity: PipelineActivity[];
}

export interface StageCount {
  stage: InvestmentStage;
  count: number;
  value_millions: number;
}

export interface PipelineActivity {
  opportunity_id: string;
  opportunity_name: string;
  activity_type: 'stage_change' | 'review_added' | 'document_uploaded' | 'meeting_scheduled';
  description: string;
  timestamp: string;
  actor: string;
}

export interface PortfolioHealth {
  total_aum_millions: number;
  alternatives_allocation_percent: number;
  target_allocation_percent: number;
  unfunded_commitments: number;
  available_liquidity: number;
  deployment_pace: DeploymentPace;
  concentration_metrics: ConcentrationMetrics;
  vintage_diversification: VintageDiversification[];
}

export interface DeploymentPace {
  ytd_called: number;
  ytd_distributed: number;
  projected_calls_12m: number;
  projected_distributions_12m: number;
  net_cash_flow_12m: number;
}

export interface ConcentrationMetrics {
  largest_position_percent: number;
  top_5_concentration_percent: number;
  single_manager_max_percent: number;
  geographic_concentration: GeographicConcentration[];
}

export interface GeographicConcentration {
  region: string;
  weight_percent: number;
}

export interface VintageDiversification {
  vintage_year: number;
  commitment: number;
  nav: number;
  weight_percent: number;
}

export interface PerformanceAttribution {
  total_portfolio_irr: number;
  alternatives_irr: number;
  benchmark_irr: number;
  alpha: number;
  attribution_by_asset_class: AttributionDetail[];
  attribution_by_vintage: AttributionDetail[];
  top_contributors: ContributionDetail[];
  bottom_contributors: ContributionDetail[];
}

export interface AttributionDetail {
  category: string;
  weight_percent: number;
  return_percent: number;
  contribution_bps: number;
}

export interface ContributionDetail {
  investment_name: string;
  return_percent: number;
  weight_percent: number;
  contribution_bps: number;
}

export interface RiskMonitoring {
  overall_risk_score: number; // 0-100
  risk_trend: 'improving' | 'stable' | 'deteriorating';
  alerts: RiskAlert[];
  stress_test_results: StressTestResult[];
  liquidity_coverage: LiquidityCoverage;
}

export interface RiskAlert {
  id: string;
  alert_type: 'threshold_breach' | 'concentration' | 'liquidity' | 'compliance' | 'market';
  severity: 'info' | 'warning' | 'critical';
  title: string;
  description: string;
  affected_investments: string[];
  suggested_action: string;
  created_at: string;
  acknowledged: boolean;
}

export interface StressTestResult {
  scenario_name: string;
  description: string;
  portfolio_impact_percent: number;
  alternatives_impact_percent: number;
  liquidity_impact_percent: number;
}

export interface LiquidityCoverage {
  current_liquidity_millions: number;
  unfunded_commitments: number;
  expected_calls_12m: number;
  coverage_ratio: number;
  minimum_required_ratio: number;
  status: 'adequate' | 'watch' | 'insufficient';
}

export interface NextAction {
  id: string;
  action_type: 'review_opportunity' | 'committee_meeting' | 'capital_call' | 'compliance_filing' | 'rebalance' | 'client_meeting';
  priority: ActionPriority;
  title: string;
  description: string;
  due_date: string;
  related_entity_id?: string;
  related_entity_type?: string;
  assigned_to?: string;
  status: 'pending' | 'in_progress' | 'completed' | 'overdue';
}

// ============================================================
// API Request/Response Types
// ============================================================

export interface CreateOpportunityRequest {
  fund_name: string;
  asset_class: AssetClass;
  vintage_year: number;
  target_size_millions: number;
  minimum_investment: number;
  expected_irr: number;
  expected_moic: number;
  investment_period_years: number;
  fund_life_years: number;
  management_fee_percent: number;
  carried_interest_percent: number;
  hurdle_rate_percent: number;
  source: string;
  gp_track_record: GPTrackRecord;
  key_terms?: Record<string, unknown>;
}

export interface SubmitReviewRequest {
  opportunity_id: string;
  recommendation: 'approve' | 'reject' | 'needs_info';
  risk_assessment: RiskAssessment;
  fit_score: number;
  notes: string;
}

export interface AllocationDecisionRequest {
  recommendation_id: string;
  decision: 'approve' | 'reject' | 'modify';
  final_amount?: number;
  override_reason?: string;
}

export interface RebalanceExecutionRequest {
  trigger_id: string;
  actions_to_execute: string[]; // action IDs
  execution_notes?: string;
}

// ============================================================
// GraphQL Subscription Types (for real-time updates)
// ============================================================

export interface PipelineUpdateEvent {
  event_type: 'opportunity_created' | 'stage_changed' | 'review_added' | 'decision_made';
  opportunity_id: string;
  opportunity_name: string;
  previous_stage?: InvestmentStage;
  current_stage: InvestmentStage;
  timestamp: string;
  actor: string;
}

export interface AllocationEvent {
  event_type: 'recommendation_generated' | 'allocation_approved' | 'allocation_executed';
  portfolio_id: string;
  recommendation_id: string;
  amount: number;
  investment_name: string;
  timestamp: string;
}

export interface RiskAlertEvent {
  alert: RiskAlert;
  portfolio_id: string;
  requires_immediate_action: boolean;
}
