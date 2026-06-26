import React from 'react';

interface ETLHealthProps {
  data?: {
    last_run: {
      etl_run_id: string;
      status: 'SUCCESS' | 'FAILED' | 'RUNNING';
      duration_ms: number;
      rules_evaluated: number;
      scenarios_evaluated: number;
      wasm_version: string;
    };
  };
  isLoading?: boolean;
  error?: Error | null;
  onTriggerRun?: () => void;
  onViewLogs?: () => void;
}

export const ETLHealth: React.FC<ETLHealthProps> = ({
  data,
  isLoading,
  error,
  onTriggerRun,
  onViewLogs,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 h-48 flex items-center justify-center">
        <div className="text-slate-500">Loading ETL health...</div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 h-48 flex items-center justify-center">
        <div className="text-red-600 dark:text-red-400 text-sm">
          {error?.message || 'Failed to load ETL health'}
        </div>
      </div>
    );
  }

  const { last_run } = data;
  const statusColor =
    last_run.status === 'SUCCESS'
      ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
      : last_run.status === 'RUNNING'
      ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400'
      : 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400';

  const statusDot =
    last_run.status === 'SUCCESS'
      ? 'bg-green-500'
      : last_run.status === 'RUNNING'
      ? 'bg-blue-500 animate-pulse'
      : 'bg-red-500';

  return (
    <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 flex flex-col">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <span className="text-lg font-bold text-slate-900 dark:text-white">ETL Run Health</span>
        </div>
        <div className={`px-3 py-1 rounded text-xs font-bold flex items-center gap-2 ${statusColor}`}>
          <span className={`w-2 h-2 rounded-full ${statusDot}`} />
          {last_run.status}
        </div>
      </div>

      <div className="grid grid-cols-3 gap-6 mb-6">
        <div className="space-y-1">
          <p className="text-xs text-slate-500 dark:text-slate-400 font-medium uppercase tracking-wider">
            Last Run Time
          </p>
          <p className="text-lg font-semibold text-slate-900 dark:text-white">
            {new Date().toLocaleTimeString()}
          </p>
        </div>
        <div className="space-y-1">
          <p className="text-xs text-slate-500 dark:text-slate-400 font-medium uppercase tracking-wider">
            Engine Version
          </p>
          <p className="text-lg font-mono font-semibold text-slate-900 dark:text-white">
            {last_run.wasm_version}
          </p>
        </div>
        <div className="space-y-1">
          <p className="text-xs text-slate-500 dark:text-slate-400 font-medium uppercase tracking-wider">
            Processing Logic
          </p>
          <p className="text-lg font-semibold text-slate-900 dark:text-white">WASM Core</p>
        </div>
      </div>

      <div className="bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-700 rounded-lg p-4 mb-6">
        <div className="flex items-center justify-between mb-4">
          <span className="text-xs font-bold uppercase text-slate-500 dark:text-slate-400">
            Pipeline Stages
          </span>
          <span className="text-xs font-medium text-slate-400">
            Total duration: {(last_run.duration_ms / 1000).toFixed(0)}s
          </span>
        </div>

        <div className="space-y-3">
          <div className="flex items-center gap-3">
            <span className="w-24 text-xs font-medium text-slate-600 dark:text-slate-300">
              Ingestion
            </span>
            <div className="flex-1 h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
              <div className="h-full bg-green-500" style={{ width: '100%' }} />
            </div>
            <span className="text-xs font-mono text-slate-600 dark:text-slate-400">42s</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="w-24 text-xs font-medium text-slate-600 dark:text-slate-300">
              Validation
            </span>
            <div className="flex-1 h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
              <div className="h-full bg-green-500" style={{ width: '100%' }} />
            </div>
            <span className="text-xs font-mono text-slate-600 dark:text-slate-400">28s</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="w-24 text-xs font-medium text-slate-600 dark:text-slate-300">
              Aggregation
            </span>
            <div className="flex-1 h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
              <div
                className="h-full bg-blue-500"
                style={{ width: `${(last_run.duration_ms / 132000) * 100}%` }}
              />
            </div>
            <span className="text-xs font-mono text-slate-600 dark:text-slate-400">62s</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <button
          onClick={onTriggerRun}
          className="py-2 bg-blue-600 dark:bg-blue-700 text-white text-sm font-bold rounded-lg hover:bg-blue-700 dark:hover:bg-blue-600 transition-colors"
        >
          Trigger Manual Run
        </button>
        <button
          onClick={onViewLogs}
          className="py-2 border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-slate-100 text-sm font-bold rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
        >
          View System Logs
        </button>
      </div>
    </div>
  );
};

interface AlertItemProps {
  type: 'error' | 'warning' | 'info';
  title: string;
  description: string;
  timestamp?: string;
}

