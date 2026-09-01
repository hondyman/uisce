import React, { useState, useEffect } from 'react';

// Types for Dashboard Data
interface TenantUpgradeStatus {
  tenantId: string;
  coreVersion: string;
  overlayVersion: string;
  totalTests: number;
  passed: number;
  failed: number;
  conflicts: number;
  snapshotId: string;
}

interface Conflict {
  id: string;
  tenantId: string;
  reason: string;
  path: string;
}

export const UpgradeDashboardPage: React.FC = () => {
  const [statuses, setStatuses] = useState<TenantUpgradeStatus[]>([]);
  const [conflicts, setConflicts] = useState<Conflict[]>([]);
  const [selectedTenant, setSelectedTenant] = useState<string | null>(null);

  useEffect(() => {
    // Mock Data Fetch
    setStatuses([
      { tenantId: 'Client A', coreVersion: 'v1.3.0', overlayVersion: 'v1.3.0-ovl', totalTests: 120, passed: 118, failed: 2, conflicts: 0, snapshotId: 'snap-001' },
      { tenantId: 'Client B', coreVersion: 'v1.3.0', overlayVersion: 'v1.3.0-ovl', totalTests: 95, passed: 95, failed: 0, conflicts: 0, snapshotId: 'snap-002' },
      { tenantId: 'Client C', coreVersion: 'v1.3.0', overlayVersion: 'v1.3.0-ovl', totalTests: 110, passed: 109, failed: 1, conflicts: 1, snapshotId: 'snap-003' },
    ]);

    setConflicts([
      { id: 'c1', tenantId: 'Client C', reason: 'View "AccountForm" field risk_rating visibility rule invalid', path: 'views.AccountForm.fields.risk_rating' },
    ]);
  }, []);

  const handleResolveConflict = (id: string) => {
    alert(`Resolving conflict ${id}... (Mock Action)`);
    setConflicts(conflicts.filter(c => c.id !== id));
  };

  return (
    <div className="p-6 bg-gray-50 min-h-screen font-sans text-gray-800">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Upgrade Dashboard</h1>
        <div className="mt-2 text-sm text-gray-600 flex space-x-6">
          <span><strong>Core Version:</strong> v1.3.0</span>
          <span><strong>Snapshot:</strong> iceberg-snap-20251122-001</span>
          <span><strong>Timestamp:</strong> 22:45 EST</span>
        </div>
      </div>

      {/* Summary Table */}
      <div className="bg-white rounded-lg shadow mb-8 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold">Tenant Upgrade Status</h2>
        </div>
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tenant</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Core Ver</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Overlay Ver</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tests</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Pass/Fail</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Evidence</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {statuses.map((s) => (
              <tr key={s.tenantId} className="hover:bg-gray-50 cursor-pointer" onClick={() => setSelectedTenant(s.tenantId)}>
                <td className="px-6 py-4 whitespace-nowrap font-medium text-gray-900">{s.tenantId}</td>
                <td className="px-6 py-4 whitespace-nowrap text-gray-500">{s.coreVersion}</td>
                <td className="px-6 py-4 whitespace-nowrap text-gray-500">{s.overlayVersion}</td>
                <td className="px-6 py-4 whitespace-nowrap text-gray-500">{s.totalTests}</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className="text-green-600 font-bold">{s.passed}</span> / <span className="text-red-600 font-bold">{s.failed}</span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  {s.conflicts > 0 ? (
                    <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-red-100 text-red-800">
                      Conflicts ({s.conflicts})
                    </span>
                  ) : s.failed > 0 ? (
                    <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800">
                      Issues
                    </span>
                  ) : (
                    <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                      Ready
                    </span>
                  )}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-blue-600 hover:underline text-sm">
                  {s.snapshotId}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Ops Panel: Conflicts */}
      {conflicts.length > 0 && (
        <div className="bg-white rounded-lg shadow mb-8 border-l-4 border-red-500">
          <div className="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
            <h2 className="text-lg font-semibold text-red-700">Conflict Queue</h2>
            <span className="bg-red-100 text-red-800 text-xs px-2 py-1 rounded-full">{conflicts.length} Pending</span>
          </div>
          <div className="p-6">
            {conflicts.map((c) => (
              <div key={c.id} className="flex items-center justify-between bg-red-50 p-4 rounded mb-2">
                <div>
                  <p className="font-bold text-gray-800">{c.tenantId}</p>
                  <p className="text-sm text-gray-600">{c.reason}</p>
                  <p className="text-xs text-gray-500 font-mono mt-1">{c.path}</p>
                </div>
                <button 
                  onClick={() => handleResolveConflict(c.id)}
                  className="bg-red-600 text-white px-4 py-2 rounded hover:bg-red-700 text-sm"
                >
                  Resolve
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Drill Down (Mock) */}
      {selectedTenant && (
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-xl font-bold mb-4">Details for {selectedTenant}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="p-4 border rounded bg-gray-50">
              <h3 className="font-semibold mb-2">Test Categories</h3>
              <ul className="space-y-2 text-sm">
                <li className="flex justify-between"><span>Business Objects</span> <span className="text-green-600">✔️ 58/58</span></li>
                <li className="flex justify-between"><span>Business Processes</span> <span className="text-green-600">✔️ 30/30</span></li>
                <li className="flex justify-between"><span>UI Views</span> <span className="text-red-600">❌ 28/30</span></li>
                <li className="flex justify-between"><span>Metrics</span> <span className="text-green-600">✔️ 4/4</span></li>
              </ul>
            </div>
            <div className="p-4 border rounded bg-gray-50">
              <h3 className="font-semibold mb-2">Actions</h3>
              <div className="space-y-2">
                <button className="w-full bg-blue-100 text-blue-700 py-2 rounded hover:bg-blue-200">View UAR Log</button>
                <button className="w-full bg-blue-100 text-blue-700 py-2 rounded hover:bg-blue-200">Download Iceberg Snapshot</button>
                <button className="w-full bg-gray-200 text-gray-700 py-2 rounded hover:bg-gray-300">Rollback to Previous Version</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
