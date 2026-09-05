import React, { useMemo, useState } from 'react';
import {
  Box,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Tooltip,
} from '@mui/material';
import { KeyboardArrowDown as ExpandIcon, KeyboardArrowRight as CollapseIcon } from '@mui/icons-material';
import type { QueryResultColumn } from '../types/queryDef';

interface ScrollAreaResultViewProps {
  columns: QueryResultColumn[];
  rows: Record<string, unknown>[];
}

const cell = (row: Record<string, unknown>, name: string) => {
  const v = row[name];
  return typeof v === 'object' && v !== null ? JSON.stringify(v) : String(v ?? '-');
};

/**
 * Renders query results PeopleSoft-scroll-style: columns from a 1:M/M:M
 * related BO ("many" cardinality) are never flattened into the primary row
 * — that would either duplicate the primary row's data once per child, or
 * silently drop children beyond the first. Instead, rows are grouped by
 * their "one"-cardinality (primary + lookup) column values, and the "many"
 * columns render as an expandable nested grid under each parent row, one
 * level per PeopleSoft would call a child scroll.
 */
const ScrollAreaResultView: React.FC<ScrollAreaResultViewProps> = ({ columns, rows }) => {
  const oneColumns = columns.filter((c) => c.cardinality !== 'many');
  const manyColumns = columns.filter((c) => c.cardinality === 'many');

  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const groups = useMemo(() => {
    if (manyColumns.length === 0) return null;
    const map = new Map<string, { parent: Record<string, unknown>; children: Record<string, unknown>[] }>();
    for (const row of rows) {
      const key = JSON.stringify(oneColumns.map((c) => row[c.name]));
      if (!map.has(key)) {
        map.set(key, { parent: row, children: [] });
      }
      const hasChildData = manyColumns.some((c) => row[c.name] !== null && row[c.name] !== undefined);
      if (hasChildData) {
        map.get(key)!.children.push(row);
      }
    }
    return Array.from(map.values());
  }, [rows, oneColumns, manyColumns]);

  if (!groups) {
    // No many-cardinality columns selected — plain flat table.
    return (
      <TableContainer sx={{ height: '100%' }}>
        <Table stickyHeader size="small">
          <TableHead>
            <TableRow>
              {columns.map((col) => (
                <TableCell key={col.name} sx={{ fontWeight: 'bold', bgcolor: '#f9f9f9' }}>
                  {col.name}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row, i) => (
              <TableRow key={i} hover>
                {columns.map((col) => (
                  <TableCell key={col.name}>{cell(row, col.name)}</TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    );
  }

  const toggle = (key: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  return (
    <TableContainer sx={{ height: '100%' }}>
      <Table stickyHeader size="small">
        <TableHead>
          <TableRow>
            <TableCell sx={{ width: 32, bgcolor: '#f9f9f9' }} />
            {oneColumns.map((col) => (
              <TableCell key={col.name} sx={{ fontWeight: 'bold', bgcolor: '#f9f9f9' }}>
                {col.name}
              </TableCell>
            ))}
            <TableCell sx={{ fontWeight: 'bold', bgcolor: '#f9f9f9' }}>
              <Tooltip title={`Child data (${manyColumns.map((c) => c.name).join(', ')}) — one-to-many, shown per row below`}>
                <span>Related ({manyColumns.length})</span>
              </Tooltip>
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {groups.map((group, gi) => {
            const key = String(gi);
            const isOpen = expanded.has(key);
            return (
              <React.Fragment key={key}>
                <TableRow hover>
                  <TableCell>
                    <IconButton size="small" onClick={() => toggle(key)} disabled={group.children.length === 0}>
                      {isOpen ? <ExpandIcon fontSize="small" /> : <CollapseIcon fontSize="small" />}
                    </IconButton>
                  </TableCell>
                  {oneColumns.map((col) => (
                    <TableCell key={col.name}>{cell(group.parent, col.name)}</TableCell>
                  ))}
                  <TableCell>
                    <Chip size="small" label={`${group.children.length} row${group.children.length === 1 ? '' : 's'}`} />
                  </TableCell>
                </TableRow>
                {isOpen && group.children.length > 0 && (
                  <TableRow>
                    <TableCell />
                    <TableCell colSpan={oneColumns.length + 1} sx={{ p: 0, bgcolor: '#fafafa' }}>
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            {manyColumns.map((col) => (
                              <TableCell key={col.name} sx={{ fontSize: '0.75rem', color: 'text.secondary' }}>
                                {col.name}
                              </TableCell>
                            ))}
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {group.children.map((child, ci) => (
                            <TableRow key={ci}>
                              {manyColumns.map((col) => (
                                <TableCell key={col.name}>{cell(child, col.name)}</TableCell>
                              ))}
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </TableCell>
                  </TableRow>
                )}
              </React.Fragment>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default ScrollAreaResultView;
