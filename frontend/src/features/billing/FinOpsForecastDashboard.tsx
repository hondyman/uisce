import React, { useState, useEffect, useCallback } from 'react';
import {
  Paper,
  Box,
  Typography,
  Stack,
  Grid,
  Card,
  CardHeader,
  CardContent,
  Chip,
  LinearProgress,
  Button,
  Alert,
  Skeleton,
  Divider,
  Tooltip,
  ButtonGroup,
  TextField,
} from '@mui/material';
import AutoModeIcon from '@mui/icons-material/AutoMode';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import BoltIcon from '@mui/icons-material/Bolt';
import SavingsIcon from '@mui/icons-material/Savings';
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import RefreshIcon from '@mui/icons-material/Refresh';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import StorageIcon from '@mui/icons-material/Storage';
import ScheduleIcon from '@mui/icons-material/Schedule';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';
import TuneIcon from '@mui/icons-material/Tune';

// ── Types ────────────────────────────────────────────────────────────────────

type ForecastOutcome = 'ACCURATE' | 'FALSE_POSITIVE' | 'MISSED_SPIKE' | 'PARTIAL_SPIKE';

interface ForecastResult {
  tenantId: string;
  windowStart: string;
  windowEnd: string;
  projectedBytes: number;
  projectedCpuMs: number;
  projectedCostUsd: number;
  confidenceScore: number;
  peakProbability: number;
  contributingFactors: string[];
  calibrationFactor: number;
  calibrationSamples: number;
}

interface CalibrationState {
  tenantId: string;
  calibrationFactor: number;
  sampleCount: number;
  lastComputedAt: string;
}

interface PrewarmResult {
  jobId?: string;
  triggered: boolean;
  peakProbability: number;
  targetsSeeded: number;
  computeCostIncurredUsd: number;
  estimatedPeakSavingsUsd: number;
  status: string;
}

interface WorkloadSmoothingPolicy {
  policyId?: string;
  policyName: string;
  isActive: boolean;
  offPeakCron: string;
  prewarmThresholdMultiplier: number;
  enableBurstDeferral: boolean;
  maxDeferralMinutes: number;
  minPeakProbabilityToPrewarm: number;
}

// ── Utility helpers ──────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
  if (bytes >= 1e6) return `${(bytes / 1e6).toFixed(1)} MB`;
  return `${bytes.toLocaleString()} B`;
}

function peakColor(prob: number): string {
  if (prob >= 0.8) return '#EF4444'; // red
  if (prob >= 0.5) return '#F59E0B'; // amber
  return '#10B981';                  // green
}

function statusChipColor(status: string): { bg: string; fg: string } {
  switch (status) {
    case 'COMPLETED':      return { bg: '#064E3B', fg: '#34D399' };
    case 'SIMULATED':      return { bg: '#1E1B4B', fg: '#818CF8' };
    case 'QUEUED':
    case 'PENDING':        return { bg: '#0C4A6E', fg: '#38BDF8' };
    case 'PARTIAL':        return { bg: '#78350F', fg: '#FCD34D' };
    case 'FAILED':         return { bg: '#7F1D1D', fg: '#FCA5A5' };
    case 'SKIPPED_BELOW_THRESHOLD':
    case 'SKIPPED_NO_TARGETS':
    case 'SKIPPED_ALREADY_IN_FLIGHT':
      return { bg: '#1E293B', fg: '#94A3B8' };
    default:               return { bg: '#1E293B', fg: '#E2E8F0' };
  }
}

// ── Component ────────────────────────────────────────────────────────────────

interface FinOpsForecastDashboardProps {
  tenantId: string;
}

/**
 * FinOpsForecastDashboard
 *
 * Displays the Autonomous Compute Forecasting & Workload Smoothing control plane.
 * Includes the forecast feedback loop: operators mark outcomes (ACCURATE /
 * FALSE_POSITIVE / MISSED_SPIKE / PARTIAL_SPIKE) which feed back into future
 * predictions via a rolling calibration factor.
 */
