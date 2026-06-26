import { useState, useCallback, useEffect } from 'react';
import { getViewIdentifier } from './types/views';
import { devError, devWarn } from './utils/devLogger';
import { FullSavedQuery, FullWorkbook, SemanticViewMeta, SemanticQuery, QueryTemplateMeta, QueryState, PageInfo, ViewMeta, SemanticModelClaim } from './types';
import { executeSemanticQuery, listSemanticViews, getQueryTemplate, getEffectiveClaims, logAccessDeniedAttempt, evaluateAccess } from './api';
import type { TabState } from './TabsManager';
import ResultsGrid from './ResultsGrid';
import SQLTab from './SQLTab';
import GraphQLTab from './GraphQLTab';
import ExplainTab from './ExplainTab';
import HistoryPanel from './HistoryPanel';
import VisualizationPanel from './VisualizationPanel';
import InsightsPanel from './InsightsPanel';
import PreviewDiffTab from './PreviewDiffTab';
import SemanticViewPicker from './SemanticViewPicker';
import SemanticQueryComposer from './SemanticQueryComposer';
import QueryTemplateBrowser from './QueryTemplateBrowser';
import ClaimSimulationPanel from './ClaimSimulationPanel';
import RequestAccessModal from './RequestAccessModal';
import ReviewerInbox from './ReviewerInbox';
import SemanticVersionPanel from './SemanticVersionPanel';
import SemanticDiffViewer from './SemanticDiffViewer';
import SnapshotPanel from './SnapshotPanel';
import SnapshotDiffViewer from './SnapshotDiffViewer';
import CommentsPanel from './CommentsPanel';
import AccessDeniedExplanation from './AccessDeniedExplanation';
import SemanticQueryInput from './SemanticQueryInput';

/* eslint-disable no-unused-vars */
interface ExplorerTabProps {
  tab: TabState;
  views: ViewMeta[];
  onChange: (_patch: Partial<TabState>) => void;
  onOpenSavedQuery: (_q: FullSavedQuery) => void;
  onOpenWorkbook: (_w: FullWorkbook) => void;
  onStartTour: (_tourId: string) => void;
}
/* eslint-enable no-unused-vars */

