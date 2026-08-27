import React, { useState } from 'react';
import { Box, Typography, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from '@mui/material';
import { 
  ChevronRight, 
  ChevronDown, 
  Layers, 
  Folder, 
  FolderOpen
} from 'lucide-react';

export interface GroupNodeModel {
  nodeId: string;
  levelIndex: number;
  groupKey: string;
  groupValue: string;
  depth: number;
  itemCount: number;
  aggregations: Record<string, number>;
  children?: GroupNodeModel[];
  leafRecords?: Record<string, any>[];
  isCollapsed?: boolean;
}

export interface ColumnConfig {
  key: string;
  header: string;
  width?: number | string;
  align?: 'left' | 'right' | 'center';
  isCurrency?: boolean;
  isPercent?: boolean;
}

interface HierarchicalReportGridProps {
  rootNodes: GroupNodeModel[];
  grandTotals: Record<string, number>;
  columns: ColumnConfig[];
  subtotalKeys: { sumMarketVal: string; weightedCoupon: string };
}

export const HierarchicalReportGrid: React.FC<HierarchicalReportGridProps> = ({
  rootNodes,
  grandTotals,
  columns,
  subtotalKeys,
}) => {
  const [collapsedMap, setCollapsedMap] = useState<Record<string, boolean>>({});

  const toggleCollapse = (nodeId: string) => {
    setCollapsedMap((prev) => ({ ...prev, [nodeId]: !prev[nodeId] }));
  };

  const renderNode = (node: GroupNodeModel): React.ReactNode => {
    const isCollapsed = collapsedMap[node.nodeId] ?? node.isCollapsed ?? false;
    const paddingLeftPx = node.depth * 20 + 8;

    return (
      <React.Fragment key={node.nodeId}>
        <TableRow sx={{ bgcolor: '#071526', borderTop: '1px solid', borderBottom: '1px solid rgba(51, 65, 85, 0.8)', fontFamily: 'monospace', fontSize: '0.75rem', fontWeight: 600, color: '#e2e8f0' }}>
          <TableCell colSpan={2} sx={{ py: 1.5, px: 2, pl: `${paddingLeftPx}px` }}>
            <Box
              component="button"
              onClick={() => toggleCollapse(node.nodeId)}
              sx={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 0.5,
                color: '#00D4FF',
                border: 'none',
                background: 'none',
                cursor: 'pointer',
                padding: 0,
                '&:hover': { color: '#67e8f9' },
                transition: 'color 0.2s',
              }}
            >
              {isCollapsed ? <ChevronRight size={16} /> : <ChevronDown size={16} />}
              {isCollapsed ? <Folder size={14} /> : <FolderOpen size={14} />}
              <Box component="span" sx={{ textTransform: 'uppercase', letterSpacing: '0.025em', fontSize: '0.6875rem' }}>
                {node.groupKey}: <Box component="span" sx={{ color: '#fff', fontWeight: 700 }}>{node.groupValue}</Box>
              </Box>
              <Box component="span" sx={{ bgcolor: '#1e293b', color: '#94a3b8', px: 1, py: 0.25, borderRadius: 0.25, fontSize: '0.625rem', fontWeight: 400, ml: 1 }}>
                {node.itemCount} items
              </Box>
            </Box>
          </TableCell>

          <TableCell sx={{ py: 1.5, px: 2, textAlign: 'right', color: '#34d399', fontWeight: 700 }}>
            ${(node.aggregations[subtotalKeys.sumMarketVal] || 0).toLocaleString(undefined, { minimumFractionDigits: 2 })}
          </TableCell>
          <TableCell sx={{ py: 1.5, px: 2, textAlign: 'right', color: '#fbbf24' }}>
            {((node.aggregations[subtotalKeys.weightedCoupon] || 0) * 100).toFixed(2)}%
          </TableCell>
          <TableCell sx={{ py: 1.5, px: 2 }}></TableCell>
        </TableRow>

        {!isCollapsed && (
          <>
            {node.children && node.children.map((child) => renderNode(child))}

            {node.leafRecords && node.leafRecords.map((row, rIdx) => (
              <TableRow key={rIdx} sx={{ '&:hover': { bgcolor: 'rgba(30, 41, 59, 0.2)' }, borderBottom: '1px solid rgba(30, 41, 59, 0.4)', fontFamily: 'monospace', fontSize: '0.6875rem', color: '#cbd5e1', transition: 'background-color 0.2s' }}>
                <TableCell sx={{ py: 0.75, px: 2, pl: `${paddingLeftPx + 24}px`, color: '#f1f5f9', fontFamily: 'sans-serif' }}>
                  {row.security_name || row.account_name || '---'}
                </TableCell>
                <TableCell sx={{ py: 0.75, px: 2, color: '#94a3b8' }}>{row.cusip || row.account_bk || '---'}</TableCell>
                <TableCell sx={{ py: 0.75, px: 2, textAlign: 'right', color: '#f1f5f9' }}>
                  ${(row.market_value || 0).toLocaleString(undefined, { minimumFractionDigits: 2 })}
                </TableCell>
                <TableCell sx={{ py: 0.75, px: 2, textAlign: 'right', color: '#cbd5e1' }}>
                  {((row.coupon_rate || 0) * 100).toFixed(2)}%
                </TableCell>
                <TableCell sx={{ py: 0.75, px: 2, textAlign: 'center', color: '#94a3b8', fontSize: '0.625rem' }}>
                  {row.rating || '---'}
                </TableCell>
              </TableRow>
            ))}

            <TableRow sx={{ bgcolor: 'rgba(11, 30, 54, 0.4)', borderBottom: '1px solid rgba(51, 65, 85, 0.6)', fontFamily: 'monospace', fontSize: '0.625rem', color: '#94a3b8' }}>
              <TableCell sx={{ py: 0.75, px: 2, pl: `${paddingLeftPx}px`, fontStyle: 'italic' }}>
                Total {node.groupValue} ({node.itemCount} records)
              </TableCell>
              <TableCell sx={{ py: 0.75, px: 2 }}></TableCell>
              <TableCell sx={{ py: 0.75, px: 2, textAlign: 'right', color: '#6ee7b7', fontWeight: 600 }}>
                ${(node.aggregations[subtotalKeys.sumMarketVal] || 0).toLocaleString(undefined, { minimumFractionDigits: 2 })}
              </TableCell>
              <TableCell sx={{ py: 0.75, px: 2, textAlign: 'right', color: '#fcd34d' }}>
                WAvg: {((node.aggregations[subtotalKeys.weightedCoupon] || 0) * 100).toFixed(2)}%
              </TableCell>
              <TableCell sx={{ py: 0.75, px: 2 }}></TableCell>
            </TableRow>
          </>
        )}
      </React.Fragment>
    );
  };

  return (
    <Box sx={{ width: '100%', border: '1px solid #1e293b', borderRadius: 2, overflow: 'hidden', bgcolor: '#050D1A', fontFamily: 'sans-serif' }}>
      <Box sx={{ bgcolor: '#071526', px: 2, py: 1.5, borderBottom: '1px solid #1e293b', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#e2e8f0', fontSize: '0.75rem', fontWeight: 600 }}>
          <Layers size={16} style={{ color: '#00D4FF' }} />
          <span>Multi-Level Hierarchical Portfolio Rollup</span>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, fontSize: '0.75rem', fontFamily: 'monospace' }}>
          <span style={{ color: '#94a3b8' }}>
            Grand Total Market Value:{' '}
            <strong style={{ color: '#34d399' }}>
              ${(grandTotals[subtotalKeys.sumMarketVal] || 0).toLocaleString(undefined, { minimumFractionDigits: 2 })}
            </strong>
          </span>
          <span style={{ color: '#94a3b8' }}>
            Weighted Average Coupon:{' '}
            <strong style={{ color: '#fbbf24' }}>
              {((grandTotals[subtotalKeys.weightedCoupon] || 0) * 100).toFixed(2)}%
            </strong>
          </span>
        </Box>
      </Box>

      <TableContainer sx={{ overflowX: 'auto' }}>
        <Table sx={{ width: '100%', textAlign: 'left', borderCollapse: 'collapse' }}>
          <TableHead>
            <TableRow sx={{ bgcolor: 'rgba(7, 21, 38, 0.8)', color: '#94a3b8', fontFamily: 'monospace', fontSize: '0.625rem', textTransform: 'uppercase', borderBottom: '1px solid #1e293b' }}>
              {columns.map((col) => (
                <TableCell
                  key={col.key}
                  sx={{ width: col.width, py: 1, px: 2, textAlign: col.align === 'right' ? 'right' : 'left' }}
                >
                  {col.header}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {rootNodes.map((root) => renderNode(root))}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};

export default HierarchicalReportGrid;