export const FinOpsForecastDashboard: React.FC<FinOpsForecastDashboardProps> = ({
  tenantId,
}) => {
  const [forecast, setForecast] = useState<ForecastResult | null>(null);
  const [policy, setPolicy] = useState<WorkloadSmoothingPolicy | null>(null);
  const [prewarmResult, setPrewarmResult] = useState<PrewarmResult | null>(null);
  const [calibration, setCalibration] = useState<CalibrationState | null>(null);
  const [feedbackOutcome, setFeedbackOutcome] = useState<ForecastOutcome | null>(null);
  const [feedbackActualCost, setFeedbackActualCost] = useState('');
  const [feedbackSubmitting, setFeedbackSubmitting] = useState(false);
  const [feedbackSubmitted, setFeedbackSubmitted] = useState(false);
  const [loadingForecast, setLoadingForecast] = useState(false);
  const [loadingPrewarm, setLoadingPrewarm] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // ── Data fetching ──────────────────────────────────────────────────────────

  const fetchForecast = useCallback(async () => {
    setLoadingForecast(true);
    setError(null);
    setFeedbackSubmitted(false);
    setFeedbackOutcome(null);
    try {
      const res = await fetch('/api/finops/forecast/today', {
        headers: { 'Content-Type': 'application/json' },
      });
      if (!res.ok) throw new Error(`Forecast fetch failed (${res.status})`);
      const data: ForecastResult = await res.json();
      setForecast(data);
    } catch (e) {
      setError((e as Error).message);
      // Fallback demo values gated strictly behind DEV mode to prevent misleading prod dashboards.
      if (import.meta.env.DEV) {
        setForecast({
          tenantId,
          windowStart: new Date().toISOString(),
          windowEnd: new Date(Date.now() + 86400000).toISOString(),
          projectedBytes: 850_000_000,
          projectedCpuMs: 135_000,
          projectedCostUsd: 420.5,
          confidenceScore: 0.88,
          peakProbability: 0.85,
          contributingFactors: ['CALENDAR_MONTH_END', 'BATCH_REPORT_BURST(4)'],
          calibrationFactor: 1.0,
          calibrationSamples: 0,
        });
      } else {
        setForecast(null);
      }
    } finally {
      setLoadingForecast(false);
    }
  }, [tenantId]);

  const fetchPrewarmStatus = useCallback(async (jobId?: string) => {
    try {
      const url = jobId
        ? `/api/finops/prewarm/status?jobId=${encodeURIComponent(jobId)}`
        : '/api/finops/prewarm/status';
      const res = await fetch(url);
      if (!res.ok) return;
      const data: PrewarmResult = await res.json();
      setPrewarmResult(data);
    } catch {
      // Non-fatal.
    }
  }, []);

  const fetchCalibration = useCallback(async () => {
    try {
      const res = await fetch('/api/finops/calibration');
      if (!res.ok) return;
      const data: CalibrationState = await res.json();
      setCalibration(data);
    } catch {
      // Non-fatal.
    }
  }, []);

  const fetchPolicy = useCallback(async () => {
    try {
      const res = await fetch('/api/finops/smoothing-policy');
      if (!res.ok) return;
      const data: WorkloadSmoothingPolicy = await res.json();
      setPolicy(data);
    } catch {
      // Non-fatal — policy display is optional UX.
    }
  }, []);

  useEffect(() => {
    fetchForecast();
    fetchPolicy();
    fetchCalibration();
    fetchPrewarmStatus();
  }, [fetchForecast, fetchPolicy, fetchCalibration, fetchPrewarmStatus]);

  // Poll prewarm status specifically for this job while PENDING or QUEUED until terminal status
  // Capped at 15 minutes to align with the backend 15-minute coordinator sweep threshold.
  useEffect(() => {
    const isRunning = prewarmResult?.status === 'PENDING' || prewarmResult?.status === 'QUEUED';
    if (!isRunning) return;

    let isMounted = true;
    const currentJobId = prewarmResult?.jobId;
    const startTime = Date.now();
    const maxPollDurationMs = 15 * 60 * 1000; // 15 minutes (aligns with backend sweep threshold)

    const timer = setInterval(async () => {
      if (!isMounted) return;
      if (Date.now() - startTime > maxPollDurationMs) {
        clearInterval(timer);
        setPrewarmResult((prev) =>
          prev
            ? { ...prev, status: 'TIMEOUT' }
            : null
        );
        setError('Pre-warming job execution exceeded maximum wait time (15m).');
        return;
      }
      await fetchPrewarmStatus(currentJobId);
    }, 2500);

    return () => {
      isMounted = false;
      clearInterval(timer);
    };
  }, [prewarmResult?.status, prewarmResult?.jobId, fetchPrewarmStatus]);

  // ── Actions ────────────────────────────────────────────────────────────────

  const triggerPrewarm = async () => {
    setLoadingPrewarm(true);
    try {
      const res = await fetch('/api/finops/prewarm/trigger', { method: 'POST' });
      if (res.status === 409) {
        setError('A pre-warming execution is already running for your tenant.');
        return;
      }
      if (!res.ok) throw new Error(`Prewarm trigger failed (${res.status})`);
      const data = await res.json();
      if (res.status === 202) {
        setPrewarmResult({
          jobId: data.jobId,
          triggered: true,
          peakProbability: forecast?.peakProbability ?? 0,
          targetsSeeded: 0,
          computeCostIncurredUsd: 0,
          estimatedPeakSavingsUsd: 0,
          status: 'PENDING',
        });
      } else {
        setPrewarmResult(data as PrewarmResult);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoadingPrewarm(false);
    }
  };

  const submitFeedback = async (outcome: ForecastOutcome) => {
    if (!forecast) return;
    setFeedbackOutcome(outcome);
    setFeedbackSubmitting(true);
    try {
      const body: Record<string, unknown> = { outcome };
      const parsed = parseFloat(feedbackActualCost);
      if (!isNaN(parsed) && parsed > 0) body.actualCostUsd = parsed;

      // forecast.forecastId comes from the persisted record — fall back to a best-effort call.
      const forecastId = (forecast as ForecastResult & { forecastId?: string }).forecastId ?? 'unknown';
      const res = await fetch(`/api/finops/forecast/${forecastId}/feedback`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(`Feedback submission failed (${res.status})`);
      setFeedbackSubmitted(true);
      // Refresh calibration state immediately so the badge updates.
      await fetchCalibration();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setFeedbackSubmitting(false);
    }
  };


  // ── Render ─────────────────────────────────────────────────────────────────

  const pc = peakColor(forecast?.peakProbability ?? 0);
  const estimatedSavings = forecast
    ? Math.max(0, forecast.projectedCostUsd * (forecast.peakProbability * 8) - forecast.projectedCostUsd)
    : 0;

  return (
    <Paper
      sx={{
        p: 3,
        bgcolor: '#071526',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2,
      }}
    >
      {/* ── Header ─────────────────────────────────────────────────────────── */}
      <Stack
        direction="row"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
        pb={2}
        borderBottom="1px solid #1E293B"
      >
        <Stack direction="row" spacing={1.5} alignItems="center">
          <AutoModeIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, lineHeight: 1.3 }}>
              Autonomous Compute Forecasting &amp; Workload Leveler
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Predictive FinOps · Off-peak cache pre-warming · Burst deferral
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={1} alignItems="center">
          <Chip
            label="Engine: Adaptive Smoothing Active"
            size="small"
            sx={{
              bgcolor: 'rgba(0,212,255,0.1)',
              color: '#38BDF8',
              fontWeight: 600,
              fontSize: 11,
            }}
          />
          <Tooltip title="Refresh forecast">
            <span>
              <Button
                size="small"
                variant="outlined"
                onClick={fetchForecast}
                disabled={loadingForecast}
                sx={{ minWidth: 0, px: 1, borderColor: '#1E293B', color: '#64748B' }}
              >
                <RefreshIcon fontSize="small" />
              </Button>
            </span>
          </Tooltip>
        </Stack>
      </Stack>

      {/* ── Error banner ───────────────────────────────────────────────────── */}
      {error && (
        <Alert
          severity={import.meta.env.DEV ? 'warning' : 'error'}
          sx={{
            mb: 2,
            bgcolor: import.meta.env.DEV ? 'rgba(245,158,11,0.08)' : 'rgba(239,68,68,0.08)',
            color: import.meta.env.DEV ? '#FCD34D' : '#FCA5A5',
            border: `1px solid ${import.meta.env.DEV ? 'rgba(245,158,11,0.25)' : 'rgba(239,68,68,0.25)'}`,
          }}
          onClose={() => setError(null)}
        >
          {error}{import.meta.env.DEV ? ' — displaying demo values.' : ' — forecast telemetry unavailable.'}
        </Alert>
      )}

      {/* ── Metric cards ───────────────────────────────────────────────────── */}
      <Grid container spacing={3} mb={3}>

        {/* Card 1: Next Peak Probability */}
        <Grid item xs={12} md={4}>
          <Card sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', color: '#F8FAFC', height: '100%' }}>
            <CardHeader
              title="Next Peak Probability"
              titleTypographyProps={{ variant: 'subtitle2', color: '#94A3B8' }}
              action={<TrendingUpIcon sx={{ color: pc }} />}
              sx={{ pb: 0 }}
            />
            <CardContent sx={{ pt: 1 }}>
              {loadingForecast ? (
                <Skeleton variant="text" width="60%" height={48} sx={{ bgcolor: '#1E293B' }} />
              ) : (
                <Typography variant="h4" sx={{ fontWeight: 700, color: pc, mb: 1 }}>
                  {((forecast?.peakProbability ?? 0) * 100).toFixed(0)}%
                </Typography>
              )}
              <LinearProgress
                variant={loadingForecast ? 'indeterminate' : 'determinate'}
                value={(forecast?.peakProbability ?? 0) * 100}
                sx={{
                  height: 6,
                  borderRadius: 3,
                  bgcolor: '#1E293B',
                  mb: 2,
                  '& .MuiLinearProgress-bar': { bgcolor: pc },
                }}
              />
              <Stack direction="row" spacing={0.5} flexWrap="wrap" useFlexGap>
                {(forecast?.contributingFactors ?? []).map((factor) => (
                  <Chip
                    key={factor}
                    icon={<CalendarMonthIcon sx={{ fontSize: 11, color: '#94A3B8 !important' }} />}
                    label={factor}
                    size="small"
                    sx={{ bgcolor: '#1E293B', color: '#CBD5E1', fontSize: 10, mb: 0.5 }}
                  />
                ))}
              </Stack>
              <Typography variant="caption" sx={{ color: '#475569', display: 'block', mt: 1 }}>
                Confidence: {((forecast?.confidenceScore ?? 0.65) * 100).toFixed(0)}%
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        {/* Card 2: Off-Peak Cache Seeding */}
        <Grid item xs={12} md={4}>
          <Card sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', color: '#F8FAFC', height: '100%' }}>
            <CardHeader
              title="Predictive Cache Seeding"
              titleTypographyProps={{ variant: 'subtitle2', color: '#94A3B8' }}
              action={<BoltIcon sx={{ color: '#00D4FF' }} />}
              sx={{ pb: 0 }}
            />
            <CardContent sx={{ pt: 1 }}>
              <Stack direction="row" spacing={0.75} alignItems="center" mb={0.5}>
                <ScheduleIcon sx={{ fontSize: 16, color: '#38BDF8' }} />
                <Typography variant="h6" sx={{ fontWeight: 700, color: '#38BDF8' }}>
                  {policy?.offPeakCron ?? '0 2 * * *'} UTC (off-peak)
                </Typography>
              </Stack>
              <Typography variant="body2" sx={{ color: '#94A3B8', fontSize: 12, mb: 2 }}>
                Auto-seeds XIRR, modified duration, and NAV rolling cubes into{' '}
                <code style={{ color: '#67E8F9' }}>calc_cache</code>.
                Threshold: {policy?.prewarmThresholdMultiplier ?? 2.5}× baseline.
              </Typography>

              {prewarmResult ? (
                (() => {
                  const sc = statusChipColor(prewarmResult.status);
                  return (
                    <Stack spacing={0.5}>
                      <Chip
                        label={`Status: ${prewarmResult.status}`}
                        size="small"
                        sx={{ bgcolor: sc.bg, color: sc.fg, fontSize: 11, fontWeight: 600, alignSelf: 'flex-start' }}
                      />
                      <Typography variant="caption" sx={{ color: '#64748B' }}>
                        {prewarmResult.targetsSeeded} BOs seeded ·
                        ${prewarmResult.computeCostIncurredUsd.toFixed(2)} incurred
                      </Typography>
                    </Stack>
                  );
                })()
              ) : (
                <Chip
                  label="Armed for Nightly Run"
                  size="small"
                  sx={{ bgcolor: '#064E3B', color: '#34D399', fontSize: 11, fontWeight: 600 }}
                />
              )}

              <Box mt={2}>
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<PlayArrowIcon />}
                  onClick={triggerPrewarm}
                  disabled={loadingPrewarm}
                  sx={{
                    borderColor: '#1E40AF',
                    color: '#60A5FA',
                    fontSize: 11,
                    '&:hover': { bgcolor: 'rgba(96,165,250,0.08)' },
                  }}
                >
                  {loadingPrewarm ? 'Triggering…' : 'Trigger Now'}
                </Button>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* Card 3: Projected FinOps Impact */}
        <Grid item xs={12} md={4}>
          <Card sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', color: '#F8FAFC', height: '100%' }}>
            <CardHeader
              title="Projected FinOps Impact"
              titleTypographyProps={{ variant: 'subtitle2', color: '#94A3B8' }}
              action={<SavingsIcon sx={{ color: '#10B981' }} />}
              sx={{ pb: 0 }}
            />
            <CardContent sx={{ pt: 1 }}>
              <Typography variant="h4" sx={{ fontWeight: 700, color: '#10B981', mb: 0.25 }}>
                ~${estimatedSavings.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
              </Typography>
              <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 2 }}>
                Avoided on-demand StarRocks/Iceberg scaling surcharges via trough utilisation.
              </Typography>

              <Divider sx={{ borderColor: '#1E293B', mb: 1.5 }} />

              <Stack spacing={0.75}>
                <Stack direction="row" justifyContent="space-between">
                  <Stack direction="row" spacing={0.5} alignItems="center">
                    <StorageIcon sx={{ fontSize: 13, color: '#64748B' }} />
                    <Typography variant="caption" sx={{ color: '#64748B' }}>
                      Projected scan volume
                    </Typography>
                  </Stack>
                  <Typography variant="caption" sx={{ color: '#CBD5E1', fontWeight: 600 }}>
                    {forecast ? formatBytes(forecast.projectedBytes) : '—'}
                  </Typography>
                </Stack>
                <Stack direction="row" justifyContent="space-between">
                  <Typography variant="caption" sx={{ color: '#64748B' }}>Projected cost</Typography>
                  <Typography variant="caption" sx={{ color: '#CBD5E1', fontWeight: 600 }}>
                    ${(forecast?.projectedCostUsd ?? 0).toFixed(2)}
                  </Typography>
                </Stack>
                <Stack direction="row" justifyContent="space-between">
                  <Typography variant="caption" sx={{ color: '#64748B' }}>CPU time</Typography>
                  <Typography variant="caption" sx={{ color: '#CBD5E1', fontWeight: 600 }}>
                    {forecast ? `${(forecast.projectedCpuMs / 1000).toFixed(1)} s` : '—'}
                  </Typography>
                </Stack>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* ── Forecast Feedback & Calibration Loop ────────────────────────────── */}
      <Box
        sx={{
          mt: 3,
          p: 2.5,
          bgcolor: '#0B1E36',
          border: '1px solid #1E293B',
          borderRadius: 2,
        }}
      >
        <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
          <Stack direction="row" spacing={1} alignItems="center">
            <TuneIcon sx={{ color: '#A78BFA', fontSize: 20 }} />
            <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
              Forecast Feedback Loop
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              — Was this forecast accurate? Mark the outcome to calibrate future predictions.
            </Typography>
          </Stack>

          {/* Live calibration badge */}
          {(() => {
            const factor = calibration?.calibrationFactor ?? forecast?.calibrationFactor ?? 1.0;
            const samples = calibration?.sampleCount ?? forecast?.calibrationSamples ?? 0;
            const drift = Math.abs(factor - 1.0);
            const badgeColor = drift < 0.05 ? '#10B981' : drift < 0.20 ? '#F59E0B' : '#EF4444';
            const label = factor > 1.05
              ? `↑ ${factor.toFixed(2)}× (under-predicting)`
              : factor < 0.95
              ? `↓ ${factor.toFixed(2)}× (over-predicting)`
              : `✓ ${factor.toFixed(2)}× (calibrated)`;
            return (
              <Tooltip title={`Calibration based on ${samples} feedback sample${samples !== 1 ? 's' : ''}`}>
                <Chip
                  label={label}
                  size="small"
                  sx={{ bgcolor: `${badgeColor}18`, color: badgeColor, fontWeight: 700, fontSize: 11 }}
                />
              </Tooltip>
            );
          })()}
        </Stack>

        {feedbackSubmitted ? (
          <Alert
            severity="success"
            sx={{ bgcolor: 'rgba(16,185,129,0.08)', border: '1px solid rgba(16,185,129,0.25)', color: '#D1FAE5' }}
          >
            <Typography variant="body2" sx={{ fontWeight: 600 }}>
              Feedback recorded — calibration model updated.
            </Typography>
            <Typography variant="caption" sx={{ color: '#6EE7B7' }}>
              Outcome: <strong>{feedbackOutcome}</strong>. The next forecast will apply the
              updated correction factor of{' '}
              <strong>{(calibration?.calibrationFactor ?? 1.0).toFixed(3)}×</strong> based on{' '}
              {calibration?.sampleCount ?? 0} sample{(calibration?.sampleCount ?? 0) !== 1 ? 's' : ''}.
            </Typography>
          </Alert>
        ) : (
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems={{ sm: 'flex-end' }}>
            {/* Outcome selector */}
            <Box flex={1}>
              <Typography variant="caption" sx={{ color: '#64748B', mb: 0.75, display: 'block' }}>
                Outcome
              </Typography>
              <ButtonGroup size="small" variant="outlined" sx={{ flexWrap: 'wrap', gap: 0.5 }}>
                {(
                  [
                    { value: 'ACCURATE' as ForecastOutcome, label: 'Accurate', icon: <ThumbUpIcon sx={{ fontSize: 13 }} />, color: '#10B981' },
                    { value: 'FALSE_POSITIVE' as ForecastOutcome, label: 'False Positive', icon: <ThumbDownIcon sx={{ fontSize: 13 }} />, color: '#F59E0B' },
                    { value: 'MISSED_SPIKE' as ForecastOutcome, label: 'Missed Spike', icon: <TrendingUpIcon sx={{ fontSize: 13 }} />, color: '#EF4444' },
                    { value: 'PARTIAL_SPIKE' as ForecastOutcome, label: 'Partial Spike', icon: <BoltIcon sx={{ fontSize: 13 }} />, color: '#8B5CF6' },
                  ] as const
                ).map(({ value, label, icon, color }) => (
                  <Button
                    key={value}
                    startIcon={icon}
                    onClick={() => setFeedbackOutcome(value)}
                    sx={{
                      borderColor: feedbackOutcome === value ? color : '#1E293B',
                      color: feedbackOutcome === value ? color : '#64748B',
                      bgcolor: feedbackOutcome === value ? `${color}15` : 'transparent',
                      fontSize: 11,
                      fontWeight: feedbackOutcome === value ? 700 : 400,
                      transition: 'all 0.15s',
                      '&:hover': { borderColor: color, color },
                    }}
                  >
                    {label}
                  </Button>
                ))}
              </ButtonGroup>
            </Box>

            {/* Optional actual cost */}
            <Box sx={{ minWidth: 160 }}>
              <Typography variant="caption" sx={{ color: '#64748B', mb: 0.75, display: 'block' }}>
                Actual cost (optional)
              </Typography>
              <TextField
                size="small"
                placeholder="e.g. 385.20"
                value={feedbackActualCost}
                onChange={(e) => setFeedbackActualCost(e.target.value)}
                InputProps={{ startAdornment: <Typography variant="caption" sx={{ color: '#64748B', mr: 0.5 }}>$</Typography> }}
                sx={{
                  '& .MuiOutlinedInput-root': {
                    color: '#E2E8F0',
                    '& fieldset': { borderColor: '#1E293B' },
                    '&:hover fieldset': { borderColor: '#334155' },
                  },
                  input: { color: '#E2E8F0', fontSize: 13 },
                }}
              />
            </Box>

            {/* Submit */}
            <Button
              variant="contained"
              disabled={!feedbackOutcome || feedbackSubmitting}
              onClick={() => feedbackOutcome && submitFeedback(feedbackOutcome)}
              sx={{
                bgcolor: '#4F46E5',
                color: '#fff',
                fontWeight: 700,
                fontSize: 12,
                alignSelf: { xs: 'stretch', sm: 'auto' },
                '&:hover': { bgcolor: '#4338CA' },
                '&.Mui-disabled': { bgcolor: '#1E293B', color: '#475569' },
              }}
            >
              {feedbackSubmitting ? 'Submitting…' : 'Submit Outcome'}
            </Button>
          </Stack>
        )}
      </Box>

      {/* ── Burst deferral sentinel ─────────────────────────────────────────── */}
      <Alert
        severity="info"
        icon={<AutoModeIcon sx={{ color: '#38BDF8' }} />}
        sx={{
          mt: 3,
          bgcolor: 'rgba(0,212,255,0.06)',
          border: '1px solid rgba(0,212,255,0.2)',
          color: '#E0F2FE',
          borderRadius: 1.5,
        }}
      >
        <Typography variant="body2" sx={{ fontWeight: 600, mb: 0.25 }}>
          Workload Queue Smoothing Active
        </Typography>
        <Typography variant="caption" sx={{ color: '#BAE6FD' }}>
          Non-critical Lakehouse compactions, historical data profiling, and MinHash sketch
          generation are deferred by up to{' '}
          <strong>{policy?.maxDeferralMinutes ?? 180} min</strong> during market-close NAV
          calculation windows (20:00–22:00 UTC). Tier-A front-office and real-time queries
          always execute immediately.
        </Typography>
      </Alert>
    </Paper>
  );
};

export default FinOpsForecastDashboard;
