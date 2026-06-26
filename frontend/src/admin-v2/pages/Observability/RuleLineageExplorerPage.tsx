import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { observabilityApi, RuleLineage } from "../../api/observabilityApi";
import { format } from "date-fns";

export function RuleLineageExplorerPage() {
  const [ruleId, setRuleId] = useState("");
  const [tenantId, setTenantId] = useState("");
  const [status, setStatus] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["rule-lineage", ruleId, tenantId, status],
    queryFn: () => observabilityApi.getRuleLineage(ruleId, { tenant_id: tenantId || undefined, status: status || undefined }),
    enabled: !!ruleId // Only fetch if we have a Rule ID
  });

  const getStatusBadge = (runStatus: string) => {
    switch (runStatus) {
      case "PASS":
        return <span className="px-2 py-1 text-xs font-bold rounded bg-green-100 text-green-800">PASS</span>;
      case "FAIL":
        return <span className="px-2 py-1 text-xs font-bold rounded bg-red-100 text-red-800">FAIL</span>;
      case "WARNING":
        return <span className="px-2 py-1 text-xs font-bold rounded bg-yellow-100 text-yellow-800">WARNING</span>;
      default:
        return <span className="px-2 py-1 text-xs font-bold rounded bg-gray-100 text-gray-800">{runStatus}</span>;
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Rule Lineage Trace</h1>
          <p className="text-sm text-gray-500 mt-1">Explore historical executions and semantic evaluation lineage for Compliance Rules.</p>
        </div>
      </div>

      <div className="mb-6 flex gap-4 p-4 bg-white rounded-lg shadow-sm border border-gray-200">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Rule ID (Required)</label>
          <input
            type="text"
            className="border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border w-72 font-mono"
            placeholder="UUID of the rule..."
            value={ruleId}
            onChange={(e) => setRuleId(e.target.value)}
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Tenant ID</label>
          <input
            type="text"
            className="border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border w-48 font-mono"
            placeholder="Optional"
            value={tenantId}
            onChange={(e) => setTenantId(e.target.value)}
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Outcome</label>
          <select
            className="border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border w-40"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="">All</option>
            <option value="PASS">Pass</option>
            <option value="FAIL">Fail</option>
            <option value="WARNING">Warning</option>
          </select>
        </div>
      </div>

      <div className="bg-white shadow-sm rounded-lg border border-gray-200 overflow-hidden">
        {!ruleId ? (
          <div className="p-8 text-center text-gray-500">Please enter a Rule ID to view execution lineage.</div>
        ) : isLoading ? (
          <div className="p-8 text-center text-gray-500">Fetching lineage traces...</div>
        ) : error ? (
          <div className="p-8 text-center text-red-500">Error loading rule lineage.</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Date</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Portfolio / Target</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Outcome</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Metric vs Threshold</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">WASM Perf</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Semantics Hit</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {data?.lineage?.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-sm text-gray-500">
                    No traces found for this Rule ID.
                  </td>
                </tr>
              ) : (
                data?.lineage?.map((trace: RuleLineage) => (
                  <tr key={trace.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      {format(new Date(trace.valuation_date), "MMM d, yyyy")}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      <div className="font-mono text-xs mb-1" title="Portfolio ID">{trace.portfolio_id.split('-')[0]}</div>
                      {trace.security_id && <div className="font-mono text-xs text-blue-600" title="Security Focus">{trace.security_id.split('-')[0]}</div>}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {getStatusBadge(trace.status)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                       {(trace.metric_value !== undefined && trace.threshold_value !== undefined) ? (
                         <div className="flex items-center gap-2">
                           <span className={`text-sm font-mono ${trace.status === 'FAIL' ? 'text-red-600 font-bold' : 'text-gray-900'}`}>{trace.metric_value}</span>
                           <span className="text-gray-400 text-xs">/</span>
                           <span className="text-sm font-mono text-gray-500">{trace.threshold_value}</span>
                         </div>
                       ) : <span className="text-gray-400">-</span>}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">{trace.duration_ms} ms</div>
                      <div className="text-xs text-gray-400 font-mono" title="WASM Engine Version">{trace.wasm_version_id.split('-')[0]}</div>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                       <div className="flex flex-wrap gap-1">
                          {trace.semantic_terms_used?.length > 0 ? trace.semantic_terms_used.map(term => (
                            <span key={term} className="px-1.5 py-0.5 bg-blue-50 text-blue-700 border border-blue-100 rounded text-xs">
                              {term}
                            </span>
                          )) : <span className="text-gray-400">None mapped</span>}
                       </div>
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
