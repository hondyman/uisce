import React from "react";
import { useQuery } from "@tanstack/react-query";
import { gql, useQuery as useGQLQuery } from "@apollo/client";
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
  // Fetch data if binding is provided
  const {data: queryData, isLoading} = useGQLQuery(
    gql(def.binding?.gql || ""),
    {
      variables: def.binding?.variables || {},
      skip: !def.binding,
    }
  );

  if (isLoading) {
    return <ChartSkeleton title={def.title} />;
  }

  // Extract data from GraphQL response using dataPath
  const chartData = def.binding
    ? extractDataFromPath(queryData, def.binding.dataPath)
    : [];

  return (
    <div className="bg-white rounded-lg shadow p-4">
      {def.title && <h3 className="text-lg font-semibold mb-4">{def.title}</h3>}
      {def.subtitle && <p className="text-sm text-gray-600 mb-2">{def.subtitle}</p>}

      <ResponsiveContainer width="100%" height={300}>
        {renderChart(def, chartData)}
      </ResponsiveContainer>
    </div>
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
          {def.yFields.map((field, idx) => (
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
          {def.yFields.map((field, idx) => (
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
          {def.yFields.map((field, idx) => (
            <Bar key={field} dataKey={field} fill={colors[idx % colors.length]} />
          ))}
        </BarChart>
      );

    default:
      return <div>Unsupported chart type: {def.chartType}</div>;
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
    <div className="bg-white rounded-lg shadow p-4 animate-pulse">
      {title && <div className="h-6 bg-gray-200 rounded w-1/3 mb-4"></div>}
      <div className="h-64 bg-gray-100 rounded"></div>
    </div>
  );
}
