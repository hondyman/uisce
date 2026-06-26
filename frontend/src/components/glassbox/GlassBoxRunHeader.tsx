import React from 'react';

interface RunHeaderProps {
  runId: string;
  client: string;
  objective: string;
  status: string;
  policyVersion: string;
  onExport: () => void;
  onReplay: () => void;
}

export const GlassBoxRunHeader: React.FC<RunHeaderProps> = ({
  runId, client, objective, status, policyVersion, onExport, onReplay
}) => {
  return (
    <div className="bg-gray-800 p-4 rounded-t-lg border-b border-gray-700 flex justify-between items-center">
      <div>
        <div className="flex items-center space-x-3">
          <h2 className="text-xl font-bold text-white">{client}</h2>
          <span className={`px-2 py-0.5 rounded text-xs font-mono uppercase ${
            status === 'published' ? 'bg-green-900 text-green-300' : 'bg-yellow-900 text-yellow-300'
          }`}>
            {status}
          </span>
        </div>
        <div className="text-sm text-gray-400 mt-1 flex space-x-4">
          <span>ID: <span className="font-mono text-gray-300">{runId.substring(0, 8)}</span></span>
          <span>Obj: {objective}</span>
          <span>Policy: <span className="font-mono text-blue-400">{policyVersion}</span></span>
        </div>
      </div>
      <div className="flex space-x-2">
        <button onClick={onReplay} className="px-3 py-1.5 bg-gray-700 hover:bg-gray-600 rounded text-sm text-white flex items-center">
          <span className="mr-1">↺</span> Replay
        </button>
        <button onClick={onExport} className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 rounded text-sm text-white flex items-center">
          <span className="mr-1">⬇</span> Export Pack
        </button>
      </div>
    </div>
  );
};
