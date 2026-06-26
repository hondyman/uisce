// Dynamic Parameters and Measures Types
export interface DynamicParameter {
  name: string;
  type: 'dimension' | 'measure' | 'filter' | 'time_range' | 'number' | 'string' | 'boolean';
  value?: any;
  defaultValue?: any;
  required: boolean;
  options?: string[];
  description: string;
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    custom?: (value: any) => boolean;
  };
}

export interface DynamicMeasure {
  name: string;
  type: 'count' | 'sum' | 'avg' | 'ratio' | 'custom';
  sql: string;
  parameters?: DynamicParameter[];
  meta?: Record<string, any>;
  description?: string;
}

export interface DynamicQueryRequest {
  baseQuery: {
    tableName: string;
    metrics: string[];
    dimensions: string[];
    filters: Array<{
      field: string;
      op: string;
      values: any[];
    }>;
  };
  parameters: DynamicParameter[];
  dynamicMeasures: DynamicMeasure[];
  timeRange?: {
    start: string;
    end: string;
  };
  context?: Record<string, any>;
}

export interface DynamicQueryResponse {
  columns: Array<{
    name: string;
    type: string;
  }>;
  rows: Record<string, any>[];
  duration: number;
  resolvedQuery?: {
    sql: string;
    parameters: Record<string, any>;
    measures: DynamicMeasure[];
  };
  insights?: {
    trends: Array<{
      metric: string;
      direction: 'up' | 'down' | 'stable';
      confidence: number;
      description: string;
    }>;
    anomalies: Array<{
      metric: string;
      severity: 'low' | 'medium' | 'high' | 'critical';
      value: number;
      expected: number;
      description: string;
    }>;
    predictions: Array<{
      metric: string;
      predictedValue: number;
      confidence: number;
      timeframe: string;
    }>;
  };
}

// PoP Metric Types
export interface PoPMetric {
  id: string;
  name: string;
  displayName: string;
  description: string;
  domain: string;
  category: string;
  metricType: 'sum' | 'avg' | 'count' | 'ratio' | 'percentage';
  baseQuery: string;
  aggregationFunction: string;
  dateColumn: string;
  valueColumn: string;
  granularity: 'day' | 'week' | 'month' | 'quarter' | 'year';
  comparisonPeriods: string[];
  ownerUserId: string;
  stewardGroup: string;
  dataSource: string;
  schemaName: string;
  tableName: string;
  slaFreshnessHours: number;
  slaCompletenessThreshold: number;
  status: 'active' | 'inactive' | 'deprecated';
  goldenPath: boolean;
  version: number;
  createdBy: string;
  tags?: Array<{
    tagName: string;
    tagValue: string;
  }>;
}

export interface PoPComputation {
  id: string;
  metricId: string;
  periodStart: string;
  periodEnd: string;
  granularity: string;
  periodLabel: string;
  currentValue: number;
  previousValue: number;
  delta: number;
  percentChange: number;
  recordCount: number;
  computationStatus: 'success' | 'error' | 'pending';
  computedAt: string;
}

export interface PoPAnomaly {
  id: string;
  metricId: string;
  computationId: string;
  anomalyType: 'z_score' | 'threshold' | 'trend_break';
  severity: 'low' | 'medium' | 'high' | 'critical';
  confidence: number;
  zScore?: number;
  expectedValue: number;
  expectedRangeMin: number;
  expectedRangeMax: number;
  actualValue: number;
  detectionMethod: string;
  detectionParams: Record<string, any>;
  status: 'open' | 'investigating' | 'resolved' | 'false_positive';
  detectedAt: string;
  resolvedAt?: string;
}

export interface PoPStewardReview {
  id: string;
  metricId: string;
  reviewPeriodStart: string;
  reviewPeriodEnd: string;
  reviewerUserId: string;
  reviewType: 'regular' | 'anomaly_investigation' | 'golden_path_review';
  overallRating: 'excellent' | 'good' | 'needs_attention' | 'critical';
  reviewNotes: string;
  status: 'in_progress' | 'completed' | 'overdue';
  dueDate: string;
  completedAt?: string;
}

export interface PoPDashboard {
  id: string;
  name: string;
  description: string;
  ownerUserId: string;
  config: {
    layout: 'grid' | 'dashboard' | 'operational';
    theme: string;
    autoRefresh: boolean;
    refreshInterval: number;
    alertThresholds?: Record<string, number>;
  };
  defaultFilters: Record<string, any>;
  isPublic: boolean;
  allowedGroups: string[];
  widgets?: PoPDashboardWidget[];
  createdAt: string;
  updatedAt: string;
}

export interface PoPDashboardWidget {
  id: string;
  dashboardId: string;
  widgetType: 'kpi_cards' | 'trend_chart' | 'metric_table' | 'anomaly_heatmap' | 'gauge' | 'sparkline';
  title: string;
  position: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  config: Record<string, any>;
  metricIds: string[];
}

// WebSocket Types
export interface WebSocketMessage {
  type: 'metric_update' | 'anomaly_alert' | 'dashboard_refresh' | 'parameter_change' | 'subscribe' | 'unsubscribe' | 'heartbeat';
  payload: any;
  timestamp: string;
  userId?: string;
}

export interface RealTimeSubscription {
  id: string;
  type: 'metric' | 'dashboard' | 'anomaly';
  filters: Record<string, any>;
  callback: (data: any) => void;
}

// Predictive Analytics Types
export interface TrendAnalysis {
  metricId: string;
  period: string;
  trend: 'increasing' | 'decreasing' | 'stable' | 'volatile';
  slope: number;
  rSquared: number;
  confidence: number;
  seasonality: boolean;
  forecast: Array<{
    date: string;
    predictedValue: number;
    upperBound: number;
    lowerBound: number;
  }>;
}

export interface PredictiveModel {
  id: string;
  metricId: string;
  modelType: 'linear' | 'exponential' | 'arima' | 'prophet';
  parameters: Record<string, any>;
  accuracy: number;
  lastTrained: string;
  nextPrediction: string;
}

// Cache Types
export interface QueryCacheEntry {
  key: string;
  query: DynamicQueryRequest;
  result: DynamicQueryResponse;
  timestamp: number;
  ttl: number;
  hits: number;
}

export interface CacheConfig {
  enabled: boolean;
  ttl: number;
  maxSize: number;
  strategy: 'lru' | 'lfu' | 'fifo';
}
