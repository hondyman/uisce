import React, { useState, useEffect } from "react";
import "../styles/my-approvals.css";

interface QueuedTask {
  instance_id: string;
  bp_key: string;
  step_key: string;
  approver_role: string;
  created_at: string;
  sla_expires_at: string;
  sla_status: "OK" | "AT_RISK" | "CRITICAL" | "BREACHED";
  hours_remaining: number;
  applicant_name?: string;
  amount?: string;
  entity?: string;
  assigned_to_user?: string;
}

export const MyApprovalsPage: React.FC = () => {
  const [tasks, setTasks] = useState<QueuedTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<"all" | "at_risk" | "critical" | "breached">("all");
  const [sortBy, setSortBy] = useState<"sla_expires_at" | "created_at" | "amount">("sla_expires_at");

  // Refresh every 5 seconds
  useEffect(() => {
    const fetchTasks = () => {
      const statusParam = filter === "all" ? "" : filter.toUpperCase();
      fetch(`/api/my-approvals?status=${statusParam}&sort=${sortBy}`)
        .then((r) => r.json())
        .then((data) => {
          setTasks(data.tasks || []);
          setLoading(false);
        })
        .catch(console.error);
    };

    fetchTasks();
    const interval = setInterval(fetchTasks, 5000);
    return () => clearInterval(interval);
  }, [filter, sortBy]);

  const assignToMe = async (instanceId: string) => {
       await fetch(`/api/instances/${instanceId}/assign-to-me`, { method: "POST" });
       // Optimistic or reload handled by interval
  };
  
  const unassign = async (instanceId: string) => {
      await fetch(`/api/instances/${instanceId}/unassign`, { method: "POST" });
  };

  return (
    <div className="my-approvals-page">
      <div className="header">
        <h1>My Approvals</h1>
        <p className="task-count">{tasks.length} pending</p>
      </div>

      <div className="controls">
        <select value={filter} onChange={(e) => setFilter(e.target.value as any)}>
          <option value="all">All</option>
          <option value="at_risk">At Risk</option>
          <option value="critical">Critical</option>
          <option value="breached">Breached</option>
        </select>

        <select value={sortBy} onChange={(e) => setSortBy(e.target.value as any)}>
          <option value="sla_expires_at">SLA Expires (Urgent First)</option>
          <option value="created_at">Oldest First</option>
          <option value="amount">Amount (Highest First)</option>
        </select>
      </div>

      {loading && tasks.length === 0 && <div className="spinner">Loading...</div>}

      <div className="task-list">
        {tasks.length === 0 && !loading ? (
          <div className="empty-state">No approvals waiting!</div>
        ) : (
          tasks.map((task) => (
            <div key={task.instance_id} className={`task-card sla-${task.sla_status.toLowerCase()}`}>
              <div className="task-header">
                <h3>{task.bp_key}</h3>
                <span className={`sla-badge ${task.sla_status.toLowerCase()}`}>
                  {task.sla_status} {task.hours_remaining ? `(${task.hours_remaining.toFixed(1)}h)` : ''}
                </span>
              </div>

              <div className="task-details">
                <p>
                  <strong>{task.applicant_name || 'Workflow Instance'}</strong> {task.entity && `- ${task.entity}`}
                </p>
                {task.amount && <p className="amount">${task.amount}</p>}
              </div>

              <div className="task-meta">
                <span className="step">{task.step_key}</span>
                <span className="created">{new Date(task.created_at).toLocaleDateString()}</span>
              </div>

              <div className="task-actions">
                <button
                  className="btn btn-primary"
                  // onClick={() => navigate(`/instances/${task.instance_id}/approve`)}
                  onClick={() => alert("Open Approval Form")}
                >
                  Open & Approve
                </button>
                {!task.assigned_to_user && (
                  <button
                    className="btn btn-secondary"
                    onClick={() => assignToMe(task.instance_id)}
                  >
                    Assign to Me
                  </button>
                )}
                {task.assigned_to_user && (
                  <button className="btn btn-outline" onClick={() => unassign(task.instance_id)}>
                    Unassign
                  </button>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
};
