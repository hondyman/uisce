import React, { useState, useEffect } from 'react';
import {
  Box, Card, CardContent, Typography, Slider, Button, Grid, Chip, Stack,
  Alert, CircularProgress, Paper, LinearProgress, Divider
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import SavingsIcon from '@mui/icons-material/Savings';
import SpeedIcon from '@mui/icons-material/Speed';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import { mdmScoringApi, type OptimalBundleResult, type OptimalBundleRequest } from '../api';

const ALL_CANDIDATE_VENDORS = [
  { id: 'BBG', name: 'Bloomberg', cost: 2140000 },
  { id: 'RFT', name: 'Refinitiv (LSEG)', cost: 1180000 },
  { id: 'FDS', name: 'FactSet', cost: 720000 },
  { id: 'ICE', name: 'ICE Data Services', cost: 540000 },
  { id: 'SPG', name: 'S&P Global MI', cost: 610000 },
];

export interface BundleOptimizerProps {
  onViewDisplacement?: (droppedVendors: string[], replacementVendors: string[]) => void;
}

export const BundleOptimizer: React.FC<BundleOptimizerProps> = ({ onViewDisplacement }) => {
  const [t1Coverage, setT1Coverage] = useState<number>(99.5);
  const [t2Coverage, setT2Coverage] = useState<number>(95.0);
  const [t3Coverage, setT3Coverage] = useState<number>(85.0);
  const [mandatoryVendors, setMandatoryVendors] = useState<string[]>([]);
  const [excludedVendors, setExcludedVendors] = useState<string[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [result, setResult] = useState<OptimalBundleResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const debounceTimerRef = React.useRef<NodeJS.Timeout | null>(null);

  const runOptimization = async (
    t1 = t1Coverage,
    t2 = t2Coverage,
    t3 = t3Coverage,
    mand = mandatoryVendors,
    excl = excludedVendors
  ) => {
    setLoading(true);
    setError(null);
    try {
      const req: OptimalBundleRequest = {
        target_t1_coverage: t1 / 100.0,
        target_t2_coverage: t2 / 100.0,
        target_t3_coverage: t3 / 100.0,
        mandatory_vendors: mand.length > 0 ? mand : undefined,
        excluded_vendors: excl.length > 0 ? excl : undefined,
        universe_size: 42000,
      };
      const res = await mdmScoringApi.optimizeBundle(req);
      setResult(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to solve optimal vendor bundle');
    } finally {
      setLoading(false);
    }
  };

  const scheduleOptimization = (newT1: number, newT2: number, newT3: number) => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = setTimeout(() => {
      runOptimization(newT1, newT2, newT3);
    }, 300);
  };

  useEffect(() => {
    runOptimization();
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  const toggleMandatory = (vid: string) => {
    setMandatoryVendors((prev) =>
      prev.includes(vid) ? prev.filter((v) => v !== vid) : [...prev, vid]
    );
    // If it's mandatory, it cannot be excluded
    setExcludedVendors((prev) => prev.filter((v) => v !== vid));
  };

  const toggleExcluded = (vid: string) => {
    setExcludedVendors((prev) =>
      prev.includes(vid) ? prev.filter((v) => v !== vid) : [...prev, vid]
    );
    // If it's excluded, it cannot be mandatory
    setMandatoryVendors((prev) => prev.filter((v) => v !== vid));
  };

  return (
    <Card variant="outlined" sx={{ borderRadius: 2 }}>
      <CardContent sx={{ p: 2.5 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2, flexWrap: 'wrap', gap: 2 }}>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 600 }}>
              Weighted Set Cover Portfolio Optimizer
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Solve the globally optimal minimal-cost vendor stack guaranteeing Tier 1–3 entity coverage requirements
            </Typography>
          </Box>
          <Button
            variant="contained"
            color="primary"
            startIcon={loading ? <CircularProgress size={16} color="inherit" /> : <SpeedIcon />}
            onClick={runOptimization}
            disabled={loading}
          >
            {loading ? 'Solving...' : 'Recalculate Optimal Stack'}
          </Button>
        </Box>

        <Divider sx={{ mb: 2.5 }} />

        <Grid container spacing={3}>
          {/* Controls Column */}
          <Grid item xs={12} md={5}>
            <Paper variant="outlined" sx={{ p: 2, borderRadius: 2 }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 2 }}>
                Coverage Thresholds (Tiers 1–3)
              </Typography>

              {/* Tier 1 Slider */}
              <Box sx={{ mb: 2.5 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    Tier 1 Target Coverage (LEI, ISIN, Prices)
                  </Typography>
                  <Chip size="small" color="warning" label={`${t1Coverage.toFixed(1)}%`} sx={{ fontWeight: 700 }} />
                </Box>
                <Slider
                  value={t1Coverage}
                  min={90.0}
                  max={100.0}
                  step={0.1}
                  valueLabelDisplay="auto"
                  onChange={(_, val) => {
                    const v = val as number;
                    setT1Coverage(v);
                    scheduleOptimization(v, t2Coverage, t3Coverage);
                  }}
                />
              </Box>

              {/* Tier 2 Slider */}
              <Box sx={{ mb: 2.5 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    Tier 2 Target Coverage (Names, Sectors, Cap)
                  </Typography>
                  <Chip size="small" color="primary" label={`${t2Coverage.toFixed(1)}%`} sx={{ fontWeight: 700 }} />
                </Box>
                <Slider
                  value={t2Coverage}
                  min={80.0}
                  max={100.0}
                  step={0.5}
                  valueLabelDisplay="auto"
                  onChange={(_, val) => {
                    const v = val as number;
                    setT2Coverage(v);
                    scheduleOptimization(t1Coverage, v, t3Coverage);
                  }}
                />
              </Box>

              {/* Tier 3 Slider */}
              <Box sx={{ mb: 3 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    Tier 3 Target Coverage (Employees, URLs, Founded)
                  </Typography>
                  <Chip size="small" label={`${t3Coverage.toFixed(1)}%`} sx={{ fontWeight: 700 }} />
                </Box>
                <Slider
                  value={t3Coverage}
                  min={70.0}
                  max={100.0}
                  step={1.0}
                  valueLabelDisplay="auto"
                  onChange={(_, val) => {
                    const v = val as number;
                    setT3Coverage(v);
                    scheduleOptimization(t1Coverage, t2Coverage, v);
                  }}
                />
              </Box>

              <Divider sx={{ my: 2 }} />

              {/* Vendor Constraints */}
              <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1.5 }}>
                Vendor Inclusion / Exclusion Constraints
              </Typography>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1.5 }}>
                Click vendor to enforce mandatory inclusion or exclusion in the mathematical solver.
              </Typography>

              <Stack spacing={1}>
                {ALL_CANDIDATE_VENDORS.map((v) => {
                  const isMandatory = mandatoryVendors.includes(v.id);
                  const isExcluded = excludedVendors.includes(v.id);
                  return (
                    <Box
                      key={v.id}
                      sx={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        p: 1,
                        borderRadius: 1,
                        bgcolor: 'background.default',
                      }}
                    >
                      <Box>
                        <Typography variant="body2" sx={{ fontWeight: 600 }}>
                          {v.name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          ${(v.cost / 1000).toFixed(0)}k/yr
                        </Typography>
                      </Box>
                      <Stack direction="row" spacing={0.8}>
                        <Chip
                          size="small"
                          label="Must Keep"
                          color={isMandatory ? 'success' : 'default'}
                          variant={isMandatory ? 'filled' : 'outlined'}
                          onClick={() => { toggleMandatory(v.id); runOptimization(); }}
                          sx={{ cursor: 'pointer' }}
                        />
                        <Chip
                          size="small"
                          label="Exclude"
                          color={isExcluded ? 'error' : 'default'}
                          variant={isExcluded ? 'filled' : 'outlined'}
                          onClick={() => { toggleExcluded(v.id); runOptimization(); }}
                          sx={{ cursor: 'pointer' }}
                        />
                      </Stack>
                    </Box>
                  );
                })}
              </Stack>
            </Paper>
          </Grid>

          {/* Results Column */}
          <Grid item xs={12} md={7}>
            {error && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {error}
              </Alert>
            )}

            {result?.solver_partial && (
              <Alert
                severity="warning"
                sx={{ mb: 2 }}
                action={
                  <Button color="inherit" size="small" onClick={() => runOptimization()} disabled={loading}>
                    Retry Solve
                  </Button>
                }
              >
                Solver execution exceeded runtime limit (&gt;50ms). Showing {result.solver_strategy === 'CACHED_LAST_VALID' ? 'cached last-valid bundle' : 'conservative fallback baseline'}.
              </Alert>
            )}

            {result?.was_relaxed && (
              <Alert severity="info" icon={<WarningAmberIcon />} sx={{ mb: 2 }}>
                Target coverage exceeded combined candidate universe capacity. Threshold was relaxed to maximum achievable coverage.
              </Alert>
            )}

            {result && (
              <Stack spacing={2.5}>
                {/* Executive Summary Card */}
                <Paper
                  variant="outlined"
                  sx={{
                    p: 2.5,
                    borderRadius: 2,
                    bgcolor: 'background.paper',
                    borderLeft: result.solver_partial ? '5px solid #ed6c02' : '5px solid #2e7d32',
                    opacity: result.solver_partial ? 0.6 : 1.0,
                    transition: 'opacity 0.25s ease-in-out',
                  }}
                >
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: 2 }}>
                    <Box>
                      <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', fontWeight: 700 }}>
                        Optimal Selected Stack ({result.selected_vendors.length} Vendors)
                      </Typography>
                      <Stack direction="row" spacing={1} sx={{ mt: 1 }} flexWrap="wrap" useFlexGap>
                        {result.selected_vendors.map((vid, idx) => (
                          <Chip
                            key={vid}
                            icon={<CheckCircleIcon />}
                            label={result.selected_vendor_names[idx] || vid}
                            color="success"
                            sx={{ fontWeight: 700 }}
                          />
                        ))}
                      </Stack>
                    </Box>
                    <Box sx={{ textAlign: 'right' }}>
                      <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', fontWeight: 700 }}>
                        Solver Latency
                      </Typography>
                      <Typography variant="body2" sx={{ fontWeight: 700, color: 'text.secondary', display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 0.5 }}>
                        <SpeedIcon fontSize="small" />
                        {result.solver_execution_ms.toFixed(2)} ms ({result.solver_strategy})
                      </Typography>
                    </Box>
                  </Box>

                  <Divider sx={{ my: 2 }} />

                  <Grid container spacing={2}>
                    <Grid item xs={4}>
                      <Typography variant="caption" color="text.secondary">Total Optimal Spend</Typography>
                      <Typography variant="h6" sx={{ fontWeight: 700, color: 'primary.main' }}>
                        ${result.total_annual_cost.toLocaleString()}
                      </Typography>
                    </Grid>
                    <Grid item xs={4}>
                      <Typography variant="caption" color="text.secondary">Previous Full Stack</Typography>
                      <Typography variant="h6" sx={{ fontWeight: 700, color: 'text.secondary' }}>
                        ${result.previous_total_cost.toLocaleString()}
                      </Typography>
                    </Grid>
                    <Grid item xs={4}>
                      <Typography variant="caption" color="text.secondary">Net Annual Savings</Typography>
                      <Typography variant="h6" sx={{ fontWeight: 700, color: 'success.main', display: 'flex', alignItems: 'center', gap: 0.5 }}>
                        <SavingsIcon fontSize="small" />
                        ${result.annual_savings.toLocaleString()} ({result.savings_pct.toFixed(1)}%)
                      </Typography>
                    </Grid>
                  </Grid>
                </Paper>

                {/* Achieved Coverage Progress */}
                <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 2 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 2 }}>
                    Coverage Achieved Across Attribute Tiers
                  </Typography>

                  <Box sx={{ mb: 2 }}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                      <Typography variant="body2" sx={{ fontWeight: 600 }}>Tier 1 (Core Attributes)</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 700 }}>
                        {(result.t1_coverage_achieved * 100).toFixed(2)}% (Target: {t1Coverage.toFixed(1)}%)
                      </Typography>
                    </Box>
                    <LinearProgress
                      variant="determinate"
                      value={Math.min(100, result.t1_coverage_achieved * 100)}
                      color="warning"
                      sx={{ height: 8, borderRadius: 4 }}
                    />
                  </Box>

                  <Box sx={{ mb: 2 }}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                      <Typography variant="body2" sx={{ fontWeight: 600 }}>Tier 2 (Reference Data)</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 700 }}>
                        {(result.t2_coverage_achieved * 100).toFixed(2)}% (Target: {t2Coverage.toFixed(1)}%)
                      </Typography>
                    </Box>
                    <LinearProgress
                      variant="determinate"
                      value={Math.min(100, result.t2_coverage_achieved * 100)}
                      color="primary"
                      sx={{ height: 8, borderRadius: 4 }}
                    />
                  </Box>

                  <Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                      <Typography variant="body2" sx={{ fontWeight: 600 }}>Tier 3 (Contextual Attributes)</Typography>
                      <Typography variant="body2" sx={{ fontWeight: 700 }}>
                        {(result.t3_coverage_achieved * 100).toFixed(2)}% (Target: {t3Coverage.toFixed(1)}%)
                      </Typography>
                    </Box>
                    <LinearProgress
                      variant="determinate"
                      value={Math.min(100, result.t3_coverage_achieved * 100)}
                      color="inherit"
                      sx={{ height: 8, borderRadius: 4 }}
                    />
                  </Box>
                </Paper>

                {/* Residual Gaps Breakdown if Relaxed */}
                {result.residual_gaps && result.residual_gaps.length > 0 && (
                  <Paper variant="outlined" sx={{ p: 2, borderRadius: 2, bgcolor: 'warning.light', color: 'warning.contrastText' }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
                      Residual Coverage Gaps Report
                    </Typography>
                    {result.residual_gaps.map((g, idx) => (
                      <Typography key={idx} variant="body2" sx={{ mb: 0.5 }}>
                        • Tier {g.tier}: Target {(g.target_coverage * 100).toFixed(1)}% relaxed to {(g.achieved_coverage * 100).toFixed(1)}% ({g.missing_entity_count.toLocaleString()} entities missing from all candidate feeds).
                      </Typography>
                    ))}
                  </Paper>
                )}

                {/* Action to simulate displacement */}
                {onViewDisplacement && (
                  <Box sx={{ display: 'flex', justifyContent: 'flex-end', pt: 1 }}>
                    <Button
                      variant="contained"
                      color="primary"
                      onClick={() => {
                        const dropped = ALL_CANDIDATE_VENDORS.map((v) => v.id).filter(
                          (id) => !result.selected_vendors.includes(id)
                        );
                        onViewDisplacement(dropped, result.selected_vendors);
                      }}
                    >
                      Simulate Displacement TCO for This Bundle
                    </Button>
                  </Box>
                )}
              </Stack>
            )}
          </Grid>
        </Grid>
      </CardContent>
    </Card>
  );
};
