import React, { useState } from 'react';

export const BitemporalController: React.FC = () => {
  const [isActive, setIsActive] = useState(false);
  const [asOfDate, setAsOfDate] = useState('');
  const [knowledgeDate, setKnowledgeDate] = useState('');

  const applyTimeTravel = () => {
    if (!isActive) {
      sessionStorage.removeItem('uisce_temporal_context');
      window.location.reload();
      return;
    }

    sessionStorage.setItem('uisce_temporal_context', JSON.stringify({
      asOfDate: asOfDate ? new Date(asOfDate).toISOString() : null,
      knowledgeDate: knowledgeDate ? new Date(knowledgeDate).toISOString() : null
    }));
    
    window.location.reload(); 
  };

  return (
    <div className="flex items-center gap-4 bg-gray-800 text-white px-4 py-2 rounded-md shadow-lg border border-gray-700">
      <div className="flex items-center gap-2">
        <label className="text-xs text-gray-300 font-semibold uppercase tracking-wider">Time Travel:</label>
        <input 
          type="checkbox" 
          checked={isActive} 
          onChange={(e) => setIsActive(e.target.checked)}
          className="toggle-checkbox cursor-pointer"
        />
      </div>

      {isActive && (
        <>
          <div className="flex items-center gap-2 border-l border-gray-600 pl-4">
            <span className="text-xs text-gray-400">As-Of (Effective):</span>
            <input 
              type="datetime-local" 
              value={asOfDate}
              onChange={(e) => setAsOfDate(e.target.value)}
              className="bg-gray-900 border border-gray-600 rounded text-sm px-2 py-1 text-white focus:outline-none focus:border-blue-500"
            />
          </div>
          <div className="flex items-center gap-2 border-l border-gray-600 pl-4">
            <span className="text-xs text-gray-400">Knowledge Date:</span>
            <input 
              type="datetime-local" 
              value={knowledgeDate}
              onChange={(e) => setKnowledgeDate(e.target.value)}
              className="bg-gray-900 border border-gray-600 rounded text-sm px-2 py-1 text-white focus:outline-none focus:border-blue-500"
            />
          </div>
          <button 
            onClick={applyTimeTravel}
            className="ml-2 bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold px-3 py-1 rounded transition-colors cursor-pointer"
          >
            APPLY LENS
          </button>
        </>
      )}
    </div>
  );
};

export default BitemporalController;
