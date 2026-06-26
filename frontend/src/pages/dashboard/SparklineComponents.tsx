import React from 'react';

interface SparklinePoint {
  date: string;
  value: number;
}

interface SparklineCardProps {
  title: string;
  data: SparklinePoint[];
  currentValue: string | number;
  color?: 'primary' | 'success' | 'error' | 'warning';
  trend?: { value: number; direction: 'up' | 'down' };
  metricKey?: string;
}

export const SparklineCard: React.FC<SparklineCardProps> = ({
  title,
  data,
  currentValue,
  color = 'primary',
  trend,
  metricKey = 'value',
}) => {
  const getColorClasses = (col: string) => {
    switch (col) {
      case 'error':
        return { bar: 'bg-red-300 dark:bg-red-600', latest: 'bg-red-500 dark:bg-red-700' };
      case 'warning':
        return { bar: 'bg-amber-300 dark:bg-amber-600', latest: 'bg-amber-500 dark:bg-amber-700' };
      case 'success':
        return { bar: 'bg-green-300 dark:bg-green-600', latest: 'bg-green-500 dark:bg-green-700' };
      default:
        return { bar: 'bg-blue-200 dark:bg-blue-600', latest: 'bg-blue-500 dark:bg-blue-700' };
    }
  };

  const colors = getColorClasses(color);

  if (!data || data.length === 0) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            {title}
          </span>
          <span className="text-sm font-bold">{currentValue}</span>
        </div>
        <div className="text-xs text-slate-400">No data available</div>
      </div>
    );
  }

  // Normalize data for visualization
  const values = data.map(d => (typeof d === 'object' ? d.value : d));
  const minVal = Math.min(...values);
  const maxVal = Math.max(...values);
  const range = maxVal - minVal || 1;

  const normalizedData = values.map(v => ((v - minVal) / range) * 100);

  return (
    <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-4 flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
          {title}
        </span>
        <span className="text-sm font-bold text-slate-900 dark:text-white">{currentValue}</span>
      </div>

      <div className="h-12 w-full flex items-end gap-1 px-1">
        {normalizedData.map((height, idx) => (
          <div
            key={idx}
            className={`flex-1 ${
              idx === normalizedData.length - 1 ? colors.latest : colors.bar
            } rounded-t-sm transition-all duration-200 hover:opacity-80`}
            style={{ height: `${Math.max(height, 5)}%` }}
            title={`${data[idx].date}: ${values[idx]}`}
          />
        ))}
      </div>

      {trend && (
        <div className="flex items-center gap-1 text-xs font-bold text-slate-600 dark:text-slate-400">
          <span className={trend.direction === 'up' ? 'text-green-600' : 'text-red-600'}>
            {trend.direction === 'up' ? '↑' : '↓'} {Math.abs(trend.value).toFixed(1)}%
          </span>
        </div>
      )}
    </div>
  );
};

interface SparklinesGridProps {
  data?: {
    pass_rate: SparklinePoint[];
    hard_breaches: SparklinePoint[];
    volatility: SparklinePoint[];
    etl_duration: SparklinePoint[];
  };
  isLoading?: boolean;
  error?: Error | null;
}

export const SparklinesGrid: React.FC<SparklinesGridProps> = ({
  data,
  isLoading,
  error,
}) => {
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-4 h-32 animate-pulse"
          />
        ))}
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 col-span-full">
          <div className="text-red-600 dark:text-red-400 text-sm">
            {error?.message || 'Failed to load sparklines'}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <SparklineCard
        title="Pass Rate"
        data={data.pass_rate}
        currentValue={`${(data.pass_rate[data.pass_rate.length - 1]?.value * 100 || 0).toFixed(0)}%`}
        color="success"
        trend={{ value: 2.1, direction: 'up' }}
      />
      <SparklineCard
        title="Hard Breaches"
        data={data.hard_breaches}
        currentValue={data.hard_breaches[data.hard_breaches.length - 1]?.value || 0}
        color="error"
        trend={{ value: 5.2, direction: 'down' }}
      />
      <SparklineCard
        title="Volatility"
        data={data.volatility}
        currentValue={data.volatility[data.volatility.length - 1]?.value.toFixed(3) || '0'}
        color="warning"
        trend={{ value: 1.5, direction: 'up' }}
      />
      <SparklineCard
        title="ETL Duration"
        data={data.etl_duration}
        currentValue={`${data.etl_duration[data.etl_duration.length - 1]?.value || 0}s`}
        color="primary"
        trend={{ value: 3.2, direction: 'down' }}
      />
    </div>
  );
};
