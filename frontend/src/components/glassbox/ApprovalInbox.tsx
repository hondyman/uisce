import React, { useState } from 'react';

interface ApprovalTask {
  workflow_id: string;
  client_id: string;
  content: string;
  violations: string[];
}

// Mock data for now, would come from API
const MOCK_TASKS: ApprovalTask[] = [
  {
    workflow_id: "wf-123",
    client_id: "client-abc",
    content: "We guarantee a 10% return on this investment.",
    violations: ["advice_guarantee_claim"]
  }
];

export const ApprovalInbox: React.FC = () => {
  const [tasks, setTasks] = useState<ApprovalTask[]>(MOCK_TASKS);

  const handleSignal = async (workflowId: string, action: string) => {
    try {
      await fetch(`/api/approvals/${workflowId}/signal`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action,
          comment: action === 'approve' ? 'Approved by advisor' : 'Rejected due to violation',
          actor_id: 'advisor-1'
        })
      });
      // Remove task from list
      setTasks(tasks.filter(t => t.workflow_id !== workflowId));
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <div className="p-6 bg-gray-900 text-white rounded-lg shadow-xl mt-6">
      <h2 className="text-2xl font-bold mb-4 flex items-center">
        <span className="mr-2">👮</span> Compliance Inbox
      </h2>
      {tasks.length === 0 ? (
        <p className="text-gray-500">No pending approvals.</p>
      ) : (
        <div className="space-y-4">
          {tasks.map(task => (
            <div key={task.workflow_id} className="border border-red-500/30 bg-red-900/10 p-4 rounded">
              <div className="flex justify-between items-start">
                <div>
                  <h3 className="font-bold text-lg">Client: {task.client_id}</h3>
                  <p className="text-sm text-red-400 mt-1">Violations: {task.violations.join(", ")}</p>
                </div>
                <span className="bg-red-500 text-xs px-2 py-1 rounded">Action Required</span>
              </div>
              <div className="mt-3 p-3 bg-black/50 rounded font-mono text-sm">
                {task.content}
              </div>
              <div className="mt-4 flex space-x-3">
                <button
                  onClick={() => handleSignal(task.workflow_id, 'approve')}
                  className="px-4 py-2 bg-green-600 hover:bg-green-700 rounded text-sm font-bold"
                >
                  Approve Override
                </button>
                <button
                  onClick={() => handleSignal(task.workflow_id, 'reject')}
                  className="px-4 py-2 bg-red-600 hover:bg-red-700 rounded text-sm font-bold"
                >
                  Reject
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
