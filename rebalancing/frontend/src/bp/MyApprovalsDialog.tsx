import React, { useEffect, useState } from 'react';

interface PendingTask {
  id: string;
  bpKey: string;
  currentStep: string;
  entity: string;
  applicantName: string;
  slaExpiresAt: string;
}

export const MyApprovalsDialog: React.FC<{ userId: string; onClose: () => void }> = ({ userId, onClose }) => {
  const [pending, setPending] = useState<PendingTask[]>([]);

  useEffect(() => {
    fetch(`/api/approvals/pending`).then(r => r.json()).then(setPending).catch(() => {});
  }, []);

  const getSLAStatus = (expiresAt: string) => {
    if (!expiresAt) return "OK";
    const ms = new Date(expiresAt).getTime() - Date.now();
    if (ms < 0) return "BREACHED";
    if (ms < 3600000) return "CRITICAL"; // < 1 hour
    if (ms < 86400000) return "AT-RISK"; // < 24 hours
    return "OK";
  };

  const getStatusColor = (status: string) => {
      switch(status) {
          case "BREACHED": return "bg-red-100 text-red-800 border-red-200";
          case "CRITICAL": return "bg-orange-100 text-orange-800 border-orange-200";
          case "AT-RISK": return "bg-yellow-100 text-yellow-800 border-yellow-200";
          default: return "bg-green-100 text-green-800 border-green-200";
      }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl max-h-[80vh] flex flex-col">
        <div className="p-4 border-b flex justify-between items-center bg-gray-50 rounded-t-lg">
            <h3 className="font-bold text-lg">My Approvals</h3>
            <button onClick={onClose} className="text-gray-500 hover:text-black">✕</button>
        </div>
        
        <div className="p-4 overflow-y-auto space-y-3 flex-1">
            {pending.length === 0 && <div className="text-center text-gray-500 py-8">No pending approvals.</div>}
            {pending.map(task => {
                const status = getSLAStatus(task.slaExpiresAt);
                return (
                    <div key={task.id} className={`p-4 rounded border flex justify-between items-center hover:shadow-md transition-shadow ${getStatusColor(status)}`}>
                        <div>
                             <div className="flex items-center space-x-2">
                                <h4 className="font-bold text-sm">{task.bpKey}</h4>
                                <span className="text-gray-400">→</span>
                                <span className="font-medium text-sm">{task.currentStep}</span>
                             </div>
                             <p className="text-sm mt-1">{task.entity} <span className="text-gray-600">({task.applicantName})</span></p>
                             <div className="text-xs mt-2 font-mono uppercase bg-white/50 inline-block px-1 rounded border border-black/10">
                                SLA: {status}
                             </div>
                        </div>
                        <div className="flex flex-col space-y-2">
                             <button className="bg-blue-600 text-white px-3 py-1 rounded text-sm hover:bg-blue-700 shadow-sm">Approve</button>
                             <button className="bg-white text-red-600 border border-red-200 px-3 py-1 rounded text-sm hover:bg-red-50">Reject</button>
                        </div>
                    </div>
                );
            })}
        </div>
      </div>
    </div>
  );
};
