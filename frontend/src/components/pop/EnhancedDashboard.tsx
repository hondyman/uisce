import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { devError } from '../../utils/devLogger';
import { PoPDashboard, PoPDashboardWidget, PoPComputation, TrendAnalysis } from '../../types/dynamic';
import { ParameterControls } from './ParameterControls';
import { WidgetRenderer } from './WidgetRenderer';
import { getWebSocketService } from '../../services/WebSocketService';
import { PredictiveAnalyticsService } from '../../services/PredictiveAnalyticsService';
import { QueryCacheService, ResultMemoizationService } from '../../services/CacheService';

interface EnhancedDashboardProps {
  dashboard: PoPDashboard;
  onParameterChange?: (parameters: Record<string, any>) => void;
  onWidgetUpdate?: (widgetId: string, data: any) => void;
  enableRealTime?: boolean;
  enablePredictive?: boolean;
  enableCaching?: boolean;
}

export const EnhancedDashboard: React.FC<EnhancedDashboardProps> = ({
  dashboard,
  onParameterChange,
  onWidgetUpdate,
  enableRealTime = true,
  enablePredictive = true,
  enableCaching = true
}) => {
  const [parameters, setParameters] = useState<Record<string, any>>(dashboard.defaultFilters || {});
  const [widgetData, setWidgetData] = useState<Record<string, any>>({});
  const [trendAnalysis, setTrendAnalysis] = useState<Record<string, TrendAnalysis>>({});
  const [isLoading, setIsLoading] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  // Initialize services
  const wsService = useMemo(() => getWebSocketService(), []);
  const predictiveService = useMemo(() => PredictiveAnalyticsService.getInstance(), []);
  const cacheService = useMemo(() => QueryCacheService.getInstance(), []);
  const memoService = useMemo(() => ResultMemoizationService.getInstance(), []);

  // Memoized parameter change handler
  const handleParameterChange = useCallback((newParameters: Record<string, any>) => {
    setParameters(newParameters);
    onParameterChange?.(newParameters);

    // Invalidate cache for parameter-dependent queries
    if (enableCaching) {
      Object.keys(newParameters).forEach(param => {
        cacheService.invalidate(`*${param}*`);
        memoService.invalidateByDependency(param);
      });
    }
  }, [onParameterChange, enableCaching, cacheService, memoService]);

  // Fetch widget data with caching and memoization
  const fetchWidgetData = useCallback(async (widget: PoPDashboardWidget) => {
    const cacheKey = `widget_${widget.id}_${JSON.stringify(parameters)}`;

    // Try cache first
    if (enableCaching) {
      const cachedData = cacheService.get(cacheKey);
      if (cachedData) {
        setWidgetData(prev => ({ ...prev, [widget.id]: cachedData }));
        return cachedData;
      }
    }

    // Memoized data fetching
    const fetchData = memoService.memoize(
      cacheKey,
      async () => {
        // Simulate API call - replace with actual backend call
        const response = await fetch(`/api/pop/dynamic/query`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            metricIds: widget.metricIds,
            parameters,
            widgetConfig: widget.config
          })
        });

        if (!response.ok) {
          throw new Error(`Failed to fetch data for widget ${widget.id}`);
        }

        return response.json();
      },
      5 * 60 * 1000, // 5 minutes TTL
      Object.keys(parameters)
    );

    try {
      const data = await fetchData;
      setWidgetData(prev => ({ ...prev, [widget.id]: data }));

      // Cache the result
      if (enableCaching) {
        cacheService.set(cacheKey, data);
      }

      onWidgetUpdate?.(widget.id, data);
      return data;
    } catch (error) {
      devError(`Error fetching data for widget ${widget.id}:`, error);
      return null;
    }
  }, [parameters, enableCaching, cacheService, memoService, onWidgetUpdate]);

  // Perform trend analysis for widgets
  const performTrendAnalysis = useCallback(async (widget: PoPDashboardWidget) => {
    if (!enablePredictive || !widget.metricIds.length) return;

    try {
      const computations: PoPComputation[] = widgetData[widget.id]?.computations || [];

      if (computations.length < 3) return;

      const analysis = await Promise.all(
        widget.metricIds.map(async (metricId) => {
          const metricComputations = computations.filter(comp => comp.metricId === metricId);
          if (metricComputations.length < 3) return null;

          return predictiveService.analyzeTrend(metricId, metricComputations);
        })
      );

      const validAnalysis = analysis.filter(Boolean) as TrendAnalysis[];
      if (validAnalysis.length > 0) {
        setTrendAnalysis(prev => ({
          ...prev,
          [widget.id]: validAnalysis[0] // Use first analysis for simplicity
        }));
      }
    } catch (error) {
      devError(`Error performing trend analysis for widget ${widget.id}:`, error);
    }
  }, [enablePredictive, widgetData, predictiveService]);

  // Set up real-time subscriptions
  useEffect(() => {
    if (!enableRealTime) return;

    const subscriptions = dashboard.widgets?.map(widget => ({
      id: `widget_${widget.id}`,
      type: 'metric' as const,
      filters: { metricIds: widget.metricIds, ...parameters },
      callback: (data: any) => {
        setWidgetData(prev => ({ ...prev, [widget.id]: data }));
        setLastUpdate(new Date());
        onWidgetUpdate?.(widget.id, data);
      }
    })) || [];

    // Subscribe to real-time updates
    subscriptions.forEach(sub => wsService.subscribe(sub));

    // Cleanup subscriptions
    return () => {
      subscriptions.forEach(sub => wsService.unsubscribe(sub.id));
    };
  }, [enableRealTime, dashboard.widgets, parameters, wsService, onWidgetUpdate]);

  // Load initial data for all widgets
  useEffect(() => {
    const loadAllWidgetData = async () => {
      if (!dashboard.widgets?.length) return;

      setIsLoading(true);
      try {
        await Promise.all(
          dashboard.widgets.map(async (widget) => {
            await fetchWidgetData(widget);
            await performTrendAnalysis(widget);
          })
        );
      } catch (error) {
        devError('Error loading widget data:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadAllWidgetData();
  }, [dashboard.widgets, fetchWidgetData, performTrendAnalysis]);

  // Auto-refresh based on dashboard config
  useEffect(() => {
    if (!dashboard.config.autoRefresh) return;

    const interval = setInterval(() => {
      dashboard.widgets?.forEach(widget => {
        fetchWidgetData(widget);
        performTrendAnalysis(widget);
      });
      setLastUpdate(new Date());
    }, dashboard.config.refreshInterval * 1000);

    return () => clearInterval(interval);
  }, [dashboard.config.autoRefresh, dashboard.config.refreshInterval, dashboard.widgets, fetchWidgetData, performTrendAnalysis]);

  return (
    <div className="enhanced-dashboard space-y-6">
      {/* Header with last update time */}
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">{dashboard.name}</h1>
        <div className="text-sm text-gray-500">
          Last updated: {lastUpdate.toLocaleTimeString()}
          {isLoading && <span className="ml-2 text-blue-500">Updating...</span>}
        </div>
      </div>

      {/* Parameter Controls */}
      <div className="bg-white p-4 rounded-lg shadow">
        <ParameterControls
          parameters={Object.entries(parameters).map(([name, value]) => ({
            name,
            type: 'string' as const,
            value,
            required: false,
            description: name
          }))}
          onParameterChange={(name, value) => handleParameterChange({ ...parameters, [name]: value })}
        />
      </div>

      {/* Dashboard Widgets */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {dashboard.widgets?.map((widget) => (
          <WidgetRenderer
            key={widget.id}
            widget={widget}
            data={widgetData[widget.id]}
            trendAnalysis={trendAnalysis[widget.id]}
            onRefresh={() => fetchWidgetData(widget)}
            isLoading={isLoading}
          />
        ))}
      </div>

      {/* Cache and Performance Stats */}
      {enableCaching && (
        <div className="bg-gray-50 p-4 rounded-lg">
          <h3 className="text-lg font-semibold mb-2">Performance Stats</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="font-medium">Cache Entries:</span>
              <span className="ml-2">{cacheService.getStats().totalEntries}</span>
            </div>
            <div>
              <span className="font-medium">Cache Hit Rate:</span>
              <span className="ml-2">{(cacheService.getStats().hitRate * 100).toFixed(1)}%</span>
            </div>
            <div>
              <span className="font-medium">Memoized Results:</span>
              <span className="ml-2">{memoService.getStats().totalEntries}</span>
            </div>
            <div>
              <span className="font-medium">Real-time:</span>
              <span className="ml-2 text-green-600">{enableRealTime ? 'Enabled' : 'Disabled'}</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default EnhancedDashboard;
