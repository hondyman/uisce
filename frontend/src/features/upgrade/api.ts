import axios from '@/utils/axiosClient';
import type { JSONValue } from '../../types/json';
import type {
  DiffReport,
  AliasMap,
  ExtensionFix,
  GoldenQuery,
  GoldenQueryResult,
  UpgradeStatusResponse,
  UpgradeOverviewResponse,
  MultiUpgradeOverviewResponse
} from '../../types/upgrade';
// DiffResponse not used

export interface VersionInfo {
  version: string;
  schema_hash: string;
  status: 'active' | 'previous' | 'available' | 'preview' | 'canary';
  warnings?: string[];
  created_at: string;
  activated_at?: string;
}

export interface CanaryState { version: string; tenants: string[]; until: string }
export interface SLOSummary { error_rate: number; p95_latency_ms: number; shadow_diff_rate: number; preagg_rebuild_ms: number; cache_hit_ratio: number; merge_duration_ms: number; validate_duration_ms: number; window: number; updated_at: string }

export async function fetchVersions() {
  const { data } = await axios.get('/api/upgrade/versions');
  return data as { versions: VersionInfo[]; canary?: CanaryState; slo: SLOSummary };
}

export async function setPreview(version: string) {
  await axios.post('/api/upgrade/preview', { version });
}

export async function startCanary(coreVersion: string, tenants: string[]) {
  await axios.post('/api/upgrade/canary', { coreVersion, tenants });
}

export async function activateVersion(coreVersion: string) {
  await axios.post('/api/upgrade/activate', { coreVersion });
}

export async function rollbackVersion(coreVersion: string) {
  await axios.post('/api/upgrade/rollback', { coreVersion });
}

export async function fetchDiff(from: string, to: string) { const { data } = await axios.get('/api/upgrade/diff', { params: { from, to } }); return data; }
export async function listBrokenRefs(version: string) { const { data } = await axios.get(`/api/upgrade/versions/${version}/broken-refs`); return data as Array<{ path: string; reason: string; suggestions: string[] }>; }
export async function applyFixes(version: string, patches: Record<string, string>) { await axios.post(`/api/upgrade/versions/${version}/apply-fixes`, patches); }
export async function runPreview(from: string, to: string, queries: string[]) { const { data } = await axios.post('/api/upgrade/preview/run', { from, to, queries }); return data as Array<{ query: string; old_rows: number; new_rows: number; diff_pct: number; totals: { old: number; new: number } }>; }
export async function listNotifications() { const { data } = await axios.get('/api/upgrade/notifications'); return data as Array<{ id: string; type: string; message: string; severity: string; created_at: string }>; }

export async function getSchemaVersion() {
  const { data } = await axios.get('/api/upgrade/schema-version');
  return data as { schema_version: string };
}

export async function getUpgradeStatus() {
  const { data } = await axios.get('/api/upgrade/status');
  return data as UpgradeStatusResponse;
}

export async function getUpgradeOverview(coreVersion: string) {
  const { data } = await axios.get('/api/upgrade/overview', { params: { coreVersion } });
  return data as UpgradeOverviewResponse;
}

export async function getMultiUpgradeOverview(
  coreVersions?: string[],
  statuses?: string[],
  sort?: string
): Promise<MultiUpgradeOverviewResponse> {
  const params = new URLSearchParams();
  if (coreVersions?.length) params.set("coreVersions", coreVersions.join(","));
  if (statuses?.length) params.set("status", statuses.join(","));
  if (sort) params.set("sort", sort);

  const { data } = await axios.get('/api/upgrade/overview/multi', { params });
  return data as MultiUpgradeOverviewResponse;
}

// New lifecycle API functions
export async function prepareUpgrade(newVersion: string) {
  const { data } = await axios.post('/api/upgrade/prepare', { new_version: newVersion });
  return data as { schema_hash: string; changes: SchemaChange[]; deprecation_map: DeprecationMap };
}

export async function generateCore(version: string) {
  await axios.post(`/api/upgrade/versions/${version}/generate-core`);
}

