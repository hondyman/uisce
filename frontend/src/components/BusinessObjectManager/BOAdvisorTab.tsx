import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  CardActions,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  CircularProgress,
  Divider,
  Alert,
} from '@mui/material';
import {
  TrendingUp,
  Speed,
  Storage,
  Bolt,
  CheckCircle,
  Warning,
  Error as ErrorIcon,
  Refresh,
} from '@mui/icons-material';

interface BOWorkloadProfile {
  tenant_id: string;
  bo_name: string;
  total_queries: number;
  slow_queries: number;
  avg_duration_ms: number;
  p95_duration_ms: number;
  avg_rows_scanned: number;
  p95_rows_scanned: number;
  top_group_bys: { terms: string[]; query_count: number; avg_duration_ms: number }[];
  top_measures: { name: string; query_count: number }[];
  top_filters: { term: string; operator: string; query_count: number }[];
}

interface PreAggCostEstimate {
  estimated_queries_per_day: number;
  avg_duration_ms: number;
  estimated_speedup_factor: number;
  estimated_storage_bytes: number;
  score: number;
}

interface PreAggRecommendation {
  tenant_id: string;
  bo_name: string;
  grain: string[];
  measures: string[];
  cost_estimate: PreAggCostEstimate;
  recommendation_type: string;
}

interface PreAggDescriptor {
  id: string;
  name: string;
  lifecycle_status: string;
  last_refreshed_at?: string;
  next_scheduled_refresh?: string;
  row_count?: number;
  size_bytes?: number;
}

interface BOAdvisorResponse {
  workload: BOWorkloadProfile;
  recommendations: PreAggRecommendation[];
  existing_pre_aggregations: PreAggDescriptor[];
}

interface BOAdvisorTabProps {
  boName: string;
  tenantId: string;
  onCreatePreAgg: (rec: PreAggRecommendation) => void;
}

const statusConfig: Record<string, { color: 'success' | 'warning' | 'error' | 'default'; icon: React.ReactElement }> = {
  active: { color: 'success', icon: <CheckCircle fontSize="small" /> },
  stale: { color: 'warning', icon: <Warning fontSize="small" /> },
  failed: { color: 'error', icon: <ErrorIcon fontSize="small" /> },
};

