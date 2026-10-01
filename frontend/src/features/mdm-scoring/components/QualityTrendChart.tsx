import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  Box, Card, CardContent, Typography, Chip, Stack, CircularProgress,
  Alert, MenuItem, Select, FormControl, InputLabel, Button, Tooltip,
  Paper, ToggleButton, ToggleButtonGroup
} from '@mui/material';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import TrendingFlatIcon from '@mui/icons-material/TrendingFlat';
import RefreshIcon from '@mui/icons-material/Refresh';
import StorageIcon from '@mui/icons-material/Storage';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import {
  mdmScoringApi,
  type TrendAnalysisReport,
  type TrendPoint,
  type VendorTrendSeries
} from '../api';

const VENDOR_COLORS: Record<string, string> = {
  BBG: '#ff8c00', // Bloomberg orange
  RFT: '#0088cc', // Refinitiv blue
  FDS: '#00b050', // FactSet green
  ICE: '#7030a0', // ICE purple
  SPG: '#c00000', // S&P red
};

export const DIMENSIONS = [
  { key: 'COMPOSITE', label: '6-Pillar Composite Quality' },
  { key: 'SUFFICIENCY', label: 'Legacy Sufficiency Rate' },
  { key: 'COVERAGE', label: 'Universe Attribute Coverage' },
  { key: 'DELIVERY_TIMELINESS', label: 'SLA Delivery Timeliness' },
  { key: 'STABILITY', label: 'Restatement Stability' },
  { key: 'FRICTION', label: 'Operational Friction Score' },
  { key: 'LICENSING', label: 'Commercial Rights Score' },
];

export const PRESETS = [
  { label: '30 Days (Hot StarRocks)', days: 30 },
  { label: '90 Days (Hot + Warm)', days: 90 },
  { label: '1 Year (Hot + Warm)', days: 365 },
  { label: '3 Years (All 3 Tiers)', days: 1095 },
];

export interface QualityTrendChartProps {
  entityDomain?: string;
  initialDimension?: string;
  initialPresetDays?: number;
  height?: number;
  onPointSelect?: (point: TrendPoint) => void;
}

