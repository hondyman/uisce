import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Box, Card, CardHeader, CardContent, Typography, IconButton,
  CircularProgress, Table, TableHead, TableBody, TableRow, TableCell, Chip
} from '@mui/material';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import RefreshIcon from '@mui/icons-material/Refresh';
import { PageWidgetDef } from '../PageDesignerTypes';
import { usePageEventBus } from '../PageEventBusContext';
import { GenericBOFormModal } from './GenericBOFormModal';

interface InfiniteScrollBOGridProps {
  widget: PageWidgetDef;
}

export const InfiniteScrollBOGridWidget: React.FC<InfiniteScrollBOGridProps> = ({ widget }) => {
  const { parameters, setParameter } = usePageEventBus();
  const [rows, setRows] = useState<Record<string, any>[]>([]);
  const [page, setPage] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const [loading, setLoading] = useState(false);
  const [modalRecordId, setModalRecordId] = useState<string | null>(null);

  const parentId = widget.subscribedParams.length > 0 ? parameters[widget.subscribedParams[0]] : null;
  const observerTarget = useRef<HTMLDivElement | null>(null);

  const fetchRecords = useCallback(async (pageNum: number, reset: boolean = false) => {
    if (!widget.boKey) return;
    setLoading(true);
    try {
      const limit = 30;
      const offset = pageNum * limit;
      const queryParams = new URLSearchParams({
        limit: String(limit),
        offset: String(offset),
        ...(parentId && { parentId: String(parentId) }),
      });

      const res = await fetch(`/api/v1/bo/${widget.boKey}/records?${queryParams}`);
      if (!res.ok) {
        // Fallback sample data if empty endpoint
        if (reset) {
          setRows([
            { id: 'REC-001', key: 'REC-001', status: 'SETTLED', description: 'Institutional Managed SMA Core Block', amount: 1500000 },
            { id: 'REC-002', key: 'REC-002', status: 'ACTIVE', description: 'Direct Fixed Income Tier 1 Deposit', amount: 850000 },
            { id: 'REC-003', key: 'REC-003', status: 'PENDING', description: 'Alternative Private Debt Tranche A', amount: 3200000 },
          ]);
          setHasMore(false);
        }
        return;
      }
      const data = await res.json();
      const newRows = data.records || [];

      setRows((prev) => (reset ? newRows : [...prev, ...newRows]));
      setHasMore(newRows.length === limit);
    } catch (err) {
      console.warn('Failed fetching infinite-scroll rows:', err);
    } finally {
      setLoading(false);
    }
  }, [widget.boKey, parentId]);

  // Reset when parent parameter or boKey changes
  useEffect(() => {
    setPage(0);
    setHasMore(true);
    fetchRecords(0, true);
  }, [fetchRecords]);

  // Infinite Scroll Intersection Observer with teardown guard
  useEffect(() => {
    const target = observerTarget.current;
    if (!target) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading) {
          setPage((prev) => {
            const next = prev + 1;
            fetchRecords(next, false);
            return next;
          });
        }
      },
      { threshold: 0.8 }
    );

    observer.observe(target);
    return () => {
      observer.disconnect();
    };
  }, [hasMore, loading, fetchRecords]);

  const handleRowClick = (row: Record<string, any>) => {
    if (widget.events?.onRowSelect) {
      widget.events.onRowSelect.forEach((action) => {
        if (action.actionType === 'SET_PARAMETER') {
          const val = row[action.sourcePropertyKey] ?? row.id ?? row.key;
          setParameter(action.targetChannel, val);
        }
      });
    }
  };

  const handleRowDoubleClick = (row: Record<string, any>) => {
    if (widget.events?.onRowDoubleClick) {
      widget.events.onRowDoubleClick.forEach((action) => {
        if (action.actionType === 'LAUNCH_MODAL_FORM') {
          const recId = row[action.sourcePropertyKey] ?? row.id;
          setModalRecordId(recId);
        } else if (action.actionType === 'SET_PARAMETER') {
          const val = row[action.sourcePropertyKey] ?? row.id;
          setParameter(action.targetChannel, val);
        }
      });
    } else {
      setModalRecordId(row.id || row.key);
    }
  };

  return (
    <Card sx={{ bgcolor: '#071526', border: '1px solid #1E293B', color: '#F8FAFC', display: 'flex', flexDirection: 'column', height: 380 }}>
      <CardHeader
        title={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="subtitle2" fontWeight={700} color="#00D4FF">
              {widget.title}
            </Typography>
            <Chip size="small" label={`${rows.length} loaded`} sx={{ fontSize: 10, height: 18, bgcolor: '#0B1E36', color: '#94A3B8' }} />
          </Box>
        }
        action={
          <IconButton size="small" onClick={() => fetchRecords(0, true)} sx={{ color: '#64748B' }}>
            <RefreshIcon sx={{ fontSize: 16 }} />
          </IconButton>
        }
        sx={{ borderBottom: '1px solid #1E293B', py: 1 }}
      />
      
      <CardContent sx={{ p: 0, flex: 1, overflowY: 'auto' }}>
        <Table stickyHeader size="small">
          <TableHead>
            <TableRow>
              <TableCell sx={{ bgcolor: '#0B1E36', color: '#94A3B8', fontSize: 11 }}>ID / Key</TableCell>
              <TableCell sx={{ bgcolor: '#0B1E36', color: '#94A3B8', fontSize: 11 }}>Status</TableCell>
              <TableCell sx={{ bgcolor: '#0B1E36', color: '#94A3B8', fontSize: 11 }}>Details</TableCell>
              <TableCell align="right" sx={{ bgcolor: '#0B1E36', color: '#94A3B8', fontSize: 11 }}>Inspect</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row, idx) => (
              <TableRow
                key={row.id || idx}
                hover
                onClick={() => handleRowClick(row)}
                onDoubleClick={() => handleRowDoubleClick(row)}
                sx={{ cursor: 'pointer', '&:hover': { bgcolor: 'rgba(0, 212, 255, 0.04) !important' } }}
              >
                <TableCell sx={{ color: '#F8FAFC', fontSize: 11, fontFamily: 'monospace' }}>{row.key || row.id || `REC-${idx}`}</TableCell>
                <TableCell sx={{ color: '#F8FAFC', fontSize: 11 }}>
                  <Chip
                    size="small"
                    label={row.status || 'ACTIVE'}
                    sx={{ fontSize: 10, height: 20, bgcolor: row.status === 'ACTIVE' || row.status === 'SETTLED' ? 'rgba(16,185,129,0.15)' : '#0B1E36', color: '#34D399' }}
                  />
                </TableCell>
                <TableCell sx={{ color: '#94A3B8', fontSize: 11 }}>{row.description || row.display_name || row.account_name || '—'}</TableCell>
                <TableCell align="right">
                  <IconButton size="small" onClick={(e) => { e.stopPropagation(); setModalRecordId(row.id || row.key); }} sx={{ color: '#00D4FF' }}>
                    <OpenInNewIcon sx={{ fontSize: 14 }} />
                  </IconButton>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>

        <Box ref={observerTarget} sx={{ display: 'flex', justifyContent: 'center', p: 1.5 }}>
          {loading && <CircularProgress size={18} sx={{ color: '#00D4FF' }} />}
        </Box>
      </CardContent>

      {modalRecordId && (
        <GenericBOFormModal
          boKey={widget.boKey || 'records'}
          recordId={modalRecordId}
          open={Boolean(modalRecordId)}
          onClose={() => setModalRecordId(null)}
          onSaved={() => fetchRecords(0, true)}
        />
      )}
    </Card>
  );
};
