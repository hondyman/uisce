import React from "react";
import {
  LineChart as RechartsLineChart,
  Line,
  BarChart as RechartsBarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import "./Charts.css";

export interface ChartData {
  name: string;
  value?: number;
  [key: string]: string | number | undefined;
}

export interface LineChartProps {
  data: ChartData[];
  dataKey: string;
  title?: string;
  height?: number;
}

export function LineChart({
  data,
  dataKey,
  title,
  height = 300,
}: LineChartProps) {
  return (
    <div className="chart-container">
      {title && <h3 className="chart-title">{title}</h3>}
      <ResponsiveContainer width="100%" height={height}>
        <RechartsLineChart data={data} margin={{ top: 5, right: 30, left: 0, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
          <XAxis dataKey="name" stroke="var(--color-muted)" />
          <YAxis stroke="var(--color-muted)" />
          <Tooltip
            contentStyle={{
              backgroundColor: "rgba(0, 0, 0, 0.8)",
              border: "1px solid var(--color-border)",
              borderRadius: "4px",
            }}
            labelStyle={{ color: "var(--color-text)" }}
          />
          <Legend />
          <Line
            type="monotone"
            dataKey={dataKey}
            stroke="var(--color-accent)"
            dot={false}
            strokeWidth={2}
          />
        </RechartsLineChart>
      </ResponsiveContainer>
    </div>
  );
}

export interface BarChartProps {
  data: ChartData[];
  dataKey: string;
  title?: string;
  height?: number;
}

export function BarChart({
  data,
  dataKey,
  title,
  height = 300,
}: BarChartProps) {
  return (
    <div className="chart-container">
      {title && <h3 className="chart-title">{title}</h3>}
      <ResponsiveContainer width="100%" height={height}>
        <RechartsBarChart data={data} margin={{ top: 5, right: 30, left: 0, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" />
          <XAxis dataKey="name" stroke="var(--color-muted)" />
          <YAxis stroke="var(--color-muted)" />
          <Tooltip
            contentStyle={{
              backgroundColor: "rgba(0, 0, 0, 0.8)",
              border: "1px solid var(--color-border)",
              borderRadius: "4px",
            }}
            labelStyle={{ color: "var(--color-text)" }}
          />
          <Legend />
          <Bar dataKey={dataKey} fill="var(--color-accent)" />
        </RechartsBarChart>
      </ResponsiveContainer>
    </div>
  );
}
