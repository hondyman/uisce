import { useState, useEffect as _useEffect } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  BarChart as _BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
  PieChart,
  Pie,
  Cell
} from 'recharts';
import {
  TrendingUp,
  Shield,
  Clock,
  AlertTriangle,
  CheckCircle,
  Database,
  Zap,
  Users as _Users,
  Activity as _Activity
} from 'lucide-react';

interface QueryRewriteDashboardProps {
  timeRange?: '1h' | '24h' | '7d' | '30d';
}

interface DashboardMetrics {
  totalRewrites: number;
  successfulRewrites: number;
  failedRewrites: number;
  averageRewriteTime: number;
  complianceViolations: number;
  performanceImprovements: number;
  cacheHitRate: number;
  anomalyDetections: number;
}

interface RewriteLog {
  id: string;
  timestamp: string;
  userId: string;
  tenantId: string;
  originalQuery: string;
  rewrittenQuery: string;
  appliedRules: number;
  performanceGain: number;
  complianceStatus: 'compliant' | 'warning' | 'violation';
  executionTime: number;
}

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8'];

export const QueryRewriteDashboard: React.FC<QueryRewriteDashboardProps> = ({
  timeRange: _timeRange = '24h'
}) => {
  const [metrics, _setMetrics] = useState<DashboardMetrics>({
    totalRewrites: 1250,
    successfulRewrites: 1180,
    failedRewrites: 70,
    averageRewriteTime: 45,
    complianceViolations: 12,
    performanceImprovements: 85,
    cacheHitRate: 78,
    anomalyDetections: 23
  });

  const [rewriteLogs, _setRewriteLogs] = useState<RewriteLog[]>([
    {
      id: '1',
      timestamp: '2025-01-08T10:30:00Z',
      userId: 'analyst123',
      tenantId: 'acme_corp',
      originalQuery: 'SELECT * FROM orders',
      rewrittenQuery: 'SELECT id, amount FROM orders WHERE tenant_id = $1',
      appliedRules: 3,
      performanceGain: 65,
      complianceStatus: 'compliant',
      executionTime: 120
    },
    {
      id: '2',
      timestamp: '2025-01-08T10:25:00Z',
      userId: 'manager456',
      tenantId: 'tech_startup',
      originalQuery: 'SELECT net_margin FROM orders_view',
      rewrittenQuery: 'SELECT certified_net_margin FROM orders_view WHERE tenant_id = $1',
      appliedRules: 2,
      performanceGain: 30,
      complianceStatus: 'warning',
      executionTime: 85
    }
  ]);

  const [performanceData] = useState([
    { time: '00:00', rewrites: 45, avgTime: 42 },
    { time: '04:00', rewrites: 32, avgTime: 38 },
    { time: '08:00', rewrites: 78, avgTime: 45 },
    { time: '12:00', rewrites: 95, avgTime: 52 },
    { time: '16:00', rewrites: 87, avgTime: 48 },
    { time: '20:00', rewrites: 63, avgTime: 41 }
  ]);

  const [ruleUsageData] = useState([
    { name: 'Column Pruning', value: 35, count: 438 },
    { name: 'Tenant Isolation', value: 28, count: 350 },
    { name: 'Performance Opt', value: 20, count: 250 },
    { name: 'Compliance Check', value: 12, count: 150 },
    { name: 'Cache Strategy', value: 5, count: 62 }
  ]);

  const successRate = (metrics.successfulRewrites / metrics.totalRewrites) * 100;
  const complianceRate = ((metrics.totalRewrites - metrics.complianceViolations) / metrics.totalRewrites) * 100;

  return (
    <div className="w-full max-w-7xl mx-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Query Rewrite Dashboard</h1>
          <p className="text-gray-600">Monitor query optimization and compliance performance</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm">
            Export Report
          </Button>
          <Button variant="outline" size="sm">
            Configure Alerts
          </Button>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Total Rewrites</p>
                <p className="text-2xl font-bold">{metrics.totalRewrites.toLocaleString()}</p>
              </div>
              <Database className="w-8 h-8 text-blue-600" />
            </div>
            <div className="mt-2">
              <Progress value={successRate} className="h-2" />
              <p className="text-xs text-gray-600 mt-1">
                {successRate.toFixed(1)}% success rate
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Avg Rewrite Time</p>
                <p className="text-2xl font-bold">{metrics.averageRewriteTime}ms</p>
              </div>
              <Clock className="w-8 h-8 text-green-600" />
            </div>
            <div className="mt-2">
              <Badge variant="secondary" className="text-xs">
                <TrendingUp className="w-3 h-3 mr-1" />
                12% faster than last week
              </Badge>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Compliance Rate</p>
                <p className="text-2xl font-bold">{complianceRate.toFixed(1)}%</p>
              </div>
              <Shield className="w-8 h-8 text-purple-600" />
            </div>
            <div className="mt-2">
              <p className="text-xs text-gray-600">
                {metrics.complianceViolations} violations detected
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Cache Hit Rate</p>
                <p className="text-2xl font-bold">{metrics.cacheHitRate}%</p>
              </div>
              <Zap className="w-8 h-8 text-orange-600" />
            </div>
            <div className="mt-2">
              <Progress value={metrics.cacheHitRate} className="h-2" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Charts and Analytics */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Performance Trends */}
        <Card>
          <CardHeader>
            <CardTitle>Performance Trends</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={performanceData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis yAxisId="left" />
                <YAxis yAxisId="right" orientation="right" />
                <Tooltip />
                <Bar yAxisId="left" dataKey="rewrites" fill="#8884d8" name="Rewrites" />
                <Line yAxisId="right" type="monotone" dataKey="avgTime" stroke="#82ca9d" name="Avg Time (ms)" />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Rule Usage Distribution */}
        <Card>
          <CardHeader>
            <CardTitle>Rule Usage Distribution</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={ruleUsageData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ name, percent = 0 }) => `${name} ${(Number(percent) * 100).toFixed(0)}%`}
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {ruleUsageData.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      {/* Detailed Analytics */}
      <Tabs defaultValue="recent" className="w-full">
        <TabsList>
          <TabsTrigger value="recent">Recent Rewrites</TabsTrigger>
          <TabsTrigger value="performance">Performance Analysis</TabsTrigger>
          <TabsTrigger value="compliance">Compliance Monitoring</TabsTrigger>
          <TabsTrigger value="anomalies">Anomaly Detection</TabsTrigger>
        </TabsList>

        <TabsContent value="recent" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Recent Query Rewrites</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {rewriteLogs.map((log) => (
                  <div key={log.id} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <Badge variant="outline">{log.userId}</Badge>
                        <Badge variant="secondary">{log.tenantId}</Badge>
                        <Badge
                          variant={
                            log.complianceStatus === 'compliant' ? 'default' :
                            log.complianceStatus === 'warning' ? 'secondary' : 'destructive'
                          }
                        >
                          {log.complianceStatus}
                        </Badge>
                      </div>
                      <div className="text-sm text-gray-500">
                        {new Date(log.timestamp).toLocaleString()}
                      </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-3">
                      <div>
                        <p className="text-sm font-medium mb-1">Original Query</p>
                        <code className="text-xs bg-red-50 p-2 rounded block">
                          {log.originalQuery}
                        </code>
                      </div>
                      <div>
                        <p className="text-sm font-medium mb-1">Rewritten Query</p>
                        <code className="text-xs bg-green-50 p-2 rounded block">
                          {log.rewrittenQuery}
                        </code>
                      </div>
                    </div>

                    <div className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-4">
                        <span className="flex items-center gap-1">
                          <CheckCircle className="w-4 h-4 text-green-600" />
                          {log.appliedRules} rules applied
                        </span>
                        <span className="flex items-center gap-1">
                          <TrendingUp className="w-4 h-4 text-blue-600" />
                          {log.performanceGain}% improvement
                        </span>
                        <span className="flex items-center gap-1">
                          <Clock className="w-4 h-4 text-gray-600" />
                          {log.executionTime}ms
                        </span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="performance" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Query Performance</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span>Fast Queries (&lt; 50ms)</span>
                      <span>68%</span>
                    </div>
                    <Progress value={68} />
                  </div>
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span>Medium Queries (50-200ms)</span>
                      <span>25%</span>
                    </div>
                    <Progress value={25} />
                  </div>
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span>Slow Queries (&gt; 200ms)</span>
                      <span>7%</span>
                    </div>
                    <Progress value={7} />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Optimization Impact</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-center">
                  <div className="text-3xl font-bold text-green-600 mb-2">
                    {metrics.performanceImprovements}%
                  </div>
                  <p className="text-sm text-gray-600">Average performance improvement</p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Cache Effectiveness</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-center">
                  <div className="text-3xl font-bold text-blue-600 mb-2">
                    {metrics.cacheHitRate}%
                  </div>
                  <p className="text-sm text-gray-600">Cache hit rate</p>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="compliance" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Card>
              <CardHeader>
                <CardTitle>Compliance Overview</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <span>Compliant Queries</span>
                    <Badge variant="default">
                      {((metrics.totalRewrites - metrics.complianceViolations) / metrics.totalRewrites * 100).toFixed(1)}%
                    </Badge>
                  </div>
                  <div className="flex items-center justify-between">
                    <span>Violations Detected</span>
                    <Badge variant="destructive">
                      {metrics.complianceViolations}
                    </Badge>
                  </div>
                  <div className="flex items-center justify-between">
                    <span>Auto-corrected</span>
                    <Badge variant="secondary">
                      {Math.round(metrics.complianceViolations * 0.8)}
                    </Badge>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Security Events</CardTitle>
              </CardHeader>
              <CardContent>
                <Alert>
                  <Shield className="h-4 w-4" />
                  <AlertDescription>
                    All queries automatically enforce tenant isolation and access controls.
                    No manual intervention required.
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="anomalies" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="w-5 h-5" />
                Anomaly Detection
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                  <div className="text-center">
                    <div className="text-2xl font-bold text-orange-600">
                      {metrics.anomalyDetections}
                    </div>
                    <p className="text-sm text-gray-600">Anomalies Detected</p>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-yellow-600">
                      {Math.round(metrics.anomalyDetections * 0.3)}
                    </div>
                    <p className="text-sm text-gray-600">False Positives</p>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-red-600">
                      {Math.round(metrics.anomalyDetections * 0.1)}
                    </div>
                    <p className="text-sm text-gray-600">Critical Issues</p>
                  </div>
                </div>

                <Alert>
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>
                    Anomaly detection is active and monitoring query patterns for unusual behavior.
                    All detected anomalies are logged and can trigger automated responses.
                  </AlertDescription>
                </Alert>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
};
