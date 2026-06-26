// Central frontend types entrypoint.
// Re-export the more specific type modules under `./types/*` and provide
// minimal aliases for a handful of domain types that are referenced across
// many components so the editor/TypeScript server can resolve imports.

export * from './types/index';
export * from './types/types';

// The codebase expects these higher-level domain types to exist. For now
// provide permissive aliases so imports resolve; later these can be
// replaced with stricter interfaces as the frontend types are expanded.
// ---- High-impact concrete types (sourced from backend models) ----

export type Explain = {
  // Backend-style fields
  used_preagg?: string;
  routing_reason?: string;
  rule_id?: string;
  freshness?: string;

  // Frontend-friendly / UI fields
  preagg_hit?: boolean;
  preagg_name?: string;
  fallback_reason?: string;
  scan_size_estimate?: number | null;
  partitions_pruned?: string[] | number[] | null;
  optimization_suggestions?: string[];
  // allow other ad-hoc fields
  [key: string]: unknown;
} | null;

export interface ExplorerColumn {
  name: string;
  type: string;
}

export interface PageInfo {
  limit?: number | null;
  offset?: number | null;
  // Some components expect snake_case, others camelCase. Provide both.
  has_next?: boolean;
  hasNext?: boolean;
  total_count?: number | null;
  totalCount?: number | null;
}

export interface CompileResult {
  sql?: string;
  graphql?: string;
  explain?: Explain | null;
}

export interface ExecuteResult {
  columns: ExplorerColumn[];
  rows: Record<string, unknown>[];
  page: PageInfo;
  duration_ms: number;
  used_preaggregation?: string;
  sql?: string;
  graphql?: string;
  explain?: Explain | null;
}

export interface QueryState {
  measures: string[];
  dimensions: string[];
  filters: Array<{ field: string; op: string; values: string[] }>;
  order?: Array<{ field: string; dir: string } | [string, string]>;
  limit?: number | null;
  offset?: number | null;
}

export interface SavedQuery {
  id: string;
  name: string;
  view_name?: string;
  tags?: string[];
  owner_user_id?: string;
  last_run_at?: string | null;
  last_duration_ms?: number | null;
  preview_available?: boolean;
}

export interface FullSavedQuery {
  id: string;
  name: string;
  description?: string | null;
  preview_diff?: Record<string, unknown>;
  tags?: string[];
  view_name: string;
  query: QueryState | Record<string, unknown>;
  viz_config?: Record<string, unknown>;
  preview?: unknown;
  last_run_at?: string | null;
  last_duration_ms?: number | null;
  last_row_count?: number | null;
  is_deleted?: boolean;
  owner_user_id?: string;
  owner_tenant_id?: string;
  created_at?: string;
  updated_at?: string;
}

export interface WorkbookTab {
  id?: string;
  workbook_id?: string;
  title: string;
  view_name: string;
  query: QueryState | Record<string, unknown>;
  viz_config?: Record<string, unknown>;
  position: number;
}

export interface Workbook {
  id: string;
  name: string;
  description?: string | null;
  owner_user_id?: string;
  tags?: string[];
  created_at?: string;
  updated_at?: string;
}

export interface FullWorkbook extends Workbook {
  tabs: WorkbookTab[];
}

export interface SemanticMember {
  name: string;
  label?: string;
  type?: string;
  description?: string | null;
}

export interface SemanticViewMeta {
  id: string;
  name: string;
  description?: string;
  owner?: string;
  certified?: boolean;
  updated_at?: string;
  dimensions?: SemanticMember[];
  metrics?: SemanticMember[];
}

export interface SemanticQuery {
  dimensions: string[];
  metrics: string[];
  filters?: Array<{ field: string; op: string; values: string[] }>;
  order?: Array<{ field: string; dir: string } | [string, string]>;
  limit?: number;
}

export interface ViewMeta {
  name: string;
  schema?: string;
  description?: string;
  tags?: string[];
  owner?: string;
  certified?: boolean;
  // Optional UI-friendly members that some components expect to receive
  measures?: ViewMember[];
  dimensions?: ViewMember[];
  templates?: ViewTemplate[];
}

export interface QueryTemplateMeta {
  id: string;
  name: string;
  description?: string;
  semantic_view?: string;
  tags?: string[];
  certified?: boolean;
}

