/**
 * Metrics Console - Detail Page
 * Shows metric metadata, PoP trends, anomalies, and job runs
 */

import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  useMetric,
  usePop,
  useAnomalies,
  useRuns,
  useTriggerPop,
  useTriggerAnomaly,
} from '../hooks/useMetricsConsole';

export default function MetricDetailPage() {
  const { metricId } = useParams<{ metricId: string }>();
  const navigate = useNavigate();
  const [from, setFrom] = useState<string | undefined>();
  const [to, setTo] = useState<string | undefined>();
  const [tab, setTab] = useState<'pop' | 'anomalies' | 'runs'>('pop');

  const { data: metric, isLoading: metricLoading } = useMetric(metricId);
  const { data: pop } = usePop(metricId, { from, to });
  const { data: anomalies } = useAnomalies(metricId, { from, to });
  const { data: runs } = useRuns(metricId);
  const { mutate: triggerPop, isPending: popRunning } = useTriggerPop(metricId!);
  const { mutate: triggerAnomaly, isPending: anomalyRunning } = useTriggerAnomaly(metricId!);

  if (metricLoading) return <div className="p-8 text-center">Loading...</div>;
  if (!metric) return <div className="p-8 text-center text-red-500">Metric not found</div>;

  return (
    <div className="p-8 max-w-7xl mx-auto">
      {/* Header & Metadata */}
      <div className="mb-8">
        <div className="flex justify-between items-start mb-4">
          <div>
            <h1 className="text-4xl font-black text-gray-900 dark:text-white mb-2">
              {metric.display_name || metric.name}
            </h1>
            <p className="text-gray-600 dark:text-gray-400">{metric.name}</p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => navigate(`/metrics/${metricId}/edit`)}
              className="px-4 h-10 bg-primary text-white rounded-lg hover:bg-primary/90 text-sm font-bold"
            >
              Edit
            </button>
            <button
              onClick={() => navigate('/metrics')}
              className="px-4 h-10 bg-gray-200 dark:bg-gray-800 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 text-sm font-bold"
            >
              Back
            </button>
          </div>
        </div>

        {/* Metadata Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 bg-gray-50 dark:bg-gray-900 rounded-lg p-4">
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400 uppercase font-semibold">Domain</p>
            <p className="text-sm font-medium text-gray-900 dark:text-white mt-1">{metric.domain}</p>
          </div>
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400 uppercase font-semibold">Granularity</p>
            <p className="text-sm font-medium text-gray-900 dark:text-white mt-1">{metric.granularity}</p>
          </div>
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400 uppercase font-semibold">Golden</p>
            <p className="text-sm font-medium text-gray-900 dark:text-white mt-1">
              {metric.golden_path ? '⭐ Yes' : 'No'}
            </p>
          </div>
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400 uppercase font-semibold">SLA Freshness</p>
            <p className="text-sm font-medium text-gray-900 dark:text-white mt-1">
              {metric.sla_freshness_hours ?? '-'} hrs
            </p>
          </div>
        </div>
      </div>

      {/* Toolbar */}
      <div className="flex flex-col md:flex-row gap-4 mb-6 pb-4 border-b border-gray-200 dark:border-gray-800">
        <div className="flex gap-2">
          <input
            type="date"
            title="Start date"
            aria-label="Start date"
            value={from ?? ''}
            onChange={(e) => setFrom(e.target.value || undefined)}
            className="px-3 h-10 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
          />
          <input
            type="date"
            title="End date"
            aria-label="End date"
            value={to ?? ''}
            onChange={(e) => setTo(e.target.value || undefined)}
            className="px-3 h-10 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
          />
        </div>
        <div className="flex gap-2 ml-auto">
          <button
            onClick={() => triggerPop({})}
            disabled={popRunning}
            className="flex items-center gap-2 px-4 h-10 bg-gray-200 dark:bg-gray-800 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 text-sm font-bold disabled:opacity-50"
          >
            <span>↻</span>
            <span>Recompute PoP</span>
          </button>
          <button
            onClick={() => triggerAnomaly({ threshold: 2.5, window_days: 90 })}
            disabled={anomalyRunning}
            className="flex items-center gap-2 px-4 h-10 bg-primary text-white rounded-lg hover:bg-primary/90 text-sm font-bold disabled:opacity-50"
          >
            <span>🔍</span>
            <span>Analyze Anomalies</span>
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="mb-6 border-b border-gray-200 dark:border-gray-800 flex gap-6">
        {(['pop', 'anomalies', 'runs'] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`pb-3 text-sm font-medium border-b-2 transition-colors ${
              tab === t
                ? 'border-primary text-primary'
                : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
            }`}
          >
            {t === 'pop' ? 'PoP Trend' : t === 'anomalies' ? 'Anomalies' : 'Runs'}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {tab === 'pop' && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="text-xs uppercase bg-gray-50 dark:bg-gray-800 text-gray-600 dark:text-gray-400 font-semibold">
                <tr>
                  <th className="px-6 py-3">Period</th>
                  <th className="px-6 py-3">Current</th>
                  <th className="px-6 py-3">Previous</th>
                  <th className="px-6 py-3">Δ</th>
                  <th className="px-6 py-3">%</th>
                  <th className="px-6 py-3">Records</th>
                  <th className="px-6 py-3">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
                {!pop || pop.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-6 py-4 text-center text-gray-500">
                      No PoP data
                    </td>
                  </tr>
                ) : (
                  pop.map((r) => (
                    <tr key={r.period_label} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                      <td className="px-6 py-4 font-medium text-gray-900 dark:text-white">
                        {r.period_label}
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">{r.current_value}</td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">
                        {r.previous_value ?? '-'}
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">{r.delta ?? '-'}</td>
                      <td
                        className={`px-6 py-4 font-medium ${
                          r.percent_change && r.percent_change > 0
                            ? 'text-green-600 dark:text-green-400'
                            : 'text-red-600 dark:text-red-400'
                        }`}
                      >
                        {r.percent_change ? `${r.percent_change.toFixed(2)}%` : '-'}
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">{r.record_count}</td>
                      <td className="px-6 py-4">
                        <span
                          className={`inline-block px-2 py-1 text-xs font-semibold rounded-full ${
                            r.computation_status === 'success'
                              ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400'
                              : r.computation_status === 'failed'
                                ? 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-400'
                                : 'bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-400'
                          }`}
                        >
                          {r.computation_status}
                        </span>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tab === 'anomalies' && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="text-xs uppercase bg-gray-50 dark:bg-gray-800 text-gray-600 dark:text-gray-400 font-semibold">
                <tr>
                  <th className="px-6 py-3">Detected</th>
                  <th className="px-6 py-3">Type</th>
                  <th className="px-6 py-3">Severity</th>
                  <th className="px-6 py-3">Confidence</th>
                  <th className="px-6 py-3">Actual</th>
                  <th className="px-6 py-3">Expected</th>
                  <th className="px-6 py-3">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
                {!anomalies || anomalies.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-6 py-4 text-center text-gray-500">
                      No anomalies detected
                    </td>
                  </tr>
                ) : (
                  anomalies.map((a, i) => (
                    <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                      <td className="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">
                        {new Date(a.detected_at).toLocaleString().slice(0, 16)}
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">{a.anomaly_type}</td>
                      <td className="px-6 py-4">
                        <span
                          className={`inline-block px-2 py-1 text-xs font-semibold rounded-full ${
                            a.severity === 'high' || a.severity === 'critical'
                              ? 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-400'
                              : a.severity === 'medium'
                                ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-400'
                                : 'bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-400'
                          }`}
                        >
                          {a.severity}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">
                        {a.confidence ? (a.confidence * 100).toFixed(0) + '%' : '-'}
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">
                        {a.actual_value ?? '-'}
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">
                        {a.expected_value ?? '-'}
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">{a.status}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tab === 'runs' && (
        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="text-xs uppercase bg-gray-50 dark:bg-gray-800 text-gray-600 dark:text-gray-400 font-semibold">
                <tr>
                  <th className="px-6 py-3">Run ID</th>
                  <th className="px-6 py-3">Type</th>
                  <th className="px-6 py-3">Status</th>
                  <th className="px-6 py-3">Period</th>
                  <th className="px-6 py-3">Started</th>
                  <th className="px-6 py-3">Ended</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
                {!runs || runs.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-6 py-4 text-center text-gray-500">
                      No runs yet
                    </td>
                  </tr>
                ) : (
                  runs.map((r) => (
                    <tr key={r.run_id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                      <td className="px-6 py-4 font-mono text-xs text-primary">
                        {r.run_id.slice(0, 8)}
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">{r.calc_type}</td>
                      <td className="px-6 py-4">
                        <span
                          className={`inline-block px-2 py-1 text-xs font-semibold rounded-full ${
                            r.status === 'success'
                              ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400'
                              : r.status === 'failed'
                                ? 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-400'
                                : r.status === 'running'
                                  ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-400'
                                  : 'bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-400'
                          }`}
                        >
                          {r.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-gray-700 dark:text-gray-300">
                        {r.period_label ?? '-'}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">
                        {new Date(r.started_at).toLocaleString().slice(0, 16)}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">
                        {r.ended_at ? new Date(r.ended_at).toLocaleString().slice(0, 16) : '-'}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
