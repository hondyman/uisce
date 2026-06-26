/**
 * SimulationProgress Component
 * 
 * Live execution dashboard showing real-time simulation progress,
 * with portfolio processing status and configuration summary.
 * 
 * Features:
 * - ✅ Live progress bar with percentage
 * - ✅ Real-time results table (completed portfolios)
 * - ✅ Configuration sidebar summary
 * - ✅ Abort simulation capability
 * - ✅ Material UI Box layout
 * - ✅ Dark mode support
 * - ✅ Responsive design
 * 
 * @example
 * <SimulationProgress
 *   simulationRun={run}
 *   results={results}
 *   onAbort={handleAbort}
 * />
 */

import React, { useMemo } from 'react';
import {
  Box,
  Paper,
  Button,
  Typography,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Card,
  CardContent,
  useTheme,
} from '@mui/material';
import {
  Download as DownloadIcon,
  Schedule as ScheduleIcon,
  CheckCircle as SuccessIcon,
} from '@mui/icons-material';
import { SimulationRun, SimulationResult } from '../../../types/scenarios';

interface SimulationProgressProps {
  simulationRun: SimulationRun;
  results: SimulationResult[];
  onAbort: () => Promise<void>;
  isAborting?: boolean;
}

/**
 * Format elapsed time (ms to "Xm Ys" or "Xs")
 */
const formatElapsedTime = (ms: number): string => {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;

  if (minutes > 0) {
    return `${minutes}m ${remainingSeconds}s`;
  }
  return `${remainingSeconds}s`;
};

/**
 * Calculate elapsed time from start
 */
const getElapsedTime = (startedAt: Date): number => {
  return Date.now() - new Date(startedAt).getTime();
};

