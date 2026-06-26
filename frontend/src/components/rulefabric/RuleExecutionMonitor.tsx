/**
 * RuleExecutionMonitor.tsx
 * 
 * Real-time rule execution monitoring dashboard providing:
 * - Live execution metrics and KPIs
 * - Rule performance heatmaps
 * - Execution timeline with drill-down
 * - Alert threshold management
 * - Real-time WebSocket updates
 */

import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Activity,
  TrendingUp,
  TrendingDown,
  Clock,
  CheckCircle,
  XCircle,
  Pause,
  RefreshCw,
  Download,
  Bell,
  BellOff,
  ChevronRight,
  ChevronDown,
  Zap,
  BarChart2,
  PieChart,
  Target,
  Eye,
  EyeOff
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface ExecutionMetric {
  ruleId: string;
  ruleName: string;
  category: string;
  executionsTotal: number;
  executionsSuccess: number;
  executionsFailed: number;
  avgLatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  lastExecutedAt: Date;
  matchRate: number;
  actionsTriggered: number;
  status: 'healthy' | 'warning' | 'critical' | 'inactive';
}

interface ExecutionEvent {
  id: string;
  ruleId: string;
  ruleName: string;
  timestamp: Date;
  status: 'success' | 'failure' | 'timeout' | 'skipped';
  latencyMs: number;
  recordsProcessed: number;
  recordsMatched: number;
  actionsTriggered: string[];
  errorMessage?: string;
  metadata?: Record<string, unknown>;
}

interface AlertConfig {
  id: string;
  name: string;
  condition: 'latency_high' | 'failure_rate' | 'match_rate_anomaly' | 'execution_spike' | 'rule_inactive';
  threshold: number;
  severity: 'info' | 'warning' | 'critical';
  enabled: boolean;
  notificationChannels: string[];
}

interface TimeRange {
  label: string;
  value: string;
  minutes: number;
}

interface RuleExecutionMonitorProps {
  tenantId: string;
  datasourceId: string;
  initialRuleFilter?: string[];
  onRuleClick?: (ruleId: string) => void;
}

// ============================================================================
// Constants
// ============================================================================

const TIME_RANGES: TimeRange[] = [
  { label: 'Last 5 min', value: '5m', minutes: 5 },
  { label: 'Last 15 min', value: '15m', minutes: 15 },
  { label: 'Last 1 hour', value: '1h', minutes: 60 },
  { label: 'Last 4 hours', value: '4h', minutes: 240 },
  { label: 'Last 24 hours', value: '24h', minutes: 1440 },
  { label: 'Last 7 days', value: '7d', minutes: 10080 }
];

const STATUS_COLORS = {
  healthy: 'bg-green-100 text-green-800 border-green-200',
  warning: 'bg-yellow-100 text-yellow-800 border-yellow-200',
  critical: 'bg-red-100 text-red-800 border-red-200',
  inactive: 'bg-gray-100 text-gray-800 border-gray-200'
};

// ============================================================================
// Simulated Data Generation
// ============================================================================

const generateMockMetrics = (): ExecutionMetric[] => {
  const categories = ['data_quality', 'compliance', 'wash_trade', 'mdm', 'values'];
  const ruleNames = [
    'Null Check - Account ID',
    'AML Transaction Threshold',
    'Same-Day Wash Detection',
    'Customer Dedup Rule',
    'ESG Score Validation',
    'KYC Completeness',
    'Trade Volume Anomaly',
    'Portfolio Drift Alert',
    'Regulatory Reporting Check',
    'Data Freshness Monitor'
  ];
  
  return ruleNames.map((name, i) => ({
    ruleId: `rule-${i + 1}`,
    ruleName: name,
    category: categories[i % categories.length],
    executionsTotal: Math.floor(Math.random() * 50000) + 1000,
    executionsSuccess: Math.floor(Math.random() * 45000) + 900,
    executionsFailed: Math.floor(Math.random() * 500),
    avgLatencyMs: Math.random() * 100 + 5,
    p95LatencyMs: Math.random() * 200 + 50,
    p99LatencyMs: Math.random() * 500 + 100,
    lastExecutedAt: new Date(Date.now() - Math.random() * 3600000),
    matchRate: Math.random() * 30 + 5,
    actionsTriggered: Math.floor(Math.random() * 10000),
    status: ['healthy', 'healthy', 'healthy', 'warning', 'critical', 'inactive'][Math.floor(Math.random() * 6)] as ExecutionMetric['status']
  }));
};