export interface QueryTemplate extends QueryTemplateMeta {
  default_dimensions?: string[];
  default_metrics?: string[];
  required_filters?: Record<string, unknown>;
  owner_user_id?: string;
  updated_at?: string;
}
export type SemanticModelClaim = {
  id: string;
  user_id: string;
  tenant_id: string;
  model_id: string;
  permission: 'read' | 'write';
  scope?: string[];
  granted_by: string; // e.g., 'role:analyst', 'direct_grant', 'bundle:finance_bundle'
  source_id?: string;
  granted_at: string;
  expires_at?: string | null;
  renewal_requested?: boolean;
  renewed_at?: string | null;
  revoked_at?: string | null;
  last_used_at?: string | null;
  status: 'active' | 'expiring' | 'renewal_requested' | 'expired' | 'revoked';
};
export type HistoryEntry = {
  id: string;
  name: string;
  request?: QueryState;
  last_run_at?: string | null;
  last_duration_ms?: number | null;
};
export interface FolderItemDetail {
  item_type: string;
  item_id: string;
  name: string;
  position: number;
}

export interface FullFolder {
  id: string;
  name: string;
  description?: string | null;
  owner_user_id?: string;
  scope_type?: string | null;
  scope_id?: string | null;
  tags?: string[];
  created_at?: string;
  updated_at?: string;
  items: FolderItemDetail[];
}

// Governance-related types (kept for compatibility with api.ts)
export type SemanticModelAccessRequest = any;
export type SemanticModelRoleClaim = any;
export type ClaimSimulationResult = any;
export type AccessControlAuditLog = any;
export type GovernanceSnapshot = any;
export type SemanticSearchResult = {
  type: 'query' | 'workbook';
  id: string;
  name: string;
  description?: { String: string; Valid: boolean };
  score: number;
  certified: boolean;
  popular: boolean;
  has_access: boolean;
  is_restricted: boolean;
  reason?: string;
  preview?: any;
  owner_user_id: string;
  // Optional fields used by ExplainMatchModal and other UI components
  matched_concepts?: string[];
  source_summary?: string;
};

export interface PolicySimulationResult {
  affected_claims: { added: number; modified: number; removed: number };
  affected_users: string[]; // list of user IDs
  affected_assets: string[]; // list of asset IDs
  risk_flags: string[];
}

// ---- Index Monitor Types ----

export interface IndexJob {
  id: string;
  job_type: 'full' | 'incremental' | 'claim-sync';
  started_at: string;
  completed_at?: string;
  status: 'pending' | 'running' | 'success' | 'failed';
  affected_assets: number;
  triggered_by: string;
}

export interface AssetFreshness {
  asset_id: string;
  asset_type: string;
  asset_name: string;
  last_indexed_at: string;
  certified: boolean;
}

export interface IndexMonitorSnapshot {
  last_full_refresh: string;
  certified_coverage: number;
  semantic_health_score: number;
  recent_jobs: IndexJob[];
  stale_assets: AssetFreshness[];
  unindexed_asset_count: number;
  claim_alignment: number;
  usage_coverage: number;
  audit_completeness: number;
  risk_exposure: number;
}

// ---- Claim Lifecycle Types ----

export interface ClaimLifecycleEvent {
  id: string;
  claim_id: string;
  event_type: 'granted' | 'renewal_requested' | 'renewed' | 'revoked' | 'expired';
  actor_user_id: string;
  timestamp: string;
  notes?: string;
}

export interface ClaimLifecycleSnapshot {
  active_count: number;
  expiring_soon_count: number;
  renewal_requested_count: number;
  expired_count: number;
  revoked_count: number;
  recent_events: ClaimLifecycleEvent[];
}

// ---- Access Control Policy Types ----

export interface AccessControlPolicy {
  id: string;
  policy_id: string;
  scope: string;
  role: string;
  permissions: string[];
  duration_days: number;
  requires_certification: boolean;
  max_claims_per_user: number;
  approval_threshold: number;
  renewal_conditions: {
    usage_within_days?: number;
    review_required?: boolean;
  };
  created_at: string;
  updated_at: string;
}

// ---- Claim Aware Lineage Types ----

export interface ClaimAwareLineageNode {
  id: string;
  type: string;
  label: string;
  data?: Record<string, unknown>;
  visibility: 'full' | 'partial' | 'none';
  reason?: string;
}

export interface ClaimAwareLineageEdge {
  source: string;
  target: string;
  label?: string;
}

export interface ClaimAwareLineageGraphData {
  nodes: ClaimAwareLineageNode[];
  edges: ClaimAwareLineageEdge[];
}

// ---- Semantic Notification Types ----

