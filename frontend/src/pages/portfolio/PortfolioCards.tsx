import React from 'react';

interface PortfolioOverviewCardProps {
  data?: {
    name: string;
    aum: number;
    currency: string;
    strategy: string;
    benchmark_id: string;
    valuation_date: string;
  };
  isLoading?: boolean;
  error?: Error | null;
}

export const PortfolioOverviewCard: React.FC<PortfolioOverviewCardProps> = ({
  data,
  isLoading,
  error,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm h-48 animate-pulse" />
    );
  }

  if (error || !data) {
    return (
      <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="text-red-600 dark:text-red-400 text-sm">
          {error?.message || 'Failed to load overview'}
        </div>
      </div>
    );
  }

  const formatCurrency = (value: number) => {
    if (value >= 1e9) return `$${(value / 1e9).toFixed(1)}B`;
    if (value >= 1e6) return `$${(value / 1e6).toFixed(1)}M`;
    return `$${value.toFixed(0)}`;
  };

  return (
    <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
      <div className="flex items-center gap-2 mb-6">
        <h3 className="font-bold text-lg text-slate-900 dark:text-white">Overview</h3>
      </div>
      <div className="space-y-4">
        <div className="flex justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
          <span className="text-sm text-slate-500 dark:text-slate-400">AUM</span>
          <span className="text-sm font-bold text-slate-900 dark:text-white tabular-nums">
            {formatCurrency(data.aum)}
          </span>
        </div>
        <div className="flex justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
          <span className="text-sm text-slate-500 dark:text-slate-400">Currency</span>
          <span className="text-sm font-bold text-slate-900 dark:text-white">
            {data.currency}
          </span>
        </div>
        <div className="flex justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
          <span className="text-sm text-slate-500 dark:text-slate-400">Strategy</span>
          <span className="text-sm font-bold text-slate-900 dark:text-white">
            {data.strategy}
          </span>
        </div>
        <div className="flex justify-between">
          <span className="text-sm text-slate-500 dark:text-slate-400">Benchmark</span>
          <span className="text-sm font-bold text-slate-900 dark:text-white">
            {data.benchmark_id}
          </span>
        </div>
      </div>
    </div>
  );
};

interface RiskSnapshotCardProps {
  data?: {
    total_volatility: number;
    var_95: number;
    var_99: number;
    worst_scenarios: Array<{ scenario_id: string; name: string; pnl: number }>;
  };
  isLoading?: boolean;
  error?: Error | null;
}

export const RiskSnapshotCard: React.FC<RiskSnapshotCardProps> = ({
  data,
  isLoading,
  error,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm h-48 animate-pulse" />
    );
  }

  if (error || !data) {
    return (
      <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="text-red-600 dark:text-red-400 text-sm">
          {error?.message || 'Failed to load risk snapshot'}
        </div>
      </div>
    );
  }

  const worstScenario = data.worst_scenarios[0];

  return (
    <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
      <div className="flex items-center gap-2 mb-6">
        <h3 className="font-bold text-lg text-slate-900 dark:text-white">Risk Snapshot</h3>
      </div>
      <div className="space-y-4">
        <div className="flex justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
          <span className="text-sm text-slate-500 dark:text-slate-400">Volatility</span>
          <span className="text-sm font-bold text-slate-900 dark:text-white tabular-nums">
            {data.total_volatility.toFixed(3)}
          </span>
        </div>
        <div className="flex justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
          <span className="text-sm text-slate-500 dark:text-slate-400">VaR 95%</span>
          <span className="text-sm font-bold text-slate-900 dark:text-white tabular-nums">
            {data.var_95.toFixed(3)}
          </span>
        </div>
        <div className="flex justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
          <span className="text-sm text-slate-500 dark:text-slate-400">VaR 99%</span>
          <span className="text-sm font-bold text-slate-900 dark:text-white tabular-nums">
            {data.var_99.toFixed(3)}
          </span>
        </div>
        {worstScenario && (
          <div className="bg-red-50 dark:bg-red-900/20 p-3 rounded border border-red-100 dark:border-red-900 mt-4">
            <div className="flex justify-between items-center">
              <span className="text-xs font-bold text-red-600 dark:text-red-400 uppercase">
                Worst Scenario
              </span>
              <span className="text-sm font-black text-red-700 dark:text-red-300 tabular-nums">
                {(worstScenario.pnl / 1000000).toFixed(1)}M
              </span>
            </div>
            <p className="text-xs text-red-600 dark:text-red-400 mt-1">{worstScenario.name}</p>
          </div>
        )}
      </div>
    </div>
  );
};

interface ComplianceSnapshotCardProps {
  data?: {
    rules_evaluated: number;
    pass_rate: number;
    hard_breaches: Array<{ rule_code: string }>;
    soft_breaches: Array<{ rule_code: string }>;
  };
  isLoading?: boolean;
  error?: Error | null;
}

export const ComplianceSnapshotCard: React.FC<ComplianceSnapshotCardProps> = ({
  data,
  isLoading,
  error,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm h-48 animate-pulse" />
    );
  }

  if (error || !data) {
    return (
      <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="text-red-600 dark:text-red-400 text-sm">
          {error?.message || 'Failed to load compliance snapshot'}
        </div>
      </div>
    );
  }

  const passRateColor =
    data.pass_rate >= 0.95
      ? 'text-green-600 dark:text-green-400'
      : data.pass_rate >= 0.90
      ? 'text-amber-600 dark:text-amber-400'
      : 'text-red-600 dark:text-red-400';

  return (
    <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
      <div className="flex items-center gap-2 mb-6">
        <h3 className="font-bold text-lg text-slate-900 dark:text-white">Compliance</h3>
      </div>
      <div className="space-y-4">
        <div className="flex justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
          <span className="text-sm text-slate-500 dark:text-slate-400">Rules Evaluated</span>
          <span className="text-sm font-bold text-slate-900 dark:text-white tabular-nums">
            {data.rules_evaluated}
          </span>
        </div>
        <div className="flex justify-between pb-3 border-b border-slate-100 dark:border-slate-800">
          <span className="text-sm text-slate-500 dark:text-slate-400">Pass Rate</span>
          <span className={`text-sm font-bold tabular-nums ${passRateColor}`}>
            {(data.pass_rate * 100).toFixed(1)}%
          </span>
        </div>
        <div className="flex gap-2 pt-2">
          {data.hard_breaches.length > 0 && (
            <span className="flex items-center gap-1 px-3 py-1 bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300 text-xs font-bold rounded-full uppercase">
              {data.hard_breaches.length} Hard
            </span>
          )}
          {data.soft_breaches.length > 0 && (
            <span className="flex items-center gap-1 px-3 py-1 bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300 text-xs font-bold rounded-full uppercase">
              {data.soft_breaches.length} Soft
            </span>
          )}
          {data.hard_breaches.length === 0 && data.soft_breaches.length === 0 && (
            <span className="flex items-center gap-1 px-3 py-1 bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300 text-xs font-bold rounded-full uppercase">
              ✓ Compliant
            </span>
          )}
        </div>
      </div>
    </div>
  );
};
