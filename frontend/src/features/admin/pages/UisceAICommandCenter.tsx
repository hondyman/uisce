import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Chip,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Stack,
  Card,
  CardContent,
  CircularProgress,
  IconButton,
  Tooltip,
  Alert,
} from '@mui/material';
import {
  Psychology as PsychologyIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  TrendingUp as TrendingUpIcon,
  AutoGraph as AutoGraphIcon,
  Refresh as RefreshIcon,
  Assessment as AssessmentIcon,
} from '@mui/icons-material';
import { apiClient } from '../../../utils/apiClient';

interface AIAlert {
  alert_id: string;
  tenant_hash: string;
  industry_vertical: string;
  anomaly_type: string;
  z_score: number;
  summary: string;
  acknowledged: boolean;
  created_at: string;
}

interface GlobalHealthData {
  status: string;
  global_risk_score: number;
  total_events_analyzed: number;
  avg_sentiment_score: number;
  active_anomalies_count: number;
  anomalies_feed: AIAlert[];
  industry_benchmarks: Array<{
    industry: string;
    total_events: number;
    role_churn_pct: number;
    avg_risk: string;
  }>;
}

export const UisceAICommandCenter: React.FC = () => {
  const [data, setData] = useState<GlobalHealthData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  const fetchData = async () => {
    try {
      setLoading(true);
      const res = await apiClient<GlobalHealthData>('/ai-intelligence/global-health');
      setData(res);
    } catch (err) {
      console.error('Failed to fetch AI command center health:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  return (
    <Box sx={{ p: 4, bgcolor: '#0B0F19', color: '#E2E8F0', minHeight: '100vh' }}>
      {/* Top Banner */}
      <Alert
        icon={<PsychologyIcon sx={{ color: '#38BDF8' }} />}
        severity="info"
        sx={{
          mb: 4,
          borderRadius: 3,
          bgcolor: '#1E293B',
          color: '#F8FAFC',
          border: '1px solid #334155',
        }}
      >
        <Typography variant="subtitle2" fontWeight={700}>
          Repo 4 AI Global Intelligence Store Active
        </Typography>
        <Typography variant="caption" sx={{ color: '#94A3B8' }}>
          Continuously scrubbing PII, evaluating LLM risk sentiment, and executing Z-score anomaly detection across global lakehouse stores.
        </Typography>
      </Alert>

      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={4}>
        <Box>
          <Typography variant="h4" fontWeight={800} sx={{ color: '#F8FAFC' }}>
            Uisce AI Command Center
          </Typography>
          <Typography variant="body2" sx={{ color: '#94A3B8' }}>
            Global Anonymized AI Flywheel, Behavioral Anomalies & Industry Benchmarks
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<RefreshIcon />}
          onClick={fetchData}
          sx={{ borderRadius: 2, bgcolor: '#0284C7', '&:hover': { bgcolor: '#0369A1' } }}
        >
          Refresh AI Models
        </Button>
      </Stack>

      {/* Top Metric Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card sx={{ bgcolor: '#1E293B', borderRadius: 3, border: '1px solid #334155' }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="body2" sx={{ color: '#94A3B8' }} fontWeight={600}>
                  Total Events Analyzed
                </Typography>
                <AutoGraphIcon sx={{ color: '#38BDF8' }} />
              </Stack>
              <Typography variant="h4" fontWeight={800} mt={1} sx={{ color: '#F8FAFC' }}>
                {data?.total_events_analyzed?.toLocaleString() ?? 0}
              </Typography>
              <Typography variant="caption" sx={{ color: '#64748B' }}>
                Pooled in Repo 4
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card sx={{ bgcolor: '#1E293B', borderRadius: 3, border: '1px solid #334155' }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="body2" sx={{ color: '#94A3B8' }} fontWeight={600}>
                  Global AI Risk Score
                </Typography>
                <TrendingUpIcon sx={{ color: '#F59E0B' }} />
              </Stack>
              <Typography variant="h4" fontWeight={800} mt={1} sx={{ color: '#F59E0B' }}>
                {data?.global_risk_score ?? 0.0} / 5.0
              </Typography>
              <Typography variant="caption" sx={{ color: '#64748B' }}>
                System-wide Risk Level
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card sx={{ bgcolor: '#1E293B', borderRadius: 3, border: '1px solid #334155' }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="body2" sx={{ color: '#94A3B8' }} fontWeight={600}>
                  Active AI Anomalies
                </Typography>
                <WarningIcon sx={{ color: '#EF4444' }} />
              </Stack>
              <Typography variant="h4" fontWeight={800} mt={1} sx={{ color: '#EF4444' }}>
                {data?.active_anomalies_count ?? 0}
              </Typography>
              <Typography variant="caption" sx={{ color: '#64748B' }}>
                Z-Score Baseline Violations
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card sx={{ bgcolor: '#1E293B', borderRadius: 3, border: '1px solid #334155' }}>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="body2" sx={{ color: '#94A3B8' }} fontWeight={600}>
                  Avg Sentiment Score
                </Typography>
                <AssessmentIcon sx={{ color: '#10B981' }} />
              </Stack>
              <Typography variant="h4" fontWeight={800} mt={1} sx={{ color: '#10B981' }}>
                +{data?.avg_sentiment_score ?? 0.0}
              </Typography>
              <Typography variant="caption" sx={{ color: '#64748B' }}>
                LLM Audit Log Tone
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Main Grid Section */}
      <Grid container spacing={3}>
        {/* Anomaly Feed */}
        <Grid size={{ xs: 12, lg: 7 }}>
          <Paper sx={{ p: 3, borderRadius: 3, bgcolor: '#1E293B', border: '1px solid #334155' }}>
            <Typography variant="h6" fontWeight={700} mb={2} sx={{ color: '#F8FAFC' }}>
              Live AI Behavioral Anomaly Feed
            </Typography>

            {loading ? (
              <Box sx={{ display: 'flex', py: 4, justifyContent: 'center' }}>
                <CircularProgress color="info" />
              </Box>
            ) : (
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell sx={{ color: '#94A3B8', fontWeight: 700 }}>Tenant Hash</TableCell>
                      <TableCell sx={{ color: '#94A3B8', fontWeight: 700 }}>Vertical</TableCell>
                      <TableCell sx={{ color: '#94A3B8', fontWeight: 700 }}>Anomaly Type</TableCell>
                      <TableCell sx={{ color: '#94A3B8', fontWeight: 700 }}>Z-Score</TableCell>
                      <TableCell sx={{ color: '#94A3B8', fontWeight: 700 }}>AI Summary</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {data?.anomalies_feed.map((alt) => (
                      <TableRow key={alt.alert_id} hover>
                        <TableCell sx={{ color: '#F8FAFC', fontFamily: 'monospace' }}>
                          {alt.tenant_hash}
                        </TableCell>
                        <TableCell sx={{ color: '#94A3B8' }}>{alt.industry_vertical}</TableCell>
                        <TableCell>
                          <Chip label={alt.anomaly_type} size="small" color="error" variant="outlined" />
                        </TableCell>
                        <TableCell sx={{ color: '#EF4444', fontWeight: 700 }}>
                          {alt.z_score} σ
                        </TableCell>
                        <TableCell sx={{ color: '#CBD5E1', fontSize: '0.8rem' }}>
                          {alt.summary}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </Paper>
        </Grid>

        {/* Industry Benchmarks */}
        <Grid size={{ xs: 12, lg: 5 }}>
          <Paper sx={{ p: 3, borderRadius: 3, bgcolor: '#1E293B', border: '1px solid #334155' }}>
            <Typography variant="h6" fontWeight={700} mb={2} sx={{ color: '#F8FAFC' }}>
              Industry Vertical Benchmarks
            </Typography>

            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ color: '#94A3B8', fontWeight: 700 }}>Industry</TableCell>
                    <TableCell sx={{ color: '#94A3B8', fontWeight: 700 }}>Total Events</TableCell>
                    <TableCell sx={{ color: '#94A3B8', fontWeight: 700 }}>Role Churn %</TableCell>
                    <TableCell sx={{ color: '#94A3B8', fontWeight: 700 }}>Risk Level</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {data?.industry_benchmarks.map((bm, i) => (
                    <TableRow key={i} hover>
                      <TableCell sx={{ color: '#F8FAFC', fontWeight: 600 }}>{bm.industry}</TableCell>
                      <TableCell sx={{ color: '#94A3B8' }}>{bm.total_events.toLocaleString()}</TableCell>
                      <TableCell sx={{ color: '#38BDF8', fontWeight: 700 }}>{bm.role_churn_pct}%</TableCell>
                      <TableCell>
                        <Chip
                          label={bm.avg_risk}
                          size="small"
                          color={bm.avg_risk === 'HIGH' ? 'error' : bm.avg_risk === 'MEDIUM' ? 'warning' : 'success'}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default UisceAICommandCenter;