export interface SemanticNotification {
  id: string;
  event_type: string;
  asset_id: string;
  asset_type: string;
  message: string;
  triggered_by: string;
  timestamp: string;
  is_read: boolean;
  status: 'sent' | 'suppressed' | 'escalated' | 'resolved';
  routing_rule_id?: string;
  routing_trace?: {
    rule_id: string;
    resolved_recipients: string[];
    reason?: string;
  };
}

// Export Tenant/Product/DataSource so other modules importing '../../../types'
// (relative paths) continue to work.
export type Tenant = import('./types/index').Tenant;
export type Product = import('./types/index').Product;
export type DataSource = import('./types/index').DataSource;

// ---- Advanced Governance Types ----

export interface ClaimSuggestion {
  id: string;
  user_id: string;
  model_id: string;
  suggested_permission: string;
  reason: string;
  evidence: {
    query_count: number;
    last_queried: string;
  };
  status: 'new' | 'dismissed' | 'granted';
  created_at: string;
}

export interface ClaimBundle {
  id: string;
  name: string;
  description: string;
  created_by: string;
  updated_at: string;
}

// Misc JSONB/raw fields from backend - prefer unknown over any
export type Jsonb = unknown;

export interface GovernanceHeatmapDataPoint {
  domain: string;
  certified_model_percent: number;
  claim_density: number;
  risky_claim_count: number;
  unresolved_request_count: number;
  claim_drift_count: number;
}

export interface ClaimConflict {
  id: string;
  user_id: string;
  model_id: string;
  conflict_type: 'overlap' | 'contradiction';
  details: {
    description: string;
    conflicting_claims: { permission: string; source: string }[];
  };
  detected_at: string;
  status: 'new' | 'resolved';
  resolution_action?: string;
}

export interface GrantClaimRequest {
  user_id: string;
  tenant_id: string;
  model_id: string;
  permission: string;
  expires_at?: string;
}

export interface EvaluateAccessRequest {
  user_id: string;
  tenant_id: string;
  asset_id: string;
  action: string;
}

export interface EvaluateAccessResponse {
  decision: 'allow' | 'deny' | 'partial';
  reason: string;
  allowed_scope?: string[];
  decision_id: string;
}

export interface AccessDecisionTrace {
  id: string;
  decision_log_id: string;
  user_id: string;
  asset_id: string;
  action: string;
  decision: string;
  evaluated_claims: any; // JSONB
  matched_policies: any; // JSONB
  tenant_scope: string;
  reason: string;
  evaluated_at: string;
}

export interface SimulatedClaim {
  model_id: string;
  permission: string;
}

export interface SimulateAccessRequest {
  user_id: string;
  tenant_id: string;
  asset_id: string;
  action: string;
  simulated_claims?: SimulatedClaim[];
}

// ---- Governance Cockpit Types ----

export interface GovernanceHealthScore {
  score: number;
  certified_coverage: number;
  claim_alignment: number;
  usage_coverage: number;
  risk_exposure: number;
}

export interface GovernanceCockpitSnapshot {
  id: string;
  tenant_id: string;
  timestamp: string;
  health_score: GovernanceHealthScore;
  active_claims_count: number;
  conflict_count: number;
  drift_count: number;
  tenant_isolation_status: 'healthy' | 'at_risk' | 'unknown';
  recent_decisions: AccessDecisionTrace[];
  policy_count: number;
  simulation_count: number;
  suppressed_alert_count: number;
  escalated_alert_count: number;
  automation_status: 'running' | 'paused';
  auto_resolved_count: number;
}

// ---- Governance Automation Types ----

export interface AutomationPolicy {
  id: string;
  policy_id: string;
  description: string;
  trigger: string;
  conditions: Record<string, unknown>;
  action: string;
  is_enabled: boolean;
  updated_at: string;
}

export interface AutomationLog {
  id: string;
  timestamp: string;
  policy_id: string;
  action: string;
  target_type: string;
  target_id: string;
  details: Record<string, unknown>;
  status: 'success' | 'failed' | 'undone';
  undone_by?: string;
  undone_at?: string;
}

// ---- Notification Routing Types ----

export interface EscalationCondition {
  asset_certified?: boolean;
  change_type?: string[];
  risk_flags?: string[];
  risk_score_gte?: number;
}

export interface SuppressionCondition {
  asset_certified?: boolean;
  change_type?: string[];
  risk_score_lte?: number;
}

