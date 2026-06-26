import { useState, useEffect } from 'react';
import { EnhancedDashboard } from './components/pop/EnhancedDashboard';
import { WebSocketService } from './services/WebSocketService';
import { QueryCacheService, ResultMemoizationService } from './services/CacheService';
import { PoPDashboard /* , PoPDashboardWidget */ } from '../types/dynamic';

// Demo data for the enhanced dashboard
const demoDashboard: PoPDashboard = {
  id: 'demo-dashboard',
  name: 'PoP Analytics Dashboard',
  description: 'Real-time Period-over-Period analytics with predictive insights',
  ownerUserId: 'demo-user',
  config: {
    layout: 'grid',
    theme: 'light',
    autoRefresh: true,
    refreshInterval: 30, // 30 seconds
    alertThresholds: {
      anomaly_score: 2.5,
      trend_confidence: 0.8
    }
  },
  defaultFilters: {
    dateRange: '30d',
    region: 'all',
    category: 'all'
  },
  isPublic: false,
  allowedGroups: ['analysts', 'managers'],
  widgets: [
    {
      id: 'kpi-widget-1',
      dashboardId: 'demo-dashboard',
      widgetType: 'kpi_cards',
      title: 'Key Performance Indicators',
      position: { x: 0, y: 0, width: 2, height: 1 },
      config: {
        metrics: ['revenue', 'users', 'conversion_rate'],
        showChange: true,
        showTrend: true
      },
      metricIds: ['revenue_metric', 'users_metric', 'conversion_metric']
    },
    {
      id: 'trend-widget-1',
      dashboardId: 'demo-dashboard',
      widgetType: 'trend_chart',
      title: 'Revenue Trend Analysis',
      position: { x: 2, y: 0, width: 2, height: 2 },
      config: {
        metric: 'revenue',
        period: '30d',
        showForecast: true,
        forecastDays: 7
      },
      metricIds: ['revenue_metric']
    },
    {
      id: 'anomaly-widget-1',
      dashboardId: 'demo-dashboard',
      widgetType: 'anomaly_heatmap',
      title: 'Anomaly Detection',
      position: { x: 0, y: 1, width: 2, height: 1 },
      config: {
        sensitivity: 'medium',
        lookbackPeriod: '90d'
      },
      metricIds: ['revenue_metric', 'users_metric']
    },
    {
      id: 'table-widget-1',
      dashboardId: 'demo-dashboard',
      widgetType: 'metric_table',
      title: 'Detailed Metrics Table',
      position: { x: 0, y: 2, width: 4, height: 1 },
      config: {
        columns: ['date', 'revenue', 'users', 'conversion_rate', 'change_pct'],
        sortable: true,
        filterable: true
      },
      metricIds: ['revenue_metric', 'users_metric', 'conversion_metric']
    }
  ],
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString()
};

export const DashboardDemo: React.FC = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [cacheStats, setCacheStats] = useState<any>({});
  const [memoStats, setMemoStats] = useState<any>({});

  // Initialize services
  useEffect(() => {
    const wsService = WebSocketService.getInstance();
    const cacheService = QueryCacheService.getInstance();
    const memoService = ResultMemoizationService.getInstance();

    // Set up WebSocket connection status
    const connectionCheck = setInterval(() => {
      setIsConnected(wsService.isConnected());
    }, 1000);

    // Update cache and memo stats
    const statsUpdate = setInterval(() => {
      setCacheStats(cacheService.getStats());
      setMemoStats(memoService.getStats());
    }, 5000);

    return () => {
      clearInterval(connectionCheck);
      clearInterval(statsUpdate);
    };
  }, []);

  const handleParameterChange = (_parameters: Record<string, any>) => {
    // Demo parameter change handler (noop for demo)
  };

  const handleWidgetUpdate = (_widgetId: string, _data: any) => {
    // Demo widget update handler (noop for demo)
  };

  return (
    <div className="dashboard-demo min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-4">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Enhanced PoP Dashboard</h1>
              <p className="text-sm text-gray-600">Real-time analytics with predictive insights</p>
            </div>
            <div className="flex items-center space-x-4">
              <div className={`flex items-center space-x-2 ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
                <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-600' : 'bg-red-600'}`} />
                <span className="text-sm">{isConnected ? 'Real-time Connected' : 'Disconnected'}</span>
              </div>
              <div className="text-sm text-gray-600">
                Cache: {cacheStats.totalEntries || 0} entries
              </div>
              <div className="text-sm text-gray-600">
                Memo: {memoStats.totalEntries || 0} results
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <EnhancedDashboard
          dashboard={demoDashboard}
          onParameterChange={handleParameterChange}
          onWidgetUpdate={handleWidgetUpdate}
          enableRealTime={true}
          enablePredictive={true}
          enableCaching={true}
        />
      </div>

      {/* Demo Instructions */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pb-8">
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-blue-900 mb-4">Demo Features</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 text-sm">
            <div>
              <h3 className="font-medium text-blue-800">Real-time Updates</h3>
              <p className="text-blue-700">WebSocket connections for live data streaming</p>
            </div>
            <div>
              <h3 className="font-medium text-blue-800">Predictive Analytics</h3>
              <p className="text-blue-700">Trend analysis and forecasting with confidence intervals</p>
            </div>
            <div>
              <h3 className="font-medium text-blue-800">Query Caching</h3>
              <p className="text-blue-700">LRU cache with configurable TTL and size limits</p>
            </div>
            <div>
              <h3 className="font-medium text-blue-800">Result Memoization</h3>
              <p className="text-blue-700">Computation caching with dependency tracking</p>
            </div>
          </div>

          <div className="mt-6">
            <h3 className="font-medium text-blue-800 mb-2">How to Use:</h3>
            <ul className="list-disc list-inside text-blue-700 space-y-1">
              <li>Adjust parameters in the controls section to filter data</li>
              <li>Click refresh buttons on individual widgets for manual updates</li>
              <li>Monitor real-time connection status in the header</li>
              <li>View cache performance statistics in the dashboard footer</li>
              <li>Widgets automatically refresh based on dashboard configuration</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DashboardDemo;
