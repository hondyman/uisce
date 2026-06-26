import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { observabilityApi, WasmModuleVersion } from "../../api/observabilityApi";
import { format } from "date-fns";

export function WasmVersionRegistryPage() {
  const [moduleName, setModuleName] = useState("risk-compliance-engine");
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["wasm-versions", moduleName],
    queryFn: () => observabilityApi.listWasmVersions(moduleName || undefined)
  });

  const activateMutation = useMutation({
    mutationFn: (id: string) => observabilityApi.activateWasmVersion(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["wasm-versions"] });
    }
  });

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">WASM Registry</h1>
          <p className="text-sm text-gray-500 mt-1">Manage and activate WASM execution artifacts for the SemLayer compute engine.</p>
        </div>
      </div>

      <div className="mb-6 flex gap-4 p-4 bg-white rounded-lg shadow-sm border border-gray-200">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Module Namespace</label>
          <input
            type="text"
            className="border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm px-3 py-2 border w-64"
            placeholder="e.g. risk-compliance-engine"
            value={moduleName}
            onChange={(e) => setModuleName(e.target.value)}
          />
        </div>
      </div>

      <div className="bg-white shadow-sm rounded-lg border border-gray-200 overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-gray-500">Loading registry data...</div>
        ) : error ? (
          <div className="p-8 text-center text-red-500">Error loading WASM versions.</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Version ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tag</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Hash (SHA-256)</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Size</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Uploaded By</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {data?.versions?.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-sm text-gray-500">
                    No WASM bundles found for this module.
                  </td>
                </tr>
              ) : (
                data?.versions?.map((v: WasmModuleVersion) => (
                  <tr key={v.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 font-mono">
                      {v.id.split('-')[0]}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="px-2 py-1 text-xs font-mono bg-gray-100 text-gray-800 rounded">{v.version_tag}</span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 font-mono text-xs">
                      {v.wasm_hash.substring(0, 16)}...
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {formatBytes(v.size_bytes)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {v.is_active ? (
                        <span className="px-2 py-1 text-xs font-bold rounded-full bg-green-100 text-green-800 border border-green-200">ACTIVE</span>
                      ) : (
                        <span className="px-2 py-1 text-xs rounded-full bg-gray-100 text-gray-500">INACTIVE</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      <div>{v.uploaded_by}</div>
                      <div className="text-xs text-gray-400">{format(new Date(v.created_at), "MMM d, yyyy HH:mm")}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      {!v.is_active && (
                        <button
                          onClick={() => activateMutation.mutate(v.id)}
                          disabled={activateMutation.isPending}
                          className="text-blue-600 hover:text-blue-900 disabled:opacity-50 border border-blue-600 rounded px-3 py-1 text-xs font-semibold"
                        >
                          {activateMutation.isPending ? "Activating..." : "Activate Now"}
                        </button>
                      )}
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