export const QualityTrendChart: React.FC<QualityTrendChartProps> = ({
  entityDomain,
  initialDimension = 'COMPOSITE',
  initialPresetDays = 90,
  height = 380,
  onPointSelect,
}) => {
  const [dimension, setDimension] = useState<string>(initialDimension);
  const [selectedPreset, setSelectedPreset] = useState<number>(initialPresetDays);
  const [report, setReport] = useState<TrendAnalysisReport | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [visibleVendors, setVisibleVendors] = useState<Record<string, boolean>>({
    BBG: true,
    RFT: true,
    FDS: true,
    ICE: true,
    SPG: true,
  });
  const [hoveredPoint, setHoveredPoint] = useState<{
    point: TrendPoint;
    x: number;
    y: number;
  } | null>(null);

  const fetchTrends = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const toDate = new Date();
      const fromDate = new Date();
      fromDate.setDate(fromDate.getDate() - selectedPreset);

      const dateToStr = toDate.toISOString().slice(0, 10);
      const dateFromStr = fromDate.toISOString().slice(0, 10);

      const res = await mdmScoringApi.trends(
        dimension,
        ['BBG', 'RFT', 'FDS', 'ICE', 'SPG'],
        dateFromStr,
        dateToStr,
        entityDomain
      );
      setReport(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to load historical trend analytics');
    } finally {
      setLoading(false);
    }
  }, [dimension, selectedPreset, entityDomain]);

  useEffect(() => {
    fetchTrends();
  }, [fetchTrends]);

  const toggleVendor = (vendorId: string) => {
    setVisibleVendors((prev) => ({
      ...prev,
      [vendorId]: !prev[vendorId],
    }));
  };

  // Dimensions of SVG canvas
  const chartWidth = 900;
  const chartHeight = height;
  const margin = { top: 35, right: 35, bottom: 45, left: 55 };
  const innerWidth = chartWidth - margin.left - margin.right;
  const innerHeight = chartHeight - margin.top - margin.bottom;

  // Thresholds from API (or default 70 and 50)
  const zoneHigh = report?.quality_zone_high_threshold ?? 70.0;
  const zoneMid = report?.quality_zone_mid_threshold ?? 50.0;

  // Time boundaries and scale
  const { minTime, maxTime, allDates } = useMemo(() => {
    if (!report || !report.series || report.series.length === 0) {
      const now = Date.now();
      return { minTime: now - 90 * 86400000, maxTime: now, allDates: [] };
    }
    const dates: number[] = [];
    report.series.forEach((s) => {
      s.points.forEach((p) => {
        dates.push(new Date(p.as_of_date).getTime());
      });
    });
    if (dates.length === 0) {
      const now = Date.now();
      return { minTime: now - 90 * 86400000, maxTime: now, allDates: [] };
    }
    dates.sort((a, b) => a - b);
    return {
      minTime: dates[0],
      maxTime: dates[dates.length - 1],
      allDates: dates,
    };
  }, [report]);

  const getX = useCallback(
    (timestamp: number) => {
      if (maxTime === minTime) return margin.left + innerWidth / 2;
      const pct = (timestamp - minTime) / (maxTime - minTime);
      return margin.left + pct * innerWidth;
    },
    [minTime, maxTime, margin.left, innerWidth]
  );

  const getY = useCallback(
    (score: number) => {
      // 0 to 100 scale
      const clamped = Math.max(0, Math.min(100, score));
      return margin.top + innerHeight - (clamped / 100) * innerHeight;
    },
    [margin.top, innerHeight]
  );

  // Calculate watermark boundary vertical lines
  const watermarkLines = useMemo(() => {
    if (!report?.watermarks) return [];
    const lines: { x: number; label: string; color: string }[] = [];
    const now = Date.now();

    // Hot/Warm boundary (30 days ago)
    const hotBoundaryTime = now - 30 * 86400000;
    if (hotBoundaryTime > minTime && hotBoundaryTime < maxTime) {
      lines.push({
        x: getX(hotBoundaryTime),
        label: 'HOT (StarRocks) | WARM (PG)',
        color: '#e65100',
      });
    }

    // Warm/Cold boundary (365 days ago)
    const warmBoundaryTime = now - 365 * 86400000;
    if (warmBoundaryTime > minTime && warmBoundaryTime < maxTime) {
      lines.push({
        x: getX(warmBoundaryTime),
        label: 'WARM (PG) | COLD (Iceberg)',
        color: '#7b1fa2',
      });
    }

    return lines;
  }, [report, minTime, maxTime, getX]);

  // Zone rectangles in SVG
  const yHigh = getY(zoneHigh);
  const yMid = getY(zoneMid);
  const yBottom = getY(0);

  return (
    <Card variant="outlined" sx={{ borderRadius: 2, mb: 3 }}>
      <CardContent sx={{ p: 2.5 }}>
        {/* Header Controls */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2, flexWrap: 'wrap', gap: 1.5 }}>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700 }}>
              Quality Trend Trajectory & Watermarks
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Three-tier routing with continuous boundary stitching & zone classification
            </Typography>
          </Box>

          <Stack direction="row" spacing={1.5} alignItems="center" flexWrap="wrap">
            <FormControl size="small" sx={{ minWidth: 200 }}>
              <InputLabel>Scoring Dimension</InputLabel>
              <Select
                value={dimension}
                label="Scoring Dimension"
                onChange={(e) => setDimension(e.target.value)}
              >
                {DIMENSIONS.map((d) => (
                  <MenuItem key={d.key} value={d.key}>
                    {d.label}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <ToggleButtonGroup
              size="small"
              value={selectedPreset}
              exclusive
              onChange={(_, val) => val && setSelectedPreset(val)}
            >
              {PRESETS.map((p) => (
                <ToggleButton key={p.days} value={p.days} sx={{ px: 1.2, py: 0.5, fontSize: '0.75rem' }}>
                  {p.days}d
                </ToggleButton>
              ))}
            </ToggleButtonGroup>

            <Button
              variant="outlined"
              size="small"
              startIcon={<RefreshIcon />}
              onClick={fetchTrends}
              disabled={loading}
            >
              Sync
            </Button>
          </Stack>
        </Box>

        {/* Vendor Toggles */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2, flexWrap: 'wrap' }}>
          <Typography variant="caption" sx={{ fontWeight: 700, mr: 1, color: 'text.secondary' }}>
            VENDORS:
          </Typography>
          {report?.series.map((s) => {
            const isVis = visibleVendors[s.vendor_id] !== false;
            const color = VENDOR_COLORS[s.vendor_id] || '#666';
            return (
              <Chip
                key={s.vendor_id}
                label={`${s.vendor_id} (${s.vendor_name.split(' ')[0]})`}
                size="small"
                onClick={() => toggleVendor(s.vendor_id)}
                sx={{
                  fontWeight: 700,
                  bgcolor: isVis ? `${color}20` : 'transparent',
                  color: isVis ? color : 'text.disabled',
                  border: `1.5px solid ${isVis ? color : '#ccc'}`,
                  cursor: 'pointer',
                  '&:hover': {
                    bgcolor: `${color}35`,
                  },
                }}
              />
            );
          })}
        </Box>

        {/* Loading / Error States */}
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: height - 100 }}>
            <CircularProgress size={32} />
            <Typography variant="body2" sx={{ ml: 2 }}>Rendering multi-tier quality trend...</Typography>
          </Box>
        )}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>
        )}

        {/* SVG Chart Area */}
        {!loading && report && (
          <Box sx={{ position: 'relative', width: '100%', overflowX: 'auto' }}>
            <svg
              viewBox={`0 0 ${chartWidth} ${chartHeight}`}
              style={{ width: '100%', height: 'auto', minWidth: '600px', display: 'block' }}
            >
              {/* Quality Zone Background Shading */}
              {/* High Quality Zone: >= zoneHigh */}
              <rect
                x={margin.left}
                y={margin.top}
                width={innerWidth}
                height={yHigh - margin.top}
                fill="rgba(46, 125, 50, 0.08)"
              />
              {/* Adequate Zone: zoneMid to zoneHigh */}
              <rect
                x={margin.left}
                y={yHigh}
                width={innerWidth}
                height={yMid - yHigh}
                fill="rgba(2, 136, 209, 0.05)"
              />
              {/* At-Risk Zone: 0 to zoneMid */}
              <rect
                x={margin.left}
                y={yMid}
                width={innerWidth}
                height={yBottom - yMid}
                fill="rgba(211, 47, 47, 0.08)"
              />

              {/* Grid Lines & Labels */}
              {[0, 25, 50, 75, 100].map((score) => {
                const y = getY(score);
                return (
                  <g key={score}>
                    <line
                      x1={margin.left}
                      y1={y}
                      x2={chartWidth - margin.right}
                      y2={y}
                      stroke="#e0e0e0"
                      strokeDasharray={score === 50 || score === 70 ? '4,4' : '2,2'}
                      strokeWidth={score === 50 || score === 70 ? 1.5 : 1}
                    />
                    <text
                      x={margin.left - 8}
                      y={y + 4}
                      textAnchor="end"
                      fontSize="11"
                      fill="#666"
                      fontWeight="600"
                    >
                      {score}%
                    </text>
                  </g>
                );
              })}

              {/* Zone Boundary Text Annotations */}
              <text
                x={chartWidth - margin.right - 10}
                y={yHigh - 6}
                textAnchor="end"
                fontSize="10"
                fill="#2e7d32"
                fontWeight="700"
              >
                HIGH QUALITY ZONE (≥ {zoneHigh}%)
              </text>
              <text
                x={chartWidth - margin.right - 10}
                y={yMid - 6}
                textAnchor="end"
                fontSize="10"
                fill="#0288d1"
                fontWeight="700"
              >
                ADEQUATE ZONE ({zoneMid}% – {zoneHigh}%)
              </text>
              <text
                x={chartWidth - margin.right - 10}
                y={yBottom - 8}
                textAnchor="end"
                fontSize="10"
                fill="#d32f2f"
                fontWeight="700"
              >
                AT-RISK QUALITY ZONE (&lt; {zoneMid}%)
              </text>

              {/* Watermark Tier Boundary Vertical Lines */}
              {watermarkLines.map((wl, idx) => (
                <g key={idx}>
                  <line
                    x1={wl.x}
                    y1={margin.top}
                    x2={wl.x}
                    y2={chartHeight - margin.bottom}
                    stroke={wl.color}
                    strokeWidth="2"
                    strokeDasharray="6,4"
                  />
                  <rect
                    x={wl.x - 70}
                    y={margin.top - 18}
                    width={140}
                    height={16}
                    rx="3"
                    fill={wl.color}
                  />
                  <text
                    x={wl.x}
                    y={margin.top - 6}
                    textAnchor="middle"
                    fontSize="9"
                    fill="#ffffff"
                    fontWeight="700"
                  >
                    {wl.label}
                  </text>
                </g>
              ))}

              {/* Date Axis X-Labels */}
              {[0, 0.25, 0.5, 0.75, 1.0].map((frac, idx) => {
                const timeVal = minTime + frac * (maxTime - minTime);
                const x = margin.left + frac * innerWidth;
                const dStr = new Date(timeVal).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
                return (
                  <text
                    key={idx}
                    x={x}
                    y={chartHeight - margin.bottom + 20}
                    textAnchor="middle"
                    fontSize="10"
                    fill="#666"
                  >
                    {dStr}
                  </text>
                );
              })}

              {/* Vendor Trend Lines & Points */}
              {report.series.map((s) => {
                if (visibleVendors[s.vendor_id] === false) return null;
                const color = VENDOR_COLORS[s.vendor_id] || '#666';

                // Sort points chronologically
                const sorted = [...s.points].sort(
                  (a, b) => new Date(a.as_of_date).getTime() - new Date(b.as_of_date).getTime()
                );
                if (sorted.length === 0) return null;

                // Build SVG path
                const pathD = sorted
                  .map((p, idx) => {
                    const px = getX(new Date(p.as_of_date).getTime());
                    const py = getY(p.score_value);
                    return `${idx === 0 ? 'M' : 'L'} ${px.toFixed(1)} ${py.toFixed(1)}`;
                  })
                  .join(' ');

                return (
                  <g key={s.vendor_id}>
                    {/* Path Line */}
                    <path
                      d={pathD}
                      fill="none"
                      stroke={color}
                      strokeWidth="2.5"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />

                    {/* Points */}
                    {sorted.map((p, pIdx) => {
                      const px = getX(new Date(p.as_of_date).getTime());
                      const py = getY(p.score_value);
                      const isHovered = hoveredPoint?.point === p;

                      return (
                        <circle
                          key={pIdx}
                          cx={px}
                          cy={py}
                          r={isHovered ? 6 : 3.5}
                          fill="#ffffff"
                          stroke={color}
                          strokeWidth={isHovered ? 3 : 2}
                          style={{ cursor: 'pointer', transition: 'r 0.15s ease' }}
                          onMouseEnter={() => setHoveredPoint({ point: p, x: px, y: py })}
                          onMouseLeave={() => setHoveredPoint(null)}
                          onClick={() => onPointSelect?.(p)}
                        />
                      );
                    })}
                  </g>
                );
              })}
            </svg>

            {/* Interactive Tooltip Overlay */}
            {hoveredPoint && (
              <Paper
                elevation={4}
                sx={{
                  position: 'absolute',
                  left: `${(hoveredPoint.x / chartWidth) * 100}%`,
                  top: `${hoveredPoint.y - 10}px`,
                  transform: 'translate(-50%, -100%)',
                  p: 1.2,
                  pointerEvents: 'none',
                  bgcolor: 'rgba(33, 33, 33, 0.95)',
                  color: '#fff',
                  borderRadius: 1.5,
                  zIndex: 10,
                  minWidth: 160,
                }}
              >
                <Typography variant="caption" sx={{ fontWeight: 700, color: VENDOR_COLORS[hoveredPoint.point.vendor_id] || '#fff' }}>
                  {hoveredPoint.point.vendor_id} ({hoveredPoint.point.vendor_name})
                </Typography>
                <Typography variant="body2" sx={{ fontWeight: 700 }}>
                  Score: {hoveredPoint.point.score_value.toFixed(1)}%
                </Typography>
                <Typography variant="caption" display="block" sx={{ color: '#ccc' }}>
                  Date: {new Date(hoveredPoint.point.as_of_date).toLocaleDateString()}
                </Typography>
                <Stack direction="row" spacing={0.5} sx={{ mt: 0.5 }}>
                  <Chip
                    size="small"
                    label={hoveredPoint.point.storage_tier}
                    sx={{
                      fontSize: '0.65rem',
                      height: 18,
                      bgcolor:
                        hoveredPoint.point.storage_tier === 'HOT'
                          ? '#e65100'
                          : hoveredPoint.point.storage_tier === 'WARM'
                          ? '#1565c0'
                          : '#6a1b9a',
                      color: '#fff',
                    }}
                  />
                  {hoveredPoint.point.rank_position && (
                    <Chip
                      size="small"
                      label={`Rank #${hoveredPoint.point.rank_position}`}
                      sx={{ fontSize: '0.65rem', height: 18, bgcolor: 'rgba(255,255,255,0.2)', color: '#fff' }}
                    />
                  )}
                </Stack>
              </Paper>
            )}
          </Box>
        )}
      </CardContent>
    </Card>
  );
};
