import { FC } from 'react';
import { PoPDashboardWidget, TrendAnalysis } from '../../types/dynamic';

interface WidgetRendererProps {
  widget: PoPDashboardWidget;
  data?: any;
  trendAnalysis?: TrendAnalysis;
  onRefresh?: () => void;
  isLoading?: boolean;
}

export const WidgetRenderer: FC<WidgetRendererProps> = ({
  widget,
  data,
  trendAnalysis,
  onRefresh,
  isLoading = false
}) => {
  const renderWidgetContent = () => {
    switch (widget.widgetType) {
      case 'kpi_cards':
        return renderKPICards();
      case 'trend_chart':
        return renderTrendChart();
      case 'metric_table':
        return renderMetricTable();
      case 'anomaly_heatmap':
        return renderAnomalyHeatmap();
      case 'gauge':
        return renderGauge();
      case 'sparkline':
        return renderSparkline();
      default:
        return <div>Unsupported widget type: {widget.widgetType}</div>;
    }
  };

  const renderKPICards = () => {
    if (!data?.kpis) return <div>No KPI data available</div>;

    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {data.kpis.map((kpi: any, index: number) => (
          <div key={index} className="bg-white p-4 rounded-lg border">
            <div className="text-sm text-gray-600">{kpi.label}</div>
            <div className="text-2xl font-bold">{kpi.value}</div>
            <div className={`text-sm ${kpi.change >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {kpi.change >= 0 ? '+' : ''}{kpi.change}%
            </div>
          </div>
        ))}
      </div>
    );
  };

  const renderTrendChart = () => {
    if (!data?.trend) return <div>No trend data available</div>;

    return (
      <div className="h-64">
        <div className="text-sm text-gray-600 mb-2">Trend Analysis</div>
        {trendAnalysis && (
          <div className="mb-2 text-xs">
            <span className={`px-2 py-1 rounded ${
              trendAnalysis.trend === 'increasing' ? 'bg-green-100 text-green-800' :
              trendAnalysis.trend === 'decreasing' ? 'bg-red-100 text-red-800' :
              'bg-gray-100 text-gray-800'
            }`}>
              {trendAnalysis.trend} ({(trendAnalysis.confidence * 100).toFixed(1)}% confidence)
            </span>
          </div>
        )}
        <div className="bg-gray-50 h-full rounded flex items-center justify-center">
          Chart placeholder - {data.trend.points?.length || 0} data points
        </div>
      </div>
    );
  };

  const renderMetricTable = () => {
    if (!data?.table) return <div>No table data available</div>;

    return (
      <div className="overflow-x-auto">
        <table className="min-w-full text-sm">
          <thead>
            <tr className="border-b">
              {data.table.columns?.map((col: string, index: number) => (
                <th key={index} className="text-left py-2 px-4">{col}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.table.rows?.map((row: any[], rowIndex: number) => (
              <tr key={rowIndex} className="border-b">
                {row.map((cell: any, cellIndex: number) => (
                  <td key={cellIndex} className="py-2 px-4">{cell}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  };

  const renderAnomalyHeatmap = () => {
    if (!data?.anomalies) return <div>No anomaly data available</div>;

    return (
      <div className="h-64 bg-gray-50 rounded flex items-center justify-center">
        Anomaly Heatmap - {data.anomalies.length} anomalies detected
      </div>
    );
  };

  const renderGauge = () => {
    if (!data?.gauge) return <div>No gauge data available</div>;

    return (
      <div className="flex items-center justify-center h-32">
        <div className="text-center">
          <div className="text-3xl font-bold">{data.gauge.value}</div>
          <div className="text-sm text-gray-600">{data.gauge.label}</div>
          <div className="w-24 h-2 bg-gray-200 rounded mt-2">
            <div
              className="h-full bg-blue-500 rounded transition-all duration-300"
            />
          </div>
        </div>
      </div>
    );
  };

  const renderSparkline = () => {
    if (!data?.sparkline) return <div>No sparkline data available</div>;

    return (
      <div className="h-16 bg-gray-50 rounded flex items-center justify-center">
        Sparkline - {data.sparkline.points?.length || 0} points
      </div>
    );
  };

  return (
    <div className="widget-container bg-white rounded-lg shadow p-4">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold">{widget.title}</h3>
        <div className="flex items-center space-x-2">
          {isLoading && <div className="text-sm text-gray-500">Loading...</div>}
          {onRefresh && (
            <button
              onClick={onRefresh}
              className="text-sm text-blue-600 hover:text-blue-800"
              disabled={isLoading}
            >
              Refresh
            </button>
          )}
        </div>
      </div>

      <div className="widget-content">
        {renderWidgetContent()}
      </div>

      {trendAnalysis && (
        <div className="mt-4 text-xs text-gray-500">
          Trend: {trendAnalysis.trend} | Confidence: {(trendAnalysis.confidence * 100).toFixed(1)}%
        </div>
      )}
    </div>
  );
};

export default WidgetRenderer;
