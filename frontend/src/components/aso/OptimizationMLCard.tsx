import React from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Stack,
  Chip,
  LinearProgress,
  Tooltip,
  Paper,
  Divider,
} from '@mui/material';
import {
  Psychology,
  TrendingUp,
  AttachMoney,
  Warning,
  CheckCircle,
  Speed,
  Shield,
  Analytics,
} from '@mui/icons-material';

// ============================================================================
// Types
// ============================================================================

export interface MLEvidence {
  score: number;
  confidence: number;
  predicted_speedup: number;
  predicted_cost_savings: number;
  risk_score: number;
  top_factors: TopFactor[];
}

export interface TopFactor {
  feature: string;
  weight: number;
  direction: 'positive' | 'negative';
}

export interface WorkloadEvidence {
  window: string;
  queries: number;
  queries_per_day: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  distinct_users: number;
  preagg_miss_rate: number;
  preagg_hit_rate: number;
}

export interface SimulationEvidence {
  expected_speedup: number;
  expected_cost_savings: number;
  queries_improved: number;
  queries_regressed: number;
  hit_rate_before: number;
  hit_rate_after: number;
  confidence: number;
}

export interface ExplainPayload {
  nl_summary: string;
  workload: WorkloadEvidence;
  ml: MLEvidence;
  simulation?: SimulationEvidence;
}

// ============================================================================
// ML Scoring Card Component
// ============================================================================

interface OptimizationMLCardProps {
  ml?: MLEvidence;
}

