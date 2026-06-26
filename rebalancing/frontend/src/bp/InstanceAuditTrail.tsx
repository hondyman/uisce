import React, { useEffect, useState } from 'react';

interface AuditEvent {
  id: string;
  createdAt: string;
  eventType: string;
  actorId: string;
  actorRole: string;
  oldValue: any;
  newValue: any;
  reason?: string;
}

export const InstanceAuditTrail: React.FC<{ instanceId: string }> = ({ instanceId }) => {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    // Mock API call or fetch real endpoint
    fetch(`/api/instances/${instanceId}/audit-trail`)
        .then(r => {
            if (r.ok) return r.json();
            return []; 
        })
        .then(setEvents)
        .finally(() => setLoading(false));
  }, [instanceId]);

  if (loading) return <div className="text-gray-500 text-sm">Loading audit trail...</div>;
  if (events.length === 0) return <div className="text-gray-400 text-sm italic">No audit events found.</div>;

  return (
    <div className="space-y-4 border-l-2 border-gray-200 pl-4 ml-2">
      {events.map(e => (
        <div key={e.id} className="relative">
             <div className="absolute -left-[21px] top-1 h-3 w-3 rounded-full bg-gray-300 ring-4 ring-white"></div>
             <div className="flex flex-col">
                 <div className="flex items-center text-sm text-gray-500 space-x-2">
                    <span className="font-mono text-xs">{new Date(e.createdAt).toLocaleString()}</span>
                    <span>•</span>
                    <span className="font-medium text-gray-900">{e.eventType}</span>
                 </div>
                 <div className="text-sm mt-1">
                    <span className="font-medium">{e.actorRole}</span> ({e.actorId})
                 </div>
                 {(e.oldValue || e.newValue) && (
                    <details className="mt-2 text-xs bg-gray-50 p-2 rounded border cursor-pointer">
                        <summary className="text-blue-600 hover:underline">Data Change</summary>
                        <pre className="mt-1 overflow-x-auto">
                            {JSON.stringify({ old: e.oldValue, new: e.newValue }, null, 2)}
                        </pre>
                    </details>
                 )}
                 {e.reason && <div className="mt-1 text-sm text-gray-600 italic">"{e.reason}"</div>}
             </div>
        </div>
      ))}
    </div>
  );
};