export async function mergeCustom(version: string) {
  await axios.post(`/api/upgrade/versions/${version}/merge-custom`);
}

export async function validateVersion(version: string) {
  const { data } = await axios.post(`/api/upgrade/versions/${version}/validate`);
  return data as ValidationReport;
}

export async function runShadow(version: string, queries: string[]) {
  const { data } = await axios.post(`/api/upgrade/versions/${version}/shadow`, { queries });
  return data as ShadowRunResult[];
}

export async function getValidationReport(version: string) {
  const { data } = await axios.get(`/api/upgrade/versions/${version}/validation-report`);
  return data as ValidationReport;
}

export async function archiveVersion(version: string) {
  await axios.post(`/api/upgrade/versions/${version}/archive`);
}

export async function getSchemaChanges(version: string) {
  const { data } = await axios.get(`/api/upgrade/versions/${version}/schema-changes`);
  return data as SchemaChange[];
}

export async function getDeprecationMap(version: string) {
  const { data } = await axios.get(`/api/upgrade/versions/${version}/deprecation-map`);
  return data as DeprecationMap;
}

export async function getPreAggRebuild(version: string) {
  const { data } = await axios.get(`/api/upgrade/versions/${version}/preagg-rebuild`);
  return data as PreAggRebuild;
}

// Batch job orchestration
export async function scheduleBatchJob(jobType: string, config: JSONValue) {
  const { data } = await axios.post('/api/upgrade/jobs/schedule', { job_type: jobType, config });
  return data as { job_id: string; status: string };
}

export async function listBatchJobs() {
  const { data } = await axios.get('/api/upgrade/jobs');
  return data as Array<{ job_id: string; job_type: string; status: string; created_at: string; completed_at?: string; result?: JSONValue }>;
}

export async function cancelBatchJob(jobId: string) {
  await axios.post(`/api/upgrade/jobs/${jobId}/cancel`);
}

// Artifact management
export async function listArtifacts(version?: string) {
  const params = version ? { version } : {};
  const { data } = await axios.get('/api/upgrade/artifacts', { params });
  return data as Array<{ artifact_id: string; version: string; type: string; size_bytes: number; created_at: string; checksum: string }>;
}

export async function downloadArtifact(artifactId: string) {
  const response = await axios.get(`/api/upgrade/artifacts/${artifactId}/download`, { responseType: 'blob' });
  return response.data;
}

export async function deleteArtifact(artifactId: string) {
  await axios.delete(`/api/upgrade/artifacts/${artifactId}`);
}

// SLO monitoring and alerting
export async function getSLOMetrics(timeRange: string = '1h') {
  const { data } = await axios.get('/api/upgrade/slo/metrics', { params: { range: timeRange } });
  return data as { metrics: SLOMetric[]; alerts: SLOAlert[] };
}

export async function configureSLOAlert(config: SLOAlertConfig) {
  const { data } = await axios.post('/api/upgrade/slo/alerts', config);
  return data;
}

export async function listSLOAlerts() {
  const { data } = await axios.get('/api/upgrade/slo/alerts');
  return data as SLOAlertConfig[];
}

// Diff Report and Alias Map APIs
export async function getDiffReport(fromVersion: string, toVersion: string): Promise<DiffReport> {
  const { data } = await axios.get('/api/upgrade/diff', { params: { from: fromVersion, to: toVersion } });
  return data;
}

export async function getAliasMap(fromVersion: string, toVersion: string): Promise<AliasMap> {
  const { data } = await axios.get('/api/upgrade/alias-map', { params: { from: fromVersion, to: toVersion } });
  return data;
}

export async function generateDiffReport(fromVersion: string, toVersion: string): Promise<DiffReport> {
  const { data } = await axios.post('/api/upgrade/diff/generate', { from_version: fromVersion, to_version: toVersion });
  return data;
}

export async function generateAliasMap(fromVersion: string, toVersion: string): Promise<AliasMap> {
  const { data } = await axios.post('/api/upgrade/alias-map/generate', { from_version: fromVersion, to_version: toVersion });
  return data;
}

