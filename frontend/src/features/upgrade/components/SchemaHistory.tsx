import React, { useState } from "react";
import type { UpgradeArtifacts } from "../../../types/upgrade-generated";

type Props = { artifacts: UpgradeArtifacts };

export const SchemaHistory: React.FC<Props> = ({ artifacts }) => {
  const [expanded, setExpanded] = useState(false);

  const changelog = artifacts.changelog || [];

  return (
    <div className="schema-history border rounded p-4 bg-white shadow-sm">
      <header className="flex justify-between items-center">
        <div>
          <h2 className="font-semibold text-lg">Schema Version</h2>
          <p className="text-sm text-gray-600">
            Current: <span className="font-mono">{artifacts.schema_version}</span>
          </p>
        </div>
        {changelog.length > 0 && (
          <button
            onClick={() => setExpanded(!expanded)}
            className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            {expanded ? "Hide History" : "View History"}
          </button>
        )}
      </header>

      {expanded && changelog.length > 0 && (
        <div className="mt-4 space-y-3">
          {changelog.map((entry, idx) => {
            const isBreaking = entry.description.toLowerCase().includes('breaking') ||
                              entry.description.toLowerCase().includes('removed') ||
                              entry.description.toLowerCase().includes('deprecated');

            return (
              <div key={idx} className={`border-l-4 pl-3 ${isBreaking ? 'border-red-500' : 'border-blue-500'}`}>
                <div className="flex justify-between items-start">
                  <span className="font-mono font-semibold">{entry.version}</span>
                  <span className="text-xs text-gray-500">
                    {new Date(entry.date).toLocaleString()}
                  </span>
                </div>
                <p className={`text-sm mt-1 ${isBreaking ? 'text-red-700 font-medium' : 'text-gray-700'}`}>
                  {entry.description}
                </p>
                {isBreaking && (
                  <span className="inline-block mt-1 px-2 py-1 text-xs bg-red-100 text-red-800 rounded">
                    Breaking Change
                  </span>
                )}
              </div>
            );
          })}
        </div>
      )}

      {changelog.length === 0 && (
        <p className="mt-2 text-sm text-gray-500">No changelog entries available.</p>
      )}
    </div>
  );
};
