import React from "react";
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
import type { Heatmap } from "../types";
import "./HeatmapChart.css";

export interface HeatmapChartProps {
  heatmap: Heatmap | null;
  title?: string;
  height?: number;
  loading?: boolean;
}

// Color scale for heatmap: green (fast) -> yellow -> red (slow)
function getColorForLatency(ms: number, maxMs: number = 1000): string {
  const ratio = Math.min(ms / maxMs, 1);

  if (ratio < 0.33) {
    // Green zone: 0-33% of max
    return `rgba(34, 197, 94, ${0.3 + ratio * 0.7})`;
  }
  if (ratio < 0.67) {
    // Yellow zone: 33-67% of max
    return `rgba(234, 179, 8, ${0.3 + (ratio - 0.33) * 0.7})`;
  }
  // Red zone: 67-100% of max
  return `rgba(220, 38, 38, ${0.3 + (ratio - 0.67) * 0.7})`;
}

export function HeatmapChart({
  heatmap,
  title = "Latency Heatmap",
  height = 400,
  loading = false,
}: HeatmapChartProps) {
  if (loading) {
    return (
      <div className="heatmap-loading">
        <div className="spinner-inline">Loading heatmap...</div>
      </div>
    );
  }

  if (!heatmap || heatmap.series.length === 0) {
    return (
      <div className="heatmap-empty">
        <p>No heatmap data available</p>
      </div>
    );
  }

  // Transform heatmap data for display
  // We'll create a matrix visualization
  const maxLatency = Math.max(
    ...heatmap.series.flatMap((s) => s.values.map((v) => v.p95_ms || v.value))
  );

  return (
    <div className="heatmap-container">
      {title && <h3 className="heatmap-title">{title}</h3>}

      <div className="heatmap-grid">
        {/* Y-axis labels (dimension values) */}
        <div className="heatmap-labels">
          {heatmap.series.map((series, i) => (
            <div key={i} className="heatmap-label">
              {series.key}
            </div>
          ))}
        </div>

        {/* Heatmap cells */}
        <div className="heatmap-cells">
          {heatmap.series.map((series, seriesIdx) => (
            <div key={seriesIdx} className="heatmap-row">
              {series.values.map((value, cellIdx) => {
                const latency = value.p95_ms || value.value;
                const color = getColorForLatency(latency, maxLatency);
                const time = new Date(value.time).toLocaleTimeString();

                return (
                  <div
                    key={cellIdx}
                    className="heatmap-cell"
                    style={{ backgroundColor: color }}
                    title={`${series.key} @ ${time}: ${Math.round(latency)}ms`}
                  >
                    <span className="heatmap-value">{Math.round(latency)}</span>
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      </div>

      {/* Legend */}
      <div className="heatmap-legend">
        <div className="legend-item">
          <div className="legend-color" style={{ backgroundColor: "rgba(34, 197, 94, 0.7)" }}></div>
          <span>Fast (&lt;{Math.round(maxLatency * 0.33)}ms)</span>
        </div>
        <div className="legend-item">
          <div className="legend-color" style={{ backgroundColor: "rgba(234, 179, 8, 0.7)" }}></div>
          <span>Medium ({Math.round(maxLatency * 0.33)}-{Math.round(maxLatency * 0.67)}ms)</span>
        </div>
        <div className="legend-item">
          <div className="legend-color" style={{ backgroundColor: "rgba(220, 38, 38, 0.7)" }}></div>
          <span>Slow (&gt;{Math.round(maxLatency * 0.67)}ms)</span>
        </div>
      </div>
    </div>
  );
}
