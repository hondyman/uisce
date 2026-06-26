import { useState, useEffect, useCallback } from 'react';
import { listAutomationPolicies, listAutomationLogs, runAutomationCycle } from './api';
import { useNotification } from './hooks/useNotification';
import { AutomationPolicy, AutomationLog } from './types';

const AutomationPolicyTable = ({ policies }: { policies: AutomationPolicy[] }) => (
  <div className="governance-card">
    <h4>Automation Policies</h4>
    <table className="governance-table">
      <thead>
        <tr>
          <th>Status</th>
          <th>Policy ID</th>
          <th>Description</th>
          <th>Trigger</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
        {policies.map(policy => (
          <tr key={policy.id}>
            <td>
              <span className={`status-badge ${policy.is_enabled ? 'status-success' : 'status-failed'}`}>
                {policy.is_enabled ? 'Active' : 'Disabled'}
              </span>
            </td>
            <td><code>{policy.policy_id}</code></td>
            <td>{policy.description}</td>
            <td>{policy.trigger}</td>
            <td>{policy.action}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const AutomationLogTable = ({ logs }: { logs: AutomationLog[] }) => (
  <div className="governance-card">
    <h4>Recent Automation Actions</h4>
    <table className="governance-table">
      <thead>
        <tr>
          <th>Timestamp</th>
          <th>Action</th>
          <th>Target</th>
          <th>Details</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        {logs.map(log => (
          <tr key={log.id}>
            <td>{new Date(log.timestamp).toLocaleString()}</td>
            <td>{log.action}</td>
            <td>{log.target_type}: <code>{log.target_id}</code></td>
            <td><pre><code>{JSON.stringify(log.details, null, 2)}</code></pre></td>
            <td>{log.status}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

export default function AutomationPanel() {
  const [policies, setPolicies] = useState<AutomationPolicy[]>([]);
  const [logs, setLogs] = useState<AutomationLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRunnning, setIsRunning] = useState(false);
  const notification = useNotification();

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [fetchedPolicies, fetchedLogs] = await Promise.all([listAutomationPolicies(), listAutomationLogs()]);
      setPolicies(fetchedPolicies);
      setLogs(fetchedLogs);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load automation data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleRunCycle = async () => {
    setIsRunning(true);
    try {
      await runAutomationCycle();
      notification.success('Automation cycle completed. Refreshing data...');
      fetchData();
    } catch (err) {
      notification.error('Failed to run automation cycle.');
    } finally {
      setIsRunning(false);
    }
  };

  if (loading) return <div>Loading Automation Status...</div>;
  if (error) return <div className="error-message">Error: {error}</div>;

  return (
    <div className="automation-panel">
      <div className="panel-header">
        <h3>Automation Status</h3>
        <button onClick={handleRunCycle} disabled={isRunnning}>{isRunnning ? 'Running...' : 'Run Cycle Now'}</button>
      </div>
      <AutomationPolicyTable policies={policies} />
      <AutomationLogTable logs={logs} />
    </div>
  );
}