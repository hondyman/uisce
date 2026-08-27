import React from "react";
import { useTheme } from "@mui/material";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Skeleton from "@mui/material/Skeleton";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "../../lib/apiClient";
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import type { ChartComponent } from "../schema";

interface ChartWidgetProps {
  def: ChartComponent;
}

export function ChartWidget({ def }: ChartWidgetProps) {
  const theme = useTheme();
  const { data: queryData, isLoading } = useQuery({
    queryKey: ['genui-chart', def.binding?.endpoint, def.binding?.variables],
    queryFn: async () => {
      if (!def.binding?.endpoint) return null;
      const response = await apiFetch(def.binding.endpoint, {
        method: def.binding.method || 'GET',
      });
      return response.json();
    },
    enabled: !!def.binding?.endpoint,
  });

  if (isLoading) {
    return <ChartSkeleton title={def.title} />;
  }

  const chartData = def.binding?.dataPath
    ? extractDataFromPath(queryData, def.binding.dataPath)
    : queryData || [];

  return (
    <Box sx={{ bgcolor: 'background.paper', borderRadius: 2, boxShadow: 1, p: 2 }}>
      {def.title && (
        <Typography variant="h6" sx={{ mb: 2 }}>
          {def.title}
        </Typography>
      )}
      {def.subtitle && (
        <Typography variant="body2" sx={{ color: 'grey.600', mb: 1 }}>
          {def.subtitle}
        </Typography>
      )}

      <ResponsiveContainer width="100%" height={300}>
        {renderChart(def, chartData)}
      </ResponsiveContainer>
    </Box>
  );
}

function renderChart(def: ChartComponent, data: any[]) {
  const colors = def.colors || ["#8884d8", "#82ca9d", "#ffc658", "#ff7c7c"];

  switch (def.chartType) {
    case "line":
      return (
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey={def.xField} />
          <YAxis />
          <Tooltip />
          {def.legend && <Legend />}
          {(def.yFields || []).map((field: any, idx: any) => (
            <Line
              key={field}
              type="monotone"
              dataKey={field}
              stroke={colors[idx % colors.length]}
              strokeWidth={2}
            />
          ))}
        </LineChart>
      );

    case "area":
      return (
        <AreaChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey={def.xField} />
          <YAxis />
          <Tooltip />
          {def.legend && <Legend />}
          {(def.yFields || []).map((field, idx) => (
            <Area
              key={field}
              type="monotone"
              dataKey={field}
              fill={colors[idx % colors.length]}
              stroke={colors[idx % colors.length]}
            />
          ))}
        </AreaChart>
      );

    case "bar":
      return (
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey={def.xField} />
          <YAxis />
          <Tooltip />
          {def.legend && <Legend />}
          {(def.yFields || []).map((field, idx) => (
            <Bar key={field} dataKey={field} fill={colors[idx % colors.length]} />
          ))}
        </BarChart>
      );

    default:
      return <Box>Unsupported chart type: {def.chartType}</Box>;
  }
}

function extractDataFromPath(obj: any, path: string): any[] {
  const parts = path.split(".");
  let current = obj;
  for (const part of parts) {
    if (current && typeof current === "object") {
      current = current[part];
    } else {
      return [];
    }
  }
  return Array.isArray(current) ? current : [];
}

function ChartSkeleton({ title }: { title?: string }) {
  return (
    <Box sx={{ bgcolor: 'background.paper', borderRadius: 2, boxShadow: 1, p: 2 }}>
      {title && <Skeleton variant="text" width="30%" height={28} sx={{ mb: 2 }} />}
      <Skeleton variant="rectangular" height={256} sx={{ borderRadius: 1 }} />
    </Box>
  );
}
