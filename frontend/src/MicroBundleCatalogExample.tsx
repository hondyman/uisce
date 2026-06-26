import React from "react";
import { useMicroBundles } from "./api/microbundles";
import { JITRequestPanel } from "./JITRequestPanel";
import { AccessExplanation } from "./AccessExplanation";

export function MicroBundleCatalogExample() {
  const { data: bundles = [], isLoading } = useMicroBundles();
  const [selected, setSelected] = React.useState<any | null>(null);
  const [showJIT, setShowJIT] = React.useState(false);

  if (isLoading) return <div className="p-4 text-gray-500">Loading micro-bundles...</div>;
  if (!bundles.length) return <div className="p-4 text-gray-500">No micro-bundles found.</div>;

  return (
    <div className="p-4">
      <h2 className="text-xl font-bold mb-2">Micro-Bundle Catalog (Example)</h2>
      <table className="w-full border mb-4 rounded overflow-hidden shadow-sm">
        <thead>
          <tr className="bg-gray-100">
            <th className="px-3 py-2 text-left">Name</th>
            <th className="px-3 py-2 text-left">Domain</th>
            <th className="px-3 py-2 text-left">Claims</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {bundles.map((b: any) => (
            <tr key={b.id} className="hover:bg-gray-50">
              <td className="px-3 py-2">{b.name}</td>
              <td className="px-3 py-2">{b.domain}</td>
              <td className="px-3 py-2">{b.claims.length}</td>
              <td className="px-3 py-2">
                <button
                  className="text-blue-600 underline"
                  onClick={() => setSelected(b)}
                >
                  Details
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {selected && (
        <div className="fixed inset-0 bg-black bg-opacity-30 flex items-center justify-center z-50">
          <div className="bg-white p-6 rounded shadow-lg max-w-lg w-full relative">
            <button
              className="absolute top-2 right-2 text-gray-500"
              onClick={() => setSelected(null)}
            >
              ×
            </button>
            <h3 className="text-lg font-bold mb-2">{selected.name}</h3>
            <p className="mb-2">{selected.description}</p>
            <div className="mb-2">
              <strong>Claims:</strong>
              <ul className="list-disc ml-6">
                {selected.claims.map((c: any, i: number) => (
                  <li key={i}>{JSON.stringify(c)}</li>
                ))}
              </ul>
            </div>
            <div className="mb-2">
              <strong>Version:</strong> {selected.version}
            </div>
            <div className="mb-2">
              <strong>Usage Example:</strong> <code>GET /api/micro-bundles/{selected.id}</code>
            </div>
            <div className="mb-2">
              <strong>Expiry Policy:</strong> JIT add-ons expire per policy (see below)
            </div>
            <button
              className="bg-blue-600 text-white px-4 py-2 rounded mt-2"
              onClick={() => {
                setShowJIT(true);
                setSelected(null);
              }}
            >
              Request JIT Add-On
            </button>
          </div>
        </div>
      )}
      {showJIT && <JITRequestPanel onClose={() => setShowJIT(false)} />}
      <AccessExplanation />
    </div>
  );
}
