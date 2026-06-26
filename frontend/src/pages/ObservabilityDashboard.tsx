import React, { useState, useEffect } from 'react';
import {
  Box,
  Grid,
  Paper,
  Typography,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Alert,
  Divider,
  LinearProgress,
  IconButton,
  Tooltip,
  Button,
  Stack,
} from '@mui/material';
import {
  CheckCircle as HealthyIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
  Refresh as RefreshIcon,
  TrendingUp as TrendingUpIcon,
  Speed as SpeedIcon,
  Storage as StorageIcon,
  Notifications as NotificationsIcon,
} from '@mui/icons-material';

interface SLOStatus {
  slo_id: string;
  slo_name: string;
  current_value: number;
  target: number;
  budget_total: number;
  budget_consumed: number;
  budget_remaining: number;
  status: 'healthy' | 'degraded' | 'breached';
  window_start: string;
  window_end: string;
  last_evaluated: string;
}

interface ActiveAlert {
  id: string;
  severity: 'critical' | 'warning' | 'info';
  message: string;
  status: string;
  fired_at: string;
}

interface MetricsSummary {
  total_queries: number;
  avg_query_latency: number;
  p95_query_latency: number;
  p99_query_latency: number;
  error_rate: number;
  cache_hit_rate: number;
  active_connections: number;
}

interface DashboardData {
  slo_statuses: SLOStatus[];
  active_alerts: ActiveAlert[];
  metrics_summary: MetricsSummary;
  system_health: {
    status: string;
    cpu_usage: number;
    memory_usage: number;
    last_health_check: string;
  };
}

const statusColors = {
  healthy: '#4caf50',
  degraded: '#ff9800',
  breached: '#f44336',
};

const severityColors = {
  critical: '#f44336',
  warning: '#ff9800',
  info: '#2196f3',
};

