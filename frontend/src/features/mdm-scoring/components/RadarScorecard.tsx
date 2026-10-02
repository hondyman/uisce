import React, { useEffect, useState, useMemo } from 'react';
import {
  Box, Card, CardContent, Typography, Chip, Stack, CircularProgress,
  Paper, GridLegacy as Grid, Divider
} from '@mui/material';
import { mdmScoringApi, type VendorDimensionProfile, type WeightProfile } from '../api';

const VENDOR_COLORS: Record<string, string> = {
  BBG: '#1976d2', // Blue
  RFT: '#2e7d32', // Green
  FDS: '#ed6c02', // Orange
  ICE: '#9c27b0', // Purple
  SPG: '#d32f2f', // Red
};

const DIMENSION_AXES = [
  { key: 'sufficiency', label: 'Sufficiency (SR)', weight: '30%', desc: 'Concordance with Golden Record truth' },
  { key: 'coverage', label: 'Coverage', weight: '20%', desc: 'Breadth across universe entities' },
  { key: 'sla', label: 'Delivery SLA', weight: '15%', desc: 'Punctuality before market close deadlines' },
  { key: 'stability', label: 'Stability (Decay)', weight: '15%', desc: 'Freedom from post-EOD restatements' },
  { key: 'friction', label: 'Steward Friction', weight: '10%', desc: 'Direct labor drag & defect ticket MTTR' },
  { key: 'licensing', label: 'Rights Matrix', weight: '10%', desc: 'Derived IP & unbundled API rights' },
] as const;

export interface RadarScorecardProps {
  asOf?: string;
  initialVendors?: string[];
}

