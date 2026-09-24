/**
 * QueryResultsPanel - the centralized Table Results / Compiled SQL /
 * Visual Charts / [extra] tab strip, built on the real
 * previewQuery/executeQuery backend calls (features/query-execution/) -
 * never a client-side SQL simulation.
 *
 * This existed twice before: Query Builder's SavedQueryEditor had its own
 * copy, and BusinessObjectDetailsPage's LiveQueryTab had a *different*
 * one whose "Pushdown SQL Engine" tab displayed a hand-rolled SQL string
 * it built itself instead of the real compiled SQL it was already
 * fetching via the same previewQuery call. Centralizing here means every
 * consumer shows the SQL that actually runs, not a plausible-looking
 * guess at it - and any future page (Report Builder) gets the same tabs
 * for free instead of writing a third copy.
 *
 * Data-agnostic by design: callers own their own run/compile calls
 * (a saved query previews through a different endpoint than an ad-hoc
 * QueryDef does) and just hand this component the results in the
 * canonical QueryResultSet shape (features/query-execution/types.ts).
 *
 * The `extraTabs` registry replaces the older single-slot
 * `executionPlan?: ReactNode` prop: consumers append tabs after the
 * three base tabs (Table Results, Compiled SQL, Visual Charts), each with
 * a stable id, label, optional icon/badge, and `content`. The house
 * rule: content must be derived from a real backend response
 * (previewQuery / executeQuery / a real EXPLAIN endpoint). Decorative or
 * simulated diagnostics do not ship here - see the LiveQueryTab fake-DAG
 * incident for why.
 */
import React, { useState } from 'react';
import {
  Box, Paper, Typography, Tabs, Tab, Alert, CircularProgress,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Stack, ToggleButtonGroup, ToggleButton, FormControl, InputLabel, Select, MenuItem,
} from '@mui/material';
import BarChartIcon from '@mui/icons-material/BarChart';
import ShowChartIcon from '@mui/icons-material/ShowChart';
import PieChartIcon from '@mui/icons-material/PieChart';
import LazyECharts from '../LazyECharts';
import { buildChartOption } from '../../features/query-builder/utils/chartOption';
import type { SavedQueryChartType } from '../../features/query-builder/types/queryDef';
import type { QueryResultSet } from '../../features/query-execution/types';

export interface QueryResultsExtraTab {
  id: string;
  label: string;
  icon?: React.ReactNode;
  badge?: string | number;
  /**
   * House rule: content must be derived from a real backend response
   * (previewQuery / executeQuery / a real EXPLAIN endpoint). Decorative or
   * simulated diagnostics do not ship here - see this file's header
   * comment for why (the LiveQueryTab fake-DAG incident).
   */
  content: React.ReactNode;
}

export interface QueryResultsPanelProps {
  /** Canonical result shape from features/query-execution/types. */
  resultSet: QueryResultSet | null;
  running?: boolean;
  runError?: string | null;

  sql: { sql: string; dialect?: string } | null;
  sqlLoading?: boolean;
  sqlError?: string | null;
  /** Called the first time the Compiled SQL tab is opened (lazy-compile),
   * and whenever the caller wants a fresh compile after the query changed. */
  onRequestCompileSql: () => void;

  /** Appended after Table Results / Compiled SQL / Visual Charts. */
  extraTabs?: QueryResultsExtraTab[];

  /** Which tab is open initially - consumers restoring editor state use this. */
  initialTabId?: string;
}

const TAB_RESULTS = 'results';
const TAB_SQL = 'sql';
const TAB_CHARTS = 'charts';

