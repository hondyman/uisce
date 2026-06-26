import React, { useState, useEffect } from 'react';
import { GlassBoxRunHeader } from '../../components/glassbox/GlassBoxRunHeader';
import { DraftWithGuardrails } from '../../components/glassbox/DraftWithGuardrails';
import { ApprovalInbox } from '../../components/glassbox/ApprovalInbox';
import { AuditLogViewer } from '../../components/glassbox/AuditLogViewer';

// Mock data fetcher
const fetchRunData = async (runId: string) => {
  // In a real app, this would hit /api/glassbox/runs/{runId}
  return {
    runId,
    client: "Acme Corp Retirement Trust",
    objective: "Rebalance & Tax Loss Harvest",
    status: "review",
    policyVersion: "v3.2.1",
    draftContent: "Based on your objective to harvest losses, we recommend selling 500 shares of TLT. We guarantee this will result in a tax benefit.",
    violations: [
      {
        ruleId: "guaranteed_returns",
        severity: "critical",
        message: "Promissory language 'guarantee' is prohibited by FINRA Rule 2210.",
        start: 65,
        end: 75
      }
    ]
  };
};

export const AdvisorWorkspace: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'workspace' | 'audit'>('workspace');
  const [runData, setRunData] = useState<any>(null);

  useEffect(() => {
    fetchRunData("run-123").then(setRunData);
  }, []);

  if (!runData) return <div className="p-8 text-white">Loading Workspace...</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white font-sans">
      {/* Top Navigation / Context */}
      <GlassBoxRunHeader 
        runId={runData.runId}
        client={runData.client}
        objective={runData.objective}
        status={runData.status}
        policyVersion={runData.policyVersion}
        onExport={() => alert("Exporting Audit Pack...")}
        onReplay={() => alert("Starting Deterministic Replay...")}
      />

      {/* Main Content Area */}
      <div className="p-6 grid grid-cols-12 gap-6">
        
        {/* Left Sidebar: Context & Navigation */}
        <div className="col-span-2 space-y-2">
          <button 
            onClick={() => setActiveTab('workspace')}
            className={`w-full text-left px-4 py-2 rounded ${activeTab === 'workspace' ? 'bg-blue-600' : 'hover:bg-gray-800'}`}
          >
            Advisor Workspace
          </button>
          <button 
            onClick={() => setActiveTab('audit')}
            className={`w-full text-left px-4 py-2 rounded ${activeTab === 'audit' ? 'bg-blue-600' : 'hover:bg-gray-800'}`}
          >
            Audit Trail
          </button>
        </div>

        {/* Center Stage */}
        <div className="col-span-10">
          {activeTab === 'workspace' && (
            <div className="space-y-6">
              {/* Draft & Guardrails */}
              <DraftWithGuardrails 
                content={runData.draftContent}
                violations={runData.violations}
              />

              {/* Approval / Decision Console */}
              <div className="mt-8">
                 <h3 className="text-xl font-bold mb-4">Pending Approvals</h3>
                 <ApprovalInbox />
              </div>
            </div>
          )}

          {activeTab === 'audit' && (
            <AuditLogViewer />
          )}
        </div>
      </div>
    </div>
  );
};