export const SimulationProgress: React.FC<SimulationProgressProps> = ({
  simulationRun,
  results,
  onAbort,
  isAborting = false,
}) => {
  const theme = useTheme();

  // Calculate elapsed time
  const elapsedMs = useMemo(() => getElapsedTime(simulationRun.startedAt), [
    simulationRun.startedAt,
  ]);

  // Separate completed results
  const completedResults = useMemo(
    () => results.filter((r) => r.validationStatus !== 'error'),
    [results]
  );

  // Calculate averages
  const avgPnL = useMemo(() => {
    if (completedResults.length === 0) return 0;
    const total = completedResults.reduce((sum, r) => sum + r.simulatedPnL, 0);
    return total / completedResults.length;
  }, [completedResults]);

  const avgConfidence = useMemo(() => {
    if (completedResults.length === 0) return 0;
    const total = completedResults.reduce((sum, r) => sum + r.confidenceLevel, 0);
    return Math.round(total / completedResults.length);
  }, [completedResults]);

  return (
    <Box sx={{ display: 'flex', height: '100%', gap: 3, p: 3 }}>
      {/* Left Sidebar: Configuration */}
      <Box
        sx={{
          width: 280,
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
          borderRight: 1,
          borderColor: 'divider',
          pr: 3,
        }}
      >
        {/* Header */}
        <Box>
          <Typography variant="h6" sx={{ fontWeight: 600, mb: 0.5 }}>
            {simulationRun.scenarioId}
          </Typography>
          <Typography variant="caption" sx={{ color: 'text.secondary' }}>
            Configuration
          </Typography>
        </Box>

        {/* Status Card */}
        <Paper
          elevation={0}
          sx={{
            p: 2,
            bgcolor:
              simulationRun.status === 'completed'
                ? 'success.lighter'
                : simulationRun.status === 'failed'
                ? 'error.lighter'
                : 'info.lighter',
            border: 1,
            borderColor:
              simulationRun.status === 'completed'
                ? 'success.light'
                : simulationRun.status === 'failed'
                ? 'error.light'
                : 'info.light',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
            {simulationRun.status === 'running' && (
              <Box
                sx={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  backgroundColor: 'info.main',
                  animation: 'pulse 2s infinite',
                }}
              />
            )}
            {simulationRun.status === 'completed' && (
              <SuccessIcon sx={{ color: 'success.main', fontSize: 20 }} />
            )}
            <Typography variant="caption" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>
              {simulationRun.status}
            </Typography>
          </Box>
          <Typography
            variant="body2"
            sx={{
              color:
                simulationRun.status === 'completed'
                  ? 'success.dark'
                  : simulationRun.status === 'failed'
                  ? 'error.main'
                  : 'info.main',
              fontWeight: 500,
            }}
          >
            {simulationRun.status === 'running'
              ? 'Executing...'
              : simulationRun.status === 'completed'
              ? 'Completed'
              : 'In Progress'}
          </Typography>
        </Paper>

        {/* Portfolio Processing */}
        <Card variant="outlined">
          <CardContent sx={{ pb: 1 }}>
            <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary' }}>
              PORTFOLIOS PROCESSED
            </Typography>
            <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1, mt: 1 }}>
              <Typography variant="h5" sx={{ fontWeight: 700 }}>
                {simulationRun.portfoliosProcessed}
              </Typography>
              <Typography variant="body2" sx={{ color: 'text.secondary' }}>
                / {simulationRun.portfoliosTotal}
              </Typography>
            </Box>
          </CardContent>
        </Card>

        {/* Elapsed Time */}
        <Card variant="outlined">
          <CardContent sx={{ pb: 1 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
              <ScheduleIcon sx={{ fontSize: 18, color: 'text.secondary' }} />
              <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary' }}>
                ELAPSED TIME
              </Typography>
            </Box>
            <Typography variant="h5" sx={{ fontWeight: 700 }}>
              {formatElapsedTime(elapsedMs)}
            </Typography>
            <Typography variant="caption" sx={{ color: 'text.secondary', mt: 0.5 }}>
              Est. {simulationRun.estimatedDuration}s total
            </Typography>
          </CardContent>
        </Card>

        {/* Abort Button */}
        <Button
          fullWidth
          variant="outlined"
          color="error"
          onClick={onAbort}
          disabled={isAborting || simulationRun.status === 'completed'}
        >
          Abort Simulation
        </Button>
      </Box>

      {/* Right Main Content */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 3 }}>
        {/* Progress Section */}
        <Paper elevation={1} sx={{ p: 3 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Box>
              <Typography variant="body2" sx={{ fontWeight: 500, mb: 0.5 }}>
                Calculating VaR &amp; Scenario PnL using WASM engine...
              </Typography>
              <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                Core affinity: Performance optimized (8 threads)
              </Typography>
            </Box>
            <Typography variant="h4" sx={{ fontWeight: 700, color: 'primary.main' }}>
              {simulationRun.progress}%
            </Typography>
          </Box>

          {/* Progress Bar */}
          <LinearProgress
            variant="determinate"
            value={simulationRun.progress}
            sx={{
              height: 4,
              borderRadius: 2,
              backgroundColor: 'action.disabledBackground',
              '& .MuiLinearProgress-bar': {
                borderRadius: 2,
                backgroundColor: 'primary.main',
              },
            }}
          />
        </Paper>

        {/* Summary Stats */}
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
          <Card variant="outlined">
            <CardContent>
              <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary' }}>
                AVG PnL
              </Typography>
              <Typography
                variant="h6"
                sx={{
                  fontWeight: 700,
                  mt: 0.5,
                  color: avgPnL < 0 ? 'error.main' : 'success.main',
                }}
              >
                ${avgPnL.toFixed(1)}M
              </Typography>
            </CardContent>
          </Card>

          <Card variant="outlined">
            <CardContent>
              <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary' }}>
                AVG CONFIDENCE
              </Typography>
              <Typography variant="h6" sx={{ fontWeight: 700, mt: 0.5 }}>
                {avgConfidence}%
              </Typography>
            </CardContent>
          </Card>
        </Box>

        {/* Results Preview Table */}
        <Paper elevation={1}>
          <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider', bgcolor: 'background.default' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                Live Results Preview
              </Typography>
              <Box sx={{ display: 'flex', gap: 2, fontSize: '0.75rem', color: 'text.secondary' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Box
                    sx={{
                      width: 8,
                      height: 8,
                      borderRadius: '50%',
                      backgroundColor: 'success.main',
                    }}
                  />
                  {completedResults.length} Completed
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Box
                    sx={{
                      width: 8,
                      height: 8,
                      borderRadius: '50%',
                      backgroundColor: 'info.main',
                      animation: 'pulse 2s infinite',
                    }}
                  />
                  {simulationRun.portfoliosTotal - simulationRun.portfoliosProcessed} Processing
                </Box>
              </Box>
            </Box>
          </Box>

          <TableContainer>
            <Table size="small" stickyHeader>
              <TableHead>
                <TableRow sx={{ backgroundColor: 'background.default' }}>
                  <TableCell sx={{ fontWeight: 600, fontSize: '0.75rem' }}>Portfolio Name</TableCell>
                  <TableCell sx={{ fontWeight: 600, fontSize: '0.75rem' }}>Status</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600, fontSize: '0.75rem' }}>
                    Simulated PnL
                  </TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600, fontSize: '0.75rem' }}>
                    Confidence
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {completedResults.slice(0, 5).map((result) => (
                  <TableRow key={result.id} hover>
                    <TableCell sx={{ py: 1, fontSize: '0.875rem' }}>
                      {result.portfolioName}
                    </TableCell>
                    <TableCell sx={{ py: 1 }}>
                      <Chip
                        label="Success"
                        size="small"
                        sx={{
                          backgroundColor: 'success.lighter',
                          color: 'success.dark',
                          fontWeight: 500,
                          height: 24,
                        }}
                      />
                    </TableCell>
                    <TableCell
                      align="right"
                      sx={{
                        py: 1,
                        fontWeight: 600,
                        color: result.simulatedPnL < 0 ? 'error.main' : 'success.main',
                      }}
                    >
                      ${result.simulatedPnL.toFixed(1)}M
                    </TableCell>
                    <TableCell align="right" sx={{ py: 1, fontSize: '0.875rem' }}>
                      {result.confidenceLevel}%
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>

          {completedResults.length > 5 && (
            <Box sx={{ p: 2, textAlign: 'center', borderTop: 1, borderColor: 'divider' }}>
              <Button size="small" sx={{ textTransform: 'uppercase', fontWeight: 600 }}>
                View all {completedResults.length} results
              </Button>
            </Box>
          )}
        </Paper>
      </Box>
    </Box>
  );
};

export default SimulationProgress;
