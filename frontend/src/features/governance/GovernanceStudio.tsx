import React, { useState, useEffect } from 'react';
import { apiClient } from '../../utils/apiClient';

interface ChangeProposal {
  proposalId: string;
  boId: string;
  makerUserId: string;
  status: 'DRAFT' | 'PENDING_APPROVAL' | 'ACTIVE' | 'REJECTED';
  diffPayload: any;
  justification: string;
}

export const GovernanceStudio: React.FC = () => {
  const [currentUser] = useState<{ id: string }>({
    id: localStorage.getItem('user_id') || 'user-current'
  });
  const [proposals, setProposals] = useState<ChangeProposal[]>([]);
  const [selectedProposal, setSelectedProposal] = useState<ChangeProposal | null>(null);

  useEffect(() => {
    // Fetch pending proposals from the Go backend governance route
    apiClient<ChangeProposal[]>('/api/v1/governance/proposals?status=PENDING_APPROVAL')
      .then(data => {
        if (Array.isArray(data)) {
          setProposals(data);
        }
      })
      .catch(() => {
        // Fallback for dev mode
        setProposals([
          {
            proposalId: 'prop-101',
            boId: 'bo-sales-ledger',
            makerUserId: 'user-maker-001',
            status: 'PENDING_APPROVAL',
            justification: 'Update High Watermark Revenue formula to include discount adjustment',
            diffPayload: {
              targetBO: 'sales_ledger',
              action: 'UPDATE_SEMANTIC_CALCULATION',
              before: 'unit_price * quantity',
              after: 'unit_price * quantity * (1 - discount)'
            }
          }
        ]);
      });
  }, []);

  const handleReview = async (proposalId: string, approve: boolean) => {
    try {
      await apiClient(`/api/v1/governance/proposals/${proposalId}/review`, {
        method: 'POST',
        body: JSON.stringify({ approve })
      });
      setProposals(prev => prev.filter(p => p.proposalId !== proposalId));
      setSelectedProposal(null);
    } catch (err: any) {
      alert(`Governance Error: ${err.message || err}`);
    }
  };

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Left Pane: Pending Queue */}
      <div className="w-1/3 border-r bg-white flex flex-col">
        <div className="p-4 border-b bg-gray-100 font-bold text-gray-700">
          Pending Semantic Approvals (Four-Eyes)
        </div>
        <div className="overflow-y-auto flex-1">
          {proposals.map(p => (
            <div 
              key={p.proposalId} 
              onClick={() => setSelectedProposal(p)}
              className={`p-4 border-b cursor-pointer hover:bg-blue-50 ${selectedProposal?.proposalId === p.proposalId ? 'bg-blue-100' : ''}`}
            >
              <div className="font-semibold text-sm">BO ID: {p.boId}</div>
              <div className="text-xs text-gray-500 mt-1 truncate">Reason: {p.justification}</div>
              <div className="mt-2 text-xs font-mono bg-gray-200 inline-block px-2 py-1 rounded">
                Maker: {p.makerUserId.substring(0,8)}...
              </div>
            </div>
          ))}
          {proposals.length === 0 && <div className="p-4 text-gray-500 text-sm">No pending proposals.</div>}
        </div>
      </div>

      {/* Right Pane: Diff Viewer & Actions */}
      <div className="flex-1 flex flex-col">
        {selectedProposal ? (
          <>
            <div className="p-4 border-b bg-white flex justify-between items-center">
              <h2 className="font-bold text-lg">Proposal Review</h2>
              <div className="flex gap-2">
                <button 
                  onClick={() => handleReview(selectedProposal.proposalId, false)}
                  className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 font-semibold"
                >
                  Reject
                </button>
                <button 
                  onClick={() => handleReview(selectedProposal.proposalId, true)}
                  disabled={currentUser.id === selectedProposal.makerUserId}
                  className={`px-4 py-2 rounded font-semibold text-white ${
                    currentUser.id === selectedProposal.makerUserId 
                      ? 'bg-gray-400 cursor-not-allowed' 
                      : 'bg-green-600 hover:bg-green-700'
                  }`}
                  title={currentUser.id === selectedProposal.makerUserId ? "Maker cannot be Checker" : ""}
                >
                  Approve & Deploy
                </button>
              </div>
            </div>
            {/* Displaying the AST / Binding Diff */}
            <div className="flex-1 bg-gray-900 text-green-400 p-4 font-mono text-sm overflow-auto">
              <pre>{JSON.stringify(selectedProposal.diffPayload, null, 2)}</pre>
            </div>
            {currentUser.id === selectedProposal.makerUserId && (
              <div className="bg-yellow-100 text-yellow-800 p-2 text-center text-sm font-semibold border-t border-yellow-300">
                🚨 Four-Eyes Principle: You authored this change. Another steward must approve it.
              </div>
            )}
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-gray-400">
            Select a proposal to view semantic diffs.
          </div>
        )}
      </div>
    </div>
  );
};

export default GovernanceStudio;
