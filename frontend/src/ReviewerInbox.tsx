import { useState, useEffect } from 'react';
import { useNotification } from './hooks/useNotification';
import TextPromptDialog from './components/TextPromptDialog';
import { devError } from './utils/devLogger';
import { listAccessRequests, approveAccessRequest, rejectAccessRequest } from './api';
import type { SemanticModelAccessRequest } from './types';

interface ReviewerInboxProps {
  reviewerId: string;
}

export default function ReviewerInbox({ reviewerId }: ReviewerInboxProps) {
  const [requests, setRequests] = useState<SemanticModelAccessRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const notification = useNotification();
  const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
  const [rejectRequestId, setRejectRequestId] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');

  const fetchRequests = () => {
    setLoading(true);
    listAccessRequests({ reviewerId })
      .then(data => setRequests(data.filter(r => r.status === 'pending')))
      .catch((e) => { devError(e); })
      .finally(() => setLoading(false));
  };

  useEffect(fetchRequests, [reviewerId]);

  const handleApprove = async (requestId: string) => {
    await approveAccessRequest(requestId);
    fetchRequests(); // Refresh list
    notification.success('Approved request');
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
      notification.success('Rejected request');
      fetchRequests();
    } catch (err) {
      notification.error('Failed to reject request');
    }
  };

  if (loading) return <div>Loading pending requests...</div>;

  return (
    <div className="reviewer-inbox">
      <h4>Pending Access Requests ({requests.length})</h4>
      {requests.length === 0 ? (
        <p>No pending requests.</p>
      ) : (
        <div className="request-list">
          {requests.map(req => {
            const meta = req as { requested_permission?: string; reason?: string; requester_notes?: string };
            const permission = meta.requested_permission ?? '—';
            const reason = meta.reason ?? meta.requester_notes ?? '—';
            return (
              <div key={req.id ?? Math.random().toString()} className="request-card">
                <p>
                  <strong>{req.user_id ?? 'unknown'}</strong> requests <strong>{permission}</strong> access to model <strong>{req.model_id ?? ''}</strong>.
                </p>
                <blockquote>Reason: {reason}</blockquote>
                <small>Requested at: {req.requested_at ? new Date(req.requested_at).toLocaleString() : '—'}</small>
                <div className="request-actions">
                  <button onClick={() => req.id && handleApprove(req.id)}>Approve</button>
                  <button onClick={() => req.id && handleReject(req.id)} className="destructive">Reject</button>
                </div>
              </div>
            );
          })}
        </div>
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