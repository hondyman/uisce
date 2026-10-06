import React from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
} from '@mui/material';
import type { QueryResultColumn } from '../types/queryDef';

/** Flat table fallback for query results (scroll-area nesting deferred). */
const ScrollAreaResultView: React.FC<{
  columns: QueryResultColumn[];
  rows: Record<string, unknown>[];
}> = ({ columns, rows }) => (
  <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 480 }}>
    <Table size="small" stickyHeader>
      <TableHead>
        <TableRow>
          {columns.map((c) => (
            <TableCell key={c.name}>{c.name}</TableCell>
          ))}
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((row, i) => (
          <TableRow key={i}>
            {columns.map((c) => (
              <TableCell key={c.name}>{String(row[c.name] ?? '')}</TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  </TableContainer>
);

export default ScrollAreaResultView;
