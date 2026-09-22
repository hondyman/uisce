/**
 * QueryResultsPanel - the centralized Table Results / Compiled SQL /
 * Visual Charts / Execution Plan tab strip, built on the real
 * previewQuery/executeQuery backend calls (features/query-builder/
 * services/queryBuilderApi.ts) - never a client-side SQL simulation.
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
 * QueryDef does) and just hand this component the results.
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

export interface QueryResultColumns { name: string; type?: string }
export interface QueryTableResult { columns: QueryResultColumns[]; rows: Record<string, unknown>[] }
export interface QueryCompiledSql { sql: string; dialect?: string }

export interface QueryResultsPanelProps {
  runResult: QueryTableResult | null;
  running?: boolean;
  runError?: string | null;

  compiledSql: QueryCompiledSql | null;
  sqlLoading?: boolean;
  sqlError?: string | null;
  /** Called the first time the Compiled SQL tab is opened (lazy-compile),
   * and whenever the caller wants a fresh compile after the query changed. */
  onRequestCompileSql: () => void;

  /** Execution plan / diagnostics: omitted entirely (no backend endpoint
   * for this yet) unless the caller explicitly supplies content - see the
   * doc comment above on why this never falls back to a fabricated DAG. */
  executionPlan?: React.ReactNode;
}

export default function QueryResultsPanel({
  runResult, running, runError, compiledSql, sqlLoading, sqlError, onRequestCompileSql, executionPlan,
}: QueryResultsPanelProps) {
  const [tab, setTab] = useState(0);
  const [chartType, setChartType] = useState<SavedQueryChartType>('bar');
  const [chartDim, setChartDim] = useState('');
  const [chartMeasure, setChartMeasure] = useState('');

  return (
    <Paper variant="outlined" sx={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 300 }}>
      <Tabs
        value={tab}
        onChange={(_, v) => { setTab(v); if (v === 1 && !compiledSql) onRequestCompileSql(); }}
        sx={{ borderBottom: 1, borderColor: 'divider', px: 1, flexShrink: 0 }}
      >
        <Tab label={runResult ? `Table Results (${runResult.rows.length})` : 'Table Results'} />
        <Tab label="Compiled SQL" />
        <Tab label="Visual Charts" />
        {executionPlan !== undefined && <Tab label="Execution Plan" />}
      </Tabs>

      {tab === 0 && (
        <Box sx={{ p: 2, flex: 1, overflow: 'auto' }}>
          {running ? (
            <Box display="flex" justifyContent="center" p={2}><CircularProgress size={20} /></Box>
          ) : runError ? (
            <Alert severity="error">{runError}</Alert>
          ) : runResult ? (
            <TableContainer>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    {runResult.columns.map((c) => <TableCell key={c.name}>{c.name}</TableCell>)}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {runResult.rows.map((row, i) => (
                    <TableRow key={i}>
                      {runResult.columns.map((c) => <TableCell key={c.name}>{String(row[c.name] ?? '')}</TableCell>)}
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

      {tab === 1 && (
        <Box sx={{ p: 2, flex: 1, overflow: 'auto' }}>
          {sqlLoading ? (
            <Box display="flex" justifyContent="center" p={2}><CircularProgress size={20} /></Box>
          ) : sqlError ? (
            <Alert severity="error">{sqlError}</Alert>
          ) : compiledSql ? (
            <Box component="pre" sx={{
              fontFamily: 'monospace', fontSize: '0.8rem', whiteSpace: 'pre-wrap', wordBreak: 'break-word',
              bgcolor: 'background.default', p: 2, borderRadius: 1, m: 0,
            }}>
              {compiledSql.sql}
            </Box>
          ) : (
            <Typography variant="body2" color="text.secondary">No fields selected yet.</Typography>
          )}
        </Box>
      )}

      {tab === 2 && (
        <Box sx={{ p: 2, flex: 1, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          {!runResult ? (
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
                  <Select value={chartDim || runResult.columns[0]?.name || ''} label={chartType === 'pie' ? 'Category' : 'X axis'}
                    onChange={(e) => setChartDim(e.target.value)}>
                    {runResult.columns.map((c) => <MenuItem key={c.name} value={c.name}>{c.name}</MenuItem>)}
                  </Select>
                </FormControl>
                <FormControl size="small" sx={{ minWidth: 160 }}>
                  <InputLabel>{chartType === 'pie' ? 'Value' : 'Y axis'}</InputLabel>
                  <Select value={chartMeasure || runResult.columns[1]?.name || ''} label={chartType === 'pie' ? 'Value' : 'Y axis'}
                    onChange={(e) => setChartMeasure(e.target.value)}>
                    {runResult.columns.map((c) => <MenuItem key={c.name} value={c.name}>{c.name}</MenuItem>)}
                  </Select>
                </FormControl>
              </Stack>
              <Box sx={{ flex: 1, minHeight: 320 }}>
                <LazyECharts
                  option={buildChartOption(runResult.rows, runResult.columns, chartType, chartDim || undefined, chartMeasure || undefined)}
                  style={{ height: '100%', width: '100%' }}
                />
              </Box>
            </>
          )}
        </Box>
      )}

      {executionPlan !== undefined && tab === 3 && (
        <Box sx={{ p: 2, flex: 1, overflow: 'auto' }}>{executionPlan}</Box>
      )}
    </Paper>
  );
}
