import { useState } from 'react';
import { devError } from './utils/devLogger';
import { useNotification } from './hooks/useNotification';
import { requestAccess } from './api';

interface RequestAccessModalProps {
  modelId: string;
  onClose: () => void;
}

export default function RequestAccessModal({ modelId, onClose }: RequestAccessModalProps) {
  const [reason, setReason] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const notification = useNotification();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!reason.trim()) return;
    setSubmitting(true);
    try {
      await requestAccess(modelId, 'read', reason);
      notification.success('Access requested successfully!');
      onClose();
    } catch (error) {
      devError('Failed to request access:', error);
      notification.error('Failed to request access. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={e => e.stopPropagation()}>
        <h3>Request Access</h3>
        <p>Request read access for semantic model: <strong>{modelId}</strong></p>
        <form onSubmit={handleSubmit}>
          <textarea
            value={reason}
            onChange={e => setReason(e.target.value)}
            placeholder="Reason for access (e.g., 'Need to explore churn metrics for Q3 analysis')"
            required
          />
          <div className="modal-actions">
            <button type="button" onClick={onClose}>Cancel</button>
            <button type="submit" disabled={submitting || !reason.trim()}>{submitting ? 'Submitting...' : 'Submit Request'}</button>
          </div>
        </form>
      </div>
    </div>
  );
}