export interface NotificationRoutingRule {
  id: string;
  rule_id: string;
  trigger: string;
  scope: string;
  asset_type: string;
  routing_logic: {
    notify: string[];
    exclude?: string[];
    suppress_if?: SuppressionCondition;
    escalate_if?: EscalationCondition;
    escalate_to?: string[];
  };
  updated_at: string;
  updated_by: string;
}

export interface SemanticChangeEvent {
  asset_id: string;
  asset_type: string;
  change_type: 'claim_grant' | 'certification_revoked' | 'metric_updated';
  user_id: string;
  asset_sensitivity: 'low' | 'medium' | 'high';
  details?: Record<string, any>;
}

// Additional permissive aliases for many domain types referenced throughout
// the frontend. These keep the typechecker and editors happy while the
// canonical types get incrementally implemented.
export interface ExplorerAlert {
  id: string;
  user_id?: string;
  asset_id?: string;
  asset_type?: string;
  alert_type?: string;
  message: string;
  severity?: 'info' | 'warning' | 'critical' | string;
  triggered_at: string;
  read?: boolean;
  created_at?: string;
  // Backwards compatibility
  is_read?: boolean;
}

export interface DashboardTile {
  id: string;
  dashboard_id?: string;
  title?: string;
  type?: string; // e.g., 'chart', 'metric', 'table'
  layout?: any;
  config?: any;
  created_at?: string;
  updated_at?: string;
}
// Workbook is defined above as a concrete interface.
export type SuggestedQuery = any;
export type PreviewDiff = any;
export interface FolderAnalytics {
  run_count_30d: number;
  export_count_30d: number;
  viewer_count_30d: number;
  updated_at?: string;
}
export type SemanticSearchRequest = any;
export type SearchFeedbackRequest = any;
export type Goal = any;
export type Tour = any;
export type NLQTranslateRequest = any;
export type NLQTranslateResponse = any;
export type LineageGraphData = any;
export type ImpactAnalysis = any;
// QueryTemplate defined above as a concrete interface
export type Approval = any;
export type DashboardSnapshot = any;
export type SemanticViewVersion = any;
export type ClaimSimulationRequest = any;
export type ProposedClaim = any;
// FolderItemDetail is defined above

// Concrete frontend types derived from backend models
export interface Comment {
  id: string;
  asset_id: string;
  asset_type: string; // 'query' | 'workbook' | 'tab'
  author_user_id: string;
  body: string;
  created_at: string;
  resolved: boolean;
  parent_id?: string | null;
}

export interface ColumnMeta {
  name: string;
  type: string;
  label?: string;
  format?: string;
}

export type VizConfig = any; // Stored as free-form JSON in backend; keep as 'any' for now

// FolderDiff is returned as a map of sections to arrays of FolderItemDetail
export type FolderDiff = Record<string, FolderItemDetail[]>;

export interface DuplicateQueryCluster {
  fingerprint: string;
  queries: Array<{ id: string; name: string; view_name?: string; last_run_at?: string | null }>;
}
// ColumnMeta defined above.
export interface ViewMember {
  name: string;
  label?: string;
  description?: string | null;
  type?: string;
  pii?: boolean;
}
export type TabState = any;
export type SearchFilters = any;
// MemberDiffItem is defined concretely elsewhere when available.
// SemanticMember defined above as a concrete interface
export type AffectedModel = any;
// SnapshotDiffItem defined elsewhere.
export interface ViewTemplate {
  id?: string;
  name: string;
  description?: string;
  query?: Partial<QueryState>;
}
export type TourStep = any;
// Explain is defined above as a concrete interface.
export type Filter = any;

// ---- Natural Language Query Types ----

export interface NLQueryRequest {
  text: string;
  user_id?: string;
  tenant_id?: string;
  datasource?: string;
  conversation_id?: string;
  context?: Record<string, string>;
}

export interface ParsedIntent {
  metrics: string[];
  dimensions: string[];
  filters: IntentFilter[];
  time_range?: TimeRange;
  aggregation?: string;
  confidence: number;
  raw_entities: Record<string, string>;
}

export interface IntentFilter {
  field: string;
  operator: string;
  values: string[];
}

export interface TimeRange {
  start?: string;
  end?: string;
  label: string;
}

export interface GeneratedQuery {
  sql: string;
  semantic_sql?: string;
  measures: string[];
  dimensions: string[];
  filters: QueryFilter[];
  order_by?: OrderBySpec[];
}

export interface QueryFilter {
  field: string;
  operator: string;
  value: string;
}

export interface OrderBySpec {
  field: string;
  dir: string;
}