export const RadarScorecard: React.FC<RadarScorecardProps> = ({ asOf, initialVendors }) => {
  const [profiles, setProfiles] = useState<VendorDimensionProfile[]>([]);
  const [activeProfile, setActiveProfile] = useState<WeightProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeVendors, setActiveVendors] = useState<Record<string, boolean>>({
    BBG: true,
    RFT: true,
    FDS: false,
    ICE: false,
    SPG: false,
  });
  const [hoveredPoint, setHoveredPoint] = useState<{
    vendor: string;
    axis: string;
    score: number;
    rawDesc: string;
    x: number;
    y: number;
  } | null>(null);

  useEffect(() => {
    let mounted = true;
    setLoading(true);
    Promise.all([
      mdmScoringApi.dimensions(asOf),
      mdmScoringApi.weightProfiles().catch(() => [] as WeightProfile[]),
    ])
      .then(([data, wProfiles]) => {
        if (!mounted) return;
        setProfiles(data);
        const active = wProfiles.find((p) => p.is_active) || wProfiles[0] || null;
        setActiveProfile(active);

        if (initialVendors && initialVendors.length > 0) {
          const map: Record<string, boolean> = {};
          data.forEach(p => { map[p.vendor_id] = initialVendors.includes(p.vendor_id); });
          setActiveVendors(map);
        }
        setLoading(false);
      })
      .catch((err) => {
        if (!mounted) return;
        setError(err.message || 'Failed to load dimension profiles');
        setLoading(false);
      });
    return () => { mounted = false; };
  }, [asOf, initialVendors]);

  const dimensionAxes = useMemo(() => {
    if (!activeProfile) {
      return [
        { key: 'sufficiency', label: 'Sufficiency (SR)', weight: '30%', desc: 'Concordance with Golden Record truth' },
        { key: 'coverage', label: 'Coverage', weight: '20%', desc: 'Breadth across universe entities' },
        { key: 'sla', label: 'Delivery SLA', weight: '15%', desc: 'Punctuality before market close deadlines' },
        { key: 'stability', label: 'Stability (Decay)', weight: '15%', desc: 'Freedom from post-EOD restatements' },
        { key: 'friction', label: 'Steward Friction', weight: '10%', desc: 'Direct labor drag & defect ticket MTTR' },
        { key: 'licensing', label: 'Rights Matrix', weight: '10%', desc: 'Derived IP & unbundled API rights' },
      ];
    }
    return [
      { key: 'sufficiency', label: 'Sufficiency (SR)', weight: `${(activeProfile.weight_suff * 100).toFixed(0)}%`, desc: 'Concordance with Golden Record truth' },
      { key: 'coverage', label: 'Coverage', weight: `${(activeProfile.weight_cov * 100).toFixed(0)}%`, desc: 'Breadth across universe entities' },
      { key: 'sla', label: 'Delivery SLA', weight: `${(activeProfile.weight_sla * 100).toFixed(0)}%`, desc: 'Punctuality before market close deadlines' },
      { key: 'stability', label: 'Stability (Decay)', weight: `${(activeProfile.weight_stab * 100).toFixed(0)}%`, desc: 'Freedom from post-EOD restatements' },
      { key: 'friction', label: 'Steward Friction', weight: `${(activeProfile.weight_oer * 100).toFixed(0)}%`, desc: 'Direct labor drag & defect ticket MTTR' },
      { key: 'licensing', label: 'Rights Matrix', weight: `${(activeProfile.weight_lic * 100).toFixed(0)}%`, desc: 'Derived IP & unbundled API rights' },
    ];
  }, [activeProfile]);

  const toggleVendor = (vid: string) => {
    setActiveVendors(prev => ({ ...prev, [vid]: !prev[vid] }));
  };

  // Radar geometry calculations
  const size = 420;
  const cx = size / 2;
  const cy = size / 2;
  const radius = 150;
  const numAxes = dimensionAxes.length;

  const getCoordinates = (axisIndex: number, value: number) => {
    const angle = -Math.PI / 2 + (axisIndex * 2 * Math.PI) / numAxes;
    const r = radius * Math.max(0, Math.min(1, value));
    return {
      x: cx + r * Math.cos(angle),
      y: cy + r * Math.sin(angle),
    };
  };

  // Concentric polygon grids (20%, 40%, 60%, 80%, 100%)
  const gridLevels = [0.2, 0.4, 0.6, 0.8, 1.0];

  const getGridPolygon = (level: number) => {
    return Array.from({ length: numAxes })
      .map((_, i) => {
        const { x, y } = getCoordinates(i, level);
        return `${x},${y}`;
      })
      .join(' ');
  };

  const vendorPolygons = useMemo(() => {
    return profiles.map((p) => {
      const coords = dimensionAxes.map((axis, i) => {
        const val = p.components[axis.key as keyof typeof p.components] || 0;
        return getCoordinates(i, val);
      });
      const pointsStr = coords.map(c => `${c.x},${c.y}`).join(' ');
      return { profile: p, coords, pointsStr };
    });
  }, [profiles, dimensionAxes]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', p: 4, minHeight: 300 }}>
        <CircularProgress size={32} />
        <Typography variant="body2" sx={{ ml: 2 }}>Loading 6-pillar vendor telemetry...</Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Paper sx={{ p: 2, bgcolor: 'error.light', color: 'error.contrastText' }}>
        <Typography variant="body2">Error loading radar scorecard: {error}</Typography>
      </Paper>
    );
  }

  return (
    <Card variant="outlined" sx={{ borderRadius: 2 }}>
      <CardContent sx={{ p: 2.5 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 2, mb: 2 }}>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 600 }}>
              360° Vendor Quality Radar
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Normalized 6-Pillar Telemetry Comparison (Default Weights: 30% Suff, 20% Cov, 15% SLA, 15% Stab, 10% Fric, 10% Lic)
            </Typography>
          </Box>

          {/* Vendor Toggle Filters */}
          <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
            {profiles.map((p) => {
              const color = VENDOR_COLORS[p.vendor_id] || '#757575';
              const active = !!activeVendors[p.vendor_id];
              return (
                <Chip
                  key={p.vendor_id}
                  label={`${p.vendor_name} (${(p.composite_quality * 100).toFixed(1)}%)`}
                  size="small"
                  onClick={() => toggleVendor(p.vendor_id)}
                  sx={{
                    fontWeight: 600,
                    cursor: 'pointer',
                    bgcolor: active ? color : 'transparent',
                    color: active ? '#ffffff' : color,
                    borderColor: color,
                    borderWidth: 1.5,
                    borderStyle: 'solid',
                    '&:hover': {
                      bgcolor: active ? color : 'action.hover',
                    },
                  }}
                />
              );
            })}
          </Stack>
        </Box>

        <Divider sx={{ mb: 2 }} />

        <Grid container spacing={2} alignItems="center">
          {/* Radar Chart SVG */}
          <Grid item xs={12} md={7} sx={{ display: 'flex', justifyContent: 'center' }}>
            <Box sx={{ position: 'relative', width: size, height: size }}>
              <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
                {/* Background Concentric Circles/Polygons */}
                {gridLevels.map((lvl) => (
                  <polygon
                    key={lvl}
                    points={getGridPolygon(lvl)}
                    fill="none"
                    stroke="#e0e0e0"
                    strokeWidth="1"
                    strokeDasharray={lvl < 1.0 ? '3,3' : undefined}
                  />
                ))}

                {/* Level Labels (20%, 40%, 60%, 80%, 100%) */}
                {gridLevels.map((lvl) => {
                  const { y } = getCoordinates(0, lvl);
                  return (
                    <text
                      key={lvl}
                      x={cx + 4}
                      y={y + 10}
                      fontSize="9"
                      fill="#9e9e9e"
                      textAnchor="start"
                    >
                      {`${Math.round(lvl * 100)}%`}
                    </text>
                  );
                })}

                {/* Radial Spokes */}
                {dimensionAxes.map((_, i) => {
                  const { x, y } = getCoordinates(i, 1.0);
                  return (
                    <line
                      key={i}
                      x1={cx}
                      y1={cy}
                      x2={x}
                      y2={y}
                      stroke="#cccccc"
                      strokeWidth="1"
                    />
                  );
                })}

                {/* Vendor Filled Polygons */}
                {vendorPolygons.map(({ profile, pointsStr, coords }) => {
                  const vid = profile.vendor_id;
                  if (!activeVendors[vid]) return null;
                  const color = VENDOR_COLORS[vid] || '#757575';
                  return (
                    <g key={vid}>
                      <polygon
                        points={pointsStr}
                        fill={color}
                        fillOpacity="0.22"
                        stroke={color}
                        strokeWidth="2.5"
                      />
                      {coords.map((c, idx) => {
                        const axis = dimensionAxes[idx];
                        const val = profile.components[axis.key as keyof typeof profile.components] || 0;
                        let rawDesc = '';
                        switch (axis.key) {
                          case 'sufficiency':
                            rawDesc = `Concordance: ${(val * 100).toFixed(1)}%`;
                            break;
                          case 'coverage':
                            rawDesc = `Coverage: ${(val * 100).toFixed(1)}%`;
                            break;
                          case 'sla':
                            rawDesc = `Avg Lag: ${profile.avg_delivery_lag_mins.toFixed(0)}m, Breaches: ${profile.sla_breach_count}`;
                            break;
                          case 'stability':
                            rawDesc = `Revisions: ${profile.total_revisions} (${profile.revision_rate_pct.toFixed(2)}%)`;
                            break;
                          case 'friction':
                            rawDesc = `Labor Drag: $${profile.friction_cost.toLocaleString()} (${profile.investigation_hours.toFixed(0)} hrs)`;
                            break;
                          case 'licensing':
                            rawDesc = `Rights Index: ${profile.rights_score.toFixed(1)}/100`;
                            break;
                        }
                        return (
                          <circle
                            key={idx}
                            cx={c.x}
                            cy={c.y}
                            r={4.5}
                            fill={color}
                            stroke="#ffffff"
                            strokeWidth="1.5"
                            style={{ cursor: 'pointer' }}
                            onMouseEnter={() => {
                              setHoveredPoint({
                                vendor: profile.vendor_name,
                                axis: axis.label,
                                score: val,
                                rawDesc,
                                x: c.x,
                                y: c.y,
                              });
                            }}
                            onMouseLeave={() => setHoveredPoint(null)}
                          />
                        );
                      })}
                    </g>
                  );
                })}

                {/* Axis Labels Outside Perimeter */}
                {dimensionAxes.map((axis, i) => {
                  const { x, y } = getCoordinates(i, 1.18);
                  const isTop = y < cy - 20;
                  const isBottom = y > cy + 20;
                  const anchor = x < cx - 20 ? 'end' : x > cx + 20 ? 'start' : 'middle';
                  return (
                    <text
                      key={axis.key}
                      x={x}
                      y={y + (isTop ? -4 : isBottom ? 12 : 4)}
                      textAnchor={anchor}
                      fontSize="11"
                      fontWeight="600"
                      fill="#424242"
                    >
                      {axis.label} ({axis.weight})
                    </text>
                  );
                })}
              </svg>

              {/* Tooltip Overlay */}
              {hoveredPoint && (
                <Paper
                  elevation={4}
                  sx={{
                    position: 'absolute',
                    left: Math.min(size - 180, Math.max(10, hoveredPoint.x - 70)),
                    top: Math.max(10, hoveredPoint.y - 75),
                    p: 1.2,
                    pointerEvents: 'none',
                    bgcolor: 'rgba(33, 33, 33, 0.95)',
                    color: '#ffffff',
                    borderRadius: 1,
                    minWidth: 160,
                    zIndex: 10,
                  }}
                >
                  <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', color: '#90caf9' }}>
                    {hoveredPoint.vendor} — {hoveredPoint.axis}
                  </Typography>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    Normalized Score: {(hoveredPoint.score * 100).toFixed(1)}%
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#e0e0e0', display: 'block' }}>
                    {hoveredPoint.rawDesc}
                  </Typography>
                </Paper>
              )}
            </Box>
          </Grid>

          {/* Metric Details Panel */}
          <Grid item xs={12} md={5}>
            <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1.5 }}>
              Active Vendors Comparison Matrix
            </Typography>
            <Stack spacing={1.5}>
              {profiles
                .filter((p) => activeVendors[p.vendor_id])
                .map((p) => {
                  const color = VENDOR_COLORS[p.vendor_id] || '#757575';
                  return (
                    <Paper
                      key={p.vendor_id}
                      variant="outlined"
                      sx={{
                        p: 1.5,
                        borderLeft: `5px solid ${color}`,
                        borderRadius: 1.5,
                      }}
                    >
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 0.5 }}>
                        <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                          {p.vendor_name}
                        </Typography>
                        <Chip
                          size="small"
                          label={`Q: ${(p.composite_quality * 100).toFixed(1)}%`}
                          sx={{ fontWeight: 700, bgcolor: color, color: '#fff' }}
                        />
                      </Box>
                      <Grid container spacing={1} sx={{ mt: 0.5 }}>
                        <Grid item xs={6}>
                          <Typography variant="caption" color="text.secondary">Annual Spend:</Typography>
                          <Typography variant="body2" sx={{ fontWeight: 600 }}>
                            ${p.annual_spend.toLocaleString()}
                          </Typography>
                        </Grid>
                        <Grid item xs={6}>
                          <Typography variant="caption" color="text.secondary">Cost / Quality Point:</Typography>
                          <Typography variant="body2" sx={{ fontWeight: 600 }}>
                            ${p.cost_per_quality_point.toLocaleString()}
                          </Typography>
                        </Grid>
                        <Grid item xs={6}>
                          <Typography variant="caption" color="text.secondary">Steward Labor Drag:</Typography>
                          <Typography variant="body2">
                            ${p.friction_cost.toLocaleString()} ({p.investigation_hours.toFixed(0)}h)
                          </Typography>
                        </Grid>
                        <Grid item xs={6}>
                          <Typography variant="caption" color="text.secondary">Delivery SLA & Lag:</Typography>
                          <Typography variant="body2">
                            {(p.components.sla * 100).toFixed(1)}% ({p.avg_delivery_lag_mins.toFixed(0)}m)
                          </Typography>
                        </Grid>
                        <Grid item xs={6}>
                          <Typography variant="caption" color="text.secondary">Stability (Revisions):</Typography>
                          <Typography variant="body2">
                            {(p.components.stability * 100).toFixed(1)}% ({p.total_revisions})
                          </Typography>
                        </Grid>
                        <Grid item xs={6}>
                          <Typography variant="caption" color="text.secondary">Commercial Rights:</Typography>
                          <Typography variant="body2">
                            {p.rights_score.toFixed(1)} / 100
                          </Typography>
                        </Grid>
                      </Grid>
                    </Paper>
                  );
                })}
              {profiles.filter((p) => activeVendors[p.vendor_id]).length === 0 && (
                <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic', py: 3, textAlign: 'center' }}>
                  Select at least one vendor chip above to view details.
                </Typography>
              )}
            </Stack>
          </Grid>
        </Grid>
      </CardContent>
    </Card>
  );
};
