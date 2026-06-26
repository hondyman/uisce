import { useState } from 'react';
import { getEnv } from '../../utils/getEnv';
import { useNotification } from '../../hooks/useNotification';
import {
  AlertCircle,
  CheckCircle,
  Clock,
  TrendingUp,
  TrendingDown,
  Plus,
  Edit2,
  Trash2,
  Play,
  ArrowLeft,
} from 'lucide-react';

// ============================================================================
// Type Definitions
// ============================================================================

interface Metric {
  metric_id: string;
  name: string;
  domain: string;
  granularity: 'day' | 'month' | 'quarter';
  aggregation_function: 'sum' | 'avg' | 'count' | 'ratio';
  sla_freshness_hours: number;
  golden_path: boolean;
  owner_user_id: string;
  updated_at: string;
}

interface PopData {
  period_label: string;
  current_value: number;
  previous_value: number;
  delta: number;
  percent_change: number;
  record_count: number;
  computation_status: 'success' | 'failed' | 'running';
}

interface Anomaly {
  id: string;
  detected_at: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  confidence: number;
  actual_value: number;
  expected_value: number;
  status: 'open' | 'resolved';
}

interface Run {
  run_id: string;
  calc_type: 'pop' | 'anomaly' | 'custom';
  period_label: string;
  status: 'success' | 'running' | 'failed';
  started_at: string;
  ended_at?: string;
}

// ============================================================================
// Mock API Layer (swap with real API calls)
// ============================================================================

const _API_BASE = getEnv('', 'VITE_API_BASE_URL', 'http://localhost:9088/api') as string;

// Mock data for demo
const MOCK_METRICS: Metric[] = [
  {
    metric_id: 'metric-1',
    name: 'Revenue',
    domain: 'Finance',
    granularity: 'month',
    aggregation_function: 'sum',
    sla_freshness_hours: 24,
    golden_path: true,
    owner_user_id: 'user-456',
    updated_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
  },
  {
    metric_id: 'metric-2',
    name: 'Customer Count',
    domain: 'Operations',
    granularity: 'day',
    aggregation_function: 'count',
    sla_freshness_hours: 6,
    golden_path: false,
    owner_user_id: 'user-789',
    updated_at: new Date(Date.now() - 5 * 60 * 60 * 1000).toISOString(),
  },
];

const MOCK_POP_DATA: PopData[] = [
  {
    period_label: '2024-11',
    current_value: 125000.5,
    previous_value: 118000.25,
    delta: 7000.25,
    percent_change: 5.93,
    record_count: 31,
    computation_status: 'success',
  },
  {
    period_label: '2024-10',
    current_value: 118000.25,
    previous_value: 115000.75,
    delta: 2999.5,
    percent_change: 2.61,
    record_count: 31,
    computation_status: 'success',
  },
];

const MOCK_ANOMALIES: Anomaly[] = [
  {
    id: 'anom-1',
    detected_at: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString(),
    severity: 'high',
    confidence: 0.95,
    actual_value: 185000.5,
    expected_value: 125000.5,
    status: 'open',
  },
];

const MOCK_RUNS: Run[] = [
  {
    run_id: 'run-1',
    calc_type: 'pop',
    period_label: '2024-11',
    status: 'success',
    started_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    ended_at: new Date(Date.now() - 25 * 60 * 1000).toISOString(),
  },
];

// ============================================================================
// Components
// ============================================================================

