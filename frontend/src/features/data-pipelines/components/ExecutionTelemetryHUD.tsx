import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Chip,
  LinearProgress,
  IconButton,
  Button,
  useTheme,
} from '@mui/material';
import {
  Activity,
  CheckCircle2,
  AlertCircle,
  Clock,
  Zap,
  Layers,
  FileCode,
  X,
} from 'lucide-react';
import { PipelineExecutionRun } from '../types/pipeline';

interface ExecutionTelemetryHUDProps {
  run: PipelineExecutionRun | null;
  onClose: () => void;
  onViewSample?: () => void;
}

export const ExecutionTelemetryHUD: React.FC<ExecutionTelemetryHUDProps> = ({
  run,
  onClose,
  onViewSample,
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  if (!run) return null;

  const isRunning = run.status === 'running';
  const isCompleted = run.status === 'completed' || run.status === 'simulated';
  const isFailed = run.status === 'failed';

  return (
    <Paper
      elevation={4}
      sx={{
        position: 'absolute',
        bottom: 24,
        left: 334,
        right: 24,
        zIndex: 15,
        p: 2,
        backgroundColor: theme.palette.background.paper,
        borderRadius: '12px',
        border: `1px solid ${theme.palette.divider}`,
        boxShadow: isDark ? '0 8px 32px rgba(0,0,0,0.5)' : '0 4px 20px rgba(0,0,0,0.08)',
      }}
    >
      {/* Top row */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Box
            sx={{
              width: 36,
              height: 36,
              borderRadius: '8px',
              backgroundColor: isRunning
                ? 'rgba(59, 130, 246, 0.15)'
                : isCompleted
                ? 'rgba(16, 185, 129, 0.15)'
                : 'rgba(239, 68, 68, 0.15)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: isRunning ? '#3b82f6' : isCompleted ? '#10b981' : '#ef4444',
            }}
          >
            <Activity size={20} className={isRunning ? 'animate-pulse' : ''} />
          </Box>
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 800, fontSize: '0.9rem', color: theme.palette.text.primary }}>
              Execution Telemetry HUD
            </Typography>
            <Typography variant="caption" sx={{ color: theme.palette.text.secondary }}>
              Run ID: {run.run_id} | Mode: <strong style={{ textTransform: 'capitalize' }}>{run.status}</strong>
            </Typography>
          </Box>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          {/* Key Metrics */}
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
            <Box sx={{ textAlign: 'right' }}>
              <Typography variant="caption" sx={{ color: theme.palette.text.secondary, display: 'block', fontSize: '0.65rem' }}>
                PROCESSED ROWS
              </Typography>
              <Typography variant="subtitle2" sx={{ fontWeight: 800, color: theme.palette.text.primary }}>
                {run.total_records_out.toLocaleString()}
              </Typography>
            </Box>

            <Box sx={{ textAlign: 'right' }}>
              <Typography variant="caption" sx={{ color: theme.palette.text.secondary, display: 'block', fontSize: '0.65rem' }}>
                PEAK THROUGHPUT
              </Typography>
              <Typography variant="subtitle2" sx={{ fontWeight: 800, color: '#3b82f6' }}>
                {Math.round(run.peak_throughput_rows_sec).toLocaleString()} rows/sec
              </Typography>
            </Box>

            <Box sx={{ textAlign: 'right' }}>
              <Typography variant="caption" sx={{ color: theme.palette.text.secondary, display: 'block', fontSize: '0.65rem' }}>
                ERRORS
              </Typography>
              <Typography variant="subtitle2" sx={{ fontWeight: 800, color: run.total_errors > 0 ? '#ef4444' : '#10b981' }}>
                {run.total_errors}
              </Typography>
            </Box>
          </Box>

          {onViewSample && run.sample_output && run.sample_output.length > 0 && (
            <Button
              size="small"
              variant="outlined"
              startIcon={<FileCode size={14} />}
              onClick={onViewSample}
              sx={{ ml: 1 }}
            >
              Inspect Output ({run.sample_output.length})
            </Button>
          )}

          <IconButton size="small" onClick={onClose}>
            <X size={16} color={isDark ? '#94a3b8' : '#475569'} />
          </IconButton>
        </Box>
      </Box>

      {/* Step breakdown */}
      <Box sx={{ display: 'flex', gap: 1.5, overflowX: 'auto', pt: 1, borderTop: `1px solid ${theme.palette.divider}` }}>
        {Object.entries(run.step_telemetry || {}).map(([nodeId, step]) => (
          <Box
            key={nodeId}
            sx={{
              flex: 1,
              minWidth: 160,
              p: 1.2,
              backgroundColor: isDark ? 'rgba(255, 255, 255, 0.04)' : '#f8fafc',
              borderRadius: '8px',
              border: `1px solid ${theme.palette.divider}`,
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
              <Typography variant="subtitle2" sx={{ fontSize: '0.75rem', fontWeight: 700, color: theme.palette.text.primary }} noWrap>
                {step.node_label}
              </Typography>
              {step.status === 'completed' && <CheckCircle2 size={12} color="#10b981" />}
              {step.status === 'running' && <LinearProgress sx={{ width: 24, height: 4, borderRadius: 2 }} />}
              {step.status === 'failed' && <AlertCircle size={12} color="#ef4444" />}
            </Box>
            <Typography variant="caption" sx={{ color: theme.palette.text.secondary, display: 'block', fontSize: '0.68rem' }}>
              In: {step.records_in.toLocaleString()} → Out: {step.records_out.toLocaleString()}
            </Typography>
            <Typography variant="caption" sx={{ color: '#3b82f6', fontWeight: 600, display: 'block', fontSize: '0.65rem' }}>
              Speed: {Math.round(step.rows_per_sec || 0).toLocaleString()} r/s
            </Typography>
          </Box>
        ))}
      </Box>
    </Paper>
  );
};
