import React from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Chip,
  Grid,
  Stack,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableRow,
  Paper,
  Button,
} from '@mui/material';
import {
  AutoFixHigh,
  TrendingUp,
  Speed,
  Schedule,
  Storage,
  QueryStats,
  CheckCircle,
  Warning,
  Info,
  Science,
} from '@mui/icons-material';
import { useParams } from 'react-router-dom';
import { ASOOptimization, useASOOptimization } from '../../hooks/useASO';
import OptimizationMLCard, { MLEvidence, ExplainPayload } from './OptimizationMLCard';
import { OptimizationExplainabilityCard, OptimizationSimulationCard } from './OptimizationMLCard';
import OptimizationActions from './OptimizationActions';

interface ASOOptimizationDetailProps {
  optimization?: ASOOptimization;
  onApply?: () => Promise<void>;
  onApprove?: () => Promise<void>;
  onReject?: (reason: string) => Promise<void>;
  onSimulate?: () => void;
}

export const ASOOptimizationDetail: React.FC<ASOOptimizationDetailProps> = ({
  optimization: propOptimization,
  onApply,
  onApprove,
  onReject,
  onSimulate,
}) => {
  const { optimizationId } = useParams<{ optimizationId: string }>();
  const { optimization: fetchedOptimization, loading, error } = useASOOptimization(propOptimization ? undefined : optimizationId);

  const optimization = propOptimization || fetchedOptimization;

  if (loading) {
    return <Box sx={{ p: 4 }}><Typography>Loading optimization details...</Typography></Box>;
  }

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }

  if (!optimization) {
     return <Alert severity="warning">Optimization not found</Alert>;
  }

  // Parse details
  const details = optimization.details || {};
  
  // Extract Explain Payload if available
  const explain = details.explain as ExplainPayload | undefined;
  
  // Construct ML Evidence if not in explain payload but available on optimization
  const mlEvidence: MLEvidence | undefined = explain?.ml || (optimization.ml_score !== undefined ? {
    score: optimization.ml_score,
    confidence: optimization.confidence || 0.8,
    predicted_speedup: optimization.predicted_speedup || 0,
    predicted_cost_savings: optimization.predicted_cost_savings || 0,
    risk_score: optimization.risk_score || 0,
    top_factors: optimization.top_factors || [],
  } as MLEvidence : undefined);

  // Construct Simulation Evidence
  const simulationEvidence = explain?.simulation;

  return (
    <Box>
      {/* Header */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Stack direction="row" spacing={2} alignItems="center">
              <AutoFixHigh sx={{ fontSize: 40, color: 'primary.main' }} />
              <Box>
                <Typography variant="h5" fontWeight="bold">
                  {formatOptimizationType(optimization.optimization_type)}
                </Typography>
                <Typography color="text.secondary">
                  Target: {optimization.target_name} ({optimization.target_type})
                </Typography>
              </Box>
            </Stack>
            <Stack direction="row" spacing={1} alignItems="center">
              <Chip label={optimization.env} size="small" />
              <Chip
                label={optimization.scope}
                size="small"
                color={optimization.scope === 'core' ? 'primary' : 'default'}
                variant="outlined"
              />
              <StatusChip status={optimization.status} />
            </Stack>
          </Stack>
          
          <Box sx={{ mt: 2, display: 'flex', justifyContent: 'flex-end' }}>
            <OptimizationActions 
              optimization={optimization}
              onApprove={onApprove}
              onApply={onApply}
              onReject={onReject}
              onSimulate={onSimulate}
            />
          </Box>
        </CardContent>
      </Card>

      {/* ML & Explainability Section (New) */}
      <Grid container spacing={3} mb={3}>
        {mlEvidence && (
          <Grid item xs={12}>
            <OptimizationMLCard ml={mlEvidence} />
          </Grid>
        )}
        {explain && (
          <Grid item xs={12}>
            <OptimizationExplainabilityCard explain={explain} />
          </Grid>
        )}
        {simulationEvidence && (
          <Grid item xs={12}>
            <OptimizationSimulationCard simulation={simulationEvidence} />
          </Grid>
        )}
      </Grid>

      {/* Legacy/Standard Details */}
      <Grid container spacing={3}>
        <Grid item xs={12}>
          <Card sx={{ mb: 3 }}>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                <Info sx={{ mr: 1, verticalAlign: 'middle' }} />
                Optimization Details
              </Typography>
              <Alert severity="info" sx={{ mb: 2 }}>
                {optimization.reason}
              </Alert>

              {/* Workload Metrics */}
              <Typography variant="subtitle2" gutterBottom sx={{ mt: 2 }}>
                Workload Metrics (Last {optimization.workload_window_days} Days)
              </Typography>
              <Grid container spacing={2}>
                {optimization.queries_per_day !== undefined && (
                  <Grid item xs={6} md={3}>
                    <MetricBox
                      label="Queries/Day"
                      value={optimization.queries_per_day.toFixed(0)}
                      icon={<QueryStats />}
                    />
                  </Grid>
                )}
                {optimization.avg_latency_ms !== undefined && (
                  <Grid item xs={6} md={3}>
                    <MetricBox
                      label="Avg Latency"
                      value={`${optimization.avg_latency_ms.toFixed(0)}ms`}
                      icon={<Speed />}
                    />
                  </Grid>
                )}
                {optimization.p95_latency_ms !== undefined && (
                  <Grid item xs={6} md={3}>
                    <MetricBox
                      label="P95 Latency"
                      value={`${optimization.p95_latency_ms.toFixed(0)}ms`}
                      icon={<Speed />}
                      highlight={optimization.p95_latency_ms > 1000}
                    />
                  </Grid>
                )}
              </Grid>
            </CardContent>
          </Card>

          {/* Type Specific Details */}
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                <TrendingUp sx={{ mr: 1, verticalAlign: 'middle' }} />
                Implementation Plan
              </Typography>
              {optimization.optimization_type === 'create_preagg' && (
                <CreatePreAggDetails details={details} />
              )}
              {optimization.optimization_type === 'tune_refresh' && (
                <TuneRefreshDetails details={details} />
              )}
              {optimization.optimization_type === 'retire_asset' && (
                <RetireAssetDetails details={details} />
              )}
              {optimization.optimization_type === 'prewarm' && (
                <PrewarmDetails details={details} />
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

// ============================================================================
// Helper Components (Reused)
// ============================================================================

const StatusChip: React.FC<{ status: string }> = ({ status }) => {
  const config: Record<string, { color: 'success' | 'warning' | 'error' | 'info' | 'default'; icon: React.ReactNode }> = {
    proposed: { color: 'warning', icon: <Warning /> },
    approved: { color: 'info', icon: <CheckCircle /> },
    applied: { color: 'success', icon: <CheckCircle /> },
    rejected: { color: 'error', icon: <Warning /> },
    failed: { color: 'error', icon: <Warning /> },
    superseded: { color: 'default', icon: null },
  };
  const c = config[status] || { color: 'default', icon: null };
  return <Chip label={status} color={c.color} size="small" />;
};

interface MetricBoxProps {
  label: string;
  value: string | number;
  icon: React.ReactNode;
  highlight?: boolean;
}

const MetricBox: React.FC<MetricBoxProps> = ({ label, value, icon, highlight }) => (
  <Paper variant="outlined" sx={{ p: 2, bgcolor: highlight ? 'error.light' : 'transparent' }}>
    <Stack direction="row" spacing={1} alignItems="center">
      {icon}
      <Box>
        <Typography variant="caption" color="text.secondary">
          {label}
        </Typography>
        <Typography variant="h6" fontWeight="bold">
          {value}
        </Typography>
      </Box>
    </Stack>
  </Paper>
);

// ============================================================================
// Detail Components by Type
// ============================================================================

const CreatePreAggDetails: React.FC<{ details: Record<string, any> }> = ({ details }) => (
  <Box>
    <Table size="small">
      <TableBody>
        <TableRow>
          <TableCell>Grain</TableCell>
          <TableCell>{(details.grain || []).join(', ')}</TableCell>
        </TableRow>
        <TableRow>
          <TableCell>Measures</TableCell>
          <TableCell>{(details.measures || []).join(', ')}</TableCell>
        </TableRow>
        {details.filters?.length > 0 && (
          <TableRow>
            <TableCell>Filters</TableCell>
            <TableCell>{details.filters.join(', ')}</TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
    
    {details.cost_estimate && (
      <Box sx={{ mt: 2 }}>
        <Typography variant="subtitle2" gutterBottom>Estimated Resources</Typography>
        <Grid container spacing={2}>
           <Grid item xs={4}>
             <Typography variant="caption">Storage</Typography>
             <Typography variant="body1">{formatBytes(details.cost_estimate.estimated_storage_bytes)}</Typography>
           </Grid>
           <Grid item xs={4}>
             <Typography variant="caption">Refresh Cost</Typography>
             <Typography variant="body1">${details.cost_estimate.estimated_refresh_cost?.toFixed(2)}</Typography>
           </Grid>
        </Grid>
      </Box>
    )}
  </Box>
);

const TuneRefreshDetails: React.FC<{ details: Record<string, any> }> = ({ details }) => (
  <Table size="small">
    <TableBody>
      <TableRow>
        <TableCell>Current</TableCell>
        <TableCell>{details.current_refresh_interval}</TableCell>
      </TableRow>
      <TableRow>
        <TableCell>Proposed</TableCell>
        <TableCell sx={{ color: 'success.main', fontWeight: 'bold' }}>
          {details.proposed_refresh_interval}
        </TableCell>
      </TableRow>
    </TableBody>
  </Table>
);

const RetireAssetDetails: React.FC<{ details: Record<string, any> }> = ({ details }) => (
   <Typography variant="body2">
     Asset unused since {details.last_used_at}. Storage: {formatBytes(details.storage_bytes)}.
   </Typography>
);

const PrewarmDetails: React.FC<{ details: Record<string, any> }> = ({ details }) => (
  <Table size="small">
    <TableBody>
      <TableRow>
        <TableCell>Schedule</TableCell>
        <TableCell>{details.proposed_schedule}</TableCell>
      </TableRow>
    </TableBody>
  </Table>
);

function formatOptimizationType(type: string): string {
  return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
}

function formatBytes(bytes: number): string {
  if (!bytes) return '0 B';
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB';
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB';
  if (bytes >= 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return bytes + ' B';
}

export default ASOOptimizationDetail;
