import { useMemo } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  BarChart,
  Bar,
  ComposedChart,
  Area,
  AreaChart
} from 'recharts';
import { Badge } from '@/components/ui/badge';
import { AlertTriangle } from 'lucide-react';

import type { PoPMetricWithContract } from './PoPMetricExplorer';

interface PoPChartProps {
  metrics: PoPMetricWithContract[];
  height?: number;
  chartType?: 'line' | 'bar' | 'area' | 'composed';
  showAnomalies?: boolean;
  showPercentChange?: boolean;
}

export const PoPChart: React.FC<PoPChartProps> = ({
  metrics,
  height = 400,
  chartType = 'composed',
  showAnomalies = true,
  showPercentChange = true
}) => {
  const chartData = useMemo(() => {
    // Group metrics by period for multi-metric comparison
    const periodMap = new Map<string, any>();

    metrics.forEach(metric => {
      const periodKey = `${metric.period_start}_${metric.period_end}`;

      if (!periodMap.has(periodKey)) {
        periodMap.set(periodKey, {
          period: formatPeriodLabel(metric.period_start, metric.period_end, 'month'), // Default granularity
          periodStart: metric.period_start,
          periodEnd: metric.period_end,
        });
      }

      const periodData = periodMap.get(periodKey);
      periodData[metric.name] = metric.current_value || 0;
      periodData[`${metric.name}_previous`] = metric.previous_value || 0;
      periodData[`${metric.name}_change`] = metric.percent_change || 0;
      periodData[`${metric.name}_anomaly`] = metric.has_anomalies;
      periodData[`${metric.name}_anomaly_count`] = metric.anomaly_count;
    });

    return Array.from(periodMap.values()).sort((a, b) =>
      new Date(a.periodStart).getTime() - new Date(b.periodStart).getTime()
    );
  }, [metrics]);

  const formatPeriodLabel = (start: string, end: string, granularity: string) => {
    const startDate = new Date(start);
    const endDate = new Date(end);

    switch (granularity) {
      case 'month':
        return startDate.toLocaleDateString('en-US', { year: 'numeric', month: 'short' });
      case 'quarter':
        const year = startDate.getFullYear();
        const quarter = Math.floor(startDate.getMonth() / 3) + 1;
        return `${year} Q${quarter}`;
      case 'year':
        return startDate.getFullYear().toString();
      default:
        return `${startDate.toLocaleDateString()} - ${endDate.toLocaleDateString()}`;
    }
  };

  const formatValue = (value: number) => {
    if (value >= 1000000) {
      return `${(value / 1000000).toFixed(1)}M`;
    } else if (value >= 1000) {
      return `${(value / 1000).toFixed(1)}K`;
    }
    return value.toLocaleString();
  };

  const formatPercent = (value: number) => {
    return `${value >= 0 ? '+' : ''}${value.toFixed(1)}%`;
  };

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 border border-gray-200 rounded-lg shadow-lg">
          <p className="font-medium mb-2">{label}</p>
          {payload.map((entry: any, index: number) => {
            if (entry.dataKey.includes('_change') && showPercentChange) {
              return (
                <p key={index} className="text-sm">
                  {entry.name.replace('_change', '')} Change: {formatPercent(entry.value)}
                </p>
              );
            } else if (!entry.dataKey.includes('_') && !entry.dataKey.includes('anomaly')) {
              return (
                <p key={index} className="text-sm">
                  {entry.name}: {formatValue(entry.value)}
                </p>
              );
            }
            return null;
          })}
        </div>
      );
    }
    return null;
  };

  const renderAnomalyIndicators = () => {
    if (!showAnomalies) return null;

    return chartData.map((data, index) => {
      const anomalies = metrics.filter(m =>
        m.period_start === data.periodStart &&
        m.period_end === data.periodEnd &&
        m.has_anomalies
      );

      if (anomalies.length === 0) return null;

      return (
        <div
          key={`anomaly-${index}`}
          className="absolute top-4 right-4 flex items-center space-x-1"
        >
          <AlertTriangle className="w-4 h-4 text-red-500" />
          <Badge variant="destructive" className="text-xs">
            {anomalies.length}
          </Badge>
        </div>
      );
    });
  };

  const colors = [
    '#3B82F6', '#EF4444', '#10B981', '#F59E0B', '#8B5CF6',
    '#06B6D4', '#84CC16', '#F97316', '#EC4899', '#6B7280'
  ];

  const renderChart = () => {
    const commonProps = {
      data: chartData,
      margin: { top: 20, right: 30, left: 20, bottom: 5 },
      height: height
    };

    switch (chartType) {
      case 'line':
        return (
          <LineChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="period" />
            <YAxis tickFormatter={formatValue} />
            <Tooltip content={<CustomTooltip />} />
            <Legend />
            {metrics.map((metric, index) => (
              <Line
                key={metric.id}
                type="monotone"
                dataKey={metric.name}
                stroke={colors[index % colors.length]}
                strokeWidth={2}
                dot={{ r: 4 }}
                activeDot={{ r: 6 }}
              />
            ))}
          </LineChart>
        );

      case 'bar':
        return (
          <BarChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="period" />
            <YAxis tickFormatter={formatValue} />
            <Tooltip content={<CustomTooltip />} />
            <Legend />
            {metrics.map((metric, index) => (
              <Bar
                key={metric.id}
                dataKey={metric.name}
                fill={colors[index % colors.length]}
              />
            ))}
          </BarChart>
        );

      case 'area':
        return (
          <AreaChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="period" />
            <YAxis tickFormatter={formatValue} />
            <Tooltip content={<CustomTooltip />} />
            <Legend />
            {metrics.map((metric, index) => (
              <Area
                key={metric.id}
                type="monotone"
                dataKey={metric.name}
                stackId="1"
                stroke={colors[index % colors.length]}
                fill={colors[index % colors.length]}
                fillOpacity={0.6}
              />
            ))}
          </AreaChart>
        );

      case 'composed':
      default:
        return (
          <ComposedChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="period" />
            <YAxis yAxisId="value" orientation="left" tickFormatter={formatValue} />
            <YAxis yAxisId="change" orientation="right" tickFormatter={formatPercent} />
            <Tooltip content={<CustomTooltip />} />
            <Legend />
            {metrics.map((metric, index) => (
              <>
                <Bar
                  yAxisId="value"
                  dataKey={metric.name}
                  fill={colors[index % colors.length]}
                  fillOpacity={0.7}
                />
                {showPercentChange && (
                  <Line
                    yAxisId="change"
                    type="monotone"
                    dataKey={`${metric.name}_change`}
                    stroke={colors[index % colors.length]}
                    strokeWidth={3}
                    dot={{ r: 4 }}
                    activeDot={{ r: 6 }}
                  />
                )}
              </>
            ))}
          </ComposedChart>
        );
    }
  };

  if (metrics.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-gray-500">
        No data available for chart
      </div>
    );
  }

  return (
    <div className="relative">
      <ResponsiveContainer width="100%" height={height}>
        {renderChart()}
      </ResponsiveContainer>
      {showAnomalies && renderAnomalyIndicators()}
    </div>
  );
};
