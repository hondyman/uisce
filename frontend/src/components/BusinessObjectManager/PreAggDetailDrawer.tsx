import React from 'react';
import {
  Drawer,
  Box,
  Typography,
  Chip,
  Stack,
  Button,
  Divider,
  IconButton,
} from '@mui/material';
import { Close, Refresh, Block, Code } from '@mui/icons-material';

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
  usage_count?: number;
  avg_latency_reduction_ms?: number;
  group_by?: string[];
  measures?: string[];
  filters_supported?: string[];
}

interface PreAggDetailDrawerProps {
  open: boolean;
  preagg: PreAggDescriptor | null;
  onClose: () => void;
  onRefresh: (id: string) => void;
  onDisable: (id: string) => void;
  onViewSQL: (id: string) => void;
}

export const PreAggDetailDrawer: React.FC<PreAggDetailDrawerProps> = ({
  open,
  preagg,
  onClose,
  onRefresh,
  onDisable,
  onViewSQL,
}) => {
  if (!preagg) return null;

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleString();
  };

  const formatBytes = (bytes?: number) => {
    if (bytes === undefined || bytes === null) return '-';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  return (
    <Drawer anchor="right" open={open} onClose={onClose}>
      <Box sx={{ width: 420, p: 2 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6">Pre-Aggregation Details</Typography>
          <IconButton size="small" onClick={onClose}>
            <Close />
          </IconButton>
        </Box>

        <Typography variant="subtitle2" color="text.secondary">Name</Typography>
        <Typography variant="body1" gutterBottom fontWeight="medium">
          {preagg.name}
        </Typography>

        <Typography variant="subtitle2" color="text.secondary">Business Object</Typography>
        <Typography variant="body2" gutterBottom>
          {preagg.bo_name}
        </Typography>

        <Typography variant="subtitle2" color="text.secondary">Target</Typography>
        <Typography variant="body2" gutterBottom>
          {preagg.target_database}.{preagg.target_name}
        </Typography>

        <Typography variant="subtitle2" color="text.secondary">Status</Typography>
        <Chip
          label={preagg.lifecycle_status || 'idle'}
          color={
            preagg.lifecycle_status === 'active' ? 'success' :
            preagg.lifecycle_status === 'failed' ? 'error' :
            preagg.lifecycle_status === 'stale' ? 'warning' : 'default'
          }
          size="small"
          sx={{ mb: 2 }}
        />

        <Divider sx={{ my: 2 }} />

        <Typography variant="subtitle2" color="text.secondary" gutterBottom>
          Group By
        </Typography>
        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ mb: 2, gap: 0.5 }}>
          {preagg.group_by?.length ? (
            preagg.group_by.map((g) => (
              <Chip key={g} label={g} size="small" variant="outlined" />
            ))
          ) : (
            <Typography variant="caption" color="text.secondary">None</Typography>
          )}
        </Stack>

        <Typography variant="subtitle2" color="text.secondary" gutterBottom>
          Measures
        </Typography>
        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ mb: 2, gap: 0.5 }}>
          {preagg.measures?.length ? (
            preagg.measures.map((m) => (
              <Chip key={m} label={m} size="small" variant="outlined" color="primary" />
            ))
          ) : (
            <Typography variant="caption" color="text.secondary">None</Typography>
          )}
        </Stack>

        <Typography variant="subtitle2" color="text.secondary" gutterBottom>
          Supported Filters
        </Typography>
        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ mb: 2, gap: 0.5 }}>
          {preagg.filters_supported?.length ? (
            preagg.filters_supported.map((f) => (
              <Chip key={f} label={f} size="small" variant="outlined" color="secondary" />
            ))
          ) : (
            <Typography variant="caption" color="text.secondary">Any</Typography>
          )}
        </Stack>

        <Divider sx={{ my: 2 }} />

        <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, mb: 2 }}>
          <Box>
            <Typography variant="subtitle2" color="text.secondary">Usage</Typography>
            <Typography variant="body1" fontWeight="medium">
              {preagg.usage_count?.toLocaleString() ?? 0} queries
            </Typography>
          </Box>
          <Box>
            <Typography variant="subtitle2" color="text.secondary">Latency Saved</Typography>
            <Typography variant="body1" fontWeight="medium">
              {preagg.avg_latency_reduction_ms
                ? `${preagg.avg_latency_reduction_ms.toFixed(0)} ms`
                : '- ms'
              }
            </Typography>
          </Box>
          <Box>
            <Typography variant="subtitle2" color="text.secondary">Rows</Typography>
            <Typography variant="body1">
              {preagg.row_count?.toLocaleString() ?? '-'}
            </Typography>
          </Box>
          <Box>
            <Typography variant="subtitle2" color="text.secondary">Size</Typography>
            <Typography variant="body1">
              {formatBytes(preagg.size_bytes)}
            </Typography>
          </Box>
        </Box>

        <Divider sx={{ my: 2 }} />

        <Typography variant="subtitle2" color="text.secondary">Last Refreshed</Typography>
        <Typography variant="body2" gutterBottom>
          {formatDate(preagg.last_refreshed_at)}
        </Typography>

        <Typography variant="subtitle2" color="text.secondary">Next Scheduled</Typography>
        <Typography variant="body2" gutterBottom>
          {formatDate(preagg.next_scheduled_refresh)}
        </Typography>

        <Typography variant="subtitle2" color="text.secondary">Refresh Interval</Typography>
        <Typography variant="body2" gutterBottom>
          {preagg.refresh_interval_minutes} minutes ({preagg.refresh_strategy})
        </Typography>

        {preagg.last_refresh_error && (
          <Box sx={{ mt: 2, p: 1, bgcolor: 'error.light', borderRadius: 1 }}>
            <Typography variant="caption" color="error.contrastText">
              Last Error: {preagg.last_refresh_error}
            </Typography>
          </Box>
        )}

        <Divider sx={{ my: 2 }} />

        <Stack direction="row" spacing={1}>
          <Button
            variant="contained"
            size="small"
            startIcon={<Refresh />}
            onClick={() => onRefresh(preagg.id)}
          >
            Refresh Now
          </Button>
          <Button
            variant="outlined"
            size="small"
            color="warning"
            startIcon={<Block />}
            onClick={() => onDisable(preagg.id)}
          >
            Disable
          </Button>
          <Button
            variant="text"
            size="small"
            startIcon={<Code />}
            onClick={() => onViewSQL(preagg.id)}
          >
            View SQL
          </Button>
        </Stack>
      </Box>
    </Drawer>
  );
};

export default PreAggDetailDrawer;
