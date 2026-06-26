import React from 'react';

interface HoldingTableProps {
  data?: {
    top_positions: Array<{
      security_id: string;
      name: string;
      weight: number;
      sector?: string;
      country?: string;
      price?: number;
      change_pct?: number;
    }>;
    sector_weights?: Array<{ sector: string; weight: number }>;
    country_weights?: Array<{ country: string; weight: number }>;
  };
  isLoading?: boolean;
  error?: Error | null;
  onViewAll?: () => void;
}

export const HoldingsTable: React.FC<HoldingTableProps> = ({
  data,
  isLoading,
  error,
  onViewAll,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm h-96 animate-pulse" />
    );
  }

  if (error || !data) {
    return (
      <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm p-6">
        <div className="text-red-600 dark:text-red-400 text-sm">
          {error?.message || 'Failed to load holdings'}
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm flex flex-col">
      <div className="px-6 py-4 border-b border-slate-100 dark:border-slate-800 flex justify-between items-center">
        <h3 className="font-bold text-slate-900 dark:text-white">Top Positions</h3>
        {onViewAll && (
          <button
            onClick={onViewAll}
            className="text-xs text-blue-600 dark:text-blue-400 font-bold hover:underline"
          >
            View All
          </button>
        )}
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 dark:bg-slate-800/50 text-slate-500 dark:text-slate-400 font-medium uppercase text-xs">
            <tr>
              <th className="px-6 py-3">Security Name</th>
              <th className="px-6 py-3 text-right">Weight</th>
              <th className="px-6 py-3">Sector</th>
              <th className="px-6 py-3 text-right">Change</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
            {data.top_positions.map((position) => (
              <tr key={position.security_id} className="hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors">
                <td className="px-6 py-4 font-medium text-slate-900 dark:text-white">
                  {position.name}
                </td>
                <td className="px-6 py-4 text-right font-bold text-blue-600 dark:text-blue-400 tabular-nums">
                  {(position.weight * 100).toFixed(2)}%
                </td>
                <td className="px-6 py-4">
                  {position.sector && (
                    <span className="px-2 py-1 bg-blue-100 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 text-xs font-bold rounded uppercase">
                      {position.sector.slice(0, 8)}
                    </span>
                  )}
                </td>
                <td className="px-6 py-4 text-right tabular-nums font-medium">
                  {position.change_pct !== undefined && (
                    <span
                      className={
                        position.change_pct >= 0
                          ? 'text-green-600 dark:text-green-400'
                          : 'text-red-600 dark:text-red-400'
                      }
                    >
                      {position.change_pct >= 0 ? '+' : ''}
                      {position.change_pct.toFixed(2)}%
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

interface SectorWeightsProps {
  data?: Array<{ sector: string; weight: number }>;
  isLoading?: boolean;
}

export const SectorWeights: React.FC<SectorWeightsProps> = ({ data, isLoading }) => {
  if (isLoading || !data) return null;

  const colors = [
    'bg-blue-500',
    'bg-blue-400',
    'bg-blue-300',
    'bg-slate-400',
    'bg-slate-300',
  ];

  return (
    <div className="space-y-5">
      {data.map((sector, idx) => (
        <div key={sector.sector}>
          <div className="flex justify-between text-xs font-bold mb-2">
            <span className="text-slate-700 dark:text-slate-300">{sector.sector}</span>
            <span className="text-slate-900 dark:text-slate-100">
              {(sector.weight * 100).toFixed(1)}%
            </span>
          </div>
          <div className="w-full h-2 bg-slate-100 dark:bg-slate-800 rounded-full overflow-hidden">
            <div
              className={`h-full ${colors[idx % colors.length]} rounded-full`}
              style={{ width: `${sector.weight * 100}%` }}
            />
          </div>
        </div>
      ))}
    </div>
  );
};

interface ScenarioChartProps {
  data?: Array<{
    scenario_id: string;
    name: string;
    pnl: number;
  }>;
  isLoading?: boolean;
  error?: Error | null;
}

export const ScenarioChart: React.FC<ScenarioChartProps> = ({
  data,
  isLoading,
  error,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm h-80 animate-pulse" />
    );
  }

  if (error || !data || data.length === 0) {
    return (
      <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="text-red-600 dark:text-red-400 text-sm">
          {error?.message || 'No scenario data available'}
        </div>
      </div>
    );
  }

  // Find min and max PnL for scaling
  const pnlValues = data.map((s) => s.pnl);
  const minPnL = Math.min(...pnlValues);
  const maxPnL = Math.max(...pnlValues);
  const range = maxPnL - minPnL || 1;

  // Normalize heights
  const normalizedData = data.map((scenario) => ({
    ...scenario,
    normalizedHeight: Math.abs((scenario.pnl - minPnL) / range) * 100,
    isNegative: scenario.pnl < 0,
  }));

  return (
    <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm">
      <div className="flex justify-between items-center mb-6">
        <h3 className="font-bold text-slate-900 dark:text-white">Scenario Results</h3>
        <span className="text-xs text-slate-400 italic">Est. Impact in USD</span>
      </div>

      <div className="h-64 flex items-end justify-between gap-2 px-2 mb-8">
        {normalizedData.map((scenario, idx) => (
          <div
            key={scenario.scenario_id}
            className="flex-1 flex flex-col items-center gap-2 group"
          >
            <div
              className={`w-full rounded-t-lg transition-all group-hover:opacity-100 opacity-80 flex items-${
                scenario.isNegative ? 'start' : 'end'
              } justify-center py-2 ${
                scenario.isNegative
                  ? 'bg-red-500 dark:bg-red-600'
                  : 'bg-green-500 dark:bg-green-600'
              }`}
              style={{ height: `${Math.max(scenario.normalizedHeight, 10)}%` }}
            >
              <span className="text-xs font-bold text-white tabular-nums">
                {scenario.pnl < 0 ? '-' : '+'}${Math.abs(scenario.pnl / 1000000).toFixed(0)}M
              </span>
            </div>
            <span className="text-xs font-bold text-slate-500 dark:text-slate-400 uppercase text-center leading-tight">
              {scenario.name.slice(0, 12)}
            </span>
          </div>
        ))}
      </div>

      {/* Legend */}
      <div className="flex justify-center gap-6">
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-red-500 rounded-sm"></div>
          <span className="text-xs font-medium text-slate-600 dark:text-slate-300">
            Downside Risk
          </span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-green-500 rounded-sm"></div>
          <span className="text-xs font-medium text-slate-600 dark:text-slate-300">
            Upside Opportunity
          </span>
        </div>
      </div>
    </div>
  );
};
