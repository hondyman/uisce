import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { observabilityApi, ScenarioLineage } from "../../api/observabilityApi";
import { format } from "date-fns";

export function ScenarioLineageExplorerPage() {
  const [scenarioId, setScenarioId] = useState("");
  const [tenantId, setTenantId] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["scenario-lineage", scenarioId, tenantId],
    queryFn: () => observabilityApi.getScenarioLineage(scenarioId, { tenant_id: tenantId || undefined }),
    enabled: !!scenarioId // Only fetch if we have a Scenario ID
  });

  const formatCurrency = (val: number) => {
    return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(val);
  };

  const getPnlBadge = (pnlAmount: number, pnlPercent: number) => {
    const isLoss = pnlAmount < 0;
    return (
      <div className={`px-2 py-1 flex items-center gap-2 rounded border ${isLoss ? 'bg-red-50 border-red-200 text-red-800' : 'bg-green-50 border-green-200 text-green-800'}`}>
         <span className="font-mono text-sm font-semibold">{formatCurrency(pnlAmount)}</span>
         <span className="text-xs">({(pnlPercent * 100).toFixed(2)}%)</span>
      </div>
    );
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Scenario Lineage Trace</h1>
          <p className="text-sm text-gray-500 mt-1">Explore historical stress test results and VaR projections mapped by semantic factors.</p>
        </div>
      </div>

      <div className="mb-6 flex gap-4 p-4 bg-white rounded-lg shadow-sm border border-gray-200">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Scenario ID (Required)</label>
          <input
            type="text"
            className="border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border w-72 font-mono"
            placeholder="UUID of the scenario..."
            value={scenarioId}
            onChange={(e) => setScenarioId(e.target.value)}
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
      </div>

      <div className="bg-white shadow-sm rounded-lg border border-gray-200 overflow-hidden">
        {!scenarioId ? (
          <div className="p-8 text-center text-gray-500">Please enter a Scenario ID to view execution lineage.</div>
        ) : isLoading ? (
          <div className="p-8 text-center text-gray-500">Fetching lineage traces...</div>
        ) : error ? (
          <div className="p-8 text-center text-red-500">Error loading scenario lineage.</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Date</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Portfolio</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Base Value</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Stressed Value</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">PnL Impact</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">WASM Perf</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Semantic Trace</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {data?.lineage?.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-sm text-gray-500">
                    No traces found for this Scenario ID.
                  </td>
                </tr>
              ) : (
                data?.lineage?.map((trace: ScenarioLineage) => (
                  <tr key={trace.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      {format(new Date(trace.valuation_date), "MMM d, yyyy")}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      <div className="font-mono text-xs" title="Portfolio ID">{trace.portfolio_id.split('-')[0]}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 font-mono text-right">
                      {formatCurrency(trace.total_base_value)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 font-mono text-right">
                       {formatCurrency(trace.total_stressed_value)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {getPnlBadge(trace.pnl_amount, trace.pnl_percent)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">{trace.duration_ms} ms</div>
                      <div className="text-xs text-gray-400 font-mono" title="WASM Engine Version">{trace.wasm_version_id.split('-')[0]}</div>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                       <div className="flex flex-wrap gap-1 max-w-xs">
                          {trace.semantic_terms_used?.length > 0 ? trace.semantic_terms_used.map(term => (
                            <span key={term} className="px-1.5 py-0.5 bg-blue-50 text-blue-700 border border-blue-100 rounded text-xs truncate">
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
