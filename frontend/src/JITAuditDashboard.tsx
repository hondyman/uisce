import { useEffect, useState } from "react";

interface JITGrantAuditEvent {
  id: string;
  grantId: string;
  userId: string;
  eventType: string;
  reason: string;
  occurredAt: string;
}

export function JITAuditDashboard() {
  const [events, setEvents] = useState<JITGrantAuditEvent[]>([]);
  const [userId, setUserId] = useState("");
  const [bundleId, setBundleId] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    let url = "/api/jit-audit";
    const params: string[] = [];
    if (userId) params.push(`user_id=${encodeURIComponent(userId)}`);
    if (bundleId) params.push(`bundle_id=${encodeURIComponent(bundleId)}`);
    if (params.length) url += "?" + params.join("&");
    fetch(url)
      .then((r) => r.json())
      .then(setEvents)
      .finally(() => setLoading(false));
  }, [userId, bundleId]);

  const exportCSV = () => {
    const header = "ID,Grant ID,User ID,Event Type,Reason,Occurred At\n";
    const rows = events.map(e =>
      [e.id, e.grantId, e.userId, e.eventType, e.reason, e.occurredAt].map(x => `"${x}"`).join(",")
    ).join("\n");
    const blob = new Blob([header + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "jit_audit_log.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="p-6">
      <h2 className="text-2xl font-bold mb-4">JIT Audit & Compliance Dashboard</h2>
      <div className="flex gap-4 mb-4">
        <input
          placeholder="Filter by User ID"
          value={userId}
          onChange={e => setUserId(e.target.value)}
          className="border px-2 py-1 rounded"
        />
        <input
          placeholder="Filter by Bundle ID"
          value={bundleId}
          onChange={e => setBundleId(e.target.value)}
          className="border px-2 py-1 rounded"
        />
        <button
          className="bg-blue-600 text-white px-4 py-1 rounded"
          onClick={exportCSV}
          disabled={!events.length}
        >
          Export CSV
        </button>
      </div>
      {loading ? (
        <div className="text-gray-500">Loading audit events...</div>
      ) : !events.length ? (
        <div className="text-gray-500">No audit events found.</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border rounded shadow-sm">
            <thead>
              <tr className="bg-gray-100">
                <th className="px-3 py-2 text-left">ID</th>
                <th className="px-3 py-2 text-left">Grant ID</th>
                <th className="px-3 py-2 text-left">User ID</th>
                <th className="px-3 py-2 text-left">Event Type</th>
                <th className="px-3 py-2 text-left">Reason</th>
                <th className="px-3 py-2 text-left">Occurred At</th>
              </tr>
            </thead>
            <tbody>
              {events.map(e => (
                <tr key={e.id} className="hover:bg-gray-50">
                  <td className="px-3 py-2">{e.id}</td>
                  <td className="px-3 py-2">{e.grantId}</td>
                  <td className="px-3 py-2">{e.userId}</td>
                  <td className="px-3 py-2">{e.eventType}</td>
                  <td className="px-3 py-2">{e.reason}</td>
                  <td className="px-3 py-2">{e.occurredAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
