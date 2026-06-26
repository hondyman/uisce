import React from "react";
import type { UpgradeArtifacts } from "../../../types/upgrade-generated";
import { SchemaHistory } from "./SchemaHistory";

interface UpgradeStatusResponse {
  core_version: string;
  status: 'pending' | 'ready' | 'canary' | 'active' | 'rolled_back';
  warnings: string[];
  blockers: string[];
}

type Props = {
  artifacts: UpgradeArtifacts;
  status: UpgradeStatusResponse;
  onNavigate: (tool: "diff" | "fixer" | "queries") => void;
};

export const UpgradeOverview: React.FC<Props> = ({ artifacts, status, onNavigate }) => {
  const { schema_version: _schema_version, report } = artifacts;

  return (
    <div className="upgrade-overview space-y-6">
      {/* Schema Version & Changelog */}
      <SchemaHistory artifacts={artifacts} />

      {/* Core Version & Status */}
      <div className="border rounded p-4 bg-white shadow-sm">
        <h2 className="font-semibold text-lg">Core Upgrade</h2>
        <div className="mt-2 space-y-1">
          <p className="text-sm text-gray-600">
            Core Version: <span className="font-mono font-semibold">{report.core_version}</span>
          </p>
          <p className="text-sm text-gray-600">
            Previous: <span className="font-mono">{report.previous_version}</span>
          </p>
          <p className="text-sm text-gray-600">
            Generated: <span className="font-mono text-xs">{new Date(report.generated_at).toLocaleString()}</span>
          </p>
        </div>
        <div className="mt-3">
          <span
            className={`inline-block px-3 py-1 rounded-full text-sm font-medium ${
              status.status === "active"
                ? "bg-green-100 text-green-800"
                : status.status === "canary"
                ? "bg-yellow-100 text-yellow-800"
                : "bg-gray-100 text-gray-800"
            }`}
          >
            {status.status.toUpperCase()}
          </span>
        </div>
      </div>

      {/* Upgrade Summary */}
      <div className="border rounded p-4 bg-white shadow-sm">
        <h3 className="font-semibold text-lg">Upgrade Summary</h3>
        <div className="mt-3 grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-green-600">{report.summary.cubes_added}</div>
            <div className="text-sm text-gray-600">Cubes Added</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-red-600">{report.summary.cubes_removed}</div>
            <div className="text-sm text-gray-600">Cubes Removed</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-600">{report.summary.views_changed}</div>
            <div className="text-sm text-gray-600">Views Changed</div>
          </div>
          <div className="text-center">
            <div className={`text-2xl font-bold ${report.summary.breaking_changes > 0 ? 'text-red-600' : 'text-green-600'}`}>
              {report.summary.breaking_changes}
            </div>
            <div className="text-sm text-gray-600">Breaking Changes</div>
          </div>
        </div>
      </div>

      {/* Warnings & Blockers */}
      {(status.warnings.length > 0 || status.blockers.length > 0 || report.warnings.length > 0) && (
        <div className="border rounded p-4 bg-white shadow-sm">
          <h3 className="font-semibold text-lg text-orange-600">⚠️ Upgrade Health</h3>

          {status.blockers.length > 0 && (
            <div className="mt-3">
              <p className="text-red-700 font-semibold">🚫 Blockers:</p>
              <ul className="list-disc list-inside text-sm text-red-700 mt-1 space-y-1">
                {status.blockers.map((b, idx) => (
                  <li key={idx}>{b}</li>
                ))}
              </ul>
            </div>
          )}

          {status.warnings.length > 0 && (
            <div className="mt-3">
              <p className="text-yellow-700 font-semibold">⚠️ Status Warnings:</p>
              <ul className="list-disc list-inside text-sm text-yellow-700 mt-1 space-y-1">
                {status.warnings.map((w, idx) => (
                  <li key={idx}>{w}</li>
                ))}
              </ul>
            </div>
          )}

          {report.warnings.length > 0 && (
            <div className="mt-3">
              <p className="text-blue-700 font-semibold">ℹ️ Diff Warnings:</p>
              <ul className="list-disc list-inside text-sm text-blue-700 mt-1 space-y-1">
                {report.warnings.map((w, idx) => (
                  <li key={idx}>{w}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* Quick Navigation */}
      <div className="border rounded p-4 bg-white shadow-sm">
        <h3 className="font-semibold text-lg">Next Steps</h3>
        <p className="text-sm text-gray-600 mt-1">Navigate through the upgrade workflow</p>
        <div className="mt-4 flex flex-wrap gap-3">
          <button
            onClick={() => onNavigate("diff")}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors flex items-center gap-2"
          >
            <span>📊</span>
            Review Diff
          </button>
          <button
            onClick={() => onNavigate("fixer")}
            className="px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700 transition-colors flex items-center gap-2"
          >
            <span>🔧</span>
            Fix Extensions
          </button>
          <button
            onClick={() => onNavigate("queries")}
            className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 transition-colors flex items-center gap-2"
          >
            <span>⚡</span>
            Run Golden Queries
          </button>
        </div>
      </div>
    </div>
  );
};