const generateMockEvents = (count: number): ExecutionEvent[] => {
  const statuses: ExecutionEvent['status'][] = ['success', 'success', 'success', 'success', 'failure', 'timeout', 'skipped'];
  const actions = ['flag', 'notify', 'block', 'escalate', 'log', 'webhook'];
  
  return Array.from({ length: count }, (_, i) => ({
    id: `event-${Date.now()}-${i}`,
    ruleId: `rule-${(i % 10) + 1}`,
    ruleName: `Rule ${(i % 10) + 1}`,
    timestamp: new Date(Date.now() - i * 30000),
    status: statuses[Math.floor(Math.random() * statuses.length)],
    latencyMs: Math.random() * 150 + 5,
    recordsProcessed: Math.floor(Math.random() * 1000) + 1,
    recordsMatched: Math.floor(Math.random() * 100),
    actionsTriggered: Math.random() > 0.5 ? [actions[Math.floor(Math.random() * actions.length)]] : [],
    errorMessage: Math.random() > 0.9 ? 'Connection timeout to downstream service' : undefined
  }));
};

// ============================================================================
// Components
// ============================================================================

const MetricCard: React.FC<{
  title: string;
  value: string | number;
  change?: number;
  icon: React.ReactNode;
  trend?: 'up' | 'down' | 'neutral';
  subtitle?: string;
}> = ({ title, value, change, icon, trend, subtitle }) => (
  <div className="bg-white rounded-lg border p-4 shadow-sm">
    <div className="flex items-center justify-between mb-2">
      <span className="text-sm font-medium text-gray-500">{title}</span>
      <div className="text-gray-400">{icon}</div>
    </div>
    <div className="flex items-end gap-2">
      <span className="text-2xl font-bold text-gray-900">{value}</span>
      {change !== undefined && (
        <span className={`flex items-center text-sm ${
          trend === 'up' ? 'text-green-600' : trend === 'down' ? 'text-red-600' : 'text-gray-500'
        }`}>
          {trend === 'up' && <TrendingUp size={14} className="mr-1" />}
          {trend === 'down' && <TrendingDown size={14} className="mr-1" />}
          {change > 0 ? '+' : ''}{change}%
        </span>
      )}
    </div>
    {subtitle && <span className="text-xs text-gray-400 mt-1">{subtitle}</span>}
  </div>
);

const ExecutionTimeline: React.FC<{
  events: ExecutionEvent[];
  onEventClick?: (event: ExecutionEvent) => void;
}> = ({ events, onEventClick }) => {
  const getStatusIcon = (status: ExecutionEvent['status']) => {
    switch (status) {
      case 'success': return <CheckCircle size={14} className="text-green-500" />;
      case 'failure': return <XCircle size={14} className="text-red-500" />;
      case 'timeout': return <Clock size={14} className="text-yellow-500" />;
      case 'skipped': return <Pause size={14} className="text-gray-400" />;
    }
  };

  return (
    <div className="space-y-2 max-h-96 overflow-y-auto">
      {events.map(event => (
        <div
          key={event.id}
          onClick={() => onEventClick?.(event)}
          className="flex items-center gap-3 p-2 rounded hover:bg-gray-50 cursor-pointer transition-colors"
        >
          {getStatusIcon(event.status)}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-gray-900 truncate">
                {event.ruleName}
              </span>
              <span className="text-xs text-gray-400">
                {event.latencyMs.toFixed(1)}ms
              </span>
            </div>
            <div className="flex items-center gap-2 text-xs text-gray-500">
              <span>{event.recordsProcessed} processed</span>
              <span>•</span>
              <span>{event.recordsMatched} matched</span>
              {event.actionsTriggered.length > 0 && (
                <>
                  <span>•</span>
                  <span className="text-purple-600">{event.actionsTriggered.join(', ')}</span>
                </>
              )}
            </div>
          </div>
          <span className="text-xs text-gray-400 whitespace-nowrap">
            {new Date(event.timestamp).toLocaleTimeString()}
          </span>
        </div>
      ))}
    </div>
  );
};

