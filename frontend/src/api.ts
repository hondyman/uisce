import { ViewMeta, QueryState, CompileResult, ExecuteResult, HistoryEntry, SavedQuery, FullSavedQuery, DashboardTile, Workbook, FullWorkbook, SuggestedQuery, PreviewDiff, FullFolder, FolderAnalytics, DuplicateQueryCluster, FolderDiff, SemanticSearchRequest, SemanticSearchResult, SearchFeedbackRequest, Goal, Tour, SemanticViewMeta, SemanticQuery, NLQTranslateRequest, NLQTranslateResponse, LineageGraphData, ImpactAnalysis, QueryTemplateMeta, QueryTemplate, Comment, Approval, ExplorerAlert, DashboardSnapshot, SnapshotDiff, SemanticViewVersion, SemanticDiff, SemanticModelAccessRequest, SemanticModelClaim, SemanticModelRoleClaim, ClaimSimulationRequest, ClaimSimulationResult, AccessControlAuditLog, GovernanceSnapshot as _GovernanceSnapshot, IndexMonitorSnapshot, ClaimLifecycleSnapshot, AccessControlPolicy, PolicySimulationResult, ClaimAwareLineageGraphData, SemanticNotification, NotificationRoutingRule, SemanticChangeEvent, ClaimSuggestion, ClaimBundle, GovernanceHeatmapDataPoint, ClaimConflict, GrantClaimRequest, EvaluateAccessRequest, EvaluateAccessResponse, AccessDecisionTrace, SimulateAccessRequest, GovernanceCockpitSnapshot, AutomationPolicy, AutomationLog, NLQueryRequest, NLQueryResponse, NLQuerySuggestion, ConversationContext, ConversationSummary, RefinementContext } from './types';
import { devLog, devDebug, devWarn, devError } from './utils/devLogger';
export { devLog, devDebug, devWarn, devError };
import apiClient from './utils/apiClient';

export async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
  const result = await apiClient<any>(path, options);

  // If result is a Response object (not already parsed by apiClient), handle it.
  if (result && typeof result === 'object' && 'ok' in result && typeof result.ok === 'boolean') {
    const resp = result as Response;
    if (!resp.ok) {
      let text = '';
      try {
        const json = await resp.json();
        text = (json && typeof json === 'object')
          ? (json.error || json.message || JSON.stringify(json))
          : String(json);
      } catch (e) {
        try { text = await resp.text(); } catch (ee) { text = ''; }
      }
      const statusText = resp.statusText || 'API request failed';
      throw new Error(`${resp.status} ${statusText}${text ? `: ${text}` : ''}`);
    }

    const contentType = resp.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      return resp.json();
    }
    const txt = await resp.text();
    if (!txt) return {} as T;
    try {
      return JSON.parse(txt) as T;
    } catch (e) {
      return (txt as unknown) as T;
    }
  }

  // Otherwise, apiClient already returned the parsed data or success.
  return result as T;
}

export function listViews(): Promise<ViewMeta[]> {
  return fetchAPI('/views');
}

export function compileView(view: string, query: QueryState): Promise<CompileResult> {
  return fetchAPI('/views/compile', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ view, ...query }),
  });
}

export function executeView(view: string, query: QueryState, savedId?: string): Promise<ExecuteResult> {
  return fetchAPI('/views/execute', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ view, ...query, savedId }),
  });
}

export function listHistory(): Promise<HistoryEntry[]> {
  return fetchAPI('/views/history');
}

export type ListSavedQueriesParams = {
  scope?: 'mine' | 'shared' | 'all';
  view?: string;
  tags?: string[];
  search?: string;
};

export function listSavedQueries(params: ListSavedQueriesParams): Promise<SavedQuery[]> {
  const query = new URLSearchParams();
  if (params.scope) query.set('scope', params.scope);
  if (params.view) query.set('view', params.view);
  if (params.search) query.set('search', params.search);
  params.tags?.forEach(tag => query.append('tags', tag));
  return fetchAPI(`/saved-queries?${query.toString()}`);
}

