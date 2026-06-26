import React, { useEffect, useState } from 'react';

interface AuditEvent {
  event_id: string;
  run_id: string;
  seq: number;
  event_type: string;
  payload_canon: string;
  payload_hash: string;
  parent_hash: string;
  timestamp: string;
}

export const AuditLogViewer: React.FC = () => {
  const [events, setEvents] = useState<AuditEvent[]>([]);

  useEffect(() => {
    fetch('/api/audit/events')
      .then(res => res.json())
      .then(data => setEvents(data))
      .catch(err => console.error(err));
  }, []);

  return (
    <div className="p-6 bg-gray-900 text-white rounded-lg shadow-xl">
      <h2 className="text-2xl font-bold mb-4 flex items-center">
        <span className="mr-2">🔒</span> Immutable Audit Trail
      </h2>
      <div className="overflow-x-auto">
        <table className="min-w-full text-sm">
          <thead>
            <tr className="border-b border-gray-700">
              <th className="px-4 py-2 text-left">Time</th>
              <th className="px-4 py-2 text-left">Event Type</th>
              <th className="px-4 py-2 text-left">Seq</th>
              <th className="px-4 py-2 text-left">Hash (SHA-256)</th>
              <th className="px-4 py-2 text-left">Payload</th>
            </tr>
          </thead>
          <tbody>
            {events.map((evt) => (
              <tr key={evt.event_id} className="border-b border-gray-800 hover:bg-gray-800 font-mono">
                <td className="px-4 py-2 text-gray-400">
                  {new Date(evt.timestamp).toLocaleString()}
                </td>
                <td className="px-4 py-2 text-blue-400 font-semibold">{evt.event_type}</td>
                <td className="px-4 py-2">{evt.seq}</td>
                <td className="px-4 py-2 text-xs text-green-500 truncate max-w-[150px]" title={evt.payload_hash}>
                  {evt.payload_hash.substring(0, 12)}...
                </td>
                <td className="px-4 py-2 text-xs text-gray-500 truncate max-w-[300px]" title={evt.payload_canon}>
                  {evt.payload_canon}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
