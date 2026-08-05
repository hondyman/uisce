import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Chip,
  Button,
  Stack,
  Card,
  CardContent,
  CircularProgress,
  LinearProgress,
  useTheme,
  Alert,
} from '@mui/material';
import {
  AutoAwesome as AutoAwesomeIcon,
  ShieldCheck as ShieldCheckIcon,
  TrendingUp as TrendingUpIcon,
  Business as BusinessIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { apiClient } from '../../../utils/apiClient';

interface ClientBenchmarkData {
  tenant_id: string;
  industry_vertical: string;
  tenant_role_churn_pct: number;
  industry_role_churn_pct: number;
  tenant_violations_count: number;
  industry_avg_violations: number;
  stability_percentile: number;
  ai_summary: string;
  evaluated_at: string;
}

export const ClientBenchmarks: React.FC = () => {
  const theme = useTheme();
  const [data, setData] = useState<ClientBenchmarkData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  const fetchBenchmarks = async () => {
    try {
      setLoading(true);
      const res = await apiClient<ClientBenchmarkData>('/ai-intelligence/benchmarks');
      setData(res);
    } catch (err) {
      console.error('Failed to load client benchmarks:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBenchmarks();
  }, []);

  return (
    <Box sx={{ p: 4, bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Top Banner */}
      <Alert
        icon={<AutoAwesomeIcon color="primary" />}
        severity="success"
        sx={{ mb: 4, borderRadius: 3, border: `1px solid ${theme.palette.success.light}` }}
      >
        <Typography variant="subtitle2" fontWeight={700}>
          Anonymized Industry Benchmarking Active
        </Typography>
        <Typography variant="caption">
          Comparing your organization's RBAC stability and access review metrics against 500,000+ scrubbed data points in Repo 4.
        </Typography>
      </Alert>

      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={4}>
        <Box>
          <Typography variant="h4" fontWeight={700}>
            How Your Organization Compares
          </Typography>
          <Typography variant="body2" color="text.secondary">
            AI-generated peer benchmarks and RBAC stability insights for {data?.industry_vertical || 'Finance'}
          </Typography>
        </Box>
        <Button variant="outlined" startIcon={<RefreshIcon />} onClick={fetchBenchmarks} sx={{ borderRadius: 2 }}>
          Recalculate Metrics
        </Button>
      </Stack>

      {loading ? (
        <Box sx={{ display: 'flex', py: 8, justifyContent: 'center' }}>
          <CircularProgress />
        </Box>
      ) : (
        <Grid container spacing={3}>
          {/* AI Executive Summary Box */}
          <Grid size={{ xs: 12 }}>
            <Paper
              elevation={0}
              sx={{
                p: 3,
                borderRadius: 3,
                bgcolor: 'primary.50',
                border: `1px solid ${theme.palette.primary.main}`,
              }}
            >
              <Stack direction="row" spacing={2} alignItems="flex-start">
                <AutoAwesomeIcon color="primary" sx={{ fontSize: 28, mt: 0.5 }} />
                <Box>
                  <Typography variant="h6" fontWeight={700} color="primary.main">
                    AI Governance Summary & Outlook
                  </Typography>
                  <Typography variant="body1" mt={1} fontWeight={500} color="text.primary">
                    "{data?.ai_summary}"
                  </Typography>
                  <Typography variant="caption" color="text.secondary" mt={1} display="block">
                    Evaluated at {new Date(data?.evaluated_at || '').toLocaleString()} via PyIceberg & LLM Engine
                  </Typography>
                </Box>
              </Stack>
            </Paper>
          </Grid>

          {/* Metric 1: Role Churn Rate */}
          <Grid size={{ xs: 12, md: 6 }}>
            <Paper elevation={0} sx={{ p: 3, borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
              <Typography variant="subtitle1" fontWeight={700} mb={2}>
                RBAC Role Churn Rate
              </Typography>
              <Typography variant="body2" color="text.secondary" mb={2}>
                Lower is better. Indicates stability of access control definitions.
              </Typography>

              <Box mb={2}>
                <Stack direction="row" justifyContent="space-between" mb={0.5}>
                  <Typography variant="caption" fontWeight={700}>Your Tenant</Typography>
                  <Typography variant="caption" fontWeight={700} color="success.main">
                    {data?.tenant_role_churn_pct}%
                  </Typography>
                </Stack>
                <LinearProgress
                  variant="determinate"
                  value={Math.min((data?.tenant_role_churn_pct || 0) * 5, 100)}
                  color="success"
                  sx={{ height: 10, borderRadius: 5 }}
                />
              </Box>

              <Box>
                <Stack direction="row" justifyContent="space-between" mb={0.5}>
                  <Typography variant="caption" fontWeight={700}>Industry Average ({data?.industry_vertical})</Typography>
                  <Typography variant="caption" fontWeight={700} color="text.secondary">
                    {data?.industry_role_churn_pct}%
                  </Typography>
                </Stack>
                <LinearProgress
                  variant="determinate"
                  value={Math.min((data?.industry_role_churn_pct || 0) * 5, 100)}
                  color="secondary"
                  sx={{ height: 10, borderRadius: 5 }}
                />
              </Box>
            </Paper>
          </Grid>

          {/* Metric 2: Compliance Violations */}
          <Grid size={{ xs: 12, md: 6 }}>
            <Paper elevation={0} sx={{ p: 3, borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
              <Typography variant="subtitle1" fontWeight={700} mb={2}>
                Active Compliance Violations
              </Typography>
              <Typography variant="body2" color="text.secondary" mb={2}>
                Compared against peers in your industry sector.
              </Typography>

              <Box mb={2}>
                <Stack direction="row" justifyContent="space-between" mb={0.5}>
                  <Typography variant="caption" fontWeight={700}>Your Tenant</Typography>
                  <Typography variant="caption" fontWeight={700} color="success.main">
                    {data?.tenant_violations_count} Violations
                  </Typography>
                </Stack>
                <LinearProgress
                  variant="determinate"
                  value={Math.min((data?.tenant_violations_count || 0) * 5, 100)}
                  color="success"
                  sx={{ height: 10, borderRadius: 5 }}
                />
              </Box>

              <Box>
                <Stack direction="row" justifyContent="space-between" mb={0.5}>
                  <Typography variant="caption" fontWeight={700}>Industry Average</Typography>
                  <Typography variant="caption" fontWeight={700} color="text.secondary">
                    {data?.industry_avg_violations} Violations
                  </Typography>
                </Stack>
                <LinearProgress
                  variant="determinate"
                  value={Math.min((data?.industry_avg_violations || 0) * 5, 100)}
                  color="warning"
                  sx={{ height: 10, borderRadius: 5 }}
                />
              </Box>
            </Paper>
          </Grid>

          {/* Metric 3: Stability Percentile */}
          <Grid size={{ xs: 12 }}>
            <Card elevation={0} sx={{ borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
              <CardContent>
                <Stack direction="row" justifyContent="space-between" alignItems="center">
                  <Box>
                    <Typography variant="h6" fontWeight={700}>
                      Governance Stability Rank
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      Your organization is performing better than {data?.stability_percentile}% of financial institutions in your peer group.
                    </Typography>
                  </Box>
                  <Chip
                    label={`TOP ${100 - Math.round(data?.stability_percentile || 90)}% PERFORMER`}
                    color="primary"
                    sx={{ fontWeight: 700, py: 2, px: 1 }}
                  />
                </Stack>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}
    </Box>
  );
};

export default ClientBenchmarks;
