import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  CircularProgress,
  IconButton,
  Chip,
  Stack,
  Tooltip,
} from '@mui/material';
import {
  Close as CloseIcon,
  Refresh as RefreshIcon,
  Download as DownloadIcon,
  Security as SecurityIcon,
  TableChart as TableIcon,
  ArrowDownward as ArrowDownRightIcon,
} from '@mui/icons-material';

export interface DrillDownProps {
  isOpen: boolean;
  tenantId: string;
  aggregatedField: string;
  filterContext: Record<string, any>;
  onClose: () => void;
}

export const DrillDownGridModal: React.FC<DrillDownProps> = ({
  isOpen,
  tenantId,
  aggregatedField,
  filterContext,
  onClose,
}) => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<{ granularBoKey: string; columns: string[]; rows: any[]; totalCount: number } | null>(null);

  useEffect(() => {
    if (isOpen && aggregatedField) {
      fetchDrillData();
    } else {
      setData(null);
    }
  }, [isOpen, aggregatedField, filterContext]);

  const fetchDrillData = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/query/drill-down', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId || '00000000-0000-0000-0000-000000000001',
        },
        body: JSON.stringify({
          tenantId: tenantId || '00000000-0000-0000-0000-000000000001',
          aggregatedField,
          filterContext: filterContext || {},
          pageSize: 50,
          offset: 0,
        }),
      });
      if (res.ok) {
        const json = await res.json();
        setData(json);
      }
    } catch (e) {
      console.error('Failed fetching drill-down details:', e);
    } finally {
      setLoading(false);
    }
  };

  const handleExportCSV = () => {
    if (!data || !data.rows.length) return;
    const headers = data.columns.join(',');
    const rows = data.rows.map(r => data.columns.map(c => JSON.stringify(r[c] ?? '')).join(',')).join('\n');
    const csvContent = "data:text/csv;charset=utf-8," + headers + "\n" + rows;
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement("a");
    link.setAttribute("href", encodedUri);
    link.setAttribute("download", `drill_down_${aggregatedField}_${new Date().toISOString().slice(0,10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
      maxWidth="lg"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: 2.5,
          boxShadow: 24,
        }
      }}
    >
      {/* Header */}
      <DialogTitle sx={{ m: 0, p: 2.5, bgcolor: 'background.paper', borderBottom: '1px solid', borderColor: 'divider' }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Box sx={{ p: 1, bgcolor: 'primary.light', borderRadius: 1.5, color: 'primary.contrastText', display: 'flex' }}>
              <ArrowDownRightIcon fontSize="small" />
            </Box>
            <Box>
              <Stack direction="row" spacing={1} alignItems="center">
                <Typography variant="h6" fontWeight={700}>
                  Drill-Down Details:
                </Typography>
                <Chip label={aggregatedField} color="primary" size="small" sx={{ fontWeight: 700, fontFamily: 'monospace' }} />
              </Stack>
              <Typography variant="caption" color="text.secondary">
                Constituent transactional records and cash flow legs backing the aggregate calculation.
              </Typography>
            </Box>
          </Stack>
          <Stack direction="row" spacing={1} alignItems="center">
            <Tooltip title="Refresh Data">
              <IconButton size="small" onClick={fetchDrillData} disabled={loading}>
                <RefreshIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <IconButton size="small" onClick={onClose}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </Stack>
      </DialogTitle>

      {/* Body */}
      <DialogContent sx={{ p: 3, minHeight: 320 }}>
        {loading ? (
          <Box sx={{ py: 8, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 2 }}>
            <CircularProgress size={36} color="primary" />
            <Typography variant="body2" color="text.secondary">
              Querying lakehouse fact tables for granular constituent legs...
            </Typography>
          </Box>
        ) : data ? (
          <Stack spacing={2.5}>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Stack direction="row" spacing={1} alignItems="center">
                <TableIcon sx={{ fontSize: 18, color: 'text.secondary' }} />
                <Typography variant="body2" color="text.secondary">
                  Target Business Object: <strong style={{ color: '#1976d2' }}>{data.granularBoKey}</strong>
                </Typography>
              </Stack>
              <Chip label={`Total Granular Rows: ${data.totalCount}`} size="small" variant="outlined" />
            </Stack>

            <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 420, borderRadius: 2 }}>
              <Table size="small" stickyHeader>
                <TableHead sx={{ bgcolor: 'action.hover' }}>
                  <TableRow>
                    {data.columns.map((col) => (
                      <TableCell key={col} sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.72rem', bgcolor: 'action.hover' }}>
                        {col}
                      </TableCell>
                    ))}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {data.rows.map((row, idx) => (
                    <TableRow key={idx} hover>
                      {data.columns.map((col) => (
                        <TableCell key={col} sx={{ fontSize: '0.78rem', fontFamily: 'monospace' }}>
                          {row[col] !== undefined ? String(row[col]) : '—'}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Stack>
        ) : (
          <Box sx={{ py: 6, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">
              No granular drill-through records found for this filter context.
            </Typography>
          </Box>
        )}
      </DialogContent>

      {/* Footer */}
      <DialogActions sx={{ px: 3, py: 2, bgcolor: 'background.paper', borderTop: '1px solid', borderColor: 'divider', justifyContent: 'space-between' }}>
        <Stack direction="row" spacing={1} alignItems="center">
          <SecurityIcon sx={{ fontSize: 18, color: 'success.main' }} />
          <Typography variant="caption" color="text.secondary">
            Audit Traceability Active (SEC Rule 17a-4 Compliant)
          </Typography>
        </Stack>
        <Stack direction="row" spacing={1.5}>
          <Button
            variant="outlined"
            startIcon={<DownloadIcon />}
            onClick={handleExportCSV}
            disabled={!data || !data.rows.length}
            size="small"
          >
            Export Details (CSV)
          </Button>
          <Button variant="contained" onClick={onClose} size="small">
            Close
          </Button>
        </Stack>
      </DialogActions>
    </Dialog>
  );
};
