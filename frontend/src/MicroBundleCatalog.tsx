import { useEffect, useState } from "react";
import { JITRequestPanel } from "./JITRequestPanel";
import { AccessExplanation } from "./AccessExplanation";
import apiClient from "./utils/apiClient";

export interface MicroBundle {
  id: string;
  name: string;
  description: string;
  claims: any[];
  domain: string;
  version: number;
}

export function MicroBundleCatalog() {
  const [bundles, setBundles] = useState<MicroBundle[]>([]);
  const [filter, setFilter] = useState({ domain: "", permission: "" });
  const [selected, setSelected] = useState<MicroBundle | null>(null);
  const [showJIT, setShowJIT] = useState(false);

  useEffect(() => {
    apiClient("micro-bundles")
      .then((r) => r.json())
      .then(setBundles);
  }, []);

  const filtered = bundles.filter(
    (b) =>
      (!filter.domain || b.domain.includes(filter.domain)) &&
      (!filter.permission || b.claims.some((c) => c.permission?.includes(filter.permission)))
  );

  return (
    <div className="p-4">
      <h2 className="text-xl font-bold mb-2">Micro-Bundle Catalog</h2>
      <div className="flex gap-2 mb-4">
        <input
          placeholder="Domain"
          value={filter.domain}
          onChange={(e) => setFilter({ ...filter, domain: e.target.value })}
          className="border px-2 py-1 rounded"
        />
        <input
          placeholder="Permission"
          value={filter.permission}
          onChange={(e) => setFilter({ ...filter, permission: e.target.value })}
          className="border px-2 py-1 rounded"
        />
      </div>
      <table className="w-full border mb-4">
        <thead>
          <tr className="bg-gray-100">
            <th>Name</th>
            <th>Domain</th>
            <th>Claims</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {filtered.map((b) => (
            <tr key={b.id}>
              <td>{b.name}</td>
              <td>{b.domain}</td>
              <td>{b.claims.length}</td>
              <td>
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
                {selected.claims.map((c) => (
                  <li key={JSON.stringify(c)}>{JSON.stringify(c)}</li>
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
