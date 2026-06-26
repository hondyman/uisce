import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { AlertTriangle, TrendingUp, TrendingDown, Activity, Clock } from 'lucide-react';
import { PoPMetricExplorer } from './PoPMetricExplorer';
import { AnomalyDashboard } from './AnomalyDashboard';
import { StewardReviewPanel } from './StewardReviewPanel';
import { PoPChart } from './PoPChart';
import type { PoPMetricWithContract } from './PoPMetricExplorer';

interface PoPDashboardProps {
  domain?: string;
}

interface DashboardData {
  metrics: PoPMetricWithContract[];
  anomaly_summary: AnomalySummary[];
  review_status: ReviewStatus[];
  last_updated: string;
}

interface PoPMetricWithLatest {
  id: string;
  name: string;
  display_name: string;
  domain: string;
  category: string;
  current_value?: number;
  previous_value?: number;
  delta?: number;
  percent_change?: number;
  period_start: string;
  period_end: string;
  last_computed_at: string;
  has_anomalies: boolean;
  anomaly_count: number;
}

interface AnomalySummary {
  domain: string;
  category: string;
  severity: string;
  anomaly_type: string;
  anomaly_count: number;
  latest_detection: string;
  affected_metrics: string[];
}

interface ReviewStatus {
  metric_id: string;
  metric_name: string;
  review_status: string;
  last_review_date?: string;
  due_date?: string;
  overdue_count: number;
}

