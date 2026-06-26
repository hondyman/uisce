import React, { useState, useEffect } from 'react';
import { devDebug } from '../../../utils/devLogger';
import {
  Drawer,
  Box,
  Typography,
  IconButton,
  CircularProgress,
  Alert,
  TableContainer,
  Paper,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Collapse,
  Chip,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { gql, useQuery } from '@apollo/client';
import { format } from 'date-fns';
import { useDrillDown, FilterState } from '../../../contexts/DrillDownContext';
import DrillDownFilters from './DrillDownFilters';
import QuickCompareControls from './QuickCompareControls';
import PinnedTabsBar, { TabState } from './PinnedTabsBar';
import DiffRunsTable from './DiffRunsTable';

const GET_HEATMAP_CELL_DETAIL = gql`
  query GetHeatmapCellDetail($context: String!, $filters: jsonb!) {
    heatmap_cell_detail(context: $context, filters: $filters) {
      run_id
      change_id
      timestamp
      decision_a
      decision_b
      violations_added {
        rule_id
        severity
        message
      }
      violations_removed {
        rule_id
        severity
        message
      }
    }
  }
`;

// --- Manual TypeScript types for the heatmap cell detail query (small, defensive subset)
interface HeatmapViolation {
  rule_id: string;
  severity: string;
  message: string;
}

interface HeatmapCellRun {
  run_id: string;
  change_id?: string | null;
  timestamp?: string | number | null;
  decision_a?: string | null;
  decision_b?: string | null;
  violations_added?: HeatmapViolation[] | null;
  violations_removed?: HeatmapViolation[] | null;
}

type GetHeatmapCellDetailResult = {
  heatmap_cell_detail: HeatmapCellRun[];
};

interface UnifiedDrillDownDrawerProps {
  open: boolean;
  onClose: () => void;
}

const ViolationList: React.FC<{ violations: any[] }> = ({ violations }) => {
  if (!violations || violations.length === 0) {
    return <Typography variant="caption" color="text.secondary">None</Typography>;
  }
  return (
    <Box component="ul" sx={{ m: 0, p: 0, pl: 2, listStyleType: 'disc' }}>
      {violations.map((v, i) => (
        <li key={i}>
          <Typography variant="caption">
            <Chip label={v.severity} size="small" color={v.severity === 'breaking' ? 'error' : 'warning'} sx={{ mr: 0.5 }} />
            <strong>{v.rule_id}:</strong> {v.message}
          </Typography>
        </li>
      ))}
    </Box>
  );
};

const RunDetailRow: React.FC<{ run: any; context: string }> = ({ run, context }) => {
  const [open, setOpen] = React.useState(false);
  const runSafe = run || {};
  const added: any[] = Array.isArray(runSafe.violations_added) ? runSafe.violations_added : [];
  const removed: any[] = Array.isArray(runSafe.violations_removed) ? runSafe.violations_removed : [];
  const hasDetails = added.length > 0 || removed.length > 0;

  const getDecisionChip = (decision?: string | null) => {
    if (!decision) return null;
    return <Chip label={decision} color={decision === 'block' ? 'error' : 'success'} size="small" />;
  };

  const shortChangeId = runSafe.change_id ? String(runSafe.change_id).substring(0, 8) : 'unknown';
  const timestampText = runSafe.timestamp ? (() => {
    try {
      return format(new Date(runSafe.timestamp), 'yyyy-MM-dd HH:mm');
    } catch (e) {
      return String(runSafe.timestamp);
    }
  })() : '—';

  return (
    <React.Fragment>
      <TableRow>
        <TableCell>{shortChangeId}</TableCell>
        <TableCell>{timestampText}</TableCell>
        {context === 'policy_compare' ? (
          <>
            <TableCell>{getDecisionChip(runSafe.decision_a)}</TableCell>
            <TableCell>{getDecisionChip(runSafe.decision_b)}</TableCell>
          </>
        ) : (
          <TableCell>{getDecisionChip(runSafe.decision_a)}</TableCell>
        )}
        <TableCell>
          {hasDetails && (
            <IconButton size="small" onClick={() => setOpen(!open)}>
              {open ? 'Hide' : 'Show'}
            </IconButton>
          )}
        </TableCell>
      </TableRow>
      {hasDetails && (
        <TableRow>
          <TableCell className="no-vertical-padding-cell" colSpan={context === 'policy_compare' ? 5 : 4}>
            <Collapse in={open} timeout="auto" unmountOnExit>
              <Box sx={{ m: 1, p: 2, backgroundColor: 'action.hover', borderRadius: 1 }}>
                <Typography variant="subtitle2" gutterBottom>Violations Added</Typography>
                <ViolationList violations={added} />
                {context === 'policy_compare' && (
                  <>
                    <Typography variant="subtitle2" gutterBottom sx={{ mt: 2 }}>Violations Removed</Typography>
                    <ViolationList violations={removed} />
                  </>
                )}
              </Box>
            </Collapse>
          </TableCell>
        </TableRow>
      )}
    </React.Fragment>
  );
};

const UnifiedHeatmapDrillDown: React.FC<UnifiedDrillDownDrawerProps> = ({ open, onClose }) => {
  const { context, filters, setFilters } = useDrillDown();
  const [tabs, setTabs] = useState<TabState[]>([]);
  const [activeTabId, setActiveTabId] = useState<string | null>(null);
  const [diffMode, setDiffMode] = useState<{ tabAId: string; tabBId: string } | null>(null);

  const activeTab = tabs.find((t) => t.id === activeTabId);

  // Initialize or reset tabs when the drawer is opened
  useEffect(() => {
    if (open && context && filters) {
      const initialTab: TabState = {
        id: 'base',
        label: 'Initial View',
        context,
        filters,
        isLoading: true,
      };
      setTabs([initialTab]);
      setActiveTabId('base');
      setDiffMode(null);
    } else if (!open) {
      setTabs([]);
      setActiveTabId(null);
    }
  }, [open, context, filters]);

  // Data fetching for the active tab (typed result)
  const { loading, error, data: fetchedData } = useQuery<GetHeatmapCellDetailResult, { context: string; filters: any }>(GET_HEATMAP_CELL_DETAIL, {
    variables: activeTab ? { context: activeTab.context, filters: activeTab.filters } : undefined,
    skip: !activeTab || !activeTab.isLoading,
  });

  // When the query returns, update local tabs state. Using useEffect avoids setting
  // local React state from within the `onCompleted` callback of Apollo, which can
  // cause subtle ordering issues and is discouraged by Apollo.
  useEffect(() => {
    if (!fetchedData) return;

    const runs = fetchedData.heatmap_cell_detail || [];

    // Quick debug: log any runs that are missing expected fields so we can spot backend issues
    const problematic = runs.filter(r => !r || !r.change_id || !r.timestamp || !Array.isArray(r.violations_added) || !Array.isArray(r.violations_removed));
    if (problematic.length > 0) {
      devDebug('UnifiedHeatmapDrillDown: runs with missing/invalid fields detected', problematic);
      // Lightweight: only log the full raw response when we detected problematic entries
      devDebug('UnifiedHeatmapDrillDown: full GraphQL response (only shown when problems detected)', fetchedData);
    }

    setTabs((prevTabs) =>
      prevTabs.map((t) =>
        t.id === activeTabId ? { ...t, data: runs, isLoading: false } : t
      )
    );
  }, [fetchedData, activeTabId]);

  const handlePinTab = (pinnedFilters: FilterState) => {
    const newId = `tab-${Date.now()}`;
    const newLabel = `Compare: v${pinnedFilters.versionA} vs v${pinnedFilters.versionB}`;
    const newTab: TabState = {
      id: newId,
      label: newLabel,
      context: 'policy_compare',
      filters: pinnedFilters,
      isLoading: true,
    };
    setTabs([...tabs, newTab]);
    setActiveTabId(newId);
    setFilters(pinnedFilters); // Update global context to trigger fetch
  };

  const handleCloseTab = (tabId: string) => {
    const newTabs = tabs.filter(t => t.id !== tabId);
    setTabs(newTabs);
    if (activeTabId === tabId) {
      setActiveTabId(newTabs[0]?.id || null);
    }
  };

  const handleStartDiff = () => {
    // Simple diff: compare first two tabs. A real implementation would have a picker.
    if (tabs.length >= 2) {
      setDiffMode({ tabAId: tabs[0].id, tabBId: tabs[1].id });
    }
  };

  const title = activeTab?.label || 'Drill Down';
  const currentData = diffMode ? null : activeTab?.data;
  const isLoading = loading || (activeTab?.isLoading && !activeTab?.data);

  return (
    <Drawer anchor="right" open={open} onClose={onClose}>
      <Box sx={{ width: 800, p: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h5">{title}</Typography>
          <IconButton onClick={onClose}>
            <CloseIcon />
          </IconButton>
        </Box>

        <PinnedTabsBar tabs={tabs} activeTabId={activeTabId!} onSelectTab={setActiveTabId} onCloseTab={handleCloseTab} onStartDiff={handleStartDiff} />

        <QuickCompareControls onPin={handlePinTab} />

        <DrillDownFilters context={context} />

        {isLoading && <CircularProgress />}
        {error && <Alert severity="error">Failed to load details: {error.message}</Alert>}

        {diffMode && (
          <DiffRunsTable
            dataA={tabs.find(t => t.id === diffMode.tabAId)?.data || []}
            dataB={tabs.find(t => t.id === diffMode.tabBId)?.data || []}
          />
        )}

        {!diffMode && currentData && (
          <TableContainer component={Paper}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Change ID</TableCell>
                  <TableCell>Timestamp</TableCell>
                  {context === 'policy_compare' ? (
                    <>
                      <TableCell>Old Decision</TableCell>
                      <TableCell>New Decision</TableCell>
                    </>
                  ) : (
                    <TableCell>Decision</TableCell>
                  )}
                  <TableCell>Details</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {currentData.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} align="center">
                      No detailed runs found for this cell.
                    </TableCell>
                  </TableRow>
                ) : (
                  currentData.map((run: any) => (
                    <RunDetailRow key={run.run_id} run={run} context={context!} />
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Box>
    </Drawer>
  );
};

export default UnifiedHeatmapDrillDown;