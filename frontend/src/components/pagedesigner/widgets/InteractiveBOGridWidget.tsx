import React, { useState, useEffect, useCallback } from 'react';
import {
  Card, CardHeader, CardContent, Typography, Box, Table,
  TableHead, TableBody, TableRow, TableCell, CircularProgress, IconButton
} from '@mui/material';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import RefreshIcon from '@mui/icons-material/Refresh';
import { useNavigate } from 'react-router-dom';
import { PageWidgetDef } from '../PageDesignerTypes';
import { usePageEventBus } from '../PageEventBusContext';
import { DrillDownGridModal } from './DrillDownGridModal';

interface InteractiveBOGridWidgetProps {
  widget: PageWidgetDef;
}

export const InteractiveBOGridWidget: React.FC<InteractiveBOGridWidgetProps> = ({ widget }) => {
  const { parameters, setParameter } = usePageEventBus();
  const navigate = useNavigate();

  const [rows, setRows] = useState<Record<string, any>[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedRowId, setSelectedRowId] = useState<string | null>(null);

  // Drill-Down Modal State
  const [drillModalOpen, setDrillModalOpen] = useState(false);
  const [drillContext, setDrillContext] = useState<Record<string, any>>({});
  const [drillMetric, setDrillMetric] = useState('portfolio_xirr');

  const subscribedFilters = (widget.subscribedParams || []).reduce((acc, pKey) => {
    if (parameters[pKey] !== undefined) acc[pKey] = parameters[pKey];
    return acc;
  }, {} as Record<string, any>);

  const fetchGridData = useCallback(async () => {
    if (!widget.boKey) return;
    setLoading(true);
    try {
      const qParams = new URLSearchParams({
        limit: '50',
        offset: '0',
        ...Object.entries(subscribedFilters).reduce((a, [k, v]) => ({ ...a, [k]: String(v) }), {}),
      });
      const res = await fetch(`/api/v1/bo/${widget.boKey}/records?${qParams}`);
      const data = await res.json();
      setRows(data.records || []);
    } catch (err) {
      console.error('Grid fetch error:', err);
    } finally {
      setLoading(false);
    }
  }, [widget.boKey, JSON.stringify(subscribedFilters)]);

  useEffect(() => {
    fetchGridData();
  }, [fetchGridData]);

  // 1. Single-Click: Broadcast selection to Event Bus (Cross-Filtering)
  const handleRowClick = (row: Record<string, any>) => {
    setSelectedRowId(row.id);
    if (widget.events?.onRowSelect) {
      widget.events.onRowSelect.forEach((action) => {
        if (action.actionType === 'SET_PARAMETER') {
          const val = row[action.sourcePropertyKey] ?? row.id;
          setParameter(action.targetChannel, val);
        }
      });
    }
  };

  // 2. Double-Click: Trigger Drill-Down (Route Navigation OR Modal Inspector)
  const handleRowDoubleClick = (row: Record<string, any>) => {
    const doubleClickAction = widget.events?.onRowDoubleClick?.[0];

    if (doubleClickAction?.actionType === 'NAVIGATE' && doubleClickAction.targetPageKey) {
      navigate(`/page-designer/${doubleClickAction.targetPageKey}?parentId=${row.id}`);
    } else {
      // Default to Modal Drill-Through Inspector
      setDrillMetric(row.calculation_type || 'portfolio_xirr');
      setDrillContext({
        account_id: row.account_id || row.id,
        category: row.category,
      });
      setDrillModalOpen(true);
    }
  };

  return (
    <Card sx={{ bgcolor: '#071526', border: '1px solid #1E293B', color: '#F8FAFC', height: 380, display: 'flex', flexDirection: 'column' }}>
      <CardHeader
        title={<Typography variant="subtitle2" fontWeight={700} color="#00D4FF">{widget.title}</Typography>}
        action={
          <IconButton size="small" onClick={fetchGridData} sx={{ color: '#64748B' }}>
            <RefreshIcon sx={{ fontSize: 16 }} />
          </IconButton>
        }
        sx={{ borderBottom: '1px solid #1E293B', py: 1 }}
      />

      <CardContent sx={{ p: 0, flex: 1, overflowY: 'auto' }}>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
            <CircularProgress size={24} sx={{ color: '#00D4FF' }} />
          </Box>
        ) : (
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell sx={{ bgcolor: '#0B1E36', color: '#94A3B8', fontSize: 11 }}>Key / ID</TableCell>
                <TableCell sx={{ bgcolor: '#0B1E36', color: '#94A3B8', fontSize: 11 }}>Entity Name</TableCell>
                <TableCell align="right" sx={{ bgcolor: '#0B1E36', color: '#94A3B8', fontSize: 11 }}>Value</TableCell>
                <TableCell align="center" sx={{ bgcolor: '#0B1E36', color: '#94A3B8', fontSize: 11 }}>Drill</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {rows.map((r, i) => (
                <TableRow
                  key={r.id || i}
                  hover
                  onClick={() => handleRowClick(r)}
                  onDoubleClick={() => handleRowDoubleClick(r)}
                  sx={{
                    cursor: 'pointer',
                    bgcolor: selectedRowId === r.id ? 'rgba(0, 212, 255, 0.08)' : 'transparent',
                    '&:hover': { bgcolor: 'rgba(0, 212, 255, 0.04) !important' },
                  }}
                >
                  <TableCell sx={{ color: '#F8FAFC', fontSize: 11, fontFamily: 'monospace' }}>{r.key || r.id}</TableCell>
                  <TableCell sx={{ color: '#F8FAFC', fontSize: 11 }}>{r.name || r.display_name || '—'}</TableCell>
                  <TableCell align="right" sx={{ color: '#38BDF8', fontSize: 11, fontFamily: 'monospace' }}>
                    {typeof r.amount === 'number' ? `$${r.amount.toLocaleString()}` : r.amount || '—'}
                  </TableCell>
                  <TableCell align="center">
                    <IconButton size="small" onClick={(e) => { e.stopPropagation(); handleRowDoubleClick(r); }} sx={{ color: '#00D4FF' }}>
                      <OpenInNewIcon sx={{ fontSize: 13 }} />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>

      <DrillDownGridModal
        open={drillModalOpen}
        aggregatedField={drillMetric}
        filterContext={drillContext}
        onClose={() => setDrillModalOpen(false)}
      />
    </Card>
  );
};
