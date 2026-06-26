import React, { useState, useEffect, useMemo } from 'react';
import { PoPDashboard, PoPDashboardWidget, PoPMetric, PoPComputation } from '../../types/dynamic';
import { ParameterControls } from './ParameterControls';
import { DynamicMeasure } from '../../types/dynamic';

interface DynamicDashboardProps {
  dashboard: PoPDashboard;
  metrics: PoPMetric[];
  computations: PoPComputation[];
  onWidgetUpdate?: (widgetId: string, config: any) => void;
  onDashboardRefresh?: () => void;
  className?: string;
}

export const DynamicDashboard: React.FC<DynamicDashboardProps> = ({
  dashboard,
  metrics,
  computations,
  onWidgetUpdate: _onWidgetUpdate,
  onDashboardRefresh,
  className = ''
}) => {
  const [dynamicParameters, setDynamicParameters] = useState<any>({});
  const [_dynamicMeasures, _setDynamicMeasures] = useState<DynamicMeasure[]>([]);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  // Auto-refresh functionality
  const handleRefresh = React.useCallback(async () => {
    setIsRefreshing(true);
    try {
      await onDashboardRefresh?.();
      setLastRefresh(new Date());
    } finally {
      setIsRefreshing(false);
    }
  }, [onDashboardRefresh]);

  useEffect(() => {
    if (!dashboard.config.autoRefresh) return;

    const interval = setInterval(() => {
      handleRefresh();
    }, dashboard.config.refreshInterval * 1000);

    return () => clearInterval(interval);
  }, [dashboard.config.autoRefresh, dashboard.config.refreshInterval, handleRefresh]);

  // (moved above as memoized callback)

  const handleParameterChange = (name: string, value: any) => {
    setDynamicParameters((prev: Record<string, any>) => ({ ...prev, [name]: value }));
  };

  const filteredComputations = useMemo(() => {
    return computations.filter(comp => {
      // Apply dashboard default filters
      if (dashboard.defaultFilters.domain) {
        const metric = metrics.find(m => m.id === comp.metricId);
        if (metric && metric.domain !== dashboard.defaultFilters.domain) {
          return false;
        }
      }
      if (dashboard.defaultFilters.category) {
        const metric = metrics.find(m => m.id === comp.metricId);
        if (metric && metric.category !== dashboard.defaultFilters.category) {
          return false;
        }
      }
      return true;
    });
  }, [computations, metrics, dashboard.defaultFilters]);

  const renderWidget = (widget: PoPDashboardWidget) => {
    const widgetData = filteredComputations.filter(comp =>
      widget.metricIds.includes(comp.metricId)
    );

    switch (widget.widgetType) {
      case 'kpi_cards':
        return <KPICardsWidget data={widgetData} metrics={metrics} config={widget.config} />;

      case 'trend_chart':
        return <TrendChartWidget data={widgetData} metrics={metrics} config={widget.config} />;

      case 'metric_table':
        return <MetricTableWidget data={widgetData} metrics={metrics} config={widget.config} />;

      case 'anomaly_heatmap':
        return <AnomalyHeatmapWidget data={widgetData} metrics={metrics} config={widget.config} />;

      case 'gauge':
        return <GaugeWidget data={widgetData} metrics={metrics} config={widget.config} />;

      case 'sparkline':
        return <SparklineWidget data={widgetData} metrics={metrics} config={widget.config} />;

      default:
        return <div className="p-4 bg-gray-100 rounded">Unknown widget type: {widget.widgetType}</div>;
    }
  };

  return (
    <div className={`space-y-6 ${className}`}>
      {/* Dashboard Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{dashboard.name}</h1>
          <p className="text-gray-600">{dashboard.description}</p>
        </div>
        <div className="flex items-center space-x-4">
          <span className="text-sm text-gray-500">
            Last updated: {lastRefresh.toLocaleTimeString()}
          </span>
          <button
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {isRefreshing ? 'Refreshing...' : 'Refresh'}
          </button>
        </div>
      </div>

      {/* Dynamic Parameters */}
      {dynamicParameters && Object.keys(dynamicParameters).length > 0 && (
        <div className="bg-white p-6 rounded-lg shadow">
          <ParameterControls
            parameters={Object.entries(dynamicParameters).map(([name, param]: [string, any]) => ({
              name,
              type: param.type || 'string',
              value: param.value,
              defaultValue: param.defaultValue,
              required: param.required || false,
              options: param.options,
              description: param.description || name
            }))}
            onParameterChange={handleParameterChange}
          />
        </div>
      )}

      {/* Dashboard Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {dashboard.widgets?.map(widget => (
          <div
            key={widget.id}
            className={`bg-white p-6 rounded-lg shadow ${
              widget.position.width === 2 ? 'md:col-span-2' : ''
            } ${widget.position.height === 2 ? 'row-span-2' : ''}`}
          >
            <h3 className="text-lg font-semibold text-gray-800 mb-4">{widget.title}</h3>
            {renderWidget(widget)}
          </div>
        ))}
      </div>
    </div>
  );
};

// Widget Components
const KPICardsWidget: React.FC<{
  data: PoPComputation[];
  metrics: PoPMetric[];
  config: any;
}> = ({ data, metrics: _metrics, config: _config }) => {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
      {data.slice(0, 4).map(comp => {
        const metric = _metrics.find(m => m.id === comp.metricId);
        return (
          <div key={comp.id} className="text-center">
            <div className="text-2xl font-bold text-gray-900">
              {comp.currentValue.toLocaleString()}
            </div>
            <div className="text-sm text-gray-600">{metric?.displayName}</div>
            <div className={`text-sm ${comp.percentChange >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {comp.percentChange >= 0 ? '+' : ''}{comp.percentChange.toFixed(2)}%
            </div>
          </div>
        );
      })}
    </div>
  );
};

const TrendChartWidget: React.FC<{
  data: PoPComputation[];
  metrics: PoPMetric[];
  config: any;
}> = ({ data, metrics: _metrics, config: _config }) => {
  // This would integrate with a charting library like Chart.js or Recharts
  return (
    <div className="h-64 flex items-center justify-center bg-gray-50 rounded">
      <div className="text-center">
        <div className="text-gray-500 mb-2">📈 Trend Chart</div>
        <div className="text-sm text-gray-400">
          {data.length} data points
        </div>
      </div>
    </div>
  );
};

const MetricTableWidget: React.FC<{
  data: PoPComputation[];
  metrics: PoPMetric[];
  config: any;
}> = ({ data, metrics: _metrics, config: _config }) => {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-sm">
        <thead>
          <tr className="border-b">
            <th className="text-left py-2">Metric</th>
            <th className="text-right py-2">Current</th>
            <th className="text-right py-2">Previous</th>
            <th className="text-right py-2">Change %</th>
          </tr>
        </thead>
        <tbody>
          {data.map(comp => {
            const metric = _metrics.find(m => m.id === comp.metricId);
            return (
              <tr key={comp.id} className="border-b">
                <td className="py-2">{metric?.displayName}</td>
                <td className="text-right py-2">{comp.currentValue.toLocaleString()}</td>
                <td className="text-right py-2">{comp.previousValue.toLocaleString()}</td>
                <td className={`text-right py-2 ${comp.percentChange >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {comp.percentChange >= 0 ? '+' : ''}{comp.percentChange.toFixed(2)}%
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

const AnomalyHeatmapWidget: React.FC<{
  data: PoPComputation[];
  metrics: PoPMetric[];
  config: any;
}> = ({ data: _data, metrics: _metrics, config: _config }) => {
  return (
    <div className="h-64 flex items-center justify-center bg-gray-50 rounded">
      <div className="text-center">
        <div className="text-gray-500 mb-2">🔥 Anomaly Heatmap</div>
        <div className="text-sm text-gray-400">
          Visual anomaly detection
        </div>
      </div>
    </div>
  );
};

const GaugeWidget: React.FC<{
  data: PoPComputation[];
  metrics: PoPMetric[];
  config: any;
}> = ({ data, metrics: _metrics, config: _config }) => {
  const comp = data[0];
  if (!comp) return <div>No data</div>;

  const percentage = Math.min(100, Math.max(0, (comp.currentValue / (comp.currentValue + Math.abs(comp.delta))) * 100));

  return (
    <div className="flex flex-col items-center">
      <div className="relative w-32 h-32">
        <svg className="w-full h-full" viewBox="0 0 36 36">
          <path
            d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
            fill="none"
            stroke="#E5E7EB"
            strokeWidth="2"
          />
          <path
            d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
            fill="none"
            stroke="#3B82F6"
            strokeWidth="2"
            strokeDasharray={`${percentage}, 100`}
          />
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-2xl font-bold">{percentage.toFixed(0)}%</span>
        </div>
      </div>
      <div className="text-center mt-2">
        <div className="text-sm text-gray-600">{comp.currentValue.toLocaleString()}</div>
      </div>
    </div>
  );
};

  const SparklineWidget: React.FC<{
  data: PoPComputation[];
  metrics: PoPMetric[];
  config: any;
}> = ({ data, metrics: _metrics, config: _config }) => {
  const maxValue = Math.max(...data.map(d => d.currentValue));
  // const _minHeight = 16; // Minimum height in pixels (unused)

  return (
    <div className="h-16 flex items-end space-x-1">
  {data.slice(-10).map((comp, _index) => {
        const heightPercent = (comp.currentValue / maxValue) * 100;
        const heightClass = heightPercent < 25 ? 'h-4' :
                           heightPercent < 50 ? 'h-8' :
                           heightPercent < 75 ? 'h-12' : 'h-16';

        return (
          <div
            key={comp.id}
            className={`bg-blue-500 rounded-sm flex-1 ${heightClass}`}
            title={`${comp.currentValue}`}
          />
        );
      })}
    </div>
  );
};