export default function ExplorerTab({ tab, views, onChange, onOpenSavedQuery, onOpenWorkbook, onStartTour }: ExplorerTabProps) {
  // Use views prop to avoid unnecessary fetches and to satisfy the linter when
  // the prop is provided by a parent component. Mark other optional callback
  // props as used to suppress unused-variable diagnostics until they're wired up.
  void onOpenSavedQuery;
  void onOpenWorkbook;
  void onStartTour;
  void views; // mark views prop as intentionally unused in favor of visibleViews
  const [activeResultTab, setActiveResultTab] = useState('grid');
  // Accept either a frontend ViewMeta or a richer SemanticViewMeta depending on
  // where the data originated. This keeps compatibility with the parent
  // `views: ViewMeta[]` while allowing the component to accept SemanticViewMeta
  // objects from API calls.
  const [selectedSemanticView, setSelectedSemanticView] = useState<SemanticViewMeta | ViewMeta | null>(null);
  const [currentSavedQuery, setCurrentSavedQuery] = useState<FullSavedQuery | null>(null);
  const [currentWorkbook, setCurrentWorkbook] = useState<FullWorkbook | null>(null);
  const [diffTarget, setDiffTarget] = useState<{ snapshotId: string; compareToId: string } | null>(null);
  const [showSimulationPanel, setShowSimulationPanel] = useState(false);
  const [requestAccessModelId, setRequestAccessModelId] = useState<string | null>(null);
  const [semanticDiffTarget, setSemanticDiffTarget] = useState<{ viewName: string; from: number; to: number } | null>(null);
  const [deniedDecision, setDeniedDecision] = useState<{ decisionId: string; reason: string } | null>(null);
  // NEW: State for claims and visible views
  const [claims, setClaims] = useState<SemanticModelClaim[]>([]);
  const [visibleViews, setVisibleViews] = useState<SemanticViewMeta[]>([]);

  // Helpers to safely extract id/name/title from view objects without unsafe casts
  const getViewIdentifierSafe = (v: SemanticViewMeta | ViewMeta | null | undefined): string | undefined => {
    if (!v) return undefined;
    const r = v as unknown as Record<string, unknown>;
    const idVal = r['id'];
    if (typeof idVal === 'string') return idVal;
    const nameVal = r['name'];
    if (typeof nameVal === 'string') return nameVal;
    return undefined;
  };

  const getViewDisplayTitle = (v: SemanticViewMeta | ViewMeta | null | undefined): string => {
    if (!v) return '';
    const r = v as unknown as Record<string, unknown>;
    const title = r['title'];
    if (typeof title === 'string' && title.length) return title;
    const name = r['name'];
    if (typeof name === 'string' && name.length) return name;
    const id = getViewIdentifierSafe(v);
    return id ? String(id).slice(0, 8) : '';
  };

  const tenantId = "acme_corp"; // Mock tenant ID
  const datasourceId = "mock-datasource-id";
  const currentUser = "patrick"; // Using a user with mock claims

  // NEW: Fetch claims and views on load
  useEffect(() => {
    const fetchData = async () => {
      try {
        const [fetchedViews, userClaims] = await Promise.all([
          listSemanticViews(datasourceId),
          getEffectiveClaims(currentUser, tenantId),
        ]);

        const filteredViews = fetchedViews.filter(view =>
          userClaims.some(c => c.model_id === view.id && c.permission === 'read')
        );

        setClaims(userClaims);
        setVisibleViews(filteredViews);
        } catch (error) {
        devError("Failed to fetch views or claims:", error);
        // Handle error state in UI if necessary
      }
    };
    fetchData();
  }, [datasourceId, currentUser, tenantId]);

  const handleExecuteSemanticQuery = useCallback(async (query: SemanticQuery, view: SemanticViewMeta | ViewMeta) => {
    if (!selectedSemanticView) return;
    setDeniedDecision(null); // Clear previous denial

    // Pre-flight check using the real-time evaluation engine
    const assetId = getViewIdentifierSafe(view) || '';
    const accessCheck = await evaluateAccess({
      user_id: currentUser,
      tenant_id: tenantId,
      asset_id: assetId,
      action: 'query',
    });

    if (accessCheck.decision === 'deny') {
      setDeniedDecision({ decisionId: accessCheck.decision_id, reason: accessCheck.reason });
      logAccessDeniedAttempt('semantic_model', assetId, accessCheck.reason);
      return;
    }

  const viewIdentifier = getViewIdentifier(view) || getViewIdentifierSafe(view) || '';
  const res = await executeSemanticQuery(viewIdentifier, query);
    onChange({
      result: { rows: res.rows, columns: res.columns, page: res.page },
      compile: { sql: res.sql, graphql: res.graphql, explain: res.explain },
      explain: res.explain,
      viz: { type: 'auto' }
    });
    setActiveResultTab('grid');
  }, [onChange, selectedSemanticView, currentUser, tenantId, setDeniedDecision]);

  const handleNLQ = async (viewName: string, query: SemanticQuery) => {
    // When a natural language query is translated, we need to ensure the
    // corresponding semantic view is selected.
    // MODIFIED: Use visibleViews instead of fetching all views
  const view = visibleViews.find(v => getViewIdentifierSafe(v) === viewName);
    if (view) {
  setSelectedSemanticView(view as SemanticViewMeta | ViewMeta);
      const queryState: QueryState = {
        measures: query.metrics,
        dimensions: query.dimensions,
        filters: query.filters || [],
        order: query.order || [],
        limit: query.limit,
        offset: 0,
      };
      onChange({ query: queryState });
      await handleExecuteSemanticQuery(query, view);
    }
  };

  const handleSelectTemplate = async (templateMeta: QueryTemplateMeta) => {
    // 1. Fetch full template
    const fullTemplate = await getQueryTemplate(templateMeta.id);

    // 2. Find and select the semantic view
    // MODIFIED: Use visibleViews instead of fetching all views
  const view = visibleViews.find(v => getViewIdentifierSafe(v) === fullTemplate.semantic_view);
    if (!view) {
      devWarn(`Template view "${fullTemplate.semantic_view}" not found.`);
      return;
    }
  setSelectedSemanticView(view as SemanticViewMeta | ViewMeta);

    // 3. Construct the new query state from the template
    const newQuery: SemanticQuery = {
      dimensions: fullTemplate.default_dimensions || [],
      metrics: fullTemplate.default_metrics || [],
      // Ensure required_filters is an array of filter objects when present
      filters: Array.isArray(fullTemplate.required_filters)
        ? (fullTemplate.required_filters as Array<{ field: string; op: string; values: string[] }>)
        : [],
      order: [], // Templates could define a default order in the future
      limit: 100,
    };

    // 4. Update the tab state and execute the query
  const queryState: QueryState = {
    measures: newQuery.metrics,
    dimensions: newQuery.dimensions,
    filters: newQuery.filters || [],
    order: newQuery.order || [],
    limit: newQuery.limit,
    offset: 0,
  };
  onChange({ query: queryState });
  await handleExecuteSemanticQuery(newQuery, view as SemanticViewMeta | ViewMeta);
  };


  const handleLoadWorkbook = async (workbookId: string) => {
    // In a real app, you'd fetch the workbook. Here we use a mock.
    const mockWorkbook: FullWorkbook = {
      id: workbookId,
      name: 'Q3 Sales Dashboard',
      owner_user_id: 'ceo',
      tabs: [],
      description: '',
      tags: [],
    };
    setCurrentWorkbook(mockWorkbook);
    setCurrentSavedQuery(null);
    setSelectedSemanticView(null);
  };

  const handleCompareSnapshot = (snapshotId: string) => {
    const currentSnapshotId = "current-state-id-mock"; // In a real app, you'd generate or fetch this
    setDiffTarget({ snapshotId, compareToId: currentSnapshotId });
  };

  // NEW: Handler to open the access request modal and log the denied attempt
  const handleRequestAccess = (assetType: 'dimensions' | 'metrics') => {
    if (selectedSemanticView) {
      const modelId = getViewIdentifierSafe(selectedSemanticView) || '';
      const reason = `User '${currentUser}' attempted to access '${assetType}' on model '${modelId}' without permission.`;
      logAccessDeniedAttempt('semantic_model_scope', modelId, reason);
      setRequestAccessModelId(modelId);
    }
  };

  // NEW: Helper to get claims for the currently selected view
  const getClaimsForSelectedView = () => {
    if (!selectedSemanticView) return [];
    const id = getViewIdentifierSafe(selectedSemanticView) || '';
    return claims.filter(c => c.model_id === id);
  };

  return (
    <div className="explorer-tab">
    <aside className="explorer-sidebar">
      <SemanticViewPicker datasourceId={datasourceId} onSelect={setSelectedSemanticView} views={visibleViews} />
        <HistoryPanel onLoadQuery={() => { /* TODO: Adapt for semantic queries */ }} />
        {selectedSemanticView && (
          <SemanticVersionPanel viewName={getViewIdentifierSafe(selectedSemanticView) || ''} onCompare={(from, to) => setSemanticDiffTarget({ viewName: getViewIdentifierSafe(selectedSemanticView) || '', from, to })} />
        )}
        <div className="admin-panel">
          <h4>Governance Tools</h4>
          <button onClick={() => setShowSimulationPanel(true)}>Simulate Claims</button>
        </div>
        <ReviewerInbox reviewerId="current_reviewer" />
      </aside>
      <main className="explorer-main">
        {selectedSemanticView ? (
          <div className="semantic-workspace">
            {deniedDecision && (
              <AccessDeniedExplanation
                reason={deniedDecision.reason}
                decisionId={deniedDecision.decisionId}
                onClose={() => setDeniedDecision(null)}
                onRequestAccess={() => {
                  if (selectedSemanticView) {
                    setRequestAccessModelId(getViewIdentifierSafe(selectedSemanticView) || '');
                  }
                }}
              />
            )}
            <div className="query-input-area">
              <SemanticQueryInput onQuery={handleNLQ} currentDatasource={datasourceId} currentUser={currentUser} />
            </div>
                <SemanticQueryComposer
              view={selectedSemanticView}
              claims={getClaimsForSelectedView()}
              onExecute={(q) => selectedSemanticView && handleExecuteSemanticQuery(q, selectedSemanticView)}
              onRequestAccess={handleRequestAccess}
            />
            <div className="results-area">
              <div className="results-main-panel">
                <div className="results-tabs">
                  <button onClick={() => setActiveResultTab('grid')} className={activeResultTab === 'grid' ? 'active' : ''}>Grid</button>
                  <button onClick={() => setActiveResultTab('viz')} className={activeResultTab === 'viz' ? 'active' : ''} disabled={!tab.result}>Visualization</button>
                  <button onClick={() => setActiveResultTab('sql')} className={activeResultTab === 'sql' ? 'active' : ''} disabled={!tab.compile}>SQL</button>
                  <button onClick={() => setActiveResultTab('graphql')} className={activeResultTab === 'graphql' ? 'active' : ''} disabled={!tab.compile}>GraphQL</button>
                  <button onClick={() => setActiveResultTab('explain')} className={activeResultTab === 'explain' ? 'active' : ''} disabled={!tab.explain}>Explain</button>
                  <button onClick={() => setActiveResultTab('diff')} className={activeResultTab === 'diff' ? 'active' : ''} disabled={!tab.savedId}>Preview Diff</button>
                </div>
                <div className="results-content">
                  {activeResultTab === 'grid' && <ResultsGrid rows={tab.result?.rows || []} columns={tab.result?.columns || []} page={tab.result?.page as PageInfo} onPageChange={() => { /* TODO */ }} />}
                  {activeResultTab === 'viz' && tab.result && <VisualizationPanel rows={tab.result.rows} columns={tab.result.columns} viz={tab.viz || { type: 'auto' }} onCrossFilter={f => onChange({ query: { ...tab.query, filters: [ ...(tab.query.filters || []), f ], offset: 0 } })} />}
                  {activeResultTab === 'sql' && <SQLTab sql={tab.compile?.sql} />}
                  {activeResultTab === 'graphql' && <GraphQLTab graphql={tab.compile?.graphql} />}
                  {activeResultTab === 'explain' && <ExplainTab explain={tab.compile?.explain} />}
                  {activeResultTab === 'diff' && tab.savedId && <PreviewDiffTab savedId={tab.savedId} />}
                </div>
              </div>
              <aside className="results-sidebar">
                <InsightsPanel result={tab.result} />
                {currentSavedQuery && (
                  <CommentsPanel assetId={currentSavedQuery.id} assetType="query" />
                )}
                {currentWorkbook && (
                  <SnapshotPanel dashboardId={currentWorkbook.id} onCompare={handleCompareSnapshot} />
                )}
                {selectedSemanticView && (
                  <button onClick={() => setRequestAccessModelId(getViewIdentifierSafe(selectedSemanticView) || '')}>
                    Request Access to {getViewDisplayTitle(selectedSemanticView)}
                  </button>)
                }
              </aside>
            </div>
          </div>
        ) : (
          <div className="explorer-placeholder">
            <h2>Welcome to the Explorer</h2>
            <div className="query-input-area placeholder-input">
              <SemanticQueryInput onQuery={handleNLQ} currentDatasource={datasourceId} currentUser={currentUser} />
            </div>
            <p>Select a Semantic View, load a dashboard, or start from a template below.</p>
            <button onClick={() => handleLoadWorkbook('wb-123')}>Load Demo Dashboard</button>
            <QueryTemplateBrowser datasourceId={datasourceId} onSelect={handleSelectTemplate} />
          </div>
        )}
        {showSimulationPanel && (
          <div className="modal-overlay" onClick={() => setShowSimulationPanel(false)}>
            <div className="modal-content wide" onClick={e => e.stopPropagation()}>
              <ClaimSimulationPanel availableModels={[{id: 'd1b6a5e0-9a9a-4b1a-8b0a-1b1b1b1b1b1b', name: 'orders_view'}]} />
              <button onClick={() => setShowSimulationPanel(false)}>Close</button>
            </div>
          </div>
        )}
        {requestAccessModelId && (
          <RequestAccessModal modelId={requestAccessModelId} onClose={() => setRequestAccessModelId(null)} />
        )}
        {diffTarget && (
          <div className="modal-overlay" onClick={() => setDiffTarget(null)}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
              <SnapshotDiffViewer snapshotId={diffTarget.snapshotId} compareToId={diffTarget.compareToId} />
              <button onClick={() => setDiffTarget(null)}>Close</button>
            </div>
          </div>
        )}
        {semanticDiffTarget && (
          <div className="modal-overlay" onClick={() => setSemanticDiffTarget(null)}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
              <SemanticDiffViewer
                viewName={semanticDiffTarget.viewName}
                fromVersion={semanticDiffTarget.from}
                toVersion={semanticDiffTarget.to}
              />
              <button onClick={() => setSemanticDiffTarget(null)}>Close</button>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}