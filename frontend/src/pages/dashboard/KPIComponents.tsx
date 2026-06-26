import React from 'react';

interface KPIItem {
  label: string;
  value: string | number;
  color?: 'success' | 'error' | 'warning' | 'default';
  trend?: {
    value: number;
    direction: 'up' | 'down';
  };
}

interface KPIGridProps {
  items: KPIItem[];
  title?: string;
  onViewMore?: () => void;
}

export const KPIGrid: React.FC<KPIGridProps> = ({ items, title, onViewMore }) => {
  const getColorClasses = (color?: string) => {
    switch (color) {
      case 'error':
        return 'text-red-600 dark:text-red-400';
      case 'warning':
        return 'text-amber-600 dark:text-amber-400';
      case 'success':
        return 'text-green-600 dark:text-green-400';
      default:
        return 'text-slate-900 dark:text-white';
    }
  };

  const getTrendIcon = (direction: 'up' | 'down', color: string) => {
    const trendColor = direction === 'up' ? 'text-green-600' : 'text-red-600';
    return (
      <span className={`text-xs font-bold ${trendColor}`}>
        {direction === 'up' ? '↑' : '↓'}
      </span>
    );
  };

  return (
    <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 shadow-sm">
      {title && (
        <div className="flex items-center justify-between mb-6">
          <h3 className="font-bold text-lg text-slate-900 dark:text-white">{title}</h3>
          {onViewMore && (
            <button
              onClick={onViewMore}
              className="text-xs text-blue-600 dark:text-blue-400 font-bold uppercase hover:underline"
            >
              View Details →
            </button>
          )}
        </div>
      )}

      <div className={`grid gap-4 grid-cols-${Math.min(items.length, 4)}`}>
        {items.map((item, idx) => (
          <div key={idx} className="flex flex-col gap-2">
            <span className="text-xs text-slate-500 dark:text-slate-400 font-medium uppercase tracking-wider">
              {item.label}
            </span>
            <span className={`text-2xl font-bold ${getColorClasses(item.color)}`}>
              {item.value}
            </span>
            {item.trend && (
              <span className="text-xs font-bold text-slate-600 dark:text-slate-400 flex items-center gap-1">
                {getTrendIcon(item.trend.direction, item.color || 'default')}
                {item.trend.direction === 'up' ? '+' : '-'}
                {Math.abs(item.trend.value).toFixed(1)}%
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

interface ComplianceKPIsProps {
  data?: {
    total_rules: number;
    pass_rate: number;
    hard_breaches: number;
    soft_breaches: number;
  };
  isLoading?: boolean;
  error?: Error | null;
}

export const ComplianceKPIs: React.FC<ComplianceKPIsProps> = ({
  data,
  isLoading,
  error,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 h-48 flex items-center justify-center">
        <div className="text-slate-500">Loading compliance metrics...</div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 h-48 flex items-center justify-center">
        <div className="text-red-600 dark:text-red-400 text-sm">
          {error?.message || 'Failed to load compliance KPIs'}
        </div>
      </div>
    );
  }

  const items: KPIItem[] = [
    {
      label: 'Rules Evaluated',
      value: data.total_rules.toLocaleString(),
      trend: { value: 5.2, direction: 'up' },
    },
    {
      label: 'Pass Rate',
      value: `${(data.pass_rate * 100).toFixed(1)}%`,
      color: 'success',
      trend: { value: 1.2, direction: 'up' },
    },
    {
      label: 'Hard Breaches',
      value: data.hard_breaches,
      color: 'error',
      trend: { value: 8.5, direction: 'down' },
    },
    {
      label: 'Soft Breaches',
      value: data.soft_breaches,
      color: 'warning',
      trend: { value: 3.2, direction: 'up' },
    },
  ];

  return <KPIGrid items={items} title="Compliance Health" />;
};

interface RiskKPIsProps {
  data?: {
    avg_volatility: number;
    avg_var_95: number;
    avg_var_99: number;
    worst_scenario: {
      scenario_id: string;
      name: string;
      pnl: number;
    };
  };
  isLoading?: boolean;
  error?: Error | null;
}

export const RiskKPIs: React.FC<RiskKPIsProps> = ({
  data,
  isLoading,
  error,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 h-48 flex items-center justify-center">
        <div className="text-slate-500">Loading risk metrics...</div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 h-48 flex items-center justify-center">
        <div className="text-red-600 dark:text-red-400 text-sm">
          {error?.message || 'Failed to load risk KPIs'}
        </div>
      </div>
    );
  }

  const items: KPIItem[] = [
    {
      label: 'Avg Volatility',
      value: data.avg_volatility.toFixed(3),
      trend: { value: 0.1, direction: 'up' },
    },
    {
      label: 'VaR 95',
      value: data.avg_var_95.toFixed(3),
      trend: { value: 0.5, direction: 'down' },
    },
    {
      label: 'VaR 99',
      value: data.avg_var_99.toFixed(3),
      trend: { value: 0.2, direction: 'down' },
    },
    {
      label: 'Worst Scenario',
      value: `$${(data.worst_scenario.pnl / 1000000).toFixed(1)}M`,
      color: 'error',
      trend: { value: 2.3, direction: 'up' },
    },
  ];

  return <KPIGrid items={items} title="Market Risk Metrics" />;
};
