import { useState, useEffect, useCallback } from 'react';
import { useNotification } from '../hooks/useNotification';
import TextPromptDialog from '../components/TextPromptDialog';
import { devError } from '../utils/devLogger';
import { listAccessRequests, approveAccessRequest, rejectAccessRequest, listSemanticViews } from '../../api';
import { SemanticModelAccessRequest, SemanticViewMeta as _SemanticViewMeta } from './types';

interface RequestInboxProps {
  domain?: string;
}

export default function RequestInbox({ domain }: RequestInboxProps) {
  const [requests, setRequests] = useState<SemanticModelAccessRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modelDomainMap, setModelDomainMap] = useState<Map<string, string>>(new Map());
  const notification = useNotification();
  const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
  const [rejectRequestId, setRejectRequestId] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');

  // Fetch all models to map their domains
  useEffect(() => {
    listSemanticViews("mock-datasource-id").then((models: any) => {
      const newMap = new Map<string, string>();
      models.forEach((m: any) => {
        if (m.domain) {
          newMap.set(m.id, m.domain);
        }
      });
      setModelDomainMap(newMap);
  }).catch((e: any) => { devError(e); });
  }, []);

  const fetchRequests = useCallback(async () => {
    try {
      setLoading(true);
      // In a real app, reviewerId would come from the user's session
      const reviewerId = 'current_reviewer';
      const data = await listAccessRequests({ reviewerId });
  const pendingRequests = data.filter((r: any) => r.status === 'pending');

  const filteredRequests = domain ? pendingRequests.filter((r: any) => modelDomainMap.get(r.model_id) === domain) : pendingRequests;
      setRequests(filteredRequests);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch requests');
    } finally {
      setLoading(false);
    }
  }, [domain, modelDomainMap]);

  useEffect(() => {
    fetchRequests();
  }, [fetchRequests]);

  const handleApprove = async (requestId: string) => {
    try {
      await approveAccessRequest(requestId);
      // Refresh the list after action
      fetchRequests();
      notification.success('Approved access request');
    } catch (err) {
      notification.error('Failed to approve request');
    }
  };

  const handleReject = async (requestId: string) => {
    setRejectRequestId(requestId);
    setRejectReason('');
    setRejectDialogOpen(true);
  };

  const handleRejectSubmit = async (reason: string) => {
    if (!rejectRequestId) return;
    try {
      await rejectAccessRequest(rejectRequestId, reason);
      setRejectDialogOpen(false);
      setRejectRequestId(null);
      notification.success('Rejected access request');
      fetchRequests();
    } catch (err) {
      notification.error('Failed to reject request');
    }
  };

  if (loading) return <div>Loading requests...</div>;
  if (error) return <div className="error">Error: {error}</div>;

  return (
    <div>
      <h2>Pending Access Requests</h2>
      {requests.length === 0 ? (
        <p>No pending requests.</p>
      ) : (
        <table className="governance-table">
          <thead>
            <tr>
              <th>User</th>
              <th>Model ID</th>
              <th>Permission</th>
              <th>Reason</th>
              <th>Requested At</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {requests.map(req => {
              const meta = req as { requested_permission?: string; permission?: string; requester_notes?: string; reason?: string };
              const permission = meta.requested_permission ?? meta.permission ?? '—';
              const notes = meta.requester_notes ?? meta.reason ?? '';
              return (
                <tr key={req.id ?? Math.random().toString()}>
                  <td>{req.user_id ?? 'unknown'}</td>
                  <td><code>{req.model_id ?? ''}</code></td>
                  <td>{permission}</td>
                  <td>{notes}</td>
                  <td>{req.requested_at ? new Date(req.requested_at).toLocaleString() : '—'}</td>
                  <td className="actions">
                    <button className="approve" onClick={() => req.id && handleApprove(req.id)}>Approve</button>
                    <button className="reject" onClick={() => req.id && handleReject(req.id)}>Reject</button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
      <TextPromptDialog
        open={Boolean(rejectDialogOpen)}
        title="Reject Access Request"
        label="Rejection reason"
        defaultValue={rejectReason}
        onClose={() => { setRejectDialogOpen(false); setRejectRequestId(null); }}
        onSubmit={handleRejectSubmit}
      />
    </div>
  );
}