export const OptimizationMLCard: React.FC<OptimizationMLCardProps> = ({ ml }) => {
  if (!ml) {
    return (
      <Card>
        <CardContent>
          <Typography variant="h6" color="text.secondary">
            <Psychology sx={{ mr: 1, verticalAlign: 'middle' }} />
            ML Scoring
          </Typography>
          <Typography color="text.secondary">
            No ML scoring available for this optimization.
          </Typography>
        </CardContent>
      </Card>
    );
  }

  const getScoreColor = (score: number) => {
    if (score >= 0.8) return 'success';
    if (score >= 0.6) return 'warning';
    return 'error';
  };

  const getRiskColor = (risk: number) => {
    if (risk < 0.2) return 'success';
    if (risk < 0.5) return 'warning';
    return 'error';
  };

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          <Psychology sx={{ mr: 1, verticalAlign: 'middle', color: 'primary.main' }} />
          ML Assessment
        </Typography>

        <Grid container spacing={3}>
          {/* ML Score */}
          <Grid item xs={6} md={3}>
            <Stack alignItems="center">
              <Typography variant="caption" color="text.secondary">
                ML Score
              </Typography>
              <Typography variant="h3" fontWeight="bold" color={`${getScoreColor(ml.score)}.main`}>
                {(ml.score * 100).toFixed(0)}
              </Typography>
              <Chip
                label={ml.score >= 0.8 ? 'Recommended' : ml.score >= 0.6 ? 'Consider' : 'Low Score'}
                size="small"
                color={getScoreColor(ml.score)}
              />
            </Stack>
          </Grid>

          {/* Predicted Speedup */}
          <Grid item xs={6} md={3}>
            <Stack alignItems="center">
              <Typography variant="caption" color="text.secondary">
                Predicted Speedup
              </Typography>
              <Stack direction="row" alignItems="baseline" spacing={0.5}>
                <Typography variant="h3" fontWeight="bold" color="success.main">
                  {ml.predicted_speedup.toFixed(1)}
                </Typography>
                <Typography variant="h6" color="text.secondary">x</Typography>
              </Stack>
              <Chip
                icon={<TrendingUp />}
                label="Performance Gain"
                size="small"
                color="success"
                variant="outlined"
              />
            </Stack>
          </Grid>

          {/* Predicted Cost Savings */}
          <Grid item xs={6} md={3}>
            <Stack alignItems="center">
              <Typography variant="caption" color="text.secondary">
                Cost Savings
              </Typography>
              <Stack direction="row" alignItems="baseline" spacing={0.5}>
                <Typography variant="h3" fontWeight="bold" color="info.main">
                  {(ml.predicted_cost_savings * 100).toFixed(0)}
                </Typography>
                <Typography variant="h6" color="text.secondary">%</Typography>
              </Stack>
              <Chip
                icon={<AttachMoney />}
                label="Predicted Savings"
                size="small"
                color="info"
                variant="outlined"
              />
            </Stack>
          </Grid>

          {/* Risk Score */}
          <Grid item xs={6} md={3}>
            <Stack alignItems="center">
              <Typography variant="caption" color="text.secondary">
                Risk Score
              </Typography>
              <Typography variant="h3" fontWeight="bold" color={`${getRiskColor(ml.risk_score)}.main`}>
                {(ml.risk_score * 100).toFixed(0)}%
              </Typography>
              <Chip
                icon={ml.risk_score < 0.2 ? <CheckCircle /> : <Warning />}
                label={ml.risk_score < 0.2 ? 'Low Risk' : ml.risk_score < 0.5 ? 'Medium Risk' : 'High Risk'}
                size="small"
                color={getRiskColor(ml.risk_score)}
                variant="outlined"
              />
            </Stack>
          </Grid>
        </Grid>

        {/* Confidence Bar */}
        <Box sx={{ mt: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" mb={1}>
            <Typography variant="caption" color="text.secondary">
              Prediction Confidence
            </Typography>
            <Typography variant="body2" fontWeight="bold">
              {(ml.confidence * 100).toFixed(0)}%
            </Typography>
          </Stack>
          <LinearProgress
            variant="determinate"
            value={ml.confidence * 100}
            color={ml.confidence >= 0.7 ? 'success' : 'warning'}
            sx={{ height: 8, borderRadius: 4 }}
          />
        </Box>

        {/* Top Factors */}
        {ml.top_factors && ml.top_factors.length > 0 && (
          <>
            <Divider sx={{ my: 2 }} />
            <Typography variant="subtitle2" gutterBottom>
              <Analytics sx={{ mr: 0.5, verticalAlign: 'middle', fontSize: 18 }} />
              Top Contributing Factors
            </Typography>
            <Stack spacing={1}>
              {ml.top_factors.slice(0, 5).map((factor, idx) => (
                <TopFactorRow key={idx} factor={factor} />
              ))}
            </Stack>
          </>
        )}
      </CardContent>
    </Card>
  );
};

// ============================================================================
// Top Factor Row
// ============================================================================

const TopFactorRow: React.FC<{ factor: TopFactor }> = ({ factor }) => {
  const formatFeatureName = (name: string) => {
    const map: Record<string, string> = {
      preagg_miss_rate: 'Pre-Agg Miss Rate',
      p95_latency_ms: 'P95 Latency',
      queries_per_day: 'Query Volume',
      usage_decline: 'Usage Decline',
      distinct_users: 'User Count',
      storage_bytes: 'Storage Size',
      refresh_cost_ms: 'Refresh Cost',
      bo_queries: 'Total Queries',
      bo_avg_latency_ms: 'Avg Latency',
    };
    return map[name] || name.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  return (
    <Paper variant="outlined" sx={{ p: 1 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center">
        <Stack direction="row" spacing={1} alignItems="center">
          {factor.direction === 'positive' ? (
            <TrendingUp color="success" fontSize="small" />
          ) : (
            <Warning color="warning" fontSize="small" />
          )}
          <Typography variant="body2">{formatFeatureName(factor.feature)}</Typography>
        </Stack>
        <Tooltip title={`${factor.direction} impact: ${(factor.weight * 100).toFixed(0)}%`}>
          <Box sx={{ width: 100 }}>
            <LinearProgress
              variant="determinate"
              value={factor.weight * 100}
              color={factor.direction === 'positive' ? 'success' : 'warning'}
              sx={{ height: 6, borderRadius: 3 }}
            />
          </Box>
        </Tooltip>
      </Stack>
    </Paper>
  );
};

// ============================================================================
// Explainability Card Component
// ============================================================================

interface OptimizationExplainabilityCardProps {
  explain?: ExplainPayload;
}

export const OptimizationExplainabilityCard: React.FC<OptimizationExplainabilityCardProps> = ({
  explain,
}) => {
  if (!explain) return null;

  const { workload, ml, simulation, nl_summary } = explain;

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          <Shield sx={{ mr: 1, verticalAlign: 'middle', color: 'info.main' }} />
          Why This Optimization
        </Typography>

        {/* Natural Language Summary */}
        {nl_summary && (
          <Paper
            elevation={0}
            sx={{
              p: 2,
              mb: 3,
              bgcolor: 'info.light',
              borderLeft: 4,
              borderColor: 'info.main',
            }}
          >
            <Typography variant="body1">{nl_summary}</Typography>
          </Paper>
        )}

        <Grid container spacing={3}>
          {/* Workload Evidence */}
          <Grid item xs={12} md={4}>
            <Typography variant="subtitle2" gutterBottom>
              <Speed sx={{ mr: 0.5, verticalAlign: 'middle', fontSize: 18 }} />
              Workload Evidence
            </Typography>
            <Stack spacing={1}>
              <EvidenceItem label="Window" value={workload.window} />
              <EvidenceItem label="Queries" value={workload.queries.toLocaleString()} />
              <EvidenceItem label="Queries/Day" value={workload.queries_per_day.toFixed(0)} />
              <EvidenceItem label="P95 Latency" value={`${workload.p95_latency_ms.toFixed(0)}ms`} highlight={workload.p95_latency_ms > 1000} />
              <EvidenceItem label="Pre-Agg Miss Rate" value={`${(workload.preagg_miss_rate * 100).toFixed(0)}%`} highlight={workload.preagg_miss_rate > 0.5} />
              <EvidenceItem label="Distinct Users" value={workload.distinct_users.toString()} />
            </Stack>
          </Grid>

          {/* ML Assessment Summary */}
          <Grid item xs={12} md={4}>
            <Typography variant="subtitle2" gutterBottom>
              <Psychology sx={{ mr: 0.5, verticalAlign: 'middle', fontSize: 18 }} />
              ML Assessment
            </Typography>
            <Stack spacing={1}>
              <EvidenceItem label="ML Score" value={`${(ml.score * 100).toFixed(0)}/100`} />
              <EvidenceItem label="Predicted Speedup" value={`${ml.predicted_speedup.toFixed(1)}x`} />
              <EvidenceItem label="Cost Savings" value={`${(ml.predicted_cost_savings * 100).toFixed(0)}%`} />
              <EvidenceItem label="Risk Score" value={`${(ml.risk_score * 100).toFixed(0)}%`} highlight={ml.risk_score > 0.3} />
              <EvidenceItem label="Confidence" value={`${(ml.confidence * 100).toFixed(0)}%`} />
            </Stack>
          </Grid>

          {/* Simulation Results */}
          {simulation && (
            <Grid item xs={12} md={4}>
              <Typography variant="subtitle2" gutterBottom>
                <Analytics sx={{ mr: 0.5, verticalAlign: 'middle', fontSize: 18 }} />
                Simulation (What-If)
              </Typography>
              <Stack spacing={1}>
                <EvidenceItem label="Expected Speedup" value={`${simulation.expected_speedup.toFixed(1)}x`} />
                <EvidenceItem label="Queries Improved" value={simulation.queries_improved.toString()} />
                <EvidenceItem label="Queries Regressed" value={simulation.queries_regressed.toString()} highlight={simulation.queries_regressed > 0} />
                <EvidenceItem label="Hit Rate Before" value={`${(simulation.hit_rate_before * 100).toFixed(0)}%`} />
                <EvidenceItem label="Hit Rate After" value={`${(simulation.hit_rate_after * 100).toFixed(0)}%`} />
                <EvidenceItem label="Sim Confidence" value={`${(simulation.confidence * 100).toFixed(0)}%`} />
              </Stack>
            </Grid>
          )}
        </Grid>
      </CardContent>
    </Card>
  );
};

// ============================================================================
// Evidence Item
// ============================================================================

interface EvidenceItemProps {
  label: string;
  value: string;
  highlight?: boolean;
}

const EvidenceItem: React.FC<EvidenceItemProps> = ({ label, value, highlight = false }) => (
  <Stack
    direction="row"
    justifyContent="space-between"
    sx={{
      py: 0.5,
      px: 1,
      borderRadius: 1,
      bgcolor: highlight ? 'warning.light' : 'grey.50',
    }}
  >
    <Typography variant="body2" color="text.secondary">
      {label}
    </Typography>
    <Typography variant="body2" fontWeight={highlight ? 'bold' : 'medium'}>
      {value}
    </Typography>
  </Stack>
);

// ============================================================================
// Simulation Card Component
// ============================================================================

interface OptimizationSimulationCardProps {
  simulation?: SimulationEvidence;
}

export const OptimizationSimulationCard: React.FC<OptimizationSimulationCardProps> = ({
  simulation,
}) => {
  if (!simulation) return null;

  const hitRateImprovement = simulation.hit_rate_after - simulation.hit_rate_before;

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          <Analytics sx={{ mr: 1, verticalAlign: 'middle', color: 'secondary.main' }} />
          Simulation Results
        </Typography>

        <Grid container spacing={2}>
          <Grid item xs={6} md={3}>
            <Stack alignItems="center">
              <Typography variant="caption" color="text.secondary">
                Expected Speedup
              </Typography>
              <Typography variant="h4" fontWeight="bold" color="success.main">
                {simulation.expected_speedup.toFixed(1)}x
              </Typography>
            </Stack>
          </Grid>

          <Grid item xs={6} md={3}>
            <Stack alignItems="center">
              <Typography variant="caption" color="text.secondary">
                Cost Savings
              </Typography>
              <Typography variant="h4" fontWeight="bold" color="info.main">
                {(simulation.expected_cost_savings * 100).toFixed(0)}%
              </Typography>
            </Stack>
          </Grid>

          <Grid item xs={6} md={3}>
            <Stack alignItems="center">
              <Typography variant="caption" color="text.secondary">
                Queries Improved
              </Typography>
              <Typography variant="h4" fontWeight="bold" color="success.main">
                {simulation.queries_improved}
              </Typography>
            </Stack>
          </Grid>

          <Grid item xs={6} md={3}>
            <Stack alignItems="center">
              <Typography variant="caption" color="text.secondary">
                Queries Regressed
              </Typography>
              <Typography
                variant="h4"
                fontWeight="bold"
                color={simulation.queries_regressed > 0 ? 'warning.main' : 'success.main'}
              >
                {simulation.queries_regressed}
              </Typography>
            </Stack>
          </Grid>
        </Grid>

        {/* Hit Rate Comparison */}
        <Box sx={{ mt: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            Pre-Agg Hit Rate Comparison
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={5}>
              <Paper variant="outlined" sx={{ p: 2, textAlign: 'center' }}>
                <Typography variant="caption" color="text.secondary">
                  Before
                </Typography>
                <Typography variant="h5">
                  {(simulation.hit_rate_before * 100).toFixed(0)}%
                </Typography>
              </Paper>
            </Grid>
            <Grid item xs={2} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <TrendingUp
                color={hitRateImprovement > 0 ? 'success' : 'error'}
                sx={{ fontSize: 32 }}
              />
            </Grid>
            <Grid item xs={5}>
              <Paper
                variant="outlined"
                sx={{ p: 2, textAlign: 'center', bgcolor: 'success.light' }}
              >
                <Typography variant="caption" color="text.secondary">
                  After
                </Typography>
                <Typography variant="h5" color="success.dark" fontWeight="bold">
                  {(simulation.hit_rate_after * 100).toFixed(0)}%
                </Typography>
              </Paper>
            </Grid>
          </Grid>
        </Box>

        {/* Confidence Bar */}
        <Box sx={{ mt: 2 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" mb={0.5}>
            <Typography variant="caption" color="text.secondary">
              Simulation Confidence
            </Typography>
            <Typography variant="body2">
              {(simulation.confidence * 100).toFixed(0)}%
            </Typography>
          </Stack>
          <LinearProgress
            variant="determinate"
            value={simulation.confidence * 100}
            color={simulation.confidence >= 0.7 ? 'success' : 'warning'}
            sx={{ height: 6, borderRadius: 3 }}
          />
        </Box>
      </CardContent>
    </Card>
  );
};

export default OptimizationMLCard;
