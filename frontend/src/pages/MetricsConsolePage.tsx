/**
 * Metrics Console - Main List Page
 * Registry discovery with filters and CRUD actions
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMetrics, useDeleteMetric } from '../hooks/useMetricsConsole';
import { useConfirm } from '../components/ConfirmProvider';
import { MetricRegistry } from '../types/metrics-console';

export default function MetricsConsolePage() {
  const navigate = useNavigate();
  const [q, setQ] = useState('');
  const [domain, setDomain] = useState('');
  const [golden, setGolden] = useState(false);

  const { data: metrics, isLoading, error } = useMetrics({ q, domain, golden });
  const { mutate: deleteMetric, isPending: isDeleting } = useDeleteMetric();
  const confirm = useConfirm();

  const handleDelete = async (metric_id: string) => {
    if (await confirm({ title: 'Delete metric', description: 'Are you sure you want to delete this metric?' })) {
      deleteMetric(metric_id);
    }
  };

  return (
    <div className="p-8 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex flex-wrap justify-between gap-4 mb-8">
        <h1 className="text-4xl font-black text-gray-900 dark:text-white">Metric Registry</h1>
        <button
          onClick={() => navigate('/metrics/create')}
          className="flex items-center gap-2 px-4 h-10 bg-primary text-white rounded-lg hover:bg-primary/90 text-sm font-bold"
        >
          <span>➕</span>
          <span>New Metric</span>
        </button>
      </div>

      {/* Filters */}
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 mb-6">
        <div className="p-4 border-b border-gray-200 dark:border-gray-800">
          <div className="flex flex-col md:flex-row gap-4">
            <input
              type="text"
              placeholder="Search by name..."
              value={q}
              onChange={(e) => setQ(e.target.value)}
              className="flex-1 px-4 h-10 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
            <input
              type="text"
              placeholder="Domain..."
              value={domain}
              onChange={(e) => setDomain(e.target.value)}
              className="flex-1 px-4 h-10 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
            <label className="flex items-center gap-2 px-4 h-10 rounded-lg bg-gray-100 dark:bg-gray-800">
              <input
                type="checkbox"
                checked={golden}
                onChange={(e) => setGolden(e.target.checked)}
                className="w-4 h-4 text-primary rounded"
              />
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Golden Only</span>
            </label>
          </div>
        </div>

        {/* Table */}
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="text-xs uppercase bg-gray-50 dark:bg-gray-900/50 text-gray-600 dark:text-gray-400 font-semibold">
              <tr>
                <th className="px-6 py-3">Name</th>
                <th className="px-6 py-3">Domain</th>
                <th className="px-6 py-3">Granularity</th>
                <th className="px-6 py-3 text-center">Golden</th>
                <th className="px-6 py-3">SLA (hrs)</th>
                <th className="px-6 py-3">Updated</th>
                <th className="px-6 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
              {isLoading ? (
                <tr>
                  <td colSpan={7} className="px-6 py-4 text-center text-gray-500">
                    Loading...
                  </td>
                </tr>
              ) : error ? (
                <tr>
                  <td colSpan={7} className="px-6 py-4 text-center text-red-500">
                    Error loading metrics
                  </td>
                </tr>
              ) : !metrics || metrics.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-4 text-center text-gray-500">
                    No metrics found
                  </td>
                </tr>
              ) : (
                metrics.map((m: MetricRegistry) => (
                  <tr
                    key={m.id}
                    className="bg-white dark:bg-gray-900/50 hover:bg-gray-50 dark:hover:bg-gray-800/50 border-b border-gray-200 dark:border-gray-800"
                  >
                    <td className="px-6 py-4 font-semibold text-gray-900 dark:text-white whitespace-nowrap">
                      <button
                        onClick={() => navigate(`/metrics/${m.id}`)}
                        className="text-primary hover:underline"
                      >
                        {m.display_name || m.name}
                      </button>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-gray-700 dark:text-gray-300">
                      {m.domain}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-gray-700 dark:text-gray-300">
                      {m.granularity}
                    </td>
                    <td className="px-6 py-4 text-center whitespace-nowrap">
                      {m.golden_path ? <span className="text-lg">⭐</span> : ''}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-gray-700 dark:text-gray-300">
                      {m.sla_freshness_hours ?? '-'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                      {m.updated_at ? new Date(m.updated_at).toLocaleString().slice(0, 16) : '-'}
                    </td>
                    <td className="px-6 py-4 text-right whitespace-nowrap">
                      <button
                        onClick={() => navigate(`/metrics/${m.id}/edit`)}
                        className="px-2 py-1 text-primary hover:text-primary/80 text-sm font-medium"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => handleDelete(m.id)}
                        disabled={isDeleting}
                        className="px-2 py-1 text-red-600 hover:text-red-800 text-sm font-medium ml-2 disabled:opacity-50"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