export const BOAdvisorTab: React.FC<BOAdvisorTabProps> = ({ boName, tenantId, onCreatePreAgg }) => {
  const [data, setData] = useState<BOAdvisorResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchAdvisor = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/bo/${encodeURIComponent(boName)}/advisor?tenant_id=${encodeURIComponent(tenantId)}&window_days=7`);
      if (!res.ok) throw new Error(await res.text());
      setData(await res.json());
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAdvisor();
  }, [boName, tenantId]);

  const formatBytes = (bytes?: number) => {
    if (!bytes) return '-';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" py={4}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }

  if (!data || !data.workload) {
    return (
      <Paper sx={{ p: 4, textAlign: 'center' }}>
        <Typography color="text.secondary">No telemetry data available yet.</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
          Query this BO through Cube.js or send telemetry to start seeing insights.
        </Typography>
      </Paper>
    );
  }

  const { workload, recommendations, existing_pre_aggregations } = data;

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h6">
          <Speed sx={{ mr: 1, verticalAlign: 'text-bottom' }} />
          Performance Advisor
        </Typography>
        <Button onClick={fetchAdvisor} startIcon={<Refresh />}>
          Refresh
        </Button>
      </Box>

      {/* Workload Summary */}
      <Paper sx={{ p: 2, mb: 3 }}>
        <Typography variant="subtitle1" gutterBottom>Workload Summary (Last 7 Days)</Typography>
        <Grid container spacing={2}>
          <Grid item xs={6} sm={2}>
            <MetricCard label="Total Queries" value={workload.total_queries.toLocaleString()} />
          </Grid>
          <Grid item xs={6} sm={2}>
            <MetricCard label="Slow Queries" value={workload.slow_queries.toLocaleString()} color="warning" />
          </Grid>
          <Grid item xs={6} sm={2}>
            <MetricCard label="Avg Duration" value={`${workload.avg_duration_ms.toFixed(0)} ms`} />
          </Grid>
          <Grid item xs={6} sm={2}>
            <MetricCard label="P95 Duration" value={`${workload.p95_duration_ms.toFixed(0)} ms`} />
          </Grid>
          <Grid item xs={6} sm={2}>
            <MetricCard label="Avg Rows Scanned" value={workload.avg_rows_scanned.toLocaleString()} />
          </Grid>
          <Grid item xs={6} sm={2}>
            <MetricCard label="P95 Rows Scanned" value={workload.p95_rows_scanned.toLocaleString()} />
          </Grid>
        </Grid>
      </Paper>

      {/* Recommendations */}
      <Typography variant="subtitle1" gutterBottom>
        <Bolt sx={{ mr: 0.5, verticalAlign: 'text-bottom', color: 'warning.main' }} />
        Recommendations
      </Typography>
      {recommendations.length === 0 ? (
        <Paper sx={{ p: 3, mb: 3, textAlign: 'center' }}>
          <CheckCircle color="success" sx={{ fontSize: 40, mb: 1 }} />
          <Typography>No new pre-aggregation recommendations at this time.</Typography>
        </Paper>
      ) : (
        <Grid container spacing={2} sx={{ mb: 3 }}>
          {recommendations.map((rec, idx) => (
            <Grid item xs={12} md={6} key={idx}>
              <RecommendationCard rec={rec} onAccept={() => onCreatePreAgg(rec)} />
            </Grid>
          ))}
        </Grid>
      )}

      {/* Existing Pre-Aggregations */}
      <Typography variant="subtitle1" gutterBottom>
        <Storage sx={{ mr: 0.5, verticalAlign: 'text-bottom' }} />
        Existing Pre-Aggregations
      </Typography>
      <TableContainer component={Paper}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Last Refresh</TableCell>
              <TableCell align="right">Rows</TableCell>
              <TableCell align="right">Size</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {existing_pre_aggregations.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center">No existing pre-aggregations</TableCell>
              </TableRow>
            ) : (
              existing_pre_aggregations.map((pa) => (
                <TableRow key={pa.id}>
                  <TableCell>{pa.name}</TableCell>
                  <TableCell>
                    <Chip
                      icon={statusConfig[pa.lifecycle_status]?.icon}
                      label={pa.lifecycle_status}
                      color={statusConfig[pa.lifecycle_status]?.color || 'default'}
                      size="small"
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>{pa.last_refreshed_at ? new Date(pa.last_refreshed_at).toLocaleString() : '-'}</TableCell>
                  <TableCell align="right">{pa.row_count?.toLocaleString() || '-'}</TableCell>
                  <TableCell align="right">{formatBytes(pa.size_bytes)}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};

const MetricCard: React.FC<{ label: string; value: string; color?: 'warning' | 'error' }> = ({ label, value, color }) => (
  <Box textAlign="center">
    <Typography variant="h5" color={color ? `${color}.main` : 'text.primary'}>
      {value}
    </Typography>
    <Typography variant="caption" color="text.secondary">{label}</Typography>
  </Box>
);

const RecommendationCard: React.FC<{ rec: PreAggRecommendation; onAccept: () => void }> = ({ rec, onAccept }) => (
  <Card variant="outlined">
    <CardContent>
      <Box display="flex" justifyContent="space-between" alignItems="start">
        <Typography variant="subtitle2">
          {rec.grain.join(', ')} → {rec.measures.join(', ')}
        </Typography>
        <Chip label={`Score: ${rec.cost_estimate.score.toFixed(1)}`} size="small" color="primary" />
      </Box>
      <Divider sx={{ my: 1 }} />
      <Grid container spacing={1}>
        <Grid item xs={4}>
          <Typography variant="caption" color="text.secondary">Speedup</Typography>
          <Typography variant="body2">
            <TrendingUp fontSize="small" color="success" sx={{ mr: 0.5, verticalAlign: 'text-bottom' }} />
            {rec.cost_estimate.estimated_speedup_factor.toFixed(1)}x
          </Typography>
        </Grid>
        <Grid item xs={4}>
          <Typography variant="caption" color="text.secondary">Queries/Day</Typography>
          <Typography variant="body2">{rec.cost_estimate.estimated_queries_per_day}</Typography>
        </Grid>
        <Grid item xs={4}>
          <Typography variant="caption" color="text.secondary">Est. Storage</Typography>
          <Typography variant="body2">
            {(rec.cost_estimate.estimated_storage_bytes / (1024 * 1024)).toFixed(1)} MB
          </Typography>
        </Grid>
      </Grid>
    </CardContent>
    <CardActions>
      <Button size="small" variant="contained" onClick={onAccept}>
        Create Pre-Aggregation
      </Button>
    </CardActions>
  </Card>
);

export default BOAdvisorTab;
