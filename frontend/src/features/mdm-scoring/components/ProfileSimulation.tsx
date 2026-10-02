import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  Box, Card, CardContent, Typography, GridLegacy as Grid, Chip, Button, Stack,
  Slider, CircularProgress, Alert, Paper, Table, TableBody, TableCell,
  TableContainer, TableHead, TableRow, MenuItem, Select, FormControl,
  InputLabel, Tooltip, Divider, ToggleButtonGroup, ToggleButton
} from '@mui/material';
import TuneIcon from '@mui/icons-material/Tune';
import CompareArrowsIcon from '@mui/icons-material/CompareArrows';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import TrendingFlatIcon from '@mui/icons-material/TrendingFlat';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import LightbulbIcon from '@mui/icons-material/Lightbulb';
import AccountBalanceWalletIcon from '@mui/icons-material/AccountBalanceWallet';
import LayersIcon from '@mui/icons-material/Layers';
import {
  mdmScoringApi,
  type ProfileSimulationResponse,
  type ProfileWeights,
  type VendorRankShift,
  type VendorRankingEntry
} from '../api';

const PRESET_PROFILES: Record<string, { name: string; weights: ProfileWeights }> = {
  default: {
    name: 'Production Default (Quality Balanced)',
    weights: {
      weight_sufficiency: 0.30,
      weight_coverage: 0.20,
      weight_sla: 0.15,
      weight_stability: 0.15,
      weight_friction: 0.10,
      weight_licensing: 0.10,
    },
  },
  operations: {
    name: 'Operations & Stability Focus',
    weights: {
      weight_sufficiency: 0.15,
      weight_coverage: 0.15,
      weight_sla: 0.30,
      weight_stability: 0.25,
      weight_friction: 0.10,
      weight_licensing: 0.05,
    },
  },
  licensing: {
    name: 'Commercial Rights & Unbundling',
    weights: {
      weight_sufficiency: 0.20,
      weight_coverage: 0.20,
      weight_sla: 0.10,
      weight_stability: 0.10,
      weight_friction: 0.05,
      weight_licensing: 0.35,
    },
  },
  coverage: {
    name: 'Coverage & Sufficiency Maximalist',
    weights: {
      weight_sufficiency: 0.40,
      weight_coverage: 0.40,
      weight_sla: 0.05,
      weight_stability: 0.05,
      weight_friction: 0.05,
      weight_licensing: 0.05,
    },
  },
};

const PILLARS = [
  { key: 'weight_sufficiency', label: 'Sufficiency (SR)', desc: 'Concordance with Golden Record truth' },
  { key: 'weight_coverage', label: 'Attribute Coverage', desc: 'Breadth across universe entities' },
  { key: 'weight_sla', label: 'Delivery SLA', desc: 'Punctuality before market close deadlines' },
  { key: 'weight_stability', label: 'Stability (Decay)', desc: 'Freedom from post-EOD restatements' },
  { key: 'weight_friction', label: 'Steward Friction', desc: 'Direct labor drag & defect MTTR' },
  { key: 'weight_licensing', label: 'Commercial Rights', desc: 'Derived IP & unbundled API rights' },
] as const;

export interface ProfileSimulationProps {
  entityDomain?: string;
  universeSize?: number;
}

