import React, { useState, useMemo } from 'react';
import {
  Box,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  TableContainer,
  TablePagination,
  Paper,
  TableSortLabel,
} from '@mui/material';
import { QueryExecutionResponse } from '../types/explorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface EnhancedFinancialGridProps {
  results: QueryExecutionResponse;
  onRowClick?: (row: Record<string, any>) => void;
}

type Order = 'asc' | 'desc';

export const EnhancedFinancialGrid: React.FC<EnhancedFinancialGridProps> = ({ results, onRowClick }) => {
  const theme = useExplorerTheme();

  const [order, setOrder] = useState<Order>('asc');
  const [orderBy, setOrderBy] = useState<string>(results.columns[0]?.key || '');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(15);

  const handleRequestSort = (property: string) => {
    const isAsc = orderBy === property && order === 'asc';
    setOrder(isAsc ? 'desc' : 'asc');
    setOrderBy(property);
  };

  const sortedRows = useMemo(() => {
    if (!orderBy) return results.rows;
    const comparator = (a: any, b: any) => {
      const valA = a[orderBy];
      const valB = b[orderBy];
      if (valB < valA) return order === 'asc' ? 1 : -1;
      if (valB > valA) return order === 'asc' ? -1 : 1;
      return 0;
    };
    return [...results.rows].sort(comparator);
  }, [results.rows, order, orderBy]);

  const paginatedRows = sortedRows.slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage);

  const formatValue = (val: any, format?: string, type?: string) => {
    if (val === null || val === undefined) return '-';

    if (type === 'number' || typeof val === 'number') {
      const num = Number(val);
      if (isNaN(num)) return String(val);
      if (format === 'currency') return `$${num.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
      if (format === 'percent') return `${num.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}%`;
      return num.toLocaleString('en-US');
    }
    return String(val);
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', p: 1 }}>
      <TableContainer
        component={Paper}
        elevation={0}
        sx={{
          flexGrow: 1,
          bgcolor: theme.backgroundElevated,
          border: `1px solid ${theme.border}`,
          borderRadius: 2,
          maxHeight: '100%',
        }}
      >
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              {results.columns.map((col) => (
                <TableCell
                  key={col.key}
                  sortDirection={orderBy === col.key ? order : false}
                  sx={{
                    bgcolor: theme.background,
                    color: theme.textMuted,
                    fontWeight: 800,
                    fontSize: '0.75rem',
                    letterSpacing: 0.5,
                    borderBottom: `1px solid ${theme.border}`,
                    whiteSpace: 'nowrap',
                  }}
                >
                  <TableSortLabel
                    active={orderBy === col.key}
                    direction={orderBy === col.key ? order : 'asc'}
                    onClick={() => handleRequestSort(col.key)}
                    sx={{ '&.MuiTableSortLabel-active': { color: theme.accent } }}
                  >
                    {col.label}
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
                onClick={() => onRowClick && onRowClick(row)}
                sx={{
                  cursor: onRowClick ? 'pointer' : 'default',
                  '&:hover': { bgcolor: theme.background },
                }}
              >
                {results.columns.map((col) => {
                  const val = row[col.key];
                  const isNumber = col.type === 'number' || typeof val === 'number';
                  const isNegative = isNumber && Number(val) < 0;

                  return (
                    <TableCell
                      key={col.key}
                      sx={{
                        color: isNegative ? theme.error : theme.text,
                        fontWeight: isNumber ? 700 : 500,
                        fontFamily: isNumber ? 'monospace' : 'inherit',
                        fontSize: '0.8rem',
                        borderBottom: `1px solid ${theme.borderSubtle}`,
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {formatValue(val, col.format, col.type)}
                    </TableCell>
                  );
                })}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <TablePagination
        component="div"
        count={results.rows.length}
        page={page}
        onPageChange={(_, newPage) => setPage(newPage)}
        rowsPerPage={rowsPerPage}
        onRowsPerPageChange={(e) => {
          setRowsPerPage(parseInt(e.target.value, 10));
          setPage(0);
        }}
        rowsPerPageOptions={[15, 50, 100]}
        sx={{ color: theme.textMuted, borderTop: `1px solid ${theme.border}`, mt: 1 }}
      />
    </Box>
  );
};

export default EnhancedFinancialGrid;