const MetricRegistryTab = ({ onSelectMetric }: { onSelectMetric: (id: string) => void }) => {
  const [metrics, setMetrics] = useState<Metric[]>(MOCK_METRICS);
  const notification = useNotification();
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState<Omit<Metric, 'metric_id' | 'owner_user_id' | 'golden_path' | 'updated_at'>>({
    name: '',
    domain: '',
    granularity: 'month',
    aggregation_function: 'sum',
    sla_freshness_hours: 24,
  });
  const [editingId, setEditingId] = useState<string | null>(null);

  const handleSubmit = () => {
    if (!formData.name || !formData.domain) {
      notification.error('Please fill in all required fields');
      return;
    }

    if (editingId) {
      setMetrics(
        metrics.map((m) =>
          m.metric_id === editingId
            ? { ...m, ...formData, updated_at: new Date().toISOString() }
            : m
        )
      );
      setEditingId(null);
    } else {
      const newMetric: Metric = {
        metric_id: `metric-${Date.now()}`,
        ...formData,
        owner_user_id: 'current-user',
        golden_path: false,
        updated_at: new Date().toISOString(),
      };
      setMetrics([...metrics, newMetric]);
    }
    setShowForm(false);
    setFormData({
      name: '',
      domain: '',
      granularity: 'month',
      aggregation_function: 'sum',
      sla_freshness_hours: 24,
    });
  };

  const handleEdit = (metric: Metric) => {
    setFormData(metric);
    setEditingId(metric.metric_id);
    setShowForm(true);
  };

  const handleDelete = (id: string) => {
    setMetrics(metrics.filter((m) => m.metric_id !== id));
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-bold text-slate-900">Metric Registry</h2>
        <button
          onClick={() => {
            setShowForm(!showForm);
            setEditingId(null);
            setFormData({
              name: '',
              domain: '',
              granularity: 'month',
              aggregation_function: 'sum',
              sla_freshness_hours: 24,
            });
          }}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition"
        >
          <Plus size={18} /> New Metric
        </button>
      </div>

      {showForm && (
        <div className="bg-slate-50 p-4 rounded border border-slate-200 space-y-3">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Metric Name *
              </label>
              <input
                type="text"
                placeholder="e.g., Revenue"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-3 py-2 border border-slate-300 rounded hover:border-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">Domain *</label>
              <input
                type="text"
                placeholder="e.g., Finance"
                value={formData.domain}
                onChange={(e) => setFormData({ ...formData, domain: e.target.value })}
                className="w-full px-3 py-2 border border-slate-300 rounded hover:border-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">Granularity</label>
              <select
                title="Select granularity"
                aria-label="Granularity"
                value={formData.granularity}
                onChange={(e) => setFormData({ ...formData, granularity: e.target.value as 'day' | 'month' | 'quarter' })}
                className="w-full px-3 py-2 border border-slate-300 rounded hover:border-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="day">Daily</option>
                <option value="month">Monthly</option>
                <option value="quarter">Quarterly</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Aggregation
              </label>
              <select
                title="Select aggregation function"
                aria-label="Aggregation function"
                value={formData.aggregation_function}
                onChange={(e) => setFormData({ ...formData, aggregation_function: e.target.value as 'sum' | 'avg' | 'count' | 'ratio' })}
                className="w-full px-3 py-2 border border-slate-300 rounded hover:border-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="sum">Sum</option>
                <option value="avg">Average</option>
                <option value="count">Count</option>
                <option value="ratio">Ratio</option>
              </select>
            </div>
          </div>
          <div className="flex gap-2">
            <button
              onClick={handleSubmit}
              className="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700 transition font-medium"
            >
              {editingId ? 'Update' : 'Create'}
            </button>
            <button
              onClick={() => setShowForm(false)}
              className="bg-slate-400 text-white px-4 py-2 rounded hover:bg-slate-500 transition font-medium"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      <div className="grid gap-3">
        {metrics.map((metric) => (
          <div
            key={metric.metric_id}
            className="bg-white border border-slate-200 rounded p-4 hover:shadow-md transition cursor-pointer hover:border-slate-300"
          >
            <div className="flex justify-between items-start">
              <div
                className="flex-1 cursor-pointer"
                onClick={() => onSelectMetric(metric.metric_id)}
              >
                <div className="flex items-center gap-2 mb-2">
                  <h3 className="font-bold text-slate-900 hover:text-blue-600">{metric.name}</h3>
                  {metric.golden_path && (
                    <span className="bg-yellow-100 text-yellow-800 px-2 py-1 text-xs rounded font-medium">
                      Golden Path
                    </span>
                  )}
                </div>
                <div className="grid grid-cols-4 gap-2 text-sm text-slate-600">
                  <div>
                    <span className="font-medium">Domain:</span> {metric.domain}
                  </div>
                  <div>
                    <span className="font-medium">Grain:</span> {metric.granularity}
                  </div>
                  <div>
                    <span className="font-medium">Agg:</span> {metric.aggregation_function}
                  </div>
                  <div>
                    <span className="font-medium">SLA:</span> {metric.sla_freshness_hours}h
                  </div>
                </div>
                <div className="text-xs text-slate-500 mt-2">
                  Updated {new Date(metric.updated_at).toLocaleString()}
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleEdit(metric);
                  }}
                  title="Edit metric"
                  aria-label={`Edit metric ${metric.name}`}
                  className="p-2 text-blue-600 hover:bg-blue-50 rounded transition"
                >
                  <Edit2 size={18} />
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDelete(metric.metric_id);
                  }}
                  title="Delete metric"
                  aria-label={`Delete metric ${metric.name}`}
                  className="p-2 text-red-600 hover:bg-red-50 rounded transition"
                >
                  <Trash2 size={18} />
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

const MetricDetailView = ({ 
  metricId, 
  onBack, 
  metrics: allMetrics 
}: { 
  metricId: string;
  onBack: () => void;
  metrics: Metric[];
}) => {
  const [activeTab, setActiveTab] = useState<'pop' | 'anomalies' | 'runs'>('pop');
  const [_popData] = useState<PopData[]>(MOCK_POP_DATA);
  const [_anomalies] = useState<Anomaly[]>(MOCK_ANOMALIES);
  const [_runs] = useState<Run[]>(MOCK_RUNS);
  const [triggering, setTriggering] = useState(false);

  const metric = allMetrics.find((m: Metric) => m.metric_id === metricId);
  const notification = useNotification();

  const handleTriggerPop = async () => {
    setTriggering(true);
    try {
      // Replace with real API call
      await new Promise((resolve) => setTimeout(resolve, 500));
      notification.success('✅ PoP compute triggered!');
    } catch (err) {
      notification.error('❌ Error triggering compute');
    } finally {
      setTriggering(false);
    }
  };

  const handleTriggerAnomaly = async () => {
    setTriggering(true);
    try {
      await new Promise((resolve) => setTimeout(resolve, 500));
      notification.success('✅ Anomaly detection triggered!');
    } catch (err) {
      notification.error('❌ Error triggering detection');
    } finally {
      setTriggering(false);
    }
  };

  if (!metric) return <div className="text-red-600">Metric not found</div>;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between mb-4">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-blue-600 hover:text-blue-800 text-sm font-medium transition"
        >
          <ArrowLeft size={16} /> Back to Registry
        </button>
        <h2 className="text-2xl font-bold text-slate-900">{metric.name}</h2>
        <div className="flex gap-2">
          <button
            onClick={handleTriggerPop}
            disabled={triggering}
            className="flex items-center gap-2 bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700 disabled:opacity-50 transition font-medium"
          >
            <Play size={16} /> Compute PoP
          </button>
          <button
            onClick={handleTriggerAnomaly}
            disabled={triggering}
            className="flex items-center gap-2 bg-orange-600 text-white px-4 py-2 rounded hover:bg-orange-700 disabled:opacity-50 transition font-medium"
          >
            <AlertCircle size={16} /> Detect
          </button>
        </div>
      </div>

      <div className="bg-white border border-slate-200 rounded p-4">
        <div className="grid grid-cols-4 gap-4 text-sm">
          <div>
            <span className="text-slate-600">Domain:</span>
            <div className="font-medium text-slate-900">{metric.domain}</div>
          </div>
          <div>
            <span className="text-slate-600">Granularity:</span>
            <div className="font-medium text-slate-900">{metric.granularity}</div>
          </div>
          <div>
            <span className="text-slate-600">Aggregation:</span>
            <div className="font-medium text-slate-900">{metric.aggregation_function}</div>
          </div>
          <div>
            <span className="text-slate-600">SLA:</span>
            <div className="font-medium text-slate-900">{metric.sla_freshness_hours}h freshness</div>
          </div>
        </div>
      </div>

      <div className="flex gap-2 border-b border-slate-200">
        {(['pop', 'anomalies', 'runs'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 font-medium border-b-2 transition ${
              activeTab === tab
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-slate-600 hover:text-slate-900'
            }`}
          >
            {tab === 'pop' && '📈 PoP Trend'}
            {tab === 'anomalies' && '🚨 Anomalies'}
            {tab === 'runs' && '⏱️ Runs'}
          </button>
        ))}
      </div>

      {activeTab === 'pop' && <PopTrendTable data={_popData} />}
      {activeTab === 'anomalies' && <AnomalyTriageTable data={_anomalies} />}
      {activeTab === 'runs' && <RunsAuditTable data={_runs} />}
    </div>
  );
};

const PopTrendTable = ({ data }: { data: PopData[] }) => (
  <div className="bg-white border border-slate-200 rounded overflow-hidden">
    <table className="w-full text-sm">
      <thead className="bg-slate-50 border-b">
        <tr>
          <th className="px-4 py-2 text-left font-medium text-slate-900">Period</th>
          <th className="px-4 py-2 text-right font-medium text-slate-900">Current</th>
          <th className="px-4 py-2 text-right font-medium text-slate-900">Previous</th>
          <th className="px-4 py-2 text-right font-medium text-slate-900">Delta</th>
          <th className="px-4 py-2 text-right font-medium text-slate-900">% Change</th>
          <th className="px-4 py-2 text-right font-medium text-slate-900">Records</th>
          <th className="px-4 py-2 text-center font-medium text-slate-900">Status</th>
        </tr>
      </thead>
      <tbody>
        {data.map((row: PopData, idx: number) => (
          <tr key={idx} className={idx % 2 === 0 ? 'bg-white' : 'bg-slate-50'}>
            <td className="px-4 py-2 font-medium text-slate-900">{row.period_label}</td>
            <td className="px-4 py-2 text-right text-slate-900">
              ${row.current_value.toLocaleString('en-US', { maximumFractionDigits: 0 })}
            </td>
            <td className="px-4 py-2 text-right text-slate-600">
              ${row.previous_value.toLocaleString('en-US', { maximumFractionDigits: 0 })}
            </td>
            <td className="px-4 py-2 text-right">
              <span
                className={row.delta > 0 ? 'text-green-600 font-medium' : 'text-red-600 font-medium'}
              >
                ${row.delta.toLocaleString('en-US', { maximumFractionDigits: 0 })}
              </span>
            </td>
            <td className="px-4 py-2 text-right">
              <div className="flex items-center justify-end gap-1">
                <span
                  className={
                    row.percent_change > 0 ? 'text-green-600 font-medium' : 'text-red-600 font-medium'
                  }
                >
                  {row.percent_change.toFixed(2)}%
                </span>
                {row.percent_change > 0 ? (
                  <TrendingUp size={16} className="text-green-600" />
                ) : (
                  <TrendingDown size={16} className="text-red-600" />
                )}
              </div>
            </td>
            <td className="px-4 py-2 text-right text-slate-600">{row.record_count}</td>
            <td className="px-4 py-2 text-center">
              <span className="inline-flex items-center gap-1 bg-green-50 text-green-700 px-2 py-1 rounded text-xs font-medium">
                <CheckCircle size={14} /> SUCCESS
              </span>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const AnomalyTriageTable = ({ data }: { data: Anomaly[] }) => (
  <div className="bg-white border border-slate-200 rounded overflow-hidden">
    <table className="w-full text-sm">
      <thead className="bg-slate-50 border-b">
        <tr>
          <th className="px-4 py-2 text-left font-medium text-slate-900">Detected</th>
          <th className="px-4 py-2 text-center font-medium text-slate-900">Severity</th>
          <th className="px-4 py-2 text-right font-medium text-slate-900">Confidence</th>
          <th className="px-4 py-2 text-right font-medium text-slate-900">Actual</th>
          <th className="px-4 py-2 text-right font-medium text-slate-900">Expected</th>
          <th className="px-4 py-2 text-center font-medium text-slate-900">Status</th>
        </tr>
      </thead>
      <tbody>
        {data.map((anom: Anomaly, idx: number) => (
          <tr key={idx} className={idx % 2 === 0 ? 'bg-white' : 'bg-slate-50'}>
            <td className="px-4 py-2">{new Date(anom.detected_at).toLocaleString()}</td>
            <td className="px-4 py-2 text-center">
              <span
                className={`inline-block px-2 py-1 rounded text-xs font-medium ${
                  anom.severity === 'critical'
                    ? 'bg-red-100 text-red-700'
                    : anom.severity === 'high'
                      ? 'bg-orange-100 text-orange-700'
                      : 'bg-yellow-100 text-yellow-700'
                }`}
              >
                {anom.severity.toUpperCase()}
              </span>
            </td>
            <td className="px-4 py-2 text-right font-medium">{(anom.confidence * 100).toFixed(1)}%</td>
            <td className="px-4 py-2 text-right font-medium text-slate-900">
              ${anom.actual_value.toLocaleString()}
            </td>
            <td className="px-4 py-2 text-right text-slate-600">
              ${anom.expected_value.toLocaleString()}
            </td>
            <td className="px-4 py-2 text-center">
              <span
                className={`text-xs font-medium px-2 py-1 rounded ${
                  anom.status === 'open' ? 'bg-red-50 text-red-700' : 'bg-blue-50 text-blue-700'
                }`}
              >
                {anom.status.toUpperCase()}
              </span>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const RunsAuditTable = ({ data }: { data: Run[] }) => (
  <div className="bg-white border border-slate-200 rounded overflow-hidden">
    <table className="w-full text-sm">
      <thead className="bg-slate-50 border-b">
        <tr>
          <th className="px-4 py-2 text-left font-medium text-slate-900">Run ID</th>
          <th className="px-4 py-2 text-left font-medium text-slate-900">Type</th>
          <th className="px-4 py-2 text-left font-medium text-slate-900">Period</th>
          <th className="px-4 py-2 text-left font-medium text-slate-900">Started</th>
          <th className="px-4 py-2 text-right font-medium text-slate-900">Duration</th>
          <th className="px-4 py-2 text-center font-medium text-slate-900">Status</th>
        </tr>
      </thead>
      <tbody>
        {data.map((run: Run, idx: number) => {
          const started = new Date(run.started_at);
          const ended = run.ended_at ? new Date(run.ended_at) : new Date();
          const durationMs = ended.getTime() - started.getTime();
          const durationSec = (durationMs / 1000).toFixed(1);

          return (
            <tr key={idx} className={idx % 2 === 0 ? 'bg-white' : 'bg-slate-50'}>
              <td className="px-4 py-2 font-mono text-slate-600">{run.run_id.slice(-8)}</td>
              <td className="px-4 py-2 font-medium">{run.calc_type.toUpperCase()}</td>
              <td className="px-4 py-2 text-slate-900">{run.period_label}</td>
              <td className="px-4 py-2 text-slate-600">{started.toLocaleTimeString()}</td>
              <td className="px-4 py-2 text-right text-slate-600">{durationSec}s</td>
              <td className="px-4 py-2 text-center">
                {run.status === 'success' && (
                  <span className="inline-flex items-center gap-1 bg-green-50 text-green-700 px-2 py-1 rounded text-xs font-medium">
                    <CheckCircle size={14} /> SUCCESS
                  </span>
                )}
                {run.status === 'running' && (
                  <span className="inline-flex items-center gap-1 bg-blue-50 text-blue-700 px-2 py-1 rounded text-xs font-medium animate-pulse">
                    <Clock size={14} /> RUNNING
                  </span>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  </div>
);

// ============================================================================
// Main App
// ============================================================================

export default function MetricCalcConsole() {
  const [metrics] = useState<Metric[]>(MOCK_METRICS);
  const [selectedMetricId, setSelectedMetricId] = useState<string | null>(null);

  if (selectedMetricId) {
    return (
      <div className="space-y-4">
        <MetricDetailView
          metricId={selectedMetricId}
          onBack={() => setSelectedMetricId(null)}
          metrics={metrics}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="bg-white rounded border border-slate-200 p-6 shadow-sm">
        <MetricRegistryTab onSelectMetric={(id: string) => setSelectedMetricId(id)} />
      </div>
    </div>
  );
}