const RuleHeatmap: React.FC<{
  metrics: ExecutionMetric[];
  onRuleClick?: (ruleId: string) => void;
}> = ({ metrics, onRuleClick }) => {
  const getHeatColor = (metric: ExecutionMetric) => {
    const failureRate = metric.executionsFailed / (metric.executionsTotal || 1);
    if (failureRate > 0.1) return 'bg-red-500';
    if (failureRate > 0.05) return 'bg-orange-500';
    if (failureRate > 0.01) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  return (
    <div className="grid grid-cols-5 gap-2">
      {metrics.map(metric => (
        <div
          key={metric.ruleId}
          onClick={() => onRuleClick?.(metric.ruleId)}
          className={`${getHeatColor(metric)} rounded p-2 cursor-pointer hover:opacity-80 transition-opacity`}
          title={`${metric.ruleName}: ${((metric.executionsFailed / metric.executionsTotal) * 100).toFixed(2)}% failure rate`}
        >
          <span className="text-xs text-white font-medium truncate block">
            {metric.ruleName.substring(0, 15)}...
          </span>
        </div>
      ))}
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

export const RuleExecutionMonitor: React.FC<RuleExecutionMonitorProps> = ({
  tenantId: _tenantId,
  datasourceId: _datasourceId,
  initialRuleFilter: _initialRuleFilter,
  onRuleClick
}) => {
  const [metrics, setMetrics] = useState<ExecutionMetric[]>([]);
  const [events, setEvents] = useState<ExecutionEvent[]>([]);
  const [selectedTimeRange, setSelectedTimeRange] = useState<TimeRange>(TIME_RANGES[2]);
  const [isLive, setIsLive] = useState(true);
  const [selectedRule, setSelectedRule] = useState<string | null>(null);
  const [showAlertConfig, setShowAlertConfig] = useState(false);
  const [alertConfigs, setAlertConfigs] = useState<AlertConfig[]>([
    { id: 'alert-1', name: 'High Latency', condition: 'latency_high', threshold: 100, severity: 'warning', enabled: true, notificationChannels: ['email'] },
    { id: 'alert-2', name: 'Failure Rate Spike', condition: 'failure_rate', threshold: 5, severity: 'critical', enabled: true, notificationChannels: ['slack', 'email'] }
  ]);
  const [expandedSection, setExpandedSection] = useState<string | null>('metrics');
  const _wsRef = useRef<WebSocket | null>(null);
  const refreshIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // Initialize data
  useEffect(() => {
    setMetrics(generateMockMetrics());
    setEvents(generateMockEvents(50));
  }, []);

  // Simulated live updates
  useEffect(() => {
    if (isLive) {
      refreshIntervalRef.current = setInterval(() => {
        // Add new events
        setEvents(prev => [generateMockEvents(1)[0], ...prev.slice(0, 99)]);
        // Update metrics slightly
        setMetrics(prev => prev.map(m => ({
          ...m,
          executionsTotal: m.executionsTotal + Math.floor(Math.random() * 10),
          lastExecutedAt: new Date()
        })));
      }, 3000);
    }
    
    return () => {
      if (refreshIntervalRef.current) {
        clearInterval(refreshIntervalRef.current);
      }
    };
  }, [isLive]);

  const handleRuleClick = useCallback((ruleId: string) => {
    setSelectedRule(ruleId);
    onRuleClick?.(ruleId);
  }, [onRuleClick]);

  const handleExport = useCallback(() => {
    const data = {
      exportedAt: new Date().toISOString(),
      timeRange: selectedTimeRange.label,
      metrics,
      events: events.slice(0, 100)
    };
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `rule-execution-report-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }, [metrics, events, selectedTimeRange]);

  const toggleAlert = useCallback((alertId: string) => {
    setAlertConfigs(prev => prev.map(a => 
      a.id === alertId ? { ...a, enabled: !a.enabled } : a
    ));
  }, []);

  // Calculate summary stats
  const totalExecutions = metrics.reduce((sum, m) => sum + m.executionsTotal, 0);
  const totalFailures = metrics.reduce((sum, m) => sum + m.executionsFailed, 0);
  const avgLatency = metrics.length > 0 
    ? metrics.reduce((sum, m) => sum + m.avgLatencyMs, 0) / metrics.length 
    : 0;
  const healthyRules = metrics.filter(m => m.status === 'healthy').length;
  const activeAlerts = alertConfigs.filter(a => a.enabled).length;

  return (
    <div className="flex flex-col h-full bg-gray-50">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b bg-white">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-green-100 rounded-lg">
            <Activity size={20} className="text-green-600" />
          </div>
          <div>
            <h2 className="font-semibold text-gray-900">Rule Execution Monitor</h2>
            <p className="text-xs text-gray-500">Real-time rule performance insights</p>
          </div>
        </div>
        
        <div className="flex items-center gap-2">
          {/* Time Range Selector */}
          <select
            value={selectedTimeRange.value}
            onChange={(e) => setSelectedTimeRange(TIME_RANGES.find(t => t.value === e.target.value) || TIME_RANGES[2])}
            className="px-3 py-1.5 border rounded text-sm"
            title="Select time range"
            aria-label="Select time range"
          >
            {TIME_RANGES.map(range => (
              <option key={range.value} value={range.value}>{range.label}</option>
            ))}
          </select>
          
          {/* Live Toggle */}
          <button
            onClick={() => setIsLive(!isLive)}
            className={`flex items-center gap-2 px-3 py-1.5 rounded text-sm transition-colors ${
              isLive 
                ? 'bg-green-100 text-green-700 border border-green-200' 
                : 'bg-gray-100 text-gray-600 border border-gray-200'
            }`}
            title={isLive ? 'Pause live updates' : 'Resume live updates'}
          >
            {isLive ? <Activity size={14} className="animate-pulse" /> : <Pause size={14} />}
            {isLive ? 'Live' : 'Paused'}
          </button>
          
          {/* Alerts */}
          <button
            onClick={() => setShowAlertConfig(!showAlertConfig)}
            className={`p-2 rounded transition-colors ${
              showAlertConfig ? 'bg-purple-100 text-purple-600' : 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'
            }`}
            title="Configure alerts"
            aria-label="Configure alerts"
          >
            {activeAlerts > 0 ? <Bell size={18} /> : <BellOff size={18} />}
          </button>
          
          {/* Export */}
          <button
            onClick={handleExport}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded transition-colors"
            title="Export report"
            aria-label="Export report"
          >
            <Download size={18} />
          </button>
          
          {/* Refresh */}
          <button
            onClick={() => {
              setMetrics(generateMockMetrics());
              setEvents(generateMockEvents(50));
            }}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded transition-colors"
            title="Refresh data"
            aria-label="Refresh data"
          >
            <RefreshCw size={18} />
          </button>
        </div>
      </div>
      
      {/* Alert Config Panel */}
      {showAlertConfig && (
        <div className="p-4 border-b bg-purple-50">
          <h3 className="font-medium text-purple-900 mb-3 flex items-center gap-2">
            <Bell size={16} />
            Alert Configuration
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {alertConfigs.map(alert => (
              <div 
                key={alert.id}
                className={`p-3 rounded border ${
                  alert.enabled ? 'bg-white border-purple-200' : 'bg-gray-50 border-gray-200 opacity-60'
                }`}
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium">{alert.name}</span>
                  <button
                    onClick={() => toggleAlert(alert.id)}
                    className={`p-1 rounded ${alert.enabled ? 'text-green-600' : 'text-gray-400'}`}
                    title={alert.enabled ? 'Disable alert' : 'Enable alert'}
                    aria-label={alert.enabled ? 'Disable alert' : 'Enable alert'}
                  >
                    {alert.enabled ? <Eye size={14} /> : <EyeOff size={14} />}
                  </button>
                </div>
                <div className="text-xs text-gray-500">
                  Threshold: {alert.threshold}{alert.condition === 'latency_high' ? 'ms' : '%'}
                </div>
                <span className={`text-xs px-1.5 py-0.5 rounded ${
                  alert.severity === 'critical' ? 'bg-red-100 text-red-700' :
                  alert.severity === 'warning' ? 'bg-yellow-100 text-yellow-700' :
                  'bg-blue-100 text-blue-700'
                }`}>
                  {alert.severity}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
      
      {/* Summary Cards */}
      <div className="p-4 grid grid-cols-2 md:grid-cols-5 gap-4">
        <MetricCard
          title="Total Executions"
          value={totalExecutions.toLocaleString()}
          change={12}
          trend="up"
          icon={<Zap size={18} />}
          subtitle={selectedTimeRange.label}
        />
        <MetricCard
          title="Success Rate"
          value={`${((1 - totalFailures / totalExecutions) * 100).toFixed(1)}%`}
          change={-0.3}
          trend="down"
          icon={<Target size={18} />}
        />
        <MetricCard
          title="Avg Latency"
          value={`${avgLatency.toFixed(1)}ms`}
          change={-5}
          trend="up"
          icon={<Clock size={18} />}
        />
        <MetricCard
          title="Active Rules"
          value={`${healthyRules}/${metrics.length}`}
          icon={<CheckCircle size={18} />}
          subtitle="Healthy"
        />
        <MetricCard
          title="Actions Triggered"
          value={metrics.reduce((sum, m) => sum + m.actionsTriggered, 0).toLocaleString()}
          change={8}
          trend="up"
          icon={<Activity size={18} />}
        />
      </div>
      
      {/* Main Content */}
      <div className="flex-1 overflow-hidden p-4">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 h-full">
          {/* Metrics Table */}
          <div className="lg:col-span-2 bg-white rounded-lg border shadow-sm overflow-hidden flex flex-col">
            <div 
              className="flex items-center justify-between p-3 border-b cursor-pointer hover:bg-gray-50"
              onClick={() => setExpandedSection(expandedSection === 'metrics' ? null : 'metrics')}
            >
              <h3 className="font-medium text-gray-900 flex items-center gap-2">
                <BarChart2 size={16} />
                Rule Performance
              </h3>
              {expandedSection === 'metrics' ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
            </div>
            
            {expandedSection === 'metrics' && (
              <div className="flex-1 overflow-auto">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 sticky top-0">
                    <tr>
                      <th className="px-4 py-2 text-left font-medium text-gray-600">Rule</th>
                      <th className="px-4 py-2 text-left font-medium text-gray-600">Category</th>
                      <th className="px-4 py-2 text-right font-medium text-gray-600">Executions</th>
                      <th className="px-4 py-2 text-right font-medium text-gray-600">Success %</th>
                      <th className="px-4 py-2 text-right font-medium text-gray-600">Avg Latency</th>
                      <th className="px-4 py-2 text-center font-medium text-gray-600">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y">
                    {metrics.map(metric => (
                      <tr 
                        key={metric.ruleId}
                        onClick={() => handleRuleClick(metric.ruleId)}
                        className={`hover:bg-gray-50 cursor-pointer ${
                          selectedRule === metric.ruleId ? 'bg-blue-50' : ''
                        }`}
                      >
                        <td className="px-4 py-3">
                          <span className="font-medium text-gray-900">{metric.ruleName}</span>
                        </td>
                        <td className="px-4 py-3">
                          <span className="text-xs px-2 py-1 bg-gray-100 rounded">
                            {metric.category}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right text-gray-600">
                          {metric.executionsTotal.toLocaleString()}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <span className={metric.executionsFailed / metric.executionsTotal > 0.05 ? 'text-red-600' : 'text-green-600'}>
                            {((1 - metric.executionsFailed / metric.executionsTotal) * 100).toFixed(1)}%
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right text-gray-600">
                          {metric.avgLatencyMs.toFixed(1)}ms
                        </td>
                        <td className="px-4 py-3">
                          <span className={`text-xs px-2 py-1 rounded-full border ${STATUS_COLORS[metric.status]}`}>
                            {metric.status}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
          
          {/* Right Sidebar */}
          <div className="flex flex-col gap-4">
            {/* Live Timeline */}
            <div className="bg-white rounded-lg border shadow-sm flex-1 overflow-hidden flex flex-col">
              <div className="flex items-center justify-between p-3 border-b">
                <h3 className="font-medium text-gray-900 flex items-center gap-2">
                  <Activity size={16} />
                  Live Executions
                  {isLive && <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />}
                </h3>
              </div>
              <div className="flex-1 overflow-hidden p-3">
                <ExecutionTimeline events={events.slice(0, 20)} onEventClick={(e) => handleRuleClick(e.ruleId)} />
              </div>
            </div>
            
            {/* Heatmap */}
            <div className="bg-white rounded-lg border shadow-sm p-4">
              <h3 className="font-medium text-gray-900 mb-3 flex items-center gap-2">
                <PieChart size={16} />
                Rule Health Heatmap
              </h3>
              <RuleHeatmap metrics={metrics} onRuleClick={handleRuleClick} />
              <div className="flex items-center justify-center gap-4 mt-3 text-xs">
                <div className="flex items-center gap-1">
                  <span className="w-3 h-3 bg-green-500 rounded" />
                  <span>&lt;1%</span>
                </div>
                <div className="flex items-center gap-1">
                  <span className="w-3 h-3 bg-yellow-500 rounded" />
                  <span>1-5%</span>
                </div>
                <div className="flex items-center gap-1">
                  <span className="w-3 h-3 bg-orange-500 rounded" />
                  <span>5-10%</span>
                </div>
                <div className="flex items-center gap-1">
                  <span className="w-3 h-3 bg-red-500 rounded" />
                  <span>&gt;10%</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default RuleExecutionMonitor;