export const PoPDashboard: React.FC<PoPDashboardProps> = ({ domain }) => {
  const [dashboardData, setDashboardData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState('overview');

  const fetchDashboardData = React.useCallback(async () => {
    try {
      setLoading(true);
      const params = new URLSearchParams();
      if (domain) params.append('domain', domain);

      const response = await fetch(`/api/pop/dashboard?${params}`);
      if (!response.ok) throw new Error('Failed to fetch dashboard data');

      const data = await response.json();
      setDashboardData(data.data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [domain]);

  useEffect(() => {
    fetchDashboardData();
  }, [domain, fetchDashboardData]);

  const handleRefresh = () => {
    fetchDashboardData();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-8">
        <AlertTriangle className="mx-auto h-12 w-12 text-red-500 mb-4" />
        <h3 className="text-lg font-medium text-gray-900 mb-2">Error Loading Dashboard</h3>
        <p className="text-gray-500 mb-4">{error}</p>
        <Button onClick={handleRefresh}>Try Again</Button>
      </div>
    );
  }

  if (!dashboardData) return null;

  const totalMetrics = dashboardData.metrics.length;
  const metricsWithAnomalies = dashboardData.metrics.filter(m => m.has_anomalies).length;
  const totalAnomalies = dashboardData.anomaly_summary.reduce((sum, a) => sum + a.anomaly_count, 0);
  const overdueReviews = dashboardData.review_status.filter(r => r.overdue_count > 0).length;

  const explorerMetrics: PoPMetricWithContract[] = dashboardData.metrics.map(m => ({
    id: m.id,
    display_name: m.name,
    node_id: m.id,
    node_type: 'metric',
    name: m.name,
    description: '', // Not available in PoPMetricWithLatest
    version: '1.0',
    base_metric: '',
    period: 'day',
    comparison: 'day_ago',
    formula: '',
    dimensions: [],
    time_dimension: '',
    granularity: 'day',
    tags: { domain: m.tags?.domain || '', category: m.tags?.category || '' },
    status: 'active',
    last_updated: m.last_computed_at,
    owner: '',
    steward_group: '',
    lineage: { upstream_sources: [], downstream_consumers: [] },
    data_quality_contract: { null_threshold_pct: 0, latency_minutes: 0, completeness_pct: 0 },
    sla: { refresh_frequency: '', max_delay_minutes: 0 },
    anomaly_detection: { method: 'zscore', threshold: 2.5, enabled: m.has_anomalies },
    golden_path: false,
    schema_hash: '',
    current_value: m.current_value,
    previous_value: m.previous_value,
    delta: m.delta,
    percent_change: m.percent_change,
    period_start: m.period_start,
    period_end: m.period_end,
    last_computed_at: m.last_computed_at,
    has_anomalies: m.has_anomalies,
    anomaly_count: m.anomaly_count,
    computation_status: 'completed',
  }));

  const anomalyMetrics: PoPMetricWithLatest[] = dashboardData.metrics.map(m => ({
    id: m.id,
    name: m.name,
    display_name: m.name,
    domain: m.tags?.domain || '',
    category: m.tags?.category || '',
    current_value: m.current_value,
    previous_value: m.previous_value,
    delta: m.delta,
    percent_change: m.percent_change,
    period_start: m.period_start,
    period_end: m.period_end,
    last_computed_at: m.last_computed_at,
    has_anomalies: m.has_anomalies,
    anomaly_count: m.anomaly_count,
  }));

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">PoP Analysis Cockpit</h1>
          <p className="text-gray-600 mt-1">
            Period-over-Period analysis, anomaly detection, and steward reviews
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Badge variant="outline" className="text-xs">
            Last updated: {new Date(dashboardData.last_updated).toLocaleString()}
          </Badge>
          <Button onClick={handleRefresh} size="sm">
            Refresh
          </Button>
        </div>
      </div>

      {/* Key Metrics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Metrics</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalMetrics}</div>
            <p className="text-xs text-muted-foreground">
              Active PoP metrics
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Metrics with Anomalies</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{metricsWithAnomalies}</div>
            <p className="text-xs text-muted-foreground">
              Require attention
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Anomalies</CardTitle>
            <TrendingDown className="h-4 w-4 text-orange-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-orange-600">{totalAnomalies}</div>
            <p className="text-xs text-muted-foreground">
              Detected issues
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Overdue Reviews</CardTitle>
            <Clock className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-yellow-600">{overdueReviews}</div>
            <p className="text-xs text-muted-foreground">
              Need steward review
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="metrics">Metrics</TabsTrigger>
          <TabsTrigger value="anomalies">Anomalies</TabsTrigger>
          <TabsTrigger value="reviews">Reviews</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Recent Metric Performance */}
            <Card>
              <CardHeader>
                <CardTitle>Recent Metric Performance</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {dashboardData.metrics.slice(0, 5).map((metric) => (
                    <div key={metric.id} className="flex items-center justify-between p-3 border rounded-lg">
                      <div>
                        <p className="font-medium">{metric.display_name}</p>
                        <p className="text-sm text-gray-500">{metric.tags?.domain || 'N/A'} • {metric.tags?.category || 'N/A'}</p>
                      </div>
                      <div className="text-right">
                        <div className="flex items-center space-x-2">
                          {metric.percent_change !== undefined && (
                            <Badge
                              variant={metric.percent_change >= 0 ? "default" : "destructive"}
                              className="text-xs"
                            >
                              {metric.percent_change >= 0 ? (
                                <TrendingUp className="w-3 h-3 mr-1" />
                              ) : (
                                <TrendingDown className="w-3 h-3 mr-1" />
                              )}
                              {Math.abs(metric.percent_change).toFixed(1)}%
                            </Badge>
                          )}
                          {metric.has_anomalies && (
                            <AlertTriangle className="w-4 h-4 text-red-500" />
                          )}
                        </div>
                        <p className="text-sm text-gray-500 mt-1">
                          {metric.current_value?.toLocaleString() || 'N/A'}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* Anomaly Summary */}
            <Card>
              <CardHeader>
                <CardTitle>Anomaly Summary</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {dashboardData.anomaly_summary.slice(0, 5).map((summary, index) => (
                    <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                      <div>
                        <p className="font-medium">{summary.domain} • {summary.category}</p>
                        <p className="text-sm text-gray-500">{summary.anomaly_type}</p>
                      </div>
                      <div className="text-right">
                        <Badge
                          variant={
                            summary.severity === 'high' ? 'destructive' :
                            summary.severity === 'medium' ? 'default' : 'secondary'
                          }
                        >
                          {summary.severity}
                        </Badge>
                        <p className="text-sm text-gray-500 mt-1">
                          {summary.anomaly_count} anomalies
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* PoP Chart */}
          <Card>
            <CardHeader>
              <CardTitle>Period-over-Period Trends</CardTitle>
            </CardHeader>
            <CardContent>
              <PoPChart metrics={dashboardData.metrics} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="metrics">
          <PoPMetricExplorer metrics={explorerMetrics} onRefresh={handleRefresh} />
        </TabsContent>

        <TabsContent value="anomalies">
          <AnomalyDashboard
            anomalySummary={dashboardData.anomaly_summary}
            metrics={anomalyMetrics}
            onRefresh={handleRefresh}
          />
        </TabsContent>

        <TabsContent value="reviews">
          <StewardReviewPanel
            reviewStatus={dashboardData.review_status}
            onRefresh={handleRefresh}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
};
