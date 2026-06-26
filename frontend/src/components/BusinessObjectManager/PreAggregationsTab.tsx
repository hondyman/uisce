import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Tooltip,
  CircularProgress,
  Button,
  Menu,
  MenuItem,
} from '@mui/material';
import {
  Refresh,
  Build,
  Code,
  MoreVert,
  AccountTree,
  CheckCircle,
  Error,
  Warning,
  HourglassEmpty,
  CloudSync,
  Pause,
} from '@mui/icons-material';

interface PreAggDescriptor {
  id: string;
  name: string;
  bo_name: string;
  tenant_id: string;
  target_database: string;
  target_name: string;
  refresh_strategy: string;
  refresh_interval_minutes: number;
  lifecycle_status: string;
  last_materialized_at?: string;
  last_refreshed_at?: string;
  last_refresh_status?: string;
  last_refresh_error?: string;
  next_scheduled_refresh?: string;
  row_count?: number;
  size_bytes?: number;
}

interface PreAggregationsTabProps {
  boName: string;
  tenantId: string;
  onCreateNew: () => void;
}

const statusConfig: Record<string, { color: 'success' | 'warning' | 'error' | 'info' | 'default'; icon: React.ReactElement; label: string }> = {
  active: { color: 'success', icon: <CheckCircle fontSize="small" />, label: 'Active' },
  idle: { color: 'default', icon: <Pause fontSize="small" />, label: 'Idle' },
  materializing: { color: 'info', icon: <CloudSync fontSize="small" />, label: 'Materializing' },
  refreshing: { color: 'info', icon: <CloudSync fontSize="small" />, label: 'Refreshing' },
  stale: { color: 'warning', icon: <Warning fontSize="small" />, label: 'Stale' },
  failed: { color: 'error', icon: <Error fontSize="small" />, label: 'Failed' },
};

export const PreAggregationsTab: React.FC<PreAggregationsTabProps> = ({
  boName,
  tenantId,
  onCreateNew,
}) => {
  const [preAggs, setPreAggs] = useState<PreAggDescriptor[]>([]);
  const [loading, setLoading] = useState(true);
  const [menuAnchor, setMenuAnchor] = useState<{ el: HTMLElement; id: string } | null>(null);

  useEffect(() => {
    fetchPreAggs();
  }, [boName, tenantId]);

  const fetchPreAggs = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/pre-aggregations?bo_name=${encodeURIComponent(boName)}&tenant_id=${encodeURIComponent(tenantId)}`);
      if (res.ok) {
        const data = await res.json();
        setPreAggs(data || []);
      }
    } catch (e) {
      console.error('Failed to fetch pre-aggregations', e);
    } finally {
      setLoading(false);
    }
  };

  const handleRefresh = async (id: string) => {
    try {
      await fetch(`/api/pre-aggregations/${id}/refresh`, { method: 'POST' });
      fetchPreAggs();
    } catch (e) {
      console.error('Refresh failed', e);
    }
  };

  const handleRebuild = async (id: string) => {
    try {
      await fetch(`/api/pre-aggregations/${id}/materialize`, { method: 'POST' });
      fetchPreAggs();
    } catch (e) {
      console.error('Rebuild failed', e);
    }
  };

  const formatBytes = (bytes?: number) => {
    if (bytes === undefined || bytes === null) return '-';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleString();
  };

  const getStatusChip = (status: string) => {
    const cfg = statusConfig[status] || statusConfig.idle;
    return (
      <Chip
        icon={cfg.icon}
        label={cfg.label}
        color={cfg.color}
        size="small"
        variant="outlined"
      />
    );
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" py={4}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h6">Pre-Aggregations</Typography>
        <Box>
          <Button onClick={fetchPreAggs} startIcon={<Refresh />} sx={{ mr: 1 }}>
            Refresh
          </Button>
          <Button variant="contained" onClick={onCreateNew}>
            Create Pre-Aggregation
          </Button>
        </Box>
      </Box>

      {preAggs.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: 'center' }}>
          <HourglassEmpty sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
          <Typography color="text.secondary">No pre-aggregations defined for this BO.</Typography>
          <Button variant="outlined" onClick={onCreateNew} sx={{ mt: 2 }}>
            Create First Pre-Aggregation
          </Button>
        </Paper>
      ) : (
        <TableContainer component={Paper}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Last Refreshed</TableCell>
                <TableCell>Next Refresh</TableCell>
                <TableCell align="right">Rows</TableCell>
                <TableCell align="right">Size</TableCell>
                <TableCell align="center">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {preAggs.map((pa) => (
                <TableRow key={pa.id} hover>
                  <TableCell>
                    <Typography variant="body2" fontWeight="medium">
                      {pa.name}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {pa.target_database}.{pa.target_name}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {getStatusChip(pa.lifecycle_status || 'idle')}
                    {pa.last_refresh_error && (
                      <Tooltip title={pa.last_refresh_error}>
                        <Error fontSize="small" color="error" sx={{ ml: 1 }} />
                      </Tooltip>
                    )}
                  </TableCell>
                  <TableCell>{formatDate(pa.last_refreshed_at)}</TableCell>
                  <TableCell>{formatDate(pa.next_scheduled_refresh)}</TableCell>
                  <TableCell align="right">
                    {pa.row_count?.toLocaleString() || '-'}
                  </TableCell>
                  <TableCell align="right">{formatBytes(pa.size_bytes)}</TableCell>
                  <TableCell align="center">
                    <Tooltip title="Refresh Now">
                      <IconButton size="small" onClick={() => handleRefresh(pa.id)}>
                        <Refresh fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Rebuild">
                      <IconButton size="small" onClick={() => handleRebuild(pa.id)}>
                        <Build fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <IconButton
                      size="small"
                      onClick={(e) => setMenuAnchor({ el: e.currentTarget, id: pa.id })}
                    >
                      <MoreVert fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Menu
        anchorEl={menuAnchor?.el}
        open={!!menuAnchor}
        onClose={() => setMenuAnchor(null)}
      >
        <MenuItem onClick={() => { window.open(`/api/pre-aggregations/${menuAnchor?.id}/ddl`, '_blank'); setMenuAnchor(null); }}>
          <Code fontSize="small" sx={{ mr: 1 }} /> View DDL
        </MenuItem>
        <MenuItem onClick={() => setMenuAnchor(null)}>
          <AccountTree fontSize="small" sx={{ mr: 1 }} /> View Lineage
        </MenuItem>
      </Menu>
    </Box>
  );
};

export default PreAggregationsTab;
