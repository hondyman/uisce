import React from "react";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Paper from "@mui/material/Paper";
import CircularProgress from "@mui/material/CircularProgress";
import {
  ComposedChart,
  Bar,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import type { Heatmap } from "@/admin-v2/types";

export interface HeatmapChartProps {
  heatmap: Heatmap | null;
  title?: string;
  height?: number;
  loading?: boolean;
}

function getColorForLatency(ms: number, maxMs: number = 1000): string {
  const ratio = Math.min(ms / maxMs, 1);

  if (ratio < 0.33) {
    return `rgba(34, 197, 94, ${0.3 + ratio * 0.7})`;
  }
  if (ratio < 0.67) {
    return `rgba(234, 179, 8, ${0.3 + (ratio - 0.33) * 0.7})`;
  }
  return `rgba(220, 38, 38, ${0.3 + (ratio - 0.67) * 0.7})`;
}

export function HeatmapChart({
  heatmap,
  title = "Latency Heatmap",
  height = 400,
  loading = false,
}: HeatmapChartProps) {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  if (loading) {
    return (
      <Paper sx={{ p: 3, display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: height }}>
        <CircularProgress size="small" sx={{ mr: 1 }} />
        <Typography variant="body2">Loading heatmap...</Typography>
      </Paper>
    );
  }

  if (!heatmap || heatmap.series.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center', minHeight: height }}>
        <Typography variant="body2" color="text.secondary">No heatmap data available</Typography>
      </Paper>
    );
  }

  const maxLatency = Math.max(
    ...heatmap.series.flatMap((s) => s.values.map((v) => v.p95_ms || v.value))
  );

  return (
    <Paper sx={{ p: 2 }}>
      {title && (
        <Typography variant="h6" sx={{ mb: 2, fontWeight: 600 }}>
          {title}
        </Typography>
      )}

      <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
        <Box sx={{ display: 'flex', flexDirection: 'column', justifyContent: 'space-around', pr: 1 }}>
          {heatmap.series.map((series, i) => (
            <Typography key={i} variant="caption" sx={{ height: 24, display: 'flex', alignItems: 'center' }}>
              {series.key}
            </Typography>
          ))}
        </Box>
        <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-around', mb: 0.5 }}>
            {heatmap.series[0]?.values.map((_, i) => (
              <Typography key={i} variant="caption" sx={{ minWidth: 40, textAlign: 'center' }}>
                {i + 1}
              </Typography>
            ))}
          </Box>
          <Box sx={{ display: 'grid', gridTemplateColumns: `repeat(${heatmap.series[0]?.values.length || 1}, 1fr)`, gap: 0.5 }}>
            {heatmap.series.map((series, seriesIdx) => (
              <React.Fragment key={seriesIdx}>
                {series.values.map((value, cellIdx) => {
                  const latency = value.p95_ms || value.value;
                  const color = getColorForLatency(latency, maxLatency);
                  const time = new Date(value.time).toLocaleTimeString();

                  return (
                    <Box
                      key={cellIdx}
                      sx={{
                        height: 24,
                        bgcolor: color,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        borderRadius: 0.5,
                        cursor: 'pointer',
                      }}
                      title={`${series.key} @ ${time}: ${Math.round(latency)}ms`}
                    >
                      <Typography variant="caption" sx={{ fontSize: '0.65rem', color: '#fff', textShadow: '0 0 2px #000' }}>
                        {Math.round(latency)}
                      </Typography>
                    </Box>
                  );
                })}
              </React.Fragment>
            ))}
          </Box>
        </Box>
      </Box>

      <Box sx={{ display: 'flex', gap: 2, justifyContent: 'center', mt: 2, pt: 2, borderTop: 1, borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <Box sx={{ width: 16, height: 16, bgcolor: 'rgba(34, 197, 94, 0.7)', borderRadius: 0.5 }} />
          <Typography variant="caption">Fast (&lt;{Math.round(maxLatency * 0.33)}ms)</Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <Box sx={{ width: 16, height: 16, bgcolor: 'rgba(234, 179, 8, 0.7)', borderRadius: 0.5 }} />
          <Typography variant="caption">Medium ({Math.round(maxLatency * 0.33)}-{Math.round(maxLatency * 0.67)}ms)</Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <Box sx={{ width: 16, height: 16, bgcolor: 'rgba(220, 38, 38, 0.7)', borderRadius: 0.5 }} />
          <Typography variant="caption">Slow (&gt;{Math.round(maxLatency * 0.67)}ms)</Typography>
        </Box>
      </Box>
    </Paper>
  );
}
