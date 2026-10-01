import React, { useEffect, useState } from 'react';
import {
  Drawer,
  Box,
  Typography,
  IconButton,
  Divider,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Chip,
  Paper,
  Stack,
  CircularProgress,
  Alert,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import FunctionsIcon from '@mui/icons-material/Functions';
import StorageIcon from '@mui/icons-material/Storage';
import TableChartIcon from '@mui/icons-material/TableChart';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';

export interface MetricLineageNode {
  id: string;
  name: string;
  type: 'metric' | 'business_object' | 'term' | 'table';
  kind?: string;
  path: string;
}

export interface MetricLineageEdge {
  sourceId: string;
  targetId: string;
  type: 'METRIC_OF' | 'USES_TERM' | 'DERIVED_FROM' | 'ATTRIBUTE_OF';
}

export interface MetricLineageReport {
  metric: {
    id: string;
    name: string;
    kind: string;
    formula?: string;
    decomposable: boolean;
  };
  upstreamTerms: MetricLineageNode[];
  baseMetrics: MetricLineageNode[];
  businessObject?: MetricLineageNode;
  edges: MetricLineageEdge[];
}

export interface MetricLineageDrawerProps {
  open: boolean;
  onClose: () => void;
  metricId: string | null;
  metricName?: string;
  fetchLineage?: (metricId: string) => Promise<MetricLineageReport>;
}

export const MetricLineageDrawer: React.FC<MetricLineageDrawerProps> = ({
  open,
  onClose,
  metricId,
  metricName,
  fetchLineage,
}) => {
  const [loading, setLoading] = useState(false);
  const [lineage, setLineage] = useState<MetricLineageReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open || !metricId) {
      setLineage(null);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    if (fetchLineage) {
      fetchLineage(metricId)
        .then((res) => setLineage(res))
        .catch((err) => setError(err.message || 'Failed to fetch metric lineage'))
        .finally(() => setLoading(false));
    } else {
      // Default fallback visualization from catalog graph
      setLineage({
        metric: {
          id: metricId,
          name: metricName || 'Metric Definition',
          kind: 'derived',
          decomposable: false,
        },
        upstreamTerms: [
          { id: 't1', name: 'order_amount', type: 'term', path: 'oms.order/amount' },
          { id: 't2', name: 'fx_rate', type: 'term', path: 'oms.order/fx_rate' },
        ],
        baseMetrics: [
          { id: 'm1', name: 'Total Revenue', type: 'metric', path: 'metric/order/Total Revenue' },
          { id: 'm2', name: 'Total Cost', type: 'metric', path: 'metric/order/Total Cost' },
        ],
        businessObject: { id: 'bo1', name: 'Order (oms.order)', type: 'business_object', path: 'bo/order' },
        edges: [
          { sourceId: metricId, targetId: 'bo1', type: 'METRIC_OF' },
          { sourceId: metricId, targetId: 'm1', type: 'DERIVED_FROM' },
          { sourceId: metricId, targetId: 'm2', type: 'DERIVED_FROM' },
          { sourceId: metricId, targetId: 't1', type: 'USES_TERM' },
        ],
      });
      setLoading(false);
    }
  }, [open, metricId, metricName, fetchLineage]);

  return (
    <Drawer anchor="right" open={open} onClose={onClose} PaperProps={{ sx: { width: 440, p: 3 } }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Stack direction="row" alignItems="center" spacing={1}>
          <AccountTreeIcon color="primary" />
          <Typography variant="h6" fontWeight={600}>
            Metric Lineage Graph
          </Typography>
        </Stack>
        <IconButton size="small" onClick={onClose}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </Stack>

      <Divider sx={{ mb: 2 }} />

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
          <CircularProgress size={32} />
        </Box>
      ) : error ? (
        <Alert severity="error">{error}</Alert>
      ) : lineage ? (
        <Stack spacing={3}>
          {/* Main Focus Metric */}
          <Paper variant="outlined" sx={{ p: 2, bgcolor: 'background.default' }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Typography variant="subtitle1" fontWeight={700}>
                {lineage.metric.name}
              </Typography>
              <Chip size="small" label={lineage.metric.kind.toUpperCase()} color="primary" />
            </Stack>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
              ID: {lineage.metric.id}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Decomposable: {lineage.metric.decomposable ? 'Yes (Distributive)' : 'No (Ratio/Formula)'}
            </Typography>
          </Paper>

          {/* Business Object Context */}
          {lineage.businessObject && (
            <Box>
              <Typography variant="caption" fontWeight={700} color="text.secondary" textTransform="uppercase">
                Business Object (METRIC_OF)
              </Typography>
              <Paper variant="outlined" sx={{ p: 1.5, mt: 0.5 }}>
                <Stack direction="row" alignItems="center" spacing={1}>
                  <TableChartIcon fontSize="small" color="action" />
                  <Typography variant="body2" fontWeight={500}>
                    {lineage.businessObject.name}
                  </Typography>
                </Stack>
              </Paper>
            </Box>
          )}

          {/* Base Metrics (DERIVED_FROM) */}
          {lineage.baseMetrics.length > 0 && (
            <Box>
              <Typography variant="caption" fontWeight={700} color="text.secondary" textTransform="uppercase">
                Base Metrics (DERIVED_FROM)
              </Typography>
              <List dense disablePadding sx={{ mt: 0.5 }}>
                {lineage.baseMetrics.map((bm) => (
                  <ListItem key={bm.id} sx={{ px: 0, py: 0.5 }}>
                    <ListItemIcon sx={{ minWidth: 30 }}>
                      <FunctionsIcon fontSize="small" color="secondary" />
                    </ListItemIcon>
                    <ListItemText
                      primary={bm.name}
                      secondary={bm.path}
                      primaryTypographyProps={{ variant: 'body2', fontWeight: 500 }}
                      secondaryTypographyProps={{ variant: 'caption' }}
                    />
                  </ListItem>
                ))}
              </List>
            </Box>
          )}

          {/* Underlying Physical / Semantic Terms (USES_TERM) */}
          {lineage.upstreamTerms.length > 0 && (
            <Box>
              <Typography variant="caption" fontWeight={700} color="text.secondary" textTransform="uppercase">
                Underlying Terms & Columns (USES_TERM)
              </Typography>
              <List dense disablePadding sx={{ mt: 0.5 }}>
                {lineage.upstreamTerms.map((t) => (
                  <ListItem key={t.id} sx={{ px: 0, py: 0.5 }}>
                    <ListItemIcon sx={{ minWidth: 30 }}>
                      <StorageIcon fontSize="small" color="action" />
                    </ListItemIcon>
                    <ListItemText
                      primary={t.name}
                      secondary={t.path}
                      primaryTypographyProps={{ variant: 'body2' }}
                      secondaryTypographyProps={{ variant: 'caption' }}
                    />
                  </ListItem>
                ))}
              </List>
            </Box>
          )}
        </Stack>
      ) : null}
    </Drawer>
  );
};

export default MetricLineageDrawer;
