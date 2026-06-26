import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { observabilityApi, ETLRun } from "../../api/observabilityApi";
import { format } from "date-fns";

export function ETLRunDashboardPage() {
  const [tenantId, setTenantId] = useState("");
  const [status, setStatus] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["etl-runs", tenantId, status],
    queryFn: () => observabilityApi.listETLRuns({ tenant_id: tenantId || undefined, status: status || undefined })
  });

  const getStatusBadge = (runStatus: string) => {
    switch (runStatus) {
      case "SUCCESS":
        return <span className="px-2 py-1 text-xs font-semibold rounded-full bg-green-100 text-green-800">Success</span>;
      case "FAILED":
        return <span className="px-2 py-1 text-xs font-semibold rounded-full bg-red-100 text-red-800">Failed</span>;
      case "STARTED":
        return <span className="px-2 py-1 text-xs font-semibold rounded-full bg-blue-100 text-blue-800">Running</span>;
      default:
        return <span className="px-2 py-1 text-xs font-semibold rounded-full bg-gray-100 text-gray-800">{runStatus}</span>;
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">ETL Telemetry Dashboard</h1>
          <p className="text-sm text-gray-500 mt-1">Monitor the semantic execution fabric runs across all tenants.</p>
        </div>
      </div>

      <div className="mb-6 flex gap-4 p-4 bg-white rounded-lg shadow-sm border border-gray-200">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Tenant ID</label>
          <input
            type="text"
            className="border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border w-64"
            placeholder="Filter by Tenant..."
            value={tenantId}
            onChange={(e) => setTenantId(e.target.value)}
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
          <select
            className="border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border w-40"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="">All Statuses</option>
            <option value="SUCCESS">Success</option>
            <option value="FAILED">Failed</option>
            <option value="STARTED">Started</option>
          </select>
        </div>
      </div>

      <div className="bg-white shadow-sm rounded-lg border border-gray-200 overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-gray-500">Loading telemetry data...</div>
        ) : error ? (
          <div className="p-8 text-center text-red-500">Error loading ETL runs.</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Run ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Valuation Date</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Duration</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Evaluations</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Version</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {data?.runs?.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-sm text-gray-500">
                    No ETL telemetry found matching criteria.
                  </td>
                </tr>
              ) : (
                data?.runs?.map((run: ETLRun) => (
                  <tr key={run.id} className="hover:bg-gray-50 cursor-pointer">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-blue-600 font-mono">{run.id.split('-')[0]}</div>
                      <div className="text-xs text-gray-500 mt-1">Tenant: {run.tenant_id.split('-')[0]}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {format(new Date(run.valuation_date), "MMM d, yyyy")}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {getStatusBadge(run.status)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {run.duration_ms ? `${run.duration_ms} ms` : "-"}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">{run.rules_evaluated} rules</div>
                      <div className="text-xs text-gray-500">{run.scenarios_evaluated} scen.</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 font-mono">
                      {run.wasm_orchestrator_version ? run.wasm_orchestrator_version.split('-')[0] : "-"}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