export default function QueryResultsPanel({
  resultSet, running, runError,
  sql, sqlLoading, sqlError, onRequestCompileSql,
  extraTabs, initialTabId,
}: QueryResultsPanelProps) {
  const [activeTabId, setActiveTabId] = useState<string>(
    initialTabId ?? (resultSet ? TAB_RESULTS : TAB_SQL)
  );
  const [chartType, setChartType] = useState<SavedQueryChartType>('bar');
  const [chartDim, setChartDim] = useState('');
  const [chartMeasure, setChartMeasure] = useState('');

  const baseTabs = [
    { id: TAB_RESULTS, label: resultSet ? `Table Results (${resultSet.rows.length})` : 'Table Results' },
    { id: TAB_SQL, label: 'Compiled SQL' },
    { id: TAB_CHARTS, label: 'Visual Charts' },
  ];
  const allTabs = [
    ...baseTabs,
    ...(extraTabs ?? []).map((t) => ({ id: t.id, label: t.label })),
  ];
  const activeIndex = Math.max(0, allTabs.findIndex((t) => t.id === activeTabId));
  const activeExtra = (extraTabs ?? []).find((t) => t.id === activeTabId);

  return (
    <Paper variant="outlined" sx={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 300 }}>
      <Tabs
        value={activeIndex}
        onChange={(_, v: number) => {
          const next = allTabs[v];
          setActiveTabId(next.id);
          if (next.id === TAB_SQL && !sql) onRequestCompileSql();
        }}
        sx={{ borderBottom: 1, borderColor: 'divider', px: 1, flexShrink: 0 }}
      >
        {allTabs.map((t) => (
          <Tab key={t.id} label={t.label} />
        ))}
      </Tabs>

      {activeTabId === TAB_RESULTS && (
        <Box sx={{ p: 2, flex: 1, overflow: 'auto' }}>
          {running ? (
            <Box display="flex" justifyContent="center" p={2}><CircularProgress size={20} /></Box>
          ) : runError ? (
            <Alert severity="error">{runError}</Alert>
          ) : resultSet ? (
            <TableContainer>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    {resultSet.columns.map((c) => (
                      <TableCell key={c.name}>{c.label ?? c.name}</TableCell>
                    ))}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {resultSet.rows.map((row, i) => (
                    <TableRow key={i}>
                      {resultSet.columns.map((c) => (
                        <TableCell key={c.name}>{String(row[c.name] ?? '')}</TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          ) : (
            <Typography variant="body2" color="text.secondary">Run the query to see results here.</Typography>
          )}
        </Box>
      )}

      {activeTabId === TAB_SQL && (
        <Box sx={{ p: 2, flex: 1, overflow: 'auto' }}>
          {sqlLoading ? (
            <Box display="flex" justifyContent="center" p={2}><CircularProgress size={20} /></Box>
          ) : sqlError ? (
            <Alert severity="error">{sqlError}</Alert>
          ) : sql ? (
            <Box component="pre" sx={{
              fontFamily: 'monospace', fontSize: '0.8rem', whiteSpace: 'pre-wrap', wordBreak: 'break-word',
              bgcolor: 'background.default', p: 2, borderRadius: 1, m: 0,
            }}>
              {sql.sql}
            </Box>
          ) : (
            <Typography variant="body2" color="text.secondary">No fields selected yet.</Typography>
          )}
        </Box>
      )}

      {activeTabId === TAB_CHARTS && (
        <Box sx={{ p: 2, flex: 1, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          {!resultSet ? (
            <Typography variant="body2" color="text.secondary">Run the query first to chart its results.</Typography>
          ) : (
            <>
              <Stack direction="row" spacing={1.5} alignItems="center" flexWrap="wrap">
                <ToggleButtonGroup size="small" exclusive value={chartType} onChange={(_, v) => v && setChartType(v)}>
                  <ToggleButton value="bar"><BarChartIcon fontSize="small" /></ToggleButton>
                  <ToggleButton value="line"><ShowChartIcon fontSize="small" /></ToggleButton>
                  <ToggleButton value="pie"><PieChartIcon fontSize="small" /></ToggleButton>
                </ToggleButtonGroup>
                <FormControl size="small" sx={{ minWidth: 160 }}>
                  <InputLabel>{chartType === 'pie' ? 'Category' : 'X axis'}</InputLabel>
                  <Select value={chartDim || resultSet.columns[0]?.name || ''} label={chartType === 'pie' ? 'Category' : 'X axis'}
                    onChange={(e) => setChartDim(e.target.value)}>
                    {resultSet.columns.map((c) => <MenuItem key={c.name} value={c.name}>{c.label ?? c.name}</MenuItem>)}
                  </Select>
                </FormControl>
                <FormControl size="small" sx={{ minWidth: 160 }}>
                  <InputLabel>{chartType === 'pie' ? 'Value' : 'Y axis'}</InputLabel>
                  <Select value={chartMeasure || resultSet.columns[1]?.name || ''} label={chartType === 'pie' ? 'Value' : 'Y axis'}
                    onChange={(e) => setChartMeasure(e.target.value)}>
                    {resultSet.columns.map((c) => <MenuItem key={c.name} value={c.name}>{c.label ?? c.name}</MenuItem>)}
                  </Select>
                </FormControl>
              </Stack>
              <Box sx={{ flex: 1, minHeight: 320 }}>
                <LazyECharts
                  option={buildChartOption(resultSet.rows, resultSet.columns, chartType, chartDim || undefined, chartMeasure || undefined)}
                  style={{ height: '100%', width: '100%' }}
                />
              </Box>
            </>
          )}
        </Box>
      )}

      {activeExtra && (
        <Box sx={{ p: 2, flex: 1, overflow: 'auto' }}>{activeExtra.content}</Box>
      )}
    </Paper>
  );
}
