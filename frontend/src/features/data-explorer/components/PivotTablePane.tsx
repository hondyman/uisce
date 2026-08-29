import React, { useMemo } from 'react';
import {
  Box,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Stack,
  Chip,
} from '@mui/material';
import type {
  ExplorerResult,
  ExplorerSource,
  ExplorerQueryState,
} from '../types/dataExplorerTypes';
import {
  EXPLORER_BG,
  EXPLORER_BORDER,
  EXPLORER_TEXT,
  EXPLORER_MUTED,
  EXPLORER_ACCENT,
} from '../types/dataExplorerTypes';

interface PivotTablePaneProps {
  source: ExplorerSource;
  state: ExplorerQueryState;
  result: ExplorerResult | null;
}

export const PivotTablePane: React.FC<PivotTablePaneProps> = ({
  source,
  state,
  result,
}) => {
  const rowDim = state.dimensions[0]?.fieldId || result?.columns[0]?.name;
  const colDim = state.timeDimensions[0]?.fieldId || state.dimensions[1]?.fieldId;
  const measure = state.measures[0]?.fieldId || result?.columns.find(c => c.type === 'number')?.name;

  const { rowHeaders, colHeaders, matrix, grandTotals } = useMemo(() => {
    if (!result || result.rows.length === 0 || !rowDim) {
      return { rowHeaders: [], colHeaders: [], matrix: new Map(), grandTotals: { rows: {}, cols: {}, total: 0 } };
    }

    const rows = result.rows;
    const rSet = new Set<string>();
    const cSet = new Set<string>();
    const mat = new Map<string, Map<string, number>>();
    const rTotals: Record<string, number> = {};
    const cTotals: Record<string, number> = {};
    let grandTotal = 0;

    rows.forEach((row) => {
      const rKey = String(row[rowDim] ?? 'Unknown');
      const cKey = colDim ? String(row[colDim] ?? 'All') : 'Total';
      let val = 0;

      // Extract measure value or infer from numeric column
      if (measure && row[measure] !== undefined) {
        val = Number(row[measure]) || 0;
      } else {
        const numEntry = Object.entries(row).find(([k, v]) => typeof v === 'number' && k !== rowDim && k !== colDim);
        val = numEntry ? Number(numEntry[1]) : 1;
      }

      rSet.add(rKey);
      cSet.add(cKey);

      if (!mat.has(rKey)) {
        mat.set(rKey, new Map());
      }
      const existing = mat.get(rKey)!.get(cKey) || 0;
      mat.get(rKey)!.set(cKey, existing + val);

      rTotals[rKey] = (rTotals[rKey] || 0) + val;
      cTotals[cKey] = (cTotals[cKey] || 0) + val;
      grandTotal += val;
    });

    return {
      rowHeaders: Array.from(rSet),
      colHeaders: Array.from(cSet),
      matrix: mat,
      grandTotals: { rows: rTotals, cols: cTotals, total: grandTotal },
    };
  }, [result, rowDim, colDim, measure]);

  if (!result || result.rows.length === 0) {
    return (
      <Box sx={{ p: 4, textAlign: 'center', color: EXPLORER_MUTED }}>
        <Typography variant="body2">No data to pivot. Select dimensions and measures to generate pivot matrix.</Typography>
      </Box>
    );
  }

  const maxVal = Math.max(...Object.values(grandTotals.rows), 1);

  return (
    <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Pivot Dimension Shelves Info Bar */}
      <Stack direction="row" spacing={2} alignItems="center" sx={{ px: 2, py: 1.2, borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper' }}>
        <Stack direction="row" spacing={1} alignItems="center">
          <Typography variant="caption" fontWeight={700} color="text.secondary">Row Dimension:</Typography>
          <Chip label={rowDim} size="small" sx={{ height: 20, fontSize: 11, fontWeight: 700 }} color="primary" />
        </Stack>
        {colDim && (
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="caption" fontWeight={700} color="text.secondary">Column Pivot:</Typography>
            <Chip label={colDim} size="small" sx={{ height: 20, fontSize: 11, fontWeight: 700 }} color="warning" />
          </Stack>
        )}
        <Stack direction="row" spacing={1} alignItems="center">
          <Typography variant="caption" fontWeight={700} color="text.secondary">Measure Values:</Typography>
          <Chip label={measure || 'Count'} size="small" sx={{ height: 20, fontSize: 11, fontWeight: 700 }} color="success" />
        </Stack>
      </Stack>

      <TableContainer component={Paper} elevation={0} sx={{ flex: 1, overflow: 'auto' }}>
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell sx={{ fontWeight: 800, bgcolor: EXPLORER_BG, color: EXPLORER_TEXT, minWidth: 160 }}>
                {rowDim.toUpperCase()}
              </TableCell>
              {colHeaders.map((col) => (
                <TableCell key={col} align="right" sx={{ fontWeight: 800, bgcolor: EXPLORER_BG, color: EXPLORER_TEXT, minWidth: 120 }}>
                  {col}
                </TableCell>
              ))}
              <TableCell align="right" sx={{ fontWeight: 800, bgcolor: EXPLORER_BG, color: EXPLORER_TEXT, minWidth: 120, borderLeft: `2px solid ${EXPLORER_BORDER}` }}>
                TOTAL
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rowHeaders.map((r) => {
              const rowTotal = grandTotals.rows[r] || 0;
              return (
                <TableRow key={r} hover>
                  <TableCell sx={{ fontWeight: 700, bgcolor: 'background.paper', color: EXPLORER_TEXT }}>
                    {r}
                  </TableCell>
                  {colHeaders.map((c) => {
                    const val = matrix.get(r)?.get(c) || 0;
                    const heatRatio = maxVal > 0 ? Math.min(val / (maxVal / colHeaders.length || 1), 1) : 0;
                    return (
                      <TableCell
                        key={c}
                        align="right"
                        sx={{
                          fontFamily: 'monospace',
                          fontSize: 12,
                          bgcolor: val > 0 ? `rgba(249, 245, 6, ${0.05 + heatRatio * 0.18})` : 'inherit',
                        }}
                      >
                        {val ? val.toLocaleString(undefined, { maximumFractionDigits: 2 }) : '—'}
                      </TableCell>
                    );
                  })}
                  <TableCell align="right" sx={{ fontWeight: 700, fontFamily: 'monospace', fontSize: 12, borderLeft: `2px solid ${EXPLORER_BORDER}`, bgcolor: 'rgba(0,0,0,0.02)' }}>
                    {rowTotal.toLocaleString(undefined, { maximumFractionDigits: 2 })}
                  </TableCell>
                </TableRow>
              );
            })}
            {/* Grand Total Row */}
            <TableRow sx={{ bgcolor: EXPLORER_BG }}>
              <TableCell sx={{ fontWeight: 800, color: EXPLORER_TEXT }}>GRAND TOTAL</TableCell>
              {colHeaders.map((c) => (
                <TableCell key={c} align="right" sx={{ fontWeight: 800, fontFamily: 'monospace', fontSize: 12 }}>
                  {(grandTotals.cols[c] || 0).toLocaleString(undefined, { maximumFractionDigits: 2 })}
                </TableCell>
              ))}
              <TableCell align="right" sx={{ fontWeight: 900, fontFamily: 'monospace', fontSize: 13, borderLeft: `2px solid ${EXPLORER_BORDER}`, color: '#16a34a' }}>
                {grandTotals.total.toLocaleString(undefined, { maximumFractionDigits: 2 })}
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};

export default PivotTablePane;