export const ProfileSimulation: React.FC<ProfileSimulationProps> = ({
  entityDomain,
  universeSize = 42000,
}) => {
  // Profile A state
  const [profileAPreset, setProfileAPreset] = useState<string>('default');
  const [nameA, setNameA] = useState<string>(PRESET_PROFILES.default.name);
  const [weightsA, setWeightsA] = useState<ProfileWeights>({ ...PRESET_PROFILES.default.weights });

  // Profile B state
  const [profileBPreset, setProfileBPreset] = useState<string>('operations');
  const [nameB, setNameB] = useState<string>(PRESET_PROFILES.operations.name);
  const [weightsB, setWeightsB] = useState<ProfileWeights>({ ...PRESET_PROFILES.operations.weights });

  // Simulation response state
  const [simulation, setSimulation] = useState<ProfileSimulationResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // Radar inspection mode
  const [radarVendor, setRadarVendor] = useState<string>('ALL_WEIGHTS');

  // Sum check
  const sumA = useMemo(
    () =>
      weightsA.weight_sufficiency +
      weightsA.weight_coverage +
      weightsA.weight_sla +
      weightsA.weight_stability +
      weightsA.weight_friction +
      weightsA.weight_licensing,
    [weightsA]
  );

  const sumB = useMemo(
    () =>
      weightsB.weight_sufficiency +
      weightsB.weight_coverage +
      weightsB.weight_sla +
      weightsB.weight_stability +
      weightsB.weight_friction +
      weightsB.weight_licensing,
    [weightsB]
  );

  const isSumAValid = Math.abs(sumA - 1.000) <= 0.002;
  const isSumBValid = Math.abs(sumB - 1.000) <= 0.002;

  // Auto-normalize helpers
  const normalizeA = () => {
    if (sumA <= 0) return;
    const factor = 1.0 / sumA;
    setWeightsA({
      weight_sufficiency: Math.round(weightsA.weight_sufficiency * factor * 1000) / 1000,
      weight_coverage: Math.round(weightsA.weight_coverage * factor * 1000) / 1000,
      weight_sla: Math.round(weightsA.weight_sla * factor * 1000) / 1000,
      weight_stability: Math.round(weightsA.weight_stability * factor * 1000) / 1000,
      weight_friction: Math.round(weightsA.weight_friction * factor * 1000) / 1000,
      weight_licensing: Math.round(weightsA.weight_licensing * factor * 1000) / 1000,
    });
  };

  const normalizeB = () => {
    if (sumB <= 0) return;
    const factor = 1.0 / sumB;
    setWeightsB({
      weight_sufficiency: Math.round(weightsB.weight_sufficiency * factor * 1000) / 1000,
      weight_coverage: Math.round(weightsB.weight_coverage * factor * 1000) / 1000,
      weight_sla: Math.round(weightsB.weight_sla * factor * 1000) / 1000,
      weight_stability: Math.round(weightsB.weight_stability * factor * 1000) / 1000,
      weight_friction: Math.round(weightsB.weight_friction * factor * 1000) / 1000,
      weight_licensing: Math.round(weightsB.weight_licensing * factor * 1000) / 1000,
    });
  };

  // Run simulation API call
  const runSimulation = useCallback(async () => {
    if (!isSumAValid || !isSumBValid) return;
    try {
      setLoading(true);
      setError(null);

      const res = await mdmScoringApi.simulateProfiles({
        entity_domain: entityDomain,
        universe_size: universeSize,
        profile_a: {
          name: nameA,
          weights: weightsA,
        },
        profile_b: {
          name: nameB,
          weights: weightsB,
        },
      });
      setSimulation(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to execute profile simulation');
    } finally {
      setLoading(false);
    }
  }, [nameA, weightsA, nameB, weightsB, entityDomain, universeSize, isSumAValid, isSumBValid]);

  useEffect(() => {
    runSimulation();
  }, [runSimulation]);

  const handlePresetA = (presetKey: string) => {
    setProfileAPreset(presetKey);
    if (presetKey !== 'custom' && PRESET_PROFILES[presetKey]) {
      setNameA(PRESET_PROFILES[presetKey].name);
      setWeightsA({ ...PRESET_PROFILES[presetKey].weights });
    }
  };

  const handlePresetB = (presetKey: string) => {
    setProfileBPreset(presetKey);
    if (presetKey !== 'custom' && PRESET_PROFILES[presetKey]) {
      setNameB(PRESET_PROFILES[presetKey].name);
      setWeightsB({ ...PRESET_PROFILES[presetKey].weights });
    }
  };

  const updateWeightA = (pillar: keyof ProfileWeights, val: number) => {
    setProfileAPreset('custom');
    setWeightsA((prev) => ({ ...prev, [pillar]: val }));
  };

  const updateWeightB = (pillar: keyof ProfileWeights, val: number) => {
    setProfileBPreset('custom');
    setWeightsB((prev) => ({ ...prev, [pillar]: val }));
  };

  // Radar geometry calculations
  const radarSize = 360;
  const cx = radarSize / 2;
  const cy = radarSize / 2;
  const radius = 120;
  const numAxes = PILLARS.length;

  const getCoordinates = (axisIndex: number, val: number) => {
    const angle = -Math.PI / 2 + (axisIndex * 2 * Math.PI) / numAxes;
    const r = radius * Math.max(0, Math.min(1, val));
    return {
      x: cx + r * Math.cos(angle),
      y: cy + r * Math.sin(angle),
    };
  };

  // Build polygons for radar
  const polygonA = useMemo(() => {
    return PILLARS.map((p, idx) => {
      // Scale weight to 0..1 (max single weight ~0.5, so normalize /0.5 for visual clarity)
      const val = Math.min(1.0, weightsA[p.key as keyof ProfileWeights] / 0.5);
      const pt = getCoordinates(idx, val);
      return `${pt.x},${pt.y}`;
    }).join(' ');
  }, [weightsA]);

  const polygonB = useMemo(() => {
    return PILLARS.map((p, idx) => {
      const val = Math.min(1.0, weightsB[p.key as keyof ProfileWeights] / 0.5);
      const pt = getCoordinates(idx, val);
      return `${pt.x},${pt.y}`;
    }).join(' ');
  }, [weightsB]);

  return (
    <Box sx={{ p: 2 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3, flexWrap: 'wrap', gap: 2 }}>
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <TuneIcon color="primary" sx={{ fontSize: 32 }} />
            <Typography variant="h5" sx={{ fontWeight: 700 }}>
              Profile Sensitivity Simulator
            </Typography>
            <Chip label="A / B What-If Mode" color="primary" size="small" sx={{ fontWeight: 700 }} />
          </Box>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            Test alternative weighting models to see immediate vendor rank shifts, composite score movement, and bundle financial impact.
          </Typography>
        </Box>

        <Button
          variant="contained"
          startIcon={<CompareArrowsIcon />}
          onClick={runSimulation}
          disabled={loading || !isSumAValid || !isSumBValid}
          sx={{ fontWeight: 700 }}
        >
          {loading ? 'Simulating...' : 'Recalculate Impact'}
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      {/* Dual Column Profile Weight Tuning Cards */}
      <Grid container spacing={3} sx={{ mb: 3 }}>
        {/* Profile A Column */}
        <Grid item xs={12} md={6}>
          <Card
            variant="outlined"
            sx={{
              borderRadius: 2,
              borderTop: '4px solid #1976d2',
              bgcolor: 'background.paper',
            }}
          >
            <CardContent sx={{ p: 2.5 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1.5 }}>
                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 700, color: '#1976d2' }}>
                    Profile A (Baseline)
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Solid radar overlay
                  </Typography>
                </Box>
                <Chip
                  label={`Sum: ${sumA.toFixed(3)}`}
                  size="small"
                  color={isSumAValid ? 'success' : 'error'}
                  sx={{ fontWeight: 700 }}
                />
              </Box>

              <FormControl fullWidth size="small" sx={{ mb: 2 }}>
                <InputLabel>Preset</InputLabel>
                <Select
                  value={profileAPreset}
                  label="Preset"
                  onChange={(e) => handlePresetA(e.target.value)}
                >
                  <MenuItem value="default">Production Default (Quality Balanced)</MenuItem>
                  <MenuItem value="operations">Operations &amp; Stability Focus</MenuItem>
                  <MenuItem value="licensing">Commercial Rights &amp; Unbundling</MenuItem>
                  <MenuItem value="coverage">Coverage &amp; Sufficiency Maximalist</MenuItem>
                  <MenuItem value="custom">Custom Sliders</MenuItem>
                </Select>
              </FormControl>

              {/* Sliders */}
              <Stack spacing={1.5}>
                {PILLARS.map((p) => {
                  const val = weightsA[p.key as keyof ProfileWeights];
                  return (
                    <Box key={p.key}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Typography variant="caption" sx={{ fontWeight: 600 }}>
                          {p.label}
                        </Typography>
                        <Typography variant="caption" sx={{ fontWeight: 700, color: '#1976d2' }}>
                          {(val * 100).toFixed(1)}%
                        </Typography>
                      </Box>
                      <Slider
                        size="small"
                        min={0}
                        max={0.60}
                        step={0.01}
                        value={val}
                        onChange={(_, v) => updateWeightA(p.key as keyof ProfileWeights, Number(v))}
                        sx={{ color: '#1976d2', py: 0.5 }}
                      />
                    </Box>
                  );
                })}
              </Stack>

              {!isSumAValid && (
                <Button
                  size="small"
                  variant="outlined"
                  color="warning"
                  startIcon={<AutoFixHighIcon />}
                  onClick={normalizeA}
                  fullWidth
                  sx={{ mt: 2 }}
                >
                  Normalize Weights to 1.000
                </Button>
              )}
            </CardContent>
          </Card>
        </Grid>

        {/* Profile B Column */}
        <Grid item xs={12} md={6}>
          <Card
            variant="outlined"
            sx={{
              borderRadius: 2,
              borderTop: '4px solid #9c27b0',
              bgcolor: 'background.paper',
            }}
          >
            <CardContent sx={{ p: 2.5 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1.5 }}>
                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 700, color: '#9c27b0' }}>
                    Profile B (Simulation)
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Dashed radar overlay
                  </Typography>
                </Box>
                <Chip
                  label={`Sum: ${sumB.toFixed(3)}`}
                  size="small"
                  color={isSumBValid ? 'success' : 'error'}
                  sx={{ fontWeight: 700 }}
                />
              </Box>

              <FormControl fullWidth size="small" sx={{ mb: 2 }}>
                <InputLabel>Preset</InputLabel>
                <Select
                  value={profileBPreset}
                  label="Preset"
                  onChange={(e) => handlePresetB(e.target.value)}
                >
                  <MenuItem value="default">Production Default (Quality Balanced)</MenuItem>
                  <MenuItem value="operations">Operations &amp; Stability Focus</MenuItem>
                  <MenuItem value="licensing">Commercial Rights &amp; Unbundling</MenuItem>
                  <MenuItem value="coverage">Coverage &amp; Sufficiency Maximalist</MenuItem>
                  <MenuItem value="custom">Custom Sliders</MenuItem>
                </Select>
              </FormControl>

              {/* Sliders */}
              <Stack spacing={1.5}>
                {PILLARS.map((p) => {
                  const val = weightsB[p.key as keyof ProfileWeights];
                  return (
                    <Box key={p.key}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Typography variant="caption" sx={{ fontWeight: 600 }}>
                          {p.label}
                        </Typography>
                        <Typography variant="caption" sx={{ fontWeight: 700, color: '#9c27b0' }}>
                          {(val * 100).toFixed(1)}%
                        </Typography>
                      </Box>
                      <Slider
                        size="small"
                        min={0}
                        max={0.60}
                        step={0.01}
                        value={val}
                        onChange={(_, v) => updateWeightB(p.key as keyof ProfileWeights, Number(v))}
                        sx={{ color: '#9c27b0', py: 0.5 }}
                      />
                    </Box>
                  );
                })}
              </Stack>

              {!isSumBValid && (
                <Button
                  size="small"
                  variant="outlined"
                  color="warning"
                  startIcon={<AutoFixHighIcon />}
                  onClick={normalizeB}
                  fullWidth
                  sx={{ mt: 2 }}
                >
                  Normalize Weights to 1.000
                </Button>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Visual Overlay: Radar & Bundle Impact */}
      {simulation && (
        <Grid container spacing={3} sx={{ mb: 3 }}>
          {/* Radar Overlay */}
          <Grid item xs={12} md={5}>
            <Card variant="outlined" sx={{ borderRadius: 2, height: '100%' }}>
              <CardContent sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                  <Typography variant="h6" sx={{ fontWeight: 700 }}>
                    Weight Allocation Radar
                  </Typography>
                  <Stack direction="row" spacing={1}>
                    <Chip
                      size="small"
                      label="Profile A (Solid)"
                      sx={{ bgcolor: 'rgba(25, 118, 210, 0.15)', color: '#1976d2', fontWeight: 700 }}
                    />
                    <Chip
                      size="small"
                      label="Profile B (Dashed)"
                      sx={{ bgcolor: 'rgba(156, 39, 176, 0.15)', color: '#9c27b0', fontWeight: 700 }}
                    />
                  </Stack>
                </Box>
                <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 2 }}>
                  Direct comparison of 6-pillar resource weighting models
                </Typography>

                <Box sx={{ display: 'flex', justifyContent: 'center' }}>
                  <svg viewBox={`0 0 ${radarSize} ${radarSize}`} style={{ width: '100%', maxWidth: '340px' }}>
                    {/* Concentric grid webs */}
                    {[0.25, 0.5, 0.75, 1.0].map((level) => {
                      const pts = PILLARS.map((_, idx) => {
                        const pt = getCoordinates(idx, level);
                        return `${pt.x},${pt.y}`;
                      }).join(' ');
                      return (
                        <polygon
                          key={level}
                          points={pts}
                          fill="none"
                          stroke="#e0e0e0"
                          strokeWidth="1"
                          strokeDasharray="2,2"
                        />
                      );
                    })}

                    {/* Radial axis lines and labels */}
                    {PILLARS.map((p, idx) => {
                      const pt = getCoordinates(idx, 1.0);
                      const labelPt = getCoordinates(idx, 1.18);
                      return (
                        <g key={idx}>
                          <line
                            x1={cx}
                            y1={cy}
                            x2={pt.x}
                            y2={pt.y}
                            stroke="#ddd"
                            strokeWidth="1"
                          />
                          <text
                            x={labelPt.x}
                            y={labelPt.y}
                            textAnchor="middle"
                            dominantBaseline="central"
                            fontSize="9"
                            fontWeight="600"
                            fill="#555"
                          >
                            {p.label.split(' ')[0]}
                          </text>
                        </g>
                      );
                    })}

                    {/* Profile A Polygon (Solid) */}
                    <polygon
                      points={polygonA}
                      fill="rgba(25, 118, 210, 0.2)"
                      stroke="#1976d2"
                      strokeWidth="2.5"
                    />

                    {/* Profile B Polygon (Dashed) */}
                    <polygon
                      points={polygonB}
                      fill="rgba(156, 39, 176, 0.15)"
                      stroke="#9c27b0"
                      strokeWidth="2.5"
                      strokeDasharray="6,4"
                    />
                  </svg>
                </Box>
              </CardContent>
            </Card>
          </Grid>

          {/* Bundle Impact & Strategic Insight Card */}
          <Grid item xs={12} md={7}>
            <Card
              variant="outlined"
              sx={{
                borderRadius: 2,
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between',
              }}
            >
              <CardContent sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                  <AccountBalanceWalletIcon color="primary" />
                  <Typography variant="h6" sx={{ fontWeight: 700 }}>
                    Optimal Bundle Financial Shift
                  </Typography>
                </Box>

                <Grid container spacing={2} sx={{ mb: 2.5 }}>
                  <Grid item xs={6}>
                    <Paper variant="outlined" sx={{ p: 2, borderRadius: 2, bgcolor: 'action.hover' }}>
                      <Typography variant="caption" sx={{ fontWeight: 700, color: '#1976d2' }}>
                        PROFILE A BUNDLE
                      </Typography>
                      <Typography variant="h5" sx={{ fontWeight: 700, mt: 0.5 }}>
                        ${(simulation.bundle_impact.profile_a_optimal.cost / 1000000).toFixed(2)}M
                      </Typography>
                      <Typography variant="caption" color="text.secondary" display="block">
                        Vendors: {simulation.bundle_impact.profile_a_optimal.vendors.join(', ')}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        T1 Coverage: {simulation.bundle_impact.profile_a_optimal.composite_coverage_pct}%
                      </Typography>
                    </Paper>
                  </Grid>

                  <Grid item xs={6}>
                    <Paper
                      variant="outlined"
                      sx={{
                        p: 2,
                        borderRadius: 2,
                        bgcolor: 'action.hover',
                        borderColor: '#9c27b0',
                      }}
                    >
                      <Typography variant="caption" sx={{ fontWeight: 700, color: '#9c27b0' }}>
                        PROFILE B BUNDLE
                      </Typography>
                      <Typography variant="h5" sx={{ fontWeight: 700, mt: 0.5 }}>
                        ${(simulation.bundle_impact.profile_b_optimal.cost / 1000000).toFixed(2)}M
                      </Typography>
                      <Typography variant="caption" color="text.secondary" display="block">
                        Vendors: {simulation.bundle_impact.profile_b_optimal.vendors.join(', ')}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        T1 Coverage: {simulation.bundle_impact.profile_b_optimal.composite_coverage_pct}%
                      </Typography>
                    </Paper>
                  </Grid>
                </Grid>

                {/* Bundle Delta */}
                <Box
                  sx={{
                    p: 2,
                    borderRadius: 2,
                    bgcolor: simulation.bundle_impact.bundle_delta_cost <= 0 ? 'rgba(46, 125, 50, 0.08)' : 'rgba(237, 108, 2, 0.08)',
                    border: `1px solid ${simulation.bundle_impact.bundle_delta_cost <= 0 ? '#2e7d32' : '#ed6c02'}`,
                    mb: 2,
                  }}
                >
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                      Net Annual Spend Variance (B vs A):
                    </Typography>
                    <Typography
                      variant="h6"
                      sx={{
                        fontWeight: 800,
                        color: simulation.bundle_impact.bundle_delta_cost <= 0 ? '#2e7d32' : '#ed6c02',
                      }}
                    >
                      {simulation.bundle_impact.bundle_delta_cost >= 0 ? '+' : ''}
                      ${(simulation.bundle_impact.bundle_delta_cost / 1000).toFixed(1)}K / yr
                    </Typography>
                  </Box>
                </Box>

                {/* Strategic Procurement Insight */}
                <Paper
                  variant="outlined"
                  sx={{
                    p: 2,
                    borderRadius: 2,
                    bgcolor: 'background.default',
                    borderLeft: '4px solid #0288d1',
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
                    <LightbulbIcon color="info" sx={{ mt: 0.2 }} />
                    <Box>
                      <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                        Executive Procurement Rationale:
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5, lineHeight: 1.5 }}>
                        {simulation.bundle_impact.insight}
                      </Typography>
                    </Box>
                  </Box>
                </Paper>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Rank Movement Table */}
      {simulation && (
        <Card variant="outlined" sx={{ borderRadius: 2 }}>
          <CardContent sx={{ p: 2.5 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="h6" sx={{ fontWeight: 700 }}>
                Vendor Ranking &amp; Score Movement (A → B)
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Positive rank delta (green) indicates vendor climbed higher under Profile B
              </Typography>
            </Box>

            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ bgcolor: 'action.hover' }}>
                    <TableCell sx={{ fontWeight: 700 }}>Vendor</TableCell>
                    <TableCell align="center" sx={{ fontWeight: 700 }}>Rank (A)</TableCell>
                    <TableCell align="center" sx={{ fontWeight: 700 }}>Rank (B)</TableCell>
                    <TableCell align="center" sx={{ fontWeight: 700 }}>Rank Shift</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 700 }}>Score (A)</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 700 }}>Score (B)</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 700 }}>Score Delta</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {simulation.rank_shifts.map((shift) => {
                    const isImproved = shift.rank_delta > 0;
                    const isDropped = shift.rank_delta < 0;

                    return (
                      <TableRow key={shift.vendor_id} hover>
                        <TableCell>
                          <strong>{shift.vendor_id}</strong>
                          <Typography variant="caption" color="text.secondary" display="block">
                            {shift.vendor_name}
                          </Typography>
                        </TableCell>
                        <TableCell align="center" sx={{ fontWeight: 600 }}>
                          #{shift.rank_a}
                        </TableCell>
                        <TableCell align="center" sx={{ fontWeight: 700 }}>
                          #{shift.rank_b}
                        </TableCell>
                        <TableCell align="center">
                          {isImproved && (
                            <Chip
                              size="small"
                              color="success"
                              icon={<TrendingUpIcon fontSize="small" />}
                              label={`+${shift.rank_delta}`}
                              sx={{ fontWeight: 700 }}
                            />
                          )}
                          {isDropped && (
                            <Chip
                              size="small"
                              color="error"
                              icon={<TrendingDownIcon fontSize="small" />}
                              label={`${shift.rank_delta}`}
                              sx={{ fontWeight: 700 }}
                            />
                          )}
                          {!isImproved && !isDropped && (
                            <Chip
                              size="small"
                              variant="outlined"
                              icon={<TrendingFlatIcon fontSize="small" />}
                              label="0"
                              sx={{ fontWeight: 600 }}
                            />
                          )}
                        </TableCell>
                        <TableCell align="right" sx={{ fontWeight: 600 }}>
                          {shift.score_a.toFixed(1)}%
                        </TableCell>
                        <TableCell align="right" sx={{ fontWeight: 700 }}>
                          {shift.score_b.toFixed(1)}%
                        </TableCell>
                        <TableCell
                          align="right"
                          sx={{
                            fontWeight: 700,
                            color: shift.score_delta > 0 ? 'success.main' : shift.score_delta < 0 ? 'error.main' : 'text.secondary',
                          }}
                        >
                          {shift.score_delta > 0 ? '+' : ''}
                          {shift.score_delta.toFixed(1)}%
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </TableContainer>
          </CardContent>
        </Card>
      )}
    </Box>
  );
};