export const ObservabilityDashboard: React.FC = () => {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDashboard = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/observability/dashboard');
      if (!response.ok) throw new Error('Failed to fetch dashboard data');
      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDashboard();
    const interval = setInterval(fetchDashboard, 30000); // Refresh every 30s
    return () => clearInterval(interval);
  }, []);

  if (loading && !data) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Observability Dashboard
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Real-time monitoring, SLO tracking, and alerting
          </Typography>
        </Box>
        <Tooltip title="Refresh">
          <IconButton onClick={fetchDashboard} disabled={loading}>
            <RefreshIcon />
          </IconButton>
        </Tooltip>
      </Stack>

      {/* Metrics Summary Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ height: '100%', background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                <Box>
                  <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                    Total Queries (24h)
                  </Typography>
                  <Typography variant="h4" sx={{ color: 'white', fontWeight: 700 }}>
                    {data?.metrics_summary.total_queries?.toLocaleString() || 0}
                  </Typography>
                </Box>
                <TrendingUpIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 40 }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ height: '100%', background: 'linear-gradient(135deg, #11998e 0%, #38ef7d 100%)' }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                <Box>
                  <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                    Avg Latency
                  </Typography>
                  <Typography variant="h4" sx={{ color: 'white', fontWeight: 700 }}>
                    {data?.metrics_summary.avg_query_latency?.toFixed(0) || 0}ms
                  </Typography>
                </Box>
                <SpeedIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 40 }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ height: '100%', background: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)' }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                <Box>
                  <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                    Error Rate
                  </Typography>
                  <Typography variant="h4" sx={{ color: 'white', fontWeight: 700 }}>
                    {((data?.metrics_summary.error_rate || 0) * 100).toFixed(2)}%
                  </Typography>
                </Box>
                <ErrorIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 40 }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ height: '100%', background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)' }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                <Box>
                  <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                    Cache Hit Rate
                  </Typography>
                  <Typography variant="h4" sx={{ color: 'white', fontWeight: 700 }}>
                    {((data?.metrics_summary.cache_hit_rate || 0) * 100).toFixed(1)}%
                  </Typography>
                </Box>
                <StorageIcon sx={{ color: 'rgba(255,255,255,0.7)', fontSize: 40 }} />
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Grid container spacing={3}>
        {/* SLO Status Cards */}
        <Grid item xs={12} md={8}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              SLO Status
            </Typography>
            {(!data?.slo_statuses || data.slo_statuses.length === 0) ? (
              <Box sx={{ textAlign: 'center', py: 4 }}>
                <Typography color="text.secondary">No SLOs configured yet</Typography>
                <Button variant="outlined" sx={{ mt: 2 }} href="/observability/slos">
                  Create SLO
                </Button>
              </Box>
            ) : (
              <Grid container spacing={2}>
                {data.slo_statuses.map((slo) => (
                  <Grid item xs={12} sm={6} key={slo.slo_id}>
                    <Card variant="outlined" sx={{ borderLeft: `4px solid ${statusColors[slo.status]}` }}>
                      <CardContent>
                        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                          <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                            {slo.slo_name}
                          </Typography>
                          <Chip
                            size="small"
                            label={slo.status.toUpperCase()}
                            sx={{
                              backgroundColor: statusColors[slo.status],
                              color: 'white',
                              fontWeight: 600,
                            }}
                          />
                        </Stack>
                        <Typography variant="h5" sx={{ mb: 1 }}>
                          {slo.current_value.toFixed(2)}% <span style={{ fontSize: '0.6em', color: '#888' }}>/ {slo.target}%</span>
                        </Typography>
                        <Box sx={{ mb: 1 }}>
                          <Typography variant="caption" color="text.secondary">
                            Error Budget: {slo.budget_remaining.toFixed(2)}% remaining
                          </Typography>
                          <LinearProgress
                            variant="determinate"
                            value={Math.max(0, (slo.budget_remaining / slo.budget_total) * 100)}
                            sx={{
                              mt: 0.5,
                              height: 6,
                              borderRadius: 3,
                              backgroundColor: 'rgba(0,0,0,0.1)',
                              '& .MuiLinearProgress-bar': {
                                backgroundColor: statusColors[slo.status],
                              },
                            }}
                          />
                        </Box>
                      </CardContent>
                    </Card>
                  </Grid>
                ))}
              </Grid>
            )}
          </Paper>
        </Grid>

        {/* Active Alerts */}
        <Grid item xs={12} md={4}>
          <Paper sx={{ p: 3, height: '100%' }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                Active Alerts
              </Typography>
              <NotificationsIcon color="action" />
            </Stack>
            {(!data?.active_alerts || data.active_alerts.length === 0) ? (
              <Box sx={{ textAlign: 'center', py: 4 }}>
                <HealthyIcon sx={{ fontSize: 48, color: '#4caf50', mb: 1 }} />
                <Typography color="text.secondary">No active alerts</Typography>
              </Box>
            ) : (
              <Stack spacing={2}>
                {data.active_alerts.slice(0, 5).map((alert) => (
                  <Card
                    key={alert.id}
                    variant="outlined"
                    sx={{
                      borderLeft: `4px solid ${severityColors[alert.severity]}`,
                      backgroundColor: `${severityColors[alert.severity]}10`,
                    }}
                  >
                    <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
                      <Stack direction="row" spacing={1} alignItems="center">
                        {alert.severity === 'critical' && <ErrorIcon sx={{ color: severityColors.critical, fontSize: 20 }} />}
                        {alert.severity === 'warning' && <WarningIcon sx={{ color: severityColors.warning, fontSize: 20 }} />}
                        <Typography variant="body2" sx={{ fontWeight: 500 }}>
                          {alert.message.slice(0, 100)}{alert.message.length > 100 ? '...' : ''}
                        </Typography>
                      </Stack>
                      <Typography variant="caption" color="text.secondary">
                        {new Date(alert.fired_at).toLocaleString()}
                      </Typography>
                    </CardContent>
                  </Card>
                ))}
              </Stack>
            )}
          </Paper>
        </Grid>

        {/* System Health */}
        <Grid item xs={12}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              System Health
            </Typography>
            <Grid container spacing={3}>
              <Grid item xs={12} sm={4}>
                <Box sx={{ textAlign: 'center' }}>
                  <Typography variant="overline" color="text.secondary">Status</Typography>
                  <Box sx={{ my: 1 }}>
                    {data?.system_health.status === 'healthy' && (
                      <HealthyIcon sx={{ fontSize: 48, color: '#4caf50' }} />
                    )}
                    {data?.system_health.status === 'degraded' && (
                      <WarningIcon sx={{ fontSize: 48, color: '#ff9800' }} />
                    )}
                    {data?.system_health.status === 'critical' && (
                      <ErrorIcon sx={{ fontSize: 48, color: '#f44336' }} />
                    )}
                  </Box>
                  <Typography variant="h6" sx={{ textTransform: 'capitalize' }}>
                    {data?.system_health.status || 'Unknown'}
                  </Typography>
                </Box>
              </Grid>
              <Grid item xs={12} sm={4}>
                <Box sx={{ textAlign: 'center' }}>
                  <Typography variant="overline" color="text.secondary">CPU Usage</Typography>
                  <Typography variant="h4" sx={{ my: 1 }}>
                    {((data?.system_health.cpu_usage || 0) * 100).toFixed(0)}%
                  </Typography>
                  <LinearProgress
                    variant="determinate"
                    value={(data?.system_health.cpu_usage || 0) * 100}
                    sx={{ height: 8, borderRadius: 4 }}
                  />
                </Box>
              </Grid>
              <Grid item xs={12} sm={4}>
                <Box sx={{ textAlign: 'center' }}>
                  <Typography variant="overline" color="text.secondary">Memory Usage</Typography>
                  <Typography variant="h4" sx={{ my: 1 }}>
                    {((data?.system_health.memory_usage || 0) * 100).toFixed(0)}%
                  </Typography>
                  <LinearProgress
                    variant="determinate"
                    value={(data?.system_health.memory_usage || 0) * 100}
                    sx={{ height: 8, borderRadius: 4 }}
                  />
                </Box>
              </Grid>
            </Grid>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default ObservabilityDashboard;
