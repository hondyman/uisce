import React, { useMemo, useState } from 'react';
import {
  Box,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  TablePagination,
  Typography,
  CircularProgress,
  Alert,
  Stack,
} from '@mui/material';
import {
  EXPLORER_BG,
  EXPLORER_BORDER,
  EXPLORER_MUTED,
  EXPLORER_TEXT,
} from '../types/dataExplorerTypes';
import type { ExplorerResult } from '../types/dataExplorerTypes';

interface ResultsTablePaneProps {
  result: ExplorerResult | null;
  isLoading: boolean;
  isPreviewing: boolean;
  error: string | null;
}

type SortDir = 'asc' | 'desc';

function formatCell(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'number') {
    if (Number.isInteger(value)) return value.toLocaleString();
    const abs = Math.abs(value);
    if (abs >= 1000 || abs < 0.001) return value.toExponential(2);
    return Number(value.toFixed(4)).toString();
  }
  if (value instanceof Date) return value.toISOString();
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

export const ResultsTablePane: React.FC<ResultsTablePaneProps> = ({
  result,
  isLoading,
  isPreviewing,
  error,
}) => {
  const [sortColumn, setSortColumn] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDir>('asc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);

  const sortedRows = useMemo(() => {
    if (!result || !result.rows) return [];
    if (!sortColumn) return result.rows;
    const rows = [...result.rows];
    rows.sort((a, b) => {
      const aVal = (a as Record<string, unknown>)[sortColumn];
      const bVal = (b as Record<string, unknown>)[sortColumn];
      if (aVal === bVal) return 0;
      if (aVal === null || aVal === undefined) return 1;
      if (bVal === null || bVal === undefined) return -1;
      if (typeof aVal === 'number' && typeof bVal === 'number') {
        return sortDirection === 'asc' ? aVal - bVal : bVal - aVal;
      }
      return sortDirection === 'asc'
        ? String(aVal).localeCompare(String(bVal))
        : String(bVal).localeCompare(String(aVal));
    });
    return rows;
  }, [result, sortColumn, sortDirection]);

  const paginatedRows = useMemo(() => {
    if (!result) return [];
    return sortedRows.slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage);
  }, [sortedRows, page, rowsPerPage, result]);

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  if (isLoading && (!result || result.rows.length === 0)) {
    return (
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: EXPLORER_BG,
        }}
      >
        <CircularProgress sx={{ color: '#f9f506' }} />
      </Box>
    );
  }

  if (!result || result.columns.length === 0) {
    return (
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: EXPLORER_BG,
          p: 6,
        }}
      >
        <Box sx={{ textAlign: 'center', maxWidth: 480 }}>
          <Typography variant="h6" fontWeight={700} sx={{ color: EXPLORER_TEXT, mb: 1 }}>
            {isPreviewing ? 'Compiling query…' : 'No results yet'}
          </Typography>
          <Typography variant="body2" sx={{ color: EXPLORER_MUTED }}>
            {isPreviewing
              ? 'Waiting for the next debounced update.'
              : 'Add dimensions or measures from the left and click Run Query.'}
          </Typography>
        </Box>
      </Box>
    );
  }

  return (
    <Box
      sx={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      {isLoading && (
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            bgcolor: 'rgba(255,255,255,0.6)',
            zIndex: 5,
          }}
        >
          <CircularProgress sx={{ color: '#f9f506' }} />
        </Box>
      )}
      <TableContainer component={Paper} elevation={0} sx={{ flex: 1, overflow: 'auto' }}>
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              {result.columns.map((col) => (
                <TableCell
                  key={col.name}
                  sx={{
                    fontWeight: 700,
                    bgcolor: EXPLORER_BG,
                    color: EXPLORER_MUTED,
                    borderBottom: `1px solid ${EXPLORER_BORDER}`,
                  }}
                >
                  <TableSortLabel
                    active={sortColumn === col.name}
                    direction={sortColumn === col.name ? sortDirection : 'asc'}
                    onClick={() => handleSort(col.name)}
                  >
                    {col.name}
                  </TableSortLabel>
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {paginatedRows.map((row, idx) => (
              <TableRow
                key={idx}
                hover
                sx={{ '&:hover': { bgcolor: 'rgba(249, 245, 6, 0.05)' } }}
              >
                {result.columns.map((col) => (
                  <TableCell
                    key={col.name}
                    sx={{
                      color: EXPLORER_TEXT,
                      borderBottom: `1px solid ${EXPLORER_BORDER}`,
                      fontFamily: 'monospace',
                      fontSize: 13,
                    }}
                  >
                    {formatCell((row as Record<string, unknown>)[col.name])}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      <Stack
        direction="row"
        justifyContent="space-between"
        alignItems="center"
        sx={{
          px: 2,
          py: 1,
          borderTop: `1px solid ${EXPLORER_BORDER}`,
          bgcolor: 'white',
        }}
      >
        <Typography variant="caption" sx={{ color: EXPLORER_MUTED }}>
          {result.rowCount.toLocaleString()} rows · sorted by{' '}
          <strong>
            {sortColumn
              ? `${sortColumn} ${sortDirection.toUpperCase()}`
              : 'insertion order'}
          </strong>
        </Typography>
        <TablePagination
          component="div"
          count={result.rows.length}
          page={page}
          onPageChange={(_, p) => setPage(p)}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={(e) => {
            setRowsPerPage(parseInt(e.target.value, 10));
            setPage(0);
          }}
          rowsPerPageOptions={[10, 25, 50, 100, 250]}
          sx={{ '.MuiTablePagination-toolbar': { minHeight: 40 } }}
        />
      </Stack>
    </Box>
  );
};

export default ResultsTablePane;
