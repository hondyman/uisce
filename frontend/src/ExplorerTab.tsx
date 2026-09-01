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
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import Modal from '@mui/material/Modal';

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
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  void onOpenSavedQuery;
  void onOpenWorkbook;
  void onStartTour;
  void views;
  const [activeResultTab, setActiveResultTab] = useState('grid');
  const [selectedSemanticView, setSelectedSemanticView] = useState<SemanticViewMeta | ViewMeta | null>(null);
  const [currentSavedQuery, setCurrentSavedQuery] = useState<FullSavedQuery | null>(null);
  const [currentWorkbook, setCurrentWorkbook] = useState<FullWorkbook | null>(null);
  const [diffTarget, setDiffTarget] = useState<{ snapshotId: string; compareToId: string } | null>(null);
  const [showSimulationPanel, setShowSimulationPanel] = useState(false);
  const [requestAccessModelId, setRequestAccessModelId] = useState<string | null>(null);
  const [semanticDiffTarget, setSemanticDiffTarget] = useState<{ viewName: string; from: number; to: number } | null>(null);
  const [deniedDecision, setDeniedDecision] = useState<{ decisionId: string; reason: string } | null>(null);
  const [claims, setClaims] = useState<SemanticModelClaim[]>([]);
  const [visibleViews, setVisibleViews] = useState<SemanticViewMeta[]>([]);

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

  const tenantId = "acme_corp";
  const datasourceId = "mock-datasource-id";
  const currentUser = "patrick";

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
      }
    };
    fetchData();
  }, [datasourceId, currentUser, tenantId]);

  const handleExecuteSemanticQuery = useCallback(async (query: SemanticQuery, view: SemanticViewMeta | ViewMeta) => {
    if (!selectedSemanticView) return;
    setDeniedDecision(null);

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
  }, [onChange, selectedSemanticView, currentUser, tenantId]);

  const handleNLQ = async (viewName: string, query: SemanticQuery) => {
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
    const fullTemplate = await getQueryTemplate(templateMeta.id);
    const view = visibleViews.find(v => getViewIdentifierSafe(v) === fullTemplate.semantic_view);
    if (!view) {
      devWarn(`Template view "${fullTemplate.semantic_view}" not found.`);
      return;
    }
    setSelectedSemanticView(view as SemanticViewMeta | ViewMeta);

    const newQuery: SemanticQuery = {
      dimensions: fullTemplate.default_dimensions || [],
      metrics: fullTemplate.default_metrics || [],
      filters: Array.isArray(fullTemplate.required_filters)
        ? (fullTemplate.required_filters as Array<{ field: string; op: string; values: string[] }>)
        : [],
      order: [],
      limit: 100,
    };

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
    const currentSnapshotId = "current-state-id-mock";
    setDiffTarget({ snapshotId, compareToId: currentSnapshotId });
  };

  const handleRequestAccess = (assetType: 'dimensions' | 'metrics') => {
    if (selectedSemanticView) {
      const modelId = getViewIdentifierSafe(selectedSemanticView) || '';
      const reason = `User '${currentUser}' attempted to access '${assetType}' on model '${modelId}' without permission.`;
      logAccessDeniedAttempt('semantic_model_scope', modelId, reason);
      setRequestAccessModelId(modelId);
    }
  };

  const getClaimsForSelectedView = () => {
    if (!selectedSemanticView) return [];
    const id = getViewIdentifierSafe(selectedSemanticView) || '';
    return claims.filter(c => c.model_id === id);
  };

  const resultTabs = [
    { id: 'grid', label: 'Grid', disabled: false },
    { id: 'viz', label: 'Visualization', disabled: !tab.result },
    { id: 'sql', label: 'SQL', disabled: !tab.compile },
    { id: 'graphql', label: 'GraphQL', disabled: !tab.compile },
    { id: 'explain', label: 'Explain', disabled: !tab.explain },
    { id: 'diff', label: 'Preview Diff', disabled: !tab.savedId },
  ];

  return (
    <Box sx={{ display: 'flex', height: '100%', width: '100%' }}>
      <Box
        component="aside"
        sx={{
          width: 280,
          borderRight: '1px solid',
          borderColor: isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)',
          overflowY: 'auto',
          p: 2,
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
        }}
      >
        <SemanticViewPicker datasourceId={datasourceId} onSelect={setSelectedSemanticView} views={visibleViews} />
        <HistoryPanel onLoadQuery={() => {}} />
        {selectedSemanticView && (
          <SemanticVersionPanel viewName={getViewIdentifierSafe(selectedSemanticView) || ''} onCompare={(from, to) => setSemanticDiffTarget({ viewName: getViewIdentifierSafe(selectedSemanticView) || '', from, to })} />
        )}
        <Box
          sx={{
            p: 2,
            borderRadius: 1,
            backgroundColor: isDark ? 'rgba(255,255,255,0.05)' : 'rgba(0,0,0,0.02)',
          }}
        >
          <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
            Governance Tools
          </Typography>
          <Button
            size="small"
            variant="outlined"
            onClick={() => setShowSimulationPanel(true)}
            fullWidth
          >
            Simulate Claims
          </Button>
        </Box>
        <ReviewerInbox reviewerId="current_reviewer" />
      </Box>

      <Box
        component="main"
        sx={{ flex: 1, overflow: 'auto', p: 2 }}
      >
        {selectedSemanticView ? (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, height: '100%' }}>
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
            <Box sx={{ p: 2, borderRadius: 1, backgroundColor: isDark ? 'rgba(255,255,255,0.02)' : 'rgba(0,0,0,0.02)' }}>
              <SemanticQueryInput onQuery={handleNLQ} currentDatasource={datasourceId} currentUser={currentUser} />
            </Box>
            <SemanticQueryComposer
              view={selectedSemanticView}
              claims={getClaimsForSelectedView()}
              onExecute={(q) => selectedSemanticView && handleExecuteSemanticQuery(q, selectedSemanticView)}
              onRequestAccess={handleRequestAccess}
            />
            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
                {resultTabs.map(tabItem => (
                  <Button
                    key={tabItem.id}
                    size="small"
                    variant={activeResultTab === tabItem.id ? 'contained' : 'outlined'}
                    onClick={() => setActiveResultTab(tabItem.id)}
                    disabled={tabItem.disabled}
                  >
                    {tabItem.label}
                  </Button>
                ))}
              </Box>
              <Paper sx={{ flex: 1, p: 2, overflow: 'auto' }}>
                {activeResultTab === 'grid' && <ResultsGrid rows={tab.result?.rows || []} columns={tab.result?.columns || []} page={tab.result?.page as PageInfo} onPageChange={() => {}} />}
                {activeResultTab === 'viz' && tab.result && <VisualizationPanel rows={tab.result.rows} columns={tab.result.columns} viz={tab.viz || { type: 'auto' }} onCrossFilter={f => onChange({ query: { ...tab.query, filters: [...(tab.query.filters || []), f], offset: 0 } })} />}
                {activeResultTab === 'sql' && <SQLTab sql={tab.compile?.sql} />}
                {activeResultTab === 'graphql' && <GraphQLTab graphql={tab.compile?.graphql} />}
                {activeResultTab === 'explain' && <ExplainTab explain={tab.compile?.explain} />}
                {activeResultTab === 'diff' && tab.savedId && <PreviewDiffTab savedId={tab.savedId} />}
              </Paper>
            </Box>
            <Box
              component="aside"
              sx={{
                width: 300,
                borderLeft: '1px solid',
                borderColor: isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)',
                p: 2,
                overflowY: 'auto',
              }}
            >
              <InsightsPanel result={tab.result} />
              {currentSavedQuery && (
                <CommentsPanel assetId={currentSavedQuery.id} assetType="query" />
              )}
              {currentWorkbook && (
                <SnapshotPanel dashboardId={currentWorkbook.id} onCompare={handleCompareSnapshot} />
              )}
              {selectedSemanticView && (
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => setRequestAccessModelId(getViewIdentifierSafe(selectedSemanticView) || '')}
                  sx={{ mt: 2 }}
                >
                  Request Access to {getViewDisplayTitle(selectedSemanticView)}
                </Button>
              )}
            </Box>
          </Box>
        ) : (
          <Box sx={{ textAlign: 'center', py: 4 }}>
            <Typography variant="h5" sx={{ mb: 3 }}>
              Welcome to the Explorer
            </Typography>
            <Box sx={{ mb: 3, maxWidth: 600, mx: 'auto' }}>
              <SemanticQueryInput onQuery={handleNLQ} currentDatasource={datasourceId} currentUser={currentUser} />
            </Box>
            <Typography sx={{ mb: 2, color: 'text.secondary' }}>
              Select a Semantic View, load a dashboard, or start from a template below.
            </Typography>
            <Button
              variant="contained"
              onClick={() => handleLoadWorkbook('wb-123')}
              sx={{ mb: 3 }}
            >
              Load Demo Dashboard
            </Button>
            <QueryTemplateBrowser datasourceId={datasourceId} onSelect={handleSelectTemplate} />
          </Box>
        )}
      </Box>

      <Modal open={showSimulationPanel} onClose={() => setShowSimulationPanel(false)}>
        <Box
          sx={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            width: 800,
            bgcolor: 'background.paper',
            p: 3,
            borderRadius: 2,
            maxHeight: '80vh',
            overflow: 'auto',
          }}
        >
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">Claim Simulation</Typography>
            <Button onClick={() => setShowSimulationPanel(false)}>Close</Button>
          </Box>
          <ClaimSimulationPanel availableModels={[{id: 'd1b6a5e0-9a9a-4b1a-8b0a-1b1b1b1b1b1b', name: 'orders_view'}]} />
        </Box>
      </Modal>

      <Modal open={!!requestAccessModelId} onClose={() => setRequestAccessModelId(null)}>
        <Box
          sx={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            width: 500,
            bgcolor: 'background.paper',
            p: 3,
            borderRadius: 2,
          }}
        >
          <RequestAccessModal modelId={requestAccessModelId || ''} onClose={() => setRequestAccessModelId(null)} />
        </Box>
      </Modal>

      <Modal open={!!diffTarget} onClose={() => setDiffTarget(null)}>
        <Box
          sx={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            width: 800,
            bgcolor: 'background.paper',
            p: 3,
            borderRadius: 2,
            maxHeight: '80vh',
            overflow: 'auto',
          }}
        >
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">Snapshot Diff</Typography>
            <Button onClick={() => setDiffTarget(null)}>Close</Button>
          </Box>
          {diffTarget && <SnapshotDiffViewer snapshotId={diffTarget.snapshotId} compareToId={diffTarget.compareToId} />}
        </Box>
      </Modal>

      <Modal open={!!semanticDiffTarget} onClose={() => setSemanticDiffTarget(null)}>
        <Box
          sx={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            width: 800,
            bgcolor: 'background.paper',
            p: 3,
            borderRadius: 2,
            maxHeight: '80vh',
            overflow: 'auto',
          }}
        >
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">Version Diff</Typography>
            <Button onClick={() => setSemanticDiffTarget(null)}>Close</Button>
          </Box>
          {semanticDiffTarget && (
            <SemanticDiffViewer
              viewName={semanticDiffTarget.viewName}
              fromVersion={semanticDiffTarget.from}
              toVersion={semanticDiffTarget.to}
            />
          )}
        </Box>
      </Modal>
    </Box>
  );
}
