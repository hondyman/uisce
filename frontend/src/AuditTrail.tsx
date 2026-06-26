import { useState, useEffect, useCallback } from 'react';
import { listAccessAuditLogs } from '../../api';
import { AccessControlAuditLog } from './types';

export default function AuditTrail() {
  const [logs, setLogs] = useState<AccessControlAuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchLogs = useCallback(async () => {
    try {
      setLoading(true);
      const data = await listAccessAuditLogs();
      setLogs(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch audit logs');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  if (loading) return <div className="loading">Loading audit trail...</div>;
  if (error) return <div className="error">Error: {error}</div>;

  return (
    <div>
      <h2>Access Control Audit Trail</h2>
      <table className="governance-table">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Actor</th>
            <th>Action</th>
            <th>Target</th>
            <th>Details</th>
          </tr>
        </thead>
        <tbody>
          {logs.map(log => (
            <tr key={log.id}>
              <td>{log.timestamp ? new Date(log.timestamp).toLocaleString() : '—'}</td>
              <td>{log.actor_user_id ?? 'system'}</td>
              <td>{(log.action ?? '').replace(/_/g, ' ')}</td>
              <td>{(log.target_type ?? 'n/a')}: <code>{log.target_id ?? ''}</code></td>
              <td><pre><code>{JSON.stringify(log.details ?? {}, null, 2)}</code></pre></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}