// Extension Fix APIs
export async function analyzeExtensionFixes(version: string, extensionFiles: string[]): Promise<ExtensionFix[]> {
  const { data } = await axios.post('/api/upgrade/extensions/analyze', { version, files: extensionFiles });
  return data;
}

export async function applyExtensionFixes(version: string, fixes: ExtensionFix[]): Promise<{ applied: number; failed: number; errors: string[] }> {
  const { data } = await axios.post('/api/upgrade/extensions/apply-fixes', { version, fixes });
  return data;
}

export async function previewExtensionFixes(version: string, fixes: ExtensionFix[]): Promise<{ preview: ExtensionFix[]; conflicts: string[] }> {
  const { data } = await axios.post('/api/upgrade/extensions/preview-fixes', { version, fixes });
  return data;
}

// Golden Query APIs
export async function listGoldenQueries(): Promise<GoldenQuery[]> {
  const { data } = await axios.get('/api/upgrade/golden-queries');
  return data;
}

export async function runGoldenQueries(fromVersion: string, toVersion: string, queryNames?: string[]): Promise<GoldenQueryResult[]> {
  const { data } = await axios.post('/api/upgrade/golden-queries/run', {
    from_version: fromVersion,
    to_version: toVersion,
    queries: queryNames
  });
  return data;
}

export async function addGoldenQuery(query: Omit<GoldenQuery, 'name'> & { name?: string }): Promise<GoldenQuery> {
  const { data } = await axios.post('/api/upgrade/golden-queries', query);
  return data;
}

export async function updateGoldenQuery(name: string, query: Partial<GoldenQuery>): Promise<GoldenQuery> {
  const { data } = await axios.put(`/api/upgrade/golden-queries/${name}`, query);
  return data;
}

export async function deleteGoldenQuery(name: string): Promise<void> {
  await axios.delete(`/api/upgrade/golden-queries/${name}`);
}

// Type definitions for new APIs

// Type definitions for new APIs
export interface SchemaChange {
  table: string;
  column?: string;
  change_type: 'added' | 'removed' | 'modified';
  old_value?: JSONValue;
  new_value?: JSONValue;
  impact: 'breaking' | 'non_breaking';
}

export interface DeprecationMap {
  deprecated_tables: string[];
  deprecated_columns: Record<string, string[]>;
  migration_paths: Record<string, string>;
}

export interface ValidationReport {
  version: string;
  passed: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
  score: number;
  validated_at: string;
}

export interface ValidationError {
  type: 'fk_violation' | 'governance_violation' | 'schema_mismatch';
  message: string;
  location: string;
  severity: 'critical' | 'high' | 'medium';
}

export interface ValidationWarning {
  type: 'performance' | 'compatibility' | 'deprecation';
  message: string;
  suggestion: string;
}

export interface ShadowRunResult {
  query: string;
  old_result_count: number;
  new_result_count: number;
  diff_percentage: number;
  execution_time_ms: number;
  errors?: string[];
}

export interface PreAggRebuild {
  version: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;
  estimated_completion: string;
  rebuild_stats: {
    total_preaggs: number;
    completed_preaggs: number;
    failed_preaggs: number;
  };
}

export interface SLOMetric {
  timestamp: string;
  error_rate: number;
  p95_latency_ms: number;
  shadow_diff_rate: number;
  cache_hit_ratio: number;
  query_throughput: number;
}

export interface SLOAlert {
  alert_id: string;
  metric: string;
  threshold: number;
  current_value: number;
  severity: 'info' | 'warning' | 'error' | 'critical';
  message: string;
  triggered_at: string;
}

export interface SLOAlertConfig {
  metric: string;
  threshold: number;
  operator: 'gt' | 'lt' | 'gte' | 'lte';
  severity: 'info' | 'warning' | 'error' | 'critical';
  enabled: boolean;
  cooldown_minutes: number;
}

// Diff Report Schema Types - Now imported from generated types
// Alias Map Schema Types - Now imported from generated types