export function saveQuery(data: Omit<FullSavedQuery, 'id'>): Promise<FullSavedQuery> {
  return fetchAPI('/saved-queries', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export function updateQuery(id: string, data: Omit<FullSavedQuery, 'id'>): Promise<void> {
  return fetchAPI(`/saved-queries/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export function getSavedQuery(id: string): Promise<FullSavedQuery> {
  return fetchAPI(`/saved-queries/${id}`);
}

export function cloneQuery(id: string): Promise<FullSavedQuery> {
  return fetchAPI(`/saved-queries/${id}/clone`, { method: 'POST' });
}

export function deleteQuery(id: string): Promise<void> {
  return fetchAPI(`/saved-queries/${id}`, { method: 'DELETE' });
}

export function getPreview(id: string): Promise<ExecuteResult> {
  return fetchAPI(`/saved-queries/${id}/preview`);
}

export function getPreviewDiff(id: string): Promise<PreviewDiff> {
  return fetchAPI(`/saved-queries/${id}/diff`);
}

export function getDuplicateQueries(): Promise<DuplicateQueryCluster[]> {
  return fetchAPI('/saved-queries/duplicates');
}

export function getFolderAnalytics(folderId: string): Promise<FolderAnalytics> {
  return fetchAPI(`/folders/${folderId}/analytics`);
}

export function getFolderDiff(folderId: string, from: string, to: string): Promise<FolderDiff> {
  const params = new URLSearchParams({ from, to });
  return fetchAPI(`/folders/${folderId}/diff?${params.toString()}`);
}

export function getSuggestions(userId: string, datasourceId: string): Promise<SemanticSearchResult[]> {
  const params = new URLSearchParams({ user_id: userId, tenant_instance_id: datasourceId });
  return fetchAPI(`/search/suggestions?${params.toString()}`);
}

export function logSearchFeedback(req: SearchFeedbackRequest): Promise<void> {
  // Fire-and-forget, no need to wait for response or handle errors that block the UI.
  apiClient('/search/feedback', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  }).catch((e) => { devError(e); });
  return Promise.resolve();
}

export function semanticSearch(req: SemanticSearchRequest): Promise<SemanticSearchResult[]> {
  return fetchAPI('/search/semantic', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
}

export function listGoals(userId: string): Promise<Goal[]> {
  const params = new URLSearchParams({ user_id: userId });
  return fetchAPI(`/goals?${params.toString()}`);
}

export function listTours(userId: string): Promise<Tour[]> {
  const params = new URLSearchParams({ user_id: userId });
  return fetchAPI(`/tours?${params.toString()}`);
}

export function getTour(tourId: string): Promise<{ steps: Array<{ id: string; title: string; content: string; target?: string; placement?: string }> }> {
  return fetchAPI(`/tours/${tourId}`);
}

export function listSemanticViews(datasourceId: string): Promise<SemanticViewMeta[]> {
  const params = new URLSearchParams({ tenant_instance_id: datasourceId });
  return fetchAPI(`/semantic-views?${params.toString()}`);
}

// Natural language translation (NLQ) helper - stubbed to match imports used in the UI.
export function translateNLQ(request: NLQTranslateRequest): Promise<NLQTranslateResponse> {
  // In the absence of a backend endpoint in local dev, provide a minimal resolved value.
  return Promise.resolve({ view_name: request.view_name || '', query: { dimensions: [], metrics: [], filters: [], order: [], limit: 0 } as SemanticQuery });
}

export function executeSemanticQuery(viewName: string, query: SemanticQuery): Promise<ExecuteResult> {
  return fetchAPI(`/semantic-views/${viewName}/execute`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(query),
  });
}

export function getLineageGraph(assetId: string): Promise<LineageGraphData> {
  return fetchAPI(`/lineage/${assetId}`);
}

export function getImpactAnalysis(assetId: string): Promise<ImpactAnalysis> {
  return fetchAPI(`/lineage/${assetId}/impact`);
}

export function listQueryTemplates(datasourceId: string): Promise<QueryTemplateMeta[]> {
  const params = new URLSearchParams({ tenant_instance_id: datasourceId });
  return fetchAPI(`/query-templates?${params.toString()}`);
}

export function getQueryTemplate(templateId: string): Promise<QueryTemplate> {
  return fetchAPI(`/query-templates/${templateId}`);
}

export function listComments(assetId: string): Promise<Comment[]> {
  const params = new URLSearchParams({ asset_id: assetId });
  return fetchAPI(`/comments?${params.toString()}`);
}

export function addComment(assetId: string, assetType: string, body: string): Promise<Comment> {
  return fetchAPI('/comments', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ asset_id: assetId, asset_type: assetType, body }),
  });
}

export function getApprovalStatus(assetId: string): Promise<Approval> {
  return fetchAPI(`/approval/${assetId}`);
}

export function listAlerts(userId: string): Promise<ExplorerAlert[]> {
  const params = new URLSearchParams({ user_id: userId });
  return fetchAPI(`/alerts?${params.toString()}`);
}

export function markAlertAsRead(alertId: string): Promise<void> {
  // Use apiClient for a fire-and-forget call
  apiClient(`/alerts/${alertId}/read`, { method: 'POST' });
  return Promise.resolve();
}

export function listDashboardSnapshots(dashboardId: string): Promise<DashboardSnapshot[]> {
  return fetchAPI(`/dashboards/${dashboardId}/snapshots`);
}

export function createDashboardSnapshot(dashboardId: string, name: string): Promise<DashboardSnapshot> {
  return fetchAPI(`/dashboards/${dashboardId}/snapshot`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
}

export function compareSnapshots(snapshotId: string, compareToId: string): Promise<SnapshotDiff> {
  const params = new URLSearchParams({ compare_to: compareToId });
  return fetchAPI(`/snapshots/${snapshotId}/diff?${params.toString()}`);
}

export function listSemanticViewVersions(viewName: string): Promise<SemanticViewVersion[]> {
  return fetchAPI(`/semantic-views/${viewName}/versions`);
}

export function compareSemanticViewVersions(viewName: string, from: number, to: number): Promise<SemanticDiff> {
  const params = new URLSearchParams({ from: from.toString(), to: to.toString() });
  return fetchAPI(`/semantic-views/${viewName}/diff?${params.toString()}`);
}

export function requestAccess(modelId: string, permission: string, reason: string): Promise<SemanticModelAccessRequest> {
  return fetchAPI('/access-request', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ model_id: modelId, permission, reason }),
  });
}

export function listAccessRequests(params: { userId?: string; reviewerId?: string }): Promise<SemanticModelAccessRequest[]> {
  const query = new URLSearchParams();
  if (params.userId) query.set('user_id', params.userId);
  if (params.reviewerId) query.set('reviewer_id', params.reviewerId);
  return fetchAPI(`/access-request?${query.toString()}`);
}

export function approveAccessRequest(requestId: string): Promise<void> {
  return fetchAPI(`/access-request/${requestId}/approve`, { method: 'POST' });
}

export function rejectAccessRequest(requestId: string, notes: string): Promise<void> {
  return fetchAPI(`/access-request/${requestId}/reject`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ notes }),
  });
}

export function simulateClaims(req: ClaimSimulationRequest): Promise<ClaimSimulationResult> {
  return fetchAPI('/claims/simulate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
}

export function listClaimSimulations(): Promise<ClaimSimulationResult[]> {
  return fetchAPI('/claims/simulations');
}

export function getIndexMonitorSnapshot(): Promise<IndexMonitorSnapshot> {
  return fetchAPI('/index/status');
}

export function getClaimLifecycleSnapshot(): Promise<ClaimLifecycleSnapshot> {
  return fetchAPI('/claims/lifecycle-snapshot');
}

export function listAccessPolicies(): Promise<AccessControlPolicy[]> {
  return fetchAPI('/policies');
}

export function simulatePolicyChange(policy: AccessControlPolicy): Promise<PolicySimulationResult> {
  return fetchAPI('/policies/simulate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(policy),
  });
}

export function getClaimAwareLineage(assetId: string, userId: string): Promise<ClaimAwareLineageGraphData> {
  const params = new URLSearchParams({ user_id: userId });
  return fetchAPI(`/lineage/${assetId}/claim-aware?${params.toString()}`);
}

export function listNotifications(): Promise<SemanticNotification[]> {
  return fetchAPI('/notifications');
}

export function markNotificationAsRead(notificationId: string): Promise<void> {
  return fetchAPI(`/notifications/${notificationId}/read`, { method: 'POST' });
}

export function listNotificationRules(): Promise<NotificationRoutingRule[]> {
  return fetchAPI('/notifications/rules');
}

export function updateNotificationRule(rule: NotificationRoutingRule): Promise<NotificationRoutingRule> {
  return fetchAPI('/notifications/rules', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rule),
  });
}

export function previewRecipients(assetId: string): Promise<Record<string, string[]>> {
  const params = new URLSearchParams({ asset_id: assetId });
  return fetchAPI(`/notifications/recipients?${params.toString()}`);
}

// --- Alert Governance API ---

export function evaluateAlert(event: SemanticChangeEvent): Promise<SemanticNotification> {
  return fetchAPI('/alerts/evaluate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(event),
  });
}

export function listSuppressedAlerts(): Promise<SemanticNotification[]> {
  return fetchAPI('/alerts/suppressed');
}

export function listEscalatedAlerts(): Promise<SemanticNotification[]> {
  return fetchAPI('/alerts/escalated');
}

export function overrideAlertStatus(alertId: string, status: string): Promise<void> {
  return fetchAPI(`/alerts/${alertId}/override`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ status }) });
}

// --- Advanced Governance API ---

export function listClaimSuggestions(): Promise<ClaimSuggestion[]> {
  return fetchAPI('/claims/suggestions');
}

export function listClaimBundles(): Promise<ClaimBundle[]> {
  return fetchAPI('/claim-bundles');
}

export function detectClaimDrift(): Promise<SemanticModelClaim[]> {
  return fetchAPI('/claims/drift');
}

export function listClaimConflicts(userId: string): Promise<ClaimConflict[]> {
  const params = new URLSearchParams({ user_id: userId });
  return fetchAPI(`/claims/conflicts?${params.toString()}`);
}

export function resolveClaimConflict(conflictId: string, action: string): Promise<void> {
  return fetchAPI(`/claims/conflicts/${conflictId}/resolve`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action }) });
}

export function getGovernanceHeatmap(): Promise<GovernanceHeatmapDataPoint[]> {
  return fetchAPI('/governance/heatmap');
}

// --- Unified Access Intelligence API ---

export function getEffectiveClaims(userId: string, tenantId: string): Promise<SemanticModelClaim[]> {
  const params = new URLSearchParams({ user_id: userId, tenant_id: tenantId });
  return fetchAPI(`/intelligence/claims/effective?${params.toString()}`);
}

export function grantClaim(req: GrantClaimRequest): Promise<SemanticModelClaim> {
  return fetchAPI('/intelligence/claims/grant', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
}

export function assignBundle(userId: string, bundleId: string): Promise<void> {
  return fetchAPI('/intelligence/bundles/assign', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, bundle_id: bundleId }),
  });
}

export function evaluateAccess(req: EvaluateAccessRequest): Promise<EvaluateAccessResponse> {
  return fetchAPI('/intelligence/evaluate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
}

export function refreshClaimsCache(userId: string, tenantId: string): Promise<void> {
  return fetchAPI('/intelligence/claims/refresh', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, tenant_id: tenantId }),
  });
}

export function getDecisionTrace(decisionId: string): Promise<AccessDecisionTrace> {
  return fetchAPI(`/intelligence/decisions/${decisionId}/trace`);
}

export function getDecisionExplanation(decisionId: string): Promise<{ explanation: string }> {
  return fetchAPI(`/intelligence/decisions/${decisionId}/explain`);
}

export function simulateAccess(req: SimulateAccessRequest): Promise<EvaluateAccessResponse> {
  return fetchAPI('/intelligence/simulate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
}

export function getGovernanceCockpitSnapshot(tenantId: string): Promise<GovernanceCockpitSnapshot> {
  const params = new URLSearchParams({ tenant_id: tenantId });
  return fetchAPI(`/intelligence/governance/cockpit?${params.toString()}`);
}

// --- Governance Automation API ---

export function runAutomationCycle(): Promise<{ status: string; logs: AutomationLog[] }> {
  return fetchAPI('/automation/run', { method: 'POST' });
}

export function listAutomationLogs(): Promise<AutomationLog[]> {
  return fetchAPI('/automation/logs');
}

export function listAutomationPolicies(): Promise<AutomationPolicy[]> {
  return fetchAPI('/automation/policies');
}

export function pauseAutomation(): Promise<{ status: string }> {
  return fetchAPI('/automation/pause', { method: 'POST' });
}

export function resumeAutomation(): Promise<{ status: string }> {
  return fetchAPI('/automation/resume', { method: 'POST' });
}

export function getStewardDomains(userId: string): Promise<string[]> {
  const params = new URLSearchParams({ user_id: userId });
  return fetchAPI(`/steward/domains?${params.toString()}`);
}

export function listAllRoles(): Promise<string[]> {
  return fetchAPI('/roles');
}

export function listRoleClaims(): Promise<SemanticModelRoleClaim[]> {
  return fetchAPI('/role-claims');
}

export function updateRoleClaim(role: string, modelId: string, permissions: string[]): Promise<SemanticModelRoleClaim> {
  return fetchAPI('/role-claims', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ role, model_id: modelId, permissions }),
  });
}

export function revokeDirectClaim(claimId: string): Promise<void> {
  return fetchAPI(`/claims/direct/${claimId}`, { method: 'DELETE' });
}

export function requestClaimRenewal(claimId: string, reason: string): Promise<void> {
  return fetchAPI(`/claims/direct/${claimId}/renew`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ reason }),
  });
}

export function listAccessAuditLogs(): Promise<AccessControlAuditLog[]> {
  return fetchAPI('/audit-logs/access');
}

export function logAccessDeniedAttempt(assetType: string, assetId: string, reason: string): Promise<void> {
  // Fire-and-forget is fine for this.
  fetchAPI('/audit/access-denied', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ asset_type: assetType, asset_id: assetId, reason }),
  }).catch(err => import('./utils/devLogger').then(({ devError }) => devError('Failed to log access denied attempt:', err)).catch(() => { }));
  return Promise.resolve();
}

export function listFolders(): Promise<FullFolder[]> {
  return fetchAPI('/folders');
}

export function addItemToFolder(folderId: string, itemId: string, itemType: 'query' | 'workbook'): Promise<void> {
  return fetchAPI(`/folders/${folderId}/items`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ item_id: itemId, item_type: itemType }),
  });
}

export function listWorkbooks(params: { scope: 'mine' | 'shared' | 'all' }): Promise<Workbook[]> {
  return fetchAPI(`/workbooks?scope=${params.scope}`);
}

export function getWorkbook(id: string): Promise<FullWorkbook> {
  return fetchAPI(`/workbooks/${id}`);
}

export function createWorkbook(data: Omit<FullWorkbook, 'id' | 'owner_user_id' | 'tags'> & { tags?: string[] }): Promise<FullWorkbook> {
  return fetchAPI('/workbooks', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export function getSuggestedQueries(viewName: string): Promise<SuggestedQuery[]> {
  return fetchAPI(`/views/${viewName}/suggestions`);
}

// ---- View Change Management API ----

export type ViewChangePlan = {
  view?: string;
  action?: 'create' | 'update' | 'delete' | 'noop' | string;
  details?: string;
  sql?: string;
  // Allow flexible shape to match backend without tight coupling
  [k: string]: unknown;
};

export function compareViews(tenantId?: string, datasourceId?: string): Promise<{ message: string; change_plans: ViewChangePlan[] }> {
  const qs = new URLSearchParams();
  if (tenantId) qs.set('tenant_id', tenantId);
  if (datasourceId) qs.set('tenant_instance_id', datasourceId);
  const path = qs.toString() ? `/views/compare?${qs.toString()}` : '/views/compare';
  return fetchAPI(path);
}

export function applyViewChanges(plans: ViewChangePlan[], tenantId?: string, datasourceId?: string): Promise<{ status: string; message: string }> {
  const qs = new URLSearchParams();
  if (tenantId) qs.set('tenant_id', tenantId);
  if (datasourceId) qs.set('tenant_instance_id', datasourceId);
  const path = qs.toString() ? `/views/apply?${qs.toString()}` : '/views/apply';
  return fetchAPI(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ views: plans }),
  });
}

export function rejectViewChanges(plans: ViewChangePlan[], reason: string, tenantId?: string, datasourceId?: string): Promise<{ status: string; message: string }> {
  const qs = new URLSearchParams();
  if (tenantId) qs.set('tenant_id', tenantId);
  if (datasourceId) qs.set('tenant_instance_id', datasourceId);
  const path = qs.toString() ? `/views/reject?${qs.toString()}` : '/views/reject';
  return fetchAPI(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ views: plans, reason }),
  });
}

// ---- Natural Language Query API Functions ----

export function compileNLQuery(request: NLQueryRequest): Promise<NLQueryResponse> {
  return fetchAPI('/nlquery/compile', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });
}

export function simulateNLQuery(request: NLQueryRequest): Promise<NLQueryResponse> {
  return fetchAPI('/nlquery/simulate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });
}

export function getNLQueryHistory(): Promise<NLQueryResponse[]> {
  return fetchAPI('/nlquery/history');
}

export function getNLQuerySuggestions(): Promise<NLQuerySuggestion[]> {
  return fetchAPI('/nlquery/suggestions');
}

// ---- Conversation API Functions ----

export function startConversation(userId: string, tenantId: string, datasource: string): Promise<ConversationContext> {
  return fetchAPI('/conversation/start', {
    method: 'POST',
    body: JSON.stringify({
      user_id: userId,
      tenant_id: tenantId,
      datasource: datasource,
    }),
  });
}

export function sendConversationMessage(conversationId: string, message: string): Promise<RefinementContext> {
  return fetchAPI(`/conversation/${conversationId}/message`, {
    method: 'POST',
    body: JSON.stringify({
      message: message,
    }),
  });
}

export function getConversationState(conversationId: string): Promise<RefinementContext> {
  return fetchAPI(`/conversation/${conversationId}/state`);
}

export function commitConversationQuery(conversationId: string): Promise<NLQueryResponse> {
  return fetchAPI(`/conversation/${conversationId}/commit`, {
    method: 'POST',
  });
}

export function getConversationSummary(conversationId: string): Promise<ConversationSummary> {
  return fetchAPI(`/conversation/${conversationId}/summary`);
}

// Placeholder for saving a dashboard tile
export function saveToDashboard(tile: Omit<DashboardTile, 'id'>): Promise<{ id: string }> {
  devLog('Saving to dashboard:', tile);
  return Promise.resolve({ id: crypto.randomUUID() });
}