const AlertItem: React.FC<AlertItemProps> = ({ type, title, description, timestamp }) => {
  const typeConfig = {
    error: {
      bg: 'bg-red-50 dark:bg-red-900/20',
      border: 'border-l-4 border-red-500',
      iconColor: 'text-red-600',
      icon: '⚠️',
    },
    warning: {
      bg: 'bg-amber-50 dark:bg-amber-900/20',
      border: 'border-l-4 border-amber-500',
      iconColor: 'text-amber-600',
      icon: '⚡',
    },
    info: {
      bg: 'bg-blue-50 dark:bg-blue-900/20',
      border: 'border-l-4 border-blue-500',
      iconColor: 'text-blue-600',
      icon: 'ℹ️',
    },
  };

  const config = typeConfig[type];

  return (
    <div className={`p-3 ${config.bg} ${config.border} flex gap-3`}>
      <span className={`text-xl ${config.iconColor}`}>{config.icon}</span>
      <div className="flex flex-col gap-1">
        <span className="text-sm font-bold text-slate-900 dark:text-white">{title}</span>
        <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
          {description}
        </p>
        {timestamp && (
          <span className="text-xs text-slate-500 uppercase font-bold mt-1">{timestamp}</span>
        )}
      </div>
    </div>
  );
};

interface AlertsPanelProps {
  data?: {
    hard_breaches: Array<{ rule_code: string; portfolio_id: string; metric: number }>;
    scenario_losses: Array<{ scenario_id: string; name: string; pnl: number }>;
    etl_failures: Array<{ message: string }>;
    soft_breaches?: Array<{ rule_code: string; description: string }>;
    reg_breaches?: Array<{ name: string; description: string }>;
  };
  isLoading?: boolean;
  error?: Error | null;
  onViewAll?: () => void;
}

export const AlertsPanel: React.FC<AlertsPanelProps> = ({
  data,
  isLoading,
  error,
  onViewAll,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 h-48 flex items-center justify-center">
        <div className="text-slate-500">Loading alerts...</div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 h-48 flex items-center justify-center">
        <div className="text-red-600 dark:text-red-400 text-sm">
          {error?.message || 'Failed to load alerts'}
        </div>
      </div>
    );
  }

  const allAlerts = [
    ...(data.hard_breaches || []).map((b) => ({
      type: 'error' as const,
      title: `Hard Breach: ${b.rule_code}`,
      description: `Breach detected in portfolio ${b.portfolio_id}. Metric: ${b.metric.toFixed(3)}`,
      timestamp: '2 mins ago',
    })),
    ...(data.scenario_losses || []).map((s) => ({
      type: 'warning' as const,
      title: `Scenario: ${s.name}`,
      description: `PnL Impact: $${(s.pnl / 1000000).toFixed(1)}M. Consider stress test mitigation.`,
      timestamp: '14 mins ago',
    })),
    ...(data.soft_breaches || []).map((b) => ({
      type: 'warning' as const,
      title: `Soft Breach: ${b.rule_code}`,
      description: b.description || 'Threshold approaching.',
      timestamp: '45 mins ago',
    })),
    ...(data.reg_breaches || []).map((b) => ({
      type: 'error' as const,
      title: `Regulatory: ${b.name}`,
      description: b.description,
      timestamp: '1 hour ago',
    })),
  ];

  const limitedAlerts = allAlerts.slice(0, 4);
  const hasMore = allAlerts.length > 4;

  return (
    <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-6 flex flex-col">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <span className="text-lg font-bold text-slate-900 dark:text-white">Critical Alerts</span>
        </div>
        {allAlerts.length > 0 && (
          <span className="text-xs font-bold px-2 py-1 bg-red-500/10 text-red-600 dark:text-red-400 rounded-full">
            {allAlerts.length} New
          </span>
        )}
      </div>

      <div className="space-y-3 flex-1 overflow-y-auto pr-2 max-h-[400px]">
        {limitedAlerts.length === 0 ? (
          <div className="text-slate-500 text-sm text-center py-8">
            No critical alerts at this time
          </div>
        ) : (
          limitedAlerts.map((alert, idx) => (
            <AlertItem
              key={idx}
              type={alert.type}
              title={alert.title}
              description={alert.description}
              timestamp={alert.timestamp}
            />
          ))
        )}
      </div>

      {hasMore || limitedAlerts.length > 0 ? (
        <div className="mt-4 pt-4 border-t border-slate-100 dark:border-slate-700">
          <button
            onClick={onViewAll}
            className="w-full text-center text-sm font-semibold text-blue-600 dark:text-blue-400 hover:underline"
          >
            Manage All Alerts ({allAlerts.length} total)
          </button>
        </div>
      ) : null}
    </div>
  );
};
