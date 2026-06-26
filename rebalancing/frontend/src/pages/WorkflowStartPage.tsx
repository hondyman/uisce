import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";

interface InitiatableWorkflow {
  id: string;
  label?: string; // API returns IDs, we might map to labels if available
}

export const WorkflowStartPage: React.FC = () => {
  const navigate = useNavigate();
  const [workflows, setWorkflows] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    fetch("/api/workflows/initiatable")
      .then((r) => r.json())
      .then((data) => {
        setWorkflows(data.workflows || []);
        setLoading(false);
      })
      .catch((e) => {
        setError(e.message);
        setLoading(false);
      });
  }, []);

  const startWorkflow = async (workflowId: string) => {
    const res = await fetch(`/api/workflows/${workflowId}/can-initiate`, {
      method: "POST",
      body: JSON.stringify({ role: "Manager" }), // Mock role
    });
    const { allowed, reason } = await res.json();

    if (!allowed) {
      alert(`Cannot start workflow: ${reason}`);
      return;
    }

    // Redirect to form/start
    navigate(`/workflows/${workflowId}/start`);
  };

  if (loading) return <div>Loading workflows...</div>;
  if (error) return <div className="text-red-500">{error}</div>;

  return (
    <div className="p-8 max-w-4xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Start a Workflow</h1>
      {workflows.length === 0 ? (
          <div className="bg-yellow-50 p-4 rounded text-yellow-700">No workflows available.</div>
      ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {workflows.map((wfId) => (
            <div key={wfId} className="border rounded-lg p-6 bg-white shadow-sm hover:shadow-md transition-shadow">
                <div className="flex items-center justify-between mb-4">
                    <div className="bg-blue-100 p-2 rounded text-blue-600 font-bold text-xl">
                        {wfId.substring(0, 2).toUpperCase()}
                    </div>
                </div>
                <h3 className="text-lg font-semibold text-gray-900 mb-2">{wfId}</h3>
                <p className="text-gray-500 text-sm mb-4">Click to initiate this business process.</p>
                <button 
                    onClick={() => startWorkflow(wfId)}
                    className="w-full py-2 bg-blue-600 text-white rounded hover:bg-blue-700 font-medium"
                >
                    Start
                </button>
            </div>
            ))}
        </div>
      )}
    </div>
  );
};