export interface GovernanceDiff {
  blocked_metrics?: string[];
  blocked_dimensions?: string[];
  added_filters?: QueryFilter[];
  removed_filters?: QueryFilter[];
  applied_policies?: AppliedPolicy[];
}

export interface AppliedPolicy {
  policy_id: string;
  rule_id: string;
  action: string;
  reason: string;
}

export interface NLQueryResponse {
  original_text: string;
  parsed_intent: ParsedIntent;
  generated_query: GeneratedQuery;
  governance_diff: GovernanceDiff;
  compliance_notes: string[];
  warnings: string[];
  query_id: string;
  timestamp: string;
}

export interface NLQuerySuggestion {
  text: string;
  category: string;
  confidence: number;
}

// ---- Conversation Types ----

export interface ConversationContext {
  conversation_id: string;
  user_id: string;
  tenant_id: string;
  datasource: string;
  query_history: ConversationQuery[];
  context_data: Record<string, any>;
  last_activity: string;
  created_at: string;
}

export interface ConversationQuery {
  query_id: string;
  user_query: string;
  parsed_intent?: ParsedIntent;
  generated_sql: string;
  executed_at: string;
  success: boolean;
  context_refs: string[];
}

export interface ConversationSummary {
  conversation_id: string;
  user_id: string;
  tenant_id: string;
  datasource: string;
  query_count: number;
  last_activity: string;
  created_at: string;
  duration: string;
  query_history: ConversationQuerySummary[];
  insights: ConversationInsights;
}

export interface ConversationQuerySummary {
  query_id: string;
  user_query: string;
  executed_at: string;
  success: boolean;
}

export interface ConversationInsights {
  total_queries: number;
  successful_queries: number;
  failed_queries: number;
  avg_confidence: number;
  common_metrics: string[];
  common_dimensions: string[];
}

// ---- Conversational Query Refinement Types ----

export interface RefinementContext {
  conversation_id: string;
  current_state: ConversationState;
  current_query: GeneratedQuery | null;
  messages: ConversationMessage[];
  clarifications: ClarificationRequest[];
  suggestions: QuerySuggestion[];
  compliance_status: ComplianceStatus;
  last_updated: string;
}

export interface ConversationMessage {
  id: string;
  type: 'user' | 'system' | 'clarification' | 'suggestion';
  content: string;
  timestamp: string;
  metadata?: Record<string, any>;
}

export interface ClarificationRequest {
  id: string;
  question: string;
  options?: string[];
  required: boolean;
  timestamp: string;
}

export interface QuerySuggestion {
  id: string;
  description: string;
  query_diff: string;
  confidence: number;
  reasoning: string;
  compliance_impact: ComplianceImpact;
  timestamp: string;
}

export interface ComplianceStatus {
  is_compliant: boolean;
  violations: ComplianceViolation[];
  applied_policies: AppliedPolicy[];
  risk_level: 'low' | 'medium' | 'high';
}

export interface ComplianceViolation {
  type: 'blocked_metric' | 'blocked_dimension' | 'missing_filter' | 'policy_violation';
  description: string;
  severity: 'low' | 'medium' | 'high';
  suggested_fix?: string;
}

export interface ComplianceImpact {
  risk_change: 'improved' | 'unchanged' | 'worsened';
  new_violations: ComplianceViolation[];
  resolved_violations: string[];
}

export type ConversationState =
  | 'initializing'
  | 'clarifying'
  | 'suggesting'
  | 'refining'
  | 'ready'
  | 'committed'
  | 'error';

// PoP Anomaly Detection Types
export interface AnomalyDetectionMethod {
  id: string;
  name: string;
  description: string;
  parameters: Record<string, any>;
}

export interface DetectionConfig {
  method: string;
  sensitivity: number;
  window_size: number;
  min_data_points: number;
  seasonal_period?: number;
  custom_parameters: Record<string, any>;
}

export interface AnomalyDetectionResult {
  success: boolean;
  anomalies: PoPAnomaly[];
  count: number;
  method_used: string;
  detection_time: string;
}

export interface PoPAnomaly {
  id: string;
  metric_id: string;
  computation_id: string;
  anomaly_type: string;
  severity: string;
  confidence?: number;
  z_score?: number;
  expected_value?: number;
  expected_range_min?: number;
  expected_range_max?: number;
  actual_value?: number;
  detection_method: string;
  detection_params: Record<string, any>;
  detected_at: string;
  status: string;
  resolved_at?: string;
  resolved_by?: string;
  resolution_notes?: string